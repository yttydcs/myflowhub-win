package topicbus

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/yttydcs/myflowhub-server/protocol/topicbus"
	"github.com/yttydcs/myflowhub-win/internal/services/logs"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
	"github.com/yttydcs/myflowhub-win/internal/services/transport"
)

type TopicBusService struct {
	session *sessionsvc.SessionService
	logs    *logs.LogService
}

func New(session *sessionsvc.SessionService, logsSvc *logs.LogService) *TopicBusService {
	return &TopicBusService{session: session, logs: logsSvc}
}

func (s *TopicBusService) Subscribe(ctx context.Context, sourceID, targetID uint32, topic string) error {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return errors.New("topic is required")
	}
	payload, err := transport.EncodeMessage(topicbus.ActionSubscribe, topicbus.SubscribeReq{Topic: topic})
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "subscribe", topic)
}

func (s *TopicBusService) SubscribeSimple(sourceID, targetID uint32, topic string) error {
	return s.Subscribe(context.Background(), sourceID, targetID, topic)
}

func (s *TopicBusService) SubscribeBatch(ctx context.Context, sourceID, targetID uint32, topics []string) error {
	topics = normalizeTopics(topics)
	if len(topics) == 0 {
		return errors.New("topics are required")
	}
	payload, err := transport.EncodeMessage(topicbus.ActionSubscribeBatch, topicbus.SubscribeBatchReq{Topics: topics})
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "subscribe_batch", "")
}

func (s *TopicBusService) SubscribeBatchSimple(sourceID, targetID uint32, topics []string) error {
	return s.SubscribeBatch(context.Background(), sourceID, targetID, topics)
}

func (s *TopicBusService) Unsubscribe(ctx context.Context, sourceID, targetID uint32, topic string) error {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return errors.New("topic is required")
	}
	payload, err := transport.EncodeMessage(topicbus.ActionUnsubscribe, topicbus.SubscribeReq{Topic: topic})
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "unsubscribe", topic)
}

func (s *TopicBusService) UnsubscribeSimple(sourceID, targetID uint32, topic string) error {
	return s.Unsubscribe(context.Background(), sourceID, targetID, topic)
}

func (s *TopicBusService) UnsubscribeBatch(ctx context.Context, sourceID, targetID uint32, topics []string) error {
	topics = normalizeTopics(topics)
	if len(topics) == 0 {
		return errors.New("topics are required")
	}
	payload, err := transport.EncodeMessage(topicbus.ActionUnsubscribeBatch, topicbus.SubscribeBatchReq{Topics: topics})
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "unsubscribe_batch", "")
}

func (s *TopicBusService) UnsubscribeBatchSimple(sourceID, targetID uint32, topics []string) error {
	return s.UnsubscribeBatch(context.Background(), sourceID, targetID, topics)
}

func (s *TopicBusService) ListSubs(ctx context.Context, sourceID, targetID uint32) error {
	payload, err := transport.EncodeMessage(topicbus.ActionListSubs, map[string]any{})
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "list_subs", "")
}

func (s *TopicBusService) ListSubsSimple(sourceID, targetID uint32) error {
	return s.ListSubs(context.Background(), sourceID, targetID)
}

func (s *TopicBusService) Publish(ctx context.Context, sourceID, targetID uint32, topic, name, payloadText string) error {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return errors.New("topic is required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name is required")
	}
	payload := normalizePayload(payloadText)
	data := topicbus.PublishReq{
		Topic:   topic,
		Name:    name,
		TS:      time.Now().UnixMilli(),
		Payload: payload,
	}
	body, err := transport.EncodeMessage(topicbus.ActionPublish, data)
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, body, "publish", topic)
}

func (s *TopicBusService) PublishSimple(sourceID, targetID uint32, topic, name, payloadText string) error {
	return s.Publish(context.Background(), sourceID, targetID, topic, name, payloadText)
}

func (s *TopicBusService) Send(ctx context.Context, sourceID, targetID uint32, action string, data any) error {
	payload, err := transport.EncodeMessage(action, data)
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, action, "")
}

func (s *TopicBusService) SendSimple(sourceID, targetID uint32, action string, data any) error {
	return s.Send(context.Background(), sourceID, targetID, action, data)
}

func (s *TopicBusService) send(_ context.Context, sourceID, targetID uint32, payload []byte, action, topic string) error {
	if s.session == nil {
		return errors.New("session service not initialized")
	}
	if err := s.session.SendCommand(topicbus.SubProtoTopicBus, sourceID, targetID, payload); err != nil {
		return err
	}
	if s.logs != nil {
		if topic != "" {
			s.logs.Appendf("info", "topicbus %s sent topic=%s", action, topic)
		} else {
			s.logs.Appendf("info", "topicbus %s sent", action)
		}
	}
	return nil
}

func normalizeTopics(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, t := range in {
		name := strings.TrimSpace(t)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func normalizePayload(payloadText string) json.RawMessage {
	payloadText = strings.TrimSpace(payloadText)
	if payloadText == "" {
		return nil
	}
	if json.Valid([]byte(payloadText)) {
		return json.RawMessage(payloadText)
	}
	wrapped, _ := json.Marshal(payloadText)
	return wrapped
}
