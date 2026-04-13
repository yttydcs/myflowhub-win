package main

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	streamsvc "github.com/yttydcs/myflowhub-win/internal/services/stream"
)

const streamPrefsKey = "stream.prefs"

type StreamSavedSource struct {
	SourceID    string   `json:"sourceId"`
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	ContentType string   `json:"contentType"`
	Mode        string   `json:"mode"`
	UnitMode    string   `json:"unitMode"`
	Tags        []string `json:"tags"`
	MetadataRaw string   `json:"metadataRaw"`
	InputKind   string   `json:"inputKind,omitempty"`
	FilePath    string   `json:"filePath,omitempty"`
}

type StreamSavedConsumer struct {
	ConsumerID  string   `json:"consumerId"`
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	ContentType string   `json:"contentType"`
	Tags        []string `json:"tags"`
	MetadataRaw string   `json:"metadataRaw"`
}

type StreamPrefs struct {
	ActiveTab string                `json:"activeTab"`
	TargetID  int                   `json:"targetId"`
	Sources   []StreamSavedSource   `json:"sources"`
	Consumers []StreamSavedConsumer `json:"consumers"`
}

func (a *App) StreamPrefs() (StreamPrefs, error) {
	if a.store == nil {
		return StreamPrefs{}, errors.New("storage not initialized")
	}
	profile := a.store.CurrentProfile()
	raw := a.store.GetString(profile, streamPrefsKey, "")
	return normalizeStreamPrefs(parseStreamPrefs(raw)), nil
}

func (a *App) SaveStreamPrefs(prefs StreamPrefs) (StreamPrefs, error) {
	if a.store == nil {
		return StreamPrefs{}, errors.New("storage not initialized")
	}
	normalized := normalizeStreamPrefs(prefs)
	data, err := json.Marshal(normalized)
	if err != nil {
		return StreamPrefs{}, err
	}
	profile := a.store.CurrentProfile()
	if err := a.store.SetString(profile, streamPrefsKey, string(data)); err != nil {
		return StreamPrefs{}, err
	}
	return normalized, nil
}

func parseStreamPrefs(raw string) StreamPrefs {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return StreamPrefs{}
	}
	var prefs StreamPrefs
	if err := json.Unmarshal([]byte(raw), &prefs); err != nil {
		return StreamPrefs{}
	}
	return prefs
}

func normalizeStreamPrefs(prefs StreamPrefs) StreamPrefs {
	targetID := prefs.TargetID
	if targetID < 0 {
		targetID = 0
	}
	return StreamPrefs{
		ActiveTab: normalizeStreamTab(prefs.ActiveTab),
		TargetID:  targetID,
		Sources:   normalizeStreamSavedSources(prefs.Sources),
		Consumers: normalizeStreamSavedConsumers(prefs.Consumers),
	}
}

func normalizeStreamTab(tab string) string {
	switch strings.ToLower(strings.TrimSpace(tab)) {
	case "source", "consumer", "control":
		return strings.ToLower(strings.TrimSpace(tab))
	default:
		return "source"
	}
}

func normalizeStreamSavedSources(items []StreamSavedSource) []StreamSavedSource {
	out := make([]StreamSavedSource, 0, len(items))
	seen := make(map[string]int, len(items))
	for _, item := range items {
		item = normalizeStreamSavedSource(item)
		if item.SourceID == "" {
			continue
		}
		if idx, ok := seen[item.SourceID]; ok {
			out[idx] = item
			continue
		}
		seen[item.SourceID] = len(out)
		out = append(out, item)
	}
	return out
}

func normalizeStreamSavedConsumers(items []StreamSavedConsumer) []StreamSavedConsumer {
	out := make([]StreamSavedConsumer, 0, len(items))
	seen := make(map[string]int, len(items))
	for _, item := range items {
		item = normalizeStreamSavedConsumer(item)
		if item.ConsumerID == "" {
			continue
		}
		if idx, ok := seen[item.ConsumerID]; ok {
			out[idx] = item
			continue
		}
		seen[item.ConsumerID] = len(out)
		out = append(out, item)
	}
	return out
}

func normalizeStreamSavedSource(item StreamSavedSource) StreamSavedSource {
	item.SourceID = strings.TrimSpace(item.SourceID)
	item.Name = strings.TrimSpace(item.Name)
	item.Kind = strings.ToLower(strings.TrimSpace(item.Kind))
	item.ContentType = strings.TrimSpace(item.ContentType)
	item.Mode = strings.TrimSpace(item.Mode)
	item.UnitMode = strings.TrimSpace(item.UnitMode)
	item.Tags = normalizeStreamSavedTags(item.Tags)
	item.MetadataRaw = strings.TrimSpace(item.MetadataRaw)
	item.InputKind = strings.ToLower(strings.TrimSpace(item.InputKind))
	item.FilePath = strings.TrimSpace(item.FilePath)
	switch item.InputKind {
	case sourceInputKindFile:
	case sourceInputKindDesktop:
	default:
		item.InputKind = ""
		item.FilePath = ""
	}
	if item.Kind == "text" {
		item.InputKind = ""
		item.FilePath = ""
	}
	if item.InputKind == sourceInputKindFile && item.FilePath == "" {
		item.InputKind = ""
	}
	if item.InputKind == sourceInputKindDesktop {
		item.FilePath = ""
		if item.Kind != "video" {
			item.InputKind = ""
		}
	}
	return item
}

func normalizeStreamSavedConsumer(item StreamSavedConsumer) StreamSavedConsumer {
	item.ConsumerID = strings.TrimSpace(item.ConsumerID)
	item.Name = strings.TrimSpace(item.Name)
	item.Kind = strings.ToLower(strings.TrimSpace(item.Kind))
	item.ContentType = strings.TrimSpace(item.ContentType)
	item.Tags = normalizeStreamSavedTags(item.Tags)
	item.MetadataRaw = strings.TrimSpace(item.MetadataRaw)
	return item
}

func normalizeStreamSavedTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	out := make([]string, 0, len(tags))
	seen := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	return out
}

const (
	sourceInputKindFile    = "file"
	sourceInputKindDesktop = "desktop"
)

func (a *App) PickStreamMediaFile() (streamsvc.StreamMediaFileChoice, error) {
	if a.ctx == nil {
		return streamsvc.StreamMediaFileChoice{}, errors.New("app context not initialized")
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select media file",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Media Files",
				Pattern:     "*.mp3;*.wav;*.ogg;*.aac;*.m4a;*.flac;*.mp4;*.webm;*.mov;*.mkv;*.ogv",
			},
		},
	})
	if err != nil {
		return streamsvc.StreamMediaFileChoice{}, err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return streamsvc.StreamMediaFileChoice{}, nil
	}
	return streamsvc.DetectMediaFile(path), nil
}
