// 本文件实现 `flow` 后端服务中与 `detail_types` 相关的辅助逻辑。

package flow

import protoflow "github.com/yttydcs/myflowhub-proto/protocol/flow"

const (
	actionDetail     = "detail"
	actionDetailResp = "detail_resp"
)

// DetailReq remains a Win-local alias for the detail request flow.
// It intentionally matches protoflow.DetailReq JSON shape for compatibility.
type DetailReq struct {
	ReqID        string `json:"req_id"`
	OriginNode   uint32 `json:"origin_node,omitempty"`
	ExecutorNode uint32 `json:"executor_node,omitempty"`
	FlowID       string `json:"flow_id"`
	RunID        string `json:"run_id,omitempty"`
	NodeID       string `json:"node_id"`
	Path         string `json:"path,omitempty"`
}

type DetailResp struct {
	ReqID        string                `json:"req_id"`
	Code         int                   `json:"code"`
	Msg          string                `json:"msg,omitempty"`
	ExecutorNode uint32                `json:"executor_node,omitempty"`
	FlowID       string                `json:"flow_id,omitempty"`
	RunID        string                `json:"run_id,omitempty"`
	Path         string                `json:"path,omitempty"`
	Node         *protoflow.NodeStatus `json:"node,omitempty"`
	Result       any                   `json:"result,omitempty"`
}
