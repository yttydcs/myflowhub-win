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
	return successResult(s.backend.Status())
}

func (s toolSet) sessionConnect(_ context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[sessionConnectArgs](raw)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	if err := s.backend.Connect(args.Endpoint); err != nil {
		return errorResult(err.Error(), nil)
	}
	return successResult(s.backend.Status())
}

func (s toolSet) sessionDisconnect(_ context.Context, _ json.RawMessage) CallToolResult {
	if err := s.backend.Disconnect(); err != nil {
		return errorResult(err.Error(), nil)
	}
	return successResult(s.backend.Status())
}

func (s toolSet) authRegister(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[authRegisterArgs](raw)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	if !s.backend.SessionConnected() {
		return errorResult("not connected", nil)
	}
	deviceID, err := s.resolveDeviceID(args.DeviceID)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	sourceID, targetID, err := s.resolveAuthRoute(args.SourceID, args.TargetID)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	resp, err := s.backend.Register(ctx, sourceID, targetID, deviceID)
	if err != nil {
		return errorResult(err.Error(), map[string]any{"source_id": sourceID, "target_id": targetID, "device_id": deviceID})
	}
	if err := s.backend.CompleteAuth(resp, deviceID); err != nil {
		return errorResult(err.Error(), nil)
	}
	return successResult(map[string]any{
		"source_id": sourceID,
		"target_id": targetID,
		"device_id": deviceID,
		"response":  resp,
		"status":    s.backend.Status(),
	})
}

func (s toolSet) authLogin(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[authLoginArgs](raw)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	if !s.backend.SessionConnected() {
		return errorResult("not connected", nil)
	}
	deviceID, err := s.resolveDeviceID(args.DeviceID)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	nodeID, err := s.resolveLoginNodeID(args.NodeID)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	sourceID, targetID, err := s.resolveAuthRoute(args.SourceID, args.TargetID)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	resp, err := s.backend.Login(ctx, sourceID, targetID, deviceID, nodeID)
	if err != nil {
		return errorResult(err.Error(), map[string]any{"source_id": sourceID, "target_id": targetID, "device_id": deviceID, "node_id": nodeID})
	}
	if err := s.backend.CompleteAuth(resp, deviceID); err != nil {
		return errorResult(err.Error(), nil)
	}
	return successResult(map[string]any{
		"source_id": sourceID,
		"target_id": targetID,
		"device_id": deviceID,
		"node_id":   nodeID,
		"response":  resp,
		"status":    s.backend.Status(),
	})
}

func (s toolSet) managementListNodes(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[managementArgs](raw)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	if !s.backend.SessionConnected() {
		return errorResult("not connected", nil)
	}
	sourceID, targetID, err := s.resolveManagementRoute(args.SourceID, args.TargetID)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	resp, err := s.backend.ListNodes(ctx, sourceID, targetID)
	if err != nil {
		return errorResult(err.Error(), map[string]any{"source_id": sourceID, "target_id": targetID})
	}
	return successResult(map[string]any{"source_id": sourceID, "target_id": targetID, "response": resp})
}

func (s toolSet) managementNodeInfo(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[managementArgs](raw)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	if !s.backend.SessionConnected() {
		return errorResult("not connected", nil)
	}
	sourceID, targetID, err := s.resolveManagementRoute(args.SourceID, args.TargetID)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	resp, err := s.backend.NodeInfo(ctx, sourceID, targetID)
	if err != nil {
		return errorResult(err.Error(), map[string]any{"source_id": sourceID, "target_id": targetID})
	}
	return successResult(map[string]any{"source_id": sourceID, "target_id": targetID, "response": resp})
}

func (s toolSet) varstoreList(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[varListArgs](raw)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	if !s.backend.SessionConnected() {
		return errorResult("not connected", nil)
	}
	sourceID, owner, targetID, err := s.resolveVarRoute(args.SourceID, args.Owner, args.TargetID)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	resp, err := s.backend.VarList(ctx, sourceID, targetID, protovarstore.ListReq{Owner: owner})
	if err != nil {
		return errorResult(err.Error(), map[string]any{"source_id": sourceID, "target_id": targetID, "owner": owner})
	}
	return successResult(map[string]any{"source_id": sourceID, "target_id": targetID, "owner": owner, "response": resp})
}

func (s toolSet) varstoreGet(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[varGetArgs](raw)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return errorResult("name is required", nil)
	}
	if !s.backend.SessionConnected() {
		return errorResult("not connected", nil)
	}
	sourceID, owner, targetID, err := s.resolveVarRoute(args.SourceID, args.Owner, args.TargetID)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	resp, err := s.backend.VarGet(ctx, sourceID, targetID, protovarstore.GetReq{Name: name, Owner: owner})
	if err != nil {
		return errorResult(err.Error(), map[string]any{"source_id": sourceID, "target_id": targetID, "owner": owner, "name": name})
	}
	return successResult(map[string]any{"source_id": sourceID, "target_id": targetID, "owner": owner, "name": name, "response": resp})
}

func (s toolSet) varstoreSet(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[varSetArgs](raw)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return errorResult("name is required", nil)
	}
	if strings.TrimSpace(args.Value) == "" {
		return errorResult("value is required", nil)
	}
	if !s.backend.AllowWrite() {
		return errorResult("write operations are disabled; restart with --allow-write=true to enable myflowhub_varstore_set", nil)
	}
	if !s.backend.SessionConnected() {
		return errorResult("not connected", nil)
	}
	sourceID, owner, targetID, err := s.resolveVarRoute(args.SourceID, args.Owner, args.TargetID)
	if err != nil {
		return errorResult(err.Error(), nil)
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
		return errorResult(err.Error(), map[string]any{"source_id": sourceID, "target_id": targetID, "owner": owner, "name": name})
	}
	return successResult(map[string]any{"source_id": sourceID, "target_id": targetID, "owner": owner, "request": req, "response": resp})
}

func (s toolSet) varstoreRevoke(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[varGetArgs](raw)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return errorResult("name is required", nil)
	}
	if !s.backend.AllowWrite() {
		return errorResult("write operations are disabled; restart with --allow-write=true to enable myflowhub_varstore_revoke", nil)
	}
	if !s.backend.SessionConnected() {
		return errorResult("not connected", nil)
	}
	sourceID, owner, targetID, err := s.resolveVarRoute(args.SourceID, args.Owner, args.TargetID)
	if err != nil {
		return errorResult(err.Error(), nil)
	}
	req := protovarstore.GetReq{Name: name, Owner: owner}
	resp, err := s.backend.VarRevoke(ctx, sourceID, targetID, req)
	if err != nil {
		return errorResult(err.Error(), map[string]any{"source_id": sourceID, "target_id": targetID, "owner": owner, "name": name})
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
