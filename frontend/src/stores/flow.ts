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

type FlowDraftSnapshot = {
  flowId: string
  flowName: string
  everyMs: number
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
  everyMs: number
  nodes: FlowNodeDraft[]
  edges: FlowEdge[]
  selectedNodeIndex: number
  selectedEdgeIndex: number
  statusRunId: string
  lastStatus: FlowStatus
  message: string
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
  historyIndex: 0,
  historyLength: 1
})

const MAX_HISTORY = 120
let draftHistory: FlowDraftSnapshot[] = []
let draftHistoryIndex = 0

const snapshotToJSON = (snapshot: FlowDraftSnapshot) => JSON.stringify(snapshot)

const takeSnapshot = (): FlowDraftSnapshot => ({
  flowId: state.flowId,
  flowName: state.flowName,
  everyMs: state.everyMs,
  nodes: state.nodes.map((node) => ({
    id: node.id,
    kind: node.kind,
    allowFail: node.allowFail,
    retry: node.retry,
    timeoutMs: node.timeoutMs,
    method: node.method,
    target: node.target,
    args: node.args,
    x: node.x,
    y: node.y
  })),
  edges: state.edges.map((edge) => ({ from: edge.from, to: edge.to })),
  selectedNodeIndex: state.selectedNodeIndex,
  selectedEdgeIndex: state.selectedEdgeIndex
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
  state.everyMs = snapshot.everyMs
  state.nodes = snapshot.nodes.map((node) => ({ ...node }))
  state.edges = snapshot.edges.map((edge) => ({ ...edge }))
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

const undo = () => {
  if (draftHistoryIndex <= 0) return false
  draftHistoryIndex -= 1
  applySnapshot(draftHistory[draftHistoryIndex])
  updateHistoryState()
  state.message = "Undo applied."
  return true
}

const redo = () => {
  if (draftHistoryIndex >= draftHistory.length - 1) return false
  draftHistoryIndex += 1
  applySnapshot(draftHistory[draftHistoryIndex])
  updateHistoryState()
  state.message = "Redo applied."
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
  resetHistory()
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
  commitHistory()
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
    throw new Error("No nodes to layout.")
  }

  const ids = state.nodes.map((node) => node.id.trim()).filter(Boolean)
  const idSet = new Set(ids)
  const nodeOrder = new Map<string, number>()
  for (const [idx, id] of ids.entries()) {
    nodeOrder.set(id, idx)
  }

  const indegree = new Map<string, number>()
  const next = new Map<string, string[]>()
  for (const id of ids) {
    indegree.set(id, 0)
    next.set(id, [])
  }

  for (const edge of state.edges) {
    const from = edge.from.trim()
    const to = edge.to.trim()
    if (!from || !to) continue
    if (!idSet.has(from) || !idSet.has(to)) {
      throw new Error("Flow graph contains invalid edge endpoints.")
    }
    next.get(from)?.push(to)
    indegree.set(to, (indegree.get(to) ?? 0) + 1)
  }

  const level = new Map<string, number>()
  const queue: string[] = []
  for (const id of ids) {
    if ((indegree.get(id) ?? 0) === 0) {
      queue.push(id)
      level.set(id, 0)
    }
  }
  queue.sort((a, b) => (nodeOrder.get(a) ?? 0) - (nodeOrder.get(b) ?? 0))

  const topo: string[] = []
  while (queue.length) {
    const cur = queue.shift()
    if (!cur) continue
    topo.push(cur)
    const base = level.get(cur) ?? 0
    for (const child of next.get(cur) ?? []) {
      level.set(child, Math.max(level.get(child) ?? 0, base + 1))
      const left = (indegree.get(child) ?? 0) - 1
      indegree.set(child, left)
      if (left === 0) {
        queue.push(child)
        queue.sort((a, b) => (nodeOrder.get(a) ?? 0) - (nodeOrder.get(b) ?? 0))
      }
    }
  }

  if (topo.length !== ids.length) {
    throw new Error("Auto layout requires a DAG (cycle detected).")
  }

  const groups = new Map<number, string[]>()
  for (const id of topo) {
    const depth = level.get(id) ?? 0
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
  resetHistory()
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
  if (!draftHistory.length) {
    resetHistory()
  }

  return {
    state,
    addEdge,
    addNode,
    autoLayoutTB,
    commitHistory,
    clearSelection: () => {
      state.selectedNodeIndex = -1
      state.selectedEdgeIndex = -1
    },
    getFlow,
    listFlows,
    newDraft,
    removeSelectedEdge,
    removeSelectedNode,
    redo,
    runFlow,
    saveFlow,
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
