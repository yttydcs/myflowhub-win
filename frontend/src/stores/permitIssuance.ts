import { reactive } from "vue"
import { callPermission, useAuthorityStore } from "@/stores/authority"

export type IssuedPermit = {
  permit: string
  deviceId: string
  role: string
  expiresAt: number
  issuedAt: string
  revoked: boolean
}

type PermitIssueResp = {
  permit?: string
  deviceId?: string
  role?: string
  expiresAt?: number
}

type PermitRevokeResp = {
  permit?: string
  deviceId?: string
  role?: string
}

type LastRevoke = {
  permit: string
  deviceId: string
  role: string
  revokedAt: string
}

type PermitIssuanceState = {
  issuing: boolean
  revoking: boolean
  lastIssued?: IssuedPermit
  lastRevoke?: LastRevoke
}

const authority = useAuthorityStore()

const state = reactive<PermitIssuanceState>({
  issuing: false,
  revoking: false,
  lastIssued: undefined,
  lastRevoke: undefined
})

const nowIso = () => new Date().toISOString()

const reset = () => {
  state.issuing = false
  state.revoking = false
  state.lastIssued = undefined
  state.lastRevoke = undefined
}

const issuePermit = async (input: { deviceId: string; role: string; expiresAt: number }) => {
  const { sourceId, authorityId } = await authority.requireAuthority()
  state.issuing = true
  try {
    const resp = await callPermission<PermitIssueResp>("IssueRegisterPermit", {
      sourceId,
      authorityId,
      deviceId: input.deviceId,
      role: input.role,
      expiresAt: input.expiresAt
    })
    state.lastIssued = {
      permit: String(resp?.permit || ""),
      deviceId: String(resp?.deviceId || input.deviceId),
      role: String(resp?.role || input.role),
      expiresAt: Number(resp?.expiresAt || 0),
      issuedAt: nowIso(),
      revoked: false
    }
    return state.lastIssued
  } finally {
    state.issuing = false
  }
}

const revokePermit = async (permit: string) => {
  const { sourceId, authorityId } = await authority.requireAuthority()
  state.revoking = true
  try {
    const resp = await callPermission<PermitRevokeResp>("RevokeRegisterPermit", {
      sourceId,
      authorityId,
      permit
    })
    state.lastRevoke = {
      permit: String(resp?.permit || permit),
      deviceId: String(resp?.deviceId || ""),
      role: String(resp?.role || ""),
      revokedAt: nowIso()
    }
    if (state.lastIssued && state.lastIssued.permit === state.lastRevoke.permit) {
      state.lastIssued.revoked = true
    }
    return state.lastRevoke
  } finally {
    state.revoking = false
  }
}

export const usePermitIssuanceStore = () => {
  return {
    state,
    issuePermit,
    revokePermit,
    reset
  }
}
