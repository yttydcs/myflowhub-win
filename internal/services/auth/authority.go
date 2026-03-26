package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/yttydcs/myflowhub-win/internal/services/transport"
)

const (
	actionListPendingRegisters     = "list_pending_registers"
	actionListPendingRegistersResp = "list_pending_registers_resp"
	actionApproveRegister          = "approve_register"
	actionApproveRegisterResp      = "approve_register_resp"
	actionRejectRegister           = "reject_register"
	actionRejectRegisterResp       = "reject_register_resp"
	actionIssueRegisterPermit      = "issue_register_permit"
	actionIssueRegisterPermitResp  = "issue_register_permit_resp"
	actionRevokeRegisterPermit     = "revoke_register_permit"
	actionRevokeRegisterPermitResp = "revoke_register_permit_resp"
)

type PendingRegisterInfo struct {
	RequestID     string `json:"request_id,omitempty"`
	DeviceID      string `json:"device_id,omitempty"`
	RequestedRole string `json:"requested_role,omitempty"`
	DisplayName   string `json:"display_name,omitempty"`
	CreatedAt     int64  `json:"created_at,omitempty"`
	ExpiresAt     int64  `json:"expires_at,omitempty"`
}

type ListPendingRegistersReq struct {
	Offset   int    `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	DeviceID string `json:"device_id,omitempty"`
}

type ListPendingRegistersResp struct {
	Code  int                   `json:"code"`
	Msg   string                `json:"msg,omitempty"`
	Total int                   `json:"total"`
	Items []PendingRegisterInfo `json:"items,omitempty"`
}

type ApproveRegisterReq struct {
	RequestID string `json:"request_id"`
	Role      string `json:"role,omitempty"`
}

type ApproveRegisterResp struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`
	NodeID    uint32 `json:"node_id,omitempty"`
	Role      string `json:"role,omitempty"`
	Status    string `json:"status,omitempty"`
}

type RejectRegisterReq struct {
	RequestID string `json:"request_id"`
	Reason    string `json:"reason,omitempty"`
}

