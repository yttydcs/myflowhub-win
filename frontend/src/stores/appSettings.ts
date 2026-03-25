import { reactive } from "vue"
import { t } from "@/i18n"

type WailsBinding = (...args: any[]) => Promise<any>

export type StartPageKey = "home" | "devices" | "flow" | "settings"
export type DensityKey = "comfortable" | "compact"

export type AppSettingsState = {
  defaultAddr: string
  defaultDeviceId: string
  autoConnect: boolean
  autoLogin: boolean
  defaultStartPage: StartPageKey
  density: DensityKey
  reduceMotion: boolean
}

export type AppAboutState = {
  appName: string
  appVersion: string
  buildTime: string
  buildMode: string
  commit: string
  platform: string
  goVersion: string
  wailsVersion: string
  profile: string
  baseDir: string
  settingsPath: string
  keysPath: string
}

type StartupMirror = Pick<AppSettingsState, "defaultStartPage" | "density" | "reduceMotion">

type AppSettingsStoreState = {
  settings: AppSettingsState
  about: AppAboutState
  loaded: boolean
  loading: boolean
  saving: boolean
  aboutLoading: boolean
  updatedAt: string
  aboutUpdatedAt: string
}

const startupMirrorStorageKey = "myflowhub.app_settings.startup"

const startPagePathMap: Record<StartPageKey, string> = {
  home: "/home",
  devices: "/devices",
  flow: "/flow",
  settings: "/settings"
}

const callApp = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.main?.App
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(t("App binding '{method}' unavailable", { method }))
  }
  return fn(...args)
}

const nowIso = () => new Date().toISOString()

export const defaultAppSettings = (): AppSettingsState => ({
  defaultAddr: "127.0.0.1:9000",
  defaultDeviceId: "",
  autoConnect: false,
  autoLogin: false,
  defaultStartPage: "home",
  density: "comfortable",
  reduceMotion: false
})

const defaultAbout = (): AppAboutState => ({
  appName: "MyFlowHub",
  appVersion: "dev",
  buildTime: "-",
  buildMode: "dev",
  commit: "-",
  platform: "-",
  goVersion: "-",
  wailsVersion: "unknown",
  profile: "default",
  baseDir: "-",
  settingsPath: "-",
  keysPath: "-"
})

export const startPageOptions: { value: StartPageKey; label: string; detail: string }[] = [
  { value: "home", label: "Home", detail: "Session and auth dashboard" },
  { value: "devices", label: "Devices", detail: "Node tree and management queries" },
  { value: "flow", label: "Flow", detail: "Design and deployment workspace" },
  { value: "settings", label: "Settings", detail: "Open the preferences page first" }
]

export const densityOptions: { value: DensityKey; label: string; detail: string }[] = [
  { value: "comfortable", label: "Comfortable", detail: "Use the current spacing rhythm" },
  { value: "compact", label: "Compact", detail: "Reduce shell and panel spacing" }
]

const normalizeStartPage = (value: any): StartPageKey => {
  switch (String(value ?? "").trim().toLowerCase()) {
    case "devices":
      return "devices"
    case "flow":
      return "flow"
    case "settings":
      return "settings"
    case "home":
    default:
      return "home"
  }
}

const normalizeDensity = (value: any): DensityKey => {
  switch (String(value ?? "").trim().toLowerCase()) {
    case "compact":
      return "compact"
    case "comfortable":
    default:
      return "comfortable"
  }
}

export const normalizeAppSettings = (data: any): AppSettingsState => ({
  defaultAddr: String(data?.defaultAddr ?? "").trim() || defaultAppSettings().defaultAddr,
  defaultDeviceId: String(data?.defaultDeviceId ?? "").trim(),
  autoConnect: Boolean(data?.autoConnect),
  autoLogin: Boolean(data?.autoLogin),
  defaultStartPage: normalizeStartPage(data?.defaultStartPage),
  density: normalizeDensity(data?.density),
  reduceMotion: Boolean(data?.reduceMotion)
})

