<script setup lang="ts">
import { computed } from "vue"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import { useI18n } from "@/i18n"
import type { FlowNodeKind } from "@/stores/flow"

interface Props {
  open: boolean
  nodeId: string
  nodeKind: FlowNodeKind
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: "close"): void
  (e: "update:nodeId", value: string): void
  (e: "update:nodeKind", value: FlowNodeKind): void
  (e: "add"): void
}>()

const { t } = useI18n()

const dialogTitleId = "flow-add-node-dialog-title"
const dialogDescriptionId = "flow-add-node-dialog-description"
const nodeIdInputId = "flow-add-node-id"
const kindGroupLabelId = "flow-add-node-kind-label"

const kindDescription = computed(() =>
  props.nodeKind === "call"
    ? t("Call nodes execute a capability and can bind ancestor outputs into args.")
    : t("Compose nodes build local JSON output from template + bindings.")
)
</script>

<template>
  <Overlay
    :open="props.open"
    :trap-focus="true"
    :initial-focus-selector="`#${nodeIdInputId}`"
    @close="emit('close')"
  >
    <div
      class="w-full max-w-md rounded-2xl border bg-card/95 p-6 shadow-xl"
      role="dialog"
      aria-modal="true"
      :aria-labelledby="dialogTitleId"
      :aria-describedby="dialogDescriptionId"
    >
      <h2 :id="dialogTitleId" class="text-lg font-semibold">{{ t("Add Node") }}</h2>
      <div class="mt-4 space-y-3">
        <div>
          <label :for="nodeIdInputId" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            {{ t("Node ID") }}
          </label>
          <input
            :id="nodeIdInputId"
            :value="props.nodeId"
            class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            @input="emit('update:nodeId', ($event.target as HTMLInputElement).value)"
          />
        </div>

        <div>
          <p :id="kindGroupLabelId" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            {{ t("Kind") }}
          </p>
          <div class="mt-2 grid grid-cols-2 gap-2" role="group" :aria-labelledby="kindGroupLabelId">
            <button
              type="button"
              class="rounded-md border px-3 py-2 text-sm transition"
              :aria-pressed="props.nodeKind === 'call'"
              :class="
                props.nodeKind === 'call'
                  ? 'border-primary bg-primary/10 text-primary'
                  : 'border-border/70 bg-background text-foreground'
              "
              @click="emit('update:nodeKind', 'call')"
            >
              {{ t("Call") }}
            </button>
            <button
              type="button"
              class="rounded-md border px-3 py-2 text-sm transition"
              :aria-pressed="props.nodeKind === 'compose'"
              :class="
                props.nodeKind === 'compose'
                  ? 'border-primary bg-primary/10 text-primary'
                  : 'border-border/70 bg-background text-foreground'
              "
              @click="emit('update:nodeKind', 'compose')"
            >
              {{ t("Compose") }}
            </button>
          </div>
          <p :id="dialogDescriptionId" class="mt-1 text-[11px] text-muted-foreground">
            {{ kindDescription }}
          </p>
        </div>
      </div>
      <div class="mt-6 flex justify-end gap-2">
        <Button variant="outline" @click="emit('close')">{{ t("Cancel") }}</Button>
        <Button @click="emit('add')">{{ t("Add") }}</Button>
      </div>
    </div>
  </Overlay>
</template>
