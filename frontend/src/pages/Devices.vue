<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import type { DeviceTreeNode, DevicesMode } from "@/stores/devices"
import { useDevicesStore } from "@/stores/devices"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"

const devicesStore = useDevicesStore()
const sessionStore = useSessionStore()
const toast = useToastStore()

type NodeInfoWire = {
  code?: number
  Code?: number
  msg?: string
  Msg?: string
  items?: Record<string, any>
  Items?: Record<string, any>
}

const autoLoaded = ref(false)

const identityLabel = computed(() => {
  const nodeId = Number(sessionStore.auth.nodeId || 0)
  const hubId = Number(sessionStore.auth.hubId || 0)
  if (!nodeId && !hubId) return "Not logged in"
  return `node=${nodeId || "-"} hub=${hubId || "-"}`
})

const ready = computed(() => {
  return Boolean(sessionStore.connected && sessionStore.auth.nodeId && sessionStore.auth.hubId)
})

const modeLabel = computed(() =>
  devicesStore.state.mode === "subtree" ? "Subtree (direct + self; not recursive)" : "Direct"
)

const flattenVisible = (root: DeviceTreeNode | null) => {
  const out: { node: DeviceTreeNode; depth: number }[] = []
  if (!root) return out

  const walk = (node: DeviceTreeNode, depth: number) => {
    out.push({ node, depth })
    if (!node.expanded) return
    if (!node.children || !node.children.length) return
    for (const child of node.children) {
      walk(child, depth + 1)
    }
  }

  walk(root, 0)
  return out
}

const visibleNodes = computed(() => flattenVisible(devicesStore.state.root))

const nodeInfoOpen = ref(false)
const nodeInfoNodeId = ref(0)
const nodeInfoLoading = ref(false)
const nodeInfoError = ref("")
const nodeInfoItems = ref<Record<string, string>>({})
let nodeInfoEpoch = 0

const callMgmt = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.management?.ManagementService
  const fn = api?.[method]
  if (!fn) {
    throw new Error(`Management binding '${method}' unavailable`)
  }
  return fn(...args)
}

const loadNodeInfo = async (targetID: number) => {
  if (!sessionStore.connected) {
    throw new Error("Connect before querying node info.")
  }
  const sourceID = Number(sessionStore.auth.nodeId || 0)
  if (!sourceID) {
    throw new Error("Login required to query node info.")
  }
  const resp = await callMgmt<NodeInfoWire>("NodeInfoSimple", sourceID, targetID)
  const itemsRaw = resp?.items ?? resp?.Items ?? {}
  const items: Record<string, string> = {}
  for (const [key, value] of Object.entries(itemsRaw || {})) {
    items[String(key)] = value == null ? "" : String(value)
  }
  return items
}

const refreshNodeInfo = async () => {
  if (!nodeInfoNodeId.value) return
  nodeInfoError.value = ""
  const myEpoch = ++nodeInfoEpoch
  nodeInfoLoading.value = true
  try {
    const items = await loadNodeInfo(nodeInfoNodeId.value)
    if (nodeInfoEpoch !== myEpoch) return
    nodeInfoItems.value = items
    nodeInfoError.value = ""
  } catch (err) {
    if (nodeInfoEpoch !== myEpoch) return
    const message = err instanceof Error ? err.message : String(err)
    nodeInfoError.value = message || "Unknown error."
    toast.errorOf(err, "Failed to load node info.")
  } finally {
    if (nodeInfoEpoch !== myEpoch) return
    nodeInfoLoading.value = false
  }
}

const openNodeInfo = async (node: DeviceTreeNode) => {
  nodeInfoOpen.value = true
  nodeInfoNodeId.value = node.nodeId
  nodeInfoItems.value = {}
  nodeInfoError.value = ""
  await refreshNodeInfo()
}

const closeNodeInfo = () => {
  nodeInfoOpen.value = false
  nodeInfoNodeId.value = 0
  nodeInfoItems.value = {}
  nodeInfoError.value = ""
  nodeInfoLoading.value = false
}

const sortedNodeInfoItems = computed(() => {
  return Object.entries(nodeInfoItems.value).sort((a, b) => a[0].localeCompare(b[0]))
})

const loadRoot = async () => {
  try {
    await devicesStore.loadRoot()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to load root.")
  }
}

const onModeChanged = async (mode: DevicesMode) => {
  devicesStore.state.mode = mode
  await loadRoot()
}

const onRootEnter = async () => {
  await loadRoot()
}

const toggleNode = async (node: DeviceTreeNode) => {
  try {
    await devicesStore.toggle(node.key)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to expand node.")
  }
}

const retryNode = async (node: DeviceTreeNode) => {
  try {
    await devicesStore.retry(node.key)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to retry node.")
  }
}

watch(
  () => ready.value,
  (isReady) => {
    if (!isReady) {
      autoLoaded.value = false
      return
    }
    if (autoLoaded.value) return
    autoLoaded.value = true
    void loadRoot()
  },
  { immediate: true }
)

onMounted(async () => {
  if (!ready.value) return
  if (devicesStore.state.root) return
  await loadRoot()
})
</script>

