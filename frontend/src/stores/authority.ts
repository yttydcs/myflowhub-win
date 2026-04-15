// 本文件维护 `authority` store，并让它与 Wails 绑定及共享前端状态保持同步。

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

// 所有权限相关请求都依赖当前登录节点和 hub，先在 store 层统一做前置校验。
const ensureIdentity = () => {
  if (!state.sourceId) {
    throw new Error(t("Login required."))
  }
  if (!state.hubId) {
    throw new Error(t("Hub ID missing."))
  }
  return { sourceId: state.sourceId, hubId: state.hubId }
}

// authority 会跟着当前身份变化，这里按需向后端解析一次并缓存，供多个页面共用。
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

// 页面只关心“拿到可用 authority”这一结果，因此把懒加载解析和失败兜底都收敛到这个入口。
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

// 身份一旦切换，旧 authority 结果就不再可信，所以这里顺手清掉缓存。
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
