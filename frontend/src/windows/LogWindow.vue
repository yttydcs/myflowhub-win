<script setup lang="ts">
// 本文件实现 Win 前端使用的独立 `LogWindow` 窗口。
import { nextTick, onMounted, ref, watch } from "vue"
import CardHeader from "@/components/CardHeader.vue"
import { Button } from "@/components/ui/button"
import LogItem from "@/components/logs/LogItem.vue"
import { useI18n } from "@/i18n"
import { useLogsStore } from "@/stores/logs"
import { useToastStore } from "@/stores/toast"

const logsStore = useLogsStore()
const toast = useToastStore()
const { t } = useI18n()
const logRef = ref<HTMLElement | null>(null)

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
  <section class="space-y-4 p-4">
    <CardHeader class="items-center" :title="t('Live Log Stream')" title-tag="h1" title-class="text-lg">
      <template #actions>
        <Button size="sm" variant="outline" @click="scrollToBottom">{{ t("Scroll to Bottom") }}</Button>
      </template>
    </CardHeader>

    <div class="flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
      <label class="flex items-center gap-2">
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
      class="max-h-[640px] overflow-y-auto rounded-xl border border-border/60 bg-background/70 p-4"
    >
      <div class="space-y-3">
        <LogItem v-for="line in logsStore.state.lines" :key="line.id" :line="line" />
      </div>
      <p v-if="logsStore.state.lines.length === 0" class="text-sm text-muted-foreground">
        {{ t("No logs yet.") }}
      </p>
    </div>

  </section>
</template>
