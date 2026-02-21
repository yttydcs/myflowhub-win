import { reactive } from "vue"
import { useSessionStore } from "@/stores/session"

type WailsBinding = (...args: any[]) => Promise<any>

const callMgmt = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.management?.ManagementService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(`Management binding '${method}' unavailable`)
  }
  return fn(...args)
}

export type DeviceNode = {
  nodeId: number
  hasChildren: boolean
}

type NodeWire = {
  node_id: number
  has_children?: boolean
}

type ListNodesWire = {
  code: number
  msg?: string
  nodes?: NodeWire[]
}

type DevicesState = {
  targetId: string
  listMode: "direct" | "subtree"
  nodes: DeviceNode[]
  message: string
}

const state = reactive<DevicesState>({
  targetId: "",
  listMode: "direct",
  nodes: [],
  message: ""
})

const resolveTargetNode = (fallbackHubId: number) => {
  const raw = state.targetId.trim()
  if (!raw) {
    if (!fallbackHubId) {
      throw new Error("Target node is required.")
    }
    return fallbackHubId
  }
  const parsed = Number.parseInt(raw, 10)
  if (Number.isNaN(parsed) || parsed <= 0) {
    throw new Error("Target node must be a positive number.")
  }
  return parsed
}

const ensureIdentity = () => {
  const session = useSessionStore()
  if (!session.connected) {
    throw new Error("Connect before querying devices.")
  }
  const sourceID = Number(session.auth.nodeId || 0)
  const hubID = Number(session.auth.hubId || 0)
  if (!sourceID) {
    throw new Error("Login required to query devices.")
  }
  if (!hubID) {
    throw new Error("Hub ID missing.")
  }
  if (!state.targetId && hubID) {
    state.targetId = String(hubID)
  }
  return { sourceID, hubID }
}

const applyListResp = (resp: ListNodesWire, mode: "direct" | "subtree", targetID: number) => {
  const nodes = Array.isArray(resp?.nodes) ? resp.nodes : []
  const list = nodes
    .map((node) => ({
      nodeId: Number(node?.node_id ?? 0),
      hasChildren: Boolean(node?.has_children ?? false)
    }))
    .filter((node) => node.nodeId > 0)
    .sort((a, b) => a.nodeId - b.nodeId)

  if (targetID && list.some((node) => node.nodeId === targetID)) {
    const current = list.find((node) => node.nodeId === targetID)
    const rest = list.filter((node) => node.nodeId !== targetID)
    state.nodes = current ? [current, ...rest] : list
  } else {
    state.nodes = list
  }

  state.listMode = mode
  state.message = mode === "direct" ? "Direct nodes loaded." : "Subtree loaded."
}

const listDirect = async () => {
  const { sourceID, hubID } = ensureIdentity()
  const targetID = resolveTargetNode(hubID)
  const resp = await callMgmt<ListNodesWire>("ListNodesSimple", sourceID, targetID)
  applyListResp(resp, "direct", targetID)
}

const listSubtree = async () => {
  const { sourceID, hubID } = ensureIdentity()
  const targetID = resolveTargetNode(hubID)
  const resp = await callMgmt<ListNodesWire>("ListSubtreeSimple", sourceID, targetID)
  applyListResp(resp, "subtree", targetID)
}

export const useDevicesStore = () => {
  return {
    state,
    listDirect,
    listSubtree,
    reset: () => {
      state.nodes = []
      state.message = ""
      state.listMode = "direct"
    }
  }
}

