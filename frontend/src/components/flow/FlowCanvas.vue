<script setup lang="ts">
import { computed } from "vue"
import { VueFlow, type Connection, type EdgeMouseEvent, type NodeDragEvent, type NodeMouseEvent } from "@vue-flow/core"
import { Background } from "@vue-flow/background"
import { Controls } from "@vue-flow/controls"
import { MiniMap } from "@vue-flow/minimap"
import "@vue-flow/controls/dist/style.css"
import "@vue-flow/minimap/dist/style.css"
import FlowNode from "@/components/flow/FlowNode.vue"
import type { FlowEdge, FlowNodeDraft, FlowStatusNode } from "@/stores/flow"

type SelectedEdge = FlowEdge | null

const props = defineProps<{
  nodes: FlowNodeDraft[]
  edges: FlowEdge[]
  selectedNodeId: string | null
  selectedEdge: SelectedEdge
  statusNodes: FlowStatusNode[]
}>()

const emit = defineEmits<{
  (event: "select-node", nodeId: string): void
  (event: "select-edge", from: string, to: string): void
  (event: "connect", from: string, to: string): void
  (event: "node-moved", nodeId: string, x: number, y: number): void
  (event: "clear-selection"): void
}>()

const hasNode = (id: string) => props.nodes.some((node) => node.id === id)

const statusByNodeId = computed(() => {
  const map = new Map<string, FlowStatusNode>()
  for (const node of props.statusNodes) {
    const id = node.id?.trim() || ""
    if (!id) continue
    map.set(id, node)
  }
  return map
})

const isValidConnection = (conn: Connection): boolean => {
  const source = conn.source?.trim() ?? ""
  const target = conn.target?.trim() ?? ""
  if (!source || !target) return false
  if (source === target) return false
  if (!hasNode(source) || !hasNode(target)) return false
  if (props.edges.some((edge) => edge.from === source && edge.to === target)) return false
  return true
}

const canvasNodes = computed(() =>
  props.nodes
    .filter((node) => node.id.trim() !== "")
    .map((node) => ({
      id: node.id,
      position: { x: Number(node.x || 0), y: Number(node.y || 0) },
      data: { label: node.id, status: statusByNodeId.value.get(node.id) },
      type: "flowNode",
      draggable: true,
      selected: props.selectedNodeId === node.id
    }))
)

const canvasEdges = computed(() =>
  props.edges
    .filter((edge) => edge.from.trim() !== "" && edge.to.trim() !== "")
    .map((edge) => ({
      id: `e:${edge.from}->${edge.to}`,
      source: edge.from,
      target: edge.to,
      type: "smoothstep",
      selected: props.selectedEdge?.from === edge.from && props.selectedEdge?.to === edge.to
    }))
)

const onConnect = (conn: Connection) => {
  const source = conn.source?.trim() ?? ""
  const target = conn.target?.trim() ?? ""
  if (!source || !target) return
  emit("connect", source, target)
}

const onNodeClick = (payload: NodeMouseEvent) => {
  const id = String(payload?.node?.id ?? "").trim()
  if (!id) return
  emit("select-node", id)
}

const onEdgeClick = (payload: EdgeMouseEvent) => {
  const from = String(payload?.edge?.source ?? "").trim()
  const to = String(payload?.edge?.target ?? "").trim()
  if (!from || !to) return
  emit("select-edge", from, to)
}

const onPaneClick = () => emit("clear-selection")

const onNodeDragStop = (payload: NodeDragEvent) => {
  const id = String(payload?.node?.id ?? "").trim()
  const x = Number(payload?.node?.position?.x ?? 0)
  const y = Number(payload?.node?.position?.y ?? 0)
  if (!id) return
  if (!Number.isFinite(x) || !Number.isFinite(y)) return
  emit("node-moved", id, x, y)
}

const nodeTypes = {
  flowNode: FlowNode
}
</script>

<template>
  <div class="h-[560px] w-full overflow-hidden rounded-xl border border-border/60 bg-background/60">
    <VueFlow
      :nodes="canvasNodes"
      :edges="canvasEdges"
      :node-types="nodeTypes"
      :fit-view-on-init="true"
      :min-zoom="0.2"
      :max-zoom="2"
      :is-valid-connection="isValidConnection"
      @connect="onConnect"
      @node-click="onNodeClick"
      @edge-click="onEdgeClick"
      @pane-click="onPaneClick"
      @node-drag-stop="onNodeDragStop"
    >
      <Background :gap="18" :size="1" class="opacity-60" />
      <MiniMap
        pannable
        zoomable
        position="bottom-right"
        class="rounded-lg border border-border/60 bg-background/70 shadow-sm"
      />
      <Controls position="bottom-left" class="rounded-lg border border-border/60 bg-background/70 shadow-sm" />
    </VueFlow>
  </div>
</template>
