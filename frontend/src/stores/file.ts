import { reactive } from "vue"
import { EventsOn } from "../../wailsjs/runtime/runtime"

type WailsBinding = (...args: any[]) => Promise<any>

const callFile = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.file?.FileService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(`File binding '${method}' unavailable`)
  }
  return fn(...args)
}

export type FilePrefs = {
  baseDir: string
  maxSizeBytes: number
  maxConcurrent: number
  chunkBytes: number
  incompleteTtlSec: number
  wantSha256: boolean
  autoAccept: boolean
}

export type FileOffer = {
  sessionId: string
  provider: number
  consumer: number
  dir: string
  name: string
  size: number
  sha256: string
  suggestDir: string
}

export type FileTask = {
  taskId: string
  sessionId: string
  createdAt: string
  updatedAt: string
  op: string
  direction: string
  status: string
  lastError: string
  provider: number
  consumer: number
  peer: number
  dir: string
  name: string
  size: number
  sha256: string
  wantHash: boolean
  localDir: string
  localName: string
  localPath: string
  filePath: string
  sentBytes: number
  ackedBytes: number
  doneBytes: number
}

export type FileEntry = {
  name: string
  isDir: boolean
}

type FileState = {
  selfNodeId: number
  hubId: number
  nodes: number[]
  currentNodeId: number
  currentDir: string
  entries: FileEntry[]
  selected: FileEntry | null
  listing: boolean
  listMessage: string
  previewOpen: boolean
  previewLoading: boolean
  previewText: string
  previewInfo: string
  previewTarget: { nodeId: number; dir: string; name: string } | null
  prefs: FilePrefs
  prefsLoaded: boolean
  tasks: FileTask[]
  tasksUpdatedAt: string
  offer: FileOffer | null
  offerSaveDir: string
}

const defaultPrefs: FilePrefs = {
  baseDir: "./file",
  maxSizeBytes: 0,
  maxConcurrent: 4,
  chunkBytes: 262144,
  incompleteTtlSec: 3600,
  wantSha256: true,
  autoAccept: false
}

const state = reactive<FileState>({
  selfNodeId: 0,
  hubId: 0,
  nodes: [],
  currentNodeId: 0,
  currentDir: "",
  entries: [],
  selected: null,
  listing: false,
  listMessage: "",
  previewOpen: false,
  previewLoading: false,
  previewText: "",
  previewInfo: "",
  previewTarget: null,
  prefs: { ...defaultPrefs },
  prefsLoaded: false,
  tasks: [],
  tasksUpdatedAt: "",
  offer: null,
  offerSaveDir: ""
})

let initialized = false
const offerQueue: FileOffer[] = []

const nowIso = () => new Date().toISOString()

const listKey = (nodeId: number, dir: string) => `${nodeId}|${dir}`

const normalizeDir = (dir: string) => {
  const trimmed = String(dir ?? "").trim()
  if (!trimmed || trimmed === "/") return ""
  return trimmed.replace(/\\/g, "/").replace(/^\/+/, "").replace(/\/+$/, "")
}

const updateEntries = (dirs: string[], files: string[]) => {
  const next: FileEntry[] = []
  for (const dir of dirs) {
    const name = String(dir ?? "").trim()
    if (name) next.push({ name, isDir: true })
  }
  for (const file of files) {
    const name = String(file ?? "").trim()
    if (name) next.push({ name, isDir: false })
  }
  state.entries = next
  state.selected = null
}

const showNextOffer = () => {
  if (state.offer || offerQueue.length === 0) return
  const next = offerQueue.shift() ?? null
  state.offer = next
  state.offerSaveDir = next?.suggestDir || next?.dir || ""
}

const enqueueOffer = (offer: FileOffer) => {
  offerQueue.push(offer)
  showNextOffer()
}

const applyPrefs = (data: any) => {
  state.prefs = {
    baseDir: data?.baseDir ?? defaultPrefs.baseDir,
    maxSizeBytes: Number(data?.maxSizeBytes ?? defaultPrefs.maxSizeBytes),
    maxConcurrent: Number(data?.maxConcurrent ?? defaultPrefs.maxConcurrent),
    chunkBytes: Number(data?.chunkBytes ?? defaultPrefs.chunkBytes),
    incompleteTtlSec: Number(data?.incompleteTtlSec ?? defaultPrefs.incompleteTtlSec),
    wantSha256: Boolean(data?.wantSha256 ?? defaultPrefs.wantSha256),
    autoAccept: Boolean(data?.autoAccept ?? defaultPrefs.autoAccept)
  }
  state.prefsLoaded = true
}

const loadPrefs = async () => {
  const data = await callFile<FilePrefs>("Prefs")
  applyPrefs(data)
}

const savePrefs = async (prefs: FilePrefs) => {
  const saved = await callFile<FilePrefs>("SavePrefs", prefs)
  applyPrefs(saved)
}

const loadNodes = async () => {
  const nodes = await callFile<number[]>("BrowserNodes")
  state.nodes = Array.isArray(nodes) ? nodes : []
}

const saveNodes = async (nodes: number[]) => {
  const normalized = await callFile<number[]>("SaveBrowserNodes", nodes)
  state.nodes = Array.isArray(normalized) ? normalized : []
}

const setIdentity = async (nodeId: number, hubId: number) => {
  const nextNode = Number(nodeId || 0)
  const nextHub = Number(hubId || 0)
  state.selfNodeId = nextNode
  state.hubId = nextHub
  await callFile("SetIdentity", nextNode, nextHub)
}

const requestList = async (nodeId: number, dir: string) => {
  const target = Number(nodeId || 0)
  if (!target) throw new Error("Node ID required.")
  const normalizedDir = normalizeDir(dir)
  state.listing = true
  state.listMessage = ""
  const sourceID = state.selfNodeId
  const hubID = state.hubId
  await callFile("ListSimple", sourceID, hubID, target, normalizedDir, false)
}

