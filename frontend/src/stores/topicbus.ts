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

const callTopicBus = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.topicbus?.TopicBusService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(`TopicBus binding '${method}' unavailable`)
  }
  return fn(...args)
}

export type TopicBusEvent = {
  topic: string
  name: string
  ts: number
  dataRaw: string
}

export type TopicBusChannelItem = {
  topic: string
  localSaved: boolean
  remoteSubscribed: boolean
}

export type TopicBusState = {
  targetId: string
  selfNodeId: number
  defaultTargetId: number
  topics: string[]
  remoteTopics: string[]
  selectedTopic: string
  maxEvents: number
  events: TopicBusEvent[]
  lastFrameAt: string
  remoteSyncedAt: string
}

const defaultMaxEvents = 500

const state = reactive<TopicBusState>({
  targetId: "",
  selfNodeId: 0,
  defaultTargetId: 0,
  topics: [],
  remoteTopics: [],
  selectedTopic: "",
  maxEvents: defaultMaxEvents,
  events: [],
  lastFrameAt: "",
  remoteSyncedAt: ""
})

const pendingEvents: TopicBusEvent[] = []
let flushTimer: number | null = null
let lastFlushAt = 0
let initialized = false

const nowIso = () => new Date().toISOString()

const formatDetail = (input: any): string => {
  if (input === null || input === undefined) return ""
  if (typeof input === "string") {
    const trimmed = input.trim()
    if (!trimmed) return ""
    if (trimmed.startsWith("{") || trimmed.startsWith("[")) {
      try {
        return JSON.stringify(JSON.parse(trimmed), null, 2)
      } catch {
        return input
      }
    }
    return input
  }
  try {
    return JSON.stringify(input, null, 2)
  } catch {
    return String(input)
  }
}

export const formatTopicBusTimestamp = (ts: number) => {
  if (!ts) return ""
  const dt = new Date(ts)
  const pad = (value: number, len = 2) => String(value).padStart(len, "0")
  return `${dt.getFullYear()}-${pad(dt.getMonth() + 1)}-${pad(dt.getDate())} ${pad(
    dt.getHours()
  )}:${pad(dt.getMinutes())}:${pad(dt.getSeconds())}.${pad(dt.getMilliseconds(), 3)}`
}

export const normalizeTopicBusEvent = (evt: any): TopicBusEvent | null => {
  const topic = String(evt?.topic ?? "").trim()
  const name = String(evt?.name ?? "").trim()
  if (!topic || !name) return null
  const ts = Number(evt?.ts ?? 0)
  return {
    topic,
    name,
    ts,
    dataRaw: formatDetail(evt)
  }
}

const normalizeTopics = (topics: string[]) => {
  const out: string[] = []
  const seen = new Set<string>()
  for (const topic of topics) {
    const trimmed = String(topic ?? "").trim()
    if (!trimmed || seen.has(trimmed)) continue
    seen.add(trimmed)
    out.push(trimmed)
  }
  return out
}

const mergeTopics = (existing: string[], add: string[]) => {
  return normalizeTopics([...existing, ...add])
}

const removeTopics = (existing: string[], remove: string[]) => {
  const removeSet = new Set(normalizeTopics(remove))
  if (!removeSet.size) return normalizeTopics(existing)
  return normalizeTopics(existing.filter((topic) => !removeSet.has(topic)))
}

const parseTopics = (raw: string) => {
  const trimmed = raw.trim()
  if (!trimmed) return []
  return normalizeTopics(trimmed.split(/[\n,，;；]+/g))
}

const selectionStillExists = () => {
  if (!state.selectedTopic) return true
  if (state.topics.includes(state.selectedTopic)) return true
  if (state.remoteTopics.includes(state.selectedTopic)) return true
  return false
}

const clearSelectionIfMissing = () => {
  if (!selectionStillExists()) {
    state.selectedTopic = ""
  }
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
    throw new Error("Login required to send TopicBus requests.")
  }
  return state.selfNodeId
}

const trimEvents = () => {
  if (state.maxEvents <= 0) {
    state.maxEvents = defaultMaxEvents
  }
  if (state.events.length <= state.maxEvents) return
  state.events = state.events.slice(-state.maxEvents)
}

const flushPending = () => {
  if (!pendingEvents.length) return
  state.events.push(...pendingEvents)
  pendingEvents.length = 0
  trimEvents()
  lastFlushAt = Date.now()
}

const scheduleFlush = () => {
  const now = Date.now()
  const elapsed = now - lastFlushAt
  if (elapsed >= 200) {
    flushPending()
    return
  }
  if (flushTimer !== null) return
  flushTimer = window.setTimeout(() => {
    flushTimer = null
    flushPending()
  }, Math.max(0, 200 - elapsed))
}

const pushEvent = (ev: TopicBusEvent) => {
  pendingEvents.push(ev)
  scheduleFlush()
}

const handleEvent = (evt: any) => {
  const normalized = normalizeTopicBusEvent(evt)
  if (!normalized) return
  pushEvent(normalized)
}

const channelItems = (): TopicBusChannelItem[] => {
  const localSet = new Set(normalizeTopics(state.topics))
  const remoteSet = new Set(normalizeTopics(state.remoteTopics))
  const merged = normalizeTopics([...state.topics, ...state.remoteTopics]).sort((left, right) =>
    left.localeCompare(right)
  )
  return merged.map((topic) => ({
    topic,
    localSaved: localSet.has(topic),
    remoteSubscribed: remoteSet.has(topic)
  }))
}

