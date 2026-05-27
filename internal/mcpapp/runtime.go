// 本文件组装无界面的 MCP 运行时，把 session、auth、management、flow、topicbus 和 varstore 服务接到一起。

package mcpapp

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	corebus "github.com/yttydcs/myflowhub-core/eventbus"
	coreperm "github.com/yttydcs/myflowhub-core/kit/permission"
	protoauth "github.com/yttydcs/myflowhub-proto/protocol/auth"
	protoexec "github.com/yttydcs/myflowhub-proto/protocol/exec"
	protoflow "github.com/yttydcs/myflowhub-proto/protocol/flow"
	protomanagement "github.com/yttydcs/myflowhub-proto/protocol/management"
	protovarstore "github.com/yttydcs/myflowhub-proto/protocol/varstore"
	authsvc "github.com/yttydcs/myflowhub-win/internal/services/auth"
	flowsvc "github.com/yttydcs/myflowhub-win/internal/services/flow"
	logssvc "github.com/yttydcs/myflowhub-win/internal/services/logs"
	mgmtsvc "github.com/yttydcs/myflowhub-win/internal/services/management"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
	topicbussvc "github.com/yttydcs/myflowhub-win/internal/services/topicbus"
	varpoolsvc "github.com/yttydcs/myflowhub-win/internal/services/varpool"
	storagesvc "github.com/yttydcs/myflowhub-win/internal/storage"
)

const (
	defaultConfigDirName = "myflowhub"
	defaultConfigProfile = "mcp-client"

	mcpEndpointKey      = "mcp.endpoint"
	mcpDeviceIDKey      = "mcp.device_id"
	mcpDisplayNameKey   = "mcp.display_name"
	mcpNodeIDKey        = "mcp.node_id"
	mcpHubIDKey         = "mcp.hub_id"
	mcpRoleKey          = "mcp.role"
	mcpDefaultTargetKey = "mcp.default_target"
	mcpAllowWriteKey    = "mcp.allow_write"

	defaultRequestTimeout = 8 * time.Second
)

type Config struct {
	Context       context.Context
	ConfigDir     string
	Endpoint      string
	DeviceID      string
	DisplayName   string
	DefaultTarget uint32
	AllowWrite    bool
	Timeout       time.Duration
	LogWriter     io.Writer
}

type Defaults struct {
	Endpoint      string `json:"endpoint,omitempty"`
	DeviceID      string `json:"device_id,omitempty"`
	DisplayName   string `json:"display_name,omitempty"`
	NodeID        uint32 `json:"node_id,omitempty"`
	HubID         uint32 `json:"hub_id,omitempty"`
	Role          string `json:"role,omitempty"`
	DefaultTarget uint32 `json:"default_target,omitempty"`
	AllowWrite    bool   `json:"allow_write"`
}

type AuthSnapshot struct {
	DeviceID string `json:"device_id,omitempty"`
	NodeID   uint32 `json:"node_id,omitempty"`
	HubID    uint32 `json:"hub_id,omitempty"`
	Role     string `json:"role,omitempty"`
	LoggedIn bool   `json:"logged_in"`
}

type ConfigState struct {
	BaseDir       string `json:"base_dir"`
	SettingsPath  string `json:"settings_path"`
	KeysPath      string `json:"keys_path"`
	AllowWrite    bool   `json:"allow_write"`
	TimeoutMillis int64  `json:"timeout_millis"`
}

type Status struct {
	Connected bool         `json:"connected"`
	Endpoint  string       `json:"endpoint,omitempty"`
	Auth      AuthSnapshot `json:"auth"`
	Defaults  Defaults     `json:"defaults"`
	Config    ConfigState  `json:"config"`
}

type Runtime struct {
	ctx    context.Context
	cancel context.CancelFunc

	bus        corebus.IBus
	logs       *logssvc.LogService
	session    *sessionsvc.SessionService
	auth       *authsvc.AuthService
	flow       *flowsvc.FlowService
	management *mgmtsvc.ManagementService
	topicbus   *topicbussvc.TopicBusService
	varpool    *varpoolsvc.VarPoolService
	store      *storagesvc.Store

	timeout time.Duration

	mu       sync.RWMutex
	defaults Defaults
	authSnap AuthSnapshot

	logWriter io.Writer
	logMu     sync.Mutex
	logToken  string
}

