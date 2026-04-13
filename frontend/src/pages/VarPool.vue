<script setup lang="ts">
// Context: implements the VarPool page in the Win frontend.
import { computed, onMounted, reactive, ref, watch } from "vue"
import CardHeader from "@/components/CardHeader.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import PageHero from "@/components/PageHero.vue"
import { Overlay } from "@/components/ui/overlay"
import NodeVarsDialog from "@/components/varpool/NodeVarsDialog.vue"
import { useI18n } from "@/i18n"
import { useProfileStore } from "@/stores/profile"
import { useSessionStore } from "@/stores/session"
import { useVarPoolStore, type VarPoolKey, type VarPoolValue } from "@/stores/varpool"
import { useToastStore } from "@/stores/toast"
import { HomeState as LoadHomeState } from "../../wailsjs/go/main/App"

type VarPoolTab = "control" | "mine" | "watch"

type VarPoolEntry = {
  key: VarPoolKey
  snapshot: VarPoolValue
}

const profileStore = useProfileStore()
const sessionStore = useSessionStore()
const varpool = useVarPoolStore()
const toast = useToastStore()
const { t } = useI18n()

const tabs: { id: VarPoolTab; label: string }[] = [
  { id: "control", label: "Control" },
  { id: "mine", label: "Mine" },
  { id: "watch", label: "Watch" }
]

const busy = ref(false)
const activeTab = ref<VarPoolTab>("control")

const editDialog = reactive({
  open: false,
  name: "",
  owner: 0,
  value: "",
  visibility: "public",
  kind: "string"
})

const addMineDialog = reactive({
  open: false,
  name: "",
  value: "",
  visibility: "public",
  kind: "string"
})

const addWatchDialog = reactive({
  open: false,
  name: "",
  owner: ""
})

const nodeVarsDialogOpen = ref(false)
const nodeVarsDialogOwnerId = ref(0)

const openNodeVarsDialog = () => {
  nodeVarsDialogOwnerId.value = 0
  nodeVarsDialogOpen.value = true
}

const closeNodeVarsDialog = () => {
  nodeVarsDialogOpen.value = false
}

const fallbackIdentity = reactive({
  nodeId: 0,
  hubId: 0
})

const inputClass =
  "mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

const connectedLabel = computed(() => (sessionStore.connected ? t("Connected") : t("Disconnected")))
const connectedTone = computed(() =>
  sessionStore.connected ? "bg-emerald-500/15 text-emerald-700" : "bg-rose-500/15 text-rose-700"
)

const selfNodeId = computed(() => sessionStore.auth.nodeId || fallbackIdentity.nodeId || 0)
const hubId = computed(() => sessionStore.auth.hubId || fallbackIdentity.hubId || 0)

const groupedKeys = computed(() => {
  const mine: VarPoolKey[] = []
  const others: VarPoolKey[] = []
  for (const key of varpool.state.keys) {
    if (selfNodeId.value && Number(key.owner ?? 0) === selfNodeId.value) {
      mine.push(key)
    } else {
      others.push(key)
    }
  }
  return { mine, others }
})

const buildEntry = (key: VarPoolKey): VarPoolEntry => ({
  key,
  snapshot: varpool.valueForKey(key)
})

const mineEntries = computed(() => groupedKeys.value.mine.map(buildEntry))
const watchEntries = computed(() => groupedKeys.value.others.map(buildEntry))
const subscribedEntries = computed(() =>
  watchEntries.value.filter(({ snapshot }) => snapshot.subKnown && snapshot.subscribed)
)

const summaryItems = computed(() => [
  { label: t("Connected"), value: connectedLabel.value },
  { label: t("NodeID"), value: selfNodeId.value ? String(selfNodeId.value) : "-" },
  { label: t("HubID"), value: hubId.value ? String(hubId.value) : "-" },
  { label: t("Cached Keys"), value: String(varpool.state.keys.length) },
  { label: t("Mine Count"), value: String(mineEntries.value.length) },
  { label: t("Watch Count"), value: String(watchEntries.value.length) },
  { label: t("Subscribed Count"), value: String(subscribedEntries.value.length) },
  { label: t("Last Frame"), value: varpool.state.lastFrameAt || "-" }
])

