package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	protoauth "github.com/yttydcs/myflowhub-proto/protocol/auth"
	protomanagement "github.com/yttydcs/myflowhub-proto/protocol/management"
	protovarstore "github.com/yttydcs/myflowhub-proto/protocol/varstore"
	"github.com/yttydcs/myflowhub-win/internal/mcpapp"
)

type fakeBackend struct {
	status           mcpapp.Status
	defaults         mcpapp.Defaults
	auth             mcpapp.AuthSnapshot
	sessionConnected bool
	allowWrite       bool

	loginArgs struct {
		sourceID uint32
		targetID uint32
		deviceID string
		nodeID   uint32
		called   bool
	}
	listNodesArgs struct {
		sourceID uint32
		targetID uint32
		called   bool
	}
	varSetArgs struct {
		sourceID uint32
		targetID uint32
		req      protovarstore.SetReq
		called   bool
	}
}

func (f *fakeBackend) Status() mcpapp.Status  { return f.status }
func (f *fakeBackend) SessionConnected() bool { return f.sessionConnected }
func (f *fakeBackend) Connect(endpoint string) error {
	f.status.Endpoint = endpoint
	f.status.Connected = true
	f.sessionConnected = true
	return nil
}
func (f *fakeBackend) Disconnect() error {
	f.status.Connected = false
	f.sessionConnected = false
	return nil
}
func (f *fakeBackend) Defaults() mcpapp.Defaults         { return f.defaults }
func (f *fakeBackend) AuthSnapshot() mcpapp.AuthSnapshot { return f.auth }
func (f *fakeBackend) AllowWrite() bool                  { return f.allowWrite }
func (f *fakeBackend) Register(context.Context, uint32, uint32, string) (protoauth.RespData, error) {
	return protoauth.RespData{}, nil
}
func (f *fakeBackend) Login(_ context.Context, sourceID, targetID uint32, deviceID string, nodeID uint32) (protoauth.RespData, error) {
	f.loginArgs = struct {
		sourceID uint32
		targetID uint32
		deviceID string
		nodeID   uint32
		called   bool
	}{sourceID: sourceID, targetID: targetID, deviceID: deviceID, nodeID: nodeID, called: true}
	return protoauth.RespData{Code: 1, DeviceID: deviceID, NodeID: nodeID, HubID: targetID}, nil
}
func (f *fakeBackend) CompleteAuth(resp protoauth.RespData, deviceID string) error {
	f.auth = mcpapp.AuthSnapshot{
		DeviceID: deviceID,
		NodeID:   resp.NodeID,
		HubID:    resp.HubID,
		Role:     resp.Role,
		LoggedIn: resp.Code == 1,
	}
	return nil
}
func (f *fakeBackend) ListNodes(_ context.Context, sourceID, targetID uint32) (protomanagement.ListNodesResp, error) {
	f.listNodesArgs = struct {
		sourceID uint32
		targetID uint32
		called   bool
	}{sourceID: sourceID, targetID: targetID, called: true}
	return protomanagement.ListNodesResp{Code: 1, Nodes: []protomanagement.NodeInfo{{NodeID: targetID}}}, nil
}
func (f *fakeBackend) NodeInfo(context.Context, uint32, uint32) (protomanagement.NodeInfoResp, error) {
	return protomanagement.NodeInfoResp{Code: 1}, nil
}
func (f *fakeBackend) VarList(context.Context, uint32, uint32, protovarstore.ListReq) (protovarstore.VarResp, error) {
	return protovarstore.VarResp{Code: 1}, nil
}
func (f *fakeBackend) VarGet(context.Context, uint32, uint32, protovarstore.GetReq) (protovarstore.VarResp, error) {
	return protovarstore.VarResp{Code: 1}, nil
}
func (f *fakeBackend) VarSet(_ context.Context, sourceID, targetID uint32, req protovarstore.SetReq) (protovarstore.VarResp, error) {
	f.varSetArgs = struct {
		sourceID uint32
		targetID uint32
		req      protovarstore.SetReq
		called   bool
	}{sourceID: sourceID, targetID: targetID, req: req, called: true}
	return protovarstore.VarResp{Code: 1, Name: req.Name, Owner: req.Owner}, nil
}
func (f *fakeBackend) VarRevoke(context.Context, uint32, uint32, protovarstore.GetReq) (protovarstore.VarResp, error) {
	return protovarstore.VarResp{Code: 1}, nil
}

