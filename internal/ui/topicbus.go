package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	core "github.com/yttydcs/myflowhub-core"
	"github.com/yttydcs/myflowhub-core/eventbus"
	"github.com/yttydcs/myflowhub-core/header"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const (
	subProtoTopicBus uint8 = 4

	topicBusActionSubscribe          = "subscribe"
	topicBusActionSubscribeResp      = "subscribe_resp"
	topicBusActionSubscribeBatch     = "subscribe_batch"
	topicBusActionSubscribeBatchResp = "subscribe_batch_resp"

	topicBusActionUnsubscribe          = "unsubscribe"
	topicBusActionUnsubscribeResp      = "unsubscribe_resp"
	topicBusActionUnsubscribeBatch     = "unsubscribe_batch"
	topicBusActionUnsubscribeBatchResp = "unsubscribe_batch_resp"

	topicBusActionListSubs     = "list_subs"
	topicBusActionListSubsResp = "list_subs_resp"

	topicBusActionPublish = "publish"

	uiEventLoggedIn = "ui.auth.logged_in"

	defaultTopicBusMaxEvents = 500
)

type topicBusMessage struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type topicBusSubscribeReq struct {
	Topic string `json:"topic"`
}

type topicBusSubscribeBatchReq struct {
	Topics []string `json:"topics"`
}

type topicBusResp struct {
	Code   int      `json:"code"`
	Msg    string   `json:"msg"`
	Topic  string   `json:"topic"`
	Topics []string `json:"topics"`
}

type topicBusListResp struct {
	Code   int      `json:"code"`
	Msg    string   `json:"msg"`
	Topics []string `json:"topics"`
}

