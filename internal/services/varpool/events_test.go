// 本文件覆盖 `varpool` 后端服务事件载荷的行为。

package varpool

import (
	"context"
	"testing"
	"time"

	corebus "github.com/yttydcs/myflowhub-core/eventbus"
	"github.com/yttydcs/myflowhub-core/header"
	"github.com/yttydcs/myflowhub-proto/protocol/varstore"
	sdktransport "github.com/yttydcs/myflowhub-sdk/transport"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
)

func TestVarPoolService_MajorCmd_VarChanged(t *testing.T) {
	bus := corebus.New(corebus.Options{})
	defer bus.Close()

	svc := New(nil, nil, bus)
	defer svc.Close()

	got := make(chan varstore.VarResp, 1)
	bus.Subscribe(EventVarPoolChanged, func(_ context.Context, evt corebus.Event) {
		resp, ok := evt.Data.(varstore.VarResp)
		if !ok {
			return
		}
		select {
		case got <- resp:
		default:
		}
	})

	payload, err := sdktransport.EncodeMessage(varstore.ActionVarChanged, varstore.VarResp{
		Code:       1,
		Msg:        "ok",
		Name:       "sys_volume_percent",
		Value:      "21",
		Owner:      7,
		Visibility: varstore.VisibilityPublic,
		Type:       "string",
	})
	if err != nil {
		t.Fatalf("encode message: %v", err)
	}

	bus.PublishSync(context.Background(), sessionsvc.EventFrame, sessionsvc.FrameEvent{
		Major:    header.MajorCmd,
		SubProto: varstore.SubProtoVarStore,
		Payload:  payload,
	}, nil)

	select {
	case resp := <-got:
		if resp.Owner != 7 || resp.Name != "sys_volume_percent" || resp.Value != "21" {
			t.Fatalf("unexpected resp: %+v", resp)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for varpool.changed")
	}
}

func TestVarPoolService_MajorCmd_VarDeleted(t *testing.T) {
	bus := corebus.New(corebus.Options{})
	defer bus.Close()

	svc := New(nil, nil, bus)
	defer svc.Close()

	got := make(chan varstore.VarResp, 1)
	bus.Subscribe(EventVarPoolDeleted, func(_ context.Context, evt corebus.Event) {
		resp, ok := evt.Data.(varstore.VarResp)
		if !ok {
			return
		}
		select {
		case got <- resp:
		default:
		}
	})

	payload, err := sdktransport.EncodeMessage(varstore.ActionVarDeleted, varstore.VarResp{
		Code:  1,
		Msg:   "ok",
		Name:  "sys_volume_percent",
		Owner: 7,
	})
	if err != nil {
		t.Fatalf("encode message: %v", err)
	}

	bus.PublishSync(context.Background(), sessionsvc.EventFrame, sessionsvc.FrameEvent{
		Major:    header.MajorCmd,
		SubProto: varstore.SubProtoVarStore,
		Payload:  payload,
	}, nil)

	select {
	case resp := <-got:
		if resp.Owner != 7 || resp.Name != "sys_volume_percent" {
			t.Fatalf("unexpected resp: %+v", resp)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for varpool.deleted")
	}
}
