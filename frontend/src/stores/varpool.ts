import { reactive } from "vue"
import { EventsOn } from "../../wailsjs/runtime/runtime"
type WailsBinding = (...args: any[]) => Promise<any>

const callApp = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.main?.App
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(`App binding '${method}' unavailable`)
  }
  return fn(...args)
}

const callVarPool = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.varpool?.VarPoolService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(`VarPool binding '${method}' unavailable`)
  }
  return fn(...args)
}

export type VarPoolKey = {
  name: string
  owner?: number
}

export type VarPoolValue = {
  value: string
  owner: number
  visibility: string
  kind: string
  subscribed: boolean
  subKnown: boolean
  lastUpdated: string
}

export type VarPoolState = {
  targetId: string
  selfNodeId: number
  defaultTargetId: number
  keys: VarPoolKey[]
  data: Record<string, VarPoolValue>
  lastFrameAt: string
}

const state = reactive<VarPoolState>({
  targetId: "",
  selfNodeId: 0,
  defaultTargetId: 0,
  keys: [],
  data: {},
  lastFrameAt: ""
})

const desiredSubs = new Map<string, boolean>()
let initialized = false

const nowIso = () => new Date().toISOString()

const keyId = (key: VarPoolKey) => `${key.name}#${key.owner ?? 0}`

const normalizeKey = (key: VarPoolKey): VarPoolKey => ({
  name: (key.name ?? "").trim(),
  owner: Number(key.owner ?? 0) || 0
})

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

const normalizeKeys = (keys: VarPoolKey[]) => {
  const out: VarPoolKey[] = []
  for (const raw of keys) {
    const key = normalizeKey(raw)
    if (!key.name) continue
    let replaced = false
    for (let i = 0; i < out.length; i += 1) {
      const existing = out[i]
      if (existing.name === key.name && (existing.owner ?? 0) === (key.owner ?? 0)) {
        replaced = true
        break
      }
      if (existing.name === key.name && (existing.owner ?? 0) === 0 && (key.owner ?? 0) !== 0) {
        out[i] = key
        replaced = true
        break
      }
    }
    if (!replaced) {
      out.push(key)
    }
  }
  return out
}

const upsertKey = (input: VarPoolKey) => {
  const key = normalizeKey(input)
  if (!key.name) return { key, changed: false }
  for (let i = 0; i < state.keys.length; i += 1) {
    const existing = state.keys[i]
    if (existing.name === key.name && (existing.owner ?? 0) === (key.owner ?? 0)) {
      return { key: existing, changed: false }
    }
    if (existing.name === key.name && (existing.owner ?? 0) === 0 && (key.owner ?? 0) !== 0) {
      state.keys[i] = key
      return { key, changed: true }
    }
  }
  state.keys.push(key)
  return { key, changed: true }
}

const isSelfKey = (key: VarPoolKey) =>
  state.selfNodeId > 0 && Number(key.owner ?? 0) === state.selfNodeId

const valueForKey = (key: VarPoolKey): VarPoolValue => {
  const id = keyId(normalizeKey(key))
  const existing = state.data[id]
  if (existing) return existing
  return {
    value: "",
    owner: Number(key.owner ?? 0) || 0,
    visibility: "",
    kind: "",
    subscribed: false,
    subKnown: false,
    lastUpdated: ""
  }
}

const updateValue = (input: VarPoolKey, patch: Partial<VarPoolValue>) => {
  const { key } = upsertKey(input)
  if (!key.name) return
  const id = keyId(key)
  const existing = state.data[id] ?? valueForKey(key)
  const merged: VarPoolValue = { ...existing }
  if (patch.value !== undefined) merged.value = patch.value
  if (patch.owner !== undefined && patch.owner !== 0) merged.owner = patch.owner
  if (patch.visibility !== undefined && patch.visibility !== "") merged.visibility = patch.visibility
  if (patch.kind !== undefined && patch.kind !== "") merged.kind = patch.kind
  if (patch.subKnown !== undefined) {
    merged.subKnown = patch.subKnown
    if (patch.subscribed !== undefined) {
      merged.subscribed = patch.subscribed
    }
  } else if (patch.subscribed !== undefined && merged.subKnown) {
    merged.subscribed = patch.subscribed
  }
  merged.lastUpdated = nowIso()
  state.data[id] = merged
}

const removeLocalKey = (input: VarPoolKey) => {
  const key = normalizeKey(input)
  if (!key.name) return
  const id = keyId(key)
  state.keys = state.keys.filter(
    (item) => !(item.name === key.name && (item.owner ?? 0) === (key.owner ?? 0))
  )
  delete state.data[id]
  desiredSubs.delete(id)
}

const setDesiredSubscribe = (key: VarPoolKey, desired: boolean) => {
  const normalized = normalizeKey(key)
  if (!normalized.name) return
  desiredSubs.set(keyId(normalized), desired)
}

const desiredSubscribe = (key: VarPoolKey) => {
  const normalized = normalizeKey(key)
  if (!normalized.name) return true
  const stored = desiredSubs.get(keyId(normalized))
  return stored ?? true
}

