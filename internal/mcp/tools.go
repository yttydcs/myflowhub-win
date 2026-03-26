package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	protoauth "github.com/yttydcs/myflowhub-proto/protocol/auth"
	protomanagement "github.com/yttydcs/myflowhub-proto/protocol/management"
	protovarstore "github.com/yttydcs/myflowhub-proto/protocol/varstore"
	"github.com/yttydcs/myflowhub-win/internal/mcpapp"
)

type Backend interface {
	Status() mcpapp.Status
	SessionConnected() bool
	Connect(endpoint string) error
	Disconnect() error
	Defaults() mcpapp.Defaults
	AuthSnapshot() mcpapp.AuthSnapshot
	AllowWrite() bool
	Register(ctx context.Context, sourceID, targetID uint32, deviceID string) (protoauth.RespData, error)
	Login(ctx context.Context, sourceID, targetID uint32, deviceID string, nodeID uint32) (protoauth.RespData, error)
	CompleteAuth(resp protoauth.RespData, deviceID string) error
	ListNodes(ctx context.Context, sourceID, targetID uint32) (protomanagement.ListNodesResp, error)
	NodeInfo(ctx context.Context, sourceID, targetID uint32) (protomanagement.NodeInfoResp, error)
	VarList(ctx context.Context, sourceID, targetID uint32, req protovarstore.ListReq) (protovarstore.VarResp, error)
	VarGet(ctx context.Context, sourceID, targetID uint32, req protovarstore.GetReq) (protovarstore.VarResp, error)
	VarSet(ctx context.Context, sourceID, targetID uint32, req protovarstore.SetReq) (protovarstore.VarResp, error)
	VarRevoke(ctx context.Context, sourceID, targetID uint32, req protovarstore.GetReq) (protovarstore.VarResp, error)
}

type toolSet struct {
	backend Backend
}

type sessionConnectArgs struct {
	Endpoint string `json:"endpoint,omitempty"`
}

type authRegisterArgs struct {
	DeviceID string  `json:"device_id,omitempty"`
	SourceID *uint32 `json:"source_id,omitempty"`
	TargetID *uint32 `json:"target_id,omitempty"`
}

type authLoginArgs struct {
	DeviceID string  `json:"device_id,omitempty"`
	NodeID   *uint32 `json:"node_id,omitempty"`
	SourceID *uint32 `json:"source_id,omitempty"`
	TargetID *uint32 `json:"target_id,omitempty"`
}

type managementArgs struct {
	SourceID *uint32 `json:"source_id,omitempty"`
	TargetID *uint32 `json:"target_id,omitempty"`
}

type varListArgs struct {
	SourceID *uint32 `json:"source_id,omitempty"`
	TargetID *uint32 `json:"target_id,omitempty"`
	Owner    *uint32 `json:"owner,omitempty"`
}

type varGetArgs struct {
	SourceID *uint32 `json:"source_id,omitempty"`
	TargetID *uint32 `json:"target_id,omitempty"`
	Name     string  `json:"name"`
	Owner    *uint32 `json:"owner,omitempty"`
}

type varSetArgs struct {
	SourceID   *uint32 `json:"source_id,omitempty"`
	TargetID   *uint32 `json:"target_id,omitempty"`
	Name       string  `json:"name"`
	Value      string  `json:"value"`
	Owner      *uint32 `json:"owner,omitempty"`
	Visibility string  `json:"visibility,omitempty"`
	Type       string  `json:"type,omitempty"`
}

type sessionStatusPayload struct {
	Connected   bool                `json:"connected"`
	Endpoint    string              `json:"endpoint,omitempty"`
	Auth        mcpapp.AuthSnapshot `json:"auth"`
	Defaults    mcpapp.Defaults     `json:"defaults"`
	Config      mcpapp.ConfigState  `json:"config"`
	Permissions statusPermissions   `json:"permissions"`
	Readiness   statusReadiness     `json:"readiness"`
	Hints       []string            `json:"hints,omitempty"`
}

type statusPermissions struct {
	AuthorizationModel string `json:"authorization_model"`
	LocalWriteGate     bool   `json:"local_write_gate"`
}

type statusReadiness struct {
	Authenticated bool `json:"authenticated"`
	HasIdentity   bool `json:"has_identity"`
	HasTarget     bool `json:"has_target"`
	CanRegister   bool `json:"can_register"`
	CanLogin      bool `json:"can_login"`
	CanManage     bool `json:"can_manage"`
	CanVarRead    bool `json:"can_var_read"`
	CanVarWrite   bool `json:"can_var_write"`
}

