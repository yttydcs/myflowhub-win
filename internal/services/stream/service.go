// 本文件实现 `stream` 后端服务，并暴露给 Win 壳层与 Wails 绑定。

package stream

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	core "github.com/yttydcs/myflowhub-core"
	corebus "github.com/yttydcs/myflowhub-core/eventbus"
	"github.com/yttydcs/myflowhub-core/header"
	proto "github.com/yttydcs/myflowhub-proto/protocol/stream"
	sdktransport "github.com/yttydcs/myflowhub-sdk/transport"
	"github.com/yttydcs/myflowhub-win/internal/services/logs"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
	"github.com/yttydcs/myflowhub-win/internal/services/transport"
)

const defaultStreamTimeout = 8 * time.Second

var streamMsgSeq atomic.Uint32
var streamMsgSeqInit sync.Once

type streamSession interface {
	Send(hdr core.IHeader, payload []byte) error
}

type ctrlWaitResult struct {
	message sdktransport.Message
	err     error
}

func nextStreamMsgID() uint32 {
	// nextStreamMsgID 为 stream 控制帧生成单调消息号，避免并发 await 时互相串响应。
	streamMsgSeqInit.Do(func() {
		var seed [4]byte
		if _, err := rand.Read(seed[:]); err != nil {
			streamMsgSeq.Store(uint32(time.Now().UnixNano()))
			return
		}
		streamMsgSeq.Store(binary.BigEndian.Uint32(seed[:]))
	})
	v := streamMsgSeq.Add(1)
	if v == 0 {
		v = streamMsgSeq.Add(1)
	}
	return v
}

type StreamService struct {
	session streamSession
	logs    *logs.LogService
	bus     corebus.IBus

	mu                 sync.Mutex
	deliveries         map[string]*deliveryRuntime
	sources            map[string]proto.SourceDescriptor
	consumers          map[string]proto.ConsumerDescriptor
	producerDeliveries map[string]*localProducerDelivery
	consumerDeliveries map[string]*localConsumerDelivery
	sourceInputs       map[string]sourceInputConfig
	fileSenders        map[string]*fileDeliverySender
	media              map[string]*mediaRuntime
	mediaServer        *mediaHTTPServer
	busTokens          []busToken
}

func New(session *sessionsvc.SessionService, logsSvc *logs.LogService, bus corebus.IBus) *StreamService {
	// New 初始化 stream 的本地目录、delivery 运行态和事件绑定，是整个子协议的内存中枢。
	svc := &StreamService{
		session:            session,
		logs:               logsSvc,
		bus:                bus,
		deliveries:         make(map[string]*deliveryRuntime),
		sources:            make(map[string]proto.SourceDescriptor),
		consumers:          make(map[string]proto.ConsumerDescriptor),
		producerDeliveries: make(map[string]*localProducerDelivery),
		consumerDeliveries: make(map[string]*localConsumerDelivery),
		sourceInputs:       make(map[string]sourceInputConfig),
		fileSenders:        make(map[string]*fileDeliverySender),
		media:              make(map[string]*mediaRuntime),
	}
	svc.bindBus()
	return svc
}

func (s *StreamService) Close() {
	s.closeMediaResources()
	s.unbindBus()
}

func (s *StreamService) DeliverySnapshot() []StreamDeliveryEvent {
	// DeliverySnapshot 导出当前内存里的 delivery 视图，给前端首次加载和窗口恢复使用。
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureStateMapsLocked()
	out := make([]StreamDeliveryEvent, 0, len(s.deliveries))
	for _, item := range s.deliveries {
		if item == nil {
			continue
		}
		out = append(out, item.snapshot())
	}
	return out
}

