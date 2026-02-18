package management

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yttydcs/myflowhub-proto/protocol/management"
	"github.com/yttydcs/myflowhub-win/internal/services/logs"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
	"github.com/yttydcs/myflowhub-win/internal/services/transport"
)

const defaultManagementTimeout = 8 * time.Second

type ManagementService struct {
	session *sessionsvc.SessionService
	logs    *logs.LogService
}

func New(session *sessionsvc.SessionService, logsSvc *logs.LogService) *ManagementService {
	return &ManagementService{session: session, logs: logsSvc}
}

func (s *ManagementService) NodeEcho(ctx context.Context, sourceID, targetID uint32, message string) (management.NodeEchoResp, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return management.NodeEchoResp{}, errors.New("message is required")
	}
	payload, err := transport.EncodeMessage(management.ActionNodeEcho, management.NodeEchoReq{Message: message})
	if err != nil {
		return management.NodeEchoResp{}, err
	}
	var resp management.NodeEchoResp
	if err := s.sendAndAwait(ctx, sourceID, targetID, payload, management.ActionNodeEcho, management.ActionNodeEchoResp, &resp); err != nil {
		return management.NodeEchoResp{}, err
	}
	return resp, nil
}

func (s *ManagementService) NodeEchoSimple(sourceID, targetID uint32, message string) (management.NodeEchoResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultManagementTimeout)
	defer cancel()
	return s.NodeEcho(ctx, sourceID, targetID, message)
}

func (s *ManagementService) ListNodes(ctx context.Context, sourceID, targetID uint32) (management.ListNodesResp, error) {
	payload, err := transport.EncodeMessage(management.ActionListNodes, management.ListNodesReq{})
	if err != nil {
		return management.ListNodesResp{}, err
	}
	var resp management.ListNodesResp
	if err := s.sendAndAwait(ctx, sourceID, targetID, payload, management.ActionListNodes, management.ActionListNodesResp, &resp); err != nil {
		return management.ListNodesResp{}, err
	}
	return resp, nil
}

func (s *ManagementService) ListNodesSimple(sourceID, targetID uint32) (management.ListNodesResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultManagementTimeout)
	defer cancel()
	return s.ListNodes(ctx, sourceID, targetID)
}

func (s *ManagementService) ListSubtree(ctx context.Context, sourceID, targetID uint32) (management.ListSubtreeResp, error) {
	payload, err := transport.EncodeMessage(management.ActionListSubtree, management.ListSubtreeReq{})
	if err != nil {
		return management.ListSubtreeResp{}, err
	}
	var resp management.ListSubtreeResp
	if err := s.sendAndAwait(ctx, sourceID, targetID, payload, management.ActionListSubtree, management.ActionListSubtreeResp, &resp); err != nil {
		return management.ListSubtreeResp{}, err
	}
	return resp, nil
}

func (s *ManagementService) ListSubtreeSimple(sourceID, targetID uint32) (management.ListSubtreeResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultManagementTimeout)
	defer cancel()
	return s.ListSubtree(ctx, sourceID, targetID)
}

func (s *ManagementService) ConfigGet(ctx context.Context, sourceID, targetID uint32, key string) (management.ConfigResp, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return management.ConfigResp{}, errors.New("key is required")
	}
	payload, err := transport.EncodeMessage(management.ActionConfigGet, management.ConfigGetReq{Key: key})
	if err != nil {
		return management.ConfigResp{}, err
	}
	var resp management.ConfigResp
	if err := s.sendAndAwait(ctx, sourceID, targetID, payload, management.ActionConfigGet, management.ActionConfigGetResp, &resp); err != nil {
		return management.ConfigResp{}, err
	}
	return resp, nil
}

func (s *ManagementService) ConfigGetSimple(sourceID, targetID uint32, key string) (management.ConfigResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultManagementTimeout)
	defer cancel()
	return s.ConfigGet(ctx, sourceID, targetID, key)
}

