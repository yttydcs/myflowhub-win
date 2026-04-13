<script setup lang="ts">
// Context: renders the showcase widget card content helper used by Showcase pages and windows.
import { computed, ref, watch } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  ensureShowcaseChartOption,
  normalizeShowcaseLineChartConfig,
  SHOWCASE_LINE_CHART_BUCKET_OPTIONS,
  SHOWCASE_LINE_CHART_DEFAULT_BUCKET_MS,
  SHOWCASE_LINE_CHART_DEFAULT_RANGE_MS,
  SHOWCASE_LINE_CHART_RANGE_OPTIONS
} from "@/lib/showcaseChart"
import { useI18n } from "@/i18n"
import { useShowcaseStore, type ShowcaseWidget } from "@/stores/showcase"

const props = withDefaults(
  defineProps<{
    widget: ShowcaseWidget
    busy?: boolean
    connected?: boolean
    selfNodeId?: number
    surface?: "columns" | "canvas"
  }>(),
  {
    busy: false,
    connected: false,
    selfNodeId: 0,
    surface: "columns"
  }
)

const emit = defineEmits<{
  (e: "send-topic"): void
  (e: "switch-change", value: boolean): void
  (e: "slider-input", value: number): void
  (e: "slider-commit"): void
}>()

const showcase = useShowcaseStore()
const { t } = useI18n()

const safeTitle = computed(() => {
  const title = props.widget.title?.trim()
  if (title) return title
  if (props.widget.kind === "topic_button" && props.widget.topicButton) {
    return `${props.widget.topicButton.topic} / ${props.widget.topicButton.name}`
  }
  if (props.widget.kind === "var" && props.widget.var) return props.widget.var.name
  return t("Widget")
})

const canInteract = computed(() => !props.busy && props.connected && !!props.selfNodeId)
const rawValue = computed(() => showcase.getVarValueText(props.widget))
const hasValue = computed(() => rawValue.value.trim().length > 0)
const displayValue = computed(() => (hasValue.value ? rawValue.value : t("No value yet.")))
const effectiveMode = computed(() => showcase.resolveEffectiveMode(props.widget))
const sliderValue = computed(() => showcase.sliderValue(props.widget))
const isLineChart = computed(() => effectiveMode.value === "line_chart")

const badgeStyle = computed(() => {
  const normalized = rawValue.value.trim().toLowerCase()
  if (!normalized) {
    return { variant: "outline" as const, className: "border-border/70 text-muted-foreground" }
  }
  if (
    normalized === "true" ||
    normalized === "on" ||
    normalized === "enabled" ||
    normalized === "connected" ||
    normalized === "ready" ||
    normalized === "online" ||
    normalized === "ok" ||
    normalized === "success" ||
    normalized === "healthy" ||
    normalized === "running"
  ) {
    return { variant: "outline" as const, className: "border-emerald-500/35 bg-emerald-500/10 text-emerald-700" }
  }
  if (
    normalized === "false" ||
    normalized === "off" ||
    normalized === "disabled" ||
    normalized === "disconnected" ||
    normalized === "error" ||
    normalized === "fail" ||
    normalized === "failed" ||
    normalized === "down" ||
    normalized === "offline" ||
    normalized === "critical" ||
    normalized === "stopped"
  ) {
    return { variant: "outline" as const, className: "border-rose-500/35 bg-rose-500/10 text-rose-700" }
  }
  if (normalized === "warning" || normalized === "warn" || normalized === "pending" || normalized === "idle" || normalized === "paused") {
    return { variant: "outline" as const, className: "border-amber-500/35 bg-amber-500/10 text-amber-700" }
  }
  return { variant: "outline" as const, className: "border-sky-500/35 bg-sky-500/10 text-sky-700" }
})

const emptyProgressState = (text: string) => ({
  ready: false,
  text,
  min: 0,
  max: 0,
  percent: 0,
  percentText: "0%"
})

