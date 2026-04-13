// Context: implements the media helper logic used by the stream backend service.

package stream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub-core/header"
	proto "github.com/yttydcs/myflowhub-proto/protocol/stream"
)

const (
	sourceInputKindFile    = "file"
	sourceInputKindDesktop = "desktop"

	streamDataFlagEOF          = 1 << 0
	streamDataFlagSessionStart = 1 << 1

	mediaStateBuffering = "buffering"
	mediaStateReady     = "ready"
	mediaStateComplete  = "complete"
	mediaStateError     = "error"
	mediaStateClosed    = "closed"

	fileSourceChunkBytes = 64 * 1024
)

type SourceInputConfigReq struct {
	SourceID  string `json:"source_id"`
	InputKind string `json:"input_kind"`
	FilePath  string `json:"file_path,omitempty"`
}

type SourceInputConfigResp struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg,omitempty"`
	SourceID  string `json:"source_id"`
	InputKind string `json:"input_kind,omitempty"`
	FilePath  string `json:"file_path,omitempty"`
}

type sourceInputConfig struct {
	SourceID  string
	InputKind string
	FilePath  string
}

type fileDeliverySender struct {
	DeliveryID string
	SourceID   string
	Producer   uint32
	Consumer   uint32
	FilePath   string
	Offset     uint64
	cancel     context.CancelFunc
}

type mediaRuntime struct {
	DeliveryID     string
	Kind           string
	ContentType    string
	State          string
	MediaURL       string
	AvailableBytes uint64
	Complete       bool
	Error          string
	UpdatedAt      time.Time
	LastEmit       time.Time
	BasePosition   uint64
	SessionSeq     uint64
	sink           *progressiveMediaSink
}

func (m *mediaRuntime) snapshot() StreamMediaEvent {
	return StreamMediaEvent{
		DeliveryID:     m.DeliveryID,
		Kind:           m.Kind,
		ContentType:    m.ContentType,
		State:          m.State,
		MediaURL:       m.MediaURL,
		AvailableBytes: m.AvailableBytes,
		Complete:       m.Complete,
		Error:          m.Error,
		UpdatedAt:      m.UpdatedAt.Format(time.RFC3339Nano),
	}
}

type progressiveMediaSink struct {
	mu             sync.Mutex
	file           *os.File
	path           string
	availableBytes uint64
	complete       bool
	closed         bool
	errText        string
}

func newProgressiveMediaSink() (*progressiveMediaSink, error) {
	file, err := os.CreateTemp("", "myflowhub-stream-media-*")
	if err != nil {
		return nil, err
	}
	return &progressiveMediaSink{
		file: file,
		path: file.Name(),
	}, nil
}

func (s *progressiveMediaSink) WriteChunk(position uint64, body []byte, final bool) (uint64, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return s.availableBytes, s.complete, errors.New("media sink closed")
	}
	if s.errText != "" {
		return s.availableBytes, s.complete, errors.New(s.errText)
	}
	if position != s.availableBytes {
		return s.availableBytes, s.complete, errors.New("media chunk position mismatch")
	}
	if len(body) > 0 {
		n, err := s.file.WriteAt(body, int64(position))
		if err != nil {
			s.errText = err.Error()
			return s.availableBytes, s.complete, err
		}
		s.availableBytes += uint64(n)
	}
	if final {
		s.complete = true
	}
	return s.availableBytes, s.complete, nil
}

func (s *progressiveMediaSink) Fail(reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "media sink failed"
	}
	s.errText = reason
}

func (s *progressiveMediaSink) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	file := s.file
	path := s.path
	s.file = nil
	s.mu.Unlock()

	var firstErr error
	if file != nil {
		if err := file.Close(); err != nil {
			firstErr = err
		}
	}
	if path != "" {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *progressiveMediaSink) snapshotState() (available uint64, complete bool, errText string, closed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.availableBytes, s.complete, s.errText, s.closed
}

