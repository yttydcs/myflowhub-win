// Context: covers the authority admin store behavior in the Win frontend.

// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from "vitest"
import { setLocale } from "@/i18n"
import { useAccessPolicyStore } from "./accessPolicy"
import { useAuthorityStore } from "./authority"
import { usePermitIssuanceStore } from "./permitIssuance"
import { useRegistrationApprovalsStore } from "./registrationApprovals"

const resolveAuthorityMock = vi.fn()
const loadPolicyMock = vi.fn()
const savePolicyMock = vi.fn()
const getNodePermsMock = vi.fn()
const listPendingRegistersMock = vi.fn()
const listRegisterPermitsMock = vi.fn()
const approveRegisterMock = vi.fn()
const rejectRegisterMock = vi.fn()
const issueRegisterPermitMock = vi.fn()
const revokeRegisterPermitMock = vi.fn()

const authority = useAuthorityStore()
const accessPolicy = useAccessPolicyStore()
const approvals = useRegistrationApprovalsStore()
const permits = usePermitIssuanceStore()

beforeEach(() => {
  setLocale("en")
  resolveAuthorityMock.mockReset()
  loadPolicyMock.mockReset()
  savePolicyMock.mockReset()
  getNodePermsMock.mockReset()
  listPendingRegistersMock.mockReset()
  listRegisterPermitsMock.mockReset()
  approveRegisterMock.mockReset()
  rejectRegisterMock.mockReset()
  issueRegisterPermitMock.mockReset()
  revokeRegisterPermitMock.mockReset()

  ;(window as any).go = {
    permission: {
      PermissionService: {
        ResolveAuthority: resolveAuthorityMock,
        LoadPolicy: loadPolicyMock,
        SavePolicy: savePolicyMock,
        GetNodePerms: getNodePermsMock,
        ListPendingRegisters: listPendingRegistersMock,
        ListRegisterPermits: listRegisterPermitsMock,
        ApproveRegister: approveRegisterMock,
        RejectRegister: rejectRegisterMock,
        IssueRegisterPermit: issueRegisterPermitMock,
        RevokeRegisterPermit: revokeRegisterPermitMock
      }
    }
  }

  authority.reset()
  accessPolicy.reset()
  approvals.reset()
  permits.reset()
})

describe("authority admin stores", () => {
  it("resolves authority using the shared authority store", async () => {
    authority.setIdentity(7, 9)
    resolveAuthorityMock.mockResolvedValue({
      authorityId: 11
    })

    const authorityId = await authority.resolveAuthority()

    expect(resolveAuthorityMock).toHaveBeenCalledWith(7, 9, 0)
    expect(authorityId).toBe(11)
    expect(authority.state.authorityId).toBe(11)
  })

  it("loads access policy through the resolved authority", async () => {
    authority.setIdentity(7, 9)
    authority.state.authorityId = 11

    loadPolicyMock.mockResolvedValue({
      authorityId: 11,
      policy: {
        defaultRole: "admin",
        defaultPerms: ["file.read", "file.write"],
        nodeRoles: [{ nodeId: 3, role: "observer" }],
        rolePerms: [{ role: "admin", perms: ["*"] }]
      },
      runtime: [{ nodeId: 3, role: "observer", perms: ["file.read"] }],
      runtimeTotal: 1
    })

    await accessPolicy.loadPolicy()

    expect(loadPolicyMock).toHaveBeenCalledWith(7, 11)
    expect(accessPolicy.state.policy.defaultRole).toBe("admin")
    expect(accessPolicy.state.runtimeTotal).toBe(1)
  })

  it("refreshes the pending queue after approve", async () => {
    authority.setIdentity(7, 9)
    authority.state.authorityId = 11

    listPendingRegistersMock
      .mockResolvedValueOnce({
        authorityId: 11,
        total: 1,
        items: [
          {
            requestId: "req-1",
            deviceId: "device-1",
            requestedRole: "node",
            displayName: "Node A",
            createdAt: 10,
            expiresAt: 20
          }
        ]
      })
      .mockResolvedValueOnce({
        authorityId: 11,
        total: 0,
        items: []
      })
    approveRegisterMock.mockResolvedValue({
      requestId: "req-1",
      deviceId: "device-1",
      nodeId: 21,
      role: "admin",
      status: "approved"
    })

    await approvals.loadPending()
    await approvals.approveRegister("req-1", "admin")

    expect(approveRegisterMock).toHaveBeenCalledWith({
      sourceId: 7,
      authorityId: 11,
      requestId: "req-1",
      role: "admin"
    })
    expect(listPendingRegistersMock).toHaveBeenCalledTimes(2)
    expect(approvals.state.total).toBe(0)
    expect(approvals.state.lastDecision).toMatchObject({
      action: "approve",
      requestId: "req-1",
      nodeId: 21
    })
  })

  it("reloads the active permit list after issue and revoke", async () => {
    authority.setIdentity(7, 9)
    authority.state.authorityId = 11

    listRegisterPermitsMock
      .mockResolvedValueOnce({
        authorityId: 11,
        total: 1,
        items: [
          {
            permit: "permit_123",
            deviceId: "device-1",
            role: "admin",
            issuedBy: 9,
            issuedAt: 100,
            expiresAt: 200
          }
        ]
      })
      .mockResolvedValueOnce({
        authorityId: 11,
        total: 2,
        items: [
          {
            permit: "permit_456",
            deviceId: "device-2",
            role: "observer",
            issuedBy: 9,
            issuedAt: 150,
            expiresAt: 250
          },
          {
            permit: "permit_123",
            deviceId: "device-1",
            role: "admin",
            issuedBy: 9,
            issuedAt: 100,
            expiresAt: 200
          }
        ]
      })
      .mockResolvedValueOnce({
        authorityId: 11,
        total: 1,
        items: [
          {
            permit: "permit_123",
            deviceId: "device-1",
            role: "admin",
            issuedBy: 9,
            issuedAt: 100,
            expiresAt: 200
          }
        ]
      })
    issueRegisterPermitMock.mockResolvedValue({
      permit: "permit_456",
      deviceId: "device-2",
      role: "observer",
      expiresAt: 23456
    })
    revokeRegisterPermitMock.mockResolvedValue({
      permit: "permit_456",
      deviceId: "device-2",
      role: "observer"
    })

    await permits.loadPermits()
    const issued = await permits.issuePermit({
      deviceId: "device-2",
      role: "observer",
      expiresAt: 23456
    })
    await permits.loadPermits()
    const revoked = await permits.revokePermit("permit_456")
    await permits.loadPermits()

    expect(issued.permit).toBe("permit_456")
    expect(revoked.permit).toBe("permit_456")
    expect(listRegisterPermitsMock).toHaveBeenCalledTimes(3)
    expect(permits.state.total).toBe(1)
    expect(permits.state.items).toHaveLength(1)
    expect(permits.state.items[0]?.permit).toBe("permit_123")
  })
})
