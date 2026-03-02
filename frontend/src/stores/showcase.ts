import { reactive } from "vue"
import { EventsOn } from "../../wailsjs/runtime/runtime"
import { useToastStore } from "@/stores/toast"

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

const callTopicBus = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.topicbus?.TopicBusService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(`TopicBus binding '${method}' unavailable`)
  }
  return fn(...args)
}

export type ShowcaseWidgetKind = "topic_button" | "var"
export type VarWidgetMode = "auto" | "display" | "slider" | "switch"

export type ShowcaseVarSlider = {
  min: number
  max: number
  step: number
  throttleMs: number
}

export type ShowcaseVarSwitch = {
  onValue: string
  offValue: string
}

export type ShowcaseVarWidget = {
  ownerId: number
  name: string
  mode: VarWidgetMode
  visibility: string
  type: string
  slider: ShowcaseVarSlider
  switch: ShowcaseVarSwitch
}

export type ShowcaseTopicButton = {
  topic: string
  name: string
  payloadText: string
}

export type ShowcaseWidget = {
  id: string
  kind: ShowcaseWidgetKind
  title: string
  targetId: number
  topicButton?: ShowcaseTopicButton
  var?: ShowcaseVarWidget
}

export type ShowcaseScreen = {
  id: string
  name: string
  widgets: ShowcaseWidget[]
}

export type ShowcaseConfig = {
  version: number
  currentScreenId: string
  screens: ShowcaseScreen[]
}

export type VarSnapshot = {
  ownerId: number
  name: string
  value: string
  visibility: string
  type: string
  lastUpdated: string
}

export type ShowcaseState = {
  loaded: boolean
  busy: boolean
  lastLoadedAt: string
  selfNodeId: number
  hubId: number
  config: ShowcaseConfig
  values: Record<string, VarSnapshot>
  lastFrameAt: string
  sliderDraft: Record<string, number>
}

const nowIso = () => new Date().toISOString()

