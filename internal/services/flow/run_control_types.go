package flow

const (
	actionCancelRun     = "cancel_run"
	actionCancelRunResp = "cancel_run_resp"
	actionListRuns      = "list_runs"
	actionListRunsResp  = "list_runs_resp"
)

// CancelRunReq and related run-control payloads remain Win-local compatibility
// types until the consumed shared proto baseline exports them.
type CancelRunReq struct {
	ReqID        string `json:"req_id"`
	OriginNode   uint32 `json:"origin_node,omitempty"`
	ExecutorNode uint32 `json:"executor_node,omitempty"`
	FlowID       string `json:"flow_id"`
	RunID        string `json:"run_id"`
}

type CancelRunResp struct {
	ReqID        string `json:"req_id"`
	Code         int    `json:"code"`
	Msg          string `json:"msg,omitempty"`
	ExecutorNode uint32 `json:"executor_node,omitempty"`
	FlowID       string `json:"flow_id,omitempty"`
	RunID        string `json:"run_id,omitempty"`
	Status       string `json:"status,omitempty"`
}

type ListRunsReq struct {
	ReqID        string `json:"req_id"`
	OriginNode   uint32 `json:"origin_node,omitempty"`
	ExecutorNode uint32 `json:"executor_node,omitempty"`
	FlowID       string `json:"flow_id"`
	Limit        int    `json:"limit,omitempty"`
}

type RunSummary struct {
	RunID       string `json:"run_id"`
	Status      string `json:"status,omitempty"`
	Code        int    `json:"code,omitempty"`
	Msg         string `json:"msg,omitempty"`
	StartedAtMs uint64 `json:"started_at_ms,omitempty"`
	EndedAtMs   uint64 `json:"ended_at_ms,omitempty"`
}

type ListRunsResp struct {
	ReqID        string       `json:"req_id"`
	Code         int          `json:"code"`
	Msg          string       `json:"msg,omitempty"`
	ExecutorNode uint32       `json:"executor_node,omitempty"`
	FlowID       string       `json:"flow_id,omitempty"`
	Runs         []RunSummary `json:"runs,omitempty"`
}
