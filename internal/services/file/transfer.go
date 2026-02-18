package file

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub-core/eventbus"
	"github.com/yttydcs/myflowhub-core/header"
	protocol "github.com/yttydcs/myflowhub-proto/protocol/file"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
	"github.com/yttydcs/myflowhub-win/internal/services/transport"
)

const (
	taskStatusWaitingResponse = "waiting_response"
	taskStatusWaitingRemote   = "waiting_remote"
	taskStatusWaitingConfirm  = "waiting_confirm"
	taskStatusPreparing       = "preparing"
	taskStatusHashing         = "hashing"
	taskStatusSending         = "sending"
	taskStatusReceiving       = "receiving"
	taskStatusWaitingAck      = "waiting_ack"
	taskStatusCompleted       = "completed"
	taskStatusFailed          = "failed"
	taskStatusCanceled        = "canceled"
	taskStatusRejected        = "rejected"
)

type fileState struct {
	mu sync.Mutex

	recv         map[[16]byte]*fileRecvSession
	send         map[[16]byte]*fileSendSession
	pendingPull  map[filePullKey]*fileTask
	pendingOffer map[string]*pendingOffer
	tasks        []*fileTask
	taskBySid    map[[16]byte]*fileTask

	lastEmit time.Time

	janitorStarted bool
	janitorCancel  context.CancelFunc
}

type filePullKey struct {
	provider uint32
	dir      string
	name     string
}

type pendingOffer struct {
	provider uint32
	req      protocol.WriteReq
	received time.Time
}

type fileTask struct {
	mu sync.Mutex

	taskID [16]byte

	sessionID    [16]byte
	hasSessionID bool

	createdAt time.Time
	updatedAt time.Time

	op        string
	direction string

	provider uint32
	consumer uint32
	peer     uint32

	dir    string
	name   string
	size   uint64
	sha256 string

	wantHash bool

	localDir  string
	localName string
	localPath string
	filePath  string

	status    string
	lastError string

	sentBytes  uint64
	ackedBytes uint64
	doneBytes  uint64
}

type fileRecvSession struct {
	id       [16]byte
	provider uint32
	consumer uint32

	dir       string
	name      string
	finalPath string
	partPath  string

	size      uint64
	sha256Hex string
	overwrite bool

	mu              sync.Mutex
	file            *os.File
	hasher          hash.Hash
	expectedOffset  uint64
	pending         map[uint64][]byte
	pendingBytes    uint64
	maxPendingBytes uint64
	finSeen         bool
	lastActive      time.Time
	lastAckOffset   uint64
	lastAckTime     time.Time

	task *fileTask
}

type fileSendSession struct {
	id       [16]byte
	provider uint32
	consumer uint32

	dir      string
	name     string
	filePath string

	size      uint64
	sha256Hex string
	startFrom uint64

	mu         sync.Mutex
	ackedUntil uint64
	lastActive time.Time
	sentEOF    bool
	cancel     context.CancelFunc

	task *fileTask
}

type busToken struct {
	name  string
	token string
}

func newFileState() *fileState {
	return &fileState{
		recv:         make(map[[16]byte]*fileRecvSession),
		send:         make(map[[16]byte]*fileSendSession),
		pendingPull:  make(map[filePullKey]*fileTask),
		pendingOffer: make(map[string]*pendingOffer),
		taskBySid:    make(map[[16]byte]*fileTask),
	}
}

func (s *FileService) bindBus() {
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
		if frame.SubProto != protocol.SubProtoFile {
			return
		}
		s.handleFrame(frame)
	})
	addToken(sessionsvc.EventState, func(data any) {
		state, ok := data.(sessionsvc.StateEvent)
		if !ok {
			return
		}
		if !state.Connected {
			s.fileOnDisconnect(errors.New("disconnected"))
		}
	})
	addToken(sessionsvc.EventError, func(data any) {
		errEvt, ok := data.(sessionsvc.ErrorEvent)
		if !ok {
			return
		}
		s.fileOnDisconnect(errors.New(errEvt.Message))
	})
}

func (s *FileService) unbindBus() {
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

func (s *FileService) stopJanitor() {
	if s == nil || s.state == nil {
		return
	}
	s.state.mu.Lock()
	cancel := s.state.janitorCancel
	s.state.janitorCancel = nil
	s.state.janitorStarted = false
	s.state.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *FileService) handleFrame(frame sessionsvc.FrameEvent) {
	if len(frame.Payload) == 0 {
		return
	}
	switch frame.Payload[0] {
	case protocol.KindCtrl:
		s.handleCtrl(frame)
		s.fileStartJanitor()
	case protocol.KindData:
		s.handleData(frame)
	case protocol.KindAck:
		s.handleAck(frame)
	default:
		return
	}
}

func (s *FileService) handleCtrl(frame sessionsvc.FrameEvent) {
	if len(frame.Payload) < 2 {
		return
	}
	var msg protocol.Message
	if err := json.Unmarshal(frame.Payload[1:], &msg); err != nil {
		return
	}
	action := strings.ToLower(strings.TrimSpace(msg.Action))
	switch action {
	case protocol.ActionRead:
		var req protocol.ReadReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			_ = s.sendReadResp(frame.TargetID, frame.SourceID, protocol.ReadResp{Code: 400, Msg: "invalid read", Op: strings.TrimSpace(req.Op)})
			return
		}
		s.handleReadRequest(frame, req)
	case protocol.ActionWrite:
		var req protocol.WriteReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			_ = s.sendWriteResp(frame.TargetID, frame.SourceID, protocol.WriteResp{Code: 400, Msg: "invalid write", Op: strings.TrimSpace(req.Op)})
			return
		}
		s.handleWriteRequest(frame, req)
	case protocol.ActionReadResp:
		var resp protocol.ReadResp
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			return
		}
		s.handleReadResp(frame, resp)
	case protocol.ActionWriteResp:
		var resp protocol.WriteResp
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			return
		}
		s.handleWriteResp(frame, resp)
	default:
		return
	}
}

func (s *FileService) handleReadRequest(frame sessionsvc.FrameEvent, req protocol.ReadReq) {
	op := strings.ToLower(strings.TrimSpace(req.Op))
	if op != protocol.OpPull && op != protocol.OpList && op != protocol.OpReadText {
		_ = s.sendReadResp(frame.TargetID, frame.SourceID, protocol.ReadResp{Code: 400, Msg: "invalid op", Op: op})
		return
	}
	localNode := s.localNodeFromFrame(frame)
	if localNode == 0 {
		return
	}
	requester := frame.SourceID
	target := req.Target
	if op == protocol.OpList && target == 0 {
		target = localNode
	}
	if target == 0 {
		_ = s.sendReadResp(localNode, requester, protocol.ReadResp{Code: 400, Msg: "target required", Op: op})
		return
	}
	if target != localNode {
		return
	}
	switch op {
	case protocol.OpPull:
		s.handlePullAsProvider(requester, req, localNode)
	case protocol.OpList:
		s.handleListLocal(requester, req)
	case protocol.OpReadText:
		s.handleReadTextLocal(requester, req)
	}
}

