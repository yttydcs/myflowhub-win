import { reactive } from "vue"
import { EventsOn } from "../../wailsjs/runtime/runtime"

type WailsBinding = (...args: any[]) => Promise<any>

const callFlow = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.flow?.FlowService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(`Flow binding '${method}' unavailable`)
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

export type FlowNodeDraft = {
  id: string
  kind: "local" | "exec" | ""
  allowFail: boolean
  retry: number
  timeoutMs: number
  method: string
  target: number
  args: string
}

export type FlowEdge = {
  from: string
  to: string
}

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

type FlowState = {
  targetId: string
  selfNodeId: number
  hubId: number
  flows: FlowSummary[]
  flowId: string
  flowName: string
  everyMs: number
  nodes: FlowNodeDraft[]
  edges: FlowEdge[]
  selectedNodeIndex: number
  selectedEdgeIndex: number
  statusRunId: string
  lastStatus: FlowStatus
  message: string
  lastFrameAt: string
}

const state = reactive<FlowState>({
  targetId: "",
  selfNodeId: 0,
  hubId: 0,
  flows: [],
  flowId: "",
  flowName: "",
  everyMs: 60000,
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
  message: "",
  lastFrameAt: ""
})

let initialized = false

const nowIso = () => new Date().toISOString()

const newReqId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID()
  }
  return `req_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`
}

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

const resolveTargetNode = () => {
  const raw = state.targetId.trim()
  if (!raw) {
    if (!state.hubId) {
      throw new Error("Target node is required.")
    }
    return state.hubId
  }
  const parsed = Number.parseInt(raw, 10)
  if (Number.isNaN(parsed) || parsed <= 0) {
    throw new Error("Target node must be a positive number.")
  }
  return parsed
}

const ensureIdentity = () => {
  if (!state.selfNodeId) {
    throw new Error("Login required to send Flow requests.")
  }
  if (!state.hubId) {
    throw new Error("Hub ID missing.")
  }
  return { sourceID: state.selfNodeId, hubID: state.hubId }
}

const mapSummary = (input: any): FlowSummary => ({
  flowId: String(input?.flow_id ?? input?.flowId ?? ""),
  name: String(input?.name ?? ""),
  everyMs: Number(input?.every_ms ?? input?.everyMs ?? 0),
  lastRunId: String(input?.last_run_id ?? input?.lastRunId ?? ""),
  lastStatus: String(input?.last_status ?? input?.lastStatus ?? "")
})

const parseArgsText = (args: any) => {
  if (args === undefined || args === null) return "{}"
  try {
    return JSON.stringify(args, null, 2)
  } catch {
    return "{}"
  }
}

const parseSpec = (spec: any) => {
  let parsed = spec
  if (typeof spec === "string") {
    try {
      parsed = JSON.parse(spec)
    } catch {
      parsed = {}
    }
  }
  if (!parsed || typeof parsed !== "object") {
    parsed = {}
  }
  const method = String((parsed as any)?.method ?? "")
  const target = Number((parsed as any)?.target ?? 0)
  const argsText = parseArgsText((parsed as any)?.args)
  return { method, target, argsText }
}

const mapNode = (input: any): FlowNodeDraft => {
  const kind = String(input?.kind ?? "").toLowerCase()
  const { method, target, argsText } = parseSpec(input?.spec)
  return {
    id: String(input?.id ?? ""),
    kind: kind === "exec" ? "exec" : "local",
    allowFail: Boolean(input?.allow_fail ?? input?.allowFail ?? false),
    retry: Number(input?.retry ?? 1),
    timeoutMs: Number(input?.timeout_ms ?? input?.timeoutMs ?? 3000),
    method,
    target,
    args: argsText
  }
}

const mapEdge = (input: any): FlowEdge => ({
  from: String(input?.from ?? ""),
  to: String(input?.to ?? "")
})

const newDraft = () => {
  state.flowId = ""
  state.flowName = ""
  state.everyMs = 60000
  state.nodes = []
  state.edges = []
  state.selectedNodeIndex = -1
  state.selectedEdgeIndex = -1
  state.statusRunId = ""
}

