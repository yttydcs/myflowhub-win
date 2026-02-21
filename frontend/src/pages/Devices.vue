<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { Button } from "@/components/ui/button"
import { useDevicesStore } from "@/stores/devices"
import { useSessionStore } from "@/stores/session"

const devicesStore = useDevicesStore()
const sessionStore = useSessionStore()

const message = ref("")

const identityLabel = computed(() => {
  const nodeId = Number(sessionStore.auth.nodeId || 0)
  const hubId = Number(sessionStore.auth.hubId || 0)
  if (!nodeId && !hubId) return "Not logged in"
  return `node=${nodeId || "-"} hub=${hubId || "-"}`
})

const listModeLabel = computed(() =>
  devicesStore.state.listMode === "subtree" ? "Subtree (direct + self)" : "Direct"
)

const refresh = async (mode: "direct" | "subtree") => {
  message.value = ""
  try {
    if (mode === "subtree") {
      await devicesStore.listSubtree()
    } else {
      await devicesStore.listDirect()
    }
  } catch (err) {
    console.warn(err)
    message.value = (err as Error)?.message || "Failed to query devices."
  }
}

const setTarget = (nodeId: number) => {
  if (!nodeId) return
  devicesStore.state.targetId = String(nodeId)
}

watch(
  () => sessionStore.auth.hubId,
  (hubId) => {
    const numeric = Number(hubId || 0)
    if (!devicesStore.state.targetId && numeric) {
      devicesStore.state.targetId = String(numeric)
    }
  },
  { immediate: true }
)

onMounted(async () => {
  if (sessionStore.connected && sessionStore.auth.nodeId && sessionStore.auth.hubId) {
    await refresh("direct")
  }
})
</script>

<template>
  <section class="space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">Session</p>
        <h1 class="text-2xl font-semibold">Devices</h1>
        <p class="text-sm text-muted-foreground">
          Query the management plane for device/node lists. Subtree is not recursive.
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
          <span class="font-semibold uppercase tracking-[0.2em]">Identity</span>
          <span class="font-mono text-[11px] text-foreground">{{ identityLabel }}</span>
        </div>
        <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
          <span class="font-semibold uppercase tracking-[0.2em]">Target</span>
          <input
            v-model="devicesStore.state.targetId"
            class="h-7 w-24 rounded-md border border-input bg-background px-2 text-xs text-foreground"
            placeholder="Node ID"
          />
        </div>
        <Button variant="outline" size="sm" @click="refresh('direct')">List Direct</Button>
        <Button variant="outline" size="sm" @click="refresh('subtree')">List Subtree</Button>
      </div>
    </div>

    <section class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 class="text-sm font-semibold">Nodes</h2>
          <p class="text-xs text-muted-foreground">
            Mode: <span class="font-semibold text-foreground">{{ listModeLabel }}</span>
          </p>
        </div>
        <p class="text-xs text-muted-foreground">
          Tip: click a node to set it as Target.
        </p>
      </div>

      <div class="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
        <button
          v-for="node in devicesStore.state.nodes"
          :key="node.nodeId"
          type="button"
          class="rounded-xl border border-transparent bg-background/70 px-3 py-2 text-left text-sm transition hover:border-border/70 hover:bg-muted/60"
          @click="setTarget(node.nodeId)"
        >
          <p class="font-semibold">Node {{ node.nodeId }}</p>
          <p class="text-xs text-muted-foreground">
            {{ node.hasChildren ? "Has children" : "Leaf node" }}
          </p>
        </button>
        <div v-if="!devicesStore.state.nodes.length" class="text-xs text-muted-foreground">
          No nodes yet. Connect, login, and query again.
        </div>
      </div>
    </section>

    <p v-if="message" class="text-sm text-rose-600">{{ message }}</p>
    <p v-else-if="devicesStore.state.message" class="text-sm text-muted-foreground">
      {{ devicesStore.state.message }}
    </p>
  </section>
</template>

