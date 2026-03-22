<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"
import type { NodeRole, Policy, RolePerm } from "@/stores/permissions"
import { usePermissionsStore } from "@/stores/permissions"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"

const sessionStore = useSessionStore()
const permissionStore = usePermissionsStore()
const toast = useToastStore()
const { t } = useI18n()

const autoLoaded = ref(false)
const validationErrors = ref<string[]>([])

const saveOptions = reactive({
  persist: true,
  applyRuntime: true,
  invalidate: true,
  refresh: true,
  verifyRuntime: true
})

const policyForm = reactive({
  defaultRole: "node",
  defaultPerms: "",
  nodeRoles: [] as Array<{ nodeId: string; role: string }>,
  rolePerms: [] as Array<{ role: string; perms: string }>
})

const nodePermsQuery = reactive({
  nodeId: ""
})

const nodePermsResult = ref<null | { nodeId: number; role: string; perms: string[] }>(null)

const inputClass =
  "mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

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

const runtimePreview = computed(() => {
  const list = Array.isArray(permissionStore.state.runtime) ? permissionStore.state.runtime : []
  return list.slice(0, 120)
})

const runtimeSummary = computed(() => {
  if (permissionStore.state.runtimeError) {
    return t("Runtime verify failed: {error}", {
      error: permissionStore.state.runtimeError
    })
  }
  const total = Number(permissionStore.state.runtimeTotal || permissionStore.state.runtime.length || 0)
  return t("Runtime entries: {count}", { count: total })
})

const authorityLabel = computed(() => {
  if (!permissionStore.state.authorityId) return "-"
  if (!permissionStore.state.authorityReason) return String(permissionStore.state.authorityId)
  return `${permissionStore.state.authorityId} (${permissionStore.state.authorityReason})`
})

const hasInvalidSeparator = (token: string) => {
  return token.includes(",") || token.includes(":") || token.includes(";")
}

const splitCsv = (raw: string) => {
  const out: string[] = []
  const seen = new Set<string>()
  for (const part of String(raw || "").split(",")) {
    const trimmed = part.trim()
    if (!trimmed || seen.has(trimmed)) {
      continue
    }
    seen.add(trimmed)
    out.push(trimmed)
  }
  return out
}

const syncFormFromStore = () => {
  const policy = permissionStore.state.policy
  policyForm.defaultRole = String(policy?.defaultRole || "node")
  policyForm.defaultPerms = Array.isArray(policy?.defaultPerms) ? policy.defaultPerms.join(",") : ""
  policyForm.nodeRoles = Array.isArray(policy?.nodeRoles)
    ? policy.nodeRoles.map((entry) => ({
        nodeId: entry?.nodeId ? String(entry.nodeId) : "",
        role: String(entry?.role || "")
      }))
    : []
  policyForm.rolePerms = Array.isArray(policy?.rolePerms)
    ? policy.rolePerms.map((entry) => ({
        role: String(entry?.role || ""),
        perms: Array.isArray(entry?.perms) ? entry.perms.join(",") : ""
      }))
    : []
}

const assignStorePolicy = (policy: Policy) => {
  permissionStore.state.policy.defaultRole = policy.defaultRole
  permissionStore.state.policy.defaultPerms = [...policy.defaultPerms]
  permissionStore.state.policy.nodeRoles = policy.nodeRoles.map((entry) => ({
    nodeId: Number(entry.nodeId || 0),
    role: entry.role
  }))
  permissionStore.state.policy.rolePerms = policy.rolePerms.map((entry) => ({
    role: entry.role,
    perms: [...entry.perms]
  }))
}

