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
	logssvc "github.com/yttydcs/myflowhub-win/internal/services/logs"
	mgmtsvc "github.com/yttydcs/myflowhub-win/internal/services/management"
	presetssvc "github.com/yttydcs/myflowhub-win/internal/services/presets"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
	topicbussvc "github.com/yttydcs/myflowhub-win/internal/services/topicbus"
	varpoolsvc "github.com/yttydcs/myflowhub-win/internal/services/varpool"
	storagesvc "github.com/yttydcs/myflowhub-win/internal/storage"
)

type App struct {
	ctx          context.Context
	bus          corebus.IBus
	logs         *logssvc.LogService
	session      *sessionsvc.SessionService
	auth         *authsvc.AuthService
	varpool      *varpoolsvc.VarPoolService
	topicbus     *topicbussvc.TopicBusService
	file         *filesvc.FileService
	flow         *flowsvc.FlowService
	management   *mgmtsvc.ManagementService
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
		auth:       authsvc.New(session, logs),
		varpool:    varpoolsvc.New(session, logs, bus),
		topicbus:   topicbussvc.New(session, logs, bus),
		file:       filesvc.New(session, logs, store, bus),
		flow:       flowsvc.New(session, logs),
		management: mgmtsvc.New(session, logs),
		debug:      debugsvc.New(session, logs),
		presets:    presetssvc.New(session, bus),
		store:      store,
	}
	if store != nil {
		current := store.CurrentProfile()
		app.auth.SetKeysPath(store.NodeKeysPath(current))
	}
	return app
}

func (a *App) Bindings() []interface{} {
	return []interface{}{a, a.logs, a.session, a.auth, a.varpool, a.topicbus, a.file, a.flow, a.management, a.debug, a.presets}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	if a.session != nil {
		a.session.SetContext(ctx)
	}
	a.bridgeEvents()
}

func (a *App) Shutdown(ctx context.Context) {
	_ = ctx
	a.unbridgeEvents()
	if a.topicbus != nil {
		a.topicbus.Close()
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
	bind(varpoolsvc.EventVarPoolChanged)
	bind(varpoolsvc.EventVarPoolDeleted)
}

func (a *App) unbridgeEvents() {
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
	if a.store == nil {
		return storagesvc.ProfileState{}, errors.New("storage not initialized")
	}
	return a.store.State(), nil
}

func (a *App) SetCurrentProfile(name string) (storagesvc.ProfileState, error) {
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
