// 本文件维护 `stream` store，并让它与 Wails 绑定及共享前端状态保持同步。

import { reactive } from "vue";
import { t } from "@/i18n";
import { EventsOn } from "../../wailsjs/runtime/runtime";

type WailsBinding = (...args: any[]) => Promise<any>;

const callApp = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.main?.App;
  const fn: WailsBinding | undefined = api?.[method];
  if (!fn) throw new Error(t("App binding '{method}' unavailable", { method }));
  return fn(...args);
};

const callStream = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.stream?.StreamService;
  const fn: WailsBinding | undefined = api?.[method];
  if (!fn)
    throw new Error(t("Stream binding '{method}' unavailable", { method }));
  return fn(...args);
};

export const streamKinds = ["music", "video", "text", "custom"] as const;
export type StreamKind = (typeof streamKinds)[number];
export type StreamTab = "source" | "consumer" | "control";

export type StreamSource = {
  sourceId: string;
  producer: number;
  name: string;
  kind: string;
  contentType: string;
  mode: string;
  unitMode: string;
  tags: string[];
  metadataRaw: string;
  inputKind: string;
  filePath: string;
};

export type StreamConsumer = {
  consumerId: string;
  consumer: number;
  name: string;
  kind: string;
  contentType: string;
  tags: string[];
  metadataRaw: string;
};

export type StreamDelivery = {
  deliveryId: string;
  sourceId: string;
  producer: number;
  consumer: number;
  consumerId: string;
  kind: string;
  contentType: string;
  mode: string;
  unitMode: string;
  state: string;
  bytesIn: number;
  framesIn: number;
  lastPosition: number;
  lastPtsMs: number;
  lastAckPos: number;
  lastFlags: number;
  lastError: string;
  updatedAt: string;
};

export type StreamTextFrame = {
  deliveryId: string;
  kind: string;
  text: string;
  position: number;
  ptsMs: number;
  flags: number;
  updatedAt: string;
};

export type StreamStatsFrame = {
  deliveryId: string;
  kind: string;
  bytesIn: number;
  framesIn: number;
  lastPosition: number;
  lastPtsMs: number;
  lastAckPos: number;
  lastFlags: number;
  updatedAt: string;
};

export type StreamSourceDraft = {
  sourceId: string;
  name: string;
  kind: string;
  contentType: string;
  mode: string;
  unitMode: string;
  tagsText: string;
  metadataText: string;
  inputKind: string;
  filePath: string;
};

export type StreamConsumerDraft = {
  consumerId: string;
  name: string;
  kind: string;
  contentType: string;
  tagsText: string;
  metadataText: string;
};

export type StreamRestoreResult = { attempted: number; failed: number };
export type StreamPublishTextResult = {
  sourceId: string;
  sent: number;
  deliveryIds: string[];
};
export type StreamPublishCaptureResult = {
  sourceId: string;
  sent: number;
  deliveryIds: string[];
};
export type StreamMediaFileChoice = {
  path: string;
  name: string;
  sizeBytes: number;
  kind: string;
  contentType: string;
};
export type StreamMediaState = {
  deliveryId: string;
  kind: string;
  contentType: string;
  state: string;
  mediaUrl: string;
  availableBytes: number;
  complete: boolean;
  error: string;
  updatedAt: string;
};

type StreamState = {
  activeTab: StreamTab;
  targetId: string;
  selfNodeId: number;
  defaultTargetId: number;
  localSources: StreamSource[];
  localConsumers: StreamConsumer[];
  sources: StreamSource[];
  consumers: StreamConsumer[];
  deliveries: StreamDelivery[];
  selectedSourceId: string;
  selectedConsumerId: string;
  selectedDeliveryId: string;
  lastSyncAt: string;
  lastEventAt: string;
  textFramesByDelivery: Record<string, StreamTextFrame[]>;
  statsByDelivery: Record<string, StreamStatsFrame>;
  mediaByDelivery: Record<string, StreamMediaState>;
};

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
  statsByDelivery: {},
  mediaByDelivery: {},
});

let initialized = false;
let restorePromise: Promise<StreamRestoreResult> | null = null;
let lastRestoreKey = "";

const nowIso = () => new Date().toISOString();

const normalizeNodeText = (value: string, field: string) => {
  const trimmed = String(value ?? "").trim();
  if (!trimmed) throw new Error(t("{field} is required.", { field }));
  const parsed = Number.parseInt(trimmed, 10);
  if (!Number.isFinite(parsed) || parsed <= 0)
    throw new Error(t("{field} must be a positive number.", { field }));
  return parsed;
};

const normalizeConfiguredTargetId = (value: unknown) => {
  const trimmed = String(value ?? "").trim();
  if (!trimmed) return "";
  const parsed = Number.parseInt(trimmed, 10);
  if (!Number.isFinite(parsed) || parsed <= 0)
    throw new Error(t("Target Node ID must be a positive number."));
  return String(parsed);
};

const normalizeTargetId = (value: string) => {
  const normalized = normalizeConfiguredTargetId(value);
  return normalized ? Number.parseInt(normalized, 10) : state.defaultTargetId;
};

