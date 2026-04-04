<script setup lang="ts">
import { computed } from "vue"
import { X } from "lucide-vue-next"
import CardHeader from "@/components/CardHeader.vue"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"
import { buildDetailStructuredFields, describeVisualCompatibilityReason, flowStatusLabelKey } from "@/stores/flow"
import type {
  FlowBindingSourceKind,
  FlowInputBindingDraft,
  FlowSourceDraft,
  FlowTransformExprMode,
  FlowNodeDetailState,
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
  nodeDetail: FlowNodeDetailState
  selectedNodeOutputSchemaText: string
  fieldDrafts: Record<string, any>
}>()

const emit = defineEmits<{
  (event: "close"): void
  (event: "update:nodeIdDraft", value: string): void
  (event: "commit-node-id"): void
  (event: "node-kind-change", value: FlowNodeKind): void
  (event: "toggle-spec-mode", value: "form" | "json"): void
  (event: "update:nodeDetailRunId", value: string): void
  (event: "update:nodeDetailPath", value: string): void
  (event: "load-node-detail"): void
  (event: "open-method"): void
  (event: "edit-foreach-body"): void
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

const updateNodeDetailRunId = (event: Event) => {
  emit("update:nodeDetailRunId", String((event.target as HTMLInputElement | null)?.value ?? ""))
}

const updateNodeDetailPath = (event: Event) => {
  emit("update:nodeDetailPath", String((event.target as HTMLInputElement | null)?.value ?? ""))
}

const emitNodeKindChange = (event: Event) => {
  const value = String((event.target as HTMLSelectElement | null)?.value ?? "call") as FlowNodeKind
  const allowedKinds: FlowNodeKind[] = ["call", "compose", "transform", "set_var", "branch", "foreach", "subflow"]
  emit("node-kind-change", allowedKinds.includes(value) ? value : "call")
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
const sourceInputId = (prefix: string, suffix: string) => inspectorFieldId(`${prefix}-${suffix}`)

const transformExprModes: Array<{ value: FlowTransformExprMode; label: string }> = [
  { value: "literal", label: "Literal" },
  { value: "source", label: "Source" },
  { value: "op", label: "Operation" },
  { value: "object", label: "Object" },
  { value: "array", label: "Array" }
]

const transformOps = [
  "add",
  "sub",
  "mul",
  "div",
  "mod",
  "neg",
  "abs",
  "min",
  "max",
  "eq",
  "ne",
  "gt",
  "gte",
  "lt",
  "lte",
  "and",
  "or",
  "not",
  "coalesce",
  "if",
  "concat",
  "lower",
  "upper",
  "trim",
  "len"
]

const branchMatchOps = ["eq", "ne", "gt", "gte", "lt", "lte", "exists"]

const normalizeSourceKind = (raw: string): FlowBindingSourceKind => {
  switch (raw) {
    case "node_result":
    case "trigger":
    case "flow_meta":
    case "run_meta":
    case "flow_var":
      return raw
    default:
      return "trigger"
  }
}

const resetSourceDraftForKind = (source: FlowSourceDraft, sourceKind: FlowBindingSourceKind) => {
  source.sourceKind = sourceKind
  source.nodeId = ""
  source.path = ""
  source.name = ""
  source.field = sourceKind === "flow_meta" ? "flow_id" : sourceKind === "run_meta" ? "run_id" : ""
}

const emitSourceKindChange = (source: FlowSourceDraft, event: Event) => {
  resetSourceDraftForKind(source, normalizeSourceKind(String((event.target as HTMLSelectElement | null)?.value ?? "trigger")))
  emit("commit-history")
}

const createBranchCaseKey = () =>
  typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `branch-case-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`

const addBranchCase = () => {
  if (!props.selectedNode) return
  props.selectedNode.branchCases.push({
    key: createBranchCaseKey(),
    name: "",
    source: {
      sourceKind: "trigger",
      nodeId: "",
      path: "",
      field: "",
      name: ""
    },
    op: "eq",
    valueJson: "null"
  })
  emit("commit-history")
}

const removeBranchCase = (index: number) => {
  if (!props.selectedNode) return
  props.selectedNode.branchCases.splice(index, 1)
  emit("commit-history")
}

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

const supportsFormMode = computed(() =>
  ["call", "compose", "transform", "set_var", "branch", "foreach", "subflow"].includes(props.selectedNode?.kind ?? "")
)

const nodeKindLabel = computed(() => {
  switch (props.selectedNode?.kind) {
    case "compose":
      return t("Compose")
    case "transform":
      return t("Transform")
    case "set_var":
      return t("Set Var")
    case "branch":
      return t("Branch")
    case "foreach":
      return t("Foreach")
    case "subflow":
      return t("Subflow")
    default:
      return t("Call")
  }
})

const jsonOnlyKindSummary = computed(() => {
  switch (props.selectedNode?.kind) {
    case "transform":
      return t("Transform nodes evaluate a structured expression tree and produce a local result without calling a capability.")
    case "branch":
      return t("Branch nodes match ordered cases and route execution through edges that declare edge.case.")
    case "foreach":
      return t("Foreach nodes iterate an array source and execute a nested body graph. Ordinary mode edits the outer fields while the body graph stays as JSON.")
    case "subflow":
      return t("Subflow nodes synchronously execute another flow on the same executor and can optionally read one result node.")
    default:
      return t("This node kind currently uses Advanced JSON authoring.")
  }
})

const hasLoadedNodeDetail = computed(
  () =>
    Boolean(props.nodeDetail.runId) ||
    Boolean(props.nodeDetail.path) ||
    Boolean(props.nodeDetail.node) ||
    Boolean(props.nodeDetail.resultText)
)

const nodeDetailStatusLabel = computed(() => t(flowStatusLabelKey(props.nodeDetail.node?.status ?? "")))
const nodeDetailResolvedPathLabel = computed(() => props.nodeDetail.path || t("Root result"))
const selectedNodeOutputSchemaLabel = computed(() => props.selectedNodeOutputSchemaText || t("No schema"))
const structuredDetailFields = computed(() =>
  buildDetailStructuredFields(
    props.selectedNodeOutputSchemaText,
    props.nodeDetail.resultValue,
    props.nodeDetail.path
  )
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
                  :aria-describedby="inspectorFieldId('allow-fail-help')"
                  @change="emit('commit-history')"
                />
                <span :id="inspectorFieldId('allow-fail-help')" class="text-muted-foreground">
                  {{ t("Continue on error") }}
                </span>
              </div>
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-3">
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
              <label :for="inspectorFieldId('retry-backoff-ms')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Retry Backoff (ms)") }}
              </label>
              <input
                :id="inspectorFieldId('retry-backoff-ms')"
                v-model.number="selectedNode.retryBackoffMs"
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
                  {{
                    supportsFormMode
                      ? t("Form mode edits the supported fields directly. Advanced JSON is the escape hatch for full spec editing.")
                      : t("This node kind currently uses Advanced JSON authoring. Ordinary form controls are not available yet.")
                  }}
                </p>
              </div>
              <div class="flex gap-2" role="group" :aria-labelledby="inspectorFieldId('spec-mode-label')">
                <Button
                  size="sm"
                  :variant="selectedNode.specEditorMode === 'form' ? 'default' : 'outline'"
                  :disabled="!supportsFormMode"
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

            <div v-else-if="selectedNode.kind === 'transform'" class="space-y-4">
              <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Transform Node") }}
                </p>
                <p class="mt-2 text-sm text-muted-foreground">
                  {{ t("Transform evaluates one structured expression and writes the computed result into the current node output.") }}
                </p>
              </div>

              <div>
                <label :for="inspectorFieldId('transform-expr-mode')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Expression Mode") }}
                </label>
                <select
                  :id="inspectorFieldId('transform-expr-mode')"
                  v-model="selectedNode.transformExprMode"
                  class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                  @change="emit('commit-history')"
                >
                  <option v-for="mode in transformExprModes" :key="mode.value" :value="mode.value">
                    {{ t(mode.label) }}
                  </option>
                </select>
              </div>

              <div v-if="selectedNode.transformExprMode === 'literal'">
                <label :for="inspectorFieldId('transform-literal')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Literal (JSON)") }}
                </label>
                <textarea
                  :id="inspectorFieldId('transform-literal')"
                  v-model="selectedNode.transformLiteralJson"
                  rows="8"
                  class="mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-xs"
                  @blur="emit('commit-history')"
                />
                <p class="mt-1 text-[11px] text-muted-foreground">
                  {{ t("Enter any valid JSON value. The transform result will be this literal value.") }}
                </p>
              </div>

              <div v-else-if="selectedNode.transformExprMode === 'source'" class="space-y-4">
                <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                  <div class="flex flex-wrap items-start justify-between gap-3">
                    <div>
                      <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                        {{ t("Source") }}
                      </p>
                      <p class="mt-1 text-[11px] text-muted-foreground">
                        {{ t("Source mode reuses the same trigger/meta/ancestor/local-var binding contract as other flow inputs.") }}
                      </p>
                    </div>
                  </div>

                  <div class="mt-4 grid gap-3 md:grid-cols-2">
                    <div>
                      <label :for="sourceInputId('transform-source', 'kind')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                        {{ t("Source Kind") }}
                      </label>
                      <select
                        :id="sourceInputId('transform-source', 'kind')"
                        :value="selectedNode.transformSource.sourceKind"
                        class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                        @change="emitSourceKindChange(selectedNode.transformSource, $event)"
                      >
                        <option value="node_result">{{ t("Ancestor Result") }}</option>
                        <option value="trigger">{{ t("Trigger") }}</option>
                        <option value="flow_meta">{{ t("Flow Meta") }}</option>
                        <option value="run_meta">{{ t("Run Meta") }}</option>
                        <option value="flow_var">{{ t("Flow Local Var") }}</option>
                      </select>
                    </div>

                    <label class="mt-6 flex items-center gap-2 text-xs text-muted-foreground md:mt-7">
                      <input
                        :id="sourceInputId('transform-source', 'required')"
                        v-model="selectedNode.transformSourceRequired"
                        type="checkbox"
                        class="h-4 w-4 rounded border"
                        @change="emit('commit-history')"
                      />
                      <span>{{ t("Required source") }}</span>
                    </label>
                  </div>

                  <div v-if="selectedNode.transformSource.sourceKind === 'node_result'" class="mt-3 grid gap-3 md:grid-cols-2">
                    <div>
                      <label :for="sourceInputId('transform-source', 'ancestor-node')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                        {{ t("Ancestor Node") }}
                      </label>
                      <select
                        :id="sourceInputId('transform-source', 'ancestor-node')"
                        v-model="selectedNode.transformSource.nodeId"
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
                      <label :for="sourceInputId('transform-source', 'result-path')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                        {{ t("Result Path") }}
                      </label>
                      <input
                        :id="sourceInputId('transform-source', 'result-path')"
                        v-model="selectedNode.transformSource.path"
                        class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                        placeholder="/user/id"
                        @blur="emit('commit-history')"
                      />
                    </div>
                  </div>

                  <div v-else-if="selectedNode.transformSource.sourceKind === 'trigger'" class="mt-3">
                    <label :for="sourceInputId('transform-source', 'trigger-path')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                      {{ t("Trigger Path") }}
                    </label>
                    <input
                      :id="sourceInputId('transform-source', 'trigger-path')"
                      v-model="selectedNode.transformSource.path"
                      class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                      placeholder="/payload/name"
                      @blur="emit('commit-history')"
                    />
                  </div>

                  <div
                    v-else-if="selectedNode.transformSource.sourceKind === 'flow_meta' || selectedNode.transformSource.sourceKind === 'run_meta'"
                    class="mt-3"
                  >
                    <label :for="sourceInputId('transform-source', 'meta-field')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                      {{ t("Meta Field") }}
                    </label>
                    <select
                      :id="sourceInputId('transform-source', 'meta-field')"
                      v-model="selectedNode.transformSource.field"
                      class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                      @change="emit('commit-history')"
                    >
                      <option v-if="selectedNode.transformSource.sourceKind === 'flow_meta'" value="flow_id">flow_id</option>
                      <option v-if="selectedNode.transformSource.sourceKind === 'run_meta'" value="run_id">run_id</option>
                    </select>
                  </div>

                  <div v-else-if="selectedNode.transformSource.sourceKind === 'flow_var'" class="mt-3 grid gap-3 md:grid-cols-2">
                    <div>
                      <label :for="sourceInputId('transform-source', 'flow-var-name')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                        {{ t("Local Var Name") }}
                      </label>
                      <input
                        :id="sourceInputId('transform-source', 'flow-var-name')"
                        v-model="selectedNode.transformSource.name"
                        class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                        placeholder="session_token"
                        @blur="emit('commit-history')"
                      />
                    </div>

                    <div>
                      <label :for="sourceInputId('transform-source', 'flow-var-path')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                        {{ t("Value Path") }}
                      </label>
                      <input
                        :id="sourceInputId('transform-source', 'flow-var-path')"
                        v-model="selectedNode.transformSource.path"
                        class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                        placeholder="/payload/id"
                        @blur="emit('commit-history')"
                      />
                    </div>
                  </div>
                </div>
              </div>

              <div v-else-if="selectedNode.transformExprMode === 'op'" class="space-y-4">
                <div>
                  <label :for="inspectorFieldId('transform-op')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    {{ t("Operation") }}
                  </label>
                  <select
                    :id="inspectorFieldId('transform-op')"
                    v-model="selectedNode.transformOp"
                    class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                    @change="emit('commit-history')"
                  >
                    <option v-for="op in transformOps" :key="op" :value="op">
                      {{ op }}
                    </option>
                  </select>
                </div>

                <div>
                  <label :for="inspectorFieldId('transform-args')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    {{ t("Arguments (JSON Array)") }}
                  </label>
                  <textarea
                    :id="inspectorFieldId('transform-args')"
                    v-model="selectedNode.transformArgsJson"
                    rows="10"
                    class="mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-xs"
                    @blur="emit('commit-history')"
                  />
                  <p class="mt-1 text-[11px] text-muted-foreground">
                    {{ t("Each array entry should be a nested transform expression object, for example {\"literal\": 1}.") }}
                  </p>
                </div>
              </div>

              <div v-else-if="selectedNode.transformExprMode === 'object'">
                <label :for="inspectorFieldId('transform-object')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Object Entries (JSON)") }}
                </label>
                <textarea
                  :id="inspectorFieldId('transform-object')"
                  v-model="selectedNode.transformObjectJson"
                  rows="10"
                  class="mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-xs"
                  @blur="emit('commit-history')"
                />
                <p class="mt-1 text-[11px] text-muted-foreground">
                  {{ t("Use an object whose values are nested transform expression objects.") }}
                </p>
              </div>

              <div v-else>
                <label :for="inspectorFieldId('transform-array')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Array Items (JSON)") }}
                </label>
                <textarea
                  :id="inspectorFieldId('transform-array')"
                  v-model="selectedNode.transformArrayJson"
                  rows="10"
                  class="mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-xs"
                  @blur="emit('commit-history')"
                />
                <p class="mt-1 text-[11px] text-muted-foreground">
                  {{ t("Use a JSON array whose entries are nested transform expression objects.") }}
                </p>
              </div>
            </div>

            <div v-else-if="selectedNode.kind === 'branch'" class="space-y-4">
              <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Branch Node") }}
                </p>
                <p class="mt-2 text-sm text-muted-foreground">
                  {{ t("Branch checks ordered cases and routes execution through edges whose edge.case matches the selected case.") }}
                </p>
              </div>

              <div>
                <label :for="inspectorFieldId('branch-default-case')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Default Case") }}
                </label>
                <select
                  :id="inspectorFieldId('branch-default-case')"
                  v-model="selectedNode.branchDefaultCase"
                  class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                  @change="emit('commit-history')"
                >
                  <option value="">{{ t("No default case") }}</option>
                  <option v-for="branchCase in selectedNode.branchCases" :key="branchCase.key" :value="branchCase.name">
                    {{ branchCase.name || t("Unnamed case") }}
                  </option>
                </select>
                <p class="mt-1 text-[11px] text-muted-foreground">
                  {{ t("If no case matches, branch activates this route. The edge itself still needs the matching edge.case value.") }}
                </p>
              </div>

              <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                      {{ t("Cases") }}
                    </p>
                    <p class="mt-1 text-[11px] text-muted-foreground">
                      {{ t("Cases run in order. Keep case names aligned with outgoing branch edge.case values.") }}
                    </p>
                  </div>
                  <Button variant="outline" size="sm" @click="addBranchCase">{{ t("Add Case") }}</Button>
                </div>

                <div
                  v-if="!selectedNode.branchCases.length"
                  class="mt-4 rounded-lg border border-dashed border-border/60 px-4 py-5 text-center text-xs text-muted-foreground"
                >
                  {{ t("No branch cases yet. Add at least one case before wiring outgoing routes.") }}
                </div>

                <div v-else class="mt-4 space-y-3">
                  <div
                    v-for="(branchCase, index) in selectedNode.branchCases"
                    :key="branchCase.key"
                    class="rounded-lg border border-border/70 bg-background/90 p-4"
                  >
                    <div class="flex items-start justify-between gap-3">
                      <div>
                        <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                          {{ t("Case {index}", { index: index + 1 }) }}
                        </p>
                        <p class="mt-1 text-[11px] text-muted-foreground">
                          {{ t("The first matching case wins. Use exact case names again on outgoing branch edges.") }}
                        </p>
                      </div>
                      <Button size="sm" variant="ghost" @click="removeBranchCase(index)">{{ t("Remove") }}</Button>
                    </div>

                    <div class="mt-3 grid gap-3 md:grid-cols-2">
                      <div>
                        <label :for="sourceInputId(`branch-case-${branchCase.key}`, 'name')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                          {{ t("Case Name") }}
                        </label>
                        <input
                          :id="sourceInputId(`branch-case-${branchCase.key}`, 'name')"
                          v-model="branchCase.name"
                          class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                          placeholder="approved"
                          @blur="emit('commit-history')"
                        />
                      </div>

                      <div>
                        <label :for="sourceInputId(`branch-case-${branchCase.key}`, 'op')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                          {{ t("Match Op") }}
                        </label>
                        <select
                          :id="sourceInputId(`branch-case-${branchCase.key}`, 'op')"
                          v-model="branchCase.op"
                          class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                          @change="emit('commit-history')"
                        >
                          <option v-for="op in branchMatchOps" :key="op" :value="op">
                            {{ op }}
                          </option>
                        </select>
                      </div>
                    </div>

                    <div class="mt-3 grid gap-3 md:grid-cols-2">
                      <div>
                        <label :for="sourceInputId(`branch-case-${branchCase.key}`, 'source-kind')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                          {{ t("Source Kind") }}
                        </label>
                        <select
                          :id="sourceInputId(`branch-case-${branchCase.key}`, 'source-kind')"
                          :value="branchCase.source.sourceKind"
                          class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                          @change="emitSourceKindChange(branchCase.source, $event)"
                        >
                          <option value="node_result">{{ t("Ancestor Result") }}</option>
                          <option value="trigger">{{ t("Trigger") }}</option>
                          <option value="flow_meta">{{ t("Flow Meta") }}</option>
                          <option value="run_meta">{{ t("Run Meta") }}</option>
                          <option value="flow_var">{{ t("Flow Local Var") }}</option>
                        </select>
                      </div>
                    </div>

                    <div v-if="branchCase.source.sourceKind === 'node_result'" class="mt-3 grid gap-3 md:grid-cols-2">
                      <div>
                        <label :for="sourceInputId(`branch-case-${branchCase.key}`, 'ancestor-node')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                          {{ t("Ancestor Node") }}
                        </label>
                        <select
                          :id="sourceInputId(`branch-case-${branchCase.key}`, 'ancestor-node')"
                          v-model="branchCase.source.nodeId"
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
                        <label :for="sourceInputId(`branch-case-${branchCase.key}`, 'result-path')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                          {{ t("Result Path") }}
                        </label>
                        <input
                          :id="sourceInputId(`branch-case-${branchCase.key}`, 'result-path')"
                          v-model="branchCase.source.path"
                          class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                          placeholder="/payload/status"
                          @blur="emit('commit-history')"
                        />
                      </div>
                    </div>

                    <div v-else-if="branchCase.source.sourceKind === 'trigger'" class="mt-3">
                      <label :for="sourceInputId(`branch-case-${branchCase.key}`, 'trigger-path')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                        {{ t("Trigger Path") }}
                      </label>
                      <input
                        :id="sourceInputId(`branch-case-${branchCase.key}`, 'trigger-path')"
                        v-model="branchCase.source.path"
                        class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                        placeholder="/payload/status"
                        @blur="emit('commit-history')"
                      />
                    </div>

                    <div
                      v-else-if="branchCase.source.sourceKind === 'flow_meta' || branchCase.source.sourceKind === 'run_meta'"
                      class="mt-3"
                    >
                      <label :for="sourceInputId(`branch-case-${branchCase.key}`, 'meta-field')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                        {{ t("Meta Field") }}
                      </label>
                      <select
                        :id="sourceInputId(`branch-case-${branchCase.key}`, 'meta-field')"
                        v-model="branchCase.source.field"
                        class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                        @change="emit('commit-history')"
                      >
                        <option v-if="branchCase.source.sourceKind === 'flow_meta'" value="flow_id">flow_id</option>
                        <option v-if="branchCase.source.sourceKind === 'run_meta'" value="run_id">run_id</option>
                      </select>
                    </div>

                    <div v-else-if="branchCase.source.sourceKind === 'flow_var'" class="mt-3 grid gap-3 md:grid-cols-2">
                      <div>
                        <label :for="sourceInputId(`branch-case-${branchCase.key}`, 'flow-var-name')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                          {{ t("Local Var Name") }}
                        </label>
                        <input
                          :id="sourceInputId(`branch-case-${branchCase.key}`, 'flow-var-name')"
                          v-model="branchCase.source.name"
                          class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                          placeholder="session_token"
                          @blur="emit('commit-history')"
                        />
                      </div>

                      <div>
                        <label :for="sourceInputId(`branch-case-${branchCase.key}`, 'flow-var-path')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                          {{ t("Value Path") }}
                        </label>
                        <input
                          :id="sourceInputId(`branch-case-${branchCase.key}`, 'flow-var-path')"
                          v-model="branchCase.source.path"
                          class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                          placeholder="/payload/status"
                          @blur="emit('commit-history')"
                        />
                      </div>
                    </div>

                    <div v-if="branchCase.op !== 'exists'" class="mt-3">
                      <label :for="sourceInputId(`branch-case-${branchCase.key}`, 'value-json')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                        {{ t("Match Value (JSON)") }}
                      </label>
                      <textarea
                        :id="sourceInputId(`branch-case-${branchCase.key}`, 'value-json')"
                        v-model="branchCase.valueJson"
                        rows="5"
                        class="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-xs"
                        @blur="emit('commit-history')"
                      />
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div v-else-if="selectedNode.kind === 'foreach'" class="space-y-4">
              <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Foreach Node") }}
                </p>
                <p class="mt-2 text-sm text-muted-foreground">
                  {{ t("Foreach reads an array source, runs the nested body graph once per item, and collects one body node result into the final array.") }}
                </p>
              </div>

              <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                      {{ t("Source") }}
                    </p>
                    <p class="mt-1 text-[11px] text-muted-foreground">
                      {{ t("Foreach source reuses the same trigger/meta/ancestor/local-var binding contract as other flow inputs.") }}
                    </p>
                  </div>
                </div>

                <div class="mt-4 grid gap-3 md:grid-cols-2">
                  <div>
                    <label :for="sourceInputId('foreach-source', 'kind')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                      {{ t("Source Kind") }}
                    </label>
                    <select
                      :id="sourceInputId('foreach-source', 'kind')"
                      :value="selectedNode.foreachSource.sourceKind"
                      class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                      @change="emitSourceKindChange(selectedNode.foreachSource, $event)"
                    >
                      <option value="node_result">{{ t("Ancestor Result") }}</option>
                      <option value="trigger">{{ t("Trigger") }}</option>
                      <option value="flow_meta">{{ t("Flow Meta") }}</option>
                      <option value="run_meta">{{ t("Run Meta") }}</option>
                      <option value="flow_var">{{ t("Flow Local Var") }}</option>
                    </select>
                  </div>

                  <label class="mt-6 flex items-center gap-2 text-xs text-muted-foreground">
                    <input
                      :id="sourceInputId('foreach-source', 'required')"
                      v-model="selectedNode.foreachRequired"
                      type="checkbox"
                      class="h-4 w-4 rounded border"
                      @change="emit('commit-history')"
                    />
                    <span>{{ t("Required source") }}</span>
                  </label>
                </div>

                <div v-if="selectedNode.foreachSource.sourceKind === 'node_result'" class="mt-3 grid gap-3 md:grid-cols-2">
                  <div>
                    <label :for="sourceInputId('foreach-source', 'ancestor-node')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                      {{ t("Ancestor Node") }}
                    </label>
                    <select
                      :id="sourceInputId('foreach-source', 'ancestor-node')"
                      v-model="selectedNode.foreachSource.nodeId"
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
                    <label :for="sourceInputId('foreach-source', 'result-path')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                      {{ t("Result Path") }}
                    </label>
                    <input
                      :id="sourceInputId('foreach-source', 'result-path')"
                      v-model="selectedNode.foreachSource.path"
                      class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                      placeholder="/items"
                      @blur="emit('commit-history')"
                    />
                  </div>
                </div>

                <div v-else-if="selectedNode.foreachSource.sourceKind === 'trigger'" class="mt-3">
                  <label :for="sourceInputId('foreach-source', 'trigger-path')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                    {{ t("Trigger Path") }}
                  </label>
                  <input
                    :id="sourceInputId('foreach-source', 'trigger-path')"
                    v-model="selectedNode.foreachSource.path"
                    class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                    placeholder="/items"
                    @blur="emit('commit-history')"
                  />
                </div>

                <div
                  v-else-if="selectedNode.foreachSource.sourceKind === 'flow_meta' || selectedNode.foreachSource.sourceKind === 'run_meta'"
                  class="mt-3"
                >
                  <label :for="sourceInputId('foreach-source', 'meta-field')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                    {{ t("Meta Field") }}
                  </label>
                  <select
                    :id="sourceInputId('foreach-source', 'meta-field')"
                    v-model="selectedNode.foreachSource.field"
                    class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                    @change="emit('commit-history')"
                  >
                    <option v-if="selectedNode.foreachSource.sourceKind === 'flow_meta'" value="flow_id">flow_id</option>
                    <option v-if="selectedNode.foreachSource.sourceKind === 'run_meta'" value="run_id">run_id</option>
                  </select>
                </div>

                <div v-else-if="selectedNode.foreachSource.sourceKind === 'flow_var'" class="mt-3 grid gap-3 md:grid-cols-2">
                  <div>
                    <label :for="sourceInputId('foreach-source', 'flow-var-name')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                      {{ t("Local Var Name") }}
                    </label>
                    <input
                      :id="sourceInputId('foreach-source', 'flow-var-name')"
                      v-model="selectedNode.foreachSource.name"
                      class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                      placeholder="items_batch"
                      @blur="emit('commit-history')"
                    />
                  </div>

                  <div>
                    <label :for="sourceInputId('foreach-source', 'flow-var-path')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                      {{ t("Value Path") }}
                    </label>
                    <input
                      :id="sourceInputId('foreach-source', 'flow-var-path')"
                      v-model="selectedNode.foreachSource.path"
                      class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                      placeholder="/items"
                      @blur="emit('commit-history')"
                    />
                  </div>
                </div>
              </div>

              <div>
                <label :for="inspectorFieldId('foreach-result-node-id')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Result Node ID") }}
                </label>
                <input
                  :id="inspectorFieldId('foreach-result-node-id')"
                  v-model="selectedNode.foreachResultNodeId"
                  class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                  placeholder="item_result"
                  @blur="emit('commit-history')"
                />
                <p class="mt-1 text-[11px] text-muted-foreground">
                  {{ t("Each iteration reads this body node result and appends it to the final array output.") }}
                </p>
              </div>

              <div>
                <label :for="inspectorFieldId('foreach-body')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Body Graph (JSON)") }}
                </label>
                <div class="mt-2 flex justify-end">
                  <Button type="button" variant="outline" size="sm" @click="emit('edit-foreach-body')">
                    {{ t("Open Visual Body Editor") }}
                  </Button>
                </div>
                <textarea
                  :id="inspectorFieldId('foreach-body')"
                  v-model="selectedNode.foreachBodyJson"
                  rows="12"
                  class="mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-xs"
                  @blur="emit('commit-history')"
                />
                <p class="mt-1 text-[11px] text-muted-foreground">
                  {{ t("Ordinary mode covers the outer foreach fields. The visual body editor reuses the same body JSON as its source of truth.") }}
                </p>
              </div>
            </div>

            <div v-else-if="selectedNode.kind === 'compose' || selectedNode.kind === 'set_var' || selectedNode.kind === 'subflow'" class="space-y-4">
              <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{
                    selectedNode.kind === "set_var"
                      ? t("Set Var Node")
                      : selectedNode.kind === "subflow"
                        ? t("Subflow Node")
                        : t("Compose Node")
                  }}
                </p>
                <p class="mt-2 text-sm text-muted-foreground">
                  {{
                    selectedNode.kind === "set_var"
                      ? t("Set var nodes materialize a JSON value, write it to a flow-local variable for the current run, and mirror it as the node result.")
                      : selectedNode.kind === "subflow"
                        ? t("Subflow nodes materialize an input payload, synchronously execute another flow on the same executor, and can optionally read one result node.")
                        : t("Compose nodes do not call capabilities. They build a JSON result locally from template + bindings.")
                  }}
                </p>
              </div>

              <div v-if="selectedNode.kind === 'set_var'">
                <label :for="inspectorFieldId('set-var-name')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Flow Local Var Name") }}
                </label>
                <input
                  :id="inspectorFieldId('set-var-name')"
                  v-model="selectedNode.setVarName"
                  class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                  placeholder="session_token"
                  @blur="emit('commit-history')"
                />
                <p class="mt-1 text-[11px] text-muted-foreground">
                  {{ t("This name is only visible inside the current flow run. It does not read or write varstore.") }}
                </p>
              </div>

              <div v-else-if="selectedNode.kind === 'subflow'" class="space-y-4">
                <div>
                  <label :for="inspectorFieldId('subflow-id')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    {{ t("Flow ID") }}
                  </label>
                  <input
                    :id="inspectorFieldId('subflow-id')"
                    v-model="selectedNode.subflowId"
                    class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                    placeholder="123e4567-e89b-12d3-a456-426614174000"
                    @blur="emit('commit-history')"
                  />
                </div>

                <div>
                  <label :for="inspectorFieldId('subflow-result-node-id')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    {{ t("Result Node ID (Optional)") }}
                  </label>
                  <input
                    :id="inspectorFieldId('subflow-result-node-id')"
                    v-model="selectedNode.subflowResultNodeId"
                    class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                    placeholder="final_result"
                    @blur="emit('commit-history')"
                  />
                  <p class="mt-1 text-[11px] text-muted-foreground">
                    {{ t("Leave blank to keep the default subflow summary result. Set a node ID to read one child node result back into this node.") }}
                  </p>
                </div>
              </div>

              <div>
                <label :for="inspectorFieldId('compose-template')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ selectedNode.kind === "subflow" ? t("Input Template (JSON)") : t("Template (JSON)") }}
                </label>
                <textarea
                  v-if="selectedNode.kind !== 'subflow'"
                  :id="inspectorFieldId('compose-template')"
                  v-model="selectedNode.composeTemplate"
                  rows="9"
                  class="mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-xs"
                  @blur="emit('commit-history')"
                />
                <textarea
                  v-else
                  :id="inspectorFieldId('compose-template')"
                  v-model="selectedNode.subflowInputTemplate"
                  rows="9"
                  class="mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-xs"
                  @blur="emit('commit-history')"
                />
                <p class="mt-1 text-[11px] text-muted-foreground">
                  {{
                    selectedNode.kind === "set_var"
                      ? t("Set var starts from this JSON template, applies bindings, then writes the result into the flow-local variable.")
                      : selectedNode.kind === "subflow"
                        ? t("Subflow starts from this JSON object, applies bindings, then uses the result as the child flow trigger input.")
                      : t("Compose starts from this JSON template and applies the same binding list as call nodes.")
                  }}
                </p>
              </div>

              <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                      {{ t("Input Bindings") }}
                    </p>
                    <p class="mt-1 text-[11px] text-muted-foreground">
                      {{
                        selectedNode.kind === "subflow"
                          ? t("Bindings materialize the child flow input object from trigger data, flow/run metadata, ancestor node results, or flow-local vars.")
                          : t("Bindings can read trigger data, flow/run metadata, ancestor node results, or flow-local vars from the same run.")
                      }}
                    </p>
                  </div>
                  <Button variant="outline" size="sm" @click="emit('add-binding')">{{ t("Add Binding") }}</Button>
                </div>

                <div
                  v-if="!selectedNode.inputs.length"
                  class="mt-4 rounded-lg border border-dashed border-border/60 px-4 py-5 text-center text-xs text-muted-foreground"
                >
                  {{
                    selectedNode.kind === "subflow"
                      ? t("No bindings yet. The child flow will receive the input template exactly as written.")
                      : t("No bindings yet. Nodes can still run with their template alone.")
                  }}
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
                          {{
                            selectedNode.kind === "subflow"
                              ? t("Destination writes into the child flow input template. Source chooses where the value comes from.")
                              : t("Destination writes into the template. Source chooses where the value comes from.")
                          }}
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
                          <option value="flow_var">{{ t("Flow Local Var") }}</option>
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

                    <div v-else-if="binding.sourceKind === 'flow_var'" class="mt-3 grid gap-3 md:grid-cols-2">
                      <div>
                        <label :for="composeBindingInputId(index, 'flow-var-name')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                          {{ t("Local Var Name") }}
                        </label>
                        <input
                          :id="composeBindingInputId(index, 'flow-var-name')"
                          v-model="binding.name"
                          class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                          placeholder="session_token"
                          @blur="emit('commit-history')"
                        />
                      </div>

                      <div>
                        <label :for="composeBindingInputId(index, 'flow-var-path')" class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                          {{ t("Value Path") }}
                        </label>
                        <input
                          :id="composeBindingInputId(index, 'flow-var-path')"
                          v-model="binding.path"
                          class="mt-1 h-9 w-full rounded-md border border-input bg-background px-3 text-xs"
                          placeholder="/payload/id"
                          @blur="emit('commit-history')"
                        />
                      </div>
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

            <div v-else class="space-y-4">
              <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
                <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ nodeKindLabel }}
                </p>
                <p class="mt-2 text-sm text-muted-foreground">
                  {{ jsonOnlyKindSummary }}
                </p>
              </div>

              <div class="rounded-xl border border-sky-500/30 bg-sky-500/10 p-4">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div class="min-w-0">
                    <p class="text-xs font-semibold uppercase tracking-[0.2em] text-sky-700">
                      {{ t("Advanced JSON only") }}
                    </p>
                    <p class="mt-2 text-sm text-foreground">
                      {{ t("This node kind is supported for safe JSON authoring. The editor preserves the spec and layout, but ordinary form controls are not available yet.") }}
                    </p>
                  </div>
                  <Button
                    v-if="selectedNode.specEditorMode !== 'json'"
                    size="sm"
                    variant="outline"
                    @click="emit('toggle-spec-mode', 'json')"
                  >
                    {{ t("Open Advanced JSON") }}
                  </Button>
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

          <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Result Detail") }}
                </p>
                <p class="mt-2 text-[11px] text-muted-foreground">
                  {{ t("Query the selected node result for a specific run and optional path.") }}
                </p>
              </div>
              <Button size="sm" :disabled="nodeDetail.loading" @click="emit('load-node-detail')">
                {{ nodeDetail.loading ? t("Refreshing...") : t("Load Detail") }}
              </Button>
            </div>

            <div class="mt-4 grid gap-4 md:grid-cols-2">
              <div>
                <label :for="inspectorFieldId('detail-run-id')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Run ID (Optional)") }}
                </label>
                <input
                  :id="inspectorFieldId('detail-run-id')"
                  :value="nodeDetail.requestedRunId"
                  class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                  @input="updateNodeDetailRunId"
                  @keydown.enter.prevent="emit('load-node-detail')"
                />
                <p class="mt-1 text-[11px] text-muted-foreground">
                  {{ t("Leave run ID blank to query the latest run.") }}
                </p>
              </div>

              <div>
                <label :for="inspectorFieldId('detail-path')" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Result Path (Optional)") }}
                </label>
                <input
                  :id="inspectorFieldId('detail-path')"
                  :value="nodeDetail.requestedPath"
                  class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                  placeholder="/payload/id"
                  @input="updateNodeDetailPath"
                  @keydown.enter.prevent="emit('load-node-detail')"
                />
                <p class="mt-1 text-[11px] text-muted-foreground">
                  {{ t("Result path must be a valid JSON Pointer.") }}
                </p>
              </div>
            </div>

            <div
              v-if="nodeDetail.error"
              class="mt-4 rounded-lg border border-rose-200 bg-rose-50/80 p-3 text-xs text-rose-700"
              role="status"
              aria-live="polite"
            >
              <p class="font-semibold uppercase tracking-[0.18em]">{{ t("Failed to load node detail.") }}</p>
              <p class="mt-2 break-words">{{ nodeDetail.error }}</p>
            </div>

            <div v-if="hasLoadedNodeDetail" class="mt-4 space-y-3">
              <div class="grid gap-3 md:grid-cols-2">
                <div class="rounded-lg border border-border/70 bg-background/90 p-3">
                  <p class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                    {{ t("Resolved Run ID") }}
                  </p>
                  <p class="mt-2 break-all text-sm font-medium">
                    {{ nodeDetail.runId || t("Unknown") }}
                  </p>
                </div>

                <div class="rounded-lg border border-border/70 bg-background/90 p-3">
                  <p class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                    {{ t("Resolved Path") }}
                  </p>
                  <p class="mt-2 break-all text-sm font-medium">
                    {{ nodeDetailResolvedPathLabel }}
                  </p>
                </div>

                <div class="rounded-lg border border-border/70 bg-background/90 p-3">
                  <p class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                    {{ t("Node ID") }}
                  </p>
                  <p class="mt-2 break-all text-sm font-medium">
                    {{ nodeDetail.node?.id || selectedNode.id }}
                  </p>
                </div>

                <div class="rounded-lg border border-border/70 bg-background/90 p-3">
                  <p class="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                    {{ t("Node Status") }}
                  </p>
                  <p class="mt-2 text-sm font-medium">
                    {{ nodeDetailStatusLabel }}
                  </p>
                  <p class="mt-1 text-[11px] text-muted-foreground">
                    {{ t("Code {code}", { code: nodeDetail.node?.code ?? 0 }) }}
                  </p>
                  <p v-if="nodeDetail.node?.msg" class="mt-1 break-words text-[11px] text-muted-foreground">
                    {{ nodeDetail.node.msg }}
                  </p>
                </div>
              </div>

              <div>
                <div v-if="structuredDetailFields.length" class="mb-4">
                  <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                    {{ t("Structured Result") }}
                  </p>
                  <div class="mt-2 space-y-3">
                    <div
                      v-for="field in structuredDetailFields"
                      :key="field.key"
                      class="rounded-lg border border-border/70 bg-background/90 p-3"
                    >
                      <div class="flex flex-wrap items-center gap-2">
                        <p class="text-sm font-semibold">{{ field.label }}</p>
                        <span class="rounded-full border border-border/70 px-2 py-0.5 font-mono text-[10px] text-muted-foreground">
                          {{ field.pointer }}
                        </span>
                      </div>
                      <p v-if="field.description" class="mt-1 text-[11px] text-muted-foreground">
                        {{ field.description }}
                      </p>
                      <pre
                        v-if="field.multiline"
                        class="mt-2 overflow-x-auto rounded-lg border border-border/70 bg-muted/20 p-3 text-xs leading-5"
                      >{{ field.missing ? t("Not set") : field.valueText }}</pre>
                      <p v-else class="mt-2 break-all text-sm font-medium">
                        {{ field.missing ? t("Not set") : field.valueText }}
                      </p>
                    </div>
                  </div>
                </div>

                <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {{ t("Result") }}
                </p>
                <pre class="mt-2 overflow-x-auto rounded-lg border border-border/70 bg-background/90 p-3 text-xs leading-5">{{ nodeDetail.resultText }}</pre>
              </div>
            </div>

            <div
              v-else-if="!nodeDetail.loading && !nodeDetail.error"
              class="mt-4 rounded-lg border border-dashed border-border/60 px-4 py-5 text-center text-xs text-muted-foreground"
            >
              {{ t("No detail loaded yet.") }}
            </div>

            <div v-if="selectedNode.kind === 'call'" class="mt-4">
              <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Output schema") }}
              </p>
              <pre class="mt-2 overflow-x-auto rounded-lg border border-border/70 bg-background/90 p-3 text-xs leading-5">{{ selectedNodeOutputSchemaLabel }}</pre>
            </div>
          </div>
        </div>
      </div>
    </div>
  </aside>
</template>
