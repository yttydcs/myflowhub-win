package permission

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	coreperm "github.com/yttydcs/myflowhub-core/kit/permission"
	protoauth "github.com/yttydcs/myflowhub-proto/protocol/auth"
	authsvc "github.com/yttydcs/myflowhub-win/internal/services/auth"
	logssvc "github.com/yttydcs/myflowhub-win/internal/services/logs"
	mgmtsvc "github.com/yttydcs/myflowhub-win/internal/services/management"
)

const (
	keyAuthorityNodeID = "authority.node_id"

	keyAuthDefaultRole  = "auth.default_role"
	keyAuthDefaultPerms = "auth.default_perms"
	keyAuthNodeRoles    = "auth.node_roles"
	keyAuthRolePerms    = "auth.role_perms"
)

type PermissionService struct {
	auth *authsvc.AuthService
	mgmt *mgmtsvc.ManagementService
	logs *logssvc.LogService
}

type AuthorityTarget struct {
	AuthorityID uint32 `json:"authorityId"`
	Reason      string `json:"reason"`
}

type NodeRole struct {
	NodeID uint32 `json:"nodeId"`
	Role   string `json:"role"`
}

type RolePerm struct {
	Role  string   `json:"role"`
	Perms []string `json:"perms"`
}

type Policy struct {
	DefaultRole  string     `json:"defaultRole"`
	DefaultPerms []string   `json:"defaultPerms"`
	NodeRoles    []NodeRole `json:"nodeRoles"`
	RolePerms    []RolePerm `json:"rolePerms"`
}

type RawPolicy struct {
	AuthDefaultRole  string `json:"authDefaultRole"`
	AuthDefaultPerms string `json:"authDefaultPerms"`
	AuthNodeRoles    string `json:"authNodeRoles"`
	AuthRolePerms    string `json:"authRolePerms"`
}

type RuntimeRole struct {
	NodeID uint32   `json:"nodeId"`
	Role   string   `json:"role"`
	Perms  []string `json:"perms"`
}

type LoadPolicyResult struct {
	AuthorityID  uint32        `json:"authorityId"`
	Raw          RawPolicy     `json:"raw"`
	Policy       Policy        `json:"policy"`
	Runtime      []RuntimeRole `json:"runtime"`
	RuntimeTotal int           `json:"runtimeTotal"`
	RuntimeError string        `json:"runtimeError,omitempty"`
	Warnings     []string      `json:"warnings,omitempty"`
}

type SavePolicyRequest struct {
	SourceID      uint32 `json:"sourceId"`
	AuthorityID   uint32 `json:"authorityId"`
	Policy        Policy `json:"policy"`
	Persist       bool   `json:"persist"`
	ApplyRuntime  bool   `json:"applyRuntime"`
	Invalidate    bool   `json:"invalidate"`
	Refresh       bool   `json:"refresh"`
	VerifyRuntime bool   `json:"verifyRuntime"`
}

type SavePolicyResult struct {
	Success      bool          `json:"success"`
	ErrorStage   string        `json:"errorStage,omitempty"`
	ErrorMessage string        `json:"errorMessage,omitempty"`
	Persisted    bool          `json:"persisted"`
	Applied      bool          `json:"applied"`
	Invalidated  bool          `json:"invalidated"`
	AuthorityID  uint32        `json:"authorityId"`
	Raw          RawPolicy     `json:"raw"`
	Policy       Policy        `json:"policy"`
	Warnings     []string      `json:"warnings,omitempty"`
	Runtime      []RuntimeRole `json:"runtime,omitempty"`
	RuntimeTotal int           `json:"runtimeTotal,omitempty"`
	RuntimeError string        `json:"runtimeError,omitempty"`
}

type NodePermsResult struct {
	NodeID uint32   `json:"nodeId"`
	Role   string   `json:"role"`
	Perms  []string `json:"perms"`
}

