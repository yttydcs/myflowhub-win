// Context: covers the settings app binding helpers and persistence rules.

package main

import (
	"runtime"
	"testing"

	storagesvc "github.com/yttydcs/myflowhub-win/internal/storage"
)

func newTestAppWithStore(t *testing.T) *App {
	t.Helper()
	base := t.TempDir()
	t.Setenv("APPDATA", base)
	t.Setenv("XDG_CONFIG_HOME", base)
	t.Setenv("HOME", base)

	store, err := storagesvc.NewStore()
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	return &App{store: store}
}

func TestSettingsState_Defaults(t *testing.T) {
	app := newTestAppWithStore(t)

	state, err := app.SettingsState()
	if err != nil {
		t.Fatalf("SettingsState() error = %v", err)
	}

	if state.DefaultAddr != appDefaultAddr {
		t.Fatalf("expected defaultAddr=%q got %q", appDefaultAddr, state.DefaultAddr)
	}
	if state.DefaultDeviceID != "" {
		t.Fatalf("expected empty defaultDeviceId got %q", state.DefaultDeviceID)
	}
	if state.AutoConnect {
		t.Fatal("expected autoConnect=false")
	}
	if state.AutoLogin {
		t.Fatal("expected autoLogin=false")
	}
	if state.DefaultStartPage != startPageHome {
		t.Fatalf("expected defaultStartPage=%q got %q", startPageHome, state.DefaultStartPage)
	}
	if state.Density != densityComfortable {
		t.Fatalf("expected density=%q got %q", densityComfortable, state.Density)
	}
	if state.ReduceMotion {
		t.Fatal("expected reduceMotion=false")
	}
}

func TestGlobalPreferencesState_Defaults(t *testing.T) {
	app := newTestAppWithStore(t)

	state, err := app.GlobalPreferencesState()
	if err != nil {
		t.Fatalf("GlobalPreferencesState() error = %v", err)
	}
	if state.Language != languageEnglish {
		t.Fatalf("expected language=%q got %q", languageEnglish, state.Language)
	}
}

func TestSettingsState_FallsBackToLegacyHomeKeys(t *testing.T) {
	app := newTestAppWithStore(t)
	profile := app.store.CurrentProfile()

	if err := app.store.SetString(profile, homeDeviceIDKey, "legacy-device"); err != nil {
		t.Fatalf("SetString(device) error = %v", err)
	}
	if err := app.store.SetBool(profile, homeAutoConnectKey, true); err != nil {
		t.Fatalf("SetBool(autoConnect) error = %v", err)
	}
	if err := app.store.SetBool(profile, homeAutoLoginKey, true); err != nil {
		t.Fatalf("SetBool(autoLogin) error = %v", err)
	}

	state, err := app.SettingsState()
	if err != nil {
		t.Fatalf("SettingsState() error = %v", err)
	}

	if state.DefaultDeviceID != "legacy-device" {
		t.Fatalf("expected fallback deviceId got %q", state.DefaultDeviceID)
	}
	if !state.AutoConnect {
		t.Fatal("expected fallback autoConnect=true")
	}
	if !state.AutoLogin {
		t.Fatal("expected fallback autoLogin=true")
	}
	if state.DefaultAddr != appDefaultAddr {
		t.Fatalf("expected defaultAddr fallback=%q got %q", appDefaultAddr, state.DefaultAddr)
	}
}

func TestSaveSettingsState_NormalizesAndPersists(t *testing.T) {
	app := newTestAppWithStore(t)

	state, err := app.SaveSettingsState(AppSettingsState{
		DefaultAddr:      "   ",
		DefaultDeviceID:  "  device-01  ",
		AutoConnect:      true,
		AutoLogin:        true,
		DefaultStartPage: "FLOW",
		Density:          "COMPACT",
		ReduceMotion:     true,
	})
	if err != nil {
		t.Fatalf("SaveSettingsState() error = %v", err)
	}

	if state.DefaultAddr != appDefaultAddr {
		t.Fatalf("expected normalized defaultAddr=%q got %q", appDefaultAddr, state.DefaultAddr)
	}
	if state.DefaultDeviceID != "device-01" {
		t.Fatalf("expected trimmed deviceId got %q", state.DefaultDeviceID)
	}
	if state.DefaultStartPage != startPageFlow {
		t.Fatalf("expected startPage=%q got %q", startPageFlow, state.DefaultStartPage)
	}
	if state.Density != densityCompact {
		t.Fatalf("expected density=%q got %q", densityCompact, state.Density)
	}
	if !state.AutoConnect || !state.AutoLogin || !state.ReduceMotion {
		t.Fatalf("expected booleans persisted got %+v", state)
	}
}

