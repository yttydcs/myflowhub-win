package stream

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/yttydcs/myflowhub-core/eventbus"
	"github.com/yttydcs/myflowhub-core/header"
	proto "github.com/yttydcs/myflowhub-proto/protocol/stream"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
)

const (
	streamFrameHeaderLen = 35
	streamAckHeaderLen   = 35
	streamStatsEvery     = 250 * time.Millisecond
	streamTextLimit      = 4096
)

type busToken struct {
	name  string
	token string
}

type deliveryRuntime struct {
	DeliveryID string
	SourceID   string
	Producer   uint32
	Consumer   uint32
	ConsumerID string
	Kind       string

	ContentType string
	Mode        string
	UnitMode    string

	State        string
	BytesIn      uint64
	FramesIn     uint64
	LastPosition uint64
	LastPtsMs    uint64
	LastAckPos   uint64
	LastFlags    uint8
	LastError    string

	LastFrameAt   time.Time
	LastStatsEmit time.Time
	UpdatedAt     time.Time
}

func (d *deliveryRuntime) snapshot() StreamDeliveryEvent {
	return StreamDeliveryEvent{
		DeliveryID:   d.DeliveryID,
		SourceID:     d.SourceID,
		Producer:     d.Producer,
		Consumer:     d.Consumer,
		ConsumerID:   d.ConsumerID,
		Kind:         d.Kind,
		ContentType:  d.ContentType,
		Mode:         d.Mode,
		UnitMode:     d.UnitMode,
		State:        d.State,
		BytesIn:      d.BytesIn,
		FramesIn:     d.FramesIn,
		LastPosition: d.LastPosition,
		LastPtsMs:    d.LastPtsMs,
		LastAckPos:   d.LastAckPos,
		LastFlags:    d.LastFlags,
		LastError:    d.LastError,
		UpdatedAt:    d.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func (s *StreamService) bindBus() {
	if s == nil || s.bus == nil {
		return
	}
	addToken := func(name string, handler func(evt any)) {
		token := s.bus.Subscribe(name, func(_ context.Context, evt eventbus.Event) {
			if handler == nil {
				return
			}
			handler(evt.Data)
		})
		if token != "" {
			s.busTokens = append(s.busTokens, busToken{name: name, token: token})
		}
	}
	addToken(sessionsvc.EventFrame, func(data any) {
		frame, ok := data.(sessionsvc.FrameEvent)
		if !ok {
			return
		}
		if frame.SubProto != proto.SubProtoStream {
			return
		}
		switch frame.Major {
		case header.MajorCmd:
			s.handleIncomingCtrl(frame)
		case header.MajorMsg:
			s.handleFrame(frame)
		}
	})
	addToken(sessionsvc.EventState, func(data any) {
		state, ok := data.(sessionsvc.StateEvent)
		if !ok || state.Connected {
			return
		}
		s.markAllClosed("disconnected")
	})
	addToken(sessionsvc.EventError, func(data any) {
		errEvt, ok := data.(sessionsvc.ErrorEvent)
		if !ok {
			return
		}
		msg := strings.TrimSpace(errEvt.Message)
		if msg == "" {
			msg = "session error"
		}
		s.markAllClosed(msg)
	})
}

func (s *StreamService) unbindBus() {
	if s == nil || s.bus == nil {
		return
	}
	for _, entry := range s.busTokens {
		if entry.token == "" {
			continue
		}
		s.bus.Unsubscribe(entry.name, entry.token)
	}
	s.busTokens = nil
}

func (s *StreamService) handleFrame(frame sessionsvc.FrameEvent) {
	if len(frame.Payload) == 0 {
		return
	}
	switch frame.Payload[0] {
	case proto.KindData:
		s.handleData(frame)
	case proto.KindAck:
		s.handleAck(frame)
	default:
		return
	}
}

func (s *StreamService) handleData(frame sessionsvc.FrameEvent) {
	packet, err := parseDataPacket(frame.Payload)
	if err != nil {
		s.logWarn("stream data parse failed: %v", err)
		return
	}
	now := time.Now()
	ackSource := uint32(0)
	ackTarget := uint32(0)
	ackPosition := uint64(0)
	shouldAck := false

	s.mu.Lock()
	s.ensureStateMapsLocked()
	rt := s.ensureDeliveryLocked(packet.DeliveryID)
	rt.BytesIn += uint64(len(packet.Body))
	rt.FramesIn++
	rt.LastPosition = packet.Position
	rt.LastPtsMs = packet.PtsMs
	rt.LastFlags = packet.Flags
	rt.State = "active"
	rt.LastFrameAt = now
	rt.UpdatedAt = now
	snapshot := rt.snapshot()
	emitStats := rt.Kind != proto.StreamKindText && (rt.LastStatsEmit.IsZero() || now.Sub(rt.LastStatsEmit) >= streamStatsEvery)
	if emitStats {
		rt.LastStatsEmit = now
	}
	if delivery := s.consumerDeliveries[packet.DeliveryID]; delivery != nil &&
		delivery.State == localDeliveryStateActive &&
		frame.SourceID == delivery.Producer &&
		frame.TargetID == delivery.Consumer {
		nextPosition := nextExpectedPosition(delivery.UnitMode, packet.Position, len(packet.Body))
		if nextPosition >= delivery.ExpectedPosition {
			delivery.ExpectedPosition = nextPosition
			delivery.LastAckPosition = nextPosition
			delivery.LastActive = now
			rt.LastAckPos = nextPosition
			ackSource = delivery.Consumer
			ackTarget = delivery.Producer
			ackPosition = nextPosition
			shouldAck = true
		}
	}
	kind := rt.Kind
	s.mu.Unlock()

	if kind == proto.StreamKindText {
		text := decodeTextPayload(packet.Body)
		s.emitText(StreamTextEvent{
			DeliveryID: packet.DeliveryID,
			Kind:       kind,
			Text:       text,
			Position:   packet.Position,
			PtsMs:      packet.PtsMs,
			Flags:      packet.Flags,
			UpdatedAt:  now.Format(time.RFC3339Nano),
		})
	} else {
		s.writeMediaChunk(packet.DeliveryID, packet)
	}
	if emitStats {
		s.emitStats(StreamStatsEvent{
			DeliveryID:   snapshot.DeliveryID,
			Kind:         snapshot.Kind,
			BytesIn:      snapshot.BytesIn,
			FramesIn:     snapshot.FramesIn,
			LastPosition: snapshot.LastPosition,
			LastPtsMs:    snapshot.LastPtsMs,
			LastAckPos:   snapshot.LastAckPos,
			LastFlags:    snapshot.LastFlags,
			UpdatedAt:    snapshot.UpdatedAt,
		})
	}
	if shouldAck {
		if err := s.sendAck(packet.DeliveryID, ackSource, ackTarget, ackPosition); err != nil {
			s.logWarn("stream ack send failed: %v", err)
		}
	}
}

func (s *StreamService) handleAck(frame sessionsvc.FrameEvent) {
	packet, err := parseAckPacket(frame.Payload)
	if err != nil {
		s.logWarn("stream ack parse failed: %v", err)
		return
	}
	now := time.Now()
	s.mu.Lock()
	s.ensureStateMapsLocked()
	rt := s.ensureDeliveryLocked(packet.DeliveryID)
	rt.LastAckPos = packet.Position
	rt.LastFlags = packet.Flags
	rt.State = "active"
	rt.UpdatedAt = now
	if delivery := s.producerDeliveries[packet.DeliveryID]; delivery != nil &&
		delivery.State == localDeliveryStateActive &&
		frame.SourceID == delivery.Consumer &&
		frame.TargetID == delivery.Producer {
		if packet.Position > delivery.AckedPosition {
			delivery.AckedPosition = packet.Position
		}
		delivery.LastActive = now
	}
	snapshot := rt.snapshot()
	emitStats := rt.LastStatsEmit.IsZero() || now.Sub(rt.LastStatsEmit) >= streamStatsEvery
	if emitStats {
		rt.LastStatsEmit = now
	}
	s.mu.Unlock()
	if emitStats {
		s.emitStats(StreamStatsEvent{
			DeliveryID:   snapshot.DeliveryID,
			Kind:         snapshot.Kind,
			BytesIn:      snapshot.BytesIn,
			FramesIn:     snapshot.FramesIn,
			LastPosition: snapshot.LastPosition,
			LastPtsMs:    snapshot.LastPtsMs,
			LastAckPos:   snapshot.LastAckPos,
			LastFlags:    snapshot.LastFlags,
			UpdatedAt:    snapshot.UpdatedAt,
		})
	}
}

func (s *StreamService) trackAcceptedDelivery(deliveryID string, source *proto.SourceDescriptor, endpoint *proto.ConsumerDescriptor, producer, consumer uint32, consumerID string, accepted bool) {
	deliveryID = strings.TrimSpace(deliveryID)
	if !accepted || deliveryID == "" {
		return
	}
	now := time.Now()
	s.mu.Lock()
	rt := s.ensureDeliveryLocked(deliveryID)
	if source != nil {
		rt.SourceID = strings.TrimSpace(source.SourceID)
		rt.Kind = normalizeRequiredKind(source.Kind)
		rt.ContentType = strings.TrimSpace(source.ContentType)
		rt.Mode = strings.TrimSpace(source.Mode)
		rt.UnitMode = strings.TrimSpace(source.UnitMode)
		if source.Producer != 0 {
			rt.Producer = source.Producer
		}
	}
	if endpoint != nil {
		rt.ConsumerID = strings.TrimSpace(endpoint.ConsumerID)
		if rt.Kind == "" {
			rt.Kind = normalizeRequiredKind(endpoint.Kind)
		}
		if rt.ContentType == "" {
			rt.ContentType = strings.TrimSpace(endpoint.ContentType)
		}
		if endpoint.Consumer != 0 {
			rt.Consumer = endpoint.Consumer
		}
	}
	if producer != 0 {
		rt.Producer = producer
	}
	if consumer != 0 {
		rt.Consumer = consumer
	}
	if strings.TrimSpace(consumerID) != "" {
		rt.ConsumerID = strings.TrimSpace(consumerID)
	}
	if rt.Kind == "" {
		rt.Kind = proto.StreamKindCustom
	}
	rt.State = "active"
	rt.LastError = ""
	rt.UpdatedAt = now
	snapshot := rt.snapshot()
	s.mu.Unlock()
	s.emitDelivery(snapshot)
}

func (s *StreamService) removeDelivery(deliveryID, state string) {
	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return
	}
	s.mu.Lock()
	rt := s.deliveries[deliveryID]
	if rt == nil {
		s.mu.Unlock()
		return
	}
	delete(s.deliveries, deliveryID)
	now := time.Now()
	rt.State = strings.TrimSpace(state)
	if rt.State == "" {
		rt.State = "closed"
	}
	rt.UpdatedAt = now
	snapshot := rt.snapshot()
	s.mu.Unlock()
	s.emitDelivery(snapshot)
	s.closeMediaRuntime(deliveryID, snapshot.LastError)
}

func (s *StreamService) removeDeliveriesBySource(sourceID string) {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return
	}
	s.removeMatchingDeliveries(func(item *deliveryRuntime) bool {
		return item != nil && item.SourceID == sourceID
	})
}

