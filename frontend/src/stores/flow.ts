// 本文件维护 `flow` store，并让它与 Wails 绑定及共享前端状态保持同步。

import { reactive } from "vue"
import { t } from "@/i18n"
import {
  FLOW_BINDING_SOURCE_KINDS as CANONICAL_FLOW_BINDING_SOURCE_KINDS,
  FLOW_BRANCH_MATCH_OPS as CANONICAL_FLOW_BRANCH_MATCH_OPS,
  FLOW_NODE_KINDS as CANONICAL_FLOW_NODE_KINDS,
  type FlowBindingSourceKind as CanonicalFlowBindingSourceKind,
  type FlowBranchMatchOp,
  type FlowNodeKind
} from "@/generated/flow_contract"
import { readValueAtPointer } from "./flow_json_pointer"
import { type MethodVisualSchema } from "./flow_method_schemas"
import { resolveMethodVisualSchema, type CapabilityRouteSchemaSource } from "./flow_schema_resolver"
import {
  buildNodeVisualFormModel,
  clearBindingForPointer,
  describeFieldBinding as describeVisualFieldBinding,
  setBindingForPointer,
  setLiteralFieldValue as setLiteralFieldValueInDoc,
  type NodeVisualFormModel,
  type VisualBindingSource
} from "./flow_visual_form"

export type { MethodFieldSchema, MethodVisualSchema } from "./flow_method_schemas"
export { describeFieldBinding, describeVisualCompatibilityReason } from "./flow_visual_form"
export type {
  FieldVisualState,
  FlowInputBindingLike,
  NodeVisualFormModel,
  VisualBindingSource,
  VisualCompatibility,
  VisualCompatibilityReason,
  VisualCompatibilityReasonCode,
  VisualFieldModel
} from "./flow_visual_form"

type WailsBinding = (...args: any[]) => Promise<any>

// callFlow 统一封装 Wails 绑定调用，并在绑定缺失时抛出可直接展示的错误。
const callFlow = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.flow?.FlowService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(t("Flow binding '{method}' unavailable", { method }))
  }
  return fn(...args)
}

export type FlowSummary = {
  flowId: string
  name: string
  everyMs: number
  lastRunId: string
  lastStatus: string
}

export type FlowTriggerType = "interval" | "cron" | "event" | "var_changed"
export type FlowSpecEditorMode = "form" | "json"
export type { FlowBranchMatchOp, FlowNodeKind } from "@/generated/flow_contract"
export type FlowBindingSourceKind = CanonicalFlowBindingSourceKind | ""
export const flowNodeKindOptions: FlowNodeKind[] = [...CANONICAL_FLOW_NODE_KINDS]
export const flowBindingSourceKindOptions: CanonicalFlowBindingSourceKind[] = [...CANONICAL_FLOW_BINDING_SOURCE_KINDS]
export const flowBranchMatchOpOptions: FlowBranchMatchOp[] = [...CANONICAL_FLOW_BRANCH_MATCH_OPS]
export const rootFlowBindingSourceKindOptions: CanonicalFlowBindingSourceKind[] = flowBindingSourceKindOptions.filter(
  (kind) => kind !== "loop_item" && kind !== "loop_index"
)
export const bodyFlowBindingSourceKindOptions: CanonicalFlowBindingSourceKind[] = [...flowBindingSourceKindOptions]

export const flowNodeKindLabelKey = (kind: FlowNodeKind) => {
  switch (kind) {
    case "compose":
      return "Compose"
    case "transform":
      return "Transform"
    case "set_var":
      return "Set Var"
    case "branch":
      return "Branch"
    case "foreach":
      return "Foreach"
    case "subflow":
      return "Subflow"
    default:
      return "Call"
  }
}

export const flowBindingSourceKindLabelKey = (kind: CanonicalFlowBindingSourceKind) => {
  switch (kind) {
    case "node_result":
      return "Ancestor Result"
    case "flow_meta":
      return "Flow Meta"
    case "run_meta":
      return "Run Meta"
    case "loop_item":
      return "Loop Item"
    case "loop_index":
      return "Loop Index"
    case "flow_var":
      return "Flow Local Var"
    default:
      return "Trigger"
  }
}

export const flowBranchMatchOpLabelKey = (op: FlowBranchMatchOp) => {
  switch (op) {
    case "eq":
      return "Equals"
    case "ne":
      return "Not Equals"
    case "gt":
      return "Greater Than"
    case "gte":
      return "Greater Than or Equal"
    case "lt":
      return "Less Than"
    case "lte":
      return "Less Than or Equal"
    default:
      return "Exists"
  }
}

export type FlowInputBindingDraft = {
  to: string
  sourceKind: FlowBindingSourceKind
  nodeId: string
  path: string
  field: string
  name: string
  required: boolean
}

export type FlowSourceDraft = {
  sourceKind: FlowBindingSourceKind
  nodeId: string
  path: string
  field: string
  name: string
}

export type FlowTransformExprMode = "literal" | "source" | "op" | "object" | "array"

export type FlowBranchCaseDraft = {
  key: string
  name: string
  source: FlowSourceDraft
  op: FlowBranchMatchOp
  valueJson: string
}

export type FlowNodeDraft = {
  id: string
  kind: FlowNodeKind
  allowFail: boolean
  retry: number
  retryBackoffMs: number
  timeoutMs: number
  method: string
  target: number
  argsTemplate: string
  composeTemplate: string
  setVarName: string
  inputs: FlowInputBindingDraft[]
  transformExprMode: FlowTransformExprMode
  transformLiteralJson: string
  transformSource: FlowSourceDraft
  transformSourceRequired: boolean
  transformOp: string
  transformArgsJson: string
  transformObjectJson: string
  transformArrayJson: string
  branchCases: FlowBranchCaseDraft[]
  branchDefaultCase: string
  foreachSource: FlowSourceDraft
  foreachRequired: boolean
  foreachBodyJson: string
  foreachResultNodeId: string
  subflowId: string
  subflowInputTemplate: string
  subflowResultNodeId: string
  specEditorMode: FlowSpecEditorMode
  specJson: string
  x: number
  y: number
}

export type FlowEdge = {
  from: string
  to: string
  case?: string
}

export type FlowPayload = {
  flow_id: string
  name: string
  max_active_runs?: number
  trigger: Record<string, any>
  graph: {
    nodes: Array<Record<string, any>>
    edges: Array<Record<string, any>>
  }
}

export type FlowGraphDraft = FlowPayload["graph"]

export type FlowStatusNode = {
  id: string
  status: string
  code: number
  msg: string
}

export type FlowStatus = {
  status: string
  runId: string
  executorNode: number
  nodes: FlowStatusNode[]
}

export type FlowRunHistoryItem = {
  runId: string
  status: string
  code: number
  msg: string
  startedAt: string
  endedAt: string
}

export type FlowNodeDetailState = {
  loading: boolean
  error: string
  requestedNodeId: string
  requestedRunId: string
  requestedPath: string
  runId: string
  path: string
  node: FlowStatusNode | null
  resultValue: unknown
  resultText: string
}

export type FlowDetailStructuredField = {
  key: string
  label: string
  pointer: string
  description?: string
  valueText: string
  missing: boolean
  multiline: boolean
}

export type FlowMessageLevel = "" | "success" | "error" | "info"

export const flowStatusLabelKey = (status: string) => {
  const normalized = String(status ?? "").trim().toLowerCase()
  switch (normalized) {
    case "cancelled":
      return "Cancelled"
    case "succeeded":
      return "Succeeded"
    case "failed":
      return "Failed"
    case "running":
      return "Running"
    case "queued":
      return "Queued"
    case "idle":
      return "Idle"
    case "":
      return "Unknown"
    default:
      return String(status ?? "").trim() || "Unknown"
  }
}

export const flowTriggerTypeLabelKey = (type: string) => {
  const normalized = String(type ?? "").trim().toLowerCase()
  switch (normalized) {
    case "cron":
      return "Cron"
    case "event":
      return "Event"
    case "var_changed":
      return "Variable Changed"
    default:
      return "Interval"
  }
}

export const flowEventModeLabelKey = (mode: string) => {
  const normalized = String(mode ?? "").trim().toLowerCase()
  switch (normalized) {
    case "received":
      return "Received"
    case "any":
      return "Any"
    default:
      return "Publish"
  }
}

export type ExecCapabilityRoute = {
  key: string
  providerNode: number
  viaNode: number
  method: string
  version: string
  defaultTimeoutMs: number
  permissions: string[]
  tags: Record<string, string>
  inputSchema: unknown
  outputSchema: unknown
  label: string
}

type FlowDraftSnapshot = {
  flowId: string
  flowName: string
  maxActiveRuns: number | null
  triggerType: FlowTriggerType
  eventMode: "publish" | "received" | "any"
  everyMs: number
  cronExpr: string
  eventName: string
  eventTopic: string
  dedupWindowMs: number
  varOwner: number
  varName: string
  nodes: FlowNodeDraft[]
  edges: FlowEdge[]
  selectedNodeIndex: number
  selectedEdgeIndex: number
}

export type FlowGraphEditorState = {
  nodes: FlowNodeDraft[]
  edges: FlowEdge[]
  selectedNodeIndex: number
  selectedEdgeIndex: number
}

type FlowState = {
  targetId: string
  selfNodeId: number
  hubId: number
  flows: FlowSummary[]
  flowId: string
  flowName: string
  maxActiveRuns: number | null
  triggerType: FlowTriggerType
  eventMode: "publish" | "received" | "any"
  everyMs: number
  cronExpr: string
  eventName: string
  eventTopic: string
  dedupWindowMs: number
  varOwner: number
  varName: string
  nodes: FlowNodeDraft[]
  edges: FlowEdge[]
  selectedNodeIndex: number
  selectedEdgeIndex: number
  statusRunId: string
  lastStatus: FlowStatus
  runHistory: FlowRunHistoryItem[]
  runHistoryLoading: boolean
  nodeDetail: FlowNodeDetailState
  execCapabilities: ExecCapabilityRoute[]
  execCapabilitiesLoading: boolean
  message: string
  messageLevel: FlowMessageLevel
  historyIndex: number
  historyLength: number
}

const createEmptyFlowStatus = (): FlowStatus => ({
  status: "",
  runId: "",
  executorNode: 0,
  nodes: []
})

const state = reactive<FlowState>({
  targetId: "",
  selfNodeId: 0,
  hubId: 0,
  flows: [],
  flowId: "",
  flowName: "",
  maxActiveRuns: null,
  triggerType: "interval",
  eventMode: "publish",
  everyMs: 60000,
  cronExpr: "",
  eventName: "",
  eventTopic: "",
  dedupWindowMs: 0,
  varOwner: 0,
  varName: "",
  nodes: [],
  edges: [],
  selectedNodeIndex: -1,
  selectedEdgeIndex: -1,
  statusRunId: "",
  lastStatus: createEmptyFlowStatus(),
  runHistory: [],
  runHistoryLoading: false,
  nodeDetail: {
    loading: false,
    error: "",
    requestedNodeId: "",
    requestedRunId: "",
    requestedPath: "",
    runId: "",
    path: "",
    node: null,
    resultValue: undefined,
    resultText: ""
  },
  execCapabilities: [],
  execCapabilitiesLoading: false,
  message: "",
  messageLevel: "",
  historyIndex: 0,
  historyLength: 1
})

const resetStatusState = () => {
  state.statusRunId = ""
  state.lastStatus = createEmptyFlowStatus()
  state.runHistory = []
  state.runHistoryLoading = false
}

const MAX_HISTORY = 120
let draftHistory: FlowDraftSnapshot[] = []
let draftHistoryIndex = 0
let execCapabilityLoadCount = 0
let execCapabilityLoadEpoch = 0
let execCapabilityCacheVersion = 0
const pendingCapabilityHydrations = new Map<string, Promise<boolean>>()
const FLOW_LOCAL_VAR_NAME_PATTERN = /^[A-Za-z_][A-Za-z0-9_]*$/
const FLOW_UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const FLOW_NODE_KIND_SET = new Set<string>(CANONICAL_FLOW_NODE_KINDS)
const FLOW_BINDING_SOURCE_KIND_SET = new Set<string>(CANONICAL_FLOW_BINDING_SOURCE_KINDS)
const FLOW_BRANCH_MATCH_OP_SET = new Set<string>(CANONICAL_FLOW_BRANCH_MATCH_OPS)
const FLOW_TRANSFORM_OPS = new Set([
  "add",
  "sub",
  "mul",
  "div",
  "mod",
  "neg",
  "abs",
  "min",
  "max",
  "eq",
  "ne",
  "gt",
  "gte",
  "lt",
  "lte",
  "and",
  "or",
  "not",
  "coalesce",
  "if",
  "concat",
  "lower",
  "upper",
  "trim",
  "len"
])
type ParseDraftOptions = {
  allowLoopSources?: boolean
}

type StrictGraphBuildOptions = {
  allowLoopSources?: boolean
  currentFlowId?: string
}

type ParsedSourceDraft = {
  source: FlowSourceDraft
  supported: boolean
}

type ParsedInputBindings = {
  inputs: FlowInputBindingDraft[]
  supported: boolean
}

const cloneBindingDraft = (binding?: FlowInputBindingDraft): FlowInputBindingDraft => ({
  ...defaultInputBinding(),
  ...(binding ?? {})
})
const cloneSourceDraft = (source?: FlowSourceDraft): FlowSourceDraft => ({
  ...defaultSourceDraft("trigger"),
  ...(source ?? {})
})
const cloneBranchCaseDraft = (item: FlowBranchCaseDraft): FlowBranchCaseDraft => ({
  ...item,
  source: cloneSourceDraft(item.source)
})

const cloneNodeDraft = (node: FlowNodeDraft): FlowNodeDraft => ({
  ...node,
  inputs: (node.inputs ?? []).map(cloneBindingDraft),
  transformSource: cloneSourceDraft(node.transformSource),
  branchCases: (node.branchCases ?? []).map(cloneBranchCaseDraft),
  foreachSource: cloneSourceDraft(node.foreachSource)
})

const cloneEdge = (edge: FlowEdge): FlowEdge => ({ ...edge })

const setMessage = (message: string, level: Exclude<FlowMessageLevel, ""> = "info") => {
  const trimmed = message.trim()
  state.message = trimmed
  state.messageLevel = trimmed ? level : ""
}

const snapshotToJSON = (snapshot: FlowDraftSnapshot) => JSON.stringify(snapshot)

// takeSnapshot 采集整个编辑草稿，供撤销/重做和恢复草稿共用同一份状态快照。
const takeSnapshot = (): FlowDraftSnapshot => ({
  flowId: state.flowId,
  flowName: state.flowName,
  maxActiveRuns: state.maxActiveRuns,
  triggerType: state.triggerType,
  eventMode: state.eventMode,
  everyMs: state.everyMs,
  cronExpr: state.cronExpr,
  eventName: state.eventName,
  eventTopic: state.eventTopic,
  dedupWindowMs: state.dedupWindowMs,
  varOwner: state.varOwner,
  varName: state.varName,
  nodes: state.nodes.map(cloneNodeDraft),
  edges: state.edges.map(cloneEdge),
  selectedNodeIndex: state.selectedNodeIndex,
  selectedEdgeIndex: state.selectedEdgeIndex
})

const takeGraphEditorState = (): FlowGraphEditorState => ({
  nodes: state.nodes.map(cloneNodeDraft),
  edges: state.edges.map(cloneEdge),
  selectedNodeIndex: state.selectedNodeIndex,
  selectedEdgeIndex: state.selectedEdgeIndex
})

const takeGraphEditorContentState = () => ({
  nodes: state.nodes.map(cloneNodeDraft),
  edges: state.edges.map(cloneEdge)
})

const updateHistoryState = () => {
  state.historyIndex = draftHistoryIndex
  state.historyLength = draftHistory.length
}

const resetHistory = () => {
  draftHistory = [takeSnapshot()]
  draftHistoryIndex = 0
  updateHistoryState()
}

const applySnapshot = (snapshot: FlowDraftSnapshot) => {
  state.flowId = snapshot.flowId
  state.flowName = snapshot.flowName
  state.maxActiveRuns = snapshot.maxActiveRuns
  state.triggerType = snapshot.triggerType
  state.eventMode = snapshot.eventMode
  state.everyMs = snapshot.everyMs
  state.cronExpr = snapshot.cronExpr ?? ""
  state.eventName = snapshot.eventName
  state.eventTopic = snapshot.eventTopic
  state.dedupWindowMs = Number.isFinite(snapshot.dedupWindowMs) && snapshot.dedupWindowMs >= 0 ? Math.trunc(snapshot.dedupWindowMs) : 0
  state.varOwner = snapshot.varOwner
  state.varName = snapshot.varName
  state.nodes = snapshot.nodes.map(cloneNodeDraft)
  state.edges = snapshot.edges.map(cloneEdge)
  state.selectedNodeIndex =
    snapshot.selectedNodeIndex >= 0 && snapshot.selectedNodeIndex < state.nodes.length
      ? snapshot.selectedNodeIndex
      : -1
  state.selectedEdgeIndex =
    snapshot.selectedEdgeIndex >= 0 && snapshot.selectedEdgeIndex < state.edges.length
      ? snapshot.selectedEdgeIndex
      : -1
}

// commitHistory 只在内容真的变化时推进历史栈，并限制最大快照数量。
const commitHistory = () => {
  const snapshot = takeSnapshot()
  if (!draftHistory.length) {
    draftHistory = [snapshot]
    draftHistoryIndex = 0
    updateHistoryState()
    return false
  }
  const current = draftHistory[draftHistoryIndex]
  if (current && snapshotToJSON(current) === snapshotToJSON(snapshot)) {
    return false
  }
  if (draftHistoryIndex < draftHistory.length - 1) {
    draftHistory = draftHistory.slice(0, draftHistoryIndex + 1)
  }
  draftHistory.push(snapshot)
  draftHistoryIndex = draftHistory.length - 1
  if (draftHistory.length > MAX_HISTORY) {
    const overflow = draftHistory.length - MAX_HISTORY
    draftHistory.splice(0, overflow)
    draftHistoryIndex = Math.max(0, draftHistoryIndex - overflow)
  }
  updateHistoryState()
  return true
}

