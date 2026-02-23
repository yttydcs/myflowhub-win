<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from "vue"
import { Button } from "@/components/ui/button"
import FlowCanvas from "@/components/flow/FlowCanvas.vue"
import { useFlowStore } from "@/stores/flow"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"

const flowStore = useFlowStore()
const sessionStore = useSessionStore()
const toast = useToastStore()

const addNodeOpen = ref(false)

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
  nodeDraft.id = ""
  nodeDraft.kind = "local"
  addNodeOpen.value = true
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

onMounted(() => {
  void refreshList().catch(() => {})
  window.addEventListener("keydown", onKeyDown)
})

onUnmounted(() => window.removeEventListener("keydown", onKeyDown))
</script>

<template>
  <section class="space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
          Flow Console
        </p>
        <h1 class="text-2xl font-semibold">Flow Builder</h1>
        <p class="text-sm text-muted-foreground">
          Design DAGs, deploy to target nodes, and monitor execution status.
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
          <span class="font-semibold uppercase tracking-[0.2em]">Executor</span>
          <input
            v-model="flowStore.state.targetId"
            class="h-7 w-24 rounded-md border border-input bg-background px-2 text-xs text-foreground"
            placeholder="Node ID"
          />
        </div>
        <Button variant="outline" size="sm" @click="refreshList">Refresh</Button>
        <Button variant="outline" size="sm" @click="startNew">New</Button>
        <Button variant="outline" size="sm" :disabled="!canUndo" @click="undo">Undo</Button>
        <Button variant="outline" size="sm" :disabled="!canRedo" @click="redo">Redo</Button>
        <Button variant="outline" size="sm" @click="autoLayout">Auto Layout</Button>
        <Button size="sm" @click="saveFlow">Save</Button>
        <Button variant="outline" size="sm" @click="runFlow">Run</Button>
        <Button variant="outline" size="sm" @click="statusFlow">Status</Button>
      </div>
    </div>

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
              every {{ flow.everyMs }} ms · last {{ flow.lastStatus || "idle" }}
            </p>
          </button>
          <div v-if="!flowStore.state.flows.length" class="text-xs text-muted-foreground">
            No flows yet. Refresh after connecting to a node.
          </div>
        </div>
      </section>

      <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
        <h2 class="text-sm font-semibold">Flow Editor</h2>
        <div class="mt-4 grid gap-4 md:grid-cols-3">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Flow ID
            </label>
            <input
              v-model="flowStore.state.flowId"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="flow_id (uuid recommended)"
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
            />
          </div>
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
            />
          </div>
        </div>

        <div class="mt-6 space-y-3">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="text-xs text-muted-foreground">
              Drag nodes to reposition. Drag from a node handle to connect nodes.
            </div>
            <div class="flex flex-wrap gap-2">
              <Button size="sm" variant="outline" @click="openAddNodeDialog">Add Node</Button>
              <Button
                size="sm"
                variant="outline"
                :disabled="flowStore.state.selectedNodeIndex < 0"
                @click="removeNode"
              >
                Remove Node
              </Button>
              <Button
                size="sm"
                variant="outline"
                :disabled="flowStore.state.selectedEdgeIndex < 0"
                @click="removeEdge"
              >
                Remove Edge
              </Button>
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
                :value="selectedNode.id"
                disabled
                class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
              />
              <p class="mt-1 text-[11px] text-muted-foreground">
                Node ID is the graph key and cannot be changed.
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
    <div v-if="addNodeOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-black/40 p-6">
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
    </div>
  </section>
</template>
