<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import CardHeader from "@/components/CardHeader.vue"
import PageHero from "@/components/PageHero.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"
import type { NodeRole, Policy, RolePerm } from "@/stores/accessPolicy"
import { useAccessPolicyStore } from "@/stores/accessPolicy"
import {
  accessPolicyPermissionCatalog,
  mergeSelectedAndUnknownPerms,
  normalizeKnownPermissionSelection,
  orderRoleOptions,
  rolePresetPerms,
  splitKnownAndUnknownPerms
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

const tabs: Array<{ id: AccessPolicyTab; label: string }> = [
  { id: "current", label: "Current Policy" },
  { id: "roles", label: "Role Management" }
]

const sessionStore = useSessionStore()
const authorityStore = useAuthorityStore()
const accessPolicyStore = useAccessPolicyStore()
const toast = useToastStore()
const { t } = useI18n()

const activeTab = ref<AccessPolicyTab>("current")
const autoLoaded = ref(false)
const roleDraftName = ref("")
const validationErrors = ref<string[]>([])
const nodePermsResult = ref<null | { nodeId: number; role: string; perms: string[] }>(null)

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

const inputClass =
  "mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
const selectClass = inputClass
const checkboxClass = "mt-0.5 h-4 w-4 rounded border border-input"

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
  if (!authorityStore.state.authorityReason) return String(authorityStore.state.authorityId)
  return `${authorityStore.state.authorityId} (${authorityStore.state.authorityReason})`
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

const extraPermCount = computed(() => {
  let total = policyForm.defaultPermsUnknown.length
  for (const row of policyForm.rolePerms) {
    total += row.unknownPerms.length
  }
  return total
})

const roleSummaryCards = computed(() => {
  return [
    {
      label: t("Roles"),
      value: String(policyForm.rolePerms.length)
    },
    {
      label: t("Built-in Roles"),
      value: String(policyForm.rolePerms.filter((row) => Boolean(rolePresetPerms(row.role))).length)
    },
    {
      label: t("Extra Permissions"),
      value: String(extraPermCount.value)
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

const tabButtonClass = (tab: AccessPolicyTab) => [
  "rounded-full px-4 py-2 text-sm font-semibold transition",
  activeTab.value === tab
    ? "bg-primary text-primary-foreground shadow-sm"
    : "text-muted-foreground hover:bg-muted/70"
]

const hasInvalidSeparator = (token: string) => {
  return token.includes(",") || token.includes(":") || token.includes(";")
}

const eventChecked = (event: Event) => {
  return Boolean((event.target as HTMLInputElement | null)?.checked)
}

const setActiveTab = (tab: AccessPolicyTab) => {
  activeTab.value = tab
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
  roleDraftName.value = ""
}

const resetLocalState = () => {
  accessPolicyStore.reset()
  syncFormFromStore()
  validationErrors.value = []
  nodePermsResult.value = null
  roleDraftName.value = ""
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

const resolveAuthorityAction = async () => {
  try {
    ensureReady()
    const authorityId = await authorityStore.resolveAuthority()
    if (!authorityId) {
      throw new Error(t("Authority ID unresolved."))
    }
    toast.success(
      t("Authority resolved."),
      t("authority={authorityId}", { authorityId })
    )
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to resolve authority."))
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

const addNodeRole = () => {
  policyForm.nodeRoles.push({
    nodeId: "",
    role: policyForm.defaultRole.trim() || roleOptions.value[0] || "node"
  })
}

const removeNodeRole = (index: number) => {
  policyForm.nodeRoles.splice(index, 1)
}

const addRole = () => {
  const role = roleDraftName.value.trim()
  if (!role) {
    toast.warn(t("Role name is required."))
    return
  }
  if (hasInvalidSeparator(role)) {
    toast.warn(t("Role name cannot contain ',', ':', or ';'."))
    return
  }
  if (policyForm.rolePerms.some((row) => row.role.trim() === role)) {
    toast.warn(t("Role '{role}' already exists.", { role }))
    return
  }
  policyForm.rolePerms.push({
    role,
    perms: rolePresetPerms(role) ?? [],
    unknownPerms: []
  })
  roleDraftName.value = ""
}

const removeRole = (index: number) => {
  policyForm.rolePerms.splice(index, 1)
}

const applyRolePreset = (index: number) => {
  const row = policyForm.rolePerms[index]
  const preset = rolePresetPerms(row?.role || "")
  if (!row || !preset) {
    return
  }
  row.perms = preset
}

const toggleSelection = (current: string[], perm: string, checked: boolean) => {
  const next = new Set(current)
  if (perm === "*") {
    return checked ? ["*"] : []
  }
  if (checked) {
    next.delete("*")
    next.add(perm)
  } else {
    next.delete(perm)
  }
  return normalizeKnownPermissionSelection([...next])
}

const onDefaultPermChange = (perm: string, event: Event) => {
  policyForm.defaultPerms = toggleSelection(policyForm.defaultPerms, perm, eventChecked(event))
}

const onRolePermChange = (index: number, perm: string, event: Event) => {
  const row = policyForm.rolePerms[index]
  if (!row) {
    return
  }
  row.perms = toggleSelection(row.perms, perm, eventChecked(event))
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
          :description="t('Review defaults, node overrides, save options, and live runtime state.')"
          title-tag="h2"
          title-class="text-lg"
        >
          <template #actions>
            <Badge variant="secondary">{{ sessionStore.connected ? t("Connected") : t("Disconnected") }}</Badge>
            <Badge variant="secondary">{{ sessionStore.auth.loggedIn ? t("Logged in") : t("Logged out") }}</Badge>
            <Badge variant="secondary">{{ identityLabel }}</Badge>
            <Badge variant="secondary">{{ t("Authority {authority}", { authority: authorityLabel }) }}</Badge>
          </template>
        </CardHeader>

        <div class="mt-5 grid gap-4 xl:grid-cols-[240px_minmax(0,1fr)_auto_auto]">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Authority Override") }}
            </label>
            <input
              v-model="authorityStore.state.authorityOverride"
              :class="inputClass"
              :placeholder="t('Default: hubId')"
            />
          </div>
          <div class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3 text-xs text-muted-foreground">
            <p class="font-semibold text-foreground">{{ t("Resolve Rule") }}</p>
            <p class="mt-1">{{ t("manual override -> authority.node_id -> hubId fallback") }}</p>
            <p class="mt-1">{{ t("Reason: {reason}", { reason: authorityStore.state.authorityReason || "-" }) }}</p>
          </div>
          <div class="self-end">
            <Button
              variant="outline"
              size="sm"
              :disabled="authorityStore.state.resolving || accessPolicyStore.state.loading || accessPolicyStore.state.saving"
              @click="resolveAuthorityAction"
            >
              {{ t("Resolve") }}
            </Button>
          </div>
          <div class="self-end">
            <Button
              size="sm"
              :disabled="authorityStore.state.resolving || accessPolicyStore.state.loading || accessPolicyStore.state.saving"
              @click="loadPolicy(false)"
            >
              {{ t("Load") }}
            </Button>
          </div>
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

      <section class="grid gap-6 xl:grid-cols-[minmax(0,1.1fr)_420px]">
        <div class="space-y-6">
          <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
            <CardHeader
              :title="t('Default Access')"
              :description="t('Set the fallback role and choose default permissions from the built-in catalog instead of hand-typing IDs.')"
              title-tag="h3"
              title-class="text-base"
            >
              <template #actions>
                <Button size="sm" variant="outline" @click="setActiveTab('roles')">{{ t("Manage Roles") }}</Button>
              </template>
            </CardHeader>

            <div class="mt-4 space-y-4">
              <div class="grid gap-4 md:grid-cols-[240px_minmax(0,1fr)]">
                <div>
                  <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    {{ t("Default Role") }}
                  </label>
                  <select v-model="policyForm.defaultRole" :class="selectClass">
                    <option v-for="role in roleOptions" :key="role" :value="role">{{ role }}</option>
                  </select>
                  <p class="mt-2 text-xs text-muted-foreground">
                    {{ t("Create new role names from the role management tab, then select them here.") }}
                  </p>
                </div>
                <div class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3 text-sm text-muted-foreground">
                  <p class="font-semibold text-foreground">{{ t("Permission Catalog") }}</p>
                  <p class="mt-1">
                    {{ t("Select permissions from grouped catalog items. Unknown permissions from existing configs stay preserved.") }}
                  </p>
                </div>
              </div>

              <div class="grid gap-4 lg:grid-cols-2">
                <section
                  v-for="group in accessPolicyPermissionCatalog"
                  :key="group.id"
                  class="rounded-2xl border border-border/60 bg-background/70 p-4"
                >
                  <div class="flex items-start justify-between gap-3">
                    <div>
                      <p class="text-sm font-semibold text-foreground">{{ t(group.label) }}</p>
                      <p class="mt-1 text-xs text-muted-foreground">{{ t(group.description) }}</p>
                    </div>
                    <Badge variant="outline">
                      {{
                        t("{count} selected", {
                          count: group.items.filter((item) => policyForm.defaultPerms.includes(item.perm)).length
                        })
                      }}
                    </Badge>
                  </div>

                  <div class="mt-4 space-y-2">
                    <label
                      v-for="item in group.items"
                      :key="item.perm"
                      class="flex items-start gap-3 rounded-xl border border-border/60 bg-card/70 px-3 py-3 text-sm"
                    >
                      <input
                        :checked="policyForm.defaultPerms.includes(item.perm)"
                        type="checkbox"
                        :class="checkboxClass"
                        @change="onDefaultPermChange(item.perm, $event)"
                      />
                      <div class="min-w-0">
                        <p class="font-medium text-foreground">{{ item.label }}</p>
                        <p class="mt-1 text-xs text-muted-foreground">{{ t(item.description) }}</p>
                      </div>
                    </label>
                  </div>
                </section>
              </div>

              <div
                v-if="policyForm.defaultPermsUnknown.length"
                class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800"
              >
                <p class="font-semibold">{{ t("Extra Permissions") }}</p>
                <p class="mt-1">{{ t("These permissions are outside the built-in catalog and will be preserved on save.") }}</p>
                <div class="mt-3 flex flex-wrap gap-2">
                  <Badge
                    v-for="perm in policyForm.defaultPermsUnknown"
                    :key="`default-extra-${perm}`"
                    variant="secondary"
                  >
                    {{ perm }}
                  </Badge>
                </div>
              </div>
            </div>
          </section>

          <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
            <CardHeader
              :title="t('Node Overrides')"
              :description="t('Pin only the exceptional nodes. Most nodes should stay on the default role path.')"
              title-tag="h3"
              title-class="text-base"
            >
              <template #actions>
                <Button size="sm" @click="addNodeRole">{{ t("Add Node Role") }}</Button>
              </template>
            </CardHeader>

            <div class="mt-4 space-y-3">
              <div
                v-for="(row, index) in policyForm.nodeRoles"
                :key="`node-role-${index}`"
                class="grid gap-4 rounded-2xl border border-border/60 bg-background/70 p-4 lg:grid-cols-[160px_minmax(0,1fr)_auto]"
              >
                <div>
                  <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    {{ t("Node ID") }}
                  </label>
                  <input
                    v-model="row.nodeId"
                    class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                    :placeholder="t('nodeId')"
                  />
                </div>
                <div>
                  <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    {{ t("Role") }}
                  </label>
                  <select v-model="row.role" class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm">
                    <option v-for="role in roleOptions" :key="`node-role-option-${role}`" :value="role">
                      {{ role }}
                    </option>
                  </select>
                </div>
                <div class="self-end">
                  <Button size="sm" variant="outline" @click="removeNodeRole(index)">{{ t("Remove") }}</Button>
                </div>
              </div>
              <p v-if="!policyForm.nodeRoles.length" class="text-sm text-muted-foreground">
                {{ t("No explicit node role entries.") }}
              </p>
            </div>
          </section>
        </div>

        <div class="space-y-6">
          <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
            <CardHeader
              :title="t('Save Panel')"
              :description="t('Choose how far to push the change: persist only, runtime only, or full rollout plus verification.')"
              title-tag="h3"
              title-class="text-base"
            />

            <div class="mt-4 space-y-3 rounded-2xl border border-border/60 bg-background/70 p-4">
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
              <label class="flex items-center gap-2 text-sm text-foreground">
                <input v-model="saveOptions.verifyRuntime" type="checkbox" class="h-4 w-4 rounded border border-input" />
                {{ t("Verify runtime by list_roles after save") }}
              </label>
            </div>

            <div class="mt-4 flex flex-wrap gap-2">
              <Button :disabled="accessPolicyStore.state.saving" @click="savePolicy">{{ t("Save Policy") }}</Button>
              <Button
                variant="outline"
                :disabled="accessPolicyStore.state.loading || accessPolicyStore.state.saving"
                @click="loadPolicy(false)"
              >
                {{ t("Reload Policy") }}
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

          <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
            <CardHeader
              :title="t('Runtime Snapshot')"
              :description="t('Use the live authority response to verify what the network is currently enforcing.')"
              title-tag="h3"
              title-class="text-base"
            >
              <template #actions>
                <Badge variant="secondary">
                  {{ t("{count} entries", { count: accessPolicyStore.state.runtimeTotal || accessPolicyStore.state.runtime.length }) }}
                </Badge>
              </template>
            </CardHeader>

            <p class="mt-4 text-sm text-muted-foreground">{{ runtimeSummary }}</p>
            <p v-if="accessPolicyStore.state.runtimeError" class="mt-2 text-sm text-amber-700">
              {{ accessPolicyStore.state.runtimeError }}
            </p>

            <div class="mt-4 max-h-[360px] space-y-2 overflow-y-auto">
              <div
                v-for="entry in runtimePreview"
                :key="`runtime-${entry.nodeId}`"
                class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3 text-sm"
              >
                <p class="font-semibold text-foreground">
                  {{ t("Node {nodeId} -> {role}", { nodeId: entry.nodeId, role: entry.role || "-" }) }}
                </p>
                <p class="mt-1 break-all text-muted-foreground">
                  {{ entry.perms.length ? entry.perms.join(",") : t("(no perms)") }}
                </p>
              </div>
              <p v-if="!runtimePreview.length" class="text-sm text-muted-foreground">
                {{ t("No runtime entries.") }}
              </p>
            </div>
          </section>

          <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
            <CardHeader
              :title="t('Node Perms Lookup')"
              :description="t('Query one node directly when you need to confirm the final role and merged permissions.')"
              title-tag="h3"
              title-class="text-base"
            />

            <div class="mt-4 grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto]">
              <input v-model="nodePermsQuery.nodeId" :class="inputClass" :placeholder="t('Node ID')" />
              <div class="self-end">
                <Button variant="outline" size="sm" @click="queryNodePerms">{{ t("Query") }}</Button>
              </div>
            </div>

            <div
              v-if="nodePermsResult"
              class="mt-4 rounded-2xl border border-border/60 bg-background/70 px-4 py-3 text-sm"
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
          </section>
        </div>
      </section>
    </section>

    <section v-else class="space-y-6">
      <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
        <CardHeader
          :title="t('Role Management')"
          :description="t('Maintain named roles through a permission catalog instead of raw CSV editing.')"
          title-tag="h2"
          title-class="text-lg"
        >
          <template #actions>
            <Badge variant="secondary">{{ t("{count} roles", { count: policyForm.rolePerms.length }) }}</Badge>
          </template>
        </CardHeader>

        <div class="mt-5 grid gap-4 xl:grid-cols-[280px_minmax(0,1fr)_auto]">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("New role name") }}
            </label>
            <input
              v-model="roleDraftName"
              :class="inputClass"
              :placeholder="t('role')"
              @keydown.enter.prevent="addRole"
            />
          </div>
          <div class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3 text-sm text-muted-foreground">
            <p class="font-semibold text-foreground">{{ t("Built-in role presets") }}</p>
            <p class="mt-1">{{ t("Role names 'superadmin', 'admin', and 'node' start with their current documented default permissions.") }}</p>
          </div>
          <div class="self-end">
            <Button size="sm" @click="addRole">{{ t("Add Role") }}</Button>
          </div>
        </div>

        <div class="mt-5 grid gap-3 md:grid-cols-3">
          <div
            v-for="card in roleSummaryCards"
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

      <section v-if="policyForm.rolePerms.length" class="space-y-4">
        <section
          v-for="(row, index) in policyForm.rolePerms"
          :key="`role-perm-${index}`"
          class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm"
        >
          <CardHeader
            :title="row.role.trim() || t('Unnamed Role')"
            :description="t('Choose permissions from grouped catalog entries. Unknown permissions remain attached until you change them elsewhere.')"
            title-tag="h3"
            title-class="text-base"
          >
            <template #actions>
              <Badge variant="secondary">
                {{ t("{count} selected", { count: row.perms.length + row.unknownPerms.length }) }}
              </Badge>
              <Button
                v-if="rolePresetPerms(row.role)"
                size="sm"
                variant="outline"
                @click="applyRolePreset(index)"
              >
                {{ t("Apply Built-in Preset") }}
              </Button>
              <Button size="sm" variant="outline" @click="removeRole(index)">{{ t("Remove") }}</Button>
            </template>
          </CardHeader>

          <div class="mt-4 space-y-4">
            <div class="grid gap-4 md:grid-cols-[280px_minmax(0,1fr)]">
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Role") }}
                </label>
                <input v-model="row.role" :class="inputClass" :placeholder="t('role')" />
                <p class="mt-2 text-xs text-muted-foreground">
                  {{ t("Role names are saved as authority identifiers.") }}
                </p>
                <p v-if="rolePresetPerms(row.role)" class="mt-2 text-xs text-emerald-700">
                  {{ t("This role matches a built-in baseline. Re-apply it anytime if you want the recommended permission set.") }}
                </p>
              </div>
              <div class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3 text-sm text-muted-foreground">
                <p class="font-semibold text-foreground">{{ t("Permission Catalog") }}</p>
                <p class="mt-1">
                  {{ t("Check the permissions this role should grant. Wildcard '*' clears the other catalog selections.") }}
                </p>
              </div>
            </div>

            <div class="grid gap-4 lg:grid-cols-2">
              <section
                v-for="group in accessPolicyPermissionCatalog"
                :key="`${row.role || index}-${group.id}`"
                class="rounded-2xl border border-border/60 bg-background/70 p-4"
              >
                <div class="flex items-start justify-between gap-3">
                  <div>
                    <p class="text-sm font-semibold text-foreground">{{ t(group.label) }}</p>
                    <p class="mt-1 text-xs text-muted-foreground">{{ t(group.description) }}</p>
                  </div>
                  <Badge variant="outline">
                    {{
                      t("{count} selected", {
                        count: group.items.filter((item) => row.perms.includes(item.perm)).length
                      })
                    }}
                  </Badge>
                </div>

                <div class="mt-4 space-y-2">
                  <label
                    v-for="item in group.items"
                    :key="item.perm"
                    class="flex items-start gap-3 rounded-xl border border-border/60 bg-card/70 px-3 py-3 text-sm"
                  >
                    <input
                      :checked="row.perms.includes(item.perm)"
                      type="checkbox"
                      :class="checkboxClass"
                      @change="onRolePermChange(index, item.perm, $event)"
                    />
                    <div class="min-w-0">
                      <p class="font-medium text-foreground">{{ item.label }}</p>
                      <p class="mt-1 text-xs text-muted-foreground">{{ t(item.description) }}</p>
                    </div>
                  </label>
                </div>
              </section>
            </div>

            <div
              v-if="row.unknownPerms.length"
              class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800"
            >
              <p class="font-semibold">{{ t("Extra Permissions") }}</p>
              <p class="mt-1">
                {{ t("These permissions are outside the built-in catalog and will be preserved on save.") }}
              </p>
              <div class="mt-3 flex flex-wrap gap-2">
                <Badge
                  v-for="perm in row.unknownPerms"
                  :key="`role-extra-${index}-${perm}`"
                  variant="secondary"
                >
                  {{ perm }}
                </Badge>
              </div>
            </div>
          </div>
        </section>
      </section>

      <section
        v-else
        class="rounded-2xl border border-dashed bg-card/70 p-6 text-sm text-muted-foreground shadow-sm"
      >
        {{ t("No explicit roles. Add a role to start managing named access bundles.") }}
      </section>
    </section>
  </section>
</template>