func (s *progressiveMediaSink) readChunk(offset int64, size int) ([]byte, bool, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, s.complete, s.errText, true
	}
	if uint64(offset) >= s.availableBytes {
		return nil, s.complete, s.errText, false
	}
	remain := int(s.availableBytes - uint64(offset))
	if remain > size {
		remain = size
	}
	buf := make([]byte, remain)
	n, err := s.file.ReadAt(buf, offset)
	if err != nil && !errors.Is(err, io.EOF) {
		s.errText = err.Error()
		return nil, s.complete, s.errText, false
	}
	return buf[:n], s.complete, s.errText, false
}

func (s *progressiveMediaSink) ServeHTTP(w http.ResponseWriter, r *http.Request, contentType string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Accept-Ranges", "none")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	flusher, _ := w.(http.Flusher)
	offset := int64(0)
	for {
		chunk, complete, errText, closed := s.readChunk(offset, fileSourceChunkBytes)
		if len(chunk) > 0 {
			if _, err := w.Write(chunk); err != nil {
				return
			}
			offset += int64(len(chunk))
			if flusher != nil {
				flusher.Flush()
			}
			continue
		}
		if closed || complete || errText != "" {
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-time.After(25 * time.Millisecond):
		}
	}
}

type mediaHTTPServer struct {
	baseURL  string
	listener net.Listener
	server   *http.Server
}

func newMediaHTTPServer(stream *StreamService) (*mediaHTTPServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}
	out := &mediaHTTPServer{
		baseURL:  "http://" + listener.Addr().String(),
		listener: listener,
		server:   server,
	}
	mux.HandleFunc("/stream/", stream.serveMediaHTTP)
	go func() {
		_ = server.Serve(listener)
	}()
	return out, nil
}

func (s *mediaHTTPServer) URLFor(deliveryID string) string {
	return s.baseURL + "/stream/" + strings.TrimSpace(deliveryID)
}

func (s *mediaHTTPServer) Close() error {
	if s == nil || s.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func (s *StreamService) MediaSnapshot() []StreamMediaEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureStateMapsLocked()
	out := make([]StreamMediaEvent, 0, len(s.media))
	for _, item := range s.media {
		if item == nil {
			continue
		}
		out = append(out, item.snapshot())
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	return out
}

func (s *StreamService) ConfigureSourceInput(ctx context.Context, sourceID uint32, req SourceInputConfigReq) (SourceInputConfigResp, error) {
	_ = ctx
	if sourceID == 0 {
		return SourceInputConfigResp{}, errors.New("sourceID is required")
	}
	sourceKey := strings.TrimSpace(req.SourceID)
	if sourceKey == "" {
		return SourceInputConfigResp{}, errors.New("source_id is required")
	}
	inputKind := strings.ToLower(strings.TrimSpace(req.InputKind))

	s.mu.Lock()
	s.ensureStateMapsLocked()
	source, ok := s.sources[sourceKey]
	if ok && source.Producer != sourceID {
		s.mu.Unlock()
		return SourceInputConfigResp{}, errors.New("source producer mismatch")
	}
	if inputKind == "" {
		delete(s.sourceInputs, sourceKey)
		s.cancelFileSendersBySourceLocked(sourceKey)
		s.mu.Unlock()
		return SourceInputConfigResp{Code: 1, Msg: "ok", SourceID: sourceKey}, nil
	}
	if inputKind != sourceInputKindFile && inputKind != sourceInputKindDesktop {
		s.mu.Unlock()
		return SourceInputConfigResp{}, errors.New("unsupported input_kind")
	}
	if ok && normalizeRequiredKind(source.Kind) == proto.StreamKindText {
		s.mu.Unlock()
		return SourceInputConfigResp{}, errors.New("text sources do not use source inputs")
	}
	if inputKind == sourceInputKindDesktop {
		if ok && normalizeRequiredKind(source.Kind) != proto.StreamKindVideo {
			s.mu.Unlock()
			return SourceInputConfigResp{}, errors.New("desktop capture requires a video source")
		}
		prev, existed := s.sourceInputs[sourceKey]
		s.sourceInputs[sourceKey] = sourceInputConfig{
			SourceID:  sourceKey,
			InputKind: inputKind,
		}
		changed := !existed || prev.InputKind != inputKind || prev.FilePath != ""
		if changed {
			s.cancelFileSendersBySourceLocked(sourceKey)
		}
		s.mu.Unlock()
		return SourceInputConfigResp{
			Code:      1,
			Msg:       "ok",
			SourceID:  sourceKey,
			InputKind: inputKind,
		}, nil
	}
	filePath := strings.TrimSpace(req.FilePath)
	s.mu.Unlock()
	if filePath == "" {
		return SourceInputConfigResp{}, errors.New("file_path is required")
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return SourceInputConfigResp{}, err
	}
	if info.IsDir() {
		return SourceInputConfigResp{}, errors.New("file_path must be a file")
	}
	if info.Size() <= 0 {
		return SourceInputConfigResp{}, errors.New("media file is empty")
	}

	s.mu.Lock()
	prev, existed := s.sourceInputs[sourceKey]
	s.sourceInputs[sourceKey] = sourceInputConfig{
		SourceID:  sourceKey,
		InputKind: inputKind,
		FilePath:  filePath,
	}
	changed := !existed || prev.InputKind != inputKind || prev.FilePath != filePath
	if changed {
		s.cancelFileSendersBySourceLocked(sourceKey)
	}
	active := s.activeProducerDeliveryIDsLocked(sourceKey)
	s.mu.Unlock()
	if changed {
		for _, deliveryID := range active {
			s.maybeStartProducerFileSender(deliveryID)
		}
	}
	return SourceInputConfigResp{
		Code:      1,
		Msg:       "ok",
		SourceID:  sourceKey,
		InputKind: inputKind,
		FilePath:  filePath,
	}, nil
}

func (s *StreamService) ConfigureSourceInputSimple(sourceID uint32, req SourceInputConfigReq) (SourceInputConfigResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStreamTimeout)
	defer cancel()
	return s.ConfigureSourceInput(ctx, sourceID, req)
}

