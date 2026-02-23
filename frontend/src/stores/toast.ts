import { reactive } from "vue"

export type ToastLevel = "success" | "info" | "warn" | "error"

export type ToastItem = {
  id: string
  level: ToastLevel
  title: string
  detail: string
  createdAt: string
}

type ToastState = {
  items: ToastItem[]
}

type PushOptions = {
  durationMs?: number
}

const state = reactive<ToastState>({
  items: []
})

const timers = new Map<string, number>()

const nowIso = () => new Date().toISOString()

const newId = () => {
  const uuid = (globalThis as any)?.crypto?.randomUUID?.()
  if (uuid) return String(uuid)
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

const defaultDurationMs = (level: ToastLevel) => {
  switch (level) {
    case "success":
      return 2500
    case "info":
      return 3000
    case "warn":
      return 4500
    case "error":
      return 6500
    default:
      return 4000
  }
}

const toText = (v: unknown) => {
  if (v == null) return ""
  if (typeof v === "string") return v
  if (v instanceof Error) return v.message || String(v)
  return String(v)
}

const remove = (id: string) => {
  const idx = state.items.findIndex((item) => item.id === id)
  if (idx < 0) return
  state.items.splice(idx, 1)
  const timer = timers.get(id)
  if (timer) {
    window.clearTimeout(timer)
    timers.delete(id)
  }
}

const clear = () => {
  for (const item of [...state.items]) {
    remove(item.id)
  }
}

const push = (level: ToastLevel, title: string, detail?: string, options?: PushOptions) => {
  const trimmedTitle = title.trim()
  if (!trimmedTitle) return ""
  const id = newId()
  const item: ToastItem = {
    id,
    level,
    title: trimmedTitle,
    detail: (detail ?? "").trim(),
    createdAt: nowIso()
  }
  state.items.push(item)

  const maxItems = 5
  while (state.items.length > maxItems) {
    const oldest = state.items[0]
    remove(oldest.id)
  }

  const durationMs = options?.durationMs ?? defaultDurationMs(level)
  if (durationMs > 0) {
    const timer = window.setTimeout(() => remove(id), durationMs)
    timers.set(id, timer)
  }
  return id
}

const success = (title: string, detail?: string, options?: PushOptions) =>
  push("success", title, detail, options)
const info = (title: string, detail?: string, options?: PushOptions) =>
  push("info", title, detail, options)
const warn = (title: string, detail?: string, options?: PushOptions) =>
  push("warn", title, detail, options)
const error = (title: string, detail?: string, options?: PushOptions) =>
  push("error", title, detail, options)

const errorOf = (err: unknown, fallbackTitle = "Operation failed.") => {
  const msg = toText(err).trim()
  const title = fallbackTitle.trim()
  if (!msg) {
    return error(title || "Operation failed.")
  }
  if (title && msg !== title) {
    return error(title, msg)
  }
  return error(msg)
}

export const useToastStore = () => {
  return {
    state,
    push,
    remove,
    clear,
    success,
    info,
    warn,
    error,
    errorOf
  }
}
