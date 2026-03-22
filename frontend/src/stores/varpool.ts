import { t } from "@/i18n"
import { reactive } from "vue"
import { EventsOn } from "../../wailsjs/runtime/runtime"
type WailsBinding = (...args: any[]) => Promise<any>

const callApp = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.main?.App
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(t("App binding '{method}' unavailable", { method }))
  }
  return fn(...args)
}

const callVarPool = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.varpool?.VarPoolService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(t("VarPool binding '{method}' unavailable", { method }))
  }
  return fn(...args)
}

export type VarPoolKey = {
  name: string
  owner?: number
}

export type VarPoolSubPref = {
  name: string
  owner: number
  subscribed: boolean
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
let restorePromise: Promise<{ attempted: number; failed: number }> | null = null
let initialized = false

const nowIso = () => new Date().toISOString()

const keyId = (key: VarPoolKey) => `${key.name}#${key.owner ?? 0}`

const normalizeKey = (key: VarPoolKey): VarPoolKey => ({
  name: (key.name ?? "").trim(),
  owner: Number(key.owner ?? 0) || 0
})

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
  return stored ?? false
}

const ensureSubPrefDefaults = () => {
  for (const key of state.keys) {
    const normalized = normalizeKey(key)
    if (!normalized.name || !normalized.owner) continue
    if (state.selfNodeId && normalized.owner === state.selfNodeId) continue
    const id = keyId(normalized)
    if (!desiredSubs.has(id)) {
      desiredSubs.set(id, false)
    }
  }
}

const loadSubPrefs = async () => {
  const raw = await callApp<VarPoolSubPref[]>("VarPoolSubPrefs")
  desiredSubs.clear()
  const prefs = Array.isArray(raw) ? raw : []
  for (const pref of prefs) {
    const name = String((pref as any)?.name ?? "").trim()
    const owner = Number((pref as any)?.owner ?? 0) || 0
    if (!name || owner <= 0) continue
    desiredSubs.set(keyId({ name, owner }), Boolean((pref as any)?.subscribed))
  }
  ensureSubPrefDefaults()
}

const saveSubPrefs = async () => {
  ensureSubPrefDefaults()
  const payload: VarPoolSubPref[] = []
  for (const key of state.keys) {
    const normalized = normalizeKey(key)
    if (!normalized.name || !normalized.owner) continue
    if (state.selfNodeId && normalized.owner === state.selfNodeId) continue
    const id = keyId(normalized)
    payload.push({ name: normalized.name, owner: normalized.owner, subscribed: desiredSubs.get(id) ?? false })
  }
  const saved = await callApp<VarPoolSubPref[]>("SaveVarPoolSubPrefs", payload)
  desiredSubs.clear()
  const prefs = Array.isArray(saved) ? saved : []
  for (const pref of prefs) {
    const name = String((pref as any)?.name ?? "").trim()
    const owner = Number((pref as any)?.owner ?? 0) || 0
    if (!name || owner <= 0) continue
    desiredSubs.set(keyId({ name, owner }), Boolean((pref as any)?.subscribed))
  }
  ensureSubPrefDefaults()
}

const saveSubPrefsBestEffort = async () => {
  try {
    await saveSubPrefs()
  } catch (err) {
    console.warn(err)
  }
}

const saveWatchList = async () => {
  const filtered = normalizeKeys(
    state.keys.filter((key) => !isSelfKey(key))
  )
  await callApp("SaveVarPoolWatchList", filtered)
}

const loadWatchList = async () => {
  const keys = await callApp<VarPoolKey[]>("VarPoolWatchList")
  const normalized = normalizeKeys(Array.isArray(keys) ? keys : [])
  const keep = new Set<string>()
  for (const key of normalized) {
    const normalizedKey = normalizeKey(key)
    if (!normalizedKey.name) continue
    keep.add(keyId(normalizedKey))
  }
  for (const id of Object.keys(state.data)) {
    if (!keep.has(id)) {
      delete state.data[id]
      desiredSubs.delete(id)
    }
  }
  state.keys = normalized
  ensureSubPrefDefaults()
}

