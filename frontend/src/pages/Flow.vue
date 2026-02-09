<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { Button } from "@/components/ui/button"
import { useFlowStore } from "@/stores/flow"
import { useSessionStore } from "@/stores/session"

const flowStore = useFlowStore()
const sessionStore = useSessionStore()

const message = ref("")

const addNodeOpen = ref(false)
const addEdgeOpen = ref(false)

const nodeDraft = reactive({
  id: "",
  kind: "local" as "local" | "exec"
})

const edgeDraft = reactive({
  from: "",
  to: ""
})

const selectedNode = computed(
  () => flowStore.state.nodes[flowStore.state.selectedNodeIndex] ?? null
)

const nodeIds = computed(() =>
  flowStore.state.nodes.map((node) => node.id).filter((id) => id.trim() !== "")
)

const refreshList = async () => {
  message.value = ""
  try {
    await flowStore.listFlows()
  } catch (err) {
    console.warn(err)
    message.value = "Failed to refresh flows."
  }
}

const startNew = () => {
  message.value = ""
  flowStore.newDraft()
}

const saveFlow = async () => {
  message.value = ""
  try {
    await flowStore.saveFlow()
  } catch (err) {
    console.warn(err)
    message.value = (err as Error)?.message || "Failed to save flow."
  }
}

const runFlow = async () => {
  message.value = ""
  try {
    await flowStore.runFlow()
  } catch (err) {
    console.warn(err)
    message.value = (err as Error)?.message || "Failed to run flow."
  }
}

const statusFlow = async () => {
  message.value = ""
  try {
    await flowStore.statusFlow(flowStore.state.statusRunId)
  } catch (err) {
    console.warn(err)
    message.value = (err as Error)?.message || "Failed to fetch status."
  }
}

const selectFlow = async (flowId: string) => {
  message.value = ""
  try {
    await flowStore.getFlow(flowId)
  } catch (err) {
    console.warn(err)
    message.value = (err as Error)?.message || "Failed to load flow."
  }
}

const openAddNodeDialog = () => {
  nodeDraft.id = ""
  nodeDraft.kind = "local"
  addNodeOpen.value = true
}

const saveNode = () => {
  message.value = ""
  try {
    flowStore.addNode(nodeDraft.id, nodeDraft.kind)
    addNodeOpen.value = false
  } catch (err) {
    console.warn(err)
    message.value = (err as Error)?.message || "Failed to add node."
  }
}

const removeNode = () => {
  flowStore.removeSelectedNode()
}

const openAddEdgeDialog = () => {
  if (nodeIds.value.length < 2) {
    message.value = "Add at least two nodes before creating edges."
    return
  }
  edgeDraft.from = nodeIds.value[0] ?? ""
  edgeDraft.to = nodeIds.value[1] ?? ""
  addEdgeOpen.value = true
}

const saveEdge = () => {
  message.value = ""
  try {
    flowStore.addEdge(edgeDraft.from, edgeDraft.to)
    addEdgeOpen.value = false
  } catch (err) {
    console.warn(err)
    message.value = (err as Error)?.message || "Failed to add edge."
  }
}

const removeEdge = () => {
  flowStore.removeSelectedEdge()
}

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  ([nodeId, hubId]) => {
    flowStore.setIdentity(Number(nodeId), Number(hubId))
  },
  { immediate: true }
)

onMounted(async () => {
  try {
    await refreshList()
  } catch {
    // handled in refreshList
  }
})
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

        <div class="mt-6 grid gap-4 lg:grid-cols-2">
          <div class="rounded-xl border border-border/60 bg-background/60 p-4">
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-semibold">Nodes</h3>
              <div class="flex gap-2">
                <Button size="sm" variant="outline" @click="openAddNodeDialog">Add</Button>
                <Button
                  size="sm"
                  variant="outline"
                  :disabled="flowStore.state.selectedNodeIndex < 0"
                  @click="removeNode"
                >
                  Remove
                </Button>
              </div>
            </div>
            <div class="mt-3 max-h-[260px] space-y-2 overflow-y-auto">
              <button
                v-for="(node, index) in flowStore.state.nodes"
                :key="`${node.id}-${index}`"
                type="button"
                class="w-full rounded-lg border px-3 py-2 text-left text-xs transition"
                :class="flowStore.state.selectedNodeIndex === index ? 'border-primary/50 bg-primary/10' : 'border-transparent hover:border-border/60 hover:bg-muted/60'"
                @click="flowStore.selectNode(index)"
              >
                <p class="font-semibold">{{ node.id || "Unnamed node" }}</p>
                <p class="text-[11px] text-muted-foreground">
                  kind {{ node.kind || "local" }} · retry {{ node.retry }} · timeout {{ node.timeoutMs }} ms
                </p>
              </button>
              <div v-if="!flowStore.state.nodes.length" class="text-xs text-muted-foreground">
                No nodes yet. Add the first node to start.
              </div>
            </div>
          </div>

          <div class="rounded-xl border border-border/60 bg-background/60 p-4">
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-semibold">Edges</h3>
              <div class="flex gap-2">
                <Button size="sm" variant="outline" @click="openAddEdgeDialog">Add</Button>
                <Button
                  size="sm"
                  variant="outline"
                  :disabled="flowStore.state.selectedEdgeIndex < 0"
                  @click="removeEdge"
                >
                  Remove
                </Button>
              </div>
            </div>
            <div class="mt-3 max-h-[260px] space-y-2 overflow-y-auto">
              <button
                v-for="(edge, index) in flowStore.state.edges"
                :key="`${edge.from}-${edge.to}-${index}`"
                type="button"
                class="w-full rounded-lg border px-3 py-2 text-left text-xs transition"
                :class="flowStore.state.selectedEdgeIndex === index ? 'border-primary/50 bg-primary/10' : 'border-transparent hover:border-border/60 hover:bg-muted/60'"
                @click="flowStore.selectEdge(index)"
              >
                <p class="font-semibold">{{ edge.from }} → {{ edge.to }}</p>
              </button>
              <div v-if="!flowStore.state.edges.length" class="text-xs text-muted-foreground">
                No edges yet. Connect nodes to create the DAG.
              </div>
            </div>
          </div>
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
                v-model="selectedNode.id"
                class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
              />
            </div>
            <div class="grid gap-3 md:grid-cols-2">
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  Kind
                </label>
                <select
                  v-model="selectedNode.kind"
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
                  <input v-model="selectedNode.allowFail" type="checkbox" class="h-4 w-4 rounded border" />
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

    <p v-if="message" class="text-sm text-rose-600">{{ message }}</p>
    <p v-else-if="flowStore.state.message" class="text-sm text-muted-foreground">
      {{ flowStore.state.message }}
    </p>

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

    <div v-if="addEdgeOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-black/40 p-6">
      <div class="w-full max-w-md rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">Add Edge</h2>
        <div class="mt-4 space-y-3">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              From
            </label>
            <select
              v-model="edgeDraft.from"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            >
              <option v-for="id in nodeIds" :key="`from-${id}`" :value="id">{{ id }}</option>
            </select>
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              To
            </label>
            <select
              v-model="edgeDraft.to"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            >
              <option v-for="id in nodeIds" :key="`to-${id}`" :value="id">{{ id }}</option>
            </select>
          </div>
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="addEdgeOpen = false">Cancel</Button>
          <Button @click="saveEdge">Add</Button>
        </div>
      </div>
    </div>
  </section>
</template>
