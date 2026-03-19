package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	coreperm "github.com/yttydcs/myflowhub-core/kit/permission"
	protoauth "github.com/yttydcs/myflowhub-proto/protocol/auth"
	"github.com/yttydcs/myflowhub-win/internal/services/transport"
)

// ListRolesResp is the auth list_roles response payload.
type ListRolesResp struct {
	Code  int                       `json:"code"`
	Msg   string                    `json:"msg,omitempty"`
	Total int                       `json:"total"`
	Roles []protoauth.RolePermEntry `json:"roles,omitempty"`
}

// InvalidatePermsReq controls auth perms_invalidate behavior.
type InvalidatePermsReq struct {
	NodeIDs []uint32 `json:"nodeIds,omitempty"`
	Reason  string   `json:"reason,omitempty"`
	Refresh bool     `json:"refresh,omitempty"`
}

func (s *AuthService) GetPerms(ctx context.Context, sourceID, targetID, nodeID uint32) (protoauth.RespData, error) {
	if nodeID == 0 {
		return protoauth.RespData{}, errors.New("node_id is required")
	}
	payload, err := transport.EncodeMessage(protoauth.ActionGetPerms, protoauth.PermsQueryData{NodeID: nodeID})
	if err != nil {
		return protoauth.RespData{}, err
	}
	resp, err := s.sendAndAwait(ctx, sourceID, targetID, payload, protoauth.ActionGetPerms, protoauth.ActionGetPermsResp)
	if err != nil {
		return protoauth.RespData{}, err
	}
	return resp, nil
}

func (s *AuthService) GetPermsSimple(sourceID, targetID, nodeID uint32) (protoauth.RespData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAuthTimeout)
	defer cancel()
	return s.GetPerms(ctx, sourceID, targetID, nodeID)
}

func (s *AuthService) ListRoles(ctx context.Context, sourceID, targetID uint32, req protoauth.ListRolesReq) (ListRolesResp, error) {
	payload, err := transport.EncodeMessage(protoauth.ActionListRoles, req)
	if err != nil {
		return ListRolesResp{}, err
	}
	if s.session == nil {
		return ListRolesResp{}, errors.New("session service not initialized")
	}
	resp, err := s.session.SendCommandAndAwait(ctx, protoauth.SubProtoAuth, sourceID, targetID, payload, protoauth.ActionListRolesResp)
	if err != nil {
		if s.logs != nil {
			s.logs.Appendf("error", "auth %s await failed: %v", strings.TrimSpace(protoauth.ActionListRoles), err)
		}
		return ListRolesResp{}, fmt.Errorf("auth %s: %w", strings.TrimSpace(protoauth.ActionListRoles), toUIError(err))
	}

	var data ListRolesResp
	if err := json.Unmarshal(resp.Message.Data, &data); err != nil {
		if s.logs != nil {
			s.logs.Appendf("error", "auth %s decode failed: %v", strings.TrimSpace(protoauth.ActionListRoles), err)
		}
		return ListRolesResp{}, err
	}
	if data.Code != 1 {
		msg := strings.TrimSpace(data.Msg)
		if msg != "" {
			if s.logs != nil {
				s.logs.Appendf("warn", "auth %s failed (code=%d msg=%q)", strings.TrimSpace(protoauth.ActionListRoles), data.Code, msg)
			}
			return ListRolesResp{}, fmt.Errorf("%s (code=%d)", msg, data.Code)
		}
		if s.logs != nil {
			s.logs.Appendf("warn", "auth %s failed (code=%d)", strings.TrimSpace(protoauth.ActionListRoles), data.Code)
		}
		return ListRolesResp{}, fmt.Errorf("auth failed (code=%d)", data.Code)
	}
	return data, nil
}

func (s *AuthService) ListRolesSimple(sourceID, targetID uint32, req protoauth.ListRolesReq) (ListRolesResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAuthTimeout)
	defer cancel()
	return s.ListRoles(ctx, sourceID, targetID, req)
}

func (s *AuthService) PushPermsSnapshot(ctx context.Context, sourceID, targetID uint32, snapshot coreperm.Snapshot) error {
	payload, err := transport.EncodeMessage(protoauth.ActionPermsSnapshot, snapshot)
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload)
}

func (s *AuthService) PushPermsSnapshotSimple(sourceID, targetID uint32, snapshot coreperm.Snapshot) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return s.PushPermsSnapshot(ctx, sourceID, targetID, snapshot)
}

func (s *AuthService) InvalidatePerms(ctx context.Context, sourceID, targetID uint32, req InvalidatePermsReq) error {
	payload, err := transport.EncodeMessage(protoauth.ActionPermsInvalidate, protoauth.InvalidateData{
		NodeIDs: req.NodeIDs,
		Reason:  strings.TrimSpace(req.Reason),
		Refresh: req.Refresh,
	})
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload)
}

func (s *AuthService) InvalidatePermsSimple(sourceID, targetID uint32, req InvalidatePermsReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return s.InvalidatePerms(ctx, sourceID, targetID, req)
}