func NewTools(backend Backend) []Tool {
	set := toolSet{backend: backend}
	return []Tool{
		{
			Name:        "myflowhub_session_status",
			Description: "Show current session, auth snapshot, config paths, and defaults.",
			InputSchema: objectSchema(nil),
			Handler:     set.sessionStatus,
		},
		{
			Name:        "myflowhub_session_connect",
			Description: "Connect the headless MyFlowHub MCP client to a hub endpoint.",
			InputSchema: objectSchema(map[string]any{
				"endpoint": stringSchema("Hub endpoint such as 127.0.0.1:9000."),
			}),
			Handler: set.sessionConnect,
		},
		{
			Name:        "myflowhub_session_disconnect",
			Description: "Disconnect the current session and clear the logged-in flag.",
			InputSchema: objectSchema(nil),
			Handler:     set.sessionDisconnect,
		},
		{
			Name:        "myflowhub_auth_register",
			Description: "Register this MCP client as a MyFlowHub node.",
			InputSchema: objectSchema(map[string]any{
				"device_id": stringSchema("Device ID. Falls back to the current snapshot or startup default."),
				"source_id": positiveIntegerSchema("Optional source node ID. Zero is allowed for initial auth."),
				"target_id": positiveIntegerSchema("Optional target node ID. Zero is allowed for initial auth."),
			}),
			Handler: set.authRegister,
		},
		{
			Name:        "myflowhub_auth_login",
			Description: "Login an existing MyFlowHub node using local node keys.",
			InputSchema: objectSchema(map[string]any{
				"device_id": stringSchema("Device ID. Falls back to the current snapshot or startup default."),
				"node_id":   positiveIntegerSchema("Node ID to login. Falls back to the current snapshot or startup default."),
				"source_id": positiveIntegerSchema("Optional source node ID. Zero is allowed for auth flows."),
				"target_id": positiveIntegerSchema("Optional target node ID. Zero is allowed for auth flows."),
			}),
			Handler: set.authLogin,
		},
		{
			Name:        "myflowhub_management_list_nodes",
			Description: "List nodes under the current hub.",
			InputSchema: objectSchema(map[string]any{
				"source_id": positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id": positiveIntegerSchema("Target node ID. Falls back to hub_id or default_target."),
			}),
			Handler: set.managementListNodes,
		},
		{
			Name:        "myflowhub_management_node_info",
			Description: "Get node information from the current hub or a specific node.",
			InputSchema: objectSchema(map[string]any{
				"source_id": positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id": positiveIntegerSchema("Target node ID. Falls back to hub_id or default_target."),
			}),
			Handler: set.managementNodeInfo,
		},
		{
			Name:        "myflowhub_varstore_list",
			Description: "List variable names for an owner node.",
			InputSchema: objectSchema(map[string]any{
				"source_id": positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id": positiveIntegerSchema("Target node ID. Falls back to owner, hub_id, or default_target."),
				"owner":     positiveIntegerSchema("Owner node ID. Falls back to source_id."),
			}),
			Handler: set.varstoreList,
		},
		{
			Name:        "myflowhub_varstore_get",
			Description: "Get a variable from the varstore.",
			InputSchema: objectSchema(map[string]any{
				"source_id": positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id": positiveIntegerSchema("Target node ID. Falls back to owner, hub_id, or default_target."),
				"name":      stringSchema("Variable name."),
				"owner":     positiveIntegerSchema("Owner node ID. Falls back to source_id."),
			}, "name"),
			Handler: set.varstoreGet,
		},
		{
			Name:        "myflowhub_varstore_set",
			Description: "Create or update a variable in the varstore. This is disabled unless allow_write=true.",
			InputSchema: objectSchema(map[string]any{
				"source_id":  positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":  positiveIntegerSchema("Target node ID. Falls back to owner, hub_id, or default_target."),
				"name":       stringSchema("Variable name."),
				"value":      stringSchema("Variable value."),
				"owner":      positiveIntegerSchema("Owner node ID. Falls back to source_id."),
				"visibility": stringSchema("Visibility. Defaults to public."),
				"type":       stringSchema("Optional logical type tag."),
			}, "name", "value"),
			Handler: set.varstoreSet,
		},
		{
			Name:        "myflowhub_varstore_revoke",
			Description: "Delete a variable in the varstore. This is disabled unless allow_write=true.",
			InputSchema: objectSchema(map[string]any{
				"source_id": positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id": positiveIntegerSchema("Target node ID. Falls back to owner, hub_id, or default_target."),
				"name":      stringSchema("Variable name."),
				"owner":     positiveIntegerSchema("Owner node ID. Falls back to source_id."),
			}, "name"),
			Handler: set.varstoreRevoke,
		},
	}
}

