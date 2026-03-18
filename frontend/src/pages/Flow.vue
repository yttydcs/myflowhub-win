<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from "vue"
import { LayoutGrid, Link2Off, ListChecks, Play, Plus, Redo2, RefreshCw, Save, Trash2, Undo2 } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import { Tooltip } from "@/components/ui/tooltip"
import FlowCanvas from "@/components/flow/FlowCanvas.vue"
import { useFlowStore } from "@/stores/flow"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"

const flowStore = useFlowStore()
const sessionStore = useSessionStore()
const toast = useToastStore()

const addNodeOpen = ref(false)
const selectedCapabilityKey = ref("")
const nodeIdDraft = ref("")

const nodeDraft = reactive({
  id: "",
  kind: "local" as "local" | "exec"
})

const selectedNode = computed(
  () => flowStore.state.nodes[flowStore.state.selectedNodeIndex] ?? null
)

const selectedEdge = computed(
  () => flowStore.state.edges[flowStore.state.selectedEdgeIndex] ?? null
)

const canUndo = computed(() => flowStore.state.historyIndex > 0)
const canRedo = computed(
  () => flowStore.state.historyIndex >= 0 && flowStore.state.historyIndex < flowStore.state.historyLength - 1
)

const refreshList = async () => {
  try {
    await flowStore.listFlows()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to refresh flows.")
  }
}

const startNew = () => {
  flowStore.newDraft()
}

const saveFlow = async () => {
  try {
    await flowStore.saveFlow()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to save flow.")
  }
}

const autoLayout = () => {
  try {
    flowStore.autoLayoutTB()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to auto layout.")
  }
}

const undo = () => {
  flowStore.undo()
}

const redo = () => {
  flowStore.redo()
}

const runFlow = async () => {
  try {
    await flowStore.runFlow()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to run flow.")
  }
}

const statusFlow = async () => {
  try {
    await flowStore.statusFlow(flowStore.state.statusRunId)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to fetch status.")
  }
}

const selectFlow = async (flowId: string) => {
  try {
    await flowStore.getFlow(flowId)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to load flow.")
  }
}