func TestManagementListNodesFallsBackToAuthSnapshot(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
	}

	result := findTool(t, NewTools(backend), "myflowhub_management_list_nodes").Handler(context.Background(), json.RawMessage(`{}`))
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.listNodesArgs.called {
		t.Fatal("expected ListNodes() called")
	}
	if backend.listNodesArgs.sourceID != 7 || backend.listNodesArgs.targetID != 9 {
		t.Fatalf("unexpected route: %+v", backend.listNodesArgs)
	}
}

func TestVarstoreSetBlockedWhenWriteDisabled(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		allowWrite:       false,
		auth:             mcpapp.AuthSnapshot{NodeID: 5, HubID: 9, LoggedIn: true},
	}

	result := findTool(t, NewTools(backend), "myflowhub_varstore_set").Handler(context.Background(), json.RawMessage(`{"name":"alpha","value":"1"}`))
	if !result.IsError {
		t.Fatalf("expected error result, got %#v", result)
	}
	if backend.varSetArgs.called {
		t.Fatal("expected VarSet() not called when allow_write=false")
	}
	payload, ok := result.StructuredContent.(toolErrorPayload)
	if !ok {
		t.Fatalf("expected toolErrorPayload, got %#v", result.StructuredContent)
	}
	if payload.Code != "write_disabled" {
		t.Fatalf("unexpected error code: %#v", payload)
	}
	if !strings.Contains(payload.Hint, "--allow-write") {
		t.Fatalf("expected allow-write hint, got %#v", payload)
	}
}

func TestAuthLoginFallsBackToStartupDefaults(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		defaults: mcpapp.Defaults{
			DeviceID:      "device-1",
			NodeID:        11,
			DefaultTarget: 99,
		},
	}

	result := findTool(t, NewTools(backend), "myflowhub_auth_login").Handler(context.Background(), json.RawMessage(`{}`))
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.loginArgs.called {
		t.Fatal("expected Login() called")
	}
	if backend.loginArgs.deviceID != "device-1" || backend.loginArgs.nodeID != 11 {
		t.Fatalf("unexpected login args: %+v", backend.loginArgs)
	}
	if backend.loginArgs.sourceID != 11 || backend.loginArgs.targetID != 99 {
		t.Fatalf("unexpected route: %+v", backend.loginArgs)
	}
}

func TestSessionStatusIncludesPermissionsAndReadiness(t *testing.T) {
	backend := &fakeBackend{
		status: mcpapp.Status{
			Connected: true,
			Endpoint:  "127.0.0.1:9000",
			Auth: mcpapp.AuthSnapshot{
				DeviceID: "ai-node",
				NodeID:   7,
				HubID:    9,
				Role:     "operator",
				LoggedIn: true,
			},
			Defaults: mcpapp.Defaults{
				DeviceID:      "ai-node",
				NodeID:        7,
				HubID:         9,
				DefaultTarget: 9,
				AllowWrite:    true,
			},
			Config: mcpapp.ConfigState{
				AllowWrite: true,
			},
		},
	}

	result := findTool(t, NewTools(backend), "myflowhub_session_status").Handler(context.Background(), json.RawMessage(`{}`))
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	payload, ok := result.StructuredContent.(sessionStatusPayload)
	if !ok {
		t.Fatalf("expected sessionStatusPayload, got %#v", result.StructuredContent)
	}
	if payload.Permissions.AuthorizationModel != "hub_role_based" {
		t.Fatalf("unexpected permissions payload: %#v", payload.Permissions)
	}
	if !payload.Readiness.CanVarWrite || !payload.Readiness.CanManage {
		t.Fatalf("unexpected readiness payload: %#v", payload.Readiness)
	}
	if len(payload.Hints) == 0 {
		t.Fatalf("expected hints, got %#v", payload)
	}
}

func TestManagementListNodesWithoutIdentityReturnsMissingIdentity(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		status: mcpapp.Status{
			Connected: true,
		},
	}

	result := findTool(t, NewTools(backend), "myflowhub_management_list_nodes").Handler(context.Background(), json.RawMessage(`{}`))
	if !result.IsError {
		t.Fatalf("expected error, got %#v", result)
	}
	payload, ok := result.StructuredContent.(toolErrorPayload)
	if !ok {
		t.Fatalf("expected toolErrorPayload, got %#v", result.StructuredContent)
	}
	if payload.Code != "missing_identity" {
		t.Fatalf("unexpected error code: %#v", payload)
	}
	if !strings.Contains(payload.Hint, "Login first") {
		t.Fatalf("expected login hint, got %#v", payload)
	}
}

func findTool(t *testing.T, tools []Tool, name string) Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %q not found", name)
	return Tool{}
}
