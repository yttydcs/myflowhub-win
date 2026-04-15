<script setup lang="ts">
// 本文件实现 Flow 编辑器中的 `FlowEditorToolbar` 组件。
import { computed } from "vue"
import { Ban, History, LayoutGrid, Link2Off, Play, Plus, Redo2, RefreshCw, Save, Trash2, Undo2 } from "lucide-vue-next"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tooltip } from "@/components/ui/tooltip"
import { useI18n } from "@/i18n"

interface Props {
  title: string
  dirty: boolean
  lastSavedLabel: string
  canUndo: boolean
  canRedo: boolean
  loading: boolean
  saveBusy: boolean
  hasSelectedNode: boolean
  hasSelectedEdge: boolean
  runBusy: boolean
  statusBusy: boolean
  runHistoryBusy: boolean
  cancelBusy: boolean
  canRunFlow: boolean
  canRefreshStatus: boolean
  canListRuns: boolean
  canCancelRun: boolean
  flowStatusLabel: string
  currentRunIdLabel: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: "add-node"): void
  (e: "remove-node"): void
  (e: "remove-edge"): void
  (e: "undo"): void
  (e: "redo"): void
  (e: "auto-layout"): void
  (e: "save-project"): void
  (e: "run-flow"): void
  (e: "refresh-status"): void
  (e: "load-run-history"): void
  (e: "cancel-run"): void
}>()

const { t } = useI18n()

const statusVariant = computed(() => (props.dirty ? "outline" : "muted"))
const hasRuntimeSummary = computed(() => Boolean(props.flowStatusLabel) || Boolean(props.currentRunIdLabel))
</script>

<template>
  <header class="flex-none border-b border-border/60 bg-card/92 px-5 py-4 shadow-sm">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-xl font-semibold">{{ props.title }}</h1>
        <div class="mt-2 flex flex-wrap items-center gap-2">
          <Badge :variant="statusVariant">
            {{ props.dirty ? t("Unsaved changes") : t("Saved") }}
          </Badge>
          <Badge v-if="props.flowStatusLabel" variant="outline">
            {{ t("Flow Status") }} · {{ props.flowStatusLabel }}
          </Badge>
          <p v-if="hasRuntimeSummary && props.currentRunIdLabel" class="text-xs text-muted-foreground">
            {{ props.currentRunIdLabel }}
          </p>
          <p v-if="props.lastSavedLabel" class="text-xs text-muted-foreground">
            {{ props.lastSavedLabel }}
          </p>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Tooltip :content="t('Run Flow')" side="bottom">
          <Button
            size="icon"
            variant="outline"
            :disabled="props.runBusy || props.loading || !props.canRunFlow"
            @click="emit('run-flow')"
          >
            <Play class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">{{ t("Run Flow") }}</span>
          </Button>
        </Tooltip>
        <Tooltip :content="t('Refresh Status')" side="bottom">
          <Button
            size="icon"
            variant="outline"
            :disabled="props.statusBusy || props.loading || !props.canRefreshStatus"
            @click="emit('refresh-status')"
          >
            <RefreshCw class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">{{ t("Refresh Status") }}</span>
          </Button>
        </Tooltip>
        <Tooltip :content="t('Run History')" side="bottom">
          <Button
            size="icon"
            variant="outline"
            :disabled="props.runHistoryBusy || props.loading || !props.canListRuns"
            @click="emit('load-run-history')"
          >
            <History class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">{{ t("Run History") }}</span>
          </Button>
        </Tooltip>
        <Tooltip :content="t('Cancel Run')" side="bottom">
          <Button
            size="icon"
            variant="outline"
            :disabled="props.cancelBusy || props.loading || !props.canCancelRun"
            @click="emit('cancel-run')"
          >
            <Ban class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">{{ t("Cancel Run") }}</span>
          </Button>
        </Tooltip>
        <Tooltip :content="t('Add Node')" side="bottom">
          <Button size="icon" variant="outline" @click="emit('add-node')">
            <Plus class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">{{ t("Add Node") }}</span>
          </Button>
        </Tooltip>
        <Tooltip :content="t('Remove Node (Delete)')" side="bottom">
          <Button size="icon" variant="outline" :disabled="!props.hasSelectedNode" @click="emit('remove-node')">
            <Trash2 class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">{{ t("Remove Node (Delete)") }}</span>
          </Button>
        </Tooltip>
        <Tooltip :content="t('Remove Edge (Delete)')" side="bottom">
          <Button size="icon" variant="outline" :disabled="!props.hasSelectedEdge" @click="emit('remove-edge')">
            <Link2Off class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">{{ t("Remove Edge (Delete)") }}</span>
          </Button>
        </Tooltip>
        <Tooltip :content="t('Undo (Ctrl+Z)')" side="bottom">
          <Button size="icon" variant="outline" :disabled="!props.canUndo" @click="emit('undo')">
            <Undo2 class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">{{ t("Undo (Ctrl+Z)") }}</span>
          </Button>
        </Tooltip>
        <Tooltip :content="t('Redo (Ctrl+Y)')" side="bottom">
          <Button size="icon" variant="outline" :disabled="!props.canRedo" @click="emit('redo')">
            <Redo2 class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">{{ t("Redo (Ctrl+Y)") }}</span>
          </Button>
        </Tooltip>
        <Tooltip :content="t('Auto Layout')" side="bottom">
          <Button size="icon" variant="outline" @click="emit('auto-layout')">
            <LayoutGrid class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">{{ t("Auto Layout") }}</span>
          </Button>
        </Tooltip>
        <Tooltip :content="t('Save Project (Ctrl+S)')" side="bottom">
          <Button size="icon" :disabled="props.saveBusy || props.loading" @click="emit('save-project')">
            <Save class="h-4 w-4" aria-hidden="true" />
            <span class="sr-only">{{ t("Save Project (Ctrl+S)") }}</span>
          </Button>
        </Tooltip>
      </div>
    </div>
  </header>
</template>
