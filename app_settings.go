// Context: persists global app settings, build/about metadata, and startup UI preferences for the Win shell.

package main

import (
	"encoding/json"
	"errors"
	"runtime"
	"runtime/debug"
	"strings"

	storagesvc "github.com/yttydcs/myflowhub-win/internal/storage"
)

const (
	appSettingsKey       = "app.settings"
	globalPreferencesKey = "app.global_preferences"

	appDefaultAddr = "127.0.0.1:9000"

	startPageHome     = "home"
	startPageDevices  = "devices"
	startPageFlow     = "flow"
	startPageSettings = "settings"

	densityComfortable = "comfortable"
	densityCompact     = "compact"

	buildModeDev     = "dev"
	buildModeRelease = "release"

	languageEnglish           = "en"
	languageSimplifiedChinese = "zh-CN"
)

type AppSettingsState struct {
	DefaultAddr      string `json:"defaultAddr"`
	DefaultDeviceID  string `json:"defaultDeviceId"`
	AutoConnect      bool   `json:"autoConnect"`
	AutoLogin        bool   `json:"autoLogin"`
	DefaultStartPage string `json:"defaultStartPage"`
	Density          string `json:"density"`
	ReduceMotion     bool   `json:"reduceMotion"`
}

type AppAboutState struct {
	AppName      string `json:"appName"`
	AppVersion   string `json:"appVersion"`
	BuildTime    string `json:"buildTime"`
	BuildMode    string `json:"buildMode"`
	Commit       string `json:"commit"`
	Platform     string `json:"platform"`
	GoVersion    string `json:"goVersion"`
	WailsVersion string `json:"wailsVersion"`
	Profile      string `json:"profile"`
	BaseDir      string `json:"baseDir"`
	SettingsPath string `json:"settingsPath"`
	KeysPath     string `json:"keysPath"`
}

type GlobalPreferencesState struct {
	Language string `json:"language"`
}

func (a *App) SettingsState() (AppSettingsState, error) {
	if a.store == nil {
		return AppSettingsState{}, errors.New("storage not initialized")
	}
	return loadAppSettingsState(a.store, a.store.CurrentProfile()), nil
}

func (a *App) SaveSettingsState(state AppSettingsState) (AppSettingsState, error) {
	if a.store == nil {
		return AppSettingsState{}, errors.New("storage not initialized")
	}
	normalized := normalizeAppSettingsState(state)
	if err := validateAppSettingsState(normalized); err != nil {
		return AppSettingsState{}, err
	}
	profile := a.store.CurrentProfile()
	if err := persistAppSettingsState(a.store, profile, normalized); err != nil {
		return AppSettingsState{}, err
	}
	return a.SettingsState()
}

func (a *App) ResetSettingsState() (AppSettingsState, error) {
	if a.store == nil {
		return AppSettingsState{}, errors.New("storage not initialized")
	}
	profile := a.store.CurrentProfile()
	state := defaultAppSettingsState()
	if err := persistAppSettingsState(a.store, profile, state); err != nil {
		return AppSettingsState{}, err
	}
	return a.SettingsState()
}

func (a *App) AboutState() (AppAboutState, error) {
	if a.store == nil {
		return AppAboutState{}, errors.New("storage not initialized")
	}
	profileState := a.store.State()
	about := AppAboutState{
		AppName:      "MyFlowHub",
		AppVersion:   buildModeDev,
		BuildTime:    "-",
		BuildMode:    buildModeDev,
		Commit:       "-",
		Platform:     runtime.GOOS + "/" + runtime.GOARCH,
		GoVersion:    runtime.Version(),
		WailsVersion: "unknown",
		Profile:      profileState.Current,
		BaseDir:      profileState.BaseDir,
		SettingsPath: profileState.SettingsPath,
		KeysPath:     profileState.KeysPath,
	}

	if bi, ok := debug.ReadBuildInfo(); ok && bi != nil {
		version := strings.TrimSpace(bi.Main.Version)
		if version != "" && version != "(devel)" {
			about.AppVersion = version
			about.BuildMode = buildModeRelease
		}
		if depVersion := resolveBuildDependencyVersion(bi, "github.com/wailsapp/wails/v2"); depVersion != "" {
			about.WailsVersion = depVersion
		}
		for _, setting := range bi.Settings {
			switch strings.TrimSpace(setting.Key) {
			case "vcs.revision":
				if commit := shortCommit(setting.Value); commit != "" {
					about.Commit = commit
				}
			case "vcs.time":
				if ts := strings.TrimSpace(setting.Value); ts != "" {
					about.BuildTime = ts
				}
			case "vcs.modified":
				if strings.EqualFold(strings.TrimSpace(setting.Value), "true") && about.Commit != "-" {
					about.Commit += "+dirty"
				}
			}
		}
	}

	return about, nil
}

func (a *App) GlobalPreferencesState() (GlobalPreferencesState, error) {
	if a.store == nil {
		return GlobalPreferencesState{}, errors.New("storage not initialized")
	}
	return loadGlobalPreferencesState(a.store), nil
}

func (a *App) SaveGlobalPreferencesState(state GlobalPreferencesState) (GlobalPreferencesState, error) {
	if a.store == nil {
		return GlobalPreferencesState{}, errors.New("storage not initialized")
	}
	normalized := normalizeGlobalPreferencesState(state)
	if err := validateGlobalPreferencesState(normalized); err != nil {
		return GlobalPreferencesState{}, err
	}
	if err := persistGlobalPreferencesState(a.store, normalized); err != nil {
		return GlobalPreferencesState{}, err
	}
	return a.GlobalPreferencesState()
}

