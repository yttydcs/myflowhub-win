import { reactive } from "vue"
import { t } from "@/i18n"
import {
  flowEventModeLabelKey,
  flowStatusLabelKey,
  flowTriggerTypeLabelKey,
  type FlowPayload
} from "@/stores/flow"
import { useSessionStore } from "@/stores/session"

const flowProjectsVersion = 1

type WailsBinding = (...args: any[]) => Promise<any>

const callApp = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.main?.App
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(t("App binding '{method}' unavailable", { method }))
  }
  return fn(...args)
}

const callFlow = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.flow?.FlowService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(t("Flow binding '{method}' unavailable", { method }))
  }
  return fn(...args)
}

const newReqId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID()
  }
  return `req_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`
}

const nowIso = () => new Date().toISOString()
const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

export type FlowTriggerDraft = {
  type: "interval" | "cron" | "event" | "var_changed"
  everyMs: number
  cronExpr: string
  eventMode: "publish" | "received" | "any"
  eventName: string
  eventTopic: string
  dedupWindowMs: number
  varOwner: number
  varName: string
}

export type FlowProjectRecord = {
  projectId: string
  flowId: string
  name: string
  maxActiveRuns: number | null
  trigger: FlowTriggerDraft
  graph: {
    nodes: Array<Record<string, any>>
    edges: Array<Record<string, any>>
  }
  updatedAt: string
}

export type FlowDeploymentRecord = {
  flowId: string
  name: string
  everyMs: number
  trigger: Record<string, any> | null
  triggerError: string
  triggerLabel: string
  lastStatus: string
  lastRunId: string
}

type FlowProjectsState = {
  projects: FlowProjectRecord[]
  loading: boolean
  saving: boolean
  deployments: FlowDeploymentRecord[]
  deploymentsLoading: boolean
  deploymentsNodeId: string
}

type FlowSummaryWire = {
  flow_id?: string
  flowId?: string
  name?: string
  every_ms?: number
  everyMs?: number
  last_status?: string
  lastStatus?: string
  last_run_id?: string
  lastRunId?: string
}

const state = reactive<FlowProjectsState>({
  projects: [],
  loading: false,
  saving: false,
  deployments: [],
  deploymentsLoading: false,
  deploymentsNodeId: ""
})

const ensureIdentity = () => {
  const sessionStore = useSessionStore()
  if (!sessionStore.connected) {
    throw new Error(t("Connect before flow deployment operations."))
  }
  const sourceID = Number(sessionStore.auth.nodeId || 0)
  const hubID = Number(sessionStore.auth.hubId || 0)
  if (!sourceID) {
    throw new Error(t("Login required to send Flow requests."))
  }
  if (!hubID) {
    throw new Error(t("Hub ID missing."))
  }
  return { sourceID, hubID }
}

const parsePositiveNodeId = (input: string | number) => {
  const raw = String(input ?? "").trim()
  if (!raw) {
    throw new Error(t("Node ID is required."))
  }
  const parsed = Number.parseInt(raw, 10)
  if (Number.isNaN(parsed) || parsed <= 0) {
    throw new Error(t("Node ID must be a positive number."))
  }
  return parsed
}

const normalizeOptionalNonNegativeInteger = (value: unknown) => {
  if (value === undefined || value === null || String(value).trim() === "") {
    return null
  }
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed >= 0 ? Math.trunc(parsed) : null
}

const assertOptionalNonNegativeInteger = (value: unknown, label: string) => {
  if (value === undefined || value === null || String(value).trim() === "") {
    return null
  }
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed < 0) {
    throw new Error(t("{label} must be a non-negative number.", { label }))
  }
  return Math.trunc(parsed)
}

const assertNonNegativeInteger = (value: unknown, label: string) => {
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed < 0) {
    throw new Error(t("{label} must be a non-negative number.", { label }))
  }
  return Math.trunc(parsed)
}

