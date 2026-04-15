// 本文件组装 Wails 应用壳层，以及绑定给 Win 前端的共享后端服务。

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	corebus "github.com/yttydcs/myflowhub-core/eventbus"
	authsvc "github.com/yttydcs/myflowhub-win/internal/services/auth"
	debugsvc "github.com/yttydcs/myflowhub-win/internal/services/debug"
	filesvc "github.com/yttydcs/myflowhub-win/internal/services/file"
	flowsvc "github.com/yttydcs/myflowhub-win/internal/services/flow"
	localhubsvc "github.com/yttydcs/myflowhub-win/internal/services/localhub"
	logssvc "github.com/yttydcs/myflowhub-win/internal/services/logs"
	mgmtsvc "github.com/yttydcs/myflowhub-win/internal/services/management"
	permissionsvc "github.com/yttydcs/myflowhub-win/internal/services/permission"
	presetssvc "github.com/yttydcs/myflowhub-win/internal/services/presets"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
	streamsvc "github.com/yttydcs/myflowhub-win/internal/services/stream"
	topicbussvc "github.com/yttydcs/myflowhub-win/internal/services/topicbus"
	varpoolsvc "github.com/yttydcs/myflowhub-win/internal/services/varpool"
	storagesvc "github.com/yttydcs/myflowhub-win/internal/storage"
)

type App struct {
	ctx          context.Context
	bus          corebus.IBus
	logs         *logssvc.LogService
	session      *sessionsvc.SessionService
	localhub     *localhubsvc.LocalHubService
	auth         *authsvc.AuthService
	varpool      *varpoolsvc.VarPoolService
	topicbus     *topicbussvc.TopicBusService
	stream       *streamsvc.StreamService
	file         *filesvc.FileService
	flow         *flowsvc.FlowService
	management   *mgmtsvc.ManagementService
	permission   *permissionsvc.PermissionService
	debug        *debugsvc.DebugService
	presets      *presetssvc.PresetService
	store        *storagesvc.Store
	bridgeTokens []busToken
}

type busToken struct {
	name  string
	token string
}

func NewApp() *App {
	// NewApp 只创建一套共享 service 实例，避免不同 Wails 绑定各自维护分裂状态。
	bus := corebus.New(corebus.Options{})
	logs := logssvc.New(bus, 2000)
	session := sessionsvc.New(context.Background(), bus, logs)
	store, err := storagesvc.NewStore()
	if err != nil {
		logs.Appendf("error", "storage init failed: %v", err)
	}
	if store != nil {
		if err := store.MigrateLegacyNodeKeysForProfiles(); err != nil {
			logs.Appendf("warn", "node keys migration warning: %v", err)
		}
	}
	app := &App{
		bus:        bus,
		logs:       logs,
		session:    session,
		localhub:   localhubsvc.New(store, logs),
		auth:       authsvc.New(session, logs, store),
		varpool:    varpoolsvc.New(session, logs, bus),
		topicbus:   topicbussvc.New(session, logs, bus),
		stream:     streamsvc.New(session, logs, bus),
		file:       filesvc.New(session, logs, store, bus),
		flow:       flowsvc.New(session, logs),
		management: mgmtsvc.New(session, logs, store),
		debug:      debugsvc.New(session, logs),
		presets:    presetssvc.New(session, bus),
		store:      store,
	}
	// Keep shared service instances to avoid duplicated state and logging.
	app.permission = permissionsvc.New(app.auth, app.management, logs)
	if store != nil {
		current := store.CurrentProfile()
		app.auth.SetKeysPath(store.NodeKeysPath(current))
	}
	return app
}

func (a *App) Bindings() []interface{} {
	// Bindings 列出前端可直接访问的 App 与后端 service 绑定对象。
	return []interface{}{a, a.logs, a.session, a.localhub, a.auth, a.varpool, a.topicbus, a.stream, a.file, a.flow, a.management, a.permission, a.debug, a.presets}
}