func (s *StreamService) removeDeliveriesByConsumer(consumerID string) {
	consumerID = strings.TrimSpace(consumerID)
	if consumerID == "" {
		return
	}
	s.removeMatchingDeliveries(func(item *deliveryRuntime) bool {
		return item != nil && item.ConsumerID == consumerID
	})
}

func (s *StreamService) removeMatchingDeliveries(match func(item *deliveryRuntime) bool) {
	if match == nil {
		return
	}
	s.mu.Lock()
	closed := make([]StreamDeliveryEvent, 0)
	closedIDs := make([]string, 0)
	now := time.Now()
	for id, item := range s.deliveries {
		if !match(item) {
			continue
		}
		delete(s.deliveries, id)
		item.State = "closed"
		item.UpdatedAt = now
		closed = append(closed, item.snapshot())
		closedIDs = append(closedIDs, id)
	}
	s.mu.Unlock()
	for _, evt := range closed {
		s.emitDelivery(evt)
	}
	for _, deliveryID := range closedIDs {
		s.closeMediaRuntime(deliveryID, "delivery removed")
	}
}

func (s *StreamService) markAllClosed(reason string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "disconnected"
	}
	s.mu.Lock()
	s.ensureStateMapsLocked()
	evts := make([]StreamDeliveryEvent, 0, len(s.deliveries))
	senders := make([]*fileDeliverySender, 0, len(s.fileSenders))
	for _, sender := range s.fileSenders {
		if sender != nil {
			senders = append(senders, sender)
		}
	}
	mediaIDs := make([]string, 0, len(s.media))
	for deliveryID := range s.media {
		mediaIDs = append(mediaIDs, deliveryID)
	}
	now := time.Now()
	for _, item := range s.deliveries {
		if item == nil {
			continue
		}
		item.State = "closed"
		item.LastError = reason
		item.UpdatedAt = now
		evts = append(evts, item.snapshot())
	}
	s.deliveries = make(map[string]*deliveryRuntime)
	s.sources = make(map[string]proto.SourceDescriptor)
	s.consumers = make(map[string]proto.ConsumerDescriptor)
	s.producerDeliveries = make(map[string]*localProducerDelivery)
	s.consumerDeliveries = make(map[string]*localConsumerDelivery)
	s.fileSenders = make(map[string]*fileDeliverySender)
	s.mu.Unlock()
	for _, sender := range senders {
		sender.cancel()
	}
	for _, evt := range evts {
		s.emitDelivery(evt)
	}
	for _, deliveryID := range mediaIDs {
		s.closeMediaRuntime(deliveryID, reason)
	}
}