func (s *StreamService) Announce(ctx context.Context, sourceID, targetID uint32, req proto.AnnounceReq) (proto.AnnounceResp, error) {
	// Announce 在真正发协议前补齐本地 source 身份、默认 ID 并校验描述子合法性。
	if sourceID == 0 {
		return proto.AnnounceResp{}, errors.New("sourceID is required")
	}
	req.ReqID = ensureReqID(req.ReqID)
	if strings.TrimSpace(req.Source.SourceID) == "" {
		req.Source.SourceID = uuid.NewString()
	}
	if req.Source.Producer == 0 {
		req.Source.Producer = sourceID
	}
	if req.Source.Producer != sourceID {
		return proto.AnnounceResp{}, errors.New("source producer must match sourceID")
	}
	normalizeSourceDescriptor(&req.Source)
	if err := validateSourceDescriptor(req.Source); err != nil {
		return proto.AnnounceResp{}, err
	}
	return s.sendAndAwaitAnnounce(ctx, sourceID, targetID, req)
}

func (s *StreamService) AnnounceSimple(sourceID, targetID uint32, req proto.AnnounceReq) (proto.AnnounceResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStreamTimeout)
	defer cancel()
	return s.Announce(ctx, sourceID, targetID, req)
}

func (s *StreamService) Withdraw(ctx context.Context, sourceID, targetID uint32, req proto.WithdrawReq) (proto.WithdrawResp, error) {
	req.ReqID = ensureReqID(req.ReqID)
	req.SourceID = strings.TrimSpace(req.SourceID)
	if req.SourceID == "" {
		return proto.WithdrawResp{}, errors.New("source_id is required")
	}
	resp, err := sendAndAwaitJSON[proto.WithdrawResp](s, ctx, sourceID, targetID, proto.ActionWithdraw, proto.ActionWithdrawResp, req)
	if err != nil {
		return proto.WithdrawResp{}, err
	}
	if resp.Code == 1 {
		s.removeDeliveriesBySource(req.SourceID)
	}
	return resp, nil
}

func (s *StreamService) WithdrawSimple(sourceID, targetID uint32, req proto.WithdrawReq) (proto.WithdrawResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStreamTimeout)
	defer cancel()
	return s.Withdraw(ctx, sourceID, targetID, req)
}

func (s *StreamService) ListSources(ctx context.Context, sourceID, targetID uint32, req proto.ListSourcesReq) (proto.ListSourcesResp, error) {
	req.ReqID = ensureReqID(req.ReqID)
	if req.Producer == 0 {
		return proto.ListSourcesResp{}, errors.New("producer is required")
	}
	req.Kind = normalizeOptionalKind(req.Kind)
	req.Tag = strings.TrimSpace(req.Tag)
	return sendAndAwaitJSON[proto.ListSourcesResp](s, ctx, sourceID, targetID, proto.ActionListSources, proto.ActionListSourcesResp, req)
}

func (s *StreamService) ListSourcesSimple(sourceID, targetID uint32, req proto.ListSourcesReq) (proto.ListSourcesResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStreamTimeout)
	defer cancel()
	return s.ListSources(ctx, sourceID, targetID, req)
}

func (s *StreamService) GetSource(ctx context.Context, sourceID, targetID uint32, req proto.GetSourceReq) (proto.GetSourceResp, error) {
	req.ReqID = ensureReqID(req.ReqID)
	req.SourceID = strings.TrimSpace(req.SourceID)
	if req.Producer == 0 {
		return proto.GetSourceResp{}, errors.New("producer is required")
	}
	if req.SourceID == "" {
		return proto.GetSourceResp{}, errors.New("source_id is required")
	}
	return sendAndAwaitJSON[proto.GetSourceResp](s, ctx, sourceID, targetID, proto.ActionGetSource, proto.ActionGetSourceResp, req)
}

func (s *StreamService) GetSourceSimple(sourceID, targetID uint32, req proto.GetSourceReq) (proto.GetSourceResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStreamTimeout)
	defer cancel()
	return s.GetSource(ctx, sourceID, targetID, req)
}

