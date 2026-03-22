<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from "vue"
import { useRoute } from "vue-router"
import { LayoutGrid, Link2Off, Plus, Redo2, Save, Trash2, Undo2, X } from "lucide-vue-next"
import CardHeader from "@/components/CardHeader.vue"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import { Tooltip } from "@/components/ui/tooltip"
import FlowCanvas from "@/components/flow/FlowCanvas.vue"
import { useI18n } from "@/i18n"
import { useFlowStore, type FlowInputBindingDraft, type FlowNodeKind } from "@/stores/flow"
import { useFlowProjectsStore } from "@/stores/flowProjects"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"
import { HomeState as LoadHomeState } from "../../wailsjs/go/main/App"

const route = useRoute()
const flowStore = useFlowStore()
const projectsStore = useFlowProjectsStore()
const sessionStore = useSessionStore()
const toast = useToastStore()
const { t } = useI18n()

const fallbackIdentity = reactive({ nodeId: 0, hubId: 0 })

const loading = ref(true)
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
  id: "",
  kind: "call" as FlowNodeKind
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
  if (!node) return t("No target selected.")
  if (node.kind !== "call") {
    return t("Compose nodes build local JSON output and do not call a capability.")
  }
  if (node.target > 0) {
    return t("Remote provider node {nodeId}", { nodeId: node.target })
  }
  if (effectiveExecutorNode.value > 0) {
    return t("Current executor node {nodeId}", { nodeId: effectiveExecutorNode.value })
  }
  return t("Current executor")
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
  if (!node || node.kind !== "call") return ""
  const method = node.method.trim()
  if (!method) return ""
  const expectedProvider = node.target > 0 ? Math.trunc(node.target) : effectiveExecutorNode.value
  if (!expectedProvider) return ""
  const matched = capabilityOptions.value.find(
    (route) => route.method === method && route.providerNode === expectedProvider
  )
  return matched?.key ?? ""
})

const selectedNodeValidation = computed(() =>
  selectedNode.value ? flowStore.getNodeValidation(selectedNode.value.id) : []
)

const ancestorNodeOptions = computed(() =>
  selectedNode.value ? flowStore.listAncestorNodeIds(selectedNode.value.id) : []
)

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
    toast.errorOf(err, t("Failed to load method capabilities."))
  }
}

const openMethodDialog = async () => {
  if (!selectedNode.value || selectedNode.value.kind !== "call") return
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
    toast.warn(t("Please select a method."))
    return
  }
  try {
    flowStore.applyCallCapability(pendingCapabilityKey.value)
    methodDialogOpen.value = false
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to apply method capability."))
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
    toast.errorOf(err, t("Failed to rename node."))
  }
}

const openAddNodeDialog = () => {
  nodeDraft.id = flowStore.suggestNodeId()
  nodeDraft.kind = "call"
  addNodeOpen.value = true
}

const addNode = () => {
  try {
    flowStore.addNode(nodeDraft.id, nodeDraft.kind)
    addNodeOpen.value = false
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to add node."))
  }
}

const setSelectedNodeKind = (kind: FlowNodeKind) => {
  const node = selectedNode.value
  if (!node) return
  try {
    flowStore.setNodeKind(node.id, kind)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to switch node kind."))
  }
}

const onSelectedNodeKindChange = (event: Event) => {
  const value = String((event.target as HTMLSelectElement | null)?.value ?? "call")
  setSelectedNodeKind(value === "compose" ? "compose" : "call")
}

const setSelectedNodeSpecMode = (mode: "form" | "json") => {
  const node = selectedNode.value
  if (!node) return
  try {
    flowStore.setNodeSpecEditorMode(node.id, mode)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, mode === "form" ? t("Failed to switch back to form mode.") : t("Failed to open advanced JSON mode."))
  }
}

const addBinding = () => {
  const node = selectedNode.value
  if (!node) return
  node.inputs.push(flowStore.createInputBinding())
  flowStore.commitHistory()
}