func (s *StreamService) maybeStartProducerFileSender(deliveryID string) {
	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return
	}
	s.mu.Lock()
	s.ensureStateMapsLocked()
	delivery := s.producerDeliveries[deliveryID]
	if delivery == nil || delivery.State != localDeliveryStateActive {
		s.mu.Unlock()
		return
	}
	if _, exists := s.fileSenders[deliveryID]; exists {
		s.mu.Unlock()
		return
	}
	source, ok := s.sources[delivery.SourceID]
	cfg, hasInput := s.sourceInputs[delivery.SourceID]
	if !ok || !hasInput || cfg.InputKind != sourceInputKindFile {
		s.mu.Unlock()
		return
	}
	if strings.TrimSpace(source.UnitMode) != proto.UnitModeChunk || strings.TrimSpace(delivery.UnitMode) != proto.UnitModeChunk {
		snapshot := s.upsertProducerSnapshotLocked(delivery, source, localDeliveryStateActive, "media file streaming requires chunk unit mode")
		s.mu.Unlock()
		s.emitDelivery(snapshot)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	sender := &fileDeliverySender{
		DeliveryID: delivery.DeliveryID,
		SourceID:   delivery.SourceID,
		Producer:   delivery.Producer,
		Consumer:   delivery.Consumer,
		FilePath:   cfg.FilePath,
		Offset:     delivery.Position,
		cancel:     cancel,
	}
	s.fileSenders[deliveryID] = sender
	s.mu.Unlock()
	go s.runFileDeliverySender(ctx, sender)
}

