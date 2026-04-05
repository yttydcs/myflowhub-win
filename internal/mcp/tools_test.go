package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	protoauth "github.com/yttydcs/myflowhub-proto/protocol/auth"
	protoexec "github.com/yttydcs/myflowhub-proto/protocol/exec"
	protoflow "github.com/yttydcs/myflowhub-proto/protocol/flow"
	protomanagement "github.com/yttydcs/myflowhub-proto/protocol/management"
	protovarstore "github.com/yttydcs/myflowhub-proto/protocol/varstore"
	"github.com/yttydcs/myflowhub-win/internal/mcpapp"
	authsvc "github.com/yttydcs/myflowhub-win/internal/services/auth"
	flowsvc "github.com/yttydcs/myflowhub-win/internal/services/flow"
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
	getPermsArgs struct {
		sourceID uint32
		targetID uint32
		nodeID   uint32
		called   bool
	}
	listRolesArgs struct {
		sourceID uint32
		targetID uint32
		req      protoauth.ListRolesReq
		called   bool
	}
	listPendingArgs struct {
		sourceID uint32
		targetID uint32
		req      authsvc.ListPendingRegistersReq
		called   bool
	}
	approveArgs struct {
		sourceID uint32
		targetID uint32
		req      authsvc.ApproveRegisterReq
		called   bool
	}
	rejectArgs struct {
		sourceID uint32
		targetID uint32
		req      authsvc.RejectRegisterReq
		called   bool
	}
	issuePermitArgs struct {
		sourceID uint32
		targetID uint32
		req      authsvc.IssueRegisterPermitReq
		called   bool
	}
	revokePermitArgs struct {
		sourceID uint32
		targetID uint32
		req      authsvc.RevokeRegisterPermitReq
		called   bool
	}
	configGetArgs struct {
		sourceID uint32
		targetID uint32
		key      string
		called   bool
	}
	configListArgs struct {
		sourceID uint32
		targetID uint32
		called   bool
	}
	nodeEchoArgs struct {
		sourceID uint32
		targetID uint32
		message  string
		called   bool
	}
	listSubtreeArgs struct {
		sourceID uint32
		targetID uint32
		called   bool
	}
	execCapQueryArgs struct {
		sourceID uint32
		targetID uint32
		req      protoexec.CapQueryReq
		called   bool
	}
	flowSetArgs struct {
		sourceID uint32
		targetID uint32
		req      protoflow.SetReq
		called   bool
	}
	flowDeleteArgs struct {
		sourceID uint32
		targetID uint32
		req      flowsvc.DeleteReq
		called   bool
	}
	flowRunArgs struct {
		sourceID uint32
		targetID uint32
		req      protoflow.RunReq
		called   bool
	}
	flowCancelRunArgs struct {
		sourceID uint32
		targetID uint32
		req      protoflow.CancelRunReq
		called   bool
	}
	flowStatusArgs struct {
		sourceID uint32
		targetID uint32
		req      protoflow.StatusReq
		called   bool
	}
	flowListRunsArgs struct {
		sourceID uint32
		targetID uint32
		req      protoflow.ListRunsReq
		called   bool
	}
	flowListArgs struct {
		sourceID uint32
		targetID uint32
		req      protoflow.ListReq
		called   bool
	}
	flowGetArgs struct {
		sourceID uint32
		targetID uint32
		req      protoflow.GetReq
		called   bool
	}
	varSetArgs struct {
		sourceID uint32
		targetID uint32
		req      protovarstore.SetReq
		called   bool
	}

	configValues  map[string]string
	configGetErr  error
	configListErr error
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
func (f *fakeBackend) GetPerms(_ context.Context, sourceID, targetID, nodeID uint32) (protoauth.RespData, error) {
	f.getPermsArgs = struct {
		sourceID uint32
		targetID uint32
		nodeID   uint32
		called   bool
	}{sourceID: sourceID, targetID: targetID, nodeID: nodeID, called: true}
	return protoauth.RespData{Code: 1, NodeID: nodeID, HubID: targetID, Role: "admin", Perms: []string{"var.read", "auth.list_roles"}}, nil
}
func (f *fakeBackend) ListRoles(_ context.Context, sourceID, targetID uint32, req protoauth.ListRolesReq) (authsvc.ListRolesResp, error) {
	f.listRolesArgs = struct {
		sourceID uint32
		targetID uint32
		req      protoauth.ListRolesReq
		called   bool
	}{sourceID: sourceID, targetID: targetID, req: req, called: true}
	return authsvc.ListRolesResp{
		Code:  1,
		Total: len(req.NodeIDs),
		Roles: []protoauth.RolePermEntry{{NodeID: 7, Role: "admin", Perms: []string{"var.read"}}},
	}, nil
}
func (f *fakeBackend) ListPendingRegisters(_ context.Context, sourceID, targetID uint32, req authsvc.ListPendingRegistersReq) (authsvc.ListPendingRegistersResp, error) {
	f.listPendingArgs = struct {
		sourceID uint32
		targetID uint32
		req      authsvc.ListPendingRegistersReq
		called   bool
	}{sourceID: sourceID, targetID: targetID, req: req, called: true}
	return authsvc.ListPendingRegistersResp{
		Code:  1,
		Total: 1,
		Items: []authsvc.PendingRegisterInfo{{RequestID: "req-1", DeviceID: "dev-1"}},
	}, nil
}
func (f *fakeBackend) ApproveRegister(_ context.Context, sourceID, targetID uint32, req authsvc.ApproveRegisterReq) (authsvc.ApproveRegisterResp, error) {
	f.approveArgs = struct {
		sourceID uint32
		targetID uint32
		req      authsvc.ApproveRegisterReq
		called   bool
	}{sourceID: sourceID, targetID: targetID, req: req, called: true}
	return authsvc.ApproveRegisterResp{Code: 1, RequestID: req.RequestID, DeviceID: "dev-1", NodeID: 88, Role: req.Role, Status: "approved"}, nil
}
func (f *fakeBackend) RejectRegister(_ context.Context, sourceID, targetID uint32, req authsvc.RejectRegisterReq) (authsvc.RejectRegisterResp, error) {
	f.rejectArgs = struct {
		sourceID uint32
		targetID uint32
		req      authsvc.RejectRegisterReq
		called   bool
	}{sourceID: sourceID, targetID: targetID, req: req, called: true}
	return authsvc.RejectRegisterResp{Code: 1, RequestID: req.RequestID, DeviceID: "dev-1", Status: "rejected", Reason: req.Reason}, nil
}
func (f *fakeBackend) IssueRegisterPermit(_ context.Context, sourceID, targetID uint32, req authsvc.IssueRegisterPermitReq) (authsvc.IssueRegisterPermitResp, error) {
	f.issuePermitArgs = struct {
		sourceID uint32
		targetID uint32
		req      authsvc.IssueRegisterPermitReq
		called   bool
	}{sourceID: sourceID, targetID: targetID, req: req, called: true}
	return authsvc.IssueRegisterPermitResp{Code: 1, Permit: "permit-1", DeviceID: req.DeviceID, Role: req.Role, ExpiresAt: req.ExpiresAt}, nil
}
func (f *fakeBackend) RevokeRegisterPermit(_ context.Context, sourceID, targetID uint32, req authsvc.RevokeRegisterPermitReq) (authsvc.RevokeRegisterPermitResp, error) {
	f.revokePermitArgs = struct {
		sourceID uint32
		targetID uint32
		req      authsvc.RevokeRegisterPermitReq
		called   bool
	}{sourceID: sourceID, targetID: targetID, req: req, called: true}
	return authsvc.RevokeRegisterPermitResp{Code: 1, Permit: req.Permit, DeviceID: "dev-1", Role: "admin"}, nil
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
func (f *fakeBackend) NodeEcho(_ context.Context, sourceID, targetID uint32, message string) (protomanagement.NodeEchoResp, error) {
	f.nodeEchoArgs = struct {
		sourceID uint32
		targetID uint32
		message  string
		called   bool
	}{sourceID: sourceID, targetID: targetID, message: message, called: true}
	return protomanagement.NodeEchoResp{Code: 1, Echo: message}, nil
}
func (f *fakeBackend) ConfigGet(_ context.Context, sourceID, targetID uint32, key string) (protomanagement.ConfigResp, error) {
	f.configGetArgs = struct {
		sourceID uint32
		targetID uint32
		key      string
		called   bool
	}{sourceID: sourceID, targetID: targetID, key: key, called: true}
	if f.configGetErr != nil {
		return protomanagement.ConfigResp{}, f.configGetErr
	}
	if value, ok := f.configValues[key]; ok {
		return protomanagement.ConfigResp{Code: 1, Key: key, Value: value}, nil
	}
	return protomanagement.ConfigResp{}, errors.New("not found (code=404)")
}
func (f *fakeBackend) ConfigList(_ context.Context, sourceID, targetID uint32) (protomanagement.ConfigListResp, error) {
	f.configListArgs = struct {
		sourceID uint32
		targetID uint32
		called   bool
	}{sourceID: sourceID, targetID: targetID, called: true}
	if f.configListErr != nil {
		return protomanagement.ConfigListResp{}, f.configListErr
	}
	keys := make([]string, 0, len(f.configValues))
	for key := range f.configValues {
		keys = append(keys, key)
	}
	return protomanagement.ConfigListResp{Code: 1, Keys: keys}, nil
}
func (f *fakeBackend) ListSubtree(_ context.Context, sourceID, targetID uint32) (protomanagement.ListSubtreeResp, error) {
	f.listSubtreeArgs = struct {
		sourceID uint32
		targetID uint32
		called   bool
	}{sourceID: sourceID, targetID: targetID, called: true}
	return protomanagement.ListSubtreeResp{Code: 1, Nodes: []protomanagement.NodeInfo{{NodeID: targetID}}}, nil
}
func (f *fakeBackend) ExecCapQuery(_ context.Context, sourceID, targetID uint32, req protoexec.CapQueryReq) (protoexec.CapQueryResp, error) {
	f.execCapQueryArgs = struct {
		sourceID uint32
		targetID uint32
		req      protoexec.CapQueryReq
		called   bool
	}{sourceID: sourceID, targetID: targetID, req: req, called: true}
	return protoexec.CapQueryResp{Code: 1, ReqID: req.ReqID, ResponderNode: targetID, Total: 1, Routes: []protoexec.CapabilityRoute{{Method: req.Method, ProviderNode: req.ProviderNode}}}, nil
}
func (f *fakeBackend) FlowSet(_ context.Context, sourceID, targetID uint32, req protoflow.SetReq) (protoflow.SetResp, error) {
	f.flowSetArgs = struct {
		sourceID uint32
		targetID uint32
		req      protoflow.SetReq
		called   bool
	}{sourceID: sourceID, targetID: targetID, req: req, called: true}
	return protoflow.SetResp{Code: 1, ReqID: req.ReqID, FlowID: req.FlowID}, nil
}
func (f *fakeBackend) FlowDelete(_ context.Context, sourceID, targetID uint32, req flowsvc.DeleteReq) (flowsvc.DeleteResp, error) {
	f.flowDeleteArgs = struct {
		sourceID uint32
		targetID uint32
		req      flowsvc.DeleteReq
		called   bool
	}{sourceID: sourceID, targetID: targetID, req: req, called: true}
	return flowsvc.DeleteResp{Code: 1, ReqID: req.ReqID, FlowID: req.FlowID}, nil
}
func (f *fakeBackend) FlowRun(_ context.Context, sourceID, targetID uint32, req protoflow.RunReq) (protoflow.RunResp, error) {
	f.flowRunArgs = struct {
		sourceID uint32
		targetID uint32
		req      protoflow.RunReq
		called   bool
	}{sourceID: sourceID, targetID: targetID, req: req, called: true}
	return protoflow.RunResp{Code: 1, ReqID: req.ReqID, FlowID: req.FlowID, RunID: "run-1"}, nil
}
func (f *fakeBackend) FlowCancelRun(_ context.Context, sourceID, targetID uint32, req protoflow.CancelRunReq) (protoflow.CancelRunResp, error) {
	f.flowCancelRunArgs = struct {
		sourceID uint32
		targetID uint32
		req      protoflow.CancelRunReq
		called   bool
	}{sourceID: sourceID, targetID: targetID, req: req, called: true}
	return protoflow.CancelRunResp{Code: 1, ReqID: req.ReqID, FlowID: req.FlowID, RunID: req.RunID, Status: "cancelled"}, nil
}
func (f *fakeBackend) FlowStatus(_ context.Context, sourceID, targetID uint32, req protoflow.StatusReq) (protoflow.StatusResp, error) {
	f.flowStatusArgs = struct {
		sourceID uint32
		targetID uint32
		req      protoflow.StatusReq
		called   bool
	}{sourceID: sourceID, targetID: targetID, req: req, called: true}
	return protoflow.StatusResp{Code: 1, ReqID: req.ReqID, FlowID: req.FlowID, RunID: req.RunID, Status: "succeeded"}, nil
}
func (f *fakeBackend) FlowListRuns(_ context.Context, sourceID, targetID uint32, req protoflow.ListRunsReq) (protoflow.ListRunsResp, error) {
	f.flowListRunsArgs = struct {
		sourceID uint32
		targetID uint32
		req      protoflow.ListRunsReq
		called   bool
	}{sourceID: sourceID, targetID: targetID, req: req, called: true}
	return protoflow.ListRunsResp{
		Code:         1,
		ReqID:        req.ReqID,
		FlowID:       req.FlowID,
		ExecutorNode: req.ExecutorNode,
		Runs: []protoflow.RunSummary{
			{RunID: "run-1", Status: "running"},
		},
	}, nil
}
func (f *fakeBackend) FlowList(_ context.Context, sourceID, targetID uint32, req protoflow.ListReq) (protoflow.ListResp, error) {
	f.flowListArgs = struct {
		sourceID uint32
		targetID uint32
		req      protoflow.ListReq
		called   bool
	}{sourceID: sourceID, targetID: targetID, req: req, called: true}
	return protoflow.ListResp{Code: 1, ReqID: req.ReqID, ExecutorNode: req.ExecutorNode, Flows: []protoflow.FlowSummary{{FlowID: "flow-1"}}}, nil
}
func (f *fakeBackend) FlowGet(_ context.Context, sourceID, targetID uint32, req protoflow.GetReq) (protoflow.GetResp, error) {
	f.flowGetArgs = struct {
		sourceID uint32
		targetID uint32
		req      protoflow.GetReq
		called   bool
	}{sourceID: sourceID, targetID: targetID, req: req, called: true}
	return protoflow.GetResp{Code: 1, ReqID: req.ReqID, ExecutorNode: req.ExecutorNode, FlowID: req.FlowID, Name: "demo"}, nil
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

func TestFlowToolsRegistered(t *testing.T) {
	tools := NewTools(&fakeBackend{})
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, name := range []string{
		"myflowhub_flow_list",
		"myflowhub_flow_get",
		"myflowhub_flow_set",
		"myflowhub_flow_run",
		"myflowhub_flow_cancel_run",
		"myflowhub_flow_status",
		"myflowhub_flow_list_runs",
		"myflowhub_flow_delete",
	} {
		if !names[name] {
			t.Fatalf("expected tool %q to be registered", name)
		}
	}
}

func TestExecAndManagementReadToolsRegistered(t *testing.T) {
	tools := NewTools(&fakeBackend{})
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, name := range []string{
		"myflowhub_exec_cap_query",
		"myflowhub_management_node_echo",
		"myflowhub_management_list_subtree",
	} {
		if !names[name] {
			t.Fatalf("expected tool %q to be registered", name)
		}
	}
}

func TestExecCapQueryFallsBackToSnapshotAndGeneratesReqID(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
	}

	result := findTool(t, NewTools(backend), "myflowhub_exec_cap_query").Handler(context.Background(), json.RawMessage(`{}`))
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.execCapQueryArgs.called {
		t.Fatal("expected ExecCapQuery() called")
	}
	if backend.execCapQueryArgs.sourceID != 7 || backend.execCapQueryArgs.targetID != 9 {
		t.Fatalf("unexpected exec route: %+v", backend.execCapQueryArgs)
	}
	if backend.execCapQueryArgs.req.RequesterNode != 7 {
		t.Fatalf("expected requester_node fallback to source_id, got %+v", backend.execCapQueryArgs.req)
	}
	if strings.TrimSpace(backend.execCapQueryArgs.req.ReqID) == "" {
		t.Fatalf("expected generated req_id, got %+v", backend.execCapQueryArgs.req)
	}
}

func TestExecCapQueryPassesExplicitFilters(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
	}

	raw := json.RawMessage(`{
		"req_id":"req-1",
		"source_id":5,
		"target_id":55,
		"requester_node":12,
		"method":"demo::run",
		"prefix":true,
		"provider_node":88,
		"limit":2,
		"include_schema":true
	}`)
	result := findTool(t, NewTools(backend), "myflowhub_exec_cap_query").Handler(context.Background(), raw)
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.execCapQueryArgs.called {
		t.Fatal("expected ExecCapQuery() called")
	}
	if backend.execCapQueryArgs.sourceID != 5 || backend.execCapQueryArgs.targetID != 55 {
		t.Fatalf("unexpected exec route: %+v", backend.execCapQueryArgs)
	}
	req := backend.execCapQueryArgs.req
	if req.ReqID != "req-1" || req.RequesterNode != 12 || req.Method != "demo::run" {
		t.Fatalf("unexpected exec request identity fields: %+v", req)
	}
	if !req.Prefix || req.ProviderNode != 88 || req.Limit != 2 || !req.IncludeSchema {
		t.Fatalf("unexpected exec request filters: %+v", req)
	}
}

