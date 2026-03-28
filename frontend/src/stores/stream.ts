import { reactive } from "vue"
import { t } from "@/i18n"
import { EventsOn } from "../../wailsjs/runtime/runtime"

type WailsBinding = (...args: any[]) => Promise<any>

const callStream = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.stream?.StreamService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(t("Stream binding '{method}' unavailable", { method }))
  }
  return fn(...args)
}

export const streamKinds = ["music", "video", "text", "custom"] as const
export type StreamKind = (typeof streamKinds)[number]

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

export type StreamState = {
  targetId: string
  selfNodeId: number
  defaultTargetId: number
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
  targetId: "",
  selfNodeId: 0,
  defaultTargetId: 0,
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

const nowIso = () => new Date().toISOString()

const normalizeNodeText = (value: string, field: string) => {
  const trimmed = String(value ?? "").trim()
  if (!trimmed) {
    throw new Error(t("{field} is required.", { field }))
  }
  const parsed = Number.parseInt(trimmed, 10)
  if (!Number.isFinite(parsed) || parsed <= 0) {
    throw new Error(t("{field} must be a positive number.", { field }))
  }
  return parsed
}

const normalizeTargetId = (value: string) => {
  const trimmed = String(value ?? "").trim()
  if (!trimmed) return state.defaultTargetId
  const parsed = Number.parseInt(trimmed, 10)
  if (!Number.isFinite(parsed) || parsed <= 0) {
    throw new Error(t("Target Node ID must be a positive number."))
  }
  return parsed
}

const ensureSourceId = () => {
  if (!state.selfNodeId) {
    throw new Error(t("Login required to use Stream controls."))
  }
  return state.selfNodeId
}

const resolveTargetId = () => normalizeTargetId(state.targetId)

const normalizeTags = (value: string) => {
  const seen = new Set<string>()
  const out: string[] = []
  for (const item of String(value ?? "").split(/[\n,，;；]+/g)) {
    const tag = item.trim()
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
  const sourceId = String(input?.source_id ?? "").trim()
  if (!sourceId) return null
  return {
    sourceId,
    producer: Number(input?.producer ?? 0),
    name: String(input?.name ?? "").trim(),
    kind: String(input?.kind ?? "").trim(),
    contentType: String(input?.content_type ?? "").trim(),
    mode: String(input?.mode ?? "").trim(),
    unitMode: String(input?.unit_mode ?? "").trim(),
    tags: Array.isArray(input?.tags)
      ? input.tags.map((item: unknown) => String(item ?? "").trim()).filter(Boolean)
      : [],
    metadataRaw: formatMetadata(input?.metadata)
  }
}

const normalizeConsumer = (input: any): StreamConsumer | null => {
  const consumerId = String(input?.consumer_id ?? "").trim()
  if (!consumerId) return null
  return {
    consumerId,
    consumer: Number(input?.consumer ?? 0),
    name: String(input?.name ?? "").trim(),
    kind: String(input?.kind ?? "").trim(),
    contentType: String(input?.content_type ?? "").trim(),
    tags: Array.isArray(input?.tags)
      ? input.tags.map((item: unknown) => String(item ?? "").trim()).filter(Boolean)
      : [],
    metadataRaw: formatMetadata(input?.metadata)
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

const upsertDelivery = (delivery: StreamDelivery) => {
  const next = [...state.deliveries]
  const index = next.findIndex((item) => item.deliveryId === delivery.deliveryId)
  if (index >= 0) {
    next[index] = { ...next[index], ...delivery }
  } else {
    next.push(delivery)
  }
  next.sort((left, right) => String(right.updatedAt).localeCompare(String(left.updatedAt)))
  state.deliveries = next
  if (!state.selectedDeliveryId) {
    state.selectedDeliveryId = delivery.deliveryId
  }
}

const appendTextFrame = (frame: StreamTextFrame) => {
  const current = Array.isArray(state.textFramesByDelivery[frame.deliveryId])
    ? [...state.textFramesByDelivery[frame.deliveryId]]
    : []
  current.push(frame)
  state.textFramesByDelivery[frame.deliveryId] = current.slice(-200)
}

const rememberStatsFrame = (frame: StreamStatsFrame) => {
  state.statsByDelivery[frame.deliveryId] = frame
}

const touchSync = () => {
  state.lastSyncAt = nowIso()
}

const touchEvent = () => {
  state.lastEventAt = nowIso()
}

const buildSourceDraftPayload = (draft: StreamSourceDraft) => ({
  req_id: "",
  source: {
    source_id: String(draft.sourceId ?? "").trim(),
    name: String(draft.name ?? "").trim(),
    kind: String(draft.kind ?? "").trim(),
    content_type: String(draft.contentType ?? "").trim(),
    mode: String(draft.mode ?? "").trim(),
    unit_mode: String(draft.unitMode ?? "").trim(),
    tags: normalizeTags(draft.tagsText),
    metadata: normalizeMetadata(draft.metadataText)
  }
})

const buildConsumerDraftPayload = (draft: StreamConsumerDraft) => ({
  req_id: "",
  consumer_endpoint: {
    consumer_id: String(draft.consumerId ?? "").trim(),
    name: String(draft.name ?? "").trim(),
    kind: String(draft.kind ?? "").trim(),
    content_type: String(draft.contentType ?? "").trim(),
    tags: normalizeTags(draft.tagsText),
    metadata: normalizeMetadata(draft.metadataText)
  }
})

const loadDeliveries = async () => {
  const snapshot = await callStream<any[]>("DeliverySnapshot")
  state.deliveries = Array.isArray(snapshot)
    ? snapshot.map(normalizeDelivery).filter(Boolean) as StreamDelivery[]
    : []
  touchSync()
}

const listSources = async (producerText: string, kind = "", tag = "") => {
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  const producer = normalizeNodeText(producerText, t("Producer Node ID"))
  const resp = await callStream<any>("ListSourcesSimple", sourceID, targetID, {
    req_id: "",
    producer,
    kind: String(kind ?? "").trim(),
    tag: String(tag ?? "").trim()
  })
  state.sources = Array.isArray(resp?.sources)
    ? resp.sources.map(normalizeSource).filter(Boolean) as StreamSource[]
    : []
  if (state.selectedSourceId && !state.sources.some((item) => item.sourceId === state.selectedSourceId)) {
    state.selectedSourceId = ""
  }
  touchSync()
  return state.sources
}

const listConsumers = async (consumerText: string, kind = "", tag = "") => {
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  const consumer = normalizeNodeText(consumerText, t("Consumer Node ID"))
  const resp = await callStream<any>("ListConsumersSimple", sourceID, targetID, {
    req_id: "",
    consumer,
    kind: String(kind ?? "").trim(),
    tag: String(tag ?? "").trim()
  })
  state.consumers = Array.isArray(resp?.consumer_endpoints)
    ? resp.consumer_endpoints.map(normalizeConsumer).filter(Boolean) as StreamConsumer[]
    : []
  if (state.selectedConsumerId && !state.consumers.some((item) => item.consumerId === state.selectedConsumerId)) {
    state.selectedConsumerId = ""
  }
  touchSync()
  return state.consumers
}

const announceSource = async (draft: StreamSourceDraft) => {
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  const resp = await callStream<any>("AnnounceSimple", sourceID, targetID, buildSourceDraftPayload(draft))
  const source = normalizeSource(resp?.source)
  if (source) {
    state.sources = [source, ...state.sources.filter((item) => item.sourceId !== source.sourceId)]
    state.selectedSourceId = source.sourceId
  }
  touchSync()
  return source
}

const withdrawSource = async (sourceId: string) => {
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  await callStream("WithdrawSimple", sourceID, targetID, {
    req_id: "",
    source_id: String(sourceId ?? "").trim()
  })
  state.sources = state.sources.filter((item) => item.sourceId !== sourceId)
  if (state.selectedSourceId === sourceId) {
    state.selectedSourceId = ""
  }
  touchSync()
}

const announceConsumer = async (draft: StreamConsumerDraft) => {
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  const resp = await callStream<any>("AnnounceConsumerSimple", sourceID, targetID, buildConsumerDraftPayload(draft))
  const consumer = normalizeConsumer(resp?.consumer_endpoint)
  if (consumer) {
    state.consumers = [consumer, ...state.consumers.filter((item) => item.consumerId !== consumer.consumerId)]
    state.selectedConsumerId = consumer.consumerId
  }
  touchSync()
  return consumer
}

const withdrawConsumer = async (consumerId: string) => {
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  await callStream("WithdrawConsumerSimple", sourceID, targetID, {
    req_id: "",
    consumer_id: String(consumerId ?? "").trim()
  })
  state.consumers = state.consumers.filter((item) => item.consumerId !== consumerId)
  if (state.selectedConsumerId === consumerId) {
    state.selectedConsumerId = ""
  }
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
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  await callStream("DisconnectSimple", sourceID, targetID, {
    req_id: "",
    delivery_id: String(deliveryId ?? "").trim(),
    reason: String(reason ?? "").trim()
  })
  upsertDelivery({
    ...(state.deliveries.find((item) => item.deliveryId === deliveryId) ?? {
      deliveryId,
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
  const sourceID = ensureSourceId()
  const targetID = resolveTargetId()
  await callStream("UnsubscribeSimple", sourceID, targetID, {
    req_id: "",
    delivery_id: String(deliveryId ?? "").trim(),
    reason: String(reason ?? "").trim()
  })
  upsertDelivery({
    ...(state.deliveries.find((item) => item.deliveryId === deliveryId) ?? {
      deliveryId,
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

const textFramesFor = (deliveryId: string) => state.textFramesByDelivery[String(deliveryId ?? "").trim()] ?? []
const statsFor = (deliveryId: string) => state.statsByDelivery[String(deliveryId ?? "").trim()] ?? null

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
  state.selfNodeId = Number(nodeId || 0)
  state.defaultTargetId = Number(hubId || 0)
}

const setTargetId = (value: string) => {
  state.targetId = String(value ?? "").trim()
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
    disconnect,
    listConsumers,
    listSources,
    loadDeliveries,
    resolveTargetId,
    selectConsumer,
    selectDelivery,
    selectSource,
    setIdentity,
    setTargetId,
    signal,
    statsFor,
    subscribe,
    textFramesFor,
    unsubscribe,
    withdrawConsumer,
    withdrawSource
  }
}
