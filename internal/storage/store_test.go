// Context: covers profile-scoped persistence behavior for the Win client store.

package storage

import (
	"path/filepath"
	"testing"
)

func TestNewStoreWithBaseDirUsesOverride(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "mcp-store")

	store, err := NewStoreWithBaseDir(baseDir)
	if err != nil {
		t.Fatalf("NewStoreWithBaseDir() error = %v", err)
	}

	wantBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	if got := store.BaseDir(); got != wantBaseDir {
		t.Fatalf("BaseDir() = %q, want %q", got, wantBaseDir)
	}
	if got := store.SettingsPath(); got != filepath.Join(wantBaseDir, settingsFile) {
		t.Fatalf("SettingsPath() = %q", got)
	}
	if got := store.NodeKeysPath(defaultProfile); got != filepath.Join(wantBaseDir, keysDirName, nodeKeysBaseName) {
		t.Fatalf("NodeKeysPath(default) = %q", got)
	}
}

func TestNewStoreWithBaseDirFallsBackToDefaultWhenEmpty(t *testing.T) {
	base := t.TempDir()
	t.Setenv("APPDATA", base)
	t.Setenv("XDG_CONFIG_HOME", base)
	t.Setenv("HOME", base)

	store, err := NewStoreWithBaseDir("")
	if err != nil {
		t.Fatalf("NewStoreWithBaseDir(empty) error = %v", err)
	}

	if got := store.BaseDir(); got != filepath.Join(base, "fyne", legacyFyneAppID) {
		t.Fatalf("BaseDir() = %q", got)
	}
}