const removeBinding = (index: number) => {
  const node = selectedNode.value
  if (!node) return
  node.inputs.splice(index, 1)
  flowStore.commitHistory()
}

const onBindingSourceKindChange = (binding: FlowInputBindingDraft, event: Event) => {
  const sourceKind = String((event.target as HTMLSelectElement | null)?.value ?? "node_result")
  binding.sourceKind = sourceKind === "trigger" || sourceKind === "flow_meta" || sourceKind === "run_meta"
    ? sourceKind
    : "node_result"

  if (binding.sourceKind === "flow_meta") {
    binding.field = "flow_id"
    binding.nodeId = ""
    binding.path = ""
  } else if (binding.sourceKind === "run_meta") {
    binding.field = "run_id"
    binding.nodeId = ""
    binding.path = ""
  } else if (binding.sourceKind === "trigger") {
    binding.field = ""
    binding.nodeId = ""
  } else {
    binding.field = ""
  }

  flowStore.commitHistory()
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
    toast.errorOf(err, t("Failed to connect nodes."))
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
    toast.errorOf(err, t("Failed to auto layout."))
  }
}

const saveProject = async () => {
  const id = projectId.value
  if (!id) {
    toast.error(t("Project ID is required in the query string."))
    return
  }
  saveBusy.value = true
  try {
    const graph = flowStore.exportGraphDraft()
    const saved = await projectsStore.saveProjectGraph(id, graph)
    loadedProjectName.value = saved.name || saved.flowId || id
    toast.success(t("Project saved."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to save project."))
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
      throw new Error(t("Project ID is required in the query string."))
    }
    const project = projectsStore.getProjectByID(id)
    if (!project) {
      throw new Error(t("Project not found."))
    }
    loadedProjectName.value = project.name || project.flowId
    flowStore.loadGraphDraft(project.graph)
    nodeIdDraft.value = selectedNode.value?.id ?? ""
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to load project."))
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
  () => selectedNode.value?.kind ?? "",
  (kind) => {
    if (kind !== "call") {
      methodDialogOpen.value = false
    }
  }
)

watch(
  () => flowStore.state.message,
  (msg) => {
    const trimmed = msg.trim()
    if (!trimmed) return
    switch (flowStore.state.messageLevel) {
      case "error":
        toast.error(trimmed)
        break
      case "success":
        toast.success(trimmed)
        break
      default:
        toast.info(trimmed)
        break
    }
    flowStore.clearMessage()
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
          <h1 class="text-xl font-semibold">{{ loadedProjectName || t("Untitled Project") }}</h1>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <Tooltip :content="t('Add Node')" side="bottom">
            <Button size="icon" variant="outline" @click="openAddNodeDialog">
              <Plus class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">{{ t("Add Node") }}</span>
            </Button>
          </Tooltip>
          <Tooltip :content="t('Remove Node (Delete)')" side="bottom">
            <Button size="icon" variant="outline" :disabled="flowStore.state.selectedNodeIndex < 0" @click="removeNode">
              <Trash2 class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">{{ t("Remove Node (Delete)") }}</span>
            </Button>
          </Tooltip>
          <Tooltip :content="t('Remove Edge (Delete)')" side="bottom">
            <Button size="icon" variant="outline" :disabled="flowStore.state.selectedEdgeIndex < 0" @click="removeEdge">
              <Link2Off class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">{{ t("Remove Edge (Delete)") }}</span>
            </Button>
          </Tooltip>
          <Tooltip :content="t('Undo (Ctrl+Z)')" side="bottom">
            <Button size="icon" variant="outline" :disabled="!canUndo" @click="flowStore.undo()">
              <Undo2 class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">{{ t("Undo (Ctrl+Z)") }}</span>
            </Button>
          </Tooltip>
          <Tooltip :content="t('Redo (Ctrl+Y)')" side="bottom">
            <Button size="icon" variant="outline" :disabled="!canRedo" @click="flowStore.redo()">
              <Redo2 class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">{{ t("Redo (Ctrl+Y)") }}</span>
            </Button>
          </Tooltip>
          <Tooltip :content="t('Auto Layout')" side="bottom">
            <Button size="icon" variant="outline" @click="autoLayout">
              <LayoutGrid class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">{{ t("Auto Layout") }}</span>
            </Button>
          </Tooltip>
          <Tooltip :content="t('Save Project (Ctrl+S)')" side="bottom">
            <Button size="icon" :disabled="saveBusy || loading" @click="saveProject">
              <Save class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">{{ t("Save Project (Ctrl+S)") }}</span>
            </Button>
          </Tooltip>
        </div>
      </div>

    </header>

    <div v-if="loading" class="flex flex-1 items-center justify-center px-6 text-sm text-muted-foreground">
      {{ t("Loading project...") }}
    </div>

    <div v-else class="relative flex-1 min-h-0 overflow-hidden">
      <div class="flex h-full min-h-0 flex-col p-4 transition-[padding] duration-200" :class="nodeDetailOpen ? 'pr-[440px]' : ''">
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
              <CardHeader class="w-full items-start" :title="selectedNode.id" title-class="text-lg">
                <template #actions>
                  <Button size="icon" variant="ghost" @click="closeNodeDetail">
                    <X class="h-4 w-4" aria-hidden="true" />
                    <span class="sr-only">{{ t("Close") }}</span>
                  </Button>
                </template>
              </CardHeader>
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
                    {{ t("Node ID must be unique. Renaming updates all connected edges.") }}
                  </p>
                </div>

                <div v-if="selectedNodeValidation.length" class="rounded-xl border border-rose-200 bg-rose-50/80 p-3 text-xs text-rose-700">
                  <p class="font-semibold uppercase tracking-[0.18em]">{{ t("Validation") }}</p>
                  <ul class="mt-2 space-y-1">
                    <li v-for="message in selectedNodeValidation" :key="message">• {{ message }}</li>
                  </ul>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <div>
                    <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                      {{ t("Kind") }}
                    </label>
                    <select
                      :value="selectedNode.kind"
                      class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                      @change="onSelectedNodeKindChange"
                    >
                      <option value="call">{{ t("Call") }}</option>
                      <option value="compose">{{ t("Compose") }}</option>
                    </select>
                  </div>

                  <div>
                    <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                      {{ t("Allow Fail") }}
                    </label>
                    <div class="mt-2 flex h-10 items-center gap-2 rounded-md border border-input bg-background px-3 text-sm">
                      <input
                        v-model="selectedNode.allowFail"
                        type="checkbox"
                        class="h-4 w-4 rounded border"
                        @change="flowStore.commitHistory()"
                      />
                      <span class="text-muted-foreground">{{ t("Continue on error") }}</span>
                    </div>
                  </div>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <div>
                    <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                      {{ t("Retry") }}
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
                      {{ t("Timeout (ms)") }}
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
                  <div class="flex flex-wrap items-center justify-between gap-3">
                    <div>
                      <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                        {{ t("Spec Mode") }}
                      </p>
                      <p class="mt-1 text-[11px] text-muted-foreground">
                        {{ t("Form mode edits the supported fields directly. Advanced JSON is the escape hatch for full spec editing.") }}
                      </p>
                    </div>
                    <div class="flex gap-2">
                      <Button
                        size="sm"
                        :variant="selectedNode.specEditorMode === 'form' ? 'default' : 'outline'"
                        @click="setSelectedNodeSpecMode('form')"
                      >
                        {{ t("Form") }}
                      </Button>
                      <Button
                        size="sm"
                        :variant="selectedNode.specEditorMode === 'json' ? 'default' : 'outline'"
                        @click="setSelectedNodeSpecMode('json')"
                      >
                        {{ t("Advanced JSON") }}
                      </Button>
                    </div>
                  </div>
                </div>

                <template v-if="selectedNode.specEditorMode === 'form'">
                  <div v-if="selectedNode.kind === 'call'" class="space-y-4">
                    <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                      <div class="flex flex-wrap items-start justify-between gap-3">
                        <div class="min-w-0">
                          <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                            {{ t("Call Method") }}
                          </p>
                          <p class="mt-2 break-all text-sm font-semibold">
                            {{ selectedNode.method || t("No method selected.") }}
                          </p>
                          <p class="mt-1 text-xs text-muted-foreground">
                            {{ selectedTargetLabel }}
                          </p>
                        </div>
                        <Button variant="outline" size="sm" @click="openMethodDialog">{{ t("Select Method") }}</Button>
                      </div>
                      <p class="mt-3 text-[11px] text-muted-foreground">
                        {{ t("Use the method dialog to choose a registered capability. The editor will keep method and target aligned.") }}
                      </p>
                    </div>

                    <div>
                      <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                        {{ t("Args Template (JSON)") }}
                      </label>
                      <textarea
                        v-model="selectedNode.argsTemplate"
                        rows="9"
                        class="mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-xs"
                        @blur="flowStore.commitHistory()"
                      />
                      <p class="mt-1 text-[11px] text-muted-foreground">
                        {{ t("This base JSON is materialized first, then input bindings write into it in order.") }}
                      </p>
                    </div>
                  </div>

                  <div v-else class="space-y-4">
                    <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                      <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                        {{ t("Compose Node") }}
                      </p>
                      <p class="mt-2 text-sm text-muted-foreground">
                        {{ t("Compose nodes do not call capabilities. They build a JSON result locally from template + bindings.") }}
                      </p>
                    </div>

                    <div>
                      <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                        {{ t("Template (JSON)") }}
                      </label>
                      <textarea
                        v-model="selectedNode.composeTemplate"
                        rows="9"
                        class="mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-xs"
                        @blur="flowStore.commitHistory()"
                      />
                      <p class="mt-1 text-[11px] text-muted-foreground">
                        {{ t("Compose starts from this JSON template and applies the same binding list as call nodes.") }}
                      </p>
                    </div>
                  </div>

                  <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                    <div class="flex flex-wrap items-start justify-between gap-3">
                      <div>
                        <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                          {{ t("Input Bindings") }}
                        </p>
                        <p class="mt-1 text-[11px] text-muted-foreground">
                          {{ t("Bindings can read trigger data, flow/run metadata, or ancestor node results.") }}
                        </p>
                      </div>
                      <Button variant="outline" size="sm" @click="addBinding">{{ t("Add Binding") }}</Button>
                    </div>

                    <div v-if="!selectedNode.inputs.length" class="mt-4 rounded-lg border border-dashed border-border/60 px-4 py-5 text-center text-xs text-muted-foreground">
                      {{ t("No bindings yet. Nodes can still run with their template alone.") }}
                    </div>

                    <div v-else class="mt-4 space-y-3">
                      <div v-for="(binding, index) in selectedNode.inputs" :key="`${selectedNode.id}-binding-${index}`" class="rounded-lg border border-border/70 bg-background/90 p-3">
                        <div class="flex items-start justify-between gap-3">
                          <div>
                            <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                              {{ t("Binding {index}", { index: index + 1 }) }}
                            </p>
                            <p class="mt-1 text-[11px] text-muted-foreground">
                              {{ t("Destination writes into the template. Source chooses where the value comes from.") }}
                            </p>
                          </div>
                          <Button size="sm" variant="ghost" @click="removeBinding(index)">{{ t("Remove") }}</Button>
                        </div>

                        <div class="mt-3 grid gap-3 md:grid-cols-2">
                          <div>
                            <label class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                              {{ t("Destination Pointer") }}
                            </label>
                            <input
                              v-model="binding.to"
                              class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                              placeholder="/payload/id"
                              @blur="flowStore.commitHistory()"
                            />
                          </div>

                          <div>
                            <label class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                              {{ t("Source Kind") }}
                            </label>
                            <select
                              :value="binding.sourceKind"
                              class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                              @change="onBindingSourceKindChange(binding, $event)"
                            >
                              <option value="node_result">{{ t("Ancestor Result") }}</option>
                              <option value="trigger">{{ t("Trigger") }}</option>
                              <option value="flow_meta">{{ t("Flow Meta") }}</option>
                              <option value="run_meta">{{ t("Run Meta") }}</option>
                            </select>
                          </div>
                        </div>

                        <div v-if="binding.sourceKind === 'node_result'" class="mt-3 grid gap-3 md:grid-cols-2">
                          <div>
                            <label class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                              {{ t("Ancestor Node") }}
                            </label>
                            <select
                              v-model="binding.nodeId"
                              class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                              @change="flowStore.commitHistory()"
                            >
                              <option value="">{{ ancestorNodeOptions.length ? t("Select ancestor node") : t("No ancestor available") }}</option>
                              <option v-for="ancestorId in ancestorNodeOptions" :key="ancestorId" :value="ancestorId">
                                {{ ancestorId }}
                              </option>
                            </select>
                          </div>

                          <div>
                            <label class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                              {{ t("Result Path") }}
                            </label>
                            <input
                              v-model="binding.path"
                              class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                              placeholder="/user/id"
                              @blur="flowStore.commitHistory()"
                            />
                          </div>
                        </div>

                        <div v-else-if="binding.sourceKind === 'trigger'" class="mt-3">
                          <label class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                            {{ t("Trigger Path") }}
                          </label>
                          <input
                            v-model="binding.path"
                            class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                            placeholder="/payload/name"
                            @blur="flowStore.commitHistory()"
                          />
                        </div>

                        <div v-else-if="binding.sourceKind === 'flow_meta' || binding.sourceKind === 'run_meta'" class="mt-3">
                          <label class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                            {{ t("Meta Field") }}
                          </label>
                          <select
                            v-model="binding.field"
                            class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                            @change="flowStore.commitHistory()"
                          >
                            <option v-if="binding.sourceKind === 'flow_meta'" value="flow_id">flow_id</option>
                            <option v-if="binding.sourceKind === 'run_meta'" value="run_id">run_id</option>
                          </select>
                        </div>

                        <label class="mt-3 flex items-center gap-2 text-xs text-muted-foreground">
                          <input
                            v-model="binding.required"
                            type="checkbox"
                            class="h-4 w-4 rounded border"
                            @change="flowStore.commitHistory()"
                          />
                          <span>{{ t("Required binding") }}</span>
                        </label>
                      </div>
                    </div>
                  </div>
                </template>

                <div v-else class="space-y-3">
                  <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                    <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                      {{ t("Advanced Spec (JSON)") }}
                    </p>
                    <p class="mt-2 text-[11px] text-muted-foreground">
                      {{ t("This is the full node spec. Switching back to form mode will validate JSON and map the supported fields into the visual editor.") }}
                    </p>
                    <textarea
                      v-model="selectedNode.specJson"
                      rows="16"
                      class="mt-3 w-full rounded-md border border-input bg-background px-3 py-2 text-xs"
                      @blur="flowStore.commitHistory()"
                    />
                  </div>
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
        <CardHeader
          class="items-start"
          :title="t('Select Capability')"
          :description="t('Pick a registered capability and the editor will keep method and target aligned.')"
          title-class="text-lg"
        >
          <template #actions>
            <div class="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
              <span class="rounded-full border border-border/60 px-3 py-1">
                {{ t("Executor {nodeId}", { nodeId: effectiveExecutorNode || "-" }) }}
              </span>
              <span class="rounded-full border border-border/60 px-3 py-1">
                {{ t("Query Node {nodeId}", { nodeId: capabilityQueryNodeLabel }) }}
              </span>
              <span class="rounded-full border border-border/60 px-3 py-1">
                {{ selectedTargetLabel }}
              </span>
            </div>
          </template>
        </CardHeader>

        <div class="mt-5 flex flex-wrap items-end gap-3">
          <div class="min-w-[200px] max-w-[240px] flex-1">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Query Node ID") }}
            </label>
            <input
              v-model="queryNodeIdDraft"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              inputmode="numeric"
              :placeholder="t('Current executor')"
            />
            <p class="mt-1 text-[11px] text-muted-foreground">
              {{ t("Used only for capability lookup. It will not be written back into the call node.") }}
            </p>
          </div>
          <div class="min-w-[240px] flex-[2]">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Filter") }}
            </label>
            <input
              v-model="methodSearch"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              :placeholder="t('Search method / provider / version')"
            />
          </div>
          <Button variant="outline" :disabled="flowStore.state.execCapabilitiesLoading" @click="refreshMethodCapabilities">
            {{ flowStore.state.execCapabilitiesLoading ? t("Refreshing...") : t("Refresh Capabilities") }}
          </Button>
        </div>

        <div class="mt-5 flex-1 overflow-y-auto rounded-xl border border-border/70 bg-background/80">
          <div
            v-if="flowStore.state.execCapabilitiesLoading && !flowStore.state.execCapabilities.length"
            class="px-4 py-10 text-center text-sm text-muted-foreground"
          >
            {{ t("Loading capability list...") }}
          </div>
          <div v-else-if="!filteredCapabilities.length" class="px-4 py-10 text-center text-sm text-muted-foreground">
            {{
              flowStore.state.execCapabilities.length
                ? t("No capability matched the current filter.")
                : t("No capability loaded yet. Refresh to query the selected node.")
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
                    {{ route.providerNode === effectiveExecutorNode ? t("Self") : t("Remote") }}
                  </span>
                </div>
                <p class="mt-1 text-xs text-muted-foreground">
                  {{ t("Provider {nodeId}", { nodeId: route.providerNode }) }}
                  <span v-if="route.viaNode > 0"> {{ t("via {nodeId}", { nodeId: route.viaNode }) }}</span>
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
                {{ pendingCapabilityKey === route.key ? t("Selected") : t("Choose") }}
              </span>
            </button>
          </div>
        </div>

        <div class="mt-5 flex flex-wrap items-center justify-between gap-3">
          <p class="text-xs text-muted-foreground">
            {{ t("Applying a capability updates the method and hidden call target. The query node stays temporary.") }}
          </p>
          <div class="flex gap-2">
            <Button variant="outline" @click="closeMethodDialog">{{ t("Cancel") }}</Button>
            <Button :disabled="!pendingCapabilityKey" @click="applyCapabilitySelection">{{ t("Apply Method") }}</Button>
          </div>
        </div>
      </div>
    </Overlay>

    <Overlay :open="addNodeOpen" @close="addNodeOpen = false">
      <div class="w-full max-w-md rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">{{ t("Add Node") }}</h2>
        <div class="mt-4 space-y-3">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Node ID") }}
            </label>
            <input
              v-model="nodeDraft.id"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>

          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Kind") }}
            </label>
            <div class="mt-2 grid grid-cols-2 gap-2">
              <button
                type="button"
                class="rounded-md border px-3 py-2 text-sm transition"
                :class="nodeDraft.kind === 'call' ? 'border-primary bg-primary/10 text-primary' : 'border-border/70 bg-background text-foreground'"
                @click="nodeDraft.kind = 'call'"
              >
                {{ t("Call") }}
              </button>
              <button
                type="button"
                class="rounded-md border px-3 py-2 text-sm transition"
                :class="nodeDraft.kind === 'compose' ? 'border-primary bg-primary/10 text-primary' : 'border-border/70 bg-background text-foreground'"
                @click="nodeDraft.kind = 'compose'"
              >
                {{ t("Compose") }}
              </button>
            </div>
            <p class="mt-1 text-[11px] text-muted-foreground">
              {{
                nodeDraft.kind === "call"
                  ? t("Call nodes execute a capability and can bind ancestor outputs into args.")
                  : t("Compose nodes build local JSON output from template + bindings.")
              }}
            </p>
          </div>
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="addNodeOpen = false">{{ t("Cancel") }}</Button>
          <Button @click="addNode">{{ t("Add") }}</Button>
        </div>
      </div>
    </Overlay>
  </section>
</template>
