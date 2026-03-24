import { reactive } from "vue"
import { t } from "@/i18n"
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

export type FlowNodeKind = "call" | "compose"
export type FlowSpecEditorMode = "form" | "json"
export type FlowBindingSourceKind = "node_result" | "trigger" | "flow_meta" | "run_meta" | ""

export type FlowInputBindingDraft = {
  to: string
  sourceKind: FlowBindingSourceKind
  nodeId: string
  path: string
  field: string
  required: boolean
}

export type FlowNodeDraft = {
  id: string
  kind: FlowNodeKind
  allowFail: boolean
  retry: number
  timeoutMs: number
  method: string
  target: number
  argsTemplate: string
  composeTemplate: string
  inputs: FlowInputBindingDraft[]
  specEditorMode: FlowSpecEditorMode
  specJson: string
  x: number
  y: number
}

export type FlowEdge = {
  from: string
  to: string
}

export type FlowPayload = {
  flow_id: string
  name: string
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

export type FlowMessageLevel = "" | "success" | "error" | "info"

export const flowStatusLabelKey = (status: string) => {
  const normalized = String(status ?? "").trim().toLowerCase()
  switch (normalized) {
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
  triggerType: "interval" | "event" | "var_changed"
  eventMode: "publish" | "received" | "any"
  everyMs: number
  eventName: string
  eventTopic: string
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
  triggerType: "interval" | "event" | "var_changed"
  eventMode: "publish" | "received" | "any"
  everyMs: number
  eventName: string
  eventTopic: string
  varOwner: number
  varName: string
  nodes: FlowNodeDraft[]
  edges: FlowEdge[]
  selectedNodeIndex: number
  selectedEdgeIndex: number
  statusRunId: string
  lastStatus: FlowStatus
  execCapabilities: ExecCapabilityRoute[]
  execCapabilitiesLoading: boolean
  message: string
  messageLevel: FlowMessageLevel
  historyIndex: number
  historyLength: number
}

const state = reactive<FlowState>({
  targetId: "",
  selfNodeId: 0,
  hubId: 0,
  flows: [],
  flowId: "",
  flowName: "",
  triggerType: "interval",
  eventMode: "publish",
  everyMs: 60000,
  eventName: "",
  eventTopic: "",
  varOwner: 0,
  varName: "",
  nodes: [],
  edges: [],
  selectedNodeIndex: -1,
  selectedEdgeIndex: -1,
  statusRunId: "",
  lastStatus: {
    status: "",
    runId: "",
    executorNode: 0,
    nodes: []
  },
  execCapabilities: [],
  execCapabilitiesLoading: false,
  message: "",
  messageLevel: "",
  historyIndex: 0,
  historyLength: 1
})

const MAX_HISTORY = 120
let draftHistory: FlowDraftSnapshot[] = []
let draftHistoryIndex = 0

const cloneBindingDraft = (binding: FlowInputBindingDraft): FlowInputBindingDraft => ({ ...binding })

const cloneNodeDraft = (node: FlowNodeDraft): FlowNodeDraft => ({
  ...node,
  inputs: node.inputs.map(cloneBindingDraft)
})

const cloneEdge = (edge: FlowEdge): FlowEdge => ({ ...edge })

const setMessage = (message: string, level: Exclude<FlowMessageLevel, ""> = "info") => {
  const trimmed = message.trim()
  state.message = trimmed
  state.messageLevel = trimmed ? level : ""
}

const snapshotToJSON = (snapshot: FlowDraftSnapshot) => JSON.stringify(snapshot)

const takeSnapshot = (): FlowDraftSnapshot => ({
  flowId: state.flowId,
  flowName: state.flowName,
  triggerType: state.triggerType,
  eventMode: state.eventMode,
  everyMs: state.everyMs,
  eventName: state.eventName,
  eventTopic: state.eventTopic,
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
  state.triggerType = snapshot.triggerType
  state.eventMode = snapshot.eventMode
  state.everyMs = snapshot.everyMs
  state.eventName = snapshot.eventName
  state.eventTopic = snapshot.eventTopic
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
  state.execCapabilities = []
  state.execCapabilitiesLoading = false
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

const formatJSONText = (value: any, fallback: any = {}) => {
  const source = value === undefined ? fallback : value
  try {
    return JSON.stringify(source ?? fallback, null, 2)
  } catch {
    return JSON.stringify(fallback, null, 2)
  }
}

const normalizeNodeKind = (raw: any): FlowNodeKind => {
  const normalized = String(raw ?? "").trim().toLowerCase()
  return normalized === "compose" ? "compose" : "call"
}

const normalizeBindingSourceKind = (raw: any): FlowBindingSourceKind => {
  const normalized = String(raw ?? "").trim().toLowerCase()
  switch (normalized) {
    case "node_result":
    case "trigger":
    case "flow_meta":
    case "run_meta":
      return normalized
    default:
      return ""
  }
}

const defaultInputBinding = (): FlowInputBindingDraft => ({
  to: "",
  sourceKind: "node_result",
  nodeId: "",
  path: "",
  field: "",
  required: false
})

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

const parseInputBindings = (raw: any): FlowInputBindingDraft[] => {
  if (!Array.isArray(raw)) {
    return []
  }
  return raw.map((binding) => {
    const source = binding?.source ?? {}
    return {
      to: String(binding?.to ?? "").trim(),
      sourceKind: normalizeBindingSourceKind(source?.kind),
      nodeId: String(source?.node_id ?? "").trim(),
      path: String(source?.path ?? "").trim(),
      field: String(source?.field ?? "").trim(),
      required: Boolean(binding?.required ?? false)
    }
  })
}

const parseSpecDraft = (kind: FlowNodeKind, spec: any) => {
  const parsed = normalizeSpecObject(spec)
  const ui = parsed?._ui ?? {}
  const x = Number(ui?.x)
  const y = Number(ui?.y)
  if (kind === "compose") {
    return {
      method: "",
      target: 0,
      argsTemplate: "{}",
      composeTemplate: formatJSONText(parsed?.template, {}),
      inputs: parseInputBindings(parsed?.inputs),
      specJson: formatJSONText(parsed, {}),
      x: Number.isFinite(x) ? x : undefined,
      y: Number.isFinite(y) ? y : undefined
    }
  }
  const method = String(parsed?.method ?? "")
  const target = Number(parsed?.target ?? 0)
  return {
    method,
    target,
    argsTemplate: formatJSONText(parsed?.args_template ?? parsed?.args, {}),
    composeTemplate: "{}",
    inputs: parseInputBindings(parsed?.inputs),
    specJson: formatJSONText(parsed, {}),
    x: Number.isFinite(x) ? x : undefined,
    y: Number.isFinite(y) ? y : undefined
  }
}

const defaultNodePosition = (index: number) => {
  const col = index % 4
  const row = Math.floor(index / 4)
  return { x: col * 240, y: row * 160 }
}

const createNodeDraft = (id: string, kind: FlowNodeKind, index: number): FlowNodeDraft => {
  const pos = defaultNodePosition(index)
  return {
    id,
    kind,
    allowFail: false,
    retry: 1,
    timeoutMs: 3000,
    method: "",
    target: 0,
    argsTemplate: "{}",
    composeTemplate: "{}",
    inputs: [],
    specEditorMode: "form",
    specJson: formatJSONText(kind === "compose" ? { template: {} } : { method: "", args_template: {} }, {}),
    x: pos.x,
    y: pos.y
  }
}

const mapNode = (input: any, index: number): FlowNodeDraft => {
  const sourceKind = String(input?.kind ?? "").toLowerCase()
  const kind = normalizeNodeKind(sourceKind)
  const { method, target, argsTemplate, composeTemplate, inputs, specJson, x, y } = parseSpecDraft(kind, input?.spec)
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
    timeoutMs: Number(input?.timeout_ms ?? input?.timeoutMs ?? 3000),
    method,
    target: mappedTarget,
    argsTemplate,
    composeTemplate,
    inputs,
    specEditorMode: "form",
    specJson,
    x: Number.isFinite(x) ? Number(x) : pos.x,
    y: Number.isFinite(y) ? Number(y) : pos.y
  }
}

const mapEdge = (input: any): FlowEdge => ({
  from: String(input?.from ?? "").trim(),
  to: String(input?.to ?? "").trim()
})

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

const newDraft = () => {
  state.flowId = ""
  state.flowName = ""
  state.triggerType = "interval"
  state.eventMode = "publish"
  state.everyMs = 60000
  state.eventName = ""
  state.eventTopic = ""
  state.varOwner = 0
  state.varName = ""
  state.nodes = []
  state.edges = []
  state.selectedNodeIndex = -1
  state.selectedEdgeIndex = -1
  state.statusRunId = ""
  state.execCapabilities = []
  state.execCapabilitiesLoading = false
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

const collectNodeIds = () => {
  const seen = new Set<string>()
  const ids: string[] = []
  for (const node of state.nodes) {
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
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error(t("Node {nodeId} args template must be a JSON object.", { nodeId }))
  }
  return parsed as Record<string, unknown>
}

const isBindingBlank = (binding: FlowInputBindingDraft) =>
  !binding.to.trim() &&
  !binding.sourceKind &&
  !binding.nodeId.trim() &&
  !binding.path.trim() &&
  !binding.field.trim() &&
  !binding.required

const buildInputBindings = (node: FlowNodeDraft, ancestors: Map<string, Set<string>>) => {
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
    } else if (sourceKind === "flow_meta" || sourceKind === "run_meta") {
      if (binding.field.trim()) source.field = binding.field.trim()
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

  const target = Number(node.target || 0)
  return {
    ...(Number.isFinite(target) && target > 0 ? { target: Math.trunc(target) } : {}),
    method: node.method.trim(),
    args_template: tryParseJSONText(node.argsTemplate, {}),
    ...(looseInputs.length ? { inputs: looseInputs } : {}),
    _ui: ui
  }
}

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

const buildFormSpec = (node: FlowNodeDraft, ancestors: Map<string, Set<string>>) => {
  const nodeId = node.id.trim() || t("Unnamed")
  const ui = { x: Math.round(Number(node.x || 0)), y: Math.round(Number(node.y || 0)) }
  const inputs = buildInputBindings(node, ancestors)
  if (node.kind === "compose") {
    return {
      template: parseJSONText(
        node.composeTemplate,
        t("Node {nodeId} template must be valid JSON.", { nodeId })
      ),
      ...(inputs.length ? { inputs } : {}),
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

const buildAdvancedSpec = (node: FlowNodeDraft, ancestors: Map<string, Set<string>>) => {
  const nodeId = node.id.trim() || t("Unnamed")
  const parsed = parseSpecJsonObject(node)
  const inputs = buildInputBindings(
    {
      ...node,
      inputs: parseInputBindings(parsed.inputs)
    },
    ancestors
  )
  const ui = { x: Math.round(Number(node.x || 0)), y: Math.round(Number(node.y || 0)) }

  if (node.kind === "compose") {
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

  const method = String(parsed.method ?? "").trim()
  if (!method) {
    throw new Error(t("Node {nodeId} requires a method.", { nodeId }))
  }
  const target = Number(parsed.target ?? 0)
  if (!Number.isFinite(target) || target < 0) {
    throw new Error(t("Node {nodeId} target must be a non-negative number.", { nodeId }))
  }
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

const buildSpec = (node: FlowNodeDraft, ancestors: Map<string, Set<string>>) =>
  node.specEditorMode === "json" ? buildAdvancedSpec(node, ancestors) : buildFormSpec(node, ancestors)

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
    buildSpec(node, ancestors)
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
  if (kind === "compose") {
    node.composeTemplate = node.composeTemplate.trim() || node.argsTemplate.trim() || "{}"
    node.method = ""
    node.target = 0
  } else {
    node.argsTemplate = node.argsTemplate.trim() || node.composeTemplate.trim() || "{}"
  }
  node.kind = kind
  node.specEditorMode = "form"
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

  const ids = collectNodeIds()
  const topology = buildTopology(ids, state.edges)
  const ancestors = buildAncestorMap(topology.order, topology.parents)
  const normalizedSpec = buildAdvancedSpec(node, ancestors)
  const next = parseSpecDraft(node.kind, normalizedSpec)
  node.method = next.method
  node.target = Number.isFinite(next.target) && next.target > 0 ? Math.trunc(next.target) : 0
  node.argsTemplate = next.argsTemplate
  node.composeTemplate = next.composeTemplate
  node.inputs = next.inputs.map((binding) => ({ ...binding }))
  node.specJson = formatJSONText(normalizedSpec, {})
  node.specEditorMode = "form"
  commitHistory()
  return true
}

const buildGraph = () => {
  if (!state.nodes.length) {
    throw new Error(t("At least one node is required."))
  }
  const ids = collectNodeIds()
  const edges = state.edges.map((edge) => ({
    from: edge.from.trim(),
    to: edge.to.trim()
  }))
  const topology = buildTopology(ids, edges)
  const ancestors = buildAncestorMap(topology.order, topology.parents)

  const nodes = state.nodes.map((node) => {
    const id = node.id.trim()
    return {
      id,
      kind: node.kind,
      allow_fail: Boolean(node.allowFail),
      retry: Number(node.retry ?? 1),
      timeout_ms: Number(node.timeoutMs ?? 3000),
      spec: buildSpec(node, ancestors)
    }
  })

  return { nodes, edges }
}

const buildTrigger = () => {
  const triggerType = state.triggerType
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
      event_topic: eventTopic || undefined
    }
  }
  if (triggerType === "var_changed") {
    const owner = Number(state.varOwner || 0)
    if (!Number.isFinite(owner) || owner < 0) {
      throw new Error(t("Var owner must be a non-negative number."))
    }
    const varName = state.varName.trim()
    return {
      type: "var_changed",
      var_owner: owner > 0 ? Math.trunc(owner) : undefined,
      var_name: varName || undefined
    }
  }
  const everyMs = Number(state.everyMs)
  if (!everyMs || everyMs <= 0) {
    throw new Error(t("EveryMs must be a positive number."))
  }
  return { type: "interval", every_ms: everyMs }
}

const listFlows = async () => {
  const { sourceID, hubID } = ensureIdentity()
  const executorNode = resolveTargetNode()
  const req = { req_id: newReqId(), origin_node: sourceID, executor_node: executorNode }
  const resp = await callFlow<any>("ListSimple", sourceID, hubID, req)
  handleListResp(resp)
}

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

const saveFlow = async () => {
  const { sourceID, hubID } = ensureIdentity()
  const executorNode = resolveTargetNode()
  const flowId = state.flowId.trim()
  if (!flowId) {
    throw new Error(t("Flow ID is required."))
  }
  const trigger = buildTrigger()
  const graph = buildGraph()
  const req = {
    req_id: newReqId(),
    origin_node: sourceID,
    executor_node: executorNode,
    flow_id: flowId,
    name: state.flowName.trim(),
    trigger,
    graph
  }
  const resp = await callFlow<any>("SetSimple", sourceID, hubID, req)
  handleSetResp(resp)
}

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
  handleRunResp(resp)
}

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
  const graph = graphSource && typeof graphSource === "object" && "graph" in graphSource ? graphSource.graph : graphSource
  const nodes = Array.isArray(graph?.nodes) ? graph.nodes : []
  const edges = Array.isArray(graph?.edges) ? graph.edges : []
  state.nodes = nodes.map((node: any, index: number) => mapNode(node, index))
  state.edges = edges.map(mapEdge)
  state.selectedNodeIndex = -1
  state.selectedEdgeIndex = -1
  state.execCapabilities = []
  state.execCapabilitiesLoading = false
  setMessage(successMessage, "success")
  resetHistory()
}

const applyFlowPayload = (data: any, successMessage: string, refreshStatus: boolean) => {
  state.flowId = String(data?.flow_id ?? data?.flowId ?? "").trim()
  state.flowName = String(data?.name ?? "").trim()
  const trigger = data?.trigger ?? {}
  const triggerType = String(trigger?.type ?? trigger?.triggerType ?? "interval").trim().toLowerCase()
  if (triggerType === "event" || triggerType === "var_changed") {
    state.triggerType = triggerType
  } else {
    state.triggerType = "interval"
  }
  const everyMs = Number(trigger?.every_ms ?? trigger?.everyMs ?? 0)
  state.everyMs = everyMs > 0 ? everyMs : 60000
  const rawEventMode = String(trigger?.event_mode ?? trigger?.eventMode ?? "publish").trim().toLowerCase()
  if (rawEventMode === "received" || rawEventMode === "any") {
    state.eventMode = rawEventMode
  } else {
    state.eventMode = "publish"
  }
  state.eventName = String(trigger?.event_name ?? trigger?.eventName ?? "").trim()
  state.eventTopic = String(trigger?.event_topic ?? trigger?.eventTopic ?? "").trim()
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
  return {
    flow_id: flowID,
    name: state.flowName.trim(),
    trigger: buildTrigger(),
    graph: buildGraph()
  }
}

const exportGraphDraft = (): FlowGraphDraft => buildGraph()

const queryExecCapabilities = async (methodFilter?: string, queryNodeId?: string | number) => {
  const { sourceID } = ensureIdentity()
  const executorNode = resolveCapabilityQueryNode(queryNodeId)
  const method = String(methodFilter ?? "").trim()
  const req = {
    req_id: newReqId(),
    requester_node: sourceID,
    method: method || undefined,
    prefix: method.length > 0,
    limit: 200,
    include_schema: true
  }
  state.execCapabilitiesLoading = true
  try {
    const data = await callFlow<any>("ExecCapQuerySimple", sourceID, executorNode, req)
    const code = Number(data?.code ?? 0)
    const msg = String(data?.msg ?? "")
    if (code !== 1) {
      state.execCapabilities = []
      setMessage(msg || t("Capability query failed."), "error")
      return
    }
    const routes: any[] = Array.isArray(data?.routes) ? data.routes : []
    state.execCapabilities = routes
      .map(mapExecCapabilityRoute)
      .filter((route: ExecCapabilityRoute) => route.providerNode > 0 && route.method.length > 0)
    setMessage(
      state.execCapabilities.length
        ? t("Capability list updated from node {nodeId} ({count}).", {
            nodeId: executorNode,
            count: state.execCapabilities.length
          })
        : t("No capability matched on node {nodeId}.", { nodeId: executorNode }),
      state.execCapabilities.length ? "success" : "info"
    )
  } finally {
    state.execCapabilitiesLoading = false
  }
}

const applyCallCapability = (key: string) => {
  const selected = state.nodes[state.selectedNodeIndex]
  if (!selected || selected.kind !== "call") {
    throw new Error(t("Select a call node first."))
  }
  const route = state.execCapabilities.find((item) => item.key === String(key).trim())
  if (!route) {
    throw new Error(t("Capability not found in current list."))
  }
  selected.method = route.method
  selected.target = normalizeCallTarget(route.providerNode)
  if (!String(selected.argsTemplate ?? "").trim()) {
    selected.argsTemplate = "{}"
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

const handleRunResp = (data: any) => {
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    setMessage(msg || t("Flow run failed."), "error")
    return
  }
  const runId = String(data?.run_id ?? "")
  state.statusRunId = runId
  setMessage(t("Flow run started."), "success")
  void statusFlow(runId).catch(() => {})
}

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
  }
  setMessage(t("Status updated."), "success")
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
    normalizeCallTarget,
    queryExecCapabilities,
    applyCallCapability,
    applyExecCapability: applyCallCapability,
    setFieldLiteralValue: setNodeFieldLiteralValue,
    setFieldBinding: setNodeFieldBinding,
    clearFieldBinding: clearNodeFieldBinding,
    describeFieldBinding: describeVisualFieldBinding,
    setNodeKind,
    setNodeSpecEditorMode,
    undo,
    selectEdgeByEndpoints: (from: string, to: string) => {
      const fromId = from.trim()
      const toId = to.trim()
      const idx = state.edges.findIndex((edge) => edge.from === fromId && edge.to === toId)
      state.selectedEdgeIndex = idx
      state.selectedNodeIndex = -1
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
