package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"strconv"
	"strings"
	"time"

	"github.com/yttydcs/myflowhub-core/header"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const (
	topicBusStressName         = "stress"
	topicBusStressUIUpdateTick = 200 * time.Millisecond
)

type topicBusStressPayload struct {
	Run   string `json:"run"`
	Seq   int    `json:"seq"`
	Total int    `json:"total"`
	Size  int    `json:"size"`
	Data  string `json:"data,omitempty"`
	CRC   uint32 `json:"crc"`
}

func (c *Controller) buildTopicBusStressReceiverCard(w fyne.Window) fyne.CanvasObject {
	targetEntry := widget.NewEntry()
	targetEntry.SetPlaceHolder("TargetID（留空=HubID）")
	if c.storedHub != 0 {
		targetEntry.SetText(fmt.Sprintf("%d", c.storedHub))
	}
	topicEntry := widget.NewEntry()
	topicEntry.SetPlaceHolder("topic（不能为空）")
	runEntry := widget.NewEntry()
	runEntry.SetPlaceHolder("run_id（发送方生成后填入，不能为空）")
	totalEntry := widget.NewEntry()
	totalEntry.SetPlaceHolder("期望条数（例如 10000）")
	sizeEntry := widget.NewEntry()
	sizeEntry.SetPlaceHolder("payload 大小（字节，可选，默认 0）")

	status := widget.NewLabel("未开始")
	status.Wrapping = fyne.TextWrapWord
	c.topicBusStressRecvStatus = status
	c.refreshTopicBusStressRecvStatus()

	startBtn := widget.NewButton("开始接收(订阅)", func() {
		win := resolveWindow(c.app, c.mainWin, w)
		cfg, err := c.parseTopicBusStressConfig(targetEntry, topicEntry, runEntry, totalEntry, sizeEntry, true)
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		if err := c.startTopicBusStressReceiver(cfg); err != nil {
			dialog.ShowError(err, win)
			return
		}
	})
	stopBtn := widget.NewButton("停止接收(退订)", func() {
		win := resolveWindow(c.app, c.mainWin, w)
		target, err := c.parseTargetWithFallback(targetEntry)
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		topic := strings.TrimSpace(valueOrPlaceholder(topicEntry))
		if strings.TrimSpace(topic) == "" {
			dialog.ShowError(fmt.Errorf("topic 不能为空"), win)
			return
		}
		_ = c.sendTopicBusUnsubscribeOne(topic, target)
		c.stopTopicBusStressReceiver()
	})
	resetBtn := widget.NewButton("重置统计", func() { c.resetTopicBusStressReceiver() })

	body := container.NewVBox(
		widget.NewLabel("接收方：订阅 topic 并统计丢包/重复/校验；发送方 publish 不回显，需要另一端节点发送。"),
		container.NewGridWithColumns(2,
			labeledEntry("TargetID", targetEntry),
			labeledEntry("Topic", topicEntry),
			labeledEntry("RunID", runEntry),
			labeledEntry("期望条数", totalEntry),
			labeledEntry("PayloadSize", sizeEntry),
			container.NewHBox(startBtn, stopBtn, resetBtn),
		),
		widget.NewSeparator(),
		status,
	)
	return widget.NewCard("TopicBus 压测 - 接收方", "SubProto=4（publish/subscribe），统计丢包与数据正确性", body)
}

