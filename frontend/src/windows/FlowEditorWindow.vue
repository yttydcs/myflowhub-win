<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from "vue"
import { useRoute } from "vue-router"
import { LayoutGrid, Link2Off, Plus, Redo2, Save, Trash2, Undo2, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import { Tooltip } from "@/components/ui/tooltip"
import FlowCanvas from "@/components/flow/FlowCanvas.vue"
import { useFlowStore } from "@/stores/flow"
import { useFlowProjectsStore } from "@/stores/flowProjects"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"
import { HomeState as LoadHomeState } from "../../wailsjs/go/main/App"

const route = useRoute()
const flowStore = useFlowStore()
const projectsStore = useFlowProjectsStore()
const sessionStore = useSessionStore()
const toast = useToastStore()

const fallbackIdentity = reactive({ nodeId: 0, hubId: 0 })

const loading = ref(true)
const loadedProjectId = ref("")
const loadedProjectName = ref("")
const saveBusy = ref(false)
const addNodeOpen = ref(false)
const methodDialogOpen = ref(false)
const methodSearch = ref("")
const queryNodeIdDraft = ref("")
const nodeIdDraft = ref("")
const pendingCapabilityKey = ref("")
const lastCapabilityQueryNode = ref("")

const nodeDraft = reactive({
  id: ""
})

const selectedNode = computed(() => flowStore.state.nodes[flowStore.state.selectedNodeIndex] ?? null)
const selectedEdge = computed(() => flowStore.state.edges[flowStore.state.selectedEdgeIndex] ?? null)
const nodeDetailOpen = computed(() => Boolean(selectedNode.value))
const selfNodeId = computed(() => Number(sessionStore.auth.nodeId || fallbackIdentity.nodeId || 0))
const hubId = computed(() => Number(sessionStore.auth.hubId || fallbackIdentity.hubId || 0))

const canUndo = computed(() => flowStore.state.historyIndex > 0)
const canRedo = computed(
  () => flowStore.state.historyIndex >= 0 && flowStore.state.historyIndex < flowStore.state.historyLength - 1
)

const projectId = computed(() => String(route.query.projectId ?? "").trim())
const effectiveExecutorNode = computed(() => {
  const rawTarget = String(flowStore.state.targetId ?? "").trim()
  if (rawTarget) {
    const parsed = Number.parseInt(rawTarget, 10)
    if (Number.isFinite(parsed) && parsed > 0) {
      return Math.trunc(parsed)
    }
  }
  const hubId = Number(sessionStore.auth.hubId || flowStore.state.hubId || 0)
  return Number.isFinite(hubId) && hubId > 0 ? Math.trunc(hubId) : 0
})
const selectedTargetLabel = computed(() => {
  const node = selectedNode.value
  if (!node) return "No target selected."
  if (node.target > 0) {
    return `Remote provider node ${node.target}`
  }
  if (effectiveExecutorNode.value > 0) {
    return `Current executor node ${effectiveExecutorNode.value}`
  }
  return "Current executor"
})
const capabilityQueryNodeLabel = computed(() => {
  const raw = queryNodeIdDraft.value.trim()
  if (raw) return raw
  return effectiveExecutorNode.value > 0 ? String(effectiveExecutorNode.value) : "-"
})
const capabilityOptions = computed(() => {
  const executorNode = effectiveExecutorNode.value
  return [...flowStore.state.execCapabilities].sort((left, right) => {
    const leftScore = left.providerNode === executorNode ? 0 : 1
    const rightScore = right.providerNode === executorNode ? 0 : 1
    if (leftScore !== rightScore) {
      return leftScore - rightScore
    }
    if (left.method !== right.method) {
      return left.method.localeCompare(right.method)
    }
    if (left.providerNode !== right.providerNode) {
      return left.providerNode - right.providerNode
    }
    return left.viaNode - right.viaNode
  })
})
const filteredCapabilities = computed(() => {
  const query = methodSearch.value.trim().toLowerCase()
  if (!query) {
    return capabilityOptions.value
  }
  return capabilityOptions.value.filter((route) => {
    const haystack = `${route.method} ${route.providerNode} ${route.viaNode} ${route.version}`.toLowerCase()
    return haystack.includes(query)
  })
})
const selectedCapabilityKey = computed(() => {
  const node = selectedNode.value
  if (!node) return ""
  const method = node.method.trim()
  if (!method) return ""
  const expectedProvider = node.target > 0 ? Math.trunc(node.target) : effectiveExecutorNode.value
  if (!expectedProvider) return ""
  const matched = capabilityOptions.value.find(
    (route) => route.method === method && route.providerNode === expectedProvider
  )
  return matched?.key ?? ""
})

const closeNodeDetail = () => {
  flowStore.clearSelection()
}

const loadHomeDefaults = async () => {
  try {
    const state = await LoadHomeState()
    fallbackIdentity.nodeId = Number(state?.nodeId ?? 0)
    fallbackIdentity.hubId = Number(state?.hubId ?? 0)
  } catch (err) {
    console.warn(err)
  }
  flowStore.setIdentity(selfNodeId.value, hubId.value)
}

const syncQueryNodeDraft = () => {
  const node = selectedNode.value
  if (node && node.target > 0) {
    queryNodeIdDraft.value = String(Math.trunc(node.target))
    return
  }
  if (effectiveExecutorNode.value > 0) {
    queryNodeIdDraft.value = String(effectiveExecutorNode.value)
    return
  }
  queryNodeIdDraft.value = ""
}

const syncPendingCapability = () => {
  pendingCapabilityKey.value = selectedCapabilityKey.value
}

const closeMethodDialog = () => {
  methodDialogOpen.value = false
}

const refreshMethodCapabilities = async () => {
  try {
    const queryNodeId = queryNodeIdDraft.value.trim()
    await flowStore.queryExecCapabilities(undefined, queryNodeId)
    lastCapabilityQueryNode.value = queryNodeId || String(effectiveExecutorNode.value || "")
    if (!flowStore.state.execCapabilities.some((route) => route.key === pendingCapabilityKey.value)) {
      syncPendingCapability()
    }
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to load method capabilities.")
  }
}

const openMethodDialog = async () => {
  if (!selectedNode.value) return
  methodSearch.value = selectedNode.value.method.trim()
  syncPendingCapability()
  syncQueryNodeDraft()
  methodDialogOpen.value = true
  if (
    !flowStore.state.execCapabilities.length ||
    lastCapabilityQueryNode.value !== (queryNodeIdDraft.value.trim() || String(effectiveExecutorNode.value || ""))
  ) {
    await refreshMethodCapabilities()
  }
}

const selectCapability = (key: string) => {
  pendingCapabilityKey.value = key
}

const applyCapabilitySelection = () => {
  if (!pendingCapabilityKey.value) {
    toast.warn("Please select a method.")
    return
  }
  try {
    flowStore.applyCallCapability(pendingCapabilityKey.value)
    methodDialogOpen.value = false
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to apply method capability.")
  }
}

const commitNodeId = () => {
  const node = selectedNode.value
  if (!node) return
  const oldId = node.id
  try {
    flowStore.renameNodeId(oldId, nodeIdDraft.value)
    nodeIdDraft.value = node.id
  } catch (err) {
    console.warn(err)
    nodeIdDraft.value = oldId
    toast.errorOf(err, "Failed to rename node.")
  }
}

const openAddNodeDialog = () => {
  nodeDraft.id = flowStore.suggestNodeId()
  addNodeOpen.value = true
}

const addNode = () => {
  try {
    flowStore.addNode(nodeDraft.id)
    addNodeOpen.value = false
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to add node.")
  }
}

const removeNode = () => {
  flowStore.removeSelectedNode()
}

const removeEdge = () => {
  flowStore.removeSelectedEdge()
}

const onCanvasConnect = (from: string, to: string) => {
  try {
    flowStore.addEdge(from, to)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to connect nodes.")
  }
}

const onCanvasSelectNode = (nodeId: string) => {
  flowStore.selectNodeById(nodeId)
}

const onCanvasSelectEdge = (from: string, to: string) => {
  flowStore.selectEdgeByEndpoints(from, to)
}

const onCanvasNodeMoved = (nodeId: string, x: number, y: number) => {
  flowStore.setNodePosition(nodeId, x, y)
  flowStore.commitHistory()
}

const onCanvasClear = () => {
  flowStore.clearSelection()
}

const autoLayout = () => {
  try {
    flowStore.autoLayoutTB()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to auto layout.")
  }
}

const saveProject = async () => {
  const id = projectId.value
  if (!id) {
    toast.error("projectId is required in query string.")
    return
  }
  saveBusy.value = true
  try {
    const graph = flowStore.exportGraphDraft()
    const saved = await projectsStore.saveProjectGraph(id, graph)
    loadedProjectName.value = saved.name || saved.flowId || id
    toast.success("Project saved.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to save project.")
  } finally {
    saveBusy.value = false
  }
}

const loadProject = async () => {
  loading.value = true
  try {
    await projectsStore.loadProjects()
    const id = projectId.value
    if (!id) {
      throw new Error("projectId is required in query string.")
    }
    const project = projectsStore.getProjectByID(id)
    if (!project) {
      throw new Error(`Project not found: ${id}`)
    }
    loadedProjectId.value = project.projectId
    loadedProjectName.value = project.name || project.flowId
    flowStore.loadGraphDraft(project.graph)
    nodeIdDraft.value = selectedNode.value?.id ?? ""
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to load project.")
  } finally {
    loading.value = false
  }
}

const isEditableTarget = (target: EventTarget | null) => {
  const el = target as HTMLElement | null
  if (!el) return false
  if (el.isContentEditable) return true
  const tag = el.tagName?.toLowerCase()
  return tag === "input" || tag === "textarea" || tag === "select"
}

const onKeyDown = (event: KeyboardEvent) => {
  if (addNodeOpen.value || methodDialogOpen.value) return

  const key = event.key || ""
  const lower = key.toLowerCase()
  const ctrl = event.ctrlKey || event.metaKey

  if (ctrl && lower === "s") {
    event.preventDefault()
    void saveProject()
    return
  }

  const editable = isEditableTarget(event.target)
  if (editable) return

  if (key === "Delete") {
    event.preventDefault()
    if (flowStore.state.selectedEdgeIndex >= 0) {
      flowStore.removeSelectedEdge()
    } else if (flowStore.state.selectedNodeIndex >= 0) {
      flowStore.removeSelectedNode()
    }
    return
  }

  if (ctrl && lower === "z" && !event.shiftKey) {
    event.preventDefault()
    flowStore.undo()
    return
  }

  if (ctrl && (lower === "y" || (lower === "z" && event.shiftKey))) {
    event.preventDefault()
    flowStore.redo()
  }
}

watch(
  () => [selfNodeId.value, hubId.value],
  ([nodeId, hubId]) => {
    flowStore.setIdentity(Number(nodeId), Number(hubId))
    if (!methodDialogOpen.value) {
      syncQueryNodeDraft()
    }
  },
  { immediate: true }
)

watch(
  () => selectedNode.value?.id ?? "",
  () => {
    nodeIdDraft.value = selectedNode.value?.id ?? ""
    if (!methodDialogOpen.value) {
      methodSearch.value = selectedNode.value?.method?.trim() ?? ""
      syncPendingCapability()
      syncQueryNodeDraft()
    }
  },
  { immediate: true }
)

watch(
  () => nodeDetailOpen.value,
  (open) => {
    if (!open) {
      methodDialogOpen.value = false
    }
  }
)

watch(
  () => flowStore.state.message,
  (msg) => {
    const trimmed = msg.trim()
    if (!trimmed) return
    const lower = trimmed.toLowerCase()
    if (lower.includes("failed") || lower.includes("error") || lower.includes("invalid")) {
      toast.error(trimmed)
    } else if (lower.includes("loaded") || lower.includes("saved") || lower.includes("updated")) {
      toast.success(trimmed)
    } else {
      toast.info(trimmed)
    }
    flowStore.state.message = ""
  }
)

onMounted(() => {
  void loadHomeDefaults().then(() => loadProject())
  window.addEventListener("keydown", onKeyDown)
})

onUnmounted(() => {
  window.removeEventListener("keydown", onKeyDown)
})
</script>

<template>
  <section class="flex h-full min-h-0 flex-col bg-card/70 text-card-foreground">
    <header class="flex-none border-b border-border/60 bg-card/92 px-5 py-4 shadow-sm">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
            Flow Project Editor
          </p>
          <h1 class="mt-1 text-xl font-semibold">{{ loadedProjectName || "Untitled Project" }}</h1>
          <p class="mt-1 text-xs text-muted-foreground">project_id {{ loadedProjectId || projectId || "-" }}</p>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <Tooltip content="Add Node" side="bottom">
            <Button size="icon" variant="outline" @click="openAddNodeDialog">
              <Plus class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">Add Node</span>
            </Button>
          </Tooltip>
          <Tooltip content="Remove Node (Delete)" side="bottom">
            <Button size="icon" variant="outline" :disabled="flowStore.state.selectedNodeIndex < 0" @click="removeNode">
              <Trash2 class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">Remove Node</span>
            </Button>
          </Tooltip>
          <Tooltip content="Remove Edge (Delete)" side="bottom">
            <Button size="icon" variant="outline" :disabled="flowStore.state.selectedEdgeIndex < 0" @click="removeEdge">
              <Link2Off class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">Remove Edge</span>
            </Button>
          </Tooltip>
          <Tooltip content="Undo (Ctrl+Z)" side="bottom">
            <Button size="icon" variant="outline" :disabled="!canUndo" @click="flowStore.undo()">
              <Undo2 class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">Undo</span>
            </Button>
          </Tooltip>
          <Tooltip content="Redo (Ctrl+Y)" side="bottom">
            <Button size="icon" variant="outline" :disabled="!canRedo" @click="flowStore.redo()">
              <Redo2 class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">Redo</span>
            </Button>
          </Tooltip>
          <Tooltip content="Auto Layout" side="bottom">
            <Button size="icon" variant="outline" @click="autoLayout">
              <LayoutGrid class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">Auto Layout</span>
            </Button>
          </Tooltip>
          <Tooltip content="Save Project (Ctrl+S)" side="bottom">
            <Button size="icon" :disabled="saveBusy || loading" @click="saveProject">
              <Save class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">Save Project</span>
            </Button>
          </Tooltip>
        </div>
      </div>

      <p class="mt-3 text-xs text-muted-foreground">
        Pure workflow editing only. Trigger and deployment settings stay in the project center.
      </p>
    </header>

    <div v-if="loading" class="flex flex-1 items-center justify-center px-6 text-sm text-muted-foreground">
      Loading project...
    </div>

    <div v-else class="relative flex-1 min-h-0 overflow-hidden">
      <div class="flex h-full min-h-0 flex-col p-4 transition-[padding] duration-200" :class="nodeDetailOpen ? 'pr-[440px]' : ''">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <div class="text-xs text-muted-foreground">
            Drag nodes to reposition. Drag from node handles to connect. Click a node to open the right-side detail drawer.
          </div>
          <div class="text-xs text-muted-foreground">
            Click blank canvas to close the drawer.
          </div>
        </div>

        <div class="flex-1 min-h-0">
          <FlowCanvas
            :nodes="flowStore.state.nodes"
            :edges="flowStore.state.edges"
            :selected-node-id="selectedNode?.id ?? null"
            :selected-edge="selectedEdge"
            :status-nodes="[]"
            @connect="onCanvasConnect"
            @select-node="onCanvasSelectNode"
            @select-edge="onCanvasSelectEdge"
            @node-moved="onCanvasNodeMoved"
            @clear-selection="onCanvasClear"
          />
        </div>
      </div>

      <div v-if="nodeDetailOpen" class="pointer-events-none absolute inset-y-0 right-0 z-20 flex w-full justify-end p-0">
        <aside
          v-if="selectedNode"
          class="pointer-events-auto h-full w-full max-w-[420px] border-l border-border/70 bg-card shadow-2xl"
        >
          <div class="flex h-full flex-col">
            <div class="flex items-start justify-between gap-4 border-b border-border/60 px-5 py-4">
              <div>
                <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">Node Detail</p>
                <h2 class="mt-1 text-lg font-semibold">{{ selectedNode.id }}</h2>
              </div>
              <Button size="icon" variant="ghost" @click="closeNodeDetail">
                <X class="h-4 w-4" aria-hidden="true" />
                <span class="sr-only">Close</span>
              </Button>
            </div>

            <div class="flex-1 overflow-y-auto px-5 py-4">
              <div class="space-y-4">
                <div>
                  <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    Node ID
                  </label>
                  <input
                    v-model="nodeIdDraft"
                    class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                    @blur="commitNodeId"
                    @keydown.enter.prevent="commitNodeId"
                  />
                  <p class="mt-1 text-[11px] text-muted-foreground">
                    Node ID must be unique. Renaming updates all connected edges.
                  </p>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <div>
                    <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                      Kind
                    </label>
                    <input
                      value="call"
                      disabled
                      class="mt-2 h-10 w-full rounded-md border border-input bg-muted/40 px-3 text-sm text-muted-foreground"
                    />
                  </div>

                  <div>
                    <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                      Allow Fail
                    </label>
                    <div class="mt-2 flex h-10 items-center gap-2 rounded-md border border-input bg-background px-3 text-sm">
                      <input
                        v-model="selectedNode.allowFail"
                        type="checkbox"
                        class="h-4 w-4 rounded border"
                        @change="flowStore.commitHistory()"
                      />
                      <span class="text-muted-foreground">Continue on error</span>
                    </div>
                  </div>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <div>
                    <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                      Retry
                    </label>
                    <input
                      v-model.number="selectedNode.retry"
                      type="number"
                      min="0"
                      class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                      @blur="flowStore.commitHistory()"
                    />
                  </div>

                  <div>
                    <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                      Timeout (ms)
                    </label>
                    <input
                      v-model.number="selectedNode.timeoutMs"
                      type="number"
                      min="0"
                      class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                      @blur="flowStore.commitHistory()"
                    />
                  </div>
                </div>

                <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                  <div class="flex flex-wrap items-start justify-between gap-3">
                    <div class="min-w-0">
                      <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                        Call Method
                      </p>
                      <p class="mt-2 break-all text-sm font-semibold">
                        {{ selectedNode.method || "No method selected." }}
                      </p>
                      <p class="mt-1 text-xs text-muted-foreground">
                        {{ selectedTargetLabel }}
                      </p>
                    </div>
                    <Button variant="outline" size="sm" @click="openMethodDialog">Select Method</Button>
                  </div>
                  <p class="mt-3 text-[11px] text-muted-foreground">
                    Use the method dialog to choose a registered capability. The editor will keep method and target aligned.
                  </p>
                </div>

                <div>
                  <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    Args (JSON)
                  </label>
                  <textarea
                    v-model="selectedNode.args"
                    rows="10"
                    class="mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-xs"
                    @blur="flowStore.commitHistory()"
                  />
                </div>
              </div>
            </div>
          </div>
        </aside>
      </div>
    </div>

    <Overlay
      :open="methodDialogOpen"
      overlayClass="bg-slate-950/60 p-4"
      zIndexClass="z-40"
      closeOnBackdrop
      @close="closeMethodDialog"
    >
      <div class="flex max-h-[80vh] w-full max-w-4xl flex-col rounded-2xl border bg-card p-6 text-card-foreground shadow-2xl">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">Call Method</p>
            <h2 class="mt-1 text-lg font-semibold">Select Capability</h2>
            <p class="mt-1 text-sm text-muted-foreground">
              Pick a registered capability and the editor will keep method and target aligned.
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            <span class="rounded-full border border-border/60 px-3 py-1">
              Executor {{ effectiveExecutorNode || "-" }}
            </span>
            <span class="rounded-full border border-border/60 px-3 py-1">
              Query Node {{ capabilityQueryNodeLabel }}
            </span>
            <span class="rounded-full border border-border/60 px-3 py-1">
              {{ selectedTargetLabel }}
            </span>
          </div>
        </div>

        <div class="mt-5 flex flex-wrap items-end gap-3">
          <div class="min-w-[200px] max-w-[240px] flex-1">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Query Node ID
            </label>
            <input
              v-model="queryNodeIdDraft"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              inputmode="numeric"
              placeholder="Current executor"
            />
            <p class="mt-1 text-[11px] text-muted-foreground">
              Used only for capability lookup. It will not be written back into the call node.
            </p>
          </div>
          <div class="min-w-[240px] flex-[2]">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Filter
            </label>
            <input
              v-model="methodSearch"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="Search method / provider / version"
            />
          </div>
          <Button variant="outline" :disabled="flowStore.state.execCapabilitiesLoading" @click="refreshMethodCapabilities">
            {{ flowStore.state.execCapabilitiesLoading ? "Refreshing..." : "Refresh Capabilities" }}
          </Button>
        </div>

        <div class="mt-5 flex-1 overflow-y-auto rounded-xl border border-border/70 bg-background/80">
          <div
            v-if="flowStore.state.execCapabilitiesLoading && !flowStore.state.execCapabilities.length"
            class="px-4 py-10 text-center text-sm text-muted-foreground"
          >
            Loading capability list...
          </div>
          <div v-else-if="!filteredCapabilities.length" class="px-4 py-10 text-center text-sm text-muted-foreground">
            {{
              flowStore.state.execCapabilities.length
                ? "No capability matched the current filter."
                : "No capability loaded yet. Refresh to query the selected node."
            }}
          </div>
          <div v-else class="divide-y divide-border/60">
            <button
              v-for="route in filteredCapabilities"
              :key="route.key"
              type="button"
              class="flex w-full items-start justify-between gap-4 px-4 py-4 text-left transition hover:bg-muted/50"
              :class="pendingCapabilityKey === route.key ? 'bg-muted/70' : ''"
              @click="selectCapability(route.key)"
            >
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <p class="break-all text-sm font-semibold">{{ route.method }}</p>
                  <span class="rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-[0.2em] text-muted-foreground">
                    {{ route.providerNode === effectiveExecutorNode ? "Self" : "Remote" }}
                  </span>
                </div>
                <p class="mt-1 text-xs text-muted-foreground">
                  Provider {{ route.providerNode }}
                  <span v-if="route.viaNode > 0"> via {{ route.viaNode }}</span>
                  <span v-if="route.version"> · {{ route.version }}</span>
                </p>
              </div>
              <span
                class="shrink-0 rounded-full border px-3 py-1 text-xs"
                :class="
                  pendingCapabilityKey === route.key
                    ? 'border-primary text-primary'
                    : 'border-border/70 text-muted-foreground'
                "
              >
                {{ pendingCapabilityKey === route.key ? "Selected" : "Choose" }}
              </span>
            </button>
          </div>
        </div>

        <div class="mt-5 flex flex-wrap items-center justify-between gap-3">
          <p class="text-xs text-muted-foreground">
            Applying a capability updates the method and hidden call target. The query node stays temporary.
          </p>
          <div class="flex gap-2">
            <Button variant="outline" @click="closeMethodDialog">Cancel</Button>
            <Button :disabled="!pendingCapabilityKey" @click="applyCapabilitySelection">Apply Method</Button>
          </div>
        </div>
      </div>
    </Overlay>

    <Overlay :open="addNodeOpen" @close="addNodeOpen = false">
      <div class="w-full max-w-md rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">Add Node</h2>
        <div class="mt-4 space-y-3">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Node ID
            </label>
            <input
              v-model="nodeDraft.id"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="addNodeOpen = false">Cancel</Button>
          <Button @click="addNode">Add</Button>
        </div>
      </div>
    </Overlay>
  </section>
</template>
