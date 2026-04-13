package localhub

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildLaunchSpecInjectsApprovalJoinEnv(t *testing.T) {
	cfg := Config{
		Host:               "127.0.0.1",
		Port:               9000,
		NodeID:             7,
		SelfID:             " edge-hub-01 ",
		Parent:             " authority.example:9000 ",
		ParentEnable:       true,
		ParentReconnectSec: 5,
		AuthDefaultRole:    "admin",
		AuthDefaultPerms:   "file.read,file.write",
		AuthNodeRoles:      "7:admin",
		AuthRolePerms:      "admin:file.read,file.write",
		ExtraArgs:          "-debug\n# ignored\n-trace",
	}

	spec, err := buildLaunchSpec(
		[]string{"PATH=/tmp/runtime"},
		cfg,
		LaunchRequest{ParentJoinPermit: " permit_123 "},
		"127.0.0.1:9000",
	)
	if err != nil {
		t.Fatalf("buildLaunchSpec returned error: %v", err)
	}

	wantArgs := []string{
		"-addr", "127.0.0.1:9000",
		"-node-id", "7",
		"-parent", "authority.example:9000",
		"-parent-enable=true",
		"-parent-reconnect", "5",
		"-auth-default-role", "admin",
		"-auth-default-perms", "file.read,file.write",
		"-auth-node-roles", "7:admin",
		"-auth-role-perms", "admin:file.read,file.write",
		"-debug",
		"-trace",
	}
	if !reflect.DeepEqual(spec.Args, wantArgs) {
		t.Fatalf("unexpected args:\nwant: %#v\ngot:  %#v", wantArgs, spec.Args)
	}

	if got := envValue(spec.Env, envHubSelfID); got != "edge-hub-01" {
		t.Fatalf("unexpected %s: %q", envHubSelfID, got)
	}
	if got := envValue(spec.Env, envHubParentJoinPermit); got != "permit_123" {
		t.Fatalf("unexpected %s: %q", envHubParentJoinPermit, got)
	}
}

func TestBuildLaunchSpecRejectsPermitWithoutSelfID(t *testing.T) {
	_, err := buildLaunchSpec(
		nil,
		Config{
			NodeID:             1,
			Parent:             "authority.example:9000",
			ParentEnable:       true,
			ParentReconnectSec: 3,
		},
		LaunchRequest{ParentJoinPermit: "permit_123"},
		"127.0.0.1:9000",
	)
	if err == nil || err.Error() != "self ID is required when parent join permit is provided" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildLaunchSpecRejectsPermitWithoutParentLink(t *testing.T) {
	_, err := buildLaunchSpec(
		nil,
		Config{
			NodeID:             1,
			SelfID:             "edge-hub-01",
			Parent:             "authority.example:9000",
			ParentEnable:       false,
			ParentReconnectSec: 3,
		},
		LaunchRequest{ParentJoinPermit: "permit_123"},
		"127.0.0.1:9000",
	)
	if err == nil || err.Error() != "enable parent link before using a parent join permit" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildLaunchSpecClearsStaleLaunchEnvWhenBlank(t *testing.T) {
	spec, err := buildLaunchSpec(
		[]string{
			"PATH=/tmp/runtime",
			"HUB_SELF_ID=old-self",
			"hub_parent_join_permit=old-permit",
		},
		Config{
			NodeID:             1,
			ParentEnable:       false,
			ParentReconnectSec: 3,
		},
		LaunchRequest{},
		"127.0.0.1:9000",
	)
	if err != nil {
		t.Fatalf("buildLaunchSpec returned error: %v", err)
	}

	if got := envValue(spec.Env, envHubSelfID); got != "" {
		t.Fatalf("expected blank %s, got %q", envHubSelfID, got)
	}
	if got := envValue(spec.Env, envHubParentJoinPermit); got != "" {
		t.Fatalf("expected blank %s, got %q", envHubParentJoinPermit, got)
	}
}

func envValue(env []string, key string) string {
	for _, entry := range env {
		name, value, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		if strings.EqualFold(name, key) {
			return value
		}
	}
	return ""
}
