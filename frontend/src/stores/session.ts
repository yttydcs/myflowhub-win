import { reactive } from "vue"
import { EventsOn } from "../../wailsjs/runtime/runtime"

export type AuthSnapshot = {
  deviceId: string
  nodeId: number
  hubId: number
  role: string
  loggedIn: boolean
  lastAuthMessage: string
  lastAuthAction: string
  lastAuthAt: string
}

export type SessionSnapshot = {
  connected: boolean
  addr: string
  lastStateAt: string
  lastError: string
  lastErrorAt: string
  lastFrameAt: string
  auth: AuthSnapshot
}

const emptyAuth: AuthSnapshot = {
  deviceId: "",
  nodeId: 0,
  hubId: 0,
  role: "",
  loggedIn: false,
  lastAuthMessage: "",
  lastAuthAction: "",
  lastAuthAt: ""
}

const store = reactive<SessionSnapshot>({
  connected: false,
  addr: "",
  lastStateAt: "",
  lastError: "",
  lastErrorAt: "",
  lastFrameAt: "",
  auth: { ...emptyAuth }
})

let initialized = false

const nowIso = () => new Date().toISOString()

const ensureListeners = () => {
  if (initialized) return
  initialized = true

  EventsOn("session.state", (evt: any) => {
    store.connected = Boolean(evt?.connected)
    store.addr = String(evt?.addr ?? "")
    store.lastStateAt = nowIso()
    if (!store.connected) {
      store.auth.loggedIn = false
    }
  })

  EventsOn("session.error", (evt: any) => {
    store.lastError = String(evt?.message ?? "")
    store.lastErrorAt = nowIso()
  })

  EventsOn("session.frame", (evt: any) => {
    store.lastFrameAt = nowIso()
  })
}

export const useSessionStore = () => {
  ensureListeners()
  return store
}
