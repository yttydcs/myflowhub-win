package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type fileBrowseKind uint8

const (
	browseKindRoot fileBrowseKind = iota
	browseKindNode
	browseKindDir
	browseKindFile
	browseKindLoading
)

const fileBrowserDoubleTapWindow = 250 * time.Millisecond

type fileEntry struct {
	Name  string
	IsDir bool
	Size  uint64
}

type fileListResult struct {
	Code  int
	Msg   string
	Dir   string
	Dirs  []string
	Files []string
}

type fileTextResult struct {
	Code      int
	Msg       string
	Dir       string
	Name      string
	Size      uint64
	Text      string
	Truncated bool
}

type fileBrowserState struct {
	mu sync.Mutex

	nodes []uint32 // persisted, excludes local node

	tree         *widget.Tree
	treeChildren map[string][]string
	treeLabel    map[string]string
	treeKind     map[string]fileBrowseKind
	treeMeta     map[string]fileEntry

	listCache    map[string]fileListResult // key=nodeID|dir
	listInFlight map[string]time.Time
	textInFlight map[string]time.Time

	lastRemoteNode uint32

	currentUID  string
	currentNode uint32
	currentDir  string

	rightSelected fileEntry
	rightHasSel   bool
	lastListSelID widget.ListItemID
	lastListSelAt time.Time

	rightTitle  *widget.Label
	rightList   *widget.List
	rightItems  []fileEntry
	previewInfo *widget.Label

	previewWindows map[string]*filePreviewWidgets

	upBtn       *widget.Button
	downloadBtn *widget.Button
	uploadBtn   *widget.Button

	removeBtn  *widget.Button
	refreshBtn *widget.Button
}

type filePreviewWidgets struct {
	win  fyne.Window
	text *widget.Entry
	info *widget.Label
}

func newFileBrowserState() *fileBrowserState {
	return &fileBrowserState{
		treeChildren:   make(map[string][]string),
		treeLabel:      make(map[string]string),
		treeKind:       make(map[string]fileBrowseKind),
		treeMeta:       make(map[string]fileEntry),
		listCache:      make(map[string]fileListResult),
		listInFlight:   make(map[string]time.Time),
		textInFlight:   make(map[string]time.Time),
		previewWindows: make(map[string]*filePreviewWidgets),
	}
}

func (c *Controller) loadFileBrowserPrefs() {
	if c == nil || c.app == nil || c.app.Preferences() == nil || c.fileBrowser == nil {
		return
	}
	raw := c.app.Preferences().StringWithFallback(c.prefKey(prefFileBrowserNodes), "")
	var ids []uint32
	_ = json.Unmarshal([]byte(raw), &ids)
	filtered := make([]uint32, 0, len(ids))
	seen := make(map[uint32]bool)
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		filtered = append(filtered, id)
	}
	c.fileBrowser.mu.Lock()
	c.fileBrowser.nodes = filtered
	c.fileBrowser.mu.Unlock()
}

func (c *Controller) saveFileBrowserPrefs() {
	if c == nil || c.app == nil || c.app.Preferences() == nil || c.fileBrowser == nil {
		return
	}
	c.fileBrowser.mu.Lock()
	ids := append([]uint32(nil), c.fileBrowser.nodes...)
	c.fileBrowser.mu.Unlock()
	raw, _ := json.Marshal(ids)
	c.app.Preferences().SetString(c.prefKey(prefFileBrowserNodes), string(raw))
}

func (c *Controller) fileBrowserRootChildren() []string {
	local := c.storedNode
	localUID := fileNodeUID(local, true)

	c.fileBrowser.mu.Lock()
	remotes := append([]uint32(nil), c.fileBrowser.nodes...)
	c.fileBrowser.mu.Unlock()
	sort.Slice(remotes, func(i, j int) bool { return remotes[i] < remotes[j] })

	out := make([]string, 0, 1+len(remotes))
	out = append(out, localUID)
	for _, id := range remotes {
		if id == 0 || id == local {
			continue
		}
		out = append(out, fileNodeUID(id, false))
	}
	return out
}

func fileNodeUID(nodeID uint32, local bool) string {
	if local {
		return fmt.Sprintf("node:local:%d", nodeID)
	}
	return fmt.Sprintf("node:%d", nodeID)
}

func fileDirUID(nodeID uint32, dir string) string {
	dir = strings.Trim(path.Clean(strings.ReplaceAll(strings.TrimSpace(dir), "\\", "/")), "/")
	if dir == "." {
		dir = ""
	}
	return fmt.Sprintf("dir:%d:%s", nodeID, dir)
}

func fileFileUID(nodeID uint32, dir, name string) string {
	dir = strings.Trim(path.Clean(strings.ReplaceAll(strings.TrimSpace(dir), "\\", "/")), "/")
	if dir == "." {
		dir = ""
	}
	name = strings.TrimSpace(name)
	if dir == "" {
		return fmt.Sprintf("file:%d:%s", nodeID, name)
	}
	return fmt.Sprintf("file:%d:%s/%s", nodeID, dir, name)
}

func fileLoadingUID(parent string) string { return "loading:" + parent }

