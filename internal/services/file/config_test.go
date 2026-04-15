// 本文件覆盖 `file` 后端服务中与 `config` 相关的行为。

package file

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRuntimeBaseDirAbsoluteStable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	got := resolveRuntimeBaseDir(dir)
	want := filepath.Clean(dir)
	if got != want {
		t.Fatalf("resolveRuntimeBaseDir(abs) mismatch: got=%q want=%q", got, want)
	}
}

func TestResolveRuntimeBaseDirRelativeUsesExecutableDir(t *testing.T) {
	exeDir := executableDir()
	if exeDir == "" {
		t.Skip("executable dir unavailable (os.Executable not absolute?)")
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()

	want := filepath.Join(exeDir, "file")

	got1 := resolveRuntimeBaseDir("./file")
	if got1 != want {
		t.Fatalf("resolveRuntimeBaseDir(./file) mismatch: got=%q want=%q", got1, want)
	}

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	got2 := resolveRuntimeBaseDir("./file")
	if got2 != want {
		t.Fatalf("resolveRuntimeBaseDir should ignore CWD changes: got=%q want=%q", got2, want)
	}
}