func (s *StreamService) AnnounceConsumer(ctx context.Context, sourceID, targetID uint32, req proto.AnnounceConsumerReq) (proto.AnnounceConsumerResp, error) {
	if sourceID == 0 {
		return proto.AnnounceConsumerResp{}, errors.New("sourceID is required")
	}
	req.ReqID = ensureReqID(req.ReqID)
	if strings.TrimSpace(req.ConsumerEndpoint.ConsumerID) == "" {
		req.ConsumerEndpoint.ConsumerID = uuid.NewString()
	}
	if req.ConsumerEndpoint.Consumer == 0 {
		req.ConsumerEndpoint.Consumer = sourceID
	}
	if req.ConsumerEndpoint.Consumer != sourceID {
		return proto.AnnounceConsumerResp{}, errors.New("consumer endpoint consumer must match sourceID")
	}
	normalizeConsumerDescriptor(&req.ConsumerEndpoint)
	if err := validateConsumerDescriptor(req.ConsumerEndpoint); err != nil {
		return proto.AnnounceConsumerResp{}, err
	}
	return sendAndAwaitJSON[proto.AnnounceConsumerResp](s, ctx, sourceID, targetID, proto.ActionAnnounceConsumer, proto.ActionAnnounceConsumerResp, req)
}

func (s *StreamService) AnnounceConsumerSimple(sourceID, targetID uint32, req proto.AnnounceConsumerReq) (proto.AnnounceConsumerResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStreamTimeout)
	defer cancel()
	return s.AnnounceConsumer(ctx, sourceID, targetID, req)
}

func (s *StreamService) WithdrawConsumer(ctx context.Context, sourceID, targetID uint32, req proto.WithdrawConsumerReq) (proto.WithdrawConsumerResp, error) {
	req.ReqID = ensureReqID(req.ReqID)
	req.ConsumerID = strings.TrimSpace(req.ConsumerID)
	if req.ConsumerID == "" {
		return proto.WithdrawConsumerResp{}, errors.New("consumer_id is required")
	}
	resp, err := sendAndAwaitJSON[proto.WithdrawConsumerResp](s, ctx, sourceID, targetID, proto.ActionWithdrawConsumer, proto.ActionWithdrawConsumerResp, req)
	if err != nil {
		return proto.WithdrawConsumerResp{}, err
	}
	if resp.Code == 1 {
		s.removeDeliveriesByConsumer(req.ConsumerID)
	}
	return resp, nil
}

func (s *StreamService) WithdrawConsumerSimple(sourceID, targetID uint32, req proto.WithdrawConsumerReq) (proto.WithdrawConsumerResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStreamTimeout)
	defer cancel()
	return s.WithdrawConsumer(ctx, sourceID, targetID, req)
}

func (s *StreamService) ListConsumers(ctx context.Context, sourceID, targetID uint32, req proto.ListConsumersReq) (proto.ListConsumersResp, error) {
	req.ReqID = ensureReqID(req.ReqID)
	if req.Consumer == 0 {
		return proto.ListConsumersResp{}, errors.New("consumer is required")
	}
	req.Kind = normalizeOptionalKind(req.Kind)
	req.Tag = strings.TrimSpace(req.Tag)
	return sendAndAwaitJSON[proto.ListConsumersResp](s, ctx, sourceID, targetID, proto.ActionListConsumers, proto.ActionListConsumersResp, req)
}

func (s *StreamService) ListConsumersSimple(sourceID, targetID uint32, req proto.ListConsumersReq) (proto.ListConsumersResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStreamTimeout)
	defer cancel()
	return s.ListConsumers(ctx, sourceID, targetID, req)
}

func (s *StreamService) GetConsumer(ctx context.Context, sourceID, targetID uint32, req proto.GetConsumerReq) (proto.GetConsumerResp, error) {
	req.ReqID = ensureReqID(req.ReqID)
	req.ConsumerID = strings.TrimSpace(req.ConsumerID)
	if req.Consumer == 0 {
		return proto.GetConsumerResp{}, errors.New("consumer is required")
	}
	if req.ConsumerID == "" {
		return proto.GetConsumerResp{}, errors.New("consumer_id is required")
	}
	return sendAndAwaitJSON[proto.GetConsumerResp](s, ctx, sourceID, targetID, proto.ActionGetConsumer, proto.ActionGetConsumerResp, req)
}

func (s *StreamService) GetConsumerSimple(sourceID, targetID uint32, req proto.GetConsumerReq) (proto.GetConsumerResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStreamTimeout)
	defer cancel()
	return s.GetConsumer(ctx, sourceID, targetID, req)
}

