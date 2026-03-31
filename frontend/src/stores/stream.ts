import { reactive } from "vue"
import { t } from "@/i18n"
import { EventsOn } from "../../wailsjs/runtime/runtime"

type WailsBinding = (...args: any[]) => Promise<any>

const callApp = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.main?.App
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) throw new Error(t("App binding '{method}' unavailable", { method }))
  return fn(...args)
}

const callStream = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.stream?.StreamService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) throw new Error(t("Stream binding '{method}' unavailable", { method }))
  return fn(...args)
}

export const streamKinds = ["music", "video", "text", "custom"] as const
export type StreamKind = (typeof streamKinds)[number]
export type StreamTab = "source" | "consumer" | "control"

export type StreamSource = {
  sourceId: string
  producer: number
  name: string
  kind: string
  contentType: string
  mode: string
  unitMode: string
  tags: string[]
  metadataRaw: string
}

export type StreamConsumer = {
  consumerId: string
  consumer: number
  name: string
  kind: string
  contentType: string
  tags: string[]
  metadataRaw: string
}

export type StreamDelivery = {
  deliveryId: string
  sourceId: string
  producer: number
  consumer: number
  consumerId: string
  kind: string
  contentType: string
  mode: string
  unitMode: string
  state: string
  bytesIn: number
  framesIn: number
  lastPosition: number
  lastPtsMs: number
  lastAckPos: number
  lastFlags: number
  lastError: string
  updatedAt: string
}

export type StreamTextFrame = {
  deliveryId: string
  kind: string
  text: string
  position: number
  ptsMs: number
  flags: number
  updatedAt: string
}

export type StreamStatsFrame = {
  deliveryId: string
  kind: string
  bytesIn: number
  framesIn: number
  lastPosition: number
  lastPtsMs: number
  lastAckPos: number
  lastFlags: number
  updatedAt: string
}

export type StreamSourceDraft = {
  sourceId: string
  name: string
  kind: string
  contentType: string
  mode: string
  unitMode: string
  tagsText: string
  metadataText: string
}

export type StreamConsumerDraft = {
  consumerId: string
  name: string
  kind: string
  contentType: string
  tagsText: string
  metadataText: string
}

export type StreamRestoreResult = { attempted: number; failed: number }
export type StreamPublishTextResult = { sourceId: string; sent: number; deliveryIds: string[] }

type StreamState = {
  activeTab: StreamTab
  targetId: string
  selfNodeId: number
  defaultTargetId: number
  localSources: StreamSource[]
  localConsumers: StreamConsumer[]
  sources: StreamSource[]
  consumers: StreamConsumer[]
  deliveries: StreamDelivery[]
  selectedSourceId: string
  selectedConsumerId: string
  selectedDeliveryId: string
  lastSyncAt: string
  lastEventAt: string
  textFramesByDelivery: Record<string, StreamTextFrame[]>
  statsByDelivery: Record<string, StreamStatsFrame>
}

const state = reactive<StreamState>({
  activeTab: "source",
  targetId: "",
  selfNodeId: 0,
  defaultTargetId: 0,
  localSources: [],
  localConsumers: [],
  sources: [],
  consumers: [],
  deliveries: [],
  selectedSourceId: "",
  selectedConsumerId: "",
  selectedDeliveryId: "",
  lastSyncAt: "",
  lastEventAt: "",
  textFramesByDelivery: {},
  statsByDelivery: {}
})

let initialized = false
let restorePromise: Promise<StreamRestoreResult> | null = null
let lastRestoreKey = ""

const nowIso = () => new Date().toISOString()

const normalizeNodeText = (value: string, field: string) => {
  const trimmed = String(value ?? "").trim()
  if (!trimmed) throw new Error(t("{field} is required.", { field }))
  const parsed = Number.parseInt(trimmed, 10)
  if (!Number.isFinite(parsed) || parsed <= 0) throw new Error(t("{field} must be a positive number.", { field }))
  return parsed
}

