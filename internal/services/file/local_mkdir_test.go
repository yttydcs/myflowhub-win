package file

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestMkdirInBase_CreateSuccess(t *testing.T) {
	baseDir := t.TempDir()
	if err := mkdirInBase(baseDir, "", "logs"); err != nil {
		t.Fatalf("mkdirInBase() error = %v", err)
	}
	info, err := os.Stat(filepath.Join(baseDir, "logs"))
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("created path is not a directory")
	}
}

func TestMkdirInBase_IdempotentWhenDirExists(t *testing.T) {
	baseDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(baseDir, "cache"), 0o755); err != nil {
		t.Fatalf("prepare dir: %v", err)
	}
	if err := mkdirInBase(baseDir, "", "cache"); err != nil {
		t.Fatalf("mkdirInBase() error = %v", err)
	}
}

func TestMkdirInBase_FileConflict(t *testing.T) {
	baseDir := t.TempDir()
	target := filepath.Join(baseDir, "conflict")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("prepare file: %v", err)
	}
	err := mkdirInBase(baseDir, "", "conflict")
	if err == nil {
		t.Fatal("mkdirInBase() error = nil, want exists")
	}
	if err.Error() != "exists" {
		t.Fatalf("mkdirInBase() error = %q, want %q", err.Error(), "exists")
	}
}

func TestMkdirInBase_InvalidInput(t *testing.T) {
	baseDir := t.TempDir()

	err := mkdirInBase(baseDir, "../escape", "x")
	if !errors.Is(err, errFileInvalidDir) {
		t.Fatalf("mkdirInBase() dir error = %v, want errFileInvalidDir", err)
	}

	err = mkdirInBase(baseDir, "", "bad/name")
	if !errors.Is(err, errFileInvalidName) {
		t.Fatalf("mkdirInBase() name error = %v, want errFileInvalidName", err)
	}
}