const normalizeTab = (value: unknown): StreamTab => {
  switch (
    String(value ?? "")
      .trim()
      .toLowerCase()
  ) {
    case "consumer":
      return "consumer";
    case "control":
      return "control";
    default:
      return "source";
  }
};

const ensureSourceId = () => {
  if (!state.selfNodeId)
    throw new Error(t("Login required to use Stream controls."));
  return state.selfNodeId;
};

const resolveTargetId = () => normalizeTargetId(state.targetId);

const normalizeTags = (value: string | string[]) => {
  const items = Array.isArray(value)
    ? value
    : String(value ?? "").split(/[\n,，;；]+/g);
  const seen = new Set<string>();
  const out: string[] = [];
  for (const item of items) {
    const tag = String(item ?? "").trim();
    if (!tag || seen.has(tag)) continue;
    seen.add(tag);
    out.push(tag);
  }
  return out;
};

const normalizeMetadata = (value: string) => {
  const trimmed = String(value ?? "").trim();
  if (!trimmed) return undefined;
  try {
    return JSON.parse(trimmed);
  } catch {
    throw new Error(t("Metadata must be valid JSON."));
  }
};

const formatMetadata = (value: any) => {
  if (value === null || value === undefined || value === "") return "";
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) return "";
    if (trimmed.startsWith("{") || trimmed.startsWith("[")) {
      try {
        return JSON.stringify(JSON.parse(trimmed), null, 2);
      } catch {
        return trimmed;
      }
    }
    return trimmed;
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
};

const normalizeSource = (input: any): StreamSource | null => {
  const sourceId = String(input?.sourceId ?? input?.source_id ?? "").trim();
  if (!sourceId) return null;
  return {
    sourceId,
    producer: Number(input?.producer ?? state.selfNodeId ?? 0),
    name: String(input?.name ?? "").trim(),
    kind: String(input?.kind ?? "").trim(),
    contentType: String(input?.contentType ?? input?.content_type ?? "").trim(),
    mode: String(input?.mode ?? "").trim(),
    unitMode: String(input?.unitMode ?? input?.unit_mode ?? "").trim(),
    tags: Array.isArray(input?.tags)
      ? input.tags
          .map((item: unknown) => String(item ?? "").trim())
          .filter(Boolean)
      : [],
    metadataRaw: formatMetadata(input?.metadataRaw ?? input?.metadata),
    inputKind: String(input?.inputKind ?? input?.input_kind ?? "").trim(),
    filePath: String(input?.filePath ?? input?.file_path ?? "").trim(),
  };
};

const normalizeConsumer = (input: any): StreamConsumer | null => {
  const consumerId = String(
    input?.consumerId ?? input?.consumer_id ?? "",
  ).trim();
  if (!consumerId) return null;
  return {
    consumerId,
    consumer: Number(input?.consumer ?? state.selfNodeId ?? 0),
    name: String(input?.name ?? "").trim(),
    kind: String(input?.kind ?? "").trim(),
    contentType: String(input?.contentType ?? input?.content_type ?? "").trim(),
    tags: Array.isArray(input?.tags)
      ? input.tags
          .map((item: unknown) => String(item ?? "").trim())
          .filter(Boolean)
      : [],
    metadataRaw: formatMetadata(input?.metadataRaw ?? input?.metadata),
  };
};

const normalizeDelivery = (input: any): StreamDelivery | null => {
  const deliveryId = String(
    input?.deliveryId ?? input?.delivery_id ?? "",
  ).trim();
  if (!deliveryId) return null;
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
    updatedAt:
      String(input?.updatedAt ?? input?.updated_at ?? nowIso()).trim() ||
      nowIso(),
  };
};

const normalizeTextFrame = (input: any): StreamTextFrame | null => {
  const deliveryId = String(
    input?.deliveryId ?? input?.delivery_id ?? "",
  ).trim();
  if (!deliveryId) return null;
  return {
    deliveryId,
    kind: String(input?.kind ?? "").trim(),
    text: String(input?.text ?? ""),
    position: Number(input?.position ?? 0),
    ptsMs: Number(input?.ptsMs ?? input?.pts_ms ?? 0),
    flags: Number(input?.flags ?? 0),
    updatedAt:
      String(input?.updatedAt ?? input?.updated_at ?? nowIso()).trim() ||
      nowIso(),
  };
};

const normalizeStatsFrame = (input: any): StreamStatsFrame | null => {
  const deliveryId = String(
    input?.deliveryId ?? input?.delivery_id ?? "",
  ).trim();
  if (!deliveryId) return null;
  return {
    deliveryId,
    kind: String(input?.kind ?? "").trim(),
    bytesIn: Number(input?.bytesIn ?? input?.bytes_in ?? 0),
    framesIn: Number(input?.framesIn ?? input?.frames_in ?? 0),
    lastPosition: Number(input?.lastPosition ?? input?.last_position ?? 0),
    lastPtsMs: Number(input?.lastPtsMs ?? input?.last_pts_ms ?? 0),
    lastAckPos: Number(input?.lastAckPos ?? input?.last_ack_pos ?? 0),
    lastFlags: Number(input?.lastFlags ?? input?.last_flags ?? 0),
    updatedAt:
      String(input?.updatedAt ?? input?.updated_at ?? nowIso()).trim() ||
      nowIso(),
  };
};

