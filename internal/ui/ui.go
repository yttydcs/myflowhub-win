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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const (
	actionLoginResp    = "login_resp"
	actionRegisterResp = "register_resp"
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

	nodeEntry      *widget.Entry
	hexToggle      *widget.Check
	truncToggle    *widget.Check
	showHex        *widget.Check
	presetSrc      *widget.Entry
	presetTgt      *widget.Entry
	homeAddr       *widget.Entry
	homeAutoCon    *widget.Check
	homeAutoLog    *widget.Check
	profileSelect  *widget.Select
	homeDevice     *widget.Entry
	homeCred       *widget.Label
	homeNode       *widget.Label
	homeHub        *widget.Label
	homeRole       *widget.Label
	homeLoginBtn   *widget.Button
	homeClearBtn   *widget.Button
	homeLastAddr   string
	homeConnCard   *widget.Card
	storedCred     string
	storedNode     uint32
	storedHub      uint32
	storedRole     string
	currentProfile string
	profiles       []string
	homeLoading    bool

	configRows []*configRow
	configList *fyne.Container
	configInfo *widget.Label

	presetCards []*widget.Card

	logPopup  *logEntry
	logWindow fyne.Window

	mainWin   fyne.Window
	connected bool

	varPoolKeys   []varKey
	varPoolData   map[varKey]varValue
	varPoolList   *fyne.Container
	varPoolTarget *widget.Entry
}

func New(app fyne.App, ctx context.Context) *Controller {
	c := &Controller{app: app, ctx: ctx}
	c.session = session.New(c.ctx, c.handleFrame, c.handleError)
	return c
}