const normalizeTriggerDraft = (input: any): FlowTriggerDraft => {
  const triggerType = String(input?.type ?? "interval").trim().toLowerCase()
  const type: FlowTriggerDraft["type"] =
    triggerType === "cron" || triggerType === "event" || triggerType === "var_changed" ? triggerType : "interval"

  const everyMs = Number(input?.every_ms ?? input?.everyMs ?? 60000)
  const cronExpr = String(input?.cron ?? input?.cronExpr ?? "").trim()
  const eventModeRaw = String(input?.event_mode ?? input?.eventMode ?? "publish").trim().toLowerCase()
  const eventMode: FlowTriggerDraft["eventMode"] =
    eventModeRaw === "received" || eventModeRaw === "any" ? eventModeRaw : "publish"
  const eventName = String(input?.event_name ?? input?.eventName ?? "").trim()
  const eventTopic = String(input?.event_topic ?? input?.eventTopic ?? "").trim()
  const dedupWindowMs = normalizeOptionalNonNegativeInteger(input?.dedup_window_ms ?? input?.dedupWindowMs) ?? 0
  const ownerRaw = Number(input?.var_owner ?? input?.varOwner ?? 0)
  const varOwner = Number.isFinite(ownerRaw) && ownerRaw > 0 ? Math.trunc(ownerRaw) : 0
  const varName = String(input?.var_name ?? input?.varName ?? "").trim()

  return {
    type,
    everyMs: everyMs > 0 ? Math.trunc(everyMs) : 60000,
    cronExpr,
    eventMode,
    eventName,
    eventTopic,
    dedupWindowMs,
    varOwner,
    varName
  }
}

const toTriggerWire = (trigger: FlowTriggerDraft, options?: { strict?: boolean }) => {
  const strict = Boolean(options?.strict)
  const dedupWindowMs = assertNonNegativeInteger(
    trigger.dedupWindowMs ?? 0,
    t("Trigger dedup window")
  )
  if (trigger.type === "cron") {
    const cron = trigger.cronExpr.trim()
    if (strict && !cron) {
      throw new Error(t("Cron trigger requires an expression."))
    }
    if (strict && dedupWindowMs > 0) {
      throw new Error(t("Cron trigger does not support dedup window."))
    }
    return {
      type: "cron",
      cron
    }
  }
  if (trigger.type === "event") {
    const eventName = trigger.eventName.trim()
    const eventTopic = trigger.eventTopic.trim()
    if (strict && !eventName && !eventTopic) {
      throw new Error(t("Event trigger requires event name or event topic."))
    }
    const out: Record<string, any> = {
      type: "event",
      event_mode: trigger.eventMode
    }
    if (dedupWindowMs > 0) {
      out.dedup_window_ms = dedupWindowMs
    }
    if (eventName) {
      out.event_name = eventName
    }
    if (eventTopic) {
      out.event_topic = eventTopic
    }
    return out
  }
  if (trigger.type === "var_changed") {
    const out: Record<string, any> = {
      type: "var_changed"
    }
    if (dedupWindowMs > 0) {
      out.dedup_window_ms = dedupWindowMs
    }
    if (trigger.varOwner > 0) {
      out.var_owner = Math.trunc(trigger.varOwner)
    }
    if (trigger.varName) {
      out.var_name = trigger.varName
    }
    return out
  }
  if (strict && (!Number.isFinite(trigger.everyMs) || trigger.everyMs <= 0)) {
    throw new Error(t("EveryMs must be a positive number."))
  }
  if (strict && dedupWindowMs > 0) {
    throw new Error(t("Interval trigger does not support dedup window."))
  }
  return {
    type: "interval",
    every_ms: Number.isFinite(trigger.everyMs) && trigger.everyMs > 0 ? Math.trunc(trigger.everyMs) : 60000
  }
}

const normalizeGraph = (input: any) => {
  const nodes: any[] = Array.isArray(input?.nodes) ? input.nodes : []
  const edges: any[] = Array.isArray(input?.edges) ? input.edges : []
  return {
    nodes: nodes.map((node: any) => ({ ...(node || {}) })),
    edges: edges.map((edge: any) => ({ ...(edge || {}) }))
  }
}

const makeProjectID = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID().toLowerCase()
  }
  return `prj_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`
}

const normalizeFlowID = (value: string) => String(value ?? "").trim().toLowerCase()

const isUUIDLike = (value: string) => uuidPattern.test(String(value ?? "").trim())

const makeUUID = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID().toLowerCase()
  }

  const bytes = new Uint8Array(16)
  if (typeof crypto !== "undefined" && "getRandomValues" in crypto) {
    crypto.getRandomValues(bytes)
  } else {
    for (let i = 0; i < bytes.length; i += 1) {
      bytes[i] = Math.floor(Math.random() * 256)
    }
  }

  bytes[6] = (bytes[6] & 0x0f) | 0x40
  bytes[8] = (bytes[8] & 0x3f) | 0x80

  const hex = Array.from(bytes, (value) => value.toString(16).padStart(2, "0"))
  return [
    hex.slice(0, 4).join(""),
    hex.slice(4, 6).join(""),
    hex.slice(6, 8).join(""),
    hex.slice(8, 10).join(""),
    hex.slice(10, 16).join("")
  ].join("-")
}

