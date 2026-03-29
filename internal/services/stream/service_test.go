package stream

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	core "github.com/yttydcs/myflowhub-core"
	corebus "github.com/yttydcs/myflowhub-core/eventbus"
	"github.com/yttydcs/myflowhub-core/header"
	proto "github.com/yttydcs/myflowhub-proto/protocol/stream"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
)

type fakeStreamSession struct {
	send func(hdr core.IHeader, payload []byte) error
}

func (f *fakeStreamSession) Send(hdr core.IHeader, payload []byte) error {
	if f == nil || f.send == nil {
		return nil
	}
	return f.send(hdr, payload)
}

func TestAnnounceSendsCtrlPayloadAndAcceptsCtrlResp(t *testing.T) {
	bus := corebus.New(corebus.Options{})
	defer bus.Close()

	var sentHdr core.IHeader
	var sentPayload []byte
	session := &fakeStreamSession{
		send: func(hdr core.IHeader, payload []byte) error {
			sentHdr = hdr
			sentPayload = append([]byte(nil), payload...)

			msg, err := decodeStreamCtrlMessage(payload)
			if err != nil {
				t.Fatalf("decode request payload: %v", err)
			}
			if msg.Action != proto.ActionAnnounce {
				t.Fatalf("unexpected request action: %s", msg.Action)
			}

			var req proto.AnnounceReq
			if err := json.Unmarshal(msg.Data, &req); err != nil {
				t.Fatalf("decode announce request: %v", err)
			}
			if req.Source.SourceID != "source-1" {
				t.Fatalf("unexpected source id: %s", req.Source.SourceID)
			}

			respPayload, err := encodeStreamCtrlPayload(proto.ActionAnnounceResp, proto.AnnounceResp{
				ReqID: req.ReqID,
				Code:  1,
				Source: &proto.SourceDescriptor{
					SourceID: "source-1",
					Producer: 7,
					Kind:     proto.StreamKindText,
				},
			})
			if err != nil {
				t.Fatalf("encode response payload: %v", err)
			}

			bus.PublishSync(context.Background(), sessionsvc.EventFrame, sessionsvc.FrameEvent{
				Major:    header.MajorOKResp,
				SubProto: proto.SubProtoStream,
				MsgID:    hdr.GetMsgID(),
				Payload:  respPayload,
			}, nil)
			return nil
		},
	}

	svc := &StreamService{
		session:    session,
		bus:        bus,
		deliveries: make(map[string]*deliveryRuntime),
	}

	resp, err := svc.Announce(context.Background(), 7, 9, proto.AnnounceReq{
		ReqID: "announce-1",
		Source: proto.SourceDescriptor{
			SourceID: "source-1",
			Producer: 7,
			Kind:     proto.StreamKindText,
		},
	})
	if err != nil {
		t.Fatalf("announce returned err: %v", err)
	}
	if resp.Code != 1 || resp.Source == nil || resp.Source.SourceID != "source-1" {
		t.Fatalf("unexpected announce resp: %+v", resp)
	}
	if sentHdr == nil {
		t.Fatal("expected request header to be captured")
	}
	if sentHdr.Major() != header.MajorCmd || sentHdr.SubProto() != proto.SubProtoStream {
		t.Fatalf("unexpected request header: major=%d sub=%d", sentHdr.Major(), sentHdr.SubProto())
	}
	if sentHdr.SourceID() != 7 || sentHdr.TargetID() != 9 || sentHdr.GetMsgID() == 0 {
		t.Fatalf("unexpected request routing: src=%d tgt=%d msg=%d", sentHdr.SourceID(), sentHdr.TargetID(), sentHdr.GetMsgID())
	}
	if len(sentPayload) == 0 || sentPayload[0] != proto.KindCtrl {
		t.Fatalf("expected ctrl payload prefix, got %v", sentPayload)
	}
}

