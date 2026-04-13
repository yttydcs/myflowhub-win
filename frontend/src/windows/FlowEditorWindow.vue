<script setup lang="ts">
// Context: implements the detached flow editor window used by the Win frontend.
import { computed, onMounted, onUnmounted, reactive, ref, watch } from "vue"
import { useRoute } from "vue-router"
import FlowCanvas from "@/components/flow/FlowCanvas.vue"
import FlowAddNodeDialog from "@/components/flow/editor/FlowAddNodeDialog.vue"
import FlowBodyNodeInspector from "@/components/flow/editor/FlowBodyNodeInspector.vue"
import FlowEdgeInspector from "@/components/flow/editor/FlowEdgeInspector.vue"
import FlowEditorToolbar from "@/components/flow/editor/FlowEditorToolbar.vue"
import FlowFieldBindingDialog from "@/components/flow/editor/FlowFieldBindingDialog.vue"
import FlowMethodPickerDialog from "@/components/flow/editor/FlowMethodPickerDialog.vue"
import FlowNodeInspector from "@/components/flow/editor/FlowNodeInspector.vue"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"
import { normalizeFormInputText, parseNumberInput, type FormInputValue } from "@/lib/numberInput"
import { readValueAtPointer } from "@/stores/flow_json_pointer"
import { resolveMethodVisualSchema } from "@/stores/flow_schema_resolver"
import {
  buildNodeVisualFormModel,
  clearBindingForPointer,
  setBindingForPointer,
  setLiteralFieldValue as setLiteralFieldValueInDoc
} from "@/stores/flow_visual_form"
import {
  createGraphEditorStateFromDraft,
  exportLooseGraphDraftFromEditorState,
  flowStatusLabelKey,
  useFlowStore,
  type ExecCapabilityRoute,
  type FlowBindingSourceKind,
  type FlowEdge,
  type FlowGraphEditorState,
  type FlowInputBindingDraft,
  type FlowNodeDraft,
  type FlowNodeKind,
  type NodeVisualFormModel,
  type VisualBindingSource,
  type VisualFieldModel
} from "@/stores/flow"
import { useFlowProjectsStore, type FlowProjectRecord } from "@/stores/flowProjects"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"
import { HomeState as LoadHomeState } from "../../wailsjs/go/main/App"

type FlowEditorBodySessionRecord = {
  parentNodeId: string
  snapshot: FlowGraphEditorState
  syncedSignature: string
}

type FlowEditorRecoveryRecord = {
  version: 1
  projectId: string
  baseSignature: string
  savedAt: string
  snapshot: FlowGraphEditorState
  bodySession?: FlowEditorBodySessionRecord | null
}

const recoveryStoragePrefix = "myflowhub.flow-editor.recovery:"

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
const runBusy = ref(false)
const statusBusy = ref(false)
const runHistoryBusy = ref(false)
const cancelBusy = ref(false)
const addNodeOpen = ref(false)
const methodDialogOpen = ref(false)
const fieldBindingDialogOpen = ref(false)
const methodSearch = ref("")
const queryNodeIdDraft = ref("")
const nodeIdDraft = ref("")
const pendingCapabilityKey = ref("")
const lastCapabilityQueryNode = ref("")
const activeBindingFieldPointer = ref("")
const fieldDrafts = reactive<Record<string, FormInputValue>>({})
const lastSavedSignature = ref("")
const lastSavedAt = ref("")
const bodyEditorSession = ref<FlowEditorBodySessionRecord | null>(null)
const bodyNodeIdDraft = ref("")
const bodySessionError = ref("")
let recoveryWriteTimer: number | null = null

const fieldBindingDraft = reactive({
  sourceKind: "trigger" as FlowBindingSourceKind,
  nodeId: "",
  path: "",
  field: "flow_id",
  name: "",
  required: false
})

const nodeDraft = reactive({
  id: "",
  kind: "call" as FlowNodeKind
})

const flowLocalVarNamePattern = /^[A-Za-z_][A-Za-z0-9_]*$/

const isJSONObject = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object" && !Array.isArray(value)

const formatJSONText = (value: unknown, fallback: unknown = {}) => JSON.stringify(value ?? fallback, null, 2)

const decodeJSONPointerToken = (token: string) => {
  let out = ""
  for (let i = 0; i < token.length; i += 1) {
    const char = token[i]
    if (char !== "~") {
      out += char
      continue
    }
    const next = token[i + 1]
    if (next === "0") {
      out += "~"
      i += 1
      continue
    }
    if (next === "1") {
      out += "/"
      i += 1
      continue
    }
    throw new Error(t("JSON Pointer contains an invalid escape sequence."))
  }
  return out
}

const validateJSONPointer = (pointer: string, errorMessage: string) => {
  const trimmed = String(pointer ?? "").trim()
  if (!trimmed) {
    return
  }
  if (!trimmed.startsWith("/")) {
    throw new Error(errorMessage)
  }
  for (const part of trimmed.slice(1).split("/")) {
    decodeJSONPointerToken(part)
  }
}

const normalizeFlowLocalVarName = (raw: unknown) => String(raw ?? "").trim()

const assertValidFlowLocalVarName = (name: string, requiredMessage: string, invalidMessage: string) => {
  if (!name) {
    throw new Error(requiredMessage)
  }
  if (!flowLocalVarNamePattern.test(name)) {
    throw new Error(invalidMessage)
  }
  return name
}

const selectedNode = computed(() => flowStore.state.nodes[flowStore.state.selectedNodeIndex] ?? null)
const selectedEdge = computed(() => flowStore.state.edges[flowStore.state.selectedEdgeIndex] ?? null)
const bodyEditorActive = computed(() => Boolean(bodyEditorSession.value))
const bodySelectedNode = computed(() => {
  const session = bodyEditorSession.value
  if (!session) return null
  return session.snapshot.nodes[session.snapshot.selectedNodeIndex] ?? null
})
const bodySelectedEdge = computed(() => {
  const session = bodyEditorSession.value
  if (!session) return null
  return session.snapshot.edges[session.snapshot.selectedEdgeIndex] ?? null
})
const bodySelectedEdgeSourceNode = computed(() => {
  const from = bodySelectedEdge.value?.from?.trim()
  if (!from) return null
  return bodyEditorSession.value?.snapshot.nodes.find((node) => node.id.trim() === from) ?? null
})
const activeCanvasNodes = computed(() => bodyEditorSession.value?.snapshot.nodes ?? flowStore.state.nodes)
const activeCanvasEdges = computed(() => bodyEditorSession.value?.snapshot.edges ?? flowStore.state.edges)
const activeSelectedNodeId = computed(() => (bodyEditorActive.value ? bodySelectedNode.value?.id ?? null : selectedNode.value?.id ?? null))
const activeSelectedEdge = computed<FlowEdge | null>(() => (bodyEditorActive.value ? bodySelectedEdge.value : selectedEdge.value))
const selectedEdgeSourceNode = computed(() => {
  const from = selectedEdge.value?.from?.trim()
  if (!from) return null
  return flowStore.state.nodes.find((node) => node.id.trim() === from) ?? null
})
const detailPanelOpen = computed(() =>
  bodyEditorActive.value ? Boolean(bodySelectedNode.value || bodySelectedEdge.value) : Boolean(selectedNode.value || selectedEdge.value)
)
const selfNodeId = computed(() => Number(sessionStore.auth.nodeId || fallbackIdentity.nodeId || 0))
const hubId = computed(() => Number(sessionStore.auth.hubId || fallbackIdentity.hubId || 0))

const canUndo = computed(() => flowStore.state.historyIndex > 0)
const canRedo = computed(
  () => flowStore.state.historyIndex >= 0 && flowStore.state.historyIndex < flowStore.state.historyLength - 1
)

const projectId = computed(() => String(route.query.projectId ?? "").trim())
const recoveryStorageKey = computed(() =>
  projectId.value ? `${recoveryStoragePrefix}${projectId.value}` : ""
)
const rootGraphSignature = computed(() => flowStore.graphEditorSignature())
const canRunFlow = computed(() => Boolean(flowStore.state.flowId.trim()))
const canRefreshStatus = computed(() => Boolean(flowStore.state.flowId.trim()))
const canListRuns = computed(() => Boolean(flowStore.state.flowId.trim()))
const canCancelRun = computed(
  () => Boolean(flowStore.state.flowId.trim()) && Boolean(flowStore.state.statusRunId.trim())
)
const flowStatusLabel = computed(() => {
  const status = flowStore.state.lastStatus.status.trim()
  return status ? t(flowStatusLabelKey(status)) : ""
})
const currentRunIdLabel = computed(() =>
  t("Current Run ID") + ": " + (flowStore.state.statusRunId.trim() || t("No run yet."))
)
const bodySessionDirty = computed(() => {
  const session = bodyEditorSession.value
  if (!session) return false
  return graphSignatureOf(session.snapshot) !== session.syncedSignature
})
const dirty = computed(() => !loading.value && (rootGraphSignature.value !== lastSavedSignature.value || bodySessionDirty.value))

const effectiveExecutorNode = computed(() => {
  const rawTarget = String(flowStore.state.targetId ?? "").trim()
  if (rawTarget) {
    const parsed = Number.parseInt(rawTarget, 10)
    if (Number.isFinite(parsed) && parsed > 0) {
      return Math.trunc(parsed)
    }
  }
  const currentHubId = Number(sessionStore.auth.hubId || flowStore.state.hubId || 0)
  return Number.isFinite(currentHubId) && currentHubId > 0 ? Math.trunc(currentHubId) : 0
})

const findCapabilityRouteForDraft = (node: FlowNodeDraft | null): ExecCapabilityRoute | null => {
  if (!node || node.kind !== "call") {
    return null
  }
  const method = node.method.trim()
  if (!method) {
    return null
  }
  const expectedProvider = node.target > 0 ? Math.trunc(node.target) : effectiveExecutorNode.value
  if (!expectedProvider) {
    return null
  }
  return (
    flowStore.state.execCapabilities.find(
      (route) => route.method === method && route.providerNode === expectedProvider
    ) ?? null
  )
}

const resolveDraftVisualSchema = (node: FlowNodeDraft | null) => {
  if (!node || node.kind !== "call") {
    return null
  }
  const route = findCapabilityRouteForDraft(node)
  return resolveMethodVisualSchema(
    node.method,
    route
      ? {
          method: route.method,
          version: route.version,
          inputSchema: route.inputSchema
        }
      : null
  )
}

const buildDraftVisualForm = (node: FlowNodeDraft | null): NodeVisualFormModel | null => {
  if (!node || node.kind !== "call") {
    return null
  }
  return buildNodeVisualFormModel({
    kind: node.kind,
    method: node.method,
    argsTemplate: node.argsTemplate,
    inputs: node.inputs,
    schema: resolveDraftVisualSchema(node)
  })
}

const parseArgsTemplateObject = (node: FlowNodeDraft) => {
  let parsed: unknown
  try {
    parsed = JSON.parse(node.argsTemplate.trim() || "{}")
  } catch {
    throw new Error(t("Node {nodeId} args template must be valid JSON.", { nodeId: node.id.trim() || t("Unnamed") }))
  }
  if (!isJSONObject(parsed)) {
    throw new Error(t("Node {nodeId} args template must be a JSON object.", { nodeId: node.id.trim() || t("Unnamed") }))
  }
  return parsed
}