const addNode = (id: string, kind: "local" | "exec") => {
  const trimmed = id.trim()
  if (!trimmed) {
    throw new Error("Node ID is required.")
  }
  if (state.nodes.find((node) => node.id.trim() === trimmed)) {
    throw new Error("Node ID must be unique.")
  }
  const node: FlowNodeDraft = {
    id: trimmed,
    kind,
    allowFail: false,
    retry: 1,
    timeoutMs: 3000,
    method: "",
    target: 0,
    args: "{}"
  }
  state.nodes.push(node)
  state.selectedNodeIndex = state.nodes.length - 1
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
}

const addEdge = (from: string, to: string) => {
  const fromId = from.trim()
  const toId = to.trim()
  if (!fromId || !toId || fromId === toId) {
    throw new Error("Edge endpoints must be different.")
  }
  if (!state.nodes.find((node) => node.id.trim() === fromId)) {
    throw new Error("From node does not exist.")
  }
  if (!state.nodes.find((node) => node.id.trim() === toId)) {
    throw new Error("To node does not exist.")
  }
  state.edges.push({ from: fromId, to: toId })
  state.selectedEdgeIndex = state.edges.length - 1
}

const removeSelectedEdge = () => {
  const idx = state.selectedEdgeIndex
  if (idx < 0 || idx >= state.edges.length) return
  state.edges = state.edges.filter((_, i) => i !== idx)
  state.selectedEdgeIndex = -1
}

const buildSpec = (node: FlowNodeDraft) => {
  const method = node.method.trim()
  if (!method) {
    throw new Error(`Node ${node.id || "<unnamed>"} requires a method.`)
  }
  let parsedArgs: any = {}
  const rawArgs = node.args?.trim() || "{}"
  try {
    parsedArgs = JSON.parse(rawArgs)
  } catch {
    throw new Error(`Node ${node.id || "<unnamed>"} args must be valid JSON.`)
  }
  if (node.kind === "exec") {
    if (!node.target) {
      throw new Error(`Node ${node.id || "<unnamed>"} requires target node.`)
    }
    return { target: node.target, method, args: parsedArgs }
  }
  return { method, args: parsedArgs }
}

const buildGraph = () => {
  if (!state.nodes.length) {
    throw new Error("At least one node is required.")
  }
  const seen = new Set<string>()
  const nodes = state.nodes.map((node) => {
    const id = node.id.trim()
    if (!id) {
      throw new Error("Node ID is required.")
    }
    if (seen.has(id)) {
      throw new Error(`Duplicate node ID: ${id}`)
    }
    seen.add(id)
    const kind = node.kind === "exec" ? "exec" : "local"
    const spec = buildSpec(node)
    return {
      id,
      kind,
      allow_fail: Boolean(node.allowFail),
      retry: Number(node.retry ?? 1),
      timeout_ms: Number(node.timeoutMs ?? 3000),
      spec
    }
  })
  const edges = state.edges.map((edge) => {
    const from = edge.from.trim()
    const to = edge.to.trim()
    if (!from || !to || from === to) {
      throw new Error("Edge endpoints are invalid.")
    }
    if (!seen.has(from) || !seen.has(to)) {
      throw new Error("Edge references unknown nodes.")
    }
    return { from, to }
  })
  return { nodes, edges }
}

const listFlows = async () => {
  const { sourceID, hubID } = ensureIdentity()
  const executorNode = resolveTargetNode()
  const req = { req_id: newReqId(), origin_node: sourceID, executor_node: executorNode }
  await callFlow("ListSimple", sourceID, hubID, req)
}

const getFlow = async (flowId: string) => {
  const { sourceID, hubID } = ensureIdentity()
  const executorNode = resolveTargetNode()
  const trimmed = flowId.trim()
  if (!trimmed) {
    throw new Error("Flow ID is required.")
  }
  const req = {
    req_id: newReqId(),
    origin_node: sourceID,
    executor_node: executorNode,
    flow_id: trimmed
  }
  await callFlow("GetSimple", sourceID, hubID, req)
}

