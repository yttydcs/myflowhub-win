package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	protoauth "github.com/yttydcs/myflowhub-proto/protocol/auth"
	protoexec "github.com/yttydcs/myflowhub-proto/protocol/exec"
	protoflow "github.com/yttydcs/myflowhub-proto/protocol/flow"
	protomanagement "github.com/yttydcs/myflowhub-proto/protocol/management"
	protovarstore "github.com/yttydcs/myflowhub-proto/protocol/varstore"
	"github.com/yttydcs/myflowhub-win/internal/mcpapp"
	authsvc "github.com/yttydcs/myflowhub-win/internal/services/auth"
	flowsvc "github.com/yttydcs/myflowhub-win/internal/services/flow"
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
	GetPerms(ctx context.Context, sourceID, targetID, nodeID uint32) (protoauth.RespData, error)
	ListRoles(ctx context.Context, sourceID, targetID uint32, req protoauth.ListRolesReq) (authsvc.ListRolesResp, error)
	ListPendingRegisters(ctx context.Context, sourceID, targetID uint32, req authsvc.ListPendingRegistersReq) (authsvc.ListPendingRegistersResp, error)
	ApproveRegister(ctx context.Context, sourceID, targetID uint32, req authsvc.ApproveRegisterReq) (authsvc.ApproveRegisterResp, error)
	RejectRegister(ctx context.Context, sourceID, targetID uint32, req authsvc.RejectRegisterReq) (authsvc.RejectRegisterResp, error)
	IssueRegisterPermit(ctx context.Context, sourceID, targetID uint32, req authsvc.IssueRegisterPermitReq) (authsvc.IssueRegisterPermitResp, error)
	RevokeRegisterPermit(ctx context.Context, sourceID, targetID uint32, req authsvc.RevokeRegisterPermitReq) (authsvc.RevokeRegisterPermitResp, error)
	CompleteAuth(resp protoauth.RespData, deviceID string) error
	ListNodes(ctx context.Context, sourceID, targetID uint32) (protomanagement.ListNodesResp, error)
	NodeInfo(ctx context.Context, sourceID, targetID uint32) (protomanagement.NodeInfoResp, error)
	NodeEcho(ctx context.Context, sourceID, targetID uint32, message string) (protomanagement.NodeEchoResp, error)
	ConfigGet(ctx context.Context, sourceID, targetID uint32, key string) (protomanagement.ConfigResp, error)
	ConfigList(ctx context.Context, sourceID, targetID uint32) (protomanagement.ConfigListResp, error)
	ListSubtree(ctx context.Context, sourceID, targetID uint32) (protomanagement.ListSubtreeResp, error)
	ExecCapQuery(ctx context.Context, sourceID, targetID uint32, req protoexec.CapQueryReq) (protoexec.CapQueryResp, error)
	FlowSet(ctx context.Context, sourceID, targetID uint32, req protoflow.SetReq) (protoflow.SetResp, error)
	FlowDelete(ctx context.Context, sourceID, targetID uint32, req flowsvc.DeleteReq) (flowsvc.DeleteResp, error)
	FlowRun(ctx context.Context, sourceID, targetID uint32, req protoflow.RunReq) (protoflow.RunResp, error)
	FlowCancelRun(ctx context.Context, sourceID, targetID uint32, req protoflow.CancelRunReq) (protoflow.CancelRunResp, error)
	FlowStatus(ctx context.Context, sourceID, targetID uint32, req protoflow.StatusReq) (protoflow.StatusResp, error)
	FlowListRuns(ctx context.Context, sourceID, targetID uint32, req protoflow.ListRunsReq) (protoflow.ListRunsResp, error)
	FlowList(ctx context.Context, sourceID, targetID uint32, req protoflow.ListReq) (protoflow.ListResp, error)
	FlowGet(ctx context.Context, sourceID, targetID uint32, req protoflow.GetReq) (protoflow.GetResp, error)
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

type authGetPermsArgs struct {
	AuthorityID *uint32 `json:"authority_id,omitempty"`
	NodeID      *uint32 `json:"node_id,omitempty"`
	SourceID    *uint32 `json:"source_id,omitempty"`
	TargetID    *uint32 `json:"target_id,omitempty"`
}

type authListRolesArgs struct {
	AuthorityID *uint32  `json:"authority_id,omitempty"`
	Offset      *int     `json:"offset,omitempty"`
	Limit       *int     `json:"limit,omitempty"`
	Role        string   `json:"role,omitempty"`
	NodeIDs     []uint32 `json:"node_ids,omitempty"`
	SourceID    *uint32  `json:"source_id,omitempty"`
	TargetID    *uint32  `json:"target_id,omitempty"`
}

type authListPendingRegistersArgs struct {
	AuthorityID *uint32 `json:"authority_id,omitempty"`
	Offset      *int    `json:"offset,omitempty"`
	Limit       *int    `json:"limit,omitempty"`
	DeviceID    string  `json:"device_id,omitempty"`
	SourceID    *uint32 `json:"source_id,omitempty"`
	TargetID    *uint32 `json:"target_id,omitempty"`
}

type authApproveRegisterArgs struct {
	AuthorityID *uint32 `json:"authority_id,omitempty"`
	RequestID   string  `json:"request_id"`
	Role        string  `json:"role,omitempty"`
	SourceID    *uint32 `json:"source_id,omitempty"`
	TargetID    *uint32 `json:"target_id,omitempty"`
}

type authRejectRegisterArgs struct {
	AuthorityID *uint32 `json:"authority_id,omitempty"`
	RequestID   string  `json:"request_id"`
	Reason      string  `json:"reason,omitempty"`
	SourceID    *uint32 `json:"source_id,omitempty"`
	TargetID    *uint32 `json:"target_id,omitempty"`
}

type authIssueRegisterPermitArgs struct {
	AuthorityID *uint32 `json:"authority_id,omitempty"`
	DeviceID    string  `json:"device_id"`
	Role        string  `json:"role"`
	ExpiresAt   *int64  `json:"expires_at,omitempty"`
	SourceID    *uint32 `json:"source_id,omitempty"`
	TargetID    *uint32 `json:"target_id,omitempty"`
}

type authRevokeRegisterPermitArgs struct {
	AuthorityID *uint32 `json:"authority_id,omitempty"`
	Permit      string  `json:"permit"`
	SourceID    *uint32 `json:"source_id,omitempty"`
	TargetID    *uint32 `json:"target_id,omitempty"`
}

type managementArgs struct {
	SourceID *uint32 `json:"source_id,omitempty"`
	TargetID *uint32 `json:"target_id,omitempty"`
}

