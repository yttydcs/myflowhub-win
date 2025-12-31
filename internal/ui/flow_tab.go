package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	core "github.com/yttydcs/myflowhub-core"
	"github.com/yttydcs/myflowhub-core/header"
)

type flowUIState struct {
	targetEntry *widget.Entry

	flowList *widget.List
	flows    []flowSummary

	// editor
	flowID   *widget.Entry
	flowName *widget.Entry
	everyMs  *widget.Entry

	nodesList *widget.List
	edgesList *widget.List

	nodes []flowUINode
	edges []flowEdge

	selectedNode int
	selectedEdge int

	// node detail widgets
	nodeID        *widget.Entry
	nodeKind      *widget.Select
	nodeAllowFail *widget.Check
	nodeRetry     *widget.Entry
	nodeTimeout   *widget.Entry
	nodeMethod    *widget.Entry
	nodeTarget    *widget.Entry
	nodeArgs      *widget.Entry

	// run/status
	statusTitle *widget.Label
	statusList  *widget.List
	lastStatus  flowStatusResp
}

type flowUINode struct {
	ID        string
	Kind      string
	AllowFail bool
	Retry     int
	TimeoutMs int

	Method string
	Target uint32
	Args   string // JSON (object) or empty
}

func newFlowUIState() *flowUIState {
	return &flowUIState{
		selectedNode: -1,
		selectedEdge: -1,
	}
}