func (s *ManagementService) ConfigSet(ctx context.Context, sourceID, targetID uint32, key, value string) (management.ConfigResp, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return management.ConfigResp{}, errors.New("key is required")
	}
	payload, err := transport.EncodeMessage(management.ActionConfigSet, management.ConfigSetReq{Key: key, Value: value})
	if err != nil {
		return management.ConfigResp{}, err
	}
	var resp management.ConfigResp
	if err := s.sendAndAwait(ctx, sourceID, targetID, payload, management.ActionConfigSet, management.ActionConfigSetResp, &resp); err != nil {
		return management.ConfigResp{}, err
	}
	return resp, nil
}

func (s *ManagementService) ConfigSetSimple(sourceID, targetID uint32, key, value string) (management.ConfigResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultManagementTimeout)
	defer cancel()
	return s.ConfigSet(ctx, sourceID, targetID, key, value)
}

func (s *ManagementService) ConfigList(ctx context.Context, sourceID, targetID uint32) (management.ConfigListResp, error) {
	payload, err := transport.EncodeMessage(management.ActionConfigList, management.ConfigListReq{})
	if err != nil {
		return management.ConfigListResp{}, err
	}
	var resp management.ConfigListResp
	if err := s.sendAndAwait(ctx, sourceID, targetID, payload, management.ActionConfigList, management.ActionConfigListResp, &resp); err != nil {
		return management.ConfigListResp{}, err
	}
	return resp, nil
}

func (s *ManagementService) ConfigListSimple(sourceID, targetID uint32) (management.ConfigListResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultManagementTimeout)
	defer cancel()
	return s.ConfigList(ctx, sourceID, targetID)
}

func (s *ManagementService) Send(ctx context.Context, sourceID, targetID uint32, action string, data any) error {
	payload, err := transport.EncodeMessage(action, data)
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, action)
}

func (s *ManagementService) send(_ context.Context, sourceID, targetID uint32, payload []byte, action string) error {
	if s.session == nil {
		return errors.New("session service not initialized")
	}
	if err := s.session.SendCommand(management.SubProtoManagement, sourceID, targetID, payload); err != nil {
		return err
	}
	if s.logs != nil {
		s.logs.Appendf("info", "management %s sent", action)
	}
	return nil
}

func (s *ManagementService) sendAndAwait(ctx context.Context, sourceID, targetID uint32, payload []byte, reqAction, respAction string, out any) error {
	if s.session == nil {
		return errors.New("session service not initialized")
	}
	resp, err := s.session.SendCommandAndAwait(ctx, management.SubProtoManagement, sourceID, targetID, payload, respAction)
	if err != nil {
		if s.logs != nil {
			s.logs.Appendf("error", "management %s await failed: %v", strings.TrimSpace(reqAction), err)
		}
		return fmt.Errorf("management %s: %w", strings.TrimSpace(reqAction), toUIError(err))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(resp.Message.Data, out); err != nil {
		if s.logs != nil {
			s.logs.Appendf("error", "management %s decode failed: %v", strings.TrimSpace(reqAction), err)
		}
		return err
	}
	code, msg := extractCodeMsg(out)
	if code != 1 {
		msg = strings.TrimSpace(msg)
		if msg != "" {
			if s.logs != nil {
				s.logs.Appendf("warn", "management %s failed (code=%d msg=%q)", strings.TrimSpace(reqAction), code, msg)
			}
			return fmt.Errorf("%s (code=%d)", msg, code)
		}
		if s.logs != nil {
			s.logs.Appendf("warn", "management %s failed (code=%d)", strings.TrimSpace(reqAction), code)
		}
		return fmt.Errorf("management %s failed (code=%d)", strings.TrimSpace(reqAction), code)
	}
	if s.logs != nil {
		s.logs.Appendf("info", "management %s ok", strings.TrimSpace(reqAction))
	}
	return nil
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

func extractCodeMsg(v any) (int, string) {
	switch t := v.(type) {
	case *management.NodeEchoResp:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	case *management.ListNodesResp:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	case *management.ListSubtreeResp:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	case *management.ConfigResp:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	case *management.ConfigListResp:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	default:
		return 0, ""
	}
}