type ListPendingRegistersRequest struct {
	SourceID    uint32 `json:"sourceId"`
	AuthorityID uint32 `json:"authorityId"`
	Offset      int    `json:"offset,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	DeviceID    string `json:"deviceId,omitempty"`
}

type PendingRegister struct {
	RequestID     string `json:"requestId"`
	DeviceID      string `json:"deviceId"`
	RequestedRole string `json:"requestedRole,omitempty"`
	DisplayName   string `json:"displayName,omitempty"`
	CreatedAt     int64  `json:"createdAt,omitempty"`
	ExpiresAt     int64  `json:"expiresAt,omitempty"`
}

type ListPendingRegistersResult struct {
	AuthorityID uint32            `json:"authorityId"`
	Total       int               `json:"total"`
	Items       []PendingRegister `json:"items,omitempty"`
}

type ApproveRegisterRequest struct {
	SourceID    uint32 `json:"sourceId"`
	AuthorityID uint32 `json:"authorityId"`
	RequestID   string `json:"requestId"`
	Role        string `json:"role,omitempty"`
}

type ApproveRegisterResult struct {
	RequestID string `json:"requestId"`
	DeviceID  string `json:"deviceId,omitempty"`
	NodeID    uint32 `json:"nodeId,omitempty"`
	Role      string `json:"role,omitempty"`
	Status    string `json:"status,omitempty"`
}

type RejectRegisterRequest struct {
	SourceID    uint32 `json:"sourceId"`
	AuthorityID uint32 `json:"authorityId"`
	RequestID   string `json:"requestId"`
	Reason      string `json:"reason,omitempty"`
}

type RejectRegisterResult struct {
	RequestID string `json:"requestId"`
	DeviceID  string `json:"deviceId,omitempty"`
	Status    string `json:"status,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type IssueRegisterPermitRequest struct {
	SourceID    uint32 `json:"sourceId"`
	AuthorityID uint32 `json:"authorityId"`
	DeviceID    string `json:"deviceId"`
	Role        string `json:"role"`
	ExpiresAt   int64  `json:"expiresAt,omitempty"`
}

type PermitIssueResult struct {
	Permit    string `json:"permit,omitempty"`
	DeviceID  string `json:"deviceId,omitempty"`
	Role      string `json:"role,omitempty"`
	ExpiresAt int64  `json:"expiresAt,omitempty"`
}

type RevokeRegisterPermitRequest struct {
	SourceID    uint32 `json:"sourceId"`
	AuthorityID uint32 `json:"authorityId"`
	Permit      string `json:"permit"`
}

type RevokeRegisterPermitResult struct {
	Permit   string `json:"permit,omitempty"`
	DeviceID string `json:"deviceId,omitempty"`
	Role     string `json:"role,omitempty"`
}

func New(auth *authsvc.AuthService, mgmt *mgmtsvc.ManagementService, logs *logssvc.LogService) *PermissionService {
	return &PermissionService{auth: auth, mgmt: mgmt, logs: logs}
}