const saveWatchList = async () => {
  const filtered = normalizeKeys(
    state.keys.filter((key) => !isSelfKey(key))
  )
  await callApp("SaveVarPoolWatchList", filtered)
}

const loadWatchList = async () => {
  const keys = await callApp<VarPoolKey[]>("VarPoolWatchList")
  state.keys = normalizeKeys(Array.isArray(keys) ? keys : [])
  state.data = {}
  desiredSubs.clear()
}

const resolveTargetId = () => {
  const raw = state.targetId.trim()
  if (!raw) return state.defaultTargetId
  const parsed = Number.parseInt(raw, 10)
  if (Number.isNaN(parsed) || parsed < 0) {
    throw new Error("Target ID must be a valid number.")
  }
  return parsed
}

const ensureSourceID = () => {
  if (!state.selfNodeId) {
    throw new Error("Login required to send VarPool requests.")
  }
  return state.selfNodeId
}

const listMine = async () => {
  const sourceID = ensureSourceID()
  const targetID = resolveTargetId()
  await callVarPool("ListSimple", sourceID, targetID, { owner: sourceID })
}

const getVar = async (input: VarPoolKey) => {
  const sourceID = ensureSourceID()
  const targetID = resolveTargetId()
  const key = normalizeKey(input)
  const owner = key.owner || sourceID
  if (!key.name) throw new Error("Variable name is required.")
  if (!owner) throw new Error("Owner is required.")
  await callVarPool("GetSimple", sourceID, targetID, { name: key.name, owner })
}

const setVar = async (input: VarPoolKey, value: string, visibility: string, kind = "string") => {
  const sourceID = ensureSourceID()
  const targetID = resolveTargetId()
  const key = normalizeKey(input)
  const owner = key.owner || sourceID
  if (!key.name) throw new Error("Variable name is required.")
  if (!value.trim()) throw new Error("Variable value is required.")
  await callVarPool("SetSimple", sourceID, targetID, {
    name: key.name,
    value,
    visibility: visibility || "public",
    type: kind,
    owner
  })
  updateValue(
    { name: key.name, owner },
    { value, owner, visibility: visibility || "public", kind }
  )
}

const revokeVar = async (input: VarPoolKey) => {
  const sourceID = ensureSourceID()
  const targetID = resolveTargetId()
  const key = normalizeKey(input)
  const owner = key.owner || sourceID
  if (!key.name) throw new Error("Variable name is required.")
  if (!owner) throw new Error("Owner is required.")
  await callVarPool("RevokeSimple", sourceID, targetID, { name: key.name, owner })
}

const subscribeVar = async (input: VarPoolKey) => {
  const sourceID = ensureSourceID()
  const targetID = resolveTargetId()
  const key = normalizeKey(input)
  const owner = key.owner || sourceID
  if (!key.name) throw new Error("Variable name is required.")
  if (!owner) throw new Error("Owner is required.")
  const desiredKey = { name: key.name, owner }
  setDesiredSubscribe(desiredKey, true)
  if (desiredKey.owner !== key.owner) {
    setDesiredSubscribe(key, true)
  }
  await callVarPool("SubscribeSimple", sourceID, targetID, {
    name: key.name,
    owner,
    subscriber: sourceID
  })
}

const unsubscribeVar = async (input: VarPoolKey) => {
  const sourceID = ensureSourceID()
  const targetID = resolveTargetId()
  const key = normalizeKey(input)
  const owner = key.owner || sourceID
  if (!key.name) throw new Error("Variable name is required.")
  if (!owner) throw new Error("Owner is required.")
  const desiredKey = { name: key.name, owner }
  setDesiredSubscribe(desiredKey, false)
  if (desiredKey.owner !== key.owner) {
    setDesiredSubscribe(key, false)
  }
  updateValue(desiredKey, { subKnown: true, subscribed: false })
  if (desiredKey.owner !== key.owner) {
    updateValue(key, { subKnown: true, subscribed: false })
  }
  await callVarPool("UnsubscribeSimple", sourceID, targetID, {
    name: key.name,
    owner,
    subscriber: sourceID
  })
}

const addWatchKey = async (input: VarPoolKey) => {
  const { key, changed } = upsertKey(input)
  if (!key.name) return
  if (changed && !isSelfKey(key)) {
    await saveWatchList()
  }
}

const removeWatchKey = async (input: VarPoolKey) => {
  const key = normalizeKey(input)
  if (!key.name) return
  removeLocalKey(key)
  if (!isSelfKey(key)) {
    await saveWatchList()
  }
}

type VarResp = {
  code: number
  msg: string
  name: string
  value: string
  owner: number
  visibility: string
  type: string
  names: string[]
}

const parseResp = (payload: any): VarResp => ({
  code: Number(payload?.code ?? 0),
  msg: String(payload?.msg ?? ""),
  name: String(payload?.name ?? ""),
  value: String(payload?.value ?? ""),
  owner: Number(payload?.owner ?? 0),
  visibility: String(payload?.visibility ?? ""),
  type: String(payload?.type ?? ""),
  names: Array.isArray(payload?.names) ? payload.names.map((name: any) => String(name)) : []
})