func New(config Config) (*Runtime, error) {
	// New 构建独立于 GUI 的 MCP runtime，并把默认值、日志桥接和持久化一起接好。
	ctx := config.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)

	baseDir, err := resolveConfigDir(config.ConfigDir)
	if err != nil {
		cancel()
		return nil, err
	}

	store, err := storagesvc.NewStoreWithBaseDir(baseDir)
	if err != nil {
		cancel()
		return nil, err
	}
	if err := store.MigrateLegacyNodeKeysForProfiles(); err != nil {
		cancel()
		return nil, err
	}

	bus := corebus.New(corebus.Options{})
	logs := logssvc.New(bus, 2000)
	session := sessionsvc.New(ctx, bus, logs)
	auth := authsvc.New(session, logs, store)
	flow := flowsvc.New(session, logs)
	management := mgmtsvc.New(session, logs, store)
	topicbus := topicbussvc.New(session, logs, bus)
	varpool := varpoolsvc.New(session, logs, bus)

	currentProfile := store.CurrentProfile()
	auth.SetKeysPath(store.NodeKeysPath(currentProfile))

	rt := &Runtime{
		ctx:        ctx,
		cancel:     cancel,
		bus:        bus,
		logs:       logs,
		session:    session,
		auth:       auth,
		flow:       flow,
		management: management,
		topicbus:   topicbus,
		varpool:    varpool,
		store:      store,
		timeout:    normalizeTimeout(config.Timeout),
		logWriter:  config.LogWriter,
	}

	rt.loadDefaultsFromStore()
	if err := rt.applyConfig(config); err != nil {
		rt.Close()
		return nil, err
	}
	rt.attachLogWriter()
	return rt, nil
}

func (r *Runtime) Close() {
	// Close 释放 runtime 独占的 topicbus/varpool/session/bus 资源，避免 CLI 退出后残留订阅。
	if r == nil {
		return
	}
	if r.topicbus != nil {
		r.topicbus.Close()
	}
	if r.varpool != nil {
		r.varpool.Close()
	}
	if r.session != nil {
		r.session.Close()
	}
	if r.bus != nil && r.logToken != "" {
		r.bus.Unsubscribe(logssvc.EventLogLine, r.logToken)
		r.logToken = ""
	}
	if r.bus != nil {
		r.bus.Close()
	}
	if r.cancel != nil {
		r.cancel()
	}
}

func (r *Runtime) Defaults() Defaults {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaults
}

func (r *Runtime) AuthSnapshot() AuthSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.authSnap
}

func (r *Runtime) AllowWrite() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaults.AllowWrite
}

func (r *Runtime) Timeout() time.Duration {
	return r.timeout
}

func (r *Runtime) SessionConnected() bool {
	if r == nil || r.session == nil {
		return false
	}
	return r.session.IsConnected()
}

func (r *Runtime) Status() Status {
	// Status 汇总“当前链路 + 持久化默认值 + 配置目录”三类信息，供 MCP status 直接输出。
	defaults := r.Defaults()
	authSnap := r.AuthSnapshot()
	endpoint := defaults.Endpoint
	if lastAddr := strings.TrimSpace(r.session.LastAddr()); lastAddr != "" {
		endpoint = lastAddr
	}
	profileState := r.store.State()
	return Status{
		Connected: r.SessionConnected(),
		Endpoint:  endpoint,
		Auth:      authSnap,
		Defaults:  defaults,
		Config: ConfigState{
			BaseDir:       profileState.BaseDir,
			SettingsPath:  profileState.SettingsPath,
			KeysPath:      profileState.KeysPath,
			AllowWrite:    defaults.AllowWrite,
			TimeoutMillis: r.timeout.Milliseconds(),
		},
	}
}

func (r *Runtime) Connect(endpoint string) error {
	// Connect 建链成功后会清掉 logged_in 标记，避免把上一条链路的身份错误复用到新连接。
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = r.Defaults().Endpoint
	}
	if endpoint == "" {
		return errors.New("endpoint is required")
	}
	if err := r.session.Connect(endpoint); err != nil {
		return err
	}
	r.mu.Lock()
	r.authSnap.LoggedIn = false
	r.mu.Unlock()
	return r.updateDefaults(func(d *Defaults) {
		d.Endpoint = endpoint
	})
}

func (r *Runtime) Disconnect() error {
	if r.session != nil {
		r.session.Close()
	}
	r.mu.Lock()
	r.authSnap.LoggedIn = false
	r.mu.Unlock()
	return nil
}

func (r *Runtime) Register(ctx context.Context, sourceID, targetID uint32, deviceID string) (protoauth.RespData, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.auth.Register(timeoutCtx, sourceID, targetID, deviceID)
}

