<script setup lang="ts">
import { computed, reactive, watch } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import CardHeader from "@/components/CardHeader.vue"
import { useI18n } from "@/i18n"
import { useAuthorityStore } from "@/stores/authority"
import { usePermitIssuanceStore } from "@/stores/permitIssuance"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"

const sessionStore = useSessionStore()
const authorityStore = useAuthorityStore()
const permitStore = usePermitIssuanceStore()
const toast = useToastStore()
const { t } = useI18n()

const autoLoaded = reactive({ value: false })
const loadState = reactive({
  error: ""
})
const dialogs = reactive({
  issueOpen: false
})

const issueForm = reactive({
  deviceId: "",
  role: "",
  expiresAt: ""
})

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

const currentNodeId = computed(() => Number(sessionStore.auth.nodeId || 0))

const identityLabel = computed(() => {
  const nodeId = currentNodeId.value
  const hubId = Number(sessionStore.auth.hubId || 0)
  if (!nodeId || !hubId) {
    return t("Not logged in")
  }
  return t("node={nodeId} hub={hubId}", { nodeId, hubId })
})

const authorityLabel = computed(() => {
  return authorityStore.state.authorityId ? String(authorityStore.state.authorityId) : "-"
})

const loadErrorDetail = computed(() => {
  return loadState.error.trim()
})

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

const resetIssueForm = () => {
  issueForm.deviceId = ""
  issueForm.role = ""
  issueForm.expiresAt = ""
}

const formatTimestamp = (value: number) => {
  const numeric = Number(value || 0)
  if (!numeric) return "-"
  const millis = numeric > 100000000000 ? numeric : numeric * 1000
  const date = new Date(millis)
  if (Number.isNaN(date.getTime())) return "-"
  return date.toLocaleString()
}

const parseExpiresAt = () => {
  const raw = issueForm.expiresAt.trim()
  if (!raw) return 0
  const millis = Date.parse(raw)
  if (Number.isNaN(millis) || millis <= 0) {
    throw new Error(t("Expires at must be a valid date time."))
  }
  return Math.floor(millis / 1000)
}

const openIssueDialog = () => {
  dialogs.issueOpen = true
}

const closeIssueDialog = () => {
  dialogs.issueOpen = false
}

const loadPermits = async () => {
  loadState.error = ""
  try {
    ensureReady()
    await permitStore.loadPermits()
    loadState.error = ""
  } catch (err) {
    console.warn(err)
    loadState.error = err instanceof Error ? err.message.trim() : String(err || "").trim()
  }
}

const refreshPermits = async () => {
  await loadPermits()
}

const issuePermit = async () => {
  try {
    ensureReady()
    const deviceId = issueForm.deviceId.trim()
    const role = issueForm.role.trim()
    if (!deviceId) {
      throw new Error(t("Device ID is required."))
    }
    if (!role) {
      throw new Error(t("Role is required."))
    }
    const issued = await permitStore.issuePermit({
      deviceId,
      role,
      expiresAt: parseExpiresAt()
    })
    closeIssueDialog()
    resetIssueForm()
    toast.success(
      t("Permit issued."),
      t("device={deviceId} role={role}", { deviceId: issued.deviceId, role: issued.role })
    )
    await loadPermits()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to issue permit."))
  }
}

const revokePermit = async (permit: string) => {
  try {
    ensureReady()
    const token = String(permit || "").trim()
    if (!token) {
      throw new Error(t("Permit token is required."))
    }
    const result = await permitStore.revokePermit(token)
    toast.success(t("Permit revoked."), result.permit)
    await loadPermits()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to revoke permit."))
  }
}

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  ([nodeId, hubId], oldValue) => {
    const [prevNodeId, prevHubId] = Array.isArray(oldValue) ? oldValue : []
    authorityStore.setIdentity(Number(nodeId || 0), Number(hubId || 0))
    if (Number(nodeId || 0) !== Number(prevNodeId || 0) || Number(hubId || 0) !== Number(prevHubId || 0)) {
      permitStore.reset()
      autoLoaded.value = false
      loadState.error = ""
      dialogs.issueOpen = false
      resetIssueForm()
    }
  },
  { immediate: true }
)

watch(
  () => ready.value,
  async (isReady) => {
    if (!isReady) {
      autoLoaded.value = false
      permitStore.reset()
      loadState.error = ""
      dialogs.issueOpen = false
      resetIssueForm()
      return
    }
    if (autoLoaded.value) {
      return
    }
    autoLoaded.value = true
    await loadPermits()
  },
  { immediate: true }
)
</script>

