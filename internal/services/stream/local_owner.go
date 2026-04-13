// Context: implements the local owner helper logic used by the stream backend service.

package stream

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/yttydcs/myflowhub-core/header"
	proto "github.com/yttydcs/myflowhub-proto/protocol/stream"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
)

const (
	actionDeliveryPrepare      = "delivery_prepare"
	actionDeliveryPrepareResp  = "delivery_prepare_resp"
	actionDeliveryActivate     = "delivery_activate"
	actionDeliveryActivateResp = "delivery_activate_resp"
	actionDeliveryAbort        = "delivery_abort"
	actionDeliveryAbortResp    = "delivery_abort_resp"
	actionDeliveryClose        = "delivery_close"
	actionDeliveryCloseResp    = "delivery_close_resp"

	deliveryRoleProducer = "producer"
	deliveryRoleConsumer = "consumer"

	localDeliveryStatePending = "pending"
	localDeliveryStateActive  = "active"
	localDeliveryStateClosing = "closing"

	defaultWindowBytes   = 256 * 1024
	defaultAckIntervalMs = 500
)

type localProducerDelivery struct {
	DeliveryID    string
	TxnID         string
	SourceID      string
	Producer      uint32
	Consumer      uint32
	ConsumerID    string
	Kind          string
	UnitMode      string
	Coordinator   uint32
	State         string
	WindowBytes   uint32
	AckIntervalMs uint32
	Position      uint64
	AckedPosition uint64
	LastActive    time.Time
}

type localConsumerDelivery struct {
	DeliveryID       string
	TxnID            string
	SourceID         string
	Producer         uint32
	Consumer         uint32
	ConsumerID       string
	Kind             string
	UnitMode         string
	Coordinator      uint32
	State            string
	WindowBytes      uint32
	AckIntervalMs    uint32
	ExpectedPosition uint64
	LastAckPosition  uint64
	LastActive       time.Time
}

type deliveryPrepareReq struct {
	ReqID         string `json:"req_id"`
	TxnID         string `json:"txn_id"`
	DeliveryID    string `json:"delivery_id"`
	Role          string `json:"role"`
	Coordinator   uint32 `json:"coordinator,omitempty"`
	Requester     uint32 `json:"requester,omitempty"`
	Producer      uint32 `json:"producer"`
	SourceID      string `json:"source_id"`
	Consumer      uint32 `json:"consumer"`
	ConsumerID    string `json:"consumer_id"`
	Kind          string `json:"kind,omitempty"`
	UnitMode      string `json:"unit_mode,omitempty"`
	ResumeFrom    uint64 `json:"resume_from,omitempty"`
	WindowBytes   uint32 `json:"window_bytes,omitempty"`
	AckIntervalMs uint32 `json:"ack_interval_ms,omitempty"`
}

type deliveryPrepareResp struct {
	ReqID            string                    `json:"req_id"`
	Code             int                       `json:"code"`
	Msg              string                    `json:"msg,omitempty"`
	Role             string                    `json:"role,omitempty"`
	Source           *proto.SourceDescriptor   `json:"source,omitempty"`
	ConsumerEndpoint *proto.ConsumerDescriptor `json:"consumer_endpoint,omitempty"`
	StartPosition    uint64                    `json:"start_position,omitempty"`
	WindowBytes      uint32                    `json:"window_bytes,omitempty"`
	AckIntervalMs    uint32                    `json:"ack_interval_ms,omitempty"`
}

type deliveryActivateReq struct {
	ReqID      string `json:"req_id"`
	TxnID      string `json:"txn_id"`
	DeliveryID string `json:"delivery_id"`
	Role       string `json:"role"`
}

type deliveryActivateResp struct {
	ReqID string `json:"req_id"`
	Code  int    `json:"code"`
	Msg   string `json:"msg,omitempty"`
	Role  string `json:"role,omitempty"`
}