func parseBrowseUID(uid string) (kind fileBrowseKind, nodeID uint32, dir string, name string, local bool) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return browseKindRoot, 0, "", "", false
	}
	if strings.HasPrefix(uid, "loading:") {
		return browseKindLoading, 0, "", "", false
	}
	if strings.HasPrefix(uid, "node:local:") {
		raw := strings.TrimPrefix(uid, "node:local:")
		if n, err := strconv.ParseUint(raw, 10, 32); err == nil {
			return browseKindNode, uint32(n), "", "", true
		}
		return browseKindNode, 0, "", "", true
	}
	if strings.HasPrefix(uid, "node:") {
		raw := strings.TrimPrefix(uid, "node:")
		if n, err := strconv.ParseUint(raw, 10, 32); err == nil {
			return browseKindNode, uint32(n), "", "", false
		}
		return browseKindNode, 0, "", "", false
	}
	if strings.HasPrefix(uid, "dir:") {
		raw := strings.TrimPrefix(uid, "dir:")
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) == 0 {
			return browseKindDir, 0, "", "", false
		}
		n, _ := strconv.ParseUint(parts[0], 10, 32)
		if len(parts) == 1 {
			return browseKindDir, uint32(n), "", "", false
		}
		return browseKindDir, uint32(n), strings.Trim(parts[1], "/"), "", false
	}
	if strings.HasPrefix(uid, "file:") {
		raw := strings.TrimPrefix(uid, "file:")
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) < 2 {
			return browseKindFile, 0, "", "", false
		}
		n, _ := strconv.ParseUint(parts[0], 10, 32)
		p := strings.Trim(parts[1], "/")
		if p == "" {
			return browseKindFile, uint32(n), "", "", false
		}
		dir = path.Dir(p)
		name = path.Base(p)
		if dir == "." {
			dir = ""
		}
		return browseKindFile, uint32(n), dir, name, false
	}
	return browseKindRoot, 0, "", "", false
}

func (c *Controller) buildFileBrowseTab(w fyne.Window) fyne.CanvasObject {
	if c.fileBrowser == nil {
		c.fileBrowser = newFileBrowserState()
	}
	c.loadFileBrowserPrefs()

	leftTitle := widget.NewLabel("节点/目录")
	addBtn := widget.NewButton("添加节点", func() { c.fileBrowserAddNodeDialog(w) })
	removeBtn := widget.NewButton("移除节点", func() { c.fileBrowserRemoveSelectedNode(w) })
	removeBtn.Disable()
	refreshBtn := widget.NewButton("刷新", func() { c.fileBrowserRefreshSelected() })

	c.fileBrowser.mu.Lock()
	c.fileBrowser.removeBtn = removeBtn
	c.fileBrowser.refreshBtn = refreshBtn
	c.fileBrowser.mu.Unlock()

	tree := widget.NewTree(
		func(uid string) []string {
			if c.fileBrowser == nil {
				return nil
			}
			if uid == "" {
				return c.fileBrowserRootChildren()
			}
			c.fileBrowser.mu.Lock()
			children, ok := c.fileBrowser.treeChildren[uid]
			kind, nodeID, dir, _, _ := parseBrowseUID(uid)
			cache, cacheOK := c.fileBrowser.listCache[listKey(nodeID, dir)]
			c.fileBrowser.mu.Unlock()
			if ok {
				return children
			}
			if cacheOK && cache.Code == 1 && (kind == browseKindNode || kind == browseKindDir) {
				dirClean := strings.Trim(path.Clean(strings.ReplaceAll(strings.TrimSpace(dir), "\\", "/")), "/")
				if dirClean == "." {
					dirClean = ""
				}
				rebuild := make([]string, 0, len(cache.Dirs))
				for _, d := range cache.Dirs {
					rebuild = append(rebuild, fileDirUID(nodeID, path.Join(dirClean, d)))
				}
				c.fileBrowser.mu.Lock()
				c.fileBrowser.treeChildren[uid] = rebuild
				c.fileBrowser.mu.Unlock()
				return rebuild
			}
			if kind == browseKindNode {
				c.fileBrowserQueueList(nodeID, "")
				return []string{fileLoadingUID(uid)}
			}
			if kind == browseKindDir {
				c.fileBrowserQueueList(nodeID, dir)
				return []string{fileLoadingUID(uid)}
			}
			return nil
		},
		func(uid string) bool {
			kind, _, _, _, _ := parseBrowseUID(uid)
			return kind == browseKindNode || kind == browseKindDir || uid == ""
		},
		func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(uid string, branch bool, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			kind, nodeID, dir, name, local := parseBrowseUID(uid)
			switch kind {
			case browseKindRoot:
				label.SetText("root")
			case browseKindLoading:
				label.SetText("加载中...")
			case browseKindNode:
				if local {
					if nodeID == 0 {
						label.SetText("本机{未登录}")
					} else {
						label.SetText(fmt.Sprintf("本机{%d}", nodeID))
					}
				} else {
					label.SetText(fmt.Sprintf("节点{%d}", nodeID))
				}
			case browseKindDir:
				if strings.TrimSpace(dir) == "" {
					label.SetText("根目录")
				} else {
					label.SetText(path.Base(dir))
				}
			case browseKindFile:
				label.SetText(name)
			default:
				label.SetText(uid)
			}
		},
	)
	tree.Root = ""
	tree.OnBranchOpened = func(uid string) {
		kind, nodeID, dir, _, _ := parseBrowseUID(uid)
		if kind == browseKindNode {
			c.fileBrowserQueueList(nodeID, "")
			return
		}
		if kind == browseKindDir {
			c.fileBrowserQueueList(nodeID, dir)
		}
	}
	tree.OnSelected = func(uid string) { c.fileBrowserOnSelected(uid) }
	tree.OnUnselected = func(uid string) { c.fileBrowserOnUnselected(uid) }

	c.fileBrowser.mu.Lock()
	c.fileBrowser.tree = tree
	c.fileBrowser.mu.Unlock()

	rightTitle := widget.NewLabel("未选择")
	rightTitle.TextStyle = fyne.TextStyle{Bold: true}

	upBtn := widget.NewButton("上级", func() { c.fileBrowserGoUp() })
	downloadBtn := widget.NewButton("下载", func() { c.fileBrowserDownloadSelected(w) })
	uploadBtn := widget.NewButton("发送(offer)", func() { c.fileBrowserOfferSelected(w) })
	upBtn.Disable()
	downloadBtn.Disable()
	uploadBtn.Disable()
	actionRow := container.NewHBox(upBtn, downloadBtn, uploadBtn)
	var rightList *widget.List
	rightList = widget.NewList(
		func() int {
			c.fileBrowser.mu.Lock()
			defer c.fileBrowser.mu.Unlock()
			return len(c.fileBrowser.rightItems)
		},
		func() fyne.CanvasObject {
			name := widget.NewLabel("")
			name.Wrapping = fyne.TextTruncate
			meta := widget.NewLabel("")
			meta.Wrapping = fyne.TextTruncate
			return container.NewBorder(nil, nil, nil, meta, name)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			c.fileBrowser.mu.Lock()
			if i < 0 || i >= len(c.fileBrowser.rightItems) {
				c.fileBrowser.mu.Unlock()
				return
			}
			e := c.fileBrowser.rightItems[i]
			c.fileBrowser.mu.Unlock()
			row := obj.(*fyne.Container)
			name := row.Objects[0].(*widget.Label)
			meta := row.Objects[1].(*widget.Label)
			if e.IsDir {
				name.SetText("[DIR] " + e.Name)
				meta.SetText("目录")
			} else {
				name.SetText(e.Name)
				meta.SetText(fmt.Sprintf("%d bytes", e.Size))
			}
		},
	)
	rightList.OnSelected = func(id widget.ListItemID) { c.fileBrowserRightListSelected(w, id) }
	previewInfo := widget.NewLabel("")

	c.fileBrowser.mu.Lock()
	c.fileBrowser.rightTitle = rightTitle
	c.fileBrowser.rightList = rightList
	c.fileBrowser.previewInfo = previewInfo
	c.fileBrowser.upBtn = upBtn
	c.fileBrowser.downloadBtn = downloadBtn
	c.fileBrowser.uploadBtn = uploadBtn
	c.fileBrowser.mu.Unlock()

	right := container.NewBorder(
		container.NewVBox(rightTitle, actionRow),
		previewInfo,
		nil,
		nil,
		wrapScroll(rightList),
	)

	leftFooter := container.NewHBox(addBtn, layout.NewSpacer(), removeBtn, refreshBtn)
	left := container.NewBorder(leftTitle, leftFooter, nil, nil, wrapScroll(tree))
	split := container.NewHSplit(left, right)
	split.Offset = 0.33
	return split
}

