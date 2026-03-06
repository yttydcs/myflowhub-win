<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue"
import { CircleHelp, Database, ExternalLink, GripVertical, Pencil, Plus, RefreshCw, Rss, Trash2 } from "lucide-vue-next"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import { Tooltip } from "@/components/ui/tooltip"
import { clampColSpan, computeColumnsCount } from "@/lib/showcaseLayout"
import { useProfileStore } from "@/stores/profile"
import { useSessionStore } from "@/stores/session"
import {
  useShowcaseStore,
  type ShowcaseWidget,
  type ShowcaseWidgetKind,
  type VarWidgetMode
} from "@/stores/showcase"
import { useToastStore } from "@/stores/toast"
import { HomeState as LoadHomeState } from "../../wailsjs/go/main/App"

const profileStore = useProfileStore()
const sessionStore = useSessionStore()
const showcase = useShowcaseStore()
const toast = useToastStore()

const busy = ref(false)

const fallbackIdentity = reactive({ nodeId: 0, hubId: 0 })
const selfNodeId = computed(() => sessionStore.auth.nodeId || fallbackIdentity.nodeId || 0)
const hubId = computed(() => sessionStore.auth.hubId || fallbackIdentity.hubId || 0)

const connectedLabel = computed(() => (sessionStore.connected ? "Connected" : "Disconnected"))
const connectedTone = computed(() =>
  sessionStore.connected ? "bg-emerald-500/15 text-emerald-700" : "bg-rose-500/15 text-rose-700"
)

const inputClass =
  "mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
const textAreaClass =
  "mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

const loadHomeDefaults = async () => {
  try {
    const state = await LoadHomeState()
    fallbackIdentity.nodeId = Number(state?.nodeId ?? 0)
    fallbackIdentity.hubId = Number(state?.hubId ?? 0)
  } catch (err) {
    console.warn(err)
  }
  showcase.setIdentity(selfNodeId.value, hubId.value)
}

const refreshVars = async () => {
  if (busy.value) return
  busy.value = true
  try {
    await showcase.enter()
    toast.success("Refreshed.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to refresh.")
  } finally {
    busy.value = false
  }
}

const promptCreateScreen = async () => {
  const name = window.prompt("New screen name")
  if (!name) return
  if (busy.value) return
  busy.value = true
  try {
    await showcase.createScreen(name)
    toast.success("Screen created.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to create screen.")
  } finally {
    busy.value = false
  }
}

const promptRenameScreen = async () => {
  const screen = showcase.currentScreen()
  if (!screen) return
  const name = window.prompt("Rename screen", screen.name)
  if (!name) return
  if (busy.value) return
  busy.value = true
  try {
    await showcase.renameScreen(screen.id, name)
    toast.success("Screen renamed.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to rename screen.")
  } finally {
    busy.value = false
  }
}

const deleteCurrentScreen = async () => {
  const screen = showcase.currentScreen()
  if (!screen) return
  if (!window.confirm(`Delete screen '${screen.name}'?`)) return
  if (busy.value) return
  busy.value = true
  try {
    await showcase.deleteScreen(screen.id)
    toast.success("Screen deleted.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to delete screen.")
  } finally {
    busy.value = false
  }
}

const widgetDialog = reactive({
  open: false,
  mode: "create" as "create" | "edit",
  widgetId: "",
  kind: "topic_button" as ShowcaseWidgetKind,
  title: "",
  targetId: "",
  colSpan: "1",
  topic: "",
  eventName: "",
  payloadText: "",
  ownerId: "",
  varName: "",
  varMode: "auto" as VarWidgetMode,
  varType: "",
  visibility: "public",
  sliderMin: "0",
  sliderMax: "100",
  sliderStep: "1",
  sliderThrottleMs: "50",
  switchOnValue: "true",
  switchOffValue: "false"
})

const defaultTypeForMode = (mode: VarWidgetMode) => {
  if (mode === "slider") return "float64"
  if (mode === "switch") return "bool"
  return "string"
}

const lastModeForType = ref<VarWidgetMode>("auto")

const resetWidgetDialog = () => {
  widgetDialog.mode = "create"
  widgetDialog.widgetId = ""
  widgetDialog.title = ""
  widgetDialog.targetId = String(selfNodeId.value || 1)
  widgetDialog.colSpan = "1"
  widgetDialog.topic = ""
  widgetDialog.eventName = ""
  widgetDialog.payloadText = ""
  widgetDialog.ownerId = String(selfNodeId.value || 1)
  widgetDialog.varName = ""
  widgetDialog.varMode = "auto"
  widgetDialog.varType = defaultTypeForMode(widgetDialog.varMode)
  lastModeForType.value = widgetDialog.varMode
  widgetDialog.visibility = "public"
  widgetDialog.sliderMin = "0"
  widgetDialog.sliderMax = "100"
  widgetDialog.sliderStep = "1"
  widgetDialog.sliderThrottleMs = "50"
  widgetDialog.switchOnValue = "true"
  widgetDialog.switchOffValue = "false"
}

const openCreateWidget = (kind: ShowcaseWidgetKind) => {
  resetWidgetDialog()
  widgetDialog.kind = kind
  widgetDialog.open = true
}

const openEditWidget = (widget: ShowcaseWidget) => {
  resetWidgetDialog()
  widgetDialog.open = true
  widgetDialog.mode = "edit"
  widgetDialog.widgetId = widget.id
  widgetDialog.kind = widget.kind
  widgetDialog.title = widget.title || ""
  widgetDialog.targetId = widget.targetId ? String(widget.targetId) : "1"
  widgetDialog.colSpan = String(widget.layout?.colSpan ?? 1)
  if (widget.kind === "topic_button" && widget.topicButton) {
    widgetDialog.topic = widget.topicButton.topic
    widgetDialog.eventName = widget.topicButton.name
    widgetDialog.payloadText = widget.topicButton.payloadText
  }
  if (widget.kind === "var" && widget.var) {
    widgetDialog.ownerId = String(widget.var.ownerId)
    widgetDialog.varName = widget.var.name
    widgetDialog.varMode = widget.var.mode
    widgetDialog.varType = widget.var.type || defaultTypeForMode(widget.var.mode)
    lastModeForType.value = widgetDialog.varMode
    widgetDialog.visibility = widget.var.visibility || "public"
    widgetDialog.sliderMin = String(widget.var.slider?.min ?? 0)
    widgetDialog.sliderMax = String(widget.var.slider?.max ?? 100)
    widgetDialog.sliderStep = String(widget.var.slider?.step ?? 1)
    widgetDialog.sliderThrottleMs = String(widget.var.slider?.throttleMs ?? 50)
    widgetDialog.switchOnValue = widget.var.switch?.onValue ?? "true"
    widgetDialog.switchOffValue = widget.var.switch?.offValue ?? "false"
  }
}

