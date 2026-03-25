<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from "vue"
import { useRoute } from "vue-router"
import FlowCanvas from "@/components/flow/FlowCanvas.vue"
import FlowAddNodeDialog from "@/components/flow/editor/FlowAddNodeDialog.vue"
import FlowEditorToolbar from "@/components/flow/editor/FlowEditorToolbar.vue"
import FlowFieldBindingDialog from "@/components/flow/editor/FlowFieldBindingDialog.vue"
import FlowMethodPickerDialog from "@/components/flow/editor/FlowMethodPickerDialog.vue"
import FlowNodeInspector from "@/components/flow/editor/FlowNodeInspector.vue"
import { useI18n } from "@/i18n"
import { normalizeFormInputText, parseNumberInput, type FormInputValue } from "@/lib/numberInput"
import {
  useFlowStore,
  type ExecCapabilityRoute,
  type FlowBindingSourceKind,
  type FlowGraphEditorState,
  type FlowInputBindingDraft,
  type FlowNodeKind,
  type NodeVisualFormModel,
  type VisualBindingSource,
  type VisualFieldModel
} from "@/stores/flow"
import { useFlowProjectsStore, type FlowProjectRecord } from "@/stores/flowProjects"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"
import { HomeState as LoadHomeState } from "../../wailsjs/go/main/App"

