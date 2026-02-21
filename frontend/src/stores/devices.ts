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

export type DevicesMode = "direct" | "subtree"

export type DeviceTreeNode = {
  key: string
  nodeId: number
  hasChildrenHint: boolean
  duplicate: boolean
  expanded: boolean
  loading: boolean
  error: string
  children: DeviceTreeNode[] | null
}

type NodeWire = {
  node_id?: number
  nodeId?: number
  has_children?: boolean
  hasChildren?: boolean
}

type ListNodesWire = {
  code: number
  msg?: string
  nodes?: NodeWire[]
}

type NodeInfo = {
  nodeId: number
  hasChildren: boolean
}

type DevicesState = {
  mode: DevicesMode
  rootTargetId: string
  root: DeviceTreeNode | null
  message: string
}

const state = reactive<DevicesState>({
  mode: "direct",
  rootTargetId: "",
  root: null,
  message: ""
})

let epoch = 0
let seenNodeIDs = new Set<number>()
const nodeIndex = new Map<string, DeviceTreeNode>()

const toErrorMessage = (err: unknown) => {
  if (!err) return "Unknown error."
  if (err instanceof Error) return err.message || "Unknown error."
  return String(err)
}

const resolveTargetNode = (fallbackHubId: number) => {
  const raw = state.rootTargetId.trim()
  if (!raw) {
    if (!fallbackHubId) {
      throw new Error("Root node is required.")
    }
    state.rootTargetId = String(fallbackHubId)
    return fallbackHubId
  }
  const parsed = Number.parseInt(raw, 10)
  if (Number.isNaN(parsed) || parsed <= 0) {
    throw new Error("Root node must be a positive number.")
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
  if (!state.rootTargetId && hubID) {
    state.rootTargetId = String(hubID)
  }
  return { sourceID, hubID }
}

const normalizeNodes = (resp: ListNodesWire): NodeInfo[] => {
  const nodes = Array.isArray(resp?.nodes) ? resp.nodes : []
  return nodes
    .map((node) => ({
      nodeId: Number(node?.node_id ?? node?.nodeId ?? 0),
      hasChildren: Boolean(node?.has_children ?? node?.hasChildren ?? false)
    }))
    .filter((node) => node.nodeId > 0)
    .sort((a, b) => a.nodeId - b.nodeId)
}

const clearRuntimeTree = () => {
  state.root = null
  state.message = ""
  nodeIndex.clear()
  seenNodeIDs = new Set<number>()
}

const makeChildKey = (parentKey: string, nodeId: number) => {
  const base = `${parentKey}/${nodeId}`
  let key = base
  let suffix = 1
  while (nodeIndex.has(key)) {
    suffix++
    key = `${base}@${suffix}`
  }
  return key
}

const registerNode = (node: DeviceTreeNode) => {
  nodeIndex.set(node.key, node)
}

const buildChildNodes = (parentKey: string, children: NodeInfo[], selfNodeId: number) => {
  const list: DeviceTreeNode[] = []
  for (const child of children) {
    if (!child?.nodeId || child.nodeId === selfNodeId) continue
    const duplicate = seenNodeIDs.has(child.nodeId)
    if (!duplicate) {
      seenNodeIDs.add(child.nodeId)
    }
    const node = reactive<DeviceTreeNode>({
      key: makeChildKey(parentKey, child.nodeId),
      nodeId: child.nodeId,
      hasChildrenHint: Boolean(child.hasChildren),
      duplicate,
      expanded: false,
      loading: false,
      error: "",
      children: null
    }) as DeviceTreeNode
    registerNode(node)
    list.push(node)
  }
  return list
}

const loadChildren = async (node: DeviceTreeNode, sourceID: number, myEpoch: number, force: boolean) => {
  if (node.duplicate) return
  if (node.loading) return
  if (!force && node.children !== null) return

  node.loading = true
  node.error = ""
  try {
    const resp = await callMgmt<ListNodesWire>("ListNodesSimple", sourceID, node.nodeId)
    if (epoch !== myEpoch) return
    const children = normalizeNodes(resp).filter((info) => info.nodeId !== node.nodeId)
    node.children = buildChildNodes(node.key, children, node.nodeId)
    node.loading = false
  } catch (err) {
    if (epoch !== myEpoch) return
    node.loading = false
    node.error = toErrorMessage(err)
    node.children = null
  }
}

const loadRoot = async () => {
  const myEpoch = ++epoch
  clearRuntimeTree()

  let sourceID = 0
  let hubID = 0
  let rootID = 0
  try {
    const identity = ensureIdentity()
    sourceID = identity.sourceID
    hubID = identity.hubID
    rootID = resolveTargetNode(hubID)
  } catch (err) {
    if (epoch !== myEpoch) return
    state.message = toErrorMessage(err)
    throw err
  }

  const root = reactive<DeviceTreeNode>({
    key: `root:${rootID}`,
    nodeId: rootID,
    hasChildrenHint: true,
    duplicate: false,
    expanded: true,
    loading: true,
    error: "",
    children: null
  }) as DeviceTreeNode
  state.root = root
  registerNode(root)
  seenNodeIDs.add(rootID)

  try {
    const resp =
      state.mode === "subtree"
        ? await callMgmt<ListNodesWire>("ListSubtreeSimple", sourceID, rootID)
        : await callMgmt<ListNodesWire>("ListNodesSimple", sourceID, rootID)
    if (epoch !== myEpoch) return
    const children = normalizeNodes(resp).filter((info) => info.nodeId !== rootID)
    root.children = buildChildNodes(root.key, children, rootID)
    root.hasChildrenHint = root.children.length > 0
    root.loading = false
    state.message =
      state.mode === "subtree"
        ? "Root loaded (subtree is direct + self; not recursive)."
        : "Root loaded."
  } catch (err) {
    if (epoch !== myEpoch) return
    root.loading = false
    root.error = toErrorMessage(err)
    state.message = root.error
    throw err
  }
}

const toggle = async (key: string) => {
  const node = nodeIndex.get(key)
  if (!node) return
  if (node.duplicate) return
  if (node.expanded) {
    node.expanded = false
    return
  }
  node.expanded = true
  if (node.children !== null) {
    return
  }
  let sourceID = 0
  try {
    const identity = ensureIdentity()
    sourceID = identity.sourceID
  } catch (err) {
    node.expanded = false
    node.error = toErrorMessage(err)
    state.message = node.error
    return
  }
  await loadChildren(node, sourceID, epoch, false)
}

const retry = async (key: string) => {
  const node = nodeIndex.get(key)
  if (!node) return
  if (node.duplicate) return
  node.expanded = true
  let sourceID = 0
  try {
    const identity = ensureIdentity()
    sourceID = identity.sourceID
  } catch (err) {
    node.error = toErrorMessage(err)
    state.message = node.error
    return
  }
  await loadChildren(node, sourceID, epoch, true)
}

export const useDevicesStore = () => {
  return {
    state,
    loadRoot,
    toggle,
    retry,
    reset: () => {
      epoch++
      clearRuntimeTree()
      state.mode = "direct"
      state.rootTargetId = ""
    }
  }
}

