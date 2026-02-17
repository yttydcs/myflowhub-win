package flow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yttydcs/myflowhub-proto/protocol/flow"
	"github.com/yttydcs/myflowhub-win/internal/services/logs"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
	"github.com/yttydcs/myflowhub-win/internal/services/transport"
)

const defaultFlowTimeout = 8 * time.Second

type FlowService struct {
	session *sessionsvc.SessionService
	logs    *logs.LogService
}

func New(session *sessionsvc.SessionService, logsSvc *logs.LogService) *FlowService {
	return &FlowService{session: session, logs: logsSvc}
}

func (s *FlowService) Set(ctx context.Context, sourceID, targetID uint32, req flow.SetReq) error {
	if strings.TrimSpace(req.ReqID) == "" {
		return errors.New("req_id is required")
	}
	if strings.TrimSpace(req.FlowID) == "" {
		return errors.New("flow_id is required")
	}
	payload, err := transport.EncodeMessage(flow.ActionSet, req)
	if err != nil {
		return err
	}
	var resp flow.SetResp
	return s.sendAndAwait(ctx, sourceID, targetID, payload, flow.ActionSet, flow.ActionSetResp, &resp, req.FlowID)
}

func (s *FlowService) SetSimple(sourceID, targetID uint32, req flow.SetReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultFlowTimeout)
	defer cancel()
	return s.Set(ctx, sourceID, targetID, req)
}

func (s *FlowService) Run(ctx context.Context, sourceID, targetID uint32, req flow.RunReq) error {
	if strings.TrimSpace(req.ReqID) == "" {
		return errors.New("req_id is required")
	}
	if strings.TrimSpace(req.FlowID) == "" {
		return errors.New("flow_id is required")
	}
	payload, err := transport.EncodeMessage(flow.ActionRun, req)
	if err != nil {
		return err
	}
	var resp flow.RunResp
	return s.sendAndAwait(ctx, sourceID, targetID, payload, flow.ActionRun, flow.ActionRunResp, &resp, req.FlowID)
}

func (s *FlowService) RunSimple(sourceID, targetID uint32, req flow.RunReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultFlowTimeout)
	defer cancel()
	return s.Run(ctx, sourceID, targetID, req)
}

func (s *FlowService) Status(ctx context.Context, sourceID, targetID uint32, req flow.StatusReq) error {
	if strings.TrimSpace(req.ReqID) == "" {
		return errors.New("req_id is required")
	}
	if strings.TrimSpace(req.FlowID) == "" {
		return errors.New("flow_id is required")
	}
	payload, err := transport.EncodeMessage(flow.ActionStatus, req)
	if err != nil {
		return err
	}
	var resp flow.StatusResp
	return s.sendAndAwait(ctx, sourceID, targetID, payload, flow.ActionStatus, flow.ActionStatusResp, &resp, req.FlowID)
}

func (s *FlowService) StatusSimple(sourceID, targetID uint32, req flow.StatusReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultFlowTimeout)
	defer cancel()
	return s.Status(ctx, sourceID, targetID, req)
}

func (s *FlowService) List(ctx context.Context, sourceID, targetID uint32, req flow.ListReq) error {
	if strings.TrimSpace(req.ReqID) == "" {
		return errors.New("req_id is required")
	}
	payload, err := transport.EncodeMessage(flow.ActionList, req)
	if err != nil {
		return err
	}
	var resp flow.ListResp
	return s.sendAndAwait(ctx, sourceID, targetID, payload, flow.ActionList, flow.ActionListResp, &resp, "")
}

func (s *FlowService) ListSimple(sourceID, targetID uint32, req flow.ListReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultFlowTimeout)
	defer cancel()
	return s.List(ctx, sourceID, targetID, req)
}

func (s *FlowService) Get(ctx context.Context, sourceID, targetID uint32, req flow.GetReq) error {
	if strings.TrimSpace(req.ReqID) == "" {
		return errors.New("req_id is required")
	}
	if strings.TrimSpace(req.FlowID) == "" {
		return errors.New("flow_id is required")
	}
	payload, err := transport.EncodeMessage(flow.ActionGet, req)
	if err != nil {
		return err
	}
	var resp flow.GetResp
	return s.sendAndAwait(ctx, sourceID, targetID, payload, flow.ActionGet, flow.ActionGetResp, &resp, req.FlowID)
}

func (s *FlowService) GetSimple(sourceID, targetID uint32, req flow.GetReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultFlowTimeout)
	defer cancel()
	return s.Get(ctx, sourceID, targetID, req)
}

func (s *FlowService) Send(ctx context.Context, sourceID, targetID uint32, action string, data any) error {
	payload, err := transport.EncodeMessage(action, data)
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, action, "")
}

func (s *FlowService) send(_ context.Context, sourceID, targetID uint32, payload []byte, action, flowID string) error {
	if s.session == nil {
		return errors.New("session service not initialized")
	}
	if err := s.session.SendCommand(flow.SubProtoFlow, sourceID, targetID, payload); err != nil {
		return err
	}
	if s.logs != nil {
		if flowID != "" {
			s.logs.Appendf("info", "flow %s sent flow_id=%s", action, flowID)
		} else {
			s.logs.Appendf("info", "flow %s sent", action)
		}
	}
	return nil
}

func (s *FlowService) sendAndAwait(ctx context.Context, sourceID, targetID uint32, payload []byte, reqAction, respAction string, out any, flowID string) error {
	if s.session == nil {
		return errors.New("session service not initialized")
	}
	if out == nil {
		return errors.New("flow out is required")
	}
	trimmedAction := strings.TrimSpace(reqAction)
	trimmedFlowID := strings.TrimSpace(flowID)

	resp, err := s.session.SendCommandAndAwait(ctx, flow.SubProtoFlow, sourceID, targetID, payload, respAction)
	if err != nil {
		return fmt.Errorf("flow %s await: %w", trimmedAction, err)
	}

	if err := json.Unmarshal(resp.Message.Data, out); err != nil {
		return err
	}
	code, msg := extractCodeMsg(out)
	if code != 1 {
		msg = strings.TrimSpace(msg)
		if msg != "" {
			return fmt.Errorf("%s (code=%d)", msg, code)
		}
		if trimmedFlowID != "" {
			return fmt.Errorf("flow %s failed flow_id=%s (code=%d)", trimmedAction, trimmedFlowID, code)
		}
		return fmt.Errorf("flow %s failed (code=%d)", trimmedAction, code)
	}

	if s.logs != nil {
		if trimmedFlowID != "" {
			s.logs.Appendf("info", "flow %s ok flow_id=%s", trimmedAction, trimmedFlowID)
		} else {
			s.logs.Appendf("info", "flow %s ok", trimmedAction)
		}
	}
	return nil
}

func extractCodeMsg(v any) (int, string) {
	switch t := v.(type) {
	case *flow.SetResp:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	case *flow.RunResp:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	case *flow.StatusResp:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	case *flow.ListResp:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	case *flow.GetResp:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	default:
		return 0, ""
	}
}