func (s *StreamService) Subscribe(ctx context.Context, sourceID, targetID uint32, req proto.SubscribeReq) (proto.SubscribeResp, error) {
	// Subscribe 成功后立即登记本地 delivery 运行态，便于后续数据/ack 帧直接接续。
	req.ReqID = ensureReqID(req.ReqID)
	req.SourceID = strings.TrimSpace(req.SourceID)
	req.ConsumerID = strings.TrimSpace(req.ConsumerID)
	if req.Producer == 0 {
		return proto.SubscribeResp{}, errors.New("producer is required")
	}
	if req.SourceID == "" {
		return proto.SubscribeResp{}, errors.New("source_id is required")
	}
	if req.ConsumerID == "" {
		return proto.SubscribeResp{}, errors.New("consumer_id is required")
	}
	resp, err := sendAndAwaitJSON[proto.SubscribeResp](s, ctx, sourceID, targetID, proto.ActionSubscribe, proto.ActionSubscribeResp, req)
	if err != nil {
		return proto.SubscribeResp{}, err
	}
	s.trackAcceptedDelivery(resp.DeliveryID, resp.Source, resp.ConsumerEndpoint, resp.Producer, resp.Consumer, resp.ConsumerID, resp.Code == 1)
	return resp, nil
}

func (s *StreamService) SubscribeSimple(sourceID, targetID uint32, req proto.SubscribeReq) (proto.SubscribeResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStreamTimeout)
	defer cancel()
	return s.Subscribe(ctx, sourceID, targetID, req)
}

func (s *StreamService) Unsubscribe(ctx context.Context, sourceID, targetID uint32, req proto.UnsubscribeReq) (proto.UnsubscribeResp, error) {
	req.ReqID = ensureReqID(req.ReqID)
	req.DeliveryID = strings.TrimSpace(req.DeliveryID)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.DeliveryID == "" {
		return proto.UnsubscribeResp{}, errors.New("delivery_id is required")
	}
	resp, err := sendAndAwaitJSON[proto.UnsubscribeResp](s, ctx, sourceID, targetID, proto.ActionUnsubscribe, proto.ActionUnsubscribeResp, req)
	if err != nil {
		return proto.UnsubscribeResp{}, err
	}
	if resp.Code == 1 {
		s.removeDelivery(req.DeliveryID, "closed")
	}
	return resp, nil
}

func (s *StreamService) UnsubscribeSimple(sourceID, targetID uint32, req proto.UnsubscribeReq) (proto.UnsubscribeResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStreamTimeout)
	defer cancel()
	return s.Unsubscribe(ctx, sourceID, targetID, req)
}

func (s *StreamService) Connect(ctx context.Context, sourceID, targetID uint32, req proto.ConnectReq) (proto.ConnectResp, error) {
	// Connect 用于双向 delivery 建链，成功后同样把 delivery 快照纳入本地追踪。
	req.ReqID = ensureReqID(req.ReqID)
	req.SourceID = strings.TrimSpace(req.SourceID)
	req.ConsumerID = strings.TrimSpace(req.ConsumerID)
	if req.Producer == 0 {
		return proto.ConnectResp{}, errors.New("producer is required")
	}
	if req.Consumer == 0 {
		return proto.ConnectResp{}, errors.New("consumer is required")
	}
	if req.SourceID == "" {
		return proto.ConnectResp{}, errors.New("source_id is required")
	}
	if req.ConsumerID == "" {
		return proto.ConnectResp{}, errors.New("consumer_id is required")
	}
	resp, err := sendAndAwaitJSON[proto.ConnectResp](s, ctx, sourceID, targetID, proto.ActionConnect, proto.ActionConnectResp, req)
	if err != nil {
		return proto.ConnectResp{}, err
	}
	s.trackAcceptedDelivery(resp.DeliveryID, resp.Source, resp.ConsumerEndpoint, resp.Producer, resp.Consumer, resp.ConsumerID, resp.Code == 1)
	return resp, nil
}

