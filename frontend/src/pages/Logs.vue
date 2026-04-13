<script setup lang="ts">
// Context: implements the Logs page in the Win frontend.
import { computed, nextTick, onMounted, ref, watch } from "vue"
import { Button } from "@/components/ui/button"
import LogItem from "@/components/logs/LogItem.vue"
import { useI18n } from "@/i18n"
import { openAuxWindow } from "@/lib/auxWindow"
import { useLogsStore } from "@/stores/logs"
import { useToastStore } from "@/stores/toast"

const logsStore = useLogsStore()
const toast = useToastStore()
const logRef = ref<HTMLElement | null>(null)
const { t } = useI18n()

const lineCount = computed(() => logsStore.state.lines.length)

const openLogWindow = async () => {
  const result = await openAuxWindow({
    routePath: "#/log-window",
    name: "log_window",
    size: "width=980,height=720"
  })
  if (result === "blocked") {
    toast.warn(t("Log window was blocked by browser popup policy."))
  }
}

const onPauseChange = async (event: Event) => {
  const target = event.target as HTMLInputElement
  try {
    await logsStore.setPaused(target.checked)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to update log pause state."))
    await logsStore.refreshPaused()
  }
}

const scrollToBottom = async () => {
  await nextTick()
  if (logRef.value) {
    logRef.value.scrollTop = logRef.value.scrollHeight
  }
}

watch(
  () => logsStore.state.lines.length,
  () => {
    if (!logsStore.state.paused) {
      void scrollToBottom()
    }
  }
)

onMounted(async () => {
  try {
    await logsStore.load()
    await logsStore.refreshPaused()
    await scrollToBottom()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to load logs."))
  }
})
</script>

<template>
  <section class="space-y-6">
    <div class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h2 class="text-sm font-semibold">{{ t("Logs") }}</h2>
        <div class="flex flex-wrap items-center gap-2">
          <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
            <span class="font-semibold uppercase tracking-[0.2em]">{{ t("Lines") }}</span>
            <span class="text-foreground">{{ lineCount }}</span>
          </div>
          <Button size="sm" variant="outline" @click="openLogWindow">{{ t("Open Window") }}</Button>
        </div>
      </div>

      <div class="mt-4 flex flex-wrap items-center justify-between gap-3">
        <label class="flex items-center gap-2 text-xs text-muted-foreground">
          <input
            type="checkbox"
            class="h-4 w-4 rounded border"
            :checked="logsStore.state.paused"
            @change="onPauseChange"
          />
          {{ t("Pause logs") }}
        </label>
      </div>

      <div
        ref="logRef"
        class="mt-4 max-h-[560px] overflow-y-auto rounded-xl border border-border/60 bg-background/70 p-4"
      >
        <div class="space-y-3">
          <LogItem v-for="line in logsStore.state.lines" :key="line.id" :line="line" />
        </div>
        <p v-if="logsStore.state.lines.length === 0" class="text-sm text-muted-foreground">
          {{ t("No logs yet.") }}
        </p>
      </div>
    </div>

  </section>
</template>
