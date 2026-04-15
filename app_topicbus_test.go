// 本文件覆盖 TopicBus 绑定助手与持久化规则的行为。

package main

import "testing"

func TestTopicBusPrefs_Defaults(t *testing.T) {
	app := newTestAppWithStore(t)

	prefs, err := app.TopicBusPrefs()
	if err != nil {
		t.Fatalf("TopicBusPrefs() error = %v", err)
	}
	if len(prefs.Topics) != 0 {
		t.Fatalf("expected no topics got %#v", prefs.Topics)
	}
	if prefs.MaxEvents != defaultTopicBusMax {
		t.Fatalf("expected default maxEvents=%d got %d", defaultTopicBusMax, prefs.MaxEvents)
	}
	if prefs.TargetID != 0 {
		t.Fatalf("expected default targetId=0 got %d", prefs.TargetID)
	}
}

func TestSaveTopicBusPrefs_NormalizesAndPersists(t *testing.T) {
	app := newTestAppWithStore(t)

	saved, err := app.SaveTopicBusPrefs(TopicBusPrefs{
		Topics:    []string{" topic.a ", "topic.b", "topic.a", ""},
		MaxEvents: 256,
		TargetID:  17,
	})
	if err != nil {
		t.Fatalf("SaveTopicBusPrefs() error = %v", err)
	}
	if len(saved.Topics) != 2 || saved.Topics[0] != "topic.a" || saved.Topics[1] != "topic.b" {
		t.Fatalf("expected normalized topics, got %#v", saved.Topics)
	}
	if saved.MaxEvents != 256 {
		t.Fatalf("expected maxEvents=256 got %d", saved.MaxEvents)
	}
	if saved.TargetID != 17 {
		t.Fatalf("expected targetId=17 got %d", saved.TargetID)
	}

	loaded, err := app.TopicBusPrefs()
	if err != nil {
		t.Fatalf("TopicBusPrefs() error = %v", err)
	}
	if len(loaded.Topics) != 2 || loaded.Topics[0] != "topic.a" || loaded.Topics[1] != "topic.b" {
		t.Fatalf("expected persisted topics, got %#v", loaded.Topics)
	}
	if loaded.MaxEvents != 256 {
		t.Fatalf("expected persisted maxEvents=256 got %d", loaded.MaxEvents)
	}
	if loaded.TargetID != 17 {
		t.Fatalf("expected persisted targetId=17 got %d", loaded.TargetID)
	}
}

func TestSaveTopicBusPrefs_NormalizesInvalidValues(t *testing.T) {
	app := newTestAppWithStore(t)

	saved, err := app.SaveTopicBusPrefs(TopicBusPrefs{
		Topics:    nil,
		MaxEvents: -1,
		TargetID:  -9,
	})
	if err != nil {
		t.Fatalf("SaveTopicBusPrefs() error = %v", err)
	}
	if saved.MaxEvents != defaultTopicBusMax {
		t.Fatalf("expected normalized maxEvents=%d got %d", defaultTopicBusMax, saved.MaxEvents)
	}
	if saved.TargetID != 0 {
		t.Fatalf("expected normalized targetId=0 got %d", saved.TargetID)
	}
}