type managementConfigGetArgs struct {
	Key      string  `json:"key"`
	SourceID *uint32 `json:"source_id,omitempty"`
	TargetID *uint32 `json:"target_id,omitempty"`
}

type managementNodeEchoArgs struct {
	Message  string  `json:"message"`
	SourceID *uint32 `json:"source_id,omitempty"`
	TargetID *uint32 `json:"target_id,omitempty"`
}

type execCapQueryArgs struct {
	ReqID         string  `json:"req_id,omitempty"`
	SourceID      *uint32 `json:"source_id,omitempty"`
	TargetID      *uint32 `json:"target_id,omitempty"`
	RequesterNode *uint32 `json:"requester_node,omitempty"`
	Method        string  `json:"method,omitempty"`
	Prefix        bool    `json:"prefix,omitempty"`
	ProviderNode  *uint32 `json:"provider_node,omitempty"`
	Limit         *int    `json:"limit,omitempty"`
	IncludeSchema bool    `json:"include_schema,omitempty"`
}

type flowListArgs struct {
	ReqID        string  `json:"req_id,omitempty"`
	SourceID     *uint32 `json:"source_id,omitempty"`
	TargetID     *uint32 `json:"target_id,omitempty"`
	ExecutorNode *uint32 `json:"executor_node,omitempty"`
}

type flowGetArgs struct {
	ReqID        string  `json:"req_id,omitempty"`
	FlowID       string  `json:"flow_id"`
	SourceID     *uint32 `json:"source_id,omitempty"`
	TargetID     *uint32 `json:"target_id,omitempty"`
	ExecutorNode *uint32 `json:"executor_node,omitempty"`
}

type flowSetArgs struct {
	ReqID        string            `json:"req_id,omitempty"`
	FlowID       string            `json:"flow_id"`
	Name         string            `json:"name,omitempty"`
	Trigger      protoflow.Trigger `json:"trigger"`
	Graph        protoflow.Graph   `json:"graph"`
	SourceID     *uint32           `json:"source_id,omitempty"`
	TargetID     *uint32           `json:"target_id,omitempty"`
	ExecutorNode *uint32           `json:"executor_node,omitempty"`
}

type flowRunArgs struct {
	ReqID        string  `json:"req_id,omitempty"`
	FlowID       string  `json:"flow_id"`
	SourceID     *uint32 `json:"source_id,omitempty"`
	TargetID     *uint32 `json:"target_id,omitempty"`
	ExecutorNode *uint32 `json:"executor_node,omitempty"`
}

type flowCancelRunArgs struct {
	ReqID        string  `json:"req_id,omitempty"`
	FlowID       string  `json:"flow_id"`
	RunID        string  `json:"run_id"`
	SourceID     *uint32 `json:"source_id,omitempty"`
	TargetID     *uint32 `json:"target_id,omitempty"`
	ExecutorNode *uint32 `json:"executor_node,omitempty"`
}

type flowStatusArgs struct {
	ReqID        string  `json:"req_id,omitempty"`
	FlowID       string  `json:"flow_id"`
	RunID        string  `json:"run_id,omitempty"`
	SourceID     *uint32 `json:"source_id,omitempty"`
	TargetID     *uint32 `json:"target_id,omitempty"`
	ExecutorNode *uint32 `json:"executor_node,omitempty"`
}

type flowListRunsArgs struct {
	ReqID        string  `json:"req_id,omitempty"`
	FlowID       string  `json:"flow_id"`
	Limit        *uint32 `json:"limit,omitempty"`
	SourceID     *uint32 `json:"source_id,omitempty"`
	TargetID     *uint32 `json:"target_id,omitempty"`
	ExecutorNode *uint32 `json:"executor_node,omitempty"`
}

