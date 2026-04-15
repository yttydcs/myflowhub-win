<script setup lang="ts">
// 本文件实现 Win 前端的 `Devices` 页面。
import { computed, onMounted, reactive, ref, watch } from "vue"
import CardHeader from "@/components/CardHeader.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import NodeVarsDialog from "@/components/varpool/NodeVarsDialog.vue"
import { useI18n } from "@/i18n"
import type { DeviceTreeNode, DevicesMode } from "@/stores/devices"
import { useDevicesStore } from "@/stores/devices"
import { useManagementStore } from "@/stores/management"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"

const devicesStore = useDevicesStore()
const mgmtStore = useManagementStore()
const sessionStore = useSessionStore()
const toast = useToastStore()
const { t } = useI18n()

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
  if (!nodeId && !hubId) return t("Not logged in")
  return t("node={nodeId} hub={hubId}", {
    nodeId: nodeId || "-",
    hubId: hubId || "-"
  })
})

const ready = computed(() => {
  return Boolean(sessionStore.connected && sessionStore.auth.nodeId && sessionStore.auth.hubId)
})

const modeLabel = computed(() =>
  devicesStore.state.mode === "subtree"
    ? t("Subtree (direct + self; not recursive)")
    : t("Direct")
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

const varsDialogOpen = ref(false)
const varsDialogOwnerId = ref(0)

const openVarsDialog = (node: DeviceTreeNode) => {
  varsDialogOwnerId.value = node.nodeId
  varsDialogOpen.value = true
}

const closeVarsDialog = () => {
  varsDialogOpen.value = false
}

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
    throw new Error(t("Management binding '{method}' unavailable", { method }))
  }
  return fn(...args)
}

const normalizeDisplayName = (value: unknown) => String(value ?? "").trim()

const resolveNodeDisplayName = (nodeId: number, fallback = "") => {
  return normalizeDisplayName(fallback) || devicesStore.getDisplayName(nodeId)
}

const formatNodeTitle = (nodeId: number, fallback = "") => {
  const displayName = resolveNodeDisplayName(nodeId, fallback)
  if (displayName) return displayName
  return t("Node {nodeId}", { nodeId: nodeId || "-" })
}

