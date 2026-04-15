// 本文件覆盖 Showcase 绑定助手与持久化规则的行为。

package main

import (
	"testing"
	"time"
)

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
	if cfg.Screens[0].UpdatedAt == "" {
		t.Fatalf("expected default screen updatedAt populated")
	}
	if _, err := time.Parse(time.RFC3339Nano, cfg.Screens[0].UpdatedAt); err != nil {
		t.Fatalf("expected valid updatedAt got %q: %v", cfg.Screens[0].UpdatedAt, err)
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
					{ID: "w6", Kind: "var", Var: &ShowcaseVarWidget{OwnerID: 7, Name: "rich", Mode: "metric"}},
				},
			},
		},
	})
	if len(cfg.Screens) != 1 {
		t.Fatalf("expected 1 screen got %d", len(cfg.Screens))
	}
	screen := cfg.Screens[0]
	if len(screen.Widgets) != 3 {
		t.Fatalf("expected 3 valid widgets got %d", len(screen.Widgets))
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
	if screen.Widgets[2].Kind != "var" || screen.Widgets[2].Var == nil || screen.Widgets[2].Var.Mode != "metric" {
		t.Fatalf("expected metric mode preserved got %+v", screen.Widgets[2].Var)
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
					{
						ID:       "w4",
						Kind:     "var",
						TargetID: 3,
						Var:      &ShowcaseVarWidget{OwnerID: 8, Name: "c", Mode: "progress", Type: ""},
					},
					{
						ID:       "w5",
						Kind:     "var",
						TargetID: 4,
						Var:      &ShowcaseVarWidget{OwnerID: 9, Name: "d", Mode: "line_chart", Type: ""},
					},
				},
			},
		},
	})
	screen := cfg.Screens[0]
	if len(screen.Widgets) != 5 {
		t.Fatalf("expected 5 widgets got %d", len(screen.Widgets))
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
	if screen.Widgets[3].Var == nil || screen.Widgets[3].Var.Type != "float64" {
		t.Fatalf("expected progress type default float64 got %+v", screen.Widgets[3].Var)
	}
	if screen.Widgets[4].Var == nil || screen.Widgets[4].Var.Type != "float64" {
		t.Fatalf("expected line_chart type default float64 got %+v", screen.Widgets[4].Var)
	}
}

func TestNormalizeShowcaseVarChart_DefaultsAndClamps(t *testing.T) {
	out := normalizeShowcaseVarChart(ShowcaseVarChart{})
	if out.RangeMs != showcaseLineChartDefaultRangeMs || out.BucketMs != showcaseLineChartDefaultBucketMs {
		t.Fatalf("unexpected chart defaults: %+v", out)
	}

	explicit := normalizeShowcaseVarChart(ShowcaseVarChart{
		RangeMs:  6 * 60 * 60 * 1000,
		BucketMs: 5 * 60 * 1000,
	})
	if explicit.RangeMs != 6*60*60*1000 || explicit.BucketMs != 5*60*1000 {
		t.Fatalf("expected explicit chart values preserved got %+v", explicit)
	}

	clamped := normalizeShowcaseVarChart(ShowcaseVarChart{
		RangeMs:  5 * 60 * 1000,
		BucketMs: 10 * 60 * 1000,
	})
	if clamped.RangeMs != 5*60*1000 {
		t.Fatalf("expected range preserved got %+v", clamped)
	}
	if clamped.BucketMs != showcaseLineChartDefaultBucketMs {
		t.Fatalf("expected invalid bucket reset to default got %+v", clamped)
	}
}

