<script setup lang="ts">
import { computed } from "vue"
import { useToastStore, type ToastItem } from "@/stores/toast"

const toast = useToastStore()
const items = computed(() => toast.state.items)

const toneClass = (level: ToastItem["level"]) => {
  switch (level) {
    case "success":
      return "border-emerald-500/30 bg-emerald-500/10"
    case "info":
      return "border-sky-500/30 bg-sky-500/10"
    case "warn":
      return "border-amber-500/30 bg-amber-500/10"
    case "error":
      return "border-rose-500/30 bg-rose-500/10"
    default:
      return "border-border/60 bg-card/95"
  }
}
</script>

<template>
  <div
    class="pointer-events-none fixed left-1/2 top-4 z-[100] w-full max-w-xl -translate-x-1/2 space-y-2 px-4"
    aria-live="polite"
    aria-relevant="additions removals"
  >
    <div
      v-for="item in items"
      :key="item.id"
      class="pointer-events-auto rounded-xl border px-4 py-3 shadow-lg backdrop-blur"
      :class="toneClass(item.level)"
      role="status"
    >
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <p class="text-sm font-semibold text-foreground">
            {{ item.title }}
          </p>
          <p
            v-if="item.detail"
            class="mt-1 whitespace-pre-wrap break-words text-xs text-muted-foreground"
          >
            {{ item.detail }}
          </p>
        </div>

        <button
          type="button"
          class="rounded-md border border-border/60 bg-background/60 px-2 py-1 text-xs font-semibold text-muted-foreground transition hover:bg-muted/70 hover:text-foreground"
          aria-label="Close"
          @click="toast.remove(item.id)"
        >
          ✕
        </button>
      </div>
    </div>
  </div>
</template>