type topicBusPublish struct {
	Topic   string          `json:"topic"`
	Name    string          `json:"name"`
	TS      int64           `json:"ts"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type topicBusEvent struct {
	Topic string
	Name  string
	TS    int64
	Data  json.RawMessage // publish.data 原始 JSON
}

func (c *Controller) emitLoggedIn() {
	if c == nil || c.bus == nil {
		return
	}
	meta := map[string]any{"node_id": c.storedNode, "hub_id": c.storedHub}
	ctx := c.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	c.bus.PublishSync(ctx, uiEventLoggedIn, nil, meta)
}

func (c *Controller) ensureTopicBusLoginSubscription() {
	if c == nil || c.bus == nil || c.topicBusLoginToken != "" {
		return
	}
	c.topicBusLoginToken = c.bus.Subscribe(uiEventLoggedIn, func(_ context.Context, _ eventbus.Event) {
		go c.resubscribeTopicBus()
	})
}

func (c *Controller) buildTopicBusTab(w fyne.Window) fyne.CanvasObject {
	c.ensureTopicBusLoginSubscription()

	if c.topicBusMaxEvents <= 0 {
		c.topicBusMaxEvents = defaultTopicBusMaxEvents
	}
	if c.topicBusTarget == nil {
		c.topicBusTarget = widget.NewEntry()
		c.topicBusTarget.SetPlaceHolder("TargetID")
		if c.storedHub != 0 {
			c.topicBusTarget.SetText(fmt.Sprintf("%d", c.storedHub))
		}
	}
	if c.topicBusInput == nil {
		c.topicBusInput = widget.NewMultiLineEntry()
		c.topicBusInput.SetPlaceHolder("输入 topic（支持多行/逗号分隔），不能为空")
	}
	if c.topicBusMaxEntry == nil {
		c.topicBusMaxEntry = widget.NewEntry()
		c.topicBusMaxEntry.SetPlaceHolder(strconv.Itoa(defaultTopicBusMaxEvents))
	}
	c.topicBusMaxEntry.SetText(strconv.Itoa(c.topicBusMaxEvents))

	if c.topicBusPubTopic == nil {
		c.topicBusPubTopic = widget.NewEntry()
		c.topicBusPubTopic.SetPlaceHolder("topic（留空可用“选中 topic”自动填充）")
	}
	if c.topicBusPubName == nil {
		c.topicBusPubName = widget.NewEntry()
		c.topicBusPubName.SetPlaceHolder("事件名 name（不能为空）")
	}
	if c.topicBusPubPayload == nil {
		c.topicBusPubPayload = widget.NewMultiLineEntry()
		c.topicBusPubPayload.SetPlaceHolder("payload（可选）：JSON 或普通文本（非 JSON 将按字符串发送）")
		c.topicBusPubPayload.Wrapping = fyne.TextWrapWord
	}

	if c.topicBusDetail == nil {
		c.topicBusDetail = newLogEntry()
		c.topicBusDetail.SetPlaceHolder("选择右侧事件后显示详情（publish.data）")
	}

	if c.topicBusSubsList == nil {
		c.topicBusSubsList = widget.NewList(
			func() int {
				c.topicBusMu.RLock()
				n := len(c.topicBusSubs)
				c.topicBusMu.RUnlock()
				return n + 1 // + 全部
			},
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(id widget.ListItemID, obj fyne.CanvasObject) {
				label, ok := obj.(*widget.Label)
				if !ok {
					return
				}
				if id == 0 {
					label.SetText("全部")
					return
				}
				c.topicBusMu.RLock()
				defer c.topicBusMu.RUnlock()
				idx := int(id - 1)
				if idx < 0 || idx >= len(c.topicBusSubs) {
					label.SetText("-")
					return
				}
				label.SetText(c.topicBusSubs[idx])
			},
		)
		c.topicBusSubsList.OnSelected = func(id widget.ListItemID) {
			c.topicBusMu.Lock()
			c.topicBusSelectedSub = int(id)
			c.rebuildTopicBusFilterLocked()
			c.topicBusMu.Unlock()
			runOnMain(c, func() {
				if c.topicBusEventList != nil {
					c.topicBusEventList.UnselectAll()
					c.topicBusEventList.Refresh()
				}
				if c.topicBusDetail != nil {
					c.topicBusDetail.SetText("")
				}
			})
		}
	}

	if c.topicBusEventList == nil {
		c.topicBusEventList = widget.NewList(
			func() int {
				c.topicBusMu.RLock()
				n := len(c.topicBusFilteredIdx)
				c.topicBusMu.RUnlock()
				return n
			},
			func() fyne.CanvasObject {
				l := widget.NewLabel("")
				l.Wrapping = fyne.TextWrapOff
				return l
			},
			func(id widget.ListItemID, obj fyne.CanvasObject) {
				label, ok := obj.(*widget.Label)
				if !ok {
					return
				}
				ev, ok := c.getTopicBusFilteredEvent(int(id))
				if !ok {
					label.SetText("")
					return
				}
				label.SetText(formatTopicBusEventLine(ev))
			},
		)
		c.topicBusEventList.OnSelected = func(id widget.ListItemID) {
			ev, ok := c.getTopicBusFilteredEvent(int(id))
			if !ok || c.topicBusDetail == nil {
				return
			}
			c.topicBusDetail.SetText(prettyJSON(ev.Data))
		}
	}

	subscribeBtn := widget.NewButton("订阅", func() { c.subscribeTopicBusFromInput(w) })
	unsubscribeBtn := widget.NewButton("退订", func() { c.unsubscribeTopicBusFromInput(w) })
	unsubscribeSelectedBtn := widget.NewButton("退订选中", func() { c.unsubscribeTopicBusSelected(w) })
	clearBtn := widget.NewButton("清空事件", func() { c.clearTopicBusEvents() })
	applyMaxBtn := widget.NewButton("应用上限", func() { c.applyTopicBusMaxEvents(w) })
	resubBtn := widget.NewButton("重订阅", func() { go c.resubscribeTopicBus() })
	fillSelectedTopicBtn := widget.NewButton("选中 topic", func() { c.fillPublishTopicFromSelection(w) })
	publishBtn := widget.NewButton("发布", func() { c.publishTopicBus(w) })
	clearPublishBtn := widget.NewButton("清空发布输入", func() { c.clearTopicBusPublishInputs() })

	targetCard := widget.NewCard("目标", "留空使用当前登录 HubID", container.NewHBox(widget.NewLabel("TargetID"), c.topicBusTarget))
	subCard := widget.NewCard("订阅/退订", "支持批量：多行/逗号分隔；topic 不允许为空", container.NewVBox(
		c.topicBusInput,
		container.NewHBox(subscribeBtn, unsubscribeBtn, unsubscribeSelectedBtn),
		container.NewHBox(resubBtn, layout.NewSpacer(), widget.NewLabel("事件上限"), c.topicBusMaxEntry, applyMaxBtn, clearBtn),
	))
	pubCard := widget.NewCard("发布", "向任意 topic 发布事件（publish 不回显）", container.NewVBox(
		container.NewBorder(nil, nil, nil, fillSelectedTopicBtn, labeledEntry("Topic", c.topicBusPubTopic)),
		labeledEntry("Name", c.topicBusPubName),
		labeledEntry("Payload", c.topicBusPubPayload),
		container.NewHBox(publishBtn, clearPublishBtn),
	))
	subsCard := widget.NewCard("订阅列表", "点击左侧项用于筛选右侧事件；包含“全部”", c.topicBusSubsList)

	controls := container.NewVBox(targetCard, subCard, pubCard)
	controlsScroll := wrapScroll(controls)
	leftSplit := container.NewVSplit(controlsScroll, subsCard)
	leftSplit.Offset = 0.35
	left := leftSplit

	rightSplit := container.NewVSplit(c.topicBusEventList, c.topicBusDetail)
	rightSplit.Offset = 0.70
	right := container.NewBorder(widget.NewLabel("事件（publish）"), nil, nil, nil, rightSplit)

	main := container.NewHSplit(left, right)
	main.Offset = 0.30

	runOnMain(c, func() {
		if c.topicBusSubsList != nil && c.topicBusSelectedSub == 0 {
			c.topicBusSubsList.Select(0)
		}
		c.refreshTopicBusUI()
	})

	return main
}

func (c *Controller) loadTopicBusPrefs() {
	if c == nil {
		return
	}
	if c.topicBusMaxEvents <= 0 {
		c.topicBusMaxEvents = defaultTopicBusMaxEvents
	}
	if c.app == nil || c.app.Preferences() == nil {
		return
	}
	p := c.app.Preferences()
	raw := p.StringWithFallback(c.prefKey(prefTopicBusSubs), "")
	if strings.TrimSpace(raw) != "" {
		var topics []string
		if err := json.Unmarshal([]byte(raw), &topics); err == nil {
			c.topicBusSubs = uniqueStableStrings(filterNonEmpty(topics))
		}
	}
	max := p.IntWithFallback(c.prefKey(prefTopicBusMaxEvt), defaultTopicBusMaxEvents)
	if max > 0 {
		c.topicBusMaxEvents = max
	}
}

func (c *Controller) saveTopicBusPrefs() {
	if c == nil || c.app == nil || c.app.Preferences() == nil {
		return
	}
	p := c.app.Preferences()
	c.topicBusMu.RLock()
	topics := append([]string(nil), c.topicBusSubs...)
	max := c.topicBusMaxEvents
	c.topicBusMu.RUnlock()

	data, _ := json.Marshal(topics)
	p.SetString(c.prefKey(prefTopicBusSubs), string(data))
	if max > 0 {
		p.SetInt(c.prefKey(prefTopicBusMaxEvt), max)
	}
}

func (c *Controller) refreshTopicBusUI() {
	if c == nil {
		return
	}
	max := 0
	c.topicBusMu.Lock()
	if c.topicBusMaxEvents <= 0 {
		c.topicBusMaxEvents = defaultTopicBusMaxEvents
	}
	max = c.topicBusMaxEvents
	c.trimTopicBusEventsLocked()
	c.rebuildTopicBusFilterLocked()
	c.topicBusMu.Unlock()

	runOnMain(c, func() {
		if c.topicBusMaxEntry != nil {
			c.topicBusMaxEntry.SetText(strconv.Itoa(max))
		}
		if c.topicBusSubsList != nil {
			c.topicBusSubsList.Refresh()
		}
		if c.topicBusEventList != nil {
			c.topicBusEventList.Refresh()
		}
	})
}

func (c *Controller) handleTopicBusFrame(payload []byte) {
	var msg topicBusMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	act := strings.ToLower(strings.TrimSpace(msg.Action))
	if act != topicBusActionPublish {
		return
	}
	var pub topicBusPublish
	if err := json.Unmarshal(msg.Data, &pub); err != nil {
		return
	}
	if strings.TrimSpace(pub.Topic) == "" || strings.TrimSpace(pub.Name) == "" {
		return
	}
	c.onTopicBusStressPublish(pub)
	c.addTopicBusEvent(topicBusEvent{Topic: pub.Topic, Name: pub.Name, TS: pub.TS, Data: msg.Data})
}

func (c *Controller) addTopicBusEvent(ev topicBusEvent) {
	if c == nil {
		return
	}
	now := time.Now()
	shouldUpdate := false

	c.topicBusMu.Lock()
	c.topicBusEvents = append(c.topicBusEvents, ev)
	c.trimTopicBusEventsLocked()
	c.rebuildTopicBusFilterLocked()
	if now.Sub(c.topicBusEventsLastUI) >= 200*time.Millisecond {
		c.topicBusEventsLastUI = now
		shouldUpdate = true
	}
	c.topicBusMu.Unlock()

	if shouldUpdate {
		runOnMain(c, func() {
			if c.topicBusEventList != nil {
				c.topicBusEventList.Refresh()
				c.topicBusEventList.ScrollToBottom()
			}
		})
	}
}

func (c *Controller) trimTopicBusEventsLocked() {
	if c.topicBusMaxEvents <= 0 {
		c.topicBusMaxEvents = defaultTopicBusMaxEvents
	}
	if len(c.topicBusEvents) <= c.topicBusMaxEvents {
		return
	}
	c.topicBusEvents = append([]topicBusEvent(nil), c.topicBusEvents[len(c.topicBusEvents)-c.topicBusMaxEvents:]...)
}

func (c *Controller) rebuildTopicBusFilterLocked() {
	selectedAll := c.topicBusSelectedSub == 0
	selectedTopic := ""
	if !selectedAll {
		idx := c.topicBusSelectedSub - 1
		if idx >= 0 && idx < len(c.topicBusSubs) {
			selectedTopic = c.topicBusSubs[idx]
		} else {
			c.topicBusSelectedSub = 0
			selectedAll = true
		}
	}
	out := make([]int, 0, len(c.topicBusEvents))
	for i := range c.topicBusEvents {
		if selectedAll || c.topicBusEvents[i].Topic == selectedTopic {
			out = append(out, i)
		}
	}
	c.topicBusFilteredIdx = out
}

func (c *Controller) getTopicBusFilteredEvent(filteredIdx int) (topicBusEvent, bool) {
	c.topicBusMu.RLock()
	defer c.topicBusMu.RUnlock()
	if filteredIdx < 0 || filteredIdx >= len(c.topicBusFilteredIdx) {
		return topicBusEvent{}, false
	}
	i := c.topicBusFilteredIdx[filteredIdx]
	if i < 0 || i >= len(c.topicBusEvents) {
		return topicBusEvent{}, false
	}
	return c.topicBusEvents[i], true
}

func (c *Controller) applyTopicBusMaxEvents(w fyne.Window) {
	if c == nil || c.topicBusMaxEntry == nil {
		return
	}
	raw := strings.TrimSpace(c.topicBusMaxEntry.Text)
	if raw == "" {
		raw = c.topicBusMaxEntry.PlaceHolder
	}
	max64, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
	if err != nil || max64 <= 0 {
		dialog.ShowError(fmt.Errorf("事件上限必须是正整数"), resolveWindow(c.app, c.mainWin, w))
		return
	}
	c.topicBusMu.Lock()
	c.topicBusMaxEvents = int(max64)
	c.trimTopicBusEventsLocked()
	c.rebuildTopicBusFilterLocked()
	c.topicBusMu.Unlock()
	c.saveTopicBusPrefs()
	c.refreshTopicBusUI()
}

func (c *Controller) clearTopicBusEvents() {
	if c == nil {
		return
	}
	c.topicBusMu.Lock()
	c.topicBusEvents = nil
	c.topicBusFilteredIdx = nil
	c.topicBusMu.Unlock()
	runOnMain(c, func() {
		if c.topicBusEventList != nil {
			c.topicBusEventList.UnselectAll()
			c.topicBusEventList.Refresh()
		}
		if c.topicBusDetail != nil {
			c.topicBusDetail.SetText("")
		}
	})
}

func (c *Controller) subscribeTopicBusFromInput(w fyne.Window) {
	topics := parseTopics(valueOrPlaceholder(c.topicBusInput))
	if len(topics) == 0 {
		dialog.ShowError(fmt.Errorf("topic 不能为空"), resolveWindow(c.app, c.mainWin, w))
		return
	}
	c.topicBusMu.Lock()
	c.topicBusSubs = mergeTopics(c.topicBusSubs, topics)
	c.topicBusMu.Unlock()
	c.saveTopicBusPrefs()
	c.refreshTopicBusUI()
	go c.sendTopicBusSubscribe(topics)
}

func (c *Controller) unsubscribeTopicBusFromInput(w fyne.Window) {
	topics := parseTopics(valueOrPlaceholder(c.topicBusInput))
	if len(topics) == 0 {
		dialog.ShowError(fmt.Errorf("topic 不能为空"), resolveWindow(c.app, c.mainWin, w))
		return
	}
	c.topicBusMu.Lock()
	c.topicBusSubs = removeTopics(c.topicBusSubs, topics)
	if c.topicBusSelectedSub > 0 {
		c.topicBusSelectedSub = 0
	}
	c.topicBusMu.Unlock()
	c.saveTopicBusPrefs()
	c.refreshTopicBusUI()
	go c.sendTopicBusUnsubscribe(topics)
}

func (c *Controller) unsubscribeTopicBusSelected(w fyne.Window) {
	c.topicBusMu.RLock()
	idx := c.topicBusSelectedSub - 1
	if idx < 0 || idx >= len(c.topicBusSubs) {
		c.topicBusMu.RUnlock()
		dialog.ShowError(fmt.Errorf("请先在左侧选择一个 topic（非“全部”）"), resolveWindow(c.app, c.mainWin, w))
		return
	}
	topic := c.topicBusSubs[idx]
	c.topicBusMu.RUnlock()
	c.topicBusMu.Lock()
	c.topicBusSubs = removeTopics(c.topicBusSubs, []string{topic})
	c.topicBusSelectedSub = 0
	c.topicBusMu.Unlock()
	c.saveTopicBusPrefs()
	c.refreshTopicBusUI()
	go c.sendTopicBusUnsubscribe([]string{topic})
}

func (c *Controller) selectedTopicBusTopic() (string, bool) {
	c.topicBusMu.RLock()
	defer c.topicBusMu.RUnlock()
	idx := c.topicBusSelectedSub - 1
	if idx < 0 || idx >= len(c.topicBusSubs) {
		return "", false
	}
	topic := strings.TrimSpace(c.topicBusSubs[idx])
	if topic == "" {
		return "", false
	}
	return topic, true
}

func (c *Controller) fillPublishTopicFromSelection(w fyne.Window) {
	if c == nil || c.topicBusPubTopic == nil {
		return
	}
	topic, ok := c.selectedTopicBusTopic()
	if !ok {
		dialog.ShowError(fmt.Errorf("请先在订阅列表选择一个 topic（非“全部”）"), resolveWindow(c.app, c.mainWin, w))
		return
	}
	c.topicBusPubTopic.SetText(topic)
}

func (c *Controller) clearTopicBusPublishInputs() {
	runOnMain(c, func() {
		if c.topicBusPubTopic != nil {
			c.topicBusPubTopic.SetText("")
		}
		if c.topicBusPubName != nil {
			c.topicBusPubName.SetText("")
		}
		if c.topicBusPubPayload != nil {
			c.topicBusPubPayload.SetText("")
		}
	})
}

func (c *Controller) publishTopicBus(w fyne.Window) {
	win := resolveWindow(c.app, c.mainWin, w)
	if c == nil || c.session == nil {
		return
	}
	if !c.loggedIn || c.storedNode == 0 {
		dialog.ShowError(fmt.Errorf("请先登录"), win)
		return
	}
	target, err := c.parseTopicBusTarget()
	if err != nil {
		dialog.ShowError(err, win)
		return
	}
	if target == 0 {
		dialog.ShowError(fmt.Errorf("TargetID 不能为空"), win)
		return
	}

	topic := ""
	if c.topicBusPubTopic != nil {
		topic = strings.TrimSpace(c.topicBusPubTopic.Text)
	}
	if topic == "" {
		if sel, ok := c.selectedTopicBusTopic(); ok {
			topic = sel
		}
	}
	if topic == "" {
		dialog.ShowError(fmt.Errorf("topic 不能为空"), win)
		return
	}

	name := ""
	if c.topicBusPubName != nil {
		name = strings.TrimSpace(c.topicBusPubName.Text)
	}
	if name == "" {
		dialog.ShowError(fmt.Errorf("name 不能为空"), win)
		return
	}

	payloadText := ""
	if c.topicBusPubPayload != nil {
		payloadText = strings.TrimSpace(c.topicBusPubPayload.Text)
	}
	var payload json.RawMessage
	if payloadText != "" {
		if json.Valid([]byte(payloadText)) {
			payload = json.RawMessage(payloadText)
		} else {
			raw, _ := json.Marshal(payloadText)
			payload = raw
		}
	}

	data, err := json.Marshal(topicBusPublish{
		Topic:   topic,
		Name:    name,
		TS:      time.Now().UnixMilli(),
		Payload: payload,
	})
	if err != nil {
		dialog.ShowError(err, win)
		return
	}
	frame, err := json.Marshal(topicBusMessage{Action: topicBusActionPublish, Data: data})
	if err != nil {
		dialog.ShowError(err, win)
		return
	}
	hdr := c.buildTopicBusHeader(target)
	if err := c.session.Send(hdr, frame); err != nil {
		dialog.ShowError(err, win)
		return
	}
	c.logTx(fmt.Sprintf("[TOPICBUS TX publish topic=%q name=%q]", topic, name), hdr, frame)
}

func (c *Controller) resubscribeTopicBus() {
	if c == nil {
		return
	}
	if !c.loggedIn || c.storedNode == 0 {
		return
	}
	target, err := c.parseTopicBusTarget()
	if err != nil || target == 0 {
		if err != nil {
			c.appendLog("[TOPICBUS][ERR] parse target: %v", err)
		}
		return
	}
	c.topicBusMu.RLock()
	topics := append([]string(nil), c.topicBusSubs...)
	c.topicBusMu.RUnlock()
	if len(topics) == 0 {
		return
	}
	_ = c.sendTopicBusSubscribeBatch(topics, target)
}

func (c *Controller) sendTopicBusSubscribe(topics []string) {
	if len(topics) == 0 {
		return
	}
	if !c.loggedIn || c.storedNode == 0 {
		c.appendLog("[TOPICBUS][WARN] 未登录，已仅保存订阅列表；登录后可点“重订阅”或等待自动重订阅")
		return
	}
	target, err := c.parseTopicBusTarget()
	if err != nil || target == 0 {
		if err != nil {
			c.appendLog("[TOPICBUS][ERR] parse target: %v", err)
		} else {
			c.appendLog("[TOPICBUS][ERR] TargetID 为空，无法订阅")
		}
		return
	}
	if len(topics) == 1 {
		_ = c.sendTopicBusSubscribeOne(topics[0], target)
		return
	}
	_ = c.sendTopicBusSubscribeBatch(topics, target)
}

func (c *Controller) sendTopicBusUnsubscribe(topics []string) {
	if len(topics) == 0 {
		return
	}
	if !c.loggedIn || c.storedNode == 0 {
		c.appendLog("[TOPICBUS][WARN] 未登录，已仅更新订阅列表；登录后不会自动退订（连接断开会自动清理）")
		return
	}
	target, err := c.parseTopicBusTarget()
	if err != nil || target == 0 {
		if err != nil {
			c.appendLog("[TOPICBUS][ERR] parse target: %v", err)
		} else {
			c.appendLog("[TOPICBUS][ERR] TargetID 为空，无法退订")
		}
		return
	}
	if len(topics) == 1 {
		_ = c.sendTopicBusUnsubscribeOne(topics[0], target)
		return
	}
	_ = c.sendTopicBusUnsubscribeBatch(topics, target)
}

func (c *Controller) sendTopicBusSubscribeOne(topic string, target uint32) error {
	if strings.TrimSpace(topic) == "" || target == 0 {
		return nil
	}
	payload, err := json.Marshal(topicBusMessage{
		Action: topicBusActionSubscribe,
		Data:   mustJSONRaw(topicBusSubscribeReq{Topic: topic}),
	})
	if err != nil {
		return err
	}
	hdr := c.buildTopicBusHeader(target)
	if err := c.session.Send(hdr, payload); err != nil {
		return err
	}
	c.logTx(fmt.Sprintf("[TOPICBUS TX subscribe %q]", topic), hdr, payload)
	return nil
}

func (c *Controller) sendTopicBusSubscribeBatch(topics []string, target uint32) error {
	topics = uniqueStableStrings(filterNonEmpty(topics))
	if len(topics) == 0 || target == 0 {
		return nil
	}
	payload, err := json.Marshal(topicBusMessage{
		Action: topicBusActionSubscribeBatch,
		Data:   mustJSONRaw(topicBusSubscribeBatchReq{Topics: topics}),
	})
	if err != nil {
		return err
	}
	hdr := c.buildTopicBusHeader(target)
	if err := c.session.Send(hdr, payload); err != nil {
		return err
	}
	c.logTx(fmt.Sprintf("[TOPICBUS TX subscribe_batch %d topics]", len(topics)), hdr, payload)
	return nil
}

func (c *Controller) sendTopicBusUnsubscribeOne(topic string, target uint32) error {
	if strings.TrimSpace(topic) == "" || target == 0 {
		return nil
	}
	payload, err := json.Marshal(topicBusMessage{
		Action: topicBusActionUnsubscribe,
		Data:   mustJSONRaw(topicBusSubscribeReq{Topic: topic}),
	})
	if err != nil {
		return err
	}
	hdr := c.buildTopicBusHeader(target)
	if err := c.session.Send(hdr, payload); err != nil {
		return err
	}
	c.logTx(fmt.Sprintf("[TOPICBUS TX unsubscribe %q]", topic), hdr, payload)
	return nil
}

func (c *Controller) sendTopicBusUnsubscribeBatch(topics []string, target uint32) error {
	topics = uniqueStableStrings(filterNonEmpty(topics))
	if len(topics) == 0 || target == 0 {
		return nil
	}
	payload, err := json.Marshal(topicBusMessage{
		Action: topicBusActionUnsubscribeBatch,
		Data:   mustJSONRaw(topicBusSubscribeBatchReq{Topics: topics}),
	})
	if err != nil {
		return err
	}
	hdr := c.buildTopicBusHeader(target)
	if err := c.session.Send(hdr, payload); err != nil {
		return err
	}
	c.logTx(fmt.Sprintf("[TOPICBUS TX unsubscribe_batch %d topics]", len(topics)), hdr, payload)
	return nil
}

func (c *Controller) buildTopicBusHeader(target uint32) core.IHeader {
	return (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(subProtoTopicBus).
		WithSourceID(c.storedNode).
		WithTargetID(target).
		WithMsgID(uint32(time.Now().UnixNano())).
		WithTimestamp(uint32(time.Now().Unix()))
}

func (c *Controller) parseTopicBusTarget() (uint32, error) {
	if c == nil {
		return 0, nil
	}
	if c.topicBusTarget == nil {
		if c.storedHub != 0 {
			return c.storedHub, nil
		}
		return 0, nil
	}
	text := strings.TrimSpace(c.topicBusTarget.Text)
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

func parseTopics(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case '\n', ',', '，', ';', '；':
			return true
		default:
			return false
		}
	})
	out := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func mergeTopics(existing []string, add []string) []string {
	if len(add) == 0 {
		return existing
	}
	seen := make(map[string]bool, len(existing)+len(add))
	out := make([]string, 0, len(existing)+len(add))
	for _, t := range existing {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	for _, t := range add {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func removeTopics(existing []string, remove []string) []string {
	if len(existing) == 0 || len(remove) == 0 {
		return existing
	}
	rm := make(map[string]bool, len(remove))
	for _, t := range remove {
		t = strings.TrimSpace(t)
		if t != "" {
			rm[t] = true
		}
	}
	if len(rm) == 0 {
		return existing
	}
	out := make([]string, 0, len(existing))
	for _, t := range existing {
		t = strings.TrimSpace(t)
		if t == "" || rm[t] {
			continue
		}
		out = append(out, t)
	}
	return out
}

func filterNonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func uniqueStableStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func mustJSONRaw(v any) json.RawMessage {
	raw, _ := json.Marshal(v)
	return raw
}

func prettyJSON(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var out bytes.Buffer
	if err := json.Indent(&out, raw, "", "  "); err != nil {
		return string(raw)
	}
	return out.String()
}

func formatTopicBusEventLine(ev topicBusEvent) string {
	// 采用本地时间显示；若 TS 为 0，则只显示 name/topic。
	ts := ""
	if ev.TS > 0 {
		ts = time.UnixMilli(ev.TS).Format("2006-01-02 15:04:05.000")
	}
	if ts == "" {
		return fmt.Sprintf("%s | %s", ev.Topic, ev.Name)
	}
	return fmt.Sprintf("%s | %s | %s", ev.Topic, ev.Name, ts)
}
