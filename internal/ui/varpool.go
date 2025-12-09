package ui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/yttydcs/myflowhub-core/header"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

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
	return wrapScroll(container.NewBorder(targetCard, nil, nil, nil, listArea))
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
			revokeBtn := widget.NewButton("撤销", func(k varKey) func() {
				return func() {
					targetID, err := c.parseVarTarget()
					if err != nil {
						c.appendLog("[VAR][ERR] parse target: %v", err)
						return
					}
					go c.sendVarRevoke(k, targetID)
				}
			}(key))
			buttons = append(buttons, revokeBtn)
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
		if c.storedNode != 0 && k.Owner == c.storedNode {
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
	filtered := make([]varKey, 0, len(c.varPoolKeys))
	for _, k := range c.varPoolKeys {
		if c.storedNode != 0 && k.Owner == c.storedNode {
			continue
		}
		filtered = append(filtered, k)
	}
	data, _ := json.Marshal(filtered)
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
	win := resolveWindow(c.app, c.mainWin, w)
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
	win := resolveWindow(c.app, c.mainWin, w)
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
	targetID, err := c.parseVarTarget()
	if err != nil {
		c.appendLog("[VAR][ERR] parse target: %v", err)
		return
	}
	if c.storedNode != 0 {
		go c.sendVarList(c.storedNode, targetID)
	}
	if len(c.varPoolKeys) == 0 {
		return
	}
	for _, k := range c.varPoolKeys {
		if c.storedNode != 0 && k.Owner == c.storedNode {
			continue
		}
		key := k
		go c.sendVarGet(key, targetID)
	}
}

func (c *Controller) sendVarList(owner uint32, targetID uint32) {
	if owner == 0 {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"action": "list",
		"data": map[string]any{
			"owner": owner,
		},
	})
	if err != nil {
		c.appendLog("[VAR][ERR] build list payload: %v", err)
		return
	}
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(3).
		WithSourceID(c.storedNode).
		WithTargetID(targetID).
		WithMsgID(uint32(time.Now().UnixNano())).
		WithTimestamp(uint32(time.Now().Unix()))
	if err := c.session.Send(hdr, payload); err != nil {
		c.appendLog("[VAR][ERR] list owner=%d: %v", owner, err)
		return
	}
	c.logTx(fmt.Sprintf("[VAR TX list owner=%d]", owner), hdr, payload)
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

func (c *Controller) sendVarRevoke(key varKey, targetID uint32) {
	key = normalizeVarKey(key)
	if key.Name == "" {
		return
	}
	owner := key.Owner
	if owner == 0 {
		owner = c.storedNode
	}
	if owner == 0 {
		c.appendLog("[VAR][ERR] revoke %s: owner not set", key.Name)
		return
	}
	payload, err := json.Marshal(map[string]any{
		"action": "revoke",
		"data": map[string]any{
			"name":  key.Name,
			"owner": owner,
		},
	})
	if err != nil {
		c.appendLog("[VAR][ERR] build revoke payload: %v", err)
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
		c.appendLog("[VAR][ERR] revoke %s(owner=%d): %v", key.Name, owner, err)
		return
	}
	c.logTx(fmt.Sprintf("[VAR TX revoke %s#%d]", key.Name, owner), hdr, payload)
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
	win := resolveWindow(c.app, c.mainWin, nil)
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

func (c *Controller) handleVarStoreFrame(payload []byte) {
	var msg struct {
		Action string          `json:"action"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	act := strings.ToLower(strings.TrimSpace(msg.Action))
	var resp struct {
		Code       int      `json:"code"`
		Msg        string   `json:"msg"`
		Name       string   `json:"name"`
		Value      string   `json:"value"`
		Owner      uint32   `json:"owner"`
		Visibility string   `json:"visibility"`
		Type       string   `json:"type"`
		Names      []string `json:"names"`
	}
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return
	}
	switch act {
	case "list_resp", "assist_list_resp":
		c.handleVarListResp(resp)
	case "get_resp", "assist_get_resp", "notify_set", "set_resp", "assist_set_resp":
		name := strings.TrimSpace(resp.Name)
		if name == "" {
			return
		}
		if (act == "get_resp" || act == "assist_get_resp" || act == "set_resp" || act == "assist_set_resp") && resp.Code != 1 {
			if resp.Owner != 0 && resp.Owner == c.storedNode {
				c.removeVarPoolKey(varKey{Name: name, Owner: resp.Owner})
			}
			return
		}
		c.updateVarPoolValue(varKey{Name: resp.Name, Owner: resp.Owner}, varValue{
			value:      resp.Value,
			owner:      resp.Owner,
			visibility: resp.Visibility,
			typ:        resp.Type,
		})
	case "revoke_resp", "assist_revoke_resp", "notify_revoke":
		c.handleVarRevokeResp(act, resp)
	default:
	}
}

func (c *Controller) handleVarListResp(resp struct {
	Code       int      `json:"code"`
	Msg        string   `json:"msg"`
	Name       string   `json:"name"`
	Value      string   `json:"value"`
	Owner      uint32   `json:"owner"`
	Visibility string   `json:"visibility"`
	Type       string   `json:"type"`
	Names      []string `json:"names"`
}) {
	if resp.Code != 1 || resp.Owner == 0 {
		return
	}
	if c.storedNode != 0 && resp.Owner != c.storedNode {
		return
	}
	filtered := make([]varKey, 0, len(c.varPoolKeys))
	for _, k := range c.varPoolKeys {
		if resp.Owner != 0 && k.Owner == resp.Owner {
			continue
		}
		filtered = append(filtered, k)
	}
	for _, name := range resp.Names {
		n := strings.TrimSpace(name)
		if n == "" {
			continue
		}
		filtered = append(filtered, varKey{Name: n, Owner: resp.Owner})
	}
	c.varPoolKeys = filtered
	c.refreshVarPoolUI()

	targetID, err := c.parseVarTarget()
	if err != nil {
		c.appendLog("[VAR][ERR] parse target after list: %v", err)
		return
	}
	for _, name := range resp.Names {
		n := strings.TrimSpace(name)
		if n == "" {
			continue
		}
		go c.sendVarGet(varKey{Name: n, Owner: resp.Owner}, targetID)
	}
}

func (c *Controller) handleVarRevokeResp(action string, resp struct {
	Code       int      `json:"code"`
	Msg        string   `json:"msg"`
	Name       string   `json:"name"`
	Value      string   `json:"value"`
	Owner      uint32   `json:"owner"`
	Visibility string   `json:"visibility"`
	Type       string   `json:"type"`
	Names      []string `json:"names"`
}) {
	name := strings.TrimSpace(resp.Name)
	if name == "" {
		return
	}
	if action != "notify_revoke" && resp.Code != 1 {
		c.appendLog("[VAR][WARN] revoke %s#%d failed code=%d msg=%s", name, resp.Owner, resp.Code, resp.Msg)
		return
	}
	c.removeVarPoolKey(varKey{Name: name, Owner: resp.Owner})
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