// applyGraphEditorState 用外部快照整体替换画布状态，同时清空状态/能力等派生态。
const applyGraphEditorState = (snapshot: FlowGraphEditorState) => {
  state.nodes = Array.isArray(snapshot?.nodes) ? snapshot.nodes.map(cloneNodeDraft) : []
  state.edges = Array.isArray(snapshot?.edges) ? snapshot.edges.map(cloneEdge) : []
  state.selectedNodeIndex =
    Number.isInteger(snapshot?.selectedNodeIndex) &&
    Number(snapshot.selectedNodeIndex) >= 0 &&
    Number(snapshot.selectedNodeIndex) < state.nodes.length
      ? Number(snapshot.selectedNodeIndex)
      : -1
  state.selectedEdgeIndex =
    Number.isInteger(snapshot?.selectedEdgeIndex) &&
    Number(snapshot.selectedEdgeIndex) >= 0 &&
    Number(snapshot.selectedEdgeIndex) < state.edges.length
      ? Number(snapshot.selectedEdgeIndex)
      : -1
  resetStatusState()
  state.nodeDetail = createNodeDetailState()
  resetExecCapabilityState()
  resetHistory()
}

const graphEditorSignature = () => JSON.stringify(takeGraphEditorContentState())

const undo = () => {
  if (draftHistoryIndex <= 0) return false
  draftHistoryIndex -= 1
  applySnapshot(draftHistory[draftHistoryIndex])
  updateHistoryState()
  setMessage(t("Undo applied."))
  return true
}

const redo = () => {
  if (draftHistoryIndex >= draftHistory.length - 1) return false
  draftHistoryIndex += 1
  applySnapshot(draftHistory[draftHistoryIndex])
  updateHistoryState()
  setMessage(t("Redo applied."))
  return true
}

const newReqId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID()
  }
  return `req_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`
}

const resolveTargetNode = () => {
  const raw = state.targetId.trim()
  if (!raw) {
    if (!state.hubId) {
      throw new Error(t("Target node is required."))
    }
    return state.hubId
  }
  const parsed = Number.parseInt(raw, 10)
  if (Number.isNaN(parsed) || parsed <= 0) {
    throw new Error(t("Target node must be a positive number."))
  }
  return parsed
}

const resolveCapabilityQueryNode = (queryNodeId?: string | number) => {
  if (queryNodeId === undefined || queryNodeId === null) {
    return resolveTargetNode()
  }
  const raw = String(queryNodeId).trim()
  if (!raw) {
    return resolveTargetNode()
  }
  if (!/^\d+$/.test(raw)) {
    throw new Error(t("Query node ID must be a positive number."))
  }
  const parsed = Number.parseInt(raw, 10)
  if (!Number.isFinite(parsed) || parsed <= 0) {
    throw new Error(t("Query node ID must be a positive number."))
  }
  return Math.trunc(parsed)
}

const ensureIdentity = () => {
  if (!state.selfNodeId) {
    throw new Error(t("Login required to send Flow requests."))
  }
  if (!state.hubId) {
    throw new Error(t("Hub ID missing."))
  }
  return { sourceID: state.selfNodeId, hubID: state.hubId }
}

const normalizeCallTarget = (providerNode: number) => {
  const normalizedProvider = Number.isFinite(providerNode) && providerNode > 0 ? Math.trunc(providerNode) : 0
  if (normalizedProvider <= 0) {
    return 0
  }
  try {
    const executorNode = resolveTargetNode()
    return normalizedProvider === executorNode ? 0 : normalizedProvider
  } catch {
    return normalizedProvider
  }
}

const mapSummary = (input: any): FlowSummary => ({
  flowId: String(input?.flow_id ?? input?.flowId ?? ""),
  name: String(input?.name ?? ""),
  everyMs: Number(input?.every_ms ?? input?.everyMs ?? 0),
  lastRunId: String(input?.last_run_id ?? input?.lastRunId ?? ""),
  lastStatus: String(input?.last_status ?? input?.lastStatus ?? "")
})

const mapRunHistoryItem = (input: any): FlowRunHistoryItem => ({
  runId: String(input?.run_id ?? input?.runId ?? ""),
  status: String(input?.status ?? ""),
  code: Number(input?.code ?? 0),
  msg: String(input?.msg ?? ""),
  startedAt: String(
    input?.started_at ??
      input?.start_at ??
      input?.startedAt ??
      input?.startAt ??
      input?.started_at_ms ??
      input?.startedAtMs ??
      ""
  ),
  endedAt: String(
    input?.ended_at ??
      input?.finished_at ??
      input?.endedAt ??
      input?.finishedAt ??
      input?.ended_at_ms ??
      input?.endedAtMs ??
      ""
  )
})

const formatJSONText = (value: any, fallback: any = {}) => {
  const source = value === undefined ? fallback : value
  try {
    return JSON.stringify(source ?? fallback, null, 2)
  } catch {
    return JSON.stringify(fallback, null, 2)
  }
}

const formatStructuredText = (value: unknown, emptyFallback = "") => {
  if (value === undefined) {
    return emptyFallback
  }
  if (typeof value === "string") {
    const trimmed = value.trim()
    if (!trimmed) {
      return emptyFallback
    }
    try {
      return JSON.stringify(JSON.parse(trimmed), null, 2)
    } catch {
      return JSON.stringify(value, null, 2)
    }
  }
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value ?? emptyFallback)
  }
}

const isJSONObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value)

type DetailSchemaFieldKind = "inline" | "json"

type DetailSchemaField = {
  key: string
  label: string
  pointer: string
  description?: string
  kind: DetailSchemaFieldKind
}

const DETAIL_SCHEMA_SUPPORTED_TYPES = new Set(["string", "number", "integer", "boolean", "object"])

const humanizeSchemaKey = (key: string) =>
  String(key ?? "")
    .replaceAll(/[_-]+/g, " ")
    .replaceAll(/\s+/g, " ")
    .trim()
    .replace(/\b\w/g, (ch) => ch.toUpperCase())

const parseStructuredSchemaText = (rawText: string): Record<string, unknown> | null => {
  const trimmed = String(rawText ?? "").trim()
  if (!trimmed) {
    return null
  }
  try {
    const parsed = JSON.parse(trimmed)
    return isJSONObject(parsed) ? parsed : null
  } catch {
    return null
  }
}

const normalizeDetailSchemaType = (schema: Record<string, unknown>) => {
  if (typeof schema.type === "string") {
    return schema.type.trim()
  }
  if (!Array.isArray(schema.type)) {
    return ""
  }
  const rawTypes = schema.type.map((item) => (typeof item === "string" ? item.trim() : "")).filter(Boolean)
  if (rawTypes.length !== 2) {
    return ""
  }
  const uniqueTypes = Array.from(new Set(rawTypes))
  if (uniqueTypes.length !== 2 || !uniqueTypes.includes("null")) {
    return ""
  }
  const resolvedType = uniqueTypes.find((item) => item !== "null") ?? ""
  return DETAIL_SCHEMA_SUPPORTED_TYPES.has(resolvedType) ? resolvedType : ""
}

const hasUnsupportedDetailSchemaFeature = (schema: Record<string, unknown>) =>
  "oneOf" in schema ||
  "anyOf" in schema ||
  "allOf" in schema ||
  "$ref" in schema ||
  normalizeDetailSchemaType(schema) === "array" ||
  (Array.isArray(schema.type) && !normalizeDetailSchemaType(schema))

const collectDetailSchemaFields = (
  schema: Record<string, unknown>,
  basePointer: string,
  out: DetailSchemaField[]
): boolean => {
  if (hasUnsupportedDetailSchemaFeature(schema)) {
    return false
  }

  const schemaType = normalizeDetailSchemaType(schema)
  const properties = isJSONObject(schema.properties) ? schema.properties : null
  if (schemaType !== "object" && !properties) {
    return false
  }
  if (!properties || Object.keys(properties).length === 0) {
    if (!basePointer) {
      return false
    }
    out.push({
      key: `detail:${basePointer}`,
      label:
        typeof schema.title === "string" && schema.title.trim()
          ? schema.title.trim()
          : basePointer.split("/").at(-1) ?? "Value",
      pointer: basePointer,
      description: typeof schema.description === "string" ? schema.description.trim() : undefined,
      kind: "json"
    })
    return true
  }

  for (const [rawKey, childValue] of Object.entries(properties)) {
    if (!isJSONObject(childValue) || hasUnsupportedDetailSchemaFeature(childValue)) {
      return false
    }
    const pointer = `/${basePointer ? `${basePointer.slice(1)}/` : ""}${rawKey.replaceAll("~", "~0").replaceAll("/", "~1")}`
    const label =
      typeof childValue.title === "string" && childValue.title.trim() ? childValue.title.trim() : humanizeSchemaKey(rawKey)
    const description = typeof childValue.description === "string" ? childValue.description.trim() : undefined
    const childType = normalizeDetailSchemaType(childValue)
    const childProperties = isJSONObject(childValue.properties) ? childValue.properties : null

    if ((childType === "object" || childProperties) && childProperties && Object.keys(childProperties).length > 0) {
      if (!collectDetailSchemaFields(childValue, pointer, out)) {
        return false
      }
      continue
    }

    if (childType === "object" || childProperties) {
      out.push({
        key: `detail:${pointer}`,
        label,
        pointer,
        description,
        kind: "json"
      })
      continue
    }

    switch (childType) {
      case "string":
      case "number":
      case "integer":
      case "boolean":
        out.push({
          key: `detail:${pointer}`,
          label,
          pointer,
          description,
          kind: "inline"
        })
        break
      default:
        return false
    }
  }

  return true
}

const formatDetailFieldValue = (value: unknown) => {
  if (typeof value === "string") {
    return value
  }
  if (typeof value === "number" || typeof value === "boolean" || value === null) {
    return String(value)
  }
  return formatStructuredText(value, "")
}

export const buildDetailStructuredFields = (
  outputSchemaText: string,
  resultValue: unknown,
  detailPath = ""
): FlowDetailStructuredField[] => {
  if (String(detailPath ?? "").trim() || !isJSONObject(resultValue)) {
    return []
  }

  const schemaDoc = parseStructuredSchemaText(outputSchemaText)
  if (!schemaDoc || hasUnsupportedDetailSchemaFeature(schemaDoc)) {
    return []
  }
  if (normalizeDetailSchemaType(schemaDoc) !== "object" && !isJSONObject(schemaDoc.properties)) {
    return []
  }

  const schemaFields: DetailSchemaField[] = []
  if (!collectDetailSchemaFields(schemaDoc, "", schemaFields) || !schemaFields.length) {
    return []
  }

  return schemaFields.map((field) => {
    const resolved = readValueAtPointer(resultValue, field.pointer)
    const multiline =
      resolved.found &&
      (field.kind === "json" || (typeof resolved.value === "object" && resolved.value !== null))
    return {
      key: field.key,
      label: field.label,
      pointer: field.pointer,
      description: field.description,
      valueText: resolved.found ? formatDetailFieldValue(resolved.value) : "",
      missing: !resolved.found,
      multiline
    }
  })
}

const createNodeDetailState = (nodeId = "", runId = ""): FlowNodeDetailState => ({
  loading: false,
  error: "",
  requestedNodeId: nodeId.trim(),
  requestedRunId: runId.trim(),
  requestedPath: "",
  runId: "",
  path: "",
  node: null,
  resultValue: undefined,
  resultText: ""
})

const normalizeNodeKind = (raw: any): FlowNodeKind => {
  const normalized = String(raw ?? "").trim().toLowerCase()
  return FLOW_NODE_KIND_SET.has(normalized) ? (normalized as FlowNodeKind) : "call"
}

const hasOwn = (value: unknown, key: string) =>
  Boolean(value) && typeof value === "object" && Object.prototype.hasOwnProperty.call(value, key)

const isLoopSourceKind = (kind: FlowBindingSourceKind) => kind === "loop_item" || kind === "loop_index"

const supportsFormMode = (kind: FlowNodeKind) => FLOW_NODE_KIND_SET.has(kind)

const defaultSpecEditorMode = (kind: FlowNodeKind): FlowSpecEditorMode => (supportsFormMode(kind) ? "form" : "json")

const kindDefaultSpec = (kind: FlowNodeKind): Record<string, any> => {
  switch (kind) {
    case "compose":
      return { template: {} }
    case "transform":
      return { expr: { literal: null } }
    case "set_var":
      return { name: "", template: null }
    case "branch":
      return { cases: [] }
    case "foreach":
      return {
        source: { kind: "trigger" },
        required: true,
        body: { nodes: [], edges: [] },
        result_node_id: ""
      }
    case "subflow":
      return { flow_id: "", input_template: {} }
    default:
      return { method: "", args_template: {} }
  }
}

const normalizeBindingSourceKind = (raw: any): FlowBindingSourceKind => {
  const normalized = String(raw ?? "").trim().toLowerCase()
  return FLOW_BINDING_SOURCE_KIND_SET.has(normalized) ? (normalized as CanonicalFlowBindingSourceKind) : ""
}

const defaultInputBinding = (): FlowInputBindingDraft => ({
  to: "",
  sourceKind: "node_result",
  nodeId: "",
  path: "",
  field: "",
  name: "",
  required: false
})

const defaultSourceDraft = (sourceKind: FlowBindingSourceKind = "trigger"): FlowSourceDraft => ({
  sourceKind,
  nodeId: "",
  path: "",
  field: sourceKind === "run_meta" ? "run_id" : sourceKind === "flow_meta" ? "flow_id" : "",
  name: ""
})

const defaultTransformDraft = () => ({
  transformExprMode: "literal" as FlowTransformExprMode,
  transformLiteralJson: "null",
  transformSource: defaultSourceDraft("trigger"),
  transformSourceRequired: true,
  transformOp: "add",
  transformArgsJson: "[]",
  transformObjectJson: "{}",
  transformArrayJson: "[]"
})

const defaultForeachBodyGraph = () => ({
  nodes: [] as any[],
  edges: [] as any[]
})

const defaultForeachDraft = () => ({
  foreachSource: defaultSourceDraft("trigger"),
  foreachRequired: true,
  foreachBodyJson: formatJSONText(defaultForeachBodyGraph(), defaultForeachBodyGraph()),
  foreachResultNodeId: ""
})

const defaultAdvancedNodeFields = () => ({
  ...defaultTransformDraft(),
  branchCases: [] as FlowBranchCaseDraft[],
  branchDefaultCase: "",
  ...defaultForeachDraft(),
  subflowId: "",
  subflowInputTemplate: "{}",
  subflowResultNodeId: ""
})

const newBranchCaseKey = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID()
  }
  return `branch_case_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`
}

const createBranchCaseDraft = (input?: Partial<FlowBranchCaseDraft>): FlowBranchCaseDraft => ({
  key: String(input?.key ?? "").trim() || newBranchCaseKey(),
  name: String(input?.name ?? "").trim(),
  source: cloneSourceDraft(input?.source ?? defaultSourceDraft("trigger")),
  op: FLOW_BRANCH_MATCH_OP_SET.has(String(input?.op ?? "")) ? (input?.op as FlowBranchMatchOp) : "eq",
  valueJson: String(input?.valueJson ?? "null").trim() || "null"
})

const hasOnlyKeys = (value: Record<string, any>, allowed: string[]) =>
  Object.keys(value).every((key) => allowed.includes(key))

const parseSourceDraft = (
  raw: any,
  fallbackKind: FlowBindingSourceKind = "trigger",
  options: ParseDraftOptions = {}
): ParsedSourceDraft => {
  const explicitKind = String(raw?.kind ?? "").trim()
  const normalizedKind = normalizeBindingSourceKind(raw?.kind)
  const sourceKind = normalizedKind || fallbackKind
  const unsupportedLoopSource = isLoopSourceKind(normalizedKind) && !options.allowLoopSources
  const supportedShape = (() => {
    switch (sourceKind) {
      case "node_result":
        return hasOnlyKeys(raw, ["kind", "node_id", "path"])
      case "trigger":
      case "loop_item":
        return hasOnlyKeys(raw, ["kind", "path"])
      case "flow_meta":
        return (
          hasOnlyKeys(raw, ["kind", "field"]) &&
          (!hasOwn(raw, "field") || !String(raw?.field ?? "").trim() || String(raw?.field ?? "").trim() === "flow_id")
        )
      case "run_meta":
        return (
          hasOnlyKeys(raw, ["kind", "field"]) &&
          (!hasOwn(raw, "field") || !String(raw?.field ?? "").trim() || String(raw?.field ?? "").trim() === "run_id")
        )
      case "loop_index":
        return (
          hasOnlyKeys(raw, ["kind"]) ||
          (hasOnlyKeys(raw, ["kind", "path"]) && !String(raw?.path ?? "").trim())
        )
      case "flow_var":
        return hasOnlyKeys(raw, ["kind", "name", "path"])
      default:
        return false
    }
  })()
  return {
    source: {
      sourceKind,
      nodeId: sourceKind === "node_result" ? String(raw?.node_id ?? "").trim() : "",
      path: sourceKind === "loop_index" ? "" : String(raw?.path ?? "").trim(),
      field: sourceKind === "run_meta" ? "run_id" : sourceKind === "flow_meta" ? "flow_id" : "",
      name: sourceKind === "flow_var" ? String(raw?.name ?? "").trim() : ""
    },
    supported: supportedShape && !(explicitKind && !normalizedKind) && !unsupportedLoopSource
  }
}

