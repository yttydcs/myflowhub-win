import { reactive } from "vue"
import { callPermission, useAuthorityStore } from "@/stores/authority"

export type RegisterPermit = {
  permit: string
  deviceId: string
  role: string
  issuedBy: number
  issuedAt: number
  expiresAt: number
}

type PermitListResp = {
  authorityId?: number
  total?: number
  items?: RegisterPermit[]
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

type PermitIssuanceState = {
  loading: boolean
  issuing: boolean
  busyPermit: string
  total: number
  items: RegisterPermit[]
}

const authority = useAuthorityStore()

const state = reactive<PermitIssuanceState>({
  loading: false,
  issuing: false,
  busyPermit: "",
  total: 0,
  items: []
})

const applyPermitList = (resp?: PermitListResp) => {
  state.total = Number(resp?.total || 0)
  state.items = Array.isArray(resp?.items)
    ? resp.items
        .map((item) => ({
          permit: String(item?.permit || "").trim(),
          deviceId: String(item?.deviceId || "").trim(),
          role: String(item?.role || "").trim(),
          issuedBy: Number(item?.issuedBy || 0),
          issuedAt: Number(item?.issuedAt || 0),
          expiresAt: Number(item?.expiresAt || 0)
        }))
        .filter((item) => item.permit || item.deviceId)
    : []
}

const reset = () => {
  state.loading = false
  state.issuing = false
  state.busyPermit = ""
  state.total = 0
  state.items = []
}

const loadPermits = async (input?: { deviceId?: string }) => {
  const { sourceId, authorityId } = await authority.requireAuthority()
  state.loading = true
  try {
    const resp = await callPermission<PermitListResp>("ListRegisterPermits", {
      sourceId,
      authorityId,
      offset: 0,
      limit: 200,
      deviceId: String(input?.deviceId || "").trim()
    })
    applyPermitList(resp)
    return state.items
  } finally {
    state.loading = false
  }
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
    return {
      permit: String(resp?.permit || "").trim(),
      deviceId: String(resp?.deviceId || input.deviceId).trim(),
      role: String(resp?.role || input.role).trim(),
      expiresAt: Number(resp?.expiresAt || 0)
    }
  } finally {
    state.issuing = false
  }
}

const revokePermit = async (permit: string) => {
  const { sourceId, authorityId } = await authority.requireAuthority()
  state.busyPermit = permit
  try {
    const resp = await callPermission<PermitRevokeResp>("RevokeRegisterPermit", {
      sourceId,
      authorityId,
      permit
    })
    return {
      permit: String(resp?.permit || permit).trim(),
      deviceId: String(resp?.deviceId || "").trim(),
      role: String(resp?.role || "").trim()
    }
  } finally {
    state.busyPermit = ""
  }
}

export const usePermitIssuanceStore = () => {
  return {
    state,
    loadPermits,
    issuePermit,
    revokePermit,
    reset
  }
}