func (c *Controller) fileBrowserOnSelected(uid string) {
	if c.fileBrowser == nil {
		return
	}
	kind, nodeID, dir, name, local := parseBrowseUID(uid)
	c.fileBrowser.mu.Lock()
	c.fileBrowser.currentUID = uid
	c.fileBrowser.currentNode = nodeID
	c.fileBrowser.currentDir = ""
	c.fileBrowser.rightHasSel = false
	c.fileBrowser.rightSelected = fileEntry{}
	c.fileBrowser.lastListSelID = -1
	c.fileBrowser.lastListSelAt = time.Time{}
	removeBtn := c.fileBrowser.removeBtn
	rightTitle := c.fileBrowser.rightTitle
	previewInfo := c.fileBrowser.previewInfo
	upBtn := c.fileBrowser.upBtn
	downloadBtn := c.fileBrowser.downloadBtn
	uploadBtn := c.fileBrowser.uploadBtn
	c.fileBrowser.rightItems = nil
	if !local && nodeID != 0 {
		c.fileBrowser.lastRemoteNode = nodeID
	}
	c.fileBrowser.mu.Unlock()

	if removeBtn != nil {
		if kind == browseKindNode && !local && nodeID != 0 && nodeID != c.storedNode {
			removeBtn.Enable()
		} else {
			removeBtn.Disable()
		}
	}

	if upBtn != nil {
		if kind == browseKindNode || kind == browseKindDir {
			upBtn.Enable()
		} else {
			upBtn.Disable()
		}
	}
	if downloadBtn != nil {
		downloadBtn.Disable()
	}
	if uploadBtn != nil {
		uploadBtn.Disable()
	}

	if kind == browseKindNode || kind == browseKindDir {
		c.fileBrowser.mu.Lock()
		c.fileBrowser.currentDir = dir
		c.fileBrowser.mu.Unlock()
		if rightTitle != nil {
			if kind == browseKindNode {
				rightTitle.SetText(fmt.Sprintf("节点%d / 根目录", nodeID))
			} else {
				if strings.TrimSpace(dir) == "" {
					rightTitle.SetText(fmt.Sprintf("节点%d / 根目录", nodeID))
				} else {
					rightTitle.SetText(fmt.Sprintf("节点%d / %s", nodeID, dir))
				}
			}
		}
		if previewInfo != nil {
			previewInfo.SetText("加载中...")
		}
		c.fileBrowserQueueList(nodeID, dir)
		c.fileBrowserUpdateRightListFromCache(nodeID, dir)
		c.fileBrowserUpdateDirInfoFromCache(nodeID, dir)
		return
	}

	if kind == browseKindFile {
		// 左侧树不再显示文件节点，这里仅做兼容。
		if rightTitle != nil {
			if strings.TrimSpace(dir) == "" {
				rightTitle.SetText(fmt.Sprintf("节点%d / %s", nodeID, name))
			} else {
				rightTitle.SetText(fmt.Sprintf("节点%d / %s/%s", nodeID, dir, name))
			}
		}
		if previewInfo != nil {
			previewInfo.SetText("提示：右侧文件列表双击预览")
		}
		return
	}
}