const buildSourceSpec = (input: {
  nodeId: string
  label: string
  source: FlowSourceDraft
  ancestors: Map<string, Set<string>>
  allowLoopSources?: boolean
}): Record<string, any> => {
  const nodeId = input.nodeId.trim() || t("Unnamed")
  const label = input.label
  const source = input.source
  switch (source.sourceKind) {
    case "node_result": {
      const sourceNodeId = source.nodeId.trim()
      if (!sourceNodeId) {
        throw new Error(t("Node {nodeId} {label}: source node is required.", { nodeId, label }))
      }
      const allowedAncestors = input.ancestors.get(nodeId) ?? new Set<string>()
      if (!allowedAncestors.has(sourceNodeId)) {
        throw new Error(t("Node {nodeId} {label}: source node must be an ancestor.", { nodeId, label }))
      }
      validateJSONPointer(
        source.path,
        t("Node {nodeId} {label}: result path must be a valid JSON Pointer.", { nodeId, label })
      )
      const out: Record<string, any> = {
        kind: "node_result",
        node_id: sourceNodeId
      }
      if (source.path.trim()) {
        out.path = source.path.trim()
      }
      return out
    }
    case "trigger": {
      validateJSONPointer(
        source.path,
        t("Node {nodeId} {label}: trigger path must be a valid JSON Pointer.", { nodeId, label })
      )
      const out: Record<string, any> = { kind: "trigger" }
      if (source.path.trim()) {
        out.path = source.path.trim()
      }
      return out
    }
    case "flow_meta":
      return { kind: "flow_meta", field: "flow_id" }
    case "run_meta":
      return { kind: "run_meta", field: "run_id" }
    case "loop_item": {
      if (!input.allowLoopSources) {
        throw new Error(t("Node {nodeId} {label}: loop sources are only available inside foreach body graphs.", { nodeId, label }))
      }
      validateJSONPointer(
        source.path,
        t("Node {nodeId} {label}: loop item path must be a valid JSON Pointer.", { nodeId, label })
      )
      const out: Record<string, any> = { kind: "loop_item" }
      if (source.path.trim()) {
        out.path = source.path.trim()
      }
      return out
    }
    case "loop_index":
      if (!input.allowLoopSources) {
        throw new Error(t("Node {nodeId} {label}: loop sources are only available inside foreach body graphs.", { nodeId, label }))
      }
      if (source.path.trim()) {
        throw new Error(t("Node {nodeId} {label}: loop index does not accept a JSON Pointer path.", { nodeId, label }))
      }
      return { kind: "loop_index" }
    case "flow_var": {
      const name = assertValidFlowLocalVarName(
        normalizeFlowLocalVarName(source.name),
        t("Node {nodeId} {label}: flow local var name is required.", { nodeId, label }),
        t("Node {nodeId} {label}: flow local var name is invalid.", { nodeId, label })
      )
      validateJSONPointer(
        source.path,
        t("Node {nodeId} {label}: flow local var path must be a valid JSON Pointer.", { nodeId, label })
      )
      const out: Record<string, any> = { kind: "flow_var", name }
      if (source.path.trim()) {
        out.path = source.path.trim()
      }
      return out
    }
    default:
      throw new Error(t("Node {nodeId} {label}: source kind is required.", { nodeId, label }))
  }
}

const buildLooseSourceSpec = (source: FlowSourceDraft): Record<string, any> => {
  switch (source.sourceKind) {
    case "node_result":
      return {
        kind: "node_result",
        ...(source.nodeId.trim() ? { node_id: source.nodeId.trim() } : {}),
        ...(source.path.trim() ? { path: source.path.trim() } : {})
      }
    case "flow_meta":
      return { kind: "flow_meta", field: "flow_id" }
    case "run_meta":
      return { kind: "run_meta", field: "run_id" }
    case "loop_item":
      return {
        kind: "loop_item",
        ...(source.path.trim() ? { path: source.path.trim() } : {})
      }
    case "loop_index":
      return { kind: "loop_index" }
    case "flow_var":
      return {
        kind: "flow_var",
        ...(source.name.trim() ? { name: source.name.trim() } : {}),
        ...(source.path.trim() ? { path: source.path.trim() } : {})
      }
    case "trigger":
    default:
      return {
        kind: "trigger",
        ...(source.path.trim() ? { path: source.path.trim() } : {})
      }
  }
}

const parseTransformDraft = (parsed: Record<string, any>, options: ParseDraftOptions = {}) => {
  const defaults = defaultTransformDraft()
  const allowedTopLevel = hasOnlyKeys(parsed, ["expr", "_ui"])
  if (!allowedTopLevel) {
    return {
      ...defaults,
      specEditorMode: "json" as FlowSpecEditorMode
    }
  }
  if (!("expr" in parsed)) {
    return {
      ...defaults,
      specEditorMode: "form" as FlowSpecEditorMode
    }
  }
  const expr = parsed?.expr
  if (!isJSONObject(expr)) {
    return {
      ...defaults,
      specEditorMode: "json" as FlowSpecEditorMode
    }
  }
  if ("literal" in expr && hasOnlyKeys(expr, ["literal"])) {
    return {
      ...defaults,
      transformExprMode: "literal" as FlowTransformExprMode,
      transformLiteralJson: formatJSONText((expr as Record<string, any>).literal, null),
      specEditorMode: "form" as FlowSpecEditorMode
    }
  }
  if ("source" in expr && hasOnlyKeys(expr, ["source", "required"]) && isJSONObject(expr.source)) {
    const parsedSource = parseSourceDraft(expr.source, "trigger", options)
    return {
      ...defaults,
      transformExprMode: "source" as FlowTransformExprMode,
      transformSource: parsedSource.source,
      transformSourceRequired: Boolean(expr.required ?? true),
      specEditorMode: parsedSource.supported ? ("form" as FlowSpecEditorMode) : ("json" as FlowSpecEditorMode)
    }
  }
  if ("op" in expr && hasOnlyKeys(expr, ["op", "args"])) {
    const op = String(expr.op ?? "").trim().toLowerCase()
    if (!FLOW_TRANSFORM_OPS.has(op)) {
      return {
        ...defaults,
        specEditorMode: "json" as FlowSpecEditorMode
      }
    }
    return {
      ...defaults,
      transformExprMode: "op" as FlowTransformExprMode,
      transformOp: op,
      transformArgsJson: formatJSONText(Array.isArray(expr.args) ? expr.args : [], []),
      specEditorMode: "form" as FlowSpecEditorMode
    }
  }
  if ("object" in expr && hasOnlyKeys(expr, ["object"]) && isJSONObject(expr.object)) {
    return {
      ...defaults,
      transformExprMode: "object" as FlowTransformExprMode,
      transformObjectJson: formatJSONText(expr.object, {}),
      specEditorMode: "form" as FlowSpecEditorMode
    }
  }
  if ("array" in expr && hasOnlyKeys(expr, ["array"]) && Array.isArray(expr.array)) {
    return {
      ...defaults,
      transformExprMode: "array" as FlowTransformExprMode,
      transformArrayJson: formatJSONText(expr.array, []),
      specEditorMode: "form" as FlowSpecEditorMode
    }
  }
  return {
    ...defaults,
    specEditorMode: "json" as FlowSpecEditorMode
  }
}

const parseBranchDraft = (parsed: Record<string, any>, options: ParseDraftOptions = {}) => {
  const branchCases: FlowBranchCaseDraft[] = []
  const allowedTopLevel = hasOnlyKeys(parsed, ["cases", "default_case", "_ui"])
  const cases = Array.isArray(parsed?.cases) ? parsed.cases : []
  let supported = allowedTopLevel
  for (const item of cases) {
    if (!isJSONObject(item) || !hasOnlyKeys(item, ["name", "match"]) || !isJSONObject(item.match)) {
      supported = false
      break
    }
    const match = item.match as Record<string, any>
    if (!hasOnlyKeys(match, ["source", "op", "value"]) || !isJSONObject(match.source)) {
      supported = false
      break
    }
    const op = String(match.op ?? "").trim().toLowerCase() as FlowBranchMatchOp
    if (!FLOW_BRANCH_MATCH_OP_SET.has(op)) {
      supported = false
      break
    }
    const parsedSource = parseSourceDraft(match.source, "trigger", options)
    supported = supported && parsedSource.supported
    branchCases.push(
      createBranchCaseDraft({
        name: String(item.name ?? "").trim(),
        source: parsedSource.source,
        op,
        valueJson: formatJSONText("value" in match ? match.value : null, null)
      })
    )
  }
  return {
    branchCases,
    branchDefaultCase: String(parsed?.default_case ?? "").trim(),
    specEditorMode: supported ? ("form" as FlowSpecEditorMode) : ("json" as FlowSpecEditorMode)
  }
}

const parseSubflowDraft = (parsed: Record<string, any>) => {
  const flowId = String(parsed?.flow_id ?? "").trim()
  const inputTemplate = "input_template" in parsed ? parsed.input_template : {}
  const isSupported =
    hasOnlyKeys(parsed, ["flow_id", "input_template", "inputs", "result_node_id", "_ui"]) &&
    isJSONObject(inputTemplate)
  return {
    subflowId: flowId,
    subflowInputTemplate: formatJSONText(inputTemplate, {}),
    subflowResultNodeId: String(parsed?.result_node_id ?? "").trim(),
    specEditorMode: isSupported ? ("form" as FlowSpecEditorMode) : ("json" as FlowSpecEditorMode)
  }
}

const parseForeachDraft = (parsed: Record<string, any>, options: ParseDraftOptions = {}) => {
  const defaults = defaultForeachDraft()
  const sourceSupported = !("source" in parsed) || isJSONObject(parsed.source)
  const body = "body" in parsed ? parsed.body : defaultForeachBodyGraph()
  const bodySupported = isJSONObject(body) && Array.isArray(body.nodes) && Array.isArray(body.edges)
  const parsedSource = "source" in parsed && isJSONObject(parsed.source) ? parseSourceDraft(parsed.source, "trigger", options) : null
  const supported =
    hasOnlyKeys(parsed, ["source", "required", "body", "result_node_id", "_ui"]) &&
    sourceSupported &&
    bodySupported &&
    (parsedSource?.supported ?? true)
  return {
    ...defaults,
    foreachSource: parsedSource?.source ?? defaults.foreachSource,
    foreachRequired: Boolean(parsed?.required ?? true),
    foreachBodyJson: formatJSONText(bodySupported ? body : defaultForeachBodyGraph(), defaultForeachBodyGraph()),
    foreachResultNodeId: String(parsed?.result_node_id ?? "").trim(),
    specEditorMode: supported ? ("form" as FlowSpecEditorMode) : ("json" as FlowSpecEditorMode)
  }
}

const normalizeSpecObject = (spec: any) => {
  let parsed = spec
  if (typeof spec === "string") {
    try {
      parsed = JSON.parse(spec)
    } catch {
      parsed = {}
    }
  }
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    parsed = {}
  }
  return parsed as Record<string, any>
}

const parseInputBindings = (raw: any, options: ParseDraftOptions = {}): ParsedInputBindings => {
  if (!Array.isArray(raw)) {
    return { inputs: [], supported: true }
  }
  let supported = true
  const inputs = raw.map((binding) => {
    const parsedSource = parseSourceDraft(binding?.source ?? {}, "trigger", options)
    supported = supported && parsedSource.supported
    return {
      to: String(binding?.to ?? "").trim(),
      sourceKind: parsedSource.source.sourceKind,
      nodeId: parsedSource.source.nodeId,
      path: parsedSource.source.path,
      field: parsedSource.source.field,
      name: parsedSource.source.name,
      required: Boolean(binding?.required ?? false)
    }
  })
  return { inputs, supported }
}

const parseSpecDraft = (kind: FlowNodeKind, spec: any, options: ParseDraftOptions = {}) => {
  const parsed = normalizeSpecObject(spec)
  const ui = parsed?._ui ?? {}
  const x = Number(ui?.x)
  const y = Number(ui?.y)
  const base = {
    method: "",
    target: 0,
    argsTemplate: "{}",
    composeTemplate: "{}",
    setVarName: "",
    inputs: [] as FlowInputBindingDraft[],
    ...defaultAdvancedNodeFields(),
    specEditorMode: defaultSpecEditorMode(kind),
    specJson: formatJSONText(parsed, {}),
    x: Number.isFinite(x) ? x : undefined,
    y: Number.isFinite(y) ? y : undefined
  }
  if (kind === "compose") {
    const parsedInputs = parseInputBindings(parsed?.inputs, options)
    return {
      ...base,
      composeTemplate: formatJSONText(parsed?.template, {}),
      inputs: parsedInputs.inputs,
      specEditorMode: parsedInputs.supported ? ("form" as FlowSpecEditorMode) : ("json" as FlowSpecEditorMode)
    }
  }
  if (kind === "set_var") {
    const parsedInputs = parseInputBindings(parsed?.inputs, options)
    return {
      ...base,
      composeTemplate: formatJSONText("template" in parsed ? parsed.template : null, null),
      setVarName: String(parsed?.name ?? "").trim(),
      inputs: parsedInputs.inputs,
      specEditorMode: parsedInputs.supported ? ("form" as FlowSpecEditorMode) : ("json" as FlowSpecEditorMode)
    }
  }
  if (kind === "transform") {
    const next = parseTransformDraft(parsed, options)
    return {
      ...base,
      ...next
    }
  }
  if (kind === "branch") {
    const next = parseBranchDraft(parsed, options)
    return {
      ...base,
      ...next
    }
  }
  if (kind === "foreach") {
    const next = parseForeachDraft(parsed, options)
    return {
      ...base,
      ...next
    }
  }
  if (kind === "subflow") {
    const parsedInputs = parseInputBindings(parsed?.inputs, options)
    const next = parseSubflowDraft(parsed)
    return {
      ...base,
      ...next,
      inputs: parsedInputs.inputs,
      specEditorMode:
        next.specEditorMode === "form" && parsedInputs.supported
          ? ("form" as FlowSpecEditorMode)
          : ("json" as FlowSpecEditorMode)
    }
  }
  if (!supportsFormMode(kind)) {
    return {
      ...base,
      specEditorMode: "json" as FlowSpecEditorMode
    }
  }
  const method = String(parsed?.method ?? "")
  const target = Number(parsed?.target ?? 0)
  const parsedInputs = parseInputBindings(parsed?.inputs, options)
  return {
    ...base,
    method,
    target,
    argsTemplate: formatJSONText(parsed?.args_template ?? parsed?.args, {}),
    inputs: parsedInputs.inputs,
    specEditorMode: parsedInputs.supported ? ("form" as FlowSpecEditorMode) : ("json" as FlowSpecEditorMode)
  }
}

const defaultNodePosition = (index: number) => {
  const col = index % 4
  const row = Math.floor(index / 4)
  return { x: col * 240, y: row * 160 }
}

const createNodeDraft = (id: string, kind: FlowNodeKind, index: number): FlowNodeDraft => {
  const pos = defaultNodePosition(index)
  const composeTemplate = kind === "set_var" ? "null" : "{}"
  const initialSpec = kindDefaultSpec(kind)
  return {
    id,
    kind,
    allowFail: false,
    retry: 1,
    retryBackoffMs: 0,
    timeoutMs: 3000,
    method: "",
    target: 0,
    argsTemplate: "{}",
    composeTemplate,
    setVarName: "",
    inputs: [],
    ...defaultAdvancedNodeFields(),
    specEditorMode: defaultSpecEditorMode(kind),
    specJson: formatJSONText(initialSpec, {}),
    x: pos.x,
    y: pos.y
  }
}

const mapNode = (input: any, index: number, options: ParseDraftOptions = {}): FlowNodeDraft => {
  const sourceKind = String(input?.kind ?? "").toLowerCase()
  const kind = normalizeNodeKind(sourceKind)
  const {
    method,
    target,
    argsTemplate,
    composeTemplate,
    setVarName,
    inputs,
    transformExprMode,
    transformLiteralJson,
    transformSource,
    transformSourceRequired,
    transformOp,
    transformArgsJson,
    transformObjectJson,
    transformArrayJson,
    branchCases,
    branchDefaultCase,
    foreachSource,
    foreachRequired,
    foreachBodyJson,
    foreachResultNodeId,
    subflowId,
    subflowInputTemplate,
    subflowResultNodeId,
    specEditorMode,
    specJson,
    x,
    y
  } = parseSpecDraft(kind, input?.spec, options)
  let mappedTarget = Number.isFinite(target) ? Number(target) : 0
  if (mappedTarget < 0) mappedTarget = 0
  if (sourceKind === "local") {
    mappedTarget = 0
  }
  const pos = defaultNodePosition(index)
  return {
    id: String(input?.id ?? "").trim(),
    kind,
    allowFail: Boolean(input?.allow_fail ?? input?.allowFail ?? false),
    retry: Number(input?.retry ?? 1),
    retryBackoffMs: Number(input?.retry_backoff_ms ?? input?.retryBackoffMs ?? 0),
    timeoutMs: Number(input?.timeout_ms ?? input?.timeoutMs ?? 3000),
    method,
    target: mappedTarget,
    argsTemplate,
    composeTemplate,
    setVarName,
    inputs,
    transformExprMode,
    transformLiteralJson,
    transformSource,
    transformSourceRequired,
    transformOp,
    transformArgsJson,
    transformObjectJson,
    transformArrayJson,
    branchCases,
    branchDefaultCase,
    foreachSource,
    foreachRequired,
    foreachBodyJson,
    foreachResultNodeId,
    subflowId,
    subflowInputTemplate,
    subflowResultNodeId,
    specEditorMode,
    specJson,
    x: Number.isFinite(x) ? Number(x) : pos.x,
    y: Number.isFinite(y) ? Number(y) : pos.y
  }
}

const mapEdge = (input: any): FlowEdge => ({
  from: String(input?.from ?? "").trim(),
  to: String(input?.to ?? "").trim(),
  case: String(input?.case ?? "").trim() || undefined
})

export const createGraphEditorStateFromDraft = (
  graphSource: any,
  options: ParseDraftOptions = {}
): FlowGraphEditorState => {
  const graph = graphSource && typeof graphSource === "object" && "graph" in graphSource ? graphSource.graph : graphSource
  const nodes = Array.isArray(graph?.nodes) ? graph.nodes : []
  const edges = Array.isArray(graph?.edges) ? graph.edges : []
  return {
    nodes: nodes.map((node: any, index: number) => mapNode(node, index, options)),
    edges: edges.map(mapEdge),
    selectedNodeIndex: -1,
    selectedEdgeIndex: -1
  }
}