<template>
  <section class="space-y-6">
    <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
      <CardHeader
        :title="t('Permit Issuance')"
        :description="t('Review active one-time join permits, create new ones, and revoke them before they are consumed.')"
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

      <p class="mt-4 text-sm text-muted-foreground">
        {{ t("Only active permits are listed here. Consumed, revoked, or expired permits disappear automatically.") }}
      </p>
    </section>

    <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
      <CardHeader
        :title="t('Active Permits')"
        :description="t('Use the list below as the single source of truth for current join permits on this authority.')"
        title-tag="h3"
        title-class="text-base"
      >
        <template #actions>
          <div data-permit-card-actions class="flex flex-wrap items-center gap-2">
            <Button
              data-refresh-permits
              size="sm"
              variant="outline"
              :disabled="!ready || permitStore.state.loading"
              @click="refreshPermits"
            >
              {{ t("Refresh") }}
            </Button>
            <Button
              data-open-issue-dialog
              size="sm"
              :disabled="!ready"
              @click="openIssueDialog"
            >
              {{ t("New Permit") }}
            </Button>
          </div>
        </template>
      </CardHeader>

      <div
        v-if="loadErrorDetail"
        data-permit-load-error
        class="mt-4 rounded-2xl border border-amber-200/70 bg-amber-50/80 px-4 py-3 text-sm text-amber-950"
      >
        <p class="font-medium">{{ t("Failed to load permits.") }}</p>
        <p class="mt-2 break-all text-xs text-amber-900/80">{{ loadErrorDetail }}</p>
      </div>

      <div
        v-if="permitStore.state.loading"
        data-permit-loading
        class="mt-4 rounded-2xl border border-dashed border-border/60 bg-background/50 px-5 py-8 text-center text-sm text-muted-foreground"
      >
        {{ t("Loading active permits...") }}
      </div>

      <div
        v-else-if="!permitStore.state.items.length && !loadErrorDetail"
        data-permit-empty
        class="mt-4 rounded-2xl border border-dashed border-border/60 bg-background/50 px-5 py-8 text-center text-sm text-muted-foreground"
      >
        {{ t("No active permits.") }}
      </div>

      <div
        v-else
        data-permit-list
        class="mt-4 overflow-hidden rounded-2xl border border-border/60 bg-background/70"
      >
        <article
          v-for="item in permitStore.state.items"
          :key="item.permit"
          data-permit-row
          class="flex flex-wrap items-center justify-between gap-4 border-b border-border/60 px-4 py-4 last:border-b-0"
        >
          <div class="min-w-0 flex-1">
            <p
              data-permit-token
              class="truncate font-mono text-xs text-foreground"
              :title="item.permit"
            >
              {{ item.permit }}
            </p>
            <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-sm text-muted-foreground">
              <span>{{ t("Device {deviceId}", { deviceId: item.deviceId || "-" }) }}</span>
              <span>{{ t("Role {role}", { role: item.role || "-" }) }}</span>
              <span>{{ t("Issued {time}", { time: formatTimestamp(item.issuedAt) }) }}</span>
              <span>{{ t("Expires {time}", { time: formatTimestamp(item.expiresAt) }) }}</span>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <Button
              data-row-revoke
              size="sm"
              variant="outline"
              :disabled="permitStore.state.busyPermit !== ''"
              @click="revokePermit(item.permit)"
            >
              {{ permitStore.state.busyPermit === item.permit ? t("Revoking...") : t("Revoke") }}
            </Button>
          </div>
        </article>
      </div>
    </section>

    <Overlay
      :open="dialogs.issueOpen"
      overlayClass="bg-black/40 p-4"
      closeOnBackdrop
      trapFocus
      initialFocusSelector="[data-issue-device-input]"
      @close="closeIssueDialog"
    >
      <div
        data-permit-issue-dialog
        class="flex max-h-[85vh] w-full max-w-xl flex-col overflow-hidden rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl"
      >
        <CardHeader
          :title="t('Issue Permit')"
          :description="t('Bind the token to a specific device and role. Leave expiry empty to let authority apply its default TTL.')"
          title-tag="h3"
          title-class="text-lg"
        />

        <div class="mt-5 min-h-0 flex-1 overflow-y-auto pr-1">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Device ID") }}
            </label>
            <input
              v-model="issueForm.deviceId"
              data-issue-device-input
              :class="inputClass"
              :placeholder="t('device-001')"
            />
          </div>

          <div class="mt-4">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Role") }}
            </label>
            <input
              v-model="issueForm.role"
              data-issue-role-input
              :class="inputClass"
              :placeholder="t('admin')"
            />
          </div>

          <div class="mt-4">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Expires At (optional)") }}
            </label>
            <input
              v-model="issueForm.expiresAt"
              data-issue-expires-input
              type="datetime-local"
              :class="inputClass"
            />
          </div>
        </div>

        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <Button variant="outline" @click="closeIssueDialog">{{ t("Cancel") }}</Button>
          <Button
            data-issue-submit
            :disabled="permitStore.state.issuing"
            @click="issuePermit"
          >
            {{ t("Issue Now") }}
          </Button>
        </div>
      </div>
    </Overlay>
  </section>
</template>
