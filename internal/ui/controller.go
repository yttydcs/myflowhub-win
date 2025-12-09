package ui

import (
	"bytes"
	"context"
	"sync"

	core "github.com/yttydcs/myflowhub-core"
	"github.com/yttydcs/myflowhub-win/internal/session"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Controller 是 UI 的聚合入口，持有共享状态与跨模块依赖。
type Controller struct {
	app fyne.App
	ctx context.Context

	// session & logs
	session *session.Session
	logBuf  bytes.Buffer
	logMu   sync.Mutex

	// main window
	mainWin   fyne.Window
	connected bool

	// log window/popup
	logPopup  *logEntry
	logWindow fyne.Window

	// profile & home
	profileSelect  *widget.Select
	currentProfile string
	profiles       []string
	homeLoading    bool
	homeLastAddr   string

	addrEntry    *widget.Entry
	homeAddr     *widget.Entry
	homeAutoCon  *widget.Check
	homeAutoLog  *widget.Check
	homeDevice   *widget.Entry
	homeCred     *widget.Label
	homeNode     *widget.Label
	homeHub      *widget.Label
	homeRole     *widget.Label
	homeLoginBtn *widget.Button
	homeClearBtn *widget.Button
	homeConnCard *widget.Card

	storedCred string
	storedNode uint32
	storedHub  uint32
	storedRole string

	// debug tab
	payload     *widget.Entry
	logView     *logEntry
	form        *headerForm
	nodeEntry   *widget.Entry
	hexToggle   *widget.Check
	truncToggle *widget.Check
	showHex     *widget.Check
	presetSrc   *widget.Entry
	presetTgt   *widget.Entry

	// config tab
	configRows []*configRow
	configList *fyne.Container
	configInfo *widget.Label

	// preset tab
	presetCards []*widget.Card

	// var pool tab
	varPoolKeys   []varKey
	varPoolData   map[varKey]varValue
	varPoolList   *fyne.Container
	varPoolTarget *widget.Entry
}

// New 创建 UI 控制器。
func New(app fyne.App, ctx context.Context) *Controller {
	c := &Controller{app: app, ctx: ctx}
	c.session = session.New(c.ctx, c.handleFrame, c.handleError)
	return c
}

// Build 构建主窗口内容。
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

// Shutdown 清理资源。
func (c *Controller) Shutdown() { c.session.Close() }

// handleFrame 分发网络帧。
func (c *Controller) handleFrame(h core.IHeader, payload []byte) {
	preview := c.formatPayloadPreview(payload)
	c.appendLog("[RX] major=%d sub=%d src=%d tgt=%d len=%d %s",
		h.Major(), h.SubProto(), h.SourceID(), h.TargetID(), len(payload), preview)
	c.handleAuthFrame(h, payload)
	if h != nil && h.SubProto() == 3 {
		c.handleVarStoreFrame(payload)
	}
}

// handleError 处理 session 错误。
func (c *Controller) handleError(err error) {
	c.connected = false
	c.setHomeConnStatus(false, c.homeLastAddr)
	c.appendLog("[ERR] %v", err)
}