const mapExecCapabilityRoute = (input: any): ExecCapabilityRoute => {
  const providerNode = Number(input?.provider_node ?? 0)
  const viaNode = Number(input?.via_node ?? 0)
  const method = String(input?.method ?? "").trim()
  const version = String(input?.version ?? "").trim()
  const defaultTimeoutMs = Number(input?.default_timeout_ms ?? 0)
  const permissions = Array.isArray(input?.permissions)
    ? input.permissions.map((item: any) => String(item ?? "").trim()).filter(Boolean)
    : []
  const tags =
    input?.tags && typeof input.tags === "object" && !Array.isArray(input.tags)
      ? Object.fromEntries(
          Object.entries(input.tags as Record<string, unknown>)
            .map(([key, value]) => [String(key).trim(), String(value ?? "").trim()])
            .filter(([key]) => Boolean(key))
        )
      : {}
  const providerText = providerNode > 0 ? String(providerNode) : "-"
  const viaText = viaNode > 0 ? ` via ${viaNode}` : ""
  const versionText = version ? `@${version}` : ""
  const label = `${providerText} · ${method}${versionText}${viaText}`
  return {
    key: `${providerNode}|${viaNode}|${method}|${version}`,
    providerNode: providerNode > 0 ? Math.trunc(providerNode) : 0,
    viaNode: viaNode > 0 ? Math.trunc(viaNode) : 0,
    method,
    version,
    defaultTimeoutMs: Number.isFinite(defaultTimeoutMs) && defaultTimeoutMs > 0 ? Math.trunc(defaultTimeoutMs) : 0,
    permissions,
    tags,
    inputSchema: input?.input_schema ?? null,
    outputSchema: input?.output_schema ?? null,
    label
  }
}

const beginExecCapabilityLoad = () => {
  const epoch = execCapabilityLoadEpoch
  execCapabilityLoadCount += 1
  state.execCapabilitiesLoading = execCapabilityLoadCount > 0
  return epoch
}

const endExecCapabilityLoad = (epoch: number) => {
  if (epoch !== execCapabilityLoadEpoch) {
    return
  }
  execCapabilityLoadCount = Math.max(0, execCapabilityLoadCount - 1)
  state.execCapabilitiesLoading = execCapabilityLoadCount > 0
}

const replaceExecCapabilityRoutes = (routes: ExecCapabilityRoute[]) => {
  state.execCapabilities = routes
}

const mergeExecCapabilityRoutes = (routes: ExecCapabilityRoute[]) => {
  if (!routes.length) {
    return
  }
  const merged = new Map(state.execCapabilities.map((route) => [route.key, route] as const))
  for (const route of routes) {
    merged.set(route.key, route)
  }
  state.execCapabilities = Array.from(merged.values())
}

const hasExecCapabilityRoute = (method: string, providerNode: number) =>
  state.execCapabilities.some(
    (route) => route.method === method && route.providerNode === providerNode
  )

// resetExecCapabilityState 在执行节点、图结构或草稿切换后让旧 capability 缓存整体失效。
const resetExecCapabilityState = () => {
  execCapabilityLoadEpoch += 1
  execCapabilityCacheVersion += 1
  state.execCapabilities = []
  state.execCapabilitiesLoading = false
  execCapabilityLoadCount = 0
  pendingCapabilityHydrations.clear()
}

// newDraft 重置当前 flow 草稿，并把历史、状态、节点详情都恢复到初始值。
const newDraft = () => {
  state.flowId = ""
  state.flowName = ""
  state.maxActiveRuns = null
  state.triggerType = "interval"
  state.eventMode = "publish"
  state.everyMs = 60000
  state.cronExpr = ""
  state.eventName = ""
  state.eventTopic = ""
  state.dedupWindowMs = 0
  state.varOwner = 0
  state.varName = ""
  state.nodes = []
  state.edges = []
  state.selectedNodeIndex = -1
  state.selectedEdgeIndex = -1
  resetStatusState()
  state.nodeDetail = createNodeDetailState()
  resetExecCapabilityState()
  resetHistory()
}

const suggestNodeId = (prefix = "n") => {
  const normalizedPrefix = prefix.trim() || "n"
  const used = new Set(state.nodes.map((node) => node.id.trim()).filter(Boolean))
  for (let i = 1; i <= 99999; i += 1) {
    const candidate = `${normalizedPrefix}${i}`
    if (!used.has(candidate)) return candidate
  }
  return `${normalizedPrefix}${Date.now().toString(36)}`
}

// addNode 负责创建新节点、切换选中态，并把本次画布变更写入历史栈。
const addNode = (id: string, kind: FlowNodeKind = "call") => {
  const trimmed = id.trim()
  if (!trimmed) {
    throw new Error(t("Node ID is required."))
  }
  if (state.nodes.find((node) => node.id.trim() === trimmed)) {
    throw new Error(t("Node ID must be unique."))
  }
  const node = createNodeDraft(trimmed, kind, state.nodes.length)
  state.nodes.push(node)
  state.selectedNodeIndex = state.nodes.length - 1
  state.selectedEdgeIndex = -1
  commitHistory()
}

// renameNodeId 在改名时同步重写所有边引用，避免图结构出现悬空节点。
const renameNodeId = (oldId: string, newId: string) => {
  const from = oldId.trim()
  const to = newId.trim()
  if (!from) {
    throw new Error(t("Old node ID is required."))
  }
  if (!to) {
    throw new Error(t("Node ID is required."))
  }
  if (from === to) {
    return false
  }

  const node = state.nodes.find((n) => n.id.trim() === from)
  if (!node) {
    throw new Error(t("Node does not exist."))
  }
  if (state.nodes.some((n) => n.id.trim() === to)) {
    throw new Error(t("Node ID must be unique."))
  }

  node.id = to
  state.edges = state.edges.map((edge) => ({
    from: edge.from.trim() === from ? to : edge.from,
    to: edge.to.trim() === from ? to : edge.to
  }))
  commitHistory()
  setMessage(t("Node ID updated."), "success")
  return true
}

const removeSelectedNode = () => {
  const idx = state.selectedNodeIndex
  if (idx < 0 || idx >= state.nodes.length) return
  const removed = state.nodes[idx]
  state.nodes = state.nodes.filter((_, i) => i !== idx)
  state.edges = state.edges.filter(
    (edge) => edge.from.trim() !== removed.id.trim() && edge.to.trim() !== removed.id.trim()
  )
  state.selectedNodeIndex = -1
  state.selectedEdgeIndex = -1
  commitHistory()
}

const buildAdjacency = (edges: FlowEdge[]) => {
  const next = new Map<string, string[]>()
  for (const edge of edges) {
    const from = edge.from.trim()
    const to = edge.to.trim()
    if (!from || !to) continue
    const list = next.get(from)
    if (list) {
      list.push(to)
    } else {
      next.set(from, [to])
    }
  }
  return next
}

const buildParents = (edges: FlowEdge[]) => {
  const parents = new Map<string, string[]>()
  for (const edge of edges) {
    const from = edge.from.trim()
    const to = edge.to.trim()
    if (!from || !to) continue
    const list = parents.get(to)
    if (list) {
      list.push(from)
    } else {
      parents.set(to, [from])
    }
  }
  return parents
}

const collectNodeIdsFromNodes = (nodes: FlowNodeDraft[]) => {
  const seen = new Set<string>()
  const ids: string[] = []
  for (const node of nodes) {
    const id = node.id.trim()
    if (!id) {
      throw new Error(t("Node ID is required."))
    }
    if (seen.has(id)) {
      throw new Error(t("Node ID must be unique."))
    }
    seen.add(id)
    ids.push(id)
  }
  return ids
}

const collectNodeIds = () => collectNodeIdsFromNodes(state.nodes)

const buildTopology = (nodeIds: string[], edges: FlowEdge[]) => {
  const idSet = new Set(nodeIds)
  const nodeOrder = new Map<string, number>()
  for (const [idx, id] of nodeIds.entries()) {
    nodeOrder.set(id, idx)
  }

  const indegree = new Map<string, number>()
  const next = new Map<string, string[]>()
  const parents = new Map<string, string[]>()
  for (const id of nodeIds) {
    indegree.set(id, 0)
    next.set(id, [])
    parents.set(id, [])
  }

  for (const edge of edges) {
    const from = edge.from.trim()
    const to = edge.to.trim()
    if (!from || !to || from === to) {
      throw new Error(t("Edge endpoints are invalid."))
    }
    if (!idSet.has(from) || !idSet.has(to)) {
      throw new Error(t("Edge references unknown nodes."))
    }
    next.get(from)?.push(to)
    parents.get(to)?.push(from)
    indegree.set(to, (indegree.get(to) ?? 0) + 1)
  }

  const queue: string[] = []
  const level = new Map<string, number>()
  for (const id of nodeIds) {
    if ((indegree.get(id) ?? 0) === 0) {
      queue.push(id)
      level.set(id, 0)
    }
  }
  queue.sort((a, b) => (nodeOrder.get(a) ?? 0) - (nodeOrder.get(b) ?? 0))

  const order: string[] = []
  while (queue.length) {
    const current = queue.shift()
    if (!current) continue
    order.push(current)
    const depth = level.get(current) ?? 0
    for (const child of next.get(current) ?? []) {
      level.set(child, Math.max(level.get(child) ?? 0, depth + 1))
      const left = (indegree.get(child) ?? 0) - 1
      indegree.set(child, left)
      if (left === 0) {
        queue.push(child)
        queue.sort((a, b) => (nodeOrder.get(a) ?? 0) - (nodeOrder.get(b) ?? 0))
      }
    }
  }

  if (order.length !== nodeIds.length) {
    throw new Error(t("Flow graph must be a DAG (cycle detected)."))
  }

  return { order, next, parents, level }
}

const buildAncestorMap = (order: string[], parents: Map<string, string[]>) => {
  const ancestors = new Map<string, Set<string>>()
  for (const id of order) {
    ancestors.set(id, new Set<string>())
  }
  for (const id of order) {
    const own = ancestors.get(id) ?? new Set<string>()
    for (const parentId of parents.get(id) ?? []) {
      own.add(parentId)
      const parentAncestors = ancestors.get(parentId)
      if (!parentAncestors) continue
      for (const ancestorId of parentAncestors) {
        own.add(ancestorId)
      }
    }
    ancestors.set(id, own)
  }
  return ancestors
}

const isReachable = (start: string, goal: string, next: Map<string, string[]>) => {
  const queue: string[] = [start]
  const visited = new Set<string>()
  while (queue.length) {
    const cur = queue.shift()
    if (!cur) continue
    if (cur === goal) return true
    if (visited.has(cur)) continue
    visited.add(cur)
    const children = next.get(cur)
    if (children?.length) {
      queue.push(...children)
    }
  }
  return false
}

const decodeJSONPointerToken = (token: string) => {
  if (!token.includes("~")) {
    return token
  }
  let out = ""
  for (let i = 0; i < token.length; i += 1) {
    const ch = token[i]
    if (ch !== "~") {
      out += ch
      continue
    }
    const next = token[i + 1]
    if (next === "0") {
      out += "~"
      i += 1
      continue
    }
    if (next === "1") {
      out += "/"
      i += 1
      continue
    }
    throw new Error(t("JSON Pointer contains an invalid escape sequence."))
  }
  return out
}

const validateJSONPointer = (pointer: string, errorMessage: string) => {
  const trimmed = String(pointer ?? "").trim()
  if (!trimmed) {
    return
  }
  if (!trimmed.startsWith("/")) {
    throw new Error(errorMessage)
  }
  for (const part of trimmed.slice(1).split("/")) {
    decodeJSONPointerToken(part)
  }
}

const parseJSONText = (raw: string, errorMessage: string) => {
  try {
    return JSON.parse(raw.trim() || "{}")
  } catch {
    throw new Error(errorMessage)
  }
}

const tryParseJSONText = (raw: string, fallback: any = {}) => {
  try {
    return JSON.parse(String(raw ?? "").trim() || JSON.stringify(fallback))
  } catch {
    return fallback
  }
}

const parseArgsTemplateObject = (node: FlowNodeDraft) => {
  const nodeId = node.id.trim() || t("Unnamed")
  const parsed = parseJSONText(
    node.argsTemplate,
    t("Node {nodeId} args template must be valid JSON.", { nodeId })
  )
  if (!isJSONObject(parsed)) {
    throw new Error(t("Node {nodeId} args template must be a JSON object.", { nodeId }))
  }
  return parsed as Record<string, unknown>
}

const parseTemplateValue = (raw: string, errorMessage: string, blankFallback = "{}") => {
  try {
    return JSON.parse(String(raw ?? "").trim() || blankFallback)
  } catch {
    throw new Error(errorMessage)
  }
}

const parseComposeTemplateObject = (node: FlowNodeDraft) => {
  const nodeId = node.id.trim() || t("Unnamed")
  const parsed = parseTemplateValue(
    node.composeTemplate,
    t("Node {nodeId} template must be valid JSON.", { nodeId }),
    "{}"
  )
  if (!isJSONObject(parsed)) {
    throw new Error(t("Node {nodeId} template must be a JSON object.", { nodeId }))
  }
  return parsed as Record<string, unknown>
}

const normalizeFlowLocalVarName = (raw: unknown) => String(raw ?? "").trim()

const assertValidFlowLocalVarName = (name: string, requiredMessage: string, invalidMessage: string) => {
  if (!name) {
    throw new Error(requiredMessage)
  }
  if (!FLOW_LOCAL_VAR_NAME_PATTERN.test(name)) {
    throw new Error(invalidMessage)
  }
  return name
}

const requireNonNegativeInteger = (value: unknown, errorMessage: string) => {
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed < 0) {
    throw new Error(errorMessage)
  }
  return Math.trunc(parsed)
}

const preferObjectTemplateText = (raw: string, fallback = "{}") => {
  const trimmed = String(raw ?? "").trim()
  if (!trimmed) {
    return fallback
  }
  try {
    const parsed = JSON.parse(trimmed)
    return isJSONObject(parsed) ? formatJSONText(parsed, {}) : fallback
  } catch {
    return trimmed
  }
}

const isBindingBlank = (binding: FlowInputBindingDraft) =>
  !binding.to.trim() &&
  !binding.sourceKind &&
  !binding.nodeId.trim() &&
  !binding.path.trim() &&
  !binding.field.trim() &&
  !binding.name.trim() &&
  !binding.required

const buildInputBindings = (
  node: FlowNodeDraft,
  ancestors: Map<string, Set<string>>,
  options: StrictGraphBuildOptions = {}
) => {
  const nodeId = node.id.trim() || t("Unnamed")
  const allowedAncestors = ancestors.get(node.id.trim()) ?? new Set<string>()
  const out: Array<Record<string, any>> = []
  for (const [index, binding] of node.inputs.entries()) {
    if (isBindingBlank(binding)) {
      continue
    }
    const rowNo = index + 1
    const to = binding.to.trim()
    if (!to) {
      throw new Error(t("Node {nodeId} binding row {rowNo}: destination pointer is required.", { nodeId, rowNo }))
    }
    validateJSONPointer(
      to,
      t("Node {nodeId} binding row {rowNo}: destination pointer must be a valid JSON Pointer.", { nodeId, rowNo })
    )
    const sourceKind = binding.sourceKind
    let source: Record<string, any>
    switch (sourceKind) {
      case "node_result": {
        const sourceNodeId = binding.nodeId.trim()
        if (!sourceNodeId) {
          throw new Error(t("Node {nodeId} binding row {rowNo}: source node is required.", { nodeId, rowNo }))
        }
        if (!allowedAncestors.has(sourceNodeId)) {
          throw new Error(t("Node {nodeId} binding row {rowNo}: source node must be an ancestor.", { nodeId, rowNo }))
        }
        validateJSONPointer(
          binding.path,
          t("Node {nodeId} binding row {rowNo}: result path must be a valid JSON Pointer.", { nodeId, rowNo })
        )
        source = { kind: "node_result", node_id: sourceNodeId }
        if (binding.path.trim()) {
          source.path = binding.path.trim()
        }
        break
      }
      case "trigger":
        validateJSONPointer(
          binding.path,
          t("Node {nodeId} binding row {rowNo}: trigger path must be a valid JSON Pointer.", { nodeId, rowNo })
        )
        source = { kind: "trigger" }
        if (binding.path.trim()) {
          source.path = binding.path.trim()
        }
        break
      case "flow_meta":
        if (binding.field.trim() !== "flow_id") {
          throw new Error(t("Node {nodeId} binding row {rowNo}: flow meta field must be flow_id.", { nodeId, rowNo }))
        }
        source = { kind: "flow_meta", field: "flow_id" }
        break
      case "run_meta":
        if (binding.field.trim() !== "run_id") {
          throw new Error(t("Node {nodeId} binding row {rowNo}: run meta field must be run_id.", { nodeId, rowNo }))
        }
        source = { kind: "run_meta", field: "run_id" }
        break
      case "loop_item":
        if (!options.allowLoopSources) {
          throw new Error(t("Node {nodeId} binding row {rowNo}: loop sources are only available inside foreach body graphs.", { nodeId, rowNo }))
        }
        validateJSONPointer(
          binding.path,
          t("Node {nodeId} binding row {rowNo}: loop item path must be a valid JSON Pointer.", { nodeId, rowNo })
        )
        source = { kind: "loop_item" }
        if (binding.path.trim()) {
          source.path = binding.path.trim()
        }
        break
      case "loop_index":
        if (!options.allowLoopSources) {
          throw new Error(t("Node {nodeId} binding row {rowNo}: loop sources are only available inside foreach body graphs.", { nodeId, rowNo }))
        }
        if (binding.path.trim()) {
          throw new Error(t("Node {nodeId} binding row {rowNo}: loop index does not accept a JSON Pointer path.", { nodeId, rowNo }))
        }
        source = { kind: "loop_index" }
        break
      case "flow_var": {
        const name = assertValidFlowLocalVarName(
          normalizeFlowLocalVarName(binding.name),
          t("Node {nodeId} binding row {rowNo}: flow local var name is required.", { nodeId, rowNo }),
          t("Node {nodeId} binding row {rowNo}: flow local var name is invalid.", { nodeId, rowNo })
        )
        validateJSONPointer(
          binding.path,
          t("Node {nodeId} binding row {rowNo}: flow local var path must be a valid JSON Pointer.", {
            nodeId,
            rowNo
          })
        )
        source = { kind: "flow_var", name }
        if (binding.path.trim()) {
          source.path = binding.path.trim()
        }
        break
      }
      default:
        throw new Error(t("Node {nodeId} binding row {rowNo}: source kind is required.", { nodeId, rowNo }))
    }
    out.push({
      to,
      source,
      required: Boolean(binding.required)
    })
  }
  return out
}