const loadPrefs = async () => {
  const prefs = await callApp<any>("TopicBusPrefs")
  const topics = normalizeTopics(Array.isArray(prefs?.topics) ? prefs.topics : [])
  const maxEvents = Number(prefs?.maxEvents ?? defaultMaxEvents)
  state.topics = topics
  state.maxEvents = maxEvents > 0 ? maxEvents : defaultMaxEvents
  clearSelectionIfMissing()
  trimEvents()
}

const savePrefs = async () => {
  const saved = await callApp<any>("SaveTopicBusPrefs", {
    topics: state.topics,
    maxEvents: state.maxEvents
  })
  if (saved) {
    state.topics = normalizeTopics(Array.isArray(saved?.topics) ? saved.topics : state.topics)
    const maxEvents = Number(saved?.maxEvents ?? state.maxEvents)
    state.maxEvents = maxEvents > 0 ? maxEvents : defaultMaxEvents
  }
  clearSelectionIfMissing()
}

const setIdentity = (nodeId: number, hubId: number) => {
  state.selfNodeId = Number(nodeId || 0)
  state.defaultTargetId = Number(hubId || 0)
  if (!state.targetId && state.defaultTargetId) {
    state.targetId = String(state.defaultTargetId)
  }
}

const setSelectedTopic = (topic: string) => {
  state.selectedTopic = String(topic ?? "").trim()
  clearSelectionIfMissing()
}

const updateTopics = async (topics: string[], mode: "add" | "remove") => {
  const normalized = normalizeTopics(topics)
  if (normalized.length === 0) return []
  if (mode === "add") {
    state.topics = mergeTopics(state.topics, normalized)
  } else {
    state.topics = removeTopics(state.topics, normalized)
  }
  clearSelectionIfMissing()
  await savePrefs()
  return normalized
}

const refreshRemoteTopics = async () => {
  const sourceID = Number(state.selfNodeId || 0)
  const targetID = Number(resolveTargetId() || 0)
  if (sourceID <= 0 || targetID <= 0) {
    state.remoteTopics = []
    state.remoteSyncedAt = ""
    clearSelectionIfMissing()
    return []
  }
  const resp = await callTopicBus<any>("ListSubsSimple", sourceID, targetID)
  const topics = normalizeTopics(Array.isArray(resp?.topics) ? resp.topics : [])
  state.remoteTopics = topics
  state.remoteSyncedAt = nowIso()
  clearSelectionIfMissing()
  return topics
}

const subscribe = async (topics: string[]) => {
  const normalized = normalizeTopics(topics)
  if (!normalized.length) {
    throw new Error("Topic is required.")
  }
  const sourceID = ensureSourceID()
  const targetID = resolveTargetId()
  if (normalized.length === 1) {
    await callTopicBus("SubscribeSimple", sourceID, targetID, normalized[0])
  } else {
    await callTopicBus("SubscribeBatchSimple", sourceID, targetID, normalized)
  }
  state.remoteTopics = mergeTopics(state.remoteTopics, normalized)
  state.remoteSyncedAt = nowIso()
}

const unsubscribe = async (topics: string[]) => {
  const normalized = normalizeTopics(topics)
  if (!normalized.length) {
    throw new Error("Topic is required.")
  }
  const sourceID = ensureSourceID()
  const targetID = resolveTargetId()
  if (normalized.length === 1) {
    await callTopicBus("UnsubscribeSimple", sourceID, targetID, normalized[0])
  } else {
    await callTopicBus("UnsubscribeBatchSimple", sourceID, targetID, normalized)
  }
  state.remoteTopics = removeTopics(state.remoteTopics, normalized)
  state.remoteSyncedAt = nowIso()
  clearSelectionIfMissing()
}

const resubscribe = async () => {
  const normalized = normalizeTopics(state.topics)
  if (!normalized.length) return
  const sourceID = ensureSourceID()
  const targetID = resolveTargetId()
  if (normalized.length === 1) {
    await callTopicBus("SubscribeSimple", sourceID, targetID, normalized[0])
  } else {
    await callTopicBus("SubscribeBatchSimple", sourceID, targetID, normalized)
  }
  state.remoteTopics = mergeTopics(state.remoteTopics, normalized)
  state.remoteSyncedAt = nowIso()
}

const publish = async (topic: string, name: string, payloadText: string) => {
  const trimmedTopic = String(topic ?? "").trim()
  const trimmedName = String(name ?? "").trim()
  if (!trimmedTopic) throw new Error("Topic is required.")
  if (!trimmedName) throw new Error("Name is required.")
  const sourceID = ensureSourceID()
  const targetID = resolveTargetId()
  await callTopicBus("PublishSimple", sourceID, targetID, trimmedTopic, trimmedName, payloadText)
}

const clearEvents = () => {
  pendingEvents.length = 0
  state.events = []
}

const setMaxEvents = async (value: number) => {
  if (!Number.isFinite(value) || value <= 0) {
    throw new Error("Max events must be a positive number.")
  }
  state.maxEvents = Math.floor(value)
  trimEvents()
  await savePrefs()
}

const ensureListeners = () => {
  if (initialized) return
  initialized = true
  EventsOn("topicbus.event", (evt: any) => {
    state.lastFrameAt = nowIso()
    handleEvent(evt)
  })
}

export const useTopicBusStore = () => {
  ensureListeners()
  return {
    state,
    channelItems,
    clearEvents,
    loadPrefs,
    parseTopics,
    publish,
    refreshRemoteTopics,
    resubscribe,
    resolveTargetId,
    setIdentity,
    setMaxEvents,
    setSelectedTopic,
    subscribe,
    unsubscribe,
    updateTopics
  }
}