func (s *StreamService) ensureDeliveryLocked(deliveryID string) *deliveryRuntime {
	s.ensureStateMapsLocked()
	rt := s.deliveries[deliveryID]
	if rt != nil {
		return rt
	}
	rt = &deliveryRuntime{
		DeliveryID: deliveryID,
		Kind:       proto.StreamKindCustom,
		State:      "observed",
		UpdatedAt:  time.Now(),
	}
	s.deliveries[deliveryID] = rt
	return rt
}

func (s *StreamService) emitDelivery(evt StreamDeliveryEvent) {
	if s == nil || s.bus == nil {
		return
	}
	_ = s.bus.Publish(context.Background(), EventStreamDelivery, evt, nil)
}

func (s *StreamService) emitText(evt StreamTextEvent) {
	if s == nil || s.bus == nil {
		return
	}
	_ = s.bus.Publish(context.Background(), EventStreamText, evt, nil)
}

func (s *StreamService) emitStats(evt StreamStatsEvent) {
	if s == nil || s.bus == nil {
		return
	}
	_ = s.bus.Publish(context.Background(), EventStreamStats, evt, nil)
}

func (s *StreamService) emitMedia(evt StreamMediaEvent) {
	if s == nil || s.bus == nil {
		return
	}
	_ = s.bus.Publish(context.Background(), EventStreamMedia, evt, nil)
}

