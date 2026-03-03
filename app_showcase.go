package main

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	showcaseConfigKey     = "showcase.config"
	showcaseConfigVersion = 1

	showcaseLayoutModeColumns = "columns"
)

type ShowcaseConfig struct {
	Version         int              `json:"version"`
	CurrentScreenID string           `json:"currentScreenId,omitempty"`
	Screens         []ShowcaseScreen `json:"screens"`
}

type ShowcaseScreen struct {
	ID      string               `json:"id"`
	Name    string               `json:"name"`
	Layout  ShowcaseScreenLayout `json:"layout,omitempty"`
	Widgets []ShowcaseWidget     `json:"widgets,omitempty"`
}

type ShowcaseScreenLayout struct {
	Mode    string                 `json:"mode,omitempty"` // columns (V1)
	Columns *ShowcaseColumnsLayout `json:"columns,omitempty"`
}

type ShowcaseColumnsLayout struct {
	MaxColumns     int `json:"maxColumns,omitempty"`
	MinColumnWidth int `json:"minColumnWidth,omitempty"` // px
	Gap            int `json:"gap,omitempty"`            // px
}

type ShowcaseWidget struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"` // "topic_button" | "var"
	Title    string `json:"title,omitempty"`
	TargetID uint32 `json:"targetId,omitempty"`

	Layout ShowcaseWidgetLayout `json:"layout,omitempty"`

	TopicButton *ShowcaseTopicButton `json:"topicButton,omitempty"`
	Var         *ShowcaseVarWidget   `json:"var,omitempty"`
}

type ShowcaseWidgetLayout struct {
	ColSpan int `json:"colSpan,omitempty"`
}

type ShowcaseTopicButton struct {
	Topic       string `json:"topic"`
	Name        string `json:"name"`
	PayloadText string `json:"payloadText,omitempty"`
}

type ShowcaseVarWidget struct {
	OwnerID    uint32             `json:"ownerId"`
	Name       string             `json:"name"`
	Mode       string             `json:"mode,omitempty"` // auto|display|slider|switch
	Visibility string             `json:"visibility,omitempty"`
	Type       string             `json:"type,omitempty"` // optional; empty means "do not override"
	Slider     *ShowcaseVarSlider `json:"slider,omitempty"`
	Switch     *ShowcaseVarSwitch `json:"switch,omitempty"`
}

type ShowcaseVarSlider struct {
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	Step       float64 `json:"step"`
	ThrottleMs int     `json:"throttleMs"`
}

type ShowcaseVarSwitch struct {
	OnValue  string `json:"onValue,omitempty"`
	OffValue string `json:"offValue,omitempty"`
}

func (a *App) ShowcaseConfig() (ShowcaseConfig, error) {
	if a.store == nil {
		return ShowcaseConfig{}, errors.New("storage not initialized")
	}
	profile := a.store.CurrentProfile()
	raw := a.store.GetString(profile, showcaseConfigKey, "")
	cfg := parseShowcaseConfig(raw)
	cfg = normalizeShowcaseConfig(cfg)
	return cfg, nil
}

func (a *App) SaveShowcaseConfig(cfg ShowcaseConfig) (ShowcaseConfig, error) {
	if a.store == nil {
		return ShowcaseConfig{}, errors.New("storage not initialized")
	}
	normalized := normalizeShowcaseConfig(cfg)
	data, err := json.Marshal(normalized)
	if err != nil {
		return ShowcaseConfig{}, err
	}
	profile := a.store.CurrentProfile()
	if err := a.store.SetString(profile, showcaseConfigKey, string(data)); err != nil {
		return ShowcaseConfig{}, err
	}
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "showcase.config_changed", map[string]any{
			"ts":      time.Now().UTC().Format(time.RFC3339Nano),
			"profile": profile,
		})
	}
	return a.ShowcaseConfig()
}

func parseShowcaseConfig(raw string) ShowcaseConfig {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ShowcaseConfig{}
	}
	var cfg ShowcaseConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return ShowcaseConfig{}
	}
	return cfg
}