func (s *StreamService) ConnectSimple(sourceID, targetID uint32, req proto.ConnectReq) (proto.ConnectResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStreamTimeout)
	defer cancel()
	return s.Connect(ctx, sourceID, targetID, req)
}

func (s *StreamService) Disconnect(ctx context.Context, sourceID, targetID uint32, req proto.DisconnectReq) (proto.DisconnectResp, error) {
	req.ReqID = ensureReqID(req.ReqID)
	req.DeliveryID = strings.TrimSpace(req.DeliveryID)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.DeliveryID == "" {
		return proto.DisconnectResp{}, errors.New("delivery_id is required")
	}
	resp, err := sendAndAwaitJSON[proto.DisconnectResp](s, ctx, sourceID, targetID, proto.ActionDisconnect, proto.ActionDisconnectResp, req)
	if err != nil {
		return proto.DisconnectResp{}, err
	}
	if resp.Code == 1 {
		s.removeDelivery(req.DeliveryID, "closed")
	}
	return resp, nil
}

func (s *StreamService) DisconnectSimple(sourceID, targetID uint32, req proto.DisconnectReq) (proto.DisconnectResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStreamTimeout)
	defer cancel()
	return s.Disconnect(ctx, sourceID, targetID, req)
}

func (s *StreamService) Signal(ctx context.Context, sourceID, targetID uint32, req proto.SignalReq) (proto.SignalResp, error) {
	req.ReqID = ensureReqID(req.ReqID)
	req.DeliveryID = strings.TrimSpace(req.DeliveryID)
	req.Op = strings.TrimSpace(req.Op)
	if req.DeliveryID == "" {
		return proto.SignalResp{}, errors.New("delivery_id is required")
	}
	if req.Op == "" {
		return proto.SignalResp{}, errors.New("op is required")
	}
	return sendAndAwaitJSON[proto.SignalResp](s, ctx, sourceID, targetID, proto.ActionSignal, proto.ActionSignalResp, req)
}

func (s *StreamService) SignalSimple(sourceID, targetID uint32, req proto.SignalReq) (proto.SignalResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStreamTimeout)
	defer cancel()
	return s.Signal(ctx, sourceID, targetID, req)
}

func (s *StreamService) sendAndAwaitAnnounce(ctx context.Context, sourceID, targetID uint32, req proto.AnnounceReq) (proto.AnnounceResp, error) {
	return sendAndAwaitJSON[proto.AnnounceResp](s, ctx, sourceID, targetID, proto.ActionAnnounce, proto.ActionAnnounceResp, req)
}

func encodeStreamCtrlPayload(action string, data any) ([]byte, error) {
	body, err := transport.EncodeMessage(action, data)
	if err != nil {
		return nil, err
	}
	payload := make([]byte, 1+len(body))
	payload[0] = proto.KindCtrl
	copy(payload[1:], body)
	return payload, nil
}

func decodeStreamCtrlMessage(payload []byte) (sdktransport.Message, error) {
	if len(payload) == 0 {
		return sdktransport.Message{}, errors.New("stream ctrl payload is empty")
	}
	if payload[0] != proto.KindCtrl {
		return sdktransport.Message{}, fmt.Errorf("unexpected stream payload kind: %d", payload[0])
	}
	return sdktransport.DecodeMessage(payload[1:])
}

