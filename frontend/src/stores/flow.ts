import { reactive } from "vue"

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
  x: number
  y: number
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
  message: ""
})

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
  const ui = (parsed as any)?._ui
  const x = Number(ui?.x)
  const y = Number(ui?.y)
  return {
    method,
    target,
    argsText,
    x: Number.isFinite(x) ? x : undefined,
    y: Number.isFinite(y) ? y : undefined
  }
}

const defaultNodePosition = (index: number) => {
  const col = index % 4
  const row = Math.floor(index / 4)
  return { x: col * 240, y: row * 160 }
}

const mapNode = (input: any, index: number): FlowNodeDraft => {
  const kind = String(input?.kind ?? "").toLowerCase()
  const { method, target, argsText, x, y } = parseSpec(input?.spec)
  const pos = defaultNodePosition(index)
  return {
    id: String(input?.id ?? "").trim(),
    kind: kind === "exec" ? "exec" : "local",
    allowFail: Boolean(input?.allow_fail ?? input?.allowFail ?? false),
    retry: Number(input?.retry ?? 1),
    timeoutMs: Number(input?.timeout_ms ?? input?.timeoutMs ?? 3000),
    method,
    target,
    args: argsText,
    x: Number.isFinite(x) ? Number(x) : pos.x,
    y: Number.isFinite(y) ? Number(y) : pos.y
  }
}

const mapEdge = (input: any): FlowEdge => ({
  from: String(input?.from ?? "").trim(),
  to: String(input?.to ?? "").trim()
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
  const pos = defaultNodePosition(state.nodes.length)
  const node: FlowNodeDraft = {
    id: trimmed,
    kind,
    allowFail: false,
    retry: 1,
    timeoutMs: 3000,
    method: "",
    target: 0,
    args: "{}",
    x: pos.x,
    y: pos.y
  }
  state.nodes.push(node)
  state.selectedNodeIndex = state.nodes.length - 1
  state.selectedEdgeIndex = -1
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
  if (state.edges.some((edge) => edge.from === fromId && edge.to === toId)) {
    throw new Error("Edge already exists.")
  }
  const next = buildAdjacency(state.edges)
  if (isReachable(toId, fromId, next)) {
    throw new Error("Edge would create a cycle.")
  }
  state.edges.push({ from: fromId, to: toId })
  state.selectedEdgeIndex = state.edges.length - 1
  state.selectedNodeIndex = -1
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
    return {
      target: node.target,
      method,
      args: parsedArgs,
      _ui: { x: Math.round(Number(node.x || 0)), y: Math.round(Number(node.y || 0)) }
    }
  }
  return {
    method,
    args: parsedArgs,
    _ui: { x: Math.round(Number(node.x || 0)), y: Math.round(Number(node.y || 0)) }
  }
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
  const resp = await callFlow<any>("ListSimple", sourceID, hubID, req)
  handleListResp(resp)
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
  const resp = await callFlow<any>("GetSimple", sourceID, hubID, req)
  handleGetResp(resp)
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
  const resp = await callFlow<any>("SetSimple", sourceID, hubID, req)
  handleSetResp(resp)
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
  const resp = await callFlow<any>("RunSimple", sourceID, hubID, req)
  handleRunResp(resp)
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
  const resp = await callFlow<any>("StatusSimple", sourceID, hubID, req)
  handleStatusResp(resp)
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
  state.nodes = nodes.map((node: any, index: number) => mapNode(node, index))
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

export const useFlowStore = () => {
  return {
    state,
    addEdge,
    addNode,
    clearSelection: () => {
      state.selectedNodeIndex = -1
      state.selectedEdgeIndex = -1
    },
    getFlow,
    listFlows,
    newDraft,
    removeSelectedEdge,
    removeSelectedNode,
    runFlow,
    saveFlow,
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
