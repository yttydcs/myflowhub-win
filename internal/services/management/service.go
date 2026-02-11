package management

import (
	"context"
	"errors"
	"strings"

	"github.com/yttydcs/myflowhub-proto/protocol/management"
	"github.com/yttydcs/myflowhub-win/internal/services/logs"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
	"github.com/yttydcs/myflowhub-win/internal/services/transport"
)

type ManagementService struct {
	session *sessionsvc.SessionService
	logs    *logs.LogService
}

func New(session *sessionsvc.SessionService, logsSvc *logs.LogService) *ManagementService {
	return &ManagementService{session: session, logs: logsSvc}
}

func (s *ManagementService) NodeEcho(ctx context.Context, sourceID, targetID uint32, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return errors.New("message is required")
	}
	payload, err := transport.EncodeMessage(management.ActionNodeEcho, management.NodeEchoReq{Message: message})
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "node_echo")
}

func (s *ManagementService) NodeEchoSimple(sourceID, targetID uint32, message string) error {
	return s.NodeEcho(context.Background(), sourceID, targetID, message)
}

func (s *ManagementService) ListNodes(ctx context.Context, sourceID, targetID uint32) error {
	payload, err := transport.EncodeMessage(management.ActionListNodes, management.ListNodesReq{})
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "list_nodes")
}

func (s *ManagementService) ListNodesSimple(sourceID, targetID uint32) error {
	return s.ListNodes(context.Background(), sourceID, targetID)
}

func (s *ManagementService) ListSubtree(ctx context.Context, sourceID, targetID uint32) error {
	payload, err := transport.EncodeMessage(management.ActionListSubtree, management.ListSubtreeReq{})
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "list_subtree")
}

func (s *ManagementService) ListSubtreeSimple(sourceID, targetID uint32) error {
	return s.ListSubtree(context.Background(), sourceID, targetID)
}

func (s *ManagementService) ConfigGet(ctx context.Context, sourceID, targetID uint32, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("key is required")
	}
	payload, err := transport.EncodeMessage(management.ActionConfigGet, management.ConfigGetReq{Key: key})
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "config_get")
}

func (s *ManagementService) ConfigGetSimple(sourceID, targetID uint32, key string) error {
	return s.ConfigGet(context.Background(), sourceID, targetID, key)
}

func (s *ManagementService) ConfigSet(ctx context.Context, sourceID, targetID uint32, key, value string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("key is required")
	}
	payload, err := transport.EncodeMessage(management.ActionConfigSet, management.ConfigSetReq{Key: key, Value: value})
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "config_set")
}

func (s *ManagementService) ConfigSetSimple(sourceID, targetID uint32, key, value string) error {
	return s.ConfigSet(context.Background(), sourceID, targetID, key, value)
}

func (s *ManagementService) ConfigList(ctx context.Context, sourceID, targetID uint32) error {
	payload, err := transport.EncodeMessage(management.ActionConfigList, management.ConfigListReq{})
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload, "config_list")
}

func (s *ManagementService) ConfigListSimple(sourceID, targetID uint32) error {
	return s.ConfigList(context.Background(), sourceID, targetID)
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