func (c *Controller) buildFlowTab(w fyne.Window) fyne.CanvasObject {
	if c.flow == nil {
		c.flow = newFlowUIState()
	}

	win := resolveWindow(c.app, c.mainWin, w)

	targetEntry := widget.NewEntry()
	targetEntry.SetPlaceHolder("目标 NodeID（flow.set 的接收者/执行者）")
	if c.storedHub != 0 {
		targetEntry.SetText(fmt.Sprintf("%d", c.storedHub))
	}

	refreshBtn := widget.NewButton("刷新列表", func() {
		if err := c.flowSendList(); err != nil && win != nil {
			dialog.ShowError(err, win)
		}
	})
	newBtn := widget.NewButton("新建", func() { c.flowNewDraft() })
	saveBtn := widget.NewButton("保存(set)", func() {
		if err := c.flowSendSet(); err != nil && win != nil {
			dialog.ShowError(err, win)
		}
	})
	runBtn := widget.NewButton("运行(run)", func() {
		if err := c.flowSendRun(); err != nil && win != nil {
			dialog.ShowError(err, win)
		}
	})
	statusBtn := widget.NewButton("状态(status)", func() {
		if err := c.flowSendStatus(""); err != nil && win != nil {
			dialog.ShowError(err, win)
		}
	})

	top := container.NewBorder(nil, nil,
		widget.NewLabel("目标:"),
		container.NewHBox(refreshBtn, newBtn, saveBtn, runBtn, statusBtn),
		targetEntry,
	)

	flowList := widget.NewList(
		func() int { return len(c.flow.flows) },
		func() fyne.CanvasObject {
			title := widget.NewLabel("")
			title.Wrapping = fyne.TextTruncate
			meta := widget.NewLabel("")
			meta.Wrapping = fyne.TextTruncate
			return container.NewBorder(nil, nil, nil, meta, title)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(c.flow.flows) {
				return
			}
			f := c.flow.flows[i]
			row := obj.(*fyne.Container)
			title := row.Objects[0].(*widget.Label)
			meta := row.Objects[1].(*widget.Label)
			name := strings.TrimSpace(f.Name)
			if name == "" {
				name = f.FlowID
			}
			title.SetText(name)
			meta.SetText(fmt.Sprintf("every_ms=%d  last=%s", f.EveryMs, strings.TrimSpace(f.LastStatus)))
		},
	)
	flowList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(c.flow.flows) {
			return
		}
		f := c.flow.flows[id]
		if strings.TrimSpace(f.FlowID) == "" {
			return
		}
		if err := c.flowSendGet(f.FlowID); err != nil && win != nil {
			dialog.ShowError(err, win)
		}
		_ = c.flowSendStatus("")
	}

	flowID := widget.NewEntry()
	flowID.SetPlaceHolder("flow_id（建议 uuid）")
	flowName := widget.NewEntry()
	flowName.SetPlaceHolder("name（可选）")
	everyMs := widget.NewEntry()
	everyMs.SetPlaceHolder("every_ms（触发间隔，毫秒）")

	nodesList := widget.NewList(
		func() int { return len(c.flow.nodes) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			l := obj.(*widget.Label)
			if i < 0 || i >= len(c.flow.nodes) {
				l.SetText("")
				return
			}
			n := c.flow.nodes[i]
			l.SetText(fmt.Sprintf("%s  [%s]", strings.TrimSpace(n.ID), strings.TrimSpace(n.Kind)))
		},
	)
	nodesList.OnSelected = func(id widget.ListItemID) { c.flowSelectNode(int(id)) }
	addNodeBtn := widget.NewButton("添加节点", func() { c.flowAddNodeDialog(win) })
	delNodeBtn := widget.NewButton("删除节点", func() { c.flowDeleteSelectedNode() })
	nodeBtns := container.NewHBox(addNodeBtn, delNodeBtn)

	edgesList := widget.NewList(
		func() int { return len(c.flow.edges) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			l := obj.(*widget.Label)
			if i < 0 || i >= len(c.flow.edges) {
				l.SetText("")
				return
			}
			e := c.flow.edges[i]
			l.SetText(fmt.Sprintf("%s -> %s", strings.TrimSpace(e.From), strings.TrimSpace(e.To)))
		},
	)
	edgesList.OnSelected = func(id widget.ListItemID) {
		if c.flow != nil {
			c.flow.selectedEdge = int(id)
		}
	}
	addEdgeBtn := widget.NewButton("添加边", func() { c.flowAddEdgeDialog(win) })
	delEdgeBtn := widget.NewButton("删除边", func() { c.flowDeleteSelectedEdge() })
	edgeBtns := container.NewHBox(addEdgeBtn, delEdgeBtn)

	left := container.NewBorder(
		widget.NewLabel("Flows"),
		nil,
		nil,
		nil,
		wrapScroll(flowList),
	)

	editorHeader := container.NewVBox(
		widget.NewLabel("编辑"),
		widget.NewForm(
			widget.NewFormItem("flow_id", flowID),
			widget.NewFormItem("name", flowName),
			widget.NewFormItem("every_ms", everyMs),
		),
		widget.NewSeparator(),
	)

	middle := container.NewVBox(
		editorHeader,
		widget.NewLabel("节点"),
		container.NewBorder(nil, nodeBtns, nil, nil, wrapScroll(nodesList)),
		widget.NewSeparator(),
		widget.NewLabel("边"),
		container.NewBorder(nil, edgeBtns, nil, nil, wrapScroll(edgesList)),
	)

	nodeID := widget.NewEntry()
	nodeKind := widget.NewSelect([]string{"local", "exec"}, func(_ string) { c.flowUpdateNodeDetailFromUI() })
	nodeAllowFail := widget.NewCheck("allow_fail", func(bool) { c.flowUpdateNodeDetailFromUI() })
	nodeRetry := widget.NewEntry()
	nodeTimeout := widget.NewEntry()
	nodeMethod := widget.NewEntry()
	nodeTarget := widget.NewEntry()
	nodeArgs := widget.NewMultiLineEntry()
	nodeArgs.SetPlaceHolder("args JSON（可为空，默认 {}）")

	nodeRetry.OnChanged = func(string) { c.flowUpdateNodeDetailFromUI() }
	nodeTimeout.OnChanged = func(string) { c.flowUpdateNodeDetailFromUI() }
	nodeID.OnChanged = func(string) { c.flowUpdateNodeDetailFromUI() }
	nodeMethod.OnChanged = func(string) { c.flowUpdateNodeDetailFromUI() }
	nodeTarget.OnChanged = func(string) { c.flowUpdateNodeDetailFromUI() }
	nodeArgs.OnChanged = func(string) { c.flowUpdateNodeDetailFromUI() }

	detailForm := widget.NewForm(
		widget.NewFormItem("id", nodeID),
		widget.NewFormItem("kind", nodeKind),
		widget.NewFormItem("", nodeAllowFail),
		widget.NewFormItem("retry", nodeRetry),
		widget.NewFormItem("timeout_ms", nodeTimeout),
		widget.NewFormItem("method", nodeMethod),
		widget.NewFormItem("target_node(exec)", nodeTarget),
		widget.NewFormItem("args", nodeArgs),
	)
	detailCard := widget.NewCard("节点详情", "选中节点后编辑", detailForm)

	statusTitle := widget.NewLabel("最近运行：未加载")
	statusList := widget.NewList(
		func() int { return len(c.flow.lastStatus.Nodes) },
		func() fyne.CanvasObject {
			title := widget.NewLabel("")
			title.Wrapping = fyne.TextTruncate
			meta := widget.NewLabel("")
			meta.Wrapping = fyne.TextTruncate
			return container.NewBorder(nil, nil, nil, meta, title)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(c.flow.lastStatus.Nodes) {
				return
			}
			n := c.flow.lastStatus.Nodes[i]
			row := obj.(*fyne.Container)
			title := row.Objects[0].(*widget.Label)
			meta := row.Objects[1].(*widget.Label)
			title.SetText(fmt.Sprintf("%s  %s", strings.TrimSpace(n.ID), strings.TrimSpace(n.Status)))
			msg := strings.TrimSpace(n.Msg)
			if msg == "" {
				msg = fmt.Sprintf("code=%d", n.Code)
			} else {
				msg = fmt.Sprintf("code=%d  %s", n.Code, msg)
			}
			meta.SetText(msg)
		},
	)
	right := container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(), statusTitle, wrapScroll(statusList)),
		nil,
		nil,
		container.NewVBox(detailCard),
	)

	mainSplit := container.NewHSplit(
		container.NewBorder(top, nil, nil, nil, left),
		container.NewHSplit(middle, right),
	)
	mainSplit.Offset = 0.25

	c.flow.targetEntry = targetEntry
	c.flow.flowList = flowList
	c.flow.flowID = flowID
	c.flow.flowName = flowName
	c.flow.everyMs = everyMs
	c.flow.nodesList = nodesList
	c.flow.edgesList = edgesList
	c.flow.nodeID = nodeID
	c.flow.nodeKind = nodeKind
	c.flow.nodeAllowFail = nodeAllowFail
	c.flow.nodeRetry = nodeRetry
	c.flow.nodeTimeout = nodeTimeout
	c.flow.nodeMethod = nodeMethod
	c.flow.nodeTarget = nodeTarget
	c.flow.nodeArgs = nodeArgs
	c.flow.statusTitle = statusTitle
	c.flow.statusList = statusList

	c.flowNewDraft()
	_ = c.flowSendList()

	return mainSplit
}

