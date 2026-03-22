import { reactive } from "vue"
import { defaultLocale, normalizeLocale, setLocale, type AppLocale } from "@/i18n"

type WailsBinding = (...args: any[]) => Promise<any>

export type GlobalPreferencesState = {
  language: AppLocale
}

type LanguageStoreState = {
  preferences: GlobalPreferencesState
  loaded: boolean
  loading: boolean
  saving: boolean
  updatedAt: string
}

const startupLanguageStorageKey = "myflowhub.global_preferences.language"

const callApp = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.main?.App
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(`App binding '${method}' unavailable`)
  }
  return fn(...args)
}

const nowIso = () => new Date().toISOString()

export const defaultGlobalPreferences = (): GlobalPreferencesState => ({
  language: defaultLocale
})

export const normalizeGlobalPreferences = (data: any): GlobalPreferencesState => ({
  language: normalizeLocale(data?.language)
})

const readStartupLanguage = (): AppLocale => {
  try {
    return normalizeLocale(globalThis.localStorage?.getItem(startupLanguageStorageKey))
  } catch {
    return defaultLocale
  }
}

const writeStartupLanguage = (language: AppLocale) => {
  try {
    globalThis.localStorage?.setItem(startupLanguageStorageKey, language)
  } catch {
    // ignore localStorage write failures; backend state remains the source of truth
  }
}

export const applyStartupLocale = () => {
  setLocale(readStartupLanguage())
}

export const languageOptions: { value: AppLocale; label: string }[] = [
  { value: "en", label: "English" },
  { value: "zh-CN", label: "Chinese (Simplified)" }
]

const state = reactive<LanguageStoreState>({
  preferences: defaultGlobalPreferences(),
  loaded: false,
  loading: false,
  saving: false,
  updatedAt: ""
})

const applyGlobalPreferencesState = (data: any) => {
  state.preferences = normalizeGlobalPreferences(data)
  state.loaded = true
  state.updatedAt = nowIso()
  writeStartupLanguage(state.preferences.language)
  setLocale(state.preferences.language)
}

const load = async () => {
  state.loading = true
  try {
    const data = await callApp<any>("GlobalPreferencesState")
    applyGlobalPreferencesState(data)
    return state.preferences
  } finally {
    state.loading = false
  }
}

const save = async (input: GlobalPreferencesState | AppLocale) => {
  state.saving = true
  try {
    const payload =
      typeof input === "string"
        ? { language: normalizeLocale(input) }
        : normalizeGlobalPreferences(input)
    const data = await callApp<any>("SaveGlobalPreferencesState", payload)
    applyGlobalPreferencesState(data)
    return state.preferences
  } finally {
    state.saving = false
  }
}

const reset = async () => {
  state.saving = true
  try {
    const data = await callApp<any>("ResetGlobalPreferencesState")
    applyGlobalPreferencesState(data)
    return state.preferences
  } finally {
    state.saving = false
  }
}

export const useLanguageStore = () => {
  return {
    state,
    load,
    save,
    reset
  }
}