const ensureFlowIDFormat = (flowId: string) => {
  const trimmedFlowID = String(flowId ?? "").trim()
  if (!trimmedFlowID) {
    throw new Error(t("Flow ID is required."))
  }
  if (!isUUIDLike(trimmedFlowID)) {
    throw new Error(t("Flow ID must be a UUID."))
  }
  return normalizeFlowID(trimmedFlowID)
}

const flowIDTaken = (projects: FlowProjectRecord[], flowId: string, excludeProjectId = "") => {
  const trimmedFlowID = normalizeFlowID(flowId)
  const trimmedProjectID = String(excludeProjectId ?? "").trim()
  if (!trimmedFlowID) return false
  return projects.some(
    (item) => normalizeFlowID(item.flowId) === trimmedFlowID && (!trimmedProjectID || item.projectId !== trimmedProjectID)
  )
}

const ensureUniqueFlowID = (projects: FlowProjectRecord[], flowId: string, excludeProjectId = "") => {
  const trimmedFlowID = ensureFlowIDFormat(flowId)
  if (flowIDTaken(projects, trimmedFlowID, excludeProjectId)) {
    throw new Error(t("Flow ID already exists in local projects."))
  }
  return trimmedFlowID
}

const makeFlowID = (projects: FlowProjectRecord[]) => {
  for (let i = 0; i < 64; i += 1) {
    const candidate = makeUUID()
    if (!flowIDTaken(projects, candidate)) {
      return candidate
    }
  }
  throw new Error(t("Unable to allocate a unique Flow ID."))
}

const normalizeProject = (input: any): FlowProjectRecord | null => {
  const projectId = String(input?.projectId ?? input?.project_id ?? "").trim()
  const flowId = String(input?.flowId ?? input?.flow_id ?? "").trim()
  if (!projectId || !flowId) {
    return null
  }
  const name = String(input?.name ?? "").trim()
  const maxActiveRuns = normalizeOptionalNonNegativeInteger(input?.maxActiveRuns ?? input?.max_active_runs)
  const trigger = normalizeTriggerDraft(input?.trigger ?? {})
  const graph = normalizeGraph(input?.graph ?? {})
  const updatedAt = String(input?.updatedAt ?? input?.updated_at ?? "").trim() || nowIso()
  return {
    projectId,
    flowId,
    name,
    maxActiveRuns,
    trigger,
    graph,
    updatedAt
  }
}

const normalizeProjects = (input: any) => {
  const projects = Array.isArray(input) ? input : []
  const out: FlowProjectRecord[] = []
  const index = new Map<string, number>()
  for (const item of projects) {
    const normalized = normalizeProject(item)
    if (!normalized) continue
    const existing = index.get(normalized.projectId)
    if (existing === undefined) {
      index.set(normalized.projectId, out.length)
      out.push(normalized)
      continue
    }
    out[existing] = normalized
  }
  out.sort((a, b) => a.updatedAt.localeCompare(b.updatedAt) * -1)
  return out
}

const toProjectWire = (project: FlowProjectRecord) => {
  const normalized = normalizeProject(project)
  if (!normalized) {
    throw new Error(t("Project is invalid."))
  }
  return {
    project_id: normalized.projectId,
    flow_id: normalized.flowId,
    name: normalized.name,
    ...(normalized.maxActiveRuns === null ? {} : { max_active_runs: normalized.maxActiveRuns }),
    trigger: toTriggerWire(normalized.trigger),
    graph: normalized.graph,
    updated_at: normalized.updatedAt
  }
}

const saveProjects = async () => {
  state.saving = true
  try {
    const payload = {
      version: flowProjectsVersion,
      projects: state.projects.map(toProjectWire)
    }
    const resp = await callApp<any>("SaveFlowProjectsState", payload)
    const projects = normalizeProjects(resp?.projects)
    state.projects = projects
  } finally {
    state.saving = false
  }
}

const loadProjects = async () => {
  state.loading = true
  try {
    state.projects = await loadProjectsSnapshot()
  } finally {
    state.loading = false
  }
}

const loadProjectsSnapshot = async () => {
  const resp = await callApp<any>("FlowProjectsState")
  return normalizeProjects(resp?.projects)
}