const normalizeConfiguredTargetId = (value: unknown) => {
  const trimmed = String(value ?? "").trim()
  if (!trimmed) return ""
  const parsed = Number.parseInt(trimmed, 10)
  if (!Number.isFinite(parsed) || parsed <= 0) throw new Error(t("Target Node ID must be a positive number."))
  return String(parsed)
}

const normalizeTargetId = (value: string) => {
  const normalized = normalizeConfiguredTargetId(value)
  return normalized ? Number.parseInt(normalized, 10) : state.defaultTargetId
}

const normalizeTab = (value: unknown): StreamTab => {
  switch (String(value ?? "").trim().toLowerCase()) {
    case "consumer":
      return "consumer"
    case "control":
      return "control"
    default:
      return "source"
  }
}

const ensureSourceId = () => {
  if (!state.selfNodeId) throw new Error(t("Login required to use Stream controls."))
  return state.selfNodeId
}

const resolveTargetId = () => normalizeTargetId(state.targetId)

const normalizeTags = (value: string | string[]) => {
  const items = Array.isArray(value) ? value : String(value ?? "").split(/[\n,，;；]+/g)
  const seen = new Set<string>()
  const out: string[] = []
  for (const item of items) {
    const tag = String(item ?? "").trim()
    if (!tag || seen.has(tag)) continue
    seen.add(tag)
    out.push(tag)
  }
  return out
}

const normalizeMetadata = (value: string) => {
  const trimmed = String(value ?? "").trim()
  if (!trimmed) return undefined
  try {
    return JSON.parse(trimmed)
  } catch {
    throw new Error(t("Metadata must be valid JSON."))
  }
}