func (c *Controller) buildTopicBusStressSenderCard(w fyne.Window) fyne.CanvasObject {
	targetEntry := widget.NewEntry()
	targetEntry.SetPlaceHolder("TargetID（留空=HubID）")
	if c.storedHub != 0 {
		targetEntry.SetText(fmt.Sprintf("%d", c.storedHub))
	}
	topicEntry := widget.NewEntry()
	topicEntry.SetPlaceHolder("topic（不能为空）")
	runEntry := widget.NewEntry()
	runEntry.SetPlaceHolder("run_id（建议生成后复制给接收方）")
	totalEntry := widget.NewEntry()
	totalEntry.SetPlaceHolder("发送条数（例如 10000）")
	sizeEntry := widget.NewEntry()
	sizeEntry.SetPlaceHolder("payload 大小（字节，可选，默认 0）")

	status := widget.NewLabel("未开始")
	status.Wrapping = fyne.TextWrapWord
	c.topicBusStressSendStatus = status
	c.refreshTopicBusStressSendStatus()

	genRunBtn := widget.NewButton("生成 RunID", func() {
		if strings.TrimSpace(runEntry.Text) != "" {
			return
		}
		runEntry.SetText(buildStressRunID())
	})
	startBtn := widget.NewButton("开始发送", func() {
		win := resolveWindow(c.app, c.mainWin, w)
		if strings.TrimSpace(runEntry.Text) == "" {
			runEntry.SetText(buildStressRunID())
		}
		cfg, err := c.parseTopicBusStressConfig(targetEntry, topicEntry, runEntry, totalEntry, sizeEntry, false)
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		if err := c.startTopicBusStressSender(cfg); err != nil {
			dialog.ShowError(err, win)
			return
		}
	})
	stopBtn := widget.NewButton("停止发送", func() { c.stopTopicBusStressSender() })

	body := container.NewVBox(
		widget.NewLabel("发送方：向 topic 连续 publish（name=stress），payload 内含 run/seq/crc。"),
		container.NewGridWithColumns(2,
			labeledEntry("TargetID", targetEntry),
			labeledEntry("Topic", topicEntry),
			container.NewBorder(nil, nil, nil, genRunBtn, labeledEntry("RunID", runEntry)),
			labeledEntry("发送条数", totalEntry),
			labeledEntry("PayloadSize", sizeEntry),
			container.NewHBox(startBtn, stopBtn, layout.NewSpacer()),
		),
		widget.NewSeparator(),
		status,
	)
	return widget.NewCard("TopicBus 压测 - 发送方", "建议与接收方在不同客户端/不同连接上运行", body)
}

type topicBusStressConfig struct {
	Target      uint32
	Topic       string
	Run         string
	Total       int
	PayloadSize int
}

func (c *Controller) parseTopicBusStressConfig(targetEntry, topicEntry, runEntry, totalEntry, sizeEntry *widget.Entry, requireRun bool) (topicBusStressConfig, error) {
	target, err := c.parseTargetWithFallback(targetEntry)
	if err != nil {
		return topicBusStressConfig{}, err
	}
	if target == 0 {
		return topicBusStressConfig{}, fmt.Errorf("TargetID 不能为空")
	}
	topic := strings.TrimSpace(valueOrPlaceholder(topicEntry))
	if topic == "" {
		return topicBusStressConfig{}, fmt.Errorf("topic 不能为空")
	}
	run := strings.TrimSpace(valueOrPlaceholder(runEntry))
	if requireRun && run == "" {
		return topicBusStressConfig{}, fmt.Errorf("run_id 不能为空")
	}
	total, err := parsePositiveInt(valueOrPlaceholder(totalEntry), "条数")
	if err != nil {
		return topicBusStressConfig{}, err
	}
	size := 0
	if strings.TrimSpace(valueOrPlaceholder(sizeEntry)) != "" {
		v, err := parseNonNegativeInt(valueOrPlaceholder(sizeEntry), "PayloadSize")
		if err != nil {
			return topicBusStressConfig{}, err
		}
		size = v
	}
	return topicBusStressConfig{Target: target, Topic: topic, Run: run, Total: total, PayloadSize: size}, nil
}

func (c *Controller) parseTargetWithFallback(entry *widget.Entry) (uint32, error) {
	if entry == nil {
		return c.storedHub, nil
	}
	text := strings.TrimSpace(valueOrPlaceholder(entry))
	if text == "" {
		return c.storedHub, nil
	}
	v, err := strconv.ParseUint(text, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("目标 NodeID 不是合法数字")
	}
	return uint32(v), nil
}

