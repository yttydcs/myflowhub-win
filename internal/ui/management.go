package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	core "github.com/yttydcs/myflowhub-core"
	"github.com/yttydcs/myflowhub-core/header"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) buildManagementTab(w fyne.Window) fyne.CanvasObject {
	c.mgmtInfo = widget.NewLabel("显示当前节点直接连接的 NodeID 列表")
	c.mgmtList = widget.NewList(
		func() int { return len(c.mgmtNodes) },
		func() fyne.CanvasObject { return newMgmtNodeItem(c) },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(c.mgmtNodes) {
				return
			}
			if item, ok := obj.(*mgmtNodeItem); ok {
				item.setEntry(c.mgmtNodes[id])
			}
		},
	)
	c.mgmtTarget = widget.NewEntry()
	c.mgmtTarget.SetPlaceHolder("TargetID")
	if c.storedHub != 0 {
		c.mgmtTarget.SetText(fmt.Sprintf("%d", c.storedHub))
		c.mgmtLastTarget = c.storedHub
	}
	refreshBtn := widget.NewButton("刷新直接连接", func() { go c.fetchMgmtNodes() })
	subtreeBtn := widget.NewButton("刷新子树", func() { go c.fetchMgmtSubtree() })
	targetWrap := container.New(layout.NewGridWrapLayout(fyne.NewSize(120, c.mgmtTarget.MinSize().Height)), c.mgmtTarget)
	controls := container.NewHBox(widget.NewLabel("Target"), targetWrap, refreshBtn, subtreeBtn)
	header := container.NewBorder(nil, nil, nil, controls, c.mgmtInfo)
	body := container.NewBorder(header, nil, nil, nil, c.mgmtList)
	return wrapScroll(body)
}

func (c *Controller) fetchMgmtNodes() {
	target, err := c.parseMgmtTarget()
	if err != nil {
		c.appendLog("[MGMT][ERR] parse target: %v", err)
		return
	}
	payload, err := json.Marshal(map[string]any{
		"action": "list_nodes",
		"data":   map[string]any{},
	})
	if err != nil {
		c.appendLog("[MGMT][ERR] build list_nodes payload: %v", err)
		return
	}
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(1).
		WithSourceID(c.storedNode).
		WithTargetID(target).
		WithMsgID(uint32(time.Now().UnixNano())).
		WithTimestamp(uint32(time.Now().Unix()))
	if err := c.session.Send(hdr, payload); err != nil {
		c.appendLog("[MGMT][ERR] list_nodes send: %v", err)
		return
	}
	c.logTx("[MGMT TX list_nodes]", hdr, payload)
}

func (c *Controller) fetchMgmtSubtree() {
	target, err := c.parseMgmtTarget()
	if err != nil {
		c.appendLog("[MGMT][ERR] parse target: %v", err)
		return
	}
	payload, err := json.Marshal(map[string]any{
		"action": "list_subtree",
		"data":   map[string]any{},
	})
	if err != nil {
		c.appendLog("[MGMT][ERR] build list_subtree payload: %v", err)
		return
	}
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(1).
		WithSourceID(c.storedNode).
		WithTargetID(target).
		WithMsgID(uint32(time.Now().UnixNano())).
		WithTimestamp(uint32(time.Now().Unix()))
	if err := c.session.Send(hdr, payload); err != nil {
		c.appendLog("[MGMT][ERR] list_subtree send: %v", err)
		return
	}
	c.logTx("[MGMT TX list_subtree]", hdr, payload)
}