const normalizeAppAbout = (data: any): AppAboutState => ({
  appName: String(data?.appName ?? "").trim() || "MyFlowHub",
  appVersion: String(data?.appVersion ?? "").trim() || "dev",
  buildTime: String(data?.buildTime ?? "").trim() || "-",
  buildMode: String(data?.buildMode ?? "").trim() || "dev",
  commit: String(data?.commit ?? "").trim() || "-",
  platform: String(data?.platform ?? "").trim() || "-",
  goVersion: String(data?.goVersion ?? "").trim() || "-",
  wailsVersion: String(data?.wailsVersion ?? "").trim() || "unknown",
  profile: String(data?.profile ?? "").trim() || "default",
  baseDir: String(data?.baseDir ?? "").trim() || "-",
  settingsPath: String(data?.settingsPath ?? "").trim() || "-",
  keysPath: String(data?.keysPath ?? "").trim() || "-"
})

const readStartupMirror = (): StartupMirror => {
  const fallback = {
    defaultStartPage: defaultAppSettings().defaultStartPage,
    density: defaultAppSettings().density,
    reduceMotion: defaultAppSettings().reduceMotion
  } satisfies StartupMirror

  try {
    const raw = globalThis.localStorage?.getItem(startupMirrorStorageKey)
    if (!raw) return fallback
    const parsed = JSON.parse(raw)
    return {
      defaultStartPage: normalizeStartPage(parsed?.defaultStartPage),
      density: normalizeDensity(parsed?.density),
      reduceMotion: Boolean(parsed?.reduceMotion)
    }
  } catch {
    return fallback
  }
}

const writeStartupMirror = (settings: AppSettingsState) => {
  try {
    globalThis.localStorage?.setItem(
      startupMirrorStorageKey,
      JSON.stringify({
        defaultStartPage: normalizeStartPage(settings.defaultStartPage),
        density: normalizeDensity(settings.density),
        reduceMotion: Boolean(settings.reduceMotion)
      } satisfies StartupMirror)
    )
  } catch {
    // ignore localStorage write failures; backend state remains the source of truth
  }
}

export const resolveStartPagePath = (page: any) => startPagePathMap[normalizeStartPage(page)] ?? "/home"

export const readStartupRoutePath = () => resolveStartPagePath(readStartupMirror().defaultStartPage)

export const applyUIPreferences = (settings: Partial<AppSettingsState> | StartupMirror | null | undefined) => {
  if (typeof document === "undefined") return
  const root = document.documentElement
  root.dataset.uiDensity = normalizeDensity(settings?.density)
  root.dataset.uiMotion = settings?.reduceMotion ? "reduce" : "full"
}

export const applyStartupUIPreferences = () => {
  applyUIPreferences(readStartupMirror())
}

const state = reactive<AppSettingsStoreState>({
  settings: defaultAppSettings(),
  about: defaultAbout(),
  loaded: false,
  loading: false,
  saving: false,
  aboutLoading: false,
  updatedAt: "",
  aboutUpdatedAt: ""
})

const applySettingsState = (data: any) => {
  state.settings = normalizeAppSettings(data)
  state.loaded = true
  state.updatedAt = nowIso()
  writeStartupMirror(state.settings)
  applyUIPreferences(state.settings)
}

const applyAboutState = (data: any) => {
  state.about = normalizeAppAbout(data)
  state.aboutUpdatedAt = nowIso()
}

const load = async () => {
  state.loading = true
  try {
    const data = await callApp<any>("SettingsState")
    applySettingsState(data)
    return state.settings
  } finally {
    state.loading = false
  }
}

const save = async (input: AppSettingsState) => {
  state.saving = true
  try {
    const saved = await callApp<any>("SaveSettingsState", normalizeAppSettings(input))
    applySettingsState(saved)
    return state.settings
  } finally {
    state.saving = false
  }
}

const reset = async () => {
  state.saving = true
  try {
    const saved = await callApp<any>("ResetSettingsState")
    applySettingsState(saved)
    return state.settings
  } finally {
    state.saving = false
  }
}

const loadAbout = async () => {
  state.aboutLoading = true
  try {
    const data = await callApp<any>("AboutState")
    applyAboutState(data)
    return state.about
  } finally {
    state.aboutLoading = false
  }
}

export const useAppSettingsStore = () => {
  return {
    state,
    load,
    save,
    reset,
    loadAbout
  }
}
