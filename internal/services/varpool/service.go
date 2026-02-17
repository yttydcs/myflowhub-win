package varpool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yttydcs/myflowhub-proto/protocol/varstore"
	"github.com/yttydcs/myflowhub-win/internal/services/logs"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
	"github.com/yttydcs/myflowhub-win/internal/services/transport"
)

const defaultVarPoolTimeout = 8 * time.Second

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
	return s.sendAndAwait(ctx, sourceID, targetID, payload, varstore.ActionSet, varstore.ActionSetResp, req.Name)
}

func (s *VarPoolService) SetSimple(sourceID, targetID uint32, req varstore.SetReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultVarPoolTimeout)
	defer cancel()
	return s.Set(ctx, sourceID, targetID, req)
}

func (s *VarPoolService) Get(ctx context.Context, sourceID, targetID uint32, req varstore.GetReq) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	payload, err := transport.EncodeMessage(varstore.ActionGet, req)
	if err != nil {
		return err
	}
	return s.sendAndAwait(ctx, sourceID, targetID, payload, varstore.ActionGet, varstore.ActionGetResp, req.Name)
}

func (s *VarPoolService) GetSimple(sourceID, targetID uint32, req varstore.GetReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultVarPoolTimeout)
	defer cancel()
	return s.Get(ctx, sourceID, targetID, req)
}

func (s *VarPoolService) List(ctx context.Context, sourceID, targetID uint32, req varstore.ListReq) error {
	payload, err := transport.EncodeMessage(varstore.ActionList, req)
	if err != nil {
		return err
	}
	return s.sendAndAwait(ctx, sourceID, targetID, payload, varstore.ActionList, varstore.ActionListResp, "")
}

func (s *VarPoolService) ListSimple(sourceID, targetID uint32, req varstore.ListReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultVarPoolTimeout)
	defer cancel()
	return s.List(ctx, sourceID, targetID, req)
}

func (s *VarPoolService) Revoke(ctx context.Context, sourceID, targetID uint32, req varstore.GetReq) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	payload, err := transport.EncodeMessage(varstore.ActionRevoke, req)
	if err != nil {
		return err
	}
	return s.sendAndAwait(ctx, sourceID, targetID, payload, varstore.ActionRevoke, varstore.ActionRevokeResp, req.Name)
}

func (s *VarPoolService) RevokeSimple(sourceID, targetID uint32, req varstore.GetReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultVarPoolTimeout)
	defer cancel()
	return s.Revoke(ctx, sourceID, targetID, req)
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
	return s.sendAndAwait(ctx, sourceID, targetID, payload, varstore.ActionSubscribe, varstore.ActionSubscribeResp, req.Name)
}

func (s *VarPoolService) SubscribeSimple(sourceID, targetID uint32, req varstore.SubscribeReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultVarPoolTimeout)
	defer cancel()
	return s.Subscribe(ctx, sourceID, targetID, req)
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
	return s.sendAndAwait(ctx, sourceID, targetID, payload, varstore.ActionUnsubscribe, varstore.ActionSubscribeResp, req.Name)
}

func (s *VarPoolService) UnsubscribeSimple(sourceID, targetID uint32, req varstore.SubscribeReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultVarPoolTimeout)
	defer cancel()
	return s.Unsubscribe(ctx, sourceID, targetID, req)
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

func (s *VarPoolService) sendAndAwait(ctx context.Context, sourceID, targetID uint32, payload []byte, reqAction, respAction, name string) error {
	if s.session == nil {
		return errors.New("session service not initialized")
	}
	trimmedAction := strings.TrimSpace(reqAction)
	trimmedName := strings.TrimSpace(name)

	resp, err := s.session.SendCommandAndAwait(ctx, varstore.SubProtoVarStore, sourceID, targetID, payload, respAction)
	if err != nil {
		return fmt.Errorf("varpool %s await: %w", trimmedAction, err)
	}

	var out varstore.VarResp
	if err := json.Unmarshal(resp.Message.Data, &out); err != nil {
		return err
	}
	if out.Code != 1 {
		msg := strings.TrimSpace(out.Msg)
		if msg != "" {
			return fmt.Errorf("%s (code=%d)", msg, out.Code)
		}
		return fmt.Errorf("varpool %s failed (code=%d)", trimmedAction, out.Code)
	}
	if s.logs != nil {
		if trimmedName != "" {
			s.logs.Appendf("info", "varpool %s ok name=%s", trimmedAction, trimmedName)
		} else {
			s.logs.Appendf("info", "varpool %s ok", trimmedAction)
		}
	}
	return nil
}