const formatMetadata = (value: any) => {
  if (value === null || value === undefined || value === "") return ""
  if (typeof value === "string") {
    const trimmed = value.trim()
    if (!trimmed) return ""
    if (trimmed.startsWith("{") || trimmed.startsWith("[")) {
      try {
        return JSON.stringify(JSON.parse(trimmed), null, 2)
      } catch {
        return trimmed
      }
    }
    return trimmed
  }
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

const normalizeSource = (input: any): StreamSource | null => {
  const sourceId = String(input?.sourceId ?? input?.source_id ?? "").trim()
  if (!sourceId) return null
  return {
    sourceId,
    producer: Number(input?.producer ?? state.selfNodeId ?? 0),
    name: String(input?.name ?? "").trim(),
    kind: String(input?.kind ?? "").trim(),
    contentType: String(input?.contentType ?? input?.content_type ?? "").trim(),
    mode: String(input?.mode ?? "").trim(),
    unitMode: String(input?.unitMode ?? input?.unit_mode ?? "").trim(),
    tags: Array.isArray(input?.tags) ? input.tags.map((item: unknown) => String(item ?? "").trim()).filter(Boolean) : [],
    metadataRaw: formatMetadata(input?.metadataRaw ?? input?.metadata)
  }
}

const normalizeConsumer = (input: any): StreamConsumer | null => {
  const consumerId = String(input?.consumerId ?? input?.consumer_id ?? "").trim()
  if (!consumerId) return null
  return {
    consumerId,
    consumer: Number(input?.consumer ?? state.selfNodeId ?? 0),
    name: String(input?.name ?? "").trim(),
    kind: String(input?.kind ?? "").trim(),
    contentType: String(input?.contentType ?? input?.content_type ?? "").trim(),
    tags: Array.isArray(input?.tags) ? input.tags.map((item: unknown) => String(item ?? "").trim()).filter(Boolean) : [],
    metadataRaw: formatMetadata(input?.metadataRaw ?? input?.metadata)
  }
}

const normalizeDelivery = (input: any): StreamDelivery | null => {
  const deliveryId = String(input?.deliveryId ?? input?.delivery_id ?? "").trim()
  if (!deliveryId) return null
  return {
    deliveryId,
    sourceId: String(input?.sourceId ?? input?.source_id ?? "").trim(),
    producer: Number(input?.producer ?? 0),
    consumer: Number(input?.consumer ?? 0),
    consumerId: String(input?.consumerId ?? input?.consumer_id ?? "").trim(),
    kind: String(input?.kind ?? "").trim(),
    contentType: String(input?.contentType ?? input?.content_type ?? "").trim(),
    mode: String(input?.mode ?? "").trim(),
    unitMode: String(input?.unitMode ?? input?.unit_mode ?? "").trim(),
    state: String(input?.state ?? "").trim(),
    bytesIn: Number(input?.bytesIn ?? input?.bytes_in ?? 0),
    framesIn: Number(input?.framesIn ?? input?.frames_in ?? 0),
    lastPosition: Number(input?.lastPosition ?? input?.last_position ?? 0),
    lastPtsMs: Number(input?.lastPtsMs ?? input?.last_pts_ms ?? 0),
    lastAckPos: Number(input?.lastAckPos ?? input?.last_ack_pos ?? 0),
    lastFlags: Number(input?.lastFlags ?? input?.last_flags ?? 0),
    lastError: String(input?.lastError ?? input?.last_error ?? "").trim(),
    updatedAt: String(input?.updatedAt ?? input?.updated_at ?? nowIso()).trim() || nowIso()
  }
}

const normalizeTextFrame = (input: any): StreamTextFrame | null => {
  const deliveryId = String(input?.deliveryId ?? input?.delivery_id ?? "").trim()
  if (!deliveryId) return null
  return {
    deliveryId,
    kind: String(input?.kind ?? "").trim(),
    text: String(input?.text ?? ""),
    position: Number(input?.position ?? 0),
    ptsMs: Number(input?.ptsMs ?? input?.pts_ms ?? 0),
    flags: Number(input?.flags ?? 0),
    updatedAt: String(input?.updatedAt ?? input?.updated_at ?? nowIso()).trim() || nowIso()
  }
}

const normalizeStatsFrame = (input: any): StreamStatsFrame | null => {
  const deliveryId = String(input?.deliveryId ?? input?.delivery_id ?? "").trim()
  if (!deliveryId) return null
  return {
    deliveryId,
    kind: String(input?.kind ?? "").trim(),
    bytesIn: Number(input?.bytesIn ?? input?.bytes_in ?? 0),
    framesIn: Number(input?.framesIn ?? input?.frames_in ?? 0),
    lastPosition: Number(input?.lastPosition ?? input?.last_position ?? 0),
    lastPtsMs: Number(input?.lastPtsMs ?? input?.last_pts_ms ?? 0),
    lastAckPos: Number(input?.lastAckPos ?? input?.last_ack_pos ?? 0),
    lastFlags: Number(input?.lastFlags ?? input?.last_flags ?? 0),
    updatedAt: String(input?.updatedAt ?? input?.updated_at ?? nowIso()).trim() || nowIso()
  }
}

const upsertSourceList = (list: StreamSource[], source: StreamSource) => {
  const next = [...list]
  const index = next.findIndex((item) => item.sourceId === source.sourceId)
  if (index >= 0) next[index] = { ...next[index], ...source }
  else next.unshift(source)
  return next
}

const upsertConsumerList = (list: StreamConsumer[], consumer: StreamConsumer) => {
  const next = [...list]
  const index = next.findIndex((item) => item.consumerId === consumer.consumerId)
  if (index >= 0) next[index] = { ...next[index], ...consumer }
  else next.unshift(consumer)
  return next
}

const removeSourceFromList = (list: StreamSource[], sourceId: string) => list.filter((item) => item.sourceId !== String(sourceId ?? "").trim())
const removeConsumerFromList = (list: StreamConsumer[], consumerId: string) => list.filter((item) => item.consumerId !== String(consumerId ?? "").trim())

const touchSync = () => {
  state.lastSyncAt = nowIso()
}

const touchEvent = () => {
  state.lastEventAt = nowIso()
}

const resetRestoreState = () => {
  lastRestoreKey = ""
}

const applyLocalIdentity = () => {
  if (!state.selfNodeId) return
  state.localSources = state.localSources.map((item) => ({ ...item, producer: state.selfNodeId }))
  state.localConsumers = state.localConsumers.map((item) => ({ ...item, consumer: state.selfNodeId }))
}

const applyPrefs = (prefs: any) => {
  state.activeTab = normalizeTab(prefs?.activeTab)
  const targetId = Number(prefs?.targetId ?? 0)
  state.targetId = Number.isFinite(targetId) && targetId > 0 ? String(Math.floor(targetId)) : ""
  state.localSources = Array.isArray(prefs?.sources) ? (prefs.sources.map(normalizeSource).filter(Boolean) as StreamSource[]) : []
  state.localConsumers = Array.isArray(prefs?.consumers) ? (prefs.consumers.map(normalizeConsumer).filter(Boolean) as StreamConsumer[]) : []
  applyLocalIdentity()
  resetRestoreState()
}

const buildSourcePayload = (source: { sourceId: string; name: string; kind: string; contentType: string; mode: string; unitMode: string; tags: string[]; metadataRaw: string }) => ({
  req_id: "",
  source: {
    source_id: String(source.sourceId ?? "").trim(),
    name: String(source.name ?? "").trim(),
    kind: String(source.kind ?? "").trim(),
    content_type: String(source.contentType ?? "").trim(),
    mode: String(source.mode ?? "").trim(),
    unit_mode: String(source.unitMode ?? "").trim(),
    tags: normalizeTags(source.tags),
    metadata: normalizeMetadata(source.metadataRaw)
  }
})

const buildConsumerPayload = (consumer: { consumerId: string; name: string; kind: string; contentType: string; tags: string[]; metadataRaw: string }) => ({
  req_id: "",
  consumer_endpoint: {
    consumer_id: String(consumer.consumerId ?? "").trim(),
    name: String(consumer.name ?? "").trim(),
    kind: String(consumer.kind ?? "").trim(),
    content_type: String(consumer.contentType ?? "").trim(),
    tags: normalizeTags(consumer.tags),
    metadata: normalizeMetadata(consumer.metadataRaw)
  }
})

const buildSourceDraftPayload = (draft: StreamSourceDraft) =>
  buildSourcePayload({
    sourceId: draft.sourceId,
    name: draft.name,
    kind: draft.kind,
    contentType: draft.contentType,
    mode: draft.mode,
    unitMode: draft.unitMode,
    tags: normalizeTags(draft.tagsText),
    metadataRaw: draft.metadataText
  })

const buildConsumerDraftPayload = (draft: StreamConsumerDraft) =>
  buildConsumerPayload({
    consumerId: draft.consumerId,
    name: draft.name,
    kind: draft.kind,
    contentType: draft.contentType,
    tags: normalizeTags(draft.tagsText),
    metadataRaw: draft.metadataText
  })

const toAppSource = (source: StreamSource) => ({
  sourceId: source.sourceId,
  name: source.name,
  kind: source.kind,
  contentType: source.contentType,
  mode: source.mode,
  unitMode: source.unitMode,
  tags: source.tags,
  metadataRaw: source.metadataRaw
})

const toAppConsumer = (consumer: StreamConsumer) => ({
  consumerId: consumer.consumerId,
  name: consumer.name,
  kind: consumer.kind,
  contentType: consumer.contentType,
  tags: consumer.tags,
  metadataRaw: consumer.metadataRaw
})

const upsertDelivery = (delivery: StreamDelivery) => {
  const next = [...state.deliveries]
  const index = next.findIndex((item) => item.deliveryId === delivery.deliveryId)
  if (index >= 0) next[index] = { ...next[index], ...delivery }
  else next.push(delivery)
  next.sort((left, right) => String(right.updatedAt).localeCompare(String(left.updatedAt)))
  state.deliveries = next
  if (!state.selectedDeliveryId) state.selectedDeliveryId = delivery.deliveryId
}

const appendTextFrame = (frame: StreamTextFrame) => {
  const current = Array.isArray(state.textFramesByDelivery[frame.deliveryId]) ? [...state.textFramesByDelivery[frame.deliveryId]] : []
  current.push(frame)
  state.textFramesByDelivery[frame.deliveryId] = current.slice(-200)
}

const rememberStatsFrame = (frame: StreamStatsFrame) => {
  state.statsByDelivery[frame.deliveryId] = frame
}

const savePrefs = async () => {
  const saved = await callApp<any>("SaveStreamPrefs", {
    activeTab: state.activeTab,
    targetId: state.targetId ? Number.parseInt(normalizeConfiguredTargetId(state.targetId), 10) : 0,
    sources: state.localSources.map(toAppSource),
    consumers: state.localConsumers.map(toAppConsumer)
  })
  applyPrefs(saved)
  return saved
}

const savePrefsBestEffort = async () => {
  try {
    await savePrefs()
  } catch (err) {
    console.warn(err)
  }
}

const loadPrefs = async () => {
  const prefs = await callApp<any>("StreamPrefs")
  applyPrefs(prefs)
  return prefs
}

const loadDeliveries = async () => {
  const snapshot = await callStream<any[]>("DeliverySnapshot")
  state.deliveries = Array.isArray(snapshot) ? (snapshot.map(normalizeDelivery).filter(Boolean) as StreamDelivery[]) : []
  touchSync()
  return state.deliveries
}

const listSources = async (producerText: string, kind = "", tag = "", scope: "catalog" | "local" = "catalog") => {
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  const producer = normalizeNodeText(producerText, t("Producer Node ID"))
  const resp = await callStream<any>("ListSourcesSimple", sourceID, targetID, {
    req_id: "",
    producer,
    kind: String(kind ?? "").trim(),
    tag: String(tag ?? "").trim()
  })
  const items = Array.isArray(resp?.sources) ? (resp.sources.map(normalizeSource).filter(Boolean) as StreamSource[]) : []
  if (scope === "local") {
    state.localSources = items.map((item) => ({ ...item, producer: state.selfNodeId || item.producer }))
  } else {
    state.sources = items
    if (state.selectedSourceId && !state.sources.some((item) => item.sourceId === state.selectedSourceId)) state.selectedSourceId = ""
  }
  touchSync()
  return items
}

const listConsumers = async (consumerText: string, kind = "", tag = "", scope: "catalog" | "local" = "catalog") => {
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  const consumer = normalizeNodeText(consumerText, t("Consumer Node ID"))
  const resp = await callStream<any>("ListConsumersSimple", sourceID, targetID, {
    req_id: "",
    consumer,
    kind: String(kind ?? "").trim(),
    tag: String(tag ?? "").trim()
  })
  const items = Array.isArray(resp?.consumer_endpoints) ? (resp.consumer_endpoints.map(normalizeConsumer).filter(Boolean) as StreamConsumer[]) : []
  if (scope === "local") {
    state.localConsumers = items.map((item) => ({ ...item, consumer: state.selfNodeId || item.consumer }))
  } else {
    state.consumers = items
    if (state.selectedConsumerId && !state.consumers.some((item) => item.consumerId === state.selectedConsumerId)) state.selectedConsumerId = ""
  }
  touchSync()
  return items
}

const announceSource = async (draft: StreamSourceDraft) => {
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  const resp = await callStream<any>("AnnounceSimple", sourceID, targetID, buildSourceDraftPayload(draft))
  const source = normalizeSource(resp?.source)
  if (source) {
    state.localSources = upsertSourceList(state.localSources, { ...source, producer: state.selfNodeId || source.producer })
    state.sources = upsertSourceList(state.sources, source)
    state.selectedSourceId = source.sourceId
    await savePrefs()
  }
  touchSync()
  return source
}

const withdrawSource = async (sourceId: string) => {
  const normalized = String(sourceId ?? "").trim()
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  await callStream("WithdrawSimple", sourceID, targetID, { req_id: "", source_id: normalized })
  state.localSources = removeSourceFromList(state.localSources, normalized)
  state.sources = removeSourceFromList(state.sources, normalized)
  if (state.selectedSourceId === normalized) state.selectedSourceId = ""
  await savePrefs()
  touchSync()
}

const announceConsumer = async (draft: StreamConsumerDraft) => {
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  const resp = await callStream<any>("AnnounceConsumerSimple", sourceID, targetID, buildConsumerDraftPayload(draft))
  const consumer = normalizeConsumer(resp?.consumer_endpoint)
  if (consumer) {
    state.localConsumers = upsertConsumerList(state.localConsumers, { ...consumer, consumer: state.selfNodeId || consumer.consumer })
    state.consumers = upsertConsumerList(state.consumers, consumer)
    state.selectedConsumerId = consumer.consumerId
    await savePrefs()
  }
  touchSync()
  return consumer
}

const withdrawConsumer = async (consumerId: string) => {
  const normalized = String(consumerId ?? "").trim()
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  await callStream("WithdrawConsumerSimple", sourceID, targetID, { req_id: "", consumer_id: normalized })
  state.localConsumers = removeConsumerFromList(state.localConsumers, normalized)
  state.consumers = removeConsumerFromList(state.consumers, normalized)
  if (state.selectedConsumerId === normalized) state.selectedConsumerId = ""
  await savePrefs()
  touchSync()
}

const connect = async (input: { producer: number; sourceId: string; consumer: number; consumerId: string }) => {
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  const resp = await callStream<any>("ConnectSimple", sourceID, targetID, {
    req_id: "",
    producer: Number(input.producer || 0),
    source_id: String(input.sourceId ?? "").trim(),
    consumer: Number(input.consumer || 0),
    consumer_id: String(input.consumerId ?? "").trim()
  })
  const delivery = normalizeDelivery({
    deliveryId: resp?.delivery_id,
    sourceId: resp?.source?.source_id,
    producer: resp?.producer,
    consumer: resp?.consumer,
    consumerId: resp?.consumer_id,
    kind: resp?.source?.kind,
    contentType: resp?.source?.content_type,
    mode: resp?.source?.mode,
    unitMode: resp?.source?.unit_mode,
    state: resp?.accept ? "active" : "pending",
    updatedAt: nowIso()
  })
  if (delivery) {
    upsertDelivery(delivery)
    state.selectedDeliveryId = delivery.deliveryId
  }
  touchSync()
  return delivery
}

const subscribe = async (input: { producer: number; sourceId: string; consumerId: string }) => {
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  const resp = await callStream<any>("SubscribeSimple", sourceID, targetID, {
    req_id: "",
    producer: Number(input.producer || 0),
    source_id: String(input.sourceId ?? "").trim(),
    consumer_id: String(input.consumerId ?? "").trim()
  })
  const delivery = normalizeDelivery({
    deliveryId: resp?.delivery_id,
    sourceId: resp?.source?.source_id,
    producer: resp?.producer,
    consumer: resp?.consumer,
    consumerId: resp?.consumer_id,
    kind: resp?.source?.kind,
    contentType: resp?.source?.content_type,
    mode: resp?.source?.mode,
    unitMode: resp?.source?.unit_mode,
    state: resp?.accept ? "active" : "pending",
    updatedAt: nowIso()
  })
  if (delivery) {
    upsertDelivery(delivery)
    state.selectedDeliveryId = delivery.deliveryId
  }
  touchSync()
  return delivery
}

const disconnect = async (deliveryId: string, reason = "") => {
  const normalized = String(deliveryId ?? "").trim()
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  await callStream("DisconnectSimple", sourceID, targetID, { req_id: "", delivery_id: normalized, reason: String(reason ?? "").trim() })
  upsertDelivery({
    ...(state.deliveries.find((item) => item.deliveryId === normalized) ?? {
      deliveryId: normalized,
      sourceId: "",
      producer: 0,
      consumer: 0,
      consumerId: "",
      kind: "custom",
      contentType: "",
      mode: "",
      unitMode: "",
      bytesIn: 0,
      framesIn: 0,
      lastPosition: 0,
      lastPtsMs: 0,
      lastAckPos: 0,
      lastFlags: 0,
      lastError: ""
    }),
    state: "closed",
    updatedAt: nowIso()
  })
  touchSync()
}

const unsubscribe = async (deliveryId: string, reason = "") => {
  const normalized = String(deliveryId ?? "").trim()
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  await callStream("UnsubscribeSimple", sourceID, targetID, { req_id: "", delivery_id: normalized, reason: String(reason ?? "").trim() })
  upsertDelivery({
    ...(state.deliveries.find((item) => item.deliveryId === normalized) ?? {
      deliveryId: normalized,
      sourceId: "",
      producer: 0,
      consumer: 0,
      consumerId: "",
      kind: "custom",
      contentType: "",
      mode: "",
      unitMode: "",
      bytesIn: 0,
      framesIn: 0,
      lastPosition: 0,
      lastPtsMs: 0,
      lastAckPos: 0,
      lastFlags: 0,
      lastError: ""
    }),
    state: "closed",
    updatedAt: nowIso()
  })
  touchSync()
}

const signal = async (deliveryId: string, op: string, data?: unknown) => {
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  await callStream("SignalSimple", sourceID, targetID, {
    req_id: "",
    delivery_id: String(deliveryId ?? "").trim(),
    op: String(op ?? "").trim(),
    data
  })
  touchSync()
}

const publishText = async (sourceId: string, text: string) => {
  const normalizedSourceID = String(sourceId ?? "").trim()
  const normalizedText = String(text ?? "")
  if (!normalizedSourceID) throw new Error(t("Source ID is required."))
  if (!normalizedText.trim()) throw new Error(t("Text content is required."))
  const source = state.localSources.find((item) => item.sourceId === normalizedSourceID)
  if (!source) throw new Error(t("Source not found."))
  if (source.kind !== "text") throw new Error(t("Only text sources support direct input."))
  const sourceID = ensureSourceId()
  const resp = await callStream<any>("PublishTextSimple", sourceID, { source_id: normalizedSourceID, text: normalizedText })
  touchSync()
  return {
    sourceId: String(resp?.source_id ?? normalizedSourceID).trim() || normalizedSourceID,
    sent: Number(resp?.sent ?? 0),
    deliveryIds: Array.isArray(resp?.delivery_ids) ? resp.delivery_ids.map((item: unknown) => String(item ?? "").trim()).filter(Boolean) : []
  } satisfies StreamPublishTextResult
}

const restoreLocalCatalogs = async (options?: { force?: boolean }) => {
  if (restorePromise) return restorePromise
  restorePromise = (async () => {
    let sourceID = 0
    let targetID = 0
    try {
      sourceID = ensureSourceId()
      targetID = resolveTargetId()
    } catch (err) {
      console.warn(err)
      return { attempted: 0, failed: 0 }
    }
    if (!targetID) return { attempted: 0, failed: 0 }
    const restoreKey = `${sourceID}:${targetID}`
    if (!options?.force && lastRestoreKey === restoreKey) return { attempted: 0, failed: 0 }

    const sources = state.localSources.slice()
    const consumers = state.localConsumers.slice()
    let failed = 0

    for (const source of sources) {
      try {
        const resp = await callStream<any>("AnnounceSimple", sourceID, targetID, buildSourcePayload(source))
        const restored = normalizeSource(resp?.source)
        if (restored) state.localSources = upsertSourceList(state.localSources, { ...restored, producer: state.selfNodeId || restored.producer })
      } catch (err) {
        console.warn(err)
        failed += 1
      }
    }

    for (const consumer of consumers) {
      try {
        const resp = await callStream<any>("AnnounceConsumerSimple", sourceID, targetID, buildConsumerPayload(consumer))
        const restored = normalizeConsumer(resp?.consumer_endpoint)
        if (restored) state.localConsumers = upsertConsumerList(state.localConsumers, { ...restored, consumer: state.selfNodeId || restored.consumer })
      } catch (err) {
        console.warn(err)
        failed += 1
      }
    }

    lastRestoreKey = restoreKey
    touchSync()
    return { attempted: sources.length + consumers.length, failed }
  })()
  try {
    return await restorePromise
  } finally {
    restorePromise = null
  }
}

const textFramesFor = (deliveryId: string) => state.textFramesByDelivery[String(deliveryId ?? "").trim()] ?? []
const statsFor = (deliveryId: string) => state.statsByDelivery[String(deliveryId ?? "").trim()] ?? null
const sourceById = (sourceId: string, scope: "local" | "catalog" | "any" = "any") => {
  const normalized = String(sourceId ?? "").trim()
  if (!normalized) return null
  if (scope === "local") return state.localSources.find((item) => item.sourceId === normalized) ?? null
  if (scope === "catalog") return state.sources.find((item) => item.sourceId === normalized) ?? null
  return state.localSources.find((item) => item.sourceId === normalized) ?? state.sources.find((item) => item.sourceId === normalized) ?? null
}

const consumerById = (consumerId: string, scope: "local" | "catalog" | "any" = "any") => {
  const normalized = String(consumerId ?? "").trim()
  if (!normalized) return null
  if (scope === "local") return state.localConsumers.find((item) => item.consumerId === normalized) ?? null
  if (scope === "catalog") return state.consumers.find((item) => item.consumerId === normalized) ?? null
  return state.localConsumers.find((item) => item.consumerId === normalized) ?? state.consumers.find((item) => item.consumerId === normalized) ?? null
}

const deliveriesForSource = (sourceId: string) => state.deliveries.filter((item) => item.sourceId === String(sourceId ?? "").trim())
const deliveriesForConsumer = (consumerId: string) => state.deliveries.filter((item) => item.consumerId === String(consumerId ?? "").trim())

const selectSource = (sourceId: string) => {
  state.selectedSourceId = String(sourceId ?? "").trim()
}

const selectConsumer = (consumerId: string) => {
  state.selectedConsumerId = String(consumerId ?? "").trim()
}

const selectDelivery = (deliveryId: string) => {
  state.selectedDeliveryId = String(deliveryId ?? "").trim()
}

const setIdentity = (nodeId: number, hubId: number) => {
  const nextNodeID = Number(nodeId || 0)
  const nextHubID = Number(hubId || 0)
  const changed = state.selfNodeId !== nextNodeID || state.defaultTargetId !== nextHubID
  state.selfNodeId = nextNodeID
  state.defaultTargetId = nextHubID
  applyLocalIdentity()
  if (changed) resetRestoreState()
}

const setTargetId = (value: string) => {
  state.targetId = String(value ?? "").trim()
  if (!state.targetId || /^\d+$/.test(state.targetId)) void savePrefsBestEffort()
}

const setActiveTab = (tab: StreamTab) => {
  state.activeTab = normalizeTab(tab)
  void savePrefsBestEffort()
}

const ensureListeners = () => {
  if (initialized) return
  initialized = true
  EventsOn("stream.delivery", (evt: any) => {
    const delivery = normalizeDelivery(evt)
    if (!delivery) return
    upsertDelivery(delivery)
    touchEvent()
  })
  EventsOn("stream.text", (evt: any) => {
    const frame = normalizeTextFrame(evt)
    if (!frame) return
    appendTextFrame(frame)
    touchEvent()
  })
  EventsOn("stream.stats", (evt: any) => {
    const frame = normalizeStatsFrame(evt)
    if (!frame) return
    rememberStatsFrame(frame)
    const current = state.deliveries.find((item) => item.deliveryId === frame.deliveryId)
    if (current) {
      upsertDelivery({
        ...current,
        bytesIn: frame.bytesIn,
        framesIn: frame.framesIn,
        lastPosition: frame.lastPosition,
        lastPtsMs: frame.lastPtsMs,
        lastAckPos: frame.lastAckPos,
        lastFlags: frame.lastFlags,
        updatedAt: frame.updatedAt
      })
    }
    touchEvent()
  })
}

export const useStreamStore = () => {
  ensureListeners()
  return {
    state,
    announceConsumer,
    announceSource,
    connect,
    consumerById,
    deliveriesForConsumer,
    deliveriesForSource,
    disconnect,
    listConsumers,
    listSources,
    loadDeliveries,
    loadPrefs,
    publishText,
    resolveTargetId,
    restoreLocalCatalogs,
    savePrefs,
    selectConsumer,
    selectDelivery,
    selectSource,
    setActiveTab,
    setIdentity,
    setTargetId,
    signal,
    sourceById,
    statsFor,
    subscribe,
    textFramesFor,
    unsubscribe,
    withdrawConsumer,
    withdrawSource
  }
}