const saveFlow = async () => {
  const { sourceID, hubID } = ensureIdentity()
  const executorNode = resolveTargetNode()
  const flowId = state.flowId.trim()
  if (!flowId) {
    throw new Error("Flow ID is required.")
  }
  const everyMs = Number(state.everyMs)
  if (!everyMs || everyMs <= 0) {
    throw new Error("EveryMs must be a positive number.")
  }
  const graph = buildGraph()
  const req = {
    req_id: newReqId(),
    origin_node: sourceID,
    executor_node: executorNode,
    flow_id: flowId,
    name: state.flowName.trim(),
    trigger: { type: "interval", every_ms: everyMs },
    graph
  }
  await callFlow("SetSimple", sourceID, hubID, req)
}

const runFlow = async () => {
  const { sourceID, hubID } = ensureIdentity()
  const executorNode = resolveTargetNode()
  const flowId = state.flowId.trim()
  if (!flowId) {
    throw new Error("Flow ID is required.")
  }
  const req = {
    req_id: newReqId(),
    origin_node: sourceID,
    executor_node: executorNode,
    flow_id: flowId
  }
  await callFlow("RunSimple", sourceID, hubID, req)
}

const statusFlow = async (runId?: string) => {
  const { sourceID, hubID } = ensureIdentity()
  const executorNode = resolveTargetNode()
  const flowId = state.flowId.trim()
  if (!flowId) {
    throw new Error("Flow ID is required.")
  }
  const req = {
    req_id: newReqId(),
    origin_node: sourceID,
    executor_node: executorNode,
    flow_id: flowId,
    run_id: runId?.trim() || undefined
  }
  await callFlow("StatusSimple", sourceID, hubID, req)
}

const handleListResp = (data: any) => {
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    state.message = msg || "Flow list failed."
    return
  }
  const flows = Array.isArray(data?.flows) ? data.flows : []
  state.flows = flows.map(mapSummary)
  state.message = "Flow list updated."
}

const handleGetResp = (data: any) => {
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    state.message = msg || "Flow load failed."
    return
  }
  state.flowId = String(data?.flow_id ?? "")
  state.flowName = String(data?.name ?? "")
  const everyMs = Number(data?.trigger?.every_ms ?? 0)
  if (everyMs > 0) {
    state.everyMs = everyMs
  }
  const nodes = Array.isArray(data?.graph?.nodes) ? data.graph.nodes : []
  const edges = Array.isArray(data?.graph?.edges) ? data.graph.edges : []
  state.nodes = nodes.map(mapNode)
  state.edges = edges.map(mapEdge)
  state.selectedNodeIndex = -1
  state.selectedEdgeIndex = -1
  state.message = "Flow loaded."
  if (state.selfNodeId && state.hubId) {
    void statusFlow("").catch(() => {})
  }
}

const handleSetResp = (data: any) => {
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    state.message = msg || "Flow save failed."
    return
  }
  state.message = "Flow saved."
  void listFlows().catch(() => {})
}

const handleRunResp = (data: any) => {
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    state.message = msg || "Flow run failed."
    return
  }
  const runId = String(data?.run_id ?? "")
  state.statusRunId = runId
  state.message = "Flow run started."
  void statusFlow(runId).catch(() => {})
}

const handleStatusResp = (data: any) => {
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    state.message = msg || "Flow status failed."
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
  state.message = "Status updated."
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
  switch (action) {
    case "list_resp":
      handleListResp(data)
      break
    case "get_resp":
      handleGetResp(data)
      break
    case "set_resp":
      handleSetResp(data)
      break
    case "run_resp":
      handleRunResp(data)
      break
    case "status_resp":
      handleStatusResp(data)
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
    if (subProto !== 6) return
    handleFrame(evt?.payload)
  })
}

export const useFlowStore = () => {
  ensureListeners()
  return {
    state,
    addEdge,
    addNode,
    getFlow,
    listFlows,
    newDraft,
    removeSelectedEdge,
    removeSelectedNode,
    runFlow,
    saveFlow,
    setIdentity: (nodeId: number, hubId: number) => {
      state.selfNodeId = Number(nodeId || 0)
      state.hubId = Number(hubId || 0)
      if (!state.targetId && state.hubId) {
        state.targetId = String(state.hubId)
      }
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