func (c *Controller) parseFlowTarget() (uint32, error) {
	if c == nil || c.flow == nil || c.flow.targetEntry == nil {
		return 0, nil
	}
	raw := strings.TrimSpace(c.flow.targetEntry.Text)
	if raw == "" {
		if c.storedHub != 0 {
			return c.storedHub, nil
		}
		return 0, fmt.Errorf("目标 NodeID 不能为空")
	}
	n, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || n == 0 {
		return 0, fmt.Errorf("目标 NodeID 非法")
	}
	return uint32(n), nil
}

func (c *Controller) flowNewDraft() {
	if c.flow == nil {
		return
	}
	if c.flow.flowID != nil {
		c.flow.flowID.SetText("")
	}
	if c.flow.flowName != nil {
		c.flow.flowName.SetText("")
	}
	if c.flow.everyMs != nil {
		c.flow.everyMs.SetText("60000")
	}
	c.flow.nodes = nil
	c.flow.edges = nil
	c.flow.selectedNode = -1
	c.flow.selectedEdge = -1
	if c.flow.nodesList != nil {
		c.flow.nodesList.Refresh()
	}
	if c.flow.edgesList != nil {
		c.flow.edgesList.Refresh()
	}
	c.flowSelectNode(-1)
}

func (c *Controller) flowSelectNode(idx int) {
	if c.flow == nil {
		return
	}
	c.flow.selectedNode = idx
	if idx < 0 || idx >= len(c.flow.nodes) {
		if c.flow.nodeID != nil {
			c.flow.nodeID.SetText("")
		}
		if c.flow.nodeKind != nil {
			c.flow.nodeKind.SetSelected("")
		}
		if c.flow.nodeAllowFail != nil {
			c.flow.nodeAllowFail.SetChecked(false)
		}
		if c.flow.nodeRetry != nil {
			c.flow.nodeRetry.SetText("")
		}
		if c.flow.nodeTimeout != nil {
			c.flow.nodeTimeout.SetText("")
		}
		if c.flow.nodeMethod != nil {
			c.flow.nodeMethod.SetText("")
		}
		if c.flow.nodeTarget != nil {
			c.flow.nodeTarget.SetText("")
		}
		if c.flow.nodeArgs != nil {
			c.flow.nodeArgs.SetText("")
		}
		return
	}
	n := c.flow.nodes[idx]
	c.flow.nodeID.SetText(n.ID)
	c.flow.nodeKind.SetSelected(n.Kind)
	c.flow.nodeAllowFail.SetChecked(n.AllowFail)
	if n.Retry >= 0 {
		c.flow.nodeRetry.SetText(fmt.Sprintf("%d", n.Retry))
	} else {
		c.flow.nodeRetry.SetText("")
	}
	if n.TimeoutMs > 0 {
		c.flow.nodeTimeout.SetText(fmt.Sprintf("%d", n.TimeoutMs))
	} else {
		c.flow.nodeTimeout.SetText("")
	}
	c.flow.nodeMethod.SetText(n.Method)
	if n.Kind == "exec" {
		c.flow.nodeTarget.SetText(fmt.Sprintf("%d", n.Target))
	} else {
		c.flow.nodeTarget.SetText("")
	}
	c.flow.nodeArgs.SetText(n.Args)
}