const progressState = computed(() => {
  if (props.widget.kind !== "var" || !props.widget.var) {
    return emptyProgressState(t("No numeric value yet."))
  }
  if (!hasValue.value) {
    return emptyProgressState(t("No value yet."))
  }
  const parsed = Number.parseFloat(rawValue.value.trim())
  if (!Number.isFinite(parsed)) {
    return emptyProgressState(t("No numeric value yet."))
  }
  const min = Number(props.widget.var.slider.min ?? 0)
  const max = Number(props.widget.var.slider.max ?? 100)
  if (!Number.isFinite(min) || !Number.isFinite(max) || max <= min) {
    return emptyProgressState(t("Invalid range."))
  }
  const percent = Math.max(0, Math.min(100, ((parsed - min) / (max - min)) * 100))
  return {
    ready: true,
    min,
    max,
    percent,
    percentText: `${Math.round(percent)}%`
  }
})

const progressWrapClass = computed(() => (props.surface === "canvas" ? "w-full" : "w-full max-w-[260px]"))
const metricValueClass = computed(() => (hasValue.value ? "text-foreground" : "text-muted-foreground"))
const layoutClass = computed(() =>
  isLineChart.value ? "flex h-full min-h-0 flex-col gap-3" : "grid h-full grid-cols-[minmax(0,1fr)_minmax(0,2fr)] items-center gap-4"
)

const chartRangeMs = ref(String(SHOWCASE_LINE_CHART_DEFAULT_RANGE_MS))
const chartBucketMs = ref(String(SHOWCASE_LINE_CHART_DEFAULT_BUCKET_MS))

const syncChartControls = () => {
  const chart = normalizeShowcaseLineChartConfig(props.widget.var?.chart ?? null)
  chartRangeMs.value = String(chart.rangeMs)
  chartBucketMs.value = String(chart.bucketMs)
}

watch(
  () => [props.widget.id, props.widget.var?.mode, props.widget.var?.chart?.rangeMs, props.widget.var?.chart?.bucketMs],
  () => {
    syncChartControls()
  },
  { immediate: true }
)

const chartRangeOptions = computed(() =>
  ensureShowcaseChartOption(
    SHOWCASE_LINE_CHART_RANGE_OPTIONS,
    normalizeShowcaseLineChartConfig({
      rangeMs: Number.parseInt(chartRangeMs.value, 10),
      bucketMs: Number.parseInt(chartBucketMs.value, 10)
    }).rangeMs
  )
)
const chartBucketOptions = computed(() =>
  ensureShowcaseChartOption(
    SHOWCASE_LINE_CHART_BUCKET_OPTIONS,
    normalizeShowcaseLineChartConfig({
      rangeMs: Number.parseInt(chartRangeMs.value, 10),
      bucketMs: Number.parseInt(chartBucketMs.value, 10)
    }).bucketMs
  )
)

const lineChartModel = computed(() =>
  showcase.lineChartState(props.widget, {
    rangeMs: Number.parseInt(chartRangeMs.value, 10),
    bucketMs: Number.parseInt(chartBucketMs.value, 10)
  })
)

const chartWidth = computed(() => (props.surface === "canvas" ? 420 : 360))
const chartHeight = computed(() => (props.surface === "canvas" ? 150 : 132))

const lineChartSvg = computed(() => {
  const width = chartWidth.value
  const height = chartHeight.value
  const padX = 10
  const padY = 10
  const innerWidth = Math.max(1, width - padX * 2)
  const innerHeight = Math.max(1, height - padY * 2)
  const state = lineChartModel.value

  if (!state.ready || state.points.length < 2) {
    return { width, height, path: "", points: [] as Array<{ x: number; y: number }> }
  }

  const spanX = Math.max(1, state.toMs - state.fromMs)
  const spanY = Math.max(1e-9, state.yMax - state.yMin)
  const points = state.points.map((point) => {
    const x = padX + ((point.timestamp - state.fromMs) / spanX) * innerWidth
    const y = padY + innerHeight - ((point.value - state.yMin) / spanY) * innerHeight
    return { x, y }
  })
  const path = points.map((point, index) => `${index === 0 ? "M" : "L"} ${point.x.toFixed(2)} ${point.y.toFixed(2)}`).join(" ")
  return { width, height, path, points }
})

