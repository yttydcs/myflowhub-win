<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import { useManagementStore } from "@/stores/management"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"

const mgmtStore = useManagementStore()
const sessionStore = useSessionStore()
const toast = useToastStore()

const editOpen = ref(false)
const configDraft = reactive({ key: "", value: "" })

const listModeLabel = computed(() =>
  mgmtStore.state.listMode === "subtree" ? "Subtree" : "Direct"
)

const refreshNodes = async (mode: "direct" | "subtree") => {
  try {
    if (mode === "subtree") {
      await mgmtStore.listSubtree()
    } else {
      await mgmtStore.listNodes()
    }
    toast.success(mode === "subtree" ? "Subtree loaded." : "Direct nodes loaded.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to load node list.")
  }
}

const selectNode = async (nodeId: number) => {
  try {
    await mgmtStore.selectNode(nodeId)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to load config.")
  }
}

const refreshConfig = async () => {
  try {
    await mgmtStore.refreshConfig()
    toast.success("Config refreshed.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to refresh config.")
  }
}

const openEdit = (key: string, value: string) => {
  configDraft.key = key
  configDraft.value = value
  editOpen.value = true
}

const saveConfig = async () => {
  try {
    await mgmtStore.setConfig(configDraft.key, configDraft.value)
    editOpen.value = false
    toast.success("Config updated.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to update config.")
  }
}

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  ([nodeId, hubId]) => {
    mgmtStore.setIdentity(Number(nodeId), Number(hubId))
  },
  { immediate: true }
)

onMounted(async () => {
  try {
    await refreshNodes("direct")
  } catch {
    // handled in refreshNodes
  }
})
</script>

<template>
  <section class="space-y-6">
    <div class="grid gap-6 xl:grid-cols-[280px_minmax(0,1fr)]">
      <section class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 class="text-sm font-semibold">Nodes</h2>
            <p class="text-xs text-muted-foreground">{{ listModeLabel }}</p>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
              <span class="font-semibold uppercase tracking-[0.2em]">Target</span>
              <input
                v-model="mgmtStore.state.targetId"
                class="h-7 w-24 rounded-md border border-input bg-background px-2 text-xs text-foreground"
                placeholder="Node ID"
              />
            </div>
            <Button variant="outline" size="sm" @click="refreshNodes('direct')">List Direct</Button>
            <Button variant="outline" size="sm" @click="refreshNodes('subtree')">List Subtree</Button>
          </div>
        </div>
        <div class="mt-3 space-y-2">
          <button
            v-for="node in mgmtStore.state.nodes"
            :key="node.nodeId"
            type="button"
            class="w-full rounded-xl border px-3 py-2 text-left text-sm transition"
            :class="mgmtStore.state.selectedNodeId === node.nodeId ? 'border-primary/60 bg-primary/10' : 'border-transparent hover:border-border/60 hover:bg-muted/60'"
            @click="selectNode(node.nodeId)"
          >
            <p class="font-semibold">Node {{ node.nodeId }}</p>
            <p class="text-xs text-muted-foreground">
              {{ node.hasChildren ? "Has children" : "Leaf node" }}
            </p>
          </button>
          <div v-if="!mgmtStore.state.nodes.length" class="text-xs text-muted-foreground">
            No nodes yet. Connect and refresh.
          </div>
        </div>
      </section>

      <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-sm font-semibold">Config</h2>
            <p class="text-xs text-muted-foreground">
              Node {{ mgmtStore.state.selectedNodeId || "-" }}
            </p>
          </div>
          <Button
            size="sm"
            variant="outline"
            :disabled="!mgmtStore.state.selectedNodeId"
            @click="refreshConfig"
          >
            Refresh
          </Button>
        </div>
        <div class="mt-4 max-h-[460px] space-y-2 overflow-y-auto">
          <div
            v-for="entry in mgmtStore.state.configEntries"
            :key="entry.key"
            class="flex items-center justify-between gap-3 rounded-xl border border-border/60 bg-background/70 px-3 py-2 text-xs"
          >
            <div class="min-w-0 flex-1">
              <p class="font-semibold">{{ entry.key }}</p>
              <p class="truncate text-muted-foreground">{{ entry.value }}</p>
            </div>
            <Button size="sm" variant="outline" @click="openEdit(entry.key, entry.value)">
              Edit
            </Button>
          </div>
          <div v-if="!mgmtStore.state.configEntries.length" class="text-xs text-muted-foreground">
            Select a node to load config entries.
          </div>
        </div>
      </section>
    </div>

    <Overlay :open="editOpen" @close="editOpen = false">
      <div class="w-full max-w-lg rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">Edit Config</h2>
        <div class="mt-4 space-y-3">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Key
            </label>
            <input
              v-model="configDraft.key"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              disabled
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Value
            </label>
            <textarea
              v-model="configDraft.value"
              rows="4"
              class="mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            />
          </div>
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="editOpen = false">Cancel</Button>
          <Button @click="saveConfig">Save</Button>
        </div>
      </div>
    </Overlay>
  </section>
</template>
