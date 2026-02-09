package presets

import (
	"context"
	"encoding/json"
	"errors"
	"hash/crc32"
	"strconv"
	"strings"
	"time"

	"github.com/yttydcs/myflowhub-core/header"
	"github.com/yttydcs/myflowhub-server/protocol/topicbus"
	"github.com/yttydcs/myflowhub-win/internal/services/transport"
)

const (
	stressPublishName = "stress"
	stressEmitTick    = 200 * time.Millisecond
)

type topicStressSenderState struct {
	active      bool
	sourceID    uint32
	targetID    uint32
	topic       string
	runID       string
	total       int
	payloadSize int
	maxPerSec   int
	sent        int
	errors      int
	startedAt   time.Time
	updatedAt   time.Time
	lastEmit    time.Time
	cancel      context.CancelFunc
}

type topicStressReceiverState struct {
	active      bool
	sourceID    uint32
	targetID    uint32
	topic       string
	runID       string
	expected    int
	payloadSize int
	rx          int
	unique      int
	dup         int
	corrupt     int
	invalid     int
	outOfOrder  int
	lastSeq     int
	startedAt   time.Time
	updatedAt   time.Time
	lastEmit    time.Time
	bitset      []uint64
}

type topicStressPayload struct {
	Run   string `json:"run"`
	Seq   int    `json:"seq"`
	Total int    `json:"total"`
	Size  int    `json:"size"`
	Data  string `json:"data,omitempty"`
	CRC   uint32 `json:"crc"`
}

func (s *PresetService) runTopicStressSender(ctx context.Context, cfg TopicStressConfig) {
	data := ""
	if cfg.PayloadSize > 0 {
		data = strings.Repeat("x", cfg.PayloadSize)
	}
	buf := make([]byte, 0, len(cfg.RunID)+len(data)+64)

	h := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(subProtoTopicBus()).
		WithSourceID(cfg.SourceID).
		WithTargetID(cfg.TargetID)

	interval := time.Duration(0)
	nextSend := time.Time{}
	if cfg.MaxPerSec > 0 {
		interval = time.Second / time.Duration(cfg.MaxPerSec)
		if interval > 0 {
			nextSend = time.Now()
		} else {
			interval = 0
		}
	}

	for seq := 1; seq <= cfg.Total; seq++ {
		select {
		case <-ctx.Done():
			s.finishSender()
			return
		default:
		}
		if interval > 0 {
			now := time.Now()
			if now.Before(nextSend) {
				timer := time.NewTimer(nextSend.Sub(now))
				select {
				case <-ctx.Done():
					timer.Stop()
					s.finishSender()
					return
				case <-timer.C:
				}
			} else {
				nextSend = now
			}
			nextSend = nextSend.Add(interval)
		}

		payload := topicStressPayload{
			Run:   cfg.RunID,
			Seq:   seq,
			Total: cfg.Total,
			Size:  cfg.PayloadSize,
			Data:  data,
		}
		payload.CRC = stressCRC32(&buf, payload.Run, payload.Seq, payload.Total, payload.Data)
		rawPayload, _ := json.Marshal(payload)

		pub := topicbus.PublishReq{
			Topic:   cfg.Topic,
			Name:    stressPublishName,
			TS:      time.Now().UnixMilli(),
			Payload: rawPayload,
		}
		frame, err := transport.EncodeMessage(topicbus.ActionPublish, pub)
		if err != nil {
			s.bumpSenderError()
			continue
		}

		now := time.Now()
		h.WithMsgID(uint32(now.UnixNano())).WithTimestamp(uint32(now.Unix()))
		if s.session == nil || s.session.Send(h, frame) != nil {
			s.bumpSenderError()
		}
		s.bumpSenderSent()
	}
	s.finishSender()
}

func (s *PresetService) bumpSenderSent() {
	shouldEmit := false
	now := time.Now()
	s.mu.Lock()
	if !s.sender.active {
		s.mu.Unlock()
		return
	}
	s.sender.sent++
	s.sender.updatedAt = now
	if s.sender.lastEmit.IsZero() || now.Sub(s.sender.lastEmit) >= stressEmitTick {
		s.sender.lastEmit = now
		shouldEmit = true
	}
	s.mu.Unlock()
	if shouldEmit {
		s.emitSenderStatus()
	}
}

func (s *PresetService) bumpSenderError() {
	shouldEmit := false
	now := time.Now()
	s.mu.Lock()
	if !s.sender.active {
		s.mu.Unlock()
		return
	}
	s.sender.errors++
	s.sender.updatedAt = now
	if s.sender.lastEmit.IsZero() || now.Sub(s.sender.lastEmit) >= stressEmitTick {
		s.sender.lastEmit = now
		shouldEmit = true
	}
	s.mu.Unlock()
	if shouldEmit {
		s.emitSenderStatus()
	}
}

func (s *PresetService) finishSender() {
	s.mu.Lock()
	s.sender.active = false
	s.sender.cancel = nil
	s.sender.updatedAt = time.Now()
	s.mu.Unlock()
	s.emitSenderStatus()
}

func (s *PresetService) handleTopicBusFrame(payload []byte) {
	if len(payload) == 0 {
		return
	}
	var msg topicbus.Message
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	action := strings.ToLower(strings.TrimSpace(msg.Action))
	if action != topicbus.ActionPublish {
		return
	}
	var pub topicbus.PublishReq
	if err := json.Unmarshal(msg.Data, &pub); err != nil {
		return
	}
	s.onTopicStressPublish(pub)
}

