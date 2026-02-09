package varpool

import (
	"context"
	"errors"
	"strings"

	"github.com/yttydcs/myflowhub-server/protocol/varstore"
	"github.com/yttydcs/myflowhub-win/internal/services/logs"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
	"github.com/yttydcs/myflowhub-win/internal/services/transport"
)

type VarPoolService struct {
	session *sessionsvc.SessionService
	logs    *logs.LogService
}

func New(session *sessionsvc.SessionService, logsSvc *logs.LogService) *VarPoolService {
	return &VarPoolService{session: session, logs: logsSvc}
}

func (s *VarPoolService) Set(ctx context.Context, sourceID, targetID uint32, req varstore.SetReq) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	payload, err := transport.EncodeMessage(varstore.ActionSet, req)
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "set", req.Name)
}

func (s *VarPoolService) SetSimple(sourceID, targetID uint32, req varstore.SetReq) error {
	return s.Set(context.Background(), sourceID, targetID, req)
}

func (s *VarPoolService) Get(ctx context.Context, sourceID, targetID uint32, req varstore.GetReq) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	payload, err := transport.EncodeMessage(varstore.ActionGet, req)
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "get", req.Name)
}

func (s *VarPoolService) GetSimple(sourceID, targetID uint32, req varstore.GetReq) error {
	return s.Get(context.Background(), sourceID, targetID, req)
}

func (s *VarPoolService) List(ctx context.Context, sourceID, targetID uint32, req varstore.ListReq) error {
	payload, err := transport.EncodeMessage(varstore.ActionList, req)
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "list", "")
}

func (s *VarPoolService) ListSimple(sourceID, targetID uint32, req varstore.ListReq) error {
	return s.List(context.Background(), sourceID, targetID, req)
}

func (s *VarPoolService) Revoke(ctx context.Context, sourceID, targetID uint32, req varstore.GetReq) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	payload, err := transport.EncodeMessage(varstore.ActionRevoke, req)
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "revoke", req.Name)
}

func (s *VarPoolService) RevokeSimple(sourceID, targetID uint32, req varstore.GetReq) error {
	return s.Revoke(context.Background(), sourceID, targetID, req)
}

func (s *VarPoolService) Subscribe(ctx context.Context, sourceID, targetID uint32, req varstore.SubscribeReq) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	if req.Owner == 0 {
		return errors.New("owner is required")
	}
	payload, err := transport.EncodeMessage(varstore.ActionSubscribe, req)
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "subscribe", req.Name)
}

func (s *VarPoolService) SubscribeSimple(sourceID, targetID uint32, req varstore.SubscribeReq) error {
	return s.Subscribe(context.Background(), sourceID, targetID, req)
}

func (s *VarPoolService) Unsubscribe(ctx context.Context, sourceID, targetID uint32, req varstore.SubscribeReq) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	if req.Owner == 0 {
		return errors.New("owner is required")
	}
	payload, err := transport.EncodeMessage(varstore.ActionUnsubscribe, req)
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "unsubscribe", req.Name)
}

func (s *VarPoolService) UnsubscribeSimple(sourceID, targetID uint32, req varstore.SubscribeReq) error {
	return s.Unsubscribe(context.Background(), sourceID, targetID, req)
}

func (s *VarPoolService) Send(ctx context.Context, sourceID, targetID uint32, action string, data any) error {
	payload, err := transport.EncodeMessage(action, data)
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, action, "")
}

func (s *VarPoolService) SendSimple(sourceID, targetID uint32, action string, data any) error {
	return s.Send(context.Background(), sourceID, targetID, action, data)
}

func (s *VarPoolService) send(_ context.Context, sourceID, targetID uint32, payload []byte, action, name string) error {
	if s.session == nil {
		return errors.New("session service not initialized")
	}
	if err := s.session.SendCommand(varstore.SubProtoVarStore, sourceID, targetID, payload); err != nil {
		return err
	}
	if s.logs != nil {
		if name != "" {
			s.logs.Appendf("info", "varpool %s sent name=%s", action, name)
		} else {
			s.logs.Appendf("info", "varpool %s sent", action)
		}
	}
	return nil
}