func TestFlowListFallsBackToSnapshotAndGeneratesReqID(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
	}

	result := findTool(t, NewTools(backend), "myflowhub_flow_list").Handler(context.Background(), json.RawMessage(`{}`))
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.flowListArgs.called {
		t.Fatal("expected FlowList() called")
	}
	if backend.flowListArgs.sourceID != 7 || backend.flowListArgs.targetID != 9 {
		t.Fatalf("unexpected flow list route: %+v", backend.flowListArgs)
	}
	if backend.flowListArgs.req.OriginNode != 7 || backend.flowListArgs.req.ExecutorNode != 9 {
		t.Fatalf("unexpected flow list request: %+v", backend.flowListArgs.req)
	}
	if strings.TrimSpace(backend.flowListArgs.req.ReqID) == "" {
		t.Fatalf("expected generated req_id, got %+v", backend.flowListArgs.req)
	}
}

func TestFlowSetUsesExplicitExecutorAndTransportTarget(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		allowWrite:       true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
	}

	raw := json.RawMessage(`{
		"target_id":55,
		"executor_node":88,
		"flow_id":"flow-1",
		"name":"demo",
		"trigger":{"type":"interval","every_ms":1000},
		"graph":{"nodes":[{"id":"n1","kind":"call","spec":{"method":"demo::run"}}],"edges":[]}
	}`)
	result := findTool(t, NewTools(backend), "myflowhub_flow_set").Handler(context.Background(), raw)
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.flowSetArgs.called {
		t.Fatal("expected FlowSet() called")
	}
	if backend.flowSetArgs.sourceID != 7 || backend.flowSetArgs.targetID != 55 {
		t.Fatalf("unexpected flow set route: %+v", backend.flowSetArgs)
	}
	if backend.flowSetArgs.req.OriginNode != 7 || backend.flowSetArgs.req.ExecutorNode != 88 {
		t.Fatalf("unexpected flow set request route fields: %+v", backend.flowSetArgs.req)
	}
	if backend.flowSetArgs.req.FlowID != "flow-1" || backend.flowSetArgs.req.Name != "demo" {
		t.Fatalf("unexpected flow set request payload: %+v", backend.flowSetArgs.req)
	}
}