func (c *Controller) fileBrowserOnUnselected(uid string) {
	if c.fileBrowser == nil {
		return
	}
	if c.fileBrowser.removeBtn != nil {
		c.fileBrowser.removeBtn.Disable()
	}
	if c.fileBrowser.upBtn != nil {
		c.fileBrowser.upBtn.Disable()
	}
	if c.fileBrowser.downloadBtn != nil {
		c.fileBrowser.downloadBtn.Disable()
	}
	if c.fileBrowser.uploadBtn != nil {
		c.fileBrowser.uploadBtn.Disable()
	}
}

func (c *Controller) fileBrowserUpdateRightListFromCache(nodeID uint32, dir string) {
	if c.fileBrowser == nil {
		return
	}
	key := listKey(nodeID, dir)
	c.fileBrowser.mu.Lock()
	cache, ok := c.fileBrowser.listCache[key]
	rightList := c.fileBrowser.rightList
	c.fileBrowser.rightItems = nil
	if ok && cache.Code == 1 {
		items := make([]fileEntry, 0, len(cache.Dirs)+len(cache.Files))
		for _, d := range cache.Dirs {
			items = append(items, fileEntry{Name: d, IsDir: true})
		}
		for _, f := range cache.Files {
			items = append(items, fileEntry{Name: f, IsDir: false})
		}
		c.fileBrowser.rightItems = items
	}
	c.fileBrowser.rightHasSel = false
	c.fileBrowser.rightSelected = fileEntry{}
	c.fileBrowser.lastListSelID = -1
	c.fileBrowser.lastListSelAt = time.Time{}
	c.fileBrowser.mu.Unlock()
	if rightList != nil {
		runOnMain(c, rightList.Refresh)
	}
	c.fileBrowserUpdateActionButtons()
}

func (c *Controller) fileBrowserUpdateDirInfoFromCache(nodeID uint32, dir string) {
	if c.fileBrowser == nil {
		return
	}
	key := listKey(nodeID, dir)
	c.fileBrowser.mu.Lock()
	cache, ok := c.fileBrowser.listCache[key]
	info := c.fileBrowser.previewInfo
	c.fileBrowser.mu.Unlock()
	if info == nil {
		return
	}
	if !ok {
		runOnMain(c, func() { info.SetText("加载中...") })
		return
	}
	if cache.Code != 1 {
		runOnMain(c, func() { info.SetText("加载失败：" + strings.TrimSpace(cache.Msg)) })
		return
	}
	runOnMain(c, func() {
		info.SetText(fmt.Sprintf("目录：%d 个文件夹，%d 个文件", len(cache.Dirs), len(cache.Files)))
	})
}

func (c *Controller) fileBrowserUpdateActionButtons() {
	if c.fileBrowser == nil {
		return
	}
	c.fileBrowser.mu.Lock()
	nodeID := c.fileBrowser.currentNode
	hasSel := c.fileBrowser.rightHasSel
	sel := c.fileBrowser.rightSelected
	upBtn := c.fileBrowser.upBtn
	downloadBtn := c.fileBrowser.downloadBtn
	uploadBtn := c.fileBrowser.uploadBtn
	c.fileBrowser.mu.Unlock()

	if upBtn != nil {
		if nodeID != 0 {
			runOnMain(c, upBtn.Enable)
		} else {
			runOnMain(c, upBtn.Disable)
		}
	}
	if downloadBtn != nil {
		if hasSel && !sel.IsDir && nodeID != 0 && nodeID != c.storedNode {
			runOnMain(c, downloadBtn.Enable)
		} else {
			runOnMain(c, downloadBtn.Disable)
		}
	}
	if uploadBtn != nil {
		if hasSel && !sel.IsDir && nodeID != 0 && nodeID == c.storedNode {
			runOnMain(c, uploadBtn.Enable)
		} else {
			runOnMain(c, uploadBtn.Disable)
		}
	}
}

func listKey(nodeID uint32, dir string) string {
	return fmt.Sprintf("%d|%s", nodeID, strings.Trim(path.Clean(strings.ReplaceAll(strings.TrimSpace(dir), "\\", "/")), "/"))
}

