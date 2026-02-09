package file

import (
	"errors"
	"strings"
	"time"
)

const (
	cfgFileBaseDir          = "file.base_dir"
	cfgFileMaxSizeBytes     = "file.max_size_bytes"
	cfgFileMaxConcurrent    = "file.max_concurrent"
	cfgFileChunkBytes       = "file.chunk_bytes"
	cfgFileIncompleteTTLSec = "file.incomplete_ttl_sec"
	cfgFileWantSHA256       = "file.want_sha256"
	cfgFileAutoAccept       = "file.auto_accept"

	cfgFileBrowserNodes = "file.browser.nodes"
)

type fileConfig struct {
	BaseDir       string
	MaxSizeBytes  uint64
	MaxConcurrent int
	ChunkBytes    int
	WantSHA256    bool
	AutoAccept    bool
	IncompleteTTL time.Duration

	AckEveryBytes uint64
	AckEvery      time.Duration
}

func defaultFilePrefs() FilePrefs {
	return FilePrefs{
		BaseDir:          "./file",
		MaxSizeBytes:     0,
		MaxConcurrent:    4,
		ChunkBytes:       256 * 1024,
		IncompleteTTLSec: 3600,
		WantSHA256:       true,
		AutoAccept:       false,
	}
}

func (s *FileService) fileConfig() fileConfig {
	prefs := s.loadFilePrefs()
	if prefs.MaxConcurrent <= 0 {
		prefs.MaxConcurrent = defaultFilePrefs().MaxConcurrent
	}
	if prefs.ChunkBytes <= 0 {
		prefs.ChunkBytes = defaultFilePrefs().ChunkBytes
	}
	if prefs.IncompleteTTLSec <= 0 {
		prefs.IncompleteTTLSec = defaultFilePrefs().IncompleteTTLSec
	}
	if strings.TrimSpace(prefs.BaseDir) == "" {
		prefs.BaseDir = defaultFilePrefs().BaseDir
	}
	return fileConfig{
		BaseDir:       prefs.BaseDir,
		MaxSizeBytes:  prefs.MaxSizeBytes,
		MaxConcurrent: prefs.MaxConcurrent,
		ChunkBytes:    prefs.ChunkBytes,
		WantSHA256:    prefs.WantSHA256,
		AutoAccept:    prefs.AutoAccept,
		IncompleteTTL: time.Duration(prefs.IncompleteTTLSec) * time.Second,
		AckEveryBytes: 512 * 1024,
		AckEvery:      500 * time.Millisecond,
	}
}

func (s *FileService) Prefs() (FilePrefs, error) {
	return s.loadFilePrefs(), nil
}

func (s *FileService) SavePrefs(prefs FilePrefs) (FilePrefs, error) {
	if s == nil || s.store == nil {
		return FilePrefs{}, errors.New("storage not initialized")
	}
	normalized, err := normalizePrefs(prefs)
	if err != nil {
		return FilePrefs{}, err
	}
	profile := s.store.CurrentProfile()
	if err := s.store.SetString(profile, cfgFileBaseDir, normalized.BaseDir); err != nil {
		return FilePrefs{}, err
	}
	if err := s.store.SetInt(profile, cfgFileMaxSizeBytes, int(normalized.MaxSizeBytes)); err != nil {
		return FilePrefs{}, err
	}
	if err := s.store.SetInt(profile, cfgFileMaxConcurrent, normalized.MaxConcurrent); err != nil {
		return FilePrefs{}, err
	}
	if err := s.store.SetInt(profile, cfgFileChunkBytes, normalized.ChunkBytes); err != nil {
		return FilePrefs{}, err
	}
	if err := s.store.SetInt(profile, cfgFileIncompleteTTLSec, int(normalized.IncompleteTTLSec)); err != nil {
		return FilePrefs{}, err
	}
	if err := s.store.SetBool(profile, cfgFileWantSHA256, normalized.WantSHA256); err != nil {
		return FilePrefs{}, err
	}
	if err := s.store.SetBool(profile, cfgFileAutoAccept, normalized.AutoAccept); err != nil {
		return FilePrefs{}, err
	}
	return normalized, nil
}

