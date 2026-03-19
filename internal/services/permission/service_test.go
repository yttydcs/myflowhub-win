package permission

import (
	"slices"
	"testing"
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