func (c *Controller) flowUpdateNodeDetailFromUI() {
	if c.flow == nil {
		return
	}
	idx := c.flow.selectedNode
	if idx < 0 || idx >= len(c.flow.nodes) {
		return
	}
	n := c.flow.nodes[idx]
	n.ID = strings.TrimSpace(c.flow.nodeID.Text)
	n.Kind = strings.TrimSpace(c.flow.nodeKind.Selected)
	n.AllowFail = c.flow.nodeAllowFail.Checked
	n.Method = strings.TrimSpace(c.flow.nodeMethod.Text)
	n.Args = strings.TrimSpace(c.flow.nodeArgs.Text)
	n.Retry = parseIntOr(n.Retry, strings.TrimSpace(c.flow.nodeRetry.Text))
	n.TimeoutMs = parseIntOr(n.TimeoutMs, strings.TrimSpace(c.flow.nodeTimeout.Text))
	if n.Kind == "exec" {
		if v, err := strconv.ParseUint(strings.TrimSpace(c.flow.nodeTarget.Text), 10, 32); err == nil {
			n.Target = uint32(v)
		}
	}
	c.flow.nodes[idx] = n
	if c.flow.nodesList != nil {
		c.flow.nodesList.Refresh()
	}
}

func parseIntOr(def int, raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return def
	}
	return int(n)
}

