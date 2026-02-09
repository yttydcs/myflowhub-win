package storage

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	keysDirName       = "config"
	nodeKeysBaseName  = "node_keys.json"
	nodeKeysFilePrefx = "node_keys_"
)

func (s *Store) NodeKeysPath(profile string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nodeKeysPathLocked(profile)
}

func (s *Store) MigrateLegacyNodeKeys(profile string) (bool, error) {
	dest := s.NodeKeysPath(profile)
	if strings.TrimSpace(dest) == "" {
		return false, errors.New("node keys path is empty")
	}
	if fileExists(dest) {
		return false, nil
	}
	candidates := legacyNodeKeysCandidates(profile)
	for _, src := range candidates {
		if !fileExists(src) {
			continue
		}
		if err := copyFile(src, dest); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *Store) MigrateLegacyNodeKeysForProfiles() error {
	profiles, _ := s.Profiles()
	var err error
	for _, profile := range profiles {
		if _, migErr := s.MigrateLegacyNodeKeys(profile); migErr != nil {
			err = migErr
		}
	}
	return err
}

func (s *Store) nodeKeysPathLocked(profile string) string {
	if strings.TrimSpace(s.baseDir) == "" {
		return ""
	}
	filename := nodeKeysBaseName
	if !isDefaultProfile(profile) {
		filename = nodeKeysFilePrefx + sanitizeProfileName(profile) + ".json"
	}
	return filepath.Join(s.baseDir, keysDirName, filename)
}

func legacyNodeKeysCandidates(profile string) []string {
	filename := nodeKeysBaseName
	if !isDefaultProfile(profile) {
		filename = nodeKeysFilePrefx + sanitizeProfileName(profile) + ".json"
	}
	candidates := make([]string, 0, 2)
	if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
		candidates = append(candidates, filepath.Join(cwd, keysDirName, filename))
	}
	if exe, err := os.Executable(); err == nil && strings.TrimSpace(exe) != "" {
		dir := filepath.Dir(exe)
		path := filepath.Join(dir, keysDirName, filename)
		if !containsPath(candidates, path) {
			candidates = append(candidates, path)
		}
	}
	return candidates
}

func copyFile(src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0o600)
}

func sanitizeProfileName(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	out := strings.Trim(re.ReplaceAllString(name, "_"), "_")
	if out == "" {
		return defaultProfile
	}
	return out
}

func isDefaultProfile(name string) bool {
	name = strings.TrimSpace(name)
	return name == "" || name == defaultProfile
}

func containsPath(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
