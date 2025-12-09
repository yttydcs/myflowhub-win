package ui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	core "github.com/yttydcs/myflowhub-core"
	coreconfig "github.com/yttydcs/myflowhub-core/config"
	"github.com/yttydcs/myflowhub-core/header"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const prefConfigKey = "config.entries"
const (
	prefHomeCredential = "home.credential"
	prefHomeDeviceID   = "home.device_id"
	prefHomeNodeID     = "home.node_id"
	prefHomeHubID      = "home.hub_id"
	prefHomeRole       = "home.role"
	prefHomeAutoCon    = "home.auto_connect"
	prefHomeAutoLog    = "home.auto_login"
	prefVarPoolNames   = "varpool.names"
	prefProfilesList   = "profiles.list"
	prefProfilesLast   = "profiles.last"
	defaultProfileName = "default"
)

var configKeys = []string{
	coreconfig.KeyProcChannelCount,
	coreconfig.KeyProcWorkersPerChan,
	coreconfig.KeyProcChannelBuffer,
	coreconfig.KeyAuthDefaultRole,
	coreconfig.KeyAuthDefaultPerms,
	coreconfig.KeyAuthNodeRoles,
	coreconfig.KeyAuthRolePerms,
	coreconfig.KeySendChannelCount,
	coreconfig.KeySendWorkersPerChan,
	coreconfig.KeySendChannelBuffer,
	coreconfig.KeySendConnBuffer,
	coreconfig.KeySendEnqueueTimeoutMS,
}

// preferences helpers
func (c *Controller) prefKey(key string) string {
	if strings.TrimSpace(c.currentProfile) == "" || c.currentProfile == defaultProfileName {
		return key
	}
	return fmt.Sprintf("%s.%s", c.currentProfile, key)
}

func (c *Controller) initProfiles() {
	if c.app == nil || c.app.Preferences() == nil {
		c.currentProfile = defaultProfileName
		c.profiles = []string{defaultProfileName}
		return
	}
	p := c.app.Preferences()
	raw := p.StringWithFallback(prefProfilesList, "")
	var list []string
	_ = json.Unmarshal([]byte(raw), &list)
	if len(list) == 0 {
		list = []string{defaultProfileName}
		prev := c.currentProfile
		c.currentProfile = defaultProfileName
		c.migrateLegacyPrefs()
		c.currentProfile = prev
		c.saveProfiles(list, defaultProfileName)
	}
	last := p.StringWithFallback(prefProfilesLast, "")
	if last == "" || !contains(list, last) {
		last = list[0]
	}
	c.profiles = list
	c.currentProfile = last
}

func (c *Controller) saveProfiles(list []string, last string) {
	if c.app == nil || c.app.Preferences() == nil {
		return
	}
	p := c.app.Preferences()
	data, _ := json.Marshal(list)
	p.SetString(prefProfilesList, string(data))
	if strings.TrimSpace(last) != "" {
		p.SetString(prefProfilesLast, last)
	}
}

func (c *Controller) migrateLegacyPrefs() {
	if c.app == nil || c.app.Preferences() == nil {
		return
	}
	p := c.app.Preferences()
	origProfile := c.currentProfile
	if strings.TrimSpace(origProfile) == "" {
		c.currentProfile = defaultProfileName
	}
	copyString := func(oldKey, newKey string) {
		oldVal := p.StringWithFallback(oldKey, "")
		if strings.TrimSpace(oldVal) == "" {
			return
		}
		if strings.TrimSpace(p.StringWithFallback(newKey, "")) == "" {
			p.SetString(newKey, oldVal)
		}
	}
	copyInt := func(oldKey, newKey string) {
		oldVal := p.IntWithFallback(oldKey, 0)
		if oldVal == 0 {
			return
		}
		if p.IntWithFallback(newKey, 0) == 0 {
			p.SetInt(newKey, oldVal)
		}
	}
	copyBool := func(oldKey, newKey string) {
		oldVal := p.BoolWithFallback(oldKey, false)
		if !oldVal {
			return
		}
		if !p.BoolWithFallback(newKey, false) {
			p.SetBool(newKey, oldVal)
		}
	}

	copyString(prefHomeCredential, c.prefKey(prefHomeCredential))
	copyString(prefHomeDeviceID, c.prefKey(prefHomeDeviceID))
	copyString(prefHomeRole, c.prefKey(prefHomeRole))
	copyInt(prefHomeNodeID, c.prefKey(prefHomeNodeID))
	copyInt(prefHomeHubID, c.prefKey(prefHomeHubID))
	copyBool(prefHomeAutoCon, c.prefKey(prefHomeAutoCon))
	copyBool(prefHomeAutoLog, c.prefKey(prefHomeAutoLog))
	copyString(prefVarPoolNames, c.prefKey(prefVarPoolNames))
	copyString(prefConfigKey, c.prefKey(prefConfigKey))

	c.currentProfile = origProfile
}

// UI helpers
func labeledEntry(label string, entry *widget.Entry) *fyne.Container {
	return container.NewVBox(widget.NewLabel(label), entry)
}

// wrapScroll 将内容放入可滚动容器并移除最小高度约束，便于窗口缩小。
func wrapScroll(obj fyne.CanvasObject) *container.Scroll {
	scroll := container.NewVScroll(obj)
	scroll.SetMinSize(fyne.NewSize(0, 0))
	return scroll
}

