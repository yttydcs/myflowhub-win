package file

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type FileImportItem struct {
	SourcePath string `json:"sourcePath"`
	Name       string `json:"name"`
	Dir        string `json:"dir"`
	SavedPath  string `json:"savedPath"`
	Size       uint64 `json:"size"`
}

type FileImportFailure struct {
	SourcePath string `json:"sourcePath"`
	Reason     string `json:"reason"`
}

type FileImportResult struct {
	Dir      string              `json:"dir"`
	Imported []FileImportItem    `json:"imported"`
	Skipped  []FileImportFailure `json:"skipped"`
}

func (s *FileService) ImportLocalFiles(targetDir string, sourcePaths []string, overwrite bool) (FileImportResult, error) {
	cfg := s.fileConfig()
	result, err := importLocalFilesToDir(cfg.BaseDir, targetDir, sourcePaths, overwrite)
	if err != nil {
		return result, err
	}
	if s != nil && s.logs != nil {
		s.logs.Appendf(
			"info",
			"file import done dir=%s imported=%d skipped=%d overwrite=%t",
			result.Dir,
			len(result.Imported),
			len(result.Skipped),
			overwrite,
		)
	}
	return result, nil
}

func importLocalFilesToDir(baseDir, targetDir string, sourcePaths []string, overwrite bool) (FileImportResult, error) {
	cleanDir, err := fileSanitizeDir(strings.ReplaceAll(strings.TrimSpace(targetDir), "\\", "/"))
	if err != nil {
		return FileImportResult{}, errors.New("invalid target dir")
	}
	paths := normalizeImportPaths(sourcePaths)
	if len(paths) == 0 {
		return FileImportResult{}, errors.New("source paths required")
	}
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		baseDir = "."
	}
	targetRoot := filepath.Join(baseDir, filepath.FromSlash(cleanDir))
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return FileImportResult{}, fmt.Errorf("prepare target dir failed: %w", err)
	}

	result := FileImportResult{
		Dir:      cleanDir,
		Imported: make([]FileImportItem, 0, len(paths)),
		Skipped:  make([]FileImportFailure, 0),
	}
	for _, sourcePath := range paths {
		item, skipped := importOneLocalFile(baseDir, cleanDir, sourcePath, overwrite)
		if skipped != nil {
			result.Skipped = append(result.Skipped, *skipped)
			continue
		}
		if item != nil {
			result.Imported = append(result.Imported, *item)
		}
	}
	return result, nil
}

func normalizeImportPaths(sourcePaths []string) []string {
	seen := make(map[string]struct{}, len(sourcePaths))
	out := make([]string, 0, len(sourcePaths))
	for _, raw := range sourcePaths {
		path := strings.TrimSpace(raw)
		if path == "" {
			continue
		}
		absPath, err := filepath.Abs(path)
		if err == nil {
			path = absPath
		}
		key := path
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, path)
	}
	return out
}

func importOneLocalFile(baseDir, targetDir, sourcePath string, overwrite bool) (*FileImportItem, *FileImportFailure) {
	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		return nil, &FileImportFailure{Reason: "empty source path"}
	}
	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return nil, &FileImportFailure{SourcePath: sourcePath, Reason: "invalid source path"}
	}
	info, err := os.Stat(absSource)
	if err != nil {
		return nil, &FileImportFailure{SourcePath: absSource, Reason: err.Error()}
	}
	if info.IsDir() {
		return nil, &FileImportFailure{SourcePath: absSource, Reason: "directory is not supported"}
	}
	if !info.Mode().IsRegular() {
		return nil, &FileImportFailure{SourcePath: absSource, Reason: "unsupported file type"}
	}
	name := filepath.Base(absSource)
	if _, err := fileSanitizeName(name); err != nil {
		return nil, &FileImportFailure{SourcePath: absSource, Reason: "invalid file name"}
	}

	finalPath, _, err := fileResolvePaths(baseDir, targetDir, name)
	if err != nil {
		return nil, &FileImportFailure{SourcePath: absSource, Reason: "invalid target path"}
	}
	absFinal, err := filepath.Abs(finalPath)
	if err == nil && sameFilePath(absSource, absFinal) {
		return nil, &FileImportFailure{SourcePath: absSource, Reason: "source and target are the same file"}
	}

	size, err := copyExternalFile(absSource, finalPath, overwrite)
	if err != nil {
		return nil, &FileImportFailure{SourcePath: absSource, Reason: err.Error()}
	}

	return &FileImportItem{
		SourcePath: absSource,
		Name:       name,
		Dir:        targetDir,
		SavedPath:  finalPath,
		Size:       uint64(size),
	}, nil
}

func copyExternalFile(sourcePath, targetPath string, overwrite bool) (int64, error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return 0, fmt.Errorf("open source failed: %w", err)
	}
	defer func() { _ = source.Close() }()

	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return 0, fmt.Errorf("prepare target failed: %w", err)
	}
	if !overwrite {
		if _, err := os.Stat(targetPath); err == nil {
			return 0, errors.New("target already exists")
		} else if !errors.Is(err, os.ErrNotExist) {
			return 0, fmt.Errorf("check target failed: %w", err)
		}
	}

	temp, err := os.CreateTemp(targetDir, ".mfh-import-*")
	if err != nil {
		return 0, fmt.Errorf("create temp failed: %w", err)
	}
	tempPath := temp.Name()
	cleanup := true
	defer func() {
		_ = temp.Close()
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()

	buf := make([]byte, 256*1024)
	written, err := io.CopyBuffer(temp, source, buf)
	if err != nil {
		return 0, fmt.Errorf("copy failed: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return 0, fmt.Errorf("flush temp failed: %w", err)
	}
	if err := temp.Close(); err != nil {
		return 0, fmt.Errorf("close temp failed: %w", err)
	}

	if overwrite {
		if _, err := os.Stat(targetPath); err == nil {
			if err := os.Remove(targetPath); err != nil {
				return 0, fmt.Errorf("replace target failed: %w", err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return 0, fmt.Errorf("check target failed: %w", err)
		}
	}

	if err := os.Rename(tempPath, targetPath); err != nil {
		return 0, fmt.Errorf("commit file failed: %w", err)
	}
	cleanup = false
	return written, nil
}

func sameFilePath(a, b string) bool {
	cleanA := filepath.Clean(strings.TrimSpace(a))
	cleanB := filepath.Clean(strings.TrimSpace(b))
	if runtime.GOOS == "windows" {
		return strings.EqualFold(cleanA, cleanB)
	}
	return cleanA == cleanB
}
