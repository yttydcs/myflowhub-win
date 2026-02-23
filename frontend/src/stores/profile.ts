import { reactive } from "vue"
import { ProfileState, SetCurrentProfile } from "../../wailsjs/go/main/App"
import { useToastStore } from "@/stores/toast"

export type ProfileStoreState = {
  profiles: string[]
  current: string
  baseDir: string
  settingsPath: string
  keysPath: string
  loading: boolean
  updatedAt: string
}

const state = reactive<ProfileStoreState>({
  profiles: ["default"],
  current: "default",
  baseDir: "",
  settingsPath: "",
  keysPath: "",
  loading: false,
  updatedAt: ""
})

let initialized = false

const nowIso = () => new Date().toISOString()

const toast = useToastStore()

const applyProfileState = (data: any) => {
  if (!data) return
  state.profiles = Array.isArray(data.profiles) && data.profiles.length ? data.profiles : ["default"]
  state.current = data.current || state.profiles[0]
  state.baseDir = data.baseDir || ""
  state.settingsPath = data.settingsPath || ""
  state.keysPath = data.keysPath || ""
  state.updatedAt = nowIso()
}

const loadProfileState = async () => {
  state.loading = true
  try {
    const data = await ProfileState()
    applyProfileState(data)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to load profile state.")
  } finally {
    state.loading = false
  }
}

const setProfile = async (name: string) => {
  const trimmed = name.trim()
  if (!trimmed) return
  state.loading = true
  try {
    const data = await SetCurrentProfile(trimmed)
    applyProfileState(data)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Unable to switch profile.")
  } finally {
    state.loading = false
  }
}

export const useProfileStore = () => {
  if (!initialized) {
    initialized = true
    void loadProfileState()
  }
  return {
    state,
    loadProfileState,
    setProfile
  }
}
