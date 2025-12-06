package ui

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	core "github.com/yttydcs/myflowhub-core"
	coreconfig "github.com/yttydcs/myflowhub-core/config"
	"github.com/yttydcs/myflowhub-core/header"
	"github.com/yttydcs/myflowhub-win/internal/session"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type Controller struct {
	app     fyne.App
	ctx     context.Context
	session *session.Session
	logBuf  bytes.Buffer
	logMu   sync.Mutex

	addrEntry *widget.Entry
	payload   *widget.Entry
	logView   *logEntry
	form      *headerForm

	nodeEntry *widget.Entry
	hexToggle *widget.Check

	configRows []*configRow
	configList *fyne.Container
	configInfo *widget.Label

	presetCards []*widget.Card

	logPopup  *logEntry
	logWindow fyne.Window
}

func New(app fyne.App, ctx context.Context) *Controller {
	c := &Controller{app: app, ctx: ctx}
	c.session = session.New(c.ctx, c.handleFrame, c.handleError)
	return c
}

func (c *Controller) Build(w fyne.Window) fyne.CanvasObject {
	debugTab := c.buildDebugTab(w)
	configTab := c.buildConfigTab(w)
	presetTab := c.buildPresetTab(w)
	tabs := container.NewAppTabs(
		container.NewTabItem("Config", configTab),
		container.NewTabItem("Debug", debugTab),
		container.NewTabItem("Presets", presetTab),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	if len(tabs.Items) > 2 {
		tabs.SelectTabIndex(1) // 默认落在 Debug，保持原体验
	}
	return tabs
}

func (c *Controller) buildDebugTab(w fyne.Window) fyne.CanvasObject {
	c.addrEntry = widget.NewEntry()
	c.addrEntry.SetPlaceHolder("127.0.0.1:9000")

	c.nodeEntry = widget.NewEntry()
	c.nodeEntry.SetPlaceHolder("debugclient")

	connectBtn := widget.NewButton("连接", func() {
		addr := valueOrPlaceholder(c.addrEntry)
		go func() {
			if err := c.session.Connect(addr); err != nil {
				c.appendLog("[ERR] connect: %v", err)
				return
			}
			nodeName := valueOrPlaceholder(c.nodeEntry)
			if nodeName == "" {
				nodeName = "debugclient"
			}
			if err := c.session.Login(nodeName); err != nil {
				c.appendLog("[ERR] login: %v", err)
			} else {
				c.appendLog("[OK] connected %s", addr)
			}
		}()
	})

	disconnectBtn := widget.NewButton("断开", func() {
		c.session.Close()
		c.appendLog("[INFO] 手动断开")
	})

	headerCard := c.buildHeaderCard()
	payloadCard := c.buildPayloadCard()
	logCard := c.buildLogCard()

	sendBtn := widget.NewButton("发送", func() {
		hdr, err := c.form.Parse()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		payload, perr := c.buildPayload()
		if perr != nil {
			dialog.ShowError(perr, w)
			return
		}
		if err := c.session.Send(hdr, payload); err != nil {
			c.appendLog("[ERR] send: %v", err)
			return
		}
		c.appendLog("[TX] sub=%d len=%d", hdr.SubProto(), len(payload))
	})
	openLogBtn := widget.NewButton("弹出日志窗口", func() {
		c.openLogWindow()
	})

	topBar := container.NewBorder(nil, nil,
		widget.NewLabel("Server"),
		container.NewHBox(connectBtn, disconnectBtn),
		container.NewMax(c.addrEntry))

	content := container.NewVSplit(
		container.NewVBox(headerCard, payloadCard),
		logCard,
	)
	content.SetOffset(0.55)

	btns := container.NewHBox(sendBtn, openLogBtn)
	return container.NewBorder(topBar, btns, nil, nil, content)
}

func (c *Controller) buildConfigTab(w fyne.Window) fyne.CanvasObject {
	c.configInfo = widget.NewLabel("展示当前生效配置（默认值+已保存覆盖）")
	c.configList = container.NewVBox()
	c.reloadConfigUI(true)

	addBtn := widget.NewButton("新增空行", func() {
		c.addConfigRow("", "")
	})
	saveBtn := widget.NewButton("保存", func() {
		if err := c.saveConfigFromUI(); err != nil {
			dialog.ShowError(err, w)
			return
		}
		c.configInfo.SetText("已保存到本地（Preferences）")
	})
	reloadBtn := widget.NewButton("重新加载", func() {
		c.reloadConfigUI(false)
		c.configInfo.SetText("已从本地加载（覆盖默认）")
	})
	resetBtn := widget.NewButton("重置默认", func() {
		c.resetConfigToDefault()
		c.configInfo.SetText("已恢复默认值（未保存）")
	})

	toolbar := container.NewHBox(addBtn, saveBtn, reloadBtn, resetBtn)
	scroll := container.NewVScroll(c.configList)
	scroll.SetMinSize(fyne.NewSize(0, 400))

	return container.NewBorder(c.configInfo, toolbar, nil, nil, scroll)
}

func (c *Controller) Shutdown() { c.session.Close() }

func (c *Controller) handleFrame(h core.IHeader, payload []byte) {
	preview := formatPayloadPreview(payload)
	c.appendLog("[RX] major=%d sub=%d src=%d tgt=%d len=%d %s",
		h.Major(), h.SubProto(), h.SourceID(), h.TargetID(), len(payload), preview)
}

func (c *Controller) handleError(err error) {
	c.appendLog("[ERR] %v", err)
}

func (c *Controller) appendLog(format string, args ...any) {
	c.logMu.Lock()
	defer c.logMu.Unlock()
	c.logBuf.WriteString(time.Now().Format("15:04:05 "))
	c.logBuf.WriteString(fmt.Sprintf(format, args...))
	c.logBuf.WriteString("\n")
	text := c.logBuf.String()
	if c.logView != nil {
		c.logView.SetText(text)
		c.logView.CursorRow = strings.Count(text, "\n")
	}
	if c.logPopup != nil {
		c.logPopup.SetText(text)
		c.logPopup.CursorRow = strings.Count(text, "\n")
	}
}

// UI helpers
func (c *Controller) buildHeaderCard() fyne.CanvasObject {
	c.form = newHeaderForm()
	return widget.NewCard("Header", "编辑 TCP 头字段", c.form.container)
}

func (c *Controller) buildPayloadCard() fyne.CanvasObject {
	c.payload = widget.NewMultiLineEntry()
	c.payload.SetPlaceHolder("请输入 payload 文本")
	c.hexToggle = widget.NewCheck("使用十六进制输入", nil)
	return widget.NewCard("Payload", "", container.NewBorder(nil, c.hexToggle, nil, nil, c.payload))
}

func (c *Controller) buildLogCard() fyne.CanvasObject {
	c.logView = newLogEntry()
	return widget.NewCard("日志", "", c.logView)
}

func (c *Controller) buildPresetTab(w fyne.Window) fyne.CanvasObject {
	defs := presetDefinitions()
	list := container.NewVBox()
	for _, def := range defs {
		list.Add(c.buildPresetCard(def, w))
	}
	scroll := container.NewVScroll(list)
	scroll.SetMinSize(fyne.NewSize(0, 400))
	return scroll
}

// header form parsing
type headerForm struct {
	container fyne.CanvasObject
	major     *widget.Entry
	sub       *widget.Entry
	source    *widget.Entry
	target    *widget.Entry
	msgID     *widget.Entry
	flags     *widget.Entry
}

func newHeaderForm() *headerForm {
	form := &headerForm{
		major:  widget.NewEntry(),
		sub:    widget.NewEntry(),
		source: widget.NewEntry(),
		target: widget.NewEntry(),
		msgID:  widget.NewEntry(),
		flags:  widget.NewEntry(),
	}
	form.major.SetText("2")
	form.sub.SetText("1")
	form.source.SetText("1")
	form.msgID.SetText("1")
	// 紧凑网格布局，一行多字段
	form.container = container.NewGridWithColumns(3,
		labeledEntry("Major", form.major),
		labeledEntry("SubProto", form.sub),
		labeledEntry("SourceID", form.source),
		labeledEntry("TargetID", form.target),
		labeledEntry("MsgID", form.msgID),
		labeledEntry("Flags", form.flags),
	)
	return form
}

func labeledEntry(label string, entry *widget.Entry) fyne.CanvasObject {
	return container.NewVBox(widget.NewLabel(label), entry)
}

// valueOrPlaceholder returns entry text trimmed; if empty, returns placeholder trimmed.
func valueOrPlaceholder(e *widget.Entry) string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Text) != "" {
		return strings.TrimSpace(e.Text)
	}
	return strings.TrimSpace(e.PlaceHolder)
}