const createProject = async (input: { projectId?: string; flowId?: string; name?: string }) => {
  const latest = await loadProjectsSnapshot()
  const projectId = String(input?.projectId ?? "").trim() || makeProjectID()
  if (latest.some((item) => item.projectId === projectId)) {
    throw new Error(t("Project ID already exists."))
  }
  const requestedFlowID = String(input?.flowId ?? "").trim()
  const flowId = requestedFlowID ? ensureUniqueFlowID(latest, requestedFlowID) : makeFlowID(latest)
  const project: FlowProjectRecord = {
    projectId,
    flowId,
    name: String(input?.name ?? "").trim(),
    maxActiveRuns: null,
    trigger: normalizeTriggerDraft({ type: "interval", every_ms: 60000 }),
    graph: { nodes: [], edges: [] },
    updatedAt: nowIso()
  }
  state.projects = [project, ...latest]
  await saveProjects()
  return project
}

const updateProjectMeta = async (input: {
  projectId: string
  flowId: string
  name?: string
  maxActiveRuns?: string | number | null
}) => {
  const trimmedProjectID = String(input?.projectId ?? "").trim()
  if (!trimmedProjectID) {
    throw new Error(t("Project ID is required."))
  }
  const latest = await loadProjectsSnapshot()
  const idx = latest.findIndex((item) => item.projectId === trimmedProjectID)
  if (idx < 0) {
    throw new Error(t("Project not found."))
  }

  const next: FlowProjectRecord = {
    ...latest[idx],
    flowId: ensureUniqueFlowID(latest, input.flowId, trimmedProjectID),
    name: String(input?.name ?? "").trim(),
    maxActiveRuns: assertOptionalNonNegativeInteger(input?.maxActiveRuns, t("Max active runs")),
    updatedAt: nowIso()
  }
  latest.splice(idx, 1, next)
  state.projects = latest
  await saveProjects()
  return next
}

const saveProjectGraph = async (projectId: string, graph: any) => {
  const trimmedProjectID = String(projectId ?? "").trim()
  if (!trimmedProjectID) {
    throw new Error(t("Project ID is required."))
  }
  const latest = await loadProjectsSnapshot()
  const idx = latest.findIndex((item) => item.projectId === trimmedProjectID)
  if (idx < 0) {
    throw new Error(t("Project not found."))
  }
  const next: FlowProjectRecord = {
    ...latest[idx],
    graph: normalizeGraph(graph),
    updatedAt: nowIso()
  }
  latest.splice(idx, 1, next)
  state.projects = latest
  await saveProjects()
  return next
}

const deleteProject = async (projectId: string) => {
  const trimmed = String(projectId ?? "").trim()
  if (!trimmed) {
    throw new Error(t("Project ID is required."))
  }
  const latest = await loadProjectsSnapshot()
  state.projects = latest.filter((item) => item.projectId !== trimmed)
  await saveProjects()
}

const getProjectByID = (projectId: string) => {
  const trimmed = String(projectId ?? "").trim()
  if (!trimmed) return null
  return state.projects.find((item) => item.projectId === trimmed) ?? null
}

const saveProjectPayload = async (projectId: string, payload: FlowPayload) => {
  const trimmedProjectID = String(projectId ?? "").trim()
  if (!trimmedProjectID) {
    throw new Error(t("Project ID is required."))
  }
  const normalizedPayloadFlowID = ensureFlowIDFormat(String(payload?.flow_id ?? ""))
  const latest = await loadProjectsSnapshot()
  const idx = latest.findIndex((item) => item.projectId === trimmedProjectID)
  if (idx < 0) {
    throw new Error(t("Project not found."))
  }
  const current = latest[idx]
  const next: FlowProjectRecord = {
    ...current,
    flowId: ensureUniqueFlowID(latest, normalizedPayloadFlowID, trimmedProjectID),
    name: String(payload?.name ?? "").trim(),
    maxActiveRuns: assertOptionalNonNegativeInteger(payload?.max_active_runs ?? (payload as any)?.maxActiveRuns, t("Max active runs")),
    trigger: normalizeTriggerDraft(payload?.trigger ?? {}),
    graph: normalizeGraph(payload?.graph ?? {}),
    updatedAt: nowIso()
  }
  latest.splice(idx, 1, next)
  state.projects = latest
  await saveProjects()
}

