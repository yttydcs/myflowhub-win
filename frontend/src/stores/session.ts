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

const toByteArray = (payload: any): Uint8Array | null => {
  if (!payload) return null
  if (payload instanceof Uint8Array) return payload
  if (payload instanceof ArrayBuffer) return new Uint8Array(payload)
  if (Array.isArray(payload)) return new Uint8Array(payload)
  if (payload && typeof payload === "object" && Array.isArray(payload.data)) {
    return new Uint8Array(payload.data)
  }
  return null
}

const decodePayloadText = (payload: any): string | null => {
  const bytes = toByteArray(payload)
  if (bytes) {
    return new TextDecoder().decode(bytes)
  }
  if (typeof payload === "string") {
    const trimmed = payload.trim()
    if (trimmed.startsWith("{") || trimmed.startsWith("[")) {
      return payload
    }
    try {
      return atob(trimmed)
    } catch {
      return payload
    }
  }
  return null
}

const parseAuthFrame = (payload: any) => {
  const text = decodePayloadText(payload)
  if (!text) return
  let message: any
  try {
    message = JSON.parse(text)
  } catch {
    return
  }
  const action = String(message?.action ?? "").toLowerCase()
  if (!action || (!action.endsWith("login_resp") && !action.endsWith("register_resp"))) {
    return
  }
  let data: any = message.data ?? {}
  if (typeof data === "string") {
    try {
      data = JSON.parse(data)
    } catch {
      data = {}
    }
  }
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  store.auth.lastAuthAction = action
  store.auth.lastAuthMessage = msg || (code === 1 ? "OK" : "Auth failed")
  store.auth.lastAuthAt = nowIso()
  if (code !== 1) {
    return
  }
  store.auth.loggedIn = true
  if (data?.device_id) store.auth.deviceId = String(data.device_id)
  if (typeof data?.node_id === "number") store.auth.nodeId = data.node_id
  if (typeof data?.hub_id === "number") store.auth.hubId = data.hub_id
  if (typeof data?.role === "string") store.auth.role = data.role
}

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
    const subProto = Number(evt?.sub_proto ?? evt?.subProto ?? 0)
    if (subProto === 2) {
      parseAuthFrame(evt?.payload)
    }
  })
}

export const useSessionStore = () => {
  ensureListeners()
  return store
}