func (f *headerForm) Parse() (core.IHeader, error) {
	major, err := parseUint("Major", f.major.Text, 0, 3)
	if err != nil {
		return nil, err
	}
	sub, err := parseUint("SubProto", f.sub.Text, 0, 63)
	if err != nil {
		return nil, err
	}
	src, err := parseUint("SourceID", f.source.Text, 0, 1<<32-1)
	if err != nil {
		return nil, err
	}
	tgt, err := parseUint("TargetID", f.target.Text, 0, 1<<32-1)
	if err != nil {
		return nil, err
	}
	msgID, err := parseUint("MsgID", f.msgID.Text, 0, 1<<32-1)
	if err != nil {
		return nil, err
	}
	flags, err := parseUint("Flags", f.flags.Text, 0, 255)
	if err != nil {
		return nil, err
	}

	hdr := &header.HeaderTcp{}
	hdr.WithMajor(uint8(major)).
		WithSubProto(uint8(sub)).
		WithSourceID(uint32(src)).
		WithTargetID(uint32(tgt)).
		WithMsgID(uint32(msgID)).
		WithFlags(uint8(flags)).
		WithTimestamp(uint32(time.Now().Unix()))
	return hdr, nil
}

func parseUint(field, text string, min, max uint64) (uint64, error) {
	if strings.TrimSpace(text) == "" {
		return 0, fmt.Errorf("%s 不能为空", field)
	}
	v, err := strconv.ParseUint(strings.TrimSpace(text), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s 不是合法整数", field)
	}
	if v < min || v > max {
		return 0, fmt.Errorf("%s 超出范围[%d,%d]", field, min, max)
	}
	return v, nil
}