func normalizePrefs(prefs FilePrefs) (FilePrefs, error) {
	defaults := defaultFilePrefs()
	prefs.BaseDir = strings.TrimSpace(prefs.BaseDir)
	if prefs.BaseDir == "" {
		prefs.BaseDir = defaults.BaseDir
	}
	if prefs.MaxConcurrent <= 0 {
		prefs.MaxConcurrent = defaults.MaxConcurrent
	}
	if prefs.ChunkBytes <= 0 {
		prefs.ChunkBytes = defaults.ChunkBytes
	}
	if prefs.IncompleteTTLSec <= 0 {
		prefs.IncompleteTTLSec = defaults.IncompleteTTLSec
	}
	return prefs, nil
}

func (s *FileService) loadFilePrefs() FilePrefs {
	if s == nil || s.store == nil {
		return defaultFilePrefs()
	}
	profile := s.store.CurrentProfile()
	baseDir := strings.TrimSpace(s.store.GetString(profile, cfgFileBaseDir, defaultFilePrefs().BaseDir))
	maxSize := s.store.GetInt(profile, cfgFileMaxSizeBytes, int(defaultFilePrefs().MaxSizeBytes))
	maxConcurrent := s.store.GetInt(profile, cfgFileMaxConcurrent, defaultFilePrefs().MaxConcurrent)
	chunkBytes := s.store.GetInt(profile, cfgFileChunkBytes, defaultFilePrefs().ChunkBytes)
	ttlSec := s.store.GetInt(profile, cfgFileIncompleteTTLSec, int(defaultFilePrefs().IncompleteTTLSec))
	wantSHA := s.store.GetBool(profile, cfgFileWantSHA256, defaultFilePrefs().WantSHA256)
	autoAccept := s.store.GetBool(profile, cfgFileAutoAccept, defaultFilePrefs().AutoAccept)
	if maxSize < 0 {
		maxSize = int(defaultFilePrefs().MaxSizeBytes)
	}
	if maxConcurrent <= 0 {
		maxConcurrent = defaultFilePrefs().MaxConcurrent
	}
	if chunkBytes <= 0 {
		chunkBytes = defaultFilePrefs().ChunkBytes
	}
	if ttlSec <= 0 {
		ttlSec = int(defaultFilePrefs().IncompleteTTLSec)
	}
	if baseDir == "" {
		baseDir = defaultFilePrefs().BaseDir
	}
	return FilePrefs{
		BaseDir:          baseDir,
		MaxSizeBytes:     uint64(maxSize),
		MaxConcurrent:    maxConcurrent,
		ChunkBytes:       chunkBytes,
		IncompleteTTLSec: int64(ttlSec),
		WantSHA256:       wantSHA,
		AutoAccept:       autoAccept,
	}
}

func (s *FileService) BrowserNodes() ([]uint32, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("storage not initialized")
	}
	profile := s.store.CurrentProfile()
	raw := s.store.GetString(profile, cfgFileBrowserNodes, "")
	return parseNodeList(raw), nil
}

func (s *FileService) SaveBrowserNodes(nodes []uint32) ([]uint32, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("storage not initialized")
	}
	normalized := normalizeNodes(nodes)
	profile := s.store.CurrentProfile()
	if err := s.store.SetString(profile, cfgFileBrowserNodes, nodesToJSON(normalized)); err != nil {
		return nil, err
	}
	return normalized, nil
}

func parseNodeList(raw string) []uint32 {
	return parseUint32List(raw)
}

func normalizeNodes(nodes []uint32) []uint32 {
	seen := make(map[uint32]struct{}, len(nodes))
	out := make([]uint32, 0, len(nodes))
	for _, node := range nodes {
		if node == 0 {
			continue
		}
		if _, ok := seen[node]; ok {
			continue
		}
		seen[node] = struct{}{}
		out = append(out, node)
	}
	return out
}

func nodesToJSON(nodes []uint32) string {
	return encodeUint32List(nodes)
}
