<script setup lang="ts">
// Context: implements the AccessPolicy page in the Win frontend.
import { computed, onMounted, reactive, ref, watch } from "vue"
import CardHeader from "@/components/CardHeader.vue"
import PageHero from "@/components/PageHero.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import { useI18n } from "@/i18n"
import type { NodeRole, Policy, RolePerm } from "@/stores/accessPolicy"
import { useAccessPolicyStore } from "@/stores/accessPolicy"
import {
  accessPolicyPermissionOptions,
  findPermissionCatalogItem,
  mergeSelectedAndUnknownPerms,
  normalizeKnownPermissionSelection,
  orderRoleOptions,
  rolePresetPerms,
  splitKnownAndUnknownPerms,
  type PermissionCatalogOption
} from "@/stores/accessPolicyCatalog"
import { useAuthorityStore } from "@/stores/authority"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"

type AccessPolicyTab = "current" | "roles"

type EditableNodeRole = {
  nodeId: string
  role: string
}

type EditableRolePerm = {
  role: string
  perms: string[]
  unknownPerms: string[]
}

type NodeOverrideEditorMode = "create" | "edit"

type NodeOverrideDialogState = {
  open: boolean
  mode: NodeOverrideEditorMode
  index: number
  nodeId: string
  role: string
}

type DefaultAccessDialogState = {
  open: boolean
  defaultRole: string
  perms: string[]
  unknownPerms: string[]
}

type RoleEditorMode = "create" | "edit"

type RoleEditorDialogState = {
  open: boolean
  mode: RoleEditorMode
  index: number
  originalRole: string
  role: string
  perms: string[]
  unknownPerms: string[]
}

type RolePermissionPickerDialogState = {
  open: boolean
}

const tabs: Array<{ id: AccessPolicyTab; label: string }> = [
  { id: "current", label: "Current Policy" },
  { id: "roles", label: "Role Management" }
]

const sessionStore = useSessionStore()
const authorityStore = useAuthorityStore()
const accessPolicyStore = useAccessPolicyStore()
const toast = useToastStore()
const { t } = useI18n()

const permissionOptions = accessPolicyPermissionOptions

const activeTab = ref<AccessPolicyTab>("current")
const autoLoaded = ref(false)
const validationErrors = ref<string[]>([])
const nodePermsResult = ref<null | { nodeId: number; role: string; perms: string[] }>(null)
const runtimeDetailsOpen = ref(false)

const saveOptions = reactive({
  persist: true,
  applyRuntime: true,
  invalidate: true,
  refresh: true,
  verifyRuntime: true
})

const policyForm = reactive({
  defaultRole: "node",
  defaultPerms: [] as string[],
  defaultPermsUnknown: [] as string[],
  nodeRoles: [] as EditableNodeRole[],
  rolePerms: [] as EditableRolePerm[]
})

const nodePermsQuery = reactive({
  nodeId: ""
})

const defaultAccessDialog = reactive<DefaultAccessDialogState>({
  open: false,
  defaultRole: "node",
  perms: [],
  unknownPerms: []
})

const nodeOverrideDialog = reactive<NodeOverrideDialogState>({
  open: false,
  mode: "create",
  index: -1,
  nodeId: "",
  role: "node"
})

const roleEditorDialog = reactive<RoleEditorDialogState>({
  open: false,
  mode: "create",
  index: -1,
  originalRole: "",
  role: "",
  perms: [],
  unknownPerms: []
})

const rolePermissionPickerDialog = reactive<RolePermissionPickerDialogState>({
  open: false
})

const inputClass =
  "mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
const selectClass = inputClass

const ready = computed(() => {
  return Boolean(
    sessionStore.connected &&
      sessionStore.auth.loggedIn &&
      Number(sessionStore.auth.nodeId || 0) > 0 &&
      Number(sessionStore.auth.hubId || 0) > 0
  )
})

const identityLabel = computed(() => {
  const nodeId = Number(sessionStore.auth.nodeId || 0)
  const hubId = Number(sessionStore.auth.hubId || 0)
  if (!nodeId || !hubId) {
    return t("Not logged in")
  }
  return t("node={nodeId} hub={hubId}", { nodeId, hubId })
})

const authorityLabel = computed(() => {
  if (!authorityStore.state.authorityId) return "-"
  return String(authorityStore.state.authorityId)
})

const roleOptions = computed(() =>
  orderRoleOptions([
    policyForm.defaultRole,
    ...policyForm.nodeRoles.map((row) => row.role),
    ...policyForm.rolePerms.map((row) => row.role)
  ])
)

const summaryCards = computed(() => {
  return [
    {
      label: t("Default Role"),
      value: policyForm.defaultRole.trim() || "-"
    },
    {
      label: t("Default Perms"),
      value: String(policyForm.defaultPerms.length + policyForm.defaultPermsUnknown.length)
    },
    {
      label: t("Node Overrides"),
      value: String(policyForm.nodeRoles.filter((row) => row.nodeId.trim() || row.role.trim()).length)
    },
    {
      label: t("Role Bundles"),
      value: String(policyForm.rolePerms.filter((row) => row.role.trim()).length)
    }
  ]
})

const runtimePreview = computed(() => {
  const list = Array.isArray(accessPolicyStore.state.runtime) ? accessPolicyStore.state.runtime : []
  return list.slice(0, 120)
})

const runtimeSummary = computed(() => {
  if (accessPolicyStore.state.runtimeError) {
    return t("Runtime verify failed: {error}", {
      error: accessPolicyStore.state.runtimeError
    })
  }
  const total = Number(accessPolicyStore.state.runtimeTotal || accessPolicyStore.state.runtime.length || 0)
  return t("Runtime entries: {count}", { count: total })
})

const runtimeDetailsPreview = computed(() => {
  return runtimePreview.value.slice(0, 12)
})

const policyLoadNotice = computed(() => {
  return accessPolicyStore.state.loading ? t("Loading current policy from authority...") : ""
})

const savePolicyButtonLabel = computed(() => {
  return accessPolicyStore.state.saving ? t("Saving Policy…") : t("Save Policy")
})

const reloadPolicyButtonLabel = computed(() => {
  return accessPolicyStore.state.loading ? t("Loading…") : t("Reload Policy")
})

const policyActionBusy = computed(() => {
  return accessPolicyStore.state.loading || accessPolicyStore.state.saving
})

const tabButtonClass = (tab: AccessPolicyTab) => [
  "rounded-full px-4 py-2 text-sm font-semibold transition",
  activeTab.value === tab
    ? "bg-primary text-primary-foreground shadow-sm"
    : "text-muted-foreground hover:bg-muted/70"
]

const hasInvalidSeparator = (token: string) => {
  return token.includes(",") || token.includes(":") || token.includes(";")
}

const eventValue = (event: Event) => {
  return String((event.target as HTMLInputElement | HTMLSelectElement | null)?.value ?? "")
}