type deliveryAbortReq struct {
	ReqID      string `json:"req_id"`
	TxnID      string `json:"txn_id"`
	DeliveryID string `json:"delivery_id"`
	Role       string `json:"role,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type deliveryAbortResp struct {
	ReqID string `json:"req_id"`
	Code  int    `json:"code"`
	Msg   string `json:"msg,omitempty"`
	Role  string `json:"role,omitempty"`
}

type deliveryCloseReq struct {
	ReqID      string `json:"req_id"`
	TxnID      string `json:"txn_id,omitempty"`
	DeliveryID string `json:"delivery_id"`
	Role       string `json:"role,omitempty"`
	Reason     string `json:"reason,omitempty"`
	CloseRoute bool   `json:"close_route,omitempty"`
}

type deliveryCloseResp struct {
	ReqID string `json:"req_id"`
	Code  int    `json:"code"`
	Msg   string `json:"msg,omitempty"`
	Role  string `json:"role,omitempty"`
}

func (s *StreamService) ensureStateMapsLocked() {
	if s.deliveries == nil {
		s.deliveries = make(map[string]*deliveryRuntime)
	}
	if s.sources == nil {
		s.sources = make(map[string]proto.SourceDescriptor)
	}
	if s.consumers == nil {
		s.consumers = make(map[string]proto.ConsumerDescriptor)
	}
	if s.producerDeliveries == nil {
		s.producerDeliveries = make(map[string]*localProducerDelivery)
	}
	if s.consumerDeliveries == nil {
		s.consumerDeliveries = make(map[string]*localConsumerDelivery)
	}
	if s.sourceInputs == nil {
		s.sourceInputs = make(map[string]sourceInputConfig)
	}
	if s.fileSenders == nil {
		s.fileSenders = make(map[string]*fileDeliverySender)
	}
	if s.media == nil {
		s.media = make(map[string]*mediaRuntime)
	}
}

func (s *StreamService) handleIncomingCtrl(frame sessionsvc.FrameEvent) {
	msg, err := decodeStreamCtrlMessage(frame.Payload)
	if err != nil {
		s.logWarn("stream inbound ctrl decode failed: %v", err)
		return
	}

	action := strings.ToLower(strings.TrimSpace(msg.Action))
	switch action {
	case proto.ActionAnnounce:
		reqID := decodeReqID(msg.Data)
		var req proto.AnnounceReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			s.sendLocalCtrlResp(frame, proto.ActionAnnounceResp, proto.AnnounceResp{ReqID: reqID, Code: 400, Msg: "invalid announce"})
			return
		}
		s.sendLocalCtrlResp(frame, proto.ActionAnnounceResp, s.handleAnnounceLocal(frame.TargetID, req))
	case proto.ActionWithdraw:
		reqID := decodeReqID(msg.Data)
		var req proto.WithdrawReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			s.sendLocalCtrlResp(frame, proto.ActionWithdrawResp, proto.WithdrawResp{ReqID: reqID, Code: 400, Msg: "invalid withdraw"})
			return
		}
		s.sendLocalCtrlResp(frame, proto.ActionWithdrawResp, s.handleWithdrawLocal(req))
	case proto.ActionListSources:
		reqID := decodeReqID(msg.Data)
		var req proto.ListSourcesReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			s.sendLocalCtrlResp(frame, proto.ActionListSourcesResp, proto.ListSourcesResp{ReqID: reqID, Code: 400, Msg: "invalid list_sources"})
			return
		}
		s.sendLocalCtrlResp(frame, proto.ActionListSourcesResp, s.handleListSourcesLocal(req))
	case proto.ActionGetSource:
		reqID := decodeReqID(msg.Data)
		var req proto.GetSourceReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			s.sendLocalCtrlResp(frame, proto.ActionGetSourceResp, proto.GetSourceResp{ReqID: reqID, Code: 400, Msg: "invalid get_source"})
			return
		}
		s.sendLocalCtrlResp(frame, proto.ActionGetSourceResp, s.handleGetSourceLocal(req))
	case proto.ActionAnnounceConsumer:
		reqID := decodeReqID(msg.Data)
		var req proto.AnnounceConsumerReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			s.sendLocalCtrlResp(frame, proto.ActionAnnounceConsumerResp, proto.AnnounceConsumerResp{ReqID: reqID, Code: 400, Msg: "invalid announce_consumer"})
			return
		}
		s.sendLocalCtrlResp(frame, proto.ActionAnnounceConsumerResp, s.handleAnnounceConsumerLocal(frame.TargetID, req))
	case proto.ActionWithdrawConsumer:
		reqID := decodeReqID(msg.Data)
		var req proto.WithdrawConsumerReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			s.sendLocalCtrlResp(frame, proto.ActionWithdrawConsumerResp, proto.WithdrawConsumerResp{ReqID: reqID, Code: 400, Msg: "invalid withdraw_consumer"})
			return
		}
		s.sendLocalCtrlResp(frame, proto.ActionWithdrawConsumerResp, s.handleWithdrawConsumerLocal(req))
	case proto.ActionListConsumers:
		reqID := decodeReqID(msg.Data)
		var req proto.ListConsumersReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			s.sendLocalCtrlResp(frame, proto.ActionListConsumersResp, proto.ListConsumersResp{ReqID: reqID, Code: 400, Msg: "invalid list_consumers"})
			return
		}
		s.sendLocalCtrlResp(frame, proto.ActionListConsumersResp, s.handleListConsumersLocal(req))
	case proto.ActionGetConsumer:
		reqID := decodeReqID(msg.Data)
		var req proto.GetConsumerReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			s.sendLocalCtrlResp(frame, proto.ActionGetConsumerResp, proto.GetConsumerResp{ReqID: reqID, Code: 400, Msg: "invalid get_consumer"})
			return
		}
		s.sendLocalCtrlResp(frame, proto.ActionGetConsumerResp, s.handleGetConsumerLocal(req))
	case actionDeliveryPrepare:
		reqID := decodeReqID(msg.Data)
		var req deliveryPrepareReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			s.sendLocalCtrlResp(frame, actionDeliveryPrepareResp, deliveryPrepareResp{ReqID: reqID, Code: 400, Msg: "invalid delivery_prepare"})
			return
		}
		s.sendLocalCtrlResp(frame, actionDeliveryPrepareResp, s.handleDeliveryPrepareLocal(frame, req))
	case actionDeliveryActivate:
		reqID := decodeReqID(msg.Data)
		var req deliveryActivateReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			s.sendLocalCtrlResp(frame, actionDeliveryActivateResp, deliveryActivateResp{ReqID: reqID, Code: 400, Msg: "invalid delivery_activate"})
			return
		}
		s.sendLocalCtrlResp(frame, actionDeliveryActivateResp, s.handleDeliveryActivateLocal(req))
	case actionDeliveryAbort:
		reqID := decodeReqID(msg.Data)
		var req deliveryAbortReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			s.sendLocalCtrlResp(frame, actionDeliveryAbortResp, deliveryAbortResp{ReqID: reqID, Code: 400, Msg: "invalid delivery_abort"})
			return
		}
		s.sendLocalCtrlResp(frame, actionDeliveryAbortResp, s.handleDeliveryAbortLocal(req))
	case actionDeliveryClose:
		reqID := decodeReqID(msg.Data)
		var req deliveryCloseReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			s.sendLocalCtrlResp(frame, actionDeliveryCloseResp, deliveryCloseResp{ReqID: reqID, Code: 400, Msg: "invalid delivery_close"})
			return
		}
		s.sendLocalCtrlResp(frame, actionDeliveryCloseResp, s.handleDeliveryCloseLocal(req))
	}
}

func (s *StreamService) sendLocalCtrlResp(frame sessionsvc.FrameEvent, action string, data any) {
	if s == nil || s.session == nil {
		return
	}
	payload, err := encodeStreamCtrlPayload(action, data)
	if err != nil {
		s.logWarn("stream local ctrl response encode failed: %v", err)
		return
	}
	sourceID := frame.TargetID
	if sourceID == 0 {
		sourceID = frame.SourceID
	}
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorOKResp).
		WithSubProto(proto.SubProtoStream).
		WithSourceID(sourceID).
		WithTargetID(frame.SourceID).
		WithMsgID(frame.MsgID).
		WithTimestamp(uint32(time.Now().Unix()))
	if err := s.session.Send(hdr, payload); err != nil {
		s.logWarn("stream local ctrl response send failed: %v", err)
	}
}

func (s *StreamService) handleAnnounceLocal(localNode uint32, req proto.AnnounceReq) proto.AnnounceResp {
	desc, code, msg := normalizeLocalSourceDescriptor(req.Source, localNode)
	if code != 0 {
		return proto.AnnounceResp{ReqID: req.ReqID, Code: code, Msg: msg}
	}

	s.mu.Lock()
	s.ensureStateMapsLocked()
	if existing, ok := s.sources[desc.SourceID]; ok && !sameSourceDescriptor(existing, desc) {
		s.mu.Unlock()
		return proto.AnnounceResp{ReqID: req.ReqID, Code: 409, Msg: "source conflict"}
	}
	s.sources[desc.SourceID] = cloneSourceDescriptor(desc)
	s.mu.Unlock()

	out := cloneSourceDescriptor(desc)
	return proto.AnnounceResp{ReqID: req.ReqID, Code: 1, Msg: "ok", Source: &out}
}

func (s *StreamService) handleWithdrawLocal(req proto.WithdrawReq) proto.WithdrawResp {
	sourceID := strings.TrimSpace(req.SourceID)
	if sourceID == "" {
		return proto.WithdrawResp{ReqID: req.ReqID, Code: 400, Msg: "source_id required"}
	}

	var owner proto.SourceDescriptor
	var snapshots []StreamDeliveryEvent

	s.mu.Lock()
	s.ensureStateMapsLocked()
	desc, ok := s.sources[sourceID]
	if !ok {
		s.mu.Unlock()
		return proto.WithdrawResp{ReqID: req.ReqID, Code: 404, Msg: "source not found", SourceID: sourceID}
	}
	owner = desc
	delete(s.sources, sourceID)
	delete(s.sourceInputs, sourceID)
	snapshots = append(snapshots, s.removeProducerDeliveriesBySourceLocked(sourceID, owner.Producer, "source withdrawn")...)
	snapshots = append(snapshots, s.removeConsumerDeliveriesBySourceLocked(sourceID, owner.Producer, "source withdrawn")...)
	s.mu.Unlock()

	s.emitDeliverySnapshots(snapshots)
	for _, snapshot := range snapshots {
		s.closeMediaRuntime(snapshot.DeliveryID, "source withdrawn")
	}
	return proto.WithdrawResp{ReqID: req.ReqID, Code: 1, Msg: "ok", SourceID: sourceID}
}

func (s *StreamService) handleListSourcesLocal(req proto.ListSourcesReq) proto.ListSourcesResp {
	kindFilter := normalizeOptionalKind(req.Kind)
	tagFilter := strings.TrimSpace(req.Tag)

	s.mu.Lock()
	s.ensureStateMapsLocked()
	items := make([]proto.SourceDescriptor, 0, len(s.sources))
	for _, item := range s.sources {
		if req.Producer != 0 && item.Producer != req.Producer {
			continue
		}
		if kindFilter != "" && item.Kind != kindFilter {
			continue
		}
		if tagFilter != "" && !containsTag(item.Tags, tagFilter) {
			continue
		}
		items = append(items, cloneSourceDescriptor(item))
	}
	s.mu.Unlock()

	sort.Slice(items, func(i, j int) bool { return items[i].SourceID < items[j].SourceID })
	return proto.ListSourcesResp{
		ReqID:    req.ReqID,
		Code:     1,
		Msg:      "ok",
		Producer: req.Producer,
		Sources:  items,
	}
}

func (s *StreamService) handleGetSourceLocal(req proto.GetSourceReq) proto.GetSourceResp {
	sourceID := strings.TrimSpace(req.SourceID)
	if sourceID == "" {
		return proto.GetSourceResp{ReqID: req.ReqID, Code: 400, Msg: "source_id required"}
	}

	s.mu.Lock()
	s.ensureStateMapsLocked()
	desc, ok := s.sources[sourceID]
	s.mu.Unlock()
	if !ok || (req.Producer != 0 && desc.Producer != req.Producer) {
		return proto.GetSourceResp{ReqID: req.ReqID, Code: 404, Msg: "source not found"}
	}

	out := cloneSourceDescriptor(desc)
	return proto.GetSourceResp{ReqID: req.ReqID, Code: 1, Msg: "ok", Source: &out}
}

func (s *StreamService) handleAnnounceConsumerLocal(localNode uint32, req proto.AnnounceConsumerReq) proto.AnnounceConsumerResp {
	desc, code, msg := normalizeLocalConsumerDescriptor(req.ConsumerEndpoint, localNode)
	if code != 0 {
		return proto.AnnounceConsumerResp{ReqID: req.ReqID, Code: code, Msg: msg}
	}

	s.mu.Lock()
	s.ensureStateMapsLocked()
	if existing, ok := s.consumers[desc.ConsumerID]; ok && !sameConsumerDescriptor(existing, desc) {
		s.mu.Unlock()
		return proto.AnnounceConsumerResp{ReqID: req.ReqID, Code: 409, Msg: "consumer conflict"}
	}
	s.consumers[desc.ConsumerID] = cloneConsumerDescriptor(desc)
	s.mu.Unlock()

	out := cloneConsumerDescriptor(desc)
	return proto.AnnounceConsumerResp{ReqID: req.ReqID, Code: 1, Msg: "ok", ConsumerEndpoint: &out}
}

func (s *StreamService) handleWithdrawConsumerLocal(req proto.WithdrawConsumerReq) proto.WithdrawConsumerResp {
	consumerID := strings.TrimSpace(req.ConsumerID)
	if consumerID == "" {
		return proto.WithdrawConsumerResp{ReqID: req.ReqID, Code: 400, Msg: "consumer_id required"}
	}

	var owner proto.ConsumerDescriptor
	var snapshots []StreamDeliveryEvent

	s.mu.Lock()
	s.ensureStateMapsLocked()
	desc, ok := s.consumers[consumerID]
	if !ok {
		s.mu.Unlock()
		return proto.WithdrawConsumerResp{ReqID: req.ReqID, Code: 404, Msg: "consumer not found", ConsumerID: consumerID}
	}
	owner = desc
	delete(s.consumers, consumerID)
	snapshots = append(snapshots, s.removeConsumerDeliveriesByConsumerLocked(consumerID, owner.Consumer, "consumer withdrawn")...)
	snapshots = append(snapshots, s.removeProducerDeliveriesByConsumerLocked(consumerID, owner.Consumer, "consumer withdrawn")...)
	s.mu.Unlock()

	s.emitDeliverySnapshots(snapshots)
	for _, snapshot := range snapshots {
		s.closeMediaRuntime(snapshot.DeliveryID, "consumer withdrawn")
	}
	return proto.WithdrawConsumerResp{ReqID: req.ReqID, Code: 1, Msg: "ok", ConsumerID: consumerID}
}

func (s *StreamService) handleListConsumersLocal(req proto.ListConsumersReq) proto.ListConsumersResp {
	kindFilter := normalizeOptionalKind(req.Kind)
	tagFilter := strings.TrimSpace(req.Tag)

	s.mu.Lock()
	s.ensureStateMapsLocked()
	items := make([]proto.ConsumerDescriptor, 0, len(s.consumers))
	for _, item := range s.consumers {
		if req.Consumer != 0 && item.Consumer != req.Consumer {
			continue
		}
		if kindFilter != "" && item.Kind != kindFilter {
			continue
		}
		if tagFilter != "" && !containsTag(item.Tags, tagFilter) {
			continue
		}
		items = append(items, cloneConsumerDescriptor(item))
	}
	s.mu.Unlock()

	sort.Slice(items, func(i, j int) bool { return items[i].ConsumerID < items[j].ConsumerID })
	return proto.ListConsumersResp{
		ReqID:             req.ReqID,
		Code:              1,
		Msg:               "ok",
		Consumer:          req.Consumer,
		ConsumerEndpoints: items,
	}
}

func (s *StreamService) handleGetConsumerLocal(req proto.GetConsumerReq) proto.GetConsumerResp {
	consumerID := strings.TrimSpace(req.ConsumerID)
	if consumerID == "" {
		return proto.GetConsumerResp{ReqID: req.ReqID, Code: 400, Msg: "consumer_id required"}
	}

	s.mu.Lock()
	s.ensureStateMapsLocked()
	desc, ok := s.consumers[consumerID]
	s.mu.Unlock()
	if !ok || (req.Consumer != 0 && desc.Consumer != req.Consumer) {
		return proto.GetConsumerResp{ReqID: req.ReqID, Code: 404, Msg: "consumer not found"}
	}

	out := cloneConsumerDescriptor(desc)
	return proto.GetConsumerResp{ReqID: req.ReqID, Code: 1, Msg: "ok", ConsumerEndpoint: &out}
}

func (s *StreamService) handleDeliveryPrepareLocal(frame sessionsvc.FrameEvent, req deliveryPrepareReq) deliveryPrepareResp {
	if strings.TrimSpace(req.ReqID) == "" || strings.TrimSpace(req.TxnID) == "" {
		return deliveryPrepareResp{ReqID: req.ReqID, Code: 400, Msg: "req_id and txn_id required", Role: req.Role}
	}
	localNode := frame.TargetID
	if localNode == 0 {
		if strings.TrimSpace(req.Role) == deliveryRoleConsumer {
			localNode = req.Consumer
		} else {
			localNode = req.Producer
		}
	}
	if req.Coordinator == 0 {
		req.Coordinator = frame.SourceID
	}

	switch strings.TrimSpace(req.Role) {
	case deliveryRoleProducer:
		return s.prepareProducerLocal(localNode, req)
	case deliveryRoleConsumer:
		return s.prepareConsumerLocal(localNode, req)
	default:
		return deliveryPrepareResp{ReqID: req.ReqID, Code: 400, Msg: "invalid role", Role: req.Role}
	}
}

func (s *StreamService) prepareProducerLocal(localNode uint32, req deliveryPrepareReq) deliveryPrepareResp {
	sourceID := strings.TrimSpace(req.SourceID)
	deliveryID := strings.TrimSpace(req.DeliveryID)
	consumerID := strings.TrimSpace(req.ConsumerID)

	s.mu.Lock()
	s.ensureStateMapsLocked()
	entry, ok := s.sources[sourceID]
	if !ok || entry.Producer != localNode {
		s.mu.Unlock()
		return deliveryPrepareResp{ReqID: req.ReqID, Code: 404, Msg: "source not found", Role: req.Role}
	}
	if existing := s.producerDeliveries[deliveryID]; existing != nil {
		if existing.TxnID != req.TxnID || existing.SourceID != sourceID || existing.Consumer != req.Consumer || existing.ConsumerID != consumerID {
			s.mu.Unlock()
			return deliveryPrepareResp{ReqID: req.ReqID, Code: 409, Msg: "delivery conflict", Role: req.Role}
		}
		out := cloneSourceDescriptor(entry)
		snapshot := s.upsertProducerSnapshotLocked(existing, entry, localDeliveryStatePending, "")
		s.mu.Unlock()
		s.emitDelivery(snapshot)
		return deliveryPrepareResp{
			ReqID:         req.ReqID,
			Code:          1,
			Msg:           "ok",
			Role:          req.Role,
			Source:        &out,
			StartPosition: existing.Position,
			WindowBytes:   existing.WindowBytes,
			AckIntervalMs: existing.AckIntervalMs,
		}
	}

	unitMode := strings.TrimSpace(entry.UnitMode)
	if unitMode == "" {
		unitMode = proto.UnitModeChunk
	}
	windowBytes := coalesceWindowBytes(req.WindowBytes)
	ackIntervalMs := coalesceAckIntervalMs(req.AckIntervalMs)
	delivery := &localProducerDelivery{
		DeliveryID:    deliveryID,
		TxnID:         req.TxnID,
		SourceID:      sourceID,
		Producer:      entry.Producer,
		Consumer:      req.Consumer,
		ConsumerID:    consumerID,
		Kind:          entry.Kind,
		UnitMode:      unitMode,
		Coordinator:   req.Coordinator,
		State:         localDeliveryStatePending,
		WindowBytes:   windowBytes,
		AckIntervalMs: ackIntervalMs,
		Position:      req.ResumeFrom,
		AckedPosition: req.ResumeFrom,
		LastActive:    time.Now(),
	}
	s.producerDeliveries[deliveryID] = delivery
	snapshot := s.upsertProducerSnapshotLocked(delivery, entry, localDeliveryStatePending, "")
	s.mu.Unlock()

	s.emitDelivery(snapshot)
	out := cloneSourceDescriptor(entry)
	return deliveryPrepareResp{
		ReqID:         req.ReqID,
		Code:          1,
		Msg:           "ok",
		Role:          req.Role,
		Source:        &out,
		StartPosition: req.ResumeFrom,
		WindowBytes:   windowBytes,
		AckIntervalMs: ackIntervalMs,
	}
}

func (s *StreamService) prepareConsumerLocal(localNode uint32, req deliveryPrepareReq) deliveryPrepareResp {
	consumerID := strings.TrimSpace(req.ConsumerID)
	deliveryID := strings.TrimSpace(req.DeliveryID)
	kind := normalizeOptionalKind(req.Kind)

	s.mu.Lock()
	s.ensureStateMapsLocked()
	entry, ok := s.consumers[consumerID]
	if !ok || entry.Consumer != localNode {
		s.mu.Unlock()
		return deliveryPrepareResp{ReqID: req.ReqID, Code: 404, Msg: "consumer not found", Role: req.Role}
	}
	if kind != "" && entry.Kind != kind {
		s.mu.Unlock()
		return deliveryPrepareResp{ReqID: req.ReqID, Code: 406, Msg: "kind mismatch", Role: req.Role}
	}
	if existing := s.consumerDeliveries[deliveryID]; existing != nil {
		if existing.TxnID != req.TxnID || existing.Producer != req.Producer || existing.ConsumerID != consumerID {
			s.mu.Unlock()
			return deliveryPrepareResp{ReqID: req.ReqID, Code: 409, Msg: "delivery conflict", Role: req.Role}
		}
		out := cloneConsumerDescriptor(entry)
		snapshot := s.upsertConsumerSnapshotLocked(existing, entry, localDeliveryStatePending, "")
		s.mu.Unlock()
		s.emitDelivery(snapshot)
		return deliveryPrepareResp{
			ReqID:            req.ReqID,
			Code:             1,
			Msg:              "ok",
			Role:             req.Role,
			ConsumerEndpoint: &out,
			StartPosition:    existing.ExpectedPosition,
			WindowBytes:      existing.WindowBytes,
			AckIntervalMs:    existing.AckIntervalMs,
		}
	}

	unitMode := strings.TrimSpace(req.UnitMode)
	if unitMode == "" {
		unitMode = proto.UnitModeChunk
	}
	windowBytes := coalesceWindowBytes(req.WindowBytes)
	ackIntervalMs := coalesceAckIntervalMs(req.AckIntervalMs)
	delivery := &localConsumerDelivery{
		DeliveryID:       deliveryID,
		TxnID:            req.TxnID,
		SourceID:         strings.TrimSpace(req.SourceID),
		Producer:         req.Producer,
		Consumer:         entry.Consumer,
		ConsumerID:       consumerID,
		Kind:             entry.Kind,
		UnitMode:         unitMode,
		Coordinator:      req.Coordinator,
		State:            localDeliveryStatePending,
		WindowBytes:      windowBytes,
		AckIntervalMs:    ackIntervalMs,
		ExpectedPosition: req.ResumeFrom,
		LastAckPosition:  req.ResumeFrom,
		LastActive:       time.Now(),
	}
	s.consumerDeliveries[deliveryID] = delivery
	snapshot := s.upsertConsumerSnapshotLocked(delivery, entry, localDeliveryStatePending, "")
	s.mu.Unlock()

	s.emitDelivery(snapshot)
	out := cloneConsumerDescriptor(entry)
	return deliveryPrepareResp{
		ReqID:            req.ReqID,
		Code:             1,
		Msg:              "ok",
		Role:             req.Role,
		ConsumerEndpoint: &out,
		StartPosition:    req.ResumeFrom,
		WindowBytes:      windowBytes,
		AckIntervalMs:    ackIntervalMs,
	}
}

func (s *StreamService) handleDeliveryActivateLocal(req deliveryActivateReq) deliveryActivateResp {
	deliveryID := strings.TrimSpace(req.DeliveryID)
	txnID := strings.TrimSpace(req.TxnID)
	if deliveryID == "" || txnID == "" {
		return deliveryActivateResp{ReqID: req.ReqID, Code: 400, Msg: "invalid activate", Role: req.Role}
	}

	var snapshot *StreamDeliveryEvent
	role := strings.TrimSpace(req.Role)

	s.mu.Lock()
	s.ensureStateMapsLocked()
	switch role {
	case deliveryRoleProducer:
		delivery := s.producerDeliveries[deliveryID]
		if delivery == nil || delivery.TxnID != txnID {
			s.mu.Unlock()
			return deliveryActivateResp{ReqID: req.ReqID, Code: 404, Msg: "delivery not found", Role: req.Role}
		}
		delivery.State = localDeliveryStateActive
		delivery.LastActive = time.Now()
		if source, ok := s.sources[delivery.SourceID]; ok {
			evt := s.upsertProducerSnapshotLocked(delivery, source, localDeliveryStateActive, "")
			snapshot = &evt
		}
	case deliveryRoleConsumer:
		delivery := s.consumerDeliveries[deliveryID]
		if delivery == nil || delivery.TxnID != txnID {
			s.mu.Unlock()
			return deliveryActivateResp{ReqID: req.ReqID, Code: 404, Msg: "delivery not found", Role: req.Role}
		}
		delivery.State = localDeliveryStateActive
		delivery.LastActive = time.Now()
		if consumer, ok := s.consumers[delivery.ConsumerID]; ok {
			evt := s.upsertConsumerSnapshotLocked(delivery, consumer, localDeliveryStateActive, "")
			snapshot = &evt
		}
	default:
		s.mu.Unlock()
		return deliveryActivateResp{ReqID: req.ReqID, Code: 400, Msg: "invalid role", Role: req.Role}
	}
	s.mu.Unlock()

	if snapshot != nil {
		s.emitDelivery(*snapshot)
	}
	switch role {
	case deliveryRoleProducer:
		s.maybeStartProducerFileSender(deliveryID)
	case deliveryRoleConsumer:
		s.prepareConsumerMediaRuntime(deliveryID)
	}
	return deliveryActivateResp{ReqID: req.ReqID, Code: 1, Msg: "ok", Role: req.Role}
}

func (s *StreamService) handleDeliveryAbortLocal(req deliveryAbortReq) deliveryAbortResp {
	deliveryID := strings.TrimSpace(req.DeliveryID)
	if deliveryID == "" {
		return deliveryAbortResp{ReqID: req.ReqID, Code: 400, Msg: "delivery_id required", Role: req.Role}
	}

	reason := strings.TrimSpace(req.Reason)
	snapshots := s.closeLocalDeliveryRoles(deliveryID, req.Role, reason)
	s.emitDeliverySnapshots(snapshots)
	s.closeMediaRuntime(deliveryID, reason)
	return deliveryAbortResp{ReqID: req.ReqID, Code: 1, Msg: "ok", Role: req.Role}
}

func (s *StreamService) handleDeliveryCloseLocal(req deliveryCloseReq) deliveryCloseResp {
	deliveryID := strings.TrimSpace(req.DeliveryID)
	if deliveryID == "" {
		return deliveryCloseResp{ReqID: req.ReqID, Code: 400, Msg: "delivery_id required", Role: req.Role}
	}

	reason := strings.TrimSpace(req.Reason)
	snapshots := s.closeLocalDeliveryRoles(deliveryID, req.Role, reason)
	s.emitDeliverySnapshots(snapshots)
	s.closeMediaRuntime(deliveryID, reason)
	return deliveryCloseResp{ReqID: req.ReqID, Code: 1, Msg: "ok", Role: req.Role}
}

func (s *StreamService) closeLocalDeliveryRoles(deliveryID, role, reason string) []StreamDeliveryEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureStateMapsLocked()

	switch strings.TrimSpace(role) {
	case deliveryRoleProducer:
		delete(s.producerDeliveries, deliveryID)
	case deliveryRoleConsumer:
		delete(s.consumerDeliveries, deliveryID)
	default:
		delete(s.producerDeliveries, deliveryID)
		delete(s.consumerDeliveries, deliveryID)
	}

	if snapshot, ok := s.snapshotAfterRoleChangeLocked(deliveryID, reason); ok {
		return []StreamDeliveryEvent{snapshot}
	}
	return nil
}

func (s *StreamService) upsertProducerSnapshotLocked(delivery *localProducerDelivery, source proto.SourceDescriptor, state, lastError string) StreamDeliveryEvent {
	s.ensureStateMapsLocked()
	rt := s.ensureDeliveryLocked(delivery.DeliveryID)
	rt.SourceID = strings.TrimSpace(source.SourceID)
	rt.Producer = source.Producer
	rt.Consumer = delivery.Consumer
	rt.ConsumerID = strings.TrimSpace(delivery.ConsumerID)
	rt.Kind = normalizeRequiredKind(source.Kind)
	rt.ContentType = strings.TrimSpace(source.ContentType)
	rt.Mode = strings.TrimSpace(source.Mode)
	rt.UnitMode = strings.TrimSpace(source.UnitMode)
	if rt.UnitMode == "" {
		rt.UnitMode = delivery.UnitMode
	}
	rt.State = strings.TrimSpace(state)
	rt.LastPosition = delivery.Position
	rt.LastAckPos = delivery.AckedPosition
	if lastError != "" {
		rt.LastError = lastError
	} else {
		rt.LastError = ""
	}
	rt.UpdatedAt = time.Now()
	return rt.snapshot()
}

func (s *StreamService) upsertConsumerSnapshotLocked(delivery *localConsumerDelivery, consumer proto.ConsumerDescriptor, state, lastError string) StreamDeliveryEvent {
	s.ensureStateMapsLocked()
	rt := s.ensureDeliveryLocked(delivery.DeliveryID)
	rt.SourceID = strings.TrimSpace(delivery.SourceID)
	rt.Producer = delivery.Producer
	rt.Consumer = consumer.Consumer
	rt.ConsumerID = strings.TrimSpace(consumer.ConsumerID)
	rt.Kind = normalizeRequiredKind(consumer.Kind)
	rt.ContentType = strings.TrimSpace(consumer.ContentType)
	rt.UnitMode = strings.TrimSpace(delivery.UnitMode)
	rt.State = strings.TrimSpace(state)
	rt.LastAckPos = delivery.LastAckPosition
	if lastError != "" {
		rt.LastError = lastError
	} else {
		rt.LastError = ""
	}
	rt.UpdatedAt = time.Now()
	return rt.snapshot()
}

func (s *StreamService) snapshotAfterRoleChangeLocked(deliveryID, reason string) (StreamDeliveryEvent, bool) {
	s.ensureStateMapsLocked()
	rt := s.deliveries[deliveryID]
	if rt == nil {
		return StreamDeliveryEvent{}, false
	}
	_, producerExists := s.producerDeliveries[deliveryID]
	_, consumerExists := s.consumerDeliveries[deliveryID]
	rt.UpdatedAt = time.Now()
	if reason != "" {
		rt.LastError = reason
	}
	if !producerExists {
		s.cancelFileSenderLocked(deliveryID)
	}
	if producerExists || consumerExists {
		rt.State = localDeliveryStateClosing
		return rt.snapshot(), true
	}
	delete(s.deliveries, deliveryID)
	rt.State = "closed"
	return rt.snapshot(), true
}

func (s *StreamService) emitDeliverySnapshots(snapshots []StreamDeliveryEvent) {
	for _, snapshot := range snapshots {
		s.emitDelivery(snapshot)
	}
}

func (s *StreamService) removeProducerDeliveriesBySourceLocked(sourceID string, producer uint32, reason string) []StreamDeliveryEvent {
	affected := make([]string, 0)
	for deliveryID, item := range s.producerDeliveries {
		if item == nil || item.SourceID != sourceID || item.Producer != producer {
			continue
		}
		delete(s.producerDeliveries, deliveryID)
		affected = append(affected, deliveryID)
	}
	return s.finalizeAffectedDeliveriesLocked(affected, reason)
}

func (s *StreamService) removeConsumerDeliveriesBySourceLocked(sourceID string, producer uint32, reason string) []StreamDeliveryEvent {
	affected := make([]string, 0)
	for deliveryID, item := range s.consumerDeliveries {
		if item == nil || item.SourceID != sourceID || item.Producer != producer {
			continue
		}
		delete(s.consumerDeliveries, deliveryID)
		affected = append(affected, deliveryID)
	}
	return s.finalizeAffectedDeliveriesLocked(affected, reason)
}

func (s *StreamService) removeConsumerDeliveriesByConsumerLocked(consumerID string, consumer uint32, reason string) []StreamDeliveryEvent {
	affected := make([]string, 0)
	for deliveryID, item := range s.consumerDeliveries {
		if item == nil || item.ConsumerID != consumerID || item.Consumer != consumer {
			continue
		}
		delete(s.consumerDeliveries, deliveryID)
		affected = append(affected, deliveryID)
	}
	return s.finalizeAffectedDeliveriesLocked(affected, reason)
}

func (s *StreamService) removeProducerDeliveriesByConsumerLocked(consumerID string, consumer uint32, reason string) []StreamDeliveryEvent {
	affected := make([]string, 0)
	for deliveryID, item := range s.producerDeliveries {
		if item == nil || item.ConsumerID != consumerID || item.Consumer != consumer {
			continue
		}
		delete(s.producerDeliveries, deliveryID)
		affected = append(affected, deliveryID)
	}
	return s.finalizeAffectedDeliveriesLocked(affected, reason)
}

func (s *StreamService) finalizeAffectedDeliveriesLocked(affected []string, reason string) []StreamDeliveryEvent {
	if len(affected) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(affected))
	out := make([]StreamDeliveryEvent, 0, len(affected))
	for _, deliveryID := range affected {
		if _, ok := seen[deliveryID]; ok {
			continue
		}
		seen[deliveryID] = struct{}{}
		if snapshot, ok := s.snapshotAfterRoleChangeLocked(deliveryID, reason); ok {
			out = append(out, snapshot)
		}
	}
	return out
}

func normalizeLocalSourceDescriptor(desc proto.SourceDescriptor, localNode uint32) (proto.SourceDescriptor, int, string) {
	normalizeSourceDescriptor(&desc)
	desc.Metadata = cloneRawMessage(desc.Metadata)
	if desc.Producer == 0 {
		desc.Producer = localNode
	}
	if desc.SourceID == "" {
		return proto.SourceDescriptor{}, 400, "source_id required"
	}
	if err := validateKind(desc.Kind); err != nil {
		return proto.SourceDescriptor{}, 406, "invalid kind"
	}
	if desc.Mode == "" {
		desc.Mode = proto.ModeLive
	} else if !isValidMode(desc.Mode) {
		return proto.SourceDescriptor{}, 406, "invalid mode"
	}
	if desc.UnitMode == "" {
		desc.UnitMode = proto.UnitModeChunk
	} else if !isValidUnitMode(desc.UnitMode) {
		return proto.SourceDescriptor{}, 406, "invalid unit_mode"
	}
	return desc, 0, ""
}

func normalizeLocalConsumerDescriptor(desc proto.ConsumerDescriptor, localNode uint32) (proto.ConsumerDescriptor, int, string) {
	normalizeConsumerDescriptor(&desc)
	desc.Metadata = cloneRawMessage(desc.Metadata)
	if desc.Consumer == 0 {
		desc.Consumer = localNode
	}
	if desc.ConsumerID == "" {
		return proto.ConsumerDescriptor{}, 400, "consumer_id required"
	}
	if err := validateKind(desc.Kind); err != nil {
		return proto.ConsumerDescriptor{}, 406, "invalid kind"
	}
	return desc, 0, ""
}

func cloneSourceDescriptor(desc proto.SourceDescriptor) proto.SourceDescriptor {
	desc.Tags = append([]string(nil), desc.Tags...)
	desc.Metadata = cloneRawMessage(desc.Metadata)
	return desc
}

func cloneConsumerDescriptor(desc proto.ConsumerDescriptor) proto.ConsumerDescriptor {
	desc.Tags = append([]string(nil), desc.Tags...)
	desc.Metadata = cloneRawMessage(desc.Metadata)
	return desc
}

func sameSourceDescriptor(a, b proto.SourceDescriptor) bool {
	return a.SourceID == b.SourceID &&
		a.Producer == b.Producer &&
		a.Name == b.Name &&
		a.Kind == b.Kind &&
		a.ContentType == b.ContentType &&
		a.Mode == b.Mode &&
		a.UnitMode == b.UnitMode &&
		strings.Join(a.Tags, "\x00") == strings.Join(b.Tags, "\x00") &&
		bytes.Equal(a.Metadata, b.Metadata)
}

func sameConsumerDescriptor(a, b proto.ConsumerDescriptor) bool {
	return a.ConsumerID == b.ConsumerID &&
		a.Consumer == b.Consumer &&
		a.Name == b.Name &&
		a.Kind == b.Kind &&
		a.ContentType == b.ContentType &&
		strings.Join(a.Tags, "\x00") == strings.Join(b.Tags, "\x00") &&
		bytes.Equal(a.Metadata, b.Metadata)
}

func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return out
}

func decodeReqID(raw json.RawMessage) string {
	var base struct {
		ReqID string `json:"req_id"`
	}
	_ = json.Unmarshal(raw, &base)
	return strings.TrimSpace(base.ReqID)
}

func containsTag(tags []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		return false
	}
	for _, item := range tags {
		if strings.ToLower(strings.TrimSpace(item)) == target {
			return true
		}
	}
	return false
}

func isValidMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case proto.ModeLive, proto.ModeBounded:
		return true
	default:
		return false
	}
}

func isValidUnitMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case proto.UnitModeFrame, proto.UnitModeChunk:
		return true
	default:
		return false
	}
}

func coalesceWindowBytes(v uint32) uint32 {
	if v == 0 {
		return defaultWindowBytes
	}
	return v
}

func coalesceAckIntervalMs(v uint32) uint32 {
	if v == 0 {
		return defaultAckIntervalMs
	}
	return v
}

func nextExpectedPosition(unitMode string, position uint64, bodyLen int) uint64 {
	if strings.TrimSpace(unitMode) == proto.UnitModeFrame {
		return position + 1
	}
	if bodyLen <= 0 {
		return position
	}
	return position + uint64(bodyLen)
}
