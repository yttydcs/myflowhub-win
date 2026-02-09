import { reactive } from "vue"
import { EventsOn } from "../../wailsjs/runtime/runtime"

type WailsBinding = (...args: any[]) => Promise<any>

const callMgmt = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.management?.ManagementService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(`Management binding '${method}' unavailable`)
  }
  return fn(...args)
}

export type MgmtNode = {
  nodeId: number
  hasChildren: boolean
}

export type MgmtConfigEntry = {
  key: string
  value: string
}

type MgmtState = {
  targetId: string
  selfNodeId: number
  hubId: number
  listMode: "direct" | "subtree"
  nodes: MgmtNode[]
  selectedNodeId: number
  configEntries: MgmtConfigEntry[]
  message: string
  lastFrameAt: string
}

const state = reactive<MgmtState>({
  targetId: "",
  selfNodeId: 0,
  hubId: 0,
  listMode: "direct",
  nodes: [],
  selectedNodeId: 0,
  configEntries: [],
  message: "",
  lastFrameAt: ""
})

let initialized = false

const nowIso = () => new Date().toISOString()

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
    throw new Error("Login required to send Management requests.")
  }
  if (!state.hubId) {
    throw new Error("Hub ID missing.")
  }
  return { sourceID: state.selfNodeId, hubID: state.hubId }
}

const listNodes = async () => {
  const { sourceID } = ensureIdentity()
  const targetID = resolveTargetNode()
  await callMgmt("ListNodesSimple", sourceID, targetID)
}

const listSubtree = async () => {
  const { sourceID } = ensureIdentity()
  const targetID = resolveTargetNode()
  await callMgmt("ListSubtreeSimple", sourceID, targetID)
}

const selectNode = async (nodeId: number) => {
  state.selectedNodeId = nodeId
  state.configEntries = []
  if (!nodeId) return
  const { sourceID } = ensureIdentity()
  await callMgmt("ConfigListSimple", sourceID, nodeId)
}

const refreshConfig = async () => {
  if (!state.selectedNodeId) {
    throw new Error("Select a node to load config.")
  }
  const { sourceID } = ensureIdentity()
  await callMgmt("ConfigListSimple", sourceID, state.selectedNodeId)
}

const setConfig = async (key: string, value: string) => {
  const trimmed = key.trim()
  if (!trimmed) {
    throw new Error("Config key is required.")
  }
  if (!state.selectedNodeId) {
    throw new Error("Select a node to update config.")
  }
  const { sourceID } = ensureIdentity()
  await callMgmt("ConfigSetSimple", sourceID, state.selectedNodeId, trimmed, value)
}

const handleListResp = (data: any, mode: "direct" | "subtree") => {
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    state.message = msg || "Management list failed."
    return
  }
  const nodes = Array.isArray(data?.nodes) ? data.nodes : []
  const list = nodes
    .map((node: any) => ({
      nodeId: Number(node?.node_id ?? 0),
      hasChildren: Boolean(node?.has_children ?? false)
    }))
    .filter((node: MgmtNode) => node.nodeId > 0)
    .sort((a: MgmtNode, b: MgmtNode) => a.nodeId - b.nodeId)
  const target = Number.parseInt(state.targetId || "0", 10)
  if (target && list.some((node) => node.nodeId === target)) {
    const current = list.find((node) => node.nodeId === target)
    const rest = list.filter((node) => node.nodeId !== target)
    if (current) {
      state.nodes = [current, ...rest]
    } else {
      state.nodes = list
    }
  } else {
    state.nodes = list
  }
  state.listMode = mode
  state.message = mode === "direct" ? "Direct nodes loaded." : "Subtree loaded."
}

const handleConfigListResp = (data: any, sourceID: number) => {
  if (!state.selectedNodeId || sourceID !== state.selectedNodeId) {
    return
  }
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    state.message = msg || "Config list failed."
    return
  }
  const keys = Array.isArray(data?.keys) ? data.keys : []
  state.configEntries = keys
    .map((key: any) => String(key ?? "").trim())
    .filter((key: string) => key)
    .map((key: string) => ({ key, value: "" }))
  if (!state.selfNodeId) {
    return
  }
  const caller = state.selfNodeId
  for (const entry of state.configEntries) {
    void callMgmt("ConfigGetSimple", caller, state.selectedNodeId, entry.key).catch(() => {})
  }
}

const handleConfigGetResp = (data: any, sourceID: number) => {
  if (!state.selectedNodeId || sourceID !== state.selectedNodeId) {
    return
  }
  const code = Number(data?.code ?? 0)
  if (code !== 1) {
    return
  }
  const key = String(data?.key ?? "").trim()
  if (!key) return
  const value = String(data?.value ?? "")
  const idx = state.configEntries.findIndex((entry) => entry.key === key)
  if (idx >= 0) {
    state.configEntries[idx].value = value
  } else {
    state.configEntries.push({ key, value })
  }
}

const handleConfigSetResp = (data: any, sourceID: number) => {
  if (!state.selectedNodeId || sourceID !== state.selectedNodeId) {
    return
  }
  const code = Number(data?.code ?? 0)
  const msg = String(data?.msg ?? "")
  if (code !== 1) {
    state.message = msg || "Config update failed."
    return
  }
  const key = String(data?.key ?? "").trim()
  if (!key) return
  const value = String(data?.value ?? "")
  const idx = state.configEntries.findIndex((entry) => entry.key === key)
  if (idx >= 0) {
    state.configEntries[idx].value = value
  } else {
    state.configEntries.push({ key, value })
  }
  state.message = "Config updated."
}

const handleFrame = (payload: any, sourceID: number) => {
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
    case "list_nodes_resp":
      handleListResp(data, "direct")
      break
    case "list_subtree_resp":
      handleListResp(data, "subtree")
      break
    case "config_list_resp":
      handleConfigListResp(data, sourceID)
      break
    case "config_get_resp":
      handleConfigGetResp(data, sourceID)
      break
    case "config_set_resp":
      handleConfigSetResp(data, sourceID)
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
    if (subProto !== 1) return
    const sourceID = Number(evt?.source_id ?? evt?.sourceId ?? 0)
    handleFrame(evt?.payload, sourceID)
  })
}

export const useManagementStore = () => {
  ensureListeners()
  return {
    state,
    listNodes,
    listSubtree,
    refreshConfig,
    selectNode,
    setConfig,
    setIdentity: (nodeId: number, hubId: number) => {
      state.selfNodeId = Number(nodeId || 0)
      state.hubId = Number(hubId || 0)
      if (!state.targetId && state.hubId) {
        state.targetId = String(state.hubId)
      }
    }
  }
}