const resolveTargetId = () => {
  const raw = state.targetId.trim()
  if (!raw) return state.defaultTargetId
  const parsed = Number.parseInt(raw, 10)
  if (Number.isNaN(parsed) || parsed < 0) {
    throw new Error(t("Target ID must be a valid number."))
  }
  return parsed
}

const resolveHubTargetId = () => {
  const hubId = Number(state.defaultTargetId || 0)
  if (!hubId || hubId <= 0) {
    throw new Error(t("Hub target is unavailable. Re-login and retry."))
  }
  return hubId
}

const ensureSourceID = () => {
  if (!state.selfNodeId) {
    throw new Error(t("Login required to send VarPool requests."))
  }
  return state.selfNodeId
}

const listOwnerNames = async (ownerId: number) => {
  const sourceID = ensureSourceID()
  const targetID = resolveHubTargetId()
  const owner = Number(ownerId || 0)
  if (!owner || owner <= 0) {
    throw new Error(t("Owner NodeID is required."))
  }
  const resp = parseResp(await callVarPool<any>("ListSimple", sourceID, targetID, { owner }))
  if (resp.code === 4) {
    return [] as string[]
  }
  if (resp.code !== 1) {
    const msg = (resp.msg || "").trim()
    if (msg) {
      throw new Error(msg)
    }
    throw new Error(t("VarPool list failed (code={code})", { code: resp.code }))
  }
  const out: string[] = []
  const seen = new Set<string>()
  for (const name of resp.names || []) {
    const trimmed = String(name || "").trim()
    if (!trimmed) continue
    const key = trimmed
    if (seen.has(key)) continue
    seen.add(key)
    out.push(trimmed)
  }
  out.sort((a, b) => a.localeCompare(b))
  return out
}

const listMine = async () => {
  const sourceID = ensureSourceID()
  const targetID = resolveTargetId()
  const resp = parseResp(await callVarPool<any>("ListSimple", sourceID, targetID, { owner: sourceID }))
  handleVarListResp(resp)
}

const getVar = async (input: VarPoolKey) => {
  const sourceID = ensureSourceID()
  const targetID = resolveTargetId()
  const key = normalizeKey(input)
  const owner = key.owner || sourceID
  if (!key.name) throw new Error(t("Variable name is required."))
  if (!owner) throw new Error(t("Owner is required."))
  const resp = parseResp(await callVarPool<any>("GetSimple", sourceID, targetID, { name: key.name, owner }))
  handleVarChanged(resp)
}

const setVar = async (input: VarPoolKey, value: string, visibility: string, kind = "string") => {
  const sourceID = ensureSourceID()
  const targetID = resolveTargetId()
  const key = normalizeKey(input)
  const owner = key.owner || sourceID
  if (!key.name) throw new Error(t("Variable name is required."))
  if (!value.trim()) throw new Error(t("Variable value is required."))
  const resp = parseResp(await callVarPool<any>("SetSimple", sourceID, targetID, {
    name: key.name,
    value,
    visibility: visibility || "public",
    type: kind,
    owner
  }))
  handleVarChanged({
    ...resp,
    value: resp.value || value,
    visibility: resp.visibility || visibility || "public",
    type: resp.type || kind
  })
}

const revokeVar = async (input: VarPoolKey) => {
  const sourceID = ensureSourceID()
  const targetID = resolveTargetId()
  const key = normalizeKey(input)
  const owner = key.owner || sourceID
  if (!key.name) throw new Error(t("Variable name is required."))
  if (!owner) throw new Error(t("Owner is required."))
  const resp = parseResp(await callVarPool<any>("RevokeSimple", sourceID, targetID, { name: key.name, owner }))
  handleVarRevokeResp("revoke_resp", resp)
}

const subscribeVar = async (input: VarPoolKey) => {
  const sourceID = ensureSourceID()
  const targetID = resolveTargetId()
  const key = normalizeKey(input)
  const owner = key.owner || sourceID
  if (!key.name) throw new Error(t("Variable name is required."))
  if (!owner) throw new Error(t("Owner is required."))
  const desiredKey = { name: key.name, owner }
  upsertKey(desiredKey)
  setDesiredSubscribe(desiredKey, true)
  if (desiredKey.owner !== key.owner) {
    setDesiredSubscribe(key, true)
  }
  await saveSubPrefsBestEffort()
  const resp = parseResp(await callVarPool<any>("SubscribeSimple", sourceID, targetID, {
    name: key.name,
    owner,
    subscriber: sourceID
  }))
  handleVarSubscribeResp(resp)
}