const handleVarListResp = (resp: VarResp) => {
  if (resp.code !== 1 || !resp.owner) {
    return
  }
  if (state.selfNodeId && resp.owner !== state.selfNodeId) {
    return
  }
  const filtered = state.keys.filter((key) => Number(key.owner ?? 0) !== resp.owner)
  for (const name of resp.names) {
    const trimmed = String(name).trim()
    if (!trimmed) continue
    filtered.push({ name: trimmed, owner: resp.owner })
  }
  state.keys = filtered
  if (!state.selfNodeId) return
  let targetID = 0
  try {
    targetID = resolveTargetId()
  } catch {
    return
  }
  if (targetID < 0) return
  for (const name of resp.names) {
    const trimmed = String(name).trim()
    if (!trimmed) continue
    void callVarPool("GetSimple", state.selfNodeId, targetID, {
      name: trimmed,
      owner: resp.owner
    }).catch(() => {})
  }
}

const handleVarRevokeResp = (action: string, resp: VarResp) => {
  const name = resp.name.trim()
  if (!name) return
  if (action !== "notify_revoke" && resp.code !== 1) {
    return
  }
  removeLocalKey({ name, owner: resp.owner })
}

const handleVarSubscribeResp = (resp: VarResp) => {
  const name = resp.name.trim()
  if (!name) return
  const key = { name, owner: resp.owner }
  if (!desiredSubscribe(key)) {
    return
  }
  if (resp.code !== 1) {
    return
  }
  updateValue(key, {
    value: resp.value,
    owner: resp.owner,
    visibility: resp.visibility,
    kind: resp.type,
    subscribed: true,
    subKnown: true
  })
}

const handleVarChanged = (resp: VarResp) => {
  const name = resp.name.trim()
  if (!name || !resp.owner) return
  updateValue(
    { name, owner: resp.owner },
    { value: resp.value, owner: resp.owner, visibility: resp.visibility, kind: resp.type }
  )
}

const handleVarDeleted = (resp: VarResp) => {
  const name = resp.name.trim()
  if (!name || !resp.owner) return
  removeLocalKey({ name, owner: resp.owner })
}

const handleFrame = (payload: any) => {
  const text = decodePayloadText(payload)
  if (!text) return
  let message: any
  try {
    message = JSON.parse(text)
  } catch {
    return
  }
  const action = String(message?.action ?? "").toLowerCase()
  if (!action) return
  let data: any = message?.data ?? {}
  if (typeof data === "string") {
    try {
      data = JSON.parse(data)
    } catch {
      data = {}
    }
  }
  const resp = parseResp(data)
  switch (action) {
    case "list_resp":
    case "assist_list_resp":
      handleVarListResp(resp)
      break
    case "get_resp":
    case "assist_get_resp":
    case "notify_set":
    case "set_resp":
    case "assist_set_resp": {
      const name = resp.name.trim()
      if (!name) return
      if (
        (action === "get_resp" ||
          action === "assist_get_resp" ||
          action === "set_resp" ||
          action === "assist_set_resp") &&
        resp.code !== 1
      ) {
        if (resp.owner && resp.owner === state.selfNodeId) {
          removeLocalKey({ name, owner: resp.owner })
        }
        return
      }
      updateValue(
        { name: resp.name, owner: resp.owner },
        { value: resp.value, owner: resp.owner, visibility: resp.visibility, kind: resp.type }
      )
      break
    }
    case "revoke_resp":
    case "assist_revoke_resp":
    case "notify_revoke":
      handleVarRevokeResp(action, resp)
      break
    case "subscribe_resp":
    case "assist_subscribe_resp":
      handleVarSubscribeResp(resp)
      break
    case "var_changed":
      handleVarChanged(resp)
      break
    case "var_deleted":
      handleVarDeleted(resp)
      break
    default:
      break
  }
}

const ensureListeners = () => {
  if (initialized) return
  initialized = true
  EventsOn("session.frame", (evt: any) => {
    state.lastFrameAt = nowIso()
    const subProto = Number(evt?.sub_proto ?? evt?.subProto ?? 0)
    if (subProto === 3) {
      handleFrame(evt?.payload)
    }
  })
}

export const useVarPoolStore = () => {
  ensureListeners()
  return {
    state,
    addWatchKey,
    getVar,
    listMine,
    loadWatchList,
    removeWatchKey,
    resolveTargetId,
    revokeVar,
    saveWatchList,
    setIdentity: (nodeId: number, hubId: number) => {
      state.selfNodeId = Number(nodeId || 0)
      state.defaultTargetId = Number(hubId || 0)
      if (!state.targetId && state.defaultTargetId) {
        state.targetId = String(state.defaultTargetId)
      }
    },
    setVar,
    subscribeVar,
    unsubscribeVar,
    updateValue,
    valueForKey
  }
}