type flowDeleteArgs struct {
	ReqID        string  `json:"req_id,omitempty"`
	FlowID       string  `json:"flow_id"`
	SourceID     *uint32 `json:"source_id,omitempty"`
	TargetID     *uint32 `json:"target_id,omitempty"`
	ExecutorNode *uint32 `json:"executor_node,omitempty"`
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

type authorityRoute struct {
	SourceID    uint32
	HubTargetID uint32
	AuthorityID uint32
	Resolution  string
}

type flowRoute struct {
	SourceID     uint32
	TargetID     uint32
	ExecutorNode uint32
}

type execQueryRoute struct {
	SourceID      uint32
	TargetID      uint32
	RequesterNode uint32
}

const authorityNodeIDConfigKey = "authority.node_id"

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
			Name:        "myflowhub_auth_get_perms",
			Description: "Query the effective role and permission set of a node.",
			InputSchema: objectSchema(map[string]any{
				"authority_id": positiveIntegerSchema("Optional authority node ID override."),
				"node_id":      positiveIntegerSchema("Node ID to inspect. Falls back to the current auth snapshot or startup defaults."),
				"source_id":    positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":    positiveIntegerSchema("Hub target used to resolve authority. Falls back to hub_id or default_target."),
			}),
			Handler: set.authGetPerms,
		},
		{
			Name:        "myflowhub_auth_list_roles",
			Description: "List role and permission assignments known to the authority.",
			InputSchema: objectSchema(map[string]any{
				"authority_id": positiveIntegerSchema("Optional authority node ID override."),
				"offset":       nonNegativeIntegerSchema("Optional result offset."),
				"limit":        nonNegativeIntegerSchema("Optional page size."),
				"role":         stringSchema("Optional role filter."),
				"node_ids":     integerArraySchema("Optional node ID filter list.", 1),
				"source_id":    positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":    positiveIntegerSchema("Hub target used to resolve authority. Falls back to hub_id or default_target."),
			}),
			Handler: set.authListRoles,
		},
		{
			Name:        "myflowhub_auth_list_pending_registers",
			Description: "List register requests waiting for authority approval.",
			InputSchema: objectSchema(map[string]any{
				"authority_id": positiveIntegerSchema("Optional authority node ID override."),
				"offset":       nonNegativeIntegerSchema("Optional result offset."),
				"limit":        nonNegativeIntegerSchema("Optional page size."),
				"device_id":    stringSchema("Optional device ID filter."),
				"source_id":    positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":    positiveIntegerSchema("Hub target used to resolve authority. Falls back to hub_id or default_target."),
			}),
			Handler: set.authListPendingRegisters,
		},
		{
			Name:        "myflowhub_auth_approve_register",
			Description: "Approve a pending register request and reserve its node identity.",
			InputSchema: objectSchema(map[string]any{
				"authority_id": positiveIntegerSchema("Optional authority node ID override."),
				"request_id":   stringSchema("Pending register request ID."),
				"role":         stringSchema("Optional role override."),
				"source_id":    positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":    positiveIntegerSchema("Hub target used to resolve authority. Falls back to hub_id or default_target."),
			}, "request_id"),
			Handler: set.authApproveRegister,
		},
		{
			Name:        "myflowhub_auth_reject_register",
			Description: "Reject a pending register request.",
			InputSchema: objectSchema(map[string]any{
				"authority_id": positiveIntegerSchema("Optional authority node ID override."),
				"request_id":   stringSchema("Pending register request ID."),
				"reason":       stringSchema("Optional rejection reason."),
				"source_id":    positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":    positiveIntegerSchema("Hub target used to resolve authority. Falls back to hub_id or default_target."),
			}, "request_id"),
			Handler: set.authRejectRegister,
		},
		{
			Name:        "myflowhub_auth_issue_register_permit",
			Description: "Issue a one-time register permit for a device and role.",
			InputSchema: objectSchema(map[string]any{
				"authority_id": positiveIntegerSchema("Optional authority node ID override."),
				"device_id":    stringSchema("Device ID bound to the permit."),
				"role":         stringSchema("Role granted by the permit."),
				"expires_at":   nonNegativeIntegerSchema("Optional Unix timestamp expiration."),
				"source_id":    positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":    positiveIntegerSchema("Hub target used to resolve authority. Falls back to hub_id or default_target."),
			}, "device_id", "role"),
			Handler: set.authIssueRegisterPermit,
		},
		{
			Name:        "myflowhub_auth_revoke_register_permit",
			Description: "Revoke a previously issued register permit.",
			InputSchema: objectSchema(map[string]any{
				"authority_id": positiveIntegerSchema("Optional authority node ID override."),
				"permit":       stringSchema("Permit token to revoke."),
				"source_id":    positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":    positiveIntegerSchema("Hub target used to resolve authority. Falls back to hub_id or default_target."),
			}, "permit"),
			Handler: set.authRevokeRegisterPermit,
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
			Name:        "myflowhub_management_config_get",
			Description: "Read a management config key from the current hub or node.",
			InputSchema: objectSchema(map[string]any{
				"key":       stringSchema("Management config key."),
				"source_id": positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id": positiveIntegerSchema("Target node ID. Falls back to hub_id or default_target."),
			}, "key"),
			Handler: set.managementConfigGet,
		},
		{
			Name:        "myflowhub_management_config_list",
			Description: "List management config keys from the current hub or node.",
			InputSchema: objectSchema(map[string]any{
				"source_id": positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id": positiveIntegerSchema("Target node ID. Falls back to hub_id or default_target."),
			}),
			Handler: set.managementConfigList,
		},
		{
			Name:        "myflowhub_management_node_echo",
			Description: "Round-trip a message through management routing to verify a node responds.",
			InputSchema: objectSchema(map[string]any{
				"message":   stringSchema("Non-empty message echoed back by the target node."),
				"source_id": positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id": positiveIntegerSchema("Target node ID. Falls back to hub_id or default_target."),
			}, "message"),
			Handler: set.managementNodeEcho,
		},
		{
			Name:        "myflowhub_management_list_subtree",
			Description: "List the subtree visible from the selected management target.",
			InputSchema: objectSchema(map[string]any{
				"source_id": positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id": positiveIntegerSchema("Target node ID. Falls back to hub_id or default_target."),
			}),
			Handler: set.managementListSubtree,
		},
		{
			Name:        "myflowhub_exec_cap_query",
			Description: "Query exec capability routes from a hub or routing node.",
			InputSchema: objectSchema(map[string]any{
				"req_id":         stringSchema("Optional request correlation ID. Generated automatically when omitted."),
				"source_id":      positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":      positiveIntegerSchema("Transport target used to route the exec query. Falls back to hub_id or default_target."),
				"requester_node": positiveIntegerSchema("Requester node written into the exec payload. Falls back to source_id when omitted."),
				"method":         stringSchema("Optional exact method filter."),
				"prefix":         booleanSchema("Treat method as a prefix filter when true."),
				"provider_node":  positiveIntegerSchema("Optional provider node filter."),
				"limit":          nonNegativeIntegerSchema("Optional result limit."),
				"include_schema": booleanSchema("Include input/output schema blobs in the response when true."),
			}),
			Handler: set.execCapQuery,
		},
		{
			Name:        "myflowhub_flow_list",
			Description: "List flow summaries from an executor node.",
			InputSchema: objectSchema(map[string]any{
				"req_id":        stringSchema("Optional request correlation ID. Generated automatically when omitted."),
				"source_id":     positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":     positiveIntegerSchema("Transport target used to route the flow request. Falls back to hub_id or default_target."),
				"executor_node": positiveIntegerSchema("Actual flow executor node. Falls back to target_id when omitted."),
			}),
			Handler: set.flowList,
		},
		{
			Name:        "myflowhub_flow_get",
			Description: "Get the full definition of a flow from an executor node.",
			InputSchema: objectSchema(map[string]any{
				"req_id":        stringSchema("Optional request correlation ID. Generated automatically when omitted."),
				"flow_id":       stringSchema("Flow ID."),
				"source_id":     positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":     positiveIntegerSchema("Transport target used to route the flow request. Falls back to hub_id or default_target."),
				"executor_node": positiveIntegerSchema("Actual flow executor node. Falls back to target_id when omitted."),
			}, "flow_id"),
			Handler: set.flowGet,
		},
		{
			Name:        "myflowhub_flow_set",
			Description: "Create or update a flow definition. This is disabled unless allow_write=true.",
			InputSchema: objectSchema(map[string]any{
				"req_id":        stringSchema("Optional request correlation ID. Generated automatically when omitted."),
				"flow_id":       stringSchema("Flow ID."),
				"name":          stringSchema("Optional flow display name."),
				"trigger":       openObjectSchema("Flow trigger object as defined by the flow protocol."),
				"graph":         openObjectSchema("Flow graph object as defined by the flow protocol."),
				"source_id":     positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":     positiveIntegerSchema("Transport target used to route the flow request. Falls back to hub_id or default_target."),
				"executor_node": positiveIntegerSchema("Actual flow executor node. Falls back to target_id when omitted."),
			}, "flow_id", "trigger", "graph"),
			Handler: set.flowSet,
		},
		{
			Name:        "myflowhub_flow_run",
			Description: "Trigger one immediate run for a flow. This is disabled unless allow_write=true.",
			InputSchema: objectSchema(map[string]any{
				"req_id":        stringSchema("Optional request correlation ID. Generated automatically when omitted."),
				"flow_id":       stringSchema("Flow ID."),
				"source_id":     positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":     positiveIntegerSchema("Transport target used to route the flow request. Falls back to hub_id or default_target."),
				"executor_node": positiveIntegerSchema("Actual flow executor node. Falls back to target_id when omitted."),
			}, "flow_id"),
			Handler: set.flowRun,
		},
		{
			Name:        "myflowhub_flow_cancel_run",
			Description: "Cancel one active flow run from an executor node. This is disabled unless allow_write=true.",
			InputSchema: objectSchema(map[string]any{
				"req_id":        stringSchema("Optional request correlation ID. Generated automatically when omitted."),
				"flow_id":       stringSchema("Flow ID."),
				"run_id":        stringSchema("Run ID."),
				"source_id":     positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":     positiveIntegerSchema("Transport target used to route the flow request. Falls back to hub_id or default_target."),
				"executor_node": positiveIntegerSchema("Actual flow executor node. Falls back to target_id when omitted."),
			}, "flow_id", "run_id"),
			Handler: set.flowCancelRun,
		},
		{
			Name:        "myflowhub_flow_status",
			Description: "Read flow run status from an executor node.",
			InputSchema: objectSchema(map[string]any{
				"req_id":        stringSchema("Optional request correlation ID. Generated automatically when omitted."),
				"flow_id":       stringSchema("Flow ID."),
				"run_id":        stringSchema("Optional run ID. Reads the latest run when omitted."),
				"source_id":     positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":     positiveIntegerSchema("Transport target used to route the flow request. Falls back to hub_id or default_target."),
				"executor_node": positiveIntegerSchema("Actual flow executor node. Falls back to target_id when omitted."),
			}, "flow_id"),
			Handler: set.flowStatus,
		},
		{
			Name:        "myflowhub_flow_list_runs",
			Description: "List retained run summaries for a flow from an executor node.",
			InputSchema: objectSchema(map[string]any{
				"req_id":        stringSchema("Optional request correlation ID. Generated automatically when omitted."),
				"flow_id":       stringSchema("Flow ID."),
				"limit":         nonNegativeIntegerSchema("Optional run summary limit. Zero or omitted returns all retained runs."),
				"source_id":     positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":     positiveIntegerSchema("Transport target used to route the flow request. Falls back to hub_id or default_target."),
				"executor_node": positiveIntegerSchema("Actual flow executor node. Falls back to target_id when omitted."),
			}, "flow_id"),
			Handler: set.flowListRuns,
		},
		{
			Name:        "myflowhub_flow_delete",
			Description: "Delete a flow definition. This is disabled unless allow_write=true.",
			InputSchema: objectSchema(map[string]any{
				"req_id":        stringSchema("Optional request correlation ID. Generated automatically when omitted."),
				"flow_id":       stringSchema("Flow ID."),
				"source_id":     positiveIntegerSchema("Source node ID. Falls back to the current auth snapshot."),
				"target_id":     positiveIntegerSchema("Transport target used to route the flow request. Falls back to hub_id or default_target."),
				"executor_node": positiveIntegerSchema("Actual flow executor node. Falls back to target_id when omitted."),
			}, "flow_id"),
			Handler: set.flowDelete,
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

func (s toolSet) authGetPerms(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[authGetPermsArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with optional authority_id, node_id, source_id, and target_id.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_auth_get_perms")
	}
	route, err := s.resolveAuthorityRoute(ctx, args.SourceID, args.TargetID, args.AuthorityID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id plus authority_id/target_id explicitly.")
	}
	nodeID, err := s.resolveQueryNodeID(args.NodeID)
	if err != nil {
		return identityErrorResult(err, "Pass node_id explicitly or login first so the MCP client has a default identity.")
	}
	resp, err := s.backend.GetPerms(ctx, route.SourceID, route.AuthorityID, nodeID)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can query permissions.", map[string]any{"source_id": route.SourceID, "target_id": route.AuthorityID, "hub_target_id": route.HubTargetID, "node_id": nodeID})
	}
	return successResult(map[string]any{
		"source_id":         route.SourceID,
		"target_id":         route.AuthorityID,
		"hub_target_id":     route.HubTargetID,
		"target_resolution": route.Resolution,
		"node_id":           nodeID,
		"response":          resp,
	})
}