const exportLooseSpecFromNodeDraft = (node: FlowNodeDraft) => {
  const draft = exportLooseGraphDraftFromEditorState({
    nodes: [JSON.parse(JSON.stringify(node)) as FlowNodeDraft],
    edges: [],
    selectedNodeIndex: 0,
    selectedEdgeIndex: -1
  })
  return draft.nodes[0]?.spec ?? {}
}

const syncDraftSpecJsonFromLooseSpec = (node: FlowNodeDraft) => {
  node.specJson = formatJSONText(exportLooseSpecFromNodeDraft(node))
}

const reconcileDraftToSchema = (node: FlowNodeDraft, schema: NonNullable<ReturnType<typeof resolveDraftVisualSchema>>) => {
  const allowedPointers = new Set(schema.fields.map((field) => field.pointer))
  const argsDoc = parseArgsTemplateObject(node)
  let nextDoc: Record<string, unknown> = {}
  for (const field of schema.fields) {
    const current = readValueAtPointer(argsDoc, field.pointer)
    if (!current.found) {
      continue
    }
    nextDoc = setLiteralFieldValueInDoc(nextDoc, field.pointer, current.value)
  }
  node.argsTemplate = formatJSONText(nextDoc)
  node.inputs = node.inputs.filter((binding) => {
    const to = binding.to.trim()
    return !to || allowedPointers.has(to)
  })
}

const applySchemaDefaultsToDraft = (node: FlowNodeDraft, schema: NonNullable<ReturnType<typeof resolveDraftVisualSchema>>) => {
  const argsDoc = parseArgsTemplateObject(node)
  let nextDoc = argsDoc as Record<string, unknown>
  let changed = false
  for (const field of schema.fields) {
    if (field.defaultValue === undefined) {
      continue
    }
    const current = readValueAtPointer(nextDoc, field.pointer)
    if (current.found) {
      continue
    }
    nextDoc = setLiteralFieldValueInDoc(nextDoc, field.pointer, field.defaultValue)
    changed = true
  }
  if (changed) {
    node.argsTemplate = formatJSONText(nextDoc)
  }
}

const listBodyAncestorNodeIds = (nodeId: string) => {
  const session = bodyEditorSession.value
  const trimmed = String(nodeId ?? "").trim()
  if (!session || !trimmed) {
    return []
  }

  const parents = new Map<string, string[]>()
  for (const edge of session.snapshot.edges) {
    const from = edge.from.trim()
    const to = edge.to.trim()
    if (!from || !to) {
      continue
    }
    const list = parents.get(to)
    if (list) {
      list.push(from)
    } else {
      parents.set(to, [from])
    }
  }

  const allowed = new Set<string>()
  const stack = [...(parents.get(trimmed) ?? [])]
  while (stack.length) {
    const current = stack.pop()
    if (!current || allowed.has(current)) {
      continue
    }
    allowed.add(current)
    stack.push(...(parents.get(current) ?? []))
  }

  return session.snapshot.nodes.map((node) => node.id.trim()).filter((id) => allowed.has(id))
}

const activeCallNode = computed(() => {
  const node = bodyEditorActive.value ? bodySelectedNode.value : selectedNode.value
  return node?.kind === "call" ? node : null
})

const selectedTargetLabel = computed(() => {
  const node = selectedNode.value
  if (!node) return t("No target selected.")
  if (node.kind !== "call") {
    return node.kind === "set_var"
      ? t("Set var nodes write a flow-local variable for the current run.")
      : t("Compose nodes build local JSON output and do not call a capability.")
  }
  if (node.target > 0) {
    return t("Remote provider node {nodeId}", { nodeId: node.target })
  }
  if (effectiveExecutorNode.value > 0) {
    return t("Current executor node {nodeId}", { nodeId: effectiveExecutorNode.value })
  }
  return t("Current executor")
})

const bodySelectedTargetLabel = computed(() => {
  const node = bodySelectedNode.value
  if (!node || node.kind !== "call") return t("No target selected.")
  if (node.target > 0) {
    return t("Remote provider node {nodeId}", { nodeId: node.target })
  }
  if (effectiveExecutorNode.value > 0) {
    return t("Current executor node {nodeId}", { nodeId: effectiveExecutorNode.value })
  }
  return t("Current executor")
})

const activeTargetLabel = computed(() =>
  bodyEditorActive.value && bodySelectedNode.value?.kind === "call"
    ? bodySelectedTargetLabel.value
    : selectedTargetLabel.value
)

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

const capabilitySearchText = (route: ExecCapabilityRoute) => {
  const tags = Object.entries(route.tags).flatMap(([key, value]) => [key, value, `${key}:${value}`])
  return [
    route.method,
    route.label,
    route.providerNode,
    route.viaNode,
    route.version,
    ...route.permissions,
    ...tags,
    route.inputSchema ? "schema" : ""
  ]
    .join(" ")
    .toLowerCase()
}

const capabilityMatchScore = (route: ExecCapabilityRoute, query: string) => {
  const normalizedMethod = route.method.toLowerCase()
  if (normalizedMethod === query) {
    return 4
  }
  if (normalizedMethod.startsWith(query)) {
    return 3
  }
  if (normalizedMethod.includes(query)) {
    return 2
  }
  return capabilitySearchText(route).includes(query) ? 1 : -1
}

const filteredCapabilities = computed(() => {
  const query = methodSearch.value.trim().toLowerCase()
  if (!query) {
    return capabilityOptions.value
  }
  return capabilityOptions.value
    .map((route) => ({ route, score: capabilityMatchScore(route, query) }))
    .filter((entry) => entry.score >= 0)
    .sort((left, right) => right.score - left.score)
    .map((entry) => entry.route)
})