func (r *Runtime) Login(ctx context.Context, sourceID, targetID uint32, deviceID string, nodeID uint32) (protoauth.RespData, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.auth.Login(timeoutCtx, sourceID, targetID, deviceID, nodeID)
}

func (r *Runtime) GetPerms(ctx context.Context, sourceID, targetID, nodeID uint32) (protoauth.RespData, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.auth.GetPerms(timeoutCtx, sourceID, targetID, nodeID)
}

func (r *Runtime) ListRoles(ctx context.Context, sourceID, targetID uint32, req protoauth.ListRolesReq) (authsvc.ListRolesResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.auth.ListRoles(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) PushPermsSnapshot(ctx context.Context, sourceID, targetID uint32, snapshot coreperm.Snapshot) error {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.auth.PushPermsSnapshot(timeoutCtx, sourceID, targetID, snapshot)
}

func (r *Runtime) ListPendingRegisters(ctx context.Context, sourceID, targetID uint32, req authsvc.ListPendingRegistersReq) (authsvc.ListPendingRegistersResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.auth.ListPendingRegisters(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) ApproveRegister(ctx context.Context, sourceID, targetID uint32, req authsvc.ApproveRegisterReq) (authsvc.ApproveRegisterResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.auth.ApproveRegister(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) RejectRegister(ctx context.Context, sourceID, targetID uint32, req authsvc.RejectRegisterReq) (authsvc.RejectRegisterResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.auth.RejectRegister(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) IssueRegisterPermit(ctx context.Context, sourceID, targetID uint32, req authsvc.IssueRegisterPermitReq) (authsvc.IssueRegisterPermitResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.auth.IssueRegisterPermit(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) RevokeRegisterPermit(ctx context.Context, sourceID, targetID uint32, req authsvc.RevokeRegisterPermitReq) (authsvc.RevokeRegisterPermitResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.auth.RevokeRegisterPermit(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) CompleteAuth(resp protoauth.RespData, deviceID string) error {
	// CompleteAuth 在 auth 成功后同步更新 snapshot 与 defaults，并把可复用身份写回 store。
	if resp.Code != 1 {
		return nil
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		deviceID = strings.TrimSpace(resp.DeviceID)
	}

	r.mu.Lock()
	r.authSnap = AuthSnapshot{
		DeviceID: deviceID,
		NodeID:   resp.NodeID,
		HubID:    resp.HubID,
		Role:     strings.TrimSpace(resp.Role),
		LoggedIn: true,
	}
	if deviceID != "" {
		r.defaults.DeviceID = deviceID
	}
	if resp.NodeID != 0 {
		r.defaults.NodeID = resp.NodeID
	}
	if resp.HubID != 0 {
		r.defaults.HubID = resp.HubID
		if r.defaults.DefaultTarget == 0 {
			r.defaults.DefaultTarget = resp.HubID
		}
	}
	if role := strings.TrimSpace(resp.Role); role != "" {
		r.defaults.Role = role
	}
	defaults := r.defaults
	r.mu.Unlock()

	return r.persistDefaults(defaults)
}

func (r *Runtime) ListNodes(ctx context.Context, sourceID, targetID uint32) (protomanagement.ListNodesResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.management.ListNodes(timeoutCtx, sourceID, targetID)
}

func (r *Runtime) NodeInfo(ctx context.Context, sourceID, targetID uint32) (protomanagement.NodeInfoResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.management.NodeInfo(timeoutCtx, sourceID, targetID)
}

func (r *Runtime) NodeEcho(ctx context.Context, sourceID, targetID uint32, message string) (protomanagement.NodeEchoResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.management.NodeEcho(timeoutCtx, sourceID, targetID, message)
}

func (r *Runtime) ConfigGet(ctx context.Context, sourceID, targetID uint32, key string) (protomanagement.ConfigResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.management.ConfigGet(timeoutCtx, sourceID, targetID, key)
}

func (r *Runtime) ConfigSet(ctx context.Context, sourceID, targetID uint32, key, value string) (protomanagement.ConfigResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.management.ConfigSet(timeoutCtx, sourceID, targetID, key, value)
}

func (r *Runtime) ConfigList(ctx context.Context, sourceID, targetID uint32) (protomanagement.ConfigListResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.management.ConfigList(timeoutCtx, sourceID, targetID)
}

func (r *Runtime) ListSubtree(ctx context.Context, sourceID, targetID uint32) (protomanagement.ListSubtreeResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.management.ListSubtree(timeoutCtx, sourceID, targetID)
}

func (r *Runtime) ExecCapQuery(ctx context.Context, sourceID, targetID uint32, req protoexec.CapQueryReq) (protoexec.CapQueryResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.flow.ExecCapQuery(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) FlowSet(ctx context.Context, sourceID, targetID uint32, req protoflow.SetReq) (protoflow.SetResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.flow.Set(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) FlowDelete(ctx context.Context, sourceID, targetID uint32, req flowsvc.DeleteReq) (flowsvc.DeleteResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.flow.Delete(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) FlowRun(ctx context.Context, sourceID, targetID uint32, req protoflow.RunReq) (protoflow.RunResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.flow.Run(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) FlowCancelRun(ctx context.Context, sourceID, targetID uint32, req flowsvc.CancelRunReq) (flowsvc.CancelRunResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.flow.CancelRun(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) FlowStatus(ctx context.Context, sourceID, targetID uint32, req protoflow.StatusReq) (protoflow.StatusResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.flow.Status(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) FlowListRuns(ctx context.Context, sourceID, targetID uint32, req flowsvc.ListRunsReq) (flowsvc.ListRunsResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.flow.ListRuns(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) FlowList(ctx context.Context, sourceID, targetID uint32, req protoflow.ListReq) (protoflow.ListResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.flow.List(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) FlowGet(ctx context.Context, sourceID, targetID uint32, req protoflow.GetReq) (protoflow.GetResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.flow.Get(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) TopicBusPublish(ctx context.Context, sourceID, targetID uint32, topic, name, payloadText string) error {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.topicbus.Publish(timeoutCtx, sourceID, targetID, topic, name, payloadText)
}

func (r *Runtime) VarList(ctx context.Context, sourceID, targetID uint32, req protovarstore.ListReq) (protovarstore.VarResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.varpool.List(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) VarGet(ctx context.Context, sourceID, targetID uint32, req protovarstore.GetReq) (protovarstore.VarResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.varpool.Get(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) VarSet(ctx context.Context, sourceID, targetID uint32, req protovarstore.SetReq) (protovarstore.VarResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.varpool.Set(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) VarRevoke(ctx context.Context, sourceID, targetID uint32, req protovarstore.GetReq) (protovarstore.VarResp, error) {
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	return r.varpool.Revoke(timeoutCtx, sourceID, targetID, req)
}

func (r *Runtime) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	// withTimeout 为所有 MCP 后端调用套统一超时，避免工具层自己分散管理时钟。
	if ctx == nil {
		ctx = r.ctx
	}
	timeout := r.timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

func (r *Runtime) applyConfig(config Config) error {
	// applyConfig 只把启动参数覆盖到 defaults，不直接触碰 auth snapshot。
	return r.updateDefaults(func(d *Defaults) {
		if endpoint := strings.TrimSpace(config.Endpoint); endpoint != "" {
			d.Endpoint = endpoint
		}
		if deviceID := strings.TrimSpace(config.DeviceID); deviceID != "" {
			d.DeviceID = deviceID
		}
		if displayName := strings.TrimSpace(config.DisplayName); displayName != "" {
			d.DisplayName = displayName
		}
		if config.DefaultTarget != 0 {
			d.DefaultTarget = config.DefaultTarget
		}
		d.AllowWrite = config.AllowWrite
	})
}

func (r *Runtime) updateDefaults(update func(*Defaults)) error {
	if update == nil {
		return nil
	}
	r.mu.Lock()
	update(&r.defaults)
	defaults := r.defaults
	r.mu.Unlock()
	return r.persistDefaults(defaults)
}

func (r *Runtime) loadDefaultsFromStore() {
	// loadDefaultsFromStore 在 runtime 初始化时恢复上次成功链路留下的默认连接与身份线索。
	defaults := Defaults{
		Endpoint:      r.rawString(mcpEndpointKey),
		DeviceID:      r.rawString(mcpDeviceIDKey),
		DisplayName:   r.rawString(mcpDisplayNameKey),
		NodeID:        r.rawUint32(mcpNodeIDKey),
		HubID:         r.rawUint32(mcpHubIDKey),
		Role:          r.rawString(mcpRoleKey),
		DefaultTarget: r.rawUint32(mcpDefaultTargetKey),
		AllowWrite:    r.rawBool(mcpAllowWriteKey),
	}
	r.mu.Lock()
	r.defaults = defaults
	r.authSnap = AuthSnapshot{
		DeviceID: defaults.DeviceID,
		NodeID:   defaults.NodeID,
		HubID:    defaults.HubID,
		Role:     defaults.Role,
		LoggedIn: false,
	}
	r.mu.Unlock()
}

func (r *Runtime) persistDefaults(defaults Defaults) error {
	// persistDefaults 统一写回 MCP 专用 key，避免 GUI profile 与 headless client 互相污染。
	pairs := []struct {
		key   string
		value any
	}{
		{key: mcpEndpointKey, value: defaults.Endpoint},
		{key: mcpDeviceIDKey, value: defaults.DeviceID},
		{key: mcpDisplayNameKey, value: defaults.DisplayName},
		{key: mcpNodeIDKey, value: defaults.NodeID},
		{key: mcpHubIDKey, value: defaults.HubID},
		{key: mcpRoleKey, value: defaults.Role},
		{key: mcpDefaultTargetKey, value: defaults.DefaultTarget},
		{key: mcpAllowWriteKey, value: defaults.AllowWrite},
	}
	for _, pair := range pairs {
		if err := r.store.SetRaw(pair.key, pair.value); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) rawString(key string) string {
	if r == nil || r.store == nil {
		return ""
	}
	raw, ok := r.store.GetRaw(key)
	if !ok {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func (r *Runtime) rawUint32(key string) uint32 {
	if r == nil || r.store == nil {
		return 0
	}
	raw, ok := r.store.GetRaw(key)
	if !ok {
		return 0
	}
	switch v := raw.(type) {
	case uint32:
		return v
	case uint64:
		return uint32(v)
	case json.Number:
		n, err := v.Int64()
		if err != nil || n < 0 {
			return 0
		}
		return uint32(n)
	case int:
		if v < 0 {
			return 0
		}
		return uint32(v)
	case int64:
		if v < 0 {
			return 0
		}
		return uint32(v)
	case float64:
		if v < 0 {
			return 0
		}
		return uint32(v)
	case string:
		n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 32)
		if err != nil {
			return 0
		}
		return uint32(n)
	default:
		return 0
	}
}

func (r *Runtime) rawBool(key string) bool {
	if r == nil || r.store == nil {
		return false
	}
	raw, ok := r.store.GetRaw(key)
	if !ok {
		return false
	}
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		out, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return false
		}
		return out
	default:
		return false
	}
}

func (r *Runtime) attachLogWriter() {
	// attachLogWriter 把内部日志转发到 stderr，保证 stdout 只保留 JSON-RPC 数据。
	if r == nil || r.bus == nil || r.logWriter == nil {
		return
	}
	r.logToken = r.bus.Subscribe(logssvc.EventLogLine, func(_ context.Context, evt corebus.Event) {
		line, ok := evt.Data.(logssvc.LogLine)
		if !ok {
			return
		}
		r.writeLogLine(line)
	})
}

func (r *Runtime) writeLogLine(line logssvc.LogLine) {
	r.logMu.Lock()
	defer r.logMu.Unlock()

	level := strings.ToUpper(strings.TrimSpace(line.Level))
	if level == "" {
		level = "INFO"
	}

	message := strings.TrimSpace(line.Message)
	if message == "" {
		message = "-"
	}

	_, _ = fmt.Fprintf(r.logWriter, "%s [%s] %s", line.Time.Format(time.RFC3339), level, message)
	if line.PayloadLen > 0 {
		_, _ = fmt.Fprintf(r.logWriter, " payload_len=%d", line.PayloadLen)
	}
	if len(line.Payload) > 0 {
		_, _ = fmt.Fprintf(r.logWriter, " payload_hex=%s", hex.EncodeToString(line.Payload))
	}
	_, _ = io.WriteString(r.logWriter, "\n")
}

func normalizeTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return defaultRequestTimeout
	}
	return timeout
}

func resolveConfigDir(configDir string) (string, error) {
	// resolveConfigDir 允许显式目录覆盖，也能在未传参时推导出稳定的用户级配置根。
	configDir = strings.TrimSpace(configDir)
	if configDir != "" {
		resolved, err := filepath.Abs(filepath.Clean(configDir))
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(resolved) == "" {
			return "", errors.New("config dir is empty")
		}
		return resolved, nil
	}

	base, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(base) == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil || strings.TrimSpace(home) == "" {
			return "", errors.New("user config dir unavailable")
		}
		if runtime.GOOS == "windows" {
			base = filepath.Join(home, "AppData", "Roaming")
		} else {
			base = filepath.Join(home, ".config")
		}
	}
	return filepath.Join(base, defaultConfigDirName, defaultConfigProfile), nil
}