func (a *App) ResetGlobalPreferencesState() (GlobalPreferencesState, error) {
	if a.store == nil {
		return GlobalPreferencesState{}, errors.New("storage not initialized")
	}
	state := defaultGlobalPreferencesState()
	if err := persistGlobalPreferencesState(a.store, state); err != nil {
		return GlobalPreferencesState{}, err
	}
	return a.GlobalPreferencesState()
}

func defaultAppSettingsState() AppSettingsState {
	return AppSettingsState{
		DefaultAddr:      appDefaultAddr,
		DefaultDeviceID:  "",
		AutoConnect:      false,
		AutoLogin:        false,
		DefaultStartPage: startPageHome,
		Density:          densityComfortable,
		ReduceMotion:     false,
	}
}

func defaultGlobalPreferencesState() GlobalPreferencesState {
	return GlobalPreferencesState{
		Language: languageEnglish,
	}
}

func loadAppSettingsState(store *storagesvc.Store, profile string) AppSettingsState {
	state := defaultAppSettingsState()
	if store == nil {
		return state
	}

	raw := store.GetString(profile, appSettingsKey, "")
	if parsed, ok := parseAppSettingsState(raw); ok {
		return normalizeAppSettingsState(parsed)
	}

	state.DefaultDeviceID = store.GetString(profile, homeDeviceIDKey, "")
	state.AutoConnect = store.GetBool(profile, homeAutoConnectKey, false)
	state.AutoLogin = store.GetBool(profile, homeAutoLoginKey, false)
	return normalizeAppSettingsState(state)
}

func parseAppSettingsState(raw string) (AppSettingsState, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return AppSettingsState{}, false
	}
	var state AppSettingsState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return AppSettingsState{}, false
	}
	return state, true
}

func loadGlobalPreferencesState(store *storagesvc.Store) GlobalPreferencesState {
	state := defaultGlobalPreferencesState()
	if store == nil {
		return state
	}
	raw, ok := store.GetRaw(globalPreferencesKey)
	if !ok {
		return state
	}
	if parsed, ok := parseGlobalPreferencesStateValue(raw); ok {
		return normalizeGlobalPreferencesState(parsed)
	}
	return state
}

func parseGlobalPreferencesStateValue(raw any) (GlobalPreferencesState, bool) {
	switch v := raw.(type) {
	case string:
		return parseGlobalPreferencesState(v)
	case []byte:
		return parseGlobalPreferencesState(string(v))
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return GlobalPreferencesState{}, false
		}
		return parseGlobalPreferencesState(string(data))
	}
}

func parseGlobalPreferencesState(raw string) (GlobalPreferencesState, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return GlobalPreferencesState{}, false
	}
	var state GlobalPreferencesState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return GlobalPreferencesState{}, false
	}
	return state, true
}

func persistAppSettingsState(store *storagesvc.Store, profile string, state AppSettingsState) error {
	data, err := json.Marshal(normalizeAppSettingsState(state))
	if err != nil {
		return err
	}
	return store.SetString(profile, appSettingsKey, string(data))
}

func persistGlobalPreferencesState(store *storagesvc.Store, state GlobalPreferencesState) error {
	data, err := json.Marshal(normalizeGlobalPreferencesState(state))
	if err != nil {
		return err
	}
	return store.SetRaw(globalPreferencesKey, string(data))
}

func normalizeAppSettingsState(state AppSettingsState) AppSettingsState {
	state.DefaultAddr = strings.TrimSpace(state.DefaultAddr)
	if state.DefaultAddr == "" {
		state.DefaultAddr = appDefaultAddr
	}
	state.DefaultDeviceID = strings.TrimSpace(state.DefaultDeviceID)
	state.DefaultStartPage = normalizeStartPage(state.DefaultStartPage)
	state.Density = normalizeDensity(state.Density)
	return state
}

func normalizeGlobalPreferencesState(state GlobalPreferencesState) GlobalPreferencesState {
	state.Language = normalizeLanguage(state.Language)
	return state
}

func validateAppSettingsState(state AppSettingsState) error {
	if len(state.DefaultAddr) > 256 {
		return errors.New("default_addr is too long")
	}
	if len(state.DefaultDeviceID) > 128 {
		return errors.New("default_device_id is too long")
	}
	if state.DefaultStartPage != normalizeStartPage(state.DefaultStartPage) {
		return errors.New("default_start_page is invalid")
	}
	if state.Density != normalizeDensity(state.Density) {
		return errors.New("density is invalid")
	}
	return nil
}

func validateGlobalPreferencesState(state GlobalPreferencesState) error {
	if state.Language != normalizeLanguage(state.Language) {
		return errors.New("language is invalid")
	}
	return nil
}

func normalizeStartPage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case startPageDevices:
		return startPageDevices
	case startPageFlow:
		return startPageFlow
	case startPageSettings:
		return startPageSettings
	case startPageHome:
		return startPageHome
	default:
		return startPageHome
	}
}

func normalizeDensity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case densityCompact:
		return densityCompact
	case densityComfortable:
		return densityComfortable
	default:
		return densityComfortable
	}
}

func normalizeLanguage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case strings.ToLower(languageSimplifiedChinese):
		return languageSimplifiedChinese
	case languageEnglish:
		return languageEnglish
	default:
		return languageEnglish
	}
}

func resolveBuildDependencyVersion(bi *debug.BuildInfo, path string) string {
	if bi == nil {
		return ""
	}
	for _, dep := range bi.Deps {
		if dep == nil || strings.TrimSpace(dep.Path) != path {
			continue
		}
		if dep.Replace != nil && strings.TrimSpace(dep.Replace.Version) != "" {
			return strings.TrimSpace(dep.Replace.Version)
		}
		return strings.TrimSpace(dep.Version)
	}
	return ""
}

func shortCommit(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 12 {
		return value[:12]
	}
	return value
}
