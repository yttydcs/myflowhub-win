<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useProfileStore } from "@/stores/profile"
import { useSessionStore } from "@/stores/session"
import { useVarPoolStore, type VarPoolKey } from "@/stores/varpool"
import { useToastStore } from "@/stores/toast"
import { HomeState as LoadHomeState } from "../../wailsjs/go/main/App"

const profileStore = useProfileStore()
const sessionStore = useSessionStore()
const varpool = useVarPoolStore()
const toast = useToastStore()

const busy = ref(false)

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

const fallbackIdentity = reactive({
  nodeId: 0,
  hubId: 0
})

const inputClass =
  "mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

const connectedLabel = computed(() => (sessionStore.connected ? "Connected" : "Disconnected"))
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

const subscribedKeys = computed(() =>
  varpool.state.keys.filter((key) => {
    const value = varpool.valueForKey(key)
    return value.subKnown && value.subscribed
  })
)

const parseOwner = (value: string, required: boolean) => {
  const trimmed = value.trim()
  if (!trimmed) {
    if (required) {
      throw new Error("Owner NodeID is required.")
    }
    return 0
  }
  const parsed = Number.parseInt(trimmed, 10)
  if (Number.isNaN(parsed) || parsed <= 0) {
    throw new Error("Owner NodeID must be a positive number.")
  }
  return parsed
}

const normalizeName = (value: string) => value.trim()

const ensureReady = () => {
  if (!sessionStore.connected) {
    throw new Error("Connect to a session before sending VarPool requests.")
  }
  if (!selfNodeId.value) {
    throw new Error("Login to a node before using VarPool operations.")
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
  try {
    ensureReady()
    await varpool.listMine()
    for (const key of varpool.state.keys) {
      if (selfNodeId.value && Number(key.owner ?? 0) === selfNodeId.value) {
        continue
      }
      await varpool.getVar(key)
    }
    toast.success("VarPool refreshed.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to refresh VarPool data.")
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
    toast.success("Variable refreshed.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to refresh variable.")
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
      throw new Error("Variable name is required.")
    }
    const owner = selfNodeId.value
    if (!owner) {
      throw new Error("Owner NodeID is required.")
    }
    const value = addMineDialog.value
    if (!value.trim()) {
      throw new Error("Variable value is required.")
    }
    const visibility = addMineDialog.visibility || "public"
    const kind = addMineDialog.kind || "string"
    await varpool.setVar({ name, owner }, value, visibility, kind)
    await varpool.getVar({ name, owner })
    closeAddMineDialog()
    toast.success("Variable added.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to add variable.")
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
    toast.success("Watch added.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to add watch.")
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
      throw new Error("Variable name is required.")
    }
    const owner = editDialog.owner || selfNodeId.value
    if (!owner) {
      throw new Error("Owner NodeID is required.")
    }
    const visibility = editDialog.visibility || "public"
    const kind = editDialog.kind || "string"
    const value = editDialog.value
    if (!value.trim()) {
      throw new Error("Variable value is required.")
    }
    await varpool.setVar({ name, owner }, value, visibility, kind)
    await varpool.getVar({ name, owner })
    closeEditDialog()
    toast.success("Variable updated.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to update variable.")
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
    toast.errorOf(err, "Failed to load watch list.")
  } finally {
    busy.value = false
  }
}

