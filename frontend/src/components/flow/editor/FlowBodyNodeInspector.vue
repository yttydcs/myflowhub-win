<script setup lang="ts">
import { computed } from "vue"
import { X } from "lucide-vue-next"
import CardHeader from "@/components/CardHeader.vue"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"
import { describeVisualCompatibilityReason, type FlowNodeDraft, type FlowNodeKind, type NodeVisualFormModel, type VisualCompatibilityReason, type VisualFieldModel } from "@/stores/flow"

const props = defineProps<{
  selectedNode: FlowNodeDraft | null
  nodeIdDraft: string
  selectedTargetLabel: string
  selectedCallVisualForm: NodeVisualFormModel | null
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
  (event: "commit-history"): void
}>()

const { t } = useI18n()

const allowedKinds: FlowNodeKind[] = ["call", "compose", "transform", "set_var", "branch", "foreach", "subflow"]

const updateNodeIdDraft = (event: Event) => {
  emit("update:nodeIdDraft", String((event.target as HTMLInputElement | null)?.value ?? ""))
}

const emitNodeKindChange = (event: Event) => {
  const value = String((event.target as HTMLSelectElement | null)?.value ?? "call") as FlowNodeKind
  emit("node-kind-change", allowedKinds.includes(value) ? value : "call")
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
  `flow-body-node-inspector-${toDomIdPart(props.selectedNode?.id || "node")}-${toDomIdPart(suffix)}`

const visualFieldInputId = (field: VisualFieldModel) => inspectorFieldId(`visual-${field.schema.pointer}`)
const visualFieldLabelId = (field: VisualFieldModel) => `${visualFieldInputId(field)}-label`
const visualFieldHelpId = (field: VisualFieldModel) => `${visualFieldInputId(field)}-help`

const visibleVisualCompatibilityReasons = computed(() =>
  (props.selectedCallVisualForm?.compatibility.reasons ?? []).filter((reason) => reason.code !== "missing_schema")
)

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
      return t("Ordinary mode allows only one binding per field.")
    case "extra_literal_field":
      return t("Remove extra literal fields or continue in Advanced JSON for this node.")
    default:
      return t("Review the current node spec in Advanced JSON.")
  }
}
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
          :title="selectedNode.id || t('Unnamed')"
          :description="t('Foreach Body Node')"
          title-class="text-lg"
          :title-id="inspectorFieldId('title')"
        >
          <template #actions>
            <Button type="button" variant="ghost" size="icon" @click="emit('close')">
              <X class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">{{ t("Close") }}</span>
            </Button>
          </template>
        </CardHeader>
      </div>

      <div class="flex-1 overflow-y-auto px-5 py-4">
        <div class="space-y-4">
          <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
            <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ selectedNode.kind === "call" ? t("Body call authoring") : t("JSON-first authoring") }}
            </p>
            <p class="mt-2 text-sm text-muted-foreground">
              {{
                selectedNode.kind === "call"
                  ? t(
                      "Call nodes inside the nested body editor can reuse capability methods, ordinary fields, and bindings here. Other body node kinds stay JSON-first."
                    )
                  : t(
                      "The nested body editor manages graph structure visually, but non-call body node payloads still use Advanced JSON here so root save and recovery stay aligned."
                    )
              }}
            </p>
          </div>

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
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label :for="inspectorFieldId('kind')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Node Kind") }}
              </label>
              <select
                :id="inspectorFieldId('kind')"
                :value="selectedNode.kind"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                @change="emitNodeKindChange"
              >
                <option value="call">{{ t("Call") }}</option>
                <option value="compose">{{ t("Compose") }}</option>
                <option value="transform">{{ t("Transform") }}</option>
                <option value="set_var">{{ t("Set Var") }}</option>
                <option value="branch">{{ t("Branch") }}</option>
                <option value="foreach">{{ t("Foreach") }}</option>
                <option value="subflow">{{ t("Subflow") }}</option>
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
                  @change="emit('commit-history')"
                />
                <span class="text-muted-foreground">
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

          <template v-if="selectedNode.kind === 'call'">
            <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div :id="inspectorFieldId('spec-mode-label')">
                  <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    {{ t("Spec Mode") }}
                  </p>
                  <p class="mt-1 text-[11px] text-muted-foreground">
                    {{ t("Body call nodes support ordinary field editing when the method exposes a compatible input schema.") }}
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

            <div v-if="selectedNode.specEditorMode === 'form'" class="space-y-4">
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
                      {{ t("This body call node cannot be edited safely in ordinary mode. Use Advanced JSON for the full spec.") }}
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
          </template>

          <div v-if="selectedNode.kind !== 'call' || selectedNode.specEditorMode === 'json'">
            <label :for="inspectorFieldId('advanced-json')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Advanced JSON") }}
            </label>
            <textarea
              :id="inspectorFieldId('advanced-json')"
              v-model="selectedNode.specJson"
              rows="18"
              class="mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-xs"
              @blur="emit('commit-history')"
            />
          </div>
        </div>
      </div>
    </div>
  </aside>
</template>
