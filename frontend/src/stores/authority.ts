import { reactive } from "vue"
import { t } from "@/i18n"

type WailsBinding = (...args: any[]) => Promise<any>

type ResolveAuthorityResp = {
  authorityId: number
  reason: string
}

export type AuthorityState = {
  sourceId: number
  hubId: number
  authorityOverride: string
  authorityId: number
  authorityReason: string
  resolving: boolean
}

export const callPermission = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.permission?.PermissionService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(t("Permission binding '{method}' unavailable", { method }))
  }
  return fn(...args)
}

const state = reactive<AuthorityState>({
  sourceId: 0,
  hubId: 0,
  authorityOverride: "",
  authorityId: 0,
  authorityReason: "",
  resolving: false
})

const ensureIdentity = () => {
  if (!state.sourceId) {
    throw new Error(t("Login required."))
  }
  if (!state.hubId) {
    throw new Error(t("Hub ID missing."))
  }
  return { sourceId: state.sourceId, hubId: state.hubId }
}

const parseOverride = () => {
  const raw = state.authorityOverride.trim()
  if (!raw) return 0
  const parsed = Number.parseInt(raw, 10)
  if (Number.isNaN(parsed) || parsed <= 0) {
    throw new Error(t("Authority override must be a positive number."))
  }
  return parsed
}

const resolveAuthority = async () => {
  const { sourceId, hubId } = ensureIdentity()
  const overrideId = parseOverride()
  state.resolving = true
  try {
    const resp = await callPermission<ResolveAuthorityResp>("ResolveAuthority", sourceId, hubId, overrideId)
    state.authorityId = Number(resp?.authorityId || 0)
    state.authorityReason = String(resp?.reason || "")
    if (!state.authorityOverride && state.authorityId) {
      state.authorityOverride = String(state.authorityId)
    }
    return state.authorityId
  } finally {
    state.resolving = false
  }
}

const requireAuthority = async () => {
  const { sourceId, hubId } = ensureIdentity()
  if (!state.authorityId) {
    await resolveAuthority()
  }
  if (!state.authorityId) {
    throw new Error(t("Authority ID unresolved."))
  }
  return {
    sourceId,
    hubId,
    authorityId: state.authorityId
  }
}

const setIdentity = (sourceId: number, hubId: number) => {
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
  }

  if (!overrideRaw || overrideMatchesPrevHub) {
    state.authorityOverride = state.hubId ? String(state.hubId) : ""
  }
}

const reset = () => {
  state.sourceId = 0
  state.hubId = 0
  state.authorityOverride = ""
  state.authorityId = 0
  state.authorityReason = ""
  state.resolving = false
}

export const useAuthorityStore = () => {
  return {
    state,
    ensureIdentity,
    resolveAuthority,
    requireAuthority,
    setIdentity,
    reset
  }
}
