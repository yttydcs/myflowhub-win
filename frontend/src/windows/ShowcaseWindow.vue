<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue"
import { useRoute } from "vue-router"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { clampColSpan, computeColumnsCount } from "@/lib/showcaseLayout"
import { useProfileStore } from "@/stores/profile"
import { useSessionStore } from "@/stores/session"
import { useShowcaseStore, type ShowcaseWidget } from "@/stores/showcase"
import { useToastStore } from "@/stores/toast"
import { HomeState as LoadHomeState } from "../../wailsjs/go/main/App"

const route = useRoute()
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

const requestedScreenIdRaw = computed(() => String(route.query.screenId ?? "").trim())
const requestedScreenId = computed(() => requestedScreenIdRaw.value || "__missing_screen__")

const screen = computed(() => showcase.screenById(requestedScreenId.value))
const screenName = computed(() => screen.value?.name || "Showcase")
const screenMissing = computed(() => Boolean(showcase.state.loaded) && (Boolean(showcase.state.screenMissing) || !screen.value))

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

const displayValueText = (widget: ShowcaseWidget) => {
  const raw = showcase.getVarValueText(widget)
  if (raw.trim()) return raw
  return "No value yet."
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

const widgetsGridRef = ref<HTMLElement | null>(null)
const widgetsGridWidth = ref(0)
const widgetsGridHeight = ref(0)
let widgetsGridObserver: ResizeObserver | null = null

const resolvedColumnsLayout = computed(() => screen.value?.layout?.columns ?? { maxColumns: 3, minColumnWidth: 360, gap: 16 })
const resolvedColumnsCount = computed(() => computeColumnsCount(widgetsGridWidth.value, resolvedColumnsLayout.value))
const widgetsGridStyle = computed(() => ({
  gridTemplateColumns: `repeat(${resolvedColumnsCount.value}, minmax(0, 1fr))`,
  gap: `${resolvedColumnsLayout.value.gap}px`
}))

const resolvedCanvasLayout = computed(() => screen.value?.layout?.canvas ?? { baseWidth: 960, baseHeight: 720 })
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

const enterScreen = async () => {
  showcase.setFixedScreenId(requestedScreenId.value)
  await showcase.enter()
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
    await enterScreen()
  }
)

watch(
  () => requestedScreenId.value,
  async () => {
    await showcase.leave()
    await enterScreen()
  }
)

watch(
  () => widgetsGridRef.value,
  () => {
    setupWidgetsGridObserver()
  }
)

onMounted(async () => {
  showcase.setFixedScreenId(requestedScreenId.value)
  await loadHomeDefaults()
  try {
    await showcase.load()
    await enterScreen()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to load showcase config.")
  }
  setupWidgetsGridObserver()
})

onBeforeUnmount(() => {
  widgetsGridObserver?.disconnect()
  showcase.clearFixedScreenId()
  void showcase.leave()
})
</script>

<template>
  <section class="space-y-6">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h1 class="text-lg font-semibold">{{ screenName }}</h1>
        <p class="mt-2 text-xs text-muted-foreground">
          ScreenId={{ requestedScreenIdRaw || "-" }} · Self={{ selfNodeId || "-" }} · Hub={{ hubId || "-" }}
        </p>
      </div>
      <Badge variant="secondary" :class="connectedTone">{{ connectedLabel }}</Badge>
    </div>

    <div v-if="!showcase.state.loaded" class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
      <h2 class="text-base font-semibold">Loading...</h2>
      <p class="mt-2 text-sm text-muted-foreground">
        Loading Showcase config.
      </p>
    </div>

    <div v-else-if="screenMissing" class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
      <h2 class="text-base font-semibold">Screen not found</h2>
      <p class="mt-2 text-sm text-muted-foreground">
        The requested screen does not exist in the current Showcase config.
      </p>
    </div>

    <div v-else-if="screen?.layout?.mode === 'columns'" ref="widgetsGridRef" class="grid" :style="widgetsGridStyle">
      <div
        v-for="widget in screen?.widgets || []"
        :key="widget.id"
        class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm"
        :style="widgetCardStyle(widget)"
      >
        <div class="grid grid-cols-[minmax(0,1fr)_minmax(0,2fr)] items-center gap-4">
          <h5 class="min-w-0 truncate whitespace-nowrap text-sm font-semibold" :title="safeTitle(widget)">
            {{ safeTitle(widget) }}
          </h5>

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
        v-if="(screen?.widgets || []).length === 0"
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
      <div class="relative" :style="canvasSurfaceStyle">
        <div
          v-for="widget in screen?.widgets || []"
          :key="widget.id"
          class="absolute overflow-hidden rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm"
          :style="canvasWidgetStyle(widget)"
        >
          <div class="grid h-full grid-cols-[minmax(0,1fr)_minmax(0,2fr)] items-center gap-4">
            <h5 class="min-w-0 truncate whitespace-nowrap text-sm font-semibold" :title="safeTitle(widget)">
              {{ safeTitle(widget) }}
            </h5>

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
          v-if="(screen?.widgets || []).length === 0"
          class="absolute inset-0 flex items-center justify-center"
        >
          <p class="text-sm text-muted-foreground">No widgets yet.</p>
        </div>
      </div>
    </div>
  </section>
</template>