const persistWatchList = async () => {
  if (busy.value) return
  busy.value = true
  try {
    await varpool.saveWatchList()
    toast.success("Watch list saved.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to save watch list.")
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
    toast.success("Variable revoked.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to revoke variable.")
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
    toast.success("Removed.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to remove variable.")
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
      toast.success("Unsubscribed.")
    } else {
      await varpool.subscribeVar(key)
      toast.success("Subscribed.")
    }
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to update subscription.")
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
  <section class="grid gap-6">
    <div class="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
      <div class="space-y-6">
        <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
                VarPool Control
              </p>
              <h3 class="text-lg font-semibold">Target & Identity</h3>
              <p class="text-sm text-muted-foreground">
                Use your logged-in node to list variables and watch other owners.
              </p>
            </div>
            <Badge :class="connectedTone">{{ connectedLabel }}</Badge>
          </div>

          <div class="mt-4 grid gap-4 lg:grid-cols-[1fr_auto]">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Target Node ID
              </label>
              <input
                v-model="varpool.state.targetId"
                :placeholder="hubId ? String(hubId) : 'Hub NodeID'"
                :class="inputClass"
              />
            </div>
            <div class="flex flex-col justify-end gap-2">
              <Button :disabled="busy" @click="refreshAll">Refresh All</Button>
              <Button variant="outline" :disabled="busy" @click="persistWatchList">
                Save Watch List
              </Button>
            </div>
          </div>

          <div class="mt-4 flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
            <span>NodeID: {{ selfNodeId || "-" }}</span>
            <span>HubID: {{ hubId || "-" }}</span>
            <span>Cached keys: {{ varpool.state.keys.length }}</span>
          </div>
        </div>
      </div>

      <div class="space-y-6">
        <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
            Subscriptions
          </p>
          <h3 class="mt-2 text-lg font-semibold">Active List</h3>
          <div class="mt-4 space-y-2 text-sm text-muted-foreground">
            <div
              v-for="key in subscribedKeys"
              :key="`${key.name}-${key.owner}`"
              class="rounded-lg border border-border/60 bg-background/70 px-3 py-2"
            >
              {{ key.name }} #{{ key.owner ?? "-" }}
            </div>
            <p v-if="subscribedKeys.length === 0">No active subscriptions.</p>
          </div>
        </div>

        <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
            Snapshot
          </p>
          <h3 class="mt-2 text-lg font-semibold">VarPool Status</h3>
          <div class="mt-4 space-y-3 text-sm text-muted-foreground">
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">Target</p>
              <p>{{ varpool.state.targetId || hubId || "-" }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">Last Frame</p>
              <p>{{ varpool.state.lastFrameAt || "-" }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">Watch Count</p>
              <p>{{ varpool.state.keys.length }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
            Cached Variables
          </p>
          <h3 class="text-lg font-semibold">VarPool Inventory</h3>
        </div>
        <Badge variant="outline">Updated: {{ varpool.state.lastFrameAt || "-" }}</Badge>
      </div>

      <div class="space-y-4">
        <div>
          <div class="flex flex-wrap items-center justify-between gap-3">
            <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
              My Variables
            </p>
            <div class="flex flex-wrap gap-2">
              <Button size="sm" variant="outline" :disabled="busy" @click="openAddMineDialog">
                Add Variable
              </Button>
            </div>
          </div>
          <div class="mt-3 grid gap-3">
            <div
              v-for="key in groupedKeys.mine"
              :key="`${key.name}-${key.owner}`"
              class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm"
            >
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <h4 class="text-base font-semibold">{{ key.name }}</h4>
                  <p class="text-xs text-muted-foreground">
                    Owner {{ key.owner ?? "-" }} ·
                    {{ varpool.valueForKey(key).visibility || "unknown" }} ·
                    {{ varpool.valueForKey(key).kind || "unknown" }}
                  </p>
                </div>
                <Badge variant="secondary">Mine</Badge>
              </div>
              <p class="mt-3 rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-sm">
                {{ varpool.valueForKey(key).value || "-" }}
              </p>
              <div class="mt-4 flex flex-wrap gap-2">
                <Button size="sm" variant="outline" :disabled="busy" @click="refreshKey(key)">
                  Refresh
                </Button>
                <Button size="sm" variant="outline" :disabled="busy" @click="openEditDialog(key)">
                  Edit
                </Button>
                <Button size="sm" variant="outline" :disabled="busy" @click="revokeKey(key)">
                  Revoke
                </Button>
                <Button size="sm" variant="ghost" :disabled="busy" @click="removeKey(key)">
                  Remove
                </Button>
              </div>
            </div>
            <p v-if="groupedKeys.mine.length === 0" class="text-sm text-muted-foreground">
              No variables yet.
            </p>
          </div>
        </div>

        <div>
          <div class="flex flex-wrap items-center justify-between gap-3">
            <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
              Watched Variables
            </p>
            <div class="flex flex-wrap gap-2">
              <Button size="sm" variant="outline" :disabled="busy" @click="openAddWatchDialog">
                Add Watch
              </Button>
              <Button size="sm" variant="ghost" :disabled="busy" @click="reloadWatchList">
                Reload Saved
              </Button>
            </div>
          </div>
          <div class="mt-3 grid gap-3">
            <div
              v-for="key in groupedKeys.others"
              :key="`${key.name}-${key.owner}`"
              class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm"
            >
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <h4 class="text-base font-semibold">{{ key.name }}</h4>
                  <p class="text-xs text-muted-foreground">
                    Owner {{ key.owner ?? "-" }} ·
                    {{ varpool.valueForKey(key).visibility || "unknown" }} ·
                    {{ varpool.valueForKey(key).kind || "unknown" }}
                  </p>
                </div>
                <Badge
                  v-if="varpool.valueForKey(key).subKnown && varpool.valueForKey(key).subscribed"
                  variant="secondary"
                >
                  Subscribed
                </Badge>
              </div>
              <p class="mt-3 rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-sm">
                {{ varpool.valueForKey(key).value || "-" }}
              </p>
              <div class="mt-4 flex flex-wrap gap-2">
                <Button size="sm" variant="outline" :disabled="busy" @click="refreshKey(key)">
                  Refresh
                </Button>
                <Button size="sm" variant="outline" :disabled="busy" @click="openEditDialog(key)">
                  Edit
                </Button>
                <Button size="sm" variant="outline" :disabled="busy" @click="revokeKey(key)">
                  Revoke
                </Button>
                <Button size="sm" variant="ghost" :disabled="busy" @click="removeKey(key)">
                  Remove
                </Button>
                <Button size="sm" variant="outline" :disabled="busy" @click="toggleSubscribe(key)">
                  {{
                    varpool.valueForKey(key).subKnown && varpool.valueForKey(key).subscribed
                      ? "Unsubscribe"
                      : "Subscribe"
                  }}
                </Button>
              </div>
            </div>
            <p v-if="groupedKeys.others.length === 0" class="text-sm text-muted-foreground">
              No watched variables yet.
            </p>
          </div>
        </div>
      </div>
    </div>

    <div
      v-if="addMineDialog.open"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      @click.self="closeAddMineDialog"
    >
      <div class="w-full max-w-lg rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl">
        <div class="flex items-start justify-between gap-3">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
              VarPool Add
            </p>
            <h3 class="mt-2 text-lg font-semibold">Create Variable</h3>
            <p class="text-sm text-muted-foreground">
              Owner defaults to your current NodeID.
            </p>
          </div>
          <Badge variant="secondary">New</Badge>
        </div>

        <div class="mt-4 grid gap-4">
          <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-sm">
            <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Owner</p>
            <p class="mt-1 font-medium">{{ selfNodeId || "-" }}</p>
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Name
            </label>
            <input v-model="addMineDialog.name" :class="inputClass" placeholder="status.flag" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Value
            </label>
            <input v-model="addMineDialog.value" :class="inputClass" placeholder="ready" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Visibility
            </label>
            <select v-model="addMineDialog.visibility" :class="inputClass">
              <option value="public">public</option>
              <option value="private">private</option>
            </select>
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Type
            </label>
            <input v-model="addMineDialog.kind" :class="inputClass" placeholder="string" />
          </div>
        </div>

        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <Button variant="outline" :disabled="busy" @click="closeAddMineDialog">Cancel</Button>
          <Button :disabled="busy" @click="submitAddMine">Save</Button>
        </div>
      </div>
    </div>

    <div
      v-if="addWatchDialog.open"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      @click.self="closeAddWatchDialog"
    >
      <div class="w-full max-w-lg rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl">
        <div class="flex items-start justify-between gap-3">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
              VarPool Watch
            </p>
            <h3 class="mt-2 text-lg font-semibold">Add Watch</h3>
            <p class="text-sm text-muted-foreground">
              Track variables owned by another node.
            </p>
          </div>
          <Badge variant="secondary">Watch</Badge>
        </div>

        <div class="mt-4 grid gap-4">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Name
            </label>
            <input v-model="addWatchDialog.name" :class="inputClass" placeholder="metrics.load" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Owner NodeID
            </label>
            <input v-model="addWatchDialog.owner" :class="inputClass" placeholder="Owner NodeID" />
          </div>
        </div>

        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <Button variant="outline" :disabled="busy" @click="closeAddWatchDialog">Cancel</Button>
          <Button :disabled="busy" @click="submitAddWatch">Save</Button>
        </div>
      </div>
    </div>

    <div
      v-if="editDialog.open"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      @click.self="closeEditDialog"
    >
      <div class="w-full max-w-lg rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl">
        <div class="flex items-start justify-between gap-3">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
              VarPool Edit
            </p>
            <h3 class="mt-2 text-lg font-semibold">Update Variable</h3>
            <p class="text-sm text-muted-foreground">
              Visibility may not apply to other node owners.
            </p>
          </div>
          <Badge variant="secondary">Edit</Badge>
        </div>

        <div class="mt-4 grid gap-4">
          <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-sm">
            <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Name</p>
            <p class="mt-1 font-medium">{{ editDialog.name || "-" }}</p>
          </div>
          <div class="grid gap-3 sm:grid-cols-2">
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-sm">
              <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Owner</p>
              <p class="mt-1 font-medium">{{ editDialog.owner || "-" }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-sm">
              <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Type</p>
              <p class="mt-1 font-medium">{{ editDialog.kind || "string" }}</p>
            </div>
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Value
            </label>
            <input v-model="editDialog.value" :class="inputClass" placeholder="value" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Visibility
            </label>
            <select v-model="editDialog.visibility" :class="inputClass">
              <option value="public">public</option>
              <option value="private">private</option>
            </select>
          </div>
        </div>

        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <Button variant="outline" :disabled="busy" @click="closeEditDialog">Cancel</Button>
          <Button :disabled="busy" @click="submitEdit">Save</Button>
        </div>
      </div>
    </div>
  </section>
</template>