func TestFlowSetBlockedWhenWriteDisabled(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		allowWrite:       false,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
	}

	raw := json.RawMessage(`{
		"flow_id":"flow-1",
		"trigger":{"type":"interval","every_ms":1000},
		"graph":{"nodes":[{"id":"n1","kind":"call","spec":{"method":"demo::run"}}],"edges":[]}
	}`)
	result := findTool(t, NewTools(backend), "myflowhub_flow_set").Handler(context.Background(), raw)
	if !result.IsError {
		t.Fatalf("expected error result, got %#v", result)
	}
	if backend.flowSetArgs.called {
		t.Fatal("expected FlowSet() not called when allow_write=false")
	}
	payload, ok := result.StructuredContent.(toolErrorPayload)
	if !ok {
		t.Fatalf("expected toolErrorPayload, got %#v", result.StructuredContent)
	}
	if payload.Code != "write_disabled" {
		t.Fatalf("unexpected error code: %#v", payload)
	}
	if !strings.Contains(payload.Hint, "myflowhub_flow_set") {
		t.Fatalf("expected flow write hint, got %#v", payload)
	}
}

func TestFlowGetRequiresFlowID(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
	}

	result := findTool(t, NewTools(backend), "myflowhub_flow_get").Handler(context.Background(), json.RawMessage(`{}`))
	if !result.IsError {
		t.Fatalf("expected error, got %#v", result)
	}
	payload, ok := result.StructuredContent.(toolErrorPayload)
	if !ok {
		t.Fatalf("expected toolErrorPayload, got %#v", result.StructuredContent)
	}
	if payload.Code != "invalid_arguments" {
		t.Fatalf("unexpected error payload: %#v", payload)
	}
}

