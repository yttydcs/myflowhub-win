// Context: keeps the access policy catalog store in sync with Wails bindings and shared Win frontend state.

export type PermissionCatalogItem = {
  perm: string
  label: string
  description: string
}

export type PermissionCatalogGroup = {
  id: string
  label: string
  description: string
  items: PermissionCatalogItem[]
}

export type PermissionCatalogOption = PermissionCatalogItem & {
  groupId: string
  groupLabel: string
  groupDescription: string
}

const adminPerms = [
  "file.read",
  "file.write",
  "flow.set",
  "flow.delete",
  "exec.call",
  "exec.cap.query",
  "exec.cap.sync",
  "var.private_set",
  "var.revoke",
  "var.subscribe",
  "auth.revoke",
  "auth.pending.list",
  "auth.register.approve",
  "auth.register.reject",
  "auth.permit.issue",
  "auth.permit.revoke"
]

const nodePerms = [
  "file.read",
  "file.write",
  "flow.set",
  "exec.call",
  "exec.cap.query",
  "exec.cap.sync"
]

export const accessPolicyPermissionCatalog: PermissionCatalogGroup[] = [
  {
    id: "global",
    label: "Global",
    description: "Global wildcard access should be granted only to trusted super-admin roles.",
    items: [
      {
        perm: "*",
        label: "*",
        description: "Grants every permission without needing individual selections."
      }
    ]
  },
  {
    id: "file",
    label: "File",
    description: "Browse, read, upload, and mutate remote files and directories.",
    items: [
      {
        perm: "file.read",
        label: "file.read",
        description: "Allows file listing, metadata reads, and text or payload reads."
      },
      {
        perm: "file.write",
        label: "file.write",
        description: "Allows file offers, directory creation, overwrite, and delete-style writes."
      }
    ]
  },
  {
    id: "flow",
    label: "Flow",
    description: "Create and remove workflow deployments on executor nodes.",
    items: [
      {
        perm: "flow.set",
        label: "flow.set",
        description: "Allows creating or updating flow deployments."
      },
      {
        perm: "flow.delete",
        label: "flow.delete",
        description: "Allows deleting existing flow deployments."
      }
    ]
  },
  {
    id: "exec",
    label: "Exec",
    description: "Call remote methods and inspect the capability registry.",
    items: [
      {
        perm: "exec.call",
        label: "exec.call",
        description: "Allows invoking registered remote capabilities."
      },
      {
        perm: "exec.cap.query",
        label: "exec.cap.query",
        description: "Allows querying the aggregated capability index."
      },
      {
        perm: "exec.cap.sync",
        label: "exec.cap.sync",
        description: "Allows syncing capability snapshots and lifecycle updates."
      }
    ]
  },
  {
    id: "var",
    label: "Var",
    description: "Override private variable access and subscription behavior.",
    items: [
      {
        perm: "var.private_set",
        label: "var.private_set",
        description: "Allows writing private variables outside the owner path."
      },
      {
        perm: "var.revoke",
        label: "var.revoke",
        description: "Allows revoking variable access or ownership exceptions."
      },
      {
        perm: "var.subscribe",
        label: "var.subscribe",
        description: "Allows subscribing to private variables as a permission exception."
      }
    ]
  },
  {
    id: "auth",
    label: "Auth",
    description: "Manage revocation, pending approvals, and permit issuance.",
    items: [
      {
        perm: "auth.revoke",
        label: "auth.revoke",
        description: "Allows revoking an existing registered node identity."
      },
      {
        perm: "auth.pending.list",
        label: "auth.pending.list",
        description: "Allows reading the pending first-register approval queue."
      },
      {
        perm: "auth.register.approve",
        label: "auth.register.approve",
        description: "Allows approving a pending register request."
      },
      {
        perm: "auth.register.reject",
        label: "auth.register.reject",
        description: "Allows rejecting a pending register request."
      },
      {
        perm: "auth.permit.issue",
        label: "auth.permit.issue",
        description: "Allows issuing a one-time join permit for a device."
      },
      {
        perm: "auth.permit.revoke",
        label: "auth.permit.revoke",
        description: "Allows revoking an issued join permit."
      }
    ]
  }
]

