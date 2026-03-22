package main

import (
	"encoding/json"
	"errors"
	"strings"
)

const (
	topicBusSubsKey      = "topicbus.subs"
	topicBusMaxEventsKey = "topicbus.max_events"
	topicBusTargetIDKey  = "topicbus.target_id"
	defaultTopicBusMax   = 500
)

type TopicBusPrefs struct {
	Topics    []string `json:"topics"`
	MaxEvents int      `json:"maxEvents"`
	TargetID  int      `json:"targetId"`
}

func (a *App) TopicBusPrefs() (TopicBusPrefs, error) {
	if a.store == nil {
		return TopicBusPrefs{}, errors.New("storage not initialized")
	}
	profile := a.store.CurrentProfile()
	raw := a.store.GetString(profile, topicBusSubsKey, "")
	topics := normalizeTopicBusTopics(parseTopicBusTopics(raw))
	maxEvents := a.store.GetInt(profile, topicBusMaxEventsKey, defaultTopicBusMax)
	if maxEvents <= 0 {
		maxEvents = defaultTopicBusMax
	}
	targetID := a.store.GetInt(profile, topicBusTargetIDKey, 0)
	if targetID < 0 {
		targetID = 0
	}
	return TopicBusPrefs{Topics: topics, MaxEvents: maxEvents, TargetID: targetID}, nil
}

func (a *App) SaveTopicBusPrefs(prefs TopicBusPrefs) (TopicBusPrefs, error) {
	if a.store == nil {
		return TopicBusPrefs{}, errors.New("storage not initialized")
	}
	normalized := normalizeTopicBusTopics(prefs.Topics)
	maxEvents := prefs.MaxEvents
	if maxEvents <= 0 {
		maxEvents = defaultTopicBusMax
	}
	targetID := prefs.TargetID
	if targetID < 0 {
		targetID = 0
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return TopicBusPrefs{}, err
	}
	profile := a.store.CurrentProfile()
	if err := a.store.SetString(profile, topicBusSubsKey, string(data)); err != nil {
		return TopicBusPrefs{}, err
	}
	if err := a.store.SetInt(profile, topicBusMaxEventsKey, maxEvents); err != nil {
		return TopicBusPrefs{}, err
	}
	if err := a.store.SetInt(profile, topicBusTargetIDKey, targetID); err != nil {
		return TopicBusPrefs{}, err
	}
	return TopicBusPrefs{Topics: normalized, MaxEvents: maxEvents, TargetID: targetID}, nil
}

func parseTopicBusTopics(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var topics []string
	if err := json.Unmarshal([]byte(raw), &topics); err == nil {
		return topics
	}
	return nil
}

func normalizeTopicBusTopics(topics []string) []string {
	out := make([]string, 0, len(topics))
	seen := make(map[string]bool, len(topics))
	for _, topic := range topics {
		topic = strings.TrimSpace(topic)
		if topic == "" || seen[topic] {
			continue
		}
		seen[topic] = true
		out = append(out, topic)
	}
	return out
}