func (a *App) Startup(ctx context.Context) {
	// Startup 在 runtime 就绪后注入 context，并开始桥接后端事件给前端。
	a.ctx = ctx
	if a.session != nil {
		a.session.SetContext(ctx)
	}
	a.bridgeEvents()
}

func (a *App) Shutdown(ctx context.Context) {
	// Shutdown 按依赖顺序关闭订阅与 service，避免窗口退出后残留 goroutine 或事件源。
	_ = ctx
	a.unbridgeEvents()
	if a.topicbus != nil {
		a.topicbus.Close()
	}
	if a.stream != nil {
		a.stream.Close()
	}
	if a.varpool != nil {
		a.varpool.Close()
	}
	if a.presets != nil {
		a.presets.Close()
	}
	if a.session != nil {
		a.session.Close()
	}
	if a.bus != nil {
		a.bus.Close()
	}
}

func (a *App) bridgeEvents() {
	// bridgeEvents 把 bus 上的后端事件原样转成前端可通过 EventsOn 监听的名称。
	if a.bus == nil || a.ctx == nil {
		return
	}
	emit := func(name string, data any) {
		runtime.EventsEmit(a.ctx, name, data)
	}
	bind := func(name string) {
		token := a.bus.Subscribe(name, func(_ context.Context, evt corebus.Event) {
			emit(name, evt.Data)
		})
		if token != "" {
			a.bridgeTokens = append(a.bridgeTokens, busToken{name: name, token: token})
		}
	}
	bind(logssvc.EventLogLine)
	bind(sessionsvc.EventFrame)
	bind(sessionsvc.EventState)
	bind(sessionsvc.EventError)
	bind(filesvc.EventFileTasks)
	bind(filesvc.EventFileList)
	bind(filesvc.EventFileText)
	bind(filesvc.EventFileOffer)
	bind(presetssvc.EventTopicStressSender)
	bind(presetssvc.EventTopicStressReceiver)
	bind(topicbussvc.EventTopicBusEvent)
	bind(streamsvc.EventStreamDelivery)
	bind(streamsvc.EventStreamText)
	bind(streamsvc.EventStreamStats)
	bind(streamsvc.EventStreamMedia)
	bind(varpoolsvc.EventVarPoolChanged)
	bind(varpoolsvc.EventVarPoolDeleted)
}

func (a *App) unbridgeEvents() {
	// unbridgeEvents 释放已注册的事件桥接，防止重复 Startup 后事件被多次转发。
	if a.bus == nil {
		return
	}
	for _, entry := range a.bridgeTokens {
		if entry.token == "" {
			continue
		}
		a.bus.Unsubscribe(entry.name, entry.token)
	}
	a.bridgeTokens = nil
}

func (a *App) Greet(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		err := errors.New("name is required")
		log.Printf("greet rejected: %v", err)
		return "", err
	}
	if len(trimmed) > 64 {
		err := fmt.Errorf("name too long: %d", len(trimmed))
		log.Printf("greet rejected: %v", err)
		return "", err
	}
	return fmt.Sprintf("Hello %s", trimmed), nil
}

func (a *App) ProfileState() (storagesvc.ProfileState, error) {
	// ProfileState 返回当前 profile 元信息，供壳层展示和切换配置目录。
	if a.store == nil {
		return storagesvc.ProfileState{}, errors.New("storage not initialized")
	}
	return a.store.State(), nil
}

func (a *App) SetCurrentProfile(name string) (storagesvc.ProfileState, error) {
	// SetCurrentProfile 切换 profile 后同步 node key 路径，保证后续 auth 使用对应密钥。
	if a.store == nil {
		return storagesvc.ProfileState{}, errors.New("storage not initialized")
	}
	if _, err := a.store.MigrateLegacyNodeKeys(name); err != nil {
		return storagesvc.ProfileState{}, err
	}
	if err := a.store.SetCurrentProfile(name); err != nil {
		return storagesvc.ProfileState{}, err
	}
	current := a.store.CurrentProfile()
	if a.auth != nil {
		a.auth.SetKeysPath(a.store.NodeKeysPath(current))
	}
	return a.store.State(), nil
}