const closeWidgetDialog = () => {
  widgetDialog.open = false
}

const parsePositiveInt = (raw: string, field: string) => {
  const parsed = Number.parseInt(raw.trim(), 10)
  if (Number.isNaN(parsed) || parsed <= 0) throw new Error(`${field} must be a positive number.`)
  return parsed
}

const parseNonNegativeInt = (raw: string, field: string) => {
  const parsed = Number.parseInt(raw.trim(), 10)
  if (Number.isNaN(parsed) || parsed < 0) throw new Error(`${field} must be a valid number (>= 0).`)
  return parsed
}

const parseIntInRange = (raw: string, field: string, min: number, max: number) => {
  const parsed = Number.parseInt(raw.trim(), 10)
  if (Number.isNaN(parsed) || parsed < min || parsed > max) {
    throw new Error(`${field} must be between ${min} and ${max}.`)
  }
  return parsed
}

const parseFloatStrict = (raw: string, field: string) => {
  const parsed = Number.parseFloat(raw.trim())
  if (!Number.isFinite(parsed)) throw new Error(`${field} must be a valid number.`)
  return parsed
}

const resolveVarTargetID = (fallback?: number) => {
  const parsedFallback =
    Number.isFinite(fallback) && Number(fallback) > 0 ? Math.floor(Number(fallback)) : 0
  if (parsedFallback > 0) return parsedFallback
  const hubTarget = Number.isFinite(hubId.value) && Number(hubId.value) > 0 ? Math.floor(Number(hubId.value)) : 0
  if (hubTarget > 0) return hubTarget
  throw new Error("Hub NodeID is required for variable widgets.")
}

const submitWidgetDialog = async () => {
  if (busy.value) return
  busy.value = true
  try {
    const title = widgetDialog.title.trim()
    const screen = showcase.currentScreen()
    const maxColumns = screen?.layout?.columns?.maxColumns ?? 12
    const colSpan = parseIntInRange(widgetDialog.colSpan, "Column Span", 1, Math.max(1, maxColumns))

    if (widgetDialog.kind === "topic_button") {
      const targetId = parsePositiveInt(widgetDialog.targetId, "Target ID")
      const topic = widgetDialog.topic.trim()
      const name = widgetDialog.eventName.trim()
      const payloadText = widgetDialog.payloadText ?? ""
      if (widgetDialog.mode === "create") {
        await showcase.addTopicButton({ title, targetId, colSpan, topic, name, payloadText })
      } else {
        const widget = screen?.widgets.find((w) => w.id === widgetDialog.widgetId)
        if (!widget || widget.kind !== "topic_button") return
        widget.title = title
        widget.targetId = targetId
        widget.layout.colSpan = colSpan
        widget.topicButton = { topic, name, payloadText }
        await showcase.save()
      }
      toast.success("Saved.")
      closeWidgetDialog()
      return
    }

    const ownerId = parsePositiveInt(widgetDialog.ownerId, "Owner NodeID")
    const varName = widgetDialog.varName.trim()
    if (!varName) throw new Error("Variable name is required.")
    const mode = widgetDialog.varMode
    const visibility = widgetDialog.visibility.trim() || "public"
    const type = widgetDialog.varType.trim() || defaultTypeForMode(mode)
    if (!type) throw new Error("Variable type is required.")
    const sliderMin = parseFloatStrict(widgetDialog.sliderMin, "Min")
    const sliderMax = parseFloatStrict(widgetDialog.sliderMax, "Max")
    const sliderStep = parseFloatStrict(widgetDialog.sliderStep, "Step")
    const throttleMs = parseNonNegativeInt(widgetDialog.sliderThrottleMs, "Throttle")
    const onValue = widgetDialog.switchOnValue.trim()
    const offValue = widgetDialog.switchOffValue.trim()
    if (mode === "switch" && (!onValue || !offValue)) throw new Error("Switch on/off values are required.")
    const switchSetting = {
      onValue: onValue || "true",
      offValue: offValue || "false"
    }

    if (widgetDialog.mode === "create") {
      const targetId = resolveVarTargetID()
      await showcase.addVarWidget({
        title,
        targetId,
        colSpan,
        ownerId,
        name: varName,
        mode,
        visibility,
        type,
        slider: { min: sliderMin, max: sliderMax, step: sliderStep, throttleMs },
        switch: switchSetting
      })
    } else {
      const widget = screen?.widgets.find((w) => w.id === widgetDialog.widgetId)
      if (!widget || widget.kind !== "var" || !widget.var) return
      widget.title = title
      widget.targetId = resolveVarTargetID(widget.targetId)
      widget.layout.colSpan = colSpan
      widget.var.ownerId = ownerId
      widget.var.name = varName
      widget.var.mode = mode
      widget.var.visibility = visibility
      widget.var.type = type
      widget.var.slider = { min: sliderMin, max: sliderMax, step: sliderStep, throttleMs }
      widget.var.switch = switchSetting
      await showcase.save()
      await showcase.leave()
      await showcase.enter()
    }
    toast.success("Saved.")
    closeWidgetDialog()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to save widget.")
  } finally {
    busy.value = false
  }
}

const removeWidget = async (widget: ShowcaseWidget) => {
  if (!window.confirm("Remove widget?")) return
  if (busy.value) return
  busy.value = true
  try {
    await showcase.removeWidget(widget.id)
    toast.success("Widget removed.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to remove widget.")
  } finally {
    busy.value = false
  }
}

type WidgetContextMenuState = {
  open: boolean
  x: number
  y: number
  widget: ShowcaseWidget | null
}

const widgetContextMenu = reactive<WidgetContextMenuState>({
  open: false,
  x: 0,
  y: 0,
  widget: null
})

const widgetContextMenuRef = ref<HTMLElement | null>(null)

const closeWidgetContextMenu = () => {
  widgetContextMenu.open = false
  widgetContextMenu.widget = null
}

const clampWidgetContextMenuToViewport = () => {
  const el = widgetContextMenuRef.value
  if (!el) return
  const padding = 8
  const maxLeft = Math.max(padding, window.innerWidth - el.offsetWidth - padding)
  const maxTop = Math.max(padding, window.innerHeight - el.offsetHeight - padding)
  widgetContextMenu.x = Math.min(Math.max(widgetContextMenu.x, padding), maxLeft)
  widgetContextMenu.y = Math.min(Math.max(widgetContextMenu.y, padding), maxTop)
}

