<script setup lang="ts">
// 本文件实现 Win 前端的 `RegistrationApprovals` 页面。
import { computed, onMounted, reactive, watch } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
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
const reviewDialog = reactive({
  open: false,
  requestId: "",
  role: "",
  reason: ""
})

const inputClass =
  "mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
const textAreaClass =
  "mt-2 min-h-[120px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

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

const activeReviewItem = computed(() => {
  const requestId = reviewDialog.requestId.trim()
  if (!requestId) {
    return null
  }
  return approvalsStore.state.items.find((item) => item.requestId === requestId) || null
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

const requestMetaLine = (item: PendingRegister) => {
  return [
    item.requestId,
    `${t("Created At")}: ${formatTimestamp(item.createdAt)}`,
    `${t("Expires At")}: ${formatTimestamp(item.expiresAt)}`
  ].join(" · ")
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

const persistReviewDraft = () => {
  const requestId = reviewDialog.requestId.trim()
  if (!requestId) {
    return
  }
  const draft = ensureDraft(requestId)
  draft.role = reviewDialog.role
  draft.reason = reviewDialog.reason
}

const resetReviewDialog = () => {
  reviewDialog.open = false
  reviewDialog.requestId = ""
  reviewDialog.role = ""
  reviewDialog.reason = ""
}

const closeReviewDialog = () => {
  persistReviewDraft()
  resetReviewDialog()
}

const openReviewDialog = (item: PendingRegister) => {
  const draft = ensureDraft(item.requestId)
  reviewDialog.open = true
  reviewDialog.requestId = item.requestId
  reviewDialog.role = draft.role
  reviewDialog.reason = draft.reason
}

const clearRequestDraft = (requestId: string) => {
  const draft = ensureDraft(requestId)
  draft.role = ""
  draft.reason = ""
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
    const role = item.requestId === reviewDialog.requestId ? reviewDialog.role.trim() : ensureDraft(item.requestId).role.trim()
    const resp = await approvalsStore.approveRegister(item.requestId, role)
    clearRequestDraft(item.requestId)
    if (item.requestId === reviewDialog.requestId) {
      resetReviewDialog()
    }
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
    const reason =
      item.requestId === reviewDialog.requestId ? reviewDialog.reason.trim() : ensureDraft(item.requestId).reason.trim()
    await approvalsStore.rejectRegister(item.requestId, reason)
    clearRequestDraft(item.requestId)
    if (item.requestId === reviewDialog.requestId) {
      resetReviewDialog()
    }
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
      resetReviewDialog()
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
      resetReviewDialog()
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

watch(activeReviewItem, (item) => {
  if (reviewDialog.open && !item) {
    resetReviewDialog()
  }
})

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
        :description="t('Use the queue as a compact inbox, then open only the request you are currently deciding.')"
        title-tag="h3"
        title-class="text-base"
      >
        <template #actions>
          <Button
            data-approval-refresh
            variant="outline"
            size="sm"
            :disabled="approvalsStore.state.loading || !!approvalsStore.state.busyRequestId"
            @click="loadPending(false)"
          >
            {{ t("Refresh") }}
          </Button>
        </template>
      </CardHeader>

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
            data-approval-filter-apply
            class="h-10"
            variant="outline"
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

      <div
        v-if="approvalsStore.state.items.length"
        class="mt-5 overflow-hidden rounded-2xl border border-border/60 bg-background/70"
      >
        <article
          v-for="item in approvalsStore.state.items"
          :key="item.requestId"
          data-approval-row
          class="border-b border-border/60 px-4 py-3 last:border-b-0"
        >
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1">
              <p class="truncate font-semibold text-foreground">{{ item.deviceId || "-" }}</p>
              <Badge variant="secondary">{{ item.requestedRole || t("No requested role") }}</Badge>
              <Badge variant="secondary">{{ item.displayName || t("No display name") }}</Badge>
              <p class="w-full text-xs text-muted-foreground">{{ requestMetaLine(item) }}</p>
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <Button
                data-approval-review-open
                size="sm"
                variant="outline"
                :disabled="approvalsStore.state.busyRequestId === item.requestId"
                @click="openReviewDialog(item)"
              >
                {{ t("Review") }}
              </Button>
            </div>
          </div>
        </article>
      </div>

      <div
        v-else-if="!approvalsStore.state.loading"
        class="mt-5 rounded-2xl border border-dashed border-border/60 bg-background/50 px-5 py-8 text-center text-sm text-muted-foreground"
      >
        {{ t("No pending registrations.") }}
      </div>
    </section>

    <Overlay
      :open="reviewDialog.open"
      overlayClass="bg-black/40 p-4"
      closeOnBackdrop
      trapFocus
      initialFocusSelector="[data-review-role-input]"
      @close="closeReviewDialog"
    >
      <div
        data-approval-review-dialog
        class="flex max-h-[85vh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl"
      >
        <CardHeader
          :title="t('Review Request')"
          :description="t('Approve or reject from one focused panel after checking the request summary.')"
          title-tag="h3"
          title-class="text-lg"
        />

        <div v-if="activeReviewItem" class="mt-5 min-h-0 flex-1 overflow-y-auto">
          <section class="rounded-2xl border border-border/60 bg-background/70 px-4 py-4 text-sm">
            <div class="flex flex-wrap items-center gap-2">
              <p class="font-semibold text-foreground">{{ activeReviewItem.deviceId || "-" }}</p>
              <Badge variant="secondary">{{ activeReviewItem.requestedRole || t("No requested role") }}</Badge>
              <Badge variant="secondary">{{ activeReviewItem.displayName || t("No display name") }}</Badge>
            </div>
            <p class="mt-2 break-all text-xs text-muted-foreground">{{ activeReviewItem.requestId }}</p>

            <div class="mt-4 grid gap-3 sm:grid-cols-2">
              <div>
                <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Created At") }}</p>
                <p class="mt-1">{{ formatTimestamp(activeReviewItem.createdAt) }}</p>
              </div>
              <div>
                <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Expires At") }}</p>
                <p class="mt-1">{{ formatTimestamp(activeReviewItem.expiresAt) }}</p>
              </div>
            </div>
          </section>

          <div class="mt-5 space-y-4">
            <section class="rounded-2xl border border-emerald-200 bg-emerald-50/70 p-4">
              <p class="text-sm font-semibold text-emerald-800">{{ t("Approve Request") }}</p>
              <p class="mt-1 text-xs text-emerald-700">
                {{ t("Leave role blank to keep authority-side default approval behavior.") }}
              </p>
              <input
                v-model="reviewDialog.role"
                data-review-role-input
                :class="inputClass"
                :placeholder="t('Optional role override')"
              />
            </section>

            <section class="rounded-2xl border border-rose-200 bg-rose-50/70 p-4">
              <p class="text-sm font-semibold text-rose-800">{{ t("Reject Request") }}</p>
              <p class="mt-1 text-xs text-rose-700">
                {{ t("Reason is optional but recommended for audit clarity.") }}
              </p>
              <textarea
                v-model="reviewDialog.reason"
                data-review-reason-input
                :class="textAreaClass"
                :placeholder="t('Optional rejection reason')"
              />
            </section>
          </div>
        </div>

        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <Button variant="outline" @click="closeReviewDialog">{{ t("Cancel") }}</Button>
          <Button
            v-if="activeReviewItem"
            data-review-reject
            variant="outline"
            :disabled="approvalsStore.state.busyRequestId === activeReviewItem.requestId"
            @click="rejectRegister(activeReviewItem)"
          >
            {{ t("Reject") }}
          </Button>
          <Button
            v-if="activeReviewItem"
            data-review-approve
            :disabled="approvalsStore.state.busyRequestId === activeReviewItem.requestId"
            @click="approveRegister(activeReviewItem)"
          >
            {{ t("Approve") }}
          </Button>
        </div>
      </div>
    </Overlay>
  </section>
</template>