func (c *Controller) buildPayload() ([]byte, error) {
	if !c.hexToggle.Checked {
		return []byte(c.payload.Text), nil
	}
	cleaned := strings.ReplaceAll(strings.TrimSpace(c.payload.Text), " ", "")
	if cleaned == "" {
		return nil, nil
	}
	data, err := hex.DecodeString(cleaned)
	if err != nil {
		return nil, fmt.Errorf("十六进制解析失败: %w", err)
	}
	return data, nil
}

const (
	payloadPreviewLimit  = 256
	payloadTextRuneLimit = 128
)

func formatPayloadPreview(payload []byte) string {
	if len(payload) == 0 {
		return "payload=<empty>"
	}

	truncated := len(payload) > payloadPreviewLimit
	preview := payload
	if truncated {
		preview = payload[:payloadPreviewLimit]
	}

	textPart := buildTextPreview(preview)
	hexPart := bytesToSpacedHex(preview)

	suffix := ""
	if truncated {
		suffix = fmt.Sprintf("...(总长 %d bytes)", len(payload))
	}
	return fmt.Sprintf("payload=text(%s)%s hex=%s%s", textPart, suffix, hexPart,
		ternarySuffix(truncated))
}

func buildTextPreview(data []byte) string {
	if utf8.Valid(data) {
		runes := []rune(string(data))
		if len(runes) > payloadTextRuneLimit {
			return string(runes[:payloadTextRuneLimit]) + "..."
		}
		return string(runes)
	}
	var b strings.Builder
	for _, bt := range data {
		if bt >= 32 && bt <= 126 {
			b.WriteByte(bt)
			continue
		}
		b.WriteByte('.')
	}
	return b.String()
}