func (c *Controller) Build(w fyne.Window) fyne.CanvasObject {
	c.mainWin = w
	c.initProfiles()
	c.loadVarPoolPrefs()
	homeTab := c.buildHomeTab(w)
	varPoolTab := c.buildVarPoolTab(w)
	debugTab := c.buildDebugTab(w)
	logTab := c.buildLogTab(w)
	configTab := c.buildConfigTab(w)
	presetTab := c.buildPresetTab(w)
	tabs := container.NewAppTabs(
		container.NewTabItem("首页", homeTab),
		container.NewTabItem("变量池", varPoolTab),
		container.NewTabItem("自定义调试", debugTab),
		container.NewTabItem("日志", logTab),
		container.NewTabItem("核心设置", configTab),
		container.NewTabItem("预设调试", presetTab),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	tabs.SelectTabIndex(0)
	c.tryAutoConnectLogin()
	return tabs
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

// migrateLegacyPrefs copies old无profile前缀的配置到默认profile，避免用户数据丢失。
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

func (c *Controller) buildProfileBar(w fyne.Window) fyne.CanvasObject {
	if len(c.profiles) == 0 {
		c.profiles = []string{defaultProfileName}
	}
	if strings.TrimSpace(c.currentProfile) == "" {
		c.currentProfile = c.profiles[0]
	}
	c.profileSelect = widget.NewSelect(c.profiles, func(name string) {
		c.switchProfile(strings.TrimSpace(name))
	})
	c.profileSelect.PlaceHolder = "选择配置"
	c.profileSelect.SetSelected(c.currentProfile)

	newEntry := widget.NewEntry()
	newEntry.SetPlaceHolder("新配置名")
	entryWrap := container.New(layout.NewGridWrapLayout(fyne.NewSize(180, newEntry.MinSize().Height)), newEntry)
	addBtn := widget.NewButton("新建配置", func() {
		name := strings.TrimSpace(newEntry.Text)
		if name == "" {
			dialog.ShowError(fmt.Errorf("配置名不能为空"), w)
			return
		}
		if contains(c.profiles, name) {
			c.switchProfile(name)
			return
		}
		c.profiles = append(c.profiles, name)
		c.saveProfiles(c.profiles, name)
		if c.profileSelect != nil {
			c.profileSelect.Options = c.profiles
			c.profileSelect.Refresh()
		}
		newEntry.SetText("")
		c.switchProfile(name)
	})

	bar := container.NewHBox(
		widget.NewLabel("配置/身份"),
		c.profileSelect,
		entryWrap,
		addBtn,
	)
	return widget.NewCard("配置切换", "选择或新建独立配置，便于多身份测试", bar)
}

func (c *Controller) switchProfile(name string) {
	name = strings.TrimSpace(name)
	if name == "" || name == c.currentProfile {
		return
	}
	if !contains(c.profiles, name) {
		c.profiles = append(c.profiles, name)
		if c.profileSelect != nil {
			c.profileSelect.Options = c.profiles
			c.profileSelect.Refresh()
		}
	}
	c.currentProfile = name
	c.saveProfiles(c.profiles, c.currentProfile)
	if c.profileSelect != nil {
		c.profileSelect.SetSelected(c.currentProfile)
	}

	// 重置状态与缓存
	c.session.Close()
	c.connected = false
	c.setHomeConnStatus(false, c.homeLastAddr)
	c.storedCred = ""
	c.storedNode = 0
	c.storedRole = ""
	c.varPoolKeys = nil
	c.varPoolData = make(map[varKey]varValue)

	// 重新加载当前配置数据
	c.loadHomePrefs()
	c.updateHomeInfo()
	c.loadVarPoolPrefs()
	c.refreshVarPoolUI()
	if c.configList != nil {
		c.reloadConfigUI(true)
	}
	c.appendLog("[INFO] 已切换配置: %s", c.currentProfile)
}

// tryAutoConnectLogin 在启动时根据用户偏好自动连接并按需登录。
func (c *Controller) tryAutoConnectLogin() {
	if c.homeAutoCon != nil && c.homeAutoCon.Checked {
		go c.homeConnect()
		return
	}
	// 已经连接且仅勾选自动登录时触发一次登录
	if c.homeAutoLog != nil && c.homeAutoLog.Checked && c.connected {
		go c.homeLogin()
	}
}

func (c *Controller) buildHomeTab(w fyne.Window) fyne.CanvasObject {
	profileBar := c.buildProfileBar(w)
	c.homeAddr = widget.NewEntry()
	c.homeAddr.SetPlaceHolder("127.0.0.1:9000")
	c.homeAutoCon = widget.NewCheck("自动连接", func(checked bool) {
		if c.homeLoading {
			return
		}
		c.saveHomeAuto()
		if checked {
			go c.homeConnect()
		}
	})
	c.homeDevice = widget.NewEntry()
	c.homeDevice.SetPlaceHolder("device-001")
	c.homeAutoLog = widget.NewCheck("自动登录", func(checked bool) {
		if c.homeLoading {
			return
		}
		c.saveHomeAuto()
		if checked && c.connected {
			go c.homeLogin()
		}
	})
	c.homeCred = widget.NewLabel("Credential: -")
	c.homeNode = widget.NewLabel("NodeID: -")
	c.homeHub = widget.NewLabel("HubID: -")
	c.homeRole = widget.NewLabel("Role: -")

	connectBtn := widget.NewButton("连接", func() { go c.homeConnect() })
	disconnectBtn := widget.NewButton("断开", func() { c.homeDisconnect() })
	c.homeLoginBtn = widget.NewButton("登录", func() { go c.homeLogin() })
	c.homeClearBtn = widget.NewButton("清除凭证", func() { c.clearCredential() })

	c.loadHomePrefs()
	c.updateHomeInfo()
	c.updateHomeLoginButton()
	c.setHomeConnStatus(false, "")

	connBar := container.NewBorder(nil, nil, nil, container.NewHBox(connectBtn, disconnectBtn, c.homeAutoCon), c.homeAddr)
	c.homeConnCard = widget.NewCard("连接", "未连接", connBar)
	loginInputs := container.NewVBox(
		widget.NewLabel("DeviceID"),
		c.homeDevice,
		container.NewHBox(c.homeLoginBtn, c.homeAutoLog),
	)
	infoBox := container.NewVBox(
		c.homeCred,
		c.homeNode,
		c.homeHub,
		c.homeRole,
	)
	statusCard := widget.NewCard("状态", "显示最近一次登录返回的信息（已持久化 credential）",
		container.NewBorder(nil, c.homeClearBtn, nil, nil, infoBox))
	return container.NewVBox(
		profileBar,
		c.homeConnCard,
		widget.NewCard("登录/注册", "输入 DeviceID，自动登录会在连接后自动触发", loginInputs),
		statusCard,
	)
}

func (c *Controller) loadHomePrefs() {
	if c.app == nil || c.app.Preferences() == nil {
		return
	}
	c.homeLoading = true
	defer func() { c.homeLoading = false }()
	p := c.app.Preferences()
	if c.homeAutoCon != nil {
		c.homeAutoCon.SetChecked(p.BoolWithFallback(c.prefKey(prefHomeAutoCon), false))
	}
	if c.homeAutoLog != nil {
		c.homeAutoLog.SetChecked(p.BoolWithFallback(c.prefKey(prefHomeAutoLog), false))
	}
	if c.homeDevice != nil {
		c.homeDevice.SetText(p.StringWithFallback(c.prefKey(prefHomeDeviceID), ""))
	}
	c.storedCred = p.StringWithFallback(c.prefKey(prefHomeCredential), "")
	c.storedNode = uint32(p.IntWithFallback(c.prefKey(prefHomeNodeID), 0))
	c.storedHub = uint32(p.IntWithFallback(c.prefKey(prefHomeHubID), 0))
	c.storedRole = p.StringWithFallback(c.prefKey(prefHomeRole), "")
}

func (c *Controller) saveHomeAuto() {
	if c.app == nil || c.app.Preferences() == nil {
		return
	}
	p := c.app.Preferences()
	if c.homeAutoCon != nil {
		p.SetBool(c.prefKey(prefHomeAutoCon), c.homeAutoCon.Checked)
	}
	if c.homeAutoLog != nil {
		p.SetBool(c.prefKey(prefHomeAutoLog), c.homeAutoLog.Checked)
	}
}

func (c *Controller) updateHomeInfo() {
	cred := c.storedCred
	if cred == "" {
		cred = "-"
	}
	nodeText := "-"
	if c.storedNode != 0 {
		nodeText = fmt.Sprintf("%d", c.storedNode)
	}
	hubText := "-"
	if c.storedHub != 0 {
		hubText = fmt.Sprintf("%d", c.storedHub)
	}
	role := c.storedRole
	if role == "" {
		role = "-"
	}
	if c.homeCred != nil {
		c.homeCred.SetText("Credential: " + cred)
	}
	if c.homeNode != nil {
		c.homeNode.SetText("NodeID: " + nodeText)
	}
	if c.homeHub != nil {
		c.homeHub.SetText("HubID: " + hubText)
	}
	if c.homeRole != nil {
		c.homeRole.SetText("Role: " + role)
	}
	c.updateHomeLoginButton()
}

func (c *Controller) setHomeConnStatus(connected bool, addr string) {
	status := "未连接"
	if connected {
		if strings.TrimSpace(addr) == "" {
			addr = "未知"
		}
		status = "已连接: " + addr
	}
	if c.homeConnCard != nil {
		c.homeConnCard.Subtitle = status
		c.homeConnCard.Refresh()
	}
}

func (c *Controller) clearCredential() {
	c.storedCred = ""
	c.storedNode = 0
	c.storedHub = 0
	c.storedRole = ""
	if c.app != nil && c.app.Preferences() != nil {
		p := c.app.Preferences()
		p.SetString(c.prefKey(prefHomeCredential), "")
		p.SetInt(c.prefKey(prefHomeNodeID), 0)
		p.SetInt(c.prefKey(prefHomeHubID), 0)
		p.SetString(c.prefKey(prefHomeRole), "")
	}
	c.updateHomeInfo()
	c.appendLog("[HOME] 已清除本地凭证")
}

func (c *Controller) showInfo(title, msg string) {
	if c.mainWin != nil {
		dialog.ShowInformation(title, msg, c.mainWin)
		return
	}
	if c.app != nil {
		if win := c.app.Driver().AllWindows(); len(win) > 0 {
			dialog.ShowInformation(title, msg, win[0])
			return
		}
	}
}

func (c *Controller) updateHomeLoginButton() {
	if c.homeLoginBtn == nil {
		return
	}
	if c.storedCred == "" {
		c.homeLoginBtn.SetText("注册")
	} else {
		c.homeLoginBtn.SetText("登录")
	}
}

func (c *Controller) homeConnect() {
	addr := valueOrPlaceholder(c.homeAddr)
	if addr == "" {
		c.appendLog("HOME 连接地址为空")
		return
	}
	if err := c.session.Connect(addr); err != nil {
		if strings.Contains(err.Error(), "已经连接") {
			c.connected = true
			c.appendLog("HOME 已连接")
			c.homeLastAddr = addr
			c.setHomeConnStatus(true, addr)
		} else {
			c.appendLog("HOME connect error: %v", err)
			return
		}
	}
	c.connected = true
	c.homeLastAddr = addr
	c.appendLog("HOME connected %s", addr)
	c.setHomeConnStatus(true, addr)
	if c.homeAutoLog != nil && c.homeAutoLog.Checked {
		go c.homeLogin()
	}
}

func (c *Controller) homeDisconnect() {
	c.session.Close()
	c.connected = false
	c.appendLog("HOME 手动断开")
	c.setHomeConnStatus(false, c.homeLastAddr)
}

func (c *Controller) homeLogin() {
	deviceID := strings.TrimSpace(valueOrPlaceholder(c.homeDevice))
	if deviceID == "" {
		c.appendLog("[HOME][ERR] DeviceID 不能为空")
		return
	}
	if c.app != nil && c.app.Preferences() != nil && deviceID != "" {
		c.app.Preferences().SetString(c.prefKey(prefHomeDeviceID), deviceID)
	}
	if c.storedCred == "" {
		c.homeRegister(deviceID)
		return
	}
	payload, err := json.Marshal(map[string]any{
		"action": "login",
		"data": map[string]any{
			"device_id":  deviceID,
			"credential": c.storedCred,
		},
	})
	if err != nil {
		c.appendLog("[HOME][ERR] build login payload: %v", err)
		return
	}
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(2).
		WithSourceID(0).
		WithTargetID(0).
		WithMsgID(uint32(time.Now().UnixNano())).
		WithTimestamp(uint32(time.Now().Unix()))
	if err := c.session.Send(hdr, payload); err != nil {
		c.appendLog("[HOME][ERR] login send: %v", err)
		return
	}
	if c.app != nil && c.app.Preferences() != nil {
		c.app.Preferences().SetString(c.prefKey(prefHomeDeviceID), deviceID)
	}
	c.logTx("[HOME TX login]", hdr, payload)
}

func (c *Controller) homeRegister(deviceID string) {
	payload, err := json.Marshal(map[string]any{
		"action": "register",
		"data": map[string]any{
			"device_id": deviceID,
		},
	})
	if err != nil {
		c.appendLog("[HOME][ERR] build register payload: %v", err)
		return
	}
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(2).
		WithSourceID(0).
		WithTargetID(0).
		WithMsgID(uint32(time.Now().UnixNano())).
		WithTimestamp(uint32(time.Now().Unix()))
	if err := c.session.Send(hdr, payload); err != nil {
		c.appendLog("[HOME][ERR] register send: %v", err)
		return
	}
	if c.app != nil && c.app.Preferences() != nil {
		c.app.Preferences().SetString(c.prefKey(prefHomeDeviceID), deviceID)
	}
	c.logTx("[HOME TX register]", hdr, payload)
}

func (c *Controller) persistCredential(deviceID string, nodeID uint32, credential, role string) {
	if credential != "" {
		c.storedCred = credential
	}
	if nodeID != 0 {
		c.storedNode = nodeID
	}
	// hub id在 login/register 响应中返回，存储由调用方处理
	if role != "" {
		c.storedRole = role
	}
	if c.app == nil || c.app.Preferences() == nil {
		return
	}
	p := c.app.Preferences()
	if credential != "" {
		p.SetString(c.prefKey(prefHomeCredential), credential)
	}
	if nodeID != 0 {
		p.SetInt(c.prefKey(prefHomeNodeID), int(nodeID))
	}
	if role != "" {
		p.SetString(c.prefKey(prefHomeRole), role)
	}
	if deviceID != "" {
		p.SetString(c.prefKey(prefHomeDeviceID), deviceID)
	}
	// 登录成功后拉取变量池
	c.fetchVarPoolAll()
}

func (c *Controller) handleAuthFrame(h core.IHeader, payload []byte) {
	if h == nil || h.SubProto() != 2 {
		return
	}
	var msg struct {
		Action string          `json:"action"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	act := strings.ToLower(strings.TrimSpace(msg.Action))
	switch act {
	case actionLoginResp:
		var resp struct {
			Code       int    `json:"code"`
			Msg        string `json:"msg"`
			DeviceID   string `json:"device_id"`
			NodeID     uint32 `json:"node_id"`
			HubID      uint32 `json:"hub_id"`
			Credential string `json:"credential"`
			Role       string `json:"role"`
		}
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			return
		}
		if resp.Code != 1 {
			return
		}
		c.storedHub = resp.HubID
		if resp.HubID != 0 && c.app != nil && c.app.Preferences() != nil {
			c.app.Preferences().SetInt(c.prefKey(prefHomeHubID), int(resp.HubID))
		}
		if c.varPoolTarget != nil && resp.HubID != 0 {
			c.varPoolTarget.SetText(fmt.Sprintf("%d", resp.HubID))
		}
		c.persistCredential(resp.DeviceID, resp.NodeID, resp.Credential, resp.Role)
		c.updateHomeInfo()
		if c.homeDevice != nil && resp.DeviceID != "" {
			c.homeDevice.SetText(resp.DeviceID)
		}
		c.showInfo("登录成功", fmt.Sprintf("DeviceID: %s\nNodeID: %d\nHubID: %d\nCredential: %s\nRole: %s",
			resp.DeviceID, resp.NodeID, resp.HubID, resp.Credential, resp.Role))
	case actionRegisterResp:
		var resp struct {
			Code       int    `json:"code"`
			Msg        string `json:"msg"`
			DeviceID   string `json:"device_id"`
			NodeID     uint32 `json:"node_id"`
			HubID      uint32 `json:"hub_id"`
			Credential string `json:"credential"`
			Role       string `json:"role"`
		}
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			return
		}
		if resp.Code != 1 {
			return
		}
		c.storedHub = resp.HubID
		if resp.HubID != 0 && c.app != nil && c.app.Preferences() != nil {
			c.app.Preferences().SetInt(c.prefKey(prefHomeHubID), int(resp.HubID))
		}
		if c.varPoolTarget != nil && resp.HubID != 0 {
			c.varPoolTarget.SetText(fmt.Sprintf("%d", resp.HubID))
		}
		c.persistCredential(resp.DeviceID, resp.NodeID, resp.Credential, resp.Role)
		c.updateHomeInfo()
		if c.homeDevice != nil && resp.DeviceID != "" {
			c.homeDevice.SetText(resp.DeviceID)
		}
		c.showInfo("注册成功", fmt.Sprintf("DeviceID: %s\nNodeID: %d\nHubID: %d\nCredential: %s\nRole: %s",
			resp.DeviceID, resp.NodeID, resp.HubID, resp.Credential, resp.Role))
	default:
		// no-op for other actions
	}
}

// handleVarStoreFrame processes varstore responses (SubProto=3).
func (c *Controller) handleVarStoreFrame(payload []byte) {
	var msg struct {
		Action string          `json:"action"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	act := strings.ToLower(strings.TrimSpace(msg.Action))
	if act != "get_resp" && act != "assist_get_resp" && act != "notify_update" {
		return
	}
	var resp struct {
		Code       int    `json:"code"`
		Msg        string `json:"msg"`
		Name       string `json:"name"`
		Value      string `json:"value"`
		Owner      uint32 `json:"owner"`
		Visibility string `json:"visibility"`
		Type       string `json:"type"`
	}
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return
	}
	if strings.TrimSpace(resp.Name) == "" {
		return
	}
	if act != "notify_update" && resp.Code != 1 {
		return
	}
	c.updateVarPoolValue(varKey{Name: resp.Name, Owner: resp.Owner}, varValue{
		value:      resp.Value,
		owner:      resp.Owner,
		visibility: resp.Visibility,
		typ:        resp.Type,
	})
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
		c.logTx("TX", hdr, payload)
	})

	topBar := container.NewBorder(nil, nil,
		widget.NewLabel("Server"),
		container.NewHBox(connectBtn, disconnectBtn),
		container.NewMax(c.addrEntry))

	content := container.NewVBox(headerCard, payloadCard)
	btns := container.NewHBox(sendBtn)
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
	preview := c.formatPayloadPreview(payload)
	c.appendLog("[RX] major=%d sub=%d src=%d tgt=%d len=%d %s",
		h.Major(), h.SubProto(), h.SourceID(), h.TargetID(), len(payload), preview)
	c.handleAuthFrame(h, payload)
	if h != nil && h.SubProto() == 3 {
		c.handleVarStoreFrame(payload)
	}
}

