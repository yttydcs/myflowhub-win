import { t } from "@/i18n"
import { reactive } from "vue"

type WailsBinding = (...args: any[]) => Promise<any>

const callMgmt = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.management?.ManagementService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(t("Management binding '{method}' unavailable", { method }))
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

const syntheticConfigKeys = ["node.display_name"] as const

type MgmtNodeWire = {
  node_id: number
  has_children?: boolean
}

type MgmtListNodesResp = {
  code: number
  msg?: string
  nodes?: MgmtNodeWire[]
}

type MgmtConfigListResp = {
  code: number
  msg?: string
  keys?: string[]
}

type MgmtConfigResp = {
  code: number
  msg?: string
  key?: string
  value?: string
}

type MgmtState = {
  targetId: string
  selfNodeId: number
  hubId: number
  listMode: "direct" | "subtree"
  nodes: MgmtNode[]
  selectedNodeId: number
  configEntries: MgmtConfigEntry[]
}

const state = reactive<MgmtState>({
  targetId: "",
  selfNodeId: 0,
  hubId: 0,
  listMode: "direct",
  nodes: [],
  selectedNodeId: 0,
  configEntries: []
})

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

const ensureIdentity = () => {
  if (!state.selfNodeId) {
    throw new Error(t("Login required to send Management requests."))
  }
  if (!state.hubId) {
    throw new Error(t("Hub ID missing."))
  }
  return { sourceID: state.selfNodeId, hubID: state.hubId }
}

const applyListResp = (resp: MgmtListNodesResp, mode: "direct" | "subtree") => {
  const nodes = Array.isArray(resp?.nodes) ? resp.nodes : []
  const list = nodes
    .map((node) => ({
      nodeId: Number(node?.node_id ?? 0),
      hasChildren: Boolean(node?.has_children ?? false)
    }))
    .filter((node: MgmtNode) => node.nodeId > 0)
    .sort((a: MgmtNode, b: MgmtNode) => a.nodeId - b.nodeId)
  const target = Number.parseInt(state.targetId || "0", 10)
  if (target && list.some((node) => node.nodeId === target)) {
    const current = list.find((node) => node.nodeId === target)
    const rest = list.filter((node) => node.nodeId !== target)
    state.nodes = current ? [current, ...rest] : list
  } else {
    state.nodes = list
  }
  state.listMode = mode
}

const applyConfigResp = (resp: MgmtConfigResp, fallbackKey: string) => {
  const key = String(resp?.key ?? fallbackKey ?? "").trim()
  if (!key) return
  const value = String(resp?.value ?? "")
  const idx = state.configEntries.findIndex((entry) => entry.key === key)
  if (idx >= 0) {
    state.configEntries[idx].value = value
  } else {
    state.configEntries.push({ key, value })
  }
}

const buildConfigEntries = (keys: unknown[]) => {
  const entries: MgmtConfigEntry[] = []
  const seen = new Set<string>()
  const pushEntry = (rawKey: unknown) => {
    const key = String(rawKey ?? "").trim()
    if (!key || seen.has(key)) return
    seen.add(key)
    entries.push({ key, value: "" })
  }

  for (const key of syntheticConfigKeys) {
    pushEntry(key)
  }
  for (const key of keys) {
    pushEntry(key)
  }

  return entries
}

const listNodes = async () => {
  const { sourceID } = ensureIdentity()
  const targetID = resolveTargetNode()
  const resp = await callMgmt<MgmtListNodesResp>("ListNodesSimple", sourceID, targetID)
  applyListResp(resp, "direct")
}

const listSubtree = async () => {
  const { sourceID } = ensureIdentity()
  const targetID = resolveTargetNode()
  const resp = await callMgmt<MgmtListNodesResp>("ListSubtreeSimple", sourceID, targetID)
  applyListResp(resp, "subtree")
}

const loadConfigValue = async (sourceID: number, nodeId: number, key: string) => {
  try {
    const resp = await callMgmt<MgmtConfigResp>("ConfigGetSimple", sourceID, nodeId, key)
    if (state.selectedNodeId !== nodeId) return
    applyConfigResp(resp, key)
  } catch {
    // ignore per-key errors to keep UI responsive
  }
}

const loadConfigKeys = async (sourceID: number, nodeId: number) => {
  const resp = await callMgmt<MgmtConfigListResp>("ConfigListSimple", sourceID, nodeId)
  if (state.selectedNodeId !== nodeId) return
  const keys = Array.isArray(resp?.keys) ? resp.keys : []
  state.configEntries = buildConfigEntries(keys)
  if (!state.selfNodeId) {
    return
  }
  const caller = state.selfNodeId
  for (const entry of state.configEntries) {
    void loadConfigValue(caller, nodeId, entry.key)
  }
}

const selectNode = async (nodeId: number) => {
  state.selectedNodeId = nodeId
  state.configEntries = []
  if (!nodeId) return
  const { sourceID } = ensureIdentity()
  await loadConfigKeys(sourceID, nodeId)
}

const refreshConfig = async () => {
  if (!state.selectedNodeId) {
    throw new Error(t("Select a node to load config."))
  }
  const { sourceID } = ensureIdentity()
  const nodeId = state.selectedNodeId
  await loadConfigKeys(sourceID, nodeId)
}

const setConfig = async (key: string, value: string) => {
  const trimmed = key.trim()
  if (!trimmed) {
    throw new Error(t("Config key is required."))
  }
  if (!state.selectedNodeId) {
    throw new Error(t("Select a node to update config."))
  }
  const { sourceID } = ensureIdentity()
  const nodeId = state.selectedNodeId
  const resp = await callMgmt<MgmtConfigResp>("ConfigSetSimple", sourceID, nodeId, trimmed, value)
  if (state.selectedNodeId !== nodeId) return
  applyConfigResp(resp, trimmed)
}

export const useManagementStore = () => {
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