const normalizeMediaState = (input: any): StreamMediaState | null => {
  const deliveryId = String(
    input?.deliveryId ?? input?.delivery_id ?? "",
  ).trim();
  if (!deliveryId) return null;
  return {
    deliveryId,
    kind: String(input?.kind ?? "").trim(),
    contentType: String(input?.contentType ?? input?.content_type ?? "").trim(),
    state: String(input?.state ?? "").trim(),
    mediaUrl: String(input?.mediaUrl ?? input?.media_url ?? "").trim(),
    availableBytes: Number(
      input?.availableBytes ?? input?.available_bytes ?? 0,
    ),
    complete: Boolean(input?.complete),
    error: String(input?.error ?? "").trim(),
    updatedAt:
      String(input?.updatedAt ?? input?.updated_at ?? nowIso()).trim() ||
      nowIso(),
  };
};

const upsertSourceList = (list: StreamSource[], source: StreamSource) => {
  const next = [...list];
  const index = next.findIndex((item) => item.sourceId === source.sourceId);
  if (index >= 0) next[index] = { ...next[index], ...source };
  else next.unshift(source);
  return next;
};

const upsertConsumerList = (
  list: StreamConsumer[],
  consumer: StreamConsumer,
) => {
  const next = [...list];
  const index = next.findIndex(
    (item) => item.consumerId === consumer.consumerId,
  );
  if (index >= 0) next[index] = { ...next[index], ...consumer };
  else next.unshift(consumer);
  return next;
};

const removeSourceFromList = (list: StreamSource[], sourceId: string) =>
  list.filter((item) => item.sourceId !== String(sourceId ?? "").trim());
const removeConsumerFromList = (list: StreamConsumer[], consumerId: string) =>
  list.filter((item) => item.consumerId !== String(consumerId ?? "").trim());

const touchSync = () => {
  state.lastSyncAt = nowIso();
};

const touchEvent = () => {
  state.lastEventAt = nowIso();
};

const resetRestoreState = () => {
  lastRestoreKey = "";
};

// 本地保存的 source/consumer 会跨 profile 持久化；身份切换后这里统一回填当前 selfNodeId。
const applyLocalIdentity = () => {
  if (!state.selfNodeId) return;
  state.localSources = state.localSources.map((item) => ({
    ...item,
    producer: state.selfNodeId,
  }));
  state.localConsumers = state.localConsumers.map((item) => ({
    ...item,
    consumer: state.selfNodeId,
  }));
};

// Stream 偏好除了当前 tab，还包括“本地要恢复的 source/consumer 目录”，因此需要一并规范化。
const applyPrefs = (prefs: any) => {
  state.activeTab = normalizeTab(prefs?.activeTab);
  const targetId = Number(prefs?.targetId ?? 0);
  state.targetId =
    Number.isFinite(targetId) && targetId > 0
      ? String(Math.floor(targetId))
      : "";
  state.localSources = Array.isArray(prefs?.sources)
    ? (prefs.sources.map(normalizeSource).filter(Boolean) as StreamSource[])
    : [];
  state.localConsumers = Array.isArray(prefs?.consumers)
    ? (prefs.consumers
        .map(normalizeConsumer)
        .filter(Boolean) as StreamConsumer[])
    : [];
  applyLocalIdentity();
  resetRestoreState();
};

// 这里把前端草稿转成服务端期望的 announce 请求结构，统一处理 tags/metadata 的规范化。
const buildSourcePayload = (source: {
  sourceId: string;
  name: string;
  kind: string;
  contentType: string;
  mode: string;
  unitMode: string;
  tags: string[];
  metadataRaw: string;
}) => ({
  req_id: "",
  source: {
    source_id: String(source.sourceId ?? "").trim(),
    name: String(source.name ?? "").trim(),
    kind: String(source.kind ?? "").trim(),
    content_type: String(source.contentType ?? "").trim(),
    mode: String(source.mode ?? "").trim(),
    unit_mode: String(source.unitMode ?? "").trim(),
    tags: normalizeTags(source.tags),
    metadata: normalizeMetadata(source.metadataRaw),
  },
});

// consumer 的持久化格式和 announce 请求结构不同，单独在这里做一次协议层映射。
const buildConsumerPayload = (consumer: {
  consumerId: string;
  name: string;
  kind: string;
  contentType: string;
  tags: string[];
  metadataRaw: string;
}) => ({
  req_id: "",
  consumer_endpoint: {
    consumer_id: String(consumer.consumerId ?? "").trim(),
    name: String(consumer.name ?? "").trim(),
    kind: String(consumer.kind ?? "").trim(),
    content_type: String(consumer.contentType ?? "").trim(),
    tags: normalizeTags(consumer.tags),
    metadata: normalizeMetadata(consumer.metadataRaw),
  },
});

