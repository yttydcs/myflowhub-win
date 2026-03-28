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
		if frame.Major != header.MajorMsg {
			return
		}
		s.handleFrame(frame)
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
		s.handleData(frame.Payload)
	case proto.KindAck:
		s.handleAck(frame.Payload)
	default:
		return
	}
}

func (s *StreamService) handleData(payload []byte) {
	packet, err := parseDataPacket(payload)
	if err != nil {
		s.logWarn("stream data parse failed: %v", err)
		return
	}
	now := time.Now()

	s.mu.Lock()
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
}

func (s *StreamService) handleAck(payload []byte) {
	packet, err := parseAckPacket(payload)
	if err != nil {
		s.logWarn("stream ack parse failed: %v", err)
		return
	}
	now := time.Now()
	s.mu.Lock()
	rt := s.ensureDeliveryLocked(packet.DeliveryID)
	rt.LastAckPos = packet.Position
	rt.LastFlags = packet.Flags
	rt.State = "active"
	rt.UpdatedAt = now
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
	now := time.Now()
	for id, item := range s.deliveries {
		if !match(item) {
			continue
		}
		delete(s.deliveries, id)
		item.State = "closed"
		item.UpdatedAt = now
		closed = append(closed, item.snapshot())
	}
	s.mu.Unlock()
	for _, evt := range closed {
		s.emitDelivery(evt)
	}
}

func (s *StreamService) markAllClosed(reason string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "disconnected"
	}
	s.mu.Lock()
	evts := make([]StreamDeliveryEvent, 0, len(s.deliveries))
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
	s.mu.Unlock()
	for _, evt := range evts {
		s.emitDelivery(evt)
	}
}

func (s *StreamService) ensureDeliveryLocked(deliveryID string) *deliveryRuntime {
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
