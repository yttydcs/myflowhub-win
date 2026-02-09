package main

import (
	"encoding/json"
	"errors"
	"strings"
)

const varpoolNamesKey = "varpool.names"

type VarPoolKey struct {
	Name  string `json:"name"`
	Owner uint32 `json:"owner,omitempty"`
}

func (a *App) VarPoolWatchList() ([]VarPoolKey, error) {
	if a.store == nil {
		return nil, errors.New("storage not initialized")
	}
	profile := a.store.CurrentProfile()
	raw := a.store.GetString(profile, varpoolNamesKey, "")
	keys := normalizeVarPoolKeys(parseVarPoolKeys(raw))
	return keys, nil
}

func (a *App) SaveVarPoolWatchList(keys []VarPoolKey) ([]VarPoolKey, error) {
	if a.store == nil {
		return nil, errors.New("storage not initialized")
	}
	normalized := normalizeVarPoolKeys(keys)
	data, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	profile := a.store.CurrentProfile()
	if err := a.store.SetString(profile, varpoolNamesKey, string(data)); err != nil {
		return nil, err
	}
	return normalized, nil
}

func parseVarPoolKeys(raw string) []VarPoolKey {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var keys []VarPoolKey
	if err := json.Unmarshal([]byte(raw), &keys); err == nil {
		return keys
	}
	var names []string
	if err := json.Unmarshal([]byte(raw), &names); err != nil {
		return nil
	}
	keys = make([]VarPoolKey, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		keys = append(keys, VarPoolKey{Name: name})
	}
	return keys
}

func normalizeVarPoolKeys(keys []VarPoolKey) []VarPoolKey {
	out := make([]VarPoolKey, 0, len(keys))
	for _, key := range keys {
		key = normalizeVarPoolKey(key)
		if key.Name == "" {
			continue
		}
		replaced := false
		for i, existing := range out {
			if existing == key {
				replaced = true
				break
			}
			if existing.Name == key.Name && existing.Owner == 0 && key.Owner != 0 {
				out[i] = key
				replaced = true
				break
			}
		}
		if !replaced {
			out = append(out, key)
		}
	}
	return out
}

func normalizeVarPoolKey(key VarPoolKey) VarPoolKey {
	key.Name = strings.TrimSpace(key.Name)
	return key
}