const formatAxisLabel = (timestampMs: number) => {
  const options: Intl.DateTimeFormatOptions =
    lineChartModel.value.chart.rangeMs >= 24 * 60 * 60 * 1000
      ? { month: "numeric", day: "numeric", hour: "2-digit", minute: "2-digit" }
      : { hour: "2-digit", minute: "2-digit" }
  return new Date(timestampMs).toLocaleString(undefined, options)
}

const lineChartStartLabel = computed(() => formatAxisLabel(lineChartModel.value.fromMs))
const lineChartEndLabel = computed(() => formatAxisLabel(lineChartModel.value.toMs))
</script>

<template>
  <div :class="layoutClass">
    <template v-if="!isLineChart">
      <div class="flex min-w-0 items-center gap-3">
        <slot name="leading" />
        <h5 class="min-w-0 flex-1 truncate whitespace-nowrap text-sm font-semibold" :title="safeTitle">
          {{ safeTitle }}
        </h5>
      </div>

      <div class="min-w-0">
        <div v-if="widget.kind === 'topic_button' && widget.topicButton" class="flex justify-end">
          <Button :disabled="!canInteract" @click="emit('send-topic')">
            {{ t("Send") }}
          </Button>
        </div>

        <div v-else-if="widget.kind === 'var' && widget.var">
          <div
            v-if="effectiveMode === 'display'"
            class="min-w-0 truncate whitespace-nowrap text-right text-sm text-muted-foreground"
            :title="displayValue"
          >
            {{ displayValue }}
          </div>

          <div v-else-if="effectiveMode === 'metric'" class="flex justify-end">
            <div class="min-w-0 text-right">
              <div class="text-[10px] font-semibold uppercase tracking-[0.24em] text-muted-foreground">
                {{ t("Live Value") }}
              </div>
              <div class="mt-1 truncate text-2xl font-semibold leading-none" :class="metricValueClass" :title="displayValue">
                {{ displayValue }}
              </div>
            </div>
          </div>

          <div v-else-if="effectiveMode === 'badge'" class="flex justify-end">
            <Badge :variant="badgeStyle.variant" :class="badgeStyle.className">
              {{ displayValue }}
            </Badge>
          </div>

          <div v-else-if="effectiveMode === 'progress'" class="flex justify-end">
            <div :class="progressWrapClass">
              <div v-if="progressState.ready" class="space-y-2 text-right">
                <div class="flex items-end justify-between gap-3">
                  <span class="min-w-0 truncate text-sm font-semibold" :title="rawValue">{{ rawValue }}</span>
                  <span class="shrink-0 text-xs text-muted-foreground">{{ progressState.percentText }}</span>
                </div>
                <div class="h-2.5 overflow-hidden rounded-full bg-muted/80">
                  <div class="h-full rounded-full bg-primary/80 transition-[width]" :style="{ width: `${progressState.percent}%` }" />
                </div>
                <div class="flex justify-between text-[11px] text-muted-foreground">
                  <span>{{ progressState.min }}</span>
                  <span>{{ progressState.max }}</span>
                </div>
              </div>
              <div v-else class="text-right text-sm text-muted-foreground">
                {{ progressState.text }}
              </div>
            </div>
          </div>

          <div v-else-if="effectiveMode === 'switch'" class="flex justify-end">
            <input
              type="checkbox"
              class="h-4 w-4 rounded"
              :checked="rawValue === widget.var.switch.onValue"
              :disabled="!canInteract"
              @change="emit('switch-change', ($event.target as HTMLInputElement).checked)"
            />
          </div>

          <div v-else class="flex flex-wrap items-center justify-end gap-3">
            <input
              class="min-w-[min(180px,100%)] flex-1"
              type="range"
              :min="widget.var.slider.min"
              :max="widget.var.slider.max"
              :step="widget.var.slider.step"
              :value="sliderValue"
              :disabled="!canInteract"
              @input="emit('slider-input', Number(($event.target as HTMLInputElement).value))"
              @change="emit('slider-commit')"
            />
            <Badge variant="outline" class="shrink-0">{{ sliderValue }}</Badge>
          </div>
        </div>
      </div>
    </template>

    <template v-else>
      <div class="flex min-w-0 items-start justify-between gap-3">
        <div class="flex min-w-0 items-center gap-3">
          <slot name="leading" />
          <h5 class="min-w-0 truncate text-sm font-semibold" :title="safeTitle">
            {{ safeTitle }}
          </h5>
        </div>
        <div class="min-w-0 text-right">
          <div class="text-[10px] font-semibold uppercase tracking-[0.24em] text-muted-foreground">
            {{ t("Live Value") }}
          </div>
          <div class="truncate text-lg font-semibold" :title="displayValue" :class="metricValueClass">
            {{ displayValue }}
          </div>
        </div>
      </div>

      <div class="flex flex-wrap items-center justify-end gap-2">
        <label class="flex items-center gap-2 text-[11px] text-muted-foreground">
          <span>{{ t("Range") }}</span>
          <select
            v-model="chartRangeMs"
            class="rounded-md border border-border/70 bg-background/90 px-2 py-1 text-xs text-foreground shadow-sm"
          >
            <option v-for="option in chartRangeOptions" :key="`range-${option.value}`" :value="String(option.value)">
              {{ option.label }}
            </option>
          </select>
        </label>
        <label class="flex items-center gap-2 text-[11px] text-muted-foreground">
          <span>{{ t("Granularity") }}</span>
          <select
            v-model="chartBucketMs"
            class="rounded-md border border-border/70 bg-background/90 px-2 py-1 text-xs text-foreground shadow-sm"
          >
            <option v-for="option in chartBucketOptions" :key="`bucket-${option.value}`" :value="String(option.value)">
              {{ option.label }}
            </option>
          </select>
        </label>
      </div>

      <div v-if="lineChartModel.ready" class="flex min-h-0 flex-1 flex-col gap-2">
        <div class="rounded-2xl border border-border/70 bg-muted/20 p-3">
          <svg class="h-auto w-full" :viewBox="`0 0 ${lineChartSvg.width} ${lineChartSvg.height}`" aria-hidden="true">
            <rect
              x="0"
              y="0"
              :width="lineChartSvg.width"
              :height="lineChartSvg.height"
              rx="12"
              fill="transparent"
            />
            <path d="" stroke="none" fill="none" />
            <path
              :d="lineChartSvg.path"
              fill="none"
              stroke="currentColor"
              stroke-width="2.25"
              class="text-primary"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
            <circle
              v-for="(point, index) in lineChartSvg.points"
              :key="`line-point-${index}`"
              :cx="point.x"
              :cy="point.y"
              r="2.5"
              class="fill-primary/80"
            />
          </svg>
        </div>
        <div class="flex items-center justify-between text-[11px] text-muted-foreground">
          <span>{{ lineChartStartLabel }}</span>
          <span>{{ lineChartEndLabel }}</span>
        </div>
      </div>

      <div
        v-else
        class="flex min-h-[120px] flex-1 items-center justify-center rounded-2xl border border-dashed border-border/70 bg-muted/15 px-4 text-center text-sm text-muted-foreground"
      >
        {{ lineChartModel.message }}
      </div>
    </template>
  </div>

  <slot name="overlay" />
</template>
