// 本文件维护 `accessPolicy` store，并让它与 Wails 绑定及共享前端状态保持同步。

import { reactive } from "vue"
import { t } from "@/i18n"
import { callPermission, useAuthorityStore } from "@/stores/authority"

export type NodeRole = {
  nodeId: number
  role: string
}

export type RolePerm = {
  role: string
  perms: string[]
}

export type Policy = {
  defaultRole: string
  defaultPerms: string[]
  nodeRoles: NodeRole[]
  rolePerms: RolePerm[]
}

export type RuntimeRole = {
  nodeId: number
  role: string
  perms: string[]
}

type LoadPolicyResp = {
  authorityId: number
  policy: Policy
  runtime: RuntimeRole[]
  runtimeTotal: number
  runtimeError?: string
  warnings?: string[]
}

type SavePolicyReq = {
  sourceId: number
  authorityId: number
  policy: Policy
  persist: boolean
  applyRuntime: boolean
  invalidate: boolean
  refresh: boolean
  verifyRuntime: boolean
}

type SavePolicyResp = {
  success: boolean
  errorStage?: string
  errorMessage?: string
  persisted: boolean
  applied: boolean
  invalidated: boolean
  authorityId: number
  policy: Policy
  warnings?: string[]
  runtime?: RuntimeRole[]
  runtimeTotal?: number
  runtimeError?: string
}

type NodePermsResp = {
  nodeId: number
  role: string
  perms: string[]
}

type AccessPolicyState = {
  loading: boolean
  saving: boolean
  policy: Policy
  runtime: RuntimeRole[]
  runtimeTotal: number
  runtimeError: string
  warnings: string[]
  lastSave?: SavePolicyResp
}

const emptyPolicy = (): Policy => ({
  defaultRole: "node",
  defaultPerms: [],
  nodeRoles: [],
  rolePerms: []
})

const state = reactive<AccessPolicyState>({
  loading: false,
  saving: false,
  policy: emptyPolicy(),
  runtime: [],
  runtimeTotal: 0,
  runtimeError: "",
  warnings: [],
  lastSave: undefined
})

const authority = useAuthorityStore()

const setPolicy = (policy: Policy | undefined) => {
  const next = policy || emptyPolicy()
  state.policy.defaultRole = String(next.defaultRole || "node")
  state.policy.defaultPerms = Array.isArray(next.defaultPerms) ? [...next.defaultPerms] : []
  state.policy.nodeRoles = Array.isArray(next.nodeRoles)
    ? next.nodeRoles.map((entry) => ({
        nodeId: Number(entry?.nodeId || 0),
        role: String(entry?.role || "")
      }))
    : []
  state.policy.rolePerms = Array.isArray(next.rolePerms)
    ? next.rolePerms.map((entry) => ({
        role: String(entry?.role || ""),
        perms: Array.isArray(entry?.perms) ? entry.perms.map((item) => String(item || "")) : []
      }))
    : []
}

const reset = () => {
  setPolicy(undefined)
  state.loading = false
  state.saving = false
  state.runtime = []
  state.runtimeTotal = 0
  state.runtimeError = ""
  state.warnings = []
  state.lastSave = undefined
}

const loadPolicy = async () => {
  const { sourceId, authorityId } = await authority.requireAuthority()
  state.loading = true
  try {
    const resp = await callPermission<LoadPolicyResp>("LoadPolicy", sourceId, authorityId)
    state.runtime = Array.isArray(resp?.runtime) ? resp.runtime : []
    state.runtimeTotal = Number(resp?.runtimeTotal || 0)
    state.runtimeError = String(resp?.runtimeError || "")
    state.warnings = Array.isArray(resp?.warnings) ? resp.warnings.map((item) => String(item || "")) : []
    setPolicy(resp?.policy)
  } finally {
    state.loading = false
  }
}

const savePolicy = async (options: {
  persist: boolean
  applyRuntime: boolean
  invalidate: boolean
  refresh: boolean
  verifyRuntime: boolean
}) => {
  const { sourceId, authorityId } = await authority.requireAuthority()
  const req: SavePolicyReq = {
    sourceId,
    authorityId,
    policy: {
      defaultRole: state.policy.defaultRole,
      defaultPerms: [...state.policy.defaultPerms],
      nodeRoles: state.policy.nodeRoles.map((entry) => ({
        nodeId: Number(entry.nodeId || 0),
        role: entry.role
      })),
      rolePerms: state.policy.rolePerms.map((entry) => ({
        role: entry.role,
        perms: [...entry.perms]
      }))
    },
    persist: Boolean(options.persist),
    applyRuntime: Boolean(options.applyRuntime),
    invalidate: Boolean(options.invalidate),
    refresh: Boolean(options.refresh),
    verifyRuntime: Boolean(options.verifyRuntime)
  }

  state.saving = true
  try {
    const resp = await callPermission<SavePolicyResp>("SavePolicy", req)
    state.lastSave = resp
    state.runtime = Array.isArray(resp?.runtime) ? resp.runtime : []
    state.runtimeTotal = Number(resp?.runtimeTotal || 0)
    state.runtimeError = String(resp?.runtimeError || "")
    state.warnings = Array.isArray(resp?.warnings) ? resp.warnings.map((item) => String(item || "")) : []
    setPolicy(resp?.policy)
    if (!resp?.success) {
      throw new Error(
        resp?.errorMessage ||
          t("Save failed at stage {stage}.", { stage: String(resp?.errorStage || "unknown") })
      )
    }
    return resp
  } finally {
    state.saving = false
  }
}

const getNodePerms = async (nodeId: number) => {
  const { sourceId, authorityId } = await authority.requireAuthority()
  const parsed = Number(nodeId || 0)
  if (!Number.isFinite(parsed) || parsed <= 0) {
    throw new Error(t("Node ID must be a positive number."))
  }
  const resp = await callPermission<NodePermsResp>("GetNodePerms", sourceId, authorityId, parsed)
  return {
    nodeId: Number(resp?.nodeId || 0),
    role: String(resp?.role || ""),
    perms: Array.isArray(resp?.perms) ? resp.perms.map((item) => String(item || "")) : []
  }
}

export const useAccessPolicyStore = () => {
  return {
    state,
    loadPolicy,
    savePolicy,
    getNodePerms,
    reset
  }
}