const openWidgetContextMenu = async (widget: ShowcaseWidget, event: MouseEvent) => {
  event.preventDefault()
  widgetContextMenu.widget = widget
  widgetContextMenu.x = event.clientX
  widgetContextMenu.y = event.clientY
  widgetContextMenu.open = true
  await nextTick()
  clampWidgetContextMenuToViewport()
}

const onWidgetContextMenuEdit = () => {
  const widget = widgetContextMenu.widget
  closeWidgetContextMenu()
  if (!widget) return
  openEditWidget(widget)
}

const onWidgetContextMenuRemove = async () => {
  const widget = widgetContextMenu.widget
  closeWidgetContextMenu()
  if (!widget) return
  await removeWidget(widget)
}

const onWidgetContextMenuBringToFront = async () => {
  const widget = widgetContextMenu.widget
  closeWidgetContextMenu()
  if (!widget) return
  await reorderWidgetZOrder(widget.id, "front")
}

const onWidgetContextMenuSendToBack = async () => {
  const widget = widgetContextMenu.widget
  closeWidgetContextMenu()
  if (!widget) return
  await reorderWidgetZOrder(widget.id, "back")
}

const onGlobalKeydown = (event: KeyboardEvent) => {
  if (!widgetContextMenu.open) return
  if (event.key !== "Escape") return
  event.preventDefault()
  closeWidgetContextMenu()
}

const safeTitle = (widget: ShowcaseWidget) => {
  const title = widget.title?.trim()
  if (title) return title
  if (widget.kind === "topic_button" && widget.topicButton) {
    return `${widget.topicButton.topic} / ${widget.topicButton.name}`
  }
  if (widget.kind === "var" && widget.var) return widget.var.name
  return "Widget"
}

const displayValueText = (widget: ShowcaseWidget) => {
  const raw = showcase.getVarValueText(widget)
  if (raw.trim()) return raw
  return "No value yet."
}

const isVarOn = (widget: ShowcaseWidget) => {
  if (widget.kind !== "var" || !widget.var) return false
  return showcase.getVarValueText(widget) === widget.var.switch.onValue
}

const sendTopicButton = async (widget: ShowcaseWidget) => {
  if (busy.value) return
  busy.value = true
  try {
    await showcase.publishTopicButton(widget)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to send event.")
  } finally {
    busy.value = false
  }
}

const openShowcaseWindow = () => {
  const screen = showcase.currentScreen()
  if (!screen) return
  const base = window.location.href.split("#")[0]
  const url = `${base}#/showcase-window?screenId=${encodeURIComponent(screen.id)}`
  const name = `showcase_${screen.id}_${Date.now()}`
  const win = window.open(url, name, "width=980,height=720")
  if (win) {
    win.focus()
  }
}

const layoutForm = reactive({
  mode: "columns" as "columns" | "canvas_percent",
  maxColumns: "3",
  minColumnWidth: "360"
})

const syncLayoutFormFromScreen = () => {
  const screen = showcase.currentScreen()
  if (!screen) return
  layoutForm.mode = screen.layout?.mode === "canvas_percent" ? "canvas_percent" : "columns"
  layoutForm.maxColumns = String(screen.layout?.columns?.maxColumns ?? 3)
  layoutForm.minColumnWidth = String(screen.layout?.columns?.minColumnWidth ?? 360)
}

const minCanvasWidgetWidthPx = 80
const minCanvasWidgetHeightPx = 48

const roundPct01 = (value: number): number => {
  const raw = Number(value)
  if (!Number.isFinite(raw)) return 0
  return Math.round(raw * 10) / 10
}

const ceilPct01 = (value: number): number => {
  const raw = Number(value)
  if (!Number.isFinite(raw)) return 0
  const out = Math.ceil(raw * 10) / 10
  if (out < 0) return 0
  if (out > 100) return 100
  return out
}

const clamp = (value: number, min: number, max: number): number => Math.min(max, Math.max(min, value))

const computeDefaultCanvasHostHeightPx = (): number => {
  const raw = Math.round(window.innerHeight * 0.7)
  return Math.min(720, Math.max(360, raw))
}

const initScreenCanvasLayout = (screen: any) => {
  const widgets: ShowcaseWidget[] = Array.isArray(screen.widgets) ? screen.widgets : []
  const cols = 2
  const rows = Math.max(1, Math.ceil(widgets.length / cols))
  const wPct = 100 / cols
  const hPct = 100 / rows

  for (let i = 0; i < widgets.length; i++) {
    const widget = widgets[i]
    const col = i % cols
    const row = Math.floor(i / cols)
    widget.layout.canvasPercent = {
      xPct: roundPct01(col * wPct),
      yPct: roundPct01(row * hPct),
      wPct: roundPct01(wPct),
      hPct: roundPct01(hPct)
    }
  }
}

const saveScreenLayout = async () => {
  const screen = showcase.currentScreen()
  if (!screen) return
  if (busy.value) return
  busy.value = true
  try {
    if (layoutForm.mode === "columns") {
      const maxColumns = parseIntInRange(layoutForm.maxColumns, "Max Columns", 1, 12)
      const minColumnWidth = parseIntInRange(layoutForm.minColumnWidth, "Min Column Width", 200, 1200)
      screen.layout.mode = "columns"
      screen.layout.columns.maxColumns = maxColumns
      screen.layout.columns.minColumnWidth = minColumnWidth
    } else {
      const prevMode = screen.layout.mode
      screen.layout.mode = "canvas_percent"

      const widgets = screen.widgets ?? []
      const hasAnyCanvas = widgets.some((w) => Boolean(w.layout?.canvasPercent))
      const hasAllCanvas = widgets.every((w) => Boolean(w.layout?.canvasPercent))

      if (!hasAnyCanvas) {
        const containerWidth = widgetsGridWidth.value || widgetsGridRef.value?.clientWidth || 960
        const baseWidth = Math.max(containerWidth, minCanvasWidgetWidthPx * 2)
        const rows = Math.max(1, Math.ceil(widgets.length / 2))
        const baseHeight = Math.max(computeDefaultCanvasHostHeightPx(), rows * minCanvasWidgetHeightPx)
        screen.layout.canvas.baseWidth = Math.round(baseWidth)
        screen.layout.canvas.baseHeight = Math.round(baseHeight)
        initScreenCanvasLayout(screen)
      } else if (!hasAllCanvas && prevMode !== "canvas_percent") {
        for (const widget of widgets) {
          if (widget.layout?.canvasPercent) continue
          widget.layout.canvasPercent = { xPct: 0, yPct: 0, wPct: 50, hPct: 10 }
        }
      }
    }

    await showcase.save()
    toast.success("Layout saved.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to save layout.")
    syncLayoutFormFromScreen()
  } finally {
    busy.value = false
  }
}