func (s toolSet) sessionStatus(_ context.Context, _ json.RawMessage) CallToolResult {
	return successResult(s.buildSessionStatus())
}

func (s toolSet) sessionConnect(_ context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[sessionConnectArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object such as {\"endpoint\":\"127.0.0.1:9000\"}.", nil)
	}
	if err := s.backend.Connect(args.Endpoint); err != nil {
		return upstreamErrorResult(err, "Check the endpoint and that the hub is reachable.", nil)
	}
	return successResult(s.buildSessionStatus())
}

func (s toolSet) sessionDisconnect(_ context.Context, _ json.RawMessage) CallToolResult {
	if err := s.backend.Disconnect(); err != nil {
		return upstreamErrorResult(err, "Check whether the session is still healthy before retrying disconnect.", nil)
	}
	return successResult(s.buildSessionStatus())
}

func (s toolSet) authRegister(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[authRegisterArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with optional device_id, source_id, and target_id.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_auth_register")
	}
	deviceID, err := s.resolveDeviceID(args.DeviceID)
	if err != nil {
		return identityErrorResult(err, "Pass device_id explicitly or configure --device-id for the MCP process.")
	}
	sourceID, targetID, err := s.resolveAuthRoute(args.SourceID, args.TargetID)
	if err != nil {
		return routeErrorResult(err, "Pass source_id and target_id explicitly for the auth flow if login defaults are not ready.")
	}
	resp, err := s.backend.Register(ctx, sourceID, targetID, deviceID)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and role permissions for auth/register.", map[string]any{"source_id": sourceID, "target_id": targetID, "device_id": deviceID})
	}
	if err := s.backend.CompleteAuth(resp, deviceID); err != nil {
		return upstreamErrorResult(err, "The auth response succeeded but the local MCP state could not be persisted.", nil)
	}
	return successResult(map[string]any{
		"source_id": sourceID,
		"target_id": targetID,
		"device_id": deviceID,
		"response":  resp,
		"status":    s.buildSessionStatus(),
	})
}

func (s toolSet) authLogin(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[authLoginArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with optional device_id, node_id, source_id, and target_id.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_auth_login")
	}
	deviceID, err := s.resolveDeviceID(args.DeviceID)
	if err != nil {
		return identityErrorResult(err, "Pass device_id explicitly or configure --device-id for the MCP process.")
	}
	nodeID, err := s.resolveLoginNodeID(args.NodeID)
	if err != nil {
		return identityErrorResult(err, "Pass node_id explicitly or login after a successful register response.")
	}
	sourceID, targetID, err := s.resolveAuthRoute(args.SourceID, args.TargetID)
	if err != nil {
		return routeErrorResult(err, "Pass source_id and target_id explicitly for the auth flow if login defaults are not ready.")
	}
	resp, err := s.backend.Login(ctx, sourceID, targetID, deviceID, nodeID)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity, node keys, and role permissions for auth/login.", map[string]any{"source_id": sourceID, "target_id": targetID, "device_id": deviceID, "node_id": nodeID})
	}
	if err := s.backend.CompleteAuth(resp, deviceID); err != nil {
		return upstreamErrorResult(err, "The auth response succeeded but the local MCP state could not be persisted.", nil)
	}
	return successResult(map[string]any{
		"source_id": sourceID,
		"target_id": targetID,
		"device_id": deviceID,
		"node_id":   nodeID,
		"response":  resp,
		"status":    s.buildSessionStatus(),
	})
}

func (s toolSet) managementListNodes(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[managementArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with optional source_id and target_id.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_management_list_nodes")
	}
	sourceID, targetID, err := s.resolveManagementRoute(args.SourceID, args.TargetID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id and target_id explicitly.")
	}
	resp, err := s.backend.ListNodes(ctx, sourceID, targetID)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can read management data.", map[string]any{"source_id": sourceID, "target_id": targetID})
	}
	return successResult(map[string]any{"source_id": sourceID, "target_id": targetID, "response": resp})
}