func bytesToSpacedHex(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	var b strings.Builder
	for i, bt := range data {
		if i > 0 && i%2 == 0 {
			b.WriteByte(' ')
		}
		b.WriteString(strings.ToUpper(fmt.Sprintf("%02x", bt)))
	}
	return b.String()
}

func ternarySuffix(cond bool) string {
	if cond {
		return " (截断)"
	}
	return ""
}

type logEntry struct {
	widget.Entry
}

func newLogEntry() *logEntry {
	e := &logEntry{}
	e.MultiLine = true
	e.Wrapping = fyne.TextWrapWord
	e.ExtendBaseWidget(e)
	return e
}

func (e *logEntry) TypedRune(r rune) {}

func (e *logEntry) TypedKey(event *fyne.KeyEvent) {
	switch event.Name {
	case fyne.KeyLeft, fyne.KeyRight, fyne.KeyUp, fyne.KeyDown,
		fyne.KeyHome, fyne.KeyEnd, fyne.KeyPageUp, fyne.KeyPageDown:
		e.Entry.TypedKey(event)
	}
}

func (e *logEntry) TypedShortcut(shortcut fyne.Shortcut) {
	switch shortcut.(type) {
	case *fyne.ShortcutCopy, *fyne.ShortcutSelectAll, *fyne.ShortcutPaste:
		// 允许复制/全选/粘贴（粘贴只改变剪贴板，不写入本区域）。
		e.Entry.TypedShortcut(shortcut)
	}
}

func (c *Controller) openLogWindow() {
	if c.app == nil {
		return
	}
	if c.logWindow != nil {
		c.logWindow.Show()
		c.logWindow.RequestFocus()
		return
	}
	c.logPopup = newLogEntry()
	c.logPopup.SetText(c.logBuf.String())
	c.logPopup.CursorRow = strings.Count(c.logBuf.String(), "\n")
	win := c.app.NewWindow("日志窗口")
	win.SetContent(container.NewBorder(nil, nil, nil, nil, c.logPopup))
	win.Resize(fyne.NewSize(700, 500))
	win.SetOnClosed(func() {
		c.logWindow = nil
		c.logPopup = nil
	})
	c.logWindow = win
	win.Show()
}

// Config UI helpers
type configRow struct {
	key   *widget.Entry
	value *widget.Entry
	card  fyne.CanvasObject
}

const prefConfigKey = "config.entries"

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
	coreconfig.KeyRoutingForwardRemote,
	coreconfig.KeyProcQueueStrategy,
	coreconfig.KeyDefaultForwardEnable,
	coreconfig.KeyDefaultForwardTarget,
	coreconfig.KeyDefaultForwardMap,
	coreconfig.KeyParentEnable,
	coreconfig.KeyParentAddr,
	coreconfig.KeyParentReconnectSec,
}

func (c *Controller) reloadConfigUI(includeBlank bool) {
	effective := c.loadEffectiveConfig()
	c.setConfigRows(effective, includeBlank)
}

func (c *Controller) resetConfigToDefault() {
	defaults := defaultConfigValues()
	c.setConfigRows(defaults, true)
}

func (c *Controller) setConfigRows(values map[string]string, includeBlank bool) {
	c.configRows = nil
	c.configList.Objects = nil
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		c.addConfigRow(k, values[k])
	}
	if includeBlank {
		c.addConfigRow("", "")
	}
	c.configList.Refresh()
}

func (c *Controller) addConfigRow(key, val string) {
	row := &configRow{
		key:   widget.NewEntry(),
		value: widget.NewEntry(),
	}
	row.key.SetText(key)
	row.value.SetText(val)
	row.card = c.buildConfigRowContainer(row)
	c.configRows = append(c.configRows, row)
	c.configList.Add(row.card)
	c.configList.Refresh()
}