const openAddNodeDialog = () => {
  nodeDraft.id = flowStore.suggestNodeId()
  nodeDraft.kind = "local"
  addNodeOpen.value = true
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

const saveNode = () => {
  try {
    flowStore.addNode(nodeDraft.id, nodeDraft.kind)
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

const loadExecCapabilities = async () => {
  const node = selectedNode.value
  if (!node || node.kind !== "exec") return
  try {
    await flowStore.queryExecCapabilities(node.method)
    selectedCapabilityKey.value = flowStore.state.execCapabilities[0]?.key ?? ""
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to query capabilities.")
  }
}

const applyExecCapability = () => {
  try {
    flowStore.applyExecCapability(selectedCapabilityKey.value)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to apply capability.")
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
  if (addNodeOpen.value) return

  const key = event.key || ""
  const lower = key.toLowerCase()
  const ctrl = event.ctrlKey || event.metaKey

  if (ctrl && lower === "s") {
    event.preventDefault()
    void saveFlow()
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
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  ([nodeId, hubId]) => {
    flowStore.setIdentity(Number(nodeId), Number(hubId))
  },
  { immediate: true }
)

watch(
  () => flowStore.state.message,
  (msg) => {
    const trimmed = msg.trim()
    if (!trimmed) return
    const lower = trimmed.toLowerCase()
    const isError =
      lower.includes("failed") ||
      lower.includes("error") ||
      lower.includes("timeout") ||
      lower.includes("timed out") ||
      lower.includes("unable")
    if (isError) {
      toast.error(trimmed)
    } else if (
      lower.includes("saved") ||
      lower.includes("loaded") ||
      lower.includes("updated") ||
      lower.includes("started") ||
      lower.includes("applied")
    ) {
      toast.success(trimmed)
    } else {
      toast.info(trimmed)
    }
    flowStore.state.message = ""
  }
)

watch(
  () => flowStore.state.selectedNodeIndex,
  () => {
    selectedCapabilityKey.value = ""
  }
)

watch(
  () => selectedNode.value?.id ?? "",
  (id) => {
    nodeIdDraft.value = id
  },
  { immediate: true }
)

onMounted(() => {
  void refreshList().catch(() => {})
  window.addEventListener("keydown", onKeyDown)
})

onUnmounted(() => window.removeEventListener("keydown", onKeyDown))
</script>

<template>
  <section class="space-y-6">
    <div class="grid gap-6 xl:grid-cols-[280px_minmax(0,1fr)_minmax(0,360px)]">
      <section class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
        <div class="flex items-center justify-between">
          <h2 class="text-sm font-semibold">Flows</h2>
          <span class="text-xs text-muted-foreground">{{ flowStore.state.flows.length }} items</span>
        </div>
        <div class="mt-3 space-y-2">
          <button
            v-for="flow in flowStore.state.flows"
            :key="flow.flowId"
            type="button"
            class="w-full rounded-xl border px-3 py-2 text-left text-sm transition"
            :class="flow.flowId === flowStore.state.flowId ? 'border-primary/60 bg-primary/10' : 'border-transparent hover:border-border/60 hover:bg-muted/60'"
            @click="selectFlow(flow.flowId)"
          >
            <p class="font-semibold">{{ flow.name || flow.flowId }}</p>
            <p class="text-xs text-muted-foreground">
              {{ flow.everyMs > 0 ? `every ${flow.everyMs} ms` : "non-interval trigger" }} · last
              {{ flow.lastStatus || "idle" }}
            </p>
          </button>
          <div v-if="!flowStore.state.flows.length" class="text-xs text-muted-foreground">
            No flows yet. Refresh after connecting to a node.
          </div>
        </div>
      </section>

      <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h2 class="text-sm font-semibold">Flow Editor</h2>
          <div class="flex flex-wrap items-center gap-2">
            <div
              class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground"
            >
              <span class="font-semibold uppercase tracking-[0.2em]">Executor</span>
              <input
                v-model="flowStore.state.targetId"
                class="h-7 w-24 rounded-md border border-input bg-background px-2 text-xs text-foreground"
                placeholder="Node ID"
              />
            </div>
            <div class="mx-1 h-6 w-px bg-border/60" aria-hidden="true" />
            <Tooltip content="Refresh Flows" side="bottom">
              <Button size="icon" variant="outline" @click="refreshList">
                <RefreshCw class="h-4 w-4" aria-hidden="true" />
                <span class="sr-only">Refresh Flows</span>
              </Button>
            </Tooltip>
            <Tooltip content="New Flow" side="bottom">
              <Button size="icon" variant="outline" @click="startNew">
                <Plus class="h-4 w-4" aria-hidden="true" />
                <span class="sr-only">New Flow</span>
              </Button>
            </Tooltip>
            <div class="mx-1 h-6 w-px bg-border/60" aria-hidden="true" />
            <Tooltip content="Undo (Ctrl+Z)" side="bottom">
              <Button size="icon" variant="outline" :disabled="!canUndo" @click="undo">
                <Undo2 class="h-4 w-4" aria-hidden="true" />
                <span class="sr-only">Undo</span>
              </Button>
            </Tooltip>
            <Tooltip content="Redo (Ctrl+Y)" side="bottom">
              <Button size="icon" variant="outline" :disabled="!canRedo" @click="redo">
                <Redo2 class="h-4 w-4" aria-hidden="true" />
                <span class="sr-only">Redo</span>
              </Button>
            </Tooltip>
            <div class="mx-1 h-6 w-px bg-border/60" aria-hidden="true" />
            <Tooltip content="Auto Layout" side="bottom">
              <Button size="icon" variant="outline" @click="autoLayout">
                <LayoutGrid class="h-4 w-4" aria-hidden="true" />
                <span class="sr-only">Auto Layout</span>
              </Button>
            </Tooltip>
            <Tooltip content="Save (Ctrl+S)" side="bottom">
              <Button size="icon" @click="saveFlow">
                <Save class="h-4 w-4" aria-hidden="true" />
                <span class="sr-only">Save</span>
              </Button>
            </Tooltip>
            <Tooltip content="Run" side="bottom">
              <Button size="icon" variant="outline" @click="runFlow">
                <Play class="h-4 w-4" aria-hidden="true" />
                <span class="sr-only">Run</span>
              </Button>
            </Tooltip>
            <Tooltip content="Status" side="bottom">
              <Button size="icon" variant="outline" @click="statusFlow">
                <ListChecks class="h-4 w-4" aria-hidden="true" />
                <span class="sr-only">Status</span>
              </Button>
            </Tooltip>
          </div>
        </div>
        <div class="mt-4 grid gap-4 md:grid-cols-3">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Flow ID
            </label>
            <input
              v-model="flowStore.state.flowId"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="flow_id (uuid recommended)"
              @blur="flowStore.commitHistory()"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Name
            </label>
            <input
              v-model="flowStore.state.flowName"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="Optional name"
              @blur="flowStore.commitHistory()"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Trigger
            </label>
            <select
              v-model="flowStore.state.triggerType"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              @change="flowStore.commitHistory()"
            >
              <option value="interval">interval</option>
              <option value="event">event</option>
              <option value="var_changed">var_changed</option>
            </select>
          </div>
        </div>
        <div v-if="flowStore.state.triggerType === 'interval'" class="mt-4 grid gap-4 md:grid-cols-2">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Every (ms)
            </label>
            <input
              v-model.number="flowStore.state.everyMs"
              type="number"
              min="1"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="60000"
              @blur="flowStore.commitHistory()"
            />
          </div>
        </div>
        <div v-else-if="flowStore.state.triggerType === 'event'" class="mt-4 grid gap-4 md:grid-cols-3">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Event Mode
            </label>
            <select
              v-model="flowStore.state.eventMode"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              @change="flowStore.commitHistory()"
            >
              <option value="publish">publish</option>
              <option value="received">received</option>
              <option value="any">any</option>
            </select>
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Event Name
            </label>
            <input
              v-model="flowStore.state.eventName"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="topicbus publish name"
              @blur="flowStore.commitHistory()"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Event Topic
            </label>
            <input
              v-model="flowStore.state.eventTopic"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="topic name"
              @blur="flowStore.commitHistory()"
            />
          </div>
        </div>
        <div v-else class="mt-4 grid gap-4 md:grid-cols-2">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Var Owner
            </label>
            <input
              v-model.number="flowStore.state.varOwner"
              type="number"
              min="0"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="0 = any owner"
              @blur="flowStore.commitHistory()"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Var Name
            </label>
            <input
              v-model="flowStore.state.varName"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="empty = any name"
              @blur="flowStore.commitHistory()"
            />
          </div>
        </div>

        <div class="mt-6 space-y-3">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="text-xs text-muted-foreground">
              Drag nodes to reposition. Drag from a node handle to connect nodes.
            </div>
            <div class="flex flex-wrap gap-2">
              <Tooltip content="Add Node" side="bottom">
                <Button size="icon" variant="outline" @click="openAddNodeDialog">
                  <Plus class="h-4 w-4" aria-hidden="true" />
                  <span class="sr-only">Add Node</span>
                </Button>
              </Tooltip>
              <Tooltip content="Remove Node (Delete)" side="bottom">
                <Button
                  size="icon"
                  variant="outline"
                  :disabled="flowStore.state.selectedNodeIndex < 0"
                  @click="removeNode"
                >
                  <Trash2 class="h-4 w-4" aria-hidden="true" />
                  <span class="sr-only">Remove Node</span>
                </Button>
              </Tooltip>
              <Tooltip content="Remove Edge (Delete)" side="bottom">
                <Button
                  size="icon"
                  variant="outline"
                  :disabled="flowStore.state.selectedEdgeIndex < 0"
                  @click="removeEdge"
                >
                  <Link2Off class="h-4 w-4" aria-hidden="true" />
                  <span class="sr-only">Remove Edge</span>
                </Button>
              </Tooltip>
            </div>
          </div>

          <FlowCanvas
            :nodes="flowStore.state.nodes"
            :edges="flowStore.state.edges"
            :selected-node-id="selectedNode?.id ?? null"
            :selected-edge="selectedEdge"
            :status-nodes="flowStore.state.lastStatus.nodes"
            @connect="onCanvasConnect"
            @select-node="onCanvasSelectNode"
            @select-edge="onCanvasSelectEdge"
            @node-moved="onCanvasNodeMoved"
            @clear-selection="onCanvasClear"
          />
        </div>
      </section>

      <section class="space-y-4">
        <div class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
          <h3 class="text-sm font-semibold">Node Detail</h3>
          <div v-if="selectedNode" class="mt-4 space-y-3">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Node ID
              </label>
              <input
                v-model="nodeIdDraft"
                class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
                @blur="commitNodeId"
                @keydown.enter.prevent="commitNodeId"
              />
              <p class="mt-1 text-[11px] text-muted-foreground">
                Node ID must be unique. Renaming updates all connected edges.
              </p>
            </div>
            <div class="grid gap-3 md:grid-cols-2">
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  Kind
                </label>
                <select
                  v-model="selectedNode.kind"
                  @change="flowStore.commitHistory()"
                  class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
                >
                  <option value="local">local</option>
                  <option value="exec">exec</option>
                </select>
              </div>
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  Allow Fail
                </label>
                <div class="mt-2 flex items-center gap-2 text-sm text-muted-foreground">
                  <input
                    v-model="selectedNode.allowFail"
                    type="checkbox"
                    class="h-4 w-4 rounded border"
                    @change="flowStore.commitHistory()"
                  />
                  <span>Continue on error</span>
                </div>
              </div>
            </div>
            <div class="grid gap-3 md:grid-cols-2">
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  Retry
                </label>
                <input
                  v-model.number="selectedNode.retry"
                  type="number"
                  min="0"
                  class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
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
                  class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
                  @blur="flowStore.commitHistory()"
                />
              </div>
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Method
              </label>
              <input
                v-model="selectedNode.method"
                class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
                placeholder="method name"
                @blur="flowStore.commitHistory()"
              />
            </div>
            <div v-if="selectedNode.kind === 'exec'">
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Target Node
              </label>
              <input
                v-model.number="selectedNode.target"
                type="number"
                min="0"
                class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
                @blur="flowStore.commitHistory()"
              />
              <div class="mt-3 space-y-2 rounded-lg border border-border/60 bg-background/60 p-3">
                <div class="flex flex-wrap items-center gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    :disabled="flowStore.state.execCapabilitiesLoading"
                    @click="loadExecCapabilities"
                  >
                    {{ flowStore.state.execCapabilitiesLoading ? "Loading..." : "Load Capabilities" }}
                  </Button>
                  <span class="text-[11px] text-muted-foreground">
                    query by current Method prefix
                  </span>
                </div>
                <div v-if="flowStore.state.execCapabilities.length" class="space-y-2">
                  <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    Capability
                  </label>
                  <select
                    v-model="selectedCapabilityKey"
                    class="h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
                  >
                    <option
                      v-for="route in flowStore.state.execCapabilities"
                      :key="route.key"
                      :value="route.key"
                    >
                      {{ route.label }}
                    </option>
                  </select>
                  <Button size="sm" variant="outline" @click="applyExecCapability">
                    Use Selected Capability
                  </Button>
                </div>
                <p v-else class="text-[11px] text-muted-foreground">
                  No capability cached. Click "Load Capabilities" to pick one.
                </p>
              </div>
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Args (JSON)
              </label>
              <textarea
                v-model="selectedNode.args"
                rows="5"
                class="mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-xs"
                @blur="flowStore.commitHistory()"
              />
            </div>
          </div>
          <div v-else class="mt-3 text-xs text-muted-foreground">
            Select a node to edit its details.
          </div>
        </div>

        <div class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-semibold">Status</h3>
            <span class="text-xs text-muted-foreground">
              run {{ flowStore.state.lastStatus.runId || "-" }}
            </span>
          </div>
          <p class="mt-2 text-xs text-muted-foreground">
            {{ flowStore.state.lastStatus.status || "No status yet." }}
          </p>
          <div class="mt-4 space-y-2">
            <div
              v-for="node in flowStore.state.lastStatus.nodes"
              :key="`${node.id}-${node.status}`"
              class="rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-xs"
            >
              <p class="font-semibold">{{ node.id || "unknown" }} · {{ node.status }}</p>
              <p class="text-muted-foreground">
                code {{ node.code }}{{ node.msg ? ` · ${node.msg}` : "" }}
              </p>
            </div>
            <div v-if="!flowStore.state.lastStatus.nodes.length" class="text-xs text-muted-foreground">
              No node status reports yet.
            </div>
          </div>
        </div>
      </section>
    </div>
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
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Kind
            </label>
            <select
              v-model="nodeDraft.kind"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            >
              <option value="local">local</option>
              <option value="exec">exec</option>
            </select>
          </div>
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="addNodeOpen = false">Cancel</Button>
          <Button @click="saveNode">Add</Button>
        </div>
      </div>
    </Overlay>
  </section>
</template>