func (s *StreamService) runFileDeliverySender(ctx context.Context, sender *fileDeliverySender) {
	defer s.clearFileSender(sender.DeliveryID, sender)

	file, err := os.Open(sender.FilePath)
	if err != nil {
		s.failProducerDelivery(sender.DeliveryID, err.Error())
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		s.failProducerDelivery(sender.DeliveryID, err.Error())
		return
	}
	totalSize := uint64(info.Size())

	if sender.Offset > 0 {
		if _, err := file.Seek(int64(sender.Offset), io.SeekStart); err != nil {
			s.failProducerDelivery(sender.DeliveryID, err.Error())
			return
		}
	}

	buf := make([]byte, fileSourceChunkBytes)
	position := sender.Offset
	for {
		n, readErr := file.Read(buf)
		if n == 0 {
			if errors.Is(readErr, io.EOF) {
				return
			}
			if readErr != nil {
				s.failProducerDelivery(sender.DeliveryID, readErr.Error())
			}
			return
		}
		if err := s.waitProducerWindow(ctx, sender.DeliveryID, position+uint64(n)); err != nil {
			if !errors.Is(err, context.Canceled) {
				s.failProducerDelivery(sender.DeliveryID, err.Error())
			}
			return
		}
		flags := uint8(0)
		if errors.Is(readErr, io.EOF) || position+uint64(n) >= totalSize {
			flags = streamDataFlagEOF
		}
		ptsMs := uint64(time.Now().UnixMilli())
		payload, err := buildDataPayloadWithFlags(sender.DeliveryID, position, ptsMs, flags, append([]byte(nil), buf[:n]...))
		if err != nil {
			s.failProducerDelivery(sender.DeliveryID, err.Error())
			return
		}
		hdr := (&header.HeaderTcp{}).
			WithMajor(header.MajorMsg).
			WithSubProto(proto.SubProtoStream).
			WithSourceID(sender.Producer).
			WithTargetID(sender.Consumer).
			WithTimestamp(uint32(time.Now().Unix()))
		if err := s.session.Send(hdr, payload); err != nil {
			s.failProducerDelivery(sender.DeliveryID, err.Error())
			return
		}
		if snapshot, ok := s.markProducerPayloadSent(sender.DeliveryID, position, uint64(n), ptsMs, flags); ok {
			s.emitDelivery(snapshot)
		}
		position += uint64(n)
		if errors.Is(readErr, io.EOF) {
			return
		}
	}
}

func (s *StreamService) waitProducerWindow(ctx context.Context, deliveryID string, nextEnd uint64) error {
	for {
		s.mu.Lock()
		s.ensureStateMapsLocked()
		delivery := s.producerDeliveries[deliveryID]
		if delivery == nil {
			s.mu.Unlock()
			return errors.New("delivery closed")
		}
		if delivery.State != localDeliveryStateActive {
			s.mu.Unlock()
			return errors.New("delivery not active")
		}
		allowed := delivery.AckedPosition + uint64(coalesceWindowBytes(delivery.WindowBytes))
		s.mu.Unlock()
		if nextEnd <= allowed {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func (s *StreamService) markProducerPayloadSent(deliveryID string, position, bodyLen, ptsMs uint64, flags uint8) (StreamDeliveryEvent, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureStateMapsLocked()
	delivery := s.producerDeliveries[deliveryID]
	if delivery == nil {
		return StreamDeliveryEvent{}, false
	}
	source, ok := s.sources[delivery.SourceID]
	if !ok {
		return StreamDeliveryEvent{}, false
	}
	delivery.Position = nextExpectedPosition(delivery.UnitMode, position, int(bodyLen))
	delivery.LastActive = time.Now()
	snapshot := s.upsertProducerSnapshotLocked(delivery, source, localDeliveryStateActive, "")
	if rt := s.deliveries[deliveryID]; rt != nil {
		rt.BytesIn += bodyLen
		rt.FramesIn++
		rt.LastPosition = position
		rt.LastPtsMs = ptsMs
		rt.LastFlags = flags
		rt.UpdatedAt = time.Now()
		snapshot = rt.snapshot()
	}
	return snapshot, true
}

func (s *StreamService) failProducerDelivery(deliveryID, reason string) {
	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "producer send failed"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureStateMapsLocked()
	if delivery := s.producerDeliveries[deliveryID]; delivery != nil {
		delivery.LastActive = time.Now()
		if source, ok := s.sources[delivery.SourceID]; ok {
			snapshot := s.upsertProducerSnapshotLocked(delivery, source, localDeliveryStateActive, reason)
			s.mu.Unlock()
			s.emitDelivery(snapshot)
			s.mu.Lock()
			return
		}
	}
	if rt := s.deliveries[deliveryID]; rt != nil {
		rt.LastError = reason
		rt.UpdatedAt = time.Now()
		snapshot := rt.snapshot()
		s.mu.Unlock()
		s.emitDelivery(snapshot)
		s.mu.Lock()
	}
}

func (s *StreamService) activeProducerDeliveryIDsLocked(sourceID string) []string {
	out := make([]string, 0)
	for deliveryID, item := range s.producerDeliveries {
		if item == nil || item.SourceID != sourceID || item.State != localDeliveryStateActive {
			continue
		}
		out = append(out, deliveryID)
	}
	return out
}

func (s *StreamService) cancelFileSenderLocked(deliveryID string) {
	if sender := s.fileSenders[deliveryID]; sender != nil {
		delete(s.fileSenders, deliveryID)
		sender.cancel()
	}
}

func (s *StreamService) cancelFileSendersBySourceLocked(sourceID string) {
	for deliveryID, sender := range s.fileSenders {
		if sender == nil || sender.SourceID != sourceID {
			continue
		}
		delete(s.fileSenders, deliveryID)
		sender.cancel()
	}
}

func (s *StreamService) clearFileSender(deliveryID string, sender *fileDeliverySender) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current := s.fileSenders[deliveryID]; current == sender {
		delete(s.fileSenders, deliveryID)
	}
}

func (s *StreamService) ensureMediaServer() (*mediaHTTPServer, error) {
	s.mu.Lock()
	if s.mediaServer != nil {
		server := s.mediaServer
		s.mu.Unlock()
		return server, nil
	}
	s.mu.Unlock()

	server, err := newMediaHTTPServer(s)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	if s.mediaServer == nil {
		s.mediaServer = server
	} else {
		_ = server.Close()
		server = s.mediaServer
	}
	s.mu.Unlock()
	return server, nil
}

func (s *StreamService) serveMediaHTTP(w http.ResponseWriter, r *http.Request) {
	deliveryID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/stream/"))
	if deliveryID == "" {
		http.NotFound(w, r)
		return
	}
	s.mu.Lock()
	entry := s.media[deliveryID]
	s.mu.Unlock()
	if entry == nil || entry.sink == nil {
		http.NotFound(w, r)
		return
	}
	entry.sink.ServeHTTP(w, r, entry.ContentType)
}

func mediaSessionURL(baseURL string, sessionSeq uint64) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" || sessionSeq == 0 {
		return baseURL
	}
	return fmt.Sprintf("%s?session=%d", baseURL, sessionSeq)
}