func (c *Controller) removeConfigRow(target *configRow) {
	filtered := make([]*configRow, 0, len(c.configRows))
	for _, r := range c.configRows {
		if r != target {
			filtered = append(filtered, r)
		}
	}
	c.configRows = filtered
	c.refreshConfigListFromRows()
}

func (c *Controller) refreshConfigListFromRows() {
	c.configList.Objects = nil
	for _, r := range c.configRows {
		// rebuild container in case entries changed
		r.card = c.buildConfigRowContainer(r)
		c.configList.Add(r.card)
	}
	c.configList.Refresh()
}

func (c *Controller) buildConfigRowContainer(r *configRow) fyne.CanvasObject {
	removeBtn := widget.NewButton("删除", func() { c.removeConfigRow(r) })
	keyRow := container.NewVBox(
		widget.NewLabel("Key"),
		r.key,
	)
	valueRow := container.NewVBox(
		widget.NewLabel("Value"),
		container.NewBorder(nil, nil, nil, removeBtn, r.value),
	)
	return widget.NewCard("", "", container.NewVBox(keyRow, valueRow))
}

func (c *Controller) saveConfigFromUI() error {
	entries := make(map[string]string)
	for _, r := range c.configRows {
		k := strings.TrimSpace(r.key.Text)
		v := r.value.Text
		if k == "" {
			continue
		}
		entries[k] = v
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	if c.app != nil && c.app.Preferences() != nil {
		c.app.Preferences().SetString(prefConfigKey, string(data))
	}
	return nil
}

func (c *Controller) loadEffectiveConfig() map[string]string {
	defaults := defaultConfigValues()
	stored := c.readStoredConfig()
	out := make(map[string]string, len(defaults)+len(stored))
	for k, v := range defaults {
		out[k] = v
	}
	for k, v := range stored {
		out[k] = v
	}
	return out
}

func defaultConfigValues() map[string]string {
	cfg := coreconfig.NewMap(map[string]string{})
	out := make(map[string]string, len(configKeys))
	for _, k := range configKeys {
		if v, ok := cfg.Get(k); ok {
			out[k] = v
		}
	}
	return out
}

func (c *Controller) readStoredConfig() map[string]string {
	out := make(map[string]string)
	if c.app == nil || c.app.Preferences() == nil {
		return out
	}
	raw := c.app.Preferences().StringWithFallback(prefConfigKey, "")
	if strings.TrimSpace(raw) == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

// Preset UI helpers
type presetDefinition struct {
	name        string
	desc        string
	param1Label string
	param1Ph    string
	param2Label string
	param2Ph    string
	build       func(p1, p2 string) (core.IHeader, []byte, error)
}

func presetDefinitions() []presetDefinition {
	return []presetDefinition{
		{
			name:        "Echo",
			desc:        "SubProto=1，直接回显字符串（TargetID 可选）",
			param1Label: "参数1（消息文本）",
			param1Ph:    "hello",
			param2Label: "参数2（TargetID，可选）",
			param2Ph:    "0",
			build: func(p1, p2 string) (core.IHeader, []byte, error) {
				if strings.TrimSpace(p1) == "" {
					return nil, nil, fmt.Errorf("消息不能为空")
				}
				target, err := parseOptionalUint32(p2, "TargetID")
				if err != nil {
					return nil, nil, err
				}
				hdr := presetHeader(1, target)
				return hdr, []byte(p1), nil
			},
		},
		{
			name:        "Login",
			desc:        "SubProto=2，action=login，payload JSON",
			param1Label: "参数1（DeviceID）",
			param1Ph:    "device-001",
			param2Label: "参数2（Credential）",
			param2Ph:    "cred-token",
			build: func(p1, p2 string) (core.IHeader, []byte, error) {
				if strings.TrimSpace(p1) == "" || strings.TrimSpace(p2) == "" {
					return nil, nil, fmt.Errorf("DeviceID/Credential 不能为空")
				}
				payload, _ := json.Marshal(map[string]any{
					"action": "login",
					"data": map[string]any{
						"device_id":  p1,
						"credential": p2,
					},
				})
				hdr := presetHeader(2, 0)
				return hdr, payload, nil
			},
		},
		{
			name:        "Register",
			desc:        "SubProto=2，action=register，payload JSON",
			param1Label: "参数1（DeviceID）",
			param1Ph:    "device-001",
			param2Label: "参数2（预留，可空）",
			param2Ph:    "",
			build: func(p1, _ string) (core.IHeader, []byte, error) {
				if strings.TrimSpace(p1) == "" {
					return nil, nil, fmt.Errorf("DeviceID 不能为空")
				}
				payload, _ := json.Marshal(map[string]any{
					"action": "register",
					"data": map[string]any{
						"device_id": p1,
					},
				})
				hdr := presetHeader(2, 0)
				return hdr, payload, nil
			},
		},
		{
			name:        "VarSet",
			desc:        "SubProto=3，action=set，默认 public string",
			param1Label: "参数1（变量名）",
			param1Ph:    "foo",
			param2Label: "参数2（变量值）",
			param2Ph:    "bar",
			build: func(p1, p2 string) (core.IHeader, []byte, error) {
				if strings.TrimSpace(p1) == "" || strings.TrimSpace(p2) == "" {
					return nil, nil, fmt.Errorf("变量名/变量值不能为空")
				}
				payload, _ := json.Marshal(map[string]any{
					"action": "set",
					"data": map[string]any{
						"name":       p1,
						"value":      p2,
						"visibility": "public",
						"type":       "string",
					},
				})
				hdr := presetHeader(3, 0)
				return hdr, payload, nil
			},
		},
		{
			name:        "VarGet",
			desc:        "SubProto=3，action=get，获取变量值",
			param1Label: "参数1（变量名）",
			param1Ph:    "foo",
			param2Label: "参数2（预留，可空）",
			param2Ph:    "",
			build: func(p1, _ string) (core.IHeader, []byte, error) {
				if strings.TrimSpace(p1) == "" {
					return nil, nil, fmt.Errorf("变量名不能为空")
				}
				payload, _ := json.Marshal(map[string]any{
					"action": "get",
					"data": map[string]any{
						"name": p1,
					},
				})
				hdr := presetHeader(3, 0)
				return hdr, payload, nil
			},
		},
	}
}

func (c *Controller) buildPresetCard(def presetDefinition, w fyne.Window) fyne.CanvasObject {
	p1 := widget.NewEntry()
	p1.SetPlaceHolder(def.param1Ph)
	p2 := widget.NewEntry()
	p2.SetPlaceHolder(def.param2Ph)
	sendBtn := widget.NewButton("发送", func() {
		if err := c.sendPreset(def, p1.Text, p2.Text); err != nil {
			dialog.ShowError(err, w)
			return
		}
	})
	body := container.NewVBox(
		widget.NewLabel(def.param1Label),
		p1,
		widget.NewLabel(def.param2Label),
		p2,
		sendBtn,
	)
	return widget.NewCard(def.name, def.desc, body)
}

func (c *Controller) sendPreset(def presetDefinition, p1, p2 string) error {
	hdr, payload, err := def.build(p1, p2)
	if err != nil {
		return err
	}
	if err := c.session.Send(hdr, payload); err != nil {
		return err
	}
	c.appendLog("[TX preset] %s sub=%d len=%d", def.name, hdr.SubProto(), len(payload))
	return nil
}

func presetHeader(sub uint8, target uint32) *header.HeaderTcp {
	h := &header.HeaderTcp{}
	h.WithMajor(header.MajorCmd).
		WithSubProto(sub).
		WithSourceID(1).
		WithTargetID(target).
		WithMsgID(uint32(time.Now().UnixNano()))
	return h
}

func parseOptionalUint32(text, field string) (uint32, error) {
	if strings.TrimSpace(text) == "" {
		return 0, nil
	}
	v, err := strconv.ParseUint(strings.TrimSpace(text), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s 不是合法整数", field)
	}
	return uint32(v), nil
}
