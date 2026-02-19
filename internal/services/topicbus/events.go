package topicbus

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/yttydcs/myflowhub-core/eventbus"
	"github.com/yttydcs/myflowhub-core/header"
	protocol "github.com/yttydcs/myflowhub-proto/protocol/topicbus"
	sdktransport "github.com/yttydcs/myflowhub-sdk/transport"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
)

const EventTopicBusEvent = "topicbus.event"

type busToken struct {
	name  string
	token string
}

func (s *TopicBusService) bindBus() {
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
		if frame.Major != header.MajorMsg {
			return
		}
		if frame.SubProto != protocol.SubProtoTopicBus {
			return
		}
		s.handleFrame(frame.Payload)
	})
}

func (s *TopicBusService) unbindBus() {
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

func (s *TopicBusService) handleFrame(payload []byte) {
	if s == nil || s.bus == nil {
		return
	}
	msg, err := sdktransport.DecodeMessage(payload)
	if err != nil {
		return
	}
	if msg.Action != protocol.ActionPublish {
		return
	}
	if len(msg.Data) == 0 {
		return
	}
	var data protocol.PublishReq
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return
	}
	data.Topic = strings.TrimSpace(data.Topic)
	data.Name = strings.TrimSpace(data.Name)
	if data.Topic == "" || data.Name == "" {
		return
	}
	_ = s.bus.Publish(context.Background(), EventTopicBusEvent, data, nil)
}