func TestFlowStatusUsesExplicitExecutorNode(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
	}

	result := findTool(t, NewTools(backend), "myflowhub_flow_status").Handler(context.Background(), json.RawMessage(`{"flow_id":"flow-1","executor_node":88,"run_id":"run-9"}`))
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.flowStatusArgs.called {
		t.Fatal("expected FlowStatus() called")
	}
	if backend.flowStatusArgs.sourceID != 7 || backend.flowStatusArgs.targetID != 9 {
		t.Fatalf("unexpected flow status transport route: %+v", backend.flowStatusArgs)
	}
	if backend.flowStatusArgs.req.ExecutorNode != 88 || backend.flowStatusArgs.req.RunID != "run-9" {
		t.Fatalf("unexpected flow status request: %+v", backend.flowStatusArgs.req)
	}
}

func TestFlowCancelRunUsesExplicitExecutorNode(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
		allowWrite:       true,
	}

	result := findTool(t, NewTools(backend), "myflowhub_flow_cancel_run").Handler(context.Background(), json.RawMessage(`{"flow_id":"flow-1","run_id":"run-9","executor_node":88}`))
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.flowCancelRunArgs.called {
		t.Fatal("expected FlowCancelRun() called")
	}
	if backend.flowCancelRunArgs.sourceID != 7 || backend.flowCancelRunArgs.targetID != 9 {
		t.Fatalf("unexpected flow cancel transport route: %+v", backend.flowCancelRunArgs)
	}
	if backend.flowCancelRunArgs.req.ExecutorNode != 88 || backend.flowCancelRunArgs.req.RunID != "run-9" {
		t.Fatalf("unexpected flow cancel request: %+v", backend.flowCancelRunArgs.req)
	}
}