func parsePositiveInt(text, field string) (int, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, fmt.Errorf("%s 不能为空", field)
	}
	v, err := strconv.ParseInt(text, 10, 32)
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("%s 必须是正整数", field)
	}
	return int(v), nil
}

func parseNonNegativeInt(text, field string) (int, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(text, 10, 32)
	if err != nil || v < 0 {
		return 0, fmt.Errorf("%s 必须是非负整数", field)
	}
	return int(v), nil
}

func buildStressRunID() string {
	return fmt.Sprintf("run-%d-%s", time.Now().UnixMilli(), generateNonce(6))
}

func (c *Controller) startTopicBusStressReceiver(cfg topicBusStressConfig) error {
	if c == nil || c.session == nil {
		return nil
	}
	if !c.loggedIn || c.storedNode == 0 {
		return fmt.Errorf("请先登录")
	}
	if strings.TrimSpace(cfg.Run) == "" {
		return fmt.Errorf("run_id 不能为空")
	}
	if err := c.sendTopicBusSubscribeOne(cfg.Topic, cfg.Target); err != nil {
		return err
	}

	c.topicBusStressMu.Lock()
	c.topicBusStressRecvActive = true
	c.topicBusStressRecvTopic = cfg.Topic
	c.topicBusStressRecvRun = cfg.Run
	c.topicBusStressRecvExpected = cfg.Total
	c.topicBusStressRecvPayloadSize = cfg.PayloadSize
	c.topicBusStressRecvStartedAt = time.Now()
	c.topicBusStressRecvLastUI = time.Time{}
	c.topicBusStressRecvBitset = make([]uint64, (cfg.Total+63)/64)
	c.topicBusStressRecvRx = 0
	c.topicBusStressRecvUnique = 0
	c.topicBusStressRecvDup = 0
	c.topicBusStressRecvCorrupt = 0
	c.topicBusStressRecvInvalid = 0
	c.topicBusStressRecvOutOfOrder = 0
	c.topicBusStressRecvLastSeq = 0
	c.topicBusStressMu.Unlock()

	c.refreshTopicBusStressRecvStatus()
	return nil
}

func (c *Controller) stopTopicBusStressReceiver() {
	if c == nil {
		return
	}
	c.topicBusStressMu.Lock()
	c.topicBusStressRecvActive = false
	c.topicBusStressMu.Unlock()
	c.refreshTopicBusStressRecvStatus()
}

func (c *Controller) resetTopicBusStressReceiver() {
	if c == nil {
		return
	}
	c.topicBusStressMu.Lock()
	c.topicBusStressRecvActive = false
	c.topicBusStressRecvTopic = ""
	c.topicBusStressRecvRun = ""
	c.topicBusStressRecvExpected = 0
	c.topicBusStressRecvPayloadSize = 0
	c.topicBusStressRecvStartedAt = time.Time{}
	c.topicBusStressRecvLastUI = time.Time{}
	c.topicBusStressRecvBitset = nil
	c.topicBusStressRecvRx = 0
	c.topicBusStressRecvUnique = 0
	c.topicBusStressRecvDup = 0
	c.topicBusStressRecvCorrupt = 0
	c.topicBusStressRecvInvalid = 0
	c.topicBusStressRecvOutOfOrder = 0
	c.topicBusStressRecvLastSeq = 0
	c.topicBusStressMu.Unlock()
	c.refreshTopicBusStressRecvStatus()
}