func (c *Controller) flowAddNodeDialog(win fyne.Window) {
	if win == nil || c.flow == nil {
		return
	}
	id := widget.NewEntry()
	id.SetPlaceHolder("节点ID（唯一）")
	kind := widget.NewSelect([]string{"local", "exec"}, nil)
	kind.SetSelected("local")
	dialog.ShowForm("添加节点", "添加", "取消",
		[]*widget.FormItem{
			widget.NewFormItem("id", id),
			widget.NewFormItem("kind", kind),
		},
		func(ok bool) {
			if !ok {
				return
			}
			nid := strings.TrimSpace(id.Text)
			if nid == "" {
				dialog.ShowError(fmt.Errorf("节点ID不能为空"), win)
				return
			}
			for _, n := range c.flow.nodes {
				if strings.TrimSpace(n.ID) == nid {
					dialog.ShowError(fmt.Errorf("节点ID重复"), win)
					return
				}
			}
			node := flowUINode{ID: nid, Kind: kind.Selected, Retry: 1, TimeoutMs: 3000, Args: "{}"}
			c.flow.nodes = append(c.flow.nodes, node)
			if c.flow.nodesList != nil {
				c.flow.nodesList.Refresh()
				c.flow.nodesList.Select(len(c.flow.nodes) - 1)
			}
		}, win)
}

func (c *Controller) flowDeleteSelectedNode() {
	if c.flow == nil {
		return
	}
	idx := c.flow.selectedNode
	if idx < 0 || idx >= len(c.flow.nodes) {
		return
	}
	id := strings.TrimSpace(c.flow.nodes[idx].ID)
	out := make([]flowUINode, 0, len(c.flow.nodes)-1)
	for i, n := range c.flow.nodes {
		if i == idx {
			continue
		}
		out = append(out, n)
	}
	c.flow.nodes = out
	// remove related edges
	edges := make([]flowEdge, 0, len(c.flow.edges))
	for _, e := range c.flow.edges {
		if strings.TrimSpace(e.From) == id || strings.TrimSpace(e.To) == id {
			continue
		}
		edges = append(edges, e)
	}
	c.flow.edges = edges
	c.flow.selectedNode = -1
	c.flow.selectedEdge = -1
	if c.flow.nodesList != nil {
		c.flow.nodesList.Refresh()
	}
	if c.flow.edgesList != nil {
		c.flow.edgesList.Refresh()
	}
	c.flowSelectNode(-1)
}

func (c *Controller) flowAddEdgeDialog(win fyne.Window) {
	if win == nil || c.flow == nil {
		return
	}
	if len(c.flow.nodes) < 2 {
		dialog.ShowError(fmt.Errorf("至少需要两个节点才能添加边"), win)
		return
	}
	ids := make([]string, 0, len(c.flow.nodes))
	for _, n := range c.flow.nodes {
		ids = append(ids, strings.TrimSpace(n.ID))
	}
	sort.Strings(ids)
	from := widget.NewSelect(ids, nil)
	to := widget.NewSelect(ids, nil)
	from.SetSelected(ids[0])
	if len(ids) > 1 {
		to.SetSelected(ids[1])
	}
	dialog.ShowForm("添加边", "添加", "取消",
		[]*widget.FormItem{
			widget.NewFormItem("from", from),
			widget.NewFormItem("to", to),
		},
		func(ok bool) {
			if !ok {
				return
			}
			f := strings.TrimSpace(from.Selected)
			t := strings.TrimSpace(to.Selected)
			if f == "" || t == "" || f == t {
				dialog.ShowError(fmt.Errorf("边非法"), win)
				return
			}
			c.flow.edges = append(c.flow.edges, flowEdge{From: f, To: t})
			if c.flow.edgesList != nil {
				c.flow.edgesList.Refresh()
			}
		}, win)
}

