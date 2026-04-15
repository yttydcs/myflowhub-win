// 本文件实现 `stream` 后端服务中与 `publish` 相关的辅助逻辑。

package stream

import (
	"context"
	"encoding/binary"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yttydcs/myflowhub-core/header"
	proto "github.com/yttydcs/myflowhub-proto/protocol/stream"
)

type PublishTextReq struct {
	SourceID string `json:"source_id"`
	Text     string `json:"text"`
}

type PublishTextResp struct {
	Code        int      `json:"code"`
	Msg         string   `json:"msg,omitempty"`
	SourceID    string   `json:"source_id"`
	Sent        int      `json:"sent"`
	DeliveryIDs []string `json:"delivery_ids,omitempty"`
}

type PublishCaptureChunkReq struct {
	SourceID     string   `json:"source_id"`
	DeliveryIDs  []string `json:"delivery_ids"`
	PtsMs        uint64   `json:"pts_ms,omitempty"`
	SessionStart bool     `json:"session_start,omitempty"`
	Final        bool     `json:"final,omitempty"`
	Payload      []byte   `json:"payload,omitempty"`
}

type PublishCaptureChunkResp struct {
	Code        int      `json:"code"`
	Msg         string   `json:"msg,omitempty"`
	SourceID    string   `json:"source_id"`
	Sent        int      `json:"sent"`
	DeliveryIDs []string `json:"delivery_ids,omitempty"`
}

type producerSendTarget struct {
	DeliveryID string
	SourceID   string
	Producer   uint32
	Consumer   uint32
	Position   uint64
	UnitMode   string
}

func (s *StreamService) PublishText(ctx context.Context, sourceID uint32, req PublishTextReq) (PublishTextResp, error) {
	// PublishText 面向 text source 的直接输入，把同一文本扇出到所有活跃 delivery。
	_ = ctx
	if sourceID == 0 {
		return PublishTextResp{}, errors.New("sourceID is required")
	}
	if s == nil || s.session == nil {
		return PublishTextResp{}, errors.New("session service not initialized")
	}

	sourceKey := strings.TrimSpace(req.SourceID)
	if sourceKey == "" {
		return PublishTextResp{}, errors.New("source_id is required")
	}
	body := []byte(req.Text)
	if strings.TrimSpace(req.Text) == "" {
		return PublishTextResp{}, errors.New("text is required")
	}

	targets, err := s.collectTextPublishTargets(sourceID, sourceKey)
	if err != nil {
		return PublishTextResp{}, err
	}

	now := time.Now()
	ptsMs := uint64(now.UnixMilli())
	sent := make([]string, 0, len(targets))
	snapshots := make([]StreamDeliveryEvent, 0, len(targets))
	var lastErr error

	for _, target := range targets {
		payload, err := buildDataPayload(target.DeliveryID, target.Position, ptsMs, body)
		if err != nil {
			lastErr = err
			continue
		}
		hdr := (&header.HeaderTcp{}).
			WithMajor(header.MajorMsg).
			WithSubProto(proto.SubProtoStream).
			WithSourceID(target.Producer).
			WithTargetID(target.Consumer).
			WithTimestamp(uint32(now.Unix()))
		if err := s.session.Send(hdr, payload); err != nil {
			lastErr = err
			s.logWarn("stream publish text send failed: %v", err)
			continue
		}
		sent = append(sent, target.DeliveryID)
		if snapshot, ok := s.markTextPublished(target, uint64(len(body)), ptsMs); ok {
			snapshots = append(snapshots, snapshot)
		}
	}

	s.emitDeliverySnapshots(snapshots)
	if len(sent) == 0 {
		if lastErr == nil {
			lastErr = errors.New("no text delivery sent")
		}
		return PublishTextResp{}, lastErr
	}
	return PublishTextResp{Code: 1, Msg: "ok", SourceID: sourceKey, Sent: len(sent), DeliveryIDs: sent}, nil
}

func (s *StreamService) PublishTextSimple(sourceID uint32, req PublishTextReq) (PublishTextResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStreamTimeout)
	defer cancel()
	return s.PublishText(ctx, sourceID, req)
}

