package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// log entry widget
type logEntry struct{ widget.Entry }

func newLogEntry() *logEntry {
	e := &logEntry{}
	e.MultiLine = true
	e.Wrapping = fyne.TextWrapWord
	e.ExtendBaseWidget(e)
	return e
}
func (e *logEntry) TypedRune(r rune) {}
func (e *logEntry) TypedKey(event *fyne.KeyEvent) {
	switch event.Name {
	case fyne.KeyLeft, fyne.KeyRight, fyne.KeyUp, fyne.KeyDown,
		fyne.KeyHome, fyne.KeyEnd, fyne.KeyPageUp, fyne.KeyPageDown:
		e.Entry.TypedKey(event)
	}
}
func (e *logEntry) TypedShortcut(shortcut fyne.Shortcut) {
	switch shortcut.(type) {
	case *fyne.ShortcutCopy, *fyne.ShortcutSelectAll, *fyne.ShortcutPaste:
		e.Entry.TypedShortcut(shortcut)
	}
}

// log window
func (c *Controller) openLogWindow() {
	if c.app == nil {
		return
	}
	if c.logWindow != nil {
		c.logWindow.Show()
		c.logWindow.RequestFocus()
		return
	}
	c.logPopup = newLogEntry()
	c.logPopup.SetText(c.logBuf.String())
	c.logPopup.CursorRow = strings.Count(c.logBuf.String(), "\n")
	win := c.app.NewWindow("日志窗口")
	toggles := c.logToggles()
	win.SetContent(container.NewBorder(toggles, nil, nil, nil, c.logPopup))
	win.Resize(fyne.NewSize(700, 500))
	win.SetOnClosed(func() {
		c.logWindow = nil
		c.logPopup = nil
	})
	c.logWindow = win
	win.Show()
}

// log helpers
func (c *Controller) appendLog(format string, args ...any) {
	c.logMu.Lock()
	defer c.logMu.Unlock()
	text := fmt.Sprintf(format, args...)
	c.logBuf.WriteString(text + "\n")
	if c.logView != nil {
		c.logView.SetText(c.logBuf.String())
		c.logView.CursorRow = strings.Count(c.logBuf.String(), "\n")
	}
	if c.logPopup != nil {
		c.logPopup.SetText(c.logBuf.String())
		c.logPopup.CursorRow = strings.Count(c.logBuf.String(), "\n")
	}
}

// shared toggles for log preview (trunc/hex)
func (c *Controller) logToggles() fyne.CanvasObject {
	if c.truncToggle == nil {
		c.truncToggle = widget.NewCheck("短 preview", nil)
	}
	if c.showHex == nil {
		c.showHex = widget.NewCheck("预览显示 HEX", nil)
	}
	return container.NewHBox(c.truncToggle, c.showHex)
}

// log tab UI
func (c *Controller) buildLogTab(w fyne.Window) fyne.CanvasObject {
	if c.logView == nil {
		c.logView = newLogEntry()
		c.logView.Wrapping = fyne.TextWrapWord
	}
	top := container.NewBorder(nil, nil, nil, container.NewHBox(c.logToggles(), widget.NewButton("弹窗", func() { c.openLogWindow() })), widget.NewLabel("运行日志"))
	return wrapScroll(container.NewBorder(top, nil, nil, nil, c.logView))
}

func (c *Controller) formatPayloadPreview(payload []byte) string {
	if len(payload) == 0 {
		return "payload=empty"
	}
	limit := 64
	showHex := c.showHex != nil && c.showHex.Checked
	textLimit := limit
	if c.truncToggle != nil && c.truncToggle.Checked {
		textLimit = 16
	}
	preview := payload
	truncated := false
	if len(payload) > limit {
		preview = payload[:limit]
		truncated = true
	}
	textPart := buildTextPreview(preview, textLimit)
	hexPart := ""
	if showHex {
		hexPart = " hex=" + bytesToSpacedHex(preview)
	}
	suffix := ""
	if truncated {
		suffix = fmt.Sprintf("...(总长 %d bytes)", len(payload))
	}
	return fmt.Sprintf("payload=text(%s)%s%s%s", textPart, suffix, hexPart, ternarySuffix(truncated))
}

func buildTextPreview(data []byte, limit int) string {
	if utf8.Valid(data) {
		runes := []rune(string(data))
		if limit >= 0 && len(runes) > limit {
			return string(runes[:limit]) + "..."
		}
		return string(runes)
	}
	var b strings.Builder
	for i, bt := range data {
		if limit >= 0 && i >= limit {
			b.WriteString("...")
			break
		}
		if bt >= 32 && bt <= 126 {
			b.WriteByte(bt)
			continue
		}
		b.WriteByte('.')
	}
	return b.String()
}

func bytesToSpacedHex(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	var b strings.Builder
	for i, bt := range data {
		if i > 0 && i%2 == 0 {
			b.WriteByte(' ')
		}
		b.WriteString(strings.ToUpper(fmt.Sprintf("%02x", bt)))
	}
	return b.String()
}

func ternarySuffix(cond bool) string {
	if cond {
		return " (已截断)"
	}
	return ""
}