const setActiveTab = (tab: AccessPolicyTab) => {
  activeTab.value = tab
}

const knownPermPreview = (perms: string[], limit = 4) => {
  return perms.slice(0, limit)
}

const knownPermOverflow = (perms: string[], limit = 4) => {
  return Math.max(0, perms.length - limit)
}

const permissionMeta = (perm: string) => {
  return findPermissionCatalogItem(perm)
}

const permissionOptionLabel = (option: PermissionCatalogOption) => {
  return `${t(option.groupLabel)} · ${option.label}`
}

const permissionDescription = (perm: string) => {
  const meta = permissionMeta(perm)
  return meta ? t(meta.description) : perm
}

const permissionGroupLabel = (perm: string) => {
  const meta = permissionMeta(perm)
  return meta ? t(meta.groupLabel) : ""
}

const roleReferenceMessages = (roleName: string) => {
  const role = String(roleName || "").trim()
  const messages: string[] = []
  if (!role) {
    return messages
  }
  if (policyForm.defaultRole.trim() === role) {
    messages.push(t("Used by default access"))
  }
  const nodeCount = policyForm.nodeRoles.filter((row) => row.role.trim() === role).length
  if (nodeCount > 0) {
    messages.push(t("Used by {count} node override(s)", { count: nodeCount }))
  }
  return messages
}

const roleSummaryLine = (row: EditableRolePerm) => {
  const parts = [t("{count} selected", { count: row.perms.length + row.unknownPerms.length })]
  if (row.unknownPerms.length) {
    parts.push(t("{count} extra", { count: row.unknownPerms.length }))
  }
  return [...parts, ...roleReferenceMessages(row.role)].join(" · ")
}

const roleDialogAvailablePermissionOptions = computed(() => {
  const current = normalizeKnownPermissionSelection(roleEditorDialog.perms)
  if (current.includes("*")) {
    return []
  }
  const selected = new Set(current)
  return permissionOptions.filter((option) => !selected.has(option.perm))
})

const closeAllDialogs = () => {
  defaultAccessDialog.open = false
  nodeOverrideDialog.open = false
  roleEditorDialog.open = false
  rolePermissionPickerDialog.open = false
}

const resetNodeOverrideDialog = () => {
  nodeOverrideDialog.open = false
  nodeOverrideDialog.mode = "create"
  nodeOverrideDialog.index = -1
  nodeOverrideDialog.nodeId = ""
  nodeOverrideDialog.role = policyForm.defaultRole.trim() || "node"
}

const resetRoleEditorDialog = () => {
  roleEditorDialog.open = false
  roleEditorDialog.mode = "create"
  roleEditorDialog.index = -1
  roleEditorDialog.originalRole = ""
  roleEditorDialog.role = ""
  roleEditorDialog.perms = []
  roleEditorDialog.unknownPerms = []
  rolePermissionPickerDialog.open = false
}

const syncFormFromStore = () => {
  const policy = accessPolicyStore.state.policy
  const defaultPerms = splitKnownAndUnknownPerms(policy?.defaultPerms ?? [])

  policyForm.defaultRole = String(policy?.defaultRole || "node")
  policyForm.defaultPerms = defaultPerms.knownPerms
  policyForm.defaultPermsUnknown = defaultPerms.unknownPerms
  policyForm.nodeRoles = Array.isArray(policy?.nodeRoles)
    ? policy.nodeRoles.map((entry) => ({
        nodeId: entry?.nodeId ? String(entry.nodeId) : "",
        role: String(entry?.role || "")
      }))
    : []
  policyForm.rolePerms = Array.isArray(policy?.rolePerms)
    ? policy.rolePerms.map((entry) => {
        const perms = splitKnownAndUnknownPerms(entry?.perms ?? [])
        return {
          role: String(entry?.role || ""),
          perms: perms.knownPerms,
          unknownPerms: perms.unknownPerms
        }
      })
    : []
  closeAllDialogs()
  resetNodeOverrideDialog()
  resetRoleEditorDialog()
}

const resetLocalState = () => {
  accessPolicyStore.reset()
  syncFormFromStore()
  validationErrors.value = []
  nodePermsResult.value = null
  runtimeDetailsOpen.value = false
  activeTab.value = "current"
}

const assignStorePolicy = (policy: Policy) => {
  accessPolicyStore.state.policy.defaultRole = policy.defaultRole
  accessPolicyStore.state.policy.defaultPerms = [...policy.defaultPerms]
  accessPolicyStore.state.policy.nodeRoles = policy.nodeRoles.map((entry) => ({
    nodeId: Number(entry.nodeId || 0),
    role: entry.role
  }))
  accessPolicyStore.state.policy.rolePerms = policy.rolePerms.map((entry) => ({
    role: entry.role,
    perms: [...entry.perms]
  }))
}

const validatePerms = (scope: string, perms: string[], errors: string[]) => {
  for (const perm of perms) {
    if (hasInvalidSeparator(perm)) {
      errors.push(t("{scope}: perm '{perm}' contains invalid separator.", { scope, perm }))
      break
    }
  }
}

const buildPolicyFromForm = (): { policy?: Policy; errors: string[] } => {
  const errors: string[] = []

  const defaultRole = policyForm.defaultRole.trim()
  if (!defaultRole) {
    errors.push(t("Default role is required."))
  } else if (hasInvalidSeparator(defaultRole)) {
    errors.push(t("Default role cannot contain ',', ':', or ';'."))
  }

  const defaultPerms = mergeSelectedAndUnknownPerms(
    policyForm.defaultPerms,
    policyForm.defaultPermsUnknown
  )
  validatePerms(t("Default Access"), defaultPerms, errors)

  const nodeRoles: NodeRole[] = []
  const nodeSeen = new Set<number>()
  for (let i = 0; i < policyForm.nodeRoles.length; i += 1) {
    const row = policyForm.nodeRoles[i]
    const rowNo = i + 1
    const nodeRaw = String(row.nodeId || "").trim()
    const role = String(row.role || "").trim()

    if (!nodeRaw && !role) {
      continue
    }

    const nodeId = Number.parseInt(nodeRaw, 10)
    if (Number.isNaN(nodeId) || nodeId <= 0) {
      errors.push(t("Node role row {rowNo}: nodeId must be a positive number.", { rowNo }))
      continue
    }
    if (!role) {
      errors.push(t("Node role row {rowNo}: role is required.", { rowNo }))
      continue
    }
    if (hasInvalidSeparator(role)) {
      errors.push(t("Node role row {rowNo}: role cannot contain ',', ':', or ';'.", { rowNo }))
      continue
    }
    if (nodeSeen.has(nodeId)) {
      errors.push(
        t("Node role row {rowNo}: duplicate nodeId {nodeId}.", { rowNo, nodeId })
      )
      continue
    }

    nodeSeen.add(nodeId)
    nodeRoles.push({ nodeId, role })
  }

  const rolePerms: RolePerm[] = []
  const roleSeen = new Set<string>()
  for (let i = 0; i < policyForm.rolePerms.length; i += 1) {
    const row = policyForm.rolePerms[i]
    const rowNo = i + 1
    const role = String(row.role || "").trim()
    if (!role) {
      errors.push(t("Role perms row {rowNo}: role is required.", { rowNo }))
      continue
    }
    if (hasInvalidSeparator(role)) {
      errors.push(t("Role perms row {rowNo}: role cannot contain ',', ':', or ';'.", { rowNo }))
      continue
    }
    if (roleSeen.has(role)) {
      errors.push(t("Role perms row {rowNo}: duplicate role '{role}'.", { rowNo, role }))
      continue
    }
    roleSeen.add(role)

    const perms = mergeSelectedAndUnknownPerms(row.perms, row.unknownPerms)
    validatePerms(t("Role Catalog"), perms, errors)
    rolePerms.push({ role, perms })
  }

  if (errors.length) {
    return { errors }
  }

  return {
    errors,
    policy: {
      defaultRole,
      defaultPerms,
      nodeRoles,
      rolePerms
    }
  }
}