const buildLooseSpecFromNode = (node: FlowNodeDraft) => {
  const ui = { x: Math.round(Number(node.x || 0)), y: Math.round(Number(node.y || 0)) }
  const looseInputs = node.inputs.filter((binding) => !isBindingBlank(binding)).map((binding) => {
    const sourceKind = binding.sourceKind || "node_result"
    const source: Record<string, any> = { kind: sourceKind }
    if (sourceKind === "node_result") {
      if (binding.nodeId.trim()) source.node_id = binding.nodeId.trim()
      if (binding.path.trim()) source.path = binding.path.trim()
    } else if (sourceKind === "trigger") {
      if (binding.path.trim()) source.path = binding.path.trim()
    } else if (sourceKind === "loop_item") {
      if (binding.path.trim()) source.path = binding.path.trim()
    } else if (sourceKind === "loop_index") {
      // loop_index intentionally has no extra fields
    } else if (sourceKind === "flow_meta" || sourceKind === "run_meta") {
      if (binding.field.trim()) source.field = binding.field.trim()
    } else if (sourceKind === "flow_var") {
      if (binding.name.trim()) source.name = binding.name.trim()
      if (binding.path.trim()) source.path = binding.path.trim()
    }
    return {
      to: binding.to.trim(),
      source,
      required: Boolean(binding.required)
    }
  })

  if (node.kind === "compose") {
    return {
      template: tryParseJSONText(node.composeTemplate, {}),
      ...(looseInputs.length ? { inputs: looseInputs } : {}),
      _ui: ui
    }
  }
  if (node.kind === "set_var") {
    return {
      name: node.setVarName.trim(),
      template: tryParseJSONText(node.composeTemplate, null),
      ...(looseInputs.length ? { inputs: looseInputs } : {}),
      _ui: ui
    }
  }
  if (node.kind === "transform") {
    const mode = node.transformExprMode || "literal"
    if (mode === "source") {
      return {
        expr: {
          source: buildLooseSourceSpec(node.transformSource),
          required: Boolean(node.transformSourceRequired)
        },
        _ui: ui
      }
    }
    if (mode === "op") {
      return {
        expr: {
          op: String(node.transformOp ?? "").trim().toLowerCase(),
          args: tryParseJSONText(node.transformArgsJson, [])
        },
        _ui: ui
      }
    }
    if (mode === "object") {
      return {
        expr: {
          object: tryParseJSONText(node.transformObjectJson, {})
        },
        _ui: ui
      }
    }
    if (mode === "array") {
      return {
        expr: {
          array: tryParseJSONText(node.transformArrayJson, [])
        },
        _ui: ui
      }
    }
    return {
      expr: {
        literal: tryParseJSONText(node.transformLiteralJson, null)
      },
      _ui: ui
    }
  }
  if (node.kind === "branch") {
    return {
      cases: node.branchCases.map((item) => {
        const op = FLOW_BRANCH_MATCH_OP_SET.has(item.op) ? item.op : "eq"
        return {
          name: item.name.trim(),
          match: {
            source: buildLooseSourceSpec(item.source),
            op,
            ...(op === "exists" ? {} : { value: tryParseJSONText(item.valueJson, null) })
          }
        }
      }),
      ...(node.branchDefaultCase.trim() ? { default_case: node.branchDefaultCase.trim() } : {}),
      _ui: ui
    }
  }
  if (node.kind === "foreach") {
    const parsedBody = tryParseJSONText(node.foreachBodyJson, defaultForeachBodyGraph())
    const body = isJSONObject(parsedBody) ? { ...parsedBody } : defaultForeachBodyGraph()
    if (!Array.isArray(body.nodes)) {
      body.nodes = []
    }
    if (!Array.isArray(body.edges)) {
      body.edges = []
    }
    return {
      source: buildLooseSourceSpec(node.foreachSource),
      required: Boolean(node.foreachRequired),
      body,
      result_node_id: node.foreachResultNodeId.trim(),
      _ui: ui
    }
  }
  if (node.kind === "subflow") {
    const inputTemplate = tryParseJSONText(node.subflowInputTemplate, {})
    return {
      flow_id: node.subflowId.trim(),
      input_template: isJSONObject(inputTemplate) ? inputTemplate : {},
      ...(looseInputs.length ? { inputs: looseInputs } : {}),
      ...(node.subflowResultNodeId.trim() ? { result_node_id: node.subflowResultNodeId.trim() } : {}),
      _ui: ui
    }
  }
  if (!supportsFormMode(node.kind)) {
    const parsed = (() => {
      try {
        return parseSpecJsonObject(node)
      } catch {
        return kindDefaultSpec(node.kind)
      }
    })()
    return {
      ...parsed,
      _ui: ui
    }
  }

  const target = Number(node.target || 0)
  return {
    ...(Number.isFinite(target) && target > 0 ? { target: Math.trunc(target) } : {}),
    method: node.method.trim(),
    args_template: tryParseJSONText(node.argsTemplate, {}),
    ...(looseInputs.length ? { inputs: looseInputs } : {}),
    _ui: ui
  }
}

const buildLooseExportSpecFromNode = (node: FlowNodeDraft) => {
  if (node.specEditorMode !== "json" && supportsFormMode(node.kind)) {
    return buildLooseSpecFromNode(node)
  }
  const parsed = parseSpecJsonObject(node)
  const ui = { x: Math.round(Number(node.x || 0)), y: Math.round(Number(node.y || 0)) }
  return {
    ...parsed,
    _ui: ui
  }
}

export const exportLooseGraphDraftFromEditorState = (snapshot: FlowGraphEditorState): FlowGraphDraft => ({
  nodes: (Array.isArray(snapshot?.nodes) ? snapshot.nodes : []).map((node) => {
    const id = node.id.trim()
    return {
      id,
      kind: node.kind,
      allow_fail: Boolean(node.allowFail),
      retry: Math.max(0, Math.trunc(Number(node.retry || 0))),
      retry_backoff_ms: Math.max(0, Math.trunc(Number(node.retryBackoffMs || 0))),
      timeout_ms: Math.max(0, Math.trunc(Number(node.timeoutMs || 0))),
      spec: buildLooseExportSpecFromNode(node)
    }
  }),
  edges: (Array.isArray(snapshot?.edges) ? snapshot.edges : []).map((edge) => ({
    from: edge.from.trim(),
    to: edge.to.trim(),
    ...(edge.case?.trim() ? { case: edge.case.trim() } : {})
  }))
})

const addEdge = (from: string, to: string) => {
  const fromId = from.trim()
  const toId = to.trim()
  if (!fromId || !toId || fromId === toId) {
    throw new Error(t("Edge endpoints must be different."))
  }
  if (!state.nodes.find((node) => node.id.trim() === fromId)) {
    throw new Error(t("From node does not exist."))
  }
  if (!state.nodes.find((node) => node.id.trim() === toId)) {
    throw new Error(t("To node does not exist."))
  }
  if (state.edges.some((edge) => edge.from === fromId && edge.to === toId)) {
    throw new Error(t("Edge already exists."))
  }
  const next = buildAdjacency(state.edges)
  if (isReachable(toId, fromId, next)) {
    throw new Error(t("Edge would create a cycle."))
  }
  state.edges.push({ from: fromId, to: toId })
  state.selectedEdgeIndex = state.edges.length - 1
  state.selectedNodeIndex = -1
  commitHistory()
}

const removeSelectedEdge = () => {
  const idx = state.selectedEdgeIndex
  if (idx < 0 || idx >= state.edges.length) return
  state.edges = state.edges.filter((_, i) => i !== idx)
  state.selectedEdgeIndex = -1
  commitHistory()
}

const autoLayoutTB = () => {
  if (!state.nodes.length) {
    throw new Error(t("No nodes to layout."))
  }

  const ids = collectNodeIds()
  const nodeOrder = new Map<string, number>()
  for (const [idx, id] of ids.entries()) {
    nodeOrder.set(id, idx)
  }
  const topology = buildTopology(ids, state.edges)

  const groups = new Map<number, string[]>()
  for (const id of topology.order) {
    const depth = topology.level.get(id) ?? 0
    const list = groups.get(depth)
    if (list) {
      list.push(id)
    } else {
      groups.set(depth, [id])
    }
  }

  const levels = [...groups.keys()].sort((a, b) => a - b)
  const maxWidth = Math.max(...levels.map((d) => groups.get(d)?.length ?? 0), 1)
  const xGap = 240
  const yGap = 170

  const positions = new Map<string, { x: number; y: number }>()
  for (const depth of levels) {
    const list = (groups.get(depth) ?? []).slice()
    list.sort((a, b) => (nodeOrder.get(a) ?? 0) - (nodeOrder.get(b) ?? 0))
    const offset = ((maxWidth - list.length) * xGap) / 2
    for (const [idx, id] of list.entries()) {
      positions.set(id, { x: Math.round(offset + idx * xGap), y: Math.round(depth * yGap) })
    }
  }

  for (const node of state.nodes) {
    const pos = positions.get(node.id.trim())
    if (!pos) continue
    node.x = pos.x
    node.y = pos.y
  }

  commitHistory()
}

const assertValidSubflowFlowID = (flowId: string, nodeId: string, currentFlowId = "") => {
  if (!FLOW_UUID_PATTERN.test(flowId)) {
    throw new Error(t("Node {nodeId} subflow flow_id must be a UUID.", { nodeId }))
  }
  if (currentFlowId && flowId === currentFlowId) {
    throw new Error(t("Node {nodeId} subflow flow_id must not call the current flow itself.", { nodeId }))
  }
  return flowId
}

const validateRawSourceSpec = (input: {
  raw: any
  nodeId: string
  label: string
  ancestors: Map<string, Set<string>>
  allowLoopSources?: boolean
}) => {
  if (!isJSONObject(input.raw)) {
    throw new Error(t("Node {nodeId} {label}: source must be a JSON object.", { nodeId: input.nodeId, label: input.label }))
  }
  const parsedSource = parseSourceDraft(input.raw, "trigger", { allowLoopSources: input.allowLoopSources })
  if (!parsedSource.supported) {
    throw new Error(t("Node {nodeId} {label}: source kind is invalid or not visible in this scope.", { nodeId: input.nodeId, label: input.label }))
  }
  buildSourceSpec({
    nodeId: input.nodeId,
    label: input.label,
    source: parsedSource.source,
    ancestors: input.ancestors,
    allowLoopSources: input.allowLoopSources
  })
}

const validateRawTransformExpr = (input: {
  expr: any
  nodeId: string
  label: string
  ancestors: Map<string, Set<string>>
  allowLoopSources?: boolean
}) => {
  const expr = input.expr
  if (!isJSONObject(expr)) {
    throw new Error(t("Node {nodeId} {label} must be a JSON object.", { nodeId: input.nodeId, label: input.label }))
  }
  const hasLiteral = hasOwn(expr, "literal")
  const hasSource = hasOwn(expr, "source")
  const hasOp = hasOwn(expr, "op") || hasOwn(expr, "args")
  const hasObject = hasOwn(expr, "object")
  const hasArray = hasOwn(expr, "array")
  const variantCount = [hasLiteral, hasSource, hasOp, hasObject, hasArray].filter(Boolean).length
  if (variantCount !== 1) {
    throw new Error(t("Node {nodeId} {label} must define exactly one transform expression variant.", { nodeId: input.nodeId, label: input.label }))
  }
  if (hasSource) {
    validateRawSourceSpec({
      raw: expr.source,
      nodeId: input.nodeId,
      label: `${input.label} ${t("source")}`,
      ancestors: input.ancestors,
      allowLoopSources: input.allowLoopSources
    })
    return
  }
  if (hasOp) {
    const op = String(expr.op ?? "").trim().toLowerCase()
    if (!FLOW_TRANSFORM_OPS.has(op)) {
      throw new Error(t("Node {nodeId} {label} transform op is invalid.", { nodeId: input.nodeId, label: input.label }))
    }
    if (!Array.isArray(expr.args)) {
      throw new Error(t("Node {nodeId} {label} transform args must be a JSON array.", { nodeId: input.nodeId, label: input.label }))
    }
    expr.args.forEach((arg: unknown, index: number) =>
      validateRawTransformExpr({
        expr: arg,
        nodeId: input.nodeId,
        label: t("transform arg {index}", { index: index + 1 }),
        ancestors: input.ancestors,
        allowLoopSources: input.allowLoopSources
      })
    )
    return
  }
  if (hasObject) {
    if (!isJSONObject(expr.object)) {
      throw new Error(t("Node {nodeId} {label} transform object must be a JSON object.", { nodeId: input.nodeId, label: input.label }))
    }
    Object.entries(expr.object).forEach(([key, value]) =>
      validateRawTransformExpr({
        expr: value,
        nodeId: input.nodeId,
        label: t("transform object field {key}", { key }),
        ancestors: input.ancestors,
        allowLoopSources: input.allowLoopSources
      })
    )
    return
  }
  if (hasArray) {
    if (!Array.isArray(expr.array)) {
      throw new Error(t("Node {nodeId} {label} transform array must be a JSON array.", { nodeId: input.nodeId, label: input.label }))
    }
    expr.array.forEach((value: unknown, index: number) =>
      validateRawTransformExpr({
        expr: value,
        nodeId: input.nodeId,
        label: t("transform array item {index}", { index: index + 1 }),
        ancestors: input.ancestors,
        allowLoopSources: input.allowLoopSources
      })
    )
  }
}

const validateRawBranchSpec = (input: {
  parsed: Record<string, any>
  nodeId: string
  ancestors: Map<string, Set<string>>
  allowLoopSources?: boolean
}) => {
  const seen = new Set<string>()
  const cases = Array.isArray(input.parsed.cases) ? input.parsed.cases : []
  cases.forEach((item, index) => {
    if (!isJSONObject(item)) {
      throw new Error(t("Node {nodeId} branch case {index} must be a JSON object.", { nodeId: input.nodeId, index: index + 1 }))
    }
    const name = String(item.name ?? "").trim()
    if (!name) {
      throw new Error(t("Node {nodeId} branch case {index} requires a name.", { nodeId: input.nodeId, index: index + 1 }))
    }
    if (seen.has(name)) {
      throw new Error(t("Node {nodeId} branch case name {name} is duplicated.", { nodeId: input.nodeId, name }))
    }
    seen.add(name)
    if (!isJSONObject(item.match)) {
      throw new Error(t("Node {nodeId} branch case {name} match must be a JSON object.", { nodeId: input.nodeId, name }))
    }
    const match = item.match as Record<string, any>
    const op = String(match.op ?? "").trim().toLowerCase() as FlowBranchMatchOp
    if (!FLOW_BRANCH_MATCH_OP_SET.has(op)) {
      throw new Error(t("Node {nodeId} branch case {name} match op is invalid.", { nodeId: input.nodeId, name }))
    }
    validateRawSourceSpec({
      raw: match.source,
      nodeId: input.nodeId,
      label: t("branch case {name}", { name }),
      ancestors: input.ancestors,
      allowLoopSources: input.allowLoopSources
    })
  })
  const defaultCase = String(input.parsed.default_case ?? "").trim()
  if (defaultCase && !seen.has(defaultCase)) {
    throw new Error(t("Node {nodeId} default case must match an existing branch case.", { nodeId: input.nodeId }))
  }
}

const buildStrictGraphDraftFromWireGraph = (
  graph: { nodes?: any[]; edges?: any[] },
  options: StrictGraphBuildOptions = {}
) => {
  const snapshot = createGraphEditorStateFromDraft(graph, { allowLoopSources: options.allowLoopSources })
  return buildGraphDraftFromState(snapshot.nodes, snapshot.edges, options)
}

const buildStrictForeachBodyGraph = (
  body: Record<string, any>,
  nodeId: string,
  currentFlowId = ""
) => {
  if (!Array.isArray(body.nodes)) {
    throw new Error(t("Node {nodeId} foreach body must include a nodes array.", { nodeId }))
  }
  if (!Array.isArray(body.edges)) {
    throw new Error(t("Node {nodeId} foreach body must include an edges array.", { nodeId }))
  }
  try {
    return buildStrictGraphDraftFromWireGraph(body, {
      allowLoopSources: true,
      currentFlowId
    })
  } catch (err) {
    throw new Error(
      t("Node {nodeId} foreach body is invalid: {message}", {
        nodeId,
        message: String((err as Error)?.message ?? err ?? t("Unknown validation error."))
      })
    )
  }
}

