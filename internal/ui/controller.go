package ui

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"strings"
	"sync"
	"time"

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
	baseTitle      string

	addrEntry    *widget.Entry
	homeAddr     *widget.Entry
	homeAutoCon  *widget.Check
	homeAutoLog  *widget.Check
	homeDevice   *widget.Entry
	homeNode     *widget.Label
	homeHub      *widget.Label
	homeRole     *widget.Label
	homeLoginBtn *widget.Button
	homeClearBtn *widget.Button
	homeConnCard *widget.Card

	storedNode uint32
	storedHub  uint32
	storedRole string

	nodePriv   *ecdsa.PrivateKey
	nodePubB64 string

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
	varPoolKeys     []varKey
	varPoolData     map[varKey]varValue
	varPoolList     *fyne.Container
	varPoolTarget   *widget.Entry
	varPoolNodeInfo *widget.Label

	// management tab
	mgmtNodes      []mgmtNodeEntry
	mgmtList       *widget.List
	mgmtInfo       *widget.Label
	mgmtTarget     *widget.Entry
	mgmtLastTarget uint32
	mgmtCfgTarget  uint32
	mgmtCfgEntries []mgmtConfigEntry
	mgmtCfgValues  map[string]string
	mgmtCfgList    *widget.List
	mgmtCfgWin     fyne.Window
	mgmtCfgLastKey string
	mgmtCfgLastTap time.Time
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
	c.baseTitle = w.Title()
	c.initProfiles()
	c.loadVarPoolPrefs()
	c.refreshWindowTitle()
	homeTab := c.buildHomeTab(w)
	varPoolTab := c.buildVarPoolTab(w)
	mgmtTab := c.buildManagementTab(w)
	logTab := c.buildLogTab(w)
	debugTab := c.buildDebugTab(w)
	configTab := c.buildConfigTab(w)
	presetTab := c.buildPresetTab(w)
	tabs := container.NewAppTabs(
		container.NewTabItem("首页", homeTab),
		container.NewTabItem("变量池", varPoolTab),
		container.NewTabItem("管理", mgmtTab),
		container.NewTabItem("日志", logTab),
		container.NewTabItem("自定义调试", debugTab),
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

// refreshWindowTitle 更新窗口标题显示的登录信息。
func (c *Controller) refreshWindowTitle() {
	if c.mainWin == nil {
		return
	}
	base := c.baseTitle
	if strings.TrimSpace(base) == "" {
		base = "MyFlowHub Debug Client"
	}
	if c.connected && c.storedNode != 0 {
		c.mainWin.SetTitle(fmt.Sprintf("%s - 已登录 NodeID: %d", base, c.storedNode))
		return
	}
	c.mainWin.SetTitle(base)
}

// handleFrame 分发网络帧。
func (c *Controller) handleFrame(h core.IHeader, payload []byte) {
	preview := c.formatPayloadPreview(payload)
	c.appendLog("[RX] major=%d sub=%d src=%d tgt=%d len=%d %s",
		h.Major(), h.SubProto(), h.SourceID(), h.TargetID(), len(payload), preview)
	c.handleAuthFrame(h, payload)
	if h != nil && h.SubProto() == 1 {
		c.handleManagementFrame(h, payload)
	} else if h != nil && h.SubProto() == 3 {
		c.handleVarStoreFrame(payload)
	}
}

// handleError 处理 session 错误。
func (c *Controller) handleError(err error) {
	c.connected = false
	c.setHomeConnStatus(false, c.homeLastAddr)
	c.appendLog("[ERR] %v", err)
}