func (s toolSet) managementNodeInfo(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[managementArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with optional source_id and target_id.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_management_node_info")
	}
	sourceID, targetID, err := s.resolveManagementRoute(args.SourceID, args.TargetID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id and target_id explicitly.")
	}
	resp, err := s.backend.NodeInfo(ctx, sourceID, targetID)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can read node information.", map[string]any{"source_id": sourceID, "target_id": targetID})
	}
	return successResult(map[string]any{"source_id": sourceID, "target_id": targetID, "response": resp})
}

func (s toolSet) varstoreList(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[varListArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with optional source_id, owner, and target_id.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_varstore_list")
	}
	sourceID, owner, targetID, err := s.resolveVarRoute(args.SourceID, args.Owner, args.TargetID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id, owner, and target_id explicitly.")
	}
	resp, err := s.backend.VarList(ctx, sourceID, targetID, protovarstore.ListReq{Owner: owner})
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can read varstore data.", map[string]any{"source_id": sourceID, "target_id": targetID, "owner": owner})
	}
	return successResult(map[string]any{"source_id": sourceID, "target_id": targetID, "owner": owner, "response": resp})
}

func (s toolSet) varstoreGet(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[varGetArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with name and optional source_id, owner, and target_id.", nil)
	}
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return invalidArgumentsResult("name is required", "Provide a non-empty variable name.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_varstore_get")
	}
	sourceID, owner, targetID, err := s.resolveVarRoute(args.SourceID, args.Owner, args.TargetID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id, owner, and target_id explicitly.")
	}
	resp, err := s.backend.VarGet(ctx, sourceID, targetID, protovarstore.GetReq{Name: name, Owner: owner})
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can read the requested variable.", map[string]any{"source_id": sourceID, "target_id": targetID, "owner": owner, "name": name})
	}
	return successResult(map[string]any{"source_id": sourceID, "target_id": targetID, "owner": owner, "name": name, "response": resp})
}

func (s toolSet) varstoreSet(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[varSetArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with name, value, and optional source_id, owner, target_id, visibility, and type.", nil)
	}
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return invalidArgumentsResult("name is required", "Provide a non-empty variable name.", nil)
	}
	if !s.backend.AllowWrite() {
		return writeDisabledResult("myflowhub_varstore_set")
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_varstore_set")
	}
	sourceID, owner, targetID, err := s.resolveVarRoute(args.SourceID, args.Owner, args.TargetID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id, owner, and target_id explicitly.")
	}
	req := protovarstore.SetReq{
		Name:       name,
		Value:      args.Value,
		Owner:      owner,
		Visibility: normalizeVisibility(args.Visibility),
		Type:       strings.TrimSpace(args.Type),
	}
	resp, err := s.backend.VarSet(ctx, sourceID, targetID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can modify the requested variable.", map[string]any{"source_id": sourceID, "target_id": targetID, "owner": owner, "name": name})
	}
	return successResult(map[string]any{"source_id": sourceID, "target_id": targetID, "owner": owner, "request": req, "response": resp})
}

func (s toolSet) varstoreRevoke(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[varGetArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with name and optional source_id, owner, and target_id.", nil)
	}
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return invalidArgumentsResult("name is required", "Provide a non-empty variable name.", nil)
	}
	if !s.backend.AllowWrite() {
		return writeDisabledResult("myflowhub_varstore_revoke")
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_varstore_revoke")
	}
	sourceID, owner, targetID, err := s.resolveVarRoute(args.SourceID, args.Owner, args.TargetID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id, owner, and target_id explicitly.")
	}
	req := protovarstore.GetReq{Name: name, Owner: owner}
	resp, err := s.backend.VarRevoke(ctx, sourceID, targetID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can delete the requested variable.", map[string]any{"source_id": sourceID, "target_id": targetID, "owner": owner, "name": name})
	}
	return successResult(map[string]any{"source_id": sourceID, "target_id": targetID, "owner": owner, "request": req, "response": resp})
}

func (s toolSet) resolveDeviceID(explicit string) (string, error) {
	if deviceID := strings.TrimSpace(explicit); deviceID != "" {
		return deviceID, nil
	}
	if snap := s.backend.AuthSnapshot(); snap.LoggedIn && strings.TrimSpace(snap.DeviceID) != "" {
		return strings.TrimSpace(snap.DeviceID), nil
	}
	if defaults := s.backend.Defaults(); strings.TrimSpace(defaults.DeviceID) != "" {
		return strings.TrimSpace(defaults.DeviceID), nil
	}
	return "", errors.New("device_id is required")
}