func normalizeShowcaseConfig(cfg ShowcaseConfig) ShowcaseConfig {
	if cfg.Version <= 0 {
		cfg.Version = showcaseConfigVersion
	}
	cfg.CurrentScreenID = strings.TrimSpace(cfg.CurrentScreenID)

	screens := make([]ShowcaseScreen, 0, len(cfg.Screens))
	seenScreens := make(map[string]bool, len(cfg.Screens))
	for _, screen := range cfg.Screens {
		screen = normalizeShowcaseScreen(screen)
		if screen.ID == "" || screen.Name == "" {
			continue
		}
		if seenScreens[screen.ID] {
			continue
		}
		seenScreens[screen.ID] = true
		screens = append(screens, screen)
	}
	if len(screens) == 0 {
		screens = append(screens, normalizeShowcaseScreen(ShowcaseScreen{
			ID:   "default",
			Name: "Default",
		}))
	}
	cfg.Screens = screens

	if cfg.CurrentScreenID == "" || !containsShowcaseScreenID(cfg.Screens, cfg.CurrentScreenID) {
		cfg.CurrentScreenID = cfg.Screens[0].ID
	}
	return cfg
}

func normalizeShowcaseScreen(screen ShowcaseScreen) ShowcaseScreen {
	screen.ID = strings.TrimSpace(screen.ID)
	if screen.ID == "" {
		screen.ID = newShowcaseID("scr")
	}
	screen.Name = strings.TrimSpace(screen.Name)
	if screen.Name == "" {
		screen.Name = "Screen"
	}

	screen.Layout = normalizeShowcaseScreenLayout(screen.Layout)

	widgets := make([]ShowcaseWidget, 0, len(screen.Widgets))
	seenWidgets := make(map[string]bool, len(screen.Widgets))
	for _, widget := range screen.Widgets {
		widget, ok := normalizeShowcaseWidget(widget)
		if !ok {
			continue
		}
		widget.Layout = normalizeShowcaseWidgetLayout(widget.Layout, screen.Layout)
		if widget.ID == "" {
			continue
		}
		if seenWidgets[widget.ID] {
			continue
		}
		seenWidgets[widget.ID] = true
		widgets = append(widgets, widget)
	}
	screen.Widgets = widgets
	return screen
}

func normalizeShowcaseWidget(widget ShowcaseWidget) (ShowcaseWidget, bool) {
	widget.ID = strings.TrimSpace(widget.ID)
	if widget.ID == "" {
		widget.ID = newShowcaseID("wgt")
	}
	widget.Kind = strings.TrimSpace(widget.Kind)
	widget.Title = strings.TrimSpace(widget.Title)
	if widget.TargetID == 0 {
		widget.TargetID = 1
	}
	widget.Layout = normalizeShowcaseWidgetLayout(widget.Layout, ShowcaseScreenLayout{})

	switch widget.Kind {
	case "topic_button":
		if widget.TopicButton == nil {
			return ShowcaseWidget{}, false
		}
		tb := normalizeShowcaseTopicButton(*widget.TopicButton)
		if tb.Topic == "" || tb.Name == "" {
			return ShowcaseWidget{}, false
		}
		widget.TopicButton = &tb
		widget.Var = nil
		return widget, true
	case "var":
		if widget.Var == nil {
			return ShowcaseWidget{}, false
		}
		v := normalizeShowcaseVarWidget(*widget.Var)
		if v.OwnerID == 0 || v.Name == "" {
			return ShowcaseWidget{}, false
		}
		widget.Var = &v
		widget.TopicButton = nil
		return widget, true
	default:
		return ShowcaseWidget{}, false
	}
}

func normalizeShowcaseTopicButton(tb ShowcaseTopicButton) ShowcaseTopicButton {
	tb.Topic = strings.TrimSpace(tb.Topic)
	tb.Name = strings.TrimSpace(tb.Name)
	tb.PayloadText = strings.TrimSpace(tb.PayloadText)
	return tb
}

func normalizeShowcaseScreenLayout(layout ShowcaseScreenLayout) ShowcaseScreenLayout {
	layout.Mode = strings.ToLower(strings.TrimSpace(layout.Mode))
	switch layout.Mode {
	case "":
		layout.Mode = showcaseLayoutModeColumns
		fallthrough
	case showcaseLayoutModeColumns:
		cols := layout.Columns
		if cols == nil {
			cols = &ShowcaseColumnsLayout{}
		}
		*cols = normalizeShowcaseColumnsLayout(*cols)
		layout.Columns = cols
	default:
		layout.Mode = showcaseLayoutModeColumns
		cols := layout.Columns
		if cols == nil {
			cols = &ShowcaseColumnsLayout{}
		}
		*cols = normalizeShowcaseColumnsLayout(*cols)
		layout.Columns = cols
	}
	return layout
}

