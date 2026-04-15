// 本文件维护 `registrationApprovals` store，并让它与 Wails 绑定及共享前端状态保持同步。

import { reactive } from "vue"
import { callPermission, useAuthorityStore } from "@/stores/authority"

export type PendingRegister = {
  requestId: string
  deviceId: string
  requestedRole: string
  displayName: string
  createdAt: number
  expiresAt: number
}

type PendingListResp = {
  authorityId: number
  total: number
  items: PendingRegister[]
}

type ApproveResp = {
  requestId: string
  deviceId?: string
  nodeId?: number
  role?: string
  status?: string
}

type RejectResp = {
  requestId: string
  deviceId?: string
  status?: string
  reason?: string
}

type LastDecision = {
  action: "approve" | "reject"
  requestId: string
  deviceId: string
  status: string
  nodeId: number
  role: string
  reason: string
}

type RegistrationApprovalsState = {
  loading: boolean
  busyRequestId: string
  filterDeviceId: string
  total: number
  items: PendingRegister[]
  lastDecision?: LastDecision
}

const authority = useAuthorityStore()

const state = reactive<RegistrationApprovalsState>({
  loading: false,
  busyRequestId: "",
  filterDeviceId: "",
  total: 0,
  items: [],
  lastDecision: undefined
})

const applyPendingList = (resp: PendingListResp | undefined) => {
  state.total = Number(resp?.total || 0)
  state.items = Array.isArray(resp?.items) ? resp.items : []
}

const reset = () => {
  state.loading = false
  state.busyRequestId = ""
  state.filterDeviceId = ""
  state.total = 0
  state.items = []
  state.lastDecision = undefined
}

const loadPending = async () => {
  const { sourceId, authorityId } = await authority.requireAuthority()
  state.loading = true
  try {
    const resp = await callPermission<PendingListResp>("ListPendingRegisters", {
      sourceId,
      authorityId,
      offset: 0,
      limit: 200,
      deviceId: state.filterDeviceId.trim()
    })
    applyPendingList(resp)
  } finally {
    state.loading = false
  }
}

const approveRegister = async (requestId: string, role: string) => {
  const { sourceId, authorityId } = await authority.requireAuthority()
  state.busyRequestId = requestId
  try {
    const resp = await callPermission<ApproveResp>("ApproveRegister", {
      sourceId,
      authorityId,
      requestId,
      role
    })
    state.lastDecision = {
      action: "approve",
      requestId: String(resp?.requestId || requestId),
      deviceId: String(resp?.deviceId || ""),
      status: String(resp?.status || ""),
      nodeId: Number(resp?.nodeId || 0),
      role: String(resp?.role || ""),
      reason: ""
    }
    await loadPending()
    return resp
  } finally {
    state.busyRequestId = ""
  }
}

const rejectRegister = async (requestId: string, reason: string) => {
  const { sourceId, authorityId } = await authority.requireAuthority()
  state.busyRequestId = requestId
  try {
    const resp = await callPermission<RejectResp>("RejectRegister", {
      sourceId,
      authorityId,
      requestId,
      reason
    })
    state.lastDecision = {
      action: "reject",
      requestId: String(resp?.requestId || requestId),
      deviceId: String(resp?.deviceId || ""),
      status: String(resp?.status || ""),
      nodeId: 0,
      role: "",
      reason: String(resp?.reason || reason || "")
    }
    await loadPending()
    return resp
  } finally {
    state.busyRequestId = ""
  }
}

export const useRegistrationApprovalsStore = () => {
  return {
    state,
    loadPending,
    approveRegister,
    rejectRegister,
    reset
  }
}