const widgetsGridRef = ref<HTMLElement | null>(null)
const widgetsGridWidth = ref(0)
const widgetsGridHeight = ref(0)
let widgetsGridObserver: ResizeObserver | null = null

const isCanvasMode = computed(() => showcase.currentScreen()?.layout?.mode === "canvas_percent")

const resolvedColumnsLayout = computed(() => showcase.currentScreen()?.layout?.columns ?? { maxColumns: 3, minColumnWidth: 360, gap: 16 })
const resolvedColumnsCount = computed(() => computeColumnsCount(widgetsGridWidth.value, resolvedColumnsLayout.value))
const widgetsGridStyle = computed(() => ({
  gridTemplateColumns: `repeat(${resolvedColumnsCount.value}, minmax(0, 1fr))`,
  gap: `${resolvedColumnsLayout.value.gap}px`
}))

const resolvedCanvasLayout = computed(() => showcase.currentScreen()?.layout?.canvas ?? { baseWidth: 960, baseHeight: 720 })
const canvasMetrics = computed(() => {
  const baseWidth = resolvedCanvasLayout.value.baseWidth > 0 ? resolvedCanvasLayout.value.baseWidth : 960
  const baseHeight = resolvedCanvasLayout.value.baseHeight > 0 ? resolvedCanvasLayout.value.baseHeight : 720
  const containerWidth = widgetsGridWidth.value > 0 ? widgetsGridWidth.value : baseWidth
  const containerHeight = widgetsGridHeight.value > 0 ? widgetsGridHeight.value : baseHeight
  const scaleX = baseWidth > 0 ? Math.min(1, containerWidth / baseWidth) : 1
  const scaleY = baseHeight > 0 ? Math.min(1, containerHeight / baseHeight) : 1
  const canvasWidth = Math.max(0, Math.round(baseWidth * scaleX))
  const canvasHeight = Math.max(0, Math.round(baseHeight * scaleY))
  return { canvasWidth, canvasHeight }
})

const canvasSurfaceStyle = computed(() => ({
  width: `${canvasMetrics.value.canvasWidth}px`,
  height: `${canvasMetrics.value.canvasHeight}px`
}))

const canvasWidgetStyle = (widget: ShowcaseWidget) => {
  const rect = widget.layout?.canvasPercent ?? { xPct: 0, yPct: 0, wPct: 50, hPct: 10 }
  const canvasWidth = canvasMetrics.value.canvasWidth
  const canvasHeight = canvasMetrics.value.canvasHeight

  const xPct = Number.isFinite(rect.xPct) ? rect.xPct : 0
  const yPct = Number.isFinite(rect.yPct) ? rect.yPct : 0
  const wPct = Number.isFinite(rect.wPct) ? rect.wPct : 50
  const hPct = Number.isFinite(rect.hPct) ? rect.hPct : 10

  const left = (xPct / 100) * canvasWidth
  const top = (yPct / 100) * canvasHeight
  const width = (wPct / 100) * canvasWidth
  const height = (hPct / 100) * canvasHeight

  return {
    left: `${left}px`,
    top: `${top}px`,
    width: `${width}px`,
    height: `${height}px`
  }
}

const widgetCardStyle = (widget: ShowcaseWidget) => {
  const span = clampColSpan(widget.layout?.colSpan ?? 1, resolvedColumnsCount.value)
  return {
    gridColumn: `span ${span} / span ${span}`
  }
}

const setupWidgetsGridObserver = () => {
  widgetsGridObserver?.disconnect()
  widgetsGridObserver = null
  const el = widgetsGridRef.value
  if (!el) return
  widgetsGridWidth.value = el.clientWidth
  widgetsGridHeight.value = el.clientHeight
  widgetsGridObserver = new ResizeObserver((entries) => {
    const entry = entries[0]
    if (!entry) return
    widgetsGridWidth.value = entry.contentRect.width
    widgetsGridHeight.value = entry.contentRect.height
  })
  widgetsGridObserver.observe(el)
}

const dragState = reactive({
  draggingId: "",
  overId: ""
})

const onDragStart = (widgetId: string, event: DragEvent) => {
  closeWidgetContextMenu()
  dragState.draggingId = widgetId
  dragState.overId = ""
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = "move"
    event.dataTransfer.setData("text/plain", widgetId)
  }
}

const onDragOver = (widgetId: string) => {
  if (!dragState.draggingId) return
  if (dragState.draggingId === widgetId) return
  dragState.overId = widgetId
}

const onDragEnd = () => {
  dragState.draggingId = ""
  dragState.overId = ""
}

const onDrop = async (widgetId: string) => {
  const fromId = dragState.draggingId
  onDragEnd()
  if (!fromId || fromId === widgetId) return
  if (busy.value) return

  const screen = showcase.currentScreen()
  if (!screen) return
  const widgets = screen.widgets
  const fromIndex = widgets.findIndex((w) => w.id === fromId)
  const toIndex = widgets.findIndex((w) => w.id === widgetId)
  if (fromIndex < 0 || toIndex < 0 || fromIndex === toIndex) return

  const next = widgets.slice()
  const [moved] = next.splice(fromIndex, 1)
  const insertAt = fromIndex < toIndex ? toIndex - 1 : toIndex
  next.splice(insertAt, 0, moved)
  screen.widgets = next

  busy.value = true
  try {
    await showcase.save()
    toast.success("Reordered.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to reorder widgets.")
    await showcase.load()
  } finally {
    busy.value = false
  }
}

const canvasSurfaceRef = ref<HTMLElement | null>(null)

type CanvasEditMode = "move" | "resize"
type CanvasRect = { xPct: number; yPct: number; wPct: number; hPct: number }

const canvasEdit = reactive<{
  active: boolean
  mode: CanvasEditMode
  widgetId: string
  widgetIndex: number
  startClientX: number
  startClientY: number
  startRect: CanvasRect
  canvasWidthPx: number
  canvasHeightPx: number
  minWPct: number
  minHPct: number
  changed: boolean
}>({
  active: false,
  mode: "move",
  widgetId: "",
  widgetIndex: -1,
  startClientX: 0,
  startClientY: 0,
  startRect: { xPct: 0, yPct: 0, wPct: 50, hPct: 10 },
  canvasWidthPx: 0,
  canvasHeightPx: 0,
  minWPct: 0,
  minHPct: 0,
  changed: false
})