const loadNodeInfo = async (targetID: number) => {
  if (!sessionStore.connected) {
    throw new Error(t("Connect before querying node info."))
  }
  const sourceID = Number(sessionStore.auth.nodeId || 0)
  if (!sourceID) {
    throw new Error(t("Login required to query node info."))
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
    nodeInfoError.value = message || t("Unknown error.")
    toast.errorOf(err, t("Failed to load node info."))
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

const nodeInfoDisplayName = computed(() => {
  return resolveNodeDisplayName(
    nodeInfoNodeId.value,
    nodeInfoItems.value.display_name ?? nodeInfoItems.value.displayName ?? ""
  )
})

const nodeInfoTitle = computed(() => formatNodeTitle(nodeInfoNodeId.value, nodeInfoDisplayName.value))

const showNodeInfoNodeId = computed(() => Boolean(nodeInfoDisplayName.value) && nodeInfoNodeId.value > 0)

const configOpen = ref(false)
const editOpen = ref(false)
const configDraft = reactive({ key: "", value: "" })

const openConfig = async (node: DeviceTreeNode) => {
  configOpen.value = true
  editOpen.value = false
  configDraft.key = ""
  configDraft.value = ""

  try {
    await mgmtStore.selectNode(node.nodeId)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to load config."))
  }
}

const closeConfig = () => {
  configOpen.value = false
  editOpen.value = false
  configDraft.key = ""
  configDraft.value = ""
  void mgmtStore.selectNode(0)
}

const refreshConfig = async () => {
  try {
    await mgmtStore.refreshConfig()
    toast.success(t("Config refreshed."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to refresh config."))
  }
}

const openEdit = (key: string, value: string) => {
  configDraft.key = key
  configDraft.value = value
  editOpen.value = true
}

const configDisplayName = computed(() => {
  const localValue = mgmtStore.state.configEntries.find((entry) => entry.key === "node.display_name")?.value
  return normalizeDisplayName(localValue) || resolveNodeDisplayName(mgmtStore.state.selectedNodeId)
})

const configTitle = computed(() => {
  return formatNodeTitle(mgmtStore.state.selectedNodeId, configDisplayName.value)
})

const showConfigNodeId = computed(() => {
  return Boolean(configDisplayName.value) && mgmtStore.state.selectedNodeId > 0
})

const saveConfig = async () => {
  try {
    await mgmtStore.setConfig(configDraft.key, configDraft.value)
    editOpen.value = false
    toast.success(t("Config updated."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to update config."))
  }
}

const loadRoot = async () => {
  try {
    await devicesStore.loadRoot()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to load root."))
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
    toast.errorOf(err, t("Failed to expand node."))
  }
}

const retryNode = async (node: DeviceTreeNode) => {
  try {
    await devicesStore.retry(node.key)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to retry node."))
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

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  ([nodeId, hubId]) => {
    mgmtStore.setIdentity(Number(nodeId), Number(hubId))
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
    <section class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-sm font-semibold">{{ t("Nodes") }}</h2>
          <p class="text-xs text-muted-foreground">
            {{ t("Mode") }}: <span class="font-semibold text-foreground">{{ modeLabel }}</span>
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
            <span class="font-semibold uppercase tracking-[0.2em]">{{ t("Identity") }}</span>
            <span class="font-mono text-[11px] text-foreground">{{ identityLabel }}</span>
          </div>
          <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
            <span class="font-semibold uppercase tracking-[0.2em]">{{ t("Mode") }}</span>
            <select
              v-model="devicesStore.state.mode"
              class="h-7 rounded-md border border-input bg-background px-2 text-xs text-foreground"
              @change="onModeChanged(devicesStore.state.mode)"
            >
              <option value="direct">{{ t("Direct") }}</option>
              <option value="subtree">{{ t("Subtree (direct + self)") }}</option>
            </select>
          </div>
          <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
            <span class="font-semibold uppercase tracking-[0.2em]">{{ t("Root") }}</span>
            <input
              v-model="devicesStore.state.rootTargetId"
              class="h-7 w-28 rounded-md border border-input bg-background px-2 text-xs text-foreground"
              :placeholder="t('Node ID')"
              @keydown.enter.prevent="onRootEnter"
            />
          </div>
          <Button variant="outline" size="sm" @click="loadRoot">{{ t("Reload") }}</Button>
        </div>
      </div>

      <div class="mt-4 space-y-2">
        <div
          v-for="{ node, depth } in visibleNodes"
          :key="node.key"
          class="cursor-pointer rounded-xl border border-border/60 bg-background/70 px-3 py-2 text-sm transition hover:border-border/80 hover:bg-muted/60 hover:shadow-sm"
          @click="openNodeInfo(node)"
        >
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="flex min-w-0 items-center gap-2">
              <button
                type="button"
                class="h-7 w-7 rounded-md border border-border/70 bg-background text-xs text-foreground transition-colors hover:bg-muted/60 disabled:opacity-50"
                :style="{ marginLeft: `${depth * 16}px` }"
                :disabled="node.duplicate || node.loading"
                @click.stop="toggleNode(node)"
              >
                <span v-if="node.loading">…</span>
                <span v-else>{{ node.expanded ? "-" : "+" }}</span>
              </button>

              <div class="min-w-0">
                <p class="truncate font-semibold">
                  {{ formatNodeTitle(node.nodeId, node.displayName) }}
                  <span v-if="depth === 0" class="text-xs font-normal text-muted-foreground">
                    {{ t("(root)") }}
                  </span>
                </p>
                <p class="truncate text-xs text-muted-foreground">
                  <span v-if="node.displayName" class="font-mono text-[11px]">
                    {{ t("Node {nodeId}", { nodeId: node.nodeId }) }}
                  </span>
                  <span v-if="node.displayName" aria-hidden="true"> · </span>
                  <span v-if="node.duplicate">{{ t("Duplicate: expansion disabled.") }}</span>
                  <span v-else-if="node.error">{{ t("Error: {error}", { error: node.error }) }}</span>
                  <span v-else-if="node.children && node.children.length === 0">{{ t("No children.") }}</span>
                  <span v-else-if="node.children && node.children.length > 0">
                    {{ t("Children: {count}", { count: node.children.length }) }}
                  </span>
                  <span v-else>{{ t("Not loaded.") }}</span>
                </p>
              </div>
            </div>

            <div class="flex flex-wrap items-center gap-2">
              <Badge v-if="node.duplicate" variant="secondary">{{ t("Duplicate") }}</Badge>
              <Badge
                v-else-if="node.children ? node.children.length > 0 : node.hasChildrenHint"
                variant="secondary"
              >
                {{ t("Has children") }}
              </Badge>
              <Badge v-else-if="node.children && node.children.length === 0" variant="secondary">
                {{ t("Leaf") }}
              </Badge>
              <Badge v-else variant="secondary">{{ t("Unknown") }}</Badge>

              <Button size="sm" variant="outline" :disabled="!ready" @click.stop="openVarsDialog(node)">
                {{ t("Vars") }}
              </Button>

              <Button size="sm" variant="outline" :disabled="!ready" @click.stop="openConfig(node)">
                {{ t("Edit") }}
              </Button>

              <Button
                v-if="node.error && !node.duplicate"
                size="sm"
                variant="outline"
                :disabled="node.loading"
                @click.stop="retryNode(node)"
              >
                {{ t("Retry") }}
              </Button>
            </div>
          </div>
        </div>

        <div v-if="!devicesStore.state.root" class="text-xs text-muted-foreground">
          {{ t("Connect, login, and open this tab to auto-load the tree.") }}
        </div>
      </div>
    </section>

    <Overlay :open="nodeInfoOpen" closeOnBackdrop @close="closeNodeInfo">
      <div class="flex max-h-[85vh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl border border-border/60 bg-card/95 p-6 text-card-foreground shadow-xl">
        <CardHeader class="items-start" :title="nodeInfoTitle" title-class="text-lg">
          <template #actions>
            <div class="flex flex-wrap items-center gap-2">
              <Button size="sm" variant="outline" :disabled="nodeInfoLoading" @click="refreshNodeInfo">
                {{ t("Reload") }}
              </Button>
              <Button size="sm" variant="outline" @click="closeNodeInfo">{{ t("Close") }}</Button>
            </div>
          </template>
        </CardHeader>

        <div class="mt-5 min-h-0 flex-1 overflow-y-auto">
          <div class="space-y-3 px-1 py-1 pr-2">
            <div v-if="showNodeInfoNodeId" class="font-mono text-xs text-muted-foreground">
              {{ t("Node {nodeId}", { nodeId: nodeInfoNodeId }) }}
            </div>
            <div v-if="nodeInfoLoading" class="text-sm text-muted-foreground">{{ t("Loading…") }}</div>
            <div v-else-if="nodeInfoError" class="text-sm text-rose-600">
              {{ t("Error: {error}", { error: nodeInfoError }) }}
            </div>
            <div v-else class="space-y-3">
              <div v-if="!sortedNodeInfoItems.length" class="text-sm text-muted-foreground">
                {{ t("No details returned.") }}
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
    </Overlay>

    <NodeVarsDialog :open="varsDialogOpen" :ownerId="varsDialogOwnerId" @close="closeVarsDialog" />

    <Overlay :open="configOpen" closeOnBackdrop @close="closeConfig">
      <div class="flex max-h-[85vh] w-full max-w-3xl flex-col overflow-hidden rounded-2xl border border-border/60 bg-card/95 p-6 text-card-foreground shadow-xl">
        <CardHeader class="items-start" :title="configTitle" title-class="text-lg">
          <template #actions>
            <div class="flex flex-wrap items-center gap-2">
              <Button
                size="sm"
                variant="outline"
                :disabled="!mgmtStore.state.selectedNodeId"
                @click="refreshConfig"
              >
                {{ t("Refresh") }}
              </Button>
              <Button size="sm" variant="outline" @click="closeConfig">{{ t("Close") }}</Button>
            </div>
          </template>
        </CardHeader>

        <div class="mt-5 min-h-0 flex-1 overflow-y-auto">
          <div class="space-y-2 px-1 py-1 pr-2">
            <div v-if="showConfigNodeId" class="font-mono text-xs text-muted-foreground">
              {{ t("Node {nodeId}", { nodeId: mgmtStore.state.selectedNodeId }) }}
            </div>
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
                {{ t("Edit") }}
              </Button>
            </div>
            <div v-if="!mgmtStore.state.configEntries.length" class="text-xs text-muted-foreground">
              {{
                mgmtStore.state.selectedNodeId
                  ? t("Loading config entries…")
                  : t("Select a node to load config entries.")
              }}
            </div>
          </div>
        </div>
      </div>
    </Overlay>

    <Overlay :open="editOpen" @close="editOpen = false">
      <div class="flex max-h-[85vh] w-full max-w-lg flex-col overflow-hidden rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl">
        <h2 class="text-lg font-semibold">{{ t("Edit Config") }}</h2>
        <div class="mt-5 min-h-0 flex-1 overflow-y-auto">
          <div class="space-y-3 px-1 py-1 pr-2">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Key") }}
              </label>
              <input
                v-model="configDraft.key"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                disabled
              />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Value") }}
              </label>
              <textarea
                v-model="configDraft.value"
                rows="4"
                class="mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              />
            </div>
          </div>
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="editOpen = false">{{ t("Cancel") }}</Button>
          <Button @click="saveConfig">{{ t("Save") }}</Button>
        </div>
      </div>
    </Overlay>

  </section>
</template>