const buildPolicyFromForm = (): { policy?: Policy; errors: string[] } => {
  const errors: string[] = []

  const defaultRole = policyForm.defaultRole.trim()
  if (!defaultRole) {
    errors.push(t("Default role is required."))
  } else if (hasInvalidSeparator(defaultRole)) {
    errors.push(t("Default role cannot contain ',', ':', or ';'."))
  }

  const defaultPermTokens = splitCsv(policyForm.defaultPerms)
  for (const token of defaultPermTokens) {
    if (hasInvalidSeparator(token)) {
      errors.push(t("Default perm '{token}' contains invalid separator.", { token }))
      break
    }
  }

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
    const permsRaw = String(row.perms || "")

    if (!role && !permsRaw.trim()) {
      continue
    }

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

    const perms = splitCsv(permsRaw)
    let invalid = ""
    for (const perm of perms) {
      if (hasInvalidSeparator(perm)) {
        invalid = perm
        break
      }
    }
    if (invalid) {
      errors.push(
        t("Role perms row {rowNo}: perm '{perm}' contains invalid separator.", {
          rowNo,
          perm: invalid
        })
      )
      continue
    }

    roleSeen.add(role)
    rolePerms.push({ role, perms })
  }

  if (errors.length) {
    return { errors }
  }

  return {
    errors,
    policy: {
      defaultRole,
      defaultPerms: defaultPermTokens,
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

const resolveAuthority = async (silent = false) => {
  ensureReady()
  const authorityId = await permissionStore.resolveAuthority()
  if (!authorityId) {
    throw new Error(t("Authority ID unresolved."))
  }
  if (!silent) {
    toast.success(
      t("Authority resolved."),
      t("authority={authorityId}", { authorityId })
    )
  }
}

const resolveAuthorityAction = async () => {
  try {
    await resolveAuthority(false)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to resolve authority."))
  }
}

const loadPolicy = async (silent = false) => {
  try {
    ensureReady()
    validationErrors.value = []
    await resolveAuthority(true)
    await permissionStore.loadPolicy()
    syncFormFromStore()
    if (!silent) {
      toast.success(
        t("Policy loaded."),
        t("authority={authorityId}", { authorityId: permissionStore.state.authorityId })
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

    const resp = await permissionStore.savePolicy({
      persist: saveOptions.persist,
      applyRuntime: saveOptions.applyRuntime,
      invalidate: saveOptions.invalidate,
      refresh: saveOptions.refresh,
      verifyRuntime: saveOptions.verifyRuntime
    })

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
    const result = await permissionStore.getNodePerms(parsed)
    nodePermsResult.value = {
      nodeId: result.nodeId,
      role: result.role,
      perms: result.perms
    }
    toast.success(t("Node perms loaded."), t("node={nodeId}", { nodeId: result.nodeId }))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to query node perms."))
  }
}

const addNodeRole = () => {
  policyForm.nodeRoles.push({ nodeId: "", role: "" })
}

const removeNodeRole = (index: number) => {
  policyForm.nodeRoles.splice(index, 1)
}

const addRolePerm = () => {
  policyForm.rolePerms.push({ role: "", perms: "" })
}

const removeRolePerm = (index: number) => {
  policyForm.rolePerms.splice(index, 1)
}

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  ([nodeId, hubId]) => {
    permissionStore.setIdentity(Number(nodeId || 0), Number(hubId || 0))
  },
  { immediate: true }
)

watch(
  () => ready.value,
  (isReady) => {
    if (!isReady) {
      autoLoaded.value = false
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
  if (ready.value && !autoLoaded.value) {
    autoLoaded.value = true
    void loadPolicy(true)
  }
})
</script>

<template>
  <section class="space-y-6">
    <section class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-sm font-semibold">{{ t("Authority Permissions") }}</h2>
          <p class="text-xs text-muted-foreground">
            {{ t("V1 uses config_set + perms_snapshot (+optional invalidate).") }}
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <Badge variant="secondary">{{ sessionStore.connected ? t("Connected") : t("Disconnected") }}</Badge>
          <Badge variant="secondary">{{ sessionStore.auth.loggedIn ? t("Logged in") : t("Logged out") }}</Badge>
          <Badge variant="secondary">{{ identityLabel }}</Badge>
          <Badge variant="secondary">{{ t("Authority {authority}", { authority: authorityLabel }) }}</Badge>
        </div>
      </div>

      <div class="mt-4 grid gap-3 md:grid-cols-[220px_minmax(0,1fr)_auto_auto]">
        <div>
          <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            {{ t("Authority Override") }}
          </label>
          <input
            v-model="permissionStore.state.authorityOverride"
            :class="inputClass"
            :placeholder="t('Default: hubId')"
          />
        </div>
        <div class="rounded-xl border border-border/60 bg-background/70 px-3 py-2 text-xs text-muted-foreground">
          <p class="font-semibold text-foreground">{{ t("Resolve Rule") }}</p>
          <p class="mt-1">{{ t("manual override -> authority.node_id -> hubId fallback") }}</p>
          <p class="mt-1">
            {{ t("Reason: {reason}", { reason: permissionStore.state.authorityReason || "-" }) }}
          </p>
        </div>
        <div class="self-end">
          <Button
            variant="outline"
            size="sm"
            :disabled="permissionStore.state.loading || permissionStore.state.saving"
            @click="resolveAuthorityAction"
          >
            {{ t("Resolve") }}
          </Button>
        </div>
        <div class="self-end">
          <Button
            size="sm"
            :disabled="permissionStore.state.loading || permissionStore.state.saving"
            @click="loadPolicy(false)"
          >
            {{ t("Load") }}
          </Button>
        </div>
      </div>

      <p class="mt-3 text-xs text-muted-foreground">{{ runtimeSummary }}</p>
      <p v-if="permissionStore.state.runtimeError" class="mt-2 text-xs text-amber-700">
        {{ permissionStore.state.runtimeError }}
      </p>
    </section>

    <section class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div class="space-y-6">
        <section class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h3 class="text-sm font-semibold">{{ t("Policy Editor") }}</h3>
              <p class="text-xs text-muted-foreground">
                {{ t("Edit persisted policy fields and push runtime snapshot.") }}
              </p>
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <Button size="sm" @click="addNodeRole">{{ t("Add Node Role") }}</Button>
              <Button size="sm" variant="outline" @click="addRolePerm">{{ t("Add Role Perm") }}</Button>
            </div>
          </div>

          <div class="mt-4 grid gap-4 md:grid-cols-2">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Default Role") }}
              </label>
              <input v-model="policyForm.defaultRole" :class="inputClass" :placeholder="t('node')" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Default Perms (CSV)") }}
              </label>
              <input v-model="policyForm.defaultPerms" :class="inputClass" :placeholder="t('file.read,file.write')" />
            </div>
          </div>

          <div class="mt-6">
            <div class="flex items-center justify-between gap-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Node Roles") }}
              </p>
              <p class="text-xs text-muted-foreground">{{ t("Format: nodeId -> role") }}</p>
            </div>
            <div class="mt-2 space-y-2">
              <div
                v-for="(row, index) in policyForm.nodeRoles"
                :key="`node-role-${index}`"
                class="grid gap-2 rounded-xl border border-border/60 bg-background/70 p-3 md:grid-cols-[140px_minmax(0,1fr)_auto]"
              >
                <input
                  v-model="row.nodeId"
                  class="h-9 rounded-md border border-input bg-background px-3 text-sm"
                  :placeholder="t('nodeId')"
                />
                <input
                  v-model="row.role"
                  class="h-9 rounded-md border border-input bg-background px-3 text-sm"
                  :placeholder="t('role')"
                />
                <Button size="sm" variant="outline" @click="removeNodeRole(index)">{{ t("Remove") }}</Button>
              </div>
              <p v-if="!policyForm.nodeRoles.length" class="text-xs text-muted-foreground">
                {{ t("No explicit node role entries.") }}
              </p>
            </div>
          </div>

          <div class="mt-6">
            <div class="flex items-center justify-between gap-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Role Perms") }}
              </p>
              <p class="text-xs text-muted-foreground">{{ t("Format: role -> perms(csv)") }}</p>
            </div>
            <div class="mt-2 space-y-2">
              <div
                v-for="(row, index) in policyForm.rolePerms"
                :key="`role-perm-${index}`"
                class="grid gap-2 rounded-xl border border-border/60 bg-background/70 p-3 md:grid-cols-[160px_minmax(0,1fr)_auto]"
              >
                <input
                  v-model="row.role"
                  class="h-9 rounded-md border border-input bg-background px-3 text-sm"
                  :placeholder="t('role')"
                />
                <input
                  v-model="row.perms"
                  class="h-9 rounded-md border border-input bg-background px-3 text-sm"
                  :placeholder="t('file.read,file.write')"
                />
                <Button size="sm" variant="outline" @click="removeRolePerm(index)">{{ t("Remove") }}</Button>
              </div>
              <p v-if="!policyForm.rolePerms.length" class="text-xs text-muted-foreground">
                {{ t("No explicit role permission entries.") }}
              </p>
            </div>
          </div>

          <div class="mt-6 rounded-xl border border-border/60 bg-background/70 p-3">
            <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Save Options") }}
            </p>
            <div class="mt-3 grid gap-3 sm:grid-cols-2">
              <label class="flex items-center gap-2 text-xs text-foreground">
                <input v-model="saveOptions.persist" type="checkbox" class="h-4 w-4 rounded border border-input" />
                {{ t("Persist auth.* config") }}
              </label>
              <label class="flex items-center gap-2 text-xs text-foreground">
                <input v-model="saveOptions.applyRuntime" type="checkbox" class="h-4 w-4 rounded border border-input" />
                {{ t("Push perms_snapshot") }}
              </label>
              <label class="flex items-center gap-2 text-xs text-foreground">
                <input v-model="saveOptions.invalidate" type="checkbox" class="h-4 w-4 rounded border border-input" />
                {{ t("perms_invalidate") }}
              </label>
              <label class="flex items-center gap-2 text-xs text-foreground">
                <input v-model="saveOptions.refresh" type="checkbox" class="h-4 w-4 rounded border border-input" />
                {{ t("invalidate.refresh=true") }}
              </label>
              <label class="flex items-center gap-2 text-xs text-foreground sm:col-span-2">
                <input v-model="saveOptions.verifyRuntime" type="checkbox" class="h-4 w-4 rounded border border-input" />
                {{ t("Verify runtime by list_roles after save") }}
              </label>
            </div>

            <div class="mt-4 flex flex-wrap gap-2">
              <Button :disabled="permissionStore.state.saving" @click="savePolicy">{{ t("Save Policy") }}</Button>
              <Button
                variant="outline"
                :disabled="permissionStore.state.loading || permissionStore.state.saving"
                @click="loadPolicy(false)"
              >
                {{ t("Reload Policy") }}
              </Button>
            </div>
          </div>

          <div v-if="validationErrors.length" class="mt-4 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-700">
            <p class="font-semibold">{{ t("Validation Errors") }}</p>
            <ul class="mt-2 list-disc space-y-1 pl-4">
              <li v-for="(msg, index) in validationErrors" :key="`validation-${index}`">{{ msg }}</li>
            </ul>
          </div>

          <div
            v-if="permissionStore.state.warnings.length"
            class="mt-4 rounded-xl border border-amber-200 bg-amber-50 p-3 text-xs text-amber-700"
          >
            <p class="font-semibold">{{ t("Policy Warnings") }}</p>
            <ul class="mt-2 list-disc space-y-1 pl-4">
              <li v-for="(msg, index) in permissionStore.state.warnings" :key="`warning-${index}`">{{ msg }}</li>
            </ul>
          </div>
        </section>
      </div>

      <div class="space-y-6">
        <section class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h3 class="text-sm font-semibold">{{ t("Runtime Roles") }}</h3>
              <p class="text-xs text-muted-foreground">
                {{ t("Current list_roles snapshot from authority.") }}
              </p>
            </div>
            <Badge variant="secondary">
              {{ t("{count} entries", { count: permissionStore.state.runtimeTotal || permissionStore.state.runtime.length }) }}
            </Badge>
          </div>

          <div class="mt-3 max-h-[380px] space-y-2 overflow-y-auto">
            <div
              v-for="entry in runtimePreview"
              :key="`runtime-${entry.nodeId}`"
              class="rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-xs"
            >
              <p class="font-semibold text-foreground">
                {{ t("Node {nodeId} -> {role}", { nodeId: entry.nodeId, role: entry.role || "-" }) }}
              </p>
              <p class="mt-1 break-all text-muted-foreground">
                {{ entry.perms.length ? entry.perms.join(",") : t("(no perms)") }}
              </p>
            </div>
            <p v-if="!runtimePreview.length" class="text-xs text-muted-foreground">
              {{ t("No runtime entries.") }}
            </p>
          </div>

          <p
            v-if="permissionStore.state.runtime.length > runtimePreview.length"
            class="mt-2 text-xs text-muted-foreground"
          >
            {{ t("Showing first {count} entries.", { count: runtimePreview.length }) }}
          </p>
        </section>

        <section class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
          <h3 class="text-sm font-semibold">{{ t("Node Perms Query") }}</h3>
          <p class="text-xs text-muted-foreground">{{ t("Call auth.get_perms for a single node.") }}</p>

          <div class="mt-3 grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto]">
            <input v-model="nodePermsQuery.nodeId" :class="inputClass" :placeholder="t('Node ID')" />
            <div class="self-end">
              <Button variant="outline" size="sm" @click="queryNodePerms">{{ t("Query") }}</Button>
            </div>
          </div>

          <div v-if="nodePermsResult" class="mt-3 rounded-xl border border-border/60 bg-background/70 px-3 py-3 text-xs">
            <p class="font-semibold text-foreground">
              {{ t("Node {nodeId}", { nodeId: nodePermsResult.nodeId }) }}
            </p>
            <p class="mt-1 text-muted-foreground">
              {{ t("Role: {role}", { role: nodePermsResult.role || "-" }) }}
            </p>
            <p class="mt-1 break-all text-muted-foreground">
              {{
                t("Perms: {perms}", {
                  perms: nodePermsResult.perms.length
                    ? nodePermsResult.perms.join(",")
                    : t("(no perms)")
                })
              }}
            </p>
          </div>
        </section>
      </div>
    </section>
  </section>
</template>
