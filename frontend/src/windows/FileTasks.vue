<script setup lang="ts">
import { computed, onMounted } from "vue"
import { Button } from "@/components/ui/button"
import { useFileStore } from "@/stores/file"

const fileStore = useFileStore()

const tasks = computed(() => fileStore.state.tasks)

const statusLabel = (status: string) => {
  switch (status) {
    case "waiting_response":
      return "Waiting for response"
    case "waiting_remote":
      return "Waiting for remote"
    case "waiting_confirm":
      return "Awaiting confirmation"
    case "preparing":
      return "Preparing"
    case "hashing":
      return "Hashing"
    case "sending":
      return "Sending"
    case "receiving":
      return "Receiving"
    case "waiting_ack":
      return "Waiting for ack"
    case "completed":
      return "Completed"
    case "failed":
      return "Failed"
    case "canceled":
      return "Canceled"
    case "rejected":
      return "Rejected"
    default:
      return status || "Unknown"
  }
}

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
  if (!size) return "0 bytes"
  if (task?.direction === "upload") {
    return `acked ${task?.ackedBytes ?? 0} / ${size}`
  }
  return `${task?.doneBytes ?? 0} / ${size}`
}

onMounted(async () => {
  await fileStore.loadTasks()
})
</script>

<template>
  <section class="space-y-6">
    <div class="flex items-center justify-between gap-4">
      <div>
        <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
          Transfers
        </p>
        <h1 class="text-2xl font-semibold">File Tasks</h1>
      </div>
      <div class="text-xs text-muted-foreground">
        {{ tasks.length }} active records
      </div>
    </div>

    <div v-if="!tasks.length" class="rounded-2xl border bg-card/90 p-6 text-muted-foreground">
      No transfers yet. Start a download or offer to see tasks here.
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
              {{ task.op }} · {{ task.direction }} · {{ task.name || "unnamed" }}
            </h2>
            <p class="text-xs text-muted-foreground">
              provider {{ task.provider }} · consumer {{ task.consumer }} · peer {{ task.peer }}
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
            <span>{{ task.size }} bytes</span>
          </div>
          <div class="mt-2 h-2 w-full rounded-full bg-muted">
            <div
              class="h-2 rounded-full bg-primary"
              :style="{ width: `${Math.round(progressValue(task) * 100)}%` }"
            />
          </div>
        </div>

        <div class="mt-3 text-xs text-muted-foreground">
          <p v-if="task.localPath">local: {{ task.localPath }}</p>
          <p v-else-if="task.localDir">save dir: {{ task.localDir }}</p>
          <p v-if="task.sha256">sha256: {{ task.sha256 }}</p>
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
            Retry
          </Button>
          <Button
            size="sm"
            variant="outline"
            :disabled="!canCancel(task.status)"
            @click="fileStore.cancelTask(task.taskId)"
          >
            Cancel
          </Button>
          <Button size="sm" variant="outline" @click="fileStore.openTaskFolder(task.taskId)">
            Open Folder
          </Button>
        </div>
      </article>
    </div>
  </section>
</template>
