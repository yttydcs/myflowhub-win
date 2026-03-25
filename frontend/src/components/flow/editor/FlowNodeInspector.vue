<script setup lang="ts">
import { computed } from "vue"
import { X } from "lucide-vue-next"
import CardHeader from "@/components/CardHeader.vue"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"
import { describeVisualCompatibilityReason } from "@/stores/flow"
import type {
  FlowInputBindingDraft,
  FlowNodeDraft,
  FlowNodeKind,
  NodeVisualFormModel,
  VisualCompatibilityReason,
  VisualFieldModel
} from "@/stores/flow"

const props = defineProps<{
  selectedNode: FlowNodeDraft | null
  nodeIdDraft: string
  selectedNodeValidation: string[]
  selectedTargetLabel: string
  selectedCallVisualForm: NodeVisualFormModel | null
  ancestorNodeOptions: string[]
  fieldDrafts: Record<string, any>
}>()

const emit = defineEmits<{
  (event: "close"): void
  (event: "update:nodeIdDraft", value: string): void
  (event: "commit-node-id"): void
  (event: "node-kind-change", value: FlowNodeKind): void
  (event: "toggle-spec-mode", value: "form" | "json"): void
  (event: "open-method"): void
  (event: "open-field-binding", field: VisualFieldModel): void
  (event: "clear-field-binding", pointer: string): void
  (event: "commit-field-literal", field: VisualFieldModel): void
  (event: "set-boolean-field-literal", payload: { field: VisualFieldModel; checked: boolean }): void
  (event: "add-binding"): void
  (event: "remove-binding", index: number): void
  (event: "binding-source-kind-change", payload: { binding: FlowInputBindingDraft; sourceKind: string }): void
  (event: "commit-history"): void
}>()

const { t } = useI18n()

const updateNodeIdDraft = (event: Event) => {
  emit("update:nodeIdDraft", String((event.target as HTMLInputElement | null)?.value ?? ""))
}

const emitNodeKindChange = (event: Event) => {
  const value = String((event.target as HTMLSelectElement | null)?.value ?? "call")
  emit("node-kind-change", value === "compose" ? "compose" : "call")
}

const emitBindingSourceKindChange = (binding: FlowInputBindingDraft, event: Event) => {
  emit("binding-source-kind-change", {
    binding,
    sourceKind: String((event.target as HTMLSelectElement | null)?.value ?? "node_result")
  })
}

const emitBooleanFieldChange = (field: VisualFieldModel, event: Event) => {
  emit("set-boolean-field-literal", {
    field,
    checked: Boolean((event.target as HTMLInputElement | null)?.checked)
  })
}