func TestFlowListRunsUsesExplicitExecutorNode(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
	}

	result := findTool(t, NewTools(backend), "myflowhub_flow_list_runs").Handler(context.Background(), json.RawMessage(`{"flow_id":"flow-1","executor_node":88,"limit":10}`))
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.flowListRunsArgs.called {
		t.Fatal("expected FlowListRuns() called")
	}
	if backend.flowListRunsArgs.sourceID != 7 || backend.flowListRunsArgs.targetID != 9 {
		t.Fatalf("unexpected flow list_runs transport route: %+v", backend.flowListRunsArgs)
	}
	if backend.flowListRunsArgs.req.ExecutorNode != 88 || backend.flowListRunsArgs.req.Limit != 10 {
		t.Fatalf("unexpected flow list_runs request: %+v", backend.flowListRunsArgs.req)
	}
}

func TestFlowCancelRunBlockedWhenWriteDisabled(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		allowWrite:       false,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
	}

	result := findTool(t, NewTools(backend), "myflowhub_flow_cancel_run").Handler(context.Background(), json.RawMessage(`{"flow_id":"flow-1","run_id":"run-1"}`))
	if !result.IsError {
		t.Fatalf("expected error result, got %#v", result)
	}
	if backend.flowCancelRunArgs.called {
		t.Fatal("expected FlowCancelRun() not called when allow_write=false")
	}
	payload, ok := result.StructuredContent.(toolErrorPayload)
	if !ok {
		t.Fatalf("expected toolErrorPayload, got %#v", result.StructuredContent)
	}
	if payload.Code != "write_disabled" {
		t.Fatalf("unexpected error code: %#v", payload)
	}
	if !strings.Contains(payload.Hint, "myflowhub_flow_cancel_run") {
		t.Fatalf("expected cancel_run hint, got %#v", payload)
	}
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