func (c *Controller) fileBrowserQueueList(nodeID uint32, dir string) {
	if c.fileBrowser == nil || nodeID == 0 {
		return
	}
	dir = strings.Trim(path.Clean(strings.ReplaceAll(strings.TrimSpace(dir), "\\", "/")), "/")
	if dir == "." {
		dir = ""
	}
	key := listKey(nodeID, dir)
	c.fileBrowser.mu.Lock()
	if _, ok := c.fileBrowser.listCache[key]; ok {
		c.fileBrowser.mu.Unlock()
		return
	}
	if t, ok := c.fileBrowser.listInFlight[key]; ok && time.Since(t) < 2*time.Second {
		c.fileBrowser.mu.Unlock()
		return
	}
	c.fileBrowser.listInFlight[key] = time.Now()
	c.fileBrowser.mu.Unlock()

	go func() {
		if nodeID == c.storedNode {
			dirs, files, err := c.fileBrowserLocalList(dir)
			res := fileListResult{Code: 1, Msg: "ok", Dir: dir, Dirs: dirs, Files: files}
			if err != nil {
				res = fileListResult{Code: 404, Msg: err.Error(), Dir: dir}
			}
			c.fileBrowserOnListResult(nodeID, dir, res)
			return
		}
		if c.storedHub == 0 {
			c.fileBrowserOnListResult(nodeID, dir, fileListResult{Code: 403, Msg: "not logged in", Dir: dir})
			return
		}
		req := fileReadReq{Op: fileOpList, Target: nodeID, Dir: dir}
		_ = c.fileSendCtrl(c.storedHub, fileMessage{Action: fileActionRead, Data: fileMustJSON(req)})
	}()
}

func (c *Controller) fileBrowserOnListResult(nodeID uint32, dir string, res fileListResult) {
	if c.fileBrowser == nil {
		return
	}
	key := listKey(nodeID, dir)
	c.fileBrowser.mu.Lock()
	delete(c.fileBrowser.listInFlight, key)
	c.fileBrowser.listCache[key] = res
	tree := c.fileBrowser.tree
	curNode := c.fileBrowser.currentNode
	curDir := c.fileBrowser.currentDir
	c.fileBrowser.mu.Unlock()

	c.fileBrowserUpdateTreeChildren(nodeID, dir, res)
	c.fileBrowserUpdateRightListFromCache(nodeID, dir)
	if curNode == nodeID && listKey(curNode, curDir) == key {
		c.fileBrowserUpdateDirInfoFromCache(nodeID, dir)
	}
	if tree != nil {
		runOnMain(c, func() { tree.Refresh() })
	}
}

func (c *Controller) fileBrowserUpdateTreeChildren(nodeID uint32, dir string, res fileListResult) {
	if c.fileBrowser == nil {
		return
	}
	parentUID := fileNodeUID(nodeID, nodeID == c.storedNode)
	if strings.TrimSpace(dir) != "" {
		parentUID = fileDirUID(nodeID, dir)
	}
	children := make([]string, 0, len(res.Dirs))
	if res.Code == 1 {
		for _, d := range res.Dirs {
			children = append(children, fileDirUID(nodeID, path.Join(dir, d)))
		}
	}
	c.fileBrowser.mu.Lock()
	c.fileBrowser.treeChildren[parentUID] = children
	c.fileBrowser.mu.Unlock()
}

func (c *Controller) fileBrowserQueueReadText(nodeID uint32, dir, name string) {
	if c.fileBrowser == nil || nodeID == 0 || strings.TrimSpace(name) == "" {
		return
	}
	key := fileTextKey(nodeID, dir, name)
	c.fileBrowser.mu.Lock()
	if t, ok := c.fileBrowser.textInFlight[key]; ok && time.Since(t) < 2*time.Second {
		c.fileBrowser.mu.Unlock()
		return
	}
	c.fileBrowser.textInFlight[key] = time.Now()
	c.fileBrowser.mu.Unlock()

	go func() {
		if nodeID == c.storedNode {
			text, truncated, size, err := c.fileBrowserLocalReadText(dir, name, 64*1024)
			if err != nil {
				c.fileBrowserOnTextResult(nodeID, dir, name, fileTextResult{Code: 415, Msg: err.Error(), Dir: dir, Name: name})
				return
			}
			c.fileBrowserOnTextResult(nodeID, dir, name, fileTextResult{Code: 1, Msg: "ok", Dir: dir, Name: name, Size: size, Text: text, Truncated: truncated})
			return
		}
		if c.storedHub == 0 {
			c.fileBrowserOnTextResult(nodeID, dir, name, fileTextResult{Code: 403, Msg: "not logged in", Dir: dir, Name: name})
			return
		}
		req := fileReadReq{Op: fileOpReadText, Target: nodeID, Dir: dir, Name: name, MaxBytes: 64 * 1024}
		_ = c.fileSendCtrl(c.storedHub, fileMessage{Action: fileActionRead, Data: fileMustJSON(req)})
	}()
}

func (c *Controller) fileBrowserOnTextResult(nodeID uint32, dir, name string, res fileTextResult) {
	if c.fileBrowser == nil {
		return
	}
	key := fileTextKey(nodeID, dir, name)
	c.fileBrowser.mu.Lock()
	delete(c.fileBrowser.textInFlight, key)
	pv := c.fileBrowser.previewWindows[key]
	c.fileBrowser.mu.Unlock()

	if pv == nil || pv.text == nil || pv.info == nil {
		return
	}
	runOnMain(c, func() {
		if res.Code != 1 {
			pv.text.SetText("")
			pv.info.SetText("不支持预览：" + strings.TrimSpace(res.Msg))
			return
		}
		pv.text.SetText(res.Text)
		info := fmt.Sprintf("size=%d", res.Size)
		if res.Truncated {
			info += "（已截断）"
		}
		pv.info.SetText(info)
	})
}

