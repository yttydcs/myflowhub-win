<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue"
import { ExternalLink, GripVertical } from "lucide-vue-next"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
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

const submitWidgetDialog = async () => {
  if (busy.value) return
  busy.value = true
  try {
    const title = widgetDialog.title.trim()
    const targetId = parsePositiveInt(widgetDialog.targetId, "Target ID")
    const screen = showcase.currentScreen()
    const maxColumns = screen?.layout?.columns?.maxColumns ?? 12
    const colSpan = parseIntInRange(widgetDialog.colSpan, "Column Span", 1, Math.max(1, maxColumns))

    if (widgetDialog.kind === "topic_button") {
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
    const throttleMs = parseNonNegativeInt(widgetDialog.sliderThrottleMs, "Throttle (ms)")
    const onValue = widgetDialog.switchOnValue.trim()
    const offValue = widgetDialog.switchOffValue.trim()
    if (!onValue || !offValue) throw new Error("Switch on/off values are required.")

    if (widgetDialog.mode === "create") {
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
        switch: { onValue, offValue }
      })
    } else {
      const widget = screen?.widgets.find((w) => w.id === widgetDialog.widgetId)
      if (!widget || widget.kind !== "var" || !widget.var) return
      widget.title = title
      widget.targetId = targetId
      widget.layout.colSpan = colSpan
      widget.var.ownerId = ownerId
      widget.var.name = varName
      widget.var.mode = mode
      widget.var.visibility = visibility
      widget.var.type = type
      widget.var.slider = { min: sliderMin, max: sliderMax, step: sliderStep, throttleMs }
      widget.var.switch = { onValue, offValue }
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

const safeTitle = (widget: ShowcaseWidget) => {
  const title = widget.title?.trim()
  if (title) return title
  if (widget.kind === "topic_button" && widget.topicButton) {
    return `${widget.topicButton.topic} / ${widget.topicButton.name}`
  }
  if (widget.kind === "var" && widget.var) return widget.var.name
  return "Widget"
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
  maxColumns: "3",
  minColumnWidth: "360"
})

const syncLayoutFormFromScreen = () => {
  const screen = showcase.currentScreen()
  if (!screen) return
  layoutForm.maxColumns = String(screen.layout?.columns?.maxColumns ?? 3)
  layoutForm.minColumnWidth = String(screen.layout?.columns?.minColumnWidth ?? 360)
}

const saveScreenLayout = async () => {
  const screen = showcase.currentScreen()
  if (!screen) return
  if (busy.value) return
  busy.value = true
  try {
    const maxColumns = parseIntInRange(layoutForm.maxColumns, "Max Columns", 1, 12)
    const minColumnWidth = parseIntInRange(layoutForm.minColumnWidth, "Min Column Width", 200, 1200)
    screen.layout.mode = "columns"
    screen.layout.columns.maxColumns = maxColumns
    screen.layout.columns.minColumnWidth = minColumnWidth
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
let widgetsGridObserver: ResizeObserver | null = null

const resolvedColumnsLayout = computed(() => showcase.currentScreen()?.layout?.columns ?? { maxColumns: 3, minColumnWidth: 360, gap: 16 })
const resolvedColumnsCount = computed(() => computeColumnsCount(widgetsGridWidth.value, resolvedColumnsLayout.value))
const widgetsGridStyle = computed(() => ({
  gridTemplateColumns: `repeat(${resolvedColumnsCount.value}, minmax(0, 1fr))`,
  gap: `${resolvedColumnsLayout.value.gap}px`
}))

const widgetCardStyle = (widget: ShowcaseWidget) => {
  const span = clampColSpan(widget.layout?.colSpan ?? 1, resolvedColumnsCount.value)
  return {
    gridColumn: `span ${span} / span ${span}`
  }
}

const dragState = reactive({
  draggingId: "",
  overId: ""
})

const onDragStart = (widgetId: string, event: DragEvent) => {
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
  () => showcase.state.config.currentScreenId,
  () => {
    syncLayoutFormFromScreen()
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

  if (widgetsGridRef.value) {
    widgetsGridWidth.value = widgetsGridRef.value.clientWidth
    widgetsGridObserver = new ResizeObserver((entries) => {
      const entry = entries[0]
      if (!entry) return
      widgetsGridWidth.value = entry.contentRect.width
    })
    widgetsGridObserver.observe(widgetsGridRef.value)
  }
})

onBeforeUnmount(() => {
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

    <div class="flex flex-wrap gap-2">
      <Button size="sm" :disabled="busy" @click="refreshVars">Refresh Vars</Button>
      <Button size="sm" variant="outline" :disabled="busy" @click="promptCreateScreen">New Screen</Button>
      <Button size="sm" variant="outline" :disabled="busy" @click="promptRenameScreen">Rename Screen</Button>
      <Button size="sm" variant="outline" :disabled="busy" @click="deleteCurrentScreen">Delete Screen</Button>
      <Button size="sm" :disabled="busy" @click="openCreateWidget('topic_button')">Add Event</Button>
      <Button size="sm" variant="outline" :disabled="busy" @click="openCreateWidget('var')">Add Var</Button>
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

          <div class="mt-5 grid gap-4 sm:grid-cols-3">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Max Columns
              </label>
              <input v-model="layoutForm.maxColumns" :class="inputClass" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Min Column Width (px)
              </label>
              <input v-model="layoutForm.minColumnWidth" :class="inputClass" />
            </div>
            <div class="flex items-end">
              <Button size="sm" :disabled="busy" @click="saveScreenLayout">Save Layout</Button>
            </div>
          </div>
        </div>

        <div ref="widgetsGridRef" class="grid" :style="widgetsGridStyle">
          <div
            v-for="widget in showcase.currentScreen()?.widgets || []"
            :key="widget.id"
            class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm"
            :class="dragState.overId === widget.id ? 'ring-2 ring-primary/40' : ''"
            :style="widgetCardStyle(widget)"
            @dragover.prevent="onDragOver(widget.id)"
            @drop.prevent="onDrop(widget.id)"
          >
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="flex items-start gap-3">
              <button
                type="button"
                class="mt-0.5 inline-flex h-9 w-9 cursor-grab items-center justify-center rounded-lg border border-border/60 bg-background/70 text-muted-foreground hover:text-foreground active:cursor-grabbing"
                draggable="true"
                title="Drag to reorder"
                @dragstart="onDragStart(widget.id, $event)"
                @dragend="onDragEnd"
              >
                <GripVertical class="h-4 w-4" />
              </button>

              <div>
                <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
                  {{ widget.kind === 'topic_button' ? 'TopicBus' : 'VarStore' }}
                </p>
                <h5 class="mt-2 text-base font-semibold">{{ safeTitle(widget) }}</h5>
                <p class="mt-1 text-xs text-muted-foreground">
                  Target={{ widget.targetId || "-" }} · Span={{ widget.layout?.colSpan || 1 }}
                </p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <Button size="sm" variant="outline" :disabled="busy" @click="openEditWidget(widget)">Edit</Button>
              <Button size="sm" variant="outline" :disabled="busy" @click="removeWidget(widget)">Remove</Button>
            </div>
          </div>

          <div v-if="widget.kind === 'topic_button' && widget.topicButton" class="mt-4 grid gap-3">
            <div class="rounded-xl border border-border/60 bg-background/70 p-4 text-xs text-muted-foreground">
              <pre class="whitespace-pre-wrap">{{ widget.topicButton.payloadText || "(empty payload)" }}</pre>
            </div>
            <Button :disabled="busy || !sessionStore.connected || !selfNodeId" @click="sendTopicButton(widget)">
              Send
            </Button>
          </div>

          <div v-else-if="widget.kind === 'var' && widget.var" class="mt-4 grid gap-4">
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-sm">
                <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Var</p>
                <p class="mt-1 font-medium break-all">
                  {{ widget.var.ownerId }} / {{ widget.var.name }}
                </p>
              </div>
              <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-sm">
                <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Value</p>
                <p class="mt-1 font-medium break-all">{{ showcase.getVarValueText(widget) || "-" }}</p>
              </div>
            </div>

            <div v-if="showcase.resolveEffectiveMode(widget) === 'display'" class="rounded-xl border border-border/60 bg-background/70 p-4">
              <pre class="whitespace-pre-wrap text-xs text-muted-foreground">
{{ showcase.getVarValueText(widget) || "No value yet." }}
              </pre>
            </div>

            <div v-else-if="showcase.resolveEffectiveMode(widget) === 'switch'" class="flex items-center justify-between rounded-xl border border-border/60 bg-background/70 px-4 py-3">
              <div class="text-sm text-muted-foreground">
                ON={{ widget.var.switch.onValue }} · OFF={{ widget.var.switch.offValue }}
              </div>
              <label class="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  class="h-4 w-4 rounded"
                  :checked="isVarOn(widget)"
                  :disabled="busy || !sessionStore.connected || !selfNodeId"
                  @change="showcase.switchToggle(widget, ($event.target as HTMLInputElement).checked)"
                />
                <span class="font-semibold">{{ isVarOn(widget) ? "ON" : "OFF" }}</span>
              </label>
            </div>

            <div v-else class="rounded-xl border border-border/60 bg-background/70 p-4">
              <div class="flex flex-wrap items-start justify-between gap-2">
                <div class="text-sm text-muted-foreground">
                  min={{ widget.var.slider.min }} · max={{ widget.var.slider.max }} · step={{ widget.var.slider.step }}
                  · throttle={{ widget.var.slider.throttleMs }}ms
                </div>
                <Badge variant="outline">{{ showcase.sliderValue(widget) }}</Badge>
              </div>
              <input
                class="mt-4 w-full"
                type="range"
                :min="widget.var.slider.min"
                :max="widget.var.slider.max"
                :step="widget.var.slider.step"
                :value="showcase.sliderValue(widget)"
                :disabled="busy || !sessionStore.connected || !selfNodeId"
                @input="showcase.sliderInput(widget, Number(($event.target as HTMLInputElement).value))"
                @change="showcase.sliderCommit(widget)"
              />
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
      </div>
    </div>

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
          <div class="grid gap-4 sm:grid-cols-3">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Title (optional)
              </label>
              <input v-model="widgetDialog.title" :class="inputClass" />
            </div>
            <div>
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
                  Throttle (ms)
                </label>
                <input v-model="widgetDialog.sliderThrottleMs" :class="inputClass" />
                <p class="mt-2 text-xs text-muted-foreground">
                  Set to 0 to disable throttling (sends on every drag update). This may cause congestion.
                </p>
              </div>
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  Type
                </label>
                <input v-model="widgetDialog.varType" :class="inputClass" placeholder="float64 / bool / string" />
              </div>
            </div>

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

        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <Button variant="outline" :disabled="busy" @click="closeWidgetDialog">Cancel</Button>
          <Button :disabled="busy" @click="submitWidgetDialog">Save</Button>
        </div>
      </div>
    </Overlay>
  </section>
</template>