const unsubscribeVar = async (input: VarPoolKey) => {
  const sourceID = ensureSourceID()
  const targetID = resolveTargetId()
  const key = normalizeKey(input)
  const owner = key.owner || sourceID
  if (!key.name) throw new Error(t("Variable name is required."))
  if (!owner) throw new Error(t("Owner is required."))
  const desiredKey = { name: key.name, owner }
  upsertKey(desiredKey)
  setDesiredSubscribe(desiredKey, false)
  if (desiredKey.owner !== key.owner) {
    setDesiredSubscribe(key, false)
  }
  updateValue(desiredKey, { subKnown: true, subscribed: false })
  if (desiredKey.owner !== key.owner) {
    updateValue(key, { subKnown: true, subscribed: false })
  }
  await saveSubPrefsBestEffort()
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
    setDesiredSubscribe(key, false)
    await saveWatchList()
    await saveSubPrefsBestEffort()
  }
}

const removeWatchKey = async (input: VarPoolKey) => {
  const key = normalizeKey(input)
  if (!key.name) return
  removeLocalKey(key)
  if (!isSelfKey(key)) {
    await saveWatchList()
    await saveSubPrefsBestEffort()
  }
}

const restoreDesiredSubscriptions = async (options?: { concurrency?: number }) => {
  if (restorePromise) return restorePromise
  const concurrency = Math.max(1, Number(options?.concurrency ?? 4) || 4)

  restorePromise = (async () => {
    ensureSubPrefDefaults()
    let sourceID = 0
    let targetID = 0
    try {
      sourceID = ensureSourceID()
      targetID = resolveTargetId()
    } catch (err) {
      console.warn(err)
      return { attempted: 0, failed: 0 }
    }
    if (!targetID) return { attempted: 0, failed: 0 }

    const pending = state.keys
      .map((key) => normalizeKey(key))
      .filter((key) => {
        if (!key.name || !key.owner) return false
        if (state.selfNodeId && key.owner === state.selfNodeId) return false
        return desiredSubs.get(keyId(key)) === true
      })

    if (!pending.length) {
      return { attempted: 0, failed: 0 }
    }

    let cursor = 0
    let failed = 0
    const workers = Array.from({ length: Math.min(concurrency, pending.length) }, async () => {
      while (true) {
        const idx = cursor
        cursor += 1
        const key = pending[idx]
        if (!key) return
        try {
          const resp = parseResp(await callVarPool<any>("SubscribeSimple", sourceID, targetID, {
            name: key.name,
            owner: key.owner,
            subscriber: sourceID
          }))
          handleVarSubscribeResp(resp)
        } catch (err) {
          console.warn(err)
          failed += 1
        }
      }
    })

    await Promise.all(workers)
    return { attempted: pending.length, failed }
  })()

  try {
    return await restorePromise
  } finally {
    restorePromise = null
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
    void callVarPool<any>("GetSimple", state.selfNodeId, targetID, {
      name: trimmed,
      owner: resp.owner
    })
      .then((raw) => handleVarChanged(parseResp(raw)))
      .catch(() => {})
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

const ensureListeners = () => {
  if (initialized) return
  initialized = true
  EventsOn("varpool.changed", (evt: any) => {
    state.lastFrameAt = nowIso()
    handleVarChanged(parseResp(evt))
  })
  EventsOn("varpool.deleted", (evt: any) => {
    state.lastFrameAt = nowIso()
    handleVarDeleted(parseResp(evt))
  })
}

export const useVarPoolStore = () => {
  ensureListeners()
  return {
    state,
    addWatchKey,
    getVar,
    listOwnerNames,
    listMine,
    loadSubPrefs,
    loadWatchList,
    removeWatchKey,
    resolveTargetId,
    revokeVar,
    restoreDesiredSubscriptions,
    saveSubPrefs,
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
