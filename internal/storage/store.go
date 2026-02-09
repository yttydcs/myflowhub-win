package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

const (
	legacyFyneAppID   = "myflowhub.debugclient"
	legacyPrefsFile   = "preferences.json"
	settingsFile      = "settings.json"
	defaultProfile    = "default"
	prefProfilesList  = "profiles.list"
	prefProfilesLast  = "profiles.last"
	maxProfileNameLen = 64
)

type ProfileState struct {
	Profiles     []string `json:"profiles"`
	Current      string   `json:"current"`
	BaseDir      string   `json:"baseDir"`
	SettingsPath string   `json:"settingsPath"`
	KeysPath     string   `json:"keysPath"`
}

type Store struct {
	mu         sync.RWMutex
	baseDir    string
	path       string
	legacyPath string
	values     map[string]any
}

func NewStore() (*Store, error) {
	baseDir, err := resolveBaseDir()
	if err != nil {
		return nil, err
	}
	store := &Store{
		baseDir:    baseDir,
		path:       filepath.Join(baseDir, settingsFile),
		legacyPath: filepath.Join(baseDir, legacyPrefsFile),
		values:     map[string]any{},
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) BaseDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.baseDir
}

func (s *Store) SettingsPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.path
}

func (s *Store) LegacyPreferencesPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.legacyPath
}

func (s *Store) State() ProfileState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profiles, current := profileStateFromValues(s.values)
	if current == "" {
		current = defaultProfile
	}
	return ProfileState{
		Profiles:     profiles,
		Current:      current,
		BaseDir:      s.baseDir,
		SettingsPath: s.path,
		KeysPath:     s.nodeKeysPathLocked(current),
	}
}

func (s *Store) Profiles() ([]string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return profileStateFromValues(s.values)
}

func (s *Store) CurrentProfile() string {
	_, current := s.Profiles()
	if current == "" {
		return defaultProfile
	}
	return current
}

func (s *Store) SetCurrentProfile(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("profile name is required")
	}
	if len(name) > maxProfileNameLen {
		return errors.New("profile name too long")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	list, _ := profileStateFromValues(s.values)
	list = normalizeProfileList(list)
	if !contains(list, name) {
		list = append(list, name)
	}
	return s.saveProfilesLocked(list, name)
}

func (s *Store) ProfileKey(profile, key string) string {
	if strings.TrimSpace(profile) == "" || profile == defaultProfile {
		return key
	}
	return profile + "." + key
}

func (s *Store) GetString(profile, key, fallback string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.values[s.ProfileKey(profile, key)]
	if !ok {
		return fallback
	}
	if out, ok := asString(val); ok {
		return out
	}
	return fallback
}

func (s *Store) GetBool(profile, key string, fallback bool) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.values[s.ProfileKey(profile, key)]
	if !ok {
		return fallback
	}
	if out, ok := asBool(val); ok {
		return out
	}
	return fallback
}

func (s *Store) GetInt(profile, key string, fallback int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.values[s.ProfileKey(profile, key)]
	if !ok {
		return fallback
	}
	if out, ok := asInt(val); ok {
		return out
	}
	return fallback
}

func (s *Store) SetString(profile, key, value string) error {
	return s.setValue(profile, key, value)
}

func (s *Store) SetBool(profile, key string, value bool) error {
	return s.setValue(profile, key, value)
}

func (s *Store) SetInt(profile, key string, value int) error {
	return s.setValue(profile, key, value)
}

func (s *Store) setValue(profile, key string, value any) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("key is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		s.values = map[string]any{}
	}
	s.values[s.ProfileKey(profile, key)] = value
	return s.saveLocked()
}

func (s *Store) saveProfilesLocked(list []string, current string) error {
	list = normalizeProfileList(list)
	if len(list) == 0 {
		list = []string{defaultProfile}
	}
	if current == "" || !contains(list, current) {
		current = list[0]
	}
	data, _ := json.Marshal(list)
	if s.values == nil {
		s.values = map[string]any{}
	}
	s.values[prefProfilesList] = string(data)
	s.values[prefProfilesLast] = current
	return s.saveLocked()
}

