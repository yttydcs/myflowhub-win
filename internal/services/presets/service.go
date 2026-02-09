package presets

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub-core/eventbus"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
)

type PresetService struct {
	session *sessionsvc.SessionService
	bus     eventbus.IBus

	mu sync.Mutex

	sender   topicStressSenderState
	receiver topicStressReceiverState

	busTokens []busToken
}

type busToken struct {
	name  string
	token string
}

func New(session *sessionsvc.SessionService, bus eventbus.IBus) *PresetService {
	svc := &PresetService{session: session, bus: bus}
	svc.bindBus()
	return svc
}

func (s *PresetService) Close() {
	s.unbindBus()
	s.StopTopicStressSender()
	s.StopTopicStressReceiver()
}

func (s *PresetService) TopicStressSenderState() (TopicStressSenderStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sender.snapshot(), nil
}

func (s *PresetService) TopicStressReceiverState() (TopicStressReceiverStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.receiver.snapshot(), nil
}

func (s *PresetService) StartTopicStressSender(cfg TopicStressConfig) error {
	cfg = normalizeStressConfig(cfg)
	if err := validateStressConfig(cfg); err != nil {
		return err
	}
	if s.session == nil {
		return errors.New("session not initialized")
	}
	s.mu.Lock()
	if s.sender.active {
		s.mu.Unlock()
		return errors.New("sender already active")
	}
	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now()
	s.sender = topicStressSenderState{
		active:      true,
		sourceID:    cfg.SourceID,
		targetID:    cfg.TargetID,
		topic:       cfg.Topic,
		runID:       cfg.RunID,
		total:       cfg.Total,
		payloadSize: cfg.PayloadSize,
		maxPerSec:   cfg.MaxPerSec,
		startedAt:   now,
		updatedAt:   now,
		cancel:      cancel,
	}
	s.mu.Unlock()
	s.emitSenderStatus()
	go s.runTopicStressSender(ctx, cfg)
	return nil
}

func (s *PresetService) StopTopicStressSender() {
	var cancel context.CancelFunc
	s.mu.Lock()
	cancel = s.sender.cancel
	s.sender.cancel = nil
	s.sender.active = false
	s.sender.updatedAt = time.Now()
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.emitSenderStatus()
}

func (s *PresetService) StartTopicStressReceiver(cfg TopicStressConfig) error {
	cfg = normalizeStressConfig(cfg)
	if err := validateStressConfig(cfg); err != nil {
		return err
	}
	if s.session == nil {
		return errors.New("session not initialized")
	}
	s.mu.Lock()
	if s.receiver.active {
		s.mu.Unlock()
		return errors.New("receiver already active")
	}
	s.mu.Unlock()
	payload, err := encodeTopicBusSubscribe(cfg.Topic)
	if err != nil {
		return err
	}
	if err := s.session.SendCommand(subProtoTopicBus(), cfg.SourceID, cfg.TargetID, payload); err != nil {
		return err
	}
	s.mu.Lock()
	now := time.Now()
	s.receiver = topicStressReceiverState{
		active:      true,
		sourceID:    cfg.SourceID,
		targetID:    cfg.TargetID,
		topic:       cfg.Topic,
		runID:       cfg.RunID,
		expected:    cfg.Total,
		payloadSize: cfg.PayloadSize,
		startedAt:   now,
		updatedAt:   now,
		bitset:      make([]uint64, (cfg.Total+63)/64),
	}
	s.mu.Unlock()
	s.emitReceiverStatus()
	return nil
}

func (s *PresetService) StopTopicStressReceiver() {
	s.mu.Lock()
	sourceID := s.receiver.sourceID
	targetID := s.receiver.targetID
	topic := s.receiver.topic
	s.receiver.active = false
	s.receiver.updatedAt = time.Now()
	s.mu.Unlock()

	if s.session != nil && strings.TrimSpace(topic) != "" && sourceID != 0 && targetID != 0 {
		if payload, err := encodeTopicBusUnsubscribe(topic); err == nil {
			_ = s.session.SendCommand(subProtoTopicBus(), sourceID, targetID, payload)
		}
	}
	s.emitReceiverStatus()
}

func (s *PresetService) ResetTopicStressReceiver() {
	s.mu.Lock()
	s.receiver = topicStressReceiverState{}
	s.mu.Unlock()
	s.emitReceiverStatus()
}

func (s *PresetService) bindBus() {
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
		if frame.SubProto != subProtoTopicBus() {
			return
		}
		s.handleTopicBusFrame(frame.Payload)
	})
}

func (s *PresetService) unbindBus() {
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

func validateStressConfig(cfg TopicStressConfig) error {
	if cfg.SourceID == 0 {
		return errors.New("source_id is required")
	}
	if cfg.TargetID == 0 {
		return errors.New("target_id is required")
	}
	if strings.TrimSpace(cfg.Topic) == "" {
		return errors.New("topic is required")
	}
	if strings.TrimSpace(cfg.RunID) == "" {
		return errors.New("run_id is required")
	}
	if cfg.Total <= 0 {
		return errors.New("total must be positive")
	}
	if cfg.PayloadSize < 0 {
		return errors.New("payload_size must be non-negative")
	}
	if cfg.MaxPerSec < 0 {
		return errors.New("max_per_sec must be non-negative")
	}
	return nil
}

func normalizeStressConfig(cfg TopicStressConfig) TopicStressConfig {
	cfg.Topic = strings.TrimSpace(cfg.Topic)
	cfg.RunID = strings.TrimSpace(cfg.RunID)
	return cfg
}

func (s *PresetService) emitSenderStatus() {
	if s == nil || s.bus == nil {
		return
	}
	status := s.senderSnapshot()
	_ = s.bus.Publish(context.Background(), EventTopicStressSender, status, nil)
}

func (s *PresetService) emitReceiverStatus() {
	if s == nil || s.bus == nil {
		return
	}
	status := s.receiverSnapshot()
	_ = s.bus.Publish(context.Background(), EventTopicStressReceiver, status, nil)
}

func (s *PresetService) senderSnapshot() TopicStressSenderStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sender.snapshot()
}

func (s *PresetService) receiverSnapshot() TopicStressReceiverStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.receiver.snapshot()
}
