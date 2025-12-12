package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	core "github.com/yttydcs/myflowhub-core"
	"github.com/yttydcs/myflowhub-core/header"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

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
	c.homeNode = widget.NewLabel("NodeID: -")
	c.homeHub = widget.NewLabel("HubID: -")
	c.homeRole = widget.NewLabel("Role: -")

	connectBtn := widget.NewButton("连接", func() { go c.homeConnect() })
	disconnectBtn := widget.NewButton("断开", func() { c.homeDisconnect() })
	c.homeLoginBtn = widget.NewButton("登录", func() { go c.homeLogin() })
	c.homeClearBtn = widget.NewButton("清除登录信息", func() { c.clearAuthState() })

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
		c.homeNode,
		c.homeHub,
		c.homeRole,
	)
	statusCard := widget.NewCard("状态", "显示最近一次登录/注册返回的信息",
		container.NewBorder(nil, c.homeClearBtn, nil, nil, infoBox))
	body := container.NewVBox(
		profileBar,
		c.homeConnCard,
		widget.NewCard("登录/注册", "输入 DeviceID，自动登录会在连接后自动触发", loginInputs),
		statusCard,
	)
	return wrapScroll(body)
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
	c.storedNode = 0
	c.storedHub = 0
	c.storedRole = ""
	c.nodePriv = nil
	c.nodePubB64 = ""
	c.varPoolKeys = nil
	c.varPoolData = make(map[varKey]varValue)

	// 重新加载当前配置数据
	c.loadHomePrefs()
	c.updateHomeInfo()
	c.loadVarPoolPrefs()
	c.refreshVarPoolUI()
	c.refreshWindowTitle()
	if c.configList != nil {
		c.reloadConfigUI(true)
	}
	c.appendLog("[INFO] 已切换配置: %s", c.currentProfile)
}

func (c *Controller) tryAutoConnectLogin() {
	if c.homeAutoCon != nil && c.homeAutoCon.Checked {
		go c.homeConnect()
		return
	}
	if c.homeAutoLog != nil && c.homeAutoLog.Checked && c.connected {
		go c.homeLogin()
	}
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
	c.refreshVarPoolLoginInfo()
	c.refreshWindowTitle()
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
	c.refreshWindowTitle()
}

func (c *Controller) clearAuthState() {
	c.storedNode = 0
	c.storedHub = 0
	c.storedRole = ""
	if c.app != nil && c.app.Preferences() != nil {
		p := c.app.Preferences()
		p.SetInt(c.prefKey(prefHomeNodeID), 0)
		p.SetInt(c.prefKey(prefHomeHubID), 0)
		p.SetString(c.prefKey(prefHomeRole), "")
	}
	c.updateHomeInfo()
	c.appendLog("[HOME] 已清除本地登录信息")
	c.refreshWindowTitle()
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
	if c.storedNode == 0 {
		c.homeLoginBtn.SetText("注册")
		return
	}
	c.homeLoginBtn.SetText("登录")
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
	if err := c.ensureNodeKeys(); err != nil {
		c.appendLog("[HOME][ERR] 加载本地密钥失败: %v", err)
		return
	}
	// 未持有 NodeID 则先注册
	if c.storedNode == 0 {
		c.homeRegister(deviceID)
		return
	}
	ts := time.Now().Unix()
	nonce := generateNonce(12)
	sig, err := signLogin(c.nodePriv, deviceID, c.storedNode, ts, nonce)
	if err != nil {
		c.appendLog("[HOME][ERR] 构造签名失败: %v", err)
		return
	}
	payload, err := json.Marshal(map[string]any{
		"action": "login",
		"data": map[string]any{
			"device_id": deviceID,
			"node_id":   c.storedNode,
			"ts":        ts,
			"nonce":     nonce,
			"sig":       sig,
			"alg":       "ES256",
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
	if err := c.ensureNodeKeys(); err != nil {
		c.appendLog("[HOME][ERR] 加载本地密钥失败: %v", err)
		return
	}
	payload, err := json.Marshal(map[string]any{
		"action": "register",
		"data": map[string]any{
			"device_id": deviceID,
			"pubkey":    c.nodePubB64,
			"node_pub":  c.nodePubB64,
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

func (c *Controller) persistAuthState(deviceID string, nodeID, hubID uint32, role string) {
	if nodeID != 0 {
		c.storedNode = nodeID
	}
	if hubID != 0 {
		c.storedHub = hubID
	}
	if role != "" {
		c.storedRole = role
	}
	if c.app == nil || c.app.Preferences() == nil {
		return
	}
	p := c.app.Preferences()
	if nodeID != 0 {
		p.SetInt(c.prefKey(prefHomeNodeID), int(nodeID))
	}
	if hubID != 0 {
		p.SetInt(c.prefKey(prefHomeHubID), int(hubID))
	}
	if role != "" {
		p.SetString(c.prefKey(prefHomeRole), role)
	}
	if deviceID != "" {
		p.SetString(c.prefKey(prefHomeDeviceID), deviceID)
	}
	c.fetchVarPoolAll()
	c.refreshVarPoolLoginInfo()
	c.refreshWindowTitle()
}

// handleAuthFrame 处理 SubProto=2 登录/注册响应。
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
			Code     int    `json:"code"`
			Msg      string `json:"msg"`
			DeviceID string `json:"device_id"`
			NodeID   uint32 `json:"node_id"`
			HubID    uint32 `json:"hub_id"`
			Role     string `json:"role"`
			PubKey   string `json:"pubkey"`
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
		c.persistAuthState(resp.DeviceID, resp.NodeID, resp.HubID, resp.Role)
		c.updateHomeInfo()
		if c.homeDevice != nil && resp.DeviceID != "" {
			c.homeDevice.SetText(resp.DeviceID)
		}
		c.showInfo("登录成功", fmt.Sprintf("DeviceID: %s\nNodeID: %d\nHubID: %d\nRole: %s",
			resp.DeviceID, resp.NodeID, resp.HubID, resp.Role))
	case actionRegisterResp:
		var resp struct {
			Code     int    `json:"code"`
			Msg      string `json:"msg"`
			DeviceID string `json:"device_id"`
			NodeID   uint32 `json:"node_id"`
			HubID    uint32 `json:"hub_id"`
			Role     string `json:"role"`
			PubKey   string `json:"pubkey"`
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
		c.persistAuthState(resp.DeviceID, resp.NodeID, resp.HubID, resp.Role)
		c.updateHomeInfo()
		if c.homeDevice != nil && resp.DeviceID != "" {
			c.homeDevice.SetText(resp.DeviceID)
		}
		c.showInfo("注册成功", fmt.Sprintf("DeviceID: %s\nNodeID: %d\nHubID: %d\nRole: %s",
			resp.DeviceID, resp.NodeID, resp.HubID, resp.Role))
	default:
	}
}
