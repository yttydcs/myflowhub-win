<script setup lang="ts">
import { computed } from "vue"
import { Position, VueFlow, type Connection } from "@vue-flow/core"
import type { FlowEdge, FlowNodeDraft } from "@/stores/flow"

type SelectedEdge = FlowEdge | null

const props = defineProps<{
  nodes: FlowNodeDraft[]
  edges: FlowEdge[]
  selectedNodeId: string | null
  selectedEdge: SelectedEdge
}>()

const emit = defineEmits<{
  (event: "select-node", nodeId: string): void
  (event: "select-edge", from: string, to: string): void
  (event: "connect", from: string, to: string): void
  (event: "node-moved", nodeId: string, x: number, y: number): void
  (event: "clear-selection"): void
}>()

const hasNode = (id: string) => props.nodes.some((node) => node.id === id)

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
      data: { label: node.id },
      sourcePosition: Position.Right,
      targetPosition: Position.Left,
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

const onNodeClick = (_: unknown, node: any) => {
  if (!node?.id) return
  emit("select-node", String(node.id))
}

const onEdgeClick = (_: unknown, edge: any) => {
  const from = String(edge?.source ?? "")
  const to = String(edge?.target ?? "")
  if (!from || !to) return
  emit("select-edge", from, to)
}

const onPaneClick = () => emit("clear-selection")

const onNodeDragStop = (_: unknown, node: any) => {
  const id = String(node?.id ?? "")
  const x = Number(node?.position?.x ?? 0)
  const y = Number(node?.position?.y ?? 0)
  if (!id) return
  if (!Number.isFinite(x) || !Number.isFinite(y)) return
  emit("node-moved", id, x, y)
}
</script>

<template>
  <div class="h-[560px] w-full overflow-hidden rounded-xl border border-border/60 bg-background/60">
    <VueFlow
      :nodes="canvasNodes"
      :edges="canvasEdges"
      :fit-view-on-init="true"
      :min-zoom="0.2"
      :max-zoom="2"
      :is-valid-connection="isValidConnection"
      @connect="onConnect"
      @nodeClick="onNodeClick"
      @edgeClick="onEdgeClick"
      @paneClick="onPaneClick"
      @nodeDragStop="onNodeDragStop"
    />
  </div>
</template>
