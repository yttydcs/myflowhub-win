<script setup lang="ts">
import { computed, onMounted, reactive, watch } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import CardHeader from "@/components/CardHeader.vue"
import { useI18n } from "@/i18n"
import { useAuthorityStore } from "@/stores/authority"
import { useRegistrationApprovalsStore, type PendingRegister } from "@/stores/registrationApprovals"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"

const sessionStore = useSessionStore()
const authorityStore = useAuthorityStore()
const approvalsStore = useRegistrationApprovalsStore()
const toast = useToastStore()
const { t } = useI18n()

const drafts = reactive<Record<string, { role: string; reason: string }>>({})
const autoLoaded = reactive({ value: false })

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

const authorityLabel = computed(() => {
  return authorityStore.state.authorityId ? String(authorityStore.state.authorityId) : "-"
})

const summaryCards = computed(() => {
  const pendingWithRole = approvalsStore.state.items.filter((item) => item.requestedRole).length
  const pendingWithDisplayName = approvalsStore.state.items.filter((item) => item.displayName).length
  return [
    {
      label: t("Pending Requests"),
      value: String(approvalsStore.state.total)
    },
    {
      label: t("Requested Roles"),
      value: String(pendingWithRole)
    },
    {
      label: t("Named Devices"),
      value: String(pendingWithDisplayName)
    }
  ]
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

const ensureDraft = (requestId: string) => {
  if (!drafts[requestId]) {
    drafts[requestId] = {
      role: "",
      reason: ""
    }
  }
  return drafts[requestId]
}

const formatTimestamp = (value: number) => {
  const numeric = Number(value || 0)
  if (!numeric) return "-"
  const millis = numeric > 100000000000 ? numeric : numeric * 1000
  const date = new Date(millis)
  if (Number.isNaN(date.getTime())) return "-"
  return date.toLocaleString()
}

const syncDrafts = () => {
  const active = new Set(approvalsStore.state.items.map((item) => item.requestId))
  for (const requestId of Object.keys(drafts)) {
    if (!active.has(requestId)) {
      delete drafts[requestId]
    }
  }
  for (const item of approvalsStore.state.items) {
    ensureDraft(item.requestId)
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

const loadPending = async (silent = false) => {
  try {
    ensureReady()
    await approvalsStore.loadPending()
    syncDrafts()
    if (!silent) {
      toast.success(
        t("Pending registrations loaded."),
        t("{count} entries", { count: approvalsStore.state.total })
      )
    }
  } catch (err) {
    if (!silent) {
      console.warn(err)
      toast.errorOf(err, t("Failed to load pending registrations."))
    }
  }
}

const approveRegister = async (item: PendingRegister) => {
  try {
    ensureReady()
    const draft = ensureDraft(item.requestId)
    const resp = await approvalsStore.approveRegister(item.requestId, draft.role.trim())
    draft.role = ""
    draft.reason = ""
    toast.success(
      t("Registration approved."),
      t("node={nodeId}", { nodeId: Number(resp?.nodeId || 0) || "-" })
    )
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to approve registration."))
  }
}

const rejectRegister = async (item: PendingRegister) => {
  try {
    ensureReady()
    const draft = ensureDraft(item.requestId)
    await approvalsStore.rejectRegister(item.requestId, draft.reason.trim())
    draft.role = ""
    draft.reason = ""
    toast.success(t("Registration rejected."), item.deviceId || item.requestId)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to reject registration."))
  }
}

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  ([nodeId, hubId], oldValue) => {
    const [prevNodeId, prevHubId] = Array.isArray(oldValue) ? oldValue : []
    authorityStore.setIdentity(Number(nodeId || 0), Number(hubId || 0))
    if (Number(nodeId || 0) !== Number(prevNodeId || 0) || Number(hubId || 0) !== Number(prevHubId || 0)) {
      approvalsStore.reset()
      autoLoaded.value = false
      for (const key of Object.keys(drafts)) {
        delete drafts[key]
      }
    }
  },
  { immediate: true }
)

watch(
  () => ready.value,
  (isReady) => {
    if (!isReady) {
      autoLoaded.value = false
      approvalsStore.reset()
      return
    }
    if (autoLoaded.value) {
      return
    }
    autoLoaded.value = true
    void loadPending(true)
  },
  { immediate: true }
)

onMounted(() => {
  if (ready.value && !autoLoaded.value) {
    autoLoaded.value = true
    void loadPending(true)
  }
})
</script>

<template>
  <section class="space-y-6">
    <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
      <CardHeader
        :title="t('Registration Approvals')"
        :description="t('Review first-register requests, assign an optional role, and decide which devices may enter the network.')"
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
            :disabled="authorityStore.state.resolving || approvalsStore.state.loading || !!approvalsStore.state.busyRequestId"
            @click="resolveAuthorityAction"
          >
            {{ t("Resolve") }}
          </Button>
        </div>
        <div class="self-end">
          <Button
            size="sm"
            :disabled="approvalsStore.state.loading || !!approvalsStore.state.busyRequestId"
            @click="loadPending(false)"
          >
            {{ t("Refresh") }}
          </Button>
        </div>
      </div>

      <div class="mt-5 grid gap-3 md:grid-cols-3">
        <div
          v-for="card in summaryCards"
          :key="card.label"
          class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3"
        >
          <p class="text-[11px] font-semibold uppercase tracking-[0.25em] text-muted-foreground">{{ card.label }}</p>
          <p class="mt-2 text-lg font-semibold text-foreground">{{ card.value }}</p>
        </div>
      </div>
    </section>

    <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
      <CardHeader
        :title="t('Pending Queue')"
        :description="t('Approve only when the device identity is expected. Leave role blank to let authority apply its configured default path.')"
        title-tag="h3"
        title-class="text-base"
      />

      <div class="mt-4 grid gap-4 md:grid-cols-[minmax(0,1fr)_auto]">
        <div>
          <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            {{ t("Device Filter") }}
          </label>
          <input
            v-model="approvalsStore.state.filterDeviceId"
            :class="inputClass"
            :placeholder="t('Filter by device ID (optional)')"
          />
        </div>
        <div class="self-end">
          <Button
            variant="outline"
            size="sm"
            :disabled="approvalsStore.state.loading || !!approvalsStore.state.busyRequestId"
            @click="loadPending(false)"
          >
            {{ t("Apply Filter") }}
          </Button>
        </div>
      </div>

      <div
        v-if="approvalsStore.state.lastDecision"
        class="mt-4 rounded-2xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700"
      >
        <p class="font-semibold">
          {{
            approvalsStore.state.lastDecision.action === "approve"
              ? t("Last action: approved")
              : t("Last action: rejected")
          }}
        </p>
        <p class="mt-1">
          {{
            t("request={requestId} device={deviceId}", {
              requestId: approvalsStore.state.lastDecision.requestId,
              deviceId: approvalsStore.state.lastDecision.deviceId || "-"
            })
          }}
        </p>
      </div>

      <div class="mt-5 space-y-4">
        <div
          v-for="item in approvalsStore.state.items"
          :key="item.requestId"
          class="rounded-2xl border border-border/60 bg-background/70 p-4"
        >
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Request") }}
              </p>
              <h4 class="mt-1 text-base font-semibold text-foreground">{{ item.deviceId || "-" }}</h4>
              <p class="mt-1 text-sm text-muted-foreground">{{ item.requestId }}</p>
            </div>
            <div class="flex flex-wrap gap-2">
              <Badge variant="secondary">{{ item.requestedRole || t("No requested role") }}</Badge>
              <Badge variant="secondary">{{ item.displayName || t("No display name") }}</Badge>
            </div>
          </div>

          <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <div class="rounded-xl border border-border/60 bg-card/80 px-3 py-2 text-sm">
              <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Requested Role") }}</p>
              <p class="mt-1">{{ item.requestedRole || "-" }}</p>
            </div>
            <div class="rounded-xl border border-border/60 bg-card/80 px-3 py-2 text-sm">
              <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Display Name") }}</p>
              <p class="mt-1">{{ item.displayName || "-" }}</p>
            </div>
            <div class="rounded-xl border border-border/60 bg-card/80 px-3 py-2 text-sm">
              <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Created At") }}</p>
              <p class="mt-1">{{ formatTimestamp(item.createdAt) }}</p>
            </div>
            <div class="rounded-xl border border-border/60 bg-card/80 px-3 py-2 text-sm">
              <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Expires At") }}</p>
              <p class="mt-1">{{ formatTimestamp(item.expiresAt) }}</p>
            </div>
          </div>

          <div class="mt-4 grid gap-4 lg:grid-cols-2">
            <div class="rounded-2xl border border-emerald-200 bg-emerald-50/70 p-4">
              <p class="text-sm font-semibold text-emerald-800">{{ t("Approve Request") }}</p>
              <p class="mt-1 text-xs text-emerald-700">
                {{ t("Leave role blank to keep authority-side default approval behavior.") }}
              </p>
              <input
                v-model="ensureDraft(item.requestId).role"
                class="mt-3 h-10 w-full rounded-md border border-emerald-300 bg-white px-3 text-sm"
                :placeholder="t('Optional role override')"
              />
              <div class="mt-3 flex justify-end">
                <Button
                  size="sm"
                  :disabled="approvalsStore.state.busyRequestId === item.requestId"
                  @click="approveRegister(item)"
                >
                  {{ t("Approve") }}
                </Button>
              </div>
            </div>

            <div class="rounded-2xl border border-rose-200 bg-rose-50/70 p-4">
              <p class="text-sm font-semibold text-rose-800">{{ t("Reject Request") }}</p>
              <p class="mt-1 text-xs text-rose-700">
                {{ t("Reason is optional but recommended for audit clarity.") }}
              </p>
              <input
                v-model="ensureDraft(item.requestId).reason"
                class="mt-3 h-10 w-full rounded-md border border-rose-300 bg-white px-3 text-sm"
                :placeholder="t('Optional rejection reason')"
              />
              <div class="mt-3 flex justify-end">
                <Button
                  size="sm"
                  variant="outline"
                  :disabled="approvalsStore.state.busyRequestId === item.requestId"
                  @click="rejectRegister(item)"
                >
                  {{ t("Reject") }}
                </Button>
              </div>
            </div>
          </div>
        </div>

        <div
          v-if="!approvalsStore.state.loading && !approvalsStore.state.items.length"
          class="rounded-2xl border border-dashed border-border/60 bg-background/50 px-5 py-8 text-center text-sm text-muted-foreground"
        >
          {{ t("No pending registrations.") }}
        </div>
      </div>
    </section>
  </section>
</template>
