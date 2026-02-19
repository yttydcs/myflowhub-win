package varpool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	corebus "github.com/yttydcs/myflowhub-core/eventbus"
	"github.com/yttydcs/myflowhub-proto/protocol/varstore"
	"github.com/yttydcs/myflowhub-win/internal/services/logs"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
	"github.com/yttydcs/myflowhub-win/internal/services/transport"
)

const defaultVarPoolTimeout = 8 * time.Second

type VarPoolService struct {
	session *sessionsvc.SessionService
	logs    *logs.LogService
	bus     corebus.IBus

	busTokens []busToken
}

func New(session *sessionsvc.SessionService, logsSvc *logs.LogService, bus corebus.IBus) *VarPoolService {
	svc := &VarPoolService{session: session, logs: logsSvc, bus: bus}
	svc.bindBus()
	return svc
}

func (s *VarPoolService) Close() {
	s.unbindBus()
}

func (s *VarPoolService) Set(ctx context.Context, sourceID, targetID uint32, req varstore.SetReq) (varstore.VarResp, error) {
	if strings.TrimSpace(req.Name) == "" {
		return varstore.VarResp{}, errors.New("name is required")
	}
	payload, err := transport.EncodeMessage(varstore.ActionSet, req)
	if err != nil {
		return varstore.VarResp{}, err
	}
	return s.sendAndAwait(ctx, sourceID, targetID, payload, varstore.ActionSet, varstore.ActionSetResp, req.Name)
}

func (s *VarPoolService) SetSimple(sourceID, targetID uint32, req varstore.SetReq) (varstore.VarResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultVarPoolTimeout)
	defer cancel()
	return s.Set(ctx, sourceID, targetID, req)
}

func (s *VarPoolService) Get(ctx context.Context, sourceID, targetID uint32, req varstore.GetReq) (varstore.VarResp, error) {
	if strings.TrimSpace(req.Name) == "" {
		return varstore.VarResp{}, errors.New("name is required")
	}
	payload, err := transport.EncodeMessage(varstore.ActionGet, req)
	if err != nil {
		return varstore.VarResp{}, err
	}
	return s.sendAndAwait(ctx, sourceID, targetID, payload, varstore.ActionGet, varstore.ActionGetResp, req.Name)
}

func (s *VarPoolService) GetSimple(sourceID, targetID uint32, req varstore.GetReq) (varstore.VarResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultVarPoolTimeout)
	defer cancel()
	return s.Get(ctx, sourceID, targetID, req)
}

func (s *VarPoolService) List(ctx context.Context, sourceID, targetID uint32, req varstore.ListReq) (varstore.VarResp, error) {
	payload, err := transport.EncodeMessage(varstore.ActionList, req)
	if err != nil {
		return varstore.VarResp{}, err
	}
	return s.sendAndAwait(ctx, sourceID, targetID, payload, varstore.ActionList, varstore.ActionListResp, "")
}

func (s *VarPoolService) ListSimple(sourceID, targetID uint32, req varstore.ListReq) (varstore.VarResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultVarPoolTimeout)
	defer cancel()
	return s.List(ctx, sourceID, targetID, req)
}

func (s *VarPoolService) Revoke(ctx context.Context, sourceID, targetID uint32, req varstore.GetReq) (varstore.VarResp, error) {
	if strings.TrimSpace(req.Name) == "" {
		return varstore.VarResp{}, errors.New("name is required")
	}
	payload, err := transport.EncodeMessage(varstore.ActionRevoke, req)
	if err != nil {
		return varstore.VarResp{}, err
	}
	return s.sendAndAwait(ctx, sourceID, targetID, payload, varstore.ActionRevoke, varstore.ActionRevokeResp, req.Name)
}

func (s *VarPoolService) RevokeSimple(sourceID, targetID uint32, req varstore.GetReq) (varstore.VarResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultVarPoolTimeout)
	defer cancel()
	return s.Revoke(ctx, sourceID, targetID, req)
}

