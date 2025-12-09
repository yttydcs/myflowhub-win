package ui

import (
	"encoding/json"
	"strings"

	coreconfig "github.com/yttydcs/myflowhub-core/config"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type configRow struct {
	key   *widget.Entry
	value *widget.Entry
	card  fyne.CanvasObject
}

func (c *Controller) addConfigRow(key, value string) {
	keyEntry := widget.NewEntry()
	keyEntry.SetText(key)
	valEntry := widget.NewEntry()
	valEntry.SetText(value)
	removeBtn := widget.NewButton("删除", func() {
		c.removeConfigRow(keyEntry, valEntry)
	})
	row := container.NewBorder(nil, nil, nil, removeBtn, container.NewVBox(
		labeledEntry("Key", keyEntry),
		labeledEntry("Value", valEntry),
	))
	c.configRows = append(c.configRows, &configRow{key: keyEntry, value: valEntry, card: row})
	c.configList.Add(row)
}

func (c *Controller) removeConfigRow(keyEntry, valEntry *widget.Entry) {
	filtered := make([]*configRow, 0, len(c.configRows))
	for _, r := range c.configRows {
		if r.key == keyEntry && r.value == valEntry {
			continue
		}
		filtered = append(filtered, r)
	}
	c.configRows = filtered
	c.reloadConfigUI(true)
}

func (c *Controller) reloadConfigUI(fromPrefs bool) {
	if c.configList == nil {
		return
	}
	c.configList.Objects = nil
	c.configRows = nil
	cfg := c.currentConfig(fromPrefs)
	for k, v := range cfg {
		c.addConfigRow(k, v)
	}
	c.configList.Refresh()
}

func (c *Controller) saveConfigFromUI() error {
	if c.app == nil || c.app.Preferences() == nil {
		return nil
	}
	cfg := make(map[string]string)
	for _, r := range c.configRows {
		k := strings.TrimSpace(r.key.Text)
		v := strings.TrimSpace(r.value.Text)
		if k != "" {
			cfg[k] = v
		}
	}
	data, _ := json.Marshal(cfg)
	c.app.Preferences().SetString(c.prefKey(prefConfigKey), string(data))
	return nil
}

func (c *Controller) resetConfigToDefault() {
	if c.configList == nil {
		return
	}
	c.reloadConfigUI(false)
}

func (c *Controller) currentConfig(fromPrefs bool) map[string]string {
	cfg := defaultConfig()
	if !fromPrefs || c.app == nil || c.app.Preferences() == nil {
		return cfg
	}
	raw := c.app.Preferences().StringWithFallback(c.prefKey(prefConfigKey), "")
	if strings.TrimSpace(raw) == "" {
		return cfg
	}
	var saved map[string]string
	if err := json.Unmarshal([]byte(raw), &saved); err != nil {
		return cfg
	}
	for k, v := range saved {
		cfg[k] = v
	}
	return cfg
}

func defaultConfig() map[string]string {
	return map[string]string{
		coreconfig.KeyProcChannelCount:   "2",
		coreconfig.KeyProcWorkersPerChan: "2",
		coreconfig.KeyProcChannelBuffer:  "128",
		coreconfig.KeySendChannelCount:   "1",
		coreconfig.KeySendWorkersPerChan: "1",
		coreconfig.KeySendChannelBuffer:  "64",
		coreconfig.KeySendConnBuffer:     "64",
	}
}
