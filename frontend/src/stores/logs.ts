import { reactive } from "vue"
import { EventsOn } from "../../wailsjs/runtime/runtime"

type WailsBinding = (...args: any[]) => Promise<any>

const callLogs = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.logs?.LogService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(`Log binding '${method}' unavailable`)
  }
  return fn(...args)
}

export type LogLine = {
  id: string
  level: string
  message: string
  time: string
  payload?: any
  payloadLen?: number
  payloadTruncated?: boolean
}

type LogState = {
  lines: LogLine[]
  paused: boolean
}

const maxLines = 2000

const state = reactive<LogState>({
  lines: [],
  paused: false
})

let initialized = false
const pending: LogLine[] = []
let flushTimer: number | null = null
let nextId = 1

const toIso = (value: any) => {
  if (!value) return ""
  if (typeof value === "string") return value
  try {
    return new Date(value).toISOString()
  } catch {
    return ""
  }
}

const mapLine = (data: any): LogLine => ({
  id: `log_${nextId++}`,
  level: String(data?.level ?? ""),
  message: String(data?.message ?? ""),
  time: toIso(data?.time),
  payload: data?.payload ?? null,
  payloadLen: Number(data?.payloadLen ?? data?.payload_len ?? 0),
  payloadTruncated: Boolean(data?.payloadTruncated ?? data?.payload_truncated ?? false)
})

const flushPending = () => {
  flushTimer = null
  if (!pending.length) return
  const next = pending.splice(0, pending.length)
  state.lines = state.lines.concat(next)
  if (state.lines.length > maxLines) {
    state.lines = state.lines.slice(-maxLines)
  }
}

const scheduleFlush = () => {
  if (flushTimer !== null) return
  flushTimer = window.setTimeout(flushPending, 60)
}

const enqueueLine = (line: LogLine) => {
  pending.push(line)
  scheduleFlush()
}

const load = async () => {
  const lines = await callLogs<LogLine[]>("Lines")
  state.lines = Array.isArray(lines) ? lines.map(mapLine) : []
  if (state.lines.length > maxLines) {
    state.lines = state.lines.slice(-maxLines)
  }
}

const refreshPaused = async () => {
  state.paused = await callLogs<boolean>("IsPaused")
}

const setPaused = async (paused: boolean) => {
  await callLogs("Pause", paused)
  state.paused = paused
}

const ensureListeners = () => {
  if (initialized) return
  initialized = true
  EventsOn("logs.line", (evt: any) => {
    enqueueLine(mapLine(evt))
  })
}

export const useLogsStore = () => {
  ensureListeners()
  return {
    state,
    load,
    refreshPaused,
    setPaused
  }
}