const buildFormSpec = (
  node: FlowNodeDraft,
  ancestors: Map<string, Set<string>>,
  options: StrictGraphBuildOptions = {}
) => {
  if (!supportsFormMode(node.kind)) {
    throw new Error(t("Node kind {kind} only supports Advanced JSON mode right now.", { kind: node.kind }))
  }
  const nodeId = node.id.trim() || t("Unnamed")
  const ui = { x: Math.round(Number(node.x || 0)), y: Math.round(Number(node.y || 0)) }
  if (node.kind === "compose") {
    const inputs = buildInputBindings(node, ancestors, options)
    return {
      template: parseComposeTemplateObject(node),
      ...(inputs.length ? { inputs } : {}),
      _ui: ui
    }
  }
  if (node.kind === "set_var") {
    const inputs = buildInputBindings(node, ancestors, options)
    const name = assertValidFlowLocalVarName(
      normalizeFlowLocalVarName(node.setVarName),
      t("Node {nodeId} requires a flow local var name.", { nodeId }),
      t("Node {nodeId} flow local var name is invalid.", { nodeId })
    )
    return {
      name,
      template: parseTemplateValue(
        node.composeTemplate,
        t("Node {nodeId} template must be valid JSON.", { nodeId }),
        "null"
      ),
      ...(inputs.length ? { inputs } : {}),
      _ui: ui
    }
  }
  if (node.kind === "transform") {
    const mode = node.transformExprMode || "literal"
    if (mode === "source") {
      return {
        expr: {
          source: buildSourceSpec({
            nodeId,
            label: t("transform source"),
            source: node.transformSource,
            ancestors,
            allowLoopSources: options.allowLoopSources
          }),
          required: Boolean(node.transformSourceRequired)
        },
        _ui: ui
      }
    }
    if (mode === "op") {
      const op = String(node.transformOp ?? "").trim().toLowerCase()
      if (!FLOW_TRANSFORM_OPS.has(op)) {
        throw new Error(t("Node {nodeId} transform op is invalid.", { nodeId }))
      }
      const args = parseTemplateValue(
        node.transformArgsJson,
        t("Node {nodeId} transform args must be valid JSON.", { nodeId }),
        "[]"
      )
      if (!Array.isArray(args)) {
        throw new Error(t("Node {nodeId} transform args must be a JSON array.", { nodeId }))
      }
      return {
        expr: {
          op,
          args
        },
        _ui: ui
      }
    }
    if (mode === "object") {
      const objectExpr = parseTemplateValue(
        node.transformObjectJson,
        t("Node {nodeId} transform object must be valid JSON.", { nodeId }),
        "{}"
      )
      if (!isJSONObject(objectExpr)) {
        throw new Error(t("Node {nodeId} transform object must be a JSON object.", { nodeId }))
      }
      return {
        expr: {
          object: objectExpr
        },
        _ui: ui
      }
    }
    if (mode === "array") {
      const arrayExpr = parseTemplateValue(
        node.transformArrayJson,
        t("Node {nodeId} transform array must be valid JSON.", { nodeId }),
        "[]"
      )
      if (!Array.isArray(arrayExpr)) {
        throw new Error(t("Node {nodeId} transform array must be a JSON array.", { nodeId }))
      }
      return {
        expr: {
          array: arrayExpr
        },
        _ui: ui
      }
    }
    return {
      expr: {
        literal: parseTemplateValue(
          node.transformLiteralJson,
          t("Node {nodeId} transform literal must be valid JSON.", { nodeId }),
          "null"
        )
      },
      _ui: ui
    }
  }
  if (node.kind === "branch") {
    const seen = new Set<string>()
    const cases = node.branchCases.map((item, index) => {
      const name = item.name.trim()
      if (!name) {
        throw new Error(t("Node {nodeId} branch case {index} requires a name.", { nodeId, index: index + 1 }))
      }
      if (seen.has(name)) {
        throw new Error(t("Node {nodeId} branch case name {name} is duplicated.", { nodeId, name }))
      }
      seen.add(name)
      const op = FLOW_BRANCH_MATCH_OP_SET.has(item.op) ? item.op : ""
      if (!op) {
        throw new Error(t("Node {nodeId} branch case {name} match op is invalid.", { nodeId, name }))
      }
      return {
        name,
        match: {
          source: buildSourceSpec({
            nodeId,
            label: t("branch case {name}", { name }),
            source: item.source,
            ancestors,
            allowLoopSources: options.allowLoopSources
          }),
          op,
          ...(op === "exists"
            ? {}
            : {
                value: parseTemplateValue(
                  item.valueJson,
                  t("Node {nodeId} branch case {name} value must be valid JSON.", { nodeId, name }),
                  "null"
                )
              })
        }
      }
    })
    const defaultCase = node.branchDefaultCase.trim()
    if (defaultCase && !seen.has(defaultCase)) {
      throw new Error(t("Node {nodeId} default case must match an existing branch case.", { nodeId }))
    }
    return {
      cases,
      ...(defaultCase ? { default_case: defaultCase } : {}),
      _ui: ui
    }
  }
  if (node.kind === "foreach") {
    const body = parseTemplateValue(
      node.foreachBodyJson,
      t("Node {nodeId} foreach body must be valid JSON.", { nodeId }),
      JSON.stringify(defaultForeachBodyGraph())
    )
    if (!isJSONObject(body)) {
      throw new Error(t("Node {nodeId} foreach body must be a JSON object.", { nodeId }))
    }
    const resultNodeId = node.foreachResultNodeId.trim()
    if (!resultNodeId) {
      throw new Error(t("Node {nodeId} foreach result node ID is required.", { nodeId }))
    }
    const strictBody = buildStrictForeachBodyGraph(body, nodeId, options.currentFlowId)
    if (!strictBody.nodes.some((bodyNode) => String(bodyNode.id ?? "").trim() === resultNodeId)) {
      throw new Error(t("Node {nodeId} foreach result node ID must exist in the body graph.", { nodeId }))
    }
    return {
      source: buildSourceSpec({
        nodeId,
        label: t("foreach source"),
        source: node.foreachSource,
        ancestors,
        allowLoopSources: options.allowLoopSources
      }),
      required: Boolean(node.foreachRequired),
      body: strictBody,
      result_node_id: resultNodeId,
      _ui: ui
    }
  }
  if (node.kind === "subflow") {
    const inputs = buildInputBindings(node, ancestors, options)
    const flowId = assertValidSubflowFlowID(node.subflowId.trim(), nodeId, options.currentFlowId)
    const inputTemplate = parseTemplateValue(
      node.subflowInputTemplate,
      t("Node {nodeId} subflow input template must be valid JSON.", { nodeId }),
      "{}"
    )
    if (!isJSONObject(inputTemplate)) {
      throw new Error(t("Node {nodeId} subflow input template must be a JSON object.", { nodeId }))
    }
    return {
      flow_id: flowId,
      input_template: inputTemplate,
      ...(inputs.length ? { inputs } : {}),
      ...(node.subflowResultNodeId.trim() ? { result_node_id: node.subflowResultNodeId.trim() } : {}),
      _ui: ui
    }
  }

  const method = node.method.trim()
  if (!method) {
    throw new Error(t("Node {nodeId} requires a method.", { nodeId }))
  }
  const target = Number(node.target || 0)
  if (!Number.isFinite(target) || target < 0) {
    throw new Error(t("Node {nodeId} target must be a non-negative number.", { nodeId }))
  }
  const inputs = buildInputBindings(node, ancestors, options)
  return {
    ...(target > 0 ? { target: Math.trunc(target) } : {}),
    method,
    args_template: parseJSONText(
      node.argsTemplate,
      t("Node {nodeId} args template must be valid JSON.", { nodeId })
    ),
    ...(inputs.length ? { inputs } : {}),
    _ui: ui
  }
}

// resolveCurrentExecutorNodeOrZero 优先使用显式 targetId，否则回退到当前 hub 作为执行节点。
const resolveCurrentExecutorNodeOrZero = () => {
  const raw = state.targetId.trim()
  if (raw) {
    const parsed = Number.parseInt(raw, 10)
    if (Number.isFinite(parsed) && parsed > 0) {
      return Math.trunc(parsed)
    }
  }
  return state.hubId > 0 ? Math.trunc(state.hubId) : 0
}

// findCapabilityRouteForNode 按“method + 实际 provider 节点”命中当前缓存里的能力路由。
const findCapabilityRouteForNode = (node: FlowNodeDraft): ExecCapabilityRoute | null => {
  if (node.kind !== "call") {
    return null
  }
  const method = node.method.trim()
  if (!method) {
    return null
  }
  const expectedProvider = node.target > 0 ? Math.trunc(node.target) : resolveCurrentExecutorNodeOrZero()
  if (!expectedProvider) {
    return null
  }
  return (
    state.execCapabilities.find(
      (route) => route.method === method && route.providerNode === expectedProvider
    ) ?? null
  )
}

const routeToSchemaSource = (route: ExecCapabilityRoute | null): CapabilityRouteSchemaSource | null => {
  if (!route) {
    return null
  }
  return {
    method: route.method,
    version: route.version,
    inputSchema: route.inputSchema
  }
}

const resolveNodeVisualSchema = (node: FlowNodeDraft): MethodVisualSchema | null =>
  resolveMethodVisualSchema(node.method, routeToSchemaSource(findCapabilityRouteForNode(node)))

// applySchemaDefaultsToNode 只补 schema 默认值里缺失的字段，不覆盖用户已经填写的参数。
const applySchemaDefaultsToNode = (node: FlowNodeDraft) => {
  if (node.kind !== "call") {
    return
  }
  const schema = resolveNodeVisualSchema(node)
  if (!schema) {
    return
  }
  const argsDoc = tryParseJSONText(node.argsTemplate, {})
  if (!argsDoc || typeof argsDoc !== "object" || Array.isArray(argsDoc)) {
    return
  }
  let nextDoc = argsDoc as Record<string, unknown>
  let changed = false
  for (const field of schema.fields) {
    if (field.defaultValue === undefined) {
      continue
    }
    const current = readValueAtPointer(nextDoc, field.pointer)
    if (current.found) {
      continue
    }
    nextDoc = setLiteralFieldValueInDoc(nextDoc, field.pointer, field.defaultValue)
    changed = true
  }
  if (changed) {
    node.argsTemplate = formatJSONText(nextDoc, {})
  }
}

// reconcileNodeFormStateToSchema 在 method 切换后裁掉新 schema 不再支持的字段和绑定。
const reconcileNodeFormStateToSchema = (node: FlowNodeDraft, schema: MethodVisualSchema) => {
  if (node.kind !== "call") {
    return
  }
  const allowedPointers = new Set(schema.fields.map((field) => field.pointer))
  const argsDoc = tryParseJSONText(node.argsTemplate, {})
  let nextDoc: Record<string, unknown> = {}
  for (const field of schema.fields) {
    const current = readValueAtPointer(argsDoc, field.pointer)
    if (!current.found) {
      continue
    }
    nextDoc = setLiteralFieldValueInDoc(nextDoc, field.pointer, current.value)
  }
  node.argsTemplate = formatJSONText(nextDoc, {})
  node.inputs = node.inputs.filter((binding) => {
    const to = binding.to.trim()
    return !to || allowedPointers.has(to)
  })
}

const parseSpecJsonObject = (node: FlowNodeDraft) => {
  const nodeId = node.id.trim() || t("Unnamed")
  let parsed: any
  try {
    parsed = JSON.parse(node.specJson.trim() || "{}")
  } catch {
    throw new Error(t("Node {nodeId} advanced spec must be valid JSON.", { nodeId }))
  }
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error(t("Node {nodeId} advanced spec must be a JSON object.", { nodeId }))
  }
  return parsed as Record<string, any>
}

const buildAdvancedSpec = (
  node: FlowNodeDraft,
  ancestors: Map<string, Set<string>>,
  options: StrictGraphBuildOptions = {}
) => {
  const nodeId = node.id.trim() || t("Unnamed")
  const parsed = parseSpecJsonObject(node)
  const ui = { x: Math.round(Number(node.x || 0)), y: Math.round(Number(node.y || 0)) }
  const buildParsedInputs = () => {
    const parsedInputs = parseInputBindings(parsed.inputs, { allowLoopSources: options.allowLoopSources })
    return buildInputBindings(
      {
        ...node,
        inputs: parsedInputs.inputs
      },
      ancestors,
      options
    )
  }

  if (node.kind === "compose") {
    const inputs = buildParsedInputs()
    if (!("template" in parsed)) {
      throw new Error(t("Node {nodeId} compose spec requires template.", { nodeId }))
    }
    return {
      ...parsed,
      ...(inputs.length ? { inputs } : {}),
      ...(inputs.length ? {} : { inputs: undefined }),
      _ui: ui
    }
  }
  if (node.kind === "set_var") {
    const inputs = buildParsedInputs()
    const name = assertValidFlowLocalVarName(
      normalizeFlowLocalVarName(parsed.name),
      t("Node {nodeId} set_var spec requires name.", { nodeId }),
      t("Node {nodeId} flow local var name is invalid.", { nodeId })
    )
    const normalizedSpec: Record<string, any> = { ...parsed, name, _ui: ui }
    if (!("template" in normalizedSpec)) {
      normalizedSpec.template = null
    }
    if (inputs.length) {
      normalizedSpec.inputs = inputs
    } else {
      delete normalizedSpec.inputs
    }
    return normalizedSpec
  }
  if (node.kind === "transform") {
    validateRawTransformExpr({
      expr: parsed.expr,
      nodeId,
      label: t("transform expr"),
      ancestors,
      allowLoopSources: options.allowLoopSources
    })
    return {
      ...parsed,
      _ui: ui
    }
  }
  if (node.kind === "branch") {
    validateRawBranchSpec({
      parsed,
      nodeId,
      ancestors,
      allowLoopSources: options.allowLoopSources
    })
    return {
      ...parsed,
      _ui: ui
    }
  }
  if (node.kind === "foreach") {
    validateRawSourceSpec({
      raw: parsed.source,
      nodeId,
      label: t("foreach source"),
      ancestors,
      allowLoopSources: options.allowLoopSources
    })
    if (!isJSONObject(parsed.body)) {
      throw new Error(t("Node {nodeId} foreach body must be a JSON object.", { nodeId }))
    }
    const strictBody = buildStrictForeachBodyGraph(parsed.body as Record<string, any>, nodeId, options.currentFlowId)
    const resultNodeId = String(parsed.result_node_id ?? "").trim()
    if (!resultNodeId) {
      throw new Error(t("Node {nodeId} foreach result node ID is required.", { nodeId }))
    }
    if (!strictBody.nodes.some((bodyNode) => String(bodyNode.id ?? "").trim() === resultNodeId)) {
      throw new Error(t("Node {nodeId} foreach result node ID must exist in the body graph.", { nodeId }))
    }
    return {
      ...parsed,
      body: strictBody,
      _ui: ui
    }
  }
  if (node.kind === "subflow") {
    const inputs = buildParsedInputs()
    assertValidSubflowFlowID(String(parsed.flow_id ?? "").trim(), nodeId, options.currentFlowId)
    if ("input_template" in parsed && !isJSONObject(parsed.input_template)) {
      throw new Error(t("Node {nodeId} subflow input template must be a JSON object.", { nodeId }))
    }
    const normalizedSpec: Record<string, any> = { ...parsed, _ui: ui }
    if (inputs.length) {
      normalizedSpec.inputs = inputs
    } else {
      delete normalizedSpec.inputs
    }
    return normalizedSpec
  }

  const method = String(parsed.method ?? "").trim()
  if (!method) {
    throw new Error(t("Node {nodeId} requires a method.", { nodeId }))
  }
  const target = Number(parsed.target ?? 0)
  if (!Number.isFinite(target) || target < 0) {
    throw new Error(t("Node {nodeId} target must be a non-negative number.", { nodeId }))
  }
  const inputs = buildParsedInputs()
  const normalizedSpec: Record<string, any> = { ...parsed, method, _ui: ui }
  if (target > 0) {
    normalizedSpec.target = Math.trunc(target)
  } else {
    delete normalizedSpec.target
  }
  if (!("args_template" in normalizedSpec) && "args" in normalizedSpec) {
    normalizedSpec.args_template = normalizedSpec.args
    delete normalizedSpec.args
  }
  if (!("args_template" in normalizedSpec)) {
    normalizedSpec.args_template = {}
  }
  if (inputs.length) {
    normalizedSpec.inputs = inputs
  } else {
    delete normalizedSpec.inputs
  }
  return normalizedSpec
}

const buildSpec = (
  node: FlowNodeDraft,
  ancestors: Map<string, Set<string>>,
  options: StrictGraphBuildOptions = {}
) =>
  node.specEditorMode === "json" || !supportsFormMode(node.kind)
    ? buildAdvancedSpec(node, ancestors, options)
    : buildFormSpec(node, ancestors, options)

const listAncestorNodeIds = (nodeId: string) => {
  const trimmed = nodeId.trim()
  if (!trimmed) return []
  try {
    const ids = collectNodeIds()
    const topology = buildTopology(ids, state.edges)
    const ancestors = buildAncestorMap(topology.order, topology.parents)
    const allowed = ancestors.get(trimmed) ?? new Set<string>()
    return state.nodes.map((node) => node.id.trim()).filter((id) => allowed.has(id))
  } catch {
    return []
  }
}

const getNodeValidation = (nodeId: string) => {
  const node = state.nodes.find((item) => item.id.trim() === nodeId.trim())
  if (!node) return []
  try {
    const ids = collectNodeIds()
    const topology = buildTopology(ids, state.edges)
    const ancestors = buildAncestorMap(topology.order, topology.parents)
    buildSpec(node, ancestors, { currentFlowId: state.flowId.trim() })
    return []
  } catch (err) {
    return [String((err as Error)?.message ?? err ?? t("Unknown validation error."))]
  }
}

const getNodeVisualForm = (nodeId: string): NodeVisualFormModel => {
  const node = state.nodes.find((item) => item.id.trim() === nodeId.trim())
  if (!node) {
    return {
      schema: null,
      compatibility: { supported: false, reasons: [t("Node does not exist.")] },
      fields: []
    }
  }
  const schema = node.kind === "call" ? resolveNodeVisualSchema(node) : null
  return buildNodeVisualFormModel({
    kind: node.kind,
    method: node.method,
    argsTemplate: node.argsTemplate,
    inputs: node.inputs,
    schema
  })
}

const setNodeFieldLiteralValue = (nodeId: string, pointer: string, value: unknown) => {
  const node = state.nodes.find((item) => item.id.trim() === nodeId.trim())
  if (!node || node.kind !== "call") {
    throw new Error(t("Select a call node first."))
  }
  validateJSONPointer(pointer, t("Field pointer must be a valid JSON Pointer."))
  const argsDoc = parseArgsTemplateObject(node)
  const nextDoc = setLiteralFieldValueInDoc(argsDoc, pointer, value)
  node.argsTemplate = formatJSONText(nextDoc, {})
  commitHistory()
}

const clearNodeFieldBinding = (nodeId: string, pointer: string) => {
  const node = state.nodes.find((item) => item.id.trim() === nodeId.trim())
  if (!node || node.kind !== "call") {
    throw new Error(t("Select a call node first."))
  }
  validateJSONPointer(pointer, t("Field pointer must be a valid JSON Pointer."))
  node.inputs = clearBindingForPointer(node.inputs, pointer).map((binding) => ({
    ...binding,
    sourceKind: binding.sourceKind as FlowBindingSourceKind
  }))
  commitHistory()
}