const buildSourceDraftPayload = (draft: StreamSourceDraft) =>
  buildSourcePayload({
    sourceId: draft.sourceId,
    name: draft.name,
    kind: draft.kind,
    contentType: draft.contentType,
    mode: draft.mode,
    unitMode: draft.unitMode,
    tags: normalizeTags(draft.tagsText),
    metadataRaw: draft.metadataText,
  });

const buildConsumerDraftPayload = (draft: StreamConsumerDraft) =>
  buildConsumerPayload({
    consumerId: draft.consumerId,
    name: draft.name,
    kind: draft.kind,
    contentType: draft.contentType,
    tags: normalizeTags(draft.tagsText),
    metadataRaw: draft.metadataText,
  });

const toAppSource = (source: StreamSource) => ({
  sourceId: source.sourceId,
  name: source.name,
  kind: source.kind,
  contentType: source.contentType,
  mode: source.mode,
  unitMode: source.unitMode,
  tags: source.tags,
  metadataRaw: source.metadataRaw,
  inputKind: source.inputKind,
  filePath: source.filePath,
});

const toAppConsumer = (consumer: StreamConsumer) => ({
  consumerId: consumer.consumerId,
  name: consumer.name,
  kind: consumer.kind,
  contentType: consumer.contentType,
  tags: consumer.tags,
  metadataRaw: consumer.metadataRaw,
});

const upsertDelivery = (delivery: StreamDelivery) => {
  const next = [...state.deliveries];
  const index = next.findIndex(
    (item) => item.deliveryId === delivery.deliveryId,
  );
  if (index >= 0) next[index] = { ...next[index], ...delivery };
  else next.push(delivery);
  next.sort((left, right) =>
    String(right.updatedAt).localeCompare(String(left.updatedAt)),
  );
  state.deliveries = next;
  if (!state.selectedDeliveryId) state.selectedDeliveryId = delivery.deliveryId;
};

const appendTextFrame = (frame: StreamTextFrame) => {
  const current = Array.isArray(state.textFramesByDelivery[frame.deliveryId])
    ? [...state.textFramesByDelivery[frame.deliveryId]]
    : [];
  current.push(frame);
  state.textFramesByDelivery[frame.deliveryId] = current.slice(-200);
};

const rememberStatsFrame = (frame: StreamStatsFrame) => {
  state.statsByDelivery[frame.deliveryId] = frame;
};

const rememberMediaState = (media: StreamMediaState) => {
  state.mediaByDelivery[media.deliveryId] = media;
};

const normalizeRequestedDeliveryIDs = (values: string[]) => {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const value of values) {
    const deliveryID = String(value ?? "").trim();
    if (!deliveryID || seen.has(deliveryID)) continue;
    seen.add(deliveryID);
    out.push(deliveryID);
  }
  return out;
};

const syncSourceInput = async (source: StreamSource) => {
  const sourceID = ensureSourceId();
  await callStream("ConfigureSourceInputSimple", sourceID, {
    source_id: source.sourceId,
    input_kind: String(source.inputKind ?? "").trim(),
    file_path: String(source.filePath ?? "").trim(),
  });
};

// 媒体类 source 在 prefs 里只保存文件路径，真正连上节点后还要再补一次输入配置。
const syncLocalSourceInputs = async () => {
  if (!state.selfNodeId) return;
  const mediaSources = state.localSources.filter(
    (item) => item.kind !== "text",
  );
  if (!mediaSources.length) return;
  for (const source of mediaSources) {
    await syncSourceInput(source);
  }
};

const savePrefs = async () => {
  const saved = await callApp<any>("SaveStreamPrefs", {
    activeTab: state.activeTab,
    targetId: state.targetId
      ? Number.parseInt(normalizeConfiguredTargetId(state.targetId), 10)
      : 0,
    sources: state.localSources.map(toAppSource),
    consumers: state.localConsumers.map(toAppConsumer),
  });
  applyPrefs(saved);
  return saved;
};

const savePrefsBestEffort = async () => {
  try {
    await savePrefs();
  } catch (err) {
    console.warn(err);
  }
};

// 读取 prefs 后立刻同步媒体输入，是为了让“恢复目录”后的 source 能直接继续推流。
const loadPrefs = async () => {
  const prefs = await callApp<any>("StreamPrefs");
  applyPrefs(prefs);
  try {
    await syncLocalSourceInputs();
  } catch (err) {
    console.warn(err);
  }
  return prefs;
};

const loadDeliveries = async () => {
  const snapshot = await callStream<any[]>("DeliverySnapshot");
  state.deliveries = Array.isArray(snapshot)
    ? (snapshot.map(normalizeDelivery).filter(Boolean) as StreamDelivery[])
    : [];
  touchSync();
  return state.deliveries;
};

const loadMedia = async () => {
  const snapshot = await callStream<any[]>("MediaSnapshot");
  state.mediaByDelivery = Array.isArray(snapshot)
    ? Object.fromEntries(
        (
          snapshot
            .map(normalizeMediaState)
            .filter(Boolean) as StreamMediaState[]
        ).map((item) => [item.deliveryId, item]),
      )
    : {};
  touchSync();
  return state.mediaByDelivery;
};

