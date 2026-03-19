import { reactive } from "vue"

type WailsBinding = (...args: any[]) => Promise<any>

const callPermission = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.permission?.PermissionService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(`Permission binding '${method}' unavailable`)
  }
  return fn(...args)
}

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

type ResolveAuthorityResp = {
  authorityId: number
  reason: string
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

type PermissionState = {
  sourceId: number
  hubId: number
  authorityOverride: string
  authorityId: number
  authorityReason: string
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

const state = reactive<PermissionState>({
  sourceId: 0,
  hubId: 0,
  authorityOverride: "",
  authorityId: 0,
  authorityReason: "",
  loading: false,
  saving: false,
  policy: emptyPolicy(),
  runtime: [],
  runtimeTotal: 0,
  runtimeError: "",
  warnings: [],
  lastSave: undefined
})

const ensureIdentity = () => {
  if (!state.sourceId) {
    throw new Error("Login required.")
  }
  if (!state.hubId) {
    throw new Error("Hub ID missing.")
  }
  return { sourceId: state.sourceId, hubId: state.hubId }
}

const parseOverride = () => {
  const raw = state.authorityOverride.trim()
  if (!raw) return 0
  const parsed = Number.parseInt(raw, 10)
  if (Number.isNaN(parsed) || parsed <= 0) {
    throw new Error("Authority override must be a positive number.")
  }
  return parsed
}

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

const resolveAuthority = async () => {
  const { sourceId, hubId } = ensureIdentity()
  const overrideId = parseOverride()
  const resp = await callPermission<ResolveAuthorityResp>("ResolveAuthority", sourceId, hubId, overrideId)
  state.authorityId = Number(resp?.authorityId || 0)
  state.authorityReason = String(resp?.reason || "")
  if (!state.authorityOverride && state.authorityId) {
    state.authorityOverride = String(state.authorityId)
  }
  return state.authorityId
}

const loadPolicy = async () => {
  const { sourceId } = ensureIdentity()
  if (!state.authorityId) {
    await resolveAuthority()
  }
  if (!state.authorityId) {
    throw new Error("Authority ID unresolved.")
  }
  state.loading = true
  try {
    const resp = await callPermission<LoadPolicyResp>("LoadPolicy", sourceId, state.authorityId)
    state.authorityId = Number(resp?.authorityId || state.authorityId)
    setPolicy(resp?.policy)
    state.runtime = Array.isArray(resp?.runtime) ? resp.runtime : []
    state.runtimeTotal = Number(resp?.runtimeTotal || 0)
    state.runtimeError = String(resp?.runtimeError || "")
    state.warnings = Array.isArray(resp?.warnings) ? resp.warnings.map((item) => String(item || "")) : []
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
  const { sourceId } = ensureIdentity()
  if (!state.authorityId) {
    throw new Error("Authority ID unresolved.")
  }
  const req: SavePolicyReq = {
    sourceId,
    authorityId: state.authorityId,
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
    setPolicy(resp?.policy)
    state.warnings = Array.isArray(resp?.warnings) ? resp.warnings.map((item) => String(item || "")) : []
    state.runtime = Array.isArray(resp?.runtime) ? resp.runtime : []
    state.runtimeTotal = Number(resp?.runtimeTotal || 0)
    state.runtimeError = String(resp?.runtimeError || "")
    if (!resp?.success) {
      throw new Error(resp?.errorMessage || `Save failed at stage ${resp?.errorStage || "unknown"}.`)
    }
    return resp
  } finally {
    state.saving = false
  }
}

const getNodePerms = async (nodeId: number) => {
  const { sourceId } = ensureIdentity()
  if (!state.authorityId) {
    throw new Error("Authority ID unresolved.")
  }
  const parsed = Number(nodeId || 0)
  if (!Number.isFinite(parsed) || parsed <= 0) {
    throw new Error("Node ID must be a positive number.")
  }
  const resp = await callPermission<NodePermsResp>("GetNodePerms", sourceId, state.authorityId, parsed)
  return {
    nodeId: Number(resp?.nodeId || 0),
    role: String(resp?.role || ""),
    perms: Array.isArray(resp?.perms) ? resp.perms.map((item) => String(item || "")) : []
  }
}

export const usePermissionsStore = () => {
  return {
    state,
    setIdentity: (sourceId: number, hubId: number) => {
      const nextSourceId = Number(sourceId || 0)
      const nextHubId = Number(hubId || 0)
      const prevSourceId = state.sourceId
      const prevHubId = state.hubId
      const changed = prevSourceId !== nextSourceId || prevHubId !== nextHubId

      const overrideRaw = state.authorityOverride.trim()
      let overrideNumber = 0
      if (overrideRaw) {
        const parsed = Number.parseInt(overrideRaw, 10)
        if (!Number.isNaN(parsed) && parsed > 0) {
          overrideNumber = parsed
        }
      }
      const overrideMatchesPrevHub = overrideNumber > 0 && overrideNumber === prevHubId

      state.sourceId = nextSourceId
      state.hubId = nextHubId

      if (changed) {
        state.authorityId = 0
        state.authorityReason = ""
        state.runtime = []
        state.runtimeTotal = 0
        state.runtimeError = ""
        state.warnings = []
        state.lastSave = undefined
      }

      if (!overrideRaw || overrideMatchesPrevHub) {
        state.authorityOverride = state.hubId ? String(state.hubId) : ""
      }
    },
    resolveAuthority,
    loadPolicy,
    savePolicy,
    getNodePerms
  }
}