func (c *Controller) flowDeleteSelectedEdge() {
	if c.flow == nil {
		return
	}
	idx := c.flow.selectedEdge
	if idx < 0 || idx >= len(c.flow.edges) {
		return
	}
	out := make([]flowEdge, 0, len(c.flow.edges)-1)
	for i, e := range c.flow.edges {
		if i == idx {
			continue
		}
		out = append(out, e)
	}
	c.flow.edges = out
	c.flow.selectedEdge = -1
	if c.flow.edgesList != nil {
		c.flow.edgesList.Refresh()
	}
}

func (c *Controller) flowSendList() error {
	if c == nil || c.session == nil || c.flow == nil {
		return nil
	}
	if c.storedNode == 0 || c.storedHub == 0 {
		return fmt.Errorf("尚未登录")
	}
	target, err := c.parseFlowTarget()
	if err != nil {
		return err
	}
	req := flowListReq{ReqID: newReqID(), OriginNode: c.storedNode, ExecutorNode: target}
	return c.flowSendCtrl(flowMessage{Action: flowActionList, Data: mustJSONRaw(req)})
}

func (c *Controller) flowSendGet(flowID string) error {
	if c == nil || c.session == nil || c.flow == nil {
		return nil
	}
	if c.storedNode == 0 || c.storedHub == 0 {
		return fmt.Errorf("尚未登录")
	}
	target, err := c.parseFlowTarget()
	if err != nil {
		return err
	}
	req := flowGetReq{ReqID: newReqID(), OriginNode: c.storedNode, ExecutorNode: target, FlowID: strings.TrimSpace(flowID)}
	return c.flowSendCtrl(flowMessage{Action: flowActionGet, Data: mustJSONRaw(req)})
}

func (c *Controller) flowSendSet() error {
	if c == nil || c.session == nil || c.flow == nil {
		return nil
	}
	if c.storedNode == 0 || c.storedHub == 0 {
		return fmt.Errorf("尚未登录")
	}
	target, err := c.parseFlowTarget()
	if err != nil {
		return err
	}
	flowID := strings.TrimSpace(c.flow.flowID.Text)
	if flowID == "" {
		return fmt.Errorf("flow_id 不能为空")
	}
	every, err := strconv.ParseUint(strings.TrimSpace(c.flow.everyMs.Text), 10, 64)
	if err != nil || every == 0 {
		return fmt.Errorf("every_ms 非法")
	}
	nodes, edges, err := c.flowBuildGraph()
	if err != nil {
		return err
	}
	req := flowSetReq{
		ReqID:        newReqID(),
		OriginNode:   c.storedNode,
		ExecutorNode: target,
		FlowID:       flowID,
		Name:         strings.TrimSpace(c.flow.flowName.Text),
		Trigger:      flowTrigger{Type: "interval", EveryMs: uint64(every)},
		Graph:        flowGraph{Nodes: nodes, Edges: edges},
	}
	return c.flowSendCtrl(flowMessage{Action: flowActionSet, Data: mustJSONRaw(req)})
}

func (c *Controller) flowSendRun() error {
	if c == nil || c.session == nil || c.flow == nil {
		return nil
	}
	if c.storedNode == 0 || c.storedHub == 0 {
		return fmt.Errorf("尚未登录")
	}
	target, err := c.parseFlowTarget()
	if err != nil {
		return err
	}
	flowID := strings.TrimSpace(c.flow.flowID.Text)
	if flowID == "" {
		return fmt.Errorf("flow_id 不能为空")
	}
	req := flowRunReq{ReqID: newReqID(), OriginNode: c.storedNode, ExecutorNode: target, FlowID: flowID}
	return c.flowSendCtrl(flowMessage{Action: flowActionRun, Data: mustJSONRaw(req)})
}

func (c *Controller) flowSendStatus(runID string) error {
	if c == nil || c.session == nil || c.flow == nil {
		return nil
	}
	if c.storedNode == 0 || c.storedHub == 0 {
		return fmt.Errorf("尚未登录")
	}
	target, err := c.parseFlowTarget()
	if err != nil {
		return err
	}
	flowID := strings.TrimSpace(c.flow.flowID.Text)
	if flowID == "" {
		return fmt.Errorf("flow_id 不能为空")
	}
	req := flowStatusReq{ReqID: newReqID(), OriginNode: c.storedNode, ExecutorNode: target, FlowID: flowID, RunID: strings.TrimSpace(runID)}
	return c.flowSendCtrl(flowMessage{Action: flowActionStatus, Data: mustJSONRaw(req)})
}