const pickMediaFile = async () => {
  const raw = await callApp<any>("PickStreamMediaFile");
  const path = String(raw?.path ?? "").trim();
  if (!path) return null;
  return {
    path,
    name: String(raw?.name ?? "").trim(),
    sizeBytes: Number(raw?.sizeBytes ?? raw?.size_bytes ?? 0),
    kind: String(raw?.kind ?? "").trim(),
    contentType: String(raw?.contentType ?? raw?.content_type ?? "").trim(),
  } satisfies StreamMediaFileChoice;
};

const updateSourceInput = async (
  sourceId: string,
  file: StreamMediaFileChoice | null,
) => {
  const normalized = String(sourceId ?? "").trim();
  const current = state.localSources.find(
    (item) => item.sourceId === normalized,
  );
  if (!current) throw new Error(t("Source not found."));
  if (current.kind === "text")
    throw new Error(t("Only media sources support file input."));
  const next = {
    ...current,
    inputKind: "",
    filePath: "",
  };
  if (file) {
    if (
      String(file.kind ?? "").trim() &&
      String(file.kind ?? "").trim() !== current.kind
    ) {
      throw new Error(
        t("Selected media file kind does not match the source kind."),
      );
    }
    if (
      String(current.contentType ?? "").trim() &&
      String(file.contentType ?? "").trim() &&
      current.contentType !== file.contentType
    ) {
      throw new Error(
        t(
          "Selected media file content type does not match the source content type.",
        ),
      );
    }
    next.inputKind = "file";
    next.filePath = String(file.path ?? "").trim();
  }
  state.localSources = upsertSourceList(state.localSources, next);
  await savePrefs();
  await syncSourceInput(next);
  return next;
};

const listSources = async (
  producerText: string,
  kind = "",
  tag = "",
  scope: "catalog" | "local" = "catalog",
) => {
  const sourceID = ensureSourceId();
  const targetID = resolveTargetId();
  const producer = normalizeNodeText(producerText, t("Producer Node ID"));
  const resp = await callStream<any>("ListSourcesSimple", sourceID, targetID, {
    req_id: "",
    producer,
    kind: String(kind ?? "").trim(),
    tag: String(tag ?? "").trim(),
  });
  const items = Array.isArray(resp?.sources)
    ? (resp.sources.map(normalizeSource).filter(Boolean) as StreamSource[])
    : [];
  if (scope === "local") {
    state.localSources = items.map((item) => ({
      ...item,
      producer: state.selfNodeId || item.producer,
    }));
  } else {
    state.sources = items;
    if (
      state.selectedSourceId &&
      !state.sources.some((item) => item.sourceId === state.selectedSourceId)
    )
      state.selectedSourceId = "";
  }
  touchSync();
  return items;
};

const listConsumers = async (
  consumerText: string,
  kind = "",
  tag = "",
  scope: "catalog" | "local" = "catalog",
) => {
  const sourceID = ensureSourceId();
  const targetID = resolveTargetId();
  const consumer = normalizeNodeText(consumerText, t("Consumer Node ID"));
  const resp = await callStream<any>(
    "ListConsumersSimple",
    sourceID,
    targetID,
    {
      req_id: "",
      consumer,
      kind: String(kind ?? "").trim(),
      tag: String(tag ?? "").trim(),
    },
  );
  const items = Array.isArray(resp?.consumer_endpoints)
    ? (resp.consumer_endpoints
        .map(normalizeConsumer)
        .filter(Boolean) as StreamConsumer[])
    : [];
  if (scope === "local") {
    state.localConsumers = items.map((item) => ({
      ...item,
      consumer: state.selfNodeId || item.consumer,
    }));
  } else {
    state.consumers = items;
    if (
      state.selectedConsumerId &&
      !state.consumers.some(
        (item) => item.consumerId === state.selectedConsumerId,
      )
    )
      state.selectedConsumerId = "";
  }
  touchSync();
  return items;
};

// source 发布时在 UI 层先拦住 desktop/file 等输入约束，减少服务端返回后才报错的来回。
const announceSource = async (draft: StreamSourceDraft) => {
  const inputKind = String(draft.inputKind ?? "")
    .trim()
    .toLowerCase();
  if (inputKind === "desktop" && draft.kind !== "video") {
    throw new Error(t("Desktop capture is only available for video sources."));
  }
  if (
    draft.kind !== "text" &&
    inputKind !== "desktop" &&
    !String(draft.filePath ?? "").trim()
  ) {
    throw new Error(t("A media file is required for non-text sources."));
  }
  const sourceID = ensureSourceId();
  const targetID = resolveTargetId();
  const resp = await callStream<any>(
    "AnnounceSimple",
    sourceID,
    targetID,
    buildSourceDraftPayload(draft),
  );
  const source = normalizeSource(resp?.source);
  if (source) {
    const nextSource = {
      ...source,
      producer: state.selfNodeId || source.producer,
      inputKind,
      filePath: String(draft.filePath ?? "").trim(),
    };
    state.localSources = upsertSourceList(state.localSources, nextSource);
    state.sources = upsertSourceList(state.sources, source);
    state.selectedSourceId = source.sourceId;
    await savePrefs();
    if (nextSource.kind !== "text") {
      try {
        await syncSourceInput(nextSource);
      } catch (err) {
        console.warn(err);
      }
    }
  }
  touchSync();
  return source;
};