func (c *Controller) startTopicBusStressSender(cfg topicBusStressConfig) error {
	if c == nil || c.session == nil {
		return nil
	}
	if !c.loggedIn || c.storedNode == 0 {
		return fmt.Errorf("请先登录")
	}
	if strings.TrimSpace(cfg.Run) == "" {
		return fmt.Errorf("run_id 不能为空")
	}

	c.topicBusStressMu.Lock()
	if c.topicBusStressSendActive {
		c.topicBusStressMu.Unlock()
		return fmt.Errorf("发送方压测进行中")
	}
	ctx := c.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	sendCtx, cancel := context.WithCancel(ctx)
	c.topicBusStressSendActive = true
	c.topicBusStressSendCancel = cancel
	c.topicBusStressSendTopic = cfg.Topic
	c.topicBusStressSendRun = cfg.Run
	c.topicBusStressSendTotal = cfg.Total
	c.topicBusStressSendPayloadSize = cfg.PayloadSize
	c.topicBusStressSendStartedAt = time.Now()
	c.topicBusStressSendLastUI = time.Time{}
	c.topicBusStressSendSent = 0
	c.topicBusStressSendErrors = 0
	c.topicBusStressMu.Unlock()

	c.refreshTopicBusStressSendStatus()
	go c.runTopicBusStressSender(sendCtx, cfg)
	return nil
}

