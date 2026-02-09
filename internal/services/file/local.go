package file

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

func (s *FileService) localList(dir string) (dirs []string, files []string, err error) {
	cfg := s.fileConfig()
	dir = strings.ReplaceAll(strings.TrimSpace(dir), "\\", "/")
	clean, derr := fileSanitizeDir(dir)
	if derr != nil {
		return nil, nil, derr
	}
	root := filepath.Join(cfg.BaseDir, filepath.FromSlash(clean))
	entries, rerr := os.ReadDir(root)
	if rerr != nil {
		return nil, nil, rerr
	}
	dirs = make([]string, 0, len(entries))
	files = make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(dirs)
	sort.Strings(files)
	return dirs, files, nil
}

func (s *FileService) localReadText(dir, name string, maxBytes int) (text string, truncated bool, size uint64, err error) {
	cfg := s.fileConfig()
	dir = strings.ReplaceAll(strings.TrimSpace(dir), "\\", "/")
	name = strings.TrimSpace(name)
	finalPath, _, rerr := fileResolvePaths(cfg.BaseDir, dir, name)
	if rerr != nil {
		return "", false, 0, rerr
	}
	info, rerr := os.Stat(finalPath)
	if rerr != nil || info == nil || info.IsDir() {
		return "", false, 0, errors.New("not found")
	}
	size = uint64(info.Size())
	if maxBytes <= 0 {
		maxBytes = 64 * 1024
	}
	if maxBytes > 256*1024 {
		maxBytes = 256 * 1024
	}
	f, rerr := os.Open(finalPath)
	if rerr != nil {
		return "", false, size, rerr
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, maxBytes)
	n, readErr := io.ReadFull(f, buf)
	if readErr == io.ErrUnexpectedEOF || readErr == io.EOF {
		// ok
	} else if readErr != nil {
		return "", false, size, readErr
	}
	buf = buf[:n]
	truncated = uint64(n) < size
	if !utf8.Valid(buf) {
		return "", truncated, size, errors.New("not text")
	}
	return string(buf), truncated, size, nil
}