const detachCanvasListeners = () => {
  window.removeEventListener("pointermove", onCanvasPointerMove)
  window.removeEventListener("pointerup", onCanvasPointerUp)
  window.removeEventListener("pointercancel", onCanvasPointerCancel)
}

const ensureWidgetCanvasRect = (widget: ShowcaseWidget): CanvasRect => {
  if (!widget.layout.canvasPercent) {
    widget.layout.canvasPercent = { xPct: 0, yPct: 0, wPct: 50, hPct: 10 }
  }
  return widget.layout.canvasPercent as CanvasRect
}

const canvasMinPct = () => {
  const baseWidth = resolvedCanvasLayout.value.baseWidth > 0 ? resolvedCanvasLayout.value.baseWidth : 960
  const baseHeight = resolvedCanvasLayout.value.baseHeight > 0 ? resolvedCanvasLayout.value.baseHeight : 720
  return {
    minWPct: ceilPct01((minCanvasWidgetWidthPx / baseWidth) * 100),
    minHPct: ceilPct01((minCanvasWidgetHeightPx / baseHeight) * 100)
  }
}

const startCanvasEdit = (widget: ShowcaseWidget, mode: CanvasEditMode, event: PointerEvent) => {
  if (!isCanvasMode.value) return
  if (busy.value) return
  const screen = showcase.currentScreen()
  if (!screen || screen.layout.mode !== "canvas_percent") return
  const idx = screen.widgets.findIndex((w) => w.id === widget.id)
  if (idx < 0) return

  const canvasEl = canvasSurfaceRef.value
  if (!canvasEl) return
  const canvasRect = canvasEl.getBoundingClientRect()
  if (!canvasRect.width || !canvasRect.height) return

  closeWidgetContextMenu()
  event.preventDefault()
  event.stopPropagation()

  const rect = ensureWidgetCanvasRect(widget)
  canvasEdit.active = true
  canvasEdit.mode = mode
  canvasEdit.widgetId = widget.id
  canvasEdit.widgetIndex = idx
  canvasEdit.startClientX = event.clientX
  canvasEdit.startClientY = event.clientY
  canvasEdit.startRect = { xPct: rect.xPct, yPct: rect.yPct, wPct: rect.wPct, hPct: rect.hPct }
  canvasEdit.canvasWidthPx = canvasRect.width
  canvasEdit.canvasHeightPx = canvasRect.height
  const { minWPct, minHPct } = canvasMinPct()
  canvasEdit.minWPct = minWPct
  canvasEdit.minHPct = minHPct
  canvasEdit.changed = false

  detachCanvasListeners()
  window.addEventListener("pointermove", onCanvasPointerMove)
  window.addEventListener("pointerup", onCanvasPointerUp)
  window.addEventListener("pointercancel", onCanvasPointerCancel)
}

const endCanvasEdit = async (save: boolean) => {
  if (!canvasEdit.active) return
  detachCanvasListeners()
  const shouldSave = Boolean(save && canvasEdit.changed)

  canvasEdit.active = false
  canvasEdit.widgetId = ""
  canvasEdit.widgetIndex = -1
  canvasEdit.changed = false

  if (!shouldSave) return
  if (busy.value) return

  busy.value = true
  try {
    await showcase.save()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to save canvas layout.")
    await showcase.load()
  } finally {
    busy.value = false
  }
}

const onCanvasPointerMove = (event: PointerEvent) => {
  if (!canvasEdit.active) return
  const screen = showcase.currentScreen()
  if (!screen || screen.layout.mode !== "canvas_percent") return
  let widget = screen.widgets[canvasEdit.widgetIndex]
  if (!widget || widget.id !== canvasEdit.widgetId) {
    widget = screen.widgets.find((w) => w.id === canvasEdit.widgetId)
  }
  if (!widget) return

  const dxPx = event.clientX - canvasEdit.startClientX
  const dyPx = event.clientY - canvasEdit.startClientY
  if (!canvasEdit.canvasWidthPx || !canvasEdit.canvasHeightPx) return

  const dxPct = (dxPx / canvasEdit.canvasWidthPx) * 100
  const dyPct = (dyPx / canvasEdit.canvasHeightPx) * 100
  const start = canvasEdit.startRect

  const rect = ensureWidgetCanvasRect(widget)
  const { minWPct, minHPct } = canvasEdit

  if (canvasEdit.mode === "move") {
    const maxX = 100 - start.wPct
    const maxY = 100 - start.hPct
    rect.xPct = roundPct01(clamp(start.xPct + dxPct, 0, Math.max(0, maxX)))
    rect.yPct = roundPct01(clamp(start.yPct + dyPct, 0, Math.max(0, maxY)))
    canvasEdit.changed = true
    return
  }

  const maxW = Math.max(0, 100 - start.xPct)
  const maxH = Math.max(0, 100 - start.yPct)
  const minW = Math.min(minWPct, maxW)
  const minH = Math.min(minHPct, maxH)
  rect.wPct = roundPct01(clamp(start.wPct + dxPct, minW, maxW))
  rect.hPct = roundPct01(clamp(start.hPct + dyPct, minH, maxH))
  canvasEdit.changed = true
}

const onCanvasPointerUp = () => {
  void endCanvasEdit(true)
}

const onCanvasPointerCancel = () => {
  void endCanvasEdit(false)
}

const reorderWidgetZOrder = async (widgetId: string, direction: "front" | "back") => {
  const screen = showcase.currentScreen()
  if (!screen) return
  const idx = screen.widgets.findIndex((w) => w.id === widgetId)
  if (idx < 0) return
  if (busy.value) return

  const next = screen.widgets.slice()
  const [moved] = next.splice(idx, 1)
  if (!moved) return
  if (direction === "front") {
    next.push(moved)
  } else {
    next.unshift(moved)
  }
  screen.widgets = next

  busy.value = true
  try {
    await showcase.save()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to reorder widgets.")
    await showcase.load()
  } finally {
    busy.value = false
  }
}

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  () => {
    showcase.setIdentity(selfNodeId.value, hubId.value)
  }
)

watch(
  () => profileStore.state.current,
  async () => {
    await showcase.leave()
    await loadHomeDefaults()
    await showcase.load()
    syncLayoutFormFromScreen()
    await showcase.enter()
  }
)

watch(
  () => showcase.state.lastLoadedAt,
  () => {
    syncLayoutFormFromScreen()
  }
)