func TestManagementNodeEchoRequiresNonEmptyMessage(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
	}

	result := findTool(t, NewTools(backend), "myflowhub_management_node_echo").Handler(context.Background(), json.RawMessage(`{"message":"   "}`))
	if !result.IsError {
		t.Fatalf("expected error, got %#v", result)
	}
	if backend.nodeEchoArgs.called {
		t.Fatal("expected NodeEcho() not called")
	}
	payload, ok := result.StructuredContent.(toolErrorPayload)
	if !ok {
		t.Fatalf("expected toolErrorPayload, got %#v", result.StructuredContent)
	}
	if payload.Code != "invalid_arguments" {
		t.Fatalf("unexpected error payload: %#v", payload)
	}
}

func TestManagementListSubtreeFallsBackToStartupDefaults(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		defaults: mcpapp.Defaults{
			NodeID:        11,
			DefaultTarget: 99,
		},
	}

	result := findTool(t, NewTools(backend), "myflowhub_management_list_subtree").Handler(context.Background(), json.RawMessage(`{}`))
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.listSubtreeArgs.called {
		t.Fatal("expected ListSubtree() called")
	}
	if backend.listSubtreeArgs.sourceID != 11 || backend.listSubtreeArgs.targetID != 99 {
		t.Fatalf("unexpected subtree route: %+v", backend.listSubtreeArgs)
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

func TestAuthGetPermsFallsBackToSnapshotNode(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
	}

	result := findTool(t, NewTools(backend), "myflowhub_auth_get_perms").Handler(context.Background(), json.RawMessage(`{}`))
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.getPermsArgs.called {
		t.Fatal("expected GetPerms() called")
	}
	if backend.getPermsArgs.sourceID != 7 || backend.getPermsArgs.targetID != 9 || backend.getPermsArgs.nodeID != 7 {
		t.Fatalf("unexpected auth get_perms args: %+v", backend.getPermsArgs)
	}
}

