// Context: persists the VarPool watch list and subscription preferences for the Win frontend.

package main

import (
	"encoding/json"
	"errors"
	"strings"
)

const (
	varpoolNamesKey    = "varpool.names"
	varpoolSubPrefsKey = "varpool.sub_prefs"
)

type VarPoolKey struct {
	Name  string `json:"name"`
	Owner uint32 `json:"owner,omitempty"`
}

type VarPoolSubPref struct {
	Name       string `json:"name"`
	Owner      uint32 `json:"owner"`
	Subscribed bool   `json:"subscribed"`
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

func (a *App) VarPoolSubPrefs() ([]VarPoolSubPref, error) {
	if a.store == nil {
		return nil, errors.New("storage not initialized")
	}
	profile := a.store.CurrentProfile()
	raw := a.store.GetString(profile, varpoolSubPrefsKey, "")
	prefs := normalizeVarPoolSubPrefs(parseVarPoolSubPrefs(raw))
	return prefs, nil
}

func (a *App) SaveVarPoolSubPrefs(prefs []VarPoolSubPref) ([]VarPoolSubPref, error) {
	if a.store == nil {
		return nil, errors.New("storage not initialized")
	}
	normalized := normalizeVarPoolSubPrefs(prefs)
	data, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	profile := a.store.CurrentProfile()
	if err := a.store.SetString(profile, varpoolSubPrefsKey, string(data)); err != nil {
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

func parseVarPoolSubPrefs(raw string) []VarPoolSubPref {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var prefs []VarPoolSubPref
	if err := json.Unmarshal([]byte(raw), &prefs); err == nil {
		return prefs
	}
	return nil
}

func normalizeVarPoolSubPrefs(prefs []VarPoolSubPref) []VarPoolSubPref {
	out := make([]VarPoolSubPref, 0, len(prefs))
	type prefKey struct {
		name  string
		owner uint32
	}
	seen := make(map[prefKey]int, len(prefs))
	for _, pref := range prefs {
		pref = normalizeVarPoolSubPref(pref)
		if pref.Name == "" || pref.Owner == 0 {
			continue
		}
		key := prefKey{name: pref.Name, owner: pref.Owner}
		if idx, ok := seen[key]; ok {
			out[idx] = pref
			continue
		}
		seen[key] = len(out)
		out = append(out, pref)
	}
	return out
}

func normalizeVarPoolSubPref(pref VarPoolSubPref) VarPoolSubPref {
	pref.Name = strings.TrimSpace(pref.Name)
	return pref
}