func (s *PermissionService) ResolveAuthority(sourceID, hubID, overrideID uint32) (AuthorityTarget, error) {
	if overrideID != 0 {
		return AuthorityTarget{
			AuthorityID: overrideID,
			Reason:      "manual_override",
		}, nil
	}
	if hubID == 0 {
		return AuthorityTarget{}, errors.New("hub_id is required")
	}
	if s.mgmt == nil {
		return AuthorityTarget{
			AuthorityID: hubID,
			Reason:      "hub_id_fallback",
		}, nil
	}
	if sourceID == 0 {
		return AuthorityTarget{
			AuthorityID: hubID,
			Reason:      "hub_id_without_login",
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	resp, err := s.mgmt.ConfigGet(ctx, sourceID, hubID, keyAuthorityNodeID)
	if err != nil {
		return AuthorityTarget{
			AuthorityID: hubID,
			Reason:      "hub_id_fallback",
		}, nil
	}
	raw := strings.TrimSpace(resp.Value)
	if raw == "" {
		return AuthorityTarget{
			AuthorityID: hubID,
			Reason:      "hub_id_fallback",
		}, nil
	}
	v, parseErr := strconv.ParseUint(raw, 10, 32)
	if parseErr != nil || v == 0 {
		return AuthorityTarget{
			AuthorityID: hubID,
			Reason:      "hub_id_fallback",
		}, nil
	}
	return AuthorityTarget{
		AuthorityID: uint32(v),
		Reason:      "authority_node_id",
	}, nil
}

func (s *PermissionService) LoadPolicy(sourceID, authorityID uint32) (LoadPolicyResult, error) {
	if sourceID == 0 {
		return LoadPolicyResult{}, errors.New("source_id is required")
	}
	if authorityID == 0 {
		return LoadPolicyResult{}, errors.New("authority_id is required")
	}
	if s.mgmt == nil {
		return LoadPolicyResult{}, errors.New("management service not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	raw := RawPolicy{}
	var err error
	raw.AuthDefaultRole, err = s.readConfig(ctx, sourceID, authorityID, keyAuthDefaultRole)
	if err != nil {
		return LoadPolicyResult{}, err
	}
	raw.AuthDefaultPerms, err = s.readConfig(ctx, sourceID, authorityID, keyAuthDefaultPerms)
	if err != nil {
		return LoadPolicyResult{}, err
	}
	raw.AuthNodeRoles, err = s.readConfig(ctx, sourceID, authorityID, keyAuthNodeRoles)
	if err != nil {
		return LoadPolicyResult{}, err
	}
	raw.AuthRolePerms, err = s.readConfig(ctx, sourceID, authorityID, keyAuthRolePerms)
	if err != nil {
		return LoadPolicyResult{}, err
	}

	policy, warnings, err := parseRawPolicy(raw)
	if err != nil {
		return LoadPolicyResult{}, err
	}

	result := LoadPolicyResult{
		AuthorityID: authorityID,
		Raw:         toRawPolicy(policy),
		Policy:      policy,
		Warnings:    warnings,
	}

	if s.auth != nil {
		resp, listErr := s.auth.ListRoles(ctx, sourceID, authorityID, protoauth.ListRolesReq{Offset: 0, Limit: 500})
		if listErr != nil {
			result.RuntimeError = listErr.Error()
		} else {
			result.Runtime = toRuntimeRoles(resp.Roles)
			result.RuntimeTotal = resp.Total
		}
	}
	return result, nil
}

func (s *PermissionService) SavePolicy(req SavePolicyRequest) (SavePolicyResult, error) {
	if req.SourceID == 0 {
		return SavePolicyResult{}, errors.New("source_id is required")
	}
	if req.AuthorityID == 0 {
		return SavePolicyResult{}, errors.New("authority_id is required")
	}
	if !req.Persist && !req.ApplyRuntime && !req.Invalidate {
		return SavePolicyResult{}, errors.New("at least one action is required: persist/applyRuntime/invalidate")
	}
	if s.mgmt == nil {
		return SavePolicyResult{}, errors.New("management service not initialized")
	}
	if (req.ApplyRuntime || req.Invalidate) && s.auth == nil {
		return SavePolicyResult{}, errors.New("auth service not initialized")
	}

	policy, warnings, err := normalizePolicy(req.Policy)
	if err != nil {
		return SavePolicyResult{}, err
	}
	raw := toRawPolicy(policy)
	result := SavePolicyResult{
		Success:     true,
		AuthorityID: req.AuthorityID,
		Raw:         raw,
		Policy:      policy,
		Warnings:    warnings,
	}

	fail := func(stage string, err error) SavePolicyResult {
		result.Success = false
		result.ErrorStage = stage
		result.ErrorMessage = err.Error()
		return result
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	if req.Persist {
		if _, err := s.mgmt.ConfigSet(ctx, req.SourceID, req.AuthorityID, keyAuthDefaultRole, raw.AuthDefaultRole); err != nil {
			return fail("persist.default_role", err), nil
		}
		if _, err := s.mgmt.ConfigSet(ctx, req.SourceID, req.AuthorityID, keyAuthDefaultPerms, raw.AuthDefaultPerms); err != nil {
			return fail("persist.default_perms", err), nil
		}
		if _, err := s.mgmt.ConfigSet(ctx, req.SourceID, req.AuthorityID, keyAuthNodeRoles, raw.AuthNodeRoles); err != nil {
			return fail("persist.node_roles", err), nil
		}
		if _, err := s.mgmt.ConfigSet(ctx, req.SourceID, req.AuthorityID, keyAuthRolePerms, raw.AuthRolePerms); err != nil {
			return fail("persist.role_perms", err), nil
		}
		result.Persisted = true
	}

	if req.ApplyRuntime {
		if err := s.auth.PushPermsSnapshot(ctx, req.SourceID, req.AuthorityID, toSnapshot(policy)); err != nil {
			return fail("runtime.snapshot", err), nil
		}
		result.Applied = true
	}

	if req.Invalidate {
		err := s.auth.InvalidatePerms(ctx, req.SourceID, req.AuthorityID, authsvc.InvalidatePermsReq{
			NodeIDs: nil,
			Reason:  "permissions_page_save",
			Refresh: req.Refresh,
		})
		if err != nil {
			return fail("runtime.invalidate", err), nil
		}
		result.Invalidated = true
	}

	if req.VerifyRuntime && s.auth != nil {
		resp, listErr := s.auth.ListRoles(ctx, req.SourceID, req.AuthorityID, protoauth.ListRolesReq{Offset: 0, Limit: 500})
		if listErr != nil {
			result.RuntimeError = listErr.Error()
		} else {
			result.Runtime = toRuntimeRoles(resp.Roles)
			result.RuntimeTotal = resp.Total
		}
	}

	return result, nil
}

func (s *PermissionService) GetNodePerms(sourceID, authorityID, nodeID uint32) (NodePermsResult, error) {
	if sourceID == 0 {
		return NodePermsResult{}, errors.New("source_id is required")
	}
	if authorityID == 0 {
		return NodePermsResult{}, errors.New("authority_id is required")
	}
	if nodeID == 0 {
		return NodePermsResult{}, errors.New("node_id is required")
	}
	if s.auth == nil {
		return NodePermsResult{}, errors.New("auth service not initialized")
	}

	resp, err := s.auth.GetPermsSimple(sourceID, authorityID, nodeID)
	if err != nil {
		return NodePermsResult{}, err
	}
	return NodePermsResult{
		NodeID: nodeID,
		Role:   strings.TrimSpace(resp.Role),
		Perms:  cloneSortedTokens(resp.Perms),
	}, nil
}

func (s *PermissionService) ListPendingRegisters(req ListPendingRegistersRequest) (ListPendingRegistersResult, error) {
	if req.SourceID == 0 {
		return ListPendingRegistersResult{}, errors.New("source_id is required")
	}
	if req.AuthorityID == 0 {
		return ListPendingRegistersResult{}, errors.New("authority_id is required")
	}
	if req.Offset < 0 {
		return ListPendingRegistersResult{}, errors.New("offset must be non-negative")
	}
	if req.Limit < 0 {
		return ListPendingRegistersResult{}, errors.New("limit must be non-negative")
	}
	if s.auth == nil {
		return ListPendingRegistersResult{}, errors.New("auth service not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	resp, err := s.auth.ListPendingRegisters(ctx, req.SourceID, req.AuthorityID, authsvc.ListPendingRegistersReq{
		Offset:   req.Offset,
		Limit:    req.Limit,
		DeviceID: strings.TrimSpace(req.DeviceID),
	})
	if err != nil {
		return ListPendingRegistersResult{}, err
	}
	return ListPendingRegistersResult{
		AuthorityID: req.AuthorityID,
		Total:       resp.Total,
		Items:       toPendingRegisters(resp.Items),
	}, nil
}

func (s *PermissionService) ApproveRegister(req ApproveRegisterRequest) (ApproveRegisterResult, error) {
	if req.SourceID == 0 {
		return ApproveRegisterResult{}, errors.New("source_id is required")
	}
	if req.AuthorityID == 0 {
		return ApproveRegisterResult{}, errors.New("authority_id is required")
	}
	if strings.TrimSpace(req.RequestID) == "" {
		return ApproveRegisterResult{}, errors.New("request_id is required")
	}
	if s.auth == nil {
		return ApproveRegisterResult{}, errors.New("auth service not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	resp, err := s.auth.ApproveRegister(ctx, req.SourceID, req.AuthorityID, authsvc.ApproveRegisterReq{
		RequestID: strings.TrimSpace(req.RequestID),
		Role:      strings.TrimSpace(req.Role),
	})
	if err != nil {
		return ApproveRegisterResult{}, err
	}
	return ApproveRegisterResult{
		RequestID: strings.TrimSpace(resp.RequestID),
		DeviceID:  strings.TrimSpace(resp.DeviceID),
		NodeID:    resp.NodeID,
		Role:      strings.TrimSpace(resp.Role),
		Status:    strings.TrimSpace(resp.Status),
	}, nil
}

func (s *PermissionService) RejectRegister(req RejectRegisterRequest) (RejectRegisterResult, error) {
	if req.SourceID == 0 {
		return RejectRegisterResult{}, errors.New("source_id is required")
	}
	if req.AuthorityID == 0 {
		return RejectRegisterResult{}, errors.New("authority_id is required")
	}
	if strings.TrimSpace(req.RequestID) == "" {
		return RejectRegisterResult{}, errors.New("request_id is required")
	}
	if s.auth == nil {
		return RejectRegisterResult{}, errors.New("auth service not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	resp, err := s.auth.RejectRegister(ctx, req.SourceID, req.AuthorityID, authsvc.RejectRegisterReq{
		RequestID: strings.TrimSpace(req.RequestID),
		Reason:    strings.TrimSpace(req.Reason),
	})
	if err != nil {
		return RejectRegisterResult{}, err
	}
	return RejectRegisterResult{
		RequestID: strings.TrimSpace(resp.RequestID),
		DeviceID:  strings.TrimSpace(resp.DeviceID),
		Status:    strings.TrimSpace(resp.Status),
		Reason:    strings.TrimSpace(resp.Reason),
	}, nil
}

func (s *PermissionService) IssueRegisterPermit(req IssueRegisterPermitRequest) (PermitIssueResult, error) {
	if req.SourceID == 0 {
		return PermitIssueResult{}, errors.New("source_id is required")
	}
	if req.AuthorityID == 0 {
		return PermitIssueResult{}, errors.New("authority_id is required")
	}
	if strings.TrimSpace(req.DeviceID) == "" {
		return PermitIssueResult{}, errors.New("device_id is required")
	}
	if strings.TrimSpace(req.Role) == "" {
		return PermitIssueResult{}, errors.New("role is required")
	}
	if req.ExpiresAt < 0 {
		return PermitIssueResult{}, errors.New("expires_at must be non-negative")
	}
	if s.auth == nil {
		return PermitIssueResult{}, errors.New("auth service not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	resp, err := s.auth.IssueRegisterPermit(ctx, req.SourceID, req.AuthorityID, authsvc.IssueRegisterPermitReq{
		DeviceID:  strings.TrimSpace(req.DeviceID),
		Role:      strings.TrimSpace(req.Role),
		ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		return PermitIssueResult{}, err
	}
	return PermitIssueResult{
		Permit:    strings.TrimSpace(resp.Permit),
		DeviceID:  strings.TrimSpace(resp.DeviceID),
		Role:      strings.TrimSpace(resp.Role),
		ExpiresAt: resp.ExpiresAt,
	}, nil
}

func (s *PermissionService) RevokeRegisterPermit(req RevokeRegisterPermitRequest) (RevokeRegisterPermitResult, error) {
	if req.SourceID == 0 {
		return RevokeRegisterPermitResult{}, errors.New("source_id is required")
	}
	if req.AuthorityID == 0 {
		return RevokeRegisterPermitResult{}, errors.New("authority_id is required")
	}
	if strings.TrimSpace(req.Permit) == "" {
		return RevokeRegisterPermitResult{}, errors.New("permit is required")
	}
	if s.auth == nil {
		return RevokeRegisterPermitResult{}, errors.New("auth service not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	resp, err := s.auth.RevokeRegisterPermit(ctx, req.SourceID, req.AuthorityID, authsvc.RevokeRegisterPermitReq{
		Permit: strings.TrimSpace(req.Permit),
	})
	if err != nil {
		return RevokeRegisterPermitResult{}, err
	}
	return RevokeRegisterPermitResult{
		Permit:   strings.TrimSpace(resp.Permit),
		DeviceID: strings.TrimSpace(resp.DeviceID),
		Role:     strings.TrimSpace(resp.Role),
	}, nil
}

func (s *PermissionService) readConfig(ctx context.Context, sourceID, targetID uint32, key string) (string, error) {
	resp, err := s.mgmt.ConfigGet(ctx, sourceID, targetID, key)
	if err != nil {
		if isNotFoundErr(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(resp.Value), nil
}

func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "not found") || strings.Contains(msg, "code=404")
}

func parseRawPolicy(raw RawPolicy) (Policy, []string, error) {
	policy := Policy{
		DefaultRole:  strings.TrimSpace(raw.AuthDefaultRole),
		DefaultPerms: parseCSV(raw.AuthDefaultPerms),
		NodeRoles:    parseNodeRoles(raw.AuthNodeRoles),
		RolePerms:    parseRolePerms(raw.AuthRolePerms),
	}
	if policy.DefaultRole == "" {
		policy.DefaultRole = "node"
	}
	normalized, warnings, err := normalizePolicy(policy)
	return normalized, warnings, err
}

func normalizePolicy(in Policy) (Policy, []string, error) {
	out := Policy{}
	warnings := make([]string, 0, 2)

	role := strings.TrimSpace(in.DefaultRole)
	if role == "" {
		return Policy{}, nil, errors.New("default role is required")
	}
	if err := validateToken("default role", role); err != nil {
		return Policy{}, nil, err
	}
	out.DefaultRole = role
	out.DefaultPerms = normalizePerms(in.DefaultPerms)

	nodeSeen := make(map[uint32]bool, len(in.NodeRoles))
	out.NodeRoles = make([]NodeRole, 0, len(in.NodeRoles))
	for _, entry := range in.NodeRoles {
		if entry.NodeID == 0 {
			return Policy{}, nil, errors.New("node role nodeId must be a positive number")
		}
		if nodeSeen[entry.NodeID] {
			return Policy{}, nil, fmt.Errorf("duplicate node role for nodeId=%d", entry.NodeID)
		}
		nodeSeen[entry.NodeID] = true

		r := strings.TrimSpace(entry.Role)
		if r == "" {
			return Policy{}, nil, fmt.Errorf("node role for nodeId=%d is empty", entry.NodeID)
		}
		if err := validateToken("node role", r); err != nil {
			return Policy{}, nil, err
		}
		out.NodeRoles = append(out.NodeRoles, NodeRole{
			NodeID: entry.NodeID,
			Role:   r,
		})
	}
	sort.Slice(out.NodeRoles, func(i, j int) bool { return out.NodeRoles[i].NodeID < out.NodeRoles[j].NodeID })

	roleSeen := make(map[string]bool, len(in.RolePerms))
	out.RolePerms = make([]RolePerm, 0, len(in.RolePerms))
	for _, entry := range in.RolePerms {
		r := strings.TrimSpace(entry.Role)
		if r == "" {
			return Policy{}, nil, errors.New("role permissions role is empty")
		}
		if err := validateToken("role permissions role", r); err != nil {
			return Policy{}, nil, err
		}
		if roleSeen[r] {
			return Policy{}, nil, fmt.Errorf("duplicate role permissions for role=%s", r)
		}
		roleSeen[r] = true
		out.RolePerms = append(out.RolePerms, RolePerm{
			Role:  r,
			Perms: normalizePerms(entry.Perms),
		})
	}
	sort.Slice(out.RolePerms, func(i, j int) bool { return out.RolePerms[i].Role < out.RolePerms[j].Role })

	for _, entry := range out.NodeRoles {
		if !roleSeen[entry.Role] {
			warnings = append(warnings, fmt.Sprintf("role '%s' used by node %d has no explicit role_perms entry; default perms will apply", entry.Role, entry.NodeID))
		}
	}

	return out, warnings, nil
}

func validateToken(name, token string) error {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return fmt.Errorf("%s is empty", name)
	}
	if strings.ContainsAny(trimmed, ",:;") {
		return fmt.Errorf("%s contains an invalid separator: %q", name, trimmed)
	}
	return nil
}

func normalizePerms(perms []string) []string {
	if len(perms) == 0 {
		return nil
	}
	set := make(map[string]bool, len(perms))
	out := make([]string, 0, len(perms))
	for _, entry := range perms {
		p := strings.TrimSpace(entry)
		if p == "" {
			continue
		}
		if strings.ContainsAny(p, ",:;") {
			continue
		}
		if set[p] {
			continue
		}
		set[p] = true
		out = append(out, p)
	}
	slices.Sort(out)
	return out
}

func parseCSV(raw string) []string {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	if len(parts) == 0 {
		return nil
	}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func parseNodeRoles(raw string) []NodeRole {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	items := strings.Split(raw, ";")
	out := make([]NodeRole, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		id, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32)
		if err != nil || id == 0 {
			continue
		}
		role := strings.TrimSpace(parts[1])
		if role == "" {
			continue
		}
		out = append(out, NodeRole{
			NodeID: uint32(id),
			Role:   role,
		})
	}
	return out
}

func parseRolePerms(raw string) []RolePerm {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	items := strings.Split(raw, ";")
	out := make([]RolePerm, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		role := strings.TrimSpace(parts[0])
		if role == "" {
			continue
		}
		perms := parseCSV(parts[1])
		out = append(out, RolePerm{
			Role:  role,
			Perms: perms,
		})
	}
	return out
}

func toRawPolicy(policy Policy) RawPolicy {
	nodeItems := make([]string, 0, len(policy.NodeRoles))
	for _, item := range policy.NodeRoles {
		nodeItems = append(nodeItems, fmt.Sprintf("%d:%s", item.NodeID, item.Role))
	}
	roleItems := make([]string, 0, len(policy.RolePerms))
	for _, item := range policy.RolePerms {
		roleItems = append(roleItems, fmt.Sprintf("%s:%s", item.Role, strings.Join(item.Perms, ",")))
	}
	return RawPolicy{
		AuthDefaultRole:  strings.TrimSpace(policy.DefaultRole),
		AuthDefaultPerms: strings.Join(policy.DefaultPerms, ","),
		AuthNodeRoles:    strings.Join(nodeItems, ";"),
		AuthRolePerms:    strings.Join(roleItems, ";"),
	}
}

func toSnapshot(policy Policy) coreperm.Snapshot {
	nodeRoles := make(map[uint32]string, len(policy.NodeRoles))
	for _, entry := range policy.NodeRoles {
		nodeRoles[entry.NodeID] = entry.Role
	}
	rolePerms := make(map[string][]string, len(policy.RolePerms))
	for _, entry := range policy.RolePerms {
		rolePerms[entry.Role] = cloneSortedTokens(entry.Perms)
	}
	return coreperm.Snapshot{
		DefaultRole:  policy.DefaultRole,
		DefaultPerms: cloneSortedTokens(policy.DefaultPerms),
		NodeRoles:    nodeRoles,
		RolePerms:    rolePerms,
	}
}

func toRuntimeRoles(items []protoauth.RolePermEntry) []RuntimeRole {
	if len(items) == 0 {
		return nil
	}
	out := make([]RuntimeRole, 0, len(items))
	for _, item := range items {
		if item.NodeID == 0 {
			continue
		}
		out = append(out, RuntimeRole{
			NodeID: item.NodeID,
			Role:   strings.TrimSpace(item.Role),
			Perms:  cloneSortedTokens(item.Perms),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NodeID < out[j].NodeID })
	return out
}

func toPendingRegisters(items []authsvc.PendingRegisterInfo) []PendingRegister {
	if len(items) == 0 {
		return nil
	}
	out := make([]PendingRegister, 0, len(items))
	for _, item := range items {
		requestID := strings.TrimSpace(item.RequestID)
		deviceID := strings.TrimSpace(item.DeviceID)
		if requestID == "" && deviceID == "" {
			continue
		}
		out = append(out, PendingRegister{
			RequestID:     requestID,
			DeviceID:      deviceID,
			RequestedRole: strings.TrimSpace(item.RequestedRole),
			DisplayName:   strings.TrimSpace(item.DisplayName),
			CreatedAt:     item.CreatedAt,
			ExpiresAt:     item.ExpiresAt,
		})
	}
	return out
}

func cloneSortedTokens(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	slices.Sort(out)
	return out
}