func TestAnnounceTimeoutMapsToUIError(t *testing.T) {
	bus := corebus.New(corebus.Options{})
	defer bus.Close()

	svc := &StreamService{
		session: &fakeStreamSession{
			send: func(hdr core.IHeader, payload []byte) error {
				return nil
			},
		},
		bus:        bus,
		deliveries: make(map[string]*deliveryRuntime),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := svc.Announce(ctx, 7, 9, proto.AnnounceReq{
		ReqID: "announce-timeout",
		Source: proto.SourceDescriptor{
			SourceID: "source-timeout",
			Producer: 7,
			Kind:     proto.StreamKindText,
		},
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if got := err.Error(); got != "stream announce: request timed out" {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestHandleDataPublishesTextEventAndSnapshot(t *testing.T) {
	bus := corebus.New(corebus.Options{})
	defer bus.Close()

	svc := New(nil, nil, bus)
	defer svc.Close()

	deliveryID := uuid.NewString()
	svc.trackAcceptedDelivery(
		deliveryID,
		&proto.SourceDescriptor{SourceID: "src-text", Producer: 11, Kind: proto.StreamKindText},
		&proto.ConsumerDescriptor{ConsumerID: "consumer-text", Consumer: 22, Kind: proto.StreamKindText},
		11,
		22,
		"consumer-text",
		true,
	)

	textCh := make(chan StreamTextEvent, 1)
	token := bus.Subscribe(EventStreamText, func(_ context.Context, evt corebus.Event) {
		if data, ok := evt.Data.(StreamTextEvent); ok {
			textCh <- data
		}
	})
	defer bus.Unsubscribe(EventStreamText, token)

	payload := buildDataPayload(t, deliveryID, 5, 120, []byte("hello stream"))
	svc.handleFrame(sessionsvc.FrameEvent{
		Major:    header.MajorMsg,
		SubProto: proto.SubProtoStream,
		Payload:  payload,
	})

	select {
	case evt := <-textCh:
		if evt.DeliveryID != deliveryID {
			t.Fatalf("unexpected delivery id: %s", evt.DeliveryID)
		}
		if evt.Text != "hello stream" {
			t.Fatalf("unexpected text: %q", evt.Text)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected text event")
	}

	snapshot := svc.DeliverySnapshot()
	if len(snapshot) != 1 {
		t.Fatalf("expected 1 delivery snapshot, got %d", len(snapshot))
	}
	if snapshot[0].BytesIn != uint64(len("hello stream")) {
		t.Fatalf("expected bytesIn to update, got %d", snapshot[0].BytesIn)
	}
	if snapshot[0].FramesIn != 1 {
		t.Fatalf("expected framesIn to update, got %d", snapshot[0].FramesIn)
	}
	if snapshot[0].LastPosition != 5 {
		t.Fatalf("expected last position 5, got %d", snapshot[0].LastPosition)
	}
}

func TestHandleAckPublishesStatsEvent(t *testing.T) {
	bus := corebus.New(corebus.Options{})
	defer bus.Close()

	svc := New(nil, nil, bus)
	defer svc.Close()

	deliveryID := uuid.NewString()
	svc.trackAcceptedDelivery(
		deliveryID,
		&proto.SourceDescriptor{SourceID: "src-video", Producer: 11, Kind: proto.StreamKindVideo},
		&proto.ConsumerDescriptor{ConsumerID: "consumer-video", Consumer: 22, Kind: proto.StreamKindVideo},
		11,
		22,
		"consumer-video",
		true,
	)

	statsCh := make(chan StreamStatsEvent, 1)
	token := bus.Subscribe(EventStreamStats, func(_ context.Context, evt corebus.Event) {
		if data, ok := evt.Data.(StreamStatsEvent); ok {
			statsCh <- data
		}
	})
	defer bus.Unsubscribe(EventStreamStats, token)

	payload := buildAckPayload(t, deliveryID, 64)
	svc.handleFrame(sessionsvc.FrameEvent{
		Major:    header.MajorMsg,
		SubProto: proto.SubProtoStream,
		Payload:  payload,
	})

	select {
	case evt := <-statsCh:
		if evt.DeliveryID != deliveryID {
			t.Fatalf("unexpected delivery id: %s", evt.DeliveryID)
		}
		if evt.LastAckPos != 64 {
			t.Fatalf("expected ack position 64, got %d", evt.LastAckPos)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected stats event")
	}

	snapshot := svc.DeliverySnapshot()
	if len(snapshot) != 1 {
		t.Fatalf("expected 1 delivery snapshot, got %d", len(snapshot))
	}
	if snapshot[0].LastAckPos != 64 {
		t.Fatalf("expected snapshot ack position 64, got %d", snapshot[0].LastAckPos)
	}
}

func buildDataPayload(t *testing.T, deliveryID string, position, pts uint64, body []byte) []byte {
	t.Helper()
	id := uuid.MustParse(deliveryID)
	payload := make([]byte, streamFrameHeaderLen+len(body))
	payload[0] = proto.KindData
	payload[1] = proto.HeaderVersionV1
	payload[2] = 0
	copy(payload[3:19], id[:])
	binary.BigEndian.PutUint64(payload[19:27], position)
	binary.BigEndian.PutUint64(payload[27:35], pts)
	copy(payload[35:], body)
	return payload
}

func buildAckPayload(t *testing.T, deliveryID string, position uint64) []byte {
	t.Helper()
	id := uuid.MustParse(deliveryID)
	payload := make([]byte, streamAckHeaderLen)
	payload[0] = proto.KindAck
	payload[1] = proto.HeaderVersionV1
	payload[2] = 0
	copy(payload[3:19], id[:])
	binary.BigEndian.PutUint64(payload[19:27], position)
	return payload
}
