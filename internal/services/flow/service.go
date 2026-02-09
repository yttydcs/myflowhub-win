package flow

import (
	"context"
	"errors"
	"strings"

	"github.com/yttydcs/myflowhub-server/protocol/flow"
	"github.com/yttydcs/myflowhub-win/internal/services/logs"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
	"github.com/yttydcs/myflowhub-win/internal/services/transport"
)

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
	return s.send(ctx, sourceID, targetID, payload, "set", req.FlowID)
}

func (s *FlowService) SetSimple(sourceID, targetID uint32, req flow.SetReq) error {
	return s.Set(context.Background(), sourceID, targetID, req)
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
	return s.send(ctx, sourceID, targetID, payload, "run", req.FlowID)
}

func (s *FlowService) RunSimple(sourceID, targetID uint32, req flow.RunReq) error {
	return s.Run(context.Background(), sourceID, targetID, req)
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
	return s.send(ctx, sourceID, targetID, payload, "status", req.FlowID)
}

func (s *FlowService) StatusSimple(sourceID, targetID uint32, req flow.StatusReq) error {
	return s.Status(context.Background(), sourceID, targetID, req)
}

func (s *FlowService) List(ctx context.Context, sourceID, targetID uint32, req flow.ListReq) error {
	if strings.TrimSpace(req.ReqID) == "" {
		return errors.New("req_id is required")
	}
	payload, err := transport.EncodeMessage(flow.ActionList, req)
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "list", "")
}

func (s *FlowService) ListSimple(sourceID, targetID uint32, req flow.ListReq) error {
	return s.List(context.Background(), sourceID, targetID, req)
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
	return s.send(ctx, sourceID, targetID, payload, "get", req.FlowID)
}

func (s *FlowService) GetSimple(sourceID, targetID uint32, req flow.GetReq) error {
	return s.Get(context.Background(), sourceID, targetID, req)
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