func (s *VarPoolService) Subscribe(ctx context.Context, sourceID, targetID uint32, req varstore.SubscribeReq) (varstore.VarResp, error) {
	if strings.TrimSpace(req.Name) == "" {
		return varstore.VarResp{}, errors.New("name is required")
	}
	if req.Owner == 0 {
		return varstore.VarResp{}, errors.New("owner is required")
	}
	payload, err := transport.EncodeMessage(varstore.ActionSubscribe, req)
	if err != nil {
		return varstore.VarResp{}, err
	}
	return s.sendAndAwait(ctx, sourceID, targetID, payload, varstore.ActionSubscribe, varstore.ActionSubscribeResp, req.Name)
}

func (s *VarPoolService) SubscribeSimple(sourceID, targetID uint32, req varstore.SubscribeReq) (varstore.VarResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultVarPoolTimeout)
	defer cancel()
	return s.Subscribe(ctx, sourceID, targetID, req)
}

func (s *VarPoolService) Unsubscribe(ctx context.Context, sourceID, targetID uint32, req varstore.SubscribeReq) (varstore.VarResp, error) {
	if strings.TrimSpace(req.Name) == "" {
		return varstore.VarResp{}, errors.New("name is required")
	}
	if req.Owner == 0 {
		return varstore.VarResp{}, errors.New("owner is required")
	}
	payload, err := transport.EncodeMessage(varstore.ActionUnsubscribe, req)
	if err != nil {
		return varstore.VarResp{}, err
	}
	return s.sendAndAwait(ctx, sourceID, targetID, payload, varstore.ActionUnsubscribe, varstore.ActionSubscribeResp, req.Name)
}

func (s *VarPoolService) UnsubscribeSimple(sourceID, targetID uint32, req varstore.SubscribeReq) (varstore.VarResp, error) {
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

func (s *VarPoolService) sendAndAwait(ctx context.Context, sourceID, targetID uint32, payload []byte, reqAction, respAction, name string) (varstore.VarResp, error) {
	if s.session == nil {
		return varstore.VarResp{}, errors.New("session service not initialized")
	}
	trimmedAction := strings.TrimSpace(reqAction)
	trimmedName := strings.TrimSpace(name)

	resp, err := s.session.SendCommandAndAwait(ctx, varstore.SubProtoVarStore, sourceID, targetID, payload, respAction)
	if err != nil {
		if s.logs != nil {
			s.logs.Appendf("error", "varpool %s await failed: %v", trimmedAction, err)
		}
		return varstore.VarResp{}, fmt.Errorf("varpool %s: %w", trimmedAction, toUIError(err))
	}

	var out varstore.VarResp
	if err := json.Unmarshal(resp.Message.Data, &out); err != nil {
		if s.logs != nil {
			s.logs.Appendf("error", "varpool %s decode failed: %v", trimmedAction, err)
		}
		return varstore.VarResp{}, err
	}
	if out.Code != 1 {
		msg := strings.TrimSpace(out.Msg)
		if msg != "" {
			if s.logs != nil {
				s.logs.Appendf("warn", "varpool %s failed (code=%d msg=%q)", trimmedAction, out.Code, msg)
			}
			return varstore.VarResp{}, fmt.Errorf("%s (code=%d)", msg, out.Code)
		}
		if s.logs != nil {
			s.logs.Appendf("warn", "varpool %s failed (code=%d)", trimmedAction, out.Code)
		}
		return varstore.VarResp{}, fmt.Errorf("varpool %s failed (code=%d)", trimmedAction, out.Code)
	}
	if s.logs != nil {
		if trimmedName != "" {
			s.logs.Appendf("info", "varpool %s ok name=%s", trimmedAction, trimmedName)
		} else {
			s.logs.Appendf("info", "varpool %s ok", trimmedAction)
		}
	}
	return out, nil
}

func toUIError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("request timed out")
	}
	if errors.Is(err, context.Canceled) {
		return errors.New("request canceled")
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(msg, "session not initialized"):
		return errors.New("not connected")
	case strings.Contains(msg, "connection") && strings.Contains(msg, "closed"):
		return errors.New("connection closed")
	default:
		return err
	}
}
