package flow

import (
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