type FlowEditorRecoveryRecord = {
  version: 1
  projectId: string
  baseSignature: string
  savedAt: string
  snapshot: FlowGraphEditorState
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
let recoveryWriteTimer: number | null = null

const fieldBindingDraft = reactive({
  sourceKind: "trigger" as FlowBindingSourceKind,
  nodeId: "",
  path: "",
  field: "flow_id",
  required: false
})

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
const recoveryStorageKey = computed(() =>
  projectId.value ? `${recoveryStoragePrefix}${projectId.value}` : ""
)
const graphSignature = computed(() => flowStore.graphEditorSignature())
const dirty = computed(() => !loading.value && graphSignature.value !== lastSavedSignature.value)

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

const bindableAncestorNodeOptions = computed(() =>
  selectedNode.value ? flowStore.listBindableAncestorNodeIds(selectedNode.value.id) : []
)

const selectedCallVisualForm = computed<NodeVisualFormModel | null>(() => {
  const node = selectedNode.value
  if (!node || node.kind !== "call") return null
  return flowStore.getNodeVisualForm(node.id)
})

const activeBindingField = computed<VisualFieldModel | null>(() => {
  const form = selectedCallVisualForm.value
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
      snapshot: parsed.snapshot as FlowGraphEditorState
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
    const payload: FlowEditorRecoveryRecord = {
      version: 1,
      projectId: projectId.value,
      baseSignature: lastSavedSignature.value,
      savedAt: new Date().toISOString(),
      snapshot: flowStore.exportGraphEditorState()
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
  lastSavedSignature.value = graphSignature.value
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
  if (recoveryGraphSignature === lastSavedSignature.value) {
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
    fieldBindingDraft.required = source.required
    return
  }
  if (source?.kind === "trigger") {
    fieldBindingDraft.sourceKind = "trigger"
    fieldBindingDraft.nodeId = ""
    fieldBindingDraft.path = source.path
    fieldBindingDraft.field = "flow_id"
    fieldBindingDraft.required = source.required
    return
  }
  if (source?.kind === "flow_meta") {
    fieldBindingDraft.sourceKind = "flow_meta"
    fieldBindingDraft.nodeId = ""
    fieldBindingDraft.path = ""
    fieldBindingDraft.field = "flow_id"
    fieldBindingDraft.required = source.required
    return
  }
  if (source?.kind === "run_meta") {
    fieldBindingDraft.sourceKind = "run_meta"
    fieldBindingDraft.nodeId = ""
    fieldBindingDraft.path = ""
    fieldBindingDraft.field = "run_id"
    fieldBindingDraft.required = source.required
    return
  }

  fieldBindingDraft.sourceKind = bindableAncestorNodeOptions.value.length ? "node_result" : "trigger"
  fieldBindingDraft.nodeId = bindableAncestorNodeOptions.value[0] ?? ""
  fieldBindingDraft.path = ""
  fieldBindingDraft.field = "flow_id"
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
    sourceKind === "node_result" || sourceKind === "flow_meta" || sourceKind === "run_meta"
      ? sourceKind
      : "trigger"
  if (fieldBindingDraft.sourceKind === "node_result") {
    fieldBindingDraft.nodeId = bindableAncestorNodeOptions.value[0] ?? ""
    fieldBindingDraft.path = ""
    fieldBindingDraft.field = "flow_id"
    return
  }
  if (fieldBindingDraft.sourceKind === "trigger") {
    fieldBindingDraft.nodeId = ""
    fieldBindingDraft.path = ""
    fieldBindingDraft.field = "flow_id"
    return
  }
  fieldBindingDraft.nodeId = ""
  fieldBindingDraft.path = ""
  fieldBindingDraft.field = fieldBindingDraft.sourceKind === "run_meta" ? "run_id" : "flow_id"
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
  const node = selectedNode.value
  const field = activeBindingField.value
  if (!node || node.kind !== "call" || !field) return
  try {
    flowStore.setFieldBinding(node.id, field.schema.pointer, buildVisualBindingSource())
    closeFieldBindingDialog()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to save field binding."))
  }
}

const clearVisualFieldBinding = (pointer?: string) => {
  const node = selectedNode.value
  const targetPointer = pointer ?? activeBindingFieldPointer.value
  if (!node || node.kind !== "call" || !targetPointer) return
  try {
    flowStore.clearFieldBinding(node.id, targetPointer)
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
  const node = selectedNode.value
  if (!node || node.kind !== "call") return
  try {
    flowStore.setFieldLiteralValue(node.id, field.schema.pointer, parseFieldDraftValue(field))
    const form = flowStore.getNodeVisualForm(node.id)
    const nextField = form.fields.find((item) => item.schema.pointer === field.schema.pointer)
    fieldDrafts[field.schema.pointer] = nextField ? stringifyFieldDraftValue(nextField) : ""
  } catch (err) {
    console.warn(err)
    fieldDrafts[field.schema.pointer] = stringifyFieldDraftValue(field)
    toast.errorOf(err, t("Failed to update field value."))
  }
}

const setBooleanFieldLiteralValue = (field: VisualFieldModel, checked: boolean) => {
  const node = selectedNode.value
  if (!node || node.kind !== "call") return
  try {
    flowStore.setFieldLiteralValue(node.id, field.schema.pointer, checked)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to update field value."))
  }
}

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
  methodSearch.value = ""
}

const ensureSelectedNodeCapabilityLoaded = async () => {
  if (loading.value) {
    return
  }
  const node = selectedNode.value
  if (!node || node.kind !== "call" || !node.method.trim()) {
    return
  }
  try {
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
  if (!selectedNode.value || selectedNode.value.kind !== "call") return
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
    flowStore.applyCallCapability(pendingCapabilityKey.value)
    closeMethodDialog()
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

const setSelectedNodeSpecMode = (mode: "form" | "json") => {
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

const onBindingSourceKindChange = (binding: FlowInputBindingDraft, sourceKind: string) => {
  binding.sourceKind =
    sourceKind === "trigger" || sourceKind === "flow_meta" || sourceKind === "run_meta"
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
    updateSavedBaseline(project.updatedAt)
    maybeRestoreRecoveryDraft(project)
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

  if (key === "Escape" && nodeDetailOpen.value) {
    event.preventDefault()
    closeNodeDetail()
    return
  }

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
    selectedNode.value?.id ?? "",
    selectedNode.value?.kind ?? "",
    selectedNode.value?.method ?? "",
    selectedNode.value?.target ?? 0,
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
    closeFieldBindingDialog()
    if (!methodDialogOpen.value) {
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
      closeMethodDialog()
      closeFieldBindingDialog()
    }
  }
)

watch(
  () => selectedNode.value?.kind ?? "",
  (kind) => {
    if (kind !== "call") {
      closeMethodDialog()
      closeFieldBindingDialog()
    }
  }
)

watch(
  () => selectedCallVisualForm.value,
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
  () => [projectId.value, loading.value, dirty.value, graphSignature.value, lastSavedSignature.value],
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
      :loading="loading"
      :can-undo="canUndo"
      :can-redo="canRedo"
      :has-selected-node="flowStore.state.selectedNodeIndex >= 0"
      :has-selected-edge="flowStore.state.selectedEdgeIndex >= 0"
      @add-node="openAddNodeDialog"
      @remove-node="removeNode"
      @remove-edge="removeEdge"
      @undo="flowStore.undo()"
      @redo="flowStore.redo()"
      @auto-layout="autoLayout"
      @save-project="saveProject"
    />

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

      <div
        v-if="nodeDetailOpen"
        class="pointer-events-none absolute inset-y-0 right-0 z-20 flex w-full justify-end p-0"
      >
        <FlowNodeInspector
          :selected-node="selectedNode"
          :node-id-draft="nodeIdDraft"
          :selected-node-validation="selectedNodeValidation"
          :selected-target-label="selectedTargetLabel"
          :selected-call-visual-form="selectedCallVisualForm"
          :ancestor-node-options="ancestorNodeOptions"
          :field-drafts="fieldDrafts"
          @close="closeNodeDetail"
          @update:node-id-draft="nodeIdDraft = $event"
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
          @commit-history="flowStore.commitHistory()"
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
      :selected-target-label="selectedTargetLabel"
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
      :bindable-ancestor-node-options="bindableAncestorNodeOptions"
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
