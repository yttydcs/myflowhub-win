package ui

import "encoding/json"

// flow 子协议（SubProto=6），用于工作流（DAG）设置/查询/触发。
const (
	subProtoFlow uint8 = 6
)

const (
	flowActionSet        = "set"
	flowActionSetResp    = "set_resp"
	flowActionRun        = "run"
	flowActionRunResp    = "run_resp"
	flowActionStatus     = "status"
	flowActionStatusResp = "status_resp"
	flowActionList       = "list"
	flowActionListResp   = "list_resp"
	flowActionGet        = "get"
	flowActionGetResp    = "get_resp"
)

type flowMessage struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type flowTrigger struct {
	Type    string `json:"type"`
	EveryMs uint64 `json:"every_ms,omitempty"`
}

type flowEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type flowNode struct {
	ID        string          `json:"id"`
	Kind      string          `json:"kind"`
	AllowFail bool            `json:"allow_fail,omitempty"`
	Retry     *int            `json:"retry,omitempty"`
	TimeoutMs *int            `json:"timeout_ms,omitempty"`
	Spec      json.RawMessage `json:"spec"`
}

type flowGraph struct {
	Nodes []flowNode `json:"nodes"`
	Edges []flowEdge `json:"edges"`
}

type flowSetReq struct {
	ReqID        string      `json:"req_id"`
	OriginNode   uint32      `json:"origin_node,omitempty"`
	ExecutorNode uint32      `json:"executor_node,omitempty"`
	FlowID       string      `json:"flow_id"`
	Name         string      `json:"name,omitempty"`
	Trigger      flowTrigger `json:"trigger"`
	Graph        flowGraph   `json:"graph"`
}

type flowSetResp struct {
	ReqID  string `json:"req_id"`
	Code   int    `json:"code"`
	Msg    string `json:"msg,omitempty"`
	FlowID string `json:"flow_id,omitempty"`
}

type flowListReq struct {
	ReqID        string `json:"req_id"`
	OriginNode   uint32 `json:"origin_node,omitempty"`
	ExecutorNode uint32 `json:"executor_node,omitempty"`
}

type flowSummary struct {
	FlowID     string `json:"flow_id"`
	Name       string `json:"name,omitempty"`
	EveryMs    uint64 `json:"every_ms,omitempty"`
	LastRunID  string `json:"last_run_id,omitempty"`
	LastStatus string `json:"last_status,omitempty"`
}

type flowListResp struct {
	ReqID        string        `json:"req_id"`
	Code         int           `json:"code"`
	Msg          string        `json:"msg,omitempty"`
	ExecutorNode uint32        `json:"executor_node,omitempty"`
	Flows        []flowSummary `json:"flows,omitempty"`
}

type flowGetReq struct {
	ReqID        string `json:"req_id"`
	OriginNode   uint32 `json:"origin_node,omitempty"`
	ExecutorNode uint32 `json:"executor_node,omitempty"`
	FlowID       string `json:"flow_id"`
}

type flowGetResp struct {
	ReqID        string      `json:"req_id"`
	Code         int         `json:"code"`
	Msg          string      `json:"msg,omitempty"`
	ExecutorNode uint32      `json:"executor_node,omitempty"`
	FlowID       string      `json:"flow_id,omitempty"`
	Name         string      `json:"name,omitempty"`
	Trigger      flowTrigger `json:"trigger,omitempty"`
	Graph        flowGraph   `json:"graph,omitempty"`
}

type flowRunReq struct {
	ReqID        string `json:"req_id"`
	OriginNode   uint32 `json:"origin_node,omitempty"`
	ExecutorNode uint32 `json:"executor_node,omitempty"`
	FlowID       string `json:"flow_id"`
}

type flowRunResp struct {
	ReqID  string `json:"req_id"`
	Code   int    `json:"code"`
	Msg    string `json:"msg,omitempty"`
	FlowID string `json:"flow_id,omitempty"`
	RunID  string `json:"run_id,omitempty"`
}

type flowNodeStatus struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Code   int    `json:"code,omitempty"`
	Msg    string `json:"msg,omitempty"`
}

type flowStatusReq struct {
	ReqID        string `json:"req_id"`
	OriginNode   uint32 `json:"origin_node,omitempty"`
	ExecutorNode uint32 `json:"executor_node,omitempty"`
	FlowID       string `json:"flow_id"`
	RunID        string `json:"run_id,omitempty"`
}

type flowStatusResp struct {
	ReqID        string           `json:"req_id"`
	Code         int              `json:"code"`
	Msg          string           `json:"msg,omitempty"`
	ExecutorNode uint32           `json:"executor_node,omitempty"`
	FlowID       string           `json:"flow_id,omitempty"`
	RunID        string           `json:"run_id,omitempty"`
	Status       string           `json:"status,omitempty"`
	Nodes        []flowNodeStatus `json:"nodes,omitempty"`
}
