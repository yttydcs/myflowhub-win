<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import CardHeader from "@/components/CardHeader.vue"
import { useI18n } from "@/i18n"
import { useAccessPolicyStore } from "@/stores/accessPolicy"
import { useAuthorityStore } from "@/stores/authority"
import type { NodeRole, Policy, RolePerm } from "@/stores/accessPolicy"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"

const sessionStore = useSessionStore()
const authorityStore = useAuthorityStore()
const accessPolicyStore = useAccessPolicyStore()
const toast = useToastStore()
const { t } = useI18n()

const autoLoaded = ref(false)
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
  defaultPerms: "",
  nodeRoles: [] as Array<{ nodeId: string; role: string }>,
  rolePerms: [] as Array<{ role: string; perms: string }>
})

const nodePermsQuery = reactive({
  nodeId: ""
})

const inputClass =
  "mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
const textAreaClass =
  "mt-2 min-h-[96px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

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
  return authorityStore.state.authorityId ? String(authorityStore.state.authorityId) : "-"
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

const summaryCards = computed(() => {
  return [
    {
      label: t("Default Role"),
      value: policyForm.defaultRole.trim() || "-"
    },
    {
      label: t("Default Perms"),
      value: String(splitCsv(policyForm.defaultPerms).length)
    },
    {
      label: t("Node Overrides"),
      value: String(policyForm.nodeRoles.filter((row) => row.nodeId.trim() || row.role.trim()).length)
    },
    {
      label: t("Role Bundles"),
      value: String(policyForm.rolePerms.filter((row) => row.role.trim() || row.perms.trim()).length)
    }
  ]
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
  const policy = accessPolicyStore.state.policy
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
  ([nodeId, hubId], oldValue) => {
    const [prevNodeId, prevHubId] = Array.isArray(oldValue) ? oldValue : []
    authorityStore.setIdentity(Number(nodeId || 0), Number(hubId || 0))
    if (Number(nodeId || 0) !== Number(prevNodeId || 0) || Number(hubId || 0) !== Number(prevHubId || 0)) {
      accessPolicyStore.reset()
      validationErrors.value = []
      nodePermsResult.value = null
      autoLoaded.value = false
    }
  },
  { immediate: true }
)

watch(
  () => ready.value,
  (isReady) => {
    if (!isReady) {
      autoLoaded.value = false
      accessPolicyStore.reset()
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
    <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
      <CardHeader
        :title="t('Access Policy')"
        :description="t('Shape authority defaults, role bundles, and node overrides without leaving the console.')"
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

      <div class="mt-5 flex flex-wrap gap-2">
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
            :description="t('Set the fallback role and baseline permissions that new or unmatched nodes inherit.')"
            title-tag="h3"
            title-class="text-base"
          />

          <div class="mt-4 grid gap-4 md:grid-cols-[240px_minmax(0,1fr)]">
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
              <textarea
                v-model="policyForm.defaultPerms"
                :class="textAreaClass"
                :placeholder="t('file.read,file.write')"
              />
            </div>
          </div>
        </section>

        <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
          <CardHeader
            :title="t('Role Catalog')"
            :description="t('Define named permission bundles first, then map nodes to those bundles only where needed.')"
            title-tag="h3"
            title-class="text-base"
          >
            <template #actions>
              <Button size="sm" variant="outline" @click="addRolePerm">{{ t("Add Role Perm") }}</Button>
            </template>
          </CardHeader>

          <div class="mt-4 space-y-3">
            <div
              v-for="(row, index) in policyForm.rolePerms"
              :key="`role-perm-${index}`"
              class="rounded-2xl border border-border/60 bg-background/70 p-4"
            >
              <div class="grid gap-4 lg:grid-cols-[220px_minmax(0,1fr)_auto]">
                <div>
                  <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    {{ t("Role") }}
                  </label>
                  <input
                    v-model="row.role"
                    class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                    :placeholder="t('role')"
                  />
                </div>
                <div>
                  <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    {{ t("Perms") }}
                  </label>
                  <textarea
                    v-model="row.perms"
                    class="mt-2 min-h-[96px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                    :placeholder="t('file.read,file.write')"
                  />
                </div>
                <div class="self-end">
                  <Button size="sm" variant="outline" @click="removeRolePerm(index)">{{ t("Remove") }}</Button>
                </div>
              </div>
            </div>
            <p v-if="!policyForm.rolePerms.length" class="text-sm text-muted-foreground">
              {{ t("No explicit role permission entries.") }}
            </p>
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
                <input
                  v-model="row.role"
                  class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                  :placeholder="t('role')"
                />
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
              <Badge variant="secondary">{{ t("{count} entries", { count: accessPolicyStore.state.runtimeTotal || accessPolicyStore.state.runtime.length }) }}</Badge>
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

          <div v-if="nodePermsResult" class="mt-4 rounded-2xl border border-border/60 bg-background/70 px-4 py-3 text-sm">
            <p class="font-semibold text-foreground">
              {{ t("Node {nodeId}", { nodeId: nodePermsResult.nodeId }) }}
            </p>
            <p class="mt-1 text-muted-foreground">{{ t("Role: {role}", { role: nodePermsResult.role || "-" }) }}</p>
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