func TestNormalizeShowcaseLayout_Defaults(t *testing.T) {
	cfg := normalizeShowcaseConfig(ShowcaseConfig{
		Version:         1,
		CurrentScreenID: "s1",
		Screens: []ShowcaseScreen{
			{
				ID:   "s1",
				Name: "Screen",
				Widgets: []ShowcaseWidget{
					{
						ID:     "w1",
						Kind:   "topic_button",
						Layout: ShowcaseWidgetLayout{ColSpan: 0},
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
	if screen.Layout.Mode != showcaseLayoutModeColumns {
		t.Fatalf("expected layout mode %q got %q", showcaseLayoutModeColumns, screen.Layout.Mode)
	}
	if screen.Layout.Columns == nil {
		t.Fatalf("expected columns layout present")
	}
	if screen.Layout.Columns.MaxColumns != 3 || screen.Layout.Columns.MinColumnWidth != 360 || screen.Layout.Columns.Gap != 16 {
		t.Fatalf("unexpected columns defaults: %+v", screen.Layout.Columns)
	}
	if got := screen.Widgets[0].Layout.ColSpan; got != 1 {
		t.Fatalf("expected widget colSpan default 1 got %d", got)
	}
}

func TestNormalizeShowcaseLayout_ClampsWidgetColSpan(t *testing.T) {
	cfg := normalizeShowcaseConfig(ShowcaseConfig{
		Version:         1,
		CurrentScreenID: "s1",
		Screens: []ShowcaseScreen{
			{
				ID:   "s1",
				Name: "Screen",
				Layout: ShowcaseScreenLayout{
					Mode: showcaseLayoutModeColumns,
					Columns: &ShowcaseColumnsLayout{
						MaxColumns:     4,
						MinColumnWidth: 100,
						Gap:            0,
					},
				},
				Widgets: []ShowcaseWidget{
					{
						ID:     "w1",
						Kind:   "topic_button",
						Layout: ShowcaseWidgetLayout{ColSpan: 9},
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
	if screen.Layout.Columns == nil {
		t.Fatalf("expected columns layout present")
	}
	if screen.Layout.Columns.MinColumnWidth != 200 {
		t.Fatalf("expected minColumnWidth clamped to 200 got %d", screen.Layout.Columns.MinColumnWidth)
	}
	if screen.Layout.Columns.Gap != 16 {
		t.Fatalf("expected gap default 16 got %d", screen.Layout.Columns.Gap)
	}
	if got := screen.Widgets[0].Layout.ColSpan; got != 4 {
		t.Fatalf("expected widget colSpan clamped to 4 got %d", got)
	}
}

func TestNormalizeShowcaseLayout_CanvasDefaults(t *testing.T) {
	cfg := normalizeShowcaseConfig(ShowcaseConfig{
		Version:         1,
		CurrentScreenID: "s1",
		Screens: []ShowcaseScreen{
			{
				ID:   "s1",
				Name: "Screen",
				Layout: ShowcaseScreenLayout{
					Mode: showcaseLayoutModeCanvas,
				},
				Widgets: []ShowcaseWidget{
					{
						ID:   "w1",
						Kind: "topic_button",
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
	if screen.Layout.Mode != showcaseLayoutModeCanvas {
		t.Fatalf("expected layout mode %q got %q", showcaseLayoutModeCanvas, screen.Layout.Mode)
	}
	if screen.Layout.Canvas == nil {
		t.Fatalf("expected canvas layout present")
	}
	if screen.Layout.Canvas.BaseWidth != showcaseCanvasDefaultBaseWidth || screen.Layout.Canvas.BaseHeight != showcaseCanvasDefaultBaseHeight {
		t.Fatalf("unexpected canvas defaults: %+v", screen.Layout.Canvas)
	}
	if screen.Widgets[0].Layout.CanvasPercent == nil {
		t.Fatalf("expected widget canvasPercent populated")
	}
}

func TestNormalizeShowcaseLayout_CanvasPercentClamps(t *testing.T) {
	cfg := normalizeShowcaseConfig(ShowcaseConfig{
		Version:         1,
		CurrentScreenID: "s1",
		Screens: []ShowcaseScreen{
			{
				ID:   "s1",
				Name: "Screen",
				Layout: ShowcaseScreenLayout{
					Mode: showcaseLayoutModeCanvas,
					Canvas: &ShowcaseCanvasLayout{
						BaseWidth:  200,
						BaseHeight: 100,
					},
				},
				Widgets: []ShowcaseWidget{
					{
						ID:   "w1",
						Kind: "topic_button",
						Layout: ShowcaseWidgetLayout{
							CanvasPercent: &ShowcaseCanvasRectPercent{
								XPct: -5,
								YPct: 200,
								WPct: 1,
								HPct: 1,
							},
						},
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
	rect := screen.Widgets[0].Layout.CanvasPercent
	if rect == nil {
		t.Fatalf("expected canvasPercent present")
	}
	if rect.WPct != 40 {
		t.Fatalf("expected wPct clamped to 40 got %v", rect.WPct)
	}
	if rect.HPct != 48 {
		t.Fatalf("expected hPct clamped to 48 got %v", rect.HPct)
	}
	if rect.XPct != 0 {
		t.Fatalf("expected xPct clamped to 0 got %v", rect.XPct)
	}
	if rect.YPct != 52 {
		t.Fatalf("expected yPct clamped to 52 got %v", rect.YPct)
	}
}

func TestNormalizeShowcaseScreen_UpdatedAt(t *testing.T) {
	const valid = "2026-03-21T10:15:20Z"

	cfg := normalizeShowcaseConfig(ShowcaseConfig{
		Version:         1,
		CurrentScreenID: "s1",
		Screens: []ShowcaseScreen{
			{
				ID:        "s1",
				Name:      "Screen",
				UpdatedAt: valid,
			},
			{
				ID:        "s2",
				Name:      "Broken Timestamp",
				UpdatedAt: "not-a-timestamp",
			},
		},
	})

	if got := cfg.Screens[0].UpdatedAt; got != valid {
		t.Fatalf("expected valid updatedAt preserved got %q", got)
	}
	if got := cfg.Screens[1].UpdatedAt; got == "" {
		t.Fatalf("expected invalid updatedAt replaced")
	} else if _, err := time.Parse(time.RFC3339Nano, got); err != nil {
		t.Fatalf("expected replacement updatedAt to be valid got %q: %v", got, err)
	}
}