const ensureReady = () => {
  if (!sessionStore.connected) {
    throw new Error(t("Connect to a session first."))
  }
  if (!sessionStore.auth.loggedIn) {
    throw new Error(t("Login is required."))
  }
  if (!Number(sessionStore.auth.nodeId || 0)) {
    throw new Error(t("Node ID missing."))
  }
  if (!Number(sessionStore.auth.hubId || 0)) {
    throw new Error(t("Hub ID missing."))
  }
}

const loadPolicy = async (silent = false) => {
  try {
    ensureReady()
    validationErrors.value = []
    await accessPolicyStore.loadPolicy()
    syncFormFromStore()
    if (!silent) {
      toast.success(
        t("Policy loaded."),
        t("authority={authorityId}", { authorityId: authorityStore.state.authorityId })
      )
    }
  } catch (err) {
    if (!silent) {
      console.warn(err)
      toast.errorOf(err, t("Failed to load policy."))
    }
  }
}

const savePolicy = async () => {
  try {
    ensureReady()
    const built = buildPolicyFromForm()
    validationErrors.value = built.errors
    if (built.errors.length) {
      toast.warn(
        t("Policy validation failed."),
        t("{count} issue(s).", { count: built.errors.length })
      )
      return
    }
    assignStorePolicy(built.policy as Policy)
    const resp = await accessPolicyStore.savePolicy(saveOptions)
    syncFormFromStore()
    if (resp.runtimeError) {
      toast.warn(t("Policy saved with runtime verify warning."), resp.runtimeError)
    } else {
      toast.success(
        t("Policy saved."),
        t("authority={authorityId}", { authorityId: resp.authorityId })
      )
    }
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to save policy."))
  }
}

const queryNodePerms = async () => {
  try {
    ensureReady()
    const parsed = Number.parseInt(String(nodePermsQuery.nodeId || "").trim(), 10)
    if (Number.isNaN(parsed) || parsed <= 0) {
      throw new Error(t("Node ID must be a positive number."))
    }
    const result = await accessPolicyStore.getNodePerms(parsed)
    nodePermsResult.value = result
    toast.success(t("Node perms loaded."), t("node={nodeId}", { nodeId: result.nodeId }))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to query node perms."))
  }
}

const openCreateNodeOverrideDialog = () => {
  resetNodeOverrideDialog()
  nodeOverrideDialog.open = true
  nodeOverrideDialog.mode = "create"
}

const openNodeOverrideEditor = (index: number) => {
  const row = policyForm.nodeRoles[index]
  if (!row) {
    return
  }
  resetNodeOverrideDialog()
  nodeOverrideDialog.open = true
  nodeOverrideDialog.mode = "edit"
  nodeOverrideDialog.index = index
  nodeOverrideDialog.nodeId = row.nodeId
  nodeOverrideDialog.role = row.role || policyForm.defaultRole.trim() || "node"
}

const closeNodeOverrideDialog = () => {
  resetNodeOverrideDialog()
}

const submitNodeOverrideDialog = () => {
  const nodeRaw = nodeOverrideDialog.nodeId.trim()
  const role = nodeOverrideDialog.role.trim()

  if (!nodeRaw) {
    toast.warn(t("Node ID is required."))
    return
  }

  const nodeId = Number.parseInt(nodeRaw, 10)
  if (Number.isNaN(nodeId) || nodeId <= 0) {
    toast.warn(t("Node ID must be a positive number."))
    return
  }

  if (!role) {
    toast.warn(t("Role is required."))
    return
  }

  if (hasInvalidSeparator(role)) {
    toast.warn(t("Role name cannot contain ',', ':', or ';'."))
    return
  }

  const duplicate = policyForm.nodeRoles.some((row, index) => {
    if (nodeOverrideDialog.mode === "edit" && index === nodeOverrideDialog.index) {
      return false
    }
    return Number.parseInt(String(row.nodeId || "").trim(), 10) === nodeId
  })
  if (duplicate) {
    toast.warn(t("Node override for node {nodeId} already exists.", { nodeId }))
    return
  }

  const nextRow: EditableNodeRole = {
    nodeId: String(nodeId),
    role
  }

  if (nodeOverrideDialog.mode === "create") {
    policyForm.nodeRoles.push(nextRow)
  } else if (policyForm.nodeRoles[nodeOverrideDialog.index]) {
    policyForm.nodeRoles[nodeOverrideDialog.index] = nextRow
  }

  validationErrors.value = []
  closeNodeOverrideDialog()
}

const removeNodeRole = (index: number) => {
  policyForm.nodeRoles.splice(index, 1)
  validationErrors.value = []
}

const openDefaultAccessDialog = () => {
  defaultAccessDialog.open = true
  defaultAccessDialog.defaultRole = policyForm.defaultRole.trim() || "node"
  defaultAccessDialog.perms = [...policyForm.defaultPerms]
  defaultAccessDialog.unknownPerms = [...policyForm.defaultPermsUnknown]
}

const closeDefaultAccessDialog = () => {
  defaultAccessDialog.open = false
}

const openCreateRoleDialog = () => {
  resetRoleEditorDialog()
  roleEditorDialog.open = true
  roleEditorDialog.mode = "create"
}

const openRoleEditor = (index: number) => {
  const row = policyForm.rolePerms[index]
  if (!row) {
    return
  }
  resetRoleEditorDialog()
  roleEditorDialog.open = true
  roleEditorDialog.mode = "edit"
  roleEditorDialog.index = index
  roleEditorDialog.originalRole = row.role
  roleEditorDialog.role = row.role
  roleEditorDialog.perms = [...row.perms]
  roleEditorDialog.unknownPerms = [...row.unknownPerms]
}

