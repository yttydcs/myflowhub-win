package ui

import (
	"fmt"

	core "github.com/yttydcs/myflowhub-core"
)

const (
	actionLoginResp    = "login_resp"
	actionRegisterResp = "register_resp"
)

// logTx 打印发送日志。
func (c *Controller) logTx(prefix string, hdr core.IHeader, payload []byte) {
	if c == nil || c.isLogPaused() || hdr == nil {
		return
	}
	line := fmt.Sprintf("%s major=%d sub=%d src=%d tgt=%d len=%d", prefix, hdr.Major(), hdr.SubProto(), hdr.SourceID(), hdr.TargetID(), len(payload))
	if c.showHex != nil && c.showHex.Checked {
		line = line + " hex=" + bytesToSpacedHex(payload)
	}
	c.appendLog(line)
}