func normalizeShowcaseColumnsLayout(c ShowcaseColumnsLayout) ShowcaseColumnsLayout {
	if c.MaxColumns <= 0 {
		c.MaxColumns = 3
	}
	if c.MaxColumns < 1 {
		c.MaxColumns = 1
	}
	if c.MaxColumns > 12 {
		c.MaxColumns = 12
	}

	if c.MinColumnWidth <= 0 {
		c.MinColumnWidth = 360
	}
	if c.MinColumnWidth < 200 {
		c.MinColumnWidth = 200
	}
	if c.MinColumnWidth > 1200 {
		c.MinColumnWidth = 1200
	}

	if c.Gap <= 0 {
		c.Gap = 16
	}
	if c.Gap > 64 {
		c.Gap = 64
	}

	return c
}

func normalizeShowcaseWidgetLayout(layout ShowcaseWidgetLayout, screenLayout ShowcaseScreenLayout) ShowcaseWidgetLayout {
	if layout.ColSpan <= 0 {
		layout.ColSpan = 1
	}
	maxCols := 12
	if screenLayout.Mode == showcaseLayoutModeColumns && screenLayout.Columns != nil && screenLayout.Columns.MaxColumns > 0 {
		maxCols = screenLayout.Columns.MaxColumns
	}
	if maxCols < 1 {
		maxCols = 1
	}
	if layout.ColSpan > maxCols {
		layout.ColSpan = maxCols
	}
	return layout
}

func normalizeShowcaseVarWidget(v ShowcaseVarWidget) ShowcaseVarWidget {
	v.Name = strings.TrimSpace(v.Name)
	v.Mode = normalizeVarMode(v.Mode)
	v.Visibility = strings.TrimSpace(v.Visibility)
	if v.Visibility == "" {
		v.Visibility = "public"
	}
	v.Type = strings.TrimSpace(v.Type)
	if v.Type == "" {
		switch v.Mode {
		case "slider":
			v.Type = "float64"
		case "switch":
			v.Type = "bool"
		default:
			v.Type = "string"
		}
	}

	slider := v.Slider
	if slider == nil {
		slider = &ShowcaseVarSlider{Min: 0, Max: 100, Step: 1, ThrottleMs: 50}
	}
	*slider = normalizeShowcaseVarSlider(*slider)
	v.Slider = slider

	sw := v.Switch
	if sw == nil {
		sw = &ShowcaseVarSwitch{OnValue: "true", OffValue: "false"}
	}
	*sw = normalizeShowcaseVarSwitch(*sw)
	v.Switch = sw

	return v
}

func normalizeShowcaseVarSlider(s ShowcaseVarSlider) ShowcaseVarSlider {
	if s.Max <= s.Min {
		s.Min, s.Max = 0, 100
	}
	if s.Step <= 0 {
		s.Step = 1
	}
	if s.ThrottleMs < 0 {
		s.ThrottleMs = 50
	}
	return s
}

func normalizeShowcaseVarSwitch(s ShowcaseVarSwitch) ShowcaseVarSwitch {
	s.OnValue = strings.TrimSpace(s.OnValue)
	s.OffValue = strings.TrimSpace(s.OffValue)
	if s.OnValue == "" {
		s.OnValue = "true"
	}
	if s.OffValue == "" {
		s.OffValue = "false"
	}
	return s
}

func normalizeVarMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "", "auto", "display", "slider", "switch":
		if mode == "" {
			return "auto"
		}
		return mode
	default:
		return "auto"
	}
}

func containsShowcaseScreenID(screens []ShowcaseScreen, id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	for _, s := range screens {
		if s.ID == id {
			return true
		}
	}
	return false
}

func newShowcaseID(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "id"
	}
	return prefix + "-" + strings.ToLower(base36(time.Now().UnixNano()))
}

func base36(n int64) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	if n < 0 {
		n = -n
	}
	if n == 0 {
		return "0"
	}
	var out [32]byte
	i := len(out)
	for n > 0 && i > 0 {
		i--
		out[i] = alphabet[n%36]
		n /= 36
	}
	return string(out[i:])
}