const toDomIdPart = (value: string | number | null | undefined) =>
  String(value ?? "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "") || "field"

const inspectorFieldId = (suffix: string) =>
  `flow-node-inspector-${toDomIdPart(props.selectedNode?.id || "node")}-${toDomIdPart(suffix)}`

const visualFieldInputId = (field: VisualFieldModel) => inspectorFieldId(`visual-${field.schema.pointer}`)
const visualFieldLabelId = (field: VisualFieldModel) => `${visualFieldInputId(field)}-label`
const visualFieldHelpId = (field: VisualFieldModel) => `${visualFieldInputId(field)}-help`
const composeBindingInputId = (index: number, suffix: string) => inspectorFieldId(`binding-${index}-${suffix}`)

const visualCompatibilityReasonCategory = (reason: VisualCompatibilityReason) => {
  switch (reason.code) {
    case "missing_method":
      return t("Method")
    case "missing_schema":
      return t("Schema")
    case "binding_target_unknown":
    case "duplicate_field_binding":
      return t("Bindings")
    case "args_template_invalid_json":
    case "args_template_not_object":
    case "extra_literal_field":
      return t("Template")
    default:
      return t("Schema")
  }
}

const visualCompatibilityReasonHelp = (reason: VisualCompatibilityReason) => {
  switch (reason.code) {
    case "missing_method":
      return t("Fix the missing call method first, then ordinary mode can resolve the matching schema.")
    case "missing_schema":
      return t("Query capabilities and choose a method that exposes a supported input schema.")
    case "args_template_invalid_json":
    case "args_template_not_object":
      return t("Correct the JSON first. Ordinary mode only works when args_template parses as an object.")
    case "binding_target_unknown":
      return t("Remove or remap bindings that point outside the supported field list.")
    case "duplicate_field_binding":
      return t("Ordinary mode allows only one binding per destination field.")
    case "extra_literal_field":
      return t("Remove extra literal fields or continue in Advanced JSON for this node.")
    default:
      return t("Review the current node spec in Advanced JSON.")
  }
}

const visibleVisualCompatibilityReasons = computed(() =>
  (props.selectedCallVisualForm?.compatibility.reasons ?? []).filter((reason) => reason.code !== "missing_schema")
)
</script>

<template>
  <aside
    v-if="selectedNode"
    class="pointer-events-auto h-full w-full max-w-[420px] border-l border-border/70 bg-card shadow-2xl"
    role="region"
    :aria-labelledby="inspectorFieldId('title')"
  >
    <div class="flex h-full flex-col">
      <div class="flex items-start justify-between gap-4 border-b border-border/60 px-5 py-4">
        <CardHeader
          class="w-full items-start"
          :title="selectedNode.id"
          title-class="text-lg"
          :title-id="inspectorFieldId('title')"
        >
          <template #actions>
            <Button size="icon" variant="ghost" @click="emit('close')">
              <X class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">{{ t("Close") }}</span>
            </Button>
          </template>
        </CardHeader>
      </div>

      <div class="flex-1 overflow-y-auto px-5 py-4">
        <div class="space-y-4">
          <div>
            <label :for="inspectorFieldId('node-id')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Node ID") }}
            </label>
            <input
              :id="inspectorFieldId('node-id')"
              :value="nodeIdDraft"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              @input="updateNodeIdDraft"
              @blur="emit('commit-node-id')"
              @keydown.enter.prevent="emit('commit-node-id')"
            />
            <p class="mt-1 text-[11px] text-muted-foreground">
              {{ t("Node ID must be unique. Renaming updates all connected edges.") }}
            </p>
          </div>

          <div
            v-if="selectedNodeValidation.length"
            class="rounded-xl border border-rose-200 bg-rose-50/80 p-3 text-xs text-rose-700"
            role="status"
            aria-live="polite"
          >
            <p class="font-semibold uppercase tracking-[0.18em]">{{ t("Validation") }}</p>
            <ul class="mt-2 space-y-1">
              <li v-for="message in selectedNodeValidation" :key="message">• {{ message }}</li>
            </ul>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label :for="inspectorFieldId('kind')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Kind") }}
              </label>
              <select
                :id="inspectorFieldId('kind')"
                :value="selectedNode.kind"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                @change="emitNodeKindChange"
              >
                <option value="call">{{ t("Call") }}</option>
                <option value="compose">{{ t("Compose") }}</option>
              </select>
            </div>

            <div>
              <p :id="inspectorFieldId('allow-fail-label')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Allow Fail") }}
              </p>
              <div class="mt-2 flex h-10 items-center gap-2 rounded-md border border-input bg-background px-3 text-sm">
                <input
                  :id="inspectorFieldId('allow-fail')"
                  v-model="selectedNode.allowFail"
                  type="checkbox"
                  class="h-4 w-4 rounded border"
                  :aria-labelledby="inspectorFieldId('allow-fail-label')"
                  :aria-describedby="inspectorFieldId('allow-fail-help')"
                  @change="emit('commit-history')"
                />
                <span :id="inspectorFieldId('allow-fail-help')" class="text-muted-foreground">
                  {{ t("Continue on error") }}
                </span>
              </div>
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label :for="inspectorFieldId('retry')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Retry") }}
              </label>
              <input
                :id="inspectorFieldId('retry')"
                v-model.number="selectedNode.retry"
                type="number"
                min="0"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                @blur="emit('commit-history')"
              />
            </div>

            <div>
              <label :for="inspectorFieldId('timeout-ms')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Timeout (ms)") }}
              </label>
              <input
                :id="inspectorFieldId('timeout-ms')"
                v-model.number="selectedNode.timeoutMs"
                type="number"
                min="0"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                @blur="emit('commit-history')"
              />
            </div>
          </div>

          <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div :id="inspectorFieldId('spec-mode-label')">
                <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Spec Mode") }}
                </p>
                <p class="mt-1 text-[11px] text-muted-foreground">
                  {{ t("Form mode edits the supported fields directly. Advanced JSON is the escape hatch for full spec editing.") }}
                </p>
              </div>
              <div class="flex gap-2" role="group" :aria-labelledby="inspectorFieldId('spec-mode-label')">
                <Button
                  size="sm"
                  :variant="selectedNode.specEditorMode === 'form' ? 'default' : 'outline'"
                  @click="emit('toggle-spec-mode', 'form')"
                >
                  {{ t("Form") }}
                </Button>
                <Button
                  size="sm"
                  :variant="selectedNode.specEditorMode === 'json' ? 'default' : 'outline'"
                  @click="emit('toggle-spec-mode', 'json')"
                >
                  {{ t("Advanced JSON") }}
                </Button>
              </div>
            </div>
          </div>

          <template v-if="selectedNode.specEditorMode === 'form'">
            <div v-if="selectedNode.kind === 'call'" class="space-y-4">
              <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div class="min-w-0">
                    <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                      {{ t("Call Method") }}
                    </p>
                    <p class="mt-2 break-all text-sm font-semibold">
                      {{ selectedNode.method || t("No method selected.") }}
                    </p>
                    <p class="mt-1 text-xs text-muted-foreground">
                      {{ selectedTargetLabel }}
                    </p>
                  </div>
                  <Button variant="outline" size="sm" @click="emit('open-method')">{{ t("Select Method") }}</Button>
                </div>
                <p class="mt-3 text-[11px] text-muted-foreground">
                  {{ t("Use the method dialog to choose a registered capability. The editor will keep method and target aligned.") }}
                </p>
              </div>

              <div
                v-if="selectedCallVisualForm?.compatibility.supported"
                class="rounded-xl border border-border/70 bg-muted/20 p-4"
              >
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div class="min-w-0">
                    <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                      {{ t("Method Fields") }}
                    </p>
                    <p class="mt-2 text-sm font-semibold">
                      {{ selectedCallVisualForm.schema?.title || selectedNode.method }}
                    </p>
                    <p class="mt-1 text-[11px] text-muted-foreground">
                      {{
                        selectedCallVisualForm.schema?.source === "local_override"
                          ? t("Schema source: local override")
                          : t("Schema source: capability input_schema")
                      }}
                    </p>
                  </div>
                  <span class="rounded-full border border-border/70 px-3 py-1 text-[10px] uppercase tracking-[0.2em] text-muted-foreground">
                    {{ t("Visual Form") }}
                  </span>
                </div>

                <div class="mt-4 space-y-3">
                  <div
                    v-for="field in selectedCallVisualForm.fields"
                    :key="field.schema.pointer"
                    class="rounded-lg border border-border/70 bg-background/90 p-4"
                  >
                    <div class="flex flex-wrap items-start justify-between gap-3">
                      <div class="min-w-0">
                        <div class="flex flex-wrap items-center gap-2">
                          <p :id="visualFieldLabelId(field)" class="text-sm font-semibold">{{ field.schema.label }}</p>
                          <span
                            v-if="field.schema.required"
                            class="rounded-full border border-border/70 px-2 py-0.5 text-[10px] uppercase tracking-[0.16em] text-muted-foreground"
                          >
                            {{ t("Required") }}
                          </span>
                          <span
                            v-if="field.state.mode === 'binding'"
                            class="rounded-full border border-primary/40 px-2 py-0.5 text-[10px] uppercase tracking-[0.16em] text-primary"
                          >
                            {{ t("Bound") }}
                          </span>
                        </div>
                        <p :id="visualFieldHelpId(field)" class="mt-1 break-all text-[11px] text-muted-foreground">
                          {{ t("Writes to {pointer}", { pointer: field.schema.pointer }) }}
                        </p>
                        <p v-if="field.schema.description" class="mt-1 text-[11px] text-muted-foreground">
                          {{ field.schema.description }}
                        </p>
                      </div>
                      <Button
                        v-if="field.schema.bindable !== false"
                        size="sm"
                        variant="outline"
                        @click="emit('open-field-binding', field)"
                      >
                        {{ field.state.mode === "binding" ? t("Edit fx") : t("Add fx") }}
                      </Button>
                    </div>

                    <div
                      v-if="field.state.mode === 'binding'"
                      class="mt-3 rounded-lg border border-dashed border-primary/30 bg-primary/5 px-3 py-3"
                    >
                      <p class="text-xs font-medium text-primary">
                        {{ t("This field currently reads from an upstream source.") }}
                      </p>
                      <p class="mt-1 text-sm font-medium">{{ field.bindingSummary || t("Binding configured.") }}</p>
                      <div class="mt-3 flex flex-wrap gap-2">
                        <Button size="sm" variant="outline" @click="emit('open-field-binding', field)">
                          {{ t("Edit Binding") }}
                        </Button>
                        <Button size="sm" variant="ghost" @click="emit('clear-field-binding', field.schema.pointer)">
                          {{ t("Use Literal Instead") }}
                        </Button>
                      </div>
                    </div>

                    <div v-else class="mt-3">
                      <input
                        v-if="field.schema.control === 'text'"
                        :id="visualFieldInputId(field)"
                        v-model="fieldDrafts[field.schema.pointer]"
                        class="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                        :aria-labelledby="visualFieldLabelId(field)"
                        :aria-describedby="visualFieldHelpId(field)"
                        @blur="emit('commit-field-literal', field)"
                      />

                      <textarea
                        v-else-if="field.schema.control === 'textarea'"
                        :id="visualFieldInputId(field)"
                        v-model="fieldDrafts[field.schema.pointer]"
                        rows="4"
                        class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                        :aria-labelledby="visualFieldLabelId(field)"
                        :aria-describedby="visualFieldHelpId(field)"
                        @blur="emit('commit-field-literal', field)"
                      />

                      <input
                        v-else-if="field.schema.control === 'number'"
                        :id="visualFieldInputId(field)"
                        v-model="fieldDrafts[field.schema.pointer]"
                        type="number"
                        class="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                        :aria-labelledby="visualFieldLabelId(field)"
                        :aria-describedby="visualFieldHelpId(field)"
                        @blur="emit('commit-field-literal', field)"
                      />

                      <select
                        v-else-if="field.schema.control === 'select'"
                        :id="visualFieldInputId(field)"
                        v-model="fieldDrafts[field.schema.pointer]"
                        class="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                        :aria-labelledby="visualFieldLabelId(field)"
                        :aria-describedby="visualFieldHelpId(field)"
                        @change="emit('commit-field-literal', field)"
                      >
                        <option v-if="!field.schema.required" value="">{{ t("Not set") }}</option>
                        <option
                          v-for="option in field.schema.options ?? []"
                          :key="JSON.stringify(option.value)"
                          :value="JSON.stringify(option.value)"
                        >
                          {{ option.label }}
                        </option>
                      </select>

                      <label
                        v-else-if="field.schema.control === 'switch'"
                        class="flex h-10 items-center gap-3 rounded-md border border-input bg-background px-3 text-sm"
                      >
                        <input
                          :id="visualFieldInputId(field)"
                          :checked="Boolean(field.state.literalValue)"
                          type="checkbox"
                          class="h-4 w-4 rounded border"
                          :aria-labelledby="visualFieldLabelId(field)"
                          :aria-describedby="visualFieldHelpId(field)"
                          @change="emitBooleanFieldChange(field, $event)"
                        />
                        <span>{{ t("Enabled") }}</span>
                      </label>

                      <textarea
                        v-else
                        :id="visualFieldInputId(field)"
                        v-model="fieldDrafts[field.schema.pointer]"
                        rows="6"
                        class="w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-xs"
                        :aria-labelledby="visualFieldLabelId(field)"
                        :aria-describedby="visualFieldHelpId(field)"
                        @blur="emit('commit-field-literal', field)"
                      />
                    </div>
                  </div>
                </div>
              </div>

              <div v-else class="rounded-xl border border-amber-500/40 bg-amber-500/10 p-4">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div class="min-w-0">
                    <p class="text-xs font-semibold uppercase tracking-[0.2em] text-amber-700">
                      {{ t("Visual form unavailable") }}
                    </p>
                    <p class="mt-2 text-sm text-foreground">
                      {{ t("This call node cannot be edited safely in ordinary mode. Use Advanced JSON for the full spec.") }}
                    </p>
                  </div>
                  <Button size="sm" variant="outline" @click="emit('toggle-spec-mode', 'json')">
                    {{ t("Open Advanced JSON") }}
                  </Button>
                </div>
                <ul v-if="visibleVisualCompatibilityReasons.length" class="mt-4 space-y-2 text-xs text-muted-foreground">
                  <li
                    v-for="reason in visibleVisualCompatibilityReasons"
                    :key="`${reason.code}:${reason.pointer ?? '-'}`"
                    class="rounded-lg border border-amber-500/20 bg-background/70 px-3 py-3"
                  >
                    <div class="flex flex-wrap items-center gap-2">
                      <span class="rounded-full border border-amber-500/30 px-2 py-0.5 text-[10px] uppercase tracking-[0.16em] text-amber-700">
                        {{ visualCompatibilityReasonCategory(reason) }}
                      </span>
                      <p class="text-sm font-medium text-foreground">
                        {{ describeVisualCompatibilityReason(reason) }}
                      </p>
                    </div>
                    <p class="mt-1 text-[11px] text-muted-foreground">
                      {{ visualCompatibilityReasonHelp(reason) }}
                    </p>
                  </li>
                </ul>
              </div>
            </div>

            <div v-else class="space-y-4">
              <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Compose Node") }}
                </p>
                <p class="mt-2 text-sm text-muted-foreground">
                  {{ t("Compose nodes do not call capabilities. They build a JSON result locally from template + bindings.") }}
                </p>
              </div>

              <div>
                <label :for="inspectorFieldId('compose-template')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Template (JSON)") }}
                </label>
                <textarea
                  :id="inspectorFieldId('compose-template')"
                  v-model="selectedNode.composeTemplate"
                  rows="9"
                  class="mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-xs"
                  @blur="emit('commit-history')"
                />
                <p class="mt-1 text-[11px] text-muted-foreground">
                  {{ t("Compose starts from this JSON template and applies the same binding list as call nodes.") }}
                </p>
              </div>

              <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                      {{ t("Input Bindings") }}
                    </p>
                    <p class="mt-1 text-[11px] text-muted-foreground">
                      {{ t("Bindings can read trigger data, flow/run metadata, or ancestor node results.") }}
                    </p>
                  </div>
                  <Button variant="outline" size="sm" @click="emit('add-binding')">{{ t("Add Binding") }}</Button>
                </div>

                <div
                  v-if="!selectedNode.inputs.length"
                  class="mt-4 rounded-lg border border-dashed border-border/60 px-4 py-5 text-center text-xs text-muted-foreground"
                >
                  {{ t("No bindings yet. Nodes can still run with their template alone.") }}
                </div>

                <div v-else class="mt-4 space-y-3">
                  <div
                    v-for="(binding, index) in selectedNode.inputs"
                    :key="`${selectedNode.id}-binding-${index}`"
                    class="rounded-lg border border-border/70 bg-background/90 p-3"
                  >
                    <div class="flex items-start justify-between gap-3">
                      <div>
                        <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                          {{ t("Binding {index}", { index: index + 1 }) }}
                        </p>
                        <p class="mt-1 text-[11px] text-muted-foreground">
                          {{ t("Destination writes into the template. Source chooses where the value comes from.") }}
                        </p>
                      </div>
                      <Button size="sm" variant="ghost" @click="emit('remove-binding', index)">{{ t("Remove") }}</Button>
                    </div>

                    <div class="mt-3 grid gap-3 md:grid-cols-2">
                      <div>
                        <label :for="composeBindingInputId(index, 'destination')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                          {{ t("Destination Pointer") }}
                        </label>
                        <input
                          :id="composeBindingInputId(index, 'destination')"
                          v-model="binding.to"
                          class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                          placeholder="/payload/id"
                          @blur="emit('commit-history')"
                        />
                      </div>

                      <div>
                        <label :for="composeBindingInputId(index, 'source-kind')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                          {{ t("Source Kind") }}
                        </label>
                        <select
                          :id="composeBindingInputId(index, 'source-kind')"
                          :value="binding.sourceKind"
                          class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                          @change="emitBindingSourceKindChange(binding, $event)"
                        >
                          <option value="node_result">{{ t("Ancestor Result") }}</option>
                          <option value="trigger">{{ t("Trigger") }}</option>
                          <option value="flow_meta">{{ t("Flow Meta") }}</option>
                          <option value="run_meta">{{ t("Run Meta") }}</option>
                        </select>
                      </div>
                    </div>

                    <div v-if="binding.sourceKind === 'node_result'" class="mt-3 grid gap-3 md:grid-cols-2">
                      <div>
                        <label :for="composeBindingInputId(index, 'ancestor-node')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                          {{ t("Ancestor Node") }}
                        </label>
                        <select
                          :id="composeBindingInputId(index, 'ancestor-node')"
                          v-model="binding.nodeId"
                          class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                          @change="emit('commit-history')"
                        >
                          <option value="">
                            {{ ancestorNodeOptions.length ? t("Select ancestor node") : t("No ancestor available") }}
                          </option>
                          <option v-for="ancestorId in ancestorNodeOptions" :key="ancestorId" :value="ancestorId">
                            {{ ancestorId }}
                          </option>
                        </select>
                      </div>

                      <div>
                        <label :for="composeBindingInputId(index, 'result-path')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                          {{ t("Result Path") }}
                        </label>
                        <input
                          :id="composeBindingInputId(index, 'result-path')"
                          v-model="binding.path"
                          class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                          placeholder="/user/id"
                          @blur="emit('commit-history')"
                        />
                      </div>
                    </div>

                    <div v-else-if="binding.sourceKind === 'trigger'" class="mt-3">
                      <label :for="composeBindingInputId(index, 'trigger-path')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                        {{ t("Trigger Path") }}
                      </label>
                      <input
                        :id="composeBindingInputId(index, 'trigger-path')"
                        v-model="binding.path"
                        class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                        placeholder="/payload/name"
                        @blur="emit('commit-history')"
                      />
                    </div>

                    <div
                      v-else-if="binding.sourceKind === 'flow_meta' || binding.sourceKind === 'run_meta'"
                      class="mt-3"
                    >
                      <label :for="composeBindingInputId(index, 'meta-field')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                        {{ t("Meta Field") }}
                      </label>
                      <select
                        :id="composeBindingInputId(index, 'meta-field')"
                        v-model="binding.field"
                        class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                        @change="emit('commit-history')"
                      >
                        <option v-if="binding.sourceKind === 'flow_meta'" value="flow_id">flow_id</option>
                        <option v-if="binding.sourceKind === 'run_meta'" value="run_id">run_id</option>
                      </select>
                    </div>

                    <label class="mt-3 flex items-center gap-2 text-xs text-muted-foreground">
                      <input
                        :id="composeBindingInputId(index, 'required')"
                        v-model="binding.required"
                        type="checkbox"
                        class="h-4 w-4 rounded border"
                        @change="emit('commit-history')"
                      />
                      <span>{{ t("Required binding") }}</span>
                    </label>
                  </div>
                </div>
              </div>
            </div>
          </template>

          <div v-else class="space-y-3">
            <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
              <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Advanced Spec (JSON)") }}
              </p>
              <p class="mt-2 text-[11px] text-muted-foreground">
                {{ t("This is the full node spec. Switching back to form mode will validate JSON and map the supported fields into the visual editor.") }}
              </p>
              <textarea
                v-model="selectedNode.specJson"
                rows="16"
                class="mt-3 w-full rounded-md border border-input bg-background px-3 py-2 text-xs"
                @blur="emit('commit-history')"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  </aside>
</template>
