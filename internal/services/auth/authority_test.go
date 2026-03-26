package auth

import (
	"strings"
	"testing"
)

func TestAuthorityActionValidation(t *testing.T) {
	svc := New(nil, nil, nil)

	if _, err := svc.ListPendingRegistersSimple(1, 2, ListPendingRegistersReq{Limit: -1}); err == nil {
		t.Fatal("expected negative limit validation error")
	}
	if _, err := svc.ApproveRegisterSimple(1, 2, ApproveRegisterReq{}); err == nil {
		t.Fatal("expected request_id validation error")
	}
	if _, err := svc.RejectRegisterSimple(1, 2, RejectRegisterReq{}); err == nil {
		t.Fatal("expected request_id validation error")
	}
	if _, err := svc.IssueRegisterPermitSimple(1, 2, IssueRegisterPermitReq{}); err == nil {
		t.Fatal("expected device_id validation error")
	}
	if _, err := svc.RevokeRegisterPermitSimple(1, 2, RevokeRegisterPermitReq{}); err == nil {
		t.Fatal("expected permit validation error")
	}
}

func TestAuthorityActionRequiresSessionAfterValidation(t *testing.T) {
	svc := New(nil, nil, nil)

	if _, err := svc.ListPendingRegistersSimple(1, 2, ListPendingRegistersReq{}); err == nil || !strings.Contains(err.Error(), "session service not initialized") {
		t.Fatalf("expected session init error, got %v", err)
	}
	if _, err := svc.IssueRegisterPermitSimple(1, 2, IssueRegisterPermitReq{DeviceID: "dev-1", Role: "admin"}); err == nil || !strings.Contains(err.Error(), "session service not initialized") {
		t.Fatalf("expected session init error, got %v", err)
	}
}