func (s *StreamService) resetMediaRuntimeSession(deliveryID string, basePosition uint64) (*mediaRuntime, error) {
	server, err := s.ensureMediaServer()
	if err != nil {
		return nil, err
	}
	sink, err := newProgressiveMediaSink()
	if err != nil {
		return nil, err
	}

	var (
		entry    *mediaRuntime
		oldSink  *progressiveMediaSink
		snapshot StreamMediaEvent
	)

	s.mu.Lock()
	s.ensureStateMapsLocked()
	entry = s.media[deliveryID]
	if entry == nil {
		s.mu.Unlock()
		_ = sink.Close()
		return nil, errors.New("media runtime not prepared")
	}
	now := time.Now()
	oldSink = entry.sink
	entry.SessionSeq++
	entry.BasePosition = basePosition
	entry.State = mediaStateBuffering
	entry.MediaURL = mediaSessionURL(server.URLFor(deliveryID), entry.SessionSeq)
	entry.AvailableBytes = 0
	entry.Complete = false
	entry.Error = ""
	entry.UpdatedAt = now
	entry.LastEmit = now
	entry.sink = sink
	snapshot = entry.snapshot()
	s.mu.Unlock()

	if oldSink != nil {
		_ = oldSink.Close()
	}
	s.emitMedia(snapshot)
	return entry, nil
}