export const accessPolicyRolePresets: Record<string, string[]> = {
  superadmin: ["*"],
  admin: adminPerms,
  node: nodePerms
}

const builtinRoleOrder = ["superadmin", "admin", "node"]

const permissionOrder = new Map<string, number>()
const permissionCatalogByPerm = new Map<string, PermissionCatalogOption>()
for (const group of accessPolicyPermissionCatalog) {
  for (const item of group.items) {
    permissionOrder.set(item.perm, permissionOrder.size)
    permissionCatalogByPerm.set(item.perm, {
      ...item,
      groupId: group.id,
      groupLabel: group.label,
      groupDescription: group.description
    })
  }
}

export const accessPolicyPermissionOptions = accessPolicyPermissionCatalog.flatMap((group) =>
  group.items.map((item) => ({
    ...item,
    groupId: group.id,
    groupLabel: group.label,
    groupDescription: group.description
  }))
)

const comparePerms = (left: string, right: string) => {
  const leftOrder = permissionOrder.get(left)
  const rightOrder = permissionOrder.get(right)
  if (leftOrder != null || rightOrder != null) {
    if (leftOrder == null) return 1
    if (rightOrder == null) return -1
    if (leftOrder !== rightOrder) {
      return leftOrder - rightOrder
    }
  }
  return left.localeCompare(right)
}

const normalizePerms = (items: string[]) => {
  const seen = new Set<string>()
  const out: string[] = []
  for (const item of items) {
    const trimmed = String(item || "").trim()
    if (!trimmed || seen.has(trimmed)) {
      continue
    }
    seen.add(trimmed)
    out.push(trimmed)
  }
  out.sort(comparePerms)
  return out
}

export const normalizeKnownPermissionSelection = (items: string[]) => {
  const known = normalizePerms(items).filter((item) => permissionOrder.has(item))
  return known.includes("*") ? ["*"] : known
}

export const splitKnownAndUnknownPerms = (items: string[]) => {
  const knownPerms: string[] = []
  const unknownPerms: string[] = []
  for (const item of normalizePerms(items)) {
    if (permissionOrder.has(item)) {
      knownPerms.push(item)
      continue
    }
    unknownPerms.push(item)
  }
  return {
    knownPerms: knownPerms.includes("*") ? ["*"] : knownPerms,
    unknownPerms
  }
}

export const mergeSelectedAndUnknownPerms = (knownPerms: string[], unknownPerms: string[]) => {
  const merged: string[] = []
  merged.push(...normalizeKnownPermissionSelection(knownPerms))
  for (const item of normalizePerms(unknownPerms)) {
    if (permissionOrder.has(item)) {
      continue
    }
    merged.push(item)
  }
  return merged
}

export const rolePresetPerms = (roleName: string) => {
  const normalized = String(roleName || "").trim()
  const preset = accessPolicyRolePresets[normalized]
  return preset ? normalizeKnownPermissionSelection(preset) : null
}

export const findPermissionCatalogItem = (perm: string) => {
  return permissionCatalogByPerm.get(String(perm || "").trim()) ?? null
}

export const orderRoleOptions = (roleNames: string[]) => {
  const seen = new Set<string>()
  const builtin: string[] = []
  const custom: string[] = []

  for (const role of builtinRoleOrder) {
    seen.add(role)
    builtin.push(role)
  }

  for (const roleName of roleNames) {
    const trimmed = String(roleName || "").trim()
    if (!trimmed || seen.has(trimmed)) {
      continue
    }
    seen.add(trimmed)
    custom.push(trimmed)
  }

  custom.sort((left, right) => left.localeCompare(right))
  return [...builtin, ...custom]
}
