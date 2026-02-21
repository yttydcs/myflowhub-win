<script setup lang="ts">
import { computed } from "vue"
import { Handle, Position } from "@vue-flow/core"

type FlowNodeStatus = {
  status?: string
  code?: number
  msg?: string
}

const props = defineProps<{
  id: string
  data?: {
    label?: string
    status?: FlowNodeStatus
  }
  selected?: boolean
}>()

const label = computed(() => props.data?.label?.trim() || props.id)
const status = computed(() => props.data?.status?.status?.trim() || "")
const statusCode = computed(() => Number(props.data?.status?.code ?? 0))
const statusMsg = computed(() => props.data?.status?.msg?.trim() || "")

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
    default:
      return "border-border/60 bg-muted/40 text-muted-foreground"
  }
})
</script>

<template>
  <div
    class="min-w-[160px] rounded-xl border bg-background/90 px-3 py-2 shadow-sm"
    :class="selected ? 'ring-2 ring-primary/40 ring-offset-2 ring-offset-background' : ''"
  >
    <Handle type="target" :position="Position.Left" class="h-2 w-2 border border-border/60 bg-background" />
    <Handle type="source" :position="Position.Right" class="h-2 w-2 border border-border/60 bg-background" />

    <div class="flex items-start justify-between gap-2">
      <p class="truncate text-xs font-semibold text-foreground">{{ label }}</p>
      <span class="shrink-0 rounded-full border px-2 py-0.5 text-[10px] font-semibold" :class="statusTone">
        {{ status || "unknown" }}
      </span>
    </div>

    <p v-if="status" class="mt-1 truncate text-[10px] text-muted-foreground">
      code {{ statusCode }}{{ statusMsg ? ` · ${statusMsg}` : "" }}
    </p>
  </div>
</template>