const withdrawSource = async (sourceId: string) => {
  const normalized = String(sourceId ?? "").trim();
  const sourceID = ensureSourceId();
  const targetID = resolveTargetId();
  await callStream("WithdrawSimple", sourceID, targetID, {
    req_id: "",
    source_id: normalized,
  });
  state.localSources = removeSourceFromList(state.localSources, normalized);
  state.sources = removeSourceFromList(state.sources, normalized);
  if (state.selectedSourceId === normalized) state.selectedSourceId = "";
  await savePrefs();
  touchSync();
};

const announceConsumer = async (draft: StreamConsumerDraft) => {
  const sourceID = ensureSourceId();
  const targetID = resolveTargetId();
  const resp = await callStream<any>(
    "AnnounceConsumerSimple",
    sourceID,
    targetID,
    buildConsumerDraftPayload(draft),
  );
  const consumer = normalizeConsumer(resp?.consumer_endpoint);
  if (consumer) {
    state.localConsumers = upsertConsumerList(state.localConsumers, {
      ...consumer,
      consumer: state.selfNodeId || consumer.consumer,
    });
    state.consumers = upsertConsumerList(state.consumers, consumer);
    state.selectedConsumerId = consumer.consumerId;
    await savePrefs();
  }
  touchSync();
  return consumer;
};

const withdrawConsumer = async (consumerId: string) => {
  const normalized = String(consumerId ?? "").trim();
  const sourceID = ensureSourceId();
  const targetID = resolveTargetId();
  await callStream("WithdrawConsumerSimple", sourceID, targetID, {
    req_id: "",
    consumer_id: normalized,
  });
  state.localConsumers = removeConsumerFromList(
    state.localConsumers,
    normalized,
  );
  state.consumers = removeConsumerFromList(state.consumers, normalized);
  if (state.selectedConsumerId === normalized) state.selectedConsumerId = "";
  await savePrefs();
  touchSync();
};

const connect = async (input: {
  producer: number;
  sourceId: string;
  consumer: number;
  consumerId: string;
}) => {
  const sourceID = ensureSourceId();
  const targetID = resolveTargetId();
  const resp = await callStream<any>("ConnectSimple", sourceID, targetID, {
    req_id: "",
    producer: Number(input.producer || 0),
    source_id: String(input.sourceId ?? "").trim(),
    consumer: Number(input.consumer || 0),
    consumer_id: String(input.consumerId ?? "").trim(),
  });
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
    updatedAt: nowIso(),
  });
  if (delivery) {
    upsertDelivery(delivery);
    state.selectedDeliveryId = delivery.deliveryId;
  }
  touchSync();
  return delivery;
};

const subscribe = async (input: {
  producer: number;
  sourceId: string;
  consumerId: string;
}) => {
  const sourceID = ensureSourceId();
  const targetID = resolveTargetId();
  const resp = await callStream<any>("SubscribeSimple", sourceID, targetID, {
    req_id: "",
    producer: Number(input.producer || 0),
    source_id: String(input.sourceId ?? "").trim(),
    consumer_id: String(input.consumerId ?? "").trim(),
  });
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
    updatedAt: nowIso(),
  });
  if (delivery) {
    upsertDelivery(delivery);
    state.selectedDeliveryId = delivery.deliveryId;
  }
  touchSync();
  return delivery;
};

const disconnect = async (deliveryId: string, reason = "") => {
  const normalized = String(deliveryId ?? "").trim();
  const sourceID = ensureSourceId();
  const targetID = resolveTargetId();
  await callStream("DisconnectSimple", sourceID, targetID, {
    req_id: "",
    delivery_id: normalized,
    reason: String(reason ?? "").trim(),
  });
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
      lastError: "",
    }),
    state: "closed",
    updatedAt: nowIso(),
  });
  touchSync();
};

const unsubscribe = async (deliveryId: string, reason = "") => {
  const normalized = String(deliveryId ?? "").trim();
  const sourceID = ensureSourceId();
  const targetID = resolveTargetId();
  await callStream("UnsubscribeSimple", sourceID, targetID, {
    req_id: "",
    delivery_id: normalized,
    reason: String(reason ?? "").trim(),
  });
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
      lastError: "",
    }),
    state: "closed",
    updatedAt: nowIso(),
  });
  touchSync();
};

const signal = async (deliveryId: string, op: string, data?: unknown) => {
  const sourceID = ensureSourceId();
  const targetID = resolveTargetId();
  await callStream("SignalSimple", sourceID, targetID, {
    req_id: "",
    delivery_id: String(deliveryId ?? "").trim(),
    op: String(op ?? "").trim(),
    data,
  });
  touchSync();
};