func (c *Controller) fileBrowserSelectDir(nodeID uint32, dir string) {
	if c.fileBrowser == nil {
		return
	}
	uid := fileDirUID(nodeID, dir)
	c.fileBrowser.mu.Lock()
	tree := c.fileBrowser.tree
	c.fileBrowser.mu.Unlock()
	if tree != nil {
		runOnMain(c, func() {
			tree.OpenBranch(uid)
			tree.Select(uid)
		})
	}
}

func (c *Controller) fileBrowserSelectFile(nodeID uint32, dir, name string) {
	if c.fileBrowser == nil {
		return
	}
	uid := fileFileUID(nodeID, dir, name)
	c.fileBrowser.mu.Lock()
	tree := c.fileBrowser.tree
	c.fileBrowser.mu.Unlock()
	if tree != nil {
		runOnMain(c, func() { tree.Select(uid) })
	}
}

func (c *Controller) fileBrowserRightListSelected(w fyne.Window, id widget.ListItemID) {
	if c.fileBrowser == nil {
		return
	}
	now := time.Now()
	c.fileBrowser.mu.Lock()
	if id < 0 || id >= widget.ListItemID(len(c.fileBrowser.rightItems)) {
		c.fileBrowser.mu.Unlock()
		return
	}
	item := c.fileBrowser.rightItems[id]
	nodeID := c.fileBrowser.currentNode
	dir := c.fileBrowser.currentDir
	lastID := c.fileBrowser.lastListSelID
	lastAt := c.fileBrowser.lastListSelAt
	double := id == lastID && !lastAt.IsZero() && now.Sub(lastAt) <= fileBrowserDoubleTapWindow
	c.fileBrowser.lastListSelID = id
	c.fileBrowser.lastListSelAt = now
	c.fileBrowser.rightHasSel = true
	c.fileBrowser.rightSelected = item
	info := c.fileBrowser.previewInfo
	title := c.fileBrowser.rightTitle
	c.fileBrowser.mu.Unlock()

	c.fileBrowserUpdateActionButtons()
	if info != nil {
		if item.IsDir {
			runOnMain(c, func() { info.SetText("已选中目录：" + strings.TrimSpace(item.Name) + "（双击进入）") })
		} else {
			runOnMain(c, func() { info.SetText("已选中文件：" + strings.TrimSpace(item.Name) + "（双击预览）") })
		}
	}
	if title != nil && nodeID != 0 {
		runOnMain(c, func() {
			if item.IsDir {
				next := path.Join(dir, item.Name)
				if strings.TrimSpace(next) == "" {
					title.SetText(fmt.Sprintf("节点%d / 根目录", nodeID))
				} else {
					title.SetText(fmt.Sprintf("节点%d / %s", nodeID, next))
				}
				return
			}
			if strings.TrimSpace(dir) == "" {
				title.SetText(fmt.Sprintf("节点%d / %s", nodeID, item.Name))
			} else {
				title.SetText(fmt.Sprintf("节点%d / %s/%s", nodeID, dir, item.Name))
			}
		})
	}
	if !double {
		return
	}
	if item.IsDir {
		c.fileBrowserSelectDir(nodeID, path.Join(dir, item.Name))
		return
	}
	c.fileBrowserOpenPreviewWindow(w, nodeID, dir, item.Name)
}

func fileTextKey(nodeID uint32, dir, name string) string {
	dir = strings.Trim(path.Clean(strings.ReplaceAll(strings.TrimSpace(dir), "\\", "/")), "/")
	if dir == "." {
		dir = ""
	}
	name = strings.TrimSpace(name)
	return fmt.Sprintf("%d|%s|%s", nodeID, dir, name)
}

func (c *Controller) fileBrowserOpenPreviewWindow(owner fyne.Window, nodeID uint32, dir, name string) {
	if c == nil || c.app == nil || c.fileBrowser == nil || nodeID == 0 || strings.TrimSpace(name) == "" {
		return
	}
	win := c.app.NewWindow("预览: " + strings.TrimSpace(name))
	win.Resize(fyne.NewSize(860, 640))

	pathText := strings.TrimSpace(dir)
	if pathText == "" {
		pathText = "根目录"
	}
	head := widget.NewLabel(fmt.Sprintf("节点%d / %s / %s", nodeID, pathText, strings.TrimSpace(name)))
	status := widget.NewLabel("加载中...")
	status.Wrapping = fyne.TextWrapWord

	text := widget.NewMultiLineEntry()
	text.Wrapping = fyne.TextWrapWord
	text.Disable()
	text.SetText("加载中...")
	closeBtn := widget.NewButton("关闭", func() { win.Close() })

	win.SetContent(container.NewBorder(
		container.NewVBox(head, status),
		container.NewHBox(layout.NewSpacer(), closeBtn),
		nil,
		nil,
		wrapScroll(text),
	))

	key := fileTextKey(nodeID, dir, name)
	c.fileBrowser.mu.Lock()
	c.fileBrowser.previewWindows[key] = &filePreviewWidgets{win: win, text: text, info: status}
	c.fileBrowser.mu.Unlock()
	win.SetOnClosed(func() {
		if c.fileBrowser == nil {
			return
		}
		c.fileBrowser.mu.Lock()
		delete(c.fileBrowser.previewWindows, key)
		c.fileBrowser.mu.Unlock()
	})
	win.Show()
	c.fileBrowserQueueReadText(nodeID, dir, name)
}