func sendAndAwaitCtrlMessage(s *StreamService, ctx context.Context, sourceID, targetID uint32, reqAction, respAction string, payload []byte) (sdktransport.Message, error) {
	// sendAndAwaitCtrlMessage 在总线上临时挂三类监听，手动实现 stream control 的按 msg_id await。
	if s.session == nil {
		return sdktransport.Message{}, errors.New("session service not initialized")
	}
	if s.bus == nil {
		return sdktransport.Message{}, errors.New("event bus not initialized")
	}
	if sourceID == 0 {
		return sdktransport.Message{}, errors.New("sourceID is required")
	}
	if targetID == 0 {
		return sdktransport.Message{}, errors.New("targetID is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(proto.SubProtoStream).
		WithSourceID(sourceID).
		WithTargetID(targetID).
		WithTimestamp(uint32(time.Now().Unix())).
		WithMsgID(nextStreamMsgID())

	resultCh := make(chan ctrlWaitResult, 1)
	var resolveOnce sync.Once
	resolve := func(message sdktransport.Message, err error) {
		resolveOnce.Do(func() {
			resultCh <- ctrlWaitResult{message: message, err: err}
		})
	}

	frameToken := s.bus.Subscribe(sessionsvc.EventFrame, func(_ context.Context, evt corebus.Event) {
		frame, ok := evt.Data.(sessionsvc.FrameEvent)
		if !ok {
			return
		}
		if frame.SubProto != proto.SubProtoStream || frame.MsgID != hdr.GetMsgID() {
			return
		}
		if frame.Major != header.MajorOKResp && frame.Major != header.MajorErrResp {
			return
		}
		message, err := decodeStreamCtrlMessage(frame.Payload)
		if err != nil {
			resolve(sdktransport.Message{}, err)
			return
		}
		if message.Action != respAction {
			resolve(sdktransport.Message{}, fmt.Errorf("unexpected stream response action: got %q want %q", message.Action, respAction))
			return
		}
		resolve(message, nil)
	})
	errorToken := s.bus.Subscribe(sessionsvc.EventError, func(_ context.Context, evt corebus.Event) {
		errEvt, ok := evt.Data.(sessionsvc.ErrorEvent)
		if !ok {
			return
		}
		msg := strings.TrimSpace(errEvt.Message)
		if msg == "" {
			msg = "session error"
		}
		resolve(sdktransport.Message{}, errors.New(msg))
	})
	stateToken := s.bus.Subscribe(sessionsvc.EventState, func(_ context.Context, evt corebus.Event) {
		stateEvt, ok := evt.Data.(sessionsvc.StateEvent)
		if !ok || stateEvt.Connected {
			return
		}
		resolve(sdktransport.Message{}, errors.New("connection closed"))
	})
	defer s.bus.Unsubscribe(sessionsvc.EventFrame, frameToken)
	defer s.bus.Unsubscribe(sessionsvc.EventError, errorToken)
	defer s.bus.Unsubscribe(sessionsvc.EventState, stateToken)

	if err := s.session.Send(hdr, payload); err != nil {
		return sdktransport.Message{}, err
	}

	if s.logs != nil {
		s.logs.Appendf("info", "stream %s sent (msg_id=%d src=%d tgt=%d)", reqAction, hdr.GetMsgID(), sourceID, targetID)
	}

	select {
	case result := <-resultCh:
		return result.message, result.err
	case <-ctx.Done():
		return sdktransport.Message{}, ctx.Err()
	}
}

func sendAndAwaitJSON[T any](s *StreamService, ctx context.Context, sourceID, targetID uint32, reqAction, respAction string, req any) (T, error) {
	// sendAndAwaitJSON 复用 control await，再补 JSON 反序列化和 code!=1 的统一错误语义。
	var zero T
	payload, err := encodeStreamCtrlPayload(reqAction, req)
	if err != nil {
		return zero, err
	}
	message, err := sendAndAwaitCtrlMessage(s, ctx, sourceID, targetID, reqAction, respAction, payload)
	if err != nil {
		if s.logs != nil {
			s.logs.Appendf("error", "stream %s await failed: %v", reqAction, err)
		}
		return zero, fmt.Errorf("stream %s: %w", reqAction, toUIError(err))
	}
	var out T
	if err := json.Unmarshal(message.Data, &out); err != nil {
		if s.logs != nil {
			s.logs.Appendf("error", "stream %s decode failed: %v", reqAction, err)
		}
		return zero, err
	}
	code, msg := extractCodeMsg(out)
	if code != 1 {
		msg = strings.TrimSpace(msg)
		if s.logs != nil {
			if msg != "" {
				s.logs.Appendf("warn", "stream %s failed (code=%d msg=%q)", reqAction, code, msg)
			} else {
				s.logs.Appendf("warn", "stream %s failed (code=%d)", reqAction, code)
			}
		}
		if msg != "" {
			return zero, fmt.Errorf("%s (code=%d)", msg, code)
		}
		return zero, fmt.Errorf("stream %s failed (code=%d)", reqAction, code)
	}
	if s.logs != nil {
		s.logs.Appendf("info", "stream %s ok", reqAction)
	}
	return out, nil
}

func ensureReqID(reqID string) string {
	reqID = strings.TrimSpace(reqID)
	if reqID != "" {
		return reqID
	}
	return uuid.NewString()
}

func normalizeSourceDescriptor(desc *proto.SourceDescriptor) {
	if desc == nil {
		return
	}
	desc.SourceID = strings.TrimSpace(desc.SourceID)
	desc.Name = strings.TrimSpace(desc.Name)
	desc.Kind = normalizeRequiredKind(desc.Kind)
	desc.ContentType = strings.TrimSpace(desc.ContentType)
	desc.Mode = strings.TrimSpace(desc.Mode)
	desc.UnitMode = strings.TrimSpace(desc.UnitMode)
	desc.Tags = normalizeTags(desc.Tags)
}

func normalizeConsumerDescriptor(desc *proto.ConsumerDescriptor) {
	if desc == nil {
		return
	}
	desc.ConsumerID = strings.TrimSpace(desc.ConsumerID)
	desc.Name = strings.TrimSpace(desc.Name)
	desc.Kind = normalizeRequiredKind(desc.Kind)
	desc.ContentType = strings.TrimSpace(desc.ContentType)
	desc.Tags = normalizeTags(desc.Tags)
}

func validateSourceDescriptor(desc proto.SourceDescriptor) error {
	if strings.TrimSpace(desc.SourceID) == "" {
		return errors.New("source_id is required")
	}
	if desc.Producer == 0 {
		return errors.New("producer is required")
	}
	if err := validateKind(desc.Kind); err != nil {
		return err
	}
	return nil
}

func validateConsumerDescriptor(desc proto.ConsumerDescriptor) error {
	if strings.TrimSpace(desc.ConsumerID) == "" {
		return errors.New("consumer_id is required")
	}
	if desc.Consumer == 0 {
		return errors.New("consumer is required")
	}
	if err := validateKind(desc.Kind); err != nil {
		return err
	}
	return nil
}

func validateKind(kind string) error {
	switch normalizeRequiredKind(kind) {
	case proto.StreamKindMusic, proto.StreamKindVideo, proto.StreamKindText, proto.StreamKindCustom:
		return nil
	default:
		return fmt.Errorf("invalid kind: %s", strings.TrimSpace(kind))
	}
}

func normalizeRequiredKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}

func normalizeOptionalKind(kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		return ""
	}
	return kind
}