func (s *FileService) handleWriteRequest(frame sessionsvc.FrameEvent, req protocol.WriteReq) {
	op := strings.ToLower(strings.TrimSpace(req.Op))
	if op != protocol.OpOffer {
		_ = s.sendWriteResp(frame.TargetID, frame.SourceID, protocol.WriteResp{Code: 400, Msg: "invalid op", Op: op})
		return
	}
	localNode := s.localNodeFromFrame(frame)
	if localNode == 0 {
		return
	}
	provider := frame.SourceID
	if req.Target == 0 || req.Target != localNode {
		_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{Code: 400, Msg: "target mismatch", Op: op, SessionID: strings.TrimSpace(req.SessionID)})
		return
	}
	sid, ok := fileParseUUID(req.SessionID)
	if !ok {
		_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{Code: 400, Msg: "invalid session", Op: op, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	cfg := s.fileConfig()
	if cfg.MaxSizeBytes > 0 && req.Size > cfg.MaxSizeBytes {
		_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{Code: 413, Msg: "too large", Op: op, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	if s.fileTotalSessions() >= cfg.MaxConcurrent {
		_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{Code: 429, Msg: "too many sessions", Op: op, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	dir := strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/")
	name := strings.TrimSpace(req.Name)
	if _, err := fileSanitizeDir(dir); err != nil {
		_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{Code: 400, Msg: "invalid dir", Op: op, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	if _, err := fileSanitizeName(name); err != nil {
		_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{Code: 400, Msg: "invalid name", Op: op, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	if s.fileGetRecvSession(sid) != nil {
		return
	}

	task := s.ensureOfferTask(sid, provider, localNode, dir, name, req.Size, req.Sha256)
	s.setTaskStatus(task, taskStatusWaitingConfirm, "")

	if cfg.AutoAccept {
		s.acceptOffer(provider, req, dir)
		return
	}

	s.state.mu.Lock()
	s.state.pendingOffer[strings.TrimSpace(req.SessionID)] = &pendingOffer{
		provider: provider,
		req:      req,
		received: time.Now(),
	}
	s.state.mu.Unlock()

	s.emitOffer(FileOfferEvent{
		SessionID:  strings.TrimSpace(req.SessionID),
		Provider:   provider,
		Consumer:   localNode,
		Dir:        dir,
		Name:       name,
		Size:       req.Size,
		Sha256:     strings.TrimSpace(req.Sha256),
		SuggestDir: dir,
	})
}

func (s *FileService) handlePullAsProvider(consumer uint32, req protocol.ReadReq, provider uint32) {
	cfg := s.fileConfig()
	dir := strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/")
	name := strings.TrimSpace(req.Name)
	finalPath, _, err := fileResolvePaths(cfg.BaseDir, dir, name)
	if err != nil {
		_ = s.sendReadResp(provider, consumer, protocol.ReadResp{Code: 400, Msg: "invalid path", Op: protocol.OpPull, Provider: provider, Consumer: consumer, Dir: dir, Name: name})
		return
	}
	info, err := os.Stat(finalPath)
	if err != nil || info == nil || info.IsDir() {
		_ = s.sendReadResp(provider, consumer, protocol.ReadResp{Code: 404, Msg: "not found", Op: protocol.OpPull, Provider: provider, Consumer: consumer, Dir: dir, Name: name})
		return
	}
	size := uint64(info.Size())
	if cfg.MaxSizeBytes > 0 && size > cfg.MaxSizeBytes {
		_ = s.sendReadResp(provider, consumer, protocol.ReadResp{Code: 413, Msg: "too large", Op: protocol.OpPull, Provider: provider, Consumer: consumer, Dir: dir, Name: name, Size: size})
		return
	}
	if s.fileTotalSessions() >= cfg.MaxConcurrent {
		_ = s.sendReadResp(provider, consumer, protocol.ReadResp{Code: 429, Msg: "too many sessions", Op: protocol.OpPull, Provider: provider, Consumer: consumer, Dir: dir, Name: name, Size: size})
		return
	}
	wantHash := cfg.WantSHA256
	if req.WantHash != nil {
		wantHash = *req.WantHash
	}
	shaHex := ""
	if wantHash {
		if sh, err := fileHashSHA256(finalPath); err == nil {
			shaHex = sh
		}
	}
	startFrom := uint64(0)
	if req.ResumeFrom > 0 && req.ResumeFrom < size {
		startFrom = req.ResumeFrom
	}
	sid, err := fileNewUUID()
	if err != nil {
		_ = s.sendReadResp(provider, consumer, protocol.ReadResp{Code: 500, Msg: "uuid failed", Op: protocol.OpPull})
		return
	}
	taskID, _ := fileNewUUID()
	task := &fileTask{
		taskID:       taskID,
		sessionID:    sid,
		hasSessionID: true,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
		op:           protocol.OpPull,
		direction:    "upload",
		provider:     provider,
		consumer:     consumer,
		peer:         consumer,
		dir:          dir,
		name:         name,
		size:         size,
		sha256:       shaHex,
		wantHash:     wantHash,
		filePath:     finalPath,
		status:       taskStatusSending,
	}
	sess := &fileSendSession{
		id:         sid,
		provider:   provider,
		consumer:   consumer,
		dir:        dir,
		name:       name,
		filePath:   finalPath,
		size:       size,
		sha256Hex:  shaHex,
		startFrom:  startFrom,
		lastActive: time.Now(),
		task:       task,
	}
	s.fileAddTask(task)
	s.fileAddSendSession(sess)

	_ = s.sendReadResp(provider, consumer, protocol.ReadResp{
		Code:      1,
		Msg:       "ok",
		Op:        protocol.OpPull,
		SessionID: fileUUIDToString(sid),
		Provider:  provider,
		Consumer:  consumer,
		Dir:       dir,
		Name:      name,
		Size:      size,
		Sha256:    shaHex,
		StartFrom: startFrom,
		Chunk:     uint32(cfg.ChunkBytes),
	})
	go s.fileSendData(sid)
}

func (s *FileService) handleListLocal(requester uint32, req protocol.ReadReq) {
	dir := strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/")
	dir, err := fileSanitizeDir(dir)
	if err != nil {
		_ = s.sendReadResp(s.localNodeFallback(requester), requester, protocol.ReadResp{Code: 400, Msg: "invalid dir", Op: protocol.OpList, Dir: dir})
		return
	}
	dirs, files, err := s.localList(dir)
	if err != nil {
		_ = s.sendReadResp(s.localNodeFallback(requester), requester, protocol.ReadResp{Code: 404, Msg: "not found", Op: protocol.OpList, Dir: dir, Files: []string{}, Dirs: []string{}})
		return
	}
	_ = s.sendReadResp(s.localNodeFallback(requester), requester, protocol.ReadResp{Code: 1, Msg: "ok", Op: protocol.OpList, Dir: dir, Files: files, Dirs: dirs})
}

func (s *FileService) handleReadTextLocal(requester uint32, req protocol.ReadReq) {
	dir := strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/")
	name := strings.TrimSpace(req.Name)
	text, truncated, size, err := s.localReadText(dir, name, int(req.MaxBytes))
	if err != nil {
		_ = s.sendReadResp(s.localNodeFallback(requester), requester, protocol.ReadResp{Code: 415, Msg: err.Error(), Op: protocol.OpReadText, Dir: dir, Name: name, Size: size})
		return
	}
	_ = s.sendReadResp(s.localNodeFallback(requester), requester, protocol.ReadResp{Code: 1, Msg: "ok", Op: protocol.OpReadText, Dir: dir, Name: name, Size: size, Text: text, Truncated: truncated})
}

func (s *FileService) handleReadResp(frame sessionsvc.FrameEvent, resp protocol.ReadResp) {
	op := strings.ToLower(strings.TrimSpace(resp.Op))
	switch op {
	case protocol.OpList:
		nodeID := frame.SourceID
		if resp.Provider != 0 {
			nodeID = resp.Provider
		}
		s.emitList(FileListEvent{
			NodeID: nodeID,
			Dir:    strings.ReplaceAll(strings.TrimSpace(resp.Dir), "\\", "/"),
			Code:   resp.Code,
			Msg:    strings.TrimSpace(resp.Msg),
			Dirs:   resp.Dirs,
			Files:  resp.Files,
		})
		return
	case protocol.OpReadText:
		nodeID := frame.SourceID
		if resp.Provider != 0 {
			nodeID = resp.Provider
		}
		s.emitText(FileTextEvent{
			NodeID:    nodeID,
			Dir:       strings.ReplaceAll(strings.TrimSpace(resp.Dir), "\\", "/"),
			Name:      strings.TrimSpace(resp.Name),
			Code:      resp.Code,
			Msg:       strings.TrimSpace(resp.Msg),
			Size:      resp.Size,
			Text:      resp.Text,
			Truncated: resp.Truncated,
		})
		return
	case protocol.OpPull:
	default:
		return
	}

	provider := frame.SourceID
	consumer := s.localNodeFromFrame(frame)
	if consumer == 0 {
		return
	}
	if resp.Consumer != 0 && resp.Consumer != consumer {
		return
	}
	if resp.Provider != 0 && resp.Provider != provider {
		return
	}
	if resp.Code != 1 {
		s.failPendingPull(provider, resp.Dir, resp.Name, fmt.Sprintf("%d %s", resp.Code, strings.TrimSpace(resp.Msg)))
		return
	}
	if strings.TrimSpace(resp.SessionID) == "" {
		return
	}
	sid, ok := fileParseUUID(resp.SessionID)
	if !ok {
		return
	}

	dir := strings.ReplaceAll(strings.TrimSpace(resp.Dir), "\\", "/")
	name := strings.TrimSpace(resp.Name)
	task := s.takePendingPull(provider, dir, name)
	if task == nil {
		taskID, _ := fileNewUUID()
		task = &fileTask{
			taskID:    taskID,
			createdAt: time.Now(),
			updatedAt: time.Now(),
			op:        protocol.OpPull,
			direction: "download",
			provider:  provider,
			consumer:  consumer,
			peer:      provider,
			dir:       dir,
			name:      name,
			wantHash:  s.fileConfig().WantSHA256,
			status:    taskStatusReceiving,
		}
		s.fileAddTask(task)
	}

	cfg := s.fileConfig()
	task.mu.Lock()
	localDir := strings.ReplaceAll(strings.TrimSpace(task.localDir), "\\", "/")
	localName := strings.TrimSpace(task.localName)
	wantHash := task.wantHash
	task.mu.Unlock()
	if localDir == "" {
		localDir = dir
	}
	saveName := name
	if localName != "" {
		saveName = localName
	}
	finalPath, partPath, err := fileResolvePaths(cfg.BaseDir, localDir, saveName)
	if err != nil {
		s.setTaskStatus(task, taskStatusFailed, "invalid save path")
		return
	}
	if cfg.MaxSizeBytes > 0 && resp.Size > cfg.MaxSizeBytes {
		s.setTaskStatus(task, taskStatusFailed, "too large")
		return
	}
	if s.fileTotalSessions() >= cfg.MaxConcurrent {
		s.setTaskStatus(task, taskStatusFailed, "too many sessions")
		return
	}
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		s.setTaskStatus(task, taskStatusFailed, "mkdir failed")
		return
	}

	startFrom := resp.StartFrom
	if startFrom > resp.Size {
		startFrom = 0
	}
	f, err := os.OpenFile(partPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		s.setTaskStatus(task, taskStatusFailed, "open failed")
		return
	}
	if err := f.Truncate(int64(startFrom)); err != nil {
		_ = f.Close()
		s.setTaskStatus(task, taskStatusFailed, "truncate failed")
		return
	}
	shaHex := strings.TrimSpace(resp.Sha256)
	var hasher hash.Hash
	if wantHash && shaHex != "" {
		hasher = sha256.New()
	}
	if hasher != nil && startFrom > 0 {
		if _, err := f.Seek(0, io.SeekStart); err == nil {
			buf := make([]byte, 256*1024)
			remain := startFrom
			for remain > 0 {
				n := int64(len(buf))
				if remain < uint64(len(buf)) {
					n = int64(remain)
				}
				readN, rerr := io.ReadFull(f, buf[:n])
				if readN > 0 {
					_, _ = hasher.Write(buf[:readN])
					remain -= uint64(readN)
				}
				if rerr != nil {
					break
				}
			}
		}
	}
	if _, err := f.Seek(int64(startFrom), io.SeekStart); err != nil {
		_ = f.Close()
		s.setTaskStatus(task, taskStatusFailed, "seek failed")
		return
	}

	task.mu.Lock()
	task.sessionID = sid
	task.hasSessionID = true
	task.size = resp.Size
	task.sha256 = shaHex
	task.localDir = localDir
	task.localName = saveName
	task.localPath = finalPath
	task.status = taskStatusReceiving
	task.doneBytes = startFrom
	task.updatedAt = time.Now()
	task.mu.Unlock()

	s.state.mu.Lock()
	s.state.taskBySid[sid] = task
	s.state.mu.Unlock()

	sess := &fileRecvSession{
		id:              sid,
		provider:        provider,
		consumer:        consumer,
		dir:             dir,
		name:            name,
		finalPath:       finalPath,
		partPath:        partPath,
		size:            resp.Size,
		sha256Hex:       shaHex,
		overwrite:       true,
		file:            f,
		hasher:          hasher,
		expectedOffset:  startFrom,
		pending:         make(map[uint64][]byte),
		maxPendingBytes: uint64(cfg.ChunkBytes) * 8,
		lastActive:      time.Now(),
		task:            task,
	}
	s.fileAddRecvSession(sess)
	s.emitTasksThrottled()
}

func (s *FileService) handleWriteResp(_ sessionsvc.FrameEvent, resp protocol.WriteResp) {
	if strings.ToLower(strings.TrimSpace(resp.Op)) != protocol.OpOffer {
		return
	}
	sid, ok := fileParseUUID(resp.SessionID)
	if !ok {
		return
	}
	sess := s.fileGetSendSession(sid)
	if sess == nil {
		return
	}

	if resp.Code != 1 || !resp.Accept {
		if sess.task != nil {
			sess.task.mu.Lock()
			sess.task.status = taskStatusFailed
			if resp.Code == 403 {
				sess.task.status = taskStatusRejected
			}
			sess.task.lastError = strings.TrimSpace(resp.Msg)
			sess.task.updatedAt = time.Now()
			sess.task.mu.Unlock()
		}
		s.fileRemoveSendSession(sid)
		s.emitTasksThrottled()
		return
	}

	sess.mu.Lock()
	sess.startFrom = resp.ResumeFrom
	if sess.task != nil {
		sess.task.mu.Lock()
		sess.task.status = taskStatusSending
		sess.task.updatedAt = time.Now()
		sess.task.mu.Unlock()
	}
	sess.mu.Unlock()
	s.emitTasksThrottled()

	if resp.ResumeFrom >= sess.size {
		sess.mu.Lock()
		if sess.task != nil {
			sess.task.mu.Lock()
			sess.task.status = taskStatusCompleted
			sess.task.ackedBytes = sess.size
			sess.task.updatedAt = time.Now()
			sess.task.mu.Unlock()
		}
		sess.mu.Unlock()
		s.fileRemoveSendSession(sid)
		s.emitTasksThrottled()
		return
	}
	go s.fileSendData(sid)
}

func (s *FileService) handleData(frame sessionsvc.FrameEvent) {
	kind, bh, body, ok := fileDecodeBinHeaderV1(frame.Payload)
	if !ok || kind != protocol.KindData || bh.Ver != fileBinVerV1 {
		return
	}
	sess := s.fileGetRecvSession(bh.SessionID)
	if sess == nil {
		return
	}
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if frame.TargetID != sess.consumer || frame.SourceID != sess.provider {
		return
	}
	if sess.file == nil {
		return
	}
	if sess.expectedOffset > sess.size {
		return
	}

	fin := (bh.Flags & fileBinFlagFIN) != 0
	offset := bh.Offset
	if offset < sess.expectedOffset {
		s.fileMaybeAckLocked(sess, false)
		return
	}
	if offset > sess.expectedOffset {
		if sess.pendingBytes+uint64(len(body)) > sess.maxPendingBytes {
			s.fileMaybeAckLocked(sess, false)
			return
		}
		if _, ok := sess.pending[offset]; !ok {
			cp := make([]byte, len(body))
			copy(cp, body)
			sess.pending[offset] = cp
			sess.pendingBytes += uint64(len(body))
		}
		s.fileMaybeAckLocked(sess, false)
		return
	}

	if len(body) > 0 {
		if sess.expectedOffset+uint64(len(body)) > sess.size {
			return
		}
		if _, err := sess.file.Write(body); err != nil {
			s.fileFailRecvLocked(sess, "write failed")
			return
		}
		if sess.hasher != nil {
			_, _ = sess.hasher.Write(body)
		}
		sess.expectedOffset += uint64(len(body))
	}
	if fin {
		sess.finSeen = true
	}
	sess.lastActive = time.Now()
	if sess.task != nil {
		sess.task.mu.Lock()
		sess.task.doneBytes = sess.expectedOffset
		sess.task.updatedAt = time.Now()
		sess.task.mu.Unlock()
	}

	for {
		next, ok := sess.pending[sess.expectedOffset]
		if !ok {
			break
		}
		delete(sess.pending, sess.expectedOffset)
		sess.pendingBytes -= uint64(len(next))
		if sess.expectedOffset+uint64(len(next)) > sess.size {
			s.fileFailRecvLocked(sess, "overflow")
			return
		}
		if _, err := sess.file.Write(next); err != nil {
			s.fileFailRecvLocked(sess, "write failed")
			return
		}
		if sess.hasher != nil {
			_, _ = sess.hasher.Write(next)
		}
		sess.expectedOffset += uint64(len(next))
		if sess.task != nil {
			sess.task.mu.Lock()
			sess.task.doneBytes = sess.expectedOffset
			sess.task.updatedAt = time.Now()
			sess.task.mu.Unlock()
		}
	}

	done := sess.finSeen && sess.expectedOffset == sess.size
	s.fileMaybeAckLocked(sess, done)
	if done {
		s.fileFinishRecvLocked(sess)
	}
	s.emitTasksThrottled()
}

func (s *FileService) handleAck(frame sessionsvc.FrameEvent) {
	kind, bh, _, ok := fileDecodeBinHeaderV1(frame.Payload)
	if !ok || kind != protocol.KindAck || bh.Ver != fileBinVerV1 {
		return
	}
	sess := s.fileGetSendSession(bh.SessionID)
	if sess == nil {
		return
	}
	done := false
	sess.mu.Lock()
	if frame.TargetID != sess.provider || frame.SourceID != sess.consumer {
		sess.mu.Unlock()
		return
	}
	if bh.Offset <= sess.size && bh.Offset > sess.ackedUntil {
		sess.ackedUntil = bh.Offset
	}
	sess.lastActive = time.Now()
	if sess.task != nil {
		sess.task.mu.Lock()
		sess.task.ackedBytes = sess.ackedUntil
		if sess.sentEOF && sess.ackedUntil == sess.size {
			sess.task.status = taskStatusCompleted
		}
		sess.task.updatedAt = time.Now()
		sess.task.mu.Unlock()
	}
	done = sess.sentEOF && sess.ackedUntil == sess.size
	id := sess.id
	sess.mu.Unlock()
	if done {
		s.fileRemoveSendSession(id)
	}
	s.emitTasksThrottled()
}

func (s *FileService) handleLocalRead(req protocol.ReadReq) error {
	op := strings.ToLower(strings.TrimSpace(req.Op))
	switch op {
	case protocol.OpList:
		dirs, files, err := s.localList(req.Dir)
		code := 1
		msg := "ok"
		if err != nil {
			code = 404
			msg = err.Error()
		}
		s.emitList(FileListEvent{
			NodeID: req.Target,
			Dir:    strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/"),
			Code:   code,
			Msg:    msg,
			Dirs:   dirs,
			Files:  files,
		})
		if code != 1 {
			msg = strings.TrimSpace(msg)
			if msg != "" {
				return fmt.Errorf("%s (code=%d)", msg, code)
			}
			return fmt.Errorf("file list failed (code=%d)", code)
		}
	case protocol.OpReadText:
		text, truncated, size, err := s.localReadText(req.Dir, req.Name, int(req.MaxBytes))
		code := 1
		msg := "ok"
		if err != nil {
			code = 415
			msg = err.Error()
		}
		s.emitText(FileTextEvent{
			NodeID:    req.Target,
			Dir:       strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/"),
			Name:      strings.TrimSpace(req.Name),
			Code:      code,
			Msg:       msg,
			Size:      size,
			Text:      text,
			Truncated: truncated,
		})
		if code != 1 {
			msg = strings.TrimSpace(msg)
			if msg != "" {
				return fmt.Errorf("%s (code=%d)", msg, code)
			}
			return fmt.Errorf("file read_text failed (code=%d)", code)
		}
	default:
		return errors.New("unsupported op")
	}
	return nil
}

func (s *FileService) StartPull(sourceID, hubID, provider uint32, dir, name, saveDir, saveName string, wantHash bool) error {
	if s.session == nil {
		return errors.New("session not initialized")
	}
	if provider == 0 {
		return errors.New("provider is required")
	}
	if sourceID == 0 || hubID == 0 {
		return errors.New("identity not set")
	}
	dir = strings.ReplaceAll(strings.TrimSpace(dir), "\\", "/")
	name = strings.TrimSpace(name)
	saveDir = strings.ReplaceAll(strings.TrimSpace(saveDir), "\\", "/")
	if _, err := fileSanitizeDir(dir); err != nil {
		return errors.New("invalid dir")
	}
	if _, err := fileSanitizeName(name); err != nil {
		return errors.New("invalid name")
	}
	if _, err := fileSanitizeDir(saveDir); err != nil {
		return errors.New("invalid save dir")
	}
	saveName = strings.TrimSpace(saveName)
	if saveName != "" {
		if _, err := fileSanitizeName(saveName); err != nil {
			return errors.New("invalid save name")
		}
	} else {
		saveName = name
	}

	cfg := s.fileConfig()
	finalPath, partPath, err := fileResolvePaths(cfg.BaseDir, saveDir, saveName)
	if err != nil {
		return errors.New("invalid save path")
	}
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return fmt.Errorf("mkdir failed: %w", err)
	}
	resumeFrom := uint64(0)
	if st, err := os.Stat(partPath); err == nil && st != nil && !st.IsDir() {
		if st.Size() > 0 {
			resumeFrom = uint64(st.Size())
		}
	}

	overwrite := true
	req := protocol.ReadReq{
		Op:         protocol.OpPull,
		Target:     provider,
		Dir:        dir,
		Name:       name,
		Overwrite:  &overwrite,
		ResumeFrom: resumeFrom,
		WantHash:   &wantHash,
	}

	taskID, _ := fileNewUUID()
	task := &fileTask{
		taskID:    taskID,
		createdAt: time.Now(),
		updatedAt: time.Now(),
		op:        protocol.OpPull,
		direction: "download",
		provider:  provider,
		consumer:  sourceID,
		peer:      provider,
		dir:       dir,
		name:      name,
		wantHash:  wantHash,
		localDir:  saveDir,
		localName: saveName,
		localPath: finalPath,
		status:    taskStatusWaitingResponse,
	}
	s.fileAddTask(task)
	s.state.mu.Lock()
	s.state.pendingPull[filePullKey{provider: provider, dir: strings.TrimSpace(dir), name: strings.TrimSpace(name)}] = task
	s.state.mu.Unlock()

	payload, err := transport.EncodeMessage(protocol.ActionRead, req)
	if err != nil {
		return err
	}
	if err := s.sendCtrl(context.Background(), sourceID, hubID, payload, "read", req.Op); err != nil {
		s.setTaskStatus(task, taskStatusFailed, err.Error())
		return err
	}
	return nil
}

func (s *FileService) StartOffer(sourceID, hubID, consumer uint32, dir, name string, wantHash bool) error {
	if s.session == nil {
		return errors.New("session not initialized")
	}
	if consumer == 0 {
		return errors.New("consumer is required")
	}
	if sourceID == 0 || hubID == 0 {
		return errors.New("identity not set")
	}
	cfg := s.fileConfig()
	dir = strings.ReplaceAll(strings.TrimSpace(dir), "\\", "/")
	name = strings.TrimSpace(name)
	if _, err := fileSanitizeDir(dir); err != nil {
		return errors.New("invalid dir")
	}
	if _, err := fileSanitizeName(name); err != nil {
		return errors.New("invalid name")
	}
	absBase, err := filepath.Abs(cfg.BaseDir)
	if err != nil {
		return errors.New("invalid base dir")
	}
	absFile := filepath.Join(absBase, filepath.FromSlash(strings.TrimSpace(dir)), strings.TrimSpace(name))
	absFile, err = filepath.Abs(absFile)
	if err != nil {
		return errors.New("invalid file path")
	}
	rel, err := filepath.Rel(absBase, absFile)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return errors.New("file must be under base dir")
	}
	info, err := os.Stat(absFile)
	if err != nil || info == nil || info.IsDir() {
		return errors.New("file not found")
	}
	size := uint64(info.Size())
	if cfg.MaxSizeBytes > 0 && size > cfg.MaxSizeBytes {
		return errors.New("file too large")
	}
	if s.fileTotalSessions() >= cfg.MaxConcurrent {
		return errors.New("too many sessions")
	}

	sid, err := fileNewUUID()
	if err != nil {
		return errors.New("session id failed")
	}
	taskID, _ := fileNewUUID()
	task := &fileTask{
		taskID:       taskID,
		sessionID:    sid,
		hasSessionID: true,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
		op:           protocol.OpOffer,
		direction:    "upload",
		provider:     sourceID,
		consumer:     consumer,
		peer:         consumer,
		dir:          dir,
		name:         name,
		size:         size,
		wantHash:     wantHash,
		filePath:     absFile,
		status:       taskStatusPreparing,
	}
	sess := &fileSendSession{
		id:         sid,
		provider:   sourceID,
		consumer:   consumer,
		dir:        dir,
		name:       name,
		filePath:   absFile,
		size:       size,
		startFrom:  0,
		lastActive: time.Now(),
		task:       task,
	}
	s.fileAddTask(task)
	s.fileAddSendSession(sess)

	go func() {
		shaHex := ""
		if wantHash {
			s.setTaskStatus(task, taskStatusHashing, "")
			if sh, err := fileHashSHA256(absFile); err == nil {
				shaHex = sh
			}
		}
		sess.mu.Lock()
		sess.sha256Hex = shaHex
		if sess.task != nil {
			sess.task.mu.Lock()
			sess.task.sha256 = shaHex
			sess.task.status = taskStatusWaitingRemote
			sess.task.updatedAt = time.Now()
			sess.task.mu.Unlock()
		}
		sess.mu.Unlock()
		s.emitTasksThrottled()

		overwrite := true
		req := protocol.WriteReq{
			Op:        protocol.OpOffer,
			Target:    consumer,
			SessionID: fileUUIDToString(sid),
			Dir:       dir,
			Name:      name,
			Size:      size,
			Sha256:    shaHex,
			Overwrite: &overwrite,
		}
		payload, err := transport.EncodeMessage(protocol.ActionWrite, req)
		if err != nil {
			s.setTaskStatus(task, taskStatusFailed, err.Error())
			s.fileRemoveSendSession(sid)
			return
		}
		if err := s.sendCtrl(context.Background(), sourceID, hubID, payload, "write", req.Op); err != nil {
			s.setTaskStatus(task, taskStatusFailed, err.Error())
			s.fileRemoveSendSession(sid)
		}
	}()
	return nil
}

func (s *FileService) ConfirmOffer(sessionID string, accept bool, saveDir string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("session id is required")
	}
	s.state.mu.Lock()
	pending := s.state.pendingOffer[sessionID]
	delete(s.state.pendingOffer, sessionID)
	s.state.mu.Unlock()
	if pending == nil {
		return errors.New("offer not found")
	}
	if !accept {
		s.rejectOffer(pending.provider, pending.req)
		return nil
	}
	return s.acceptOffer(pending.provider, pending.req, saveDir)
}

func (s *FileService) TasksSnapshot() ([]FileTaskView, error) {
	return s.snapshotTasks(), nil
}

func (s *FileService) RetryTask(taskID string) error {
	task := s.findTask(taskID)
	if task == nil {
		return errors.New("task not found")
	}
	task.mu.Lock()
	op := task.op
	dir := task.dir
	name := task.name
	provider := task.provider
	consumer := task.consumer
	localDir := task.localDir
	localName := task.localName
	filePath := task.filePath
	wantHash := task.wantHash
	task.mu.Unlock()

	s.mu.RLock()
	sourceID := s.localNode
	hubID := s.hubID
	s.mu.RUnlock()

	if op == protocol.OpPull {
		return s.StartPull(sourceID, hubID, provider, dir, name, localDir, localName, wantHash)
	}
	if op == protocol.OpOffer {
		if strings.TrimSpace(filePath) == "" {
			return errors.New("file path missing")
		}
		return s.StartOffer(sourceID, hubID, consumer, dir, name, wantHash)
	}
	return errors.New("unsupported task")
}

func (s *FileService) CancelTask(taskID string) error {
	task := s.findTask(taskID)
	if task == nil {
		return errors.New("task not found")
	}
	task.mu.Lock()
	sid := task.sessionID
	hasSid := task.hasSessionID
	task.status = taskStatusCanceled
	task.updatedAt = time.Now()
	task.mu.Unlock()

	if hasSid {
		s.fileRemoveRecvSession(sid)
		s.fileRemoveSendSession(sid)
		s.emitTasksThrottled()
		return nil
	}
	s.state.mu.Lock()
	delete(s.state.pendingPull, filePullKey{provider: task.provider, dir: strings.TrimSpace(task.dir), name: strings.TrimSpace(task.name)})
	s.state.mu.Unlock()
	s.emitTasksThrottled()
	return nil
}

func (s *FileService) OpenTaskFolder(taskID string) error {
	task := s.findTask(taskID)
	if task == nil {
		return errors.New("task not found")
	}
	task.mu.Lock()
	localPath := strings.TrimSpace(task.localPath)
	localDir := strings.TrimSpace(task.localDir)
	task.mu.Unlock()

	cfg := s.fileConfig()
	baseAbs, _ := filepath.Abs(cfg.BaseDir)
	folder := baseAbs
	if localPath != "" {
		folder = filepath.Dir(localPath)
	} else if localDir != "" {
		folder = filepath.Join(baseAbs, filepath.FromSlash(localDir))
	}
	return openFolder(folder)
}

func (s *FileService) localNodeFromFrame(frame sessionsvc.FrameEvent) uint32 {
	s.mu.RLock()
	local := s.localNode
	s.mu.RUnlock()
	if local == 0 {
		if frame.TargetID != 0 {
			s.mu.Lock()
			if s.localNode == 0 {
				s.localNode = frame.TargetID
			}
			local = s.localNode
			s.mu.Unlock()
		}
		if local == 0 {
			return frame.TargetID
		}
	}
	return local
}

func (s *FileService) localNodeFallback(fallback uint32) uint32 {
	s.mu.RLock()
	local := s.localNode
	s.mu.RUnlock()
	if local == 0 {
		return fallback
	}
	return local
}

func (s *FileService) fileStartJanitor() {
	if s == nil || s.state == nil {
		return
	}
	s.state.mu.Lock()
	if s.state.janitorStarted {
		s.state.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.state.janitorStarted = true
	s.state.janitorCancel = cancel
	s.state.mu.Unlock()

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.fileJanitorOnce()
			}
		}
	}()
}

func (s *FileService) fileJanitorOnce() {
	cfg := s.fileConfig()
	if cfg.IncompleteTTL <= 0 {
		return
	}
	baseAbs, err := filepath.Abs(cfg.BaseDir)
	if err != nil {
		return
	}
	now := time.Now()
	active := make(map[string]struct{})
	s.state.mu.Lock()
	for _, sess := range s.state.recv {
		if sess != nil && strings.TrimSpace(sess.partPath) != "" {
			active[sess.partPath] = struct{}{}
		}
	}
	s.state.mu.Unlock()
	_ = filepath.WalkDir(baseAbs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d == nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".part") {
			return nil
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		if _, ok := active[abs]; ok {
			return nil
		}
		info, err := d.Info()
		if err != nil || info == nil {
			return nil
		}
		if now.Sub(info.ModTime()) <= cfg.IncompleteTTL {
			return nil
		}
		_ = os.Remove(abs)
		return nil
	})
}

func (s *FileService) fileSendData(id [16]byte) {
	sess := s.fileGetSendSession(id)
	if sess == nil {
		return
	}
	sess.mu.Lock()
	task := sess.task
	provider := sess.provider
	consumer := sess.consumer
	path := sess.filePath
	startFrom := sess.startFrom
	size := sess.size
	sess.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	sess.mu.Lock()
	sess.cancel = cancel
	sess.mu.Unlock()
	defer cancel()

	f, err := os.Open(path)
	if err != nil {
		if task != nil {
			s.setTaskStatus(task, taskStatusFailed, "open failed")
		}
		s.fileRemoveSendSession(id)
		return
	}
	defer func() { _ = f.Close() }()

	if startFrom > 0 {
		if _, err := f.Seek(int64(startFrom), io.SeekStart); err != nil {
			if task != nil {
				s.setTaskStatus(task, taskStatusFailed, "seek failed")
			}
			s.fileRemoveSendSession(id)
			return
		}
	}

	chunkBytes := s.fileConfig().ChunkBytes
	if chunkBytes <= 0 {
		chunkBytes = 256 * 1024
	}
	buf := make([]byte, chunkBytes)
	offset := startFrom

	sendData := func(body []byte, fin bool) error {
		payload := fileEncodeBinHeaderV1(protocol.KindData, id, offset, fin, body)
		hdr := (&header.HeaderTcp{}).
			WithMajor(header.MajorMsg).
			WithSubProto(protocol.SubProtoFile).
			WithSourceID(provider).
			WithTargetID(consumer).
			WithMsgID(uint32(time.Now().UnixNano()))
		return s.session.Send(hdr, payload)
	}

	if size == 0 {
		if err := sendData(nil, true); err != nil {
			if task != nil {
				s.setTaskStatus(task, taskStatusFailed, "send failed")
			}
			s.fileRemoveSendSession(id)
			return
		}
		sess.mu.Lock()
		sess.sentEOF = true
		if task != nil {
			task.mu.Lock()
			task.status = taskStatusWaitingAck
			task.updatedAt = time.Now()
			task.mu.Unlock()
		}
		sess.mu.Unlock()
		s.emitTasksThrottled()
		return
	}

	for {
		select {
		case <-ctx.Done():
			if task != nil {
				s.setTaskStatus(task, taskStatusCanceled, "")
			}
			return
		default:
		}
		n, rerr := f.Read(buf)
		if n == 0 && rerr == io.EOF {
			break
		}
		if rerr != nil && rerr != io.EOF {
			if task != nil {
				s.setTaskStatus(task, taskStatusFailed, "read failed")
			}
			s.fileRemoveSendSession(id)
			return
		}
		body := buf[:n]
		fin := offset+uint64(n) == size
		if err := sendData(body, fin); err != nil {
			if task != nil {
				s.setTaskStatus(task, taskStatusFailed, "send failed")
			}
			s.fileRemoveSendSession(id)
			return
		}
		offset += uint64(n)
		sess.mu.Lock()
		sess.lastActive = time.Now()
		if task != nil {
			task.mu.Lock()
			task.sentBytes = offset
			task.updatedAt = time.Now()
			task.mu.Unlock()
		}
		if fin {
			sess.sentEOF = true
			if task != nil {
				task.mu.Lock()
				task.status = taskStatusWaitingAck
				task.mu.Unlock()
			}
		}
		sess.mu.Unlock()
		s.emitTasksThrottled()
		if fin {
			break
		}
	}
}

func (s *FileService) fileMaybeAckLocked(sess *fileRecvSession, force bool) {
	if sess == nil {
		return
	}
	cfg := s.fileConfig()
	need := force
	if !need && sess.expectedOffset-sess.lastAckOffset >= cfg.AckEveryBytes {
		need = true
	}
	if !need && time.Since(sess.lastAckTime) >= cfg.AckEvery {
		need = true
	}
	if !need {
		return
	}
	sess.lastAckOffset = sess.expectedOffset
	sess.lastAckTime = time.Now()
	go s.fileSendAck(sess.provider, sess.consumer, sess.id, sess.expectedOffset)
}

func (s *FileService) fileSendAck(provider, consumer uint32, sid [16]byte, offset uint64) {
	if s.session == nil || consumer == 0 || provider == 0 {
		return
	}
	payload := fileEncodeBinHeaderV1(protocol.KindAck, sid, offset, false, nil)
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorMsg).
		WithSubProto(protocol.SubProtoFile).
		WithSourceID(consumer).
		WithTargetID(provider).
		WithMsgID(uint32(time.Now().UnixNano()))
	_ = s.session.Send(hdr, payload)
}

func (s *FileService) fileFailRecvLocked(sess *fileRecvSession, reason string) {
	if sess == nil {
		return
	}
	if sess.task != nil {
		s.setTaskStatus(sess.task, taskStatusFailed, reason)
	}
	go s.fileRemoveRecvSession(sess.id)
}

func (s *FileService) fileFinishRecvLocked(sess *fileRecvSession) {
	if sess == nil {
		return
	}
	if sess.file != nil {
		_ = sess.file.Close()
		sess.file = nil
	}
	ok := true
	if sess.size != 0 {
		info, err := os.Stat(sess.partPath)
		if err != nil || info == nil || info.IsDir() || uint64(info.Size()) != sess.size {
			ok = false
		}
	}
	if ok && sess.hasher != nil && strings.TrimSpace(sess.sha256Hex) != "" {
		got := hex.EncodeToString(sess.hasher.Sum(nil))
		if !strings.EqualFold(got, strings.TrimSpace(sess.sha256Hex)) {
			ok = false
		}
	}
	if !ok {
		if sess.task != nil {
			s.setTaskStatus(sess.task, taskStatusFailed, "integrity failed")
		}
		go s.fileRemoveRecvSession(sess.id)
		return
	}
	if sess.overwrite {
		_ = os.Remove(sess.finalPath)
	}
	_ = os.Rename(sess.partPath, sess.finalPath)
	if sess.task != nil {
		sess.task.mu.Lock()
		sess.task.status = taskStatusCompleted
		sess.task.localPath = sess.finalPath
		sess.task.updatedAt = time.Now()
		sess.task.mu.Unlock()
	}
	go s.fileRemoveRecvSession(sess.id)
}

func (s *FileService) fileOnDisconnect(reason error) {
	if s == nil || s.state == nil {
		return
	}
	msg := "connection closed"
	if reason != nil {
		msg = reason.Error()
	}
	s.state.mu.Lock()
	recvIDs := make([][16]byte, 0, len(s.state.recv))
	for id := range s.state.recv {
		recvIDs = append(recvIDs, id)
	}
	sendIDs := make([][16]byte, 0, len(s.state.send))
	for id := range s.state.send {
		sendIDs = append(sendIDs, id)
	}
	for _, t := range s.state.tasks {
		if t == nil {
			continue
		}
		t.mu.Lock()
		switch t.status {
		case taskStatusSending, taskStatusReceiving, taskStatusWaitingAck, taskStatusWaitingResponse, taskStatusWaitingRemote, taskStatusPreparing, taskStatusHashing, taskStatusWaitingConfirm:
			t.status = taskStatusFailed
			t.lastError = msg
			t.updatedAt = time.Now()
		}
		t.mu.Unlock()
	}
	s.state.mu.Unlock()
	for _, id := range recvIDs {
		s.fileRemoveRecvSession(id)
	}
	for _, id := range sendIDs {
		s.fileRemoveSendSession(id)
	}
	s.emitTasksThrottled()
}

func (s *FileService) sendReadResp(sourceID, targetID uint32, data protocol.ReadResp) error {
	data.Op = strings.ToLower(strings.TrimSpace(data.Op))
	payload, err := transport.EncodeMessage(protocol.ActionReadResp, data)
	if err != nil {
		return err
	}
	return s.sendCtrl(context.Background(), sourceID, targetID, payload, "read_resp", data.Op)
}

func (s *FileService) sendWriteResp(sourceID, targetID uint32, data protocol.WriteResp) error {
	data.Op = strings.ToLower(strings.TrimSpace(data.Op))
	payload, err := transport.EncodeMessage(protocol.ActionWriteResp, data)
	if err != nil {
		return err
	}
	return s.sendCtrl(context.Background(), sourceID, targetID, payload, "write_resp", data.Op)
}

func (s *FileService) fileTotalSessions() int {
	if s == nil || s.state == nil {
		return 0
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	return len(s.state.recv) + len(s.state.send)
}

func (s *FileService) fileAddRecvSession(sess *fileRecvSession) {
	if s == nil || s.state == nil || sess == nil {
		return
	}
	s.state.mu.Lock()
	s.state.recv[sess.id] = sess
	s.state.mu.Unlock()
}

func (s *FileService) fileAddSendSession(sess *fileSendSession) {
	if s == nil || s.state == nil || sess == nil {
		return
	}
	s.state.mu.Lock()
	s.state.send[sess.id] = sess
	s.state.mu.Unlock()
}

func (s *FileService) fileGetRecvSession(id [16]byte) *fileRecvSession {
	if s == nil || s.state == nil {
		return nil
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	return s.state.recv[id]
}

func (s *FileService) fileGetSendSession(id [16]byte) *fileSendSession {
	if s == nil || s.state == nil {
		return nil
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	return s.state.send[id]
}

func (s *FileService) fileRemoveRecvSession(id [16]byte) {
	if s == nil || s.state == nil {
		return
	}
	s.state.mu.Lock()
	sess := s.state.recv[id]
	delete(s.state.recv, id)
	s.state.mu.Unlock()
	if sess != nil {
		sess.mu.Lock()
		if sess.file != nil {
			_ = sess.file.Close()
			sess.file = nil
		}
		sess.mu.Unlock()
	}
}

func (s *FileService) fileRemoveSendSession(id [16]byte) {
	if s == nil || s.state == nil {
		return
	}
	s.state.mu.Lock()
	sess := s.state.send[id]
	delete(s.state.send, id)
	s.state.mu.Unlock()
	if sess != nil {
		sess.mu.Lock()
		if sess.cancel != nil {
			sess.cancel()
			sess.cancel = nil
		}
		sess.mu.Unlock()
	}
}

func (s *FileService) fileAddTask(task *fileTask) {
	if s == nil || s.state == nil || task == nil {
		return
	}
	s.state.mu.Lock()
	s.state.tasks = append(s.state.tasks, task)
	if task.hasSessionID {
		s.state.taskBySid[task.sessionID] = task
	}
	s.state.mu.Unlock()
	s.emitTasksThrottled()
}

func (s *FileService) ensureOfferTask(id [16]byte, provider, consumer uint32, dir, name string, size uint64, sha string) *fileTask {
	s.state.mu.Lock()
	task := s.state.taskBySid[id]
	s.state.mu.Unlock()
	if task != nil {
		return task
	}
	taskID, _ := fileNewUUID()
	task = &fileTask{
		taskID:       taskID,
		sessionID:    id,
		hasSessionID: true,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
		op:           protocol.OpOffer,
		direction:    "download",
		provider:     provider,
		consumer:     consumer,
		peer:         provider,
		dir:          dir,
		name:         name,
		size:         size,
		sha256:       strings.TrimSpace(sha),
		wantHash:     strings.TrimSpace(sha) != "",
		status:       taskStatusWaitingConfirm,
	}
	s.fileAddTask(task)
	s.state.mu.Lock()
	s.state.taskBySid[id] = task
	s.state.mu.Unlock()
	return task
}

func (s *FileService) setTaskStatus(task *fileTask, status, reason string) {
	if task == nil {
		return
	}
	task.mu.Lock()
	task.status = status
	if strings.TrimSpace(reason) != "" {
		task.lastError = strings.TrimSpace(reason)
	}
	task.updatedAt = time.Now()
	task.mu.Unlock()
	s.emitTasksThrottled()
}

func (s *FileService) takePendingPull(provider uint32, dir, name string) *fileTask {
	key := filePullKey{provider: provider, dir: strings.TrimSpace(dir), name: strings.TrimSpace(name)}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	task := s.state.pendingPull[key]
	delete(s.state.pendingPull, key)
	return task
}

func (s *FileService) failPendingPull(provider uint32, dir, name, reason string) {
	task := s.takePendingPull(provider, dir, name)
	if task != nil {
		s.setTaskStatus(task, taskStatusFailed, reason)
	}
}

func (s *FileService) rejectOffer(provider uint32, req protocol.WriteReq) {
	localNode := s.localNodeFallback(0)
	if localNode == 0 {
		return
	}
	sid, ok := fileParseUUID(req.SessionID)
	if ok {
		s.state.mu.Lock()
		task := s.state.taskBySid[sid]
		s.state.mu.Unlock()
		if task != nil {
			s.setTaskStatus(task, taskStatusRejected, "rejected")
		}
	}
	_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{Code: 403, Msg: "rejected", Op: protocol.OpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
}

func (s *FileService) acceptOffer(provider uint32, req protocol.WriteReq, saveDir string) error {
	localNode := s.localNodeFallback(0)
	if localNode == 0 {
		return errors.New("identity not set")
	}
	cfg := s.fileConfig()
	if s.fileTotalSessions() >= cfg.MaxConcurrent {
		_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{Code: 429, Msg: "too many sessions", Op: protocol.OpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return errors.New("too many sessions")
	}
	sid, ok := fileParseUUID(req.SessionID)
	if !ok {
		_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{Code: 400, Msg: "invalid session", Op: protocol.OpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return errors.New("invalid session")
	}
	dir := strings.ReplaceAll(strings.TrimSpace(saveDir), "\\", "/")
	if _, err := fileSanitizeDir(dir); err != nil {
		_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{Code: 400, Msg: "invalid dir", Op: protocol.OpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return errors.New("invalid dir")
	}
	name := strings.TrimSpace(req.Name)
	finalPath, partPath, err := fileResolvePaths(cfg.BaseDir, dir, name)
	if err != nil {
		_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{Code: 400, Msg: "invalid path", Op: protocol.OpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return errors.New("invalid path")
	}
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{Code: 500, Msg: "mkdir failed", Op: protocol.OpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return errors.New("mkdir failed")
	}

	overwrite := true
	if req.Overwrite != nil {
		overwrite = *req.Overwrite
	}
	if !overwrite {
		if _, err := os.Stat(finalPath); err == nil {
			_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{Code: 409, Msg: "exists", Op: protocol.OpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
			return errors.New("file exists")
		}
	}

	resumeFrom := uint64(0)
	if st, err := os.Stat(partPath); err == nil && st != nil && !st.IsDir() {
		if uint64(st.Size()) <= req.Size {
			resumeFrom = uint64(st.Size())
		} else {
			_ = os.Remove(partPath)
		}
	}
	if resumeFrom == req.Size && req.Size != 0 {
		if fileVerifyPart(partPath, req.Size, strings.TrimSpace(req.Sha256)) {
			if overwrite {
				_ = os.Remove(finalPath)
			}
			_ = os.Rename(partPath, finalPath)
		} else {
			_ = os.Remove(partPath)
			resumeFrom = 0
		}
	}
	if req.Size != 0 && resumeFrom == req.Size {
		s.state.mu.Lock()
		task := s.state.taskBySid[sid]
		s.state.mu.Unlock()
		if task != nil {
			task.mu.Lock()
			task.status = taskStatusCompleted
			task.localDir = dir
			task.localPath = finalPath
			task.doneBytes = req.Size
			task.updatedAt = time.Now()
			task.mu.Unlock()
			s.emitTasksThrottled()
		}
		_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{
			Code:       1,
			Msg:        "ok",
			Op:         protocol.OpOffer,
			SessionID:  strings.TrimSpace(req.SessionID),
			Provider:   provider,
			Consumer:   localNode,
			Dir:        strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/"),
			Name:       name,
			Size:       req.Size,
			Sha256:     strings.TrimSpace(req.Sha256),
			Accept:     true,
			ResumeFrom: resumeFrom,
		})
		return nil
	}

	f, err := os.OpenFile(partPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{Code: 500, Msg: "open failed", Op: protocol.OpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return errors.New("open failed")
	}
	if err := f.Truncate(int64(resumeFrom)); err != nil {
		_ = f.Close()
		_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{Code: 500, Msg: "truncate failed", Op: protocol.OpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return errors.New("truncate failed")
	}
	var hasher hash.Hash
	shaHex := strings.TrimSpace(req.Sha256)
	if shaHex != "" {
		hasher = sha256.New()
	}
	if hasher != nil && resumeFrom > 0 {
		if _, err := f.Seek(0, io.SeekStart); err == nil {
			buf := make([]byte, 256*1024)
			remain := resumeFrom
			for remain > 0 {
				n := int64(len(buf))
				if remain < uint64(len(buf)) {
					n = int64(remain)
				}
				readN, rerr := io.ReadFull(f, buf[:n])
				if readN > 0 {
					_, _ = hasher.Write(buf[:readN])
					remain -= uint64(readN)
				}
				if rerr != nil {
					break
				}
			}
		}
	}
	if _, err := f.Seek(int64(resumeFrom), io.SeekStart); err != nil {
		_ = f.Close()
		_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{Code: 500, Msg: "seek failed", Op: protocol.OpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return errors.New("seek failed")
	}

	s.state.mu.Lock()
	task := s.state.taskBySid[sid]
	s.state.mu.Unlock()
	if task == nil {
		taskID, _ := fileNewUUID()
		task = &fileTask{
			taskID:       taskID,
			sessionID:    sid,
			hasSessionID: true,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
			op:           protocol.OpOffer,
			direction:    "download",
			provider:     provider,
			consumer:     localNode,
			peer:         provider,
			dir:          strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/"),
			name:         name,
			size:         req.Size,
			sha256:       shaHex,
			wantHash:     shaHex != "",
			status:       taskStatusReceiving,
		}
		s.fileAddTask(task)
		s.state.mu.Lock()
		s.state.taskBySid[sid] = task
		s.state.mu.Unlock()
	} else {
		task.mu.Lock()
		task.status = taskStatusReceiving
		task.localDir = dir
		task.localPath = finalPath
		task.size = req.Size
		task.sha256 = shaHex
		task.doneBytes = resumeFrom
		task.updatedAt = time.Now()
		task.mu.Unlock()
		s.emitTasksThrottled()
	}

	sess := &fileRecvSession{
		id:              sid,
		provider:        provider,
		consumer:        localNode,
		dir:             strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/"),
		name:            name,
		finalPath:       finalPath,
		partPath:        partPath,
		size:            req.Size,
		sha256Hex:       shaHex,
		overwrite:       overwrite,
		file:            f,
		hasher:          hasher,
		expectedOffset:  resumeFrom,
		pending:         make(map[uint64][]byte),
		maxPendingBytes: uint64(cfg.ChunkBytes) * 8,
		lastActive:      time.Now(),
		task:            task,
	}
	s.fileAddRecvSession(sess)
	s.emitTasksThrottled()

	_ = s.sendWriteResp(localNode, provider, protocol.WriteResp{
		Code:       1,
		Msg:        "ok",
		Op:         protocol.OpOffer,
		SessionID:  strings.TrimSpace(req.SessionID),
		Provider:   provider,
		Consumer:   localNode,
		Dir:        strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/"),
		Name:       name,
		Size:       req.Size,
		Sha256:     shaHex,
		Accept:     true,
		ResumeFrom: resumeFrom,
	})
	return nil
}

func (s *FileService) snapshotTasks() []FileTaskView {
	if s == nil || s.state == nil {
		return nil
	}
	s.state.mu.Lock()
	tasks := append([]*fileTask(nil), s.state.tasks...)
	s.state.mu.Unlock()
	out := make([]FileTaskView, 0, len(tasks))
	for _, task := range tasks {
		if task == nil {
			continue
		}
		task.mu.Lock()
		view := FileTaskView{
			TaskID:     fileUUIDToString(task.taskID),
			SessionID:  "",
			CreatedAt:  task.createdAt.Format(time.RFC3339),
			UpdatedAt:  task.updatedAt.Format(time.RFC3339),
			Op:         task.op,
			Direction:  task.direction,
			Status:     task.status,
			LastError:  task.lastError,
			Provider:   task.provider,
			Consumer:   task.consumer,
			Peer:       task.peer,
			Dir:        task.dir,
			Name:       task.name,
			Size:       task.size,
			Sha256:     task.sha256,
			WantHash:   task.wantHash,
			LocalDir:   task.localDir,
			LocalName:  task.localName,
			LocalPath:  task.localPath,
			FilePath:   task.filePath,
			SentBytes:  task.sentBytes,
			AckedBytes: task.ackedBytes,
			DoneBytes:  task.doneBytes,
		}
		if task.hasSessionID {
			view.SessionID = fileUUIDToString(task.sessionID)
		}
		task.mu.Unlock()
		out = append(out, view)
	}
	return out
}

func (s *FileService) findTask(taskID string) *fileTask {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" || s == nil || s.state == nil {
		return nil
	}
	s.state.mu.Lock()
	tasks := append([]*fileTask(nil), s.state.tasks...)
	s.state.mu.Unlock()
	for _, task := range tasks {
		if task == nil {
			continue
		}
		task.mu.Lock()
		id := fileUUIDToString(task.taskID)
		task.mu.Unlock()
		if id == taskID {
			return task
		}
	}
	return nil
}

func (s *FileService) emitList(evt FileListEvent) {
	if s == nil || s.bus == nil {
		return
	}
	_ = s.bus.Publish(context.Background(), EventFileList, evt, nil)
}

func (s *FileService) emitText(evt FileTextEvent) {
	if s == nil || s.bus == nil {
		return
	}
	_ = s.bus.Publish(context.Background(), EventFileText, evt, nil)
}

func (s *FileService) emitOffer(evt FileOfferEvent) {
	if s == nil || s.bus == nil {
		return
	}
	_ = s.bus.Publish(context.Background(), EventFileOffer, evt, nil)
}

func (s *FileService) emitTasksThrottled() {
	if s == nil || s.bus == nil || s.state == nil {
		return
	}
	now := time.Now()
	s.state.mu.Lock()
	if !s.state.lastEmit.IsZero() && now.Sub(s.state.lastEmit) < 200*time.Millisecond {
		s.state.mu.Unlock()
		return
	}
	s.state.lastEmit = now
	s.state.mu.Unlock()
	s.emitTasks()
}

func (s *FileService) emitTasks() {
	if s == nil || s.bus == nil {
		return
	}
	evt := FileTasksEvent{
		Tasks:     s.snapshotTasks(),
		UpdatedAt: time.Now(),
	}
	_ = s.bus.Publish(context.Background(), EventFileTasks, evt, nil)
}

func fileHashSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func fileVerifyPart(partPath string, size uint64, shaHex string) bool {
	info, err := os.Stat(partPath)
	if err != nil || info == nil || info.IsDir() {
		return false
	}
	if uint64(info.Size()) != size {
		return false
	}
	shaHex = strings.TrimSpace(strings.ToLower(shaHex))
	if shaHex == "" {
		return true
	}
	got, err := fileHashSHA256(partPath)
	if err != nil {
		return false
	}
	return strings.EqualFold(got, shaHex)
}

func openFolder(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("path is required")
	}
	if runtime.GOOS == "windows" {
		return exec.Command("explorer", path).Start()
	}
	return exec.Command("xdg-open", path).Start()
}