func TestSaveGlobalPreferencesState_NormalizesAndPersistsToRawKey(t *testing.T) {
	app := newTestAppWithStore(t)
	if err := app.store.SetCurrentProfile("work"); err != nil {
		t.Fatalf("SetCurrentProfile() error = %v", err)
	}

	state, err := app.SaveGlobalPreferencesState(GlobalPreferencesState{
		Language: "  ZH-cn  ",
	})
	if err != nil {
		t.Fatalf("SaveGlobalPreferencesState() error = %v", err)
	}
	if state.Language != languageSimplifiedChinese {
		t.Fatalf("expected normalized language=%q got %q", languageSimplifiedChinese, state.Language)
	}

	raw, ok := app.store.GetRaw(globalPreferencesKey)
	if !ok {
		t.Fatalf("expected raw key %q persisted", globalPreferencesKey)
	}
	parsed, ok := parseGlobalPreferencesStateValue(raw)
	if !ok {
		t.Fatalf("expected persisted raw value parseable, got %#v", raw)
	}
	if parsed.Language != languageSimplifiedChinese {
		t.Fatalf("expected persisted raw language=%q got %q", languageSimplifiedChinese, parsed.Language)
	}

	if _, ok := app.store.GetRaw("work." + globalPreferencesKey); ok {
		t.Fatalf("expected no profile-scoped key for global preferences")
	}
}

func TestSaveGlobalPreferencesState_InvalidLanguageFallsBackToDefault(t *testing.T) {
	app := newTestAppWithStore(t)

	state, err := app.SaveGlobalPreferencesState(GlobalPreferencesState{
		Language: "fr",
	})
	if err != nil {
		t.Fatalf("SaveGlobalPreferencesState() error = %v", err)
	}
	if state.Language != languageEnglish {
		t.Fatalf("expected invalid language to fall back to %q got %q", languageEnglish, state.Language)
	}
}

func TestSaveHomeState_DoesNotMutateSavedSettings(t *testing.T) {
	app := newTestAppWithStore(t)

	if _, err := app.SaveSettingsState(AppSettingsState{
		DefaultAddr:      "10.0.0.8:9000",
		DefaultDeviceID:  "default-device",
		AutoConnect:      true,
		AutoLogin:        true,
		DefaultStartPage: startPageSettings,
		Density:          densityCompact,
		ReduceMotion:     true,
	}); err != nil {
		t.Fatalf("SaveSettingsState() error = %v", err)
	}

	_, err := app.SaveHomeState(HomeState{
		DeviceID:    "device-02",
		AutoConnect: true,
		AutoLogin:   false,
		NodeID:      7,
		HubID:       9,
		Role:        "node",
	})
	if err != nil {
		t.Fatalf("SaveHomeState() error = %v", err)
	}

	settings, err := app.SettingsState()
	if err != nil {
		t.Fatalf("SettingsState() error = %v", err)
	}
	if settings.DefaultDeviceID != "default-device" {
		t.Fatalf("expected settings deviceId unchanged got %q", settings.DefaultDeviceID)
	}
	if settings.DefaultAddr != "10.0.0.8:9000" {
		t.Fatalf("expected defaultAddr unchanged got %q", settings.DefaultAddr)
	}
	if !settings.AutoConnect || !settings.AutoLogin || !settings.ReduceMotion {
		t.Fatalf("expected settings flags unchanged got %+v", settings)
	}
}

func TestResetSettingsState_RestoresDefaults(t *testing.T) {
	app := newTestAppWithStore(t)

	if _, err := app.SaveSettingsState(AppSettingsState{
		DefaultAddr:      "192.168.1.10:9000",
		DefaultDeviceID:  "device-03",
		AutoConnect:      true,
		AutoLogin:        true,
		DefaultStartPage: startPageSettings,
		Density:          densityCompact,
		ReduceMotion:     true,
	}); err != nil {
		t.Fatalf("SaveSettingsState() error = %v", err)
	}

	state, err := app.ResetSettingsState()
	if err != nil {
		t.Fatalf("ResetSettingsState() error = %v", err)
	}
	if state != defaultAppSettingsState() {
		t.Fatalf("expected defaults after reset got %+v", state)
	}
}

func TestResetGlobalPreferencesState_RestoresDefaults(t *testing.T) {
	app := newTestAppWithStore(t)

	if _, err := app.SaveGlobalPreferencesState(GlobalPreferencesState{
		Language: languageSimplifiedChinese,
	}); err != nil {
		t.Fatalf("SaveGlobalPreferencesState() error = %v", err)
	}

	state, err := app.ResetGlobalPreferencesState()
	if err != nil {
		t.Fatalf("ResetGlobalPreferencesState() error = %v", err)
	}
	if state != defaultGlobalPreferencesState() {
		t.Fatalf("expected defaults after reset got %+v", state)
	}
}

func TestAboutState_ContainsStableFields(t *testing.T) {
	app := newTestAppWithStore(t)

	about, err := app.AboutState()
	if err != nil {
		t.Fatalf("AboutState() error = %v", err)
	}

	if about.AppName != "MyFlowHub" {
		t.Fatalf("expected appName=MyFlowHub got %q", about.AppName)
	}
	if about.Platform != runtime.GOOS+"/"+runtime.GOARCH {
		t.Fatalf("expected platform=%q got %q", runtime.GOOS+"/"+runtime.GOARCH, about.Platform)
	}
	if about.GoVersion != runtime.Version() {
		t.Fatalf("expected goVersion=%q got %q", runtime.Version(), about.GoVersion)
	}
	if about.Profile != "default" {
		t.Fatalf("expected profile=default got %q", about.Profile)
	}
	if about.BaseDir == "" || about.SettingsPath == "" || about.KeysPath == "" {
		t.Fatalf("expected profile paths populated got %+v", about)
	}
	if about.BuildMode != buildModeDev && about.BuildMode != buildModeRelease {
		t.Fatalf("unexpected buildMode=%q", about.BuildMode)
	}
}
