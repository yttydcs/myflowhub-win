<script setup lang="ts">
import { computed } from "vue"
import CardHeader from "@/components/CardHeader.vue"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import { useI18n } from "@/i18n"
import { describeFieldBinding, type FlowBindingSourceKind, type VisualFieldModel } from "@/stores/flow"

const props = defineProps<{
  open: boolean
  activeBindingField: VisualFieldModel | null
  bindableAncestorNodeOptions: string[]
  fieldBindingDraft: {
    sourceKind: FlowBindingSourceKind
    nodeId: string
    path: string
    field: string
    required: boolean
  }
}>()

const emit = defineEmits<{
  (event: "close"): void
  (event: "apply"): void
  (event: "clear"): void
  (event: "source-kind-change", value: string): void
}>()

const { t } = useI18n()
const dialogTitleId = "flow-field-binding-dialog-title"
const dialogDescriptionId = "flow-field-binding-dialog-description"
const sourceKindInputId = "flow-field-binding-source-kind"
const ancestorNodeInputId = "flow-field-binding-ancestor-node"
const nodeResultPathInputId = "flow-field-binding-result-path"
const triggerPathInputId = "flow-field-binding-trigger-path"
const requiredBindingInputId = "flow-field-binding-required"

const draftSourcePreview = computed(() => {
  switch (props.fieldBindingDraft.sourceKind) {
    case "node_result":
      return describeFieldBinding({
        kind: "node_result",
        nodeId: props.fieldBindingDraft.nodeId.trim(),
        path: props.fieldBindingDraft.path.trim(),
        required: props.fieldBindingDraft.required
      })
    case "flow_meta":
      return describeFieldBinding({
        kind: "flow_meta",
        field: "flow_id",
        required: props.fieldBindingDraft.required
      })
    case "run_meta":
      return describeFieldBinding({
        kind: "run_meta",
        field: "run_id",
        required: props.fieldBindingDraft.required
      })
    case "trigger":
    default:
      return describeFieldBinding({
        kind: "trigger",
        path: props.fieldBindingDraft.path.trim(),
        required: props.fieldBindingDraft.required
      })
  }
})

const sourceHelpText = computed(() => {
  switch (props.fieldBindingDraft.sourceKind) {
    case "node_result":
      return t("Read from a previous node result. Only ancestor nodes can be used here.")
    case "flow_meta":
      return t("Read stable flow metadata without a JSON path.")
    case "run_meta":
      return t("Read stable run metadata without a JSON path.")
    case "trigger":
    default:
      return t("Read from the trigger payload for this flow run.")
  }
})

const canApply = computed(() => {
  if (!props.activeBindingField) {
    return false
  }
  if (props.fieldBindingDraft.sourceKind !== "node_result") {
    return true
  }
  return Boolean(props.fieldBindingDraft.nodeId.trim())
})
</script>