const setNodeFieldBinding = (nodeId: string, pointer: string, source: VisualBindingSource) => {
  const node = state.nodes.find((item) => item.id.trim() === nodeId.trim())
  if (!node || node.kind !== "call") {
    throw new Error(t("Select a call node first."))
  }
  validateJSONPointer(pointer, t("Field pointer must be a valid JSON Pointer."))

  if (source.kind === "node_result") {
    const sourceNodeId = source.nodeId.trim()
    if (!sourceNodeId) {
      throw new Error(t("Source node is required."))
    }
    const allowedAncestors = new Set(listAncestorNodeIds(node.id))
    if (!allowedAncestors.has(sourceNodeId)) {
      throw new Error(t("Source node must be an ancestor."))
    }
    validateJSONPointer(source.path, t("Result path must be a valid JSON Pointer."))
  } else if (source.kind === "trigger") {
    validateJSONPointer(source.path, t("Trigger path must be a valid JSON Pointer."))
  } else if (source.kind === "flow_meta") {
    if (source.field !== "flow_id") {
      throw new Error(t("Flow meta field must be flow_id."))
    }
  } else if (source.kind === "run_meta") {
    if (source.field !== "run_id") {
      throw new Error(t("Run meta field must be run_id."))
    }
  } else if (source.kind === "loop_item" || source.kind === "loop_index") {
    throw new Error(t("Loop sources are only available inside foreach body graphs."))
  } else if (source.kind === "flow_var") {
    const name = assertValidFlowLocalVarName(
      normalizeFlowLocalVarName(source.name),
      t("Flow local var name is required."),
      t("Flow local var name is invalid.")
    )
    if (name !== source.name) {
      source = {
        ...source,
        name
      }
    }
    validateJSONPointer(source.path, t("Flow local var path must be a valid JSON Pointer."))
  }

  node.inputs = setBindingForPointer(node.inputs, pointer, source).map((binding) => ({
    ...binding,
    sourceKind: binding.sourceKind as FlowBindingSourceKind
  }))
  commitHistory()
}

const createInputBinding = () => defaultInputBinding()

const setNodeKind = (nodeId: string, kind: FlowNodeKind) => {
  const node = state.nodes.find((item) => item.id.trim() === nodeId.trim())
  if (!node) {
    throw new Error(t("Node does not exist."))
  }
  if (node.kind === kind) {
    return false
  }
  if (!supportsFormMode(kind)) {
    node.kind = kind
    node.specEditorMode = "json"
    node.specJson = formatJSONText(kindDefaultSpec(kind), {})
    commitHistory()
    return true
  }
  if (kind === "transform") {
    const defaults = defaultTransformDraft()
    node.transformExprMode = node.transformExprMode || defaults.transformExprMode
    node.transformLiteralJson = node.transformLiteralJson.trim() || defaults.transformLiteralJson
    node.transformSource = cloneSourceDraft(node.transformSource?.sourceKind ? node.transformSource : defaults.transformSource)
    node.transformOp = String(node.transformOp ?? "").trim() || defaults.transformOp
    node.transformArgsJson = node.transformArgsJson.trim() || defaults.transformArgsJson
    node.transformObjectJson = node.transformObjectJson.trim() || defaults.transformObjectJson
    node.transformArrayJson = node.transformArrayJson.trim() || defaults.transformArrayJson
  } else if (kind === "branch") {
    node.branchCases = node.branchCases.map(cloneBranchCaseDraft)
  } else if (kind === "foreach") {
    const defaults = defaultForeachDraft()
    node.foreachSource = cloneSourceDraft(node.foreachSource?.sourceKind ? node.foreachSource : defaults.foreachSource)
    node.foreachRequired = typeof node.foreachRequired === "boolean" ? node.foreachRequired : defaults.foreachRequired
    node.foreachBodyJson = String(node.foreachBodyJson ?? "").trim() || defaults.foreachBodyJson
    node.foreachResultNodeId = String(node.foreachResultNodeId ?? "").trim()
  } else if (kind === "subflow") {
    node.subflowInputTemplate = node.subflowInputTemplate.trim() || "{}"
  } else if (kind === "compose") {
    node.composeTemplate = node.composeTemplate.trim() || preferObjectTemplateText(node.argsTemplate, "{}")
    node.method = ""
    node.target = 0
  } else if (kind === "set_var") {
    node.composeTemplate = node.composeTemplate.trim() || node.argsTemplate.trim() || "null"
    node.method = ""
    node.target = 0
  } else {
    node.argsTemplate = node.argsTemplate.trim() || preferObjectTemplateText(node.composeTemplate, "{}")
  }
  node.kind = kind
  node.specEditorMode = defaultSpecEditorMode(kind)
  node.specJson = formatJSONText(buildLooseSpecFromNode(node), {})
  commitHistory()
  return true
}

const setNodeSpecEditorMode = (nodeId: string, mode: FlowSpecEditorMode) => {
  const node = state.nodes.find((item) => item.id.trim() === nodeId.trim())
  if (!node) {
    throw new Error(t("Node does not exist."))
  }
  if (node.specEditorMode === mode) {
    return false
  }
  if (mode === "json") {
    node.specJson = formatJSONText(buildLooseSpecFromNode(node), {})
    node.specEditorMode = "json"
    commitHistory()
    return true
  }
  if (!supportsFormMode(node.kind)) {
    throw new Error(t("Node kind {kind} only supports Advanced JSON mode right now.", { kind: node.kind }))
  }

  const ids = collectNodeIds()
  const topology = buildTopology(ids, state.edges)
  const ancestors = buildAncestorMap(topology.order, topology.parents)
  const normalizedSpec = buildAdvancedSpec(node, ancestors, { currentFlowId: state.flowId.trim() })
  const next = parseSpecDraft(node.kind, normalizedSpec)
  if (next.specEditorMode !== "form") {
    throw new Error(t("Node kind {kind} advanced spec contains fields that ordinary mode cannot represent yet.", { kind: node.kind }))
  }
  node.method = next.method
  node.target = Number.isFinite(next.target) && next.target > 0 ? Math.trunc(next.target) : 0
  node.argsTemplate = next.argsTemplate
  node.composeTemplate = next.composeTemplate
  node.setVarName = next.setVarName
  node.inputs = next.inputs.map(cloneBindingDraft)
  node.transformExprMode = next.transformExprMode
  node.transformLiteralJson = next.transformLiteralJson
  node.transformSource = cloneSourceDraft(next.transformSource)
  node.transformSourceRequired = next.transformSourceRequired
  node.transformOp = next.transformOp
  node.transformArgsJson = next.transformArgsJson
  node.transformObjectJson = next.transformObjectJson
  node.transformArrayJson = next.transformArrayJson
  node.branchCases = next.branchCases.map(cloneBranchCaseDraft)
  node.branchDefaultCase = next.branchDefaultCase
  node.foreachSource = cloneSourceDraft(next.foreachSource)
  node.foreachRequired = next.foreachRequired
  node.foreachBodyJson = next.foreachBodyJson
  node.foreachResultNodeId = next.foreachResultNodeId
  node.subflowId = next.subflowId
  node.subflowInputTemplate = next.subflowInputTemplate
  node.subflowResultNodeId = next.subflowResultNodeId
  node.specJson = formatJSONText(normalizedSpec, {})
  node.specEditorMode = "form"
  commitHistory()
  return true
}

const buildGraphDraftFromState = (
  nodesDraft: FlowNodeDraft[],
  edgesDraft: FlowEdge[],
  options: StrictGraphBuildOptions = {}
) => {
  if (!nodesDraft.length) {
    throw new Error(t("At least one node is required."))
  }
  const ids = collectNodeIdsFromNodes(nodesDraft)
  const edges = edgesDraft.map((edge) => ({
    from: edge.from.trim(),
    to: edge.to.trim(),
    ...(edge.case?.trim() ? { case: edge.case.trim() } : {})
  }))
  const topology = buildTopology(ids, edges)
  const ancestors = buildAncestorMap(topology.order, topology.parents)

  const nodes = nodesDraft.map((node) => {
    const id = node.id.trim()
    const retry = requireNonNegativeInteger(node.retry, t("Node {nodeId} retry must be a non-negative number.", { nodeId: id || t("Unnamed") }))
    const retryBackoffMs = requireNonNegativeInteger(
      node.retryBackoffMs,
      t("Node {nodeId} retry backoff must be a non-negative number.", { nodeId: id || t("Unnamed") })
    )
    const timeoutMs = requireNonNegativeInteger(
      node.timeoutMs,
      t("Node {nodeId} timeout must be a non-negative number.", { nodeId: id || t("Unnamed") })
    )
    return {
      id,
      kind: node.kind,
      allow_fail: Boolean(node.allowFail),
      retry,
      retry_backoff_ms: retryBackoffMs,
      timeout_ms: timeoutMs,
      spec: buildSpec(node, ancestors, options)
    }
  })

  return { nodes, edges }
}

const buildGraph = (options: StrictGraphBuildOptions = {}) =>
  buildGraphDraftFromState(state.nodes, state.edges, options)

const buildTrigger = () => {
  const triggerType = state.triggerType
  const dedupWindowMs = requireNonNegativeInteger(
    state.dedupWindowMs,
    t("Trigger dedup window must be a non-negative number.")
  )
  if (triggerType === "cron") {
    const cron = state.cronExpr.trim()
    if (!cron) {
      throw new Error(t("Cron trigger requires an expression."))
    }
    if (dedupWindowMs > 0) {
      throw new Error(t("Cron trigger does not support dedup window."))
    }
    return {
      type: "cron",
      cron
    }
  }
  if (triggerType === "event") {
    const eventName = state.eventName.trim()
    const eventTopic = state.eventTopic.trim()
    if (!eventName && !eventTopic) {
      throw new Error(t("Event trigger requires event name or event topic."))
    }
    return {
      type: "event",
      event_mode: state.eventMode,
      event_name: eventName || undefined,
      event_topic: eventTopic || undefined,
      ...(dedupWindowMs > 0 ? { dedup_window_ms: dedupWindowMs } : {})
    }
  }
  if (triggerType === "var_changed") {
    const owner = requireNonNegativeInteger(state.varOwner || 0, t("Var owner must be a non-negative number."))
    const varName = state.varName.trim()
    return {
      type: "var_changed",
      var_owner: owner > 0 ? Math.trunc(owner) : undefined,
      var_name: varName || undefined,
      ...(dedupWindowMs > 0 ? { dedup_window_ms: dedupWindowMs } : {})
    }
  }
  if (dedupWindowMs > 0) {
    throw new Error(t("Interval trigger does not support dedup window."))
  }
  const everyMs = Number(state.everyMs)
  if (!everyMs || everyMs <= 0) {
    throw new Error(t("EveryMs must be a positive number."))
  }
  return { type: "interval", every_ms: everyMs }
}

// listFlows 拉取当前执行节点上的 flow 摘要列表，供项目切换和刷新列表使用。
const listFlows = async () => {
  const { sourceID, hubID } = ensureIdentity()
  const executorNode = resolveTargetNode()
  const req = { req_id: newReqId(), origin_node: sourceID, executor_node: executorNode }
  const resp = await callFlow<any>("ListSimple", sourceID, hubID, req)
  handleListResp(resp)
}

// getFlow 按 flow_id 读取完整定义，并交给响应处理逻辑刷新当前草稿。
const getFlow = async (flowId: string) => {
  const { sourceID, hubID } = ensureIdentity()
  const executorNode = resolveTargetNode()
  const trimmed = flowId.trim()
  if (!trimmed) {
    throw new Error(t("Flow ID is required."))
  }
  const req = {
    req_id: newReqId(),
    origin_node: sourceID,
    executor_node: executorNode,
    flow_id: trimmed
  }
  const resp = await callFlow<any>("GetSimple", sourceID, hubID, req)
  handleGetResp(resp)
}

// saveFlow 把当前编辑态导出成严格 payload，再通过 set 接口一次性保存到后端。
const saveFlow = async () => {
  const { sourceID, hubID } = ensureIdentity()
  const executorNode = resolveTargetNode()
  const payload = exportPayload()
  const req = {
    req_id: newReqId(),
    origin_node: sourceID,
    executor_node: executorNode,
    ...payload
  }
  const resp = await callFlow<any>("SetSimple", sourceID, hubID, req)
  handleSetResp(resp)
}

// runFlow 触发当前 flow 执行，后续的 run_id、状态与历史刷新都交给 handleRunResp 串起。
const runFlow = async () => {
  const { sourceID, hubID } = ensureIdentity()
  const executorNode = resolveTargetNode()
  const flowId = state.flowId.trim()
  if (!flowId) {
    throw new Error(t("Flow ID is required."))
  }
  const req = {
    req_id: newReqId(),
    origin_node: sourceID,
    executor_node: executorNode,
    flow_id: flowId
  }
  const resp = await callFlow<any>("RunSimple", sourceID, hubID, req)
  await handleRunResp(resp)
}

// listRunsFlow 读取当前 flow 的运行历史，并在本地维护“当前 run_id”的默认选择。
const listRunsFlow = async (limit?: number) => {
  const { sourceID, hubID } = ensureIdentity()
  const executorNode = resolveTargetNode()
  const flowId = state.flowId.trim()
  if (!flowId) {
    throw new Error(t("Flow ID is required."))
  }
  const parsedLimit = Number(limit)
  const req = {
    req_id: newReqId(),
    origin_node: sourceID,
    executor_node: executorNode,
    flow_id: flowId,
    ...(Number.isFinite(parsedLimit) && parsedLimit > 0 ? { limit: Math.trunc(parsedLimit) } : {})
  }
  state.runHistoryLoading = true
  try {
    const resp = await callFlow<any>("ListRunsSimple", sourceID, hubID, req)
    handleListRunsResp(resp)
  } finally {
    state.runHistoryLoading = false
  }
}

// statusFlow 查询指定或最近一次运行的状态摘要，供画布 badge 和详情面板复用。
const statusFlow = async (runId?: string) => {
  const { sourceID, hubID } = ensureIdentity()
  const executorNode = resolveTargetNode()
  const flowId = state.flowId.trim()
  if (!flowId) {
    throw new Error(t("Flow ID is required."))
  }
  const req = {
    req_id: newReqId(),
    origin_node: sourceID,
    executor_node: executorNode,
    flow_id: flowId,
    run_id: runId?.trim() || undefined
  }
  const resp = await callFlow<any>("StatusSimple", sourceID, hubID, req)
  handleStatusResp(resp)
}

// cancelRunFlow 对当前或指定 run 发取消请求，并在成功后主动刷新历史与状态。
const cancelRunFlow = async (runId?: string) => {
  const { sourceID, hubID } = ensureIdentity()
  const executorNode = resolveTargetNode()
  const flowId = state.flowId.trim()
  if (!flowId) {
    throw new Error(t("Flow ID is required."))
  }
  const resolvedRunID = String(runId ?? state.statusRunId ?? "").trim()
  if (!resolvedRunID) {
    throw new Error(t("Run ID is required."))
  }
  const req = {
    req_id: newReqId(),
    origin_node: sourceID,
    executor_node: executorNode,
    flow_id: flowId,
    run_id: resolvedRunID
  }
  const resp = await callFlow<any>("CancelRunSimple", sourceID, hubID, req)
  await handleCancelRunResp(resp)
}

// loadNodeDetail 在真正发请求前先规范 flow/node/path 输入，并把详情面板状态重置到可恢复形态。
const loadNodeDetail = async (nodeId: string, runId?: string, path?: string) => {
  const { sourceID, hubID } = ensureIdentity()
  const executorNode = resolveTargetNode()
  const flowId = state.flowId.trim()
  const requestedNodeId = nodeId.trim()
  const requestedRunId = String(runId ?? state.nodeDetail.requestedRunId ?? "").trim()
  const requestedPath = String(path ?? state.nodeDetail.requestedPath ?? "").trim()

  if (!flowId) {
    state.nodeDetail = {
      ...createNodeDetailState(requestedNodeId, requestedRunId),
      requestedPath,
      error: t("Flow ID is required.")
    }
    return false
  }
  if (!requestedNodeId) {
    state.nodeDetail = {
      ...createNodeDetailState("", requestedRunId),
      requestedPath,
      error: t("Node ID is required.")
    }
    return false
  }

  try {
    validateJSONPointer(requestedPath, t("Result path must be a valid JSON Pointer."))
  } catch (err) {
    state.nodeDetail = {
      ...createNodeDetailState(requestedNodeId, requestedRunId),
      requestedPath,
      error: String((err as Error)?.message ?? err ?? t("Result path must be a valid JSON Pointer."))
    }
    return false
  }

  state.nodeDetail = {
    ...createNodeDetailState(requestedNodeId, requestedRunId),
    requestedPath,
    loading: true
  }

  try {
    const req = {
      req_id: newReqId(),
      origin_node: sourceID,
      executor_node: executorNode,
      flow_id: flowId,
      run_id: requestedRunId || undefined,
      node_id: requestedNodeId,
      path: requestedPath || undefined
    }
    const resp = await callFlow<any>("DetailSimple", sourceID, hubID, req)
    handleDetailResp(resp, requestedNodeId, requestedRunId, requestedPath)
    return true
  } catch (err) {
    state.nodeDetail = {
      ...createNodeDetailState(requestedNodeId, requestedRunId),
      requestedPath,
      error: String((err as Error)?.message ?? err ?? t("Failed to load node detail."))
    }
    return false
  }
}

const handleListResp = (data: any) => {
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    setMessage(msg || t("Flow list failed."), "error")
    return
  }
  const flows = Array.isArray(data?.flows) ? data.flows : []
  state.flows = flows.map(mapSummary)
  setMessage(t("Flow list updated."), "success")
}

const handleGetResp = (data: any) => {
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    setMessage(msg || t("Flow load failed."), "error")
    return
  }
  applyFlowPayload(data, t("Flow loaded."), true)
}

const applyGraphDraft = (graphSource: any, successMessage: string) => {
  const snapshot = createGraphEditorStateFromDraft(graphSource)
  state.nodes = snapshot.nodes
  state.edges = snapshot.edges
  state.selectedNodeIndex = -1
  state.selectedEdgeIndex = -1
  resetStatusState()
  state.nodeDetail = createNodeDetailState()
  resetExecCapabilityState()
  setMessage(successMessage, "success")
  resetHistory()
}

