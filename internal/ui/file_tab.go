package ui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) buildFileTab(w fyne.Window) fyne.CanvasObject {
	tabs := container.NewAppTabs(
		container.NewTabItem("文件浏览", c.buildFileBrowseTab(w)),
		container.NewTabItem("任务列表", c.buildFileTasksTab(w)),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}

func (c *Controller) buildFileTasksTab(w fyne.Window) fyne.CanvasObject {
	baseDirLabel := widget.NewLabel("")
	refreshBaseDir := func() {
		cfg := c.fileConfig()
		baseDirLabel.SetText("base_dir: " + strings.TrimSpace(cfg.BaseDir))
	}
	refreshBaseDir()

	taskList := widget.NewList(
		func() int {
			if c == nil || c.file == nil {
				return 0
			}
			c.file.mu.Lock()
			defer c.file.mu.Unlock()
			return len(c.file.tasks)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabel("")
			title.Wrapping = fyne.TextTruncate
			status := widget.NewLabel("")
			top := container.NewHBox(title, layout.NewSpacer(), status)
			bar := widget.NewProgressBar()
			detail := widget.NewLabel("")
			detail.Wrapping = fyne.TextTruncate
			retryBtn := widget.NewButton("重试", nil)
			cancelBtn := widget.NewButton("取消", nil)
			openBtn := widget.NewButton("打开目录", nil)
			btns := container.NewHBox(retryBtn, cancelBtn, layout.NewSpacer(), openBtn)
			return container.NewVBox(top, bar, detail, btns, widget.NewSeparator())
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if c == nil || c.file == nil {
				return
			}
			c.file.mu.Lock()
			if i < 0 || i >= len(c.file.tasks) {
				c.file.mu.Unlock()
				return
			}
			task := c.file.tasks[i]
			c.file.mu.Unlock()
			if task == nil {
				return
			}
			task.mu.Lock()
			op := task.op
			direction := task.direction
			provider := task.provider
			consumer := task.consumer
			peer := task.peer
			dir := task.dir
			name := task.name
			size := task.size
			sha := task.sha256
			wantHash := task.wantHash
			localDir := task.localDir
			localName := task.localName
			localPath := task.localPath
			filePath := task.filePath
			statusText := task.status
			errText := task.lastError
			doneBytes := task.doneBytes
			ackedBytes := task.ackedBytes
			task.mu.Unlock()

			cont, _ := obj.(*fyne.Container)
			if cont == nil || len(cont.Objects) < 4 {
				return
			}
			top := cont.Objects[0].(*fyne.Container)
			titleLabel := top.Objects[0].(*widget.Label)
			statusLabel := top.Objects[2].(*widget.Label)
			bar := cont.Objects[1].(*widget.ProgressBar)
			detailLabel := cont.Objects[2].(*widget.Label)
			btns := cont.Objects[3].(*fyne.Container)
			retryBtn := btns.Objects[0].(*widget.Button)
			cancelBtn := btns.Objects[1].(*widget.Button)
			openBtn := btns.Objects[3].(*widget.Button)

			titleLabel.SetText(fmt.Sprintf("[%s/%s] %s/%s  peer=%d", op, direction, strings.TrimSpace(dir), strings.TrimSpace(name), peer))
			statusLabel.SetText(statusText)

			var progress float64
			var progText string
			if size == 0 {
				if statusText == "完成" {
					progress = 1
				} else {
					progress = 0
				}
				progText = "0 bytes"
			} else if direction == "download" {
				progress = float64(doneBytes) / float64(size)
				progText = fmt.Sprintf("%d / %d", doneBytes, size)
			} else {
				progress = float64(ackedBytes) / float64(size)
				progText = fmt.Sprintf("acked %d / %d", ackedBytes, size)
			}
			if progress < 0 {
				progress = 0
			}
			if progress > 1 {
				progress = 1
			}
			bar.SetValue(progress)

			detail := fmt.Sprintf("provider=%d consumer=%d  %s", provider, consumer, progText)
			if strings.TrimSpace(localDir) != "" {
				detail += "  saveDir=" + strings.TrimSpace(localDir)
			}
			if strings.TrimSpace(localPath) != "" {
				detail += "  path=" + strings.TrimSpace(localPath)
			}
			if wantHash && strings.TrimSpace(sha) != "" {
				detail += "  sha256=" + strings.TrimSpace(sha)
			}
			if statusText == "失败" && strings.TrimSpace(errText) != "" {
				detail += "  err=" + strings.TrimSpace(errText)
			}
			detailLabel.SetText(detail)

			retryBtn.Disable()
			cancelBtn.Disable()
			openBtn.Disable()

			if statusText == "失败" {
				retryBtn.Enable()
				retryBtn.OnTapped = func() {
					win := resolveWindow(c.app, c.mainWin, w)
					if direction == "download" && op == fileOpPull {
						save := localDir
						if strings.TrimSpace(save) == "" {
							save = dir
						}
						if err := c.fileStartPull(provider, dir, name, save, localName, wantHash); err != nil {
							dialog.ShowError(err, win)
						}
						return
					}
					if direction == "upload" && op == fileOpOffer {
						if strings.TrimSpace(filePath) == "" {
							dialog.ShowError(fmt.Errorf("没有可用的本地文件路径"), win)
							return
						}
						if err := c.fileStartOffer(consumer, filePath, wantHash); err != nil {
							dialog.ShowError(err, win)
						}
					}
				}
			}

			if statusText == "发送中" || statusText == "接收中" || statusText == "等待确认" || statusText == "等待响应" || statusText == "等待对方确认" || statusText == "准备中" || statusText == "计算SHA256" {
				cancelBtn.Enable()
				cancelBtn.OnTapped = func() {
					task.mu.Lock()
					sid := task.sessionID
					hasSid := task.hasSessionID
					task.status = "取消"
					task.updatedAt = time.Now()
					task.mu.Unlock()
					if hasSid {
						c.fileRemoveRecvSession(sid)
						c.fileRemoveSendSession(sid)
						c.fileRefreshUIThrottled()
						return
					}
					c.file.mu.Lock()
					delete(c.file.pendingPull, filePullKey{provider: provider, dir: strings.TrimSpace(dir), name: strings.TrimSpace(name)})
					c.file.mu.Unlock()
					c.fileRefreshUIThrottled()
				}
			}

			if strings.TrimSpace(localPath) != "" || strings.TrimSpace(localDir) != "" {
				openBtn.Enable()
				openBtn.OnTapped = func() {
					cfg := c.fileConfig()
					baseAbs, _ := filepath.Abs(cfg.BaseDir)
					folder := baseAbs
					if strings.TrimSpace(localPath) != "" {
						folder = filepath.Dir(localPath)
					} else if strings.TrimSpace(localDir) != "" {
						folder = filepath.Join(baseAbs, filepath.FromSlash(strings.TrimSpace(localDir)))
					}
					openFolder(folder)
				}
			}
		},
	)

	if c.file != nil {
		c.file.mu.Lock()
		c.file.list = taskList
		c.file.mu.Unlock()
	}

	refreshBtn := widget.NewButton("刷新 base_dir", func() { refreshBaseDir() })
	infoRow := container.NewHBox(baseDirLabel, layout.NewSpacer(), refreshBtn)
	return wrapScroll(container.NewVBox(infoRow, widget.NewCard("任务", "传输任务列表（进度/状态/重试）", taskList)))
}

func openFolder(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("explorer", path).Start()
		return
	}
	_ = exec.Command("xdg-open", path).Start()
}
