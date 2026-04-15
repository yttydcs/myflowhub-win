// 本文件负责持久化 Home 页面最近使用的连接、设备和认证快照。

package main

import (
	"errors"
	"strings"
)

const (
	homeDeviceIDKey    = "home.device_id"
	homeNodeIDKey      = "home.node_id"
	homeHubIDKey       = "home.hub_id"
	homeRoleKey        = "home.role"
	homeAutoConnectKey = "home.auto_connect"
	homeAutoLoginKey   = "home.auto_login"
)

type HomeState struct {
	DeviceID    string `json:"deviceId"`
	AutoConnect bool   `json:"autoConnect"`
	AutoLogin   bool   `json:"autoLogin"`
	NodeID      uint32 `json:"nodeId"`
	HubID       uint32 `json:"hubId"`
	Role        string `json:"role"`
}

func (a *App) HomeState() (HomeState, error) {
	// HomeState 读取当前 profile 下 Home 页保存的连接与认证快照。
	if a.store == nil {
		return HomeState{}, errors.New("storage not initialized")
	}
	profile := a.store.CurrentProfile()
	nodeID := a.store.GetInt(profile, homeNodeIDKey, 0)
	hubID := a.store.GetInt(profile, homeHubIDKey, 0)
	if nodeID < 0 {
		nodeID = 0
	}
	if hubID < 0 {
		hubID = 0
	}
	return HomeState{
		DeviceID:    a.store.GetString(profile, homeDeviceIDKey, ""),
		AutoConnect: a.store.GetBool(profile, homeAutoConnectKey, false),
		AutoLogin:   a.store.GetBool(profile, homeAutoLoginKey, false),
		NodeID:      uint32(nodeID),
		HubID:       uint32(hubID),
		Role:        a.store.GetString(profile, homeRoleKey, ""),
	}, nil
}

func (a *App) SaveHomeState(state HomeState) (HomeState, error) {
	// SaveHomeState 在入库前裁剪与校验字段，确保首页恢复的是干净状态。
	if a.store == nil {
		return HomeState{}, errors.New("storage not initialized")
	}
	state.DeviceID = strings.TrimSpace(state.DeviceID)
	state.Role = strings.TrimSpace(state.Role)
	if err := validateHomeState(state); err != nil {
		return HomeState{}, err
	}
	profile := a.store.CurrentProfile()
	if err := a.store.SetString(profile, homeDeviceIDKey, state.DeviceID); err != nil {
		return HomeState{}, err
	}
	if err := a.store.SetBool(profile, homeAutoConnectKey, state.AutoConnect); err != nil {
		return HomeState{}, err
	}
	if err := a.store.SetBool(profile, homeAutoLoginKey, state.AutoLogin); err != nil {
		return HomeState{}, err
	}
	if err := a.store.SetInt(profile, homeNodeIDKey, int(state.NodeID)); err != nil {
		return HomeState{}, err
	}
	if err := a.store.SetInt(profile, homeHubIDKey, int(state.HubID)); err != nil {
		return HomeState{}, err
	}
	if err := a.store.SetString(profile, homeRoleKey, state.Role); err != nil {
		return HomeState{}, err
	}
	return a.HomeState()
}

func (a *App) ClearHomeAuth() (HomeState, error) {
	// ClearHomeAuth 只清掉首页缓存的认证结果，不重置设备与自动连接偏好。
	state, err := a.HomeState()
	if err != nil {
		return HomeState{}, err
	}
	state.NodeID = 0
	state.HubID = 0
	state.Role = ""
	return a.SaveHomeState(state)
}

func validateHomeState(state HomeState) error {
	// validateHomeState 约束首页持久化字段长度，避免把异常输入长期写入 profile。
	if len(state.DeviceID) > 128 {
		return errors.New("device_id is too long")
	}
	if len(state.Role) > 64 {
		return errors.New("role is too long")
	}
	return nil
}
