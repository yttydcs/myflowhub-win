package main

import "testing"

func TestNormalizeShowcaseConfig_Default(t *testing.T) {
	cfg := normalizeShowcaseConfig(ShowcaseConfig{})
	if cfg.Version != showcaseConfigVersion {
		t.Fatalf("expected version %d got %d", showcaseConfigVersion, cfg.Version)
	}
	if len(cfg.Screens) != 1 {
		t.Fatalf("expected 1 screen got %d", len(cfg.Screens))
	}
	if cfg.Screens[0].ID == "" || cfg.Screens[0].Name == "" {
		t.Fatalf("expected default screen populated got %+v", cfg.Screens[0])
	}
	if cfg.CurrentScreenID != cfg.Screens[0].ID {
		t.Fatalf("expected currentScreenId=%q got %q", cfg.Screens[0].ID, cfg.CurrentScreenID)
	}
}

func TestNormalizeShowcaseConfig_DropsInvalidWidgets(t *testing.T) {
	cfg := normalizeShowcaseConfig(ShowcaseConfig{
		Version:         1,
		CurrentScreenID: "s1",
		Screens: []ShowcaseScreen{
			{
				ID:   "s1",
				Name: "Screen",
				Widgets: []ShowcaseWidget{
					{ID: "w1", Kind: "unknown"},
					{ID: "w2", Kind: "topic_button", TopicButton: &ShowcaseTopicButton{Topic: "", Name: "x"}},
					{ID: "w3", Kind: "var", Var: &ShowcaseVarWidget{OwnerID: 0, Name: "a"}},
					{ID: "w4", Kind: "topic_button", TopicButton: &ShowcaseTopicButton{Topic: "t", Name: "n"}},
					{ID: "w5", Kind: "var", Var: &ShowcaseVarWidget{OwnerID: 7, Name: "v", Mode: "bad"}},
				},
			},
		},
	})
	if len(cfg.Screens) != 1 {
		t.Fatalf("expected 1 screen got %d", len(cfg.Screens))
	}
	screen := cfg.Screens[0]
	if len(screen.Widgets) != 2 {
		t.Fatalf("expected 2 valid widgets got %d", len(screen.Widgets))
	}
	if screen.Widgets[0].Kind == "var" {
		if screen.Widgets[0].Var == nil || screen.Widgets[0].Var.Mode != "auto" {
			t.Fatalf("expected var mode auto got %+v", screen.Widgets[0].Var)
		}
	}
	if screen.Widgets[1].Kind == "var" {
		if screen.Widgets[1].Var == nil || screen.Widgets[1].Var.Mode != "auto" {
			t.Fatalf("expected var mode auto got %+v", screen.Widgets[1].Var)
		}
	}
}

func TestNormalizeShowcaseVarSlider_Defaults(t *testing.T) {
	out := normalizeShowcaseVarSlider(ShowcaseVarSlider{Min: 0, Max: 100, Step: 1, ThrottleMs: -1})
	if out.Min != 0 || out.Max != 100 || out.Step != 1 || out.ThrottleMs != 50 {
		t.Fatalf("unexpected slider defaults: %+v", out)
	}

	explicit := normalizeShowcaseVarSlider(ShowcaseVarSlider{Min: 0, Max: 100, Step: 1, ThrottleMs: 0})
	if explicit.ThrottleMs != 0 {
		t.Fatalf("expected throttleMs=0 preserved got %+v", explicit)
	}
}

func TestNormalizeShowcaseWidget_TargetAndTypeDefaults(t *testing.T) {
	cfg := normalizeShowcaseConfig(ShowcaseConfig{
		Version:         1,
		CurrentScreenID: "s1",
		Screens: []ShowcaseScreen{
			{
				ID:   "s1",
				Name: "Screen",
				Widgets: []ShowcaseWidget{
					{
						ID:       "w1",
						Kind:     "var",
						TargetID: 0,
						Var:      &ShowcaseVarWidget{OwnerID: 7, Name: "a", Mode: "switch", Type: "", Switch: &ShowcaseVarSwitch{OnValue: "on", OffValue: "off"}},
					},
					{
						ID:       "w2",
						Kind:     "var",
						TargetID: 2,
						Var:      &ShowcaseVarWidget{OwnerID: 7, Name: "b", Mode: "slider", Type: ""},
					},
					{
						ID:       "w3",
						Kind:     "topic_button",
						TargetID: 0,
						TopicButton: &ShowcaseTopicButton{
							Topic: "t",
							Name:  "n",
						},
					},
				},
			},
		},
	})
	screen := cfg.Screens[0]
	if len(screen.Widgets) != 3 {
		t.Fatalf("expected 3 widgets got %d", len(screen.Widgets))
	}
	if screen.Widgets[0].TargetID != 1 {
		t.Fatalf("expected target default 1 got %d", screen.Widgets[0].TargetID)
	}
	if screen.Widgets[0].Var == nil || screen.Widgets[0].Var.Type != "bool" {
		t.Fatalf("expected switch type default bool got %+v", screen.Widgets[0].Var)
	}
	if screen.Widgets[1].Var == nil || screen.Widgets[1].Var.Type != "float64" {
		t.Fatalf("expected slider type default float64 got %+v", screen.Widgets[1].Var)
	}
	if screen.Widgets[2].TargetID != 1 {
		t.Fatalf("expected topic_button target default 1 got %d", screen.Widgets[2].TargetID)
	}
}