const closeRoleEditorDialog = () => {
  resetRoleEditorDialog()
}

const getAddPermissionCandidate = (perms: string[]) => {
  const current = normalizeKnownPermissionSelection(perms)
  if (current.includes("*")) {
    return null
  }
  const selected = new Set(current)
  const nextKnown = permissionOptions.find((option) => option.perm !== "*" && !selected.has(option.perm))
  if (nextKnown) {
    return nextKnown.perm
  }
  if (!selected.size) {
    return permissionOptions[0]?.perm ?? null
  }
  return null
}

const canAddPermission = (perms: string[]) => {
  return Boolean(getAddPermissionCandidate(perms))
}

const addPermissionRow = (target: { perms: string[] }) => {
  const nextPerm = getAddPermissionCandidate(target.perms)
  if (!nextPerm) {
    return
  }
  target.perms = normalizeKnownPermissionSelection([...target.perms, nextPerm])
}

const replacePermissionAt = (target: { perms: string[] }, index: number, nextPerm: string) => {
  const current = [...normalizeKnownPermissionSelection(target.perms)]
  if (index < 0 || index >= current.length) {
    return
  }
  current[index] = nextPerm
  target.perms = normalizeKnownPermissionSelection(current)
}

const removePermissionAt = (target: { perms: string[] }, index: number) => {
  const current = [...normalizeKnownPermissionSelection(target.perms)]
  if (index < 0 || index >= current.length) {
    return
  }
  current.splice(index, 1)
  target.perms = normalizeKnownPermissionSelection(current)
}

const onDefaultDialogPermissionChange = (index: number, event: Event) => {
  replacePermissionAt(defaultAccessDialog, index, eventValue(event))
}

const openRolePermissionPickerDialog = () => {
  if (!roleDialogAvailablePermissionOptions.value.length) {
    return
  }
  rolePermissionPickerDialog.open = true
}

const closeRolePermissionPickerDialog = () => {
  rolePermissionPickerDialog.open = false
}

const addRoleDialogPermission = (perm: string) => {
  const nextPerm = String(perm || "").trim()
  if (!nextPerm) {
    return
  }
  if (!roleDialogAvailablePermissionOptions.value.some((option) => option.perm === nextPerm)) {
    return
  }
  roleEditorDialog.perms = normalizeKnownPermissionSelection([...roleEditorDialog.perms, nextPerm])
  closeRolePermissionPickerDialog()
}

const isPermissionOptionDisabled = (selectedPerms: string[], index: number, perm: string) => {
  return selectedPerms[index] !== perm && selectedPerms.includes(perm)
}

const submitDefaultAccessDialog = () => {
  const defaultRole = defaultAccessDialog.defaultRole.trim()
  if (!defaultRole) {
    toast.warn(t("Default role is required."))
    return
  }
  if (hasInvalidSeparator(defaultRole)) {
    toast.warn(t("Default role cannot contain ',', ':', or ';'."))
    return
  }
  policyForm.defaultRole = defaultRole
  policyForm.defaultPerms = normalizeKnownPermissionSelection(defaultAccessDialog.perms)
  policyForm.defaultPermsUnknown = [...defaultAccessDialog.unknownPerms]
  validationErrors.value = []
  closeDefaultAccessDialog()
}

const applyRoleReferenceRename = (oldRoleName: string, nextRoleName: string) => {
  const oldRole = String(oldRoleName || "").trim()
  const nextRole = String(nextRoleName || "").trim()
  if (!oldRole || oldRole === nextRole) {
    return
  }
  if (policyForm.defaultRole.trim() === oldRole) {
    policyForm.defaultRole = nextRole
  }
  for (const row of policyForm.nodeRoles) {
    if (row.role.trim() === oldRole) {
      row.role = nextRole
    }
  }
}

const submitRoleEditorDialog = () => {
  const role = roleEditorDialog.role.trim()
  if (!role) {
    toast.warn(t("Role name is required."))
    return
  }
  if (hasInvalidSeparator(role)) {
    toast.warn(t("Role name cannot contain ',', ':', or ';'."))
    return
  }

  const duplicate = policyForm.rolePerms.some((row, index) => {
    if (roleEditorDialog.mode === "edit" && index === roleEditorDialog.index) {
      return false
    }
    return row.role.trim() === role
  })
  if (duplicate) {
    toast.warn(t("Role '{role}' already exists.", { role }))
    return
  }

  const nextRow: EditableRolePerm = {
    role,
    perms: normalizeKnownPermissionSelection(roleEditorDialog.perms),
    unknownPerms: [...roleEditorDialog.unknownPerms]
  }

  if (roleEditorDialog.mode === "create") {
    policyForm.rolePerms.push(nextRow)
  } else {
    const current = policyForm.rolePerms[roleEditorDialog.index]
    if (!current) {
      closeRoleEditorDialog()
      return
    }
    applyRoleReferenceRename(current.role, role)
    policyForm.rolePerms[roleEditorDialog.index] = nextRow
  }

  validationErrors.value = []
  closeRoleEditorDialog()
}

const removeRole = (index: number) => {
  const row = policyForm.rolePerms[index]
  if (!row) {
    return
  }
  const references = roleReferenceMessages(row.role)
  if (references.length) {
    toast.warn(t("Role is still referenced."), references.join(" · "))
    return
  }
  policyForm.rolePerms.splice(index, 1)
  validationErrors.value = []
}

const applyRolePresetToDialog = () => {
  const preset = rolePresetPerms(roleEditorDialog.role)
  if (!preset) {
    return
  }
  roleEditorDialog.perms = normalizeKnownPermissionSelection([...preset])
  closeRolePermissionPickerDialog()
  toast.success(t("Built-in preset applied."), roleEditorDialog.role.trim())
}

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  ([nodeId, hubId], oldValue) => {
    const [prevNodeId, prevHubId] = Array.isArray(oldValue) ? oldValue : []
    authorityStore.setIdentity(Number(nodeId || 0), Number(hubId || 0))
    if (
      Number(nodeId || 0) !== Number(prevNodeId || 0) ||
      Number(hubId || 0) !== Number(prevHubId || 0)
    ) {
      autoLoaded.value = false
      resetLocalState()
    }
  },
  { immediate: true }
)

watch(
  () => ready.value,
  (isReady) => {
    if (!isReady) {
      autoLoaded.value = false
      resetLocalState()
      return
    }
    if (autoLoaded.value) {
      return
    }
    autoLoaded.value = true
    void loadPolicy(true)
  },
  { immediate: true }
)

onMounted(() => {
  syncFormFromStore()
  if (ready.value && !autoLoaded.value) {
    autoLoaded.value = true
    void loadPolicy(true)
  }
})
</script>