func (s toolSet) authListRoles(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[authListRolesArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with optional authority_id, offset, limit, role, node_ids, source_id, and target_id.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_auth_list_roles")
	}
	route, err := s.resolveAuthorityRoute(ctx, args.SourceID, args.TargetID, args.AuthorityID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id plus authority_id/target_id explicitly.")
	}
	req, err := normalizeListRolesArgs(args)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Use non-negative offset/limit values and positive node_ids.", nil)
	}
	resp, err := s.backend.ListRoles(ctx, route.SourceID, route.AuthorityID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can list authority roles.", map[string]any{"source_id": route.SourceID, "target_id": route.AuthorityID, "hub_target_id": route.HubTargetID, "request": req})
	}
	return successResult(map[string]any{
		"source_id":         route.SourceID,
		"target_id":         route.AuthorityID,
		"hub_target_id":     route.HubTargetID,
		"target_resolution": route.Resolution,
		"request":           req,
		"response":          resp,
	})
}

func (s toolSet) authListPendingRegisters(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[authListPendingRegistersArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with optional authority_id, offset, limit, device_id, source_id, and target_id.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_auth_list_pending_registers")
	}
	route, err := s.resolveAuthorityRoute(ctx, args.SourceID, args.TargetID, args.AuthorityID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id plus authority_id/target_id explicitly.")
	}
	req, err := normalizeListPendingRegistersArgs(args)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Use non-negative offset/limit values.", nil)
	}
	resp, err := s.backend.ListPendingRegisters(ctx, route.SourceID, route.AuthorityID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can list pending register requests.", map[string]any{"source_id": route.SourceID, "target_id": route.AuthorityID, "hub_target_id": route.HubTargetID, "request": req})
	}
	return successResult(map[string]any{
		"source_id":         route.SourceID,
		"target_id":         route.AuthorityID,
		"hub_target_id":     route.HubTargetID,
		"target_resolution": route.Resolution,
		"request":           req,
		"response":          resp,
	})
}

func (s toolSet) authApproveRegister(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[authApproveRegisterArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with request_id and optional authority_id, role, source_id, and target_id.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_auth_approve_register")
	}
	route, err := s.resolveAuthorityRoute(ctx, args.SourceID, args.TargetID, args.AuthorityID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id plus authority_id/target_id explicitly.")
	}
	req := authsvc.ApproveRegisterReq{
		RequestID: strings.TrimSpace(args.RequestID),
		Role:      strings.TrimSpace(args.Role),
	}
	if req.RequestID == "" {
		return invalidArgumentsResult("request_id is required", "Pass a non-empty request_id.", nil)
	}
	resp, err := s.backend.ApproveRegister(ctx, route.SourceID, route.AuthorityID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can approve register requests.", map[string]any{"source_id": route.SourceID, "target_id": route.AuthorityID, "hub_target_id": route.HubTargetID, "request": req})
	}
	return successResult(map[string]any{
		"source_id":         route.SourceID,
		"target_id":         route.AuthorityID,
		"hub_target_id":     route.HubTargetID,
		"target_resolution": route.Resolution,
		"request":           req,
		"response":          resp,
	})
}