func TestAuthGetPermsUsesResolvedAuthorityNode(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
		configValues:     map[string]string{authorityNodeIDConfigKey: "88"},
	}

	result := findTool(t, NewTools(backend), "myflowhub_auth_get_perms").Handler(context.Background(), json.RawMessage(`{}`))
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.configGetArgs.called {
		t.Fatal("expected ConfigGet() called for authority resolution")
	}
	if backend.getPermsArgs.targetID != 88 {
		t.Fatalf("expected authority target 88, got %+v", backend.getPermsArgs)
	}
}

func TestAuthListRolesPassesFilters(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
	}

	result := findTool(t, NewTools(backend), "myflowhub_auth_list_roles").Handler(context.Background(), json.RawMessage(`{"offset":1,"limit":2,"role":"admin","node_ids":[7,9]}`))
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.listRolesArgs.called {
		t.Fatal("expected ListRoles() called")
	}
	if backend.listRolesArgs.sourceID != 7 || backend.listRolesArgs.targetID != 9 {
		t.Fatalf("unexpected auth list_roles route: %+v", backend.listRolesArgs)
	}
	if backend.listRolesArgs.req.Offset != 1 || backend.listRolesArgs.req.Limit != 2 || backend.listRolesArgs.req.Role != "admin" {
		t.Fatalf("unexpected auth list_roles req: %+v", backend.listRolesArgs.req)
	}
	if len(backend.listRolesArgs.req.NodeIDs) != 2 || backend.listRolesArgs.req.NodeIDs[0] != 7 || backend.listRolesArgs.req.NodeIDs[1] != 9 {
		t.Fatalf("unexpected auth list_roles node_ids: %+v", backend.listRolesArgs.req.NodeIDs)
	}
}

