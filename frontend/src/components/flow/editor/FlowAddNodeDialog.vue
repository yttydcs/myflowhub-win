<script setup lang="ts">
// Context: renders the flow add node dialog panel or dialog used by the Flow editor.
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
    : props.nodeKind === "compose"
      ? t("Compose nodes build local JSON output from template + bindings.")
      : props.nodeKind === "transform"
        ? t("Transform nodes evaluate a structured expression tree and produce a local result.")
        : props.nodeKind === "set_var"
          ? t("Set var nodes materialize a value and write it to a flow-local variable for this run.")
          : props.nodeKind === "branch"
            ? t("Branch nodes match ordered cases and route execution through edge.case labels.")
            : props.nodeKind === "foreach"
              ? t("Foreach nodes iterate an array source and execute a nested body graph.")
              : t("Subflow nodes synchronously execute another flow on the same executor.")
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
      data-flow-add-node-dialog
      class="flex max-h-[85vh] w-full max-w-md flex-col overflow-hidden rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl"
      role="dialog"
      aria-modal="true"
      :aria-labelledby="dialogTitleId"
      :aria-describedby="dialogDescriptionId"
    >
      <h2 :id="dialogTitleId" class="text-lg font-semibold">{{ t("Add Node") }}</h2>
      <div data-flow-add-node-scroll class="mt-5 min-h-0 flex-1 overflow-y-auto">
        <div class="space-y-3 px-1 py-1 pr-2">
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
            <div class="mt-2 grid grid-cols-2 gap-2 sm:grid-cols-4" role="group" :aria-labelledby="kindGroupLabelId">
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
              <button
                type="button"
                class="rounded-md border px-3 py-2 text-sm transition"
                :aria-pressed="props.nodeKind === 'transform'"
                :class="
                  props.nodeKind === 'transform'
                    ? 'border-primary bg-primary/10 text-primary'
                    : 'border-border/70 bg-background text-foreground'
                "
                @click="emit('update:nodeKind', 'transform')"
              >
                {{ t("Transform") }}
              </button>
              <button
                type="button"
                class="rounded-md border px-3 py-2 text-sm transition"
                :aria-pressed="props.nodeKind === 'set_var'"
                :class="
                  props.nodeKind === 'set_var'
                    ? 'border-primary bg-primary/10 text-primary'
                    : 'border-border/70 bg-background text-foreground'
                "
                @click="emit('update:nodeKind', 'set_var')"
              >
                {{ t("Set Var") }}
              </button>
              <button
                type="button"
                class="rounded-md border px-3 py-2 text-sm transition"
                :aria-pressed="props.nodeKind === 'branch'"
                :class="
                  props.nodeKind === 'branch'
                    ? 'border-primary bg-primary/10 text-primary'
                    : 'border-border/70 bg-background text-foreground'
                "
                @click="emit('update:nodeKind', 'branch')"
              >
                {{ t("Branch") }}
              </button>
              <button
                type="button"
                class="rounded-md border px-3 py-2 text-sm transition"
                :aria-pressed="props.nodeKind === 'foreach'"
                :class="
                  props.nodeKind === 'foreach'
                    ? 'border-primary bg-primary/10 text-primary'
                    : 'border-border/70 bg-background text-foreground'
                "
                @click="emit('update:nodeKind', 'foreach')"
              >
                {{ t("Foreach") }}
              </button>
              <button
                type="button"
                class="rounded-md border px-3 py-2 text-sm transition"
                :aria-pressed="props.nodeKind === 'subflow'"
                :class="
                  props.nodeKind === 'subflow'
                    ? 'border-primary bg-primary/10 text-primary'
                    : 'border-border/70 bg-background text-foreground'
                "
                @click="emit('update:nodeKind', 'subflow')"
              >
                {{ t("Subflow") }}
              </button>
            </div>
            <p :id="dialogDescriptionId" class="mt-1 text-[11px] text-muted-foreground">
              {{ kindDescription }}
            </p>
          </div>
        </div>
      </div>
      <div class="mt-6 flex justify-end gap-2">
        <Button variant="outline" @click="emit('close')">{{ t("Cancel") }}</Button>
        <Button @click="emit('add')">{{ t("Add") }}</Button>
      </div>
    </div>
  </Overlay>
</template>
