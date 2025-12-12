package ui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	core "github.com/yttydcs/myflowhub-core"
	"github.com/yttydcs/myflowhub-core/header"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) buildDebugTab(w fyne.Window) fyne.CanvasObject {
	c.addrEntry = widget.NewEntry()
	c.addrEntry.SetPlaceHolder("127.0.0.1:9000")

	c.nodeEntry = widget.NewEntry()
	c.nodeEntry.SetPlaceHolder("debugclient")

	connectBtn := widget.NewButton("连接", func() {
		addr := valueOrPlaceholder(c.addrEntry)
		go func() {
			if err := c.session.Connect(addr); err != nil {
				c.appendLog("[ERR] connect: %v", err)
				return
			}
			nodeName := valueOrPlaceholder(c.nodeEntry)
			if nodeName == "" {
				nodeName = "debugclient"
			}
			if err := c.session.Login(nodeName); err != nil {
				c.appendLog("[ERR] login: %v", err)
			} else {
				c.appendLog("[OK] connected %s", addr)
			}
		}()
	})

	disconnectBtn := widget.NewButton("断开", func() {
		c.session.Close()
		c.appendLog("[INFO] 手动断开")
	})

	headerCard := c.buildHeaderCard()
	payloadCard := c.buildPayloadCard()

	sendBtn := widget.NewButton("发送", func() {
		hdr, err := c.form.Parse()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		payload, perr := c.buildPayload()
		if perr != nil {
			dialog.ShowError(perr, w)
			return
		}
		if err := c.session.Send(hdr, payload); err != nil {
			c.appendLog("[ERR] send: %v", err)
			return
		}
		c.logTx("TX", hdr, payload)
	})

	topBar := container.NewBorder(nil, nil,
		widget.NewLabel("Server"),
		container.NewHBox(connectBtn, disconnectBtn),
		container.NewMax(c.addrEntry))

	content := container.NewVBox(headerCard, payloadCard)
	btns := container.NewHBox(sendBtn)
	return wrapScroll(container.NewBorder(topBar, btns, nil, nil, content))
}

func (c *Controller) buildConfigTab(w fyne.Window) fyne.CanvasObject {
	c.configInfo = widget.NewLabel("展示当前生效配置（默认值+已保存覆盖）")
	c.configList = container.NewVBox()
	c.reloadConfigUI(true)

	addBtn := widget.NewButton("新增空行", func() {
		c.addConfigRow("", "")
	})
	saveBtn := widget.NewButton("保存", func() {
		if err := c.saveConfigFromUI(); err != nil {
			dialog.ShowError(err, w)
			return
		}
		c.configInfo.SetText("已保存到本地（Preferences）")
	})
	reloadBtn := widget.NewButton("重新加载", func() {
		c.reloadConfigUI(false)
		c.configInfo.SetText("已从本地加载（覆盖默认）")
	})
	resetBtn := widget.NewButton("重置默认", func() {
		c.resetConfigToDefault()
		c.configInfo.SetText("已恢复默认值（未保存）")
	})

	toolbar := container.NewHBox(addBtn, saveBtn, reloadBtn, resetBtn)
	scroll := wrapScroll(c.configList)
	return wrapScroll(container.NewBorder(c.configInfo, toolbar, nil, nil, scroll))
}

func (c *Controller) buildPresetTab(w fyne.Window) fyne.CanvasObject {
	defs := c.presetDefinitions()
	list := container.NewVBox()
	for _, def := range defs {
		list.Add(c.buildPresetCard(def, w))
	}
	return wrapScroll(list)
}

func (c *Controller) buildPresetCard(def presetDefinition, w fyne.Window) fyne.CanvasObject {
	p1 := widget.NewEntry()
	p1.SetPlaceHolder(def.param1Ph)
	p2 := widget.NewEntry()
	p2.SetPlaceHolder(def.param2Ph)
	sendBtn := widget.NewButton("发送", func() {
		if err := c.sendPreset(def, valueOrPlaceholder(p1), valueOrPlaceholder(p2)); err != nil {
			dialog.ShowError(err, w)
			return
		}
	})
	body := container.NewVBox(
		widget.NewLabel(def.param1Label),
		p1,
		widget.NewLabel(def.param2Label),
		p2,
		sendBtn,
	)
	return widget.NewCard(def.name, def.desc, body)
}

func (c *Controller) sendPreset(def presetDefinition, p1, p2 string) error {
	hdr, payload, err := def.build(p1, p2)
	if err != nil {
		return err
	}
	if strings.EqualFold(def.name, "VarSet") {
		c.addVarPoolKey(varKey{Name: strings.TrimSpace(p1), Owner: c.storedNode})
	}
	hdr, err = c.applyPresetHeaderOverrides(hdr)
	if err != nil {
		return err
	}
	if err := c.session.Send(hdr, payload); err != nil {
		return err
	}
	c.logTx("[TX preset] "+def.name, hdr, payload)
	return nil
}

func (c *Controller) applyPresetHeaderOverrides(h core.IHeader) (core.IHeader, error) {
	if h == nil {
		return h, nil
	}
	if c.presetSrc != nil {
		if s := strings.TrimSpace(valueOrPlaceholder(c.presetSrc)); s != "" {
			v, err := parseOptionalUint32(s, "SourceID")
			if err != nil {
				return nil, err
			}
			h = h.WithSourceID(v)
		}
	}
	if c.presetTgt != nil {
		if s := strings.TrimSpace(valueOrPlaceholder(c.presetTgt)); s != "" {
			v, err := parseOptionalUint32(s, "TargetID")
			if err != nil {
				return nil, err
			}
			h = h.WithTargetID(v)
		}
	}
	return h, nil
}

