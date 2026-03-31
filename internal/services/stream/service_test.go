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

type sentFrame struct {
	Major    uint8
	SubProto uint8
	SourceID uint32
	TargetID uint32
	MsgID    uint32
	Payload  []byte
}

type recordingStreamSession struct {
	frames []sentFrame
	send   func(hdr core.IHeader, payload []byte) error
}

func (r *recordingStreamSession) Send(hdr core.IHeader, payload []byte) error {
	if hdr != nil {
		r.frames = append(r.frames, sentFrame{
			Major:    hdr.Major(),
			SubProto: hdr.SubProto(),
			SourceID: hdr.SourceID(),
			TargetID: hdr.TargetID(),
			MsgID:    hdr.GetMsgID(),
			Payload:  append([]byte(nil), payload...),
		})
	}
	if r.send != nil {
		return r.send(hdr, payload)
	}
	return nil
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

	payload := buildTestDataPayload(t, deliveryID, 5, 120, []byte("hello stream"))
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

func TestAnnounceCompletesWhenOwnerRequestLoopsBackToLocalWin(t *testing.T) {
	bus := corebus.New(corebus.Options{})
	defer bus.Close()

	session := &recordingStreamSession{}
	sendCount := 0
	session.send = func(hdr core.IHeader, payload []byte) error {
		sendCount++
		switch sendCount {
		case 1:
			if hdr.Major() != header.MajorCmd {
				t.Fatalf("expected cmd request, got major=%d", hdr.Major())
			}
			bus.PublishSync(context.Background(), sessionsvc.EventFrame, sessionsvc.FrameEvent{
				Major:    header.MajorCmd,
				SubProto: proto.SubProtoStream,
				SourceID: hdr.SourceID(),
				TargetID: hdr.SourceID(),
				MsgID:    hdr.GetMsgID(),
				Payload:  append([]byte(nil), payload...),
			}, nil)
		case 2:
			if hdr.Major() != header.MajorOKResp {
				t.Fatalf("expected ok response, got major=%d", hdr.Major())
			}
			bus.PublishSync(context.Background(), sessionsvc.EventFrame, sessionsvc.FrameEvent{
				Major:    header.MajorOKResp,
				SubProto: proto.SubProtoStream,
				SourceID: hdr.SourceID(),
				TargetID: hdr.TargetID(),
				MsgID:    hdr.GetMsgID(),
				Payload:  append([]byte(nil), payload...),
			}, nil)
		default:
			t.Fatalf("unexpected send count: %d", sendCount)
		}
		return nil
	}

	svc := &StreamService{
		session:    session,
		bus:        bus,
		deliveries: make(map[string]*deliveryRuntime),
	}
	svc.bindBus()
	defer svc.unbindBus()

	resp, err := svc.Announce(context.Background(), 7, 9, proto.AnnounceReq{
		ReqID: "announce-loopback",
		Source: proto.SourceDescriptor{
			SourceID: "source-loopback",
			Producer: 7,
			Kind:     proto.StreamKindText,
		},
	})
	if err != nil {
		t.Fatalf("announce returned err: %v", err)
	}
	if resp.Code != 1 || resp.Source == nil || resp.Source.SourceID != "source-loopback" {
		t.Fatalf("unexpected announce resp: %+v", resp)
	}
	if got := svc.handleListSourcesLocal(proto.ListSourcesReq{ReqID: "list", Producer: 7}); len(got.Sources) != 1 || got.Sources[0].SourceID != "source-loopback" {
		t.Fatalf("expected local source catalog to contain source-loopback, got %+v", got)
	}
	if got := svc.handleGetSourceLocal(proto.GetSourceReq{ReqID: "get", Producer: 7, SourceID: "source-loopback"}); got.Source == nil || got.Source.SourceID != "source-loopback" {
		t.Fatalf("expected get_source to return local source, got %+v", got)
	}
}

func TestAnnounceConsumerCompletesWhenOwnerRequestLoopsBackToLocalWin(t *testing.T) {
	bus := corebus.New(corebus.Options{})
	defer bus.Close()

	session := &recordingStreamSession{}
	sendCount := 0
	session.send = func(hdr core.IHeader, payload []byte) error {
		sendCount++
		switch sendCount {
		case 1:
			bus.PublishSync(context.Background(), sessionsvc.EventFrame, sessionsvc.FrameEvent{
				Major:    header.MajorCmd,
				SubProto: proto.SubProtoStream,
				SourceID: hdr.SourceID(),
				TargetID: hdr.SourceID(),
				MsgID:    hdr.GetMsgID(),
				Payload:  append([]byte(nil), payload...),
			}, nil)
		case 2:
			bus.PublishSync(context.Background(), sessionsvc.EventFrame, sessionsvc.FrameEvent{
				Major:    header.MajorOKResp,
				SubProto: proto.SubProtoStream,
				SourceID: hdr.SourceID(),
				TargetID: hdr.TargetID(),
				MsgID:    hdr.GetMsgID(),
				Payload:  append([]byte(nil), payload...),
			}, nil)
		default:
			t.Fatalf("unexpected send count: %d", sendCount)
		}
		return nil
	}

	svc := &StreamService{
		session:    session,
		bus:        bus,
		deliveries: make(map[string]*deliveryRuntime),
	}
	svc.bindBus()
	defer svc.unbindBus()

	resp, err := svc.AnnounceConsumer(context.Background(), 7, 9, proto.AnnounceConsumerReq{
		ReqID: "announce-consumer-loopback",
		ConsumerEndpoint: proto.ConsumerDescriptor{
			ConsumerID: "consumer-loopback",
			Consumer:   7,
			Kind:       proto.StreamKindText,
		},
	})
	if err != nil {
		t.Fatalf("announce consumer returned err: %v", err)
	}
	if resp.Code != 1 || resp.ConsumerEndpoint == nil || resp.ConsumerEndpoint.ConsumerID != "consumer-loopback" {
		t.Fatalf("unexpected announce consumer resp: %+v", resp)
	}
	if got := svc.handleListConsumersLocal(proto.ListConsumersReq{ReqID: "list", Consumer: 7}); len(got.ConsumerEndpoints) != 1 || got.ConsumerEndpoints[0].ConsumerID != "consumer-loopback" {
		t.Fatalf("expected local consumer catalog to contain consumer-loopback, got %+v", got)
	}
	if got := svc.handleGetConsumerLocal(proto.GetConsumerReq{ReqID: "get", Consumer: 7, ConsumerID: "consumer-loopback"}); got.ConsumerEndpoint == nil || got.ConsumerEndpoint.ConsumerID != "consumer-loopback" {
		t.Fatalf("expected get_consumer to return local consumer, got %+v", got)
	}
}

func TestLocalConsumerDeliveryLifecycleSendsAckAndCloses(t *testing.T) {
	bus := corebus.New(corebus.Options{})
	defer bus.Close()

	session := &recordingStreamSession{}
	svc := &StreamService{
		session:    session,
		bus:        bus,
		deliveries: make(map[string]*deliveryRuntime),
	}
	svc.bindBus()
	defer svc.unbindBus()

	consumerResp := svc.handleAnnounceConsumerLocal(21, proto.AnnounceConsumerReq{
		ReqID: "local-consumer",
		ConsumerEndpoint: proto.ConsumerDescriptor{
			ConsumerID: "consumer-1",
			Consumer:   21,
			Kind:       proto.StreamKindText,
		},
	})
	if consumerResp.Code != 1 {
		t.Fatalf("announce consumer failed: %+v", consumerResp)
	}

	textCh := make(chan StreamTextEvent, 1)
	token := bus.Subscribe(EventStreamText, func(_ context.Context, evt corebus.Event) {
		if data, ok := evt.Data.(StreamTextEvent); ok {
			textCh <- data
		}
	})
	defer bus.Unsubscribe(EventStreamText, token)

	deliveryID := uuid.NewString()
	preparePayload, err := encodeStreamCtrlPayload(actionDeliveryPrepare, deliveryPrepareReq{
		ReqID:      "prepare-1",
		TxnID:      "txn-1",
		DeliveryID: deliveryID,
		Role:       deliveryRoleConsumer,
		Producer:   99,
		SourceID:   "source-1",
		Consumer:   21,
		ConsumerID: "consumer-1",
		Kind:       proto.StreamKindText,
		UnitMode:   proto.UnitModeFrame,
	})
	if err != nil {
		t.Fatalf("encode prepare payload: %v", err)
	}
	bus.PublishSync(context.Background(), sessionsvc.EventFrame, sessionsvc.FrameEvent{
		Major:    header.MajorCmd,
		SubProto: proto.SubProtoStream,
		SourceID: 99,
		TargetID: 21,
		MsgID:    11,
		Payload:  preparePayload,
	}, nil)

	activatePayload, err := encodeStreamCtrlPayload(actionDeliveryActivate, deliveryActivateReq{
		ReqID:      "activate-1",
		TxnID:      "txn-1",
		DeliveryID: deliveryID,
		Role:       deliveryRoleConsumer,
	})
	if err != nil {
		t.Fatalf("encode activate payload: %v", err)
	}
	bus.PublishSync(context.Background(), sessionsvc.EventFrame, sessionsvc.FrameEvent{
		Major:    header.MajorCmd,
		SubProto: proto.SubProtoStream,
		SourceID: 99,
		TargetID: 21,
		MsgID:    12,
		Payload:  activatePayload,
	}, nil)

	bus.PublishSync(context.Background(), sessionsvc.EventFrame, sessionsvc.FrameEvent{
		Major:    header.MajorMsg,
		SubProto: proto.SubProtoStream,
		SourceID: 99,
		TargetID: 21,
		Payload:  buildTestDataPayload(t, deliveryID, 5, 120, []byte("hello local consumer")),
	}, nil)

	select {
	case evt := <-textCh:
		if evt.DeliveryID != deliveryID || evt.Text != "hello local consumer" {
			t.Fatalf("unexpected text event: %+v", evt)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected text event")
	}

	if len(session.frames) < 3 {
		t.Fatalf("expected prepare resp, activate resp, and ack, got %d sends", len(session.frames))
	}
	ack := session.frames[2]
	if ack.Major != header.MajorMsg || ack.SourceID != 21 || ack.TargetID != 99 {
		t.Fatalf("unexpected ack routing: %+v", ack)
	}
	ackPacket, err := parseAckPacket(ack.Payload)
	if err != nil {
		t.Fatalf("parse ack payload: %v", err)
	}
	if ackPacket.DeliveryID != deliveryID || ackPacket.Position != 6 {
		t.Fatalf("unexpected ack packet: %+v", ackPacket)
	}

	snapshot := svc.DeliverySnapshot()
	if len(snapshot) != 1 || snapshot[0].LastAckPos != 6 || snapshot[0].FramesIn != 1 {
		t.Fatalf("unexpected delivery snapshot after data: %+v", snapshot)
	}

	closePayload, err := encodeStreamCtrlPayload(actionDeliveryClose, deliveryCloseReq{
		ReqID:      "close-1",
		TxnID:      "txn-1",
		DeliveryID: deliveryID,
		Role:       deliveryRoleConsumer,
		Reason:     "test close",
	})
	if err != nil {
		t.Fatalf("encode close payload: %v", err)
	}
	bus.PublishSync(context.Background(), sessionsvc.EventFrame, sessionsvc.FrameEvent{
		Major:    header.MajorCmd,
		SubProto: proto.SubProtoStream,
		SourceID: 99,
		TargetID: 21,
		MsgID:    13,
		Payload:  closePayload,
	}, nil)

	if got := svc.DeliverySnapshot(); len(got) != 0 {
		t.Fatalf("expected close to remove delivery snapshot, got %+v", got)
	}
}

func TestLocalProducerDeliveryLifecycleTracksAckAndCloses(t *testing.T) {
	bus := corebus.New(corebus.Options{})
	defer bus.Close()

	session := &recordingStreamSession{}
	svc := &StreamService{
		session:    session,
		bus:        bus,
		deliveries: make(map[string]*deliveryRuntime),
	}
	svc.bindBus()
	defer svc.unbindBus()

	sourceResp := svc.handleAnnounceLocal(21, proto.AnnounceReq{
		ReqID: "local-source",
		Source: proto.SourceDescriptor{
			SourceID: "source-1",
			Producer: 21,
			Kind:     proto.StreamKindText,
			Mode:     proto.ModeLive,
			UnitMode: proto.UnitModeFrame,
		},
	})
	if sourceResp.Code != 1 {
		t.Fatalf("announce source failed: %+v", sourceResp)
	}

	deliveryID := uuid.NewString()
	preparePayload, err := encodeStreamCtrlPayload(actionDeliveryPrepare, deliveryPrepareReq{
		ReqID:      "prepare-1",
		TxnID:      "txn-1",
		DeliveryID: deliveryID,
		Role:       deliveryRoleProducer,
		Producer:   21,
		SourceID:   "source-1",
		Consumer:   99,
		ConsumerID: "consumer-remote",
	})
	if err != nil {
		t.Fatalf("encode prepare payload: %v", err)
	}
	bus.PublishSync(context.Background(), sessionsvc.EventFrame, sessionsvc.FrameEvent{
		Major:    header.MajorCmd,
		SubProto: proto.SubProtoStream,
		SourceID: 99,
		TargetID: 21,
		MsgID:    21,
		Payload:  preparePayload,
	}, nil)

	activatePayload, err := encodeStreamCtrlPayload(actionDeliveryActivate, deliveryActivateReq{
		ReqID:      "activate-1",
		TxnID:      "txn-1",
		DeliveryID: deliveryID,
		Role:       deliveryRoleProducer,
	})
	if err != nil {
		t.Fatalf("encode activate payload: %v", err)
	}
	bus.PublishSync(context.Background(), sessionsvc.EventFrame, sessionsvc.FrameEvent{
		Major:    header.MajorCmd,
		SubProto: proto.SubProtoStream,
		SourceID: 99,
		TargetID: 21,
		MsgID:    22,
		Payload:  activatePayload,
	}, nil)

	bus.PublishSync(context.Background(), sessionsvc.EventFrame, sessionsvc.FrameEvent{
		Major:    header.MajorMsg,
		SubProto: proto.SubProtoStream,
		SourceID: 99,
		TargetID: 21,
		Payload:  buildAckPayload(t, deliveryID, 64),
	}, nil)

	snapshot := svc.DeliverySnapshot()
	if len(snapshot) != 1 || snapshot[0].LastAckPos != 64 || snapshot[0].State != "active" {
		t.Fatalf("unexpected producer snapshot after ack: %+v", snapshot)
	}

	closePayload, err := encodeStreamCtrlPayload(actionDeliveryClose, deliveryCloseReq{
		ReqID:      "close-1",
		TxnID:      "txn-1",
		DeliveryID: deliveryID,
		Role:       deliveryRoleProducer,
		Reason:     "test close",
	})
	if err != nil {
		t.Fatalf("encode close payload: %v", err)
	}
	bus.PublishSync(context.Background(), sessionsvc.EventFrame, sessionsvc.FrameEvent{
		Major:    header.MajorCmd,
		SubProto: proto.SubProtoStream,
		SourceID: 99,
		TargetID: 21,
		MsgID:    23,
		Payload:  closePayload,
	}, nil)

	if got := svc.DeliverySnapshot(); len(got) != 0 {
		t.Fatalf("expected close to remove producer delivery snapshot, got %+v", got)
	}
}

func TestPublishTextSimpleSendsDataToActiveProducerDeliveries(t *testing.T) {
	session := &recordingStreamSession{}
	svc := &StreamService{
		session:    session,
		deliveries: make(map[string]*deliveryRuntime),
	}

	sourceResp := svc.handleAnnounceLocal(21, proto.AnnounceReq{
		ReqID: "source-1",
		Source: proto.SourceDescriptor{
			SourceID: "source-text",
			Producer: 21,
			Kind:     proto.StreamKindText,
			Mode:     proto.ModeLive,
			UnitMode: proto.UnitModeFrame,
		},
	})
	if sourceResp.Code != 1 {
		t.Fatalf("announce source failed: %+v", sourceResp)
	}

	deliveryID := uuid.NewString()
	prepareResp := svc.prepareProducerLocal(21, deliveryPrepareReq{
		ReqID:      "prepare-1",
		TxnID:      "txn-1",
		DeliveryID: deliveryID,
		Role:       deliveryRoleProducer,
		Producer:   21,
		SourceID:   "source-text",
		Consumer:   99,
		ConsumerID: "consumer-1",
	})
	if prepareResp.Code != 1 {
		t.Fatalf("prepare producer failed: %+v", prepareResp)
	}
	activateResp := svc.handleDeliveryActivateLocal(deliveryActivateReq{
		ReqID:      "activate-1",
		TxnID:      "txn-1",
		DeliveryID: deliveryID,
		Role:       deliveryRoleProducer,
	})
	if activateResp.Code != 1 {
		t.Fatalf("activate producer failed: %+v", activateResp)
	}

	resp, err := svc.PublishTextSimple(21, PublishTextReq{SourceID: "source-text", Text: "hello producer"})
	if err != nil {
		t.Fatalf("PublishTextSimple() error = %v", err)
	}
	if resp.Code != 1 || resp.Sent != 1 || len(resp.DeliveryIDs) != 1 || resp.DeliveryIDs[0] != deliveryID {
		t.Fatalf("unexpected publish response: %+v", resp)
	}
	if len(session.frames) != 1 {
		t.Fatalf("expected 1 data frame got %d", len(session.frames))
	}
	sent := session.frames[0]
	if sent.Major != header.MajorMsg || sent.SubProto != proto.SubProtoStream || sent.SourceID != 21 || sent.TargetID != 99 {
		t.Fatalf("unexpected sent frame routing: %+v", sent)
	}
	packet, err := parseDataPacket(sent.Payload)
	if err != nil {
		t.Fatalf("parse data payload: %v", err)
	}
	if packet.DeliveryID != deliveryID || packet.Position != 0 || string(packet.Body) != "hello producer" {
		t.Fatalf("unexpected packet: %+v", packet)
	}

	snapshot := svc.DeliverySnapshot()
	if len(snapshot) != 1 || snapshot[0].FramesIn != 1 || snapshot[0].BytesIn != uint64(len("hello producer")) || snapshot[0].LastPosition != 0 {
		t.Fatalf("unexpected snapshot after publish: %+v", snapshot)
	}
}

func TestPublishTextSimpleRejectsSourceWithoutActiveDeliveries(t *testing.T) {
	svc := &StreamService{
		session:    &recordingStreamSession{},
		deliveries: make(map[string]*deliveryRuntime),
	}

	sourceResp := svc.handleAnnounceLocal(21, proto.AnnounceReq{
		ReqID: "source-1",
		Source: proto.SourceDescriptor{
			SourceID: "source-text",
			Producer: 21,
			Kind:     proto.StreamKindText,
		},
	})
	if sourceResp.Code != 1 {
		t.Fatalf("announce source failed: %+v", sourceResp)
	}

	_, err := svc.PublishTextSimple(21, PublishTextReq{SourceID: "source-text", Text: "hello"})
	if err == nil || err.Error() != "no active deliveries" {
		t.Fatalf("expected no active deliveries error got %v", err)
	}
}

func buildTestDataPayload(t *testing.T, deliveryID string, position, pts uint64, body []byte) []byte {
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