func (s toolSet) authRejectRegister(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[authRejectRegisterArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with request_id and optional authority_id, reason, source_id, and target_id.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_auth_reject_register")
	}
	route, err := s.resolveAuthorityRoute(ctx, args.SourceID, args.TargetID, args.AuthorityID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id plus authority_id/target_id explicitly.")
	}
	req := authsvc.RejectRegisterReq{
		RequestID: strings.TrimSpace(args.RequestID),
		Reason:    strings.TrimSpace(args.Reason),
	}
	if req.RequestID == "" {
		return invalidArgumentsResult("request_id is required", "Pass a non-empty request_id.", nil)
	}
	resp, err := s.backend.RejectRegister(ctx, route.SourceID, route.AuthorityID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can reject register requests.", map[string]any{"source_id": route.SourceID, "target_id": route.AuthorityID, "hub_target_id": route.HubTargetID, "request": req})
	}
	return successResult(map[string]any{
		"source_id":         route.SourceID,
		"target_id":         route.AuthorityID,
		"hub_target_id":     route.HubTargetID,
		"target_resolution": route.Resolution,
		"request":           req,
		"response":          resp,
	})
}

func (s toolSet) authIssueRegisterPermit(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[authIssueRegisterPermitArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with device_id, role, and optional authority_id, expires_at, source_id, and target_id.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_auth_issue_register_permit")
	}
	route, err := s.resolveAuthorityRoute(ctx, args.SourceID, args.TargetID, args.AuthorityID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id plus authority_id/target_id explicitly.")
	}
	req, err := normalizeIssueRegisterPermitArgs(args)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass non-empty device_id and role, and use a non-negative expires_at.", nil)
	}
	resp, err := s.backend.IssueRegisterPermit(ctx, route.SourceID, route.AuthorityID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can issue register permits.", map[string]any{"source_id": route.SourceID, "target_id": route.AuthorityID, "hub_target_id": route.HubTargetID, "request": req})
	}
	return successResult(map[string]any{
		"source_id":         route.SourceID,
		"target_id":         route.AuthorityID,
		"hub_target_id":     route.HubTargetID,
		"target_resolution": route.Resolution,
		"request":           req,
		"response":          resp,
	})
}

func (s toolSet) authRevokeRegisterPermit(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[authRevokeRegisterPermitArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with permit and optional authority_id, source_id, and target_id.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_auth_revoke_register_permit")
	}
	route, err := s.resolveAuthorityRoute(ctx, args.SourceID, args.TargetID, args.AuthorityID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id plus authority_id/target_id explicitly.")
	}
	req := authsvc.RevokeRegisterPermitReq{Permit: strings.TrimSpace(args.Permit)}
	if req.Permit == "" {
		return invalidArgumentsResult("permit is required", "Pass a non-empty permit token.", nil)
	}
	resp, err := s.backend.RevokeRegisterPermit(ctx, route.SourceID, route.AuthorityID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can revoke register permits.", map[string]any{"source_id": route.SourceID, "target_id": route.AuthorityID, "hub_target_id": route.HubTargetID, "request": req})
	}
	return successResult(map[string]any{
		"source_id":         route.SourceID,
		"target_id":         route.AuthorityID,
		"hub_target_id":     route.HubTargetID,
		"target_resolution": route.Resolution,
		"request":           req,
		"response":          resp,
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

func (s toolSet) managementConfigGet(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[managementConfigGetArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with key and optional source_id and target_id.", nil)
	}
	key := strings.TrimSpace(args.Key)
	if key == "" {
		return invalidArgumentsResult("key is required", "Pass a non-empty config key.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_management_config_get")
	}
	sourceID, targetID, err := s.resolveManagementRoute(args.SourceID, args.TargetID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id and target_id explicitly.")
	}
	resp, err := s.backend.ConfigGet(ctx, sourceID, targetID, key)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can read management config.", map[string]any{"source_id": sourceID, "target_id": targetID, "key": key})
	}
	return successResult(map[string]any{"source_id": sourceID, "target_id": targetID, "key": key, "response": resp})
}

func (s toolSet) managementConfigList(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[managementArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with optional source_id and target_id.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_management_config_list")
	}
	sourceID, targetID, err := s.resolveManagementRoute(args.SourceID, args.TargetID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id and target_id explicitly.")
	}
	resp, err := s.backend.ConfigList(ctx, sourceID, targetID)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can list management config keys.", map[string]any{"source_id": sourceID, "target_id": targetID})
	}
	return successResult(map[string]any{"source_id": sourceID, "target_id": targetID, "response": resp})
}

func (s toolSet) managementNodeEcho(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[managementNodeEchoArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with message and optional source_id and target_id.", nil)
	}
	message := strings.TrimSpace(args.Message)
	if message == "" {
		return invalidArgumentsResult("message is required", "Pass a non-empty message.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_management_node_echo")
	}
	sourceID, targetID, err := s.resolveManagementRoute(args.SourceID, args.TargetID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id and target_id explicitly.")
	}
	resp, err := s.backend.NodeEcho(ctx, sourceID, targetID, message)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the selected target can answer management echo requests.", map[string]any{"source_id": sourceID, "target_id": targetID, "message": message})
	}
	return successResult(map[string]any{"source_id": sourceID, "target_id": targetID, "message": message, "response": resp})
}

func (s toolSet) managementListSubtree(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[managementArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with optional source_id and target_id.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_management_list_subtree")
	}
	sourceID, targetID, err := s.resolveManagementRoute(args.SourceID, args.TargetID)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id and target_id explicitly.")
	}
	resp, err := s.backend.ListSubtree(ctx, sourceID, targetID)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can inspect the target subtree.", map[string]any{"source_id": sourceID, "target_id": targetID})
	}
	return successResult(map[string]any{"source_id": sourceID, "target_id": targetID, "response": resp})
}

func (s toolSet) execCapQuery(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[execCapQueryArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with optional req_id, source_id, target_id, requester_node, method, prefix, provider_node, limit, and include_schema.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_exec_cap_query")
	}
	route, err := s.resolveExecQueryRoute(args.SourceID, args.TargetID, args.RequesterNode)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id, target_id, and requester_node explicitly.")
	}
	req, err := normalizeExecCapQueryReq(args, route)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass positive requester/provider node IDs and a non-negative limit when set.", nil)
	}
	resp, err := s.backend.ExecCapQuery(ctx, route.SourceID, route.TargetID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the selected routing target can answer exec capability queries.", map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "requester_node": route.RequesterNode, "request": req})
	}
	return successResult(map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "requester_node": route.RequesterNode, "request": req, "response": resp})
}

