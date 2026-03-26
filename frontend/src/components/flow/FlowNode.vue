<script setup lang="ts">
import { computed } from "vue"
import { Handle, Position, type NodeProps } from "@vue-flow/core"
import { useI18n } from "@/i18n"
import { flowStatusLabelKey } from "@/stores/flow"

type FlowNodeStatus = {
  status?: string
  code?: number
  msg?: string
}

type FlowNodeData = {
  label?: string
  kind?: string
  meta?: string
  status?: FlowNodeStatus
}

const props = defineProps<NodeProps<FlowNodeData>>()
const { t } = useI18n()

const label = computed(() => props.data?.label?.trim() || props.id)
const kind = computed(() => props.data?.kind?.trim().toLowerCase() || "call")
const meta = computed(() => props.data?.meta?.trim() || "")
const status = computed(() => props.data?.status?.status?.trim() || "")
const statusCode = computed(() => Number(props.data?.status?.code ?? 0))
const statusMsg = computed(() => props.data?.status?.msg?.trim() || "")
const statusLabel = computed(() => t(flowStatusLabelKey(status.value)))

const statusTone = computed(() => {
  switch (status.value) {
    case "succeeded":
      return "border-emerald-200 bg-emerald-50 text-emerald-800"
    case "failed":
      return "border-rose-200 bg-rose-50 text-rose-800"
    case "running":
      return "border-sky-200 bg-sky-50 text-sky-800"
    case "queued":
      return "border-amber-200 bg-amber-50 text-amber-800"
    case "cancelled":
      return "border-slate-200 bg-slate-100 text-slate-800"
    default:
      return "border-border/60 bg-muted/40 text-muted-foreground"
  }
})

const kindLabel = computed(() => {
  switch (kind.value) {
    case "compose":
      return t("Compose")
    case "set_var":
      return t("Set Var")
    default:
      return t("Call")
  }
})
</script>

<template>
  <div
    class="min-w-[176px] rounded-xl border bg-background/95 px-3 py-2 shadow-sm transition-shadow"
    :class="selected ? 'ring-2 ring-primary/40 ring-offset-2 ring-offset-background shadow-md' : ''"
  >
    <Handle
      type="target"
      :position="props.targetPosition ?? Position.Left"
      :connectable="props.connectable"
      class="!-left-2.5 !h-4 !w-4 !border-2 !border-sky-700 !bg-sky-400 shadow-sm"
    />
    <Handle
      type="source"
      :position="props.sourcePosition ?? Position.Right"
      :connectable="props.connectable"
      class="!-right-2.5 !h-4 !w-4 !border-2 !border-emerald-700 !bg-emerald-400 shadow-sm"
    />

    <div class="mb-2 flex items-center justify-between gap-3 text-[10px] font-semibold uppercase tracking-[0.24em]">
      <span class="rounded-full border border-sky-200 bg-sky-50 px-2 py-0.5 text-sky-700">{{ t("In") }}</span>
      <span class="rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-emerald-700">{{ t("Out") }}</span>
    </div>

    <div class="flex items-start justify-between gap-2">
      <div class="min-w-0">
        <p class="truncate text-xs font-semibold text-foreground">{{ label }}</p>
        <p class="mt-1 truncate text-[10px] uppercase tracking-[0.2em] text-muted-foreground">
          {{ kindLabel }}<span v-if="meta"> · {{ meta }}</span>
        </p>
      </div>
      <span class="shrink-0 rounded-full border px-2 py-0.5 text-[10px] font-semibold" :class="statusTone">
        {{ status ? statusLabel : t("Unknown") }}
      </span>
    </div>

    <p v-if="status" class="mt-1 truncate text-[10px] text-muted-foreground">
      {{ t("Code {code}", { code: statusCode }) }}{{ statusMsg ? ` · ${statusMsg}` : "" }}
    </p>
  </div>
</template>

