<script setup lang="ts">
import { computed, onMounted, reactive, watch } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
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
  if (!authorityStore.state.authorityId) return "-"
  if (!authorityStore.state.authorityReason) return String(authorityStore.state.authorityId)
  return `${authorityStore.state.authorityId} (${authorityStore.state.authorityReason})`
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
    toast.success(
      t("Permit issued."),
      t("device={deviceId} role={role}", { deviceId: lastIssued.deviceId, role: lastIssued.role })
    )
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to issue permit."))
  }
}

const revokePermit = async (permitOverride?: string) => {
  try {
    ensureReady()
    const permit = String(permitOverride ?? revokeForm.permit).trim()
    if (!permit) {
      throw new Error(t("Permit token is required."))
    }
    const result = await permitStore.revokePermit(permit)
    if (result.permit === revokeForm.permit.trim()) {
      revokeForm.permit = ""
    }
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
      revokeForm.permit = ""
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

      <div class="mt-5 grid gap-4 xl:grid-cols-[240px_minmax(0,1fr)_auto]">
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
          <p class="font-semibold text-foreground">{{ t("Protocol Boundary") }}</p>
          <p class="mt-1">{{ t("Current auth protocol supports issue + revoke, but not permit history listing.") }}</p>
          <p class="mt-1">{{ t("The console only keeps the latest successful issuance in memory for quick copy or revoke.") }}</p>
        </div>
        <div class="self-end">
          <Button
            variant="outline"
            size="sm"
            :disabled="authorityStore.state.resolving || permitStore.state.issuing || permitStore.state.revoking"
            @click="resolveAuthorityAction"
          >
            {{ t("Resolve") }}
          </Button>
        </div>
      </div>
    </section>

    <section class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div class="space-y-6">
        <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
          <CardHeader
            :title="t('Issue Permit')"
            :description="t('Bind the token to a specific device and role. Leave expiry empty to let authority apply its default TTL.')"
            title-tag="h3"
            title-class="text-base"
          />

          <div class="mt-4 grid gap-4 md:grid-cols-2">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Device ID") }}
              </label>
              <input v-model="issueForm.deviceId" :class="inputClass" :placeholder="t('device-001')" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Role") }}
              </label>
              <input v-model="issueForm.role" :class="inputClass" :placeholder="t('admin')" />
            </div>
          </div>

          <div class="mt-4">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Expires At (optional)") }}
            </label>
            <input v-model="issueForm.expiresAt" type="datetime-local" :class="inputClass" />
          </div>

          <div class="mt-5 flex flex-wrap gap-2">
            <Button :disabled="permitStore.state.issuing" @click="issuePermit">{{ t("Issue Permit") }}</Button>
          </div>
        </section>

        <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
          <CardHeader
            :title="t('Revoke Permit')"
            :description="t('Use revoke when the token should be burned immediately, even if the device has not consumed it yet.')"
            title-tag="h3"
            title-class="text-base"
          />

          <div class="mt-4">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Permit Token") }}
            </label>
            <textarea
              v-model="revokeForm.permit"
              :class="textAreaClass"
              :placeholder="t('permit_xxx')"
            />
          </div>

          <div class="mt-5 flex flex-wrap gap-2">
            <Button variant="outline" :disabled="permitStore.state.revoking" @click="revokePermit()">
              {{ t("Revoke Permit") }}
            </Button>
          </div>
        </section>
      </div>

      <div class="space-y-6">
        <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
          <CardHeader
            :title="t('Latest Permit')"
            :description="t('This panel reflects only the latest successful issue action from the current session.')"
            title-tag="h3"
            title-class="text-base"
          >
            <template #actions>
              <Badge variant="secondary">
                {{
                  permitStore.state.lastIssued?.revoked
                    ? t("Revoked")
                    : permitStore.state.lastIssued?.permit
                      ? t("Active")
                      : t("Idle")
                }}
              </Badge>
            </template>
          </CardHeader>

          <div
            v-if="permitStore.state.lastIssued"
            class="mt-4 space-y-3 rounded-2xl border border-border/60 bg-background/70 p-4 text-sm"
          >
            <div>
              <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Permit Token") }}</p>
              <p class="mt-2 break-all font-mono text-xs text-foreground">{{ permitStore.state.lastIssued.permit }}</p>
            </div>
            <div class="grid gap-3 sm:grid-cols-2">
              <div>
                <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Device ID") }}</p>
                <p class="mt-1">{{ permitStore.state.lastIssued.deviceId }}</p>
              </div>
              <div>
                <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Role") }}</p>
                <p class="mt-1">{{ permitStore.state.lastIssued.role }}</p>
              </div>
              <div>
                <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Expires At") }}</p>
                <p class="mt-1">{{ formatTimestamp(permitStore.state.lastIssued.expiresAt) }}</p>
              </div>
              <div>
                <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Issued At") }}</p>
                <p class="mt-1">{{ formatTimestamp(Date.parse(permitStore.state.lastIssued.issuedAt)) }}</p>
              </div>
            </div>

            <div class="flex flex-wrap gap-2">
              <Button size="sm" variant="outline" @click="copyPermit">{{ t("Copy Permit") }}</Button>
              <Button
                size="sm"
                variant="outline"
                :disabled="permitStore.state.lastIssued.revoked || permitStore.state.revoking"
                @click="revokePermit(permitStore.state.lastIssued.permit)"
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
  </section>
</template>