const publishText = async (sourceId: string, text: string) => {
  const normalizedSourceID = String(sourceId ?? "").trim();
  const normalizedText = String(text ?? "");
  if (!normalizedSourceID) throw new Error(t("Source ID is required."));
  if (!normalizedText.trim()) throw new Error(t("Text content is required."));
  const source = state.localSources.find(
    (item) => item.sourceId === normalizedSourceID,
  );
  if (!source) throw new Error(t("Source not found."));
  if (source.kind !== "text")
    throw new Error(t("Only text sources support direct input."));
  const sourceID = ensureSourceId();
  const resp = await callStream<any>("PublishTextSimple", sourceID, {
    source_id: normalizedSourceID,
    text: normalizedText,
  });
  touchSync();
  return {
    sourceId:
      String(resp?.source_id ?? normalizedSourceID).trim() ||
      normalizedSourceID,
    sent: Number(resp?.sent ?? 0),
    deliveryIds: Array.isArray(resp?.delivery_ids)
      ? resp.delivery_ids
          .map((item: unknown) => String(item ?? "").trim())
          .filter(Boolean)
      : [],
  } satisfies StreamPublishTextResult;
};

// 桌面采集分片只允许视频 desktop source 使用，并要求当前至少存在一个有效 delivery。
const publishCaptureChunk = async (input: {
  sourceId: string;
  deliveryIds: string[];
  payload?: ArrayLike<number>;
  ptsMs?: number;
  final?: boolean;
  sessionStart?: boolean;
}) => {
  const normalizedSourceID = String(input.sourceId ?? "").trim();
  if (!normalizedSourceID) throw new Error(t("Source ID is required."));
  const source = state.localSources.find(
    (item) => item.sourceId === normalizedSourceID,
  );
  if (!source) throw new Error(t("Source not found."));
  if (
    source.kind !== "video" ||
    String(source.inputKind ?? "")
      .trim()
      .toLowerCase() !== "desktop"
  ) {
    throw new Error(t("Only desktop video sources support capture input."));
  }
  const deliveryIds = normalizeRequestedDeliveryIDs(input.deliveryIds);
  if (!deliveryIds.length)
    throw new Error(
      t("Desktop capture requires at least one active delivery."),
    );
  const payload = Array.from(
    input.payload ?? [],
    (value) => Number(value ?? 0) & 0xff,
  );
  if (!payload.length && !input.final) {
    throw new Error(
      t("Capture payload is required unless the chunk is final."),
    );
  }
  const sourceID = ensureSourceId();
  const ptsMs = Number(input.ptsMs ?? 0);
  const resp = await callStream<any>("PublishCaptureChunkSimple", sourceID, {
    source_id: normalizedSourceID,
    delivery_ids: deliveryIds,
    pts_ms: Number.isFinite(ptsMs) && ptsMs > 0 ? Math.trunc(ptsMs) : 0,
    session_start: Boolean(input.sessionStart),
    final: Boolean(input.final),
    payload,
  });
  touchSync();
  return {
    sourceId:
      String(resp?.source_id ?? normalizedSourceID).trim() ||
      normalizedSourceID,
    sent: Number(resp?.sent ?? 0),
    deliveryIds: Array.isArray(resp?.delivery_ids)
      ? resp.delivery_ids
          .map((item: unknown) => String(item ?? "").trim())
          .filter(Boolean)
      : [],
  } satisfies StreamPublishCaptureResult;
};

// 应用重开或重新登录后，通过重新 announce 本地目录把上次创建的 source/consumer 恢复回来。
const restoreLocalCatalogs = async (options?: { force?: boolean }) => {
  if (restorePromise) return restorePromise;
  restorePromise = (async () => {
    let sourceID = 0;
    let targetID = 0;
    try {
      sourceID = ensureSourceId();
      targetID = resolveTargetId();
    } catch (err) {
      console.warn(err);
      return { attempted: 0, failed: 0 };
    }
    if (!targetID) return { attempted: 0, failed: 0 };
    const restoreKey = `${sourceID}:${targetID}`;
    if (!options?.force && lastRestoreKey === restoreKey)
      return { attempted: 0, failed: 0 };

    const sources = state.localSources.slice();
    const consumers = state.localConsumers.slice();
    let failed = 0;

    for (const source of sources) {
      try {
        const resp = await callStream<any>(
          "AnnounceSimple",
          sourceID,
          targetID,
          buildSourcePayload(source),
        );
        const restored = normalizeSource(resp?.source);
        if (restored) {
          const nextSource = upsertSourceList(state.localSources, {
            ...source,
            ...restored,
            producer: state.selfNodeId || restored.producer,
          });
          state.localSources = nextSource;
          const savedSource = nextSource.find(
            (item) => item.sourceId === restored.sourceId,
          );
          if (savedSource && savedSource.kind !== "text") {
            await syncSourceInput(savedSource);
          }
        }
      } catch (err) {
        console.warn(err);
        failed += 1;
      }
    }

    for (const consumer of consumers) {
      try {
        const resp = await callStream<any>(
          "AnnounceConsumerSimple",
          sourceID,
          targetID,
          buildConsumerPayload(consumer),
        );
        const restored = normalizeConsumer(resp?.consumer_endpoint);
        if (restored)
          state.localConsumers = upsertConsumerList(state.localConsumers, {
            ...restored,
            consumer: state.selfNodeId || restored.consumer,
          });
      } catch (err) {
        console.warn(err);
        failed += 1;
      }
    }

    lastRestoreKey = restoreKey;
    touchSync();
    return { attempted: sources.length + consumers.length, failed };
  })();
  try {
    return await restorePromise;
  } finally {
    restorePromise = null;
  }
};