const requestReadText = async (nodeId: number, dir: string, name: string) => {
  const target = Number(nodeId || 0)
  if (!target) throw new Error("Node ID required.")
  const normalizedDir = normalizeDir(dir)
  const sourceID = state.selfNodeId
  const hubID = state.hubId
  await callFile("ReadTextSimple", sourceID, hubID, target, normalizedDir, name, 65536)
}

const startPull = async (provider: number, dir: string, name: string, saveDir: string, saveName: string, wantHash: boolean) => {
  const sourceID = state.selfNodeId
  const hubID = state.hubId
  await callFile("StartPull", sourceID, hubID, provider, dir, name, saveDir, saveName, wantHash)
}

const startOffer = async (consumer: number, dir: string, name: string, wantHash: boolean) => {
  const sourceID = state.selfNodeId
  const hubID = state.hubId
  await callFile("StartOffer", sourceID, hubID, consumer, dir, name, wantHash)
}

const loadTasks = async () => {
  const tasks = await callFile<FileTask[]>("TasksSnapshot")
  state.tasks = Array.isArray(tasks) ? tasks : []
  state.tasksUpdatedAt = nowIso()
}

const retryTask = async (taskId: string) => {
  await callFile("RetryTask", taskId)
}

const cancelTask = async (taskId: string) => {
  await callFile("CancelTask", taskId)
}

const openTaskFolder = async (taskId: string) => {
  await callFile("OpenTaskFolder", taskId)
}

const acceptOffer = async () => {
  if (!state.offer) return
  const sessionId = state.offer.sessionId
  const saveDir = state.offerSaveDir || state.offer.suggestDir || state.offer.dir || ""
  await callFile("ConfirmOffer", sessionId, true, saveDir)
  state.offer = null
  state.offerSaveDir = ""
  showNextOffer()
}

const rejectOffer = async () => {
  if (!state.offer) return
  const sessionId = state.offer.sessionId
  await callFile("ConfirmOffer", sessionId, false, "")
  state.offer = null
  state.offerSaveDir = ""
  showNextOffer()
}

const openPreview = async (nodeId: number, dir: string, name: string) => {
  state.previewOpen = true
  state.previewLoading = true
  state.previewText = ""
  state.previewInfo = ""
  state.previewTarget = { nodeId, dir, name }
  await requestReadText(nodeId, dir, name)
}

const closePreview = () => {
  state.previewOpen = false
  state.previewLoading = false
  state.previewText = ""
  state.previewInfo = ""
  state.previewTarget = null
}

const openTasksWindow = () => {
  const base = window.location.href.split("#")[0]
  const url = `${base}#/file-tasks`
  const win = window.open(url, "file_tasks", "width=920,height=680")
  if (win) {
    win.focus()
    return true
  }
  return false
}

const ensureListeners = () => {
  if (initialized) return
  initialized = true

  EventsOn("file.list", (evt: any) => {
    const nodeId = Number(evt?.nodeId ?? 0)
    const dir = normalizeDir(evt?.dir ?? "")
    if (nodeId !== state.currentNodeId || dir !== normalizeDir(state.currentDir)) {
      return
    }
    state.listing = false
    state.listMessage = String(evt?.msg ?? "")
    if (Number(evt?.code ?? 0) !== 1) {
      state.entries = []
      state.selected = null
      return
    }
    const dirs = Array.isArray(evt?.dirs) ? evt.dirs : []
    const files = Array.isArray(evt?.files) ? evt.files : []
    updateEntries(dirs, files)
  })

  EventsOn("file.text", (evt: any) => {
    if (!state.previewTarget) return
    const nodeId = Number(evt?.nodeId ?? 0)
    const dir = normalizeDir(evt?.dir ?? "")
    const name = String(evt?.name ?? "")
    if (
      nodeId !== state.previewTarget.nodeId ||
      dir !== normalizeDir(state.previewTarget.dir) ||
      name !== state.previewTarget.name
    ) {
      return
    }
    state.previewLoading = false
    if (Number(evt?.code ?? 0) !== 1) {
      state.previewText = ""
      state.previewInfo = `Preview failed: ${String(evt?.msg ?? "error")}`
      return
    }
    state.previewText = String(evt?.text ?? "")
    const size = Number(evt?.size ?? 0)
    const truncated = Boolean(evt?.truncated)
    state.previewInfo = `size=${size}${truncated ? " (truncated)" : ""}`
  })

  EventsOn("file.tasks", (evt: any) => {
    const tasks = Array.isArray(evt?.tasks) ? evt.tasks : []
    state.tasks = tasks
    state.tasksUpdatedAt = nowIso()
  })

  EventsOn("file.offer", (evt: any) => {
    const offer: FileOffer = {
      sessionId: String(evt?.sessionId ?? ""),
      provider: Number(evt?.provider ?? 0),
      consumer: Number(evt?.consumer ?? 0),
      dir: String(evt?.dir ?? ""),
      name: String(evt?.name ?? ""),
      size: Number(evt?.size ?? 0),
      sha256: String(evt?.sha256 ?? ""),
      suggestDir: String(evt?.suggestDir ?? "")
    }
    if (!offer.sessionId) return
    enqueueOffer(offer)
  })
}

export const useFileStore = () => {
  ensureListeners()
  return {
    state,
    acceptOffer,
    cancelTask,
    closePreview,
    loadNodes,
    loadPrefs,
    loadTasks,
    openPreview,
    openTaskFolder,
    openTasksWindow,
    rejectOffer,
    requestList,
    saveNodes,
    savePrefs,
    setIdentity,
    startOffer,
    startPull,
    retryTask
  }
}
