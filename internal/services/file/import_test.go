// Context: covers the import helper logic inside the file backend service.

package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportLocalFilesToDir_ImportAndSkip(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := t.TempDir()

	sourceFile := filepath.Join(sourceDir, "alpha.txt")
	if err := os.WriteFile(sourceFile, []byte("alpha"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	sourceFolder := filepath.Join(sourceDir, "folder")
	if err := os.MkdirAll(sourceFolder, 0o755); err != nil {
		t.Fatalf("mkdir source folder: %v", err)
	}

	result, err := importLocalFilesToDir(baseDir, "", []string{sourceFile, sourceFolder}, false)
	if err != nil {
		t.Fatalf("importLocalFilesToDir() error = %v", err)
	}
	if len(result.Imported) != 1 {
		t.Fatalf("imported count = %d, want 1", len(result.Imported))
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("skipped count = %d, want 1", len(result.Skipped))
	}
	if !strings.Contains(result.Skipped[0].Reason, "directory") {
		t.Fatalf("skip reason = %q, want contains %q", result.Skipped[0].Reason, "directory")
	}

	targetFile := filepath.Join(baseDir, "alpha.txt")
	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("read imported file: %v", err)
	}
	if string(data) != "alpha" {
		t.Fatalf("imported content = %q, want %q", string(data), "alpha")
	}
}

func TestImportLocalFilesToDir_SkipExistingWhenNoOverwrite(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := t.TempDir()

	targetFile := filepath.Join(baseDir, "dup.txt")
	if err := os.WriteFile(targetFile, []byte("old"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	sourceFile := filepath.Join(sourceDir, "dup.txt")
	if err := os.WriteFile(sourceFile, []byte("new"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	result, err := importLocalFilesToDir(baseDir, "", []string{sourceFile}, false)
	if err != nil {
		t.Fatalf("importLocalFilesToDir() error = %v", err)
	}
	if len(result.Imported) != 0 {
		t.Fatalf("imported count = %d, want 0", len(result.Imported))
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("skipped count = %d, want 1", len(result.Skipped))
	}
	if !strings.Contains(result.Skipped[0].Reason, "exists") {
		t.Fatalf("skip reason = %q, want contains %q", result.Skipped[0].Reason, "exists")
	}

	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("read target file: %v", err)
	}
	if string(data) != "old" {
		t.Fatalf("target content = %q, want %q", string(data), "old")
	}
}

func TestImportLocalFilesToDir_OverwriteExisting(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := t.TempDir()

	targetFile := filepath.Join(baseDir, "dup.txt")
	if err := os.WriteFile(targetFile, []byte("old"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	sourceFile := filepath.Join(sourceDir, "dup.txt")
	if err := os.WriteFile(sourceFile, []byte("new"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	result, err := importLocalFilesToDir(baseDir, "", []string{sourceFile}, true)
	if err != nil {
		t.Fatalf("importLocalFilesToDir() error = %v", err)
	}
	if len(result.Imported) != 1 {
		t.Fatalf("imported count = %d, want 1", len(result.Imported))
	}
	if len(result.Skipped) != 0 {
		t.Fatalf("skipped count = %d, want 0", len(result.Skipped))
	}

	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("read target file: %v", err)
	}
	if string(data) != "new" {
		t.Fatalf("target content = %q, want %q", string(data), "new")
	}
}

func TestImportLocalFilesToDir_InvalidTargetDir(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "alpha.txt")
	if err := os.WriteFile(sourceFile, []byte("alpha"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	_, err := importLocalFilesToDir(baseDir, "../escape", []string{sourceFile}, false)
	if err == nil {
		t.Fatal("importLocalFilesToDir() error = nil, want invalid target dir error")
	}
}