func (s *StreamService) PublishCaptureChunk(ctx context.Context, sourceID uint32, req PublishCaptureChunkReq) (PublishCaptureChunkResp, error) {
	// PublishCaptureChunk 在发送桌面采集块前会等待 producer 窗口，避免无界堆积未确认数据。
	if sourceID == 0 {
		return PublishCaptureChunkResp{}, errors.New("sourceID is required")
	}
	if s == nil || s.session == nil {
		return PublishCaptureChunkResp{}, errors.New("session service not initialized")
	}

	sourceKey := strings.TrimSpace(req.SourceID)
	if sourceKey == "" {
		return PublishCaptureChunkResp{}, errors.New("source_id is required")
	}
	deliveryIDs := normalizeRequestedDeliveryIDs(req.DeliveryIDs)
	if len(deliveryIDs) == 0 {
		return PublishCaptureChunkResp{}, errors.New("delivery_ids are required")
	}
	if len(req.Payload) == 0 && !req.Final {
		return PublishCaptureChunkResp{}, errors.New("payload is required unless final is true")
	}
	targets, err := s.collectCapturePublishTargets(sourceID, sourceKey, deliveryIDs)
	if err != nil {
		return PublishCaptureChunkResp{}, err
	}

	now := time.Now()
	ptsMs := req.PtsMs
	if ptsMs == 0 {
		ptsMs = uint64(now.UnixMilli())
	}
	flags := uint8(0)
	if req.Final {
		flags |= streamDataFlagEOF
	}
	if req.SessionStart {
		flags |= streamDataFlagSessionStart
	}

	sent := make([]string, 0, len(targets))
	snapshots := make([]StreamDeliveryEvent, 0, len(targets))
	var lastErr error

	for _, target := range targets {
		if err := s.waitProducerWindow(ctx, target.DeliveryID, target.Position+uint64(len(req.Payload))); err != nil {
			lastErr = err
			continue
		}
		payload, err := buildDataPayloadWithFlags(target.DeliveryID, target.Position, ptsMs, flags, req.Payload)
		if err != nil {
			lastErr = err
			continue
		}
		hdr := (&header.HeaderTcp{}).
			WithMajor(header.MajorMsg).
			WithSubProto(proto.SubProtoStream).
			WithSourceID(target.Producer).
			WithTargetID(target.Consumer).
			WithTimestamp(uint32(now.Unix()))
		if err := s.session.Send(hdr, payload); err != nil {
			lastErr = err
			s.logWarn("stream publish capture send failed: %v", err)
			continue
		}
		sent = append(sent, target.DeliveryID)
		if snapshot, ok := s.markProducerPayloadSent(target.DeliveryID, target.Position, uint64(len(req.Payload)), ptsMs, flags); ok {
			snapshots = append(snapshots, snapshot)
		}
	}

	s.emitDeliverySnapshots(snapshots)
	if len(sent) == 0 {
		if lastErr == nil {
			lastErr = errors.New("no capture delivery sent")
		}
		return PublishCaptureChunkResp{}, lastErr
	}
	return PublishCaptureChunkResp{Code: 1, Msg: "ok", SourceID: sourceKey, Sent: len(sent), DeliveryIDs: sent}, nil
}

func (s *StreamService) PublishCaptureChunkSimple(sourceID uint32, req PublishCaptureChunkReq) (PublishCaptureChunkResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStreamTimeout)
	defer cancel()
	return s.PublishCaptureChunk(ctx, sourceID, req)
}

func (s *StreamService) collectTextPublishTargets(sourceID uint32, sourceKey string) ([]producerSendTarget, error) {
	// collectTextPublishTargets 只挑出当前 source 下仍处于 active 的文本 delivery。
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureStateMapsLocked()

	source, ok := s.sources[sourceKey]
	if !ok || source.Producer != sourceID {
		return nil, errors.New("source not found")
	}
	if normalizeRequiredKind(source.Kind) != proto.StreamKindText {
		return nil, errors.New("only text sources support direct input")
	}

	out := make([]producerSendTarget, 0)
	for _, delivery := range s.producerDeliveries {
		if delivery == nil || delivery.SourceID != sourceKey || delivery.Producer != sourceID {
			continue
		}
		if delivery.State != localDeliveryStateActive || delivery.Consumer == 0 {
			continue
		}
		out = append(out, producerSendTarget{
			DeliveryID: delivery.DeliveryID,
			SourceID:   delivery.SourceID,
			Producer:   delivery.Producer,
			Consumer:   delivery.Consumer,
			Position:   delivery.Position,
			UnitMode:   delivery.UnitMode,
		})
	}
	if len(out) == 0 {
		return nil, errors.New("no active deliveries")
	}
	return out, nil
}