func normalizeTags(in []string) []string {
	// normalizeTags 去掉空值和重复项，保证 source/consumer 描述子落到 wire 前稳定可比较。
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		name := strings.TrimSpace(item)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func extractCodeMsg(v any) (int, string) {
	switch t := any(v).(type) {
	case proto.AnnounceResp:
		return t.Code, t.Msg
	case proto.WithdrawResp:
		return t.Code, t.Msg
	case proto.ListSourcesResp:
		return t.Code, t.Msg
	case proto.GetSourceResp:
		return t.Code, t.Msg
	case proto.AnnounceConsumerResp:
		return t.Code, t.Msg
	case proto.WithdrawConsumerResp:
		return t.Code, t.Msg
	case proto.ListConsumersResp:
		return t.Code, t.Msg
	case proto.GetConsumerResp:
		return t.Code, t.Msg
	case proto.SubscribeResp:
		return t.Code, t.Msg
	case proto.UnsubscribeResp:
		return t.Code, t.Msg
	case proto.ConnectResp:
		return t.Code, t.Msg
	case proto.DisconnectResp:
		return t.Code, t.Msg
	case proto.SignalResp:
		return t.Code, t.Msg
	default:
		return 0, ""
	}
}

func toUIError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("request timed out")
	}
	if errors.Is(err, context.Canceled) {
		return errors.New("request canceled")
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(msg, "session not initialized"):
		return errors.New("not connected")
	case strings.Contains(msg, "connection") && strings.Contains(msg, "closed"):
		return errors.New("connection closed")
	default:
		return err
	}
}