// preset definitions
type presetDefinition struct {
	name        string
	desc        string
	param1Label string
	param1Ph    string
	param2Label string
	param2Ph    string
	build       func(p1, p2 string) (core.IHeader, []byte, error)
}

func (c *Controller) presetDefinitions() []presetDefinition {
	return []presetDefinition{
		{
			name:        "Echo",
			desc:        "SubProto=1，回显字符串（TargetID 可选）",
			param1Label: "参数1（消息文本）",
			param1Ph:    "hello",
			param2Label: "参数2（TargetID，可选）",
			param2Ph:    "0",
			build: func(p1, p2 string) (core.IHeader, []byte, error) {
				if strings.TrimSpace(p1) == "" {
					return nil, nil, fmt.Errorf("消息不能为空")
				}
				target, err := parseOptionalUint32(p2, "TargetID")
				if err != nil {
					return nil, nil, err
				}
				hdr := presetHeader(1, target)
				return hdr, []byte(p1), nil
			},
		},
		{
			name:        "Login",
			desc:        "SubProto=2，action=login，payload JSON",
			param1Label: "参数1（DeviceID）",
			param1Ph:    "device-001",
			param2Label: "参数2（NodeID，可选，默认使用已登录的 NodeID）",
			param2Ph:    "",
			build: func(p1, p2 string) (core.IHeader, []byte, error) {
				if strings.TrimSpace(p1) == "" {
					return nil, nil, fmt.Errorf("DeviceID 不能为空")
				}
				nodeID := c.storedNode
				if strings.TrimSpace(p2) != "" {
					val, err := strconv.ParseUint(strings.TrimSpace(p2), 10, 32)
					if err != nil {
						return nil, nil, fmt.Errorf("NodeID 无法解析: %w", err)
					}
					nodeID = uint32(val)
				}
				if err := c.ensureNodeKeys(); err != nil {
					return nil, nil, fmt.Errorf("加载本地密钥失败: %w", err)
				}
				ts := time.Now().Unix()
				nonce := generateNonce(12)
				sig, err := signLogin(c.nodePriv, p1, nodeID, ts, nonce)
				if err != nil {
					return nil, nil, fmt.Errorf("签名失败: %w", err)
				}
				payload, _ := json.Marshal(map[string]any{
					"action": "login",
					"data": map[string]any{
						"device_id": p1,
						"node_id":   nodeID,
						"ts":        ts,
						"nonce":     nonce,
						"sig":       sig,
						"alg":       "ES256",
					},
				})
				hdr := presetHeader(2, 0)
				return hdr, payload, nil
			},
		},
		{
			name:        "Register",
			desc:        "SubProto=2，action=register，payload JSON",
			param1Label: "参数1（DeviceID）",
			param1Ph:    "device-001",
			param2Label: "参数2（无）",
			param2Ph:    "",
			build: func(p1, _ string) (core.IHeader, []byte, error) {
				if strings.TrimSpace(p1) == "" {
					return nil, nil, fmt.Errorf("DeviceID 不能为空")
				}
				if err := c.ensureNodeKeys(); err != nil {
					return nil, nil, fmt.Errorf("加载本地密钥失败: %w", err)
				}
				payload, _ := json.Marshal(map[string]any{
					"action": "register",
					"data": map[string]any{
						"device_id": p1,
						"pubkey":    c.nodePubB64,
						"node_pub":  c.nodePubB64,
					},
				})
				hdr := presetHeader(2, 0)
				return hdr, payload, nil
			},
		},
		{
			name:        "VarSet",
			desc:        "SubProto=3，action=set，默认 public string",
			param1Label: "参数1（变量名）",
			param1Ph:    "foo",
			param2Label: "参数2（变量值）",
			param2Ph:    "bar",
			build: func(p1, p2 string) (core.IHeader, []byte, error) {
				if strings.TrimSpace(p1) == "" || strings.TrimSpace(p2) == "" {
					return nil, nil, fmt.Errorf("变量名/变量值不能为空")
				}
				payload, _ := json.Marshal(map[string]any{
					"action": "set",
					"data": map[string]any{
						"name":       p1,
						"value":      p2,
						"visibility": "public",
						"type":       "string",
					},
				})
				hdr := presetHeader(3, 0)
				return hdr, payload, nil
			},
		},
		{
			name:        "VarGet",
			desc:        "SubProto=3，action=get，获取变量值",
			param1Label: "参数1（变量名）",
			param1Ph:    "foo",
			param2Label: "参数2（预留，可空）",
			param2Ph:    "",
			build: func(p1, _ string) (core.IHeader, []byte, error) {
				if strings.TrimSpace(p1) == "" {
					return nil, nil, fmt.Errorf("变量名不能为空")
				}
				payload, _ := json.Marshal(map[string]any{
					"action": "get",
					"data": map[string]any{
						"name": p1,
					},
				})
				hdr := presetHeader(3, 0)
				return hdr, payload, nil
			},
		},
	}
}

func presetHeader(sub uint8, target uint32) *header.HeaderTcp {
	h := &header.HeaderTcp{}
	h.WithMajor(header.MajorCmd).
		WithSubProto(sub).
		WithSourceID(1).
		WithTargetID(target).
		WithMsgID(uint32(time.Now().UnixNano()))
	return h
}