<template>
  <section class="space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">Session</p>
        <h1 class="text-2xl font-semibold">Devices</h1>
        <p class="text-sm text-muted-foreground">
          Query the management plane and visualize nodes as a lazy-loaded tree. Subtree is not recursive.
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
          <span class="font-semibold uppercase tracking-[0.2em]">Identity</span>
          <span class="font-mono text-[11px] text-foreground">{{ identityLabel }}</span>
        </div>
        <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
          <span class="font-semibold uppercase tracking-[0.2em]">Mode</span>
          <select
            v-model="devicesStore.state.mode"
            class="h-7 rounded-md border border-input bg-background px-2 text-xs text-foreground"
            @change="onModeChanged(devicesStore.state.mode)"
          >
            <option value="direct">Direct</option>
            <option value="subtree">Subtree (direct + self)</option>
          </select>
        </div>
        <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
          <span class="font-semibold uppercase tracking-[0.2em]">Root</span>
          <input
            v-model="devicesStore.state.rootTargetId"
            class="h-7 w-28 rounded-md border border-input bg-background px-2 text-xs text-foreground"
            placeholder="Node ID"
            @keydown.enter.prevent="onRootEnter"
          />
        </div>
        <Button variant="outline" size="sm" @click="loadRoot">Reload</Button>
      </div>
    </div>

    <section class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 class="text-sm font-semibold">Nodes</h2>
          <p class="text-xs text-muted-foreground">
            Mode: <span class="font-semibold text-foreground">{{ modeLabel }}</span>
          </p>
        </div>
      </div>

      <div class="mt-4 space-y-2">
        <div
          v-for="{ node, depth } in visibleNodes"
          :key="node.key"
          class="cursor-pointer rounded-xl border border-border/60 bg-background/70 px-3 py-2 text-sm transition-colors hover:bg-background/80"
          @click="openNodeInfo(node)"
        >
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="flex min-w-0 items-center gap-2">
              <button
                type="button"
                class="h-7 w-7 rounded-md border border-border/70 bg-background text-xs text-foreground disabled:opacity-50"
                :style="{ marginLeft: `${depth * 16}px` }"
                :disabled="node.duplicate || node.loading"
                @click.stop="toggleNode(node)"
              >
                <span v-if="node.loading">…</span>
                <span v-else>{{ node.expanded ? "-" : "+" }}</span>
              </button>

              <div class="min-w-0">
                <p class="truncate font-semibold">
                  Node {{ node.nodeId }}
                  <span v-if="node.key.startsWith('root:')" class="text-xs font-normal text-muted-foreground">
                    (root)
                  </span>
                </p>
                <p class="truncate text-xs text-muted-foreground">
                  <span v-if="node.duplicate">Duplicate: expansion disabled.</span>
                  <span v-else-if="node.error">Error: {{ node.error }}</span>
                  <span v-else-if="node.children && node.children.length === 0">No children.</span>
                  <span v-else-if="node.children && node.children.length > 0">
                    Children: {{ node.children.length }}
                  </span>
                  <span v-else>Not loaded.</span>
                </p>
              </div>
            </div>

            <div class="flex flex-wrap items-center gap-2">
              <Badge v-if="node.duplicate" variant="secondary">Duplicate</Badge>
              <Badge
                v-else-if="node.children ? node.children.length > 0 : node.hasChildrenHint"
                variant="secondary"
              >
                Has children
              </Badge>
              <Badge v-else-if="node.children && node.children.length === 0" variant="secondary">
                Leaf
              </Badge>
              <Badge v-else variant="secondary">Unknown</Badge>

              <Button
                v-if="node.error && !node.duplicate"
                size="sm"
                variant="outline"
                :disabled="node.loading"
                @click.stop="retryNode(node)"
              >
                Retry
              </Button>
            </div>
          </div>
        </div>

        <div v-if="!devicesStore.state.root" class="text-xs text-muted-foreground">
          Connect, login, and open this tab to auto-load the tree.
        </div>
      </div>
    </section>

    <div
      v-if="nodeInfoOpen"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-6"
      @click.self="closeNodeInfo"
    >
      <div class="w-full max-w-2xl rounded-2xl border border-border/60 bg-card/95 p-6 shadow-xl">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
              Device
            </p>
            <h2 class="mt-1 text-lg font-semibold">Node {{ nodeInfoNodeId }}</h2>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <Button size="sm" variant="outline" :disabled="nodeInfoLoading" @click="refreshNodeInfo">
              Reload
            </Button>
            <Button size="sm" variant="outline" @click="closeNodeInfo">Close</Button>
          </div>
        </div>

        <div class="mt-4">
          <div v-if="nodeInfoLoading" class="text-sm text-muted-foreground">Loading…</div>
          <div v-else-if="nodeInfoError" class="text-sm text-rose-600">Error: {{ nodeInfoError }}</div>
          <div v-else class="space-y-3">
            <div v-if="!sortedNodeInfoItems.length" class="text-sm text-muted-foreground">
              No details returned.
            </div>
            <div v-else class="overflow-hidden rounded-xl border border-border/60">
              <div
                v-for="[key, value] in sortedNodeInfoItems"
                :key="key"
                class="grid grid-cols-1 gap-1 border-b border-border/50 bg-background/70 px-4 py-3 text-sm last:border-b-0 md:grid-cols-[220px_minmax(0,1fr)]"
              >
                <div class="font-mono text-[12px] text-muted-foreground">{{ key }}</div>
                <div class="break-words font-mono text-[12px] text-foreground">{{ value }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

  </section>
</template>

