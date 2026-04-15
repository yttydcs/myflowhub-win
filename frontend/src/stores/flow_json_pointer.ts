// 本文件维护 `flow_json_pointer` store，并让它与 Wails 绑定及共享前端状态保持同步。

import { t } from "@/i18n"

const decodeJsonPointerToken = (token: string) => {
  if (!token.includes("~")) {
    return token
  }
  let out = ""
  for (let i = 0; i < token.length; i += 1) {
    const ch = token[i]
    if (ch !== "~") {
      out += ch
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

const parseJsonPointer = (pointer: string) => {
  const trimmed = String(pointer ?? "").trim()
  if (!trimmed) {
    return [] as string[]
  }
  if (!trimmed.startsWith("/")) {
    throw new Error(t("JSON Pointer must start with '/'."))
  }
  return trimmed
    .slice(1)
    .split("/")
    .map((part) => decodeJsonPointerToken(part))
}

const cloneJson = <T>(value: T): T => {
  if (value === null || value === undefined) return value
  return JSON.parse(JSON.stringify(value)) as T
}

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value)

const pruneEmptyObjects = (value: unknown): unknown => {
  if (Array.isArray(value)) {
    return value.map((item) => pruneEmptyObjects(item))
  }
  if (!isPlainObject(value)) {
    return value
  }
  const next: Record<string, unknown> = {}
  for (const [key, raw] of Object.entries(value)) {
    const pruned = pruneEmptyObjects(raw)
    if (isPlainObject(pruned) && Object.keys(pruned).length === 0) {
      continue
    }
    next[key] = pruned
  }
  return next
}

export const escapeJsonPointerToken = (token: string) =>
  String(token ?? "")
    .replaceAll("~", "~0")
    .replaceAll("/", "~1")

export const appendJsonPointer = (base: string, token: string) => {
  const encoded = escapeJsonPointerToken(token)
  const trimmedBase = String(base ?? "").trim()
  return trimmedBase ? `${trimmedBase}/${encoded}` : `/${encoded}`
}

export const readValueAtPointer = (doc: unknown, pointer: string): { found: boolean; value: unknown } => {
  const tokens = parseJsonPointer(pointer)
  if (!tokens.length) {
    return { found: true, value: doc }
  }
  let current: unknown = doc
  for (const token of tokens) {
    if (Array.isArray(current)) {
      const idx = Number.parseInt(token, 10)
      if (!Number.isFinite(idx) || idx < 0 || idx >= current.length) {
        return { found: false, value: undefined }
      }
      current = current[idx]
      continue
    }
    if (!isPlainObject(current) || !(token in current)) {
      return { found: false, value: undefined }
    }
    current = current[token]
  }
  return { found: true, value: cloneJson(current) }
}

export const setValueAtPointer = (doc: unknown, pointer: string, value: unknown): unknown => {
  const tokens = parseJsonPointer(pointer)
  if (!tokens.length) {
    return cloneJson(value)
  }
  const root = isPlainObject(doc) ? cloneJson(doc) : {}
  let current: Record<string, unknown> = root
  for (let i = 0; i < tokens.length - 1; i += 1) {
    const token = tokens[i]
    const existing = current[token]
    if (isPlainObject(existing)) {
      current[token] = cloneJson(existing)
      current = current[token] as Record<string, unknown>
      continue
    }
    current[token] = {}
    current = current[token] as Record<string, unknown>
  }
  current[tokens[tokens.length - 1]] = cloneJson(value)
  return root
}

export const deleteValueAtPointer = (doc: unknown, pointer: string): unknown => {
  const tokens = parseJsonPointer(pointer)
  if (!tokens.length) {
    return {}
  }
  if (!isPlainObject(doc)) {
    return {}
  }
  const root = cloneJson(doc)
  const walk = (current: Record<string, unknown>, index: number): boolean => {
    const token = tokens[index]
    if (!(token in current)) {
      return false
    }
    if (index === tokens.length - 1) {
      delete current[token]
      return true
    }
    const next = current[token]
    if (!isPlainObject(next)) {
      return false
    }
    const changed = walk(next, index + 1)
    if (!changed) {
      return false
    }
    const pruned = pruneEmptyObjects(next)
    if (isPlainObject(pruned) && Object.keys(pruned).length === 0) {
      delete current[token]
    } else {
      current[token] = pruned
    }
    return true
  }
  walk(root, 0)
  return pruneEmptyObjects(root)
}

export const collectLeafPointers = (doc: unknown, basePointer = ""): string[] => {
  if (Array.isArray(doc)) {
    return basePointer ? [basePointer] : []
  }
  if (!isPlainObject(doc)) {
    return basePointer ? [basePointer] : []
  }
  const entries = Object.entries(doc)
  if (!entries.length) {
    return []
  }
  const out: string[] = []
  for (const [key, value] of entries) {
    out.push(...collectLeafPointers(value, appendJsonPointer(basePointer, key)))
  }
  return out
}