const textFramesFor = (deliveryId: string) =>
  state.textFramesByDelivery[String(deliveryId ?? "").trim()] ?? [];
const statsFor = (deliveryId: string) =>
  state.statsByDelivery[String(deliveryId ?? "").trim()] ?? null;
const mediaForDelivery = (deliveryId: string) =>
  state.mediaByDelivery[String(deliveryId ?? "").trim()] ?? null;
const sourceById = (
  sourceId: string,
  scope: "local" | "catalog" | "any" = "any",
) => {
  const normalized = String(sourceId ?? "").trim();
  if (!normalized) return null;
  if (scope === "local")
    return (
      state.localSources.find((item) => item.sourceId === normalized) ?? null
    );
  if (scope === "catalog")
    return state.sources.find((item) => item.sourceId === normalized) ?? null;
  return (
    state.localSources.find((item) => item.sourceId === normalized) ??
    state.sources.find((item) => item.sourceId === normalized) ??
    null
  );
};

const consumerById = (
  consumerId: string,
  scope: "local" | "catalog" | "any" = "any",
) => {
  const normalized = String(consumerId ?? "").trim();
  if (!normalized) return null;
  if (scope === "local")
    return (
      state.localConsumers.find((item) => item.consumerId === normalized) ??
      null
    );
  if (scope === "catalog")
    return (
      state.consumers.find((item) => item.consumerId === normalized) ?? null
    );
  return (
    state.localConsumers.find((item) => item.consumerId === normalized) ??
    state.consumers.find((item) => item.consumerId === normalized) ??
    null
  );
};

const deliveriesForSource = (sourceId: string) =>
  state.deliveries.filter(
    (item) => item.sourceId === String(sourceId ?? "").trim(),
  );
const deliveriesForConsumer = (consumerId: string) =>
  state.deliveries.filter(
    (item) => item.consumerId === String(consumerId ?? "").trim(),
  );

const selectSource = (sourceId: string) => {
  state.selectedSourceId = String(sourceId ?? "").trim();
};

const selectConsumer = (consumerId: string) => {
  state.selectedConsumerId = String(consumerId ?? "").trim();
};

const selectDelivery = (deliveryId: string) => {
  state.selectedDeliveryId = String(deliveryId ?? "").trim();
};

const setIdentity = (nodeId: number, hubId: number) => {
  const nextNodeID = Number(nodeId || 0);
  const nextHubID = Number(hubId || 0);
  const changed =
    state.selfNodeId !== nextNodeID || state.defaultTargetId !== nextHubID;
  state.selfNodeId = nextNodeID;
  state.defaultTargetId = nextHubID;
  applyLocalIdentity();
  if (changed) resetRestoreState();
};

const setTargetId = (value: string) => {
  state.targetId = String(value ?? "").trim();
  if (!state.targetId || /^\d+$/.test(state.targetId))
    void savePrefsBestEffort();
};

const setActiveTab = (tab: StreamTab) => {
  state.activeTab = normalizeTab(tab);
  void savePrefsBestEffort();
};

// 事件监听器只负责把 runtime 快照投影进 store，避免 UI 页面各自重复订阅 Wails 事件。
const ensureListeners = () => {
  if (initialized) return;
  initialized = true;
  EventsOn("stream.delivery", (evt: any) => {
    const delivery = normalizeDelivery(evt);
    if (!delivery) return;
    upsertDelivery(delivery);
    touchEvent();
  });
  EventsOn("stream.text", (evt: any) => {
    const frame = normalizeTextFrame(evt);
    if (!frame) return;
    appendTextFrame(frame);
    touchEvent();
  });
  EventsOn("stream.stats", (evt: any) => {
    const frame = normalizeStatsFrame(evt);
    if (!frame) return;
    rememberStatsFrame(frame);
    const current = state.deliveries.find(
      (item) => item.deliveryId === frame.deliveryId,
    );
    if (current) {
      upsertDelivery({
        ...current,
        bytesIn: frame.bytesIn,
        framesIn: frame.framesIn,
        lastPosition: frame.lastPosition,
        lastPtsMs: frame.lastPtsMs,
        lastAckPos: frame.lastAckPos,
        lastFlags: frame.lastFlags,
        updatedAt: frame.updatedAt,
      });
    }
    touchEvent();
  });
  EventsOn("stream.media", (evt: any) => {
    const media = normalizeMediaState(evt);
    if (!media) return;
    rememberMediaState(media);
    touchEvent();
  });
};

export const useStreamStore = () => {
  ensureListeners();
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
    loadMedia,
    loadPrefs,
    mediaForDelivery,
    pickMediaFile,
    publishCaptureChunk,
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
    updateSourceInput,
    unsubscribe,
    withdrawConsumer,
    withdrawSource,
  };
};
