package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yttydcs/myflowhub-core/header"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) buildManagementTab(w fyne.Window) fyne.CanvasObject {
	c.mgmtInfo = widget.NewLabel("显示当前节点直接连接的 NodeID 列表")
	c.mgmtList = widget.NewList(
		func() int { return len(c.mgmtNodes) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(c.mgmtNodes) {
				return
			}
			if lbl, ok := obj.(*widget.Label); ok {
				entry := c.mgmtNodes[id]
				tag := ""
				if entry.HasChildren {
					tag = " (HasChildren)"
				}
				lbl.SetText(fmt.Sprintf("NodeID: %d%s", entry.ID, tag))
			}
		},
	)
	refreshBtn := widget.NewButton("刷新直接连接", func() { go c.fetchMgmtNodes() })
	subtreeBtn := widget.NewButton("刷新子树", func() { go c.fetchMgmtSubtree() })
	header := container.NewBorder(nil, nil, nil, container.NewHBox(refreshBtn, subtreeBtn), c.mgmtInfo)
	return wrapScroll(container.NewBorder(header, nil, nil, nil, c.mgmtList))
}

func (c *Controller) fetchMgmtNodes() {
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
		WithTargetID(c.storedHub).
		WithMsgID(uint32(time.Now().UnixNano())).
		WithTimestamp(uint32(time.Now().Unix()))
	if err := c.session.Send(hdr, payload); err != nil {
		c.appendLog("[MGMT][ERR] list_nodes send: %v", err)
		return
	}
	c.logTx("[MGMT TX list_nodes]", hdr, payload)
}

func (c *Controller) fetchMgmtSubtree() {
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
		WithTargetID(c.storedHub).
		WithMsgID(uint32(time.Now().UnixNano())).
		WithTimestamp(uint32(time.Now().Unix()))
	if err := c.session.Send(hdr, payload); err != nil {
		c.appendLog("[MGMT][ERR] list_subtree send: %v", err)
		return
	}
	c.logTx("[MGMT TX list_subtree]", hdr, payload)
}

func (c *Controller) handleManagementFrame(payload []byte) {
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
		c.updateMgmtNodes(resp.Nodes)
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
		c.updateMgmtNodes(resp.Nodes)
	}
}

func (c *Controller) updateMgmtNodes(nodes []struct {
	NodeID      uint32 `json:"node_id"`
	HasChildren bool   `json:"has_children"`
}) {
	entries := make([]mgmtNodeEntry, 0, len(nodes))
	for _, n := range nodes {
		if n.NodeID == 0 {
			continue
		}
		entries = append(entries, mgmtNodeEntry{ID: n.NodeID, HasChildren: n.HasChildren})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
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
