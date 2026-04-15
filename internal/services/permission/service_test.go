// 本文件覆盖 `permission` 后端服务的行为。

package permission

import (
	"slices"
	"testing"

	authsvc "github.com/yttydcs/myflowhub-win/internal/services/auth"
)

func TestNormalizePolicySortsAndDedupes(t *testing.T) {
	in := Policy{
		DefaultRole:  "admin",
		DefaultPerms: []string{"flow.set", "flow.set", "exec.call", " "},
		NodeRoles: []NodeRole{
			{NodeID: 9, Role: "writer"},
			{NodeID: 1, Role: "admin"},
		},
		RolePerms: []RolePerm{
			{Role: "writer", Perms: []string{"file.write", "file.write", "file.read"}},
			{Role: "admin", Perms: []string{"*", "file.write"}},
		},
	}

	got, warnings, err := normalizePolicy(in)
	if err != nil {
		t.Fatalf("normalizePolicy error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if got.DefaultRole != "admin" {
		t.Fatalf("default role=%q, want admin", got.DefaultRole)
	}
	wantDefaultPerms := []string{"exec.call", "flow.set"}
	if !slices.Equal(got.DefaultPerms, wantDefaultPerms) {
		t.Fatalf("default perms=%v, want %v", got.DefaultPerms, wantDefaultPerms)
	}
	if len(got.NodeRoles) != 2 || got.NodeRoles[0].NodeID != 1 || got.NodeRoles[1].NodeID != 9 {
		t.Fatalf("node roles not sorted: %#v", got.NodeRoles)
	}
	if len(got.RolePerms) != 2 || got.RolePerms[0].Role != "admin" || got.RolePerms[1].Role != "writer" {
		t.Fatalf("role perms not sorted: %#v", got.RolePerms)
	}
}

func TestNormalizePolicyRejectsDuplicateNodeRole(t *testing.T) {
	_, _, err := normalizePolicy(Policy{
		DefaultRole: "node",
		NodeRoles: []NodeRole{
			{NodeID: 1, Role: "admin"},
			{NodeID: 1, Role: "writer"},
		},
	})
	if err == nil {
		t.Fatal("expected error for duplicate node role, got nil")
	}
}

func TestParseRawPolicyDefaultsToNodeRole(t *testing.T) {
	policy, warnings, err := parseRawPolicy(RawPolicy{
		AuthDefaultRole:  "",
		AuthDefaultPerms: "",
		AuthNodeRoles:    "1:admin",
		AuthRolePerms:    "admin:file.read,file.write",
	})
	if err != nil {
		t.Fatalf("parseRawPolicy error: %v", err)
	}
	if policy.DefaultRole != "node" {
		t.Fatalf("default role=%q, want node", policy.DefaultRole)
	}
	if len(policy.NodeRoles) != 1 || policy.NodeRoles[0].NodeID != 1 {
		t.Fatalf("unexpected node roles: %#v", policy.NodeRoles)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
}

func TestToPendingRegistersPreservesFields(t *testing.T) {
	got := toPendingRegisters([]authsvc.PendingRegisterInfo{
		{
			RequestID:     " req-1 ",
			DeviceID:      " dev-1 ",
			RequestedRole: " worker ",
			DisplayName:   " node-a ",
			CreatedAt:     10,
			ExpiresAt:     20,
		},
	})
	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	if got[0].RequestID != "req-1" || got[0].DeviceID != "dev-1" {
		t.Fatalf("unexpected ids: %#v", got[0])
	}
	if got[0].RequestedRole != "worker" || got[0].DisplayName != "node-a" {
		t.Fatalf("unexpected role/display: %#v", got[0])
	}
	if got[0].CreatedAt != 10 || got[0].ExpiresAt != 20 {
		t.Fatalf("unexpected timestamps: %#v", got[0])
	}
}

func TestAuthorityActionValidation(t *testing.T) {
	svc := New(nil, nil, nil)

	if _, err := svc.ListPendingRegisters(ListPendingRegistersRequest{}); err == nil {
		t.Fatal("expected source_id validation error")
	}
	if _, err := svc.ApproveRegister(ApproveRegisterRequest{SourceID: 1, AuthorityID: 2}); err == nil {
		t.Fatal("expected request_id validation error")
	}
	if _, err := svc.RejectRegister(RejectRegisterRequest{SourceID: 1, AuthorityID: 2}); err == nil {
		t.Fatal("expected reject request_id validation error")
	}
	if _, err := svc.IssueRegisterPermit(IssueRegisterPermitRequest{SourceID: 1, AuthorityID: 2, Role: "admin"}); err == nil {
		t.Fatal("expected device_id validation error")
	}
	if _, err := svc.RevokeRegisterPermit(RevokeRegisterPermitRequest{SourceID: 1, AuthorityID: 2}); err == nil {
		t.Fatal("expected permit validation error")
	}
}

func TestPermitActionsAllowRemoteAuthorityBeforeAuthInvocation(t *testing.T) {
	svc := New(nil, nil, nil)

	checkErr := func(name string, err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("%s: expected error, got nil", name)
		}
		if err.Error() != "auth service not initialized" {
			t.Fatalf("%s: unexpected error %q", name, err.Error())
		}
	}

	_, err := svc.ListRegisterPermits(ListRegisterPermitsRequest{
		SourceID:    7,
		AuthorityID: 11,
		Limit:       10,
	})
	checkErr("list", err)

	_, err = svc.IssueRegisterPermit(IssueRegisterPermitRequest{
		SourceID:    7,
		AuthorityID: 11,
		DeviceID:    "device-1",
		Role:        "admin",
	})
	checkErr("issue", err)

	_, err = svc.RevokeRegisterPermit(RevokeRegisterPermitRequest{
		SourceID:    7,
		AuthorityID: 11,
		Permit:      "permit-1",
	})
	checkErr("revoke", err)
}