const displayMeta = (value: unknown, fallback = "unknown") => {
  const normalized = String(value ?? "").trim()
  return t(normalized || fallback)
}

const setActiveTab = (tab: VarPoolTab) => {
  activeTab.value = tab
}

const tabButtonClass = (tab: VarPoolTab) => [
  "rounded-full px-4 py-2 text-sm font-semibold transition",
  activeTab.value === tab
    ? "bg-primary text-primary-foreground shadow-sm"
    : "text-muted-foreground hover:bg-muted/70"
]

const parseOwner = (value: string, required: boolean) => {
  const trimmed = value.trim()
  if (!trimmed) {
    if (required) {
      throw new Error(t("Owner NodeID is required."))
    }
    return 0
  }
  const parsed = Number.parseInt(trimmed, 10)
  if (Number.isNaN(parsed) || parsed <= 0) {
    throw new Error(t("Owner NodeID must be a positive number."))
  }
  return parsed
}

const normalizeName = (value: string) => value.trim()

const ensureReady = () => {
  if (!sessionStore.connected) {
    throw new Error(t("Connect to a session before sending VarPool requests."))
  }
  if (!selfNodeId.value) {
    throw new Error(t("Login to a node before using VarPool operations."))
  }
}

const loadHomeDefaults = async () => {
  try {
    const state = await LoadHomeState()
    fallbackIdentity.nodeId = Number(state?.nodeId ?? 0)
    fallbackIdentity.hubId = Number(state?.hubId ?? 0)
  } catch (err) {
    console.warn(err)
  }
  varpool.state.targetId = ""
  varpool.setIdentity(selfNodeId.value, hubId.value)
}

const refreshAll = async () => {
  if (busy.value) return
  busy.value = true
  const errorMessage = (err: unknown) => (err instanceof Error ? err.message : String(err))
  const isCode4NotFound = (err: unknown) => {
    const msg = errorMessage(err).toLowerCase()
    return msg.includes("(code=4)") || /\bcode=4\b/.test(msg)
  }

  let mineFailed = false
  let watchAttempted = 0
  let watchNotFound = 0
  let watchFailed = 0

  try {
    ensureReady()

    try {
      await varpool.listMine()
    } catch (err) {
      if (!isCode4NotFound(err)) {
        mineFailed = true
        console.warn(err)
      }
    }

    const keys = varpool.state.keys.slice()
    for (const key of keys) {
      if (selfNodeId.value && Number(key.owner ?? 0) === selfNodeId.value) {
        continue
      }
      watchAttempted += 1
      try {
        await varpool.getVar(key)
      } catch (err) {
        if (isCode4NotFound(err)) {
          watchNotFound += 1
        } else {
          watchFailed += 1
          console.warn(err)
        }
      }
    }

    if (!mineFailed && watchNotFound === 0 && watchFailed === 0) {
      toast.success(t("VarPool refreshed."))
      return
    }

    const parts: string[] = []
    if (mineFailed) parts.push(t("mine list failed"))
    if (watchNotFound) parts.push(t("not found: {count}/{total}", { count: watchNotFound, total: watchAttempted }))
    if (watchFailed) parts.push(t("failed: {count}/{total}", { count: watchFailed, total: watchAttempted }))
    toast.warn(t("VarPool refreshed with issues."), parts.join(" · "))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to refresh VarPool data."))
  } finally {
    busy.value = false
  }
}