func (c *Controller) fileBrowserGoUp() {
	if c.fileBrowser == nil {
		return
	}
	c.fileBrowser.mu.Lock()
	nodeID := c.fileBrowser.currentNode
	dir := c.fileBrowser.currentDir
	c.fileBrowser.mu.Unlock()

	if nodeID == 0 {
		return
	}
	clean := strings.Trim(path.Clean(strings.ReplaceAll(strings.TrimSpace(dir), "\\", "/")), "/")
	if clean == "." || clean == "" {
		c.fileBrowserSelectDir(nodeID, "")
		return
	}
	parent := path.Dir(clean)
	if parent == "." {
		parent = ""
	}
	c.fileBrowserSelectDir(nodeID, parent)
}

func (c *Controller) fileBrowserDownloadSelected(w fyne.Window) {
	win := resolveWindow(c.app, c.mainWin, w)
	if win == nil || c.fileBrowser == nil {
		return
	}
	c.fileBrowser.mu.Lock()
	nodeID := c.fileBrowser.currentNode
	dir := c.fileBrowser.currentDir
	hasSel := c.fileBrowser.rightHasSel
	sel := c.fileBrowser.rightSelected
	c.fileBrowser.mu.Unlock()

	if nodeID == 0 || nodeID == c.storedNode || !hasSel || sel.IsDir || strings.TrimSpace(sel.Name) == "" {
		return
	}
	name := sel.Name

	cfg := c.fileConfig()
	baseAbs, _ := filepath.Abs(cfg.BaseDir)
	if strings.TrimSpace(baseAbs) == "" {
		baseAbs = "."
	}

	saveDir := widget.NewEntry()
	saveDir.SetPlaceHolder("相对 base_dir，可为空")
	saveDir.SetText(strings.TrimSpace(dir))
	saveName := widget.NewEntry()
	saveName.SetPlaceHolder("可为空，默认使用原文件名")
	saveName.SetText(strings.TrimSpace(name))
	wantHash := widget.NewCheck("请求 sha256（可选）", nil)

	chooseBtn := widget.NewButton("选择保存目录", func() {
		showBaseDirFolderPicker(c, win, cfg.BaseDir, func(abs string) {
			baseAbs, _ := filepath.Abs(cfg.BaseDir)
			rel, err := filepath.Rel(baseAbs, abs)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
				dialog.ShowError(fmt.Errorf("请选择 base_dir(%s) 内的目录", baseAbs), win)
				return
			}
			relSlash := filepath.ToSlash(rel)
			if relSlash == "." {
				relSlash = ""
			}
			saveDir.SetText(relSlash)
		})
	})

	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("从节点 %d 下载文件", nodeID)),
		widget.NewLabel(fmt.Sprintf("远端路径: %s/%s", strings.TrimSpace(dir), strings.TrimSpace(name))),
		widget.NewSeparator(),
		widget.NewLabel("保存目录（相对 base_dir，可为空）"),
		saveDir,
		chooseBtn,
		widget.NewLabel("保存文件名（可为空，默认原文件名）"),
		saveName,
		wantHash,
	)

	dialog.ShowCustomConfirm("下载文件", "开始下载", "取消", content, func(ok bool) {
		if !ok {
			return
		}
		if err := c.fileStartPull(nodeID, dir, name, saveDir.Text, saveName.Text, wantHash.Checked); err != nil {
			dialog.ShowError(err, win)
		}
	}, win)
}

func (c *Controller) fileBrowserOfferSelected(w fyne.Window) {
	win := resolveWindow(c.app, c.mainWin, w)
	if win == nil || c.fileBrowser == nil {
		return
	}
	c.fileBrowser.mu.Lock()
	nodeID := c.fileBrowser.currentNode
	dir := c.fileBrowser.currentDir
	hasSel := c.fileBrowser.rightHasSel
	sel := c.fileBrowser.rightSelected
	lastRemote := c.fileBrowser.lastRemoteNode
	remotes := append([]uint32(nil), c.fileBrowser.nodes...)
	c.fileBrowser.mu.Unlock()

	if nodeID == 0 || nodeID != c.storedNode || !hasSel || sel.IsDir || strings.TrimSpace(sel.Name) == "" {
		return
	}
	name := sel.Name
	cfg := c.fileConfig()
	abs := filepath.Join(cfg.BaseDir, filepath.FromSlash(strings.TrimSpace(dir)), strings.TrimSpace(name))

	target := widget.NewEntry()
	target.SetPlaceHolder("目标 NodeID，例如 3")
	if lastRemote != 0 && lastRemote != c.storedNode {
		target.SetText(fmt.Sprintf("%d", lastRemote))
	} else if len(remotes) > 0 {
		sort.Slice(remotes, func(i, j int) bool { return remotes[i] < remotes[j] })
		for _, id := range remotes {
			if id != 0 && id != c.storedNode {
				target.SetText(fmt.Sprintf("%d", id))
				break
			}
		}
	}
	wantHash := widget.NewCheck("请求 sha256（可选）", nil)

	content := container.NewVBox(
		widget.NewLabel("发送本机文件（offer）"),
		widget.NewLabel(fmt.Sprintf("本机路径: %s/%s", strings.TrimSpace(dir), strings.TrimSpace(name))),
		widget.NewSeparator(),
		widget.NewLabel("目标节点 NodeID"),
		target,
		wantHash,
	)
	dialog.ShowCustomConfirm("发送文件", "发送", "取消", content, func(ok bool) {
		if !ok {
			return
		}
		n, err := strconv.ParseUint(strings.TrimSpace(target.Text), 10, 32)
		if err != nil || n == 0 {
			dialog.ShowError(fmt.Errorf("目标 NodeID 非法"), win)
			return
		}
		if err := c.fileStartOffer(uint32(n), abs, wantHash.Checked); err != nil {
			dialog.ShowError(err, win)
		}
	}, win)
}