<template>
  <section class="space-y-6">
    <PageHero
      :title="t('Access Policy')"
      :description="t('Manage authority access rules, role catalog, and runtime policy validation.')"
    >
      <template #actions>
        <div class="inline-flex rounded-full border border-border/70 bg-background/80 p-1">
          <button
            v-for="tab in tabs"
            :key="tab.id"
            type="button"
            :class="tabButtonClass(tab.id)"
            :aria-pressed="activeTab === tab.id"
            @click="setActiveTab(tab.id)"
          >
            {{ t(tab.label) }}
          </button>
        </div>
      </template>
    </PageHero>

    <section v-if="activeTab === 'current'" class="space-y-6">
      <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
        <CardHeader
          :title="t('Current Policy')"
          :description="t('Access defaults, node overrides, and runtime checks from a lighter review layout.')"
          title-tag="h2"
          title-class="text-lg"
        >
          <template #actions>
            <Badge variant="secondary">{{ sessionStore.connected ? t("Connected") : t("Disconnected") }}</Badge>
            <Badge variant="secondary">{{ sessionStore.auth.loggedIn ? t("Logged in") : t("Logged out") }}</Badge>
            <Badge variant="secondary">{{ identityLabel }}</Badge>
            <Badge variant="secondary">{{ t("Authority {authority}", { authority: authorityLabel }) }}</Badge>
            <Button
              size="sm"
              variant="outline"
              :disabled="policyActionBusy"
              @click="loadPolicy(false)"
            >
              {{ reloadPolicyButtonLabel }}
            </Button>
          </template>
        </CardHeader>

        <p class="mt-4 text-sm text-muted-foreground">
          {{ t("Current identity drives authority resolution automatically.") }}
        </p>

        <div
          v-if="accessPolicyStore.state.loading"
          data-policy-loading-notice
          class="mt-4 rounded-2xl border border-sky-200 bg-sky-50 px-4 py-3 text-sm text-sky-800"
        >
          <p class="font-semibold">{{ t("Loading…") }}</p>
          <p class="mt-1">{{ policyLoadNotice }}</p>
        </div>

        <div class="mt-5 grid gap-3 md:grid-cols-4">
          <div
            v-for="card in summaryCards"
            :key="card.label"
            class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3"
          >
            <p class="text-[11px] font-semibold uppercase tracking-[0.25em] text-muted-foreground">
              {{ card.label }}
            </p>
            <p class="mt-2 text-lg font-semibold text-foreground">{{ card.value }}</p>
          </div>
        </div>
      </section>

      <section class="grid gap-6 xl:grid-cols-[minmax(0,1.02fr)_400px]">
        <div class="space-y-6">
          <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
            <CardHeader
              :title="t('Default Access')"
              :description="t('View the fallback role, preserve historical extras, and open a focused editor only when you need to change it.')"
              title-tag="h3"
              title-class="text-base"
            >
              <template #actions>
                <Button size="sm" variant="outline" @click="setActiveTab('roles')">{{ t("Manage Roles") }}</Button>
                <Button size="sm" variant="outline" :disabled="policyActionBusy" @click="savePolicy">
                  {{ savePolicyButtonLabel }}
                </Button>
                <Button size="sm" @click="openDefaultAccessDialog">{{ t("Edit Default Access") }}</Button>
              </template>
            </CardHeader>

            <div class="mt-4 grid gap-3 sm:grid-cols-3">
              <div class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3">
                <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Default Role") }}
                </p>
                <p class="mt-2 text-base font-semibold text-foreground">{{ policyForm.defaultRole || "-" }}</p>
              </div>
              <div class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3">
                <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Known Permissions") }}
                </p>
                <p class="mt-2 text-base font-semibold text-foreground">{{ policyForm.defaultPerms.length }}</p>
              </div>
              <div class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3">
                <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Preserved Extras") }}
                </p>
                <p class="mt-2 text-base font-semibold text-foreground">
                  {{ policyForm.defaultPermsUnknown.length }}
                </p>
              </div>
            </div>

            <div class="mt-4 rounded-2xl border border-border/60 bg-background/70 px-4 py-4">
              <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Selected Permissions") }}
              </p>
              <div v-if="policyForm.defaultPerms.length" class="mt-3 flex flex-wrap gap-2">
                <Badge
                  v-for="perm in knownPermPreview(policyForm.defaultPerms)"
                  :key="`default-known-${perm}`"
                  variant="secondary"
                >
                  {{ perm }}
                </Badge>
                <Badge
                  v-if="knownPermOverflow(policyForm.defaultPerms) > 0"
                  variant="outline"
                >
                  {{ t("+{count} more", { count: knownPermOverflow(policyForm.defaultPerms) }) }}
                </Badge>
              </div>
              <p v-else class="mt-3 text-sm text-muted-foreground">{{ t("No catalog permissions selected.") }}</p>

              <div v-if="policyForm.defaultPermsUnknown.length" class="mt-4 flex flex-wrap gap-2">
                <Badge
                  v-for="perm in policyForm.defaultPermsUnknown"
                  :key="`default-extra-${perm}`"
                  variant="outline"
                >
                  {{ perm }}
                </Badge>
              </div>
            </div>
          </section>

          <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
            <CardHeader
              :title="t('Node Overrides')"
              :description="t('Keep overrides in a compact list and edit only the node you are working on.')"
              title-tag="h3"
              title-class="text-base"
            >
              <template #actions>
                <Button size="sm" variant="outline" :disabled="policyActionBusy" @click="savePolicy">
                  {{ savePolicyButtonLabel }}
                </Button>
                <Button size="sm" @click="openCreateNodeOverrideDialog">{{ t("Add Node Role") }}</Button>
              </template>
            </CardHeader>

            <div
              v-if="policyForm.nodeRoles.length"
              class="mt-4 overflow-hidden rounded-2xl border border-border/60 bg-background/70"
            >
              <article
                v-for="(row, index) in policyForm.nodeRoles"
                :key="`node-role-${row.nodeId || index}`"
                class="flex flex-wrap items-start justify-between gap-4 border-b border-border/60 px-4 py-4 last:border-b-0"
              >
                <div class="min-w-0 flex-1">
                  <p class="text-base font-semibold text-foreground">
                    {{ t("Node {nodeId}", { nodeId: row.nodeId || "-" }) }}
                  </p>
                  <p class="mt-1 text-sm text-muted-foreground">
                    {{ t("Role: {role}", { role: row.role || "-" }) }}
                  </p>
                </div>
                <div class="flex flex-wrap items-center gap-2">
                  <Button size="sm" variant="outline" @click="openNodeOverrideEditor(index)">{{ t("Edit") }}</Button>
                  <Button size="sm" variant="ghost" @click="removeNodeRole(index)">{{ t("Remove") }}</Button>
                </div>
              </article>
            </div>
            <p v-else class="mt-4 text-sm text-muted-foreground">
              {{ t("No explicit node role entries.") }}
            </p>
          </section>
        </div>

        <div class="space-y-6">
          <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
            <CardHeader
              :title="t('Operations Panel')"
              :description="t('Save changes, inspect runtime state, and query one node from the same compact panel.')"
              title-tag="h3"
              title-class="text-base"
            />

            <div class="mt-4 space-y-5">
              <section class="rounded-2xl border border-border/60 bg-background/70 p-4">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p class="text-sm font-semibold text-foreground">{{ t("Apply & Save") }}</p>
                    <p class="mt-1 text-xs text-muted-foreground">
                      {{ t("Choose how far to push the change, then keep validation feedback anchored in this panel.") }}
                    </p>
                  </div>
                </div>

                <div class="mt-4 grid gap-3 sm:grid-cols-2">
                  <label class="flex items-center gap-2 text-sm text-foreground">
                    <input v-model="saveOptions.persist" type="checkbox" class="h-4 w-4 rounded border border-input" />
                    {{ t("Persist auth.* config") }}
                  </label>
                  <label class="flex items-center gap-2 text-sm text-foreground">
                    <input v-model="saveOptions.applyRuntime" type="checkbox" class="h-4 w-4 rounded border border-input" />
                    {{ t("Push perms_snapshot") }}
                  </label>
                  <label class="flex items-center gap-2 text-sm text-foreground">
                    <input v-model="saveOptions.invalidate" type="checkbox" class="h-4 w-4 rounded border border-input" />
                    {{ t("perms_invalidate") }}
                  </label>
                  <label class="flex items-center gap-2 text-sm text-foreground">
                    <input v-model="saveOptions.refresh" type="checkbox" class="h-4 w-4 rounded border border-input" />
                    {{ t("invalidate.refresh=true") }}
                  </label>
                  <label class="flex items-center gap-2 text-sm text-foreground sm:col-span-2">
                    <input v-model="saveOptions.verifyRuntime" type="checkbox" class="h-4 w-4 rounded border border-input" />
                    {{ t("Verify runtime by list_roles after save") }}
                  </label>
                </div>

                <div class="mt-4 flex flex-wrap gap-2">
                  <Button :disabled="policyActionBusy" @click="savePolicy">{{ savePolicyButtonLabel }}</Button>
                  <Button
                    variant="outline"
                    :disabled="policyActionBusy"
                    @click="loadPolicy(false)"
                  >
                    {{ reloadPolicyButtonLabel }}
                  </Button>
                </div>

                <div
                  v-if="validationErrors.length"
                  class="mt-4 rounded-2xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700"
                >
                  <p class="font-semibold">{{ t("Validation Errors") }}</p>
                  <ul class="mt-2 list-disc space-y-1 pl-5">
                    <li v-for="(msg, index) in validationErrors" :key="`validation-${index}`">{{ msg }}</li>
                  </ul>
                </div>

                <div
                  v-if="accessPolicyStore.state.warnings.length"
                  class="mt-4 rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700"
                >
                  <p class="font-semibold">{{ t("Policy Warnings") }}</p>
                  <ul class="mt-2 list-disc space-y-1 pl-5">
                    <li v-for="(msg, index) in accessPolicyStore.state.warnings" :key="`warning-${index}`">{{ msg }}</li>
                  </ul>
                </div>
              </section>

              <section class="rounded-2xl border border-border/60 bg-background/70 p-4">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p class="text-sm font-semibold text-foreground">{{ t("Runtime Snapshot") }}</p>
                    <p class="mt-1 text-xs text-muted-foreground">{{ runtimeSummary }}</p>
                  </div>
                  <div class="flex flex-wrap items-center gap-2">
                    <Badge variant="secondary">
                      {{ t("{count} entries", { count: accessPolicyStore.state.runtimeTotal || accessPolicyStore.state.runtime.length }) }}
                    </Badge>
                    <Button
                      size="sm"
                      variant="outline"
                      @click="runtimeDetailsOpen = !runtimeDetailsOpen"
                    >
                      {{ runtimeDetailsOpen ? t("Hide Runtime Details") : t("Show Runtime Details") }}
                    </Button>
                  </div>
                </div>

                <p v-if="accessPolicyStore.state.runtimeError" class="mt-3 text-sm text-amber-700">
                  {{ accessPolicyStore.state.runtimeError }}
                </p>

                <div v-if="runtimeDetailsOpen" class="mt-4 max-h-[280px] space-y-2 overflow-y-auto">
                  <div
                    v-for="entry in runtimeDetailsPreview"
                    :key="`runtime-${entry.nodeId}`"
                    class="rounded-2xl border border-border/60 bg-card/80 px-4 py-3 text-sm"
                  >
                    <p class="font-semibold text-foreground">
                      {{ t("Node {nodeId} -> {role}", { nodeId: entry.nodeId, role: entry.role || "-" }) }}
                    </p>
                    <p class="mt-1 break-all text-muted-foreground">
                      {{ entry.perms.length ? entry.perms.join(",") : t("(no perms)") }}
                    </p>
                  </div>
                  <p v-if="!runtimeDetailsPreview.length" class="text-sm text-muted-foreground">
                    {{ t("No runtime entries.") }}
                  </p>
                </div>
              </section>

              <section class="rounded-2xl border border-border/60 bg-background/70 p-4">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p class="text-sm font-semibold text-foreground">{{ t("Node Perms Lookup") }}</p>
                    <p class="mt-1 text-xs text-muted-foreground">
                      {{ t("Query one node without leaving the operations panel.") }}
                    </p>
                  </div>
                </div>

                <div class="mt-4 grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto]">
                  <input
                    v-model="nodePermsQuery.nodeId"
                    :class="inputClass"
                    :placeholder="t('Node ID')"
                    @keydown.enter.prevent="queryNodePerms"
                  />
                  <div class="self-end">
                    <Button variant="outline" size="sm" @click="queryNodePerms">{{ t("Query") }}</Button>
                  </div>
                </div>

                <div
                  v-if="nodePermsResult"
                  class="mt-4 rounded-2xl border border-border/60 bg-card/80 px-4 py-3 text-sm"
                >
                  <p class="font-semibold text-foreground">
                    {{ t("Node {nodeId}", { nodeId: nodePermsResult.nodeId }) }}
                  </p>
                  <p class="mt-1 text-muted-foreground">{{ t("Role: {role}", { role: nodePermsResult.role || "-" }) }}</p>
                  <p class="mt-1 break-all text-muted-foreground">
                    {{
                      t("Perms: {perms}", {
                        perms: nodePermsResult.perms.length ? nodePermsResult.perms.join(",") : t("(no perms)")
                      })
                    }}
                  </p>
                </div>
                <p v-else class="mt-4 text-sm text-muted-foreground">
                  {{ t("No node queried yet.") }}
                </p>
              </section>
            </div>
          </section>
        </div>
      </section>
    </section>

    <section v-else class="space-y-6">
      <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
        <CardHeader
          :title="t('Role Management')"
          :description="t('Manage named roles from a compact list. Open a dialog only for the role you want to change.')"
          title-tag="h2"
          title-class="text-lg"
        >
          <template #actions>
            <Badge variant="secondary">{{ t("{count} roles", { count: policyForm.rolePerms.length }) }}</Badge>
            <Button size="sm" variant="outline" :disabled="policyActionBusy" @click="savePolicy">
              {{ savePolicyButtonLabel }}
            </Button>
            <Button size="sm" @click="openCreateRoleDialog">{{ t("Add Role") }}</Button>
          </template>
        </CardHeader>
      </section>

      <section
        v-if="accessPolicyStore.state.loading"
        data-policy-loading-notice
        class="rounded-2xl border border-sky-200 bg-sky-50 px-5 py-4 text-sm text-sky-800 shadow-sm"
      >
        <p class="font-semibold">{{ t("Loading…") }}</p>
        <p class="mt-1">{{ policyLoadNotice }}</p>
      </section>

      <section
        v-if="policyForm.rolePerms.length"
        class="overflow-hidden rounded-2xl border bg-card/90 text-card-foreground shadow-sm"
      >
        <article
          v-for="(row, index) in policyForm.rolePerms"
          :key="`role-row-${row.role || index}`"
          class="border-b border-border/60 bg-background/70 px-4 py-3 last:border-b-0"
        >
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1">
              <p class="truncate font-semibold text-foreground">{{ row.role.trim() || t("Unnamed Role") }}</p>
              <Badge v-if="rolePresetPerms(row.role)" variant="secondary">{{ t("Built-in Baseline") }}</Badge>
              <p class="text-xs text-muted-foreground">{{ roleSummaryLine(row) }}</p>
            </div>

            <div class="flex items-center gap-2">
              <Button size="sm" variant="outline" @click="openRoleEditor(index)">{{ t("Edit") }}</Button>
              <Button size="sm" variant="ghost" @click="removeRole(index)">{{ t("Remove") }}</Button>
            </div>
          </div>
        </article>
      </section>

      <section
        v-else
        class="rounded-2xl border border-dashed bg-card/70 p-6 text-sm text-muted-foreground shadow-sm"
      >
        {{ t("No roles defined yet. Create one to start managing named access bundles.") }}
      </section>
    </section>

    <Overlay
      :open="nodeOverrideDialog.open"
      overlayClass="bg-black/40 p-4"
      closeOnBackdrop
      trapFocus
      initialFocusSelector="[data-node-override-id]"
      @close="closeNodeOverrideDialog"
    >
      <div class="w-full max-w-xl rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl">
        <CardHeader
          :title="nodeOverrideDialog.mode === 'create' ? t('Create Node Override') : t('Edit Node Override')"
          :description="t('Override one node at a time so the page stays focused on the summary list.')"
          title-tag="h3"
          title-class="text-lg"
        />

        <div class="mt-5 grid gap-4">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Node ID") }}
            </label>
            <input
              v-model="nodeOverrideDialog.nodeId"
              data-node-override-id
              :class="inputClass"
              :placeholder="t('nodeId')"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Role") }}
            </label>
            <select v-model="nodeOverrideDialog.role" :class="selectClass">
              <option v-for="role in roleOptions" :key="`node-override-role-${role}`" :value="role">
                {{ role }}
              </option>
            </select>
            <p class="mt-2 text-xs text-muted-foreground">
              {{ t("Choose one existing role bundle for this exceptional node.") }}
            </p>
          </div>
        </div>

        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <Button variant="outline" @click="closeNodeOverrideDialog">{{ t("Cancel") }}</Button>
          <Button @click="submitNodeOverrideDialog">{{ t("Save") }}</Button>
        </div>
      </div>
    </Overlay>

    <Overlay
      :open="defaultAccessDialog.open"
      overlayClass="bg-black/40 p-4"
      closeOnBackdrop
      trapFocus
      initialFocusSelector="[data-default-role-select]"
      @close="closeDefaultAccessDialog"
    >
      <div class="w-full max-w-3xl rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl">
        <CardHeader
          :title="t('Edit Default Access')"
          :description="t('Create or update one focused permission list instead of editing the entire page.')"
          title-tag="h3"
          title-class="text-lg"
        />

        <div class="mt-5 grid gap-4 lg:grid-cols-[260px_minmax(0,1fr)]">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Default Role") }}
            </label>
            <select
              v-model="defaultAccessDialog.defaultRole"
              data-default-role-select
              :class="selectClass"
            >
              <option v-for="role in roleOptions" :key="`default-role-${role}`" :value="role">
                {{ role }}
              </option>
            </select>
            <p class="mt-2 text-xs text-muted-foreground">
              {{ t("Open the role management tab when you need to create or rename role bundles.") }}
            </p>
          </div>

          <div class="rounded-2xl border border-border/60 bg-background/70 p-4">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div>
                <p class="text-sm font-semibold text-foreground">{{ t("Permission List") }}</p>
                <p class="mt-1 text-xs text-muted-foreground">
                  {{ t("Permissions in this editor come from the built-in catalog only. Add rows as needed, then remove the ones you no longer want.") }}
                </p>
              </div>
              <Button
                size="sm"
                variant="outline"
                :disabled="!canAddPermission(defaultAccessDialog.perms)"
                @click="addPermissionRow(defaultAccessDialog)"
              >
                {{ t("Add Permission") }}
              </Button>
            </div>

            <div v-if="defaultAccessDialog.perms.length" class="mt-4 space-y-3">
              <div
                v-for="(perm, index) in defaultAccessDialog.perms"
                :key="`default-perm-row-${index}-${perm}`"
                class="grid gap-3 rounded-2xl border border-border/60 bg-card/80 p-4 lg:grid-cols-[minmax(0,1fr)_auto]"
              >
                <div>
                  <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    {{ t("Permission") }}
                  </label>
                  <select
                    :value="perm"
                    :class="selectClass"
                    @change="onDefaultDialogPermissionChange(index, $event)"
                  >
                    <option
                      v-for="option in permissionOptions"
                      :key="`default-option-${index}-${option.perm}`"
                      :value="option.perm"
                      :disabled="isPermissionOptionDisabled(defaultAccessDialog.perms, index, option.perm)"
                    >
                      {{ permissionOptionLabel(option) }}
                    </option>
                  </select>
                  <p class="mt-2 text-xs text-muted-foreground">
                    {{ permissionDescription(perm) }}
                  </p>
                </div>
                <div class="self-end">
                  <Button size="sm" variant="outline" @click="removePermissionAt(defaultAccessDialog, index)">
                    {{ t("Remove") }}
                  </Button>
                </div>
              </div>
            </div>

            <p v-else class="mt-4 text-sm text-muted-foreground">{{ t("No permissions selected yet.") }}</p>
          </div>
        </div>

        <div
          v-if="defaultAccessDialog.unknownPerms.length"
          class="mt-5 rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800"
        >
          <p class="font-semibold">{{ t("Preserved extra permissions") }}</p>
          <p class="mt-1">{{ t("These permissions come from existing policy data and stay preserved because they are outside the built-in catalog.") }}</p>
          <div class="mt-3 flex flex-wrap gap-2">
            <Badge
              v-for="perm in defaultAccessDialog.unknownPerms"
              :key="`default-dialog-extra-${perm}`"
              variant="secondary"
            >
              {{ perm }}
            </Badge>
          </div>
        </div>

        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <Button variant="outline" @click="closeDefaultAccessDialog">{{ t("Cancel") }}</Button>
          <Button @click="submitDefaultAccessDialog">{{ t("Save") }}</Button>
        </div>
      </div>
    </Overlay>

    <Overlay
      :open="roleEditorDialog.open"
      overlayClass="bg-black/40 p-4"
      closeOnBackdrop
      trapFocus
      initialFocusSelector="[data-role-editor-name]"
      @close="closeRoleEditorDialog"
    >
      <div
        data-role-editor-dialog
        class="flex max-h-[85vh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl"
      >
        <CardHeader
          :title="roleEditorDialog.mode === 'create' ? t('Create Role') : t('Edit Role')"
          :description="t('Create or update one role at a time, then return to the list view.')"
          title-tag="h3"
          title-class="text-lg"
        />

        <div data-role-editor-scroll class="mt-5 min-h-0 flex-1 overflow-y-auto">
          <div class="space-y-5 px-1 py-1 pr-2">
            <section class="rounded-2xl border border-border/60 bg-background/70 p-4">
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Role") }}
              </label>
              <input
                v-model="roleEditorDialog.role"
                data-role-editor-name
                :class="inputClass"
                :placeholder="t('role')"
              />
              <p class="mt-2 text-xs text-muted-foreground">
                {{ t("Role names are saved as authority identifiers.") }}
              </p>
              <div class="mt-3 flex flex-wrap items-center gap-2">
                <Badge v-if="rolePresetPerms(roleEditorDialog.role)" variant="secondary">{{ t("Built-in Baseline") }}</Badge>
                <Button
                  v-if="rolePresetPerms(roleEditorDialog.role)"
                  size="sm"
                  variant="outline"
                  @click="applyRolePresetToDialog"
                >
                  {{ t("Apply Built-in Preset") }}
                </Button>
              </div>
            </section>

            <section class="rounded-2xl border border-border/60 bg-background/70 p-4">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <p class="text-sm font-semibold text-foreground">{{ t("Permission List") }}</p>
                  <p class="mt-1 text-xs text-muted-foreground">
                    {{ t("Choose from the built-in catalog, then remove permissions from this list when they are no longer needed.") }}
                  </p>
                </div>
                <Button
                  size="sm"
                  variant="outline"
                  :disabled="!roleDialogAvailablePermissionOptions.length"
                  @click="openRolePermissionPickerDialog"
                >
                  {{ t("Add Permission") }}
                </Button>
              </div>

              <div v-if="roleEditorDialog.perms.length" class="mt-4 space-y-2">
                <article
                  v-for="(perm, index) in roleEditorDialog.perms"
                  :key="`role-editor-perm-${index}-${perm}`"
                  data-role-perm-row
                  class="rounded-xl border border-border/60 bg-card/80 px-4 py-3"
                >
                  <div class="flex flex-wrap items-center justify-between gap-3">
                    <div class="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1">
                      <p data-role-perm-label class="truncate font-semibold text-foreground">{{ perm }}</p>
                      <p class="text-xs text-muted-foreground">{{ permissionGroupLabel(perm) }}</p>
                    </div>
                    <Button size="sm" variant="ghost" @click="removePermissionAt(roleEditorDialog, index)">
                      {{ t("Remove") }}
                    </Button>
                  </div>
                </article>
              </div>

              <p v-else class="mt-4 text-sm text-muted-foreground">{{ t("No permissions selected yet.") }}</p>
            </section>

            <div
              v-if="roleEditorDialog.unknownPerms.length"
              class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800"
            >
              <p class="font-semibold">{{ t("Preserved extra permissions") }}</p>
              <p class="mt-1">{{ t("These permissions come from existing policy data and stay preserved because they are outside the built-in catalog.") }}</p>
              <div class="mt-3 flex flex-wrap gap-2">
                <Badge
                  v-for="perm in roleEditorDialog.unknownPerms"
                  :key="`role-dialog-extra-${perm}`"
                  variant="secondary"
                >
                  {{ perm }}
                </Badge>
              </div>
            </div>
          </div>
        </div>

        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <Button variant="outline" @click="closeRoleEditorDialog">{{ t("Cancel") }}</Button>
          <Button @click="submitRoleEditorDialog">{{ t("Save") }}</Button>
        </div>
      </div>
    </Overlay>

    <Overlay
      :open="rolePermissionPickerDialog.open"
      overlayClass="bg-black/40 p-4"
      closeOnBackdrop
      trapFocus
      initialFocusSelector="[data-role-picker-select]"
      @close="closeRolePermissionPickerDialog"
    >
      <div
        data-role-permission-picker-dialog
        class="flex max-h-[70vh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl"
      >
        <CardHeader
          :title="t('Select Permission')"
          :description="t('Select one catalog permission to append to this role.')"
          title-tag="h3"
          title-class="text-lg"
        />

        <div class="mt-4 min-h-0 flex-1 overflow-y-auto">
          <div class="space-y-2 pr-2">
            <article
              v-for="option in roleDialogAvailablePermissionOptions"
              :key="`role-picker-${option.perm}`"
              :data-role-picker-option="option.perm"
              class="rounded-xl border border-border/60 bg-background/70 px-4 py-3"
            >
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <div class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
                    <p class="truncate font-semibold text-foreground">{{ option.label }}</p>
                    <p class="text-xs text-muted-foreground">{{ t(option.groupLabel) }}</p>
                  </div>
                  <p class="mt-1 text-xs text-muted-foreground">{{ t(option.description) }}</p>
                </div>
                <Button
                  data-role-picker-select
                  size="sm"
                  variant="outline"
                  @click="addRoleDialogPermission(option.perm)"
                >
                  {{ t("Select") }}
                </Button>
              </div>
            </article>

            <p
              v-if="!roleDialogAvailablePermissionOptions.length"
              class="rounded-xl border border-dashed border-border/60 bg-background/60 px-4 py-6 text-sm text-muted-foreground"
            >
              {{ t("No catalog permissions remain for this role.") }}
            </p>
          </div>
        </div>

        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="closeRolePermissionPickerDialog">{{ t("Cancel") }}</Button>
        </div>
      </div>
    </Overlay>
  </section>
</template>