func (c *Controller) handleManagementFrame(h core.IHeader, payload []byte) {
	var msg struct {
		Action string          `json:"action"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	act := strings.ToLower(strings.TrimSpace(msg.Action))
	switch act {
	case "list_nodes_resp":
		var resp struct {
			Code  int    `json:"code"`
			Msg   string `json:"msg"`
			Nodes []struct {
				NodeID      uint32 `json:"node_id"`
				HasChildren bool   `json:"has_children"`
			} `json:"nodes"`
		}
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			return
		}
		if resp.Code != 1 {
			c.appendLog("[MGMT][WARN] list_nodes resp code=%d msg=%s", resp.Code, resp.Msg)
			return
		}
		c.updateMgmtNodes(resp.Nodes, false)
	case "list_subtree_resp":
		var resp struct {
			Code  int    `json:"code"`
			Msg   string `json:"msg"`
			Nodes []struct {
				NodeID      uint32 `json:"node_id"`
				HasChildren bool   `json:"has_children"`
			} `json:"nodes"`
		}
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			return
		}
		if resp.Code != 1 {
			c.appendLog("[MGMT][WARN] list_subtree resp code=%d msg=%s", resp.Code, resp.Msg)
			return
		}
		c.updateMgmtNodes(resp.Nodes, true)
	case "config_list_resp":
		var resp struct {
			Code int      `json:"code"`
			Msg  string   `json:"msg"`
			Keys []string `json:"keys"`
		}
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			return
		}
		if resp.Code != 1 {
			c.appendLog("[MGMT][WARN] config_list resp code=%d msg=%s", resp.Code, resp.Msg)
			return
		}
		if c.mgmtCfgTarget == 0 || (h != nil && h.SourceID() != 0 && h.SourceID() != c.mgmtCfgTarget) {
			return
		}
		c.mgmtCfgKeys = resp.Keys
		c.mgmtCfgValues = make(map[string]string)
		c.refreshMgmtConfigUI()
		for _, k := range resp.Keys {
			c.sendMgmtConfigGet(c.mgmtCfgTarget, strings.TrimSpace(k))
		}
	case "config_get_resp":
		var resp struct {
			Code  int    `json:"code"`
			Msg   string `json:"msg"`
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			return
		}
		if resp.Code != 1 {
			return
		}
		if c.mgmtCfgTarget == 0 || (h != nil && h.SourceID() != 0 && h.SourceID() != c.mgmtCfgTarget) {
			return
		}
		if c.mgmtCfgValues == nil {
			c.mgmtCfgValues = make(map[string]string)
		}
		c.mgmtCfgValues[strings.TrimSpace(resp.Key)] = resp.Value
		c.refreshMgmtConfigUI()
	}
}

func (c *Controller) updateMgmtNodes(nodes []struct {
	NodeID      uint32 `json:"node_id"`
	HasChildren bool   `json:"has_children"`
}, subtree bool) {
	target, _ := c.parseMgmtTarget()
	entries := make([]mgmtNodeEntry, 0, len(nodes))
	for _, n := range nodes {
		if n.NodeID == 0 {
			continue
		}
		entries = append(entries, mgmtNodeEntry{ID: n.NodeID, HasChildren: n.HasChildren})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	if target != 0 {
		c.mgmtLastTarget = target
		// move target to front if exists
		for i, e := range entries {
			if e.ID == target {
				entries = append([]mgmtNodeEntry{e}, append(entries[:i], entries[i+1:]...)...)
				break
			}
		}
	}
	c.mgmtNodes = entries
	if c.app != nil {
		if drv := c.app.Driver(); drv != nil {
			if runner, ok := drv.(interface{ RunOnMain(func()) }); ok {
				runner.RunOnMain(func() {
					if c.mgmtList != nil {
						c.mgmtList.Refresh()
					}
				})
				return
			}
		}
	}
	if c.mgmtList != nil {
		c.mgmtList.Refresh()
	}
}

func (c *Controller) parseMgmtTarget() (uint32, error) {
	if c.mgmtTarget == nil {
		if c.storedHub != 0 {
			return c.storedHub, nil
		}
		return 0, nil
	}
	text := strings.TrimSpace(c.mgmtTarget.Text)
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

// mgmt node list item with context menu
type mgmtNodeItem struct {
	widget.Label
	entry mgmtNodeEntry
	ctrl  *Controller
}

func newMgmtNodeItem(c *Controller) *mgmtNodeItem {
	item := &mgmtNodeItem{ctrl: c}
	item.ExtendBaseWidget(item)
	return item
}

func (i *mgmtNodeItem) setEntry(e mgmtNodeEntry) {
	i.entry = e
	tag := ""
	if e.HasChildren {
		tag = " (HasChildren)"
	}
	i.SetText(fmt.Sprintf("NodeID: %d%s", e.ID, tag))
}

func (i *mgmtNodeItem) Tapped(_ *fyne.PointEvent) {}

func (i *mgmtNodeItem) TappedSecondary(ev *fyne.PointEvent) {
	if i.ctrl == nil || i.ctrl.mainWin == nil {
		return
	}
	i.ctrl.showMgmtNodeMenu(i.entry, ev.AbsolutePosition)
}

func (c *Controller) showMgmtNodeMenu(entry mgmtNodeEntry, pos fyne.Position) {
	if entry.ID == 0 || c.mainWin == nil {
		return
	}
	cfgItem := fyne.NewMenuItem("查看配置", func() {
		c.openMgmtConfigWindow(entry.ID)
	})
	menu := fyne.NewMenu("", cfgItem)
	pop := widget.NewPopUpMenu(menu, c.mainWin.Canvas())
	pop.Move(pos)
	pop.Show()
}

func (c *Controller) openMgmtConfigWindow(target uint32) {
	if target == 0 || c.session == nil {
		return
	}
	c.mgmtCfgTarget = target
	c.mgmtCfgKeys = nil
	c.mgmtCfgValues = make(map[string]string)
	list := widget.NewList(
		func() int { return len(c.mgmtCfgKeys) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(c.mgmtCfgKeys) {
				return
			}
			key := c.mgmtCfgKeys[id]
			val := c.mgmtCfgValues[key]
			if lbl, ok := obj.(*widget.Label); ok {
				lbl.SetText(fmt.Sprintf("%s: %s", key, val))
			}
		},
	)
	c.mgmtCfgList = list
	refreshBtn := widget.NewButton("刷新配置", func() { go c.sendMgmtConfigList(target) })
	header := container.NewBorder(nil, nil, nil, refreshBtn, widget.NewLabel(fmt.Sprintf("Node %d 配置", target)))
	content := container.NewBorder(header, nil, nil, nil, list)
	win := fyne.CurrentApp().NewWindow(fmt.Sprintf("Node %d 配置", target))
	c.mgmtCfgWin = win
	win.SetContent(content)
	win.Resize(fyne.NewSize(400, 500))
	go c.sendMgmtConfigList(target)
	win.Show()
}

func (c *Controller) refreshMgmtConfigUI() {
	if c.mgmtCfgList != nil {
		c.mgmtCfgList.Refresh()
	}
}

func (c *Controller) sendMgmtConfigList(target uint32) {
	if target == 0 {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"action": "config_list",
		"data":   map[string]any{},
	})
	if err != nil {
		c.appendLog("[MGMT][ERR] build config_list: %v", err)
		return
	}
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(1).
		WithSourceID(c.storedNode).
		WithTargetID(target).
		WithMsgID(uint32(time.Now().UnixNano())).
		WithTimestamp(uint32(time.Now().Unix()))
	if err := c.session.Send(hdr, payload); err != nil {
		c.appendLog("[MGMT][ERR] send config_list: %v", err)
		return
	}
	c.logTx(fmt.Sprintf("[MGMT TX config_list target=%d]", target), hdr, payload)
}

func (c *Controller) sendMgmtConfigGet(target uint32, key string) {
	if target == 0 || strings.TrimSpace(key) == "" {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"action": "config_get",
		"data":   map[string]any{"key": key},
	})
	if err != nil {
		c.appendLog("[MGMT][ERR] build config_get: %v", err)
		return
	}
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(1).
		WithSourceID(c.storedNode).
		WithTargetID(target).
		WithMsgID(uint32(time.Now().UnixNano())).
		WithTimestamp(uint32(time.Now().Unix()))
	if err := c.session.Send(hdr, payload); err != nil {
		c.appendLog("[MGMT][ERR] send config_get %s: %v", key, err)
		return
	}
	c.logTx(fmt.Sprintf("[MGMT TX config_get %s target=%d]", key, target), hdr, payload)
}
