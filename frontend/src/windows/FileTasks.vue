<script setup lang="ts">
import { computed, onMounted } from "vue"
import CardHeader from "@/components/CardHeader.vue"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"
import { useFileStore } from "@/stores/file"

const fileStore = useFileStore()
const { t } = useI18n()

const tasks = computed(() => fileStore.state.tasks)

const statusLabel = (status: string) => {
  switch (status) {
    case "waiting_response":
      return t("Waiting for response")
    case "waiting_remote":
      return t("Waiting for remote")
    case "waiting_confirm":
      return t("Awaiting confirmation")
    case "preparing":
      return t("Preparing")
    case "hashing":
      return t("Hashing")
    case "sending":
      return t("Sending")
    case "receiving":
      return t("Receiving")
    case "waiting_ack":
      return t("Waiting for ack")
    case "completed":
      return t("Completed")
    case "failed":
      return t("Failed")
    case "canceled":
      return t("Canceled")
    case "rejected":
      return t("Rejected")
    default:
      return status ? t(status) : t("Unknown")
  }
}

const opLabel = (op: string) => t(op || "Unknown")
const directionLabel = (direction: string) => t(direction || "Unknown")

const canRetry = (status: string) => status === "failed"

const canCancel = (status: string) =>
  [
    "sending",
    "receiving",
    "waiting_ack",
    "waiting_response",
    "waiting_remote",
    "waiting_confirm",
    "preparing",
    "hashing"
  ].includes(status)

const progressValue = (task: any) => {
  const size = Number(task?.size ?? 0)
  if (!size) return task?.status === "completed" ? 1 : 0
  if (task?.direction === "upload") {
    return Math.min(1, Number(task?.ackedBytes ?? 0) / size)
  }
  return Math.min(1, Number(task?.doneBytes ?? 0) / size)
}

const progressText = (task: any) => {
  const size = Number(task?.size ?? 0)
  if (!size) return t("0 bytes")
  if (task?.direction === "upload") {
    return t("acked {done} / {size}", { done: task?.ackedBytes ?? 0, size })
  }
  return t("{done} / {size}", { done: task?.doneBytes ?? 0, size })
}

onMounted(async () => {
  await fileStore.loadTasks()
})
</script>

<template>
  <section class="space-y-6">
    <CardHeader class="items-center" :title="t('File Tasks')" title-tag="h1" title-class="text-2xl">
      <template #actions>
        <div class="text-xs text-muted-foreground">
          {{ t("{count} active records", { count: tasks.length }) }}
        </div>
      </template>
    </CardHeader>

    <div v-if="!tasks.length" class="rounded-2xl border bg-card/90 p-6 text-muted-foreground">
      {{ t("No transfers yet. Start a download or offer to see tasks here.") }}
    </div>

    <div v-else class="space-y-4">
      <article
        v-for="task in tasks"
        :key="task.taskId"
        class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm"
      >
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h2 class="text-base font-semibold">
              {{ opLabel(task.op) }} · {{ directionLabel(task.direction) }} · {{ task.name || t("unnamed") }}
            </h2>
            <p class="text-xs text-muted-foreground">
              {{ t("provider {provider} · consumer {consumer} · peer {peer}", { provider: task.provider, consumer: task.consumer, peer: task.peer }) }}
            </p>
          </div>
          <span
            class="rounded-full bg-muted px-3 py-1 text-xs font-semibold text-foreground"
          >
            {{ statusLabel(task.status) }}
          </span>
        </div>

        <div class="mt-4">
          <div class="flex items-center justify-between text-xs text-muted-foreground">
            <span>{{ progressText(task) }}</span>
            <span>{{ t("{count} bytes", { count: task.size }) }}</span>
          </div>
          <div class="mt-2 h-2 w-full rounded-full bg-muted">
            <div
              class="h-2 rounded-full bg-primary"
              :style="{ width: `${Math.round(progressValue(task) * 100)}%` }"
            />
          </div>
        </div>

        <div class="mt-3 text-xs text-muted-foreground">
          <p v-if="task.localPath">{{ t("local: {path}", { path: task.localPath }) }}</p>
          <p v-else-if="task.localDir">{{ t("save dir: {dir}", { dir: task.localDir }) }}</p>
          <p v-if="task.sha256">{{ t("sha256: {value}", { value: task.sha256 }) }}</p>
          <p v-if="task.status === 'failed' && task.lastError" class="text-rose-600">
            {{ task.lastError }}
          </p>
        </div>

        <div class="mt-4 flex flex-wrap gap-2">
          <Button
            size="sm"
            variant="outline"
            :disabled="!canRetry(task.status)"
            @click="fileStore.retryTask(task.taskId)"
          >
            {{ t("Retry") }}
          </Button>
          <Button
            size="sm"
            variant="outline"
            :disabled="!canCancel(task.status)"
            @click="fileStore.cancelTask(task.taskId)"
          >
            {{ t("Cancel") }}
          </Button>
          <Button size="sm" variant="outline" @click="fileStore.openTaskFolder(task.taskId)">
            {{ t("Open Folder") }}
          </Button>
        </div>
      </article>
    </div>
  </section>
</template>
