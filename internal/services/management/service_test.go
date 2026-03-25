package management

import (
	"testing"

	storagesvc "github.com/yttydcs/myflowhub-win/internal/storage"
)

func newTestStore(t *testing.T) *storagesvc.Store {
	t.Helper()
	base := t.TempDir()
	t.Setenv("APPDATA", base)
	t.Setenv("XDG_CONFIG_HOME", base)
	t.Setenv("HOME", base)

	store, err := storagesvc.NewStore()
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	return store
}

func TestNodeInfoSimpleSelfIncludesRawDisplayName(t *testing.T) {
	store := newTestStore(t)
	if err := store.SetRaw(nodeDisplayNameKey, "  raw-self  "); err != nil {
		t.Fatalf("SetRaw() error = %v", err)
	}
	if err := store.SetCurrentProfile("work"); err != nil {
		t.Fatalf("SetCurrentProfile() error = %v", err)
	}
	if err := store.SetString("work", nodeDisplayNameKey, "profile-self"); err != nil {
		t.Fatalf("SetString() error = %v", err)
	}

	svc := New(nil, nil, store)
	resp, err := svc.NodeInfoSimple(7, 7)
	if err != nil {
		t.Fatalf("NodeInfoSimple() error = %v", err)
	}

	if got := resp.Items["display_name"]; got != "raw-self" {
		t.Fatalf("expected raw display_name, got %q", got)
	}
}

func TestNodeInfoSimpleSelfPrefersMCPDisplayName(t *testing.T) {
	store := newTestStore(t)
	if err := store.SetRaw(mcpDisplayNameKey, "  mcp-self  "); err != nil {
		t.Fatalf("SetRaw(mcp) error = %v", err)
	}
	if err := store.SetRaw(nodeDisplayNameKey, "legacy-self"); err != nil {
		t.Fatalf("SetRaw(legacy) error = %v", err)
	}

	svc := New(nil, nil, store)
	resp, err := svc.NodeInfoSimple(8, 8)
	if err != nil {
		t.Fatalf("NodeInfoSimple() error = %v", err)
	}

	if got := resp.Items["display_name"]; got != "mcp-self" {
		t.Fatalf("expected mcp display_name, got %q", got)
	}
}

func TestNodeInfoSimpleSelfFallsBackToProfileDisplayName(t *testing.T) {
	store := newTestStore(t)
	if err := store.SetCurrentProfile("work"); err != nil {
		t.Fatalf("SetCurrentProfile() error = %v", err)
	}
	if err := store.SetString("work", nodeDisplayNameKey, "  profile-self  "); err != nil {
		t.Fatalf("SetString() error = %v", err)
	}

	svc := New(nil, nil, store)
	resp, err := svc.NodeInfoSimple(9, 9)
	if err != nil {
		t.Fatalf("NodeInfoSimple() error = %v", err)
	}

	if got := resp.Items["display_name"]; got != "profile-self" {
		t.Fatalf("expected profile display_name, got %q", got)
	}
}
