package flow

import (
	"encoding/json"
	"strings"
	"testing"
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
