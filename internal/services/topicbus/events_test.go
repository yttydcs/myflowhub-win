// 本文件覆盖 `topicbus` 后端服务事件载荷的行为。

package topicbus

import (
	"context"
	"testing"
	"time"

	corebus "github.com/yttydcs/myflowhub-core/eventbus"
	"github.com/yttydcs/myflowhub-core/header"
	protocol "github.com/yttydcs/myflowhub-proto/protocol/topicbus"
	sdktransport "github.com/yttydcs/myflowhub-sdk/transport"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
)

func TestTopicBusService_MajorCmd_PublishEvent(t *testing.T) {
	assertTopicBusPublishEvent(t, header.MajorCmd)
}

func TestTopicBusService_MajorMsg_PublishEvent(t *testing.T) {
	assertTopicBusPublishEvent(t, header.MajorMsg)
}

func assertTopicBusPublishEvent(t *testing.T, major uint8) {
	t.Helper()

	bus := corebus.New(corebus.Options{})
	defer bus.Close()

	svc := New(nil, nil, bus)
	defer svc.Close()

	got := make(chan protocol.PublishReq, 1)
	bus.Subscribe(EventTopicBusEvent, func(_ context.Context, evt corebus.Event) {
		req, ok := evt.Data.(protocol.PublishReq)
		if !ok {
			return
		}
		select {
		case got <- req:
		default:
		}
	})

	payload, err := sdktransport.EncodeMessage(protocol.ActionPublish, protocol.PublishReq{
		Topic:   "dev.codex.msg",
		Name:    "codex.win-topic-test",
		TS:      1770000000000,
		Payload: []byte(`{"body":"hello"}`),
	})
	if err != nil {
		t.Fatalf("encode message: %v", err)
	}

	bus.PublishSync(context.Background(), sessionsvc.EventFrame, sessionsvc.FrameEvent{
		Major:    major,
		SubProto: protocol.SubProtoTopicBus,
		Payload:  payload,
	}, nil)

	select {
	case req := <-got:
		if req.Topic != "dev.codex.msg" || req.Name != "codex.win-topic-test" || string(req.Payload) != `{"body":"hello"}` {
			t.Fatalf("unexpected publish event: %+v", req)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for topicbus.event")
	}
}