const selectedCapabilityKey = computed(() => {
  const node = activeCallNode.value
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

const selectedNodeValidation = computed(() =>
  selectedNode.value ? flowStore.getNodeValidation(selectedNode.value.id) : []
)

const ancestorNodeOptions = computed(() =>
  selectedNode.value ? flowStore.listAncestorNodeIds(selectedNode.value.id) : []
)

const bindableAncestorNodeOptions = computed(() =>
  selectedNode.value ? flowStore.listBindableAncestorNodeIds(selectedNode.value.id) : []
)

const bodyBindableAncestorNodeOptions = computed(() =>
  bodySelectedNode.value ? listBodyAncestorNodeIds(bodySelectedNode.value.id) : []
)

const selectedCallVisualForm = computed<NodeVisualFormModel | null>(() => {
  const node = selectedNode.value
  if (!node || node.kind !== "call") return null
  return flowStore.getNodeVisualForm(node.id)
})

const bodySelectedCallVisualForm = computed<NodeVisualFormModel | null>(() => buildDraftVisualForm(bodySelectedNode.value))

const activeCallVisualForm = computed<NodeVisualFormModel | null>(() =>
  bodyEditorActive.value ? bodySelectedCallVisualForm.value : selectedCallVisualForm.value
)

const activeBindableAncestorNodeOptions = computed(() =>
  bodyEditorActive.value ? bodyBindableAncestorNodeOptions.value : bindableAncestorNodeOptions.value
)
const allowLoopBindingSources = computed(() => bodyEditorActive.value)

const selectedNodeOutputSchemaText = computed(() => {
  const node = selectedNode.value
  return node ? flowStore.getNodeOutputSchemaText(node.id) : ""
})

const activeBindingField = computed<VisualFieldModel | null>(() => {
  const form = activeCallVisualForm.value
  if (!form) return null
  return form.fields.find((field) => field.schema.pointer === activeBindingFieldPointer.value) ?? null
})

const formatTimestamp = (value: string) => {
  const raw = String(value ?? "").trim()
  if (!raw) return "-"
  const parsed = Date.parse(raw)
  if (Number.isNaN(parsed)) return raw
  return new Date(parsed).toLocaleString()
}

const lastSavedLabel = computed(() => {
  if (!lastSavedAt.value) return ""
  return t("Last saved {time}", { time: formatTimestamp(lastSavedAt.value) })
})

const graphSignatureOf = (snapshot: FlowGraphEditorState) =>
  JSON.stringify({
    nodes: Array.isArray(snapshot?.nodes) ? snapshot.nodes : [],
    edges: Array.isArray(snapshot?.edges) ? snapshot.edges : []
  })

const cloneGraphEditorState = (snapshot: FlowGraphEditorState): FlowGraphEditorState =>
  JSON.parse(
    JSON.stringify({
      nodes: Array.isArray(snapshot?.nodes) ? snapshot.nodes : [],
      edges: Array.isArray(snapshot?.edges) ? snapshot.edges : [],
      selectedNodeIndex: Number.isInteger(snapshot?.selectedNodeIndex) ? Number(snapshot.selectedNodeIndex) : -1,
      selectedEdgeIndex: Number.isInteger(snapshot?.selectedEdgeIndex) ? Number(snapshot.selectedEdgeIndex) : -1
    })
  ) as FlowGraphEditorState

const normalizeFlowId = (value: unknown) => String(value ?? "").trim().toLowerCase()

const collectSubflowTargetsFromGraph = (graph: unknown, out = new Set<string>()) => {
  const nodes = Array.isArray((graph as any)?.nodes) ? (graph as any).nodes : []
  for (const node of nodes) {
    const kind = String(node?.kind ?? "").trim().toLowerCase()
    const spec = isJSONObject(node?.spec) ? node.spec : null
    if (!spec) {
      continue
    }
    if (kind === "subflow") {
      const flowId = normalizeFlowId(spec.flow_id)
      if (flowId) {
        out.add(flowId)
      }
      continue
    }
    if (kind === "foreach" && isJSONObject(spec.body)) {
      collectSubflowTargetsFromGraph(spec.body, out)
    }
  }
  return out
}

const buildLocalSubflowDependencyMap = (
  projects: FlowProjectRecord[],
  current: { projectId: string; flowId: string; graph: unknown }
) => {
  const dependencies = new Map<string, Set<string>>()
  for (const project of projects) {
    const isCurrent = project.projectId === current.projectId
    const flowId = normalizeFlowId(isCurrent ? current.flowId : project.flowId)
    if (!flowId) {
      continue
    }
    dependencies.set(flowId, collectSubflowTargetsFromGraph(isCurrent ? current.graph : project.graph))
  }
  if (!dependencies.has(normalizeFlowId(current.flowId))) {
    dependencies.set(normalizeFlowId(current.flowId), collectSubflowTargetsFromGraph(current.graph))
  }
  return dependencies
}

const findRecursiveSubflowChain = (startFlowId: string, dependencies: Map<string, Set<string>>) => {
  const visited = new Set<string>()
  const visiting = new Set<string>()
  const stack: string[] = []

  const dfs = (flowId: string): string[] | null => {
    if (visiting.has(flowId)) {
      const startIndex = stack.indexOf(flowId)
      return startIndex >= 0 ? [...stack.slice(startIndex), flowId] : [flowId, flowId]
    }
    if (visited.has(flowId)) {
      return null
    }
    visiting.add(flowId)
    stack.push(flowId)
    for (const targetFlowId of dependencies.get(flowId) ?? []) {
      const cycle = dfs(targetFlowId)
      if (cycle) {
        return cycle
      }
    }
    stack.pop()
    visiting.delete(flowId)
    visited.add(flowId)
    return null
  }

  return dfs(startFlowId)
}

const foreachBodySeedSpec = (kind: FlowNodeKind): Record<string, any> => {
  switch (kind) {
    case "compose":
      return { template: {} }
    case "transform":
      return { expr: { literal: null } }
    case "set_var":
      return { name: "", template: null }
    case "branch":
      return { cases: [] }
    case "foreach":
      return {
        source: { kind: "trigger" },
        required: true,
        body: { nodes: [], edges: [] },
        result_node_id: ""
      }
    case "subflow":
      return { flow_id: "", input_template: {} }
    default:
      return { method: "", args_template: {} }
  }
}

const bodyNodePosition = (index: number) => {
  const col = index % 4
  const row = Math.floor(index / 4)
  return { x: col * 240, y: row * 160 }
}

const createBodyNodeDraft = (id: string, kind: FlowNodeKind, index: number): FlowNodeDraft => {
  const node =
    createGraphEditorStateFromDraft({
      nodes: [
        {
          id,
          kind,
          allow_fail: false,
          retry: 1,
          timeout_ms: 3000,
          spec: foreachBodySeedSpec(kind)
        }
      ],
      edges: []
    }, { allowLoopSources: true }).nodes[0] ?? null
  if (!node) {
    throw new Error(t("Failed to create body node draft."))
  }
  const pos = bodyNodePosition(index)
  node.x = pos.x
  node.y = pos.y
  return node
}

const normalizeBodySessionSnapshot = (snapshot: FlowGraphEditorState): FlowGraphEditorState => {
  const next = cloneGraphEditorState(snapshot)
  next.selectedNodeIndex =
    next.selectedNodeIndex >= 0 && next.selectedNodeIndex < next.nodes.length ? next.selectedNodeIndex : -1
  next.selectedEdgeIndex =
    next.selectedEdgeIndex >= 0 && next.selectedEdgeIndex < next.edges.length ? next.selectedEdgeIndex : -1
  return next
}

const findBodySessionParentNode = () => {
  const session = bodyEditorSession.value
  if (!session) {
    throw new Error(t("Foreach body editor is not active."))
  }
  const parent = flowStore.state.nodes.find((node) => node.id.trim() === session.parentNodeId.trim()) ?? null
  if (!parent || parent.kind !== "foreach") {
    throw new Error(t("The parent foreach node is no longer available."))
  }
  if (parent.specEditorMode !== "form") {
    throw new Error(t("Switch the foreach node back to ordinary mode before editing its body graph visually."))
  }
  return parent
}

const serializeBodySessionGraph = (snapshot: FlowGraphEditorState) =>
  JSON.stringify(exportLooseGraphDraftFromEditorState(snapshot), null, 2)

const syncBodySessionToParent = ({ commitHistory = false, silent = false }: { commitHistory?: boolean; silent?: boolean } = {}) => {
  const session = bodyEditorSession.value
  if (!session) return true
  try {
    const parent = findBodySessionParentNode()
    const nextJson = serializeBodySessionGraph(session.snapshot)
    const changed = parent.foreachBodyJson.trim() !== nextJson.trim()
    parent.foreachBodyJson = nextJson
    session.syncedSignature = graphSignatureOf(session.snapshot)
    bodySessionError.value = ""
    if (changed && commitHistory) {
      flowStore.commitHistory()
    }
    return true
  } catch (err) {
    bodySessionError.value = String((err as Error)?.message ?? err ?? t("Failed to sync foreach body graph."))
    if (!silent) {
      console.warn(err)
      toast.errorOf(err, t("Failed to sync foreach body graph."))
    }
    return false
  }
}

const restoreBodyEditorSession = (record: FlowEditorBodySessionRecord | null | undefined) => {
  if (!record?.parentNodeId || !record.snapshot) {
    bodyEditorSession.value = null
    bodyNodeIdDraft.value = ""
    bodySessionError.value = ""
    return
  }
  const snapshot = normalizeBodySessionSnapshot(record.snapshot)
  bodyEditorSession.value = {
    parentNodeId: String(record.parentNodeId ?? "").trim(),
    snapshot,
    syncedSignature: String(record.syncedSignature ?? "").trim() || graphSignatureOf(snapshot)
  }
  bodyNodeIdDraft.value = snapshot.nodes[snapshot.selectedNodeIndex]?.id ?? ""
  bodySessionError.value = ""
}

const openForeachBodyEditor = () => {
  const node = selectedNode.value
  if (!node || node.kind !== "foreach") return
  if (node.specEditorMode !== "form") {
    toast.error(t("Switch the foreach node back to ordinary mode before opening the visual body editor."))
    return
  }
  try {
    const body = JSON.parse(node.foreachBodyJson.trim() || "{\"nodes\":[],\"edges\":[]}")
    if (!body || typeof body !== "object" || Array.isArray(body)) {
      throw new Error(t("Foreach body must be a JSON object."))
    }
    if (!Array.isArray(body.nodes) || !Array.isArray(body.edges)) {
      throw new Error(t("Foreach body must include nodes and edges arrays."))
    }
    const snapshot = normalizeBodySessionSnapshot(createGraphEditorStateFromDraft(body, { allowLoopSources: true }))
    bodyEditorSession.value = {
      parentNodeId: node.id,
      snapshot,
      syncedSignature: graphSignatureOf(snapshot)
    }
    bodyNodeIdDraft.value = snapshot.nodes[snapshot.selectedNodeIndex]?.id ?? ""
    bodySessionError.value = ""
    closeMethodDialog()
    closeFieldBindingDialog()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to open foreach body graph."))
  }
}

const closeForeachBodyEditor = () => {
  if (bodySessionDirty.value && !syncBodySessionToParent({ commitHistory: true })) {
    return
  }
  bodyEditorSession.value = null
  bodyNodeIdDraft.value = ""
  bodySessionError.value = ""
}

const setBodySelectedNodeSpecMode = (mode: "form" | "json") => {
  const session = bodyEditorSession.value
  const node = bodySelectedNode.value
  const index = session?.snapshot.selectedNodeIndex ?? -1
  if (!session || !node || index < 0) {
    return
  }
  if (node.specEditorMode === mode) {
    return
  }

  if (mode === "json") {
    syncDraftSpecJsonFromLooseSpec(node)
    node.specEditorMode = "json"
    syncBodySessionToParent({ commitHistory: true })
    return
  }

  let parsedSpec: unknown
  try {
    parsedSpec = JSON.parse(node.specJson.trim() || "{}")
  } catch {
    throw new Error(t("Node {nodeId} advanced spec must be valid JSON.", { nodeId: node.id.trim() || t("Unnamed") }))
  }
  if (!isJSONObject(parsedSpec)) {
    throw new Error(t("Node {nodeId} advanced spec must be a JSON object.", { nodeId: node.id.trim() || t("Unnamed") }))
  }

  const nextSnapshot = createGraphEditorStateFromDraft({
    nodes: [
      {
        id: node.id,
        kind: node.kind,
        allow_fail: node.allowFail,
        retry: node.retry,
        timeout_ms: node.timeoutMs,
        spec: {
          ...(parsedSpec as Record<string, unknown>),
          _ui: { x: Math.round(Number(node.x || 0)), y: Math.round(Number(node.y || 0)) }
        }
      }
    ],
    edges: []
  }, { allowLoopSources: true })
  const next = nextSnapshot.nodes[0] ?? null
  if (!next || next.specEditorMode !== "form") {
    throw new Error(
      t("Body node kind {kind} advanced spec contains fields that ordinary mode cannot represent yet.", {
        kind: node.kind
      })
    )
  }
  next.x = node.x
  next.y = node.y
  session.snapshot.nodes.splice(index, 1, next)
  bodyNodeIdDraft.value = next.id
  syncBodySessionToParent({ commitHistory: true })
}

const setBodyNodeFieldLiteralValue = (node: FlowNodeDraft, pointer: string, value: unknown) => {
  validateJSONPointer(pointer, t("Field pointer must be a valid JSON Pointer."))
  const argsDoc = parseArgsTemplateObject(node)
  const nextDoc = setLiteralFieldValueInDoc(argsDoc, pointer, value)
  node.argsTemplate = formatJSONText(nextDoc)
}

const clearBodyNodeFieldBinding = (node: FlowNodeDraft, pointer: string) => {
  validateJSONPointer(pointer, t("Field pointer must be a valid JSON Pointer."))
  node.inputs = clearBindingForPointer(node.inputs, pointer).map((binding) => ({
    ...binding,
    sourceKind: binding.sourceKind as FlowBindingSourceKind
  }))
}

const setBodyNodeFieldBinding = (node: FlowNodeDraft, pointer: string, source: VisualBindingSource) => {
  validateJSONPointer(pointer, t("Field pointer must be a valid JSON Pointer."))

  if (source.kind === "node_result") {
    const sourceNodeId = source.nodeId.trim()
    if (!sourceNodeId) {
      throw new Error(t("Source node is required."))
    }
    const allowedAncestors = new Set(listBodyAncestorNodeIds(node.id))
    if (!allowedAncestors.has(sourceNodeId)) {
      throw new Error(t("Source node must be an ancestor."))
    }
    validateJSONPointer(source.path, t("Result path must be a valid JSON Pointer."))
  } else if (source.kind === "trigger") {
    validateJSONPointer(source.path, t("Trigger path must be a valid JSON Pointer."))
  } else if (source.kind === "flow_meta") {
    if (source.field !== "flow_id") {
      throw new Error(t("Flow meta field must be flow_id."))
    }
  } else if (source.kind === "run_meta") {
    if (source.field !== "run_id") {
      throw new Error(t("Run meta field must be run_id."))
    }
  } else if (source.kind === "loop_item") {
    validateJSONPointer(source.path, t("Loop item path must be a valid JSON Pointer."))
  } else if (source.kind === "loop_index") {
    if (source.path?.trim()) {
      throw new Error(t("Loop index does not accept a JSON Pointer path."))
    }
  } else if (source.kind === "flow_var") {
    source.name = assertValidFlowLocalVarName(
      normalizeFlowLocalVarName(source.name),
      t("Flow local var name is required."),
      t("Flow local var name is invalid.")
    )
    validateJSONPointer(source.path, t("Flow local var path must be a valid JSON Pointer."))
  }

  node.inputs = setBindingForPointer(node.inputs, pointer, source).map((binding) => ({
    ...binding,
    sourceKind: binding.sourceKind as FlowBindingSourceKind
  }))
}

const applyBodyCallCapability = (key: string) => {
  const node = bodySelectedNode.value
  if (!node || node.kind !== "call") {
    throw new Error(t("Select a call node first."))
  }
  const route = flowStore.state.execCapabilities.find((item) => item.key === String(key).trim()) ?? null
  if (!route) {
    throw new Error(t("Capability not found in current list."))
  }
  const methodChanged = node.method !== route.method
  const nextSchema = resolveMethodVisualSchema(route.method, {
    method: route.method,
    version: route.version,
    inputSchema: route.inputSchema
  })
  node.method = route.method
  node.target = flowStore.normalizeCallTarget(route.providerNode)
  if (!String(node.argsTemplate ?? "").trim()) {
    node.argsTemplate = "{}"
  }
  if (methodChanged && node.specEditorMode === "form" && nextSchema) {
    reconcileDraftToSchema(node, nextSchema)
  }
  if (nextSchema) {
    applySchemaDefaultsToDraft(node, nextSchema)
  }
  if (node.specEditorMode === "json") {
    syncDraftSpecJsonFromLooseSpec(node)
  }
  if (!syncBodySessionToParent({ commitHistory: true })) {
    throw new Error(bodySessionError.value || t("Failed to sync foreach body graph."))
  }
}

const clearRecoveryWriteTimer = () => {
  if (recoveryWriteTimer !== null) {
    window.clearTimeout(recoveryWriteTimer)
    recoveryWriteTimer = null
  }
}

const clearRecoveryDraft = () => {
  clearRecoveryWriteTimer()
  const storageKey = recoveryStorageKey.value
  if (!storageKey || typeof globalThis.localStorage === "undefined") return
  try {
    globalThis.localStorage.removeItem(storageKey)
  } catch (err) {
    console.warn(err)
  }
}

const readRecoveryDraft = (): FlowEditorRecoveryRecord | null => {
  const storageKey = recoveryStorageKey.value
  if (!storageKey || typeof globalThis.localStorage === "undefined") return null
  try {
    const raw = globalThis.localStorage.getItem(storageKey)
    if (!raw) return null
    const parsed = JSON.parse(raw)
    if (
      !parsed ||
      typeof parsed !== "object" ||
      String(parsed.projectId ?? "").trim() !== projectId.value ||
      typeof parsed.snapshot !== "object"
    ) {
      clearRecoveryDraft()
      return null
    }
    return {
      version: 1,
      projectId: String(parsed.projectId ?? "").trim(),
      baseSignature: String(parsed.baseSignature ?? ""),
      savedAt: String(parsed.savedAt ?? "").trim(),
      snapshot: parsed.snapshot as FlowGraphEditorState,
      bodySession:
        parsed.bodySession &&
        typeof parsed.bodySession === "object" &&
        typeof parsed.bodySession.snapshot === "object" &&
        String(parsed.bodySession.parentNodeId ?? "").trim()
          ? {
              parentNodeId: String(parsed.bodySession.parentNodeId ?? "").trim(),
              snapshot: parsed.bodySession.snapshot as FlowGraphEditorState,
              syncedSignature: String(parsed.bodySession.syncedSignature ?? "").trim()
            }
          : null
    }
  } catch (err) {
    console.warn(err)
    return null
  }
}

const writeRecoveryDraft = () => {
  if (!dirty.value || loading.value || typeof globalThis.localStorage === "undefined") return
  const storageKey = recoveryStorageKey.value
  if (!storageKey) return
  try {
    const bodySession = bodyEditorSession.value
    const payload: FlowEditorRecoveryRecord = {
      version: 1,
      projectId: projectId.value,
      baseSignature: lastSavedSignature.value,
      savedAt: new Date().toISOString(),
      snapshot: flowStore.exportGraphEditorState(),
      ...(bodySession
        ? {
            bodySession: {
              parentNodeId: bodySession.parentNodeId,
              snapshot: cloneGraphEditorState(bodySession.snapshot),
              syncedSignature: bodySession.syncedSignature
            }
          }
        : {})
    }
    globalThis.localStorage.setItem(storageKey, JSON.stringify(payload))
  } catch (err) {
    console.warn(err)
  }
}

const scheduleRecoveryDraftWrite = () => {
  clearRecoveryWriteTimer()
  recoveryWriteTimer = window.setTimeout(() => {
    recoveryWriteTimer = null
    writeRecoveryDraft()
  }, 400)
}

const updateSavedBaseline = (updatedAt: string) => {
  lastSavedSignature.value = rootGraphSignature.value
  lastSavedAt.value = String(updatedAt ?? "").trim() || new Date().toISOString()
}

const maybeRestoreRecoveryDraft = (project: FlowProjectRecord) => {
  const recovery = readRecoveryDraft()
  if (!recovery) return
  if (recovery.baseSignature !== lastSavedSignature.value) {
    clearRecoveryDraft()
    return
  }

  const recoveryGraphSignature = graphSignatureOf(recovery.snapshot)
  const recoveryBodyDirty =
    Boolean(recovery.bodySession) &&
    graphSignatureOf(recovery.bodySession!.snapshot) !== String(recovery.bodySession!.syncedSignature ?? "")
  if (recoveryGraphSignature === lastSavedSignature.value && !recoveryBodyDirty) {
    clearRecoveryDraft()
    return
  }

  const restore = window.confirm(
    t("A local draft from {time} was found for this project. Restore it now?", {
      time: formatTimestamp(recovery.savedAt || project.updatedAt)
    })
  )
  if (!restore) {
    clearRecoveryDraft()
    return
  }

  try {
    flowStore.loadGraphEditorState(recovery.snapshot)
    restoreBodyEditorSession(recovery.bodySession)
    toast.info(
      t("Recovered local draft from {time}. Save project to persist it.", {
        time: formatTimestamp(recovery.savedAt || project.updatedAt)
      })
    )
  } catch (err) {
    console.warn(err)
    clearRecoveryDraft()
    toast.error(t("Failed to restore local draft."))
  }
}

const resetFieldBindingDraft = (source?: VisualBindingSource | null) => {
  if (source?.kind === "node_result") {
    fieldBindingDraft.sourceKind = "node_result"
    fieldBindingDraft.nodeId = source.nodeId
    fieldBindingDraft.path = source.path
    fieldBindingDraft.field = "flow_id"
    fieldBindingDraft.name = ""
    fieldBindingDraft.required = source.required
    return
  }
  if (source?.kind === "trigger") {
    fieldBindingDraft.sourceKind = "trigger"
    fieldBindingDraft.nodeId = ""
    fieldBindingDraft.path = source.path
    fieldBindingDraft.field = "flow_id"
    fieldBindingDraft.name = ""
    fieldBindingDraft.required = source.required
    return
  }
  if (source?.kind === "flow_meta") {
    fieldBindingDraft.sourceKind = "flow_meta"
    fieldBindingDraft.nodeId = ""
    fieldBindingDraft.path = ""
    fieldBindingDraft.field = "flow_id"
    fieldBindingDraft.name = ""
    fieldBindingDraft.required = source.required
    return
  }
  if (source?.kind === "run_meta") {
    fieldBindingDraft.sourceKind = "run_meta"
    fieldBindingDraft.nodeId = ""
    fieldBindingDraft.path = ""
    fieldBindingDraft.field = "run_id"
    fieldBindingDraft.name = ""
    fieldBindingDraft.required = source.required
    return
  }
  if (source?.kind === "loop_item") {
    fieldBindingDraft.sourceKind = "loop_item"
    fieldBindingDraft.nodeId = ""
    fieldBindingDraft.path = source.path
    fieldBindingDraft.field = ""
    fieldBindingDraft.name = ""
    fieldBindingDraft.required = source.required
    return
  }
  if (source?.kind === "loop_index") {
    fieldBindingDraft.sourceKind = "loop_index"
    fieldBindingDraft.nodeId = ""
    fieldBindingDraft.path = ""
    fieldBindingDraft.field = ""
    fieldBindingDraft.name = ""
    fieldBindingDraft.required = source.required
    return
  }
  if (source?.kind === "flow_var") {
    fieldBindingDraft.sourceKind = "flow_var"
    fieldBindingDraft.nodeId = ""
    fieldBindingDraft.path = source.path
    fieldBindingDraft.field = ""
    fieldBindingDraft.name = source.name
    fieldBindingDraft.required = source.required
    return
  }

  fieldBindingDraft.sourceKind = activeBindableAncestorNodeOptions.value.length ? "node_result" : "trigger"
  fieldBindingDraft.nodeId = activeBindableAncestorNodeOptions.value[0] ?? ""
  fieldBindingDraft.path = ""
  fieldBindingDraft.field = "flow_id"
  fieldBindingDraft.name = ""
  fieldBindingDraft.required = false
}

const closeFieldBindingDialog = () => {
  fieldBindingDialogOpen.value = false
  activeBindingFieldPointer.value = ""
}

const openFieldBindingDialog = (field: VisualFieldModel) => {
  if (field.schema.bindable === false) return
  activeBindingFieldPointer.value = field.schema.pointer
  resetFieldBindingDraft(field.state.binding)
  fieldBindingDialogOpen.value = true
}

const onFieldBindingSourceKindChange = (sourceKind: string) => {
  fieldBindingDraft.sourceKind =
    sourceKind === "node_result" ||
    sourceKind === "flow_meta" ||
    sourceKind === "run_meta" ||
    (allowLoopBindingSources.value && sourceKind === "loop_item") ||
    (allowLoopBindingSources.value && sourceKind === "loop_index") ||
    sourceKind === "flow_var"
      ? sourceKind
      : "trigger"
  if (fieldBindingDraft.sourceKind === "node_result") {
    fieldBindingDraft.nodeId = activeBindableAncestorNodeOptions.value[0] ?? ""
    fieldBindingDraft.path = ""
    fieldBindingDraft.field = "flow_id"
    fieldBindingDraft.name = ""
    return
  }
  if (fieldBindingDraft.sourceKind === "trigger") {
    fieldBindingDraft.nodeId = ""
    fieldBindingDraft.path = ""
    fieldBindingDraft.field = "flow_id"
    fieldBindingDraft.name = ""
    return
  }
  if (fieldBindingDraft.sourceKind === "flow_var") {
    fieldBindingDraft.nodeId = ""
    fieldBindingDraft.path = ""
    fieldBindingDraft.field = ""
    fieldBindingDraft.name = ""
    return
  }
  if (fieldBindingDraft.sourceKind === "loop_item") {
    fieldBindingDraft.nodeId = ""
    fieldBindingDraft.path = ""
    fieldBindingDraft.field = ""
    fieldBindingDraft.name = ""
    return
  }
  if (fieldBindingDraft.sourceKind === "loop_index") {
    fieldBindingDraft.nodeId = ""
    fieldBindingDraft.path = ""
    fieldBindingDraft.field = ""
    fieldBindingDraft.name = ""
    return
  }
  fieldBindingDraft.nodeId = ""
  fieldBindingDraft.path = ""
  fieldBindingDraft.field = fieldBindingDraft.sourceKind === "run_meta" ? "run_id" : "flow_id"
  fieldBindingDraft.name = ""
}

const buildVisualBindingSource = (): VisualBindingSource => {
  switch (fieldBindingDraft.sourceKind) {
    case "node_result":
      return {
        kind: "node_result",
        nodeId: fieldBindingDraft.nodeId,
        path: fieldBindingDraft.path,
        required: fieldBindingDraft.required
      }
    case "flow_meta":
      return {
        kind: "flow_meta",
        field: "flow_id",
        required: fieldBindingDraft.required
      }
    case "run_meta":
      return {
        kind: "run_meta",
        field: "run_id",
        required: fieldBindingDraft.required
      }
    case "loop_item":
      return {
        kind: "loop_item",
        path: fieldBindingDraft.path,
        required: fieldBindingDraft.required
      }
    case "loop_index":
      return {
        kind: "loop_index",
        required: fieldBindingDraft.required
      }
    case "flow_var":
      return {
        kind: "flow_var",
        name: fieldBindingDraft.name,
        path: fieldBindingDraft.path,
        required: fieldBindingDraft.required
      }
    case "trigger":
    default:
      return {
        kind: "trigger",
        path: fieldBindingDraft.path,
        required: fieldBindingDraft.required
      }
  }
}

const applyFieldBinding = () => {
  const node = activeCallNode.value
  const field = activeBindingField.value
  if (!node || !field) return
  try {
    if (bodyEditorActive.value) {
      setBodyNodeFieldBinding(node, field.schema.pointer, buildVisualBindingSource())
      if (!syncBodySessionToParent({ commitHistory: true })) {
        throw new Error(bodySessionError.value || t("Failed to sync foreach body graph."))
      }
    } else {
      flowStore.setFieldBinding(node.id, field.schema.pointer, buildVisualBindingSource())
    }
    closeFieldBindingDialog()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to save field binding."))
  }
}

const clearVisualFieldBinding = (pointer?: string) => {
  const node = activeCallNode.value
  const targetPointer = pointer ?? activeBindingFieldPointer.value
  if (!node || !targetPointer) return
  try {
    if (bodyEditorActive.value) {
      clearBodyNodeFieldBinding(node, targetPointer)
      if (!syncBodySessionToParent({ commitHistory: true })) {
        throw new Error(bodySessionError.value || t("Failed to sync foreach body graph."))
      }
    } else {
      flowStore.clearFieldBinding(node.id, targetPointer)
    }
    if (!pointer) {
      closeFieldBindingDialog()
    }
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to clear field binding."))
  }
}

const stringifyFieldDraftValue = (field: VisualFieldModel) => {
  const value = field.state.literalValue
  switch (field.schema.control) {
    case "number":
      return value === undefined || value === null ? "" : String(value)
    case "select":
      return value === undefined ? "" : JSON.stringify(value)
    case "json":
      return value === undefined ? "" : JSON.stringify(value, null, 2)
    case "switch":
      return value ? "true" : "false"
    case "textarea":
    case "text":
    default:
      return value === undefined || value === null ? "" : String(value)
  }
}

const syncFieldDrafts = (form: NodeVisualFormModel | null) => {
  for (const key of Object.keys(fieldDrafts)) {
    delete fieldDrafts[key]
  }
  if (!form?.compatibility.supported) {
    return
  }
  for (const field of form.fields) {
    fieldDrafts[field.schema.pointer] = stringifyFieldDraftValue(field)
  }
}

const parseFieldDraftValue = (field: VisualFieldModel): unknown => {
  const raw = fieldDrafts[field.schema.pointer]
  switch (field.schema.control) {
    case "number":
      return parseNumberInput(raw, {
        allowBlank: true,
        blankValue: undefined,
        invalidMessage: t("Field {label} must be a valid number.", { label: field.schema.label })
      })
    case "select": {
      const normalized = normalizeFormInputText(raw)
      return normalized ? JSON.parse(normalized) : undefined
    }
    case "json": {
      const trimmed = normalizeFormInputText(raw).trim()
      if (!trimmed) return undefined
      try {
        return JSON.parse(trimmed)
      } catch {
        throw new Error(t("Field {label} must be valid JSON.", { label: field.schema.label }))
      }
    }
    case "textarea":
    case "text":
    default:
      return normalizeFormInputText(raw)
  }
}

const commitFieldLiteralValue = (field: VisualFieldModel) => {
  const node = activeCallNode.value
  if (!node) return
  try {
    if (bodyEditorActive.value) {
      setBodyNodeFieldLiteralValue(node, field.schema.pointer, parseFieldDraftValue(field))
      if (!syncBodySessionToParent({ commitHistory: true })) {
        throw new Error(bodySessionError.value || t("Failed to sync foreach body graph."))
      }
    } else {
      flowStore.setFieldLiteralValue(node.id, field.schema.pointer, parseFieldDraftValue(field))
    }
    const form = bodyEditorActive.value ? buildDraftVisualForm(node) : flowStore.getNodeVisualForm(node.id)
    const nextField = form?.fields.find((item) => item.schema.pointer === field.schema.pointer) ?? null
    fieldDrafts[field.schema.pointer] = nextField ? stringifyFieldDraftValue(nextField) : ""
  } catch (err) {
    console.warn(err)
    fieldDrafts[field.schema.pointer] = stringifyFieldDraftValue(field)
    toast.errorOf(err, t("Failed to update field value."))
  }
}

const setBooleanFieldLiteralValue = (field: VisualFieldModel, checked: boolean) => {
  const node = activeCallNode.value
  if (!node) return
  try {
    if (bodyEditorActive.value) {
      setBodyNodeFieldLiteralValue(node, field.schema.pointer, checked)
      if (!syncBodySessionToParent({ commitHistory: true })) {
        throw new Error(bodySessionError.value || t("Failed to sync foreach body graph."))
      }
    } else {
      flowStore.setFieldLiteralValue(node.id, field.schema.pointer, checked)
    }
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to update field value."))
  }
}

const collectGraphNodeIds = (nodes: FlowNodeDraft[]) => {
  const seen = new Set<string>()
  const ids: string[] = []
  for (const node of nodes) {
    const id = node.id.trim()
    if (!id) {
      throw new Error(t("Node ID is required."))
    }
    if (seen.has(id)) {
      throw new Error(t("Node ID must be unique."))
    }
    seen.add(id)
    ids.push(id)
  }
  return ids
}

const buildGraphAdjacency = (edges: FlowEdge[]) => {
  const next = new Map<string, string[]>()
  for (const edge of edges) {
    const from = edge.from.trim()
    const to = edge.to.trim()
    if (!from || !to) continue
    const list = next.get(from)
    if (list) {
      list.push(to)
    } else {
      next.set(from, [to])
    }
  }
  return next
}

const isReachable = (start: string, goal: string, next: Map<string, string[]>) => {
  const queue: string[] = [start]
  const visited = new Set<string>()
  while (queue.length) {
    const current = queue.shift()
    if (!current || visited.has(current)) continue
    if (current === goal) return true
    visited.add(current)
    for (const child of next.get(current) ?? []) {
      if (!visited.has(child)) {
        queue.push(child)
      }
    }
  }
  return false
}

const buildGraphTopology = (nodeIds: string[], edges: FlowEdge[]) => {
  const idSet = new Set(nodeIds)
  const nodeOrder = new Map<string, number>()
  for (const [idx, id] of nodeIds.entries()) {
    nodeOrder.set(id, idx)
  }

  const indegree = new Map<string, number>()
  const next = new Map<string, string[]>()
  const parents = new Map<string, string[]>()
  for (const id of nodeIds) {
    indegree.set(id, 0)
    next.set(id, [])
    parents.set(id, [])
  }

  for (const edge of edges) {
    const from = edge.from.trim()
    const to = edge.to.trim()
    if (!from || !to || from === to) {
      throw new Error(t("Edge endpoints are invalid."))
    }
    if (!idSet.has(from) || !idSet.has(to)) {
      throw new Error(t("Edge references unknown nodes."))
    }
    next.get(from)?.push(to)
    parents.get(to)?.push(from)
    indegree.set(to, (indegree.get(to) ?? 0) + 1)
  }

  const queue: string[] = []
  const level = new Map<string, number>()
  for (const id of nodeIds) {
    if ((indegree.get(id) ?? 0) === 0) {
      queue.push(id)
      level.set(id, 0)
    }
  }
  queue.sort((a, b) => (nodeOrder.get(a) ?? 0) - (nodeOrder.get(b) ?? 0))

  const order: string[] = []
  while (queue.length) {
    const current = queue.shift()
    if (!current) continue
    order.push(current)
    const depth = level.get(current) ?? 0
    for (const child of next.get(current) ?? []) {
      level.set(child, Math.max(level.get(child) ?? 0, depth + 1))
      const left = (indegree.get(child) ?? 0) - 1
      indegree.set(child, left)
      if (left === 0) {
        queue.push(child)
        queue.sort((a, b) => (nodeOrder.get(a) ?? 0) - (nodeOrder.get(b) ?? 0))
      }
    }
  }

  if (order.length !== nodeIds.length) {
    throw new Error(t("Flow graph must be a DAG (cycle detected)."))
  }

  return { order, level }
}

const suggestBodyNodeId = (kind: FlowNodeKind) => {
  const session = bodyEditorSession.value
  const nodes = session?.snapshot.nodes ?? []
  const prefixes: Record<FlowNodeKind, string> = {
    call: "call",
    compose: "compose",
    transform: "transform",
    set_var: "set_var",
    branch: "branch",
    foreach: "foreach",
    subflow: "subflow"
  }
  const prefix = prefixes[kind]
  const existing = new Set(nodes.map((node) => node.id.trim()))
  let index = 1
  while (existing.has(`${prefix}${index}`)) {
    index += 1
  }
  return `${prefix}${index}`
}

const clearBodySelection = () => {
  const session = bodyEditorSession.value
  if (!session) return
  session.snapshot.selectedNodeIndex = -1
  session.snapshot.selectedEdgeIndex = -1
  bodyNodeIdDraft.value = ""
}

const selectBodyNodeById = (nodeId: string) => {
  const session = bodyEditorSession.value
  if (!session) return
  const idx = session.snapshot.nodes.findIndex((node) => node.id.trim() === nodeId.trim())
  session.snapshot.selectedNodeIndex = idx
  session.snapshot.selectedEdgeIndex = -1
  bodyNodeIdDraft.value = idx >= 0 ? session.snapshot.nodes[idx]?.id ?? "" : ""
}

const selectBodyEdgeByEndpoints = (from: string, to: string) => {
  const session = bodyEditorSession.value
  if (!session) return
  const idx = session.snapshot.edges.findIndex((edge) => edge.from.trim() === from.trim() && edge.to.trim() === to.trim())
  session.snapshot.selectedEdgeIndex = idx
  session.snapshot.selectedNodeIndex = -1
  bodyNodeIdDraft.value = ""
}

const commitBodyNodeId = () => {
  const session = bodyEditorSession.value
  const node = bodySelectedNode.value
  if (!session || !node) return
  const oldId = node.id.trim()
  const nextId = bodyNodeIdDraft.value.trim()
  if (!nextId) {
    toast.error(t("Node ID is required."))
    bodyNodeIdDraft.value = oldId
    return
  }
  if (session.snapshot.nodes.some((item) => item !== node && item.id.trim() === nextId)) {
    toast.error(t("Node ID must be unique."))
    bodyNodeIdDraft.value = oldId
    return
  }
  node.id = nextId
  for (const edge of session.snapshot.edges) {
    if (edge.from.trim() === oldId) edge.from = nextId
    if (edge.to.trim() === oldId) edge.to = nextId
  }
  if (!syncBodySessionToParent({ commitHistory: true })) {
    node.id = oldId
    for (const edge of session.snapshot.edges) {
      if (edge.from.trim() === nextId) edge.from = oldId
      if (edge.to.trim() === nextId) edge.to = oldId
    }
    bodyNodeIdDraft.value = oldId
    return
  }
  bodyNodeIdDraft.value = nextId
}

const setSelectedBodyNodeKind = (kind: FlowNodeKind) => {
  const session = bodyEditorSession.value
  const node = bodySelectedNode.value
  if (!session || !node) return
  const index = session.snapshot.selectedNodeIndex
  const replacement = createBodyNodeDraft(node.id, kind, index >= 0 ? index : session.snapshot.nodes.length)
  replacement.allowFail = node.allowFail
  replacement.retry = node.retry
  replacement.timeoutMs = node.timeoutMs
  replacement.x = node.x
  replacement.y = node.y
  session.snapshot.nodes.splice(index, 1, replacement)
  bodyNodeIdDraft.value = replacement.id
  syncBodySessionToParent({ commitHistory: true })
}

const commitBodySessionHistory = () => {
  if (!bodyEditorSession.value) return
  syncBodySessionToParent({ commitHistory: true })
}

const addBodyNode = () => {
  const session = bodyEditorSession.value
  if (!session) return
  const id = nodeDraft.id.trim()
  if (!id) {
    toast.error(t("Node ID is required."))
    return
  }
  if (session.snapshot.nodes.some((node) => node.id.trim() === id)) {
    toast.error(t("Node ID must be unique."))
    return
  }
  const node = createBodyNodeDraft(id, nodeDraft.kind, session.snapshot.nodes.length)
  session.snapshot.nodes.push(node)
  session.snapshot.selectedNodeIndex = session.snapshot.nodes.length - 1
  session.snapshot.selectedEdgeIndex = -1
  bodyNodeIdDraft.value = node.id
  if (!syncBodySessionToParent({ commitHistory: true })) {
    session.snapshot.nodes.pop()
    session.snapshot.selectedNodeIndex = -1
    bodyNodeIdDraft.value = ""
    return
  }
  addNodeOpen.value = false
}

const removeBodySelectedNode = () => {
  const session = bodyEditorSession.value
  const idx = session?.snapshot.selectedNodeIndex ?? -1
  if (!session || idx < 0 || idx >= session.snapshot.nodes.length) return
  const removed = session.snapshot.nodes[idx]
  session.snapshot.nodes.splice(idx, 1)
  session.snapshot.edges = session.snapshot.edges.filter(
    (edge) => edge.from.trim() !== removed.id.trim() && edge.to.trim() !== removed.id.trim()
  )
  session.snapshot.selectedNodeIndex = -1
  session.snapshot.selectedEdgeIndex = -1
  bodyNodeIdDraft.value = ""
  syncBodySessionToParent({ commitHistory: true })
}

const removeBodySelectedEdge = () => {
  const session = bodyEditorSession.value
  const idx = session?.snapshot.selectedEdgeIndex ?? -1
  if (!session || idx < 0 || idx >= session.snapshot.edges.length) return
  session.snapshot.edges.splice(idx, 1)
  session.snapshot.selectedEdgeIndex = -1
  syncBodySessionToParent({ commitHistory: true })
}

const updateBodySelectedEdgeCase = (value: string) => {
  const session = bodyEditorSession.value
  const idx = session?.snapshot.selectedEdgeIndex ?? -1
  if (!session || idx < 0 || idx >= session.snapshot.edges.length) return
  const trimmed = String(value ?? "").trim()
  session.snapshot.edges[idx].case = trimmed || undefined
  syncBodySessionToParent({ commitHistory: true })
}

const connectBodyNodes = (from: string, to: string) => {
  const session = bodyEditorSession.value
  if (!session) return
  const fromId = from.trim()
  const toId = to.trim()
  if (!fromId || !toId || fromId === toId) {
    throw new Error(t("Edge endpoints must be different."))
  }
  if (!session.snapshot.nodes.find((node) => node.id.trim() === fromId)) {
    throw new Error(t("From node does not exist."))
  }
  if (!session.snapshot.nodes.find((node) => node.id.trim() === toId)) {
    throw new Error(t("To node does not exist."))
  }
  if (session.snapshot.edges.some((edge) => edge.from.trim() === fromId && edge.to.trim() === toId)) {
    throw new Error(t("Edge already exists."))
  }
  const next = buildGraphAdjacency(session.snapshot.edges)
  if (isReachable(toId, fromId, next)) {
    throw new Error(t("Edge would create a cycle."))
  }
  session.snapshot.edges.push({ from: fromId, to: toId })
  session.snapshot.selectedEdgeIndex = session.snapshot.edges.length - 1
  session.snapshot.selectedNodeIndex = -1
  syncBodySessionToParent({ commitHistory: true })
}

const moveBodyNode = (nodeId: string, x: number, y: number) => {
  const session = bodyEditorSession.value
  if (!session) return
  const node = session.snapshot.nodes.find((item) => item.id.trim() === nodeId.trim()) ?? null
  if (!node) return
  if (!Number.isFinite(x) || !Number.isFinite(y)) return
  node.x = x
  node.y = y
  syncBodySessionToParent({ commitHistory: true })
}

const autoLayoutBodyEditor = () => {
  const session = bodyEditorSession.value
  if (!session) return
  if (!session.snapshot.nodes.length) {
    throw new Error(t("No nodes to layout."))
  }
  const ids = collectGraphNodeIds(session.snapshot.nodes)
  const topology = buildGraphTopology(ids, session.snapshot.edges)
  const nodeOrder = new Map<string, number>()
  for (const [idx, id] of ids.entries()) {
    nodeOrder.set(id, idx)
  }
  const groups = new Map<number, string[]>()
  for (const id of topology.order) {
    const depth = topology.level.get(id) ?? 0
    const list = groups.get(depth)
    if (list) {
      list.push(id)
    } else {
      groups.set(depth, [id])
    }
  }
  const levels = [...groups.keys()].sort((a, b) => a - b)
  const maxWidth = Math.max(...levels.map((depth) => groups.get(depth)?.length ?? 0), 1)
  const xGap = 240
  const yGap = 170
  for (const depth of levels) {
    const list = (groups.get(depth) ?? []).slice()
    list.sort((left, right) => (nodeOrder.get(left) ?? 0) - (nodeOrder.get(right) ?? 0))
    const offset = ((maxWidth - list.length) * xGap) / 2
    for (const [index, id] of list.entries()) {
      const node = session.snapshot.nodes.find((item) => item.id.trim() === id) ?? null
      if (!node) continue
      node.x = Math.round(offset + index * xGap)
      node.y = Math.round(depth * yGap)
    }
  }
  syncBodySessionToParent({ commitHistory: true })
}

const closeNodeDetail = () => {
  if (bodyEditorActive.value) {
    clearBodySelection()
    return
  }
  flowStore.clearSelection()
}

const updateSelectedEdgeCase = (value: string) => {
  if (bodyEditorActive.value) {
    updateBodySelectedEdgeCase(value)
    return
  }
  try {
    flowStore.setSelectedEdgeCase(value)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to update edge case."))
  }
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
  const node = activeCallNode.value
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
  methodSearch.value = ""
}

const ensureSelectedNodeCapabilityLoaded = async () => {
  if (loading.value) {
    return
  }
  const node = activeCallNode.value
  if (!node || !node.method.trim()) {
    return
  }
  try {
    if (bodyEditorActive.value) {
      const providerNode = node.target > 0 ? Math.trunc(node.target) : effectiveExecutorNode.value
      if (!providerNode) {
        return
      }
      await flowStore.ensureCapabilityRouteLoaded(node.method, providerNode)
      return
    }
    await flowStore.ensureNodeCapabilityLoaded(node.id)
  } catch (err) {
    console.warn(err)
  }
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
  if (!activeCallNode.value) return
  methodSearch.value = ""
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
    if (bodyEditorActive.value) {
      applyBodyCallCapability(pendingCapabilityKey.value)
    } else {
      flowStore.applyCallCapability(pendingCapabilityKey.value)
    }
    closeMethodDialog()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to apply method capability."))
  }
}

const commitNodeId = () => {
  if (bodyEditorActive.value) {
    commitBodyNodeId()
    return
  }
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

const updateSelectedNodeDetailRunId = (value: string) => {
  flowStore.state.nodeDetail.requestedRunId = String(value ?? "")
}

const updateSelectedNodeDetailPath = (value: string) => {
  flowStore.state.nodeDetail.requestedPath = String(value ?? "")
}

const loadSelectedNodeDetail = async () => {
  const node = selectedNode.value
  if (!node) return
  try {
    await flowStore.loadNodeDetail(node.id, flowStore.state.nodeDetail.requestedRunId, flowStore.state.nodeDetail.requestedPath)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to load node detail."))
  }
}

const openAddNodeDialog = () => {
  nodeDraft.kind = "call"
  nodeDraft.id = bodyEditorActive.value ? suggestBodyNodeId("call") : flowStore.suggestNodeId()
  addNodeOpen.value = true
}

const addNode = () => {
  if (bodyEditorActive.value) {
    addBodyNode()
    return
  }
  try {
    flowStore.addNode(nodeDraft.id, nodeDraft.kind)
    addNodeOpen.value = false
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to add node."))
  }
}

const setSelectedNodeKind = (kind: FlowNodeKind) => {
  if (bodyEditorActive.value) {
    setSelectedBodyNodeKind(kind)
    return
  }
  const node = selectedNode.value
  if (!node) return
  try {
    flowStore.setNodeKind(node.id, kind)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to switch node kind."))
  }
}

const setSelectedNodeSpecMode = (mode: "form" | "json") => {
  if (bodyEditorActive.value) {
    try {
      setBodySelectedNodeSpecMode(mode)
    } catch (err) {
      console.warn(err)
      toast.errorOf(
        err,
        mode === "form" ? t("Failed to switch back to form mode.") : t("Failed to open advanced JSON mode.")
      )
    }
    return
  }
  const node = selectedNode.value
  if (!node) return
  try {
    flowStore.setNodeSpecEditorMode(node.id, mode)
  } catch (err) {
    console.warn(err)
    toast.errorOf(
      err,
      mode === "form" ? t("Failed to switch back to form mode.") : t("Failed to open advanced JSON mode.")
    )
  }
}

const addBinding = () => {
  const node = bodyEditorActive.value ? bodySelectedNode.value : selectedNode.value
  if (!node) return
  node.inputs.push(flowStore.createInputBinding())
  if (bodyEditorActive.value) {
    syncBodySessionToParent({ commitHistory: true })
    return
  }
  flowStore.commitHistory()
}

const removeBinding = (index: number) => {
  const node = bodyEditorActive.value ? bodySelectedNode.value : selectedNode.value
  if (!node) return
  node.inputs.splice(index, 1)
  if (bodyEditorActive.value) {
    syncBodySessionToParent({ commitHistory: true })
    return
  }
  flowStore.commitHistory()
}

const onBindingSourceKindChange = (binding: FlowInputBindingDraft, sourceKind: string) => {
  binding.sourceKind =
    sourceKind === "trigger" ||
    sourceKind === "flow_meta" ||
    sourceKind === "run_meta" ||
    sourceKind === "flow_var"
      ? sourceKind
      : "node_result"

  if (binding.sourceKind === "flow_meta") {
    binding.field = "flow_id"
    binding.nodeId = ""
    binding.path = ""
    binding.name = ""
  } else if (binding.sourceKind === "run_meta") {
    binding.field = "run_id"
    binding.nodeId = ""
    binding.path = ""
    binding.name = ""
  } else if (binding.sourceKind === "flow_var") {
    binding.field = ""
    binding.nodeId = ""
    binding.path = ""
    binding.name = ""
  } else if (binding.sourceKind === "trigger") {
    binding.field = ""
    binding.nodeId = ""
    binding.name = ""
  } else {
    binding.field = ""
    binding.name = ""
  }

  if (bodyEditorActive.value) {
    syncBodySessionToParent({ commitHistory: true })
    return
  }
  flowStore.commitHistory()
}

const removeNode = () => {
  if (bodyEditorActive.value) {
    removeBodySelectedNode()
    return
  }
  flowStore.removeSelectedNode()
}

const removeEdge = () => {
  if (bodyEditorActive.value) {
    removeBodySelectedEdge()
    return
  }
  flowStore.removeSelectedEdge()
}

const onCanvasConnect = (from: string, to: string) => {
  if (bodyEditorActive.value) {
    try {
      connectBodyNodes(from, to)
    } catch (err) {
      console.warn(err)
      toast.errorOf(err, t("Failed to connect nodes."))
    }
    return
  }
  try {
    flowStore.addEdge(from, to)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to connect nodes."))
  }
}

const onCanvasSelectNode = (nodeId: string) => {
  if (bodyEditorActive.value) {
    selectBodyNodeById(nodeId)
    return
  }
  flowStore.selectNodeById(nodeId)
}

const onCanvasSelectEdge = (from: string, to: string) => {
  if (bodyEditorActive.value) {
    selectBodyEdgeByEndpoints(from, to)
    return
  }
  flowStore.selectEdgeByEndpoints(from, to)
}

const onCanvasNodeMoved = (nodeId: string, x: number, y: number) => {
  if (bodyEditorActive.value) {
    moveBodyNode(nodeId, x, y)
    return
  }
  flowStore.setNodePosition(nodeId, x, y)
  flowStore.commitHistory()
}

const onCanvasClear = () => {
  if (bodyEditorActive.value) {
    clearBodySelection()
    return
  }
  flowStore.clearSelection()
}

const autoLayout = () => {
  try {
    if (bodyEditorActive.value) {
      autoLayoutBodyEditor()
      return
    }
    flowStore.autoLayoutTB()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to auto layout."))
  }
}

const runCurrentFlow = async () => {
  if (!syncBodySessionToParent({ commitHistory: true, silent: true })) {
    toast.error(bodySessionError.value || t("Fix the foreach body editor errors before running the flow."))
    return
  }
  runBusy.value = true
  try {
    await flowStore.runFlow()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Flow run failed."))
  } finally {
    runBusy.value = false
  }
}

const refreshFlowStatus = async () => {
  statusBusy.value = true
  try {
    await flowStore.statusFlow(flowStore.state.statusRunId.trim())
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Flow status failed."))
  } finally {
    statusBusy.value = false
  }
}

const loadRunHistory = async (options?: { silent?: boolean }) => {
  runHistoryBusy.value = true
  try {
    await flowStore.listRunsFlow(50)
  } catch (err) {
    console.warn(err)
    if (!options?.silent) {
      toast.errorOf(err, t("Failed to load run history."))
    }
  } finally {
    runHistoryBusy.value = false
  }
}

const cancelCurrentRun = async () => {
  cancelBusy.value = true
  try {
    await flowStore.cancelRunFlow(flowStore.state.statusRunId.trim())
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to cancel flow run."))
  } finally {
    cancelBusy.value = false
  }
}

const saveProject = async () => {
  const id = projectId.value
  if (!id) {
    toast.error(t("Project ID is required in the query string."))
    return
  }
  if (!syncBodySessionToParent({ commitHistory: true, silent: true })) {
    toast.error(bodySessionError.value || t("Fix the foreach body editor errors before saving."))
    return
  }
  saveBusy.value = true
  try {
    const project = projectsStore.getProjectByID(id)
    if (!project) {
      throw new Error(t("Project not found."))
    }
    const graph = flowStore.exportGraphDraft()
    const localProjects = Array.isArray((projectsStore as any)?.state?.projects)
      ? (((projectsStore as any).state.projects as FlowProjectRecord[]) ?? [])
      : []
    const dependencyMap = buildLocalSubflowDependencyMap(localProjects, {
      projectId: id,
      flowId: project.flowId,
      graph
    })
    const recursiveChain = findRecursiveSubflowChain(normalizeFlowId(project.flowId), dependencyMap)
    if (recursiveChain && recursiveChain.length > 1) {
      throw new Error(
        t("Subflow recursion detected across local projects: {chain}", {
          chain: recursiveChain.join(" -> ")
        })
      )
    }
    const saved = await projectsStore.saveProjectGraph(id, graph)
    loadedProjectName.value = saved.name || saved.flowId || id
    updateSavedBaseline(saved.updatedAt)
    clearRecoveryDraft()
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
    bodyEditorSession.value = null
    bodyNodeIdDraft.value = ""
    bodySessionError.value = ""
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
    flowStore.state.flowId = project.flowId
    flowStore.state.flowName = project.name || ""
    flowStore.loadGraphDraft(project.graph)
    void loadRunHistory({ silent: true })
    nodeIdDraft.value = selectedNode.value?.id ?? ""
    updateSavedBaseline(project.updatedAt)
    maybeRestoreRecoveryDraft(project)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to load project."))
  } finally {
    loading.value = false
  }
}

const undoEditor = () => {
  if (!syncBodySessionToParent({ commitHistory: true, silent: true })) {
    toast.error(bodySessionError.value || t("Fix the foreach body editor errors before undo."))
    return
  }
  flowStore.undo()
}

const redoEditor = () => {
  if (!syncBodySessionToParent({ commitHistory: true, silent: true })) {
    toast.error(bodySessionError.value || t("Fix the foreach body editor errors before redo."))
    return
  }
  flowStore.redo()
}

const isEditableTarget = (target: EventTarget | null) => {
  const el = target as HTMLElement | null
  if (!el) return false
  if (el.isContentEditable) return true
  const tag = el.tagName?.toLowerCase()
  return tag === "input" || tag === "textarea" || tag === "select"
}

const onKeyDown = (event: KeyboardEvent) => {
  if (addNodeOpen.value || methodDialogOpen.value || fieldBindingDialogOpen.value) return

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

  if (key === "Escape" && detailPanelOpen.value) {
    event.preventDefault()
    closeNodeDetail()
    return
  }

  if (key === "Delete") {
    event.preventDefault()
    if (bodyEditorActive.value) {
      if (bodyEditorSession.value?.snapshot.selectedEdgeIndex !== undefined && bodyEditorSession.value.snapshot.selectedEdgeIndex >= 0) {
        removeBodySelectedEdge()
      } else if (bodyEditorSession.value?.snapshot.selectedNodeIndex !== undefined && bodyEditorSession.value.snapshot.selectedNodeIndex >= 0) {
        removeBodySelectedNode()
      }
    } else if (flowStore.state.selectedEdgeIndex >= 0) {
      flowStore.removeSelectedEdge()
    } else if (flowStore.state.selectedNodeIndex >= 0) {
      flowStore.removeSelectedNode()
    }
    return
  }

  if (ctrl && lower === "z" && !event.shiftKey) {
    event.preventDefault()
    undoEditor()
    return
  }

  if (ctrl && (lower === "y" || (lower === "z" && event.shiftKey))) {
    event.preventDefault()
    redoEditor()
  }
}

const onBeforeUnload = (event: BeforeUnloadEvent) => {
  if (!dirty.value) return
  event.preventDefault()
  event.returnValue = ""
}

watch(
  () => [selfNodeId.value, hubId.value],
  ([nodeId, currentHubId]) => {
    flowStore.setIdentity(Number(nodeId), Number(currentHubId))
    if (!methodDialogOpen.value) {
      syncQueryNodeDraft()
    }
  },
  { immediate: true }
)

watch(
  () => [
    loading.value,
    bodyEditorActive.value,
    activeCallNode.value?.id ?? "",
    activeCallNode.value?.method ?? "",
    activeCallNode.value?.target ?? 0,
    effectiveExecutorNode.value
  ],
  ([isLoading]) => {
    if (isLoading) {
      return
    }
    void ensureSelectedNodeCapabilityLoaded()
  },
  { immediate: true }
)

watch(
  () => selectedNode.value?.id ?? "",
  () => {
    nodeIdDraft.value = selectedNode.value?.id ?? ""
    flowStore.resetNodeDetail(selectedNode.value?.id ?? "")
    closeFieldBindingDialog()
    if (!methodDialogOpen.value) {
      syncPendingCapability()
      syncQueryNodeDraft()
    }
  },
  { immediate: true }
)

watch(
  () => [bodyEditorActive.value, bodySelectedNode.value?.id ?? "", bodySelectedNode.value?.kind ?? ""],
  () => {
    bodyNodeIdDraft.value = bodySelectedNode.value?.id ?? ""
    if (!bodyEditorActive.value) {
      return
    }
    closeFieldBindingDialog()
    if (bodySelectedNode.value?.kind !== "call") {
      closeMethodDialog()
    }
    if (!methodDialogOpen.value) {
      syncPendingCapability()
      syncQueryNodeDraft()
    }
  },
  { immediate: true }
)

watch(
  () => rootGraphSignature.value,
  () => {
    const session = bodyEditorSession.value
    if (!session || bodySessionDirty.value) {
      return
    }
    const parent = flowStore.state.nodes.find((node) => node.id.trim() === session.parentNodeId.trim()) ?? null
    if (!parent || parent.kind !== "foreach" || parent.specEditorMode !== "form") {
      bodyEditorSession.value = null
      bodyNodeIdDraft.value = ""
      bodySessionError.value = ""
      return
    }
    try {
      const body = JSON.parse(parent.foreachBodyJson.trim() || "{\"nodes\":[],\"edges\":[]}")
      if (!body || typeof body !== "object" || Array.isArray(body) || !Array.isArray(body.nodes) || !Array.isArray(body.edges)) {
        throw new Error(t("Foreach body must include nodes and edges arrays."))
      }
      const nextSnapshot = normalizeBodySessionSnapshot(createGraphEditorStateFromDraft(body, { allowLoopSources: true }))
      const nextSignature = graphSignatureOf(nextSnapshot)
      if (nextSignature === graphSignatureOf(session.snapshot)) {
        session.syncedSignature = nextSignature
        return
      }
      session.snapshot = nextSnapshot
      session.syncedSignature = nextSignature
      bodyNodeIdDraft.value = nextSnapshot.nodes[nextSnapshot.selectedNodeIndex]?.id ?? ""
      bodySessionError.value = ""
    } catch (err) {
      console.warn(err)
      bodySessionError.value = String((err as Error)?.message ?? err ?? t("Failed to refresh foreach body editor state."))
    }
  }
)

watch(
  () => detailPanelOpen.value,
  (open) => {
    if (!open) {
      closeMethodDialog()
      closeFieldBindingDialog()
    }
  }
)

watch(
  () => (bodyEditorActive.value ? bodySelectedNode.value?.kind ?? "" : selectedNode.value?.kind ?? ""),
  (kind) => {
    if (kind !== "call") {
      closeMethodDialog()
      closeFieldBindingDialog()
    }
  }
)

watch(
  () => activeCallVisualForm.value,
  (form) => {
    syncFieldDrafts(form)
    if (!fieldBindingDialogOpen.value || !activeBindingFieldPointer.value) return
    const stillExists = form?.fields.some((field) => field.schema.pointer === activeBindingFieldPointer.value)
    if (!stillExists) {
      closeFieldBindingDialog()
    }
  },
  { immediate: true }
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

watch(
  () => [
    projectId.value,
    loading.value,
    dirty.value,
    rootGraphSignature.value,
    lastSavedSignature.value,
    bodySessionDirty.value,
    bodyEditorSession.value ? graphSignatureOf(bodyEditorSession.value.snapshot) : ""
  ],
  ([id, isLoading, isDirty]) => {
    if (!id || isLoading) return
    if (!isDirty) {
      clearRecoveryDraft()
      return
    }
    scheduleRecoveryDraftWrite()
  }
)

onMounted(() => {
  void loadHomeDefaults().then(() => loadProject())
  window.addEventListener("keydown", onKeyDown)
  window.addEventListener("beforeunload", onBeforeUnload)
})

onUnmounted(() => {
  clearRecoveryWriteTimer()
  window.removeEventListener("keydown", onKeyDown)
  window.removeEventListener("beforeunload", onBeforeUnload)
})
</script>

<template>
  <section class="flex h-full min-h-0 flex-col bg-card/70 text-card-foreground">
    <FlowEditorToolbar
      :title="loadedProjectName || t('Untitled Project')"
      :dirty="dirty"
      :last-saved-label="lastSavedLabel"
      :save-busy="saveBusy"
      :run-busy="runBusy"
      :status-busy="statusBusy"
      :run-history-busy="runHistoryBusy || flowStore.state.runHistoryLoading"
      :cancel-busy="cancelBusy"
      :loading="loading"
      :can-undo="canUndo"
      :can-redo="canRedo"
      :has-selected-node="bodyEditorActive ? bodyEditorSession?.snapshot.selectedNodeIndex >= 0 : flowStore.state.selectedNodeIndex >= 0"
      :has-selected-edge="bodyEditorActive ? bodyEditorSession?.snapshot.selectedEdgeIndex >= 0 : flowStore.state.selectedEdgeIndex >= 0"
      :can-run-flow="canRunFlow"
      :can-refresh-status="canRefreshStatus"
      :can-list-runs="canListRuns"
      :can-cancel-run="canCancelRun"
      :flow-status-label="flowStatusLabel"
      :current-run-id-label="currentRunIdLabel"
      @add-node="openAddNodeDialog"
      @remove-node="removeNode"
      @remove-edge="removeEdge"
      @undo="undoEditor"
      @redo="redoEditor"
      @auto-layout="autoLayout"
      @save-project="saveProject"
      @run-flow="runCurrentFlow"
      @refresh-status="refreshFlowStatus"
      @load-run-history="loadRunHistory"
      @cancel-run="cancelCurrentRun"
    />

    <div v-if="!loading" class="border-b border-border/60 bg-card/85 px-5 py-3">
      <div class="flex flex-wrap items-center gap-3 text-xs">
        <label class="font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Selected Run") }}</label>
        <select
          v-model="flowStore.state.statusRunId"
          class="h-8 min-w-[220px] rounded-md border border-input bg-background px-2 text-xs text-foreground"
        >
          <option value="">{{ t("Latest run") }}</option>
          <option v-for="item in flowStore.state.runHistory" :key="item.runId" :value="item.runId">
            {{ item.runId }} · {{ t(flowStatusLabelKey(item.status || "unknown")) }}
          </option>
        </select>
        <span class="text-muted-foreground">
          {{ t("{count} runs", { count: flowStore.state.runHistory.length }) }}
        </span>
      </div>
    </div>

    <div v-if="loading" class="flex flex-1 items-center justify-center px-6 text-sm text-muted-foreground">
      {{ t("Loading project...") }}
    </div>

    <div v-else class="relative flex-1 min-h-0 overflow-hidden">
      <div class="flex h-full min-h-0 flex-col p-4 transition-[padding] duration-200" :class="detailPanelOpen ? 'pr-[440px]' : ''">
        <div
          v-if="bodyEditorActive"
          class="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border/70 bg-background/70 px-4 py-3"
        >
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Foreach Body Editor") }}
            </p>
            <p class="mt-1 text-sm text-muted-foreground">
              {{ t("Editing nested body graph for foreach node {nodeId}.", { nodeId: bodyEditorSession?.parentNodeId || "-" }) }}
            </p>
            <p v-if="bodySessionError" class="mt-1 text-xs text-destructive">
              {{ bodySessionError }}
            </p>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-xs text-muted-foreground">
              {{ bodySessionDirty ? t("Unsynced local body changes") : t("Body graph synced to parent foreach JSON") }}
            </span>
            <Button type="button" variant="outline" size="sm" @click="closeForeachBodyEditor">
              {{ t("Return to Flow") }}
            </Button>
          </div>
        </div>

        <div class="flex-1 min-h-0">
          <FlowCanvas
            :nodes="activeCanvasNodes"
            :edges="activeCanvasEdges"
            :selected-node-id="activeSelectedNodeId"
            :selected-edge="activeSelectedEdge"
            :status-nodes="flowStore.state.lastStatus.nodes"
            @connect="onCanvasConnect"
            @select-node="onCanvasSelectNode"
            @select-edge="onCanvasSelectEdge"
            @node-moved="onCanvasNodeMoved"
            @clear-selection="onCanvasClear"
          />
        </div>
      </div>

      <div
        v-if="detailPanelOpen"
        class="pointer-events-none absolute inset-y-0 right-0 z-20 flex w-full justify-end p-0"
      >
        <FlowBodyNodeInspector
          v-if="bodyEditorActive && bodySelectedNode"
          :selected-node="bodySelectedNode"
          :node-id-draft="bodyNodeIdDraft"
          :selected-target-label="bodySelectedTargetLabel"
          :selected-call-visual-form="bodySelectedCallVisualForm"
          :ancestor-node-options="bodyBindableAncestorNodeOptions"
          :field-drafts="fieldDrafts"
          @close="closeNodeDetail"
          @update:node-id-draft="bodyNodeIdDraft = $event"
          @commit-node-id="commitNodeId"
          @node-kind-change="setSelectedNodeKind"
          @toggle-spec-mode="setSelectedNodeSpecMode"
          @open-method="openMethodDialog"
          @open-field-binding="openFieldBindingDialog"
          @clear-field-binding="clearVisualFieldBinding"
          @commit-field-literal="commitFieldLiteralValue"
          @set-boolean-field-literal="setBooleanFieldLiteralValue($event.field, $event.checked)"
          @add-binding="addBinding"
          @remove-binding="removeBinding"
          @binding-source-kind-change="onBindingSourceKindChange($event.binding, $event.sourceKind)"
          @commit-history="commitBodySessionHistory"
        />
        <FlowEdgeInspector
          v-else-if="bodyEditorActive && bodySelectedEdge"
          :selected-edge="bodySelectedEdge"
          :source-node-kind="bodySelectedEdgeSourceNode?.kind ?? null"
          @close="closeNodeDetail"
          @update:edge-case="updateSelectedEdgeCase"
        />
        <FlowNodeInspector
          v-else-if="selectedNode"
          :selected-node="selectedNode"
          :node-id-draft="nodeIdDraft"
          :selected-node-validation="selectedNodeValidation"
          :selected-target-label="selectedTargetLabel"
          :selected-call-visual-form="selectedCallVisualForm"
          :ancestor-node-options="ancestorNodeOptions"
          :node-detail="flowStore.state.nodeDetail"
          :selected-node-output-schema-text="selectedNodeOutputSchemaText"
          :field-drafts="fieldDrafts"
          @close="closeNodeDetail"
          @update:node-id-draft="nodeIdDraft = $event"
          @commit-node-id="commitNodeId"
          @node-kind-change="setSelectedNodeKind"
          @toggle-spec-mode="setSelectedNodeSpecMode"
          @update:node-detail-run-id="updateSelectedNodeDetailRunId"
          @update:node-detail-path="updateSelectedNodeDetailPath"
          @load-node-detail="loadSelectedNodeDetail"
          @open-method="openMethodDialog"
          @edit-foreach-body="openForeachBodyEditor"
          @open-field-binding="openFieldBindingDialog"
          @clear-field-binding="clearVisualFieldBinding"
          @commit-field-literal="commitFieldLiteralValue"
          @set-boolean-field-literal="setBooleanFieldLiteralValue($event.field, $event.checked)"
          @add-binding="addBinding"
          @remove-binding="removeBinding"
          @binding-source-kind-change="onBindingSourceKindChange($event.binding, $event.sourceKind)"
          @commit-history="flowStore.commitHistory()"
        />
        <FlowEdgeInspector
          v-else-if="selectedEdge"
          :selected-edge="selectedEdge"
          :source-node-kind="selectedEdgeSourceNode?.kind ?? null"
          @close="closeNodeDetail"
          @update:edge-case="updateSelectedEdgeCase"
        />
      </div>
    </div>

    <FlowMethodPickerDialog
      :open="methodDialogOpen"
      :loading="flowStore.state.execCapabilitiesLoading"
      :capability-count="flowStore.state.execCapabilities.length"
      :filtered-capabilities="filteredCapabilities"
      :effective-executor-node="effectiveExecutorNode"
      :capability-query-node-label="capabilityQueryNodeLabel"
      :selected-target-label="activeTargetLabel"
      :query-node-id-draft="queryNodeIdDraft"
      :method-search="methodSearch"
      :pending-capability-key="pendingCapabilityKey"
      @close="closeMethodDialog"
      @refresh="refreshMethodCapabilities"
      @select-capability="selectCapability"
      @apply="applyCapabilitySelection"
      @update:query-node-id-draft="queryNodeIdDraft = $event"
      @update:method-search="methodSearch = $event"
    />

    <FlowFieldBindingDialog
      :open="fieldBindingDialogOpen"
      :active-binding-field="activeBindingField"
      :bindable-ancestor-node-options="activeBindableAncestorNodeOptions"
      :allow-loop-sources="allowLoopBindingSources"
      :field-binding-draft="fieldBindingDraft"
      @close="closeFieldBindingDialog"
      @apply="applyFieldBinding"
      @clear="clearVisualFieldBinding()"
      @source-kind-change="onFieldBindingSourceKindChange"
    />

    <FlowAddNodeDialog
      :open="addNodeOpen"
      :node-id="nodeDraft.id"
      :node-kind="nodeDraft.kind"
      @close="addNodeOpen = false"
      @update:node-id="nodeDraft.id = $event"
      @update:node-kind="nodeDraft.kind = $event"
      @add="addNode"
    />
  </section>
</template>