const applyFlowPayload = (data: any, successMessage: string, refreshStatus: boolean) => {
  state.flowId = String(data?.flow_id ?? data?.flowId ?? "").trim()
  state.flowName = String(data?.name ?? "").trim()
  const rawMaxActiveRuns = hasOwn(data, "max_active_runs")
    ? data?.max_active_runs
    : hasOwn(data, "maxActiveRuns")
      ? data?.maxActiveRuns
      : undefined
  state.maxActiveRuns =
    rawMaxActiveRuns === undefined || rawMaxActiveRuns === null || String(rawMaxActiveRuns).trim() === ""
      ? null
      : Number.isFinite(Number(rawMaxActiveRuns)) && Number(rawMaxActiveRuns) >= 0
        ? Math.trunc(Number(rawMaxActiveRuns))
        : null
  const trigger = data?.trigger ?? {}
  const triggerType = String(trigger?.type ?? trigger?.triggerType ?? "interval").trim().toLowerCase()
  if (triggerType === "cron" || triggerType === "event" || triggerType === "var_changed") {
    state.triggerType = triggerType
  } else {
    state.triggerType = "interval"
  }
  const everyMs = Number(trigger?.every_ms ?? trigger?.everyMs ?? 0)
  state.everyMs = everyMs > 0 ? everyMs : 60000
  state.cronExpr = String(trigger?.cron ?? "").trim()
  const rawEventMode = String(trigger?.event_mode ?? trigger?.eventMode ?? "publish").trim().toLowerCase()
  if (rawEventMode === "received" || rawEventMode === "any") {
    state.eventMode = rawEventMode
  } else {
    state.eventMode = "publish"
  }
  state.eventName = String(trigger?.event_name ?? trigger?.eventName ?? "").trim()
  state.eventTopic = String(trigger?.event_topic ?? trigger?.eventTopic ?? "").trim()
  const dedupWindowMs = Number(trigger?.dedup_window_ms ?? trigger?.dedupWindowMs ?? 0)
  state.dedupWindowMs = Number.isFinite(dedupWindowMs) && dedupWindowMs >= 0 ? Math.trunc(dedupWindowMs) : 0
  const varOwner = Number(trigger?.var_owner ?? trigger?.varOwner ?? 0)
  state.varOwner = Number.isFinite(varOwner) && varOwner > 0 ? Math.trunc(varOwner) : 0
  state.varName = String(trigger?.var_name ?? trigger?.varName ?? "").trim()
  applyGraphDraft(data?.graph ?? {}, successMessage)
  if (refreshStatus && state.selfNodeId && state.hubId) {
    void statusFlow("").catch(() => {})
  }
}

const loadFromPayload = (data: any) => {
  const flowID = String(data?.flow_id ?? data?.flowId ?? "").trim()
  if (!flowID) {
    throw new Error(t("Flow ID is required."))
  }
  applyFlowPayload(
    {
      flow_id: flowID,
      name: String(data?.name ?? "").trim(),
      max_active_runs: data?.max_active_runs ?? data?.maxActiveRuns,
      trigger: data?.trigger ?? {},
      graph: data?.graph ?? {}
    },
    t("Draft loaded."),
    false
  )
}

const loadGraphDraft = (graph: any) => {
  applyGraphDraft(graph ?? {}, t("Draft loaded."))
}

const exportPayload = (): FlowPayload => {
  const flowID = state.flowId.trim()
  if (!flowID) {
    throw new Error(t("Flow ID is required."))
  }
  const maxActiveRuns =
    state.maxActiveRuns === null
      ? null
      : requireNonNegativeInteger(state.maxActiveRuns, t("Max active runs must be a non-negative number."))
  return {
    flow_id: flowID,
    name: state.flowName.trim(),
    ...(maxActiveRuns === null ? {} : { max_active_runs: maxActiveRuns }),
    trigger: buildTrigger(),
    graph: buildGraph({ currentFlowId: flowID })
  }
}

const exportGraphDraft = (): FlowGraphDraft => buildGraph({ currentFlowId: state.flowId.trim() })

const fetchExecCapabilityRoutes = async (executorNode: number, methodFilter?: string) => {
  const { sourceID } = ensureIdentity()
  const method = String(methodFilter ?? "").trim()
  const req = {
    req_id: newReqId(),
    requester_node: sourceID,
    method: method || undefined,
    prefix: method.length > 0,
    limit: 200,
    include_schema: true
  }
  const data = await callFlow<any>("ExecCapQuerySimple", sourceID, executorNode, req)
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    throw new Error(msg || t("Capability query failed."))
  }
  const routes: any[] = Array.isArray(data?.routes) ? data.routes : []
  return routes
    .map(mapExecCapabilityRoute)
    .filter((route: ExecCapabilityRoute) => route.providerNode > 0 && route.method.length > 0)
}

// queryExecCapabilities 负责整批刷新 method 能力列表，并用 epoch/cacheVersion 防止旧响应回写。
const queryExecCapabilities = async (methodFilter?: string, queryNodeId?: string | number) => {
  const executorNode = resolveCapabilityQueryNode(queryNodeId)
  const cacheVersion = execCapabilityCacheVersion
  const loadEpoch = beginExecCapabilityLoad()
  try {
    const routes = await fetchExecCapabilityRoutes(executorNode, methodFilter)
    if (cacheVersion !== execCapabilityCacheVersion) {
      return
    }
    replaceExecCapabilityRoutes(routes)
    setMessage(
      routes.length
        ? t("Capability list updated from node {nodeId} ({count}).", {
            nodeId: executorNode,
            count: routes.length
          })
        : t("No capability matched on node {nodeId}.", { nodeId: executorNode }),
      routes.length ? "success" : "info"
    )
  } catch (err) {
    if (cacheVersion !== execCapabilityCacheVersion) {
      return
    }
    replaceExecCapabilityRoutes([])
    setMessage(String((err as Error)?.message ?? err ?? t("Capability query failed.")), "error")
  } finally {
    endExecCapabilityLoad(loadEpoch)
  }
}

// ensureCapabilityRouteLoaded 对单个 method/provider 做懒加载，并合并并发的重复查询。
const ensureCapabilityRouteLoaded = async (method: string, providerNode: number) => {
  const normalizedMethod = String(method ?? "").trim()
  const normalizedProvider =
    Number.isFinite(providerNode) && providerNode > 0 ? Math.trunc(providerNode) : 0
  if (!normalizedMethod || !normalizedProvider || hasExecCapabilityRoute(normalizedMethod, normalizedProvider)) {
    return false
  }

  const hydrationKey = `${normalizedProvider}|${normalizedMethod}`
  const pending = pendingCapabilityHydrations.get(hydrationKey)
  if (pending) {
    return pending
  }

  const task = (async () => {
    const cacheVersion = execCapabilityCacheVersion
    const loadEpoch = beginExecCapabilityLoad()
    try {
      const routes = await fetchExecCapabilityRoutes(normalizedProvider, normalizedMethod)
      if (cacheVersion !== execCapabilityCacheVersion) {
        return false
      }
      const matched = routes.filter(
        (route) => route.providerNode === normalizedProvider && route.method === normalizedMethod
      )
      if (!matched.length) {
        return false
      }
      mergeExecCapabilityRoutes(matched)
      return true
    } finally {
      endExecCapabilityLoad(loadEpoch)
    }
  })()

  pendingCapabilityHydrations.set(hydrationKey, task)
  try {
    return await task
  } finally {
    pendingCapabilityHydrations.delete(hydrationKey)
  }
}

const ensureNodeCapabilityLoaded = async (nodeId: string) => {
  const trimmedNodeId = String(nodeId ?? "").trim()
  if (!trimmedNodeId) {
    return false
  }
  const node = state.nodes.find((item) => item.id.trim() === trimmedNodeId)
  if (!node || node.kind !== "call") {
    return false
  }
  const method = node.method.trim()
  if (!method) {
    return false
  }
  const providerNode = node.target > 0 ? Math.trunc(node.target) : resolveCurrentExecutorNodeOrZero()
  if (!providerNode || hasExecCapabilityRoute(method, providerNode)) {
    return false
  }
  return ensureCapabilityRouteLoaded(method, providerNode)
}

// applyCallCapability 把选中的 capability 映射回 call 节点，并同步修正 form/json 两套 spec 表达。
const applyCallCapability = (key: string) => {
  const selected = state.nodes[state.selectedNodeIndex]
  if (!selected || selected.kind !== "call") {
    throw new Error(t("Select a call node first."))
  }
  const route = state.execCapabilities.find((item) => item.key === String(key).trim())
  if (!route) {
    throw new Error(t("Capability not found in current list."))
  }
  const methodChanged = selected.method !== route.method
  const nextSchema = resolveMethodVisualSchema(route.method, routeToSchemaSource(route))
  selected.method = route.method
  selected.target = normalizeCallTarget(route.providerNode)
  if (!String(selected.argsTemplate ?? "").trim()) {
    selected.argsTemplate = "{}"
  }
  if (methodChanged && selected.specEditorMode === "form" && nextSchema) {
    reconcileNodeFormStateToSchema(selected, nextSchema)
  }
  applySchemaDefaultsToNode(selected)
  if (selected.specEditorMode === "json") {
    try {
      const parsed = parseSpecJsonObject(selected)
      parsed.method = route.method
      if (selected.target > 0) {
        parsed.target = selected.target
      } else {
        delete parsed.target
      }
      if (!("args_template" in parsed) && !("args" in parsed)) {
        parsed.args_template = tryParseJSONText(selected.argsTemplate, {})
      }
      selected.specJson = formatJSONText(parsed, {})
    } catch {
      selected.specJson = formatJSONText(buildLooseSpecFromNode(selected), {})
    }
  }
  commitHistory()
  const targetLabel = selected.target > 0 ? t("Node {nodeId}", { nodeId: selected.target }) : t("Current executor")
  setMessage(t("Capability applied: {method} @ {targetLabel}.", { method: route.method, targetLabel }), "success")
}

const handleSetResp = (data: any) => {
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    setMessage(msg || t("Flow save failed."), "error")
    return
  }
  setMessage(t("Flow saved."), "success")
  void listFlows().catch(() => {})
}

// handleRunResp 在成功启动后立即串起状态与历史刷新，避免窗口继续停留在旧 run 上。
const handleRunResp = async (data: any) => {
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    setMessage(msg || t("Flow run failed."), "error")
    return
  }
  const runId = String(data?.run_id ?? "")
  state.statusRunId = runId
  setMessage(t("Flow run started."), "success")
  const refreshTasks: Promise<unknown>[] = [statusFlow(runId)]
  if (runId) {
    refreshTasks.unshift(listRunsFlow(20))
  }
  await Promise.allSettled(refreshTasks)
}

const handleListRunsResp = (data: any) => {
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    setMessage(msg || t("Flow run history failed."), "error")
    return
  }
  const runs = Array.isArray(data?.runs) ? data.runs : []
  state.runHistory = runs.map(mapRunHistoryItem).filter((item) => item.runId.trim().length > 0)
  if (state.statusRunId.trim()) {
    const hasCurrent = state.runHistory.some((item) => item.runId === state.statusRunId.trim())
    if (!hasCurrent && state.runHistory.length > 0) {
      state.statusRunId = state.runHistory[0].runId
    }
  } else if (state.runHistory.length > 0) {
    state.statusRunId = state.runHistory[0].runId
  }
}

// handleStatusResp 归一化 flow.status 响应，并顺带维护当前 run_id 与节点详情面板的默认 run。
const handleStatusResp = (data: any) => {
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    setMessage(msg || t("Flow status failed."), "error")
    return
  }
  const nodes = Array.isArray(data?.nodes) ? data.nodes : []
  state.lastStatus = {
    status: String(data?.status ?? ""),
    runId: String(data?.run_id ?? ""),
    executorNode: Number(data?.executor_node ?? 0),
    nodes: nodes.map((node: any) => ({
      id: String(node?.id ?? ""),
      status: String(node?.status ?? ""),
      code: Number(node?.code ?? 0),
      msg: String(node?.msg ?? "")
    }))
  }
  if (state.lastStatus.runId) {
    state.statusRunId = state.lastStatus.runId
    if (state.nodeDetail.requestedNodeId && !state.nodeDetail.requestedRunId) {
      state.nodeDetail.requestedRunId = state.lastStatus.runId
    }
  }
  setMessage(t("Status updated."), "success")
}

const handleCancelRunResp = async (data: any) => {
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    setMessage(msg || t("Cancel run failed."), "error")
    return
  }
  const runID = String(data?.run_id ?? "").trim()
  if (runID) {
    state.statusRunId = runID
  }
  setMessage(t("Run cancellation requested."), "success")
  await Promise.allSettled([listRunsFlow(20), statusFlow(runID || state.statusRunId)])
}

// handleDetailResp 统一落盘 detail 查询结果，让画布节点详情和 JSON Pointer 过滤保持同一来源。
const handleDetailResp = (data: any, requestedNodeId: string, requestedRunId: string, requestedPath: string) => {
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    state.nodeDetail = {
      ...createNodeDetailState(requestedNodeId, requestedRunId),
      requestedPath,
      error: msg || t("Flow detail failed.")
    }
    return
  }

  const detailNode = data?.node
  state.nodeDetail = {
    loading: false,
    error: "",
    requestedNodeId,
    requestedRunId,
    requestedPath,
    runId: String(data?.run_id ?? "").trim(),
    path: String(data?.path ?? "").trim(),
    node:
      detailNode && typeof detailNode === "object"
        ? {
            id: String(detailNode?.id ?? requestedNodeId).trim(),
            status: String(detailNode?.status ?? "").trim(),
            code: Number(detailNode?.code ?? 0),
            msg: String(detailNode?.msg ?? "").trim()
          }
        : null,
    resultValue: data?.result,
    resultText: formatStructuredText(data?.result, "")
  }
  if (state.nodeDetail.runId) {
    state.statusRunId = state.nodeDetail.runId
  }
}

const resetNodeDetail = (nodeId = "", runId = "") => {
  state.nodeDetail = createNodeDetailState(nodeId, runId || state.statusRunId)
}

const getNodeOutputSchemaText = (nodeId: string) => {
  const node = state.nodes.find((item) => item.id.trim() === nodeId.trim())
  if (!node || node.kind !== "call") {
    return ""
  }
  const route = findCapabilityRouteForNode(node)
  if (!route || route.outputSchema === undefined || route.outputSchema === null) {
    return ""
  }
  return formatStructuredText(route.outputSchema, "")
}

export const useFlowStore = () => {
  if (!draftHistory.length) {
    resetHistory()
  }

  return {
    state,
    addEdge,
    addNode,
    autoLayoutTB,
    commitHistory,
    clearMessage: () => {
      state.message = ""
      state.messageLevel = ""
    },
    clearSelection: () => {
      state.selectedNodeIndex = -1
      state.selectedEdgeIndex = -1
      state.nodeDetail = createNodeDetailState()
    },
    getFlow,
    listFlows,
    newDraft,
    suggestNodeId,
    renameNodeId,
    removeSelectedEdge,
    removeSelectedNode,
    redo,
    runFlow,
    listRunsFlow,
    cancelRunFlow,
    saveFlow,
    loadFromPayload,
    loadGraphDraft,
    loadGraphEditorState: (snapshot: FlowGraphEditorState) => {
      applyGraphEditorState(snapshot)
    },
    exportPayload,
    exportGraphDraft,
    exportGraphEditorState: () => takeGraphEditorState(),
    graphEditorSignature,
    createInputBinding,
    getNodeVisualForm,
    getNodeValidation,
    listAncestorNodeIds,
    listBindableAncestorNodeIds: listAncestorNodeIds,
    getNodeOutputSchemaText,
    normalizeCallTarget,
    queryExecCapabilities,
    ensureNodeCapabilityLoaded,
    ensureCapabilityRouteLoaded,
    applyCallCapability,
    applyExecCapability: applyCallCapability,
    setFieldLiteralValue: setNodeFieldLiteralValue,
    setFieldBinding: setNodeFieldBinding,
    clearFieldBinding: clearNodeFieldBinding,
    describeFieldBinding: describeVisualFieldBinding,
    loadNodeDetail,
    resetNodeDetail,
    setNodeKind,
    setNodeSpecEditorMode,
    setSelectedEdgeCase: (value: string) => {
      const idx = state.selectedEdgeIndex
      if (idx < 0 || idx >= state.edges.length) {
        throw new Error(t("Edge does not exist."))
      }
      const trimmed = String(value ?? "").trim()
      const edge = state.edges[idx]
      edge.case = trimmed || undefined
      commitHistory()
    },
    undo,
    selectEdgeByEndpoints: (from: string, to: string) => {
      const fromId = from.trim()
      const toId = to.trim()
      const idx = state.edges.findIndex((edge) => edge.from === fromId && edge.to === toId)
      state.selectedEdgeIndex = idx
      state.selectedNodeIndex = -1
      state.nodeDetail = createNodeDetailState()
    },
    selectNodeById: (nodeId: string) => {
      const trimmed = nodeId.trim()
      const idx = state.nodes.findIndex((node) => node.id.trim() === trimmed)
      state.selectedNodeIndex = idx
      state.selectedEdgeIndex = -1
    },
    setIdentity: (nodeId: number, hubId: number) => {
      state.selfNodeId = Number(nodeId || 0)
      state.hubId = Number(hubId || 0)
      if (!state.targetId && state.hubId) {
        state.targetId = String(state.hubId)
      }
    },
    setNodePosition: (nodeId: string, x: number, y: number) => {
      const trimmed = nodeId.trim()
      const node = state.nodes.find((n) => n.id.trim() === trimmed)
      if (!node) return
      if (!Number.isFinite(x) || !Number.isFinite(y)) return
      node.x = x
      node.y = y
    },
    selectEdge: (index: number) => {
      state.selectedEdgeIndex = index
    },
    selectNode: (index: number) => {
      state.selectedNodeIndex = index
    },
    statusFlow
  }
}