func TestAuthListPendingRegistersResolvesAuthorityNode(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
		configValues:     map[string]string{authorityNodeIDConfigKey: "55"},
	}

	result := findTool(t, NewTools(backend), "myflowhub_auth_list_pending_registers").Handler(context.Background(), json.RawMessage(`{"offset":1,"limit":2,"device_id":"dev-1"}`))
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.listPendingArgs.called {
		t.Fatal("expected ListPendingRegisters() called")
	}
	if backend.listPendingArgs.targetID != 55 {
		t.Fatalf("unexpected authority route: %+v", backend.listPendingArgs)
	}
	if backend.listPendingArgs.req.Offset != 1 || backend.listPendingArgs.req.Limit != 2 || backend.listPendingArgs.req.DeviceID != "dev-1" {
		t.Fatalf("unexpected list pending req: %+v", backend.listPendingArgs.req)
	}
}

func TestAuthApproveRegisterUsesExplicitAuthorityID(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
	}

	result := findTool(t, NewTools(backend), "myflowhub_auth_approve_register").Handler(context.Background(), json.RawMessage(`{"authority_id":77,"request_id":"req-1","role":"admin"}`))
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.approveArgs.called {
		t.Fatal("expected ApproveRegister() called")
	}
	if backend.approveArgs.targetID != 77 {
		t.Fatalf("unexpected approve route: %+v", backend.approveArgs)
	}
	if backend.configGetArgs.called {
		t.Fatal("did not expect ConfigGet() when authority_id is explicit")
	}
}

func TestAuthIssueRegisterPermitRejectsNegativeExpiry(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
	}

	result := findTool(t, NewTools(backend), "myflowhub_auth_issue_register_permit").Handler(context.Background(), json.RawMessage(`{"device_id":"dev-1","role":"admin","expires_at":-1}`))
	if !result.IsError {
		t.Fatalf("expected error, got %#v", result)
	}
	payload, ok := result.StructuredContent.(toolErrorPayload)
	if !ok {
		t.Fatalf("expected toolErrorPayload, got %#v", result.StructuredContent)
	}
	if payload.Code != "invalid_arguments" {
		t.Fatalf("unexpected error payload: %#v", payload)
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

func TestAuthListRolesRejectsZeroNodeIDFilter(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
	}

	result := findTool(t, NewTools(backend), "myflowhub_auth_list_roles").Handler(context.Background(), json.RawMessage(`{"node_ids":[0]}`))
	if !result.IsError {
		t.Fatalf("expected error, got %#v", result)
	}
	payload, ok := result.StructuredContent.(toolErrorPayload)
	if !ok {
		t.Fatalf("expected toolErrorPayload, got %#v", result.StructuredContent)
	}
	if payload.Code != "invalid_arguments" {
		t.Fatalf("unexpected error payload: %#v", payload)
	}
}

func TestManagementConfigGetUsesRouteAndKey(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
		configValues:     map[string]string{"authority.node_id": "55"},
	}

	result := findTool(t, NewTools(backend), "myflowhub_management_config_get").Handler(context.Background(), json.RawMessage(`{"key":"authority.node_id"}`))
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.configGetArgs.called {
		t.Fatal("expected ConfigGet() called")
	}
	if backend.configGetArgs.sourceID != 7 || backend.configGetArgs.targetID != 9 || backend.configGetArgs.key != "authority.node_id" {
		t.Fatalf("unexpected config get args: %+v", backend.configGetArgs)
	}
}

func TestManagementConfigListFallsBackToAuthSnapshot(t *testing.T) {
	backend := &fakeBackend{
		sessionConnected: true,
		auth:             mcpapp.AuthSnapshot{NodeID: 7, HubID: 9, LoggedIn: true},
		configValues:     map[string]string{"authority.node_id": "55"},
	}

	result := findTool(t, NewTools(backend), "myflowhub_management_config_list").Handler(context.Background(), json.RawMessage(`{}`))
	if result.IsError {
		t.Fatalf("expected success, got %#v", result)
	}
	if !backend.configListArgs.called {
		t.Fatal("expected ConfigList() called")
	}
	if backend.configListArgs.sourceID != 7 || backend.configListArgs.targetID != 9 {
		t.Fatalf("unexpected config list args: %+v", backend.configListArgs)
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