const openEditorWindow = (projectId: string) => {
  const trimmed = String(projectId ?? "").trim()
  if (!trimmed) {
    throw new Error(t("Project ID is required."))
  }
  const base = window.location.href.split("#")[0]
  const url = `${base}#/flow-editor-window?projectId=${encodeURIComponent(trimmed)}`
  const name = `flow_editor_${trimmed}_${Date.now()}`
  const win = window.open(url, name, "width=1500,height=920")
  if (win) {
    win.focus()
    return true
  }
  return false
}

const mapSummary = (input: FlowSummaryWire) => {
  return {
    flowId: String(input?.flow_id ?? input?.flowId ?? "").trim(),
    name: String(input?.name ?? "").trim(),
    everyMs: Number(input?.every_ms ?? input?.everyMs ?? 0),
    lastStatus: String(input?.last_status ?? input?.lastStatus ?? "").trim(),
    lastRunId: String(input?.last_run_id ?? input?.lastRunId ?? "").trim()
  }
}

const formatTriggerLabel = (trigger: any, fallbackEveryMs = 0) => {
  const type = String(trigger?.type ?? "").trim().toLowerCase()
  if (type === "cron") {
    const cron = String(trigger?.cron ?? trigger?.cronExpr ?? "").trim()
    return cron ? t("Cron · {cron}", { cron }) : t(flowTriggerTypeLabelKey(type))
  }
  if (type === "event") {
    const mode = t(flowEventModeLabelKey(String(trigger?.event_mode ?? "publish")))
    const eventName = String(trigger?.event_name ?? "").trim()
    const eventTopic = String(trigger?.event_topic ?? "").trim()
    const signal = eventName || eventTopic || t("Name or topic empty")
    return t("Event · {mode} · {signal}", { mode, signal })
  }
  if (type === "var_changed") {
    const owner = Number(trigger?.var_owner ?? 0)
    const name = String(trigger?.var_name ?? "").trim()
    if (owner > 0 || name) {
      return t("Variable changed · owner {owner} · {name}", {
        owner: owner > 0 ? owner : t("Any"),
        name: name || t("Any")
      })
    }
    return t(flowTriggerTypeLabelKey(type))
  }
  const everyMs = Number(trigger?.every_ms ?? fallbackEveryMs ?? 0)
  if (everyMs > 0) {
    return t("Interval · every {everyMs} ms", { everyMs })
  }
  return t(type ? flowTriggerTypeLabelKey(type) : "Trigger unavailable")
}

const describeDeploymentStatus = (status: string) => t(flowStatusLabelKey(status || "idle"))

const mapWithConcurrency = async <T, R>(
  items: T[],
  concurrency: number,
  mapper: (item: T, index: number) => Promise<R>
): Promise<R[]> => {
  const size = items.length
  if (!size) return []
  const limit = Math.max(1, Math.min(concurrency, size))
  const results = new Array<R>(size)
  let cursor = 0

  const worker = async () => {
    for (;;) {
      const index = cursor
      cursor += 1
      if (index >= size) return
      results[index] = await mapper(items[index], index)
    }
  }

  await Promise.all(Array.from({ length: limit }, () => worker()))
  return results
}

const loadDeployments = async (nodeIdInput: string | number) => {
  const targetNodeID = parsePositiveNodeId(nodeIdInput)
  const { sourceID, hubID } = ensureIdentity()
  state.deploymentsLoading = true
  try {
    const listResp = await callFlow<any>("ListSimple", sourceID, hubID, {
      req_id: newReqId(),
      origin_node: sourceID,
      executor_node: targetNodeID
    })
    const listCode = Number(listResp?.code ?? 0)
    if (listCode !== 1) {
      throw new Error(String(listResp?.msg ?? t("Flow list failed.")))
    }

    const flowItems: FlowSummaryWire[] = Array.isArray(listResp?.flows) ? listResp.flows : []
    const summaries: ReturnType<typeof mapSummary>[] = flowItems.map((item: FlowSummaryWire) => mapSummary(item)).filter(
      (item: ReturnType<typeof mapSummary>) => item.flowId.length > 0
    )

    const deployments = await mapWithConcurrency(summaries, 6, async (summary) => {
      let trigger: Record<string, any> | null = null
      let triggerError = ""
      try {
        const getResp = await callFlow<any>("GetSimple", sourceID, hubID, {
          req_id: newReqId(),
          origin_node: sourceID,
          executor_node: targetNodeID,
          flow_id: summary.flowId
        })
        if (Number(getResp?.code ?? 0) === 1) {
          trigger = getResp?.trigger && typeof getResp.trigger === "object" ? { ...getResp.trigger } : null
        } else {
          triggerError = String(getResp?.msg ?? t("Flow load failed."))
        }
      } catch (err) {
        trigger = null
        triggerError = String((err as Error)?.message ?? err ?? t("Flow load failed."))
      }

      return {
        flowId: summary.flowId,
        name: summary.name,
        everyMs: summary.everyMs,
        trigger,
        triggerError,
        triggerLabel: formatTriggerLabel(trigger, summary.everyMs),
        lastStatus: summary.lastStatus,
        lastRunId: summary.lastRunId
      }
    })

    deployments.sort((a, b) => a.flowId.localeCompare(b.flowId))
    state.deployments = deployments
    state.deploymentsNodeId = String(targetNodeID)
  } finally {
    state.deploymentsLoading = false
  }
}

