const isPrintableAscii = (code: number) => code >= 32 && code <= 126

export const toByteArray = (payload: any): Uint8Array | null => {
  if (!payload) return null
  if (payload instanceof Uint8Array) return payload
  if (payload instanceof ArrayBuffer) return new Uint8Array(payload)
  if (Array.isArray(payload)) return new Uint8Array(payload)
  if (typeof payload === "string") {
    const trimmed = payload.trim()
    if (!trimmed) return null
    try {
      const binary = atob(trimmed)
      const bytes = new Uint8Array(binary.length)
      for (let i = 0; i < binary.length; i += 1) {
        bytes[i] = binary.charCodeAt(i)
      }
      return bytes
    } catch {
      return new TextEncoder().encode(trimmed)
    }
  }
  if (payload && typeof payload === "object" && Array.isArray(payload.data)) {
    return new Uint8Array(payload.data)
  }
  return null
}

export const formatTimestamp = (raw: string) => {
  if (!raw) return ""
  const dt = new Date(raw)
  if (Number.isNaN(dt.getTime())) return raw
  const pad = (value: number, len = 2) => String(value).padStart(len, "0")
  return `${dt.getFullYear()}-${pad(dt.getMonth() + 1)}-${pad(dt.getDate())} ${pad(
    dt.getHours()
  )}:${pad(dt.getMinutes())}:${pad(dt.getSeconds())}`
}

const decodeUtf8 = (bytes: Uint8Array) => {
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes)
  } catch {
    return ""
  }
}

export const buildTextPreview = (bytes: Uint8Array, limit: number) => {
  const sliced = limit > 0 && bytes.length > limit ? bytes.slice(0, limit) : bytes
  const truncated = limit > 0 && bytes.length > limit
  const utf8 = decodeUtf8(sliced)
  if (utf8) {
    return { text: utf8, truncated }
  }
  let text = ""
  for (const value of sliced) {
    text += isPrintableAscii(value) ? String.fromCharCode(value) : "."
  }
  return { text, truncated }
}

export const bytesToSpacedHex = (bytes: Uint8Array) => {
  const chunks: string[] = []
  for (let i = 0; i < bytes.length; i += 1) {
    const hex = bytes[i].toString(16).padStart(2, "0").toUpperCase()
    chunks.push(hex)
  }
  return chunks.map((hex, index) => (index > 0 && index % 2 === 0 ? ` ${hex}` : hex)).join("")
}

export const buildPayloadText = (bytes: Uint8Array) => buildTextPreview(bytes, -1).text

export const tryFormatJson = (text: string) => {
  if (!text.trim()) {
    return { ok: false, formatted: "", error: "格式化失败" }
  }
  try {
    const parsed = JSON.parse(text)
    return { ok: true, formatted: JSON.stringify(parsed, null, 2), error: "" }
  } catch (err) {
    return { ok: false, formatted: "", error: "格式化失败" }
  }
}