func valueOrPlaceholder(e *widget.Entry) string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Text) != "" {
		return e.Text
	}
	return strings.TrimSpace(e.PlaceHolder)
}

func contains(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

func parseOptionalUint32(text, field string) (uint32, error) {
	if strings.TrimSpace(text) == "" {
		return 0, nil
	}
	v, err := strconv.ParseUint(strings.TrimSpace(text), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s 不是合法数字", field)
	}
	return uint32(v), nil
}

func resolveWindow(app fyne.App, main fyne.Window, candidate fyne.Window) fyne.Window {
	if candidate != nil {
		return candidate
	}
	if main != nil {
		return main
	}
	if app != nil {
		if ws := app.Driver().AllWindows(); len(ws) > 0 {
			return ws[0]
		}
	}
	return nil
}

// payload builder for debug tab
func (c *Controller) buildPayload() ([]byte, error) {
	text := strings.TrimSpace(valueOrPlaceholder(c.payload))
	if c.hexToggle != nil && c.hexToggle.Checked {
		return parseHex(text)
	}
	return []byte(text), nil
}

// header form helpers
type headerForm struct {
	major    *widget.Entry
	subproto *widget.Entry
	source   *widget.Entry
	target   *widget.Entry
	msgID    *widget.Entry
}

func (f *headerForm) Parse() (core.IHeader, error) {
	major, err := strconv.ParseUint(strings.TrimSpace(valueOrPlaceholder(f.major)), 10, 8)
	if err != nil {
		return nil, fmt.Errorf("major invalid")
	}
	sub, err := strconv.ParseUint(strings.TrimSpace(valueOrPlaceholder(f.subproto)), 10, 8)
	if err != nil {
		return nil, fmt.Errorf("subproto invalid")
	}
	source, err := parseOptionalUint32(valueOrPlaceholder(f.source), "source")
	if err != nil {
		return nil, err
	}
	target, err := parseOptionalUint32(valueOrPlaceholder(f.target), "target")
	if err != nil {
		return nil, err
	}
	msgID, err := strconv.ParseUint(strings.TrimSpace(valueOrPlaceholder(f.msgID)), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("msgID invalid")
	}
	return (&header.HeaderTcp{}).
		WithMajor(uint8(major)).
		WithSubProto(uint8(sub)).
		WithSourceID(source).
		WithTargetID(target).
		WithMsgID(uint32(msgID)), nil
}

// sample hex parser
func parseHex(text string) ([]byte, error) {
	text = strings.ReplaceAll(strings.TrimSpace(text), " ", "")
	if text == "" {
		return []byte{}, nil
	}
	if len(text)%2 != 0 {
		return nil, fmt.Errorf("十六进制长度应为偶数")
	}
	out := make([]byte, len(text)/2)
	for i := 0; i < len(text); i += 2 {
		v, err := strconv.ParseUint(text[i:i+2], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("非法十六进制: %w", err)
		}
		out[i/2] = byte(v)
	}
	return out, nil
}

// preset header helpers
func (c *Controller) buildHeaderCard() fyne.CanvasObject {
	c.form = &headerForm{
		major:    widget.NewEntry(),
		subproto: widget.NewEntry(),
		source:   widget.NewEntry(),
		target:   widget.NewEntry(),
		msgID:    widget.NewEntry(),
	}
	c.form.major.SetText("3")
	c.form.subproto.SetText("1")
	c.form.source.SetText("1")
	c.form.target.SetText("0")
	c.form.msgID.SetText(fmt.Sprintf("%d", time.Now().UnixNano()))

	c.hexToggle = widget.NewCheck("Hex payload", nil)
	c.truncToggle = widget.NewCheck("短 preview", nil)
	c.showHex = widget.NewCheck("预览显示 HEX", nil)
	c.presetSrc = widget.NewEntry()
	c.presetSrc.SetPlaceHolder("覆写 SourceID")
	c.presetTgt = widget.NewEntry()
	c.presetTgt.SetPlaceHolder("覆写 TargetID")

	grid := container.NewGridWithColumns(2,
		labeledEntry("Major", c.form.major),
		labeledEntry("SubProto", c.form.subproto),
		labeledEntry("SourceID", c.form.source),
		labeledEntry("TargetID", c.form.target),
		labeledEntry("MsgID", c.form.msgID),
		container.NewVBox(c.hexToggle, c.truncToggle, c.showHex),
		container.NewVBox(labeledEntry("Preset SourceID", c.presetSrc), labeledEntry("Preset TargetID", c.presetTgt)),
	)
	return widget.NewCard("Header", "构造 Header 及调试选项", grid)
}

func (c *Controller) buildPayloadCard() fyne.CanvasObject {
	c.payload = widget.NewMultiLineEntry()
	c.payload.SetMinRowsVisible(6)
	card := widget.NewCard("Payload", "输入文本或十六进制（Hex payload 开启）", c.payload)
	return card
}

func (c *Controller) buildLogTab(w fyne.Window) fyne.CanvasObject {
	c.logView = newLogEntry()
	c.logView.Wrapping = fyne.TextWrapWord
	c.logView.Disable()
	openWinBtn := widget.NewButton("弹窗", func() { c.openLogWindow() })
	top := container.NewBorder(nil, nil, nil, openWinBtn, widget.NewLabel("运行日志"))
	return wrapScroll(container.NewBorder(top, nil, nil, nil, c.logView))
}