type RejectRegisterResp struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`
	Status    string `json:"status,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type IssueRegisterPermitReq struct {
	DeviceID  string `json:"device_id"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}

type IssueRegisterPermitResp struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg,omitempty"`
	Permit    string `json:"permit,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`
	Role      string `json:"role,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}

type RevokeRegisterPermitReq struct {
	Permit string `json:"permit"`
}

type RevokeRegisterPermitResp struct {
	Code     int    `json:"code"`
	Msg      string `json:"msg,omitempty"`
	Permit   string `json:"permit,omitempty"`
	DeviceID string `json:"device_id,omitempty"`
	Role     string `json:"role,omitempty"`
}

func (s *AuthService) ListPendingRegisters(ctx context.Context, sourceID, targetID uint32, req ListPendingRegistersReq) (ListPendingRegistersResp, error) {
	if req.Offset < 0 {
		return ListPendingRegistersResp{}, errors.New("offset must be non-negative")
	}
	if req.Limit < 0 {
		return ListPendingRegistersResp{}, errors.New("limit must be non-negative")
	}
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	payload, err := transport.EncodeMessage(actionListPendingRegisters, req)
	if err != nil {
		return ListPendingRegistersResp{}, err
	}
	var resp ListPendingRegistersResp
	if err := s.sendAndAwaitInto(ctx, sourceID, targetID, payload, actionListPendingRegisters, actionListPendingRegistersResp, &resp); err != nil {
		return ListPendingRegistersResp{}, err
	}
	return resp, nil
}

func (s *AuthService) ListPendingRegistersSimple(sourceID, targetID uint32, req ListPendingRegistersReq) (ListPendingRegistersResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAuthTimeout)
	defer cancel()
	return s.ListPendingRegisters(ctx, sourceID, targetID, req)
}

func (s *AuthService) ApproveRegister(ctx context.Context, sourceID, targetID uint32, req ApproveRegisterReq) (ApproveRegisterResp, error) {
	req.RequestID = strings.TrimSpace(req.RequestID)
	req.Role = strings.TrimSpace(req.Role)
	if req.RequestID == "" {
		return ApproveRegisterResp{}, errors.New("request_id is required")
	}
	payload, err := transport.EncodeMessage(actionApproveRegister, req)
	if err != nil {
		return ApproveRegisterResp{}, err
	}
	var resp ApproveRegisterResp
	if err := s.sendAndAwaitInto(ctx, sourceID, targetID, payload, actionApproveRegister, actionApproveRegisterResp, &resp); err != nil {
		return ApproveRegisterResp{}, err
	}
	return resp, nil
}

func (s *AuthService) ApproveRegisterSimple(sourceID, targetID uint32, req ApproveRegisterReq) (ApproveRegisterResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAuthTimeout)
	defer cancel()
	return s.ApproveRegister(ctx, sourceID, targetID, req)
}

func (s *AuthService) RejectRegister(ctx context.Context, sourceID, targetID uint32, req RejectRegisterReq) (RejectRegisterResp, error) {
	req.RequestID = strings.TrimSpace(req.RequestID)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.RequestID == "" {
		return RejectRegisterResp{}, errors.New("request_id is required")
	}
	payload, err := transport.EncodeMessage(actionRejectRegister, req)
	if err != nil {
		return RejectRegisterResp{}, err
	}
	var resp RejectRegisterResp
	if err := s.sendAndAwaitInto(ctx, sourceID, targetID, payload, actionRejectRegister, actionRejectRegisterResp, &resp); err != nil {
		return RejectRegisterResp{}, err
	}
	return resp, nil
}

func (s *AuthService) RejectRegisterSimple(sourceID, targetID uint32, req RejectRegisterReq) (RejectRegisterResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAuthTimeout)
	defer cancel()
	return s.RejectRegister(ctx, sourceID, targetID, req)
}

func (s *AuthService) IssueRegisterPermit(ctx context.Context, sourceID, targetID uint32, req IssueRegisterPermitReq) (IssueRegisterPermitResp, error) {
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	req.Role = strings.TrimSpace(req.Role)
	if req.DeviceID == "" {
		return IssueRegisterPermitResp{}, errors.New("device_id is required")
	}
	if req.Role == "" {
		return IssueRegisterPermitResp{}, errors.New("role is required")
	}
	if req.ExpiresAt < 0 {
		return IssueRegisterPermitResp{}, errors.New("expires_at must be non-negative")
	}
	payload, err := transport.EncodeMessage(actionIssueRegisterPermit, req)
	if err != nil {
		return IssueRegisterPermitResp{}, err
	}
	var resp IssueRegisterPermitResp
	if err := s.sendAndAwaitInto(ctx, sourceID, targetID, payload, actionIssueRegisterPermit, actionIssueRegisterPermitResp, &resp); err != nil {
		return IssueRegisterPermitResp{}, err
	}
	return resp, nil
}

func (s *AuthService) IssueRegisterPermitSimple(sourceID, targetID uint32, req IssueRegisterPermitReq) (IssueRegisterPermitResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAuthTimeout)
	defer cancel()
	return s.IssueRegisterPermit(ctx, sourceID, targetID, req)
}

func (s *AuthService) RevokeRegisterPermit(ctx context.Context, sourceID, targetID uint32, req RevokeRegisterPermitReq) (RevokeRegisterPermitResp, error) {
	req.Permit = strings.TrimSpace(req.Permit)
	if req.Permit == "" {
		return RevokeRegisterPermitResp{}, errors.New("permit is required")
	}
	payload, err := transport.EncodeMessage(actionRevokeRegisterPermit, req)
	if err != nil {
		return RevokeRegisterPermitResp{}, err
	}
	var resp RevokeRegisterPermitResp
	if err := s.sendAndAwaitInto(ctx, sourceID, targetID, payload, actionRevokeRegisterPermit, actionRevokeRegisterPermitResp, &resp); err != nil {
		return RevokeRegisterPermitResp{}, err
	}
	return resp, nil
}

func (s *AuthService) RevokeRegisterPermitSimple(sourceID, targetID uint32, req RevokeRegisterPermitReq) (RevokeRegisterPermitResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAuthTimeout)
	defer cancel()
	return s.RevokeRegisterPermit(ctx, sourceID, targetID, req)
}