func (c *Controller) handleError(err error) {
	c.connected = false
	c.setHomeConnStatus(false, c.homeLastAddr)
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

func (c *Controller) logTx(tag string, hdr core.IHeader, payload []byte) {
	prefix := normalizeTag(tag)
	preview := c.formatPayloadPreview(payload)
	if hdr == nil {
		c.appendLog("%s header=<nil> %s", prefix, preview)
		return
	}
	c.appendLog("%s header{major=%d sub=%d src=%d tgt=%d msgID=%d flags=%d ts=%d len=%d} %s",
		prefix,
		hdr.Major(), hdr.SubProto(), hdr.SourceID(), hdr.TargetID(),
		hdr.GetMsgID(), hdr.GetFlags(), hdr.GetTimestamp(), len(payload),
		preview,
	)
}

func (c *Controller) formatPayloadPreview(payload []byte) string {
	truncate := true
	if c.truncToggle != nil {
		truncate = c.truncToggle.Checked
	}
	showHex := false
	if c.showHex != nil {
		showHex = c.showHex.Checked
	}
	maxBytes := payloadPreviewLimit
	textLimit := payloadTextRuneLimit
	if !truncate {
		maxBytes = -1  // 不截断
		textLimit = -1 // 不截断文本预览
	}
	return formatPayloadPreview(payload, maxBytes, textLimit, showHex)
}

func normalizeTag(tag string) string {
	t := strings.TrimSpace(tag)
	if t == "" {
		return "[TX]"
	}
	if strings.HasPrefix(t, "[") {
		return t
	}
	return "[" + t + "]"
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
	c.truncToggle = widget.NewCheck("截断日志预览", nil)
	c.truncToggle.SetChecked(true)
	c.showHex = widget.NewCheck("显示 Hex", nil)
	c.showHex.SetChecked(false)
	toggles := container.NewHBox(c.truncToggle, c.showHex)
	content := container.NewBorder(toggles, nil, nil, nil, c.logView)
	return widget.NewCard("日志", "", content)
}

func (c *Controller) buildLogTab(w fyne.Window) fyne.CanvasObject {
	logCard := c.buildLogCard()
	openLogBtn := widget.NewButton("弹出日志窗口", func() {
		c.openLogWindow()
	})
	return container.NewBorder(nil, openLogBtn, nil, nil, logCard)
}

func (c *Controller) buildPresetTab(w fyne.Window) fyne.CanvasObject {
	defs := presetDefinitions()
	list := container.NewVBox()
	for _, def := range defs {
		list.Add(c.buildPresetCard(def, w))
	}
	c.presetSrc = widget.NewEntry()
	c.presetSrc.SetPlaceHolder("1")
	c.presetTgt = widget.NewEntry()
	c.presetTgt.SetPlaceHolder("0")
	commonHeader := widget.NewCard("公共 Header", "空则使用占位值，应用于所有预设发送", container.NewGridWithColumns(2,
		labeledEntry("SourceID", c.presetSrc),
		labeledEntry("TargetID", c.presetTgt),
	))
	scroll := container.NewVScroll(list)
	scroll.SetMinSize(fyne.NewSize(0, 400))
	return container.NewVBox(commonHeader, scroll)
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

// prefKey applies current profile prefix to a preference key.
func (c *Controller) prefKey(key string) string {
	p := strings.TrimSpace(c.currentProfile)
	if p == "" {
		return key
	}
	return fmt.Sprintf("profile.%s.%s", p, key)
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
		return 0, fmt.Errorf("%s 不是合法数字", field)
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

func formatPayloadPreview(payload []byte, maxBytes int, textLimit int, showHex bool) string {
	if len(payload) == 0 {
		return "payload=<empty>"
	}

	truncated := false
	preview := payload
	if maxBytes >= 0 && len(payload) > maxBytes {
		preview = payload[:maxBytes]
		truncated = true
	}

	textPart := buildTextPreview(preview, textLimit)
	hexPart := ""
	if showHex {
		hexPart = " hex=" + bytesToSpacedHex(preview)
	}

	suffix := ""
	if truncated {
		suffix = fmt.Sprintf("...(总长 %d bytes)", len(payload))
	}
	return fmt.Sprintf("payload=text(%s)%s%s%s", textPart, suffix, hexPart, ternarySuffix(truncated))
}

func buildTextPreview(data []byte, limit int) string {
	if utf8.Valid(data) {
		runes := []rune(string(data))
		if limit >= 0 && len(runes) > limit {
			return string(runes[:limit]) + "..."
		}
		return string(runes)
	}
	var b strings.Builder
	for i, bt := range data {
		if limit >= 0 && i >= limit {
			b.WriteString("...")
			break
		}
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
		return " (已截断)"
	}
	return ""
}

type logEntry struct {
	widget.Entry
}

type varKey struct {
	Name  string `json:"name"`
	Owner uint32 `json:"owner,omitempty"`
}

type varValue struct {
	value      string
	owner      uint32
	visibility string
	typ        string
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
		// 允许复制/全选/粘贴，禁止其他快捷键修改内容
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
	removeBtn := widget.NewButton("鍒犻櫎", func() { c.removeConfigRow(r) })
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
		c.app.Preferences().SetString(c.prefKey(prefConfigKey), string(data))
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
	raw := c.app.Preferences().StringWithFallback(c.prefKey(prefConfigKey), "")
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
			desc:        "SubProto=1，回显字符串（TargetID 可选）",
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
		if err := c.sendPreset(def, valueOrPlaceholder(p1), valueOrPlaceholder(p2)); err != nil {
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
	if strings.EqualFold(def.name, "VarSet") {
		c.addVarPoolKey(varKey{Name: strings.TrimSpace(p1), Owner: c.storedNode})
	}
	hdr, err = c.applyPresetHeaderOverrides(hdr)
	if err != nil {
		return err
	}
	if err := c.session.Send(hdr, payload); err != nil {
		return err
	}
	c.logTx("[TX preset] "+def.name, hdr, payload)
	return nil
}

func (c *Controller) applyPresetHeaderOverrides(h core.IHeader) (core.IHeader, error) {
	if h == nil {
		return h, nil
	}
	if c.presetSrc != nil {
		if s := strings.TrimSpace(valueOrPlaceholder(c.presetSrc)); s != "" {
			v, err := parseOptionalUint32(s, "SourceID")
			if err != nil {
				return nil, err
			}
			h = h.WithSourceID(v)
		}
	}
	if c.presetTgt != nil {
		if s := strings.TrimSpace(valueOrPlaceholder(c.presetTgt)); s != "" {
			v, err := parseOptionalUint32(s, "TargetID")
			if err != nil {
				return nil, err
			}
			h = h.WithTargetID(v)
		}
	}
	return h, nil
}

// Var pool UI
func (c *Controller) buildVarPoolTab(w fyne.Window) fyne.CanvasObject {
	if c.varPoolList == nil {
		c.varPoolList = container.NewVBox()
	}
	c.refreshVarPoolUI()

	c.varPoolTarget = widget.NewEntry()
	c.varPoolTarget.SetPlaceHolder("路由 TargetID（留空=当前登录 HubID）")
	if c.storedHub != 0 {
		c.varPoolTarget.SetText(fmt.Sprintf("%d", c.storedHub))
	}
	refreshBtn := widget.NewButton("刷新全部", func() { c.fetchVarPoolAll() })
	addMineBtn := widget.NewButton("新增我的变量", func() { c.openAddMineDialog(w) })
	addWatchBtn := widget.NewButton("新增监视", func() { c.openAddWatchDialog(w) })
	actions := container.NewHBox(refreshBtn, addMineBtn, addWatchBtn)
	info := widget.NewLabel("按 owner 分组展示缓存变量，默认使用登录 HubID 进行 get")
	listScroll := container.NewVScroll(c.varPoolList)

	targetCard := widget.NewCard("查询 TargetID", "留空使用当前登录 HubID 进行 get/set", labeledEntry("TargetID", c.varPoolTarget))
	listArea := container.NewBorder(info, actions, nil, nil, listScroll)
	return container.NewBorder(targetCard, nil, nil, nil, listArea)
}

func (c *Controller) refreshVarPoolUI() {
	if c.varPoolList == nil {
		return
	}
	c.varPoolList.Objects = nil
	mine := make([]varKey, 0)
	others := make([]varKey, 0)
	for _, k := range c.varPoolKeys {
		if k.Owner != 0 && c.storedNode != 0 && k.Owner == c.storedNode {
			mine = append(mine, k)
		} else {
			others = append(others, k)
		}
	}
	addGroup := func(title string, keys []varKey, showPlaceholder bool) {
		header := widget.NewLabel(title)
		header.TextStyle = fyne.TextStyle{Bold: true}
		c.varPoolList.Add(header)
		if len(keys) == 0 {
			if showPlaceholder {
				c.varPoolList.Add(widget.NewLabel("暂无记录"))
			}
			return
		}
		for _, key := range keys {
			val := c.varPoolData[key]
			displayOwner := key.Owner
			if val.owner != 0 {
				displayOwner = val.owner
			}
			value := strings.TrimSpace(val.value)
			if value == "" {
				value = "-"
			}
			vis := strings.TrimSpace(val.visibility)
			if vis == "" {
				vis = "-"
			}
			typ := strings.TrimSpace(val.typ)
			if typ == "" {
				typ = "-"
			}
			meta := fmt.Sprintf("Owner=%d  Vis=%s  Type=%s", displayOwner, vis, typ)
			valueLabel := widget.NewLabel(value)

			var buttons []fyne.CanvasObject
			refreshBtn := widget.NewButton("刷新", func(k varKey) func() {
				return func() {
					targetID, err := c.parseVarTarget()
					if err != nil {
						c.appendLog("[VAR][ERR] parse target: %v", err)
						return
					}
					go c.sendVarGet(k, targetID)
				}
			}(key))
			buttons = append(buttons, refreshBtn)
			editBtn := widget.NewButton("修改", func(k varKey, v varValue) func() {
				return func() {
					c.openVarEditDialog(k, v)
				}
			}(key, val))
			buttons = append(buttons, editBtn)
			removeBtn := widget.NewButton("本地移除", func(k varKey) func() {
				return func() {
					c.removeVarPoolKey(k)
				}
			}(key))
			buttons = append(buttons, removeBtn)
			actionRow := container.NewHBox(buttons...)

			card := widget.NewCard(key.Name, meta, container.NewBorder(nil, actionRow, nil, nil, valueLabel))
			c.varPoolList.Add(card)
		}
	}
	addGroup("我的变量", mine, true)
	c.varPoolList.Add(widget.NewSeparator())
	addGroup("别人的变量", others, true)
	c.varPoolList.Refresh()
}

func (c *Controller) loadVarPoolPrefs() {
	if c.varPoolData == nil {
		c.varPoolData = make(map[varKey]varValue)
	}
	if c.app == nil || c.app.Preferences() == nil {
		return
	}
	raw := c.app.Preferences().StringWithFallback(c.prefKey(prefVarPoolNames), "")
	if strings.TrimSpace(raw) == "" {
		return
	}
	var keys []varKey
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		var names []string
		if err2 := json.Unmarshal([]byte(raw), &names); err2 != nil {
			return
		}
		for _, n := range names {
			n = strings.TrimSpace(n)
			if n == "" {
				continue
			}
			keys = append(keys, varKey{Name: n})
		}
	}
	seen := make(map[varKey]bool)
	for _, k := range keys {
		k = normalizeVarKey(k)
		if k.Name == "" || seen[k] {
			continue
		}
		seen[k] = true
		c.varPoolKeys = append(c.varPoolKeys, k)
	}
}

func (c *Controller) saveVarPoolPrefs() {
	if c.app == nil || c.app.Preferences() == nil {
		return
	}
	data, _ := json.Marshal(c.varPoolKeys)
	c.app.Preferences().SetString(c.prefKey(prefVarPoolNames), string(data))
}

func (c *Controller) addVarPoolKey(key varKey) {
	key, changed := c.upsertVarKey(key)
	if key.Name == "" {
		return
	}
	if changed {
		c.saveVarPoolPrefs()
	}
	c.refreshVarPoolUI()
}

func (c *Controller) upsertVarKey(key varKey) (varKey, bool) {
	key = normalizeVarKey(key)
	if key.Name == "" {
		return key, false
	}
	if c.varPoolData == nil {
		c.varPoolData = make(map[varKey]varValue)
	}
	for i, k := range c.varPoolKeys {
		if k == key {
			return key, false
		}
		if k.Name == key.Name && k.Owner == 0 && key.Owner != 0 {
			if val, ok := c.varPoolData[k]; ok {
				c.varPoolData[key] = val
			}
			delete(c.varPoolData, k)
			c.varPoolKeys[i] = key
			return key, true
		}
	}
	c.varPoolKeys = append(c.varPoolKeys, key)
	return key, true
}

func normalizeVarKey(key varKey) varKey {
	key.Name = strings.TrimSpace(key.Name)
	return key
}

func (c *Controller) openAddMineDialog(w fyne.Window) {
	win := w
	if win == nil {
		win = c.mainWin
		if win == nil && c.app != nil {
			if ws := c.app.Driver().AllWindows(); len(ws) > 0 {
				win = ws[0]
			}
		}
	}
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("变量名")
	valEntry := widget.NewEntry()
	valEntry.SetPlaceHolder("初始值")
	visSelect := widget.NewSelect([]string{"public", "private"}, nil)
	visSelect.SetSelected("public")
	content := container.NewVBox(
		widget.NewLabel("变量名"),
		nameEntry,
		widget.NewLabel("变量值"),
		valEntry,
		widget.NewLabel("可见性"),
		visSelect,
	)
	dialog.ShowCustomConfirm("新增我的变量", "保存", "取消", content, func(ok bool) {
		if !ok {
			return
		}
		name := strings.TrimSpace(valueOrPlaceholder(nameEntry))
		val := valEntry.Text
		vis := visSelect.Selected
		if vis == "" {
			vis = "public"
		}
		if name == "" {
			dialog.ShowError(fmt.Errorf("变量名不能为空"), win)
			return
		}
		if strings.TrimSpace(val) == "" {
			dialog.ShowError(fmt.Errorf("变量值不能为空"), win)
			return
		}
		if c.storedNode == 0 {
			dialog.ShowError(fmt.Errorf("请先登录获取 NodeID 后再新增变量"), win)
			return
		}
		targetID, err := c.parseVarTarget()
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		key := varKey{Name: name, Owner: c.storedNode}
		if err := c.sendVarSet(key, val, vis, targetID); err != nil {
			dialog.ShowError(err, win)
			return
		}
	}, win)
}

func (c *Controller) openAddWatchDialog(w fyne.Window) {
	win := w
	if win == nil {
		win = c.mainWin
		if win == nil && c.app != nil {
			if ws := c.app.Driver().AllWindows(); len(ws) > 0 {
				win = ws[0]
			}
		}
	}
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("变量名")
	ownerEntry := widget.NewEntry()
	ownerEntry.SetPlaceHolder("Owner NodeID")
	content := container.NewVBox(
		widget.NewLabel("变量名"),
		nameEntry,
		widget.NewLabel("Owner NodeID"),
		ownerEntry,
	)
	dialog.ShowCustomConfirm("新增监视变量", "保存", "取消", content, func(ok bool) {
		if !ok {
			return
		}
		name := strings.TrimSpace(valueOrPlaceholder(nameEntry))
		ownerText := strings.TrimSpace(ownerEntry.Text)
		if name == "" {
			dialog.ShowError(fmt.Errorf("变量名不能为空"), win)
			return
		}
		if ownerText == "" {
			dialog.ShowError(fmt.Errorf("Owner NodeID 不能为空"), win)
			return
		}
		ownerID, err := strconv.ParseUint(ownerText, 10, 32)
		if err != nil || ownerID == 0 {
			dialog.ShowError(fmt.Errorf("Owner NodeID 必须是正整数"), win)
			return
		}
		key := varKey{Name: name, Owner: uint32(ownerID)}
		c.addVarPoolKey(key)
		targetID, err := c.parseVarTarget()
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		go c.sendVarGet(key, targetID)
	}, win)
}

func (c *Controller) fetchVarPoolAll() {
	if len(c.varPoolKeys) == 0 {
		return
	}
	targetID, err := c.parseVarTarget()
	if err != nil {
		c.appendLog("[VAR][ERR] parse target: %v", err)
		return
	}
	for _, k := range c.varPoolKeys {
		key := k
		go c.sendVarGet(key, targetID)
	}
}

func (c *Controller) sendVarGet(key varKey, targetID uint32) {
	key = normalizeVarKey(key)
	if key.Name == "" {
		return
	}
	owner := key.Owner
	if owner == 0 {
		owner = c.storedNode
	}
	if owner == 0 {
		c.appendLog("[VAR][ERR] get %s: owner not set", key.Name)
		return
	}
	payload, err := json.Marshal(map[string]any{
		"action": "get",
		"data": map[string]any{
			"name":  key.Name,
			"owner": owner,
		},
	})
	if err != nil {
		c.appendLog("[VAR][ERR] build get payload: %v", err)
		return
	}
	sourceID := c.storedNode
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(3).
		WithSourceID(sourceID).
		WithTargetID(targetID).
		WithMsgID(uint32(time.Now().UnixNano())).
		WithTimestamp(uint32(time.Now().Unix()))
	if err := c.session.Send(hdr, payload); err != nil {
		c.appendLog("[VAR][ERR] get %s(owner=%d): %v", key.Name, owner, err)
		return
	}
	c.logTx(fmt.Sprintf("[VAR TX get %s#%d]", key.Name, owner), hdr, payload)
}

func (c *Controller) sendVarSet(key varKey, value, visibility string, targetID uint32) error {
	key = normalizeVarKey(key)
	if key.Name == "" {
		return fmt.Errorf("变量名不能为空")
	}
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("变量值不能为空")
	}
	if visibility == "" {
		visibility = "public"
	}
	if c.storedNode == 0 {
		return fmt.Errorf("请先登录获取 NodeID 后再新增变量")
	}
	owner := key.Owner
	if owner == 0 {
		owner = c.storedNode
	}
	payload, err := json.Marshal(map[string]any{
		"action": "set",
		"data": map[string]any{
			"name":       key.Name,
			"value":      value,
			"visibility": visibility,
			"type":       "string",
			"owner":      owner,
		},
	})
	if err != nil {
		return err
	}
	sourceID := c.storedNode
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(3).
		WithSourceID(sourceID).
		WithTargetID(targetID).
		WithMsgID(uint32(time.Now().UnixNano())).
		WithTimestamp(uint32(time.Now().Unix()))
	if err := c.session.Send(hdr, payload); err != nil {
		return err
	}
	ownerForCache := key.Owner
	if ownerForCache == 0 {
		ownerForCache = owner
	}
	cacheKey := varKey{Name: key.Name, Owner: ownerForCache}
	c.updateVarPoolValue(cacheKey, varValue{
		value:      value,
		owner:      ownerForCache,
		visibility: visibility,
		typ:        "string",
	})
	c.logTx(fmt.Sprintf("[VAR TX set %s#%d]", key.Name, ownerForCache), hdr, payload)
	return nil
}

func (c *Controller) updateVarPoolValue(key varKey, val varValue) {
	if c.varPoolData == nil {
		c.varPoolData = make(map[varKey]varValue)
	}
	key, changed := c.upsertVarKey(key)
	if key.Name == "" {
		return
	}
	existing, ok := c.varPoolData[key]
	merged := existing
	// 仅在新值存在时覆盖，否则保留旧值
	if val.value != "" || !ok {
		merged.value = val.value
	}
	if val.owner != 0 || !ok {
		merged.owner = val.owner
	}
	if strings.TrimSpace(val.visibility) != "" || !ok {
		merged.visibility = val.visibility
	}
	if strings.TrimSpace(val.typ) != "" || !ok {
		merged.typ = val.typ
	}
	c.varPoolData[key] = merged
	if changed {
		c.saveVarPoolPrefs()
	}
	// 尝试在主线程刷新 UI，避免后台回调直接操作组件
	if c.app != nil {
		if drv := c.app.Driver(); drv != nil {
			if runner, ok := drv.(interface{ RunOnMain(func()) }); ok {
				runner.RunOnMain(c.refreshVarPoolUI)
				return
			}
		}
	}
	c.refreshVarPoolUI()
}
func (c *Controller) removeVarPoolKey(key varKey) {
	key = normalizeVarKey(key)
	if key.Name == "" {
		return
	}
	filtered := make([]varKey, 0, len(c.varPoolKeys))
	for _, k := range c.varPoolKeys {
		if k != key {
			filtered = append(filtered, k)
		}
	}
	c.varPoolKeys = filtered
	delete(c.varPoolData, key)
	c.saveVarPoolPrefs()
	c.refreshVarPoolUI()
}

func (c *Controller) openVarEditDialog(key varKey, val varValue) {
	win := c.mainWin
	if win == nil && c.app != nil {
		if ws := c.app.Driver().AllWindows(); len(ws) > 0 {
			win = ws[0]
		}
	}
	if win == nil {
		return
	}
	valEntry := widget.NewEntry()
	valEntry.SetText(val.value)
	visSelect := widget.NewSelect([]string{"public", "private"}, nil)
	if strings.TrimSpace(val.visibility) != "" {
		visSelect.SetSelected(val.visibility)
	} else {
		visSelect.SetSelected("public")
	}
	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("变量名: %s", key.Name)),
		widget.NewLabel("变量值"),
		valEntry,
		widget.NewLabel("可见性（对他人节点不生效）"),
		visSelect,
	)
	dialog.ShowCustomConfirm("修改变量", "保存", "取消", content, func(ok bool) {
		if !ok {
			return
		}
		targetID, err := c.parseVarTarget()
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		value := valEntry.Text
		if strings.TrimSpace(value) == "" {
			dialog.ShowError(fmt.Errorf("变量值不能为空"), win)
			return
		}
		vis := visSelect.Selected
		if vis == "" {
			vis = "public"
		}
		if err := c.sendVarSet(key, value, vis, targetID); err != nil {
			dialog.ShowError(err, win)
			return
		}
	}, win)
}

func (c *Controller) parseVarTarget() (uint32, error) {
	if c.varPoolTarget == nil {
		if c.storedHub != 0 {
			return c.storedHub, nil
		}
		return 0, nil
	}
	text := strings.TrimSpace(c.varPoolTarget.Text)
	if text == "" {
		if c.storedHub != 0 {
			return c.storedHub, nil
		}
		return 0, nil
	}
	v, err := strconv.ParseUint(text, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("目标 NodeID 不是合法数字")
	}
	return uint32(v), nil
}

func contains(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
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
		return 0, fmt.Errorf("%s 不是合法数字", field)
	}
	return uint32(v), nil
}
