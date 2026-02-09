package debug

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/yttydcs/myflowhub-core/header"
	"github.com/yttydcs/myflowhub-win/internal/services/logs"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
)

type DebugFrame struct {
	Major     uint8  `json:"major"`
	SubProto  uint8  `json:"sub_proto"`
	SourceID  uint32 `json:"source_id"`
	TargetID  uint32 `json:"target_id"`
	Flags     uint8  `json:"flags"`
	MsgID     uint32 `json:"msg_id"`
	Timestamp uint32 `json:"timestamp"`
}

type DebugService struct {
	session *sessionsvc.SessionService
	logs    *logs.LogService
}

func New(session *sessionsvc.SessionService, logsSvc *logs.LogService) *DebugService {
	return &DebugService{session: session, logs: logsSvc}
}

func (s *DebugService) Send(ctx context.Context, frame DebugFrame, payload string, payloadIsHex bool) error {
	if frame.Major == 0 {
		return errors.New("major is required")
	}
	if frame.SubProto == 0 {
		return errors.New("sub_proto is required")
	}
	body, err := decodePayload(payload, payloadIsHex)
	if err != nil {
		return err
	}
	msgID := frame.MsgID
	if msgID == 0 {
		msgID = uint32(time.Now().UnixNano())
	}
	ts := frame.Timestamp
	if ts == 0 {
		ts = uint32(time.Now().Unix())
	}
	hdr := (&header.HeaderTcp{}).
		WithMajor(frame.Major).
		WithSubProto(frame.SubProto).
		WithSourceID(frame.SourceID).
		WithTargetID(frame.TargetID).
		WithFlags(frame.Flags).
		WithMsgID(msgID).
		WithTimestamp(ts)
	if s.session == nil {
		return errors.New("session service not initialized")
	}
	if err := s.session.Send(hdr, body); err != nil {
		return err
	}
	if s.logs != nil {
		s.logs.Appendf("info", "debug send major=%d sub=%d len=%d", frame.Major, frame.SubProto, len(body))
	}
	return nil
}

func decodePayload(payload string, payloadIsHex bool) ([]byte, error) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return nil, nil
	}
	if !payloadIsHex {
		return []byte(payload), nil
	}
	clean := strings.ReplaceAll(payload, " ", "")
	clean = strings.ReplaceAll(clean, "\n", "")
	clean = strings.ReplaceAll(clean, "\t", "")
	if len(clean)%2 != 0 {
		return nil, errors.New("hex payload must be even length")
	}
	out := make([]byte, len(clean)/2)
	_, err := hex.Decode(out, []byte(clean))
	return out, err
}