watch(
  () => showcase.state.config.currentScreenId,
  () => {
    syncLayoutFormFromScreen()
  }
)

watch(
  () => widgetsGridRef.value,
  () => {
    setupWidgetsGridObserver()
  }
)

watch(
  () => widgetDialog.varMode,
  (nextMode) => {
    if (widgetDialog.mode !== "create") {
      lastModeForType.value = nextMode
      return
    }
    const current = widgetDialog.varType.trim()
    const prevDefault = defaultTypeForMode(lastModeForType.value)
    if (!current || current === prevDefault) {
      widgetDialog.varType = defaultTypeForMode(nextMode)
    }
    lastModeForType.value = nextMode
  }
)

onMounted(async () => {
  await loadHomeDefaults()
  try {
    await showcase.load()
    syncLayoutFormFromScreen()
    await showcase.enter()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to load showcase config.")
  }

  setupWidgetsGridObserver()

  window.addEventListener("keydown", onGlobalKeydown)
  window.addEventListener("resize", closeWidgetContextMenu)
  window.addEventListener("scroll", closeWidgetContextMenu, true)
})

onBeforeUnmount(() => {
  detachCanvasListeners()
  window.removeEventListener("keydown", onGlobalKeydown)
  window.removeEventListener("resize", closeWidgetContextMenu)
  window.removeEventListener("scroll", closeWidgetContextMenu, true)
  widgetsGridObserver?.disconnect()
  void showcase.leave()
})
</script>