func (s *StreamService) collectCapturePublishTargets(sourceID uint32, sourceKey string, deliveryIDs []string) ([]producerSendTarget, error) {
	// collectCapturePublishTargets 限定为桌面视频 chunk delivery，防止把捕获数据发到错误消费端。
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureStateMapsLocked()

	source, ok := s.sources[sourceKey]
	if !ok || source.Producer != sourceID {
		return nil, errors.New("source not found")
	}
	if normalizeRequiredKind(source.Kind) != proto.StreamKindVideo {
		return nil, errors.New("only video sources support desktop capture")
	}
	if strings.TrimSpace(source.UnitMode) != proto.UnitModeChunk {
		return nil, errors.New("desktop capture requires chunk unit mode")
	}
	cfg, hasInput := s.sourceInputs[sourceKey]
	if !hasInput || cfg.InputKind != sourceInputKindDesktop {
		return nil, errors.New("source is not configured for desktop capture")
	}

	allowed := make(map[string]struct{}, len(deliveryIDs))
	for _, deliveryID := range deliveryIDs {
		allowed[deliveryID] = struct{}{}
	}

	out := make([]producerSendTarget, 0, len(deliveryIDs))
	for _, delivery := range s.producerDeliveries {
		if delivery == nil || delivery.SourceID != sourceKey || delivery.Producer != sourceID {
			continue
		}
		if delivery.State != localDeliveryStateActive || delivery.Consumer == 0 {
			continue
		}
		if _, ok := allowed[delivery.DeliveryID]; !ok {
			continue
		}
		if strings.TrimSpace(delivery.UnitMode) != proto.UnitModeChunk {
			return nil, errors.New("desktop capture requires active chunk deliveries")
		}
		out = append(out, producerSendTarget{
			DeliveryID: delivery.DeliveryID,
			SourceID:   delivery.SourceID,
			Producer:   delivery.Producer,
			Consumer:   delivery.Consumer,
			Position:   delivery.Position,
			UnitMode:   delivery.UnitMode,
		})
	}
	if len(out) == 0 {
		return nil, errors.New("no active deliveries")
	}
	return out, nil
}

func normalizeRequestedDeliveryIDs(items []string) []string {
	// normalizeRequestedDeliveryIDs 清理调用方传入的 delivery 列表，保证发送目标唯一且非空。
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		deliveryID := strings.TrimSpace(item)
		if deliveryID == "" {
			continue
		}
		if _, ok := seen[deliveryID]; ok {
			continue
		}
		seen[deliveryID] = struct{}{}
		out = append(out, deliveryID)
	}
	return out
}

func buildDataPayload(deliveryID string, position, ptsMs uint64, body []byte) ([]byte, error) {
	return buildDataPayloadWithFlags(deliveryID, position, ptsMs, 0, body)
}

func buildDataPayloadWithFlags(deliveryID string, position, ptsMs uint64, flags uint8, body []byte) ([]byte, error) {
	// buildDataPayloadWithFlags 负责把通用 body 包装成 stream/data 帧，不关心上层来源是文本还是采集块。
	id, err := uuid.Parse(strings.TrimSpace(deliveryID))
	if err != nil {
		return nil, err
	}
	payload := make([]byte, streamFrameHeaderLen+len(body))
	payload[0] = proto.KindData
	payload[1] = proto.HeaderVersionV1
	payload[2] = flags
	copy(payload[3:19], id[:])
	binary.BigEndian.PutUint64(payload[19:27], position)
	binary.BigEndian.PutUint64(payload[27:35], ptsMs)
	copy(payload[35:], body)
	return payload, nil
}

func (s *StreamService) markTextPublished(target producerSendTarget, bodyLen, ptsMs uint64) (StreamDeliveryEvent, bool) {
	// markTextPublished 在 text 发送成功后同步推进 producer 位置和前端快照统计。
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureStateMapsLocked()

	delivery := s.producerDeliveries[target.DeliveryID]
	source, ok := s.sources[target.SourceID]
	if delivery == nil || !ok {
		return StreamDeliveryEvent{}, false
	}

	delivery.Position = nextExpectedPosition(delivery.UnitMode, target.Position, int(bodyLen))
	delivery.LastActive = time.Now()
	snapshot := s.upsertProducerSnapshotLocked(delivery, source, localDeliveryStateActive, "")
	if rt := s.deliveries[target.DeliveryID]; rt != nil {
		rt.BytesIn += bodyLen
		rt.FramesIn++
		rt.LastPosition = target.Position
		rt.LastPtsMs = ptsMs
		rt.LastFlags = 0
		rt.UpdatedAt = time.Now()
		snapshot = rt.snapshot()
	}
	return snapshot, true
}
