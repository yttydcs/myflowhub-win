package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	core "github.com/yttydcs/myflowhub-core"
	"github.com/yttydcs/myflowhub-core/eventbus"
	"github.com/yttydcs/myflowhub-core/header"
	sdkawait "github.com/yttydcs/myflowhub-sdk/await"
	protocolfile "github.com/yttydcs/myflowhub-proto/protocol/file"
	"github.com/yttydcs/myflowhub-win/internal/services/logs"
	winsession "github.com/yttydcs/myflowhub-win/internal/session"
)

const (
	EventFrame = "session.frame"
	EventError = "session.error"
	EventState = "session.state"

	logPayloadLimit = 256
)

type FrameEvent struct {
	Major      uint8  `json:"major"`
	SubProto   uint8  `json:"sub_proto"`
	SourceID   uint32 `json:"source_id"`
	TargetID   uint32 `json:"target_id"`
	MsgID      uint32 `json:"msg_id"`
	Timestamp  uint32 `json:"timestamp"`
	Payload    []byte `json:"payload"`
	PayloadLen int    `json:"payload_len"`
}

type StateEvent struct {
	Connected bool      `json:"connected"`
	Addr      string    `json:"addr"`
	Time      time.Time `json:"time"`
}

type ErrorEvent struct {
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

type SessionService struct {
	mu        sync.Mutex
	ctx       context.Context
	sess      *winsession.Session
	bus       eventbus.IBus
	logs      *logs.LogService
	connected atomic.Bool
	lastAddr  string
}

func New(ctx context.Context, bus eventbus.IBus, logsSvc *logs.LogService) *SessionService {
	if ctx == nil {
		ctx = context.Background()
	}
	s := &SessionService{ctx: ctx, bus: bus, logs: logsSvc}
	s.sess = winsession.New(ctx, s.handleFrame, s.handleError)
	return s
}

func (s *SessionService) SetContext(ctx context.Context) {
	if ctx == nil {
		return
	}
	s.ctx = ctx
}

func (s *SessionService) Connect(addr string) error {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return errors.New("addr is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sess == nil {
		return errors.New("session not initialized")
	}
	if err := s.sess.Connect(addr); err != nil {
		if !strings.Contains(err.Error(), "已经连接") {
			return err
		}
	}
	s.connected.Store(true)
	s.lastAddr = addr
	s.publishState(true, addr)
	s.logs.Appendf("info", "session connected: %s", addr)
	return nil
}

func (s *SessionService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sess != nil {
		s.sess.Close()
	}
	s.connected.Store(false)
	s.publishState(false, s.lastAddr)
	s.logs.Append("info", "session closed")
}

func (s *SessionService) IsConnected() bool {
	return s.connected.Load()
}

func (s *SessionService) LastAddr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastAddr
}

func (s *SessionService) LoginLegacy(nodeName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sess == nil {
		return errors.New("session not initialized")
	}
	return s.sess.Login(strings.TrimSpace(nodeName))
}

func (s *SessionService) Send(hdr core.IHeader, payload []byte) error {
	if hdr == nil {
		return errors.New("header is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sess == nil {
		return errors.New("session not initialized")
	}
	return s.sess.Send(hdr, payload)
}

func (s *SessionService) SendCommand(subProto uint8, sourceID, targetID uint32, payload []byte) error {
	if subProto == 0 {
		return errors.New("subProto is required")
	}
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(subProto).
		WithSourceID(sourceID).
		WithTargetID(targetID).
		WithMsgID(uint32(time.Now().UnixNano())).
		WithTimestamp(uint32(time.Now().Unix()))
	if err := s.Send(hdr, payload); err != nil {
		return err
	}
	if s.logs != nil {
		trimmed, truncated := trimPayload(payload, logPayloadLimit)
		s.logs.AppendPayload("info",
			fmt.Sprintf("[TX] major=%d sub=%d src=%d tgt=%d len=%d",
				hdr.Major(), hdr.SubProto(), hdr.SourceID(), hdr.TargetID(), len(payload)),
			trimmed,
			len(payload),
			truncated,
		)
	}
	return nil
}

func (s *SessionService) SendCommandAndAwait(ctx context.Context, subProto uint8, sourceID, targetID uint32, payload []byte, expectAction string) (sdkawait.Response, error) {
	if subProto == 0 {
		return sdkawait.Response{}, errors.New("subProto is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(subProto).
		WithSourceID(sourceID).
		WithTargetID(targetID).
		WithTimestamp(uint32(time.Now().Unix()))

	// 不持锁等待：避免长时间占用 mu，影响 Close/Connect 等操作。
	s.mu.Lock()
	sess := s.sess
	s.mu.Unlock()
	if sess == nil {
		return sdkawait.Response{}, errors.New("session not initialized")
	}

	if s.logs != nil {
		trimmed, truncated := trimPayload(payload, logPayloadLimit)
		s.logs.AppendPayload("info",
			fmt.Sprintf("[TX] major=%d sub=%d src=%d tgt=%d len=%d",
				hdr.Major(), hdr.SubProto(), hdr.SourceID(), hdr.TargetID(), len(payload)),
			trimmed,
			len(payload),
			truncated,
		)
	}

	return sess.SendAndAwait(ctx, hdr, payload, expectAction)
}

func (s *SessionService) handleFrame(hdr core.IHeader, payload []byte) {
	if hdr == nil {
		return
	}
	if s.bus != nil {
		evt := FrameEvent{
			Major:      hdr.Major(),
			SubProto:   hdr.SubProto(),
			SourceID:   hdr.SourceID(),
			TargetID:   hdr.TargetID(),
			MsgID:      hdr.GetMsgID(),
			Timestamp:  hdr.GetTimestamp(),
			Payload:    payload,
			PayloadLen: len(payload),
		}
		_ = s.bus.Publish(context.Background(), EventFrame, evt, nil)
	}
	if s.logs == nil {
		return
	}
	if shouldSkipLog(hdr.SubProto(), payload) {
		return
	}
	trimmed, truncated := trimPayload(payload, logPayloadLimit)
	s.logs.AppendPayload("info",
		fmt.Sprintf("[RX] major=%d sub=%d src=%d tgt=%d len=%d",
			hdr.Major(), hdr.SubProto(), hdr.SourceID(), hdr.TargetID(), len(payload)),
		trimmed,
		len(payload),
		truncated,
	)
}

func (s *SessionService) handleError(err error) {
	if err == nil {
		return
	}
	if s.bus != nil {
		_ = s.bus.Publish(context.Background(), EventError, ErrorEvent{Message: err.Error(), Time: time.Now()}, nil)
	}
	if s.logs != nil {
		s.logs.Appendf("error", "session error: %v", err)
	}
	s.connected.Store(false)
	s.publishState(false, s.lastAddr)
}

func (s *SessionService) publishState(connected bool, addr string) {
	if s.bus == nil {
		return
	}
	evt := StateEvent{Connected: connected, Addr: addr, Time: time.Now()}
	_ = s.bus.Publish(context.Background(), EventState, evt, nil)
}

func shouldSkipLog(subProto uint8, payload []byte) bool {
	if subProto != protocolfile.SubProtoFile || len(payload) == 0 {
		return false
	}
	kind := payload[0]
	return kind == protocolfile.KindData || kind == protocolfile.KindAck
}

func trimPayload(payload []byte, limit int) ([]byte, bool) {
	if len(payload) == 0 || limit <= 0 {
		return payload, false
	}
	if len(payload) <= limit {
		return payload, false
	}
	out := make([]byte, limit)
	copy(out, payload[:limit])
	return out, true
}