func (c *Controller) flowBuildGraph() ([]flowNode, []flowEdge, error) {
	if c.flow == nil {
		return nil, nil, fmt.Errorf("no state")
	}
	seen := make(map[string]bool)
	outNodes := make([]flowNode, 0, len(c.flow.nodes))
	for _, n := range c.flow.nodes {
		id := strings.TrimSpace(n.ID)
		if id == "" {
			return nil, nil, fmt.Errorf("节点ID不能为空")
		}
		if seen[id] {
			return nil, nil, fmt.Errorf("节点ID重复: %s", id)
		}
		seen[id] = true
		kind := strings.ToLower(strings.TrimSpace(n.Kind))
		if kind != "local" && kind != "exec" {
			return nil, nil, fmt.Errorf("节点kind非法: %s", kind)
		}
		retry := n.Retry
		timeout := n.TimeoutMs
		spec, err := buildNodeSpec(n)
		if err != nil {
			return nil, nil, err
		}
		rp := retry
		tp := timeout
		outNodes = append(outNodes, flowNode{
			ID:        id,
			Kind:      kind,
			AllowFail: n.AllowFail,
			Retry:     &rp,
			TimeoutMs: &tp,
			Spec:      spec,
		})
	}
	outEdges := make([]flowEdge, 0, len(c.flow.edges))
	for _, e := range c.flow.edges {
		f := strings.TrimSpace(e.From)
		t := strings.TrimSpace(e.To)
		if f == "" || t == "" || f == t {
			return nil, nil, fmt.Errorf("边非法")
		}
		if !seen[f] || !seen[t] {
			return nil, nil, fmt.Errorf("边引用未知节点")
		}
		outEdges = append(outEdges, flowEdge{From: f, To: t})
	}
	return outNodes, outEdges, nil
}

func buildNodeSpec(n flowUINode) (json.RawMessage, error) {
	kind := strings.ToLower(strings.TrimSpace(n.Kind))
	method := strings.TrimSpace(n.Method)
	if method == "" {
		return nil, fmt.Errorf("method 不能为空")
	}
	argsRaw := strings.TrimSpace(n.Args)
	if argsRaw == "" {
		argsRaw = "{}"
	}
	var args json.RawMessage
	if err := json.Unmarshal([]byte(argsRaw), &args); err != nil {
		return nil, fmt.Errorf("args 不是合法 JSON: %v", err)
	}
	if kind == "local" {
		spec := map[string]any{"method": method, "args": json.RawMessage(args)}
		raw, _ := json.Marshal(spec)
		return raw, nil
	}
	if kind == "exec" {
		if n.Target == 0 {
			return nil, fmt.Errorf("exec 节点 target_node 不能为空")
		}
		spec := map[string]any{"target": n.Target, "method": method, "args": json.RawMessage(args)}
		raw, _ := json.Marshal(spec)
		return raw, nil
	}
	return nil, fmt.Errorf("unknown kind")
}