func (s *StreamService) logWarn(format string, args ...any) {
	if s == nil || s.logs == nil {
		return
	}
	s.logs.Appendf("warn", format, args...)
}

type dataPacket struct {
	DeliveryID string
	Flags      uint8
	Position   uint64
	PtsMs      uint64
	Body       []byte
}

func parseDataPacket(payload []byte) (dataPacket, error) {
	if len(payload) < streamFrameHeaderLen {
		return dataPacket{}, errors.New("data payload too short")
	}
	if payload[0] != proto.KindData {
		return dataPacket{}, errors.New("not a data payload")
	}
	if payload[1] != proto.HeaderVersionV1 {
		return dataPacket{}, errors.New("unsupported data version")
	}
	id, err := bytesToUUID(payload[3:19])
	if err != nil {
		return dataPacket{}, err
	}
	return dataPacket{
		DeliveryID: id,
		Flags:      payload[2],
		Position:   binary.BigEndian.Uint64(payload[19:27]),
		PtsMs:      binary.BigEndian.Uint64(payload[27:35]),
		Body:       append([]byte(nil), payload[35:]...),
	}, nil
}

type ackPacket struct {
	DeliveryID string
	Flags      uint8
	Position   uint64
}

func parseAckPacket(payload []byte) (ackPacket, error) {
	if len(payload) < streamAckHeaderLen {
		return ackPacket{}, errors.New("ack payload too short")
	}
	if payload[0] != proto.KindAck {
		return ackPacket{}, errors.New("not an ack payload")
	}
	if payload[1] != proto.HeaderVersionV1 {
		return ackPacket{}, errors.New("unsupported ack version")
	}
	id, err := bytesToUUID(payload[3:19])
	if err != nil {
		return ackPacket{}, err
	}
	return ackPacket{
		DeliveryID: id,
		Flags:      payload[2],
		Position:   binary.BigEndian.Uint64(payload[19:27]),
	}, nil
}

func bytesToUUID(raw []byte) (string, error) {
	if len(raw) != 16 {
		return "", errors.New("invalid uuid length")
	}
	var id uuid.UUID
	copy(id[:], raw)
	return id.String(), nil
}

func decodeTextPayload(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	if len(body) > streamTextLimit {
		body = body[:streamTextLimit]
	}
	if utf8.Valid(body) {
		return string(body)
	}
	return hex.EncodeToString(body)
}

func (s *StreamService) sendAck(deliveryID string, sourceID, targetID uint32, position uint64) error {
	if s == nil || s.session == nil {
		return errors.New("session service not initialized")
	}
	if sourceID == 0 || targetID == 0 {
		return errors.New("ack route is incomplete")
	}
	id, err := uuid.Parse(strings.TrimSpace(deliveryID))
	if err != nil {
		return err
	}
	payload := make([]byte, streamAckHeaderLen)
	payload[0] = proto.KindAck
	payload[1] = proto.HeaderVersionV1
	payload[2] = 0
	copy(payload[3:19], id[:])
	binary.BigEndian.PutUint64(payload[19:27], position)
	binary.BigEndian.PutUint32(payload[27:31], 0)
	binary.BigEndian.PutUint32(payload[31:35], 0)

	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorMsg).
		WithSubProto(proto.SubProtoStream).
		WithSourceID(sourceID).
		WithTargetID(targetID).
		WithTimestamp(uint32(time.Now().Unix()))
	return s.session.Send(hdr, payload)
}