func (s *StreamService) prepareConsumerMediaRuntime(deliveryID string) {
	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return
	}
	s.mu.Lock()
	s.ensureStateMapsLocked()
	delivery := s.consumerDeliveries[deliveryID]
	rt := s.deliveries[deliveryID]
	if delivery == nil || rt == nil || delivery.State != localDeliveryStateActive {
		s.mu.Unlock()
		return
	}
	kind := normalizeRequiredKind(rt.Kind)
	contentType := strings.TrimSpace(rt.ContentType)
	unitMode := strings.TrimSpace(rt.UnitMode)
	old := s.media[deliveryID]
	delete(s.media, deliveryID)
	s.mu.Unlock()
	if old != nil && old.sink != nil {
		_ = old.sink.Close()
	}
	if kind == proto.StreamKindText {
		return
	}
	if reason := mediaPlaybackSupportError(kind, contentType, unitMode); reason != "" {
		s.setMediaState(deliveryID, kind, contentType, mediaStateError, "", 0, false, reason, nil, true)
		return
	}
	server, err := s.ensureMediaServer()
	if err != nil {
		s.setMediaState(deliveryID, kind, contentType, mediaStateError, "", 0, false, err.Error(), nil, true)
		return
	}
	sink, err := newProgressiveMediaSink()
	if err != nil {
		s.setMediaState(deliveryID, kind, contentType, mediaStateError, "", 0, false, err.Error(), nil, true)
		return
	}
	s.setMediaState(deliveryID, kind, contentType, mediaStateBuffering, server.URLFor(deliveryID), 0, false, "", sink, true)
}

func (s *StreamService) writeMediaChunk(deliveryID string, packet dataPacket) {
	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return
	}
	s.mu.Lock()
	entry := s.media[deliveryID]
	s.mu.Unlock()
	if entry == nil || entry.sink == nil {
		return
	}
	if packet.Flags&streamDataFlagSessionStart != 0 {
		if _, err := s.resetMediaRuntimeSession(deliveryID, packet.Position); err != nil {
			s.setMediaState(deliveryID, entry.Kind, entry.ContentType, mediaStateError, entry.MediaURL, entry.AvailableBytes, entry.Complete, err.Error(), entry.sink, true)
			return
		}
		s.mu.Lock()
		entry = s.media[deliveryID]
		s.mu.Unlock()
		if entry == nil || entry.sink == nil {
			return
		}
	}
	if packet.Position < entry.BasePosition {
		err := errors.New("media chunk position is before current session start")
		entry.sink.Fail(err.Error())
		s.setMediaState(deliveryID, entry.Kind, entry.ContentType, mediaStateError, entry.MediaURL, entry.AvailableBytes, entry.Complete, err.Error(), entry.sink, true)
		return
	}
	relativePosition := packet.Position - entry.BasePosition
	available, complete, err := entry.sink.WriteChunk(relativePosition, packet.Body, packet.Flags&streamDataFlagEOF != 0)
	if err != nil {
		entry.sink.Fail(err.Error())
		s.setMediaState(deliveryID, entry.Kind, entry.ContentType, mediaStateError, entry.MediaURL, available, complete, err.Error(), entry.sink, true)
		return
	}
	state := mediaStateBuffering
	if complete {
		state = mediaStateComplete
	} else if available > 0 {
		state = mediaStateReady
	}
	s.setMediaState(deliveryID, entry.Kind, entry.ContentType, state, entry.MediaURL, available, complete, "", entry.sink, false)
}

func (s *StreamService) setMediaState(deliveryID, kind, contentType, state, mediaURL string, available uint64, complete bool, errText string, sink *progressiveMediaSink, force bool) {
	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return
	}
	s.mu.Lock()
	s.ensureStateMapsLocked()
	entry := s.media[deliveryID]
	if entry == nil {
		entry = &mediaRuntime{DeliveryID: deliveryID}
		s.media[deliveryID] = entry
	}
	now := time.Now()
	shouldEmit := force || now.Sub(entry.LastEmit) >= streamStatsEvery
	if entry.State != state || entry.Error != strings.TrimSpace(errText) || entry.MediaURL != strings.TrimSpace(mediaURL) || entry.AvailableBytes != available || entry.Complete != complete {
		shouldEmit = true
	}
	entry.Kind = kind
	entry.ContentType = contentType
	entry.State = state
	entry.MediaURL = strings.TrimSpace(mediaURL)
	entry.AvailableBytes = available
	entry.Complete = complete
	entry.Error = strings.TrimSpace(errText)
	entry.UpdatedAt = now
	if sink != nil {
		entry.sink = sink
	}
	if !shouldEmit {
		s.mu.Unlock()
		return
	}
	entry.LastEmit = now
	snapshot := entry.snapshot()
	s.mu.Unlock()
	s.emitMedia(snapshot)
}

