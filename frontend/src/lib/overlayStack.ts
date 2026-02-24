export type OverlayEntry = {
  id: number
  closeOnEsc: boolean
  onEsc: () => void
}

let nextID = 1
const stack: OverlayEntry[] = []
let keyListenerEnabled = false

const onKeyDown = (e: KeyboardEvent) => {
  if (e.key !== "Escape") return
  const top = stack.length > 0 ? stack[stack.length - 1] : undefined
  if (!top || !top.closeOnEsc) return
  e.preventDefault()
  e.stopPropagation()
  top.onEsc()
}

const ensureKeyListener = () => {
  if (keyListenerEnabled) return
  if (typeof window === "undefined") return
  window.addEventListener("keydown", onKeyDown)
  keyListenerEnabled = true
}

const teardownKeyListener = () => {
  if (!keyListenerEnabled) return
  if (typeof window === "undefined") return
  window.removeEventListener("keydown", onKeyDown)
  keyListenerEnabled = false
}

export const createOverlayID = () => nextID++

export const registerOverlay = (entry: OverlayEntry) => {
  const idx = stack.findIndex((x) => x.id === entry.id)
  if (idx === -1) {
    stack.push(entry)
    ensureKeyListener()
    return
  }
  stack[idx] = entry
}

export const updateOverlay = (id: number, patch: Partial<OverlayEntry>) => {
  const idx = stack.findIndex((x) => x.id === id)
  if (idx === -1) return
  stack[idx] = { ...stack[idx], ...patch }
}

export const unregisterOverlay = (id: number) => {
  for (let i = stack.length - 1; i >= 0; i--) {
    if (stack[i].id !== id) continue
    stack.splice(i, 1)
  }
  if (stack.length === 0) {
    teardownKeyListener()
  }
}

export const isTopOverlay = (id: number) => {
  const top = stack.length > 0 ? stack[stack.length - 1] : undefined
  return Boolean(top && top.id === id)
}