func (s toolSet) resolveLoginNodeID(explicit *uint32) (uint32, error) {
	if explicit != nil {
		if *explicit == 0 {
			return 0, errors.New("node_id must be greater than 0")
		}
		return *explicit, nil
	}
	if snap := s.backend.AuthSnapshot(); snap.LoggedIn && snap.NodeID != 0 {
		return snap.NodeID, nil
	}
	if defaults := s.backend.Defaults(); defaults.NodeID != 0 {
		return defaults.NodeID, nil
	}
	return 0, errors.New("node_id is required")
}

func (s toolSet) resolveAuthRoute(sourceID, targetID *uint32) (uint32, uint32, error) {
	defaults := s.backend.Defaults()
	source, err := authRouteID(sourceID, s.backend.AuthSnapshot().NodeID, defaults.NodeID)
	if err != nil {
		return 0, 0, err
	}
	targetFallback := defaults.DefaultTarget
	if targetFallback == 0 {
		targetFallback = defaults.HubID
	}
	target, err := authRouteID(targetID, s.backend.AuthSnapshot().HubID, targetFallback)
	if err != nil {
		return 0, 0, err
	}
	return source, target, nil
}

func (s toolSet) resolveManagementRoute(sourceID, targetID *uint32) (uint32, uint32, error) {
	source, explicit, err := positiveNodeID(sourceID, "source_id")
	if err != nil {
		return 0, 0, err
	}
	if !explicit {
		if snap := s.backend.AuthSnapshot(); snap.LoggedIn && snap.NodeID != 0 {
			source = snap.NodeID
		} else if defaults := s.backend.Defaults(); defaults.NodeID != 0 {
			source = defaults.NodeID
		}
	}
	if source == 0 {
		return 0, 0, errors.New("source_id is required; login first or pass source_id")
	}

	target, explicit, err := positiveNodeID(targetID, "target_id")
	if err != nil {
		return 0, 0, err
	}
	if !explicit {
		if snap := s.backend.AuthSnapshot(); snap.LoggedIn && snap.HubID != 0 {
			target = snap.HubID
		} else {
			defaults := s.backend.Defaults()
			if defaults.DefaultTarget != 0 {
				target = defaults.DefaultTarget
			} else if defaults.HubID != 0 {
				target = defaults.HubID
			}
		}
	}
	if target == 0 {
		return 0, 0, errors.New("target_id is required; login first or pass target_id")
	}
	return source, target, nil
}

func (s toolSet) resolveVarRoute(sourceID, ownerID, targetID *uint32) (uint32, uint32, uint32, error) {
	source, err := s.resolveVarSource(sourceID)
	if err != nil {
		return 0, 0, 0, err
	}
	owner, explicit, err := positiveNodeID(ownerID, "owner")
	if err != nil {
		return 0, 0, 0, err
	}
	if !explicit || owner == 0 {
		owner = source
	}

	target, explicit, err := positiveNodeID(targetID, "target_id")
	if err != nil {
		return 0, 0, 0, err
	}
	if !explicit {
		target = owner
		if target == 0 {
			if snap := s.backend.AuthSnapshot(); snap.LoggedIn && snap.HubID != 0 {
				target = snap.HubID
			} else {
				defaults := s.backend.Defaults()
				if defaults.DefaultTarget != 0 {
					target = defaults.DefaultTarget
				} else if defaults.HubID != 0 {
					target = defaults.HubID
				}
			}
		}
	}
	if target == 0 {
		return 0, 0, 0, errors.New("target_id is required; login first, pass owner, or pass target_id")
	}
	return source, owner, target, nil
}

func (s toolSet) resolveVarSource(sourceID *uint32) (uint32, error) {
	source, explicit, err := positiveNodeID(sourceID, "source_id")
	if err != nil {
		return 0, err
	}
	if !explicit {
		if snap := s.backend.AuthSnapshot(); snap.LoggedIn && snap.NodeID != 0 {
			source = snap.NodeID
		} else if defaults := s.backend.Defaults(); defaults.NodeID != 0 {
			source = defaults.NodeID
		}
	}
	if source == 0 {
		return 0, errors.New("source_id is required; login first or pass source_id")
	}
	return source, nil
}

func authRouteID(explicit *uint32, snapshot uint32, fallback uint32) (uint32, error) {
	if explicit != nil {
		return *explicit, nil
	}
	if snapshot != 0 {
		return snapshot, nil
	}
	if fallback != 0 {
		return fallback, nil
	}
	return 0, nil
}