func (s toolSet) flowList(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[flowListArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with optional req_id, source_id, target_id, and executor_node.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_flow_list")
	}
	route, err := s.resolveFlowRoute(args.SourceID, args.TargetID, args.ExecutorNode)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id, target_id, and executor_node explicitly.")
	}
	req := normalizeFlowListReq(args, route)
	resp, err := s.backend.FlowList(ctx, route.SourceID, route.TargetID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the selected executor node is reachable.", map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "executor_node": route.ExecutorNode, "request": req})
	}
	return successResult(map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "executor_node": route.ExecutorNode, "request": req, "response": resp})
}

func (s toolSet) flowGet(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[flowGetArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with flow_id and optional req_id, source_id, target_id, and executor_node.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_flow_get")
	}
	route, err := s.resolveFlowRoute(args.SourceID, args.TargetID, args.ExecutorNode)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id, target_id, and executor_node explicitly.")
	}
	req, err := normalizeFlowGetReq(args, route)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a non-empty flow_id.", nil)
	}
	resp, err := s.backend.FlowGet(ctx, route.SourceID, route.TargetID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the selected executor node can return the requested flow.", map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "executor_node": route.ExecutorNode, "request": req})
	}
	return successResult(map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "executor_node": route.ExecutorNode, "request": req, "response": resp})
}

func (s toolSet) flowSet(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[flowSetArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with flow_id, trigger, graph, and optional req_id, name, source_id, target_id, and executor_node.", nil)
	}
	if !s.backend.AllowWrite() {
		return writeDisabledResult("myflowhub_flow_set")
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_flow_set")
	}
	route, err := s.resolveFlowRoute(args.SourceID, args.TargetID, args.ExecutorNode)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id, target_id, and executor_node explicitly.")
	}
	req, err := normalizeFlowSetReq(args, route)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a non-empty flow_id plus trigger and graph objects that match the flow protocol.", nil)
	}
	resp, err := s.backend.FlowSet(ctx, route.SourceID, route.TargetID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity, write gating, and whether the current role can persist flows.", map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "executor_node": route.ExecutorNode, "request": req})
	}
	return successResult(map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "executor_node": route.ExecutorNode, "request": req, "response": resp})
}

func (s toolSet) flowRun(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[flowRunArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with flow_id and optional req_id, source_id, target_id, and executor_node.", nil)
	}
	if !s.backend.AllowWrite() {
		return writeDisabledResult("myflowhub_flow_run")
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_flow_run")
	}
	route, err := s.resolveFlowRoute(args.SourceID, args.TargetID, args.ExecutorNode)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id, target_id, and executor_node explicitly.")
	}
	req, err := normalizeFlowRunReq(args, route)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a non-empty flow_id.", nil)
	}
	resp, err := s.backend.FlowRun(ctx, route.SourceID, route.TargetID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the selected executor node can run the requested flow.", map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "executor_node": route.ExecutorNode, "request": req})
	}
	return successResult(map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "executor_node": route.ExecutorNode, "request": req, "response": resp})
}

func (s toolSet) flowCancelRun(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[flowCancelRunArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with flow_id, run_id, and optional req_id, source_id, target_id, and executor_node.", nil)
	}
	if !s.backend.AllowWrite() {
		return writeDisabledResult("myflowhub_flow_cancel_run")
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_flow_cancel_run")
	}
	route, err := s.resolveFlowRoute(args.SourceID, args.TargetID, args.ExecutorNode)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id, target_id, and executor_node explicitly.")
	}
	req, err := normalizeFlowCancelRunReq(args, route)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass non-empty flow_id and run_id.", nil)
	}
	resp, err := s.backend.FlowCancelRun(ctx, route.SourceID, route.TargetID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the selected executor node can cancel the requested run.", map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "executor_node": route.ExecutorNode, "request": req})
	}
	return successResult(map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "executor_node": route.ExecutorNode, "request": req, "response": resp})
}

func (s toolSet) flowStatus(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[flowStatusArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with flow_id and optional run_id, req_id, source_id, target_id, and executor_node.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_flow_status")
	}
	route, err := s.resolveFlowRoute(args.SourceID, args.TargetID, args.ExecutorNode)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id, target_id, and executor_node explicitly.")
	}
	req, err := normalizeFlowStatusReq(args, route)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a non-empty flow_id and only include run_id when it is non-empty.", nil)
	}
	resp, err := s.backend.FlowStatus(ctx, route.SourceID, route.TargetID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the selected executor node can read flow status.", map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "executor_node": route.ExecutorNode, "request": req})
	}
	return successResult(map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "executor_node": route.ExecutorNode, "request": req, "response": resp})
}

func (s toolSet) flowListRuns(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[flowListRunsArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with flow_id and optional limit, req_id, source_id, target_id, and executor_node.", nil)
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_flow_list_runs")
	}
	route, err := s.resolveFlowRoute(args.SourceID, args.TargetID, args.ExecutorNode)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id, target_id, and executor_node explicitly.")
	}
	req, err := normalizeFlowListRunsReq(args, route)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a non-empty flow_id and a non-negative limit when set.", nil)
	}
	resp, err := s.backend.FlowListRuns(ctx, route.SourceID, route.TargetID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the selected executor node can list retained runs.", map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "executor_node": route.ExecutorNode, "request": req})
	}
	return successResult(map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "executor_node": route.ExecutorNode, "request": req, "response": resp})
}

func (s toolSet) flowDelete(ctx context.Context, raw json.RawMessage) CallToolResult {
	args, err := decodeArgs[flowDeleteArgs](raw)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a JSON object with flow_id and optional req_id, source_id, target_id, and executor_node.", nil)
	}
	if !s.backend.AllowWrite() {
		return writeDisabledResult("myflowhub_flow_delete")
	}
	if !s.backend.SessionConnected() {
		return notConnectedResult("myflowhub_flow_delete")
	}
	route, err := s.resolveFlowRoute(args.SourceID, args.TargetID, args.ExecutorNode)
	if err != nil {
		return routeErrorResult(err, "Login first or pass source_id, target_id, and executor_node explicitly.")
	}
	req, err := normalizeFlowDeleteReq(args, route)
	if err != nil {
		return invalidArgumentsResult(err.Error(), "Pass a non-empty flow_id.", nil)
	}
	resp, err := s.backend.FlowDelete(ctx, route.SourceID, route.TargetID, req)
	if err != nil {
		return upstreamErrorResult(err, "Check hub connectivity and whether the current role can delete flows.", map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "executor_node": route.ExecutorNode, "request": req})
	}
	return successResult(map[string]any{"source_id": route.SourceID, "target_id": route.TargetID, "executor_node": route.ExecutorNode, "request": req, "response": resp})
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