func (c *Controller) stopTopicBusStressSender() {
	if c == nil {
		return
	}
	var cancel context.CancelFunc
	c.topicBusStressMu.Lock()
	cancel = c.topicBusStressSendCancel
	c.topicBusStressSendCancel = nil
	c.topicBusStressMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (c *Controller) runTopicBusStressSender(ctx context.Context, cfg topicBusStressConfig) {
	data := ""
	if cfg.PayloadSize > 0 {
		data = strings.Repeat("x", cfg.PayloadSize)
	}
	h := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(subProtoTopicBus).
		WithSourceID(c.storedNode).
		WithTargetID(cfg.Target)

	buf := make([]byte, 0, len(cfg.Run)+len(data)+64)
	for seq := 1; seq <= cfg.Total; seq++ {
		select {
		case <-ctx.Done():
			c.finishTopicBusStressSender()
			return
		default:
		}
		payload := topicBusStressPayload{
			Run:   cfg.Run,
			Seq:   seq,
			Total: cfg.Total,
			Size:  cfg.PayloadSize,
			Data:  data,
		}
		payload.CRC = stressCRC32(&buf, payload.Run, payload.Seq, payload.Total, payload.Data)
		rawPayload, _ := json.Marshal(payload)
		pub := topicBusPublish{
			Topic:   cfg.Topic,
			Name:    topicBusStressName,
			TS:      time.Now().UnixMilli(),
			Payload: rawPayload,
		}
		rawPub, _ := json.Marshal(pub)
		frame, _ := json.Marshal(topicBusMessage{Action: topicBusActionPublish, Data: rawPub})

		now := time.Now()
		h.WithMsgID(uint32(now.UnixNano())).WithTimestamp(uint32(now.Unix()))
		if err := c.session.Send(h, frame); err != nil {
			c.topicBusStressMu.Lock()
			c.topicBusStressSendErrors++
			c.topicBusStressMu.Unlock()
		}
		needUpdate := c.bumpTopicBusStressSendSent()
		if needUpdate {
			c.refreshTopicBusStressSendStatus()
		}
	}
	c.finishTopicBusStressSender()
}

func (c *Controller) bumpTopicBusStressSendSent() bool {
	now := time.Now()
	c.topicBusStressMu.Lock()
	c.topicBusStressSendSent++
	should := now.Sub(c.topicBusStressSendLastUI) >= topicBusStressUIUpdateTick
	if should {
		c.topicBusStressSendLastUI = now
	}
	c.topicBusStressMu.Unlock()
	return should
}

func (c *Controller) finishTopicBusStressSender() {
	c.topicBusStressMu.Lock()
	c.topicBusStressSendActive = false
	c.topicBusStressSendCancel = nil
	c.topicBusStressMu.Unlock()
	c.refreshTopicBusStressSendStatus()
}

func stressCRC32(buf *[]byte, run string, seq, total int, data string) uint32 {
	b := *buf
	b = b[:0]
	b = append(b, run...)
	b = append(b, '|')
	b = strconv.AppendInt(b, int64(seq), 10)
	b = append(b, '|')
	b = strconv.AppendInt(b, int64(total), 10)
	b = append(b, '|')
	b = append(b, data...)
	*buf = b
	return crc32.ChecksumIEEE(b)
}

func (c *Controller) onTopicBusStressPublish(pub topicBusPublish) {
	if c == nil {
		return
	}
	// 快速过滤：仅处理 name=stress。
	if pub.Name != topicBusStressName {
		return
	}

	// 复制配置，避免长时间持锁反序列化。
	c.topicBusStressMu.Lock()
	active := c.topicBusStressRecvActive
	topic := c.topicBusStressRecvTopic
	run := c.topicBusStressRecvRun
	expect := c.topicBusStressRecvExpected
	expectSize := c.topicBusStressRecvPayloadSize
	c.topicBusStressMu.Unlock()

	if !active || strings.TrimSpace(topic) == "" || strings.TrimSpace(run) == "" || expect <= 0 {
		return
	}
	if pub.Topic != topic {
		return
	}

	var pl topicBusStressPayload
	if err := json.Unmarshal(pub.Payload, &pl); err != nil {
		c.bumpTopicBusStressRecvCounters(false, 0, false, true, false)
		return
	}
	if pl.Run != run {
		return
	}

	validSeq := pl.Seq >= 1 && pl.Seq <= expect
	totalOK := pl.Total == expect
	sizeOK := expectSize == 0 || pl.Size == expectSize
	if !validSeq || !totalOK || !sizeOK {
		c.bumpTopicBusStressRecvCounters(false, pl.Seq, false, false, true)
		return
	}

	buf := make([]byte, 0, len(pl.Run)+len(pl.Data)+64)
	want := stressCRC32(&buf, pl.Run, pl.Seq, pl.Total, pl.Data)
	if want != pl.CRC {
		c.bumpTopicBusStressRecvCounters(false, pl.Seq, true, false, false)
		return
	}

	c.bumpTopicBusStressRecvCounters(true, pl.Seq, false, false, false)
}

func (c *Controller) bumpTopicBusStressRecvCounters(ok bool, seq int, corrupt, parseFail, invalid bool) {
	if c == nil {
		return
	}
	now := time.Now()
	shouldUpdate := false
	c.topicBusStressMu.Lock()
	if !c.topicBusStressRecvActive {
		c.topicBusStressMu.Unlock()
		return
	}
	c.topicBusStressRecvRx++
	if parseFail || corrupt {
		c.topicBusStressRecvCorrupt++
	} else if invalid {
		c.topicBusStressRecvInvalid++
	} else if ok && seq >= 1 && seq <= c.topicBusStressRecvExpected {
		idx := seq - 1
		word := idx / 64
		bit := uint(idx % 64)
		mask := uint64(1) << bit
		if c.topicBusStressRecvBitset[word]&mask != 0 {
			c.topicBusStressRecvDup++
		} else {
			c.topicBusStressRecvBitset[word] |= mask
			c.topicBusStressRecvUnique++
			if c.topicBusStressRecvLastSeq != 0 && seq < c.topicBusStressRecvLastSeq {
				c.topicBusStressRecvOutOfOrder++
			}
			c.topicBusStressRecvLastSeq = seq
		}
	} else {
		c.topicBusStressRecvInvalid++
	}

	if now.Sub(c.topicBusStressRecvLastUI) >= topicBusStressUIUpdateTick || c.topicBusStressRecvUnique >= c.topicBusStressRecvExpected {
		c.topicBusStressRecvLastUI = now
		shouldUpdate = true
	}
	c.topicBusStressMu.Unlock()

	if shouldUpdate {
		c.refreshTopicBusStressRecvStatus()
	}
}

func (c *Controller) refreshTopicBusStressRecvStatus() {
	if c == nil || c.topicBusStressRecvStatus == nil {
		return
	}
	text := c.topicBusStressRecvStatusText()
	runOnMain(c, func() {
		if c.topicBusStressRecvStatus != nil {
			c.topicBusStressRecvStatus.SetText(text)
		}
	})
}

func (c *Controller) refreshTopicBusStressSendStatus() {
	if c == nil || c.topicBusStressSendStatus == nil {
		return
	}
	text := c.topicBusStressSendStatusText()
	runOnMain(c, func() {
		if c.topicBusStressSendStatus != nil {
			c.topicBusStressSendStatus.SetText(text)
		}
	})
}

func (c *Controller) topicBusStressRecvStatusText() string {
	c.topicBusStressMu.Lock()
	defer c.topicBusStressMu.Unlock()
	if c.topicBusStressRecvExpected <= 0 || strings.TrimSpace(c.topicBusStressRecvTopic) == "" || strings.TrimSpace(c.topicBusStressRecvRun) == "" {
		return "未开始"
	}
	elapsed := time.Since(c.topicBusStressRecvStartedAt)
	missing := c.topicBusStressRecvExpected - c.topicBusStressRecvUnique
	if missing < 0 {
		missing = 0
	}
	rate := float64(0)
	if elapsed > 0 {
		rate = float64(c.topicBusStressRecvUnique) / elapsed.Seconds()
	}
	state := "已停止"
	if c.topicBusStressRecvActive {
		state = "接收中"
		if c.topicBusStressRecvUnique >= c.topicBusStressRecvExpected {
			state = "已完成"
		}
	}
	return fmt.Sprintf(
		"%s\nTopic=%s\nRun=%s\nExpected=%d  PayloadSize=%d\nRx=%d  Unique=%d  Missing=%d  Dup=%d\nCorrupt=%d  Invalid=%d  OutOfOrder=%d\nElapsed=%s  Rate=%.0f msg/s",
		state,
		c.topicBusStressRecvTopic,
		c.topicBusStressRecvRun,
		c.topicBusStressRecvExpected,
		c.topicBusStressRecvPayloadSize,
		c.topicBusStressRecvRx,
		c.topicBusStressRecvUnique,
		missing,
		c.topicBusStressRecvDup,
		c.topicBusStressRecvCorrupt,
		c.topicBusStressRecvInvalid,
		c.topicBusStressRecvOutOfOrder,
		elapsed.Round(100*time.Millisecond),
		rate,
	)
}

func (c *Controller) topicBusStressSendStatusText() string {
	c.topicBusStressMu.Lock()
	defer c.topicBusStressMu.Unlock()
	if c.topicBusStressSendTotal <= 0 || strings.TrimSpace(c.topicBusStressSendTopic) == "" || strings.TrimSpace(c.topicBusStressSendRun) == "" {
		return "未开始"
	}
	elapsed := time.Since(c.topicBusStressSendStartedAt)
	rate := float64(0)
	if elapsed > 0 {
		rate = float64(c.topicBusStressSendSent) / elapsed.Seconds()
	}
	state := "已停止"
	if c.topicBusStressSendActive {
		state = "发送中"
		if c.topicBusStressSendSent >= c.topicBusStressSendTotal {
			state = "已完成"
		}
	}
	return fmt.Sprintf(
		"%s\nTopic=%s\nRun=%s\nTotal=%d  PayloadSize=%d\nSent=%d  Errors=%d\nElapsed=%s  Rate=%.0f msg/s",
		state,
		c.topicBusStressSendTopic,
		c.topicBusStressSendRun,
		c.topicBusStressSendTotal,
		c.topicBusStressSendPayloadSize,
		c.topicBusStressSendSent,
		c.topicBusStressSendErrors,
		elapsed.Round(100*time.Millisecond),
		rate,
	)
}