func (s *Store) load() error {
	values := map[string]any{}
	newExists := fileExists(s.path)
	legacyExists := fileExists(s.legacyPath)

	if newExists {
		loaded, err := readValues(s.path)
		if err != nil {
			return err
		}
		values = loaded
	}
	if legacyExists {
		legacyValues, err := readValues(s.legacyPath)
		if err != nil {
			return err
		}
		if !newExists {
			values = legacyValues
		} else if mergeMissing(values, legacyValues) {
			newExists = false
		}
	}
	normalized := normalizeProfiles(values)
	s.values = values
	if !newExists || normalized {
		return s.saveLocked()
	}
	return nil
}

func (s *Store) saveLocked() error {
	if strings.TrimSpace(s.path) == "" {
		return errors.New("settings path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	return writeValues(s.path, s.values)
}

func resolveBaseDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(dir) == "" {
		home, _ := os.UserHomeDir()
		if strings.TrimSpace(home) == "" {
			return "", errors.New("user config dir unavailable")
		}
		if runtime.GOOS == "windows" {
			dir = filepath.Join(home, "AppData", "Roaming")
		} else {
			dir = filepath.Join(home, ".config")
		}
	}
	return filepath.Join(dir, "fyne", legacyFyneAppID), nil
}

func profileStateFromValues(values map[string]any) ([]string, string) {
	list := parseProfileList(values[prefProfilesList])
	list = normalizeProfileList(list)
	current, _ := asString(values[prefProfilesLast])
	if current == "" || !contains(list, current) {
		if len(list) > 0 {
			current = list[0]
		}
	}
	return list, current
}

func normalizeProfiles(values map[string]any) bool {
	if values == nil {
		return false
	}
	list := parseProfileList(values[prefProfilesList])
	list = normalizeProfileList(list)
	if len(list) == 0 {
		list = []string{defaultProfile}
	}
	current, _ := asString(values[prefProfilesLast])
	if current == "" || !contains(list, current) {
		current = list[0]
	}
	data, _ := json.Marshal(list)
	changed := false
	if values[prefProfilesList] != string(data) {
		values[prefProfilesList] = string(data)
		changed = true
	}
	if values[prefProfilesLast] != current {
		values[prefProfilesLast] = current
		changed = true
	}
	return changed
}

func normalizeProfileList(list []string) []string {
	out := make([]string, 0, len(list))
	seen := map[string]struct{}{}
	for _, item := range list {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	if !contains(out, defaultProfile) {
		out = append(out, defaultProfile)
	}
	return out
}

func parseProfileList(raw any) []string {
	switch v := raw.(type) {
	case string:
		var list []string
		if err := json.Unmarshal([]byte(v), &list); err == nil {
			return list
		}
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := asString(item); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func asString(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case json.Number:
		return v.String(), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), true
	case int:
		return strconv.Itoa(v), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case uint64:
		return strconv.FormatUint(v, 10), true
	case []byte:
		return string(v), true
	default:
		return "", false
	}
}

func asBool(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return false, false
		}
		return i != 0, true
	case float64:
		return v != 0, true
	case int:
		return v != 0, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}

func asInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func readValues(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]any{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var values map[string]any
	if err := dec.Decode(&values); err != nil {
		return nil, err
	}
	if values == nil {
		values = map[string]any{}
	}
	return values, nil
}

func writeValues(path string, values map[string]any) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "settings-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(values); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(path)
		if err := os.Rename(tmpPath, path); err != nil {
			_ = os.Remove(tmpPath)
			return err
		}
	}
	return os.Chmod(path, 0o600)
}

func mergeMissing(dst, src map[string]any) bool {
	if len(src) == 0 {
		return false
	}
	changed := false
	for k, v := range src {
		if _, ok := dst[k]; ok {
			continue
		}
		dst[k] = v
		changed = true
	}
	return changed
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