func (s toolSet) resolveFlowRoute(sourceID, targetID, executorNode *uint32) (flowRoute, error) {
	source, target, err := s.resolveManagementRoute(sourceID, targetID)
	if err != nil {
		return flowRoute{}, err
	}
	executor, explicit, err := positiveNodeID(executorNode, "executor_node")
	if err != nil {
		return flowRoute{}, err
	}
	if !explicit || executor == 0 {
		executor = target
	}
	if executor == 0 {
		return flowRoute{}, errors.New("executor_node is required; pass executor_node or target_id")
	}
	return flowRoute{
		SourceID:     source,
		TargetID:     target,
		ExecutorNode: executor,
	}, nil
}

func (s toolSet) resolveExecQueryRoute(sourceID, targetID, requesterNode *uint32) (execQueryRoute, error) {
	source, target, err := s.resolveManagementRoute(sourceID, targetID)
	if err != nil {
		return execQueryRoute{}, err
	}
	requester, explicit, err := positiveNodeID(requesterNode, "requester_node")
	if err != nil {
		return execQueryRoute{}, err
	}
	if !explicit || requester == 0 {
		requester = source
	}
	if requester == 0 {
		return execQueryRoute{}, errors.New("requester_node is required; login first or pass requester_node")
	}
	return execQueryRoute{
		SourceID:      source,
		TargetID:      target,
		RequesterNode: requester,
	}, nil
}