<template>
  <section class="grid gap-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">Showcase</p>
        <h3 class="mt-2 text-lg font-semibold">Screens & Widgets</h3>
      </div>
      <Badge variant="secondary" :class="connectedTone">{{ connectedLabel }}</Badge>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <div class="flex flex-wrap items-center gap-2">
        <Tooltip content="Refresh Vars" side="bottom">
          <Button size="icon" :disabled="busy" @click="refreshVars">
            <RefreshCw class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">Refresh Vars</span>
          </Button>
        </Tooltip>
        <Tooltip content="New Screen" side="bottom">
          <Button size="icon" variant="outline" :disabled="busy" @click="promptCreateScreen">
            <Plus class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">New Screen</span>
          </Button>
        </Tooltip>
        <Tooltip content="Rename Screen" side="bottom">
          <Button size="icon" variant="outline" :disabled="busy" @click="promptRenameScreen">
            <Pencil class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">Rename Screen</span>
          </Button>
        </Tooltip>
        <Tooltip content="Delete Screen" side="bottom">
          <Button size="icon" variant="outline" :disabled="busy" @click="deleteCurrentScreen">
            <Trash2 class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">Delete Screen</span>
          </Button>
        </Tooltip>
      </div>
      <div class="mx-1 h-6 w-px bg-border/60" aria-hidden="true" />
      <div class="flex flex-wrap items-center gap-2">
        <Tooltip content="Add Event" side="bottom">
          <Button size="icon" :disabled="busy" @click="openCreateWidget('topic_button')">
            <Rss class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">Add Event</span>
          </Button>
        </Tooltip>
        <Tooltip content="Add Var" side="bottom">
          <Button size="icon" variant="outline" :disabled="busy" @click="openCreateWidget('var')">
            <Database class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">Add Var</span>
          </Button>
        </Tooltip>
      </div>
    </div>

    <div class="grid gap-6 lg:grid-cols-[0.35fr_0.65fr]">
      <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <div class="flex items-center justify-between gap-3">
          <h4 class="text-sm font-semibold">Screens</h4>
          <Badge variant="outline">{{ showcase.state.config.screens.length }}</Badge>
        </div>
        <div class="mt-4 space-y-2">
          <button
            v-for="screen in showcase.state.config.screens"
            :key="screen.id"
            type="button"
            class="flex w-full items-center justify-between gap-3 rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-left text-sm transition hover:border-primary"
            :class="showcase.state.config.currentScreenId === screen.id ? 'border-primary text-foreground' : 'text-muted-foreground'"
            @click="showcase.selectScreen(screen.id)"
          >
            <span class="truncate font-semibold">{{ screen.name }}</span>
            <span class="text-xs">{{ screen.widgets.length }}</span>
          </button>
        </div>
      </div>

      <div class="space-y-4">
        <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">Current</p>
              <h4 class="mt-2 text-lg font-semibold">{{ showcase.currentScreen()?.name || "Screen" }}</h4>
              <p class="mt-2 text-xs text-muted-foreground">
                Self={{ selfNodeId || "-" }} · Hub={{ hubId || "-" }} · LastVar={{ showcase.state.lastFrameAt || "-" }}
              </p>
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <Button size="sm" variant="outline" :disabled="busy" @click="openShowcaseWindow">
                <ExternalLink class="mr-2 h-4 w-4" />
                Open Window
              </Button>
            </div>
          </div>

          <div class="mt-5 grid gap-4 sm:grid-cols-4">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Layout Mode
              </label>
              <select v-model="layoutForm.mode" :class="inputClass">
                <option value="columns">columns</option>
                <option value="canvas_percent">canvas_percent</option>
              </select>
            </div>
            <div v-if="layoutForm.mode === 'columns'">
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Max Columns
              </label>
              <input v-model="layoutForm.maxColumns" :class="inputClass" />
            </div>
            <div v-if="layoutForm.mode === 'columns'">
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Min Column Width (px)
              </label>
              <input v-model="layoutForm.minColumnWidth" :class="inputClass" />
            </div>
            <div class="flex items-end">
              <Button size="sm" :disabled="busy" @click="saveScreenLayout">Save Layout</Button>
            </div>
          </div>
          <p v-if="layoutForm.mode === 'canvas_percent'" class="mt-3 text-xs text-muted-foreground">
            Canvas mode: drag the handle to move, use the bottom-right handle to resize, and right-click for z-order.
          </p>
        </div>

        <div v-if="!isCanvasMode" ref="widgetsGridRef" class="grid" :style="widgetsGridStyle">
          <div
            v-for="widget in showcase.currentScreen()?.widgets || []"
            :key="widget.id"
            class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm"
            :class="dragState.overId === widget.id ? 'ring-2 ring-primary/40' : ''"
            :style="widgetCardStyle(widget)"
            @contextmenu.prevent="openWidgetContextMenu(widget, $event)"
            @dragover.prevent="onDragOver(widget.id)"
            @drop.prevent="onDrop(widget.id)"
          >
            <div class="grid grid-cols-[minmax(0,1fr)_minmax(0,2fr)] items-center gap-4">
              <div class="flex min-w-0 items-center gap-3">
                <button
                  type="button"
                  class="inline-flex h-9 w-9 cursor-grab items-center justify-center rounded-lg border border-border/60 bg-background/70 text-muted-foreground hover:text-foreground active:cursor-grabbing"
                  draggable="true"
                  title="Drag to reorder"
                  @dragstart="onDragStart(widget.id, $event)"
                  @dragend="onDragEnd"
                >
                  <GripVertical class="h-4 w-4" />
                </button>

                <h5 class="min-w-0 flex-1 truncate whitespace-nowrap text-sm font-semibold" :title="safeTitle(widget)">
                  {{ safeTitle(widget) }}
                </h5>
              </div>

              <div class="min-w-0">
                <div v-if="widget.kind === 'topic_button' && widget.topicButton" class="flex justify-end">
                  <Button :disabled="busy || !sessionStore.connected || !selfNodeId" @click="sendTopicButton(widget)">
                    Send
                  </Button>
                </div>

                <div v-else-if="widget.kind === 'var' && widget.var">
                  <div
                    v-if="showcase.resolveEffectiveMode(widget) === 'display'"
                    class="min-w-0 truncate whitespace-nowrap text-right text-sm text-muted-foreground"
                    :title="displayValueText(widget)"
                  >
                    {{ displayValueText(widget) }}
                  </div>

                  <div v-else-if="showcase.resolveEffectiveMode(widget) === 'switch'" class="flex justify-end">
                    <input
                      type="checkbox"
                      class="h-4 w-4 rounded"
                      :checked="isVarOn(widget)"
                      :disabled="busy || !sessionStore.connected || !selfNodeId"
                      @change="showcase.switchToggle(widget, ($event.target as HTMLInputElement).checked)"
                    />
                  </div>

                  <div v-else class="flex flex-wrap items-center justify-end gap-3">
                    <input
                      class="min-w-[min(180px,100%)] flex-1"
                      type="range"
                      :min="widget.var.slider.min"
                      :max="widget.var.slider.max"
                      :step="widget.var.slider.step"
                      :value="showcase.sliderValue(widget)"
                      :disabled="busy || !sessionStore.connected || !selfNodeId"
                      @input="showcase.sliderInput(widget, Number(($event.target as HTMLInputElement).value))"
                      @change="showcase.sliderCommit(widget)"
                    />
                    <Badge variant="outline" class="shrink-0">{{ showcase.sliderValue(widget) }}</Badge>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div
            v-if="(showcase.currentScreen()?.widgets || []).length === 0"
            class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm"
            :style="{ gridColumn: '1 / -1' }"
          >
            <p class="text-sm text-muted-foreground">No widgets yet.</p>
          </div>
        </div>

        <div
          v-else
          ref="widgetsGridRef"
          class="relative flex h-[min(70vh,720px)] min-h-[360px] w-full items-center justify-center overflow-hidden rounded-2xl border bg-card/90 p-2 text-card-foreground shadow-sm"
        >
          <div ref="canvasSurfaceRef" class="relative" :style="canvasSurfaceStyle">
            <div
              v-for="widget in showcase.currentScreen()?.widgets || []"
              :key="widget.id"
              class="absolute overflow-hidden rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm"
              :class="canvasEdit.active && canvasEdit.widgetId === widget.id ? 'ring-2 ring-primary/40' : ''"
              :style="canvasWidgetStyle(widget)"
              @contextmenu.prevent="openWidgetContextMenu(widget, $event)"
            >
              <div class="grid h-full grid-cols-[minmax(0,1fr)_minmax(0,2fr)] items-center gap-4">
                <div class="flex min-w-0 items-center gap-3">
                  <button
                    type="button"
                    class="inline-flex h-9 w-9 cursor-grab touch-none items-center justify-center rounded-lg border border-border/60 bg-background/70 text-muted-foreground hover:text-foreground active:cursor-grabbing"
                    title="Drag to move"
                    @pointerdown.stop.prevent="startCanvasEdit(widget, 'move', $event)"
                  >
                    <GripVertical class="h-4 w-4" />
                  </button>

                  <h5 class="min-w-0 flex-1 truncate whitespace-nowrap text-sm font-semibold" :title="safeTitle(widget)">
                    {{ safeTitle(widget) }}
                  </h5>
                </div>

                <div class="min-w-0">
                  <div v-if="widget.kind === 'topic_button' && widget.topicButton" class="flex justify-end">
                    <Button :disabled="busy || !sessionStore.connected || !selfNodeId" @click="sendTopicButton(widget)">
                      Send
                    </Button>
                  </div>

                  <div v-else-if="widget.kind === 'var' && widget.var">
                    <div
                      v-if="showcase.resolveEffectiveMode(widget) === 'display'"
                      class="min-w-0 truncate whitespace-nowrap text-right text-sm text-muted-foreground"
                      :title="displayValueText(widget)"
                    >
                      {{ displayValueText(widget) }}
                    </div>

                    <div v-else-if="showcase.resolveEffectiveMode(widget) === 'switch'" class="flex justify-end">
                      <input
                        type="checkbox"
                        class="h-4 w-4 rounded"
                        :checked="isVarOn(widget)"
                        :disabled="busy || !sessionStore.connected || !selfNodeId"
                        @change="showcase.switchToggle(widget, ($event.target as HTMLInputElement).checked)"
                      />
                    </div>

                    <div v-else class="flex flex-wrap items-center justify-end gap-3">
                      <input
                        class="min-w-[min(180px,100%)] flex-1"
                        type="range"
                        :min="widget.var.slider.min"
                        :max="widget.var.slider.max"
                        :step="widget.var.slider.step"
                        :value="showcase.sliderValue(widget)"
                        :disabled="busy || !sessionStore.connected || !selfNodeId"
                        @input="showcase.sliderInput(widget, Number(($event.target as HTMLInputElement).value))"
                        @change="showcase.sliderCommit(widget)"
                      />
                      <Badge variant="outline" class="shrink-0">{{ showcase.sliderValue(widget) }}</Badge>
                    </div>
                  </div>
                </div>
              </div>

              <button
                type="button"
                class="absolute bottom-2 right-2 h-4 w-4 cursor-nwse-resize touch-none rounded-sm border border-border/60 bg-background/70"
                title="Resize"
                @pointerdown.stop.prevent="startCanvasEdit(widget, 'resize', $event)"
              />
            </div>

            <div
              v-if="(showcase.currentScreen()?.widgets || []).length === 0"
              class="absolute inset-0 flex items-center justify-center"
            >
              <p class="text-sm text-muted-foreground">No widgets yet.</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div
        v-if="widgetContextMenu.open"
        class="fixed inset-0 z-40"
        @pointerdown="closeWidgetContextMenu"
        @contextmenu.prevent="closeWidgetContextMenu"
      />
      <div
        v-if="widgetContextMenu.open"
        ref="widgetContextMenuRef"
        class="fixed z-50 w-44 rounded-xl border border-border/60 bg-card/95 p-1 text-sm shadow-xl backdrop-blur"
        role="menu"
        aria-label="Widget actions"
        :style="{ left: `${widgetContextMenu.x}px`, top: `${widgetContextMenu.y}px` }"
        @pointerdown.stop
        @click.stop
        @contextmenu.prevent
      >
        <button
          type="button"
          class="flex w-full items-center rounded-lg px-3 py-2 text-left hover:bg-muted/60 disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="busy || !widgetContextMenu.widget"
          role="menuitem"
          @click="onWidgetContextMenuEdit"
        >
          Edit
        </button>

        <template v-if="isCanvasMode">
          <div class="my-1 h-px bg-border/60" />
          <button
            type="button"
            class="flex w-full items-center rounded-lg px-3 py-2 text-left hover:bg-muted/60 disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="busy || !widgetContextMenu.widget"
            role="menuitem"
            @click="onWidgetContextMenuBringToFront"
          >
            Bring to Front
          </button>
          <button
            type="button"
            class="flex w-full items-center rounded-lg px-3 py-2 text-left hover:bg-muted/60 disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="busy || !widgetContextMenu.widget"
            role="menuitem"
            @click="onWidgetContextMenuSendToBack"
          >
            Send to Back
          </button>
        </template>

        <div class="my-1 h-px bg-border/60" />
        <button
          type="button"
          class="flex w-full items-center rounded-lg px-3 py-2 text-left text-rose-700 hover:bg-muted/60 disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="busy || !widgetContextMenu.widget"
          role="menuitem"
          @click="onWidgetContextMenuRemove"
        >
          Remove
        </button>
      </div>
    </Teleport>

    <Overlay :open="widgetDialog.open" overlayClass="bg-black/40 p-4" closeOnBackdrop @close="closeWidgetDialog">
      <div class="w-full max-w-2xl rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl">
        <div class="flex items-start justify-between gap-3">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
              {{ widgetDialog.kind === "topic_button" ? "TopicBus" : "VarStore" }}
            </p>
            <h3 class="mt-2 text-lg font-semibold">
              {{ widgetDialog.mode === "create" ? "Add Widget" : "Edit Widget" }}
            </h3>
          </div>
          <Badge variant="secondary">{{ widgetDialog.kind }}</Badge>
        </div>

        <div class="mt-5 grid gap-4">
          <div class="grid gap-4" :class="widgetDialog.kind === 'topic_button' ? 'sm:grid-cols-3' : 'sm:grid-cols-2'">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Title (optional)
              </label>
              <input v-model="widgetDialog.title" :class="inputClass" />
            </div>
            <div v-if="widgetDialog.kind === 'topic_button'">
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Target ID
              </label>
              <input v-model="widgetDialog.targetId" :class="inputClass" :placeholder="String(selfNodeId || 1)" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Column Span
              </label>
              <input v-model="widgetDialog.colSpan" :class="inputClass" />
            </div>
          </div>

          <div v-if="widgetDialog.kind === 'topic_button'" class="grid gap-4">
            <div class="grid gap-4 sm:grid-cols-2">
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Topic</label>
                <input v-model="widgetDialog.topic" :class="inputClass" />
              </div>
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Name</label>
                <input v-model="widgetDialog.eventName" :class="inputClass" />
              </div>
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Payload (Auto)
              </label>
              <textarea v-model="widgetDialog.payloadText" :class="textAreaClass" rows="6" />
            </div>
          </div>

          <div v-else class="grid gap-4">
            <div class="grid gap-4 sm:grid-cols-3">
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  Owner NodeID
                </label>
                <input v-model="widgetDialog.ownerId" :class="inputClass" :placeholder="String(selfNodeId || '')" />
              </div>
              <div class="sm:col-span-2">
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  Variable Name
                </label>
                <input v-model="widgetDialog.varName" :class="inputClass" />
              </div>
            </div>

            <div class="grid gap-4 sm:grid-cols-2">
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Mode</label>
                <select v-model="widgetDialog.varMode" :class="inputClass">
                  <option value="auto">auto</option>
                  <option value="display">display</option>
                  <option value="slider">slider</option>
                  <option value="switch">switch</option>
                </select>
              </div>
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Visibility</label>
                <select v-model="widgetDialog.visibility" :class="inputClass">
                  <option value="public">public</option>
                  <option value="private">private</option>
                </select>
              </div>
            </div>

            <div class="grid gap-4 sm:grid-cols-5">
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Min</label>
                <input v-model="widgetDialog.sliderMin" :class="inputClass" />
              </div>
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Max</label>
                <input v-model="widgetDialog.sliderMax" :class="inputClass" />
              </div>
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Step</label>
                <input v-model="widgetDialog.sliderStep" :class="inputClass" />
              </div>
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  <Tooltip
                    content="Unit: milliseconds (ms). Set to 0 to disable throttling (sends on every drag update). This may cause congestion."
                    side="bottom"
                  >
                    <span class="inline-flex cursor-help items-center gap-1">
                      Throttle
                      <CircleHelp class="h-3.5 w-3.5" aria-hidden="true" />
                    </span>
                  </Tooltip>
                </label>
                <input v-model="widgetDialog.sliderThrottleMs" :class="inputClass" />
              </div>
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  Type
                </label>
                <input v-model="widgetDialog.varType" :class="inputClass" placeholder="float64 / bool / string" />
              </div>
            </div>

            <div v-if="widgetDialog.varMode === 'switch'" class="grid gap-4">
              <div class="h-px bg-border/60" />
              <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Switch Settings
              </p>
              <div class="grid gap-4 sm:grid-cols-2">
                <div>
                  <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    On Value
                  </label>
                  <input v-model="widgetDialog.switchOnValue" :class="inputClass" />
                </div>
                <div>
                  <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    Off Value
                  </label>
                  <input v-model="widgetDialog.switchOffValue" :class="inputClass" />
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <Button variant="outline" :disabled="busy" @click="closeWidgetDialog">Cancel</Button>
          <Button :disabled="busy" @click="submitWidgetDialog">Save</Button>
        </div>
      </div>
    </Overlay>
  </section>
</template>