const newId = () => {
  const uuid = (globalThis as any)?.crypto?.randomUUID?.()
  if (uuid) return String(uuid)
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

const varKey = (ownerId: number, name: string) => `${ownerId}:${name}`
const subKey = (targetId: number, ownerId: number, name: string) => `${targetId}:${ownerId}:${name}`

const defaultSlider = (): ShowcaseVarSlider => ({ min: 0, max: 100, step: 1, throttleMs: 50 })
const defaultSwitch = (): ShowcaseVarSwitch => ({ onValue: "true", offValue: "false" })
const defaultTypeForMode = (mode: VarWidgetMode): string => {
  switch (mode) {
    case "slider":
      return "float64"
    case "switch":
      return "bool"
    default:
      return "string"
  }
}

const emptyConfig = (): ShowcaseConfig => ({
  version: 1,
  currentScreenId: "default",
  screens: [{ id: "default", name: "Default", widgets: [] }]
})

const normalizeVarMode = (mode: any): VarWidgetMode => {
  const raw = String(mode ?? "").trim().toLowerCase()
  switch (raw) {
    case "":
    case "auto":
      return "auto"
    case "display":
      return "display"
    case "slider":
      return "slider"
    case "switch":
      return "switch"
    default:
      return "auto"
  }
}

const normalizeSlider = (raw: any): ShowcaseVarSlider => {
  const out: ShowcaseVarSlider = {
    min: Number(raw?.min ?? 0),
    max: Number(raw?.max ?? 100),
    step: Number(raw?.step ?? 1),
    throttleMs: Number(raw?.throttleMs ?? 50)
  }
  if (!Number.isFinite(out.min)) out.min = 0
  if (!Number.isFinite(out.max)) out.max = 100
  if (out.max <= out.min) {
    out.min = 0
    out.max = 100
  }
  if (!Number.isFinite(out.step) || out.step <= 0) out.step = 1
  if (!Number.isFinite(out.throttleMs) || out.throttleMs < 0) out.throttleMs = 50
  return out
}

const normalizeSwitch = (raw: any): ShowcaseVarSwitch => {
  const onValue = String(raw?.onValue ?? "").trim() || "true"
  const offValue = String(raw?.offValue ?? "").trim() || "false"
  return { onValue, offValue }
}

const normalizeVarWidget = (raw: any): ShowcaseVarWidget | null => {
  const name = String(raw?.name ?? "").trim()
  const ownerId = Number(raw?.ownerId ?? 0)
  if (!name || !Number.isFinite(ownerId) || ownerId <= 0) return null
  const mode = normalizeVarMode(raw?.mode)
  const type = String(raw?.type ?? "").trim() || defaultTypeForMode(mode)
  return {
    ownerId,
    name,
    mode,
    visibility: String(raw?.visibility ?? "public").trim() || "public",
    type,
    slider: normalizeSlider(raw?.slider ?? defaultSlider()),
    switch: normalizeSwitch(raw?.switch ?? defaultSwitch())
  }
}

const normalizeTopicButton = (raw: any): ShowcaseTopicButton | null => {
  const topic = String(raw?.topic ?? "").trim()
  const name = String(raw?.name ?? "").trim()
  if (!topic || !name) return null
  return {
    topic,
    name,
    payloadText: String(raw?.payloadText ?? "").trim()
  }
}

const normalizeWidget = (raw: any): ShowcaseWidget | null => {
  const id = String(raw?.id ?? "").trim() || newId()
  const kind = String(raw?.kind ?? "").trim() as ShowcaseWidgetKind
  const title = String(raw?.title ?? "").trim()
  let targetId = Number(raw?.targetId ?? 0)
  if (!Number.isFinite(targetId) || targetId <= 0) targetId = 1

  if (kind === "topic_button") {
    const topicButton = normalizeTopicButton(raw?.topicButton)
    if (!topicButton) return null
    return { id, kind, title, targetId, topicButton }
  }
  if (kind === "var") {
    const v = normalizeVarWidget(raw?.var)
    if (!v) return null
    return { id, kind, title, targetId, var: v }
  }
  return null
}

const normalizeScreen = (raw: any): ShowcaseScreen | null => {
  const id = String(raw?.id ?? "").trim() || newId()
  const name = String(raw?.name ?? "").trim()
  if (!name) return null
  const widgets: ShowcaseWidget[] = []
  const seen = new Set<string>()
  const list = Array.isArray(raw?.widgets) ? raw.widgets : []
  for (const item of list) {
    const widget = normalizeWidget(item)
    if (!widget) continue
    if (seen.has(widget.id)) continue
    seen.add(widget.id)
    widgets.push(widget)
  }
  return { id, name, widgets }
}

const normalizeConfig = (raw: any): ShowcaseConfig => {
  const version = Number(raw?.version ?? 1)
  const screens: ShowcaseScreen[] = []
  const seen = new Set<string>()
  const list = Array.isArray(raw?.screens) ? raw.screens : []
  for (const item of list) {
    const screen = normalizeScreen(item)
    if (!screen) continue
    if (seen.has(screen.id)) continue
    seen.add(screen.id)
    screens.push(screen)
  }
  if (!screens.length) {
    return emptyConfig()
  }
  const currentScreenId = String(raw?.currentScreenId ?? "").trim()
  const resolved =
    currentScreenId && screens.some((s) => s.id === currentScreenId)
      ? currentScreenId
      : screens[0].id
  return {
    version: Number.isFinite(version) && version > 0 ? Math.floor(version) : 1,
    currentScreenId: resolved,
    screens
  }
}

const state = reactive<ShowcaseState>({
  loaded: false,
  busy: false,
  lastLoadedAt: "",
  selfNodeId: 0,
  hubId: 0,
  config: emptyConfig(),
  values: {},
  lastFrameAt: "",
  sliderDraft: {}
})

const activeSubs = new Set<string>()
const sliderTimers = new Map<string, number>()
const sliderLastSentAt = new Map<string, number>()
let initialized = false

const toast = useToastStore()

const ensureReady = () => {
  if (!state.selfNodeId) {
    throw new Error("Login required.")
  }
  return state.selfNodeId
}

const resolveTarget = (widgetTargetId: number) => {
  if (Number.isFinite(widgetTargetId) && widgetTargetId > 0) {
    return Math.floor(widgetTargetId)
  }
  throw new Error("Target ID is required.")
}

const parseVarResp = (payload: any) => {
  const owner = Number(payload?.owner ?? 0)
  const name = String(payload?.name ?? "").trim()
  if (!Number.isFinite(owner) || owner <= 0 || !name) return null
  return {
    code: Number(payload?.code ?? 0),
    msg: String(payload?.msg ?? "").trim(),
    ownerId: owner,
    name,
    value: String(payload?.value ?? ""),
    visibility: String(payload?.visibility ?? ""),
    type: String(payload?.type ?? "")
  }
}

const upsertSnapshot = (resp: ReturnType<typeof parseVarResp>) => {
  if (!resp) return
  const key = varKey(resp.ownerId, resp.name)
  const existing = state.values[key]
  state.values[key] = {
    ownerId: resp.ownerId,
    name: resp.name,
    value: resp.value !== "" ? resp.value : existing?.value ?? "",
    visibility: resp.visibility !== "" ? resp.visibility : existing?.visibility ?? "",
    type: resp.type !== "" ? resp.type : existing?.type ?? "",
    lastUpdated: nowIso()
  }
}

const removeSnapshot = (resp: ReturnType<typeof parseVarResp>) => {
  if (!resp) return
  const key = varKey(resp.ownerId, resp.name)
  delete state.values[key]
}

const ensureListeners = () => {
  if (initialized) return
  initialized = true

  EventsOn("varpool.changed", (evt: any) => {
    state.lastFrameAt = nowIso()
    upsertSnapshot(parseVarResp(evt))
  })

  EventsOn("varpool.deleted", (evt: any) => {
    state.lastFrameAt = nowIso()
    removeSnapshot(parseVarResp(evt))
  })
}

const load = async () => {
  ensureListeners()
  if (state.busy) return
  state.busy = true
  try {
    const raw = await callApp<any>("ShowcaseConfig")
    state.config = normalizeConfig(raw)
    state.loaded = true
    state.lastLoadedAt = nowIso()
  } finally {
    state.busy = false
  }
}

const save = async () => {
  ensureListeners()
  if (state.busy) return
  state.busy = true
  try {
    const raw = await callApp<any>("SaveShowcaseConfig", state.config)
    state.config = normalizeConfig(raw)
    state.loaded = true
    state.lastLoadedAt = nowIso()
  } finally {
    state.busy = false
  }
}

const setIdentity = (nodeId: number, hubId: number) => {
  state.selfNodeId = Number(nodeId || 0)
  state.hubId = Number(hubId || 0)
}

const currentScreen = () => {
  const id = state.config.currentScreenId
  return state.config.screens.find((s) => s.id === id) ?? state.config.screens[0]
}

const selectScreen = async (id: string) => {
  const trimmed = id.trim()
  if (!trimmed || trimmed === state.config.currentScreenId) return
  await leave()
  state.config.currentScreenId = trimmed
  await save()
  await enter()
}

const createScreen = async (name: string) => {
  const trimmed = name.trim()
  if (!trimmed) throw new Error("Screen name is required.")
  const id = newId()
  state.config.screens.push({ id, name: trimmed, widgets: [] })
  state.config.currentScreenId = id
  await save()
  await enter()
}

const renameScreen = async (id: string, name: string) => {
  const trimmed = name.trim()
  if (!trimmed) throw new Error("Screen name is required.")
  const screen = state.config.screens.find((s) => s.id === id)
  if (!screen) return
  screen.name = trimmed
  await save()
}

const deleteScreen = async (id: string) => {
  const remaining = state.config.screens.filter((s) => s.id !== id)
  if (!remaining.length) {
    throw new Error("At least one screen is required.")
  }
  const wasCurrent = state.config.currentScreenId === id
  if (wasCurrent) {
    await leave()
  }
  state.config.screens = remaining
  if (wasCurrent) {
    state.config.currentScreenId = remaining[0].id
  }
  await save()
  if (wasCurrent) {
    await enter()
  }
}

const removeWidget = async (widgetId: string) => {
  const screen = currentScreen()
  if (!screen) return
  const widget = screen.widgets.find((w) => w.id === widgetId)
  if (!widget) return
  if (widget.kind === "var") {
    await unsubscribeVarWidget(widget)
  }
  screen.widgets = screen.widgets.filter((w) => w.id !== widgetId)
  await save()
}

const addTopicButton = async (input: Partial<ShowcaseTopicButton> & { title?: string; targetId?: number }) => {
  const screen = currentScreen()
  const topic = String(input.topic ?? "").trim()
  const name = String(input.name ?? "").trim()
  if (!topic) throw new Error("Topic is required.")
  if (!name) throw new Error("Name is required.")
  const payloadText = String(input.payloadText ?? "").trim()
  const title = String(input.title ?? "").trim()
  const targetId = Number(input.targetId ?? 0)
  if (!Number.isFinite(targetId) || targetId <= 0) throw new Error("Target ID is required.")

  screen.widgets.push({
    id: newId(),
    kind: "topic_button",
    title,
    targetId: Math.floor(targetId),
    topicButton: { topic, name, payloadText }
  })
  await save()
}

const addVarWidget = async (input: {
  title?: string
  targetId: number
  ownerId: number
  name: string
  mode?: VarWidgetMode
  visibility?: string
  type: string
  slider?: Partial<ShowcaseVarSlider>
  switch?: Partial<ShowcaseVarSwitch>
}) => {
  const screen = currentScreen()
  const ownerId = Number(input.ownerId ?? 0)
  const name = String(input.name ?? "").trim()
  if (!Number.isFinite(ownerId) || ownerId <= 0) throw new Error("Owner NodeID is required.")
  if (!name) throw new Error("Variable name is required.")
  const title = String(input.title ?? "").trim()
  const targetId = Number(input.targetId ?? 0)
  if (!Number.isFinite(targetId) || targetId <= 0) throw new Error("Target ID is required.")
  const mode = input.mode ?? "auto"
  const visibility = String(input.visibility ?? "public").trim() || "public"
  const type = String(input.type ?? "").trim() || defaultTypeForMode(mode)
  const slider = normalizeSlider({ ...defaultSlider(), ...(input.slider ?? {}) })
  const sw = normalizeSwitch({ ...defaultSwitch(), ...(input.switch ?? {}) })

  const widget: ShowcaseWidget = {
    id: newId(),
    kind: "var",
    title,
    targetId: Math.floor(targetId),
    var: {
      ownerId,
      name,
      mode,
      visibility,
      type,
      slider,
      switch: sw
    }
  }
  screen.widgets.push(widget)
  await save()
  await ensureVarActive(widget)
}

const resolveEffectiveMode = (widget: ShowcaseWidget): VarWidgetMode => {
  if (widget.kind !== "var" || !widget.var) return "display"
  const configured = widget.var.mode
  if (configured !== "auto") return configured
  const snap = valueForVar(widget.var.ownerId, widget.var.name)
  const rawType = String(snap?.type || widget.var.type || "").trim().toLowerCase()
  if (rawType === "bool" || rawType === "boolean") return "switch"
  if (rawType === "int" || rawType === "int32" || rawType === "int64" || rawType === "float" || rawType === "float32" || rawType === "float64" || rawType === "number") {
    return "slider"
  }
  return "display"
}

const valueForVar = (ownerId: number, name: string): VarSnapshot | null => {
  const key = varKey(ownerId, name.trim())
  return state.values[key] ?? null
}

const getVarValueText = (widget: ShowcaseWidget): string => {
  if (widget.kind !== "var" || !widget.var) return ""
  const snap = valueForVar(widget.var.ownerId, widget.var.name)
  return snap?.value ?? ""
}

const sliderValue = (widget: ShowcaseWidget): number => {
  if (widget.kind !== "var" || !widget.var) return 0
  const draft = state.sliderDraft[widget.id]
  if (Number.isFinite(draft)) return draft
  const raw = getVarValueText(widget).trim()
  const parsed = Number.parseFloat(raw)
  if (Number.isFinite(parsed)) return parsed
  return widget.var.slider.min
}

const sendVarSet = async (widget: ShowcaseWidget, value: string, awaitResp: boolean) => {
  if (widget.kind !== "var" || !widget.var) throw new Error("Invalid var widget.")
  const sourceID = ensureReady()
  const targetID = resolveTarget(widget.targetId)
  const owner = widget.var.ownerId
  const name = widget.var.name.trim()
  const visibility = widget.var.visibility || "public"
  const type = widget.var.type.trim()
  if (!name) throw new Error("Variable name is required.")
  if (!value.trim()) throw new Error("Variable value is required.")
  if (!type) throw new Error("Variable type is required.")

  if (!awaitResp) {
    await callVarPool("SendSimple", sourceID, targetID, "set", {
      name,
      value,
      visibility,
      type,
      owner
    })
    return
  }

  const resp = parseVarResp(
    await callVarPool<any>("SetSimple", sourceID, targetID, {
      name,
      value,
      visibility,
      type,
      owner
    })
  )
  if (!resp) return
  upsertSnapshot({
    ...resp,
    value: resp.value || value,
    visibility: resp.visibility || visibility,
    type: resp.type || type
  })
}

const publishTopicButton = async (widget: ShowcaseWidget) => {
  if (widget.kind !== "topic_button" || !widget.topicButton) {
    throw new Error("Invalid topic button.")
  }
  const sourceID = ensureReady()
  const targetID = resolveTarget(widget.targetId)
  const topic = widget.topicButton.topic.trim()
  const name = widget.topicButton.name.trim()
  const payloadText = widget.topicButton.payloadText ?? ""
  if (!topic) throw new Error("Topic is required.")
  if (!name) throw new Error("Name is required.")
  await callTopicBus("PublishSimple", sourceID, targetID, topic, name, payloadText)
  toast.success("Event sent.")
}

const getAndSubscribe = async (targetId: number, ownerId: number, name: string) => {
  const sourceID = ensureReady()
  const targetID = resolveTarget(targetId)
  const trimmedName = name.trim()
  if (!trimmedName) return
  const key = subKey(targetID, ownerId, trimmedName)
  if (!activeSubs.has(key)) {
    activeSubs.add(key)
    try {
      await callVarPool("SubscribeSimple", sourceID, targetID, {
        name: trimmedName,
        owner: ownerId,
        subscriber: sourceID
      })
    } catch {
      // silent: background subscribe should not spam toasts
    }
  }
  try {
    const resp = parseVarResp(await callVarPool<any>("GetSimple", sourceID, targetID, { name: trimmedName, owner: ownerId }))
    if (resp) upsertSnapshot(resp)
  } catch {
    // silent
  }
}

const unsubscribeAll = async () => {
  if (!activeSubs.size) return
  const sourceID = state.selfNodeId
  if (!sourceID) {
    activeSubs.clear()
    return
  }
  const keys = Array.from(activeSubs)
  activeSubs.clear()
  for (const raw of keys) {
    const [targetRaw, ownerRaw, ...nameParts] = raw.split(":")
    const targetID = Number.parseInt(targetRaw ?? "", 10)
    const owner = Number.parseInt(ownerRaw ?? "", 10)
    const name = nameParts.join(":")
    if (!targetID || !owner || !name) continue
    try {
      await callVarPool("UnsubscribeSimple", sourceID, targetID, {
        name,
        owner,
        subscriber: sourceID
      })
    } catch {
      // ignore
    }
  }
}

const enter = async () => {
  ensureListeners()
  const screen = currentScreen()
  if (!screen) return
  if (!state.selfNodeId) return

  const refs: Array<{ targetId: number; ownerId: number; name: string }> = []
  const seen = new Set<string>()
  for (const widget of screen.widgets) {
    if (widget.kind !== "var" || !widget.var) continue
    const ownerId = widget.var.ownerId
    const name = widget.var.name.trim()
    if (!ownerId || !name) continue
    const targetId = Number.isFinite(widget.targetId) && widget.targetId > 0 ? Math.floor(widget.targetId) : 0
    if (!targetId) continue
    const k = subKey(targetId, ownerId, name)
    if (seen.has(k)) continue
    seen.add(k)
    refs.push({ targetId, ownerId, name })
  }

  for (const ref of refs) {
    await getAndSubscribe(ref.targetId, ref.ownerId, ref.name)
  }
}

const leave = async () => {
  ensureListeners()
  sliderTimers.forEach((timer) => window.clearTimeout(timer))
  sliderTimers.clear()
  sliderLastSentAt.clear()
  state.sliderDraft = {}
  await unsubscribeAll()
}

const ensureVarActive = async (widget: ShowcaseWidget) => {
  if (widget.kind !== "var" || !widget.var) return
  if (!state.selfNodeId) return
  const ownerId = widget.var.ownerId
  const name = widget.var.name.trim()
  if (!ownerId || !name) return
  const targetId = Number.isFinite(widget.targetId) && widget.targetId > 0 ? Math.floor(widget.targetId) : 0
  if (!targetId) return
  await getAndSubscribe(targetId, ownerId, name)
}

const unsubscribeVarWidget = async (widget: ShowcaseWidget) => {
  if (widget.kind !== "var" || !widget.var) return
  const sourceID = state.selfNodeId
  if (!sourceID) return
  const ownerId = widget.var.ownerId
  const name = widget.var.name.trim()
  if (!ownerId || !name) return
  const targetId = Number.isFinite(widget.targetId) && widget.targetId > 0 ? Math.floor(widget.targetId) : 0
  if (!targetId) return
  const key = subKey(targetId, ownerId, name)
  if (!activeSubs.has(key)) return
  activeSubs.delete(key)
  await callVarPool("UnsubscribeSimple", sourceID, targetId, { name, owner: ownerId, subscriber: sourceID })
}

const sliderInput = async (widget: ShowcaseWidget, value: number) => {
  if (widget.kind !== "var" || !widget.var) return
  if (!Number.isFinite(value)) return
  state.sliderDraft[widget.id] = value

  const throttleMsRaw = Number(widget.var.slider.throttleMs)
  const throttleMs =
    Number.isFinite(throttleMsRaw) && throttleMsRaw >= 0 ? Math.floor(throttleMsRaw) : 50
  if (throttleMs === 0) {
    const timer = sliderTimers.get(widget.id)
    if (timer) {
      window.clearTimeout(timer)
      sliderTimers.delete(widget.id)
    }
    void sendVarSet(widget, String(value), false).catch(() => {})
    return
  }
  const lastSentAt = sliderLastSentAt.get(widget.id) ?? 0
  const elapsed = Date.now() - lastSentAt
  if (elapsed >= throttleMs) {
    sliderLastSentAt.set(widget.id, Date.now())
    void sendVarSet(widget, String(value), false).catch(() => {})
    return
  }

  if (sliderTimers.has(widget.id)) return
  const timer = window.setTimeout(async () => {
    sliderTimers.delete(widget.id)
    sliderLastSentAt.set(widget.id, Date.now())
    const latest = state.sliderDraft[widget.id]
    if (!Number.isFinite(latest)) return
    void sendVarSet(widget, String(latest), false).catch(() => {})
  }, Math.max(0, throttleMs - elapsed))
  sliderTimers.set(widget.id, timer)
}

const sliderCommit = async (widget: ShowcaseWidget) => {
  if (widget.kind !== "var" || !widget.var) return
  const value = state.sliderDraft[widget.id]
  if (!Number.isFinite(value)) return

  const timer = sliderTimers.get(widget.id)
  if (timer) {
    window.clearTimeout(timer)
    sliderTimers.delete(widget.id)
  }

  try {
    await sendVarSet(widget, String(value), true)
    delete state.sliderDraft[widget.id]
  } catch (err) {
    toast.errorOf(err, "Failed to update variable.")
    return
  }
}

const switchToggle = async (widget: ShowcaseWidget, desiredOn: boolean) => {
  if (widget.kind !== "var" || !widget.var) return
  const onValue = widget.var.switch.onValue
  const offValue = widget.var.switch.offValue
  const next = desiredOn ? onValue : offValue
  try {
    await sendVarSet(widget, next, true)
  } catch (err) {
    toast.errorOf(err, "Failed to update variable.")
  }
}

export const useShowcaseStore = () => {
  ensureListeners()
  return {
    state,
    addTopicButton,
    addVarWidget,
    createScreen,
    currentScreen,
    deleteScreen,
    enter,
    getVarValueText,
    leave,
    load,
    save,
    publishTopicButton,
    removeWidget,
    renameScreen,
    resolveEffectiveMode,
    selectScreen,
    setIdentity,
    sliderCommit,
    sliderInput,
    sliderValue,
    switchToggle,
    valueForVar
  }
}
