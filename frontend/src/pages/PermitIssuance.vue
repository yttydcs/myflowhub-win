<script setup lang="ts">
import { computed, onMounted, reactive, watch } from "vue"
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
const dialogs = reactive({
  issueOpen: false,
  revokeOpen: false
})

const issueForm = reactive({
  deviceId: "",
  role: "",
  expiresAt: ""
})

const revokeForm = reactive({
  permit: ""
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

const latestPermitStateLabel = computed(() => {
  if (permitStore.state.lastIssued?.revoked) {
    return t("Revoked")
  }
  if (permitStore.state.lastIssued?.permit) {
    return t("Active")
  }
  return t("Idle")
})

const latestPermitDetails = computed(() => {
  const latest = permitStore.state.lastIssued
  if (!latest) {
    return []
  }
  return [
    { label: t("Device ID"), value: latest.deviceId || "-" },
    { label: t("Role"), value: latest.role || "-" },
    { label: t("Expires At"), value: formatTimestamp(latest.expiresAt) },
    { label: t("Issued At"), value: formatTimestamp(Date.parse(latest.issuedAt)) }
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

const resetDialogs = () => {
  dialogs.issueOpen = false
  dialogs.revokeOpen = false
}

const resetForms = () => {
  issueForm.deviceId = ""
  issueForm.role = ""
  issueForm.expiresAt = ""
  revokeForm.permit = ""
}

const formatTimestamp = (value: number) => {
  const numeric = Number(value || 0)
  if (!numeric) return t("Authority default TTL")
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

const closeIssueDialog = () => {
  dialogs.issueOpen = false
}

const openIssueDialog = () => {
  dialogs.issueOpen = true
}

const closeRevokeDialog = () => {
  dialogs.revokeOpen = false
}

const openRevokeDialog = (permitOverride?: string) => {
  const nextPermit = String(
    permitOverride ?? revokeForm.permit ?? permitStore.state.lastIssued?.permit ?? ""
  ).trim()
  revokeForm.permit = nextPermit
  dialogs.revokeOpen = true
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
    const lastIssued = await permitStore.issuePermit({
      deviceId,
      role,
      expiresAt: parseExpiresAt()
    })
    revokeForm.permit = lastIssued.permit
    closeIssueDialog()
    toast.success(
      t("Permit issued."),
      t("device={deviceId} role={role}", { deviceId: lastIssued.deviceId, role: lastIssued.role })
    )
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to issue permit."))
  }
}

const revokePermit = async () => {
  try {
    ensureReady()
    const permit = String(revokeForm.permit).trim()
    if (!permit) {
      throw new Error(t("Permit token is required."))
    }
    const result = await permitStore.revokePermit(permit)
    if (result.permit === revokeForm.permit.trim()) {
      revokeForm.permit = ""
    }
    closeRevokeDialog()
    toast.success(t("Permit revoked."), result.permit)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to revoke permit."))
  }
}

const copyPermit = async () => {
  try {
    const permit = permitStore.state.lastIssued?.permit || ""
    if (!permit) {
      throw new Error(t("No permit available to copy."))
    }
    if (!navigator?.clipboard?.writeText) {
      throw new Error(t("Clipboard unavailable."))
    }
    await navigator.clipboard.writeText(permit)
    toast.success(t("Permit copied."), permit)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to copy permit."))
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
      resetDialogs()
      resetForms()
    }
  },
  { immediate: true }
)

watch(
  () => ready.value,
  (isReady) => {
    if (!isReady) {
      autoLoaded.value = false
      permitStore.reset()
      resetDialogs()
      resetForms()
      return
    }
    if (autoLoaded.value) {
      return
    }
    autoLoaded.value = true
  },
  { immediate: true }
)

onMounted(() => {
  if (ready.value && !autoLoaded.value) {
    autoLoaded.value = true
  }
})
</script>

<template>
  <section class="space-y-6">
    <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
      <CardHeader
        :title="t('Permit Issuance')"
        :description="t('Create one-time join permits for expected devices, then revoke them explicitly when the window closes.')"
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

      <div class="mt-5 rounded-2xl border border-border/60 bg-background/70 px-4 py-3 text-xs text-muted-foreground">
          <p class="font-semibold text-foreground">{{ t("Protocol Boundary") }}</p>
          <p class="mt-1">{{ t("Current auth protocol supports issue + revoke, but not permit history listing.") }}</p>
          <p class="mt-1">{{ t("The console only keeps the latest successful issuance in memory for quick copy or revoke.") }}</p>
      </div>
    </section>

    <section class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div class="space-y-6">
        <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
          <CardHeader
            :title="t('Permit Actions')"
            :description="t('Open focused dialogs for issue and revoke so the page stays readable between operations.')"
            title-tag="h3"
            title-class="text-base"
          />

          <div class="mt-4 overflow-hidden rounded-2xl border border-border/60 bg-background/70">
            <article
              data-permit-issue-row
              class="flex flex-wrap items-start justify-between gap-4 border-b border-border/60 px-4 py-4"
            >
              <div class="min-w-0 flex-1">
                <p class="text-base font-semibold text-foreground">{{ t("Issue Permit") }}</p>
                <p class="mt-1 text-sm text-muted-foreground">
                  {{ t("Open the issue dialog only when you need to mint a token for one device.") }}
                </p>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <Button
                  data-open-issue-dialog
                  size="sm"
                  @click="openIssueDialog"
                >
                  {{ t("Issue Permit") }}
                </Button>
              </div>
            </article>

            <article
              data-permit-revoke-row
              class="flex flex-wrap items-start justify-between gap-4 px-4 py-4"
            >
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-2">
                  <p class="text-base font-semibold text-foreground">{{ t("Revoke Permit") }}</p>
                  <Badge v-if="permitStore.state.lastIssued?.permit" variant="secondary">
                    {{ latestPermitStateLabel }}
                  </Badge>
                </div>
                <p class="mt-1 text-sm text-muted-foreground">
                  {{ t("Open the revoke dialog only when you need to burn a token from the latest result or a pasted value.") }}
                </p>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <Button
                  v-if="permitStore.state.lastIssued?.permit"
                  data-open-revoke-latest
                  size="sm"
                  variant="outline"
                  :disabled="permitStore.state.lastIssued.revoked"
                  @click="openRevokeDialog(permitStore.state.lastIssued.permit)"
                >
                  {{ t("Use Latest Permit") }}
                </Button>
                <Button
                  data-open-revoke-dialog
                  size="sm"
                  variant="outline"
                  @click="openRevokeDialog()"
                >
                  {{ t("Revoke Permit") }}
                </Button>
              </div>
            </article>
          </div>
        </section>
      </div>

      <div class="space-y-6">
        <section
          data-latest-permit-card
          class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm"
        >
          <CardHeader
            :title="t('Latest Permit')"
            :description="t('Review the latest issued token, then copy it or send it into the revoke flow.')"
            title-tag="h3"
            title-class="text-base"
          >
            <template #actions>
              <Badge variant="secondary">{{ latestPermitStateLabel }}</Badge>
            </template>
          </CardHeader>

          <div
            v-if="permitStore.state.lastIssued"
            class="mt-4 rounded-2xl border border-border/60 bg-background/70 p-4 text-sm"
          >
            <div>
              <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Permit Token") }}</p>
              <p class="mt-2 break-all rounded-xl border border-border/60 bg-card/80 px-3 py-2 font-mono text-xs text-foreground">
                {{ permitStore.state.lastIssued.permit }}
              </p>
            </div>
            <div
              data-latest-permit-details
              class="mt-3 overflow-hidden rounded-xl border border-border/60 bg-card/80"
            >
              <div
                v-for="detail in latestPermitDetails"
                :key="detail.label"
                class="flex flex-wrap items-center justify-between gap-2 border-b border-border/60 px-3 py-2 last:border-b-0"
              >
                <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ detail.label }}
                </p>
                <p class="text-right text-foreground">{{ detail.value }}</p>
              </div>
            </div>

            <div class="mt-3 flex flex-wrap gap-2">
              <Button data-copy-permit size="sm" variant="outline" @click="copyPermit">{{ t("Copy Permit") }}</Button>
              <Button
                data-open-latest-revoke-dialog
                size="sm"
                variant="outline"
                :disabled="permitStore.state.lastIssued.revoked"
                @click="openRevokeDialog(permitStore.state.lastIssued.permit)"
              >
                {{ t("Revoke Latest Permit") }}
              </Button>
            </div>
          </div>

          <div
            v-else
            class="mt-4 rounded-2xl border border-dashed border-border/60 bg-background/50 px-5 py-8 text-center text-sm text-muted-foreground"
          >
            {{ t("No permit has been issued in this session yet.") }}
          </div>
        </section>

        <section
          v-if="permitStore.state.lastRevoke"
          class="rounded-2xl border border-amber-200 bg-amber-50 px-5 py-4 text-sm text-amber-800 shadow-sm"
        >
          <p class="font-semibold">{{ t("Last revoke") }}</p>
          <p class="mt-1 break-all">{{ permitStore.state.lastRevoke.permit }}</p>
        </section>
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

    <Overlay
      :open="dialogs.revokeOpen"
      overlayClass="bg-black/40 p-4"
      closeOnBackdrop
      trapFocus
      initialFocusSelector="[data-revoke-permit-input]"
      @close="closeRevokeDialog"
    >
      <div
        data-permit-revoke-dialog
        class="flex max-h-[85vh] w-full max-w-xl flex-col overflow-hidden rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl"
      >
        <CardHeader
          :title="t('Revoke Permit')"
          :description="t('Use revoke when the token should be burned immediately, even if the device has not consumed it yet.')"
          title-tag="h3"
          title-class="text-lg"
        />

        <div class="mt-5 min-h-0 flex-1 overflow-y-auto pr-1">
          <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            {{ t("Permit Token") }}
          </label>
          <input
            v-model="revokeForm.permit"
            data-revoke-permit-input
            :class="inputClass"
            :placeholder="t('permit_xxx')"
          />
        </div>

        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <Button variant="outline" @click="closeRevokeDialog">{{ t("Cancel") }}</Button>
          <Button
            data-revoke-submit
            variant="outline"
            :disabled="permitStore.state.revoking"
            @click="revokePermit"
          >
            {{ t("Revoke Now") }}
          </Button>
        </div>
      </div>
    </Overlay>
  </section>
</template>
