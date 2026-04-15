<script setup lang="ts">
// 本文件实现 Logs 页面使用的 `LogItem` 组件。
import { computed, ref } from "vue"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"
import type { LogLine } from "@/stores/logs"
import {
  buildPayloadText,
  buildTextPreview,
  bytesToSpacedHex,
  formatTimestamp,
  toByteArray,
  tryFormatJson
} from "@/lib/logs"

const props = defineProps<{
  line: LogLine
}>()
const { t } = useI18n()

const expanded = ref(false)
const hexMode = ref(false)
const jsonMode = ref(false)
const jsonError = ref("")
const formattedPayload = ref("")

const payloadBytes = computed(() => toByteArray(props.line.payload) ?? new Uint8Array())
const payloadLen = computed(() => {
  const explicit = Number(props.line.payloadLen ?? 0)
  if (explicit > 0) return explicit
  return payloadBytes.value.length
})

const timestamp = computed(() => formatTimestamp(props.line.time))
const levelLabel = computed(() => (props.line.level || "info").toUpperCase())

const previewPayload = computed(() => {
  const bytes = payloadBytes.value
  if (!bytes.length) return t("payload=empty")
  if (hexMode.value) {
    const previewBytes = bytes.slice(0, 80)
    const hex = bytesToSpacedHex(previewBytes)
    const suffix = bytes.length > previewBytes.length ? "..." : ""
    return t("payload=hex({value})", { value: `${hex}${suffix}` })
  }
  const { text, truncated } = buildTextPreview(bytes, 160)
  const suffix = props.line.payloadTruncated || truncated ? "..." : ""
  return t("payload=text({value})", { value: `${text}${suffix}` })
})

const previewLine = computed(() => {
  const msg = props.line.message || ""
  if (!msg) return previewPayload.value
  return `${msg} ${previewPayload.value}`
})

const payloadTextRaw = computed(() => buildPayloadText(payloadBytes.value))
const payloadText = computed(() => (jsonMode.value ? formattedPayload.value : payloadTextRaw.value))
const payloadHex = computed(() => bytesToSpacedHex(payloadBytes.value))

const toggleHex = () => {
  hexMode.value = !hexMode.value
}

const toggleJson = () => {
  jsonError.value = ""
  if (jsonMode.value) {
    jsonMode.value = false
    formattedPayload.value = ""
    return
  }
  const result = tryFormatJson(payloadTextRaw.value)
  if (!result.ok) {
    jsonError.value = result.error
    return
  }
  formattedPayload.value = result.formatted
  jsonMode.value = true
}
</script>

<template>
  <div class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="space-y-1">
        <div class="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
          <span class="font-semibold text-foreground">{{ timestamp || "-" }}</span>
          <span class="rounded-full border border-border/60 px-2 py-0.5 text-[10px] font-semibold">
            {{ levelLabel }}
          </span>
          <span class="text-foreground">{{ props.line.message || "-" }}</span>
        </div>
        <div class="text-xs text-muted-foreground">
          {{ t("Payload length: {count}", { count: payloadLen }) }}
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <Button size="sm" variant="outline" @click="toggleHex">
          {{ hexMode ? t("Hex On") : t("Hex Off") }}
        </Button>
        <Button size="sm" variant="outline" @click="expanded = !expanded">
          {{ expanded ? t("Collapse") : t("Expand") }}
        </Button>
      </div>
    </div>

    <div class="mt-2 truncate text-xs text-muted-foreground">
      {{ previewLine }}
    </div>

    <div v-if="expanded" class="mt-4 space-y-3">
      <div class="flex flex-wrap items-center gap-2">
        <Button size="sm" variant="outline" @click="toggleJson">
          {{ jsonMode ? t("Raw") : t("Format JSON") }}
        </Button>
        <span v-if="jsonError" class="text-xs text-rose-600">{{ jsonError }}</span>
        <span v-if="props.line.payloadTruncated" class="text-xs text-muted-foreground">
          {{ t("Payload truncated") }}
        </span>
      </div>

      <div class="rounded-xl border border-border/60 bg-background/70 p-3">
        <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
          {{ t("Payload Text") }}
        </p>
        <pre class="mt-2 whitespace-pre-wrap text-xs text-muted-foreground">
{{ payloadText || t("No payload.") }}
        </pre>
      </div>

      <div v-if="hexMode" class="rounded-xl border border-border/60 bg-background/70 p-3">
        <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
          {{ t("Payload Hex") }}
        </p>
        <pre class="mt-2 whitespace-pre-wrap text-xs text-muted-foreground">
{{ payloadHex || t("No payload.") }}
        </pre>
      </div>
    </div>
  </div>
</template>