const deployProject = async (input: {
  projectId: string
  nodeId: string | number
  trigger: FlowTriggerDraft
  overwrite: boolean
}) => {
  const project = getProjectByID(input.projectId)
  if (!project) {
    throw new Error(t("Project not found."))
  }
  if (!Array.isArray(project.graph?.nodes) || project.graph.nodes.length === 0) {
    throw new Error(t("Project graph requires at least one node."))
  }
  const flowId = ensureFlowIDFormat(project.flowId)
  const targetNodeID = parsePositiveNodeId(input.nodeId)
  const trigger = normalizeTriggerDraft(input.trigger)
  const { sourceID, hubID } = ensureIdentity()

  const listResp = await callFlow<any>("ListSimple", sourceID, hubID, {
    req_id: newReqId(),
    origin_node: sourceID,
    executor_node: targetNodeID
  })
  const listCode = Number(listResp?.code ?? 0)
  if (listCode !== 1) {
    throw new Error(String(listResp?.msg ?? t("Flow list failed.")))
  }
  const existingFlows: FlowSummaryWire[] = Array.isArray(listResp?.flows) ? listResp.flows : []
  const flowExists = existingFlows
    .map((item: FlowSummaryWire) => normalizeFlowID(String(item?.flow_id ?? item?.flowId ?? "")))
    .includes(flowId)

  if (flowExists && !input.overwrite) {
    return { overwriteRequired: true }
  }

  const setResp = await callFlow<any>("SetSimple", sourceID, hubID, {
    req_id: newReqId(),
    origin_node: sourceID,
    executor_node: targetNodeID,
    flow_id: flowId,
    name: project.name,
    ...(project.maxActiveRuns === null ? {} : { max_active_runs: project.maxActiveRuns }),
    trigger: toTriggerWire(trigger, { strict: true }),
    graph: project.graph
  })
  const setCode = Number(setResp?.code ?? 0)
  if (setCode !== 1) {
    throw new Error(String(setResp?.msg ?? t("Flow deploy failed.")))
  }

  const latest = await loadProjectsSnapshot()
  const idx = latest.findIndex((item) => item.projectId === project.projectId)
  if (idx >= 0) {
    latest.splice(idx, 1, {
      ...latest[idx],
      trigger,
      updatedAt: nowIso()
    })
    state.projects = latest
    await saveProjects()
  }

  return { overwriteRequired: false }
}

const deleteDeployment = async (nodeIdInput: string | number, flowId: string) => {
  const targetNodeID = parsePositiveNodeId(nodeIdInput)
  const trimmedFlowID = String(flowId ?? "").trim()
  if (!trimmedFlowID) {
    throw new Error(t("Flow ID is required."))
  }
  const { sourceID, hubID } = ensureIdentity()
  const resp = await callFlow<any>("DeleteSimple", sourceID, hubID, {
    req_id: newReqId(),
    origin_node: sourceID,
    executor_node: targetNodeID,
    flow_id: trimmedFlowID
  })
  const code = Number(resp?.code ?? 0)
  if (code !== 1) {
    throw new Error(String(resp?.msg ?? t("Flow delete failed.")))
  }
  state.deployments = state.deployments.filter((item) => item.flowId !== trimmedFlowID)
}

export const useFlowProjectsStore = () => {
  return {
    state,
    loadProjects,
    saveProjects,
    createProject,
    deleteProject,
    getProjectByID,
    updateProjectMeta,
    saveProjectGraph,
    saveProjectPayload,
    openEditorWindow,
    loadDeployments,
    deployProject,
    deleteDeployment,
    describeDeploymentStatus,
    describeTrigger: formatTriggerLabel,
    normalizeTriggerDraft,
    toTriggerWire
  }
}
