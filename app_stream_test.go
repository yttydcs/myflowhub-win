package main

import "testing"

func TestStreamPrefs_Defaults(t *testing.T) {
	app := newTestAppWithStore(t)

	prefs, err := app.StreamPrefs()
	if err != nil {
		t.Fatalf("StreamPrefs() error = %v", err)
	}
	if prefs.ActiveTab != "source" {
		t.Fatalf("expected default activeTab=source got %q", prefs.ActiveTab)
	}
	if prefs.TargetID != 0 {
		t.Fatalf("expected default targetId=0 got %d", prefs.TargetID)
	}
	if len(prefs.Sources) != 0 || len(prefs.Consumers) != 0 {
		t.Fatalf("expected empty stream prefs got %+v", prefs)
	}
}

func TestSaveStreamPrefs_NormalizesAndPersists(t *testing.T) {
	app := newTestAppWithStore(t)

	saved, err := app.SaveStreamPrefs(StreamPrefs{
		ActiveTab: " Consumer ",
		TargetID:  19,
		Sources: []StreamSavedSource{
			{
				SourceID:    " src-a ",
				Name:        " Source A ",
				Kind:        " Text ",
				ContentType: " text/plain ",
				Mode:        " live ",
				UnitMode:    " frame ",
				Tags:        []string{" alpha ", "beta", "alpha", ""},
				MetadataRaw: "  {\"room\":1}  ",
			},
			{
				SourceID:  "src-a",
				Name:      "Source A Override",
				Kind:      "video",
				InputKind: " file ",
				FilePath:  " C:/media/demo.mp4 ",
			},
		},
		Consumers: []StreamSavedConsumer{
			{
				ConsumerID:  " consumer-a ",
				Name:        " Consumer A ",
				Kind:        " Text ",
				ContentType: " text/plain ",
				Tags:        []string{" alpha ", "beta", "alpha"},
				MetadataRaw: "  {\"buffer\":32} ",
			},
			{
				ConsumerID: "consumer-a",
				Name:       "Consumer A Override",
				Kind:       "text",
			},
		},
	})
	if err != nil {
		t.Fatalf("SaveStreamPrefs() error = %v", err)
	}
	if saved.ActiveTab != "consumer" {
		t.Fatalf("expected normalized activeTab=consumer got %q", saved.ActiveTab)
	}
	if saved.TargetID != 19 {
		t.Fatalf("expected targetId=19 got %d", saved.TargetID)
	}
	if len(saved.Sources) != 1 {
		t.Fatalf("expected 1 source got %+v", saved.Sources)
	}
	if saved.Sources[0].SourceID != "src-a" || saved.Sources[0].Name != "Source A Override" || saved.Sources[0].Kind != "video" {
		t.Fatalf("unexpected saved source %+v", saved.Sources[0])
	}
	if saved.Sources[0].InputKind != "file" || saved.Sources[0].FilePath != "C:/media/demo.mp4" {
		t.Fatalf("expected normalized file input on saved source got %+v", saved.Sources[0])
	}
	if len(saved.Consumers) != 1 {
		t.Fatalf("expected 1 consumer got %+v", saved.Consumers)
	}
	if saved.Consumers[0].ConsumerID != "consumer-a" || saved.Consumers[0].Name != "Consumer A Override" || saved.Consumers[0].Kind != "text" {
		t.Fatalf("unexpected saved consumer %+v", saved.Consumers[0])
	}

	loaded, err := app.StreamPrefs()
	if err != nil {
		t.Fatalf("StreamPrefs() error = %v", err)
	}
	if loaded.ActiveTab != "consumer" || loaded.TargetID != 19 {
		t.Fatalf("unexpected loaded prefs %+v", loaded)
	}
	if len(loaded.Sources) != 1 || loaded.Sources[0].SourceID != "src-a" {
		t.Fatalf("unexpected loaded sources %+v", loaded.Sources)
	}
	if len(loaded.Consumers) != 1 || loaded.Consumers[0].ConsumerID != "consumer-a" {
		t.Fatalf("unexpected loaded consumers %+v", loaded.Consumers)
	}
}

func TestSaveStreamPrefs_NormalizesInvalidValues(t *testing.T) {
	app := newTestAppWithStore(t)

	saved, err := app.SaveStreamPrefs(StreamPrefs{
		ActiveTab: "unknown",
		TargetID:  -5,
		Sources: []StreamSavedSource{
			{SourceID: " ", Name: "drop-me"},
		},
		Consumers: []StreamSavedConsumer{
			{ConsumerID: "", Name: "drop-me"},
		},
	})
	if err != nil {
		t.Fatalf("SaveStreamPrefs() error = %v", err)
	}
	if saved.ActiveTab != "source" {
		t.Fatalf("expected fallback activeTab=source got %q", saved.ActiveTab)
	}
	if saved.TargetID != 0 {
		t.Fatalf("expected normalized targetId=0 got %d", saved.TargetID)
	}
	if len(saved.Sources) != 0 || len(saved.Consumers) != 0 {
		t.Fatalf("expected invalid entries to be dropped got %+v", saved)
	}
}

func TestSaveStreamPrefs_KeepsDesktopInputOnlyForVideoSources(t *testing.T) {
	app := newTestAppWithStore(t)

	saved, err := app.SaveStreamPrefs(StreamPrefs{
		Sources: []StreamSavedSource{
			{
				SourceID:    "video-source",
				Name:        "Desktop Video",
				Kind:        "video",
				ContentType: "video/webm",
				Mode:        "bounded",
				UnitMode:    "chunk",
				InputKind:   " desktop ",
				FilePath:    " C:/should/not/persist.webm ",
			},
			{
				SourceID:    "music-source",
				Name:        "Bad Desktop Music",
				Kind:        "music",
				ContentType: "audio/webm",
				Mode:        "bounded",
				UnitMode:    "chunk",
				InputKind:   "desktop",
			},
		},
	})
	if err != nil {
		t.Fatalf("SaveStreamPrefs() error = %v", err)
	}
	if len(saved.Sources) != 2 {
		t.Fatalf("expected 2 sources got %+v", saved.Sources)
	}
	if saved.Sources[0].InputKind != "desktop" || saved.Sources[0].FilePath != "" {
		t.Fatalf("expected video desktop source to keep desktop mode without file path, got %+v", saved.Sources[0])
	}
	if saved.Sources[1].InputKind != "" || saved.Sources[1].FilePath != "" {
		t.Fatalf("expected non-video desktop source to be cleared, got %+v", saved.Sources[1])
	}
}