const refreshKey = async (key: VarPoolKey) => {
  if (busy.value) return
  busy.value = true
  try {
    ensureReady()
    await varpool.getVar(key)
    toast.success(t("Variable refreshed."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to refresh variable."))
  } finally {
    busy.value = false
  }
}

const openAddMineDialog = () => {
  addMineDialog.open = true
  addMineDialog.name = ""
  addMineDialog.value = ""
  addMineDialog.visibility = "public"
  addMineDialog.kind = "string"
}

const closeAddMineDialog = () => {
  addMineDialog.open = false
}

const submitAddMine = async () => {
  if (busy.value) return
  busy.value = true
  try {
    ensureReady()
    const name = normalizeName(addMineDialog.name)
    if (!name) {
      throw new Error(t("Variable name is required."))
    }
    const owner = selfNodeId.value
    if (!owner) {
      throw new Error(t("Owner NodeID is required."))
    }
    const value = addMineDialog.value
    if (!value.trim()) {
      throw new Error(t("Variable value is required."))
    }
    const visibility = addMineDialog.visibility || "public"
    const kind = addMineDialog.kind || "string"
    await varpool.setVar({ name, owner }, value, visibility, kind)
    await varpool.getVar({ name, owner })
    closeAddMineDialog()
    toast.success(t("Variable added."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to add variable."))
  } finally {
    busy.value = false
  }
}

const openAddWatchDialog = () => {
  addWatchDialog.open = true
  addWatchDialog.name = ""
  addWatchDialog.owner = ""
}

const closeAddWatchDialog = () => {
  addWatchDialog.open = false
}

const submitAddWatch = async () => {
  if (busy.value) return
  busy.value = true
  try {
    ensureReady()
    const name = normalizeName(addWatchDialog.name)
    const owner = parseOwner(addWatchDialog.owner, true)
    await varpool.addWatchKey({ name, owner })
    await varpool.getVar({ name, owner })
    closeAddWatchDialog()
    toast.success(t("Watch added."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to add watch."))
  } finally {
    busy.value = false
  }
}

const openEditDialog = (key: VarPoolKey) => {
  const value = varpool.valueForKey(key)
  editDialog.open = true
  editDialog.name = key.name
  editDialog.owner = Number(key.owner ?? value.owner ?? selfNodeId.value ?? 0)
  editDialog.value = value.value
  editDialog.visibility = value.visibility || "public"
  editDialog.kind = value.kind || "string"
}

const closeEditDialog = () => {
  editDialog.open = false
}

const submitEdit = async () => {
  if (busy.value) return
  busy.value = true
  try {
    ensureReady()
    const name = normalizeName(editDialog.name)
    if (!name) {
      throw new Error(t("Variable name is required."))
    }
    const owner = editDialog.owner || selfNodeId.value
    if (!owner) {
      throw new Error(t("Owner NodeID is required."))
    }
    const visibility = editDialog.visibility || "public"
    const kind = editDialog.kind || "string"
    const value = editDialog.value
    if (!value.trim()) {
      throw new Error(t("Variable value is required."))
    }
    await varpool.setVar({ name, owner }, value, visibility, kind)
    await varpool.getVar({ name, owner })
    closeEditDialog()
    toast.success(t("Variable updated."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to update variable."))
  } finally {
    busy.value = false
  }
}

const reloadWatchList = async (force = false) => {
  if (busy.value && !force) return
  busy.value = true
  try {
    await varpool.loadWatchList()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to load watch list."))
  } finally {
    busy.value = false
  }
}

const persistWatchList = async () => {
  if (busy.value) return
  busy.value = true
  try {
    await varpool.saveWatchList()
    toast.success(t("Watch list saved."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to save watch list."))
  } finally {
    busy.value = false
  }
}

const revokeKey = async (key: VarPoolKey) => {
  if (busy.value) return
  busy.value = true
  try {
    ensureReady()
    await varpool.revokeVar(key)
    toast.success(t("Variable revoked."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to revoke variable."))
  } finally {
    busy.value = false
  }
}

const removeKey = async (key: VarPoolKey) => {
  if (busy.value) return
  busy.value = true
  try {
    const value = varpool.valueForKey(key)
    if (value.subKnown && value.subscribed) {
      await varpool.unsubscribeVar(key)
    }
    await varpool.removeWatchKey(key)
    toast.success(t("Removed."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to remove variable."))
  } finally {
    busy.value = false
  }
}

const toggleSubscribe = async (key: VarPoolKey) => {
  if (busy.value) return
  busy.value = true
  try {
    ensureReady()
    const value = varpool.valueForKey(key)
    if (value.subKnown && value.subscribed) {
      await varpool.unsubscribeVar(key)
      toast.success(t("Unsubscribed."))
    } else {
      await varpool.subscribeVar(key)
      toast.success(t("Subscribed."))
    }
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to update subscription."))
  } finally {
    busy.value = false
  }
}

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  () => {
    varpool.setIdentity(selfNodeId.value, hubId.value)
  }
)

watch(
  () => profileStore.state.current,
  async () => {
    await loadHomeDefaults()
    await reloadWatchList(true)
    if (sessionStore.connected && selfNodeId.value) {
      void refreshAll()
    }
  }
)

onMounted(async () => {
  await loadHomeDefaults()
  await reloadWatchList(true)
  if (sessionStore.connected && selfNodeId.value) {
    void refreshAll()
  }
})
</script>

<template>
  <section class="space-y-6">
    <PageHero>
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

    <section v-if="activeTab === 'control'" class="space-y-4">
      <section class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <CardHeader
          class="items-center"
          :title="t('Target & Identity')"
          :description="t('Use your logged-in node to list variables and manage watch targets.')"
          title-tag="h3"
          title-class="text-lg"
        >
          <template #actions>
            <Badge :class="connectedTone">{{ connectedLabel }}</Badge>
          </template>
        </CardHeader>

        <div class="mt-4 space-y-4">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Target Node ID") }}
            </label>
            <input
              v-model="varpool.state.targetId"
              :placeholder="hubId ? String(hubId) : t('Hub NodeID')"
              :class="inputClass"
            />
          </div>

          <div class="flex flex-wrap gap-2">
            <Button :disabled="busy" @click="refreshAll">{{ t("Refresh All") }}</Button>
            <Button variant="outline" :disabled="busy" @click="persistWatchList">{{ t("Save Watch List") }}</Button>
          </div>
        </div>
      </section>

      <section class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <CardHeader :title="t('VarPool Status')" title-tag="h3" title-class="text-lg" />
        <div class="mt-4 space-y-3 text-sm text-muted-foreground">
          <div
            v-for="item in summaryItems"
            :key="item.label"
            class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border/60 bg-background/70 px-3 py-2"
          >
            <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ item.label }}</p>
            <p class="font-medium text-foreground">{{ item.value }}</p>
          </div>
        </div>
      </section>

      <section class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <CardHeader class="items-center" :title="t('Active List')" title-tag="h3" title-class="text-lg">
          <template #actions>
            <Badge variant="outline">{{ t("{count} active", { count: subscribedEntries.length }) }}</Badge>
          </template>
        </CardHeader>

        <div class="mt-4 space-y-3">
          <div
            v-for="entry in subscribedEntries"
            :key="`${entry.key.name}-${entry.key.owner}`"
            class="rounded-xl border border-border/60 bg-background/70 px-4 py-3"
          >
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div>
                <p class="font-semibold">{{ entry.key.name }}</p>
                <p class="text-xs text-muted-foreground">{{ t("Owner {owner}", { owner: entry.key.owner ?? "-" }) }}</p>
              </div>
              <Badge variant="secondary">{{ t("Subscribed") }}</Badge>
            </div>
          </div>
          <p v-if="subscribedEntries.length === 0" class="text-sm text-muted-foreground">
            {{ t("No active subscriptions.") }}
          </p>
        </div>
      </section>
    </section>

    <section v-if="activeTab === 'mine'" class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
      <CardHeader class="items-center" :title="t('My Variables')" title-tag="h3" title-class="text-lg">
        <template #actions>
          <div class="flex flex-wrap items-center gap-2">
            <Badge variant="outline">{{ t("Updated: {time}", { time: varpool.state.lastFrameAt || "-" }) }}</Badge>
            <Button size="sm" variant="outline" :disabled="busy" @click="openAddMineDialog">
              {{ t("Add Variable") }}
            </Button>
          </div>
        </template>
      </CardHeader>

      <div class="mt-4 space-y-3">
        <article
          v-for="entry in mineEntries"
          :key="`${entry.key.name}-${entry.key.owner}`"
          class="rounded-2xl border border-border/60 bg-background/70 p-5"
        >
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h4 class="text-base font-semibold">{{ entry.key.name }}</h4>
              <p class="text-xs text-muted-foreground">
                {{ t("Owner {owner}", { owner: entry.key.owner ?? "-" }) }} ·
                {{ displayMeta(entry.snapshot.visibility) }} ·
                {{ displayMeta(entry.snapshot.kind) }}
              </p>
            </div>
            <Badge variant="secondary">{{ t("Mine") }}</Badge>
          </div>
          <p class="mt-3 rounded-lg border border-border/60 bg-card/90 px-3 py-2 text-sm">
            {{ entry.snapshot.value || "-" }}
          </p>
          <div class="mt-4 flex flex-wrap gap-2">
            <Button size="sm" variant="outline" :disabled="busy" @click="refreshKey(entry.key)">
              {{ t("Refresh") }}
            </Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="openEditDialog(entry.key)">
              {{ t("Edit") }}
            </Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="revokeKey(entry.key)">
              {{ t("Revoke") }}
            </Button>
            <Button size="sm" variant="ghost" :disabled="busy" @click="removeKey(entry.key)">
              {{ t("Remove") }}
            </Button>
          </div>
        </article>

        <p v-if="mineEntries.length === 0" class="text-sm text-muted-foreground">{{ t("No variables yet.") }}</p>
      </div>
    </section>

    <section v-if="activeTab === 'watch'" class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
      <CardHeader class="items-center" :title="t('Watched Variables')" title-tag="h3" title-class="text-lg">
        <template #actions>
          <div class="flex flex-wrap items-center gap-2">
            <Button size="sm" variant="outline" :disabled="busy" @click="openNodeVarsDialog">{{ t("Node Vars") }}</Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="openAddWatchDialog">{{ t("Add Watch") }}</Button>
            <Button size="sm" variant="ghost" :disabled="busy" @click="reloadWatchList">{{ t("Reload Saved") }}</Button>
          </div>
        </template>
      </CardHeader>

      <div class="mt-4 space-y-3">
        <article
          v-for="entry in watchEntries"
          :key="`${entry.key.name}-${entry.key.owner}`"
          class="rounded-2xl border border-border/60 bg-background/70 p-5"
        >
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h4 class="text-base font-semibold">{{ entry.key.name }}</h4>
              <p class="text-xs text-muted-foreground">
                {{ t("Owner {owner}", { owner: entry.key.owner ?? "-" }) }} ·
                {{ displayMeta(entry.snapshot.visibility) }} ·
                {{ displayMeta(entry.snapshot.kind) }}
              </p>
            </div>
            <Badge v-if="entry.snapshot.subKnown && entry.snapshot.subscribed" variant="secondary">
              {{ t("Subscribed") }}
            </Badge>
          </div>
          <p class="mt-3 rounded-lg border border-border/60 bg-card/90 px-3 py-2 text-sm">
            {{ entry.snapshot.value || "-" }}
          </p>
          <div class="mt-4 flex flex-wrap gap-2">
            <Button size="sm" variant="outline" :disabled="busy" @click="refreshKey(entry.key)">
              {{ t("Refresh") }}
            </Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="openEditDialog(entry.key)">
              {{ t("Edit") }}
            </Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="revokeKey(entry.key)">
              {{ t("Revoke") }}
            </Button>
            <Button size="sm" variant="ghost" :disabled="busy" @click="removeKey(entry.key)">
              {{ t("Remove") }}
            </Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="toggleSubscribe(entry.key)">
              {{ entry.snapshot.subKnown && entry.snapshot.subscribed ? t("Unsubscribe") : t("Subscribe") }}
            </Button>
          </div>
        </article>

        <p v-if="watchEntries.length === 0" class="text-sm text-muted-foreground">
          {{ t("No watched variables yet.") }}
        </p>
      </div>
    </section>

    <Overlay
      :open="addMineDialog.open"
      overlayClass="bg-black/40 p-4"
      closeOnBackdrop
      @close="closeAddMineDialog"
    >
      <div class="flex max-h-[85vh] w-full max-w-lg flex-col overflow-hidden rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl">
        <CardHeader
          class="items-start"
          :title="t('Create Variable')"
          :description="t('Owner defaults to your current NodeID.')"
          title-tag="h3"
          title-class="text-lg"
        >
          <template #actions>
            <Badge variant="secondary">{{ t("New") }}</Badge>
          </template>
        </CardHeader>

        <div class="mt-5 min-h-0 flex-1 overflow-y-auto">
          <div class="grid gap-4 px-1 py-1 pr-2">
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-sm">
              <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Owner") }}</p>
              <p class="mt-1 font-medium">{{ selfNodeId || "-" }}</p>
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Name") }}
              </label>
              <input v-model="addMineDialog.name" :class="inputClass" :placeholder="t('status.flag')" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Value") }}
              </label>
              <input v-model="addMineDialog.value" :class="inputClass" :placeholder="t('ready')" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Visibility") }}
              </label>
              <select v-model="addMineDialog.visibility" :class="inputClass">
                <option value="public">{{ t("public") }}</option>
                <option value="private">{{ t("private") }}</option>
              </select>
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Type") }}
              </label>
              <input v-model="addMineDialog.kind" :class="inputClass" :placeholder="t('string')" />
            </div>
          </div>
        </div>

        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <Button variant="outline" :disabled="busy" @click="closeAddMineDialog">{{ t("Cancel") }}</Button>
          <Button :disabled="busy" @click="submitAddMine">{{ t("Save") }}</Button>
        </div>
      </div>
    </Overlay>

    <NodeVarsDialog :open="nodeVarsDialogOpen" :ownerId="nodeVarsDialogOwnerId" @close="closeNodeVarsDialog" />

    <Overlay
      :open="addWatchDialog.open"
      overlayClass="bg-black/40 p-4"
      closeOnBackdrop
      @close="closeAddWatchDialog"
    >
      <div class="flex max-h-[85vh] w-full max-w-lg flex-col overflow-hidden rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl">
        <CardHeader
          class="items-start"
          :title="t('Add Watch')"
          :description="t('Track variables owned by another node.')"
          title-tag="h3"
          title-class="text-lg"
        >
          <template #actions>
            <Badge variant="secondary">{{ t("Watch") }}</Badge>
          </template>
        </CardHeader>

        <div class="mt-5 min-h-0 flex-1 overflow-y-auto">
          <div class="grid gap-4 px-1 py-1 pr-2">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Name") }}
              </label>
              <input v-model="addWatchDialog.name" :class="inputClass" :placeholder="t('metrics.load')" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Owner NodeID") }}
              </label>
              <input v-model="addWatchDialog.owner" :class="inputClass" :placeholder="t('Owner NodeID')" />
            </div>
          </div>
        </div>

        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <Button variant="outline" :disabled="busy" @click="closeAddWatchDialog">{{ t("Cancel") }}</Button>
          <Button :disabled="busy" @click="submitAddWatch">{{ t("Save") }}</Button>
        </div>
      </div>
    </Overlay>

    <Overlay
      :open="editDialog.open"
      overlayClass="bg-black/40 p-4"
      closeOnBackdrop
      @close="closeEditDialog"
    >
      <div class="flex max-h-[85vh] w-full max-w-lg flex-col overflow-hidden rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl">
        <CardHeader
          class="items-start"
          :title="t('Update Variable')"
          :description="t('Visibility may not apply to other node owners.')"
          title-tag="h3"
          title-class="text-lg"
        >
          <template #actions>
            <Badge variant="secondary">{{ t("Edit") }}</Badge>
          </template>
        </CardHeader>

        <div class="mt-5 min-h-0 flex-1 overflow-y-auto">
          <div class="grid gap-4 px-1 py-1 pr-2">
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-sm">
              <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Name") }}</p>
              <p class="mt-1 font-medium">{{ editDialog.name || "-" }}</p>
            </div>
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-sm">
                <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Owner") }}</p>
                <p class="mt-1 font-medium">{{ editDialog.owner || "-" }}</p>
              </div>
              <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-sm">
                <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Type") }}</p>
                <p class="mt-1 font-medium">{{ displayMeta(editDialog.kind, "string") }}</p>
              </div>
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Value") }}
              </label>
              <input v-model="editDialog.value" :class="inputClass" :placeholder="t('value')" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Visibility") }}
              </label>
              <select v-model="editDialog.visibility" :class="inputClass">
                <option value="public">{{ t("public") }}</option>
                <option value="private">{{ t("private") }}</option>
              </select>
            </div>
          </div>
        </div>

        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <Button variant="outline" :disabled="busy" @click="closeEditDialog">{{ t("Cancel") }}</Button>
          <Button :disabled="busy" @click="submitEdit">{{ t("Save") }}</Button>
        </div>
      </div>
    </Overlay>
  </section>
</template>
