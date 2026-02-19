package varpool

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/yttydcs/myflowhub-core/eventbus"
	"github.com/yttydcs/myflowhub-core/header"
	"github.com/yttydcs/myflowhub-proto/protocol/varstore"
	sdktransport "github.com/yttydcs/myflowhub-sdk/transport"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
)

const (
	EventVarPoolChanged = "varpool.changed"
	EventVarPoolDeleted = "varpool.deleted"
)

type busToken struct {
	name  string
	token string
}

func (s *VarPoolService) bindBus() {
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
		if frame.SubProto != varstore.SubProtoVarStore {
			return
		}
		s.handleFrame(frame.Payload)
	})
}

func (s *VarPoolService) unbindBus() {
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

func (s *VarPoolService) handleFrame(payload []byte) {
	if s == nil || s.bus == nil {
		return
	}
	msg, err := sdktransport.DecodeMessage(payload)
	if err != nil {
		return
	}

	switch msg.Action {
	case varstore.ActionNotifySet, varstore.ActionUpSet, varstore.ActionVarChanged:
		s.publishVarEvent(EventVarPoolChanged, msg.Data)
	case varstore.ActionNotifyRevoke, varstore.ActionUpRevoke, varstore.ActionVarDeleted:
		s.publishVarEvent(EventVarPoolDeleted, msg.Data)
	default:
		return
	}
}

func (s *VarPoolService) publishVarEvent(name string, raw json.RawMessage) {
	if s == nil || s.bus == nil || len(raw) == 0 {
		return
	}
	var out varstore.VarResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return
	}
	out.Name = strings.TrimSpace(out.Name)
	if out.Name == "" || out.Owner == 0 {
		return
	}
	_ = s.bus.Publish(context.Background(), name, out, nil)
}