func (c *Controller) flowSendCtrl(msg flowMessage) error {
	if c == nil || c.session == nil {
		return nil
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(subProtoFlow).
		WithSourceID(c.storedNode).
		WithTargetID(c.storedHub).
		WithMsgID(uint32(time.Now().UnixNano())).
		WithTimestamp(uint32(time.Now().Unix()))
	if err := c.session.Send(hdr, payload); err != nil {
		return err
	}
	c.logTx("[FLOW TX "+msg.Action+"]", hdr, payload)
	return nil
}

func newReqID() string {
	// reuse existing UUID generator in file protocol
	id, _ := fileNewUUID()
	return fileUUIDToString(id)
}

func (c *Controller) handleFlowFrame(h core.IHeader, payload []byte) {
	if c == nil || c.flow == nil {
		return
	}
	var msg flowMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	switch msg.Action {
	case flowActionListResp:
		var resp flowListResp
		_ = json.Unmarshal(msg.Data, &resp)
		if resp.Code == 1 {
			c.flow.flows = resp.Flows
		}
		if c.flow.flowList != nil {
			runOnMain(c, c.flow.flowList.Refresh)
		}
	case flowActionGetResp:
		var resp flowGetResp
		_ = json.Unmarshal(msg.Data, &resp)
		if resp.Code != 1 {
			return
		}
		runOnMain(c, func() {
			if c.flow.flowID != nil {
				c.flow.flowID.SetText(strings.TrimSpace(resp.FlowID))
			}
			if c.flow.flowName != nil {
				c.flow.flowName.SetText(strings.TrimSpace(resp.Name))
			}
			if c.flow.everyMs != nil {
				c.flow.everyMs.SetText(fmt.Sprintf("%d", resp.Trigger.EveryMs))
			}
		})
		c.flow.nodes = nil
		for _, n := range resp.Graph.Nodes {
			ui := flowUINode{
				ID:        strings.TrimSpace(n.ID),
				Kind:      strings.TrimSpace(n.Kind),
				AllowFail: n.AllowFail,
				Retry:     1,
				TimeoutMs: 3000,
				Args:      "{}",
			}
			if n.Retry != nil {
				ui.Retry = *n.Retry
			}
			if n.TimeoutMs != nil {
				ui.TimeoutMs = *n.TimeoutMs
			}
			// parse spec for local/exec
			var m map[string]any
			_ = json.Unmarshal(n.Spec, &m)
			if v, ok := m["method"].(string); ok {
				ui.Method = v
			}
			if ui.Kind == "exec" {
				switch tv := m["target"].(type) {
				case float64:
					ui.Target = uint32(tv)
				}
			}
			if av, ok := m["args"]; ok {
				if raw, err := json.Marshal(av); err == nil {
					ui.Args = string(raw)
				}
			}
			c.flow.nodes = append(c.flow.nodes, ui)
		}
		c.flow.edges = resp.Graph.Edges
		runOnMain(c, func() {
			if c.flow.nodesList != nil {
				c.flow.nodesList.Refresh()
			}
			if c.flow.edgesList != nil {
				c.flow.edgesList.Refresh()
			}
			c.flowSelectNode(-1)
		})
		go func() {
			time.Sleep(80 * time.Millisecond)
			_ = c.flowSendStatus("")
		}()
	case flowActionSetResp:
		var resp flowSetResp
		_ = json.Unmarshal(msg.Data, &resp)
		c.appendLog("[FLOW] set_resp code=%d msg=%s flow=%s", resp.Code, strings.TrimSpace(resp.Msg), strings.TrimSpace(resp.FlowID))
		_ = c.flowSendList()
	case flowActionRunResp:
		var resp flowRunResp
		_ = json.Unmarshal(msg.Data, &resp)
		c.appendLog("[FLOW] run_resp code=%d msg=%s flow=%s run=%s", resp.Code, strings.TrimSpace(resp.Msg), strings.TrimSpace(resp.FlowID), strings.TrimSpace(resp.RunID))
		_ = c.flowSendStatus(strings.TrimSpace(resp.RunID))
	case flowActionStatusResp:
		var resp flowStatusResp
		_ = json.Unmarshal(msg.Data, &resp)
		c.flow.lastStatus = resp
		runOnMain(c, func() {
			if c.flow.statusTitle != nil {
				c.flow.statusTitle.SetText(fmt.Sprintf("最近运行：%s  run=%s", strings.TrimSpace(resp.Status), strings.TrimSpace(resp.RunID)))
			}
			if c.flow.statusList != nil {
				c.flow.statusList.Refresh()
			}
		})
	}
}