<template>
  <Overlay
    :open="open"
    :trap-focus="true"
    :initial-focus-selector="`#${sourceKindInputId}`"
    overlayClass="bg-slate-950/60 p-4"
    zIndexClass="z-40"
    closeOnBackdrop
    @close="emit('close')"
  >
    <div
      class="w-full max-w-xl rounded-2xl border bg-card p-6 text-card-foreground shadow-2xl"
      role="dialog"
      aria-modal="true"
      :aria-labelledby="dialogTitleId"
      :aria-describedby="dialogDescriptionId"
    >
      <CardHeader
        class="items-start"
        :title="
          activeBindingField
            ? t('Bind Field: {label}', { label: activeBindingField.schema.label })
            : t('Bind Field')
        "
        :description="t('Choose an upstream source for this field. The editor will write the binding back into the node spec.')"
        title-class="text-lg"
        :title-id="dialogTitleId"
        :description-id="dialogDescriptionId"
      />

      <div v-if="activeBindingField" class="mt-4 space-y-4">
        <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
          <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            {{ t("Destination") }}
          </p>
          <p class="mt-2 text-sm font-semibold">{{ activeBindingField.schema.label }}</p>
          <p class="mt-1 break-all text-[11px] text-muted-foreground">
            {{ t("Writes to {pointer}", { pointer: activeBindingField.schema.pointer }) }}
          </p>
          <p v-if="activeBindingField.schema.description" class="mt-1 text-[11px] text-muted-foreground">
            {{ activeBindingField.schema.description }}
          </p>
        </div>

        <div>
          <label :for="sourceKindInputId" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            {{ t("Source Kind") }}
          </label>
          <select
            :id="sourceKindInputId"
            :value="fieldBindingDraft.sourceKind"
            class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            @change="emit('source-kind-change', String(($event.target as HTMLSelectElement | null)?.value ?? 'trigger'))"
          >
            <option v-if="bindableAncestorNodeOptions.length" value="node_result">{{ t("Ancestor Result") }}</option>
            <option value="trigger">{{ t("Trigger") }}</option>
            <option value="flow_meta">{{ t("Flow Meta") }}</option>
            <option value="run_meta">{{ t("Run Meta") }}</option>
          </select>
        </div>

        <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
          <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            {{ t("Source Preview") }}
          </p>
          <p class="mt-2 text-sm font-semibold">
            {{ draftSourcePreview || t("No source selected.") }}
          </p>
          <p class="mt-1 text-[11px] text-muted-foreground">
            {{ sourceHelpText }}
          </p>
        </div>

        <div v-if="fieldBindingDraft.sourceKind === 'node_result'" class="grid gap-3 md:grid-cols-2">
          <div>
            <label :for="ancestorNodeInputId" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Ancestor Node") }}
            </label>
            <select
              :id="ancestorNodeInputId"
              v-model="fieldBindingDraft.nodeId"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            >
              <option value="">
                {{ bindableAncestorNodeOptions.length ? t("Select ancestor node") : t("No ancestor available") }}
              </option>
              <option v-for="ancestorId in bindableAncestorNodeOptions" :key="ancestorId" :value="ancestorId">
                {{ t("Node {nodeId}", { nodeId: ancestorId }) }}
              </option>
            </select>
          </div>

          <div>
            <label :for="nodeResultPathInputId" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Result Path (Optional)") }}
            </label>
            <input
              :id="nodeResultPathInputId"
              v-model="fieldBindingDraft.path"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="/user/id"
            />
            <p class="mt-1 text-[11px] text-muted-foreground">
              {{ t("Path can stay empty to read the whole node result.") }}
            </p>
          </div>
        </div>

        <div v-else-if="fieldBindingDraft.sourceKind === 'trigger'">
          <label :for="triggerPathInputId" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            {{ t("Trigger Path (Optional)") }}
          </label>
          <input
            :id="triggerPathInputId"
            v-model="fieldBindingDraft.path"
            class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            placeholder="/payload/name"
          />
          <p class="mt-1 text-[11px] text-muted-foreground">
            {{ t("Path can stay empty to read the whole trigger payload.") }}
          </p>
        </div>

        <div v-else class="rounded-xl border border-border/70 bg-muted/20 p-4 text-sm text-muted-foreground">
          {{
            fieldBindingDraft.sourceKind === "flow_meta"
              ? t("This field will read from flow meta: flow_id.")
              : t("This field will read from run meta: run_id.")
          }}
        </div>

        <label class="flex items-center gap-2 text-sm text-muted-foreground">
          <input
            :id="requiredBindingInputId"
            v-model="fieldBindingDraft.required"
            type="checkbox"
            class="h-4 w-4 rounded border"
          />
          <span :id="`${requiredBindingInputId}-label`">{{ t("Required binding") }}</span>
        </label>
      </div>

      <div class="mt-6 flex flex-wrap items-center justify-between gap-3">
        <Button variant="ghost" :disabled="!activeBindingField?.state.binding" @click="emit('clear')">
          {{ t("Clear Binding") }}
        </Button>
        <div class="flex gap-2">
          <Button variant="outline" @click="emit('close')">{{ t("Cancel") }}</Button>
          <Button :disabled="!canApply" @click="emit('apply')">{{ t("Apply Binding") }}</Button>
        </div>
      </div>
    </div>
  </Overlay>
</template>