func (s *PresetService) onTopicStressPublish(pub topicbus.PublishReq) {
	if pub.Name != stressPublishName {
		return
	}

	s.mu.Lock()
	active := s.receiver.active
	topic := s.receiver.topic
	run := s.receiver.runID
	expect := s.receiver.expected
	expectSize := s.receiver.payloadSize
	s.mu.Unlock()

	if !active || strings.TrimSpace(topic) == "" || strings.TrimSpace(run) == "" || expect <= 0 {
		return
	}
	if pub.Topic != topic {
		return
	}
	if len(pub.Payload) == 0 {
		s.bumpReceiverCounters(false, 0, true, false, false)
		return
	}
	var pl topicStressPayload
	if err := json.Unmarshal(pub.Payload, &pl); err != nil {
		s.bumpReceiverCounters(false, 0, true, false, false)
		return
	}
	if pl.Run != run {
		return
	}

	validSeq := pl.Seq >= 1 && pl.Seq <= expect
	totalOK := pl.Total == expect
	sizeOK := expectSize == 0 || pl.Size == expectSize
	if !validSeq || !totalOK || !sizeOK {
		s.bumpReceiverCounters(false, pl.Seq, false, false, true)
		return
	}

	buf := make([]byte, 0, len(pl.Run)+len(pl.Data)+64)
	want := stressCRC32(&buf, pl.Run, pl.Seq, pl.Total, pl.Data)
	if want != pl.CRC {
		s.bumpReceiverCounters(false, pl.Seq, true, false, false)
		return
	}

	s.bumpReceiverCounters(true, pl.Seq, false, false, false)
}

func (s *PresetService) bumpReceiverCounters(ok bool, seq int, corrupt, parseFail, invalid bool) {
	shouldEmit := false
	now := time.Now()
	s.mu.Lock()
	if !s.receiver.active {
		s.mu.Unlock()
		return
	}
	s.receiver.rx++
	if parseFail || corrupt {
		s.receiver.corrupt++
	} else if invalid {
		s.receiver.invalid++
	} else if ok && seq >= 1 && seq <= s.receiver.expected {
		idx := seq - 1
		word := idx / 64
		bit := uint(idx % 64)
		mask := uint64(1) << bit
		if word < len(s.receiver.bitset) && (s.receiver.bitset[word]&mask) != 0 {
			s.receiver.dup++
		} else {
			if word < len(s.receiver.bitset) {
				s.receiver.bitset[word] |= mask
			}
			s.receiver.unique++
			if s.receiver.lastSeq != 0 && seq < s.receiver.lastSeq {
				s.receiver.outOfOrder++
			}
			s.receiver.lastSeq = seq
		}
	} else {
		s.receiver.invalid++
	}
	s.receiver.updatedAt = now
	if s.receiver.lastEmit.IsZero() || now.Sub(s.receiver.lastEmit) >= stressEmitTick || s.receiver.unique >= s.receiver.expected {
		s.receiver.lastEmit = now
		shouldEmit = true
	}
	s.mu.Unlock()

	if shouldEmit {
		s.emitReceiverStatus()
	}
}

func (s topicStressSenderState) snapshot() TopicStressSenderStatus {
	return TopicStressSenderStatus{
		Active:      s.active,
		Topic:       s.topic,
		RunID:       s.runID,
		Total:       s.total,
		PayloadSize: s.payloadSize,
		MaxPerSec:   s.maxPerSec,
		Sent:        s.sent,
		Errors:      s.errors,
		StartedAt:   s.startedAt,
		UpdatedAt:   s.updatedAt,
	}
}

func (s topicStressReceiverState) snapshot() TopicStressReceiverStatus {
	return TopicStressReceiverStatus{
		Active:      s.active,
		Topic:       s.topic,
		RunID:       s.runID,
		Expected:    s.expected,
		PayloadSize: s.payloadSize,
		Rx:          s.rx,
		Unique:      s.unique,
		Dup:         s.dup,
		Corrupt:     s.corrupt,
		Invalid:     s.invalid,
		OutOfOrder:  s.outOfOrder,
		LastSeq:     s.lastSeq,
		StartedAt:   s.startedAt,
		UpdatedAt:   s.updatedAt,
	}
}

func stressCRC32(buf *[]byte, run string, seq, total int, data string) uint32 {
	b := *buf
	b = b[:0]
	b = append(b, run...)
	b = append(b, '|')
	b = strconv.AppendInt(b, int64(seq), 10)
	b = append(b, '|')
	b = strconv.AppendInt(b, int64(total), 10)
	b = append(b, '|')
	b = append(b, data...)
	*buf = b
	return crc32.ChecksumIEEE(b)
}

func encodeTopicBusSubscribe(topic string) ([]byte, error) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil, errors.New("topic is required")
	}
	return transport.EncodeMessage(topicbus.ActionSubscribe, topicbus.SubscribeReq{Topic: topic})
}

func encodeTopicBusUnsubscribe(topic string) ([]byte, error) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil, errors.New("topic is required")
	}
	return transport.EncodeMessage(topicbus.ActionUnsubscribe, topicbus.SubscribeReq{Topic: topic})
}

func subProtoTopicBus() uint8 {
	return topicbus.SubProtoTopicBus
}