func (s *StreamService) closeMediaRuntime(deliveryID, reason string) {
	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return
	}
	s.mu.Lock()
	entry := s.media[deliveryID]
	if entry == nil {
		s.mu.Unlock()
		return
	}
	delete(s.media, deliveryID)
	entry.State = mediaStateClosed
	reason = strings.TrimSpace(reason)
	if reason != "" && entry.Error == "" {
		entry.Error = reason
	}
	entry.UpdatedAt = time.Now()
	snapshot := entry.snapshot()
	sink := entry.sink
	s.mu.Unlock()
	if sink != nil {
		_ = sink.Close()
	}
	s.emitMedia(snapshot)
}

func (s *StreamService) closeMediaResources() {
	s.mu.Lock()
	s.ensureStateMapsLocked()
	senders := make([]*fileDeliverySender, 0, len(s.fileSenders))
	for _, sender := range s.fileSenders {
		if sender != nil {
			senders = append(senders, sender)
		}
	}
	s.fileSenders = make(map[string]*fileDeliverySender)
	mediaItems := make([]*mediaRuntime, 0, len(s.media))
	for _, item := range s.media {
		if item != nil {
			mediaItems = append(mediaItems, item)
		}
	}
	s.media = make(map[string]*mediaRuntime)
	server := s.mediaServer
	s.mediaServer = nil
	s.mu.Unlock()

	for _, sender := range senders {
		sender.cancel()
	}
	for _, item := range mediaItems {
		if item.sink != nil {
			_ = item.sink.Close()
		}
	}
	if server != nil {
		_ = server.Close()
	}
}

func mediaPlaybackSupportError(kind, contentType, unitMode string) string {
	if unitMode != proto.UnitModeChunk {
		return "media playback requires chunk unit mode"
	}
	switch kind {
	case proto.StreamKindMusic:
		if !strings.HasPrefix(strings.ToLower(contentType), "audio/") {
			return "music sources require an audio content type"
		}
	case proto.StreamKindVideo:
		if !strings.HasPrefix(strings.ToLower(contentType), "video/") {
			return "video sources require a video content type"
		}
	default:
		return "runtime playback is only available for audio and video kinds"
	}
	return ""
}

type StreamMediaFileChoice struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	SizeBytes   int64  `json:"sizeBytes"`
	Kind        string `json:"kind"`
	ContentType string `json:"contentType"`
}

func DetectMediaFile(path string) StreamMediaFileChoice {
	path = strings.TrimSpace(path)
	contentType := detectMediaContentType(path)
	return StreamMediaFileChoice{
		Path:        path,
		Name:        filepath.Base(path),
		SizeBytes:   fileSizeBestEffort(path),
		Kind:        detectMediaKind(contentType, path),
		ContentType: contentType,
	}
}

func detectMediaContentType(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	contentType := strings.TrimSpace(mime.TypeByExtension(strings.ToLower(filepath.Ext(path))))
	if contentType != "" {
		if idx := strings.Index(contentType, ";"); idx >= 0 {
			contentType = strings.TrimSpace(contentType[:idx])
		}
		return contentType
	}
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return ""
	}
	contentType = http.DetectContentType(buf[:n])
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	return contentType
}

func detectMediaKind(contentType, path string) string {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.HasPrefix(contentType, "audio/"):
		return proto.StreamKindMusic
	case strings.HasPrefix(contentType, "video/"):
		return proto.StreamKindVideo
	}
	switch strings.ToLower(strings.TrimSpace(filepath.Ext(path))) {
	case ".mp3", ".wav", ".ogg", ".aac", ".m4a", ".flac":
		return proto.StreamKindMusic
	case ".mp4", ".webm", ".mov", ".mkv", ".ogv":
		return proto.StreamKindVideo
	default:
		return ""
	}
}

func fileSizeBestEffort(path string) int64 {
	info, err := os.Stat(strings.TrimSpace(path))
	if err != nil {
		return 0
	}
	return info.Size()
}