func decodeArgs[T any](raw json.RawMessage) (T, error) {
	var zero T
	if len(raw) == 0 || string(raw) == "null" {
		return zero, nil
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return zero, err
	}
	return out, nil
}

func positiveNodeID(value *uint32, field string) (uint32, bool, error) {
	if value == nil {
		return 0, false, nil
	}
	if *value == 0 {
		return 0, true, fmt.Errorf("%s must be greater than 0", field)
	}
	return *value, true, nil
}

func normalizeVisibility(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "private" {
		return "private"
	}
	return "public"
}

func (s toolSet) buildSessionStatus() sessionStatusPayload {
	status := s.backend.Status()
	hasIdentity := status.Auth.NodeID != 0 || status.Defaults.NodeID != 0
	hasTarget := status.Auth.HubID != 0 || status.Defaults.DefaultTarget != 0 || status.Defaults.HubID != 0
	payload := sessionStatusPayload{
		Connected: status.Connected,
		Endpoint:  status.Endpoint,
		Auth:      status.Auth,
		Defaults:  status.Defaults,
		Config:    status.Config,
		Permissions: statusPermissions{
			AuthorizationModel: "hub_role_based",
			LocalWriteGate:     status.Config.AllowWrite,
		},
		Readiness: statusReadiness{
			Authenticated: status.Auth.LoggedIn,
			HasIdentity:   hasIdentity,
			HasTarget:     hasTarget,
			CanRegister:   status.Connected,
			CanLogin:      status.Connected && hasIdentity,
			CanManage:     status.Connected && hasIdentity && hasTarget,
			CanVarRead:    status.Connected && hasIdentity,
			CanVarWrite:   status.Connected && hasIdentity && status.Config.AllowWrite,
		},
	}
	payload.Hints = statusHints(payload)
	return payload
}

func statusHints(status sessionStatusPayload) []string {
	hints := make([]string, 0, 3)
	if !status.Connected {
		hints = append(hints, "Call myflowhub_session_connect before invoking auth, management, or varstore tools.")
		return hints
	}
	if !status.Auth.LoggedIn {
		if !status.Readiness.HasIdentity {
			hints = append(hints, "Call myflowhub_auth_register for a new node or pass explicit device_id/node_id values to myflowhub_auth_login.")
		} else {
			hints = append(hints, "Call myflowhub_auth_login to refresh an authenticated session before management or varstore operations.")
		}
	}
	if !status.Permissions.LocalWriteGate {
		hints = append(hints, "Restart the MCP process with --allow-write to enable myflowhub_varstore_set and myflowhub_varstore_revoke.")
	}
	if len(hints) == 0 {
		hints = append(hints, "Session is ready for management and varstore read operations.")
	}
	return hints
}

func invalidArgumentsResult(message, hint string, details any) CallToolResult {
	return errorResult("invalid_arguments", message, hint, details)
}

func notConnectedResult(toolName string) CallToolResult {
	return errorResult(
		"not_connected",
		"not connected",
		fmt.Sprintf("Call myflowhub_session_connect before %s.", strings.TrimSpace(toolName)),
		nil,
	)
}

func identityErrorResult(err error, hint string) CallToolResult {
	if err == nil {
		return errorResult("missing_identity", "missing identity", strings.TrimSpace(hint), nil)
	}
	msg := strings.TrimSpace(err.Error())
	if strings.Contains(msg, "must be greater than 0") {
		return invalidArgumentsResult(msg, "Pass a positive node_id, source_id, target_id, or owner value.", nil)
	}
	return errorResult("missing_identity", msg, strings.TrimSpace(hint), nil)
}

func routeErrorResult(err error, hint string) CallToolResult {
	return identityErrorResult(err, hint)
}

func writeDisabledResult(toolName string) CallToolResult {
	return errorResult(
		"write_disabled",
		"write operations are disabled",
		fmt.Sprintf("Restart the MCP process with --allow-write to enable %s.", strings.TrimSpace(toolName)),
		nil,
	)
}

func upstreamErrorResult(err error, hint string, details any) CallToolResult {
	message := "upstream request failed"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		message = strings.TrimSpace(err.Error())
	}
	return errorResult("upstream_error", message, strings.TrimSpace(hint), details)
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	out := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           map[string]any{},
	}
	if len(properties) > 0 {
		out["properties"] = properties
	}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
}

func stringSchema(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
	}
}

func positiveIntegerSchema(description string) map[string]any {
	return map[string]any{
		"type":        "integer",
		"minimum":     0,
		"description": description,
	}
}
