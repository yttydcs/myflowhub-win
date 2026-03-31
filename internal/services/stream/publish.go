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

type producerSendTarget struct {
	DeliveryID string
	SourceID   string
	Producer   uint32
	Consumer   uint32
	Position   uint64
	UnitMode   string
}

func (s *StreamService) PublishText(ctx context.Context, sourceID uint32, req PublishTextReq) (PublishTextResp, error) {
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

func (s *StreamService) collectTextPublishTargets(sourceID uint32, sourceKey string) ([]producerSendTarget, error) {
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

func buildDataPayload(deliveryID string, position, ptsMs uint64, body []byte) ([]byte, error) {
	id, err := uuid.Parse(strings.TrimSpace(deliveryID))
	if err != nil {
		return nil, err
	}
	payload := make([]byte, streamFrameHeaderLen+len(body))
	payload[0] = proto.KindData
	payload[1] = proto.HeaderVersionV1
	payload[2] = 0
	copy(payload[3:19], id[:])
	binary.BigEndian.PutUint64(payload[19:27], position)
	binary.BigEndian.PutUint64(payload[27:35], ptsMs)
	copy(payload[35:], body)
	return payload, nil
}

func (s *StreamService) markTextPublished(target producerSendTarget, bodyLen, ptsMs uint64) (StreamDeliveryEvent, bool) {
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
