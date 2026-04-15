// 本文件覆盖 `flow` 后端服务的行为。

package flow

import (
	"encoding/json"
	"strings"
	"testing"

	protoflow "github.com/yttydcs/myflowhub-proto/protocol/flow"
)

func TestDetailValidation(t *testing.T) {
	svc := New(nil, nil)

	if _, err := svc.DetailSimple(1, 2, DetailReq{}); err == nil {
		t.Fatal("expected req_id validation error")
	}
	if _, err := svc.DetailSimple(1, 2, DetailReq{ReqID: "req-1"}); err == nil {
		t.Fatal("expected flow_id validation error")
	}
	if _, err := svc.DetailSimple(1, 2, DetailReq{ReqID: "req-1", FlowID: "flow-1"}); err == nil {
		t.Fatal("expected node_id validation error")
	}
}

func TestDetailRequiresSessionAfterValidation(t *testing.T) {
	svc := New(nil, nil)

	_, err := svc.DetailSimple(1, 2, DetailReq{
		ReqID:  "req-1",
		FlowID: "flow-1",
		NodeID: "node-1",
	})
	if err == nil || !strings.Contains(err.Error(), "session service not initialized") {
		t.Fatalf("expected session init error, got %v", err)
	}
}

func TestExtractCodeMsgWithDetailResp(t *testing.T) {
	code, msg := extractCodeMsg(&DetailResp{
		Code: 207,
		Msg:  "detail ok",
	})
	if code != 207 || msg != "detail ok" {
		t.Fatalf("unexpected code/msg: %d %q", code, msg)
	}
}

func TestDetailRespJSONShape(t *testing.T) {
	var resp DetailResp
	if err := json.Unmarshal([]byte(`{
		"req_id":"req-1",
		"code":1,
		"executor_node":42,
		"flow_id":"flow-1",
		"run_id":"run-1",
		"path":"/result",
		"node":{"id":"node-1","status":"ok","code":200,"msg":"done"},
		"result":{"value":"hello"}
	}`), &resp); err != nil {
		t.Fatalf("unmarshal detail resp: %v", err)
	}
	if resp.ReqID != "req-1" || resp.ExecutorNode != 42 || resp.RunID != "run-1" || resp.Path != "/result" {
		t.Fatalf("unexpected detail resp header: %+v", resp)
	}
	if resp.Node == nil {
		t.Fatal("expected node to be populated")
	}
	if resp.Node.ID != "node-1" || resp.Node.Status != "ok" || resp.Node.Code != 200 || resp.Node.Msg != "done" {
		t.Fatalf("unexpected node payload: %+v", resp.Node)
	}
}

func TestCancelRunValidation(t *testing.T) {
	svc := New(nil, nil)

	if _, err := svc.CancelRunSimple(1, 2, protoflow.CancelRunReq{}); err == nil {
		t.Fatal("expected req_id validation error")
	}
	if _, err := svc.CancelRunSimple(1, 2, protoflow.CancelRunReq{ReqID: "req-1"}); err == nil {
		t.Fatal("expected flow_id validation error")
	}
	if _, err := svc.CancelRunSimple(1, 2, protoflow.CancelRunReq{ReqID: "req-1", FlowID: "flow-1"}); err == nil {
		t.Fatal("expected run_id validation error")
	}
}

func TestListRunsValidation(t *testing.T) {
	svc := New(nil, nil)

	if _, err := svc.ListRunsSimple(1, 2, protoflow.ListRunsReq{}); err == nil {
		t.Fatal("expected req_id validation error")
	}
	if _, err := svc.ListRunsSimple(1, 2, protoflow.ListRunsReq{ReqID: "req-1"}); err == nil {
		t.Fatal("expected flow_id validation error")
	}
}

func TestExtractCodeMsgWithCancelRunResp(t *testing.T) {
	code, msg := extractCodeMsg(&protoflow.CancelRunResp{
		Code: 205,
		Msg:  "cancelled",
	})
	if code != 205 || msg != "cancelled" {
		t.Fatalf("unexpected code/msg: %d %q", code, msg)
	}
}

func TestExtractCodeMsgWithListRunsResp(t *testing.T) {
	code, msg := extractCodeMsg(&protoflow.ListRunsResp{
		Code: 201,
		Msg:  "runs ok",
	})
	if code != 201 || msg != "runs ok" {
		t.Fatalf("unexpected code/msg: %d %q", code, msg)
	}
}

func TestCancelRunRespJSONShape(t *testing.T) {
	var resp protoflow.CancelRunResp
	if err := json.Unmarshal([]byte(`{
		"req_id":"req-1",
		"code":1,
		"executor_node":42,
		"flow_id":"flow-1",
		"run_id":"run-1",
		"status":"cancelled"
	}`), &resp); err != nil {
		t.Fatalf("unmarshal cancel_run resp: %v", err)
	}
	if resp.ReqID != "req-1" || resp.Code != 1 || resp.ExecutorNode != 42 || resp.FlowID != "flow-1" || resp.RunID != "run-1" || resp.Status != "cancelled" {
		t.Fatalf("unexpected cancel_run resp: %+v", resp)
	}
}

func TestListRunsRespJSONShape(t *testing.T) {
	var resp protoflow.ListRunsResp
	if err := json.Unmarshal([]byte(`{
		"req_id":"req-1",
		"code":1,
		"executor_node":42,
		"flow_id":"flow-1",
		"runs":[
			{
				"run_id":"run-1",
				"status":"running",
				"started_at_ms":1710000000000,
				"ended_at_ms":1710000001000,
				"msg":"still working"
			}
		]
	}`), &resp); err != nil {
		t.Fatalf("unmarshal list_runs resp: %v", err)
	}
	if resp.ReqID != "req-1" || resp.Code != 1 || resp.ExecutorNode != 42 || resp.FlowID != "flow-1" {
		t.Fatalf("unexpected list_runs resp header: %+v", resp)
	}
	if len(resp.Runs) != 1 {
		t.Fatalf("expected 1 run, got %+v", resp.Runs)
	}
	run := resp.Runs[0]
	if run.RunID != "run-1" || run.Status != "running" || run.StartedAtMs != 1710000000000 || run.EndedAtMs != 1710000001000 || run.Msg != "still working" {
		t.Fatalf("unexpected run summary: %+v", run)
	}
}