func (s toolSet) resolveAuthorityRoute(ctx context.Context, sourceID, targetID, authorityID *uint32) (authorityRoute, error) {
	source, hubTarget, err := s.resolveManagementRoute(sourceID, targetID)
	if err != nil {
		return authorityRoute{}, err
	}
	if authorityID != nil {
		if *authorityID == 0 {
			return authorityRoute{}, errors.New("authority_id must be greater than 0")
		}
		return authorityRoute{
			SourceID:    source,
			HubTargetID: hubTarget,
			AuthorityID: *authorityID,
			Resolution:  "authority_id",
		}, nil
	}
	authorityTarget := hubTarget
	resolution := "hub_target"
	resp, err := s.backend.ConfigGet(ctx, source, hubTarget, authorityNodeIDConfigKey)
	if err == nil {
		if parsed, ok := parsePositiveUint32String(resp.Value); ok {
			authorityTarget = parsed
			resolution = "authority.node_id"
		}
	}
	return authorityRoute{
		SourceID:    source,
		HubTargetID: hubTarget,
		AuthorityID: authorityTarget,
		Resolution:  resolution,
	}, nil
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

func (s toolSet) resolveQueryNodeID(nodeID *uint32) (uint32, error) {
	resolved, explicit, err := positiveNodeID(nodeID, "node_id")
	if err != nil {
		return 0, err
	}
	if !explicit {
		if snap := s.backend.AuthSnapshot(); snap.LoggedIn && snap.NodeID != 0 {
			resolved = snap.NodeID
		} else if defaults := s.backend.Defaults(); defaults.NodeID != 0 {
			resolved = defaults.NodeID
		}
	}
	if resolved == 0 {
		return 0, errors.New("node_id is required; login first or pass node_id")
	}
	return resolved, nil
}

func normalizeListRolesArgs(args authListRolesArgs) (protoauth.ListRolesReq, error) {
	req := protoauth.ListRolesReq{
		Role: strings.TrimSpace(args.Role),
	}
	if args.Offset != nil {
		if *args.Offset < 0 {
			return protoauth.ListRolesReq{}, errors.New("offset must be greater than or equal to 0")
		}
		req.Offset = *args.Offset
	}
	if args.Limit != nil {
		if *args.Limit < 0 {
			return protoauth.ListRolesReq{}, errors.New("limit must be greater than or equal to 0")
		}
		req.Limit = *args.Limit
	}
	if len(args.NodeIDs) > 0 {
		nodeIDs := make([]uint32, 0, len(args.NodeIDs))
		for _, nodeID := range args.NodeIDs {
			if nodeID == 0 {
				return protoauth.ListRolesReq{}, errors.New("node_ids entries must be greater than 0")
			}
			nodeIDs = append(nodeIDs, nodeID)
		}
		req.NodeIDs = nodeIDs
	}
	return req, nil
}

func normalizeListPendingRegistersArgs(args authListPendingRegistersArgs) (authsvc.ListPendingRegistersReq, error) {
	req := authsvc.ListPendingRegistersReq{
		DeviceID: strings.TrimSpace(args.DeviceID),
	}
	if args.Offset != nil {
		if *args.Offset < 0 {
			return authsvc.ListPendingRegistersReq{}, errors.New("offset must be greater than or equal to 0")
		}
		req.Offset = *args.Offset
	}
	if args.Limit != nil {
		if *args.Limit < 0 {
			return authsvc.ListPendingRegistersReq{}, errors.New("limit must be greater than or equal to 0")
		}
		req.Limit = *args.Limit
	}
	return req, nil
}

func normalizeIssueRegisterPermitArgs(args authIssueRegisterPermitArgs) (authsvc.IssueRegisterPermitReq, error) {
	req := authsvc.IssueRegisterPermitReq{
		DeviceID: strings.TrimSpace(args.DeviceID),
		Role:     strings.TrimSpace(args.Role),
	}
	if req.DeviceID == "" {
		return authsvc.IssueRegisterPermitReq{}, errors.New("device_id is required")
	}
	if req.Role == "" {
		return authsvc.IssueRegisterPermitReq{}, errors.New("role is required")
	}
	if args.ExpiresAt != nil {
		if *args.ExpiresAt < 0 {
			return authsvc.IssueRegisterPermitReq{}, errors.New("expires_at must be greater than or equal to 0")
		}
		req.ExpiresAt = *args.ExpiresAt
	}
	return req, nil
}

func normalizeExecCapQueryReq(args execCapQueryArgs, route execQueryRoute) (protoexec.CapQueryReq, error) {
	req := protoexec.CapQueryReq{
		ReqID:         ensureReqID(args.ReqID),
		RequesterNode: route.RequesterNode,
		Method:        strings.TrimSpace(args.Method),
		Prefix:        args.Prefix,
		IncludeSchema: args.IncludeSchema,
	}
	if args.ProviderNode != nil {
		if *args.ProviderNode == 0 {
			return protoexec.CapQueryReq{}, errors.New("provider_node must be greater than 0")
		}
		req.ProviderNode = *args.ProviderNode
	}
	if args.Limit != nil {
		if *args.Limit < 0 {
			return protoexec.CapQueryReq{}, errors.New("limit must be greater than or equal to 0")
		}
		req.Limit = *args.Limit
	}
	return req, nil
}

func normalizeFlowListReq(args flowListArgs, route flowRoute) protoflow.ListReq {
	return protoflow.ListReq{
		ReqID:        ensureReqID(args.ReqID),
		OriginNode:   route.SourceID,
		ExecutorNode: route.ExecutorNode,
	}
}

func normalizeFlowGetReq(args flowGetArgs, route flowRoute) (protoflow.GetReq, error) {
	flowID := strings.TrimSpace(args.FlowID)
	if flowID == "" {
		return protoflow.GetReq{}, errors.New("flow_id is required")
	}
	return protoflow.GetReq{
		ReqID:        ensureReqID(args.ReqID),
		OriginNode:   route.SourceID,
		ExecutorNode: route.ExecutorNode,
		FlowID:       flowID,
	}, nil
}

func normalizeFlowSetReq(args flowSetArgs, route flowRoute) (protoflow.SetReq, error) {
	flowID := strings.TrimSpace(args.FlowID)
	if flowID == "" {
		return protoflow.SetReq{}, errors.New("flow_id is required")
	}
	trigger := args.Trigger
	trigger.Type = strings.TrimSpace(trigger.Type)
	trigger.EventMode = strings.TrimSpace(trigger.EventMode)
	trigger.EventName = strings.TrimSpace(trigger.EventName)
	trigger.EventTopic = strings.TrimSpace(trigger.EventTopic)
	trigger.VarName = strings.TrimSpace(trigger.VarName)
	if trigger.Type == "" {
		return protoflow.SetReq{}, errors.New("trigger.type is required")
	}
	graph := args.Graph
	if len(graph.Nodes) == 0 && len(graph.Edges) == 0 {
		return protoflow.SetReq{}, errors.New("graph is required")
	}
	return protoflow.SetReq{
		ReqID:        ensureReqID(args.ReqID),
		OriginNode:   route.SourceID,
		ExecutorNode: route.ExecutorNode,
		FlowID:       flowID,
		Name:         strings.TrimSpace(args.Name),
		Trigger:      trigger,
		Graph:        graph,
	}, nil
}

func normalizeFlowRunReq(args flowRunArgs, route flowRoute) (protoflow.RunReq, error) {
	flowID := strings.TrimSpace(args.FlowID)
	if flowID == "" {
		return protoflow.RunReq{}, errors.New("flow_id is required")
	}
	return protoflow.RunReq{
		ReqID:        ensureReqID(args.ReqID),
		OriginNode:   route.SourceID,
		ExecutorNode: route.ExecutorNode,
		FlowID:       flowID,
	}, nil
}

func normalizeFlowCancelRunReq(args flowCancelRunArgs, route flowRoute) (protoflow.CancelRunReq, error) {
	flowID := strings.TrimSpace(args.FlowID)
	if flowID == "" {
		return protoflow.CancelRunReq{}, errors.New("flow_id is required")
	}
	runID := strings.TrimSpace(args.RunID)
	if runID == "" {
		return protoflow.CancelRunReq{}, errors.New("run_id is required")
	}
	return protoflow.CancelRunReq{
		ReqID:        ensureReqID(args.ReqID),
		OriginNode:   route.SourceID,
		ExecutorNode: route.ExecutorNode,
		FlowID:       flowID,
		RunID:        runID,
	}, nil
}

func normalizeFlowStatusReq(args flowStatusArgs, route flowRoute) (protoflow.StatusReq, error) {
	flowID := strings.TrimSpace(args.FlowID)
	if flowID == "" {
		return protoflow.StatusReq{}, errors.New("flow_id is required")
	}
	return protoflow.StatusReq{
		ReqID:        ensureReqID(args.ReqID),
		OriginNode:   route.SourceID,
		ExecutorNode: route.ExecutorNode,
		FlowID:       flowID,
		RunID:        strings.TrimSpace(args.RunID),
	}, nil
}

func normalizeFlowListRunsReq(args flowListRunsArgs, route flowRoute) (protoflow.ListRunsReq, error) {
	flowID := strings.TrimSpace(args.FlowID)
	if flowID == "" {
		return protoflow.ListRunsReq{}, errors.New("flow_id is required")
	}
	req := protoflow.ListRunsReq{
		ReqID:        ensureReqID(args.ReqID),
		OriginNode:   route.SourceID,
		ExecutorNode: route.ExecutorNode,
		FlowID:       flowID,
	}
	if args.Limit != nil {
		req.Limit = *args.Limit
	}
	return req, nil
}

func normalizeFlowDeleteReq(args flowDeleteArgs, route flowRoute) (flowsvc.DeleteReq, error) {
	flowID := strings.TrimSpace(args.FlowID)
	if flowID == "" {
		return flowsvc.DeleteReq{}, errors.New("flow_id is required")
	}
	return flowsvc.DeleteReq{
		ReqID:        ensureReqID(args.ReqID),
		OriginNode:   route.SourceID,
		ExecutorNode: route.ExecutorNode,
		FlowID:       flowID,
	}, nil
}

func ensureReqID(value string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return uuid.NewString()
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

func parsePositiveUint32String(value string) (uint32, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil || parsed == 0 {
		return 0, false
	}
	return uint32(parsed), true
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
		hints = append(hints, "Call myflowhub_session_connect before invoking auth, management, exec, flow, or varstore tools.")
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
		hints = append(hints, "Restart the MCP process with --allow-write to enable myflowhub_flow_set, myflowhub_flow_run, myflowhub_flow_cancel_run, myflowhub_flow_delete, myflowhub_varstore_set, and myflowhub_varstore_revoke.")
	}
	if len(hints) == 0 {
		hints = append(hints, "Session is ready for management, exec, flow, and varstore read operations.")
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

func booleanSchema(description string) map[string]any {
	return map[string]any{
		"type":        "boolean",
		"description": description,
	}
}

func openObjectSchema(description string) map[string]any {
	return map[string]any{
		"type":        "object",
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

func nonNegativeIntegerSchema(description string) map[string]any {
	return map[string]any{
		"type":        "integer",
		"minimum":     0,
		"description": description,
	}
}

func integerArraySchema(description string, minimum int) map[string]any {
	return map[string]any{
		"type":        "array",
		"description": description,
		"items": map[string]any{
			"type":    "integer",
			"minimum": minimum,
		},
	}
}
