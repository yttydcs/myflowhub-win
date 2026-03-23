<script setup lang="ts">
import { computed } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
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
</script>

<template>
  <div class="grid h-full grid-cols-[minmax(0,1fr)_minmax(0,2fr)] items-center gap-4">
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
  </div>

  <slot name="overlay" />
</template>
