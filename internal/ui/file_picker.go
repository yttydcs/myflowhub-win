package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type baseDirEntry struct {
	Name  string
	IsDir bool
	Size  int64
}

func showBaseDirFilePicker(c *Controller, owner fyne.Window, baseDir string, onPick func(absPath string)) {
	showBaseDirPicker(c, owner, baseDir, true, onPick)
}

func showBaseDirFolderPicker(c *Controller, owner fyne.Window, baseDir string, onPick func(absPath string)) {
	showBaseDirPicker(c, owner, baseDir, false, onPick)
}

func showBaseDirPicker(c *Controller, owner fyne.Window, baseDir string, pickFile bool, onPick func(absPath string)) {
	win := resolveWindow(c.app, c.mainWin, owner)
	if win == nil {
		return
	}
	baseAbs, err := filepath.Abs(strings.TrimSpace(baseDir))
	if err != nil || strings.TrimSpace(baseAbs) == "" {
		dialog.ShowError(fmt.Errorf("base_dir 无效"), win)
		return
	}
	_ = os.MkdirAll(baseAbs, 0o755)

	picker := c.app.NewWindow("选择" + map[bool]string{true: "文件", false: "目录"}[pickFile])
	picker.Resize(fyne.NewSize(720, 520))

	pathLabel := widget.NewLabel("")
	pathLabel.Wrapping = fyne.TextTruncate
	errLabel := widget.NewLabel("")

	var currentAbs string
	var entries []baseDirEntry

	list := widget.NewList(
		func() int { return len(entries) },
		func() fyne.CanvasObject {
			name := widget.NewLabel("")
			name.Wrapping = fyne.TextTruncate
			meta := widget.NewLabel("")
			meta.Wrapping = fyne.TextTruncate
			return container.NewBorder(nil, nil, nil, meta, name)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(entries) {
				return
			}
			row := obj.(*fyne.Container)
			name := row.Objects[0].(*widget.Label)
			meta := row.Objects[1].(*widget.Label)
			e := entries[i]
			if e.IsDir {
				name.SetText("[DIR] " + e.Name)
				meta.SetText("目录")
				return
			}
			name.SetText(e.Name)
			meta.SetText(fmt.Sprintf("%d bytes", e.Size))
		},
	)

	load := func(abs string) {
		abs = strings.TrimSpace(abs)
		if abs == "" {
			return
		}
		abs, err := filepath.Abs(abs)
		if err != nil {
			return
		}
		rel, err := filepath.Rel(baseAbs, abs)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return
		}
		currentAbs = abs
		pathLabel.SetText(fmt.Sprintf("base_dir: %s   当前: %s", baseAbs, abs))
		errLabel.SetText("")
		go func() {
			ents, err := os.ReadDir(abs)
			if err != nil {
				runOnMain(c, func() { errLabel.SetText("读取失败: " + err.Error()) })
				return
			}
			out := make([]baseDirEntry, 0, len(ents))
			for _, it := range ents {
				if it == nil {
					continue
				}
				if it.IsDir() {
					out = append(out, baseDirEntry{Name: it.Name(), IsDir: true})
					continue
				}
				info, _ := it.Info()
				size := int64(0)
				if info != nil {
					size = info.Size()
				}
				out = append(out, baseDirEntry{Name: it.Name(), IsDir: false, Size: size})
			}
			sort.Slice(out, func(i, j int) bool {
				if out[i].IsDir != out[j].IsDir {
					return out[i].IsDir
				}
				return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
			})
			runOnMain(c, func() {
				entries = out
				list.Refresh()
			})
		}()
	}

	upBtn := widget.NewButton("上级", func() {
		if strings.TrimSpace(currentAbs) == "" {
			return
		}
		parent := filepath.Dir(currentAbs)
		if parent == currentAbs {
			return
		}
		load(parent)
	})
	homeBtn := widget.NewButton("回到 base_dir", func() { load(baseAbs) })

	confirmBtn := widget.NewButton("选择当前目录", func() {
		if pickFile {
			return
		}
		if onPick != nil && strings.TrimSpace(currentAbs) != "" {
			onPick(currentAbs)
		}
		picker.Close()
	})
	if pickFile {
		confirmBtn.Hide()
	}

	list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(entries) {
			return
		}
		e := entries[id]
		abs := filepath.Join(currentAbs, e.Name)
		if e.IsDir {
			load(abs)
			return
		}
		if !pickFile {
			return
		}
		if onPick != nil {
			onPick(abs)
		}
		picker.Close()
	}

	top := container.NewBorder(nil, nil,
		container.NewHBox(upBtn, homeBtn),
		container.NewHBox(confirmBtn, layout.NewSpacer()),
		pathLabel,
	)
	body := container.NewBorder(top, errLabel, nil, nil, list)
	picker.SetContent(body)
	load(baseAbs)
	picker.Show()
}
