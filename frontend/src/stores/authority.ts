// Context: keeps the authority store in sync with Wails bindings and shared Win frontend state.

import { reactive } from "vue"
import { t } from "@/i18n"

type WailsBinding = (...args: any[]) => Promise<any>

type ResolveAuthorityResp = {
  authorityId: number
}

export type AuthorityState = {
  sourceId: number
  hubId: number
  authorityId: number
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
  authorityId: 0,
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

const resolveAuthority = async () => {
  const { sourceId, hubId } = ensureIdentity()
  state.resolving = true
  try {
    const resp = await callPermission<ResolveAuthorityResp>("ResolveAuthority", sourceId, hubId, 0)
    state.authorityId = Number(resp?.authorityId || 0)
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
  const changed = state.sourceId !== nextSourceId || state.hubId !== nextHubId

  state.sourceId = nextSourceId
  state.hubId = nextHubId

  if (changed) {
    state.authorityId = 0
  }
}

const reset = () => {
  state.sourceId = 0
  state.hubId = 0
  state.authorityId = 0
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