func (c *Controller) fileBrowserAddNodeDialog(w fyne.Window) {
	win := resolveWindow(c.app, c.mainWin, w)
	if win == nil {
		return
	}
	entry := widget.NewEntry()
	entry.SetPlaceHolder("NodeID，例如 3")
	dialog.ShowForm("添加节点", "添加", "取消",
		[]*widget.FormItem{widget.NewFormItem("NodeID", entry)},
		func(ok bool) {
			if !ok {
				return
			}
			val := strings.TrimSpace(entry.Text)
			n, err := strconv.ParseUint(val, 10, 32)
			if err != nil || n == 0 {
				dialog.ShowError(fmt.Errorf("NodeID 非法"), win)
				return
			}
			c.fileBrowserAddNode(uint32(n))
		}, win)
}

func (c *Controller) fileBrowserAddNode(nodeID uint32) {
	if c.fileBrowser == nil || nodeID == 0 || nodeID == c.storedNode {
		return
	}
	c.fileBrowser.mu.Lock()
	seen := false
	for _, id := range c.fileBrowser.nodes {
		if id == nodeID {
			seen = true
			break
		}
	}
	if !seen {
		c.fileBrowser.nodes = append(c.fileBrowser.nodes, nodeID)
	}
	tree := c.fileBrowser.tree
	c.fileBrowser.mu.Unlock()
	c.saveFileBrowserPrefs()
	if tree != nil {
		runOnMain(c, tree.Refresh)
	}
}

func (c *Controller) fileBrowserRemoveSelectedNode(w fyne.Window) {
	win := resolveWindow(c.app, c.mainWin, w)
	if win == nil || c.fileBrowser == nil {
		return
	}
	c.fileBrowser.mu.Lock()
	uid := c.fileBrowser.currentUID
	c.fileBrowser.mu.Unlock()
	kind, nodeID, _, _, local := parseBrowseUID(uid)
	if kind != browseKindNode || local || nodeID == 0 || nodeID == c.storedNode {
		return
	}
	dialog.ShowConfirm("移除节点", fmt.Sprintf("确认移除 节点%d ？", nodeID), func(ok bool) {
		if !ok {
			return
		}
		c.fileBrowserRemoveNode(nodeID)
	}, win)
}

func (c *Controller) fileBrowserRemoveNode(nodeID uint32) {
	if c.fileBrowser == nil || nodeID == 0 {
		return
	}
	c.fileBrowser.mu.Lock()
	filtered := make([]uint32, 0, len(c.fileBrowser.nodes))
	for _, id := range c.fileBrowser.nodes {
		if id != nodeID {
			filtered = append(filtered, id)
		}
	}
	c.fileBrowser.nodes = filtered
	tree := c.fileBrowser.tree
	c.fileBrowser.mu.Unlock()
	c.saveFileBrowserPrefs()
	if tree != nil {
		runOnMain(c, tree.Refresh)
	}
}

func (c *Controller) fileBrowserRefreshSelected() {
	if c.fileBrowser == nil {
		return
	}
	c.fileBrowser.mu.Lock()
	nodeID := c.fileBrowser.currentNode
	dir := c.fileBrowser.currentDir
	info := c.fileBrowser.previewInfo
	c.fileBrowser.mu.Unlock()
	if nodeID == 0 {
		return
	}
	if info != nil {
		runOnMain(c, func() { info.SetText("刷新中...") })
	}
	key := listKey(nodeID, dir)
	c.fileBrowser.mu.Lock()
	delete(c.fileBrowser.listCache, key)
	delete(c.fileBrowser.treeChildren, fileDirUID(nodeID, dir))
	if strings.TrimSpace(dir) == "" {
		delete(c.fileBrowser.treeChildren, fileNodeUID(nodeID, nodeID == c.storedNode))
	}
	c.fileBrowser.mu.Unlock()
	c.fileBrowserQueueList(nodeID, dir)
}

func (c *Controller) fileBrowserHandleListResp(hdrSource uint32, resp fileReadResp) {
	dir := strings.ReplaceAll(strings.TrimSpace(resp.Dir), "\\", "/")
	c.fileBrowserOnListResult(hdrSource, dir, fileListResult{Code: resp.Code, Msg: resp.Msg, Dir: dir, Dirs: resp.Dirs, Files: resp.Files})
}

func (c *Controller) fileBrowserHandleReadTextResp(hdrSource uint32, resp fileReadResp) {
	dir := strings.ReplaceAll(strings.TrimSpace(resp.Dir), "\\", "/")
	name := strings.TrimSpace(resp.Name)
	provider := hdrSource
	if resp.Provider != 0 {
		provider = resp.Provider
	}
	c.fileBrowserOnTextResult(provider, dir, name, fileTextResult{
		Code:      resp.Code,
		Msg:       resp.Msg,
		Dir:       dir,
		Name:      name,
		Size:      resp.Size,
		Text:      resp.Text,
		Truncated: resp.Truncated,
	})
}
