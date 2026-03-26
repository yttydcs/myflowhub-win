import { t } from "@/i18n"
import { collectLeafPointers, deleteValueAtPointer, readValueAtPointer, setValueAtPointer } from "./flow_json_pointer"
import type { MethodFieldSchema, MethodVisualSchema } from "./flow_method_schemas"

export type FlowInputBindingLike = {
  to: string
  sourceKind: string
  nodeId: string
  path: string
  field: string
  name: string
  required: boolean
}

export type VisualBindingSource =
  | { kind: "node_result"; nodeId: string; path: string; required: boolean }
  | { kind: "trigger"; path: string; required: boolean }
  | { kind: "flow_meta"; field: "flow_id"; required: boolean }
  | { kind: "run_meta"; field: "run_id"; required: boolean }
  | { kind: "flow_var"; name: string; path: string; required: boolean }

export type FieldVisualState = {
  mode: "literal" | "binding"
  literalValue: unknown
  binding: VisualBindingSource | null
}

export type VisualCompatibilityReasonCode =
  | "not_call_node"
  | "missing_method"
  | "missing_schema"
  | "args_template_invalid_json"
  | "args_template_not_object"
  | "binding_target_unknown"
  | "duplicate_field_binding"
  | "extra_literal_field"

export type VisualCompatibilityReason = {
  code: VisualCompatibilityReasonCode
  pointer?: string
}

export type VisualCompatibility = {
  supported: boolean
  reasons: VisualCompatibilityReason[]
}

export type VisualFieldModel = {
  schema: MethodFieldSchema
  state: FieldVisualState
  bindingSummary: string
}

export type NodeVisualFormModel = {
  schema: MethodVisualSchema | null
  compatibility: VisualCompatibility
  fields: VisualFieldModel[]
}

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value)

const isBindingBlank = (binding: FlowInputBindingLike) =>
  !String(binding.to ?? "").trim() &&
  !String(binding.sourceKind ?? "").trim() &&
  !String(binding.nodeId ?? "").trim() &&
  !String(binding.path ?? "").trim() &&
  !String(binding.field ?? "").trim() &&
  !String(binding.name ?? "").trim() &&
  !binding.required

const normalizeBindings = (inputs: FlowInputBindingLike[]) =>
  inputs.filter((binding) => !isBindingBlank(binding)).map((binding) => ({
    to: String(binding.to ?? "").trim(),
    sourceKind: String(binding.sourceKind ?? "").trim(),
    nodeId: String(binding.nodeId ?? "").trim(),
    path: String(binding.path ?? "").trim(),
    field: String(binding.field ?? "").trim(),
    name: String(binding.name ?? "").trim(),
    required: Boolean(binding.required)
  }))

const parseArgsTemplateObject = (
  raw: string
): { ok: true; doc: Record<string, unknown> } | { ok: false; reason: VisualCompatibilityReason } => {
  const trimmed = String(raw ?? "").trim() || "{}"
  try {
    const parsed = JSON.parse(trimmed)
    if (!isPlainObject(parsed)) {
      return { ok: false, reason: { code: "args_template_not_object" } }
    }
    return { ok: true, doc: parsed }
  } catch {
    return { ok: false, reason: { code: "args_template_invalid_json" } }
  }
}

const toVisualBindingSource = (binding: FlowInputBindingLike): VisualBindingSource | null => {
  switch (binding.sourceKind) {
    case "node_result":
      return {
        kind: "node_result",
        nodeId: binding.nodeId,
        path: binding.path,
        required: binding.required
      }
    case "trigger":
      return {
        kind: "trigger",
        path: binding.path,
        required: binding.required
      }
    case "flow_meta":
      return {
        kind: "flow_meta",
        field: "flow_id",
        required: binding.required
      }
    case "run_meta":
      return {
        kind: "run_meta",
        field: "run_id",
        required: binding.required
      }
    case "flow_var":
      return {
        kind: "flow_var",
        name: binding.name,
        path: binding.path,
        required: binding.required
      }
    default:
      return null
  }
}

export const describeFieldBinding = (source: VisualBindingSource | null) => {
  if (!source) {
    return ""
  }
  switch (source.kind) {
    case "node_result":
      return source.path
        ? t("Node {nodeId} result at {path}", { nodeId: source.nodeId || "?", path: source.path })
        : t("Node {nodeId} full result", { nodeId: source.nodeId || "?" })
    case "trigger":
      return source.path ? t("Trigger data at {path}", { path: source.path }) : t("Trigger payload")
    case "flow_meta":
      return t("Flow metadata · {field}", { field: source.field })
    case "run_meta":
      return t("Run metadata · {field}", { field: source.field })
    case "flow_var":
      return source.path
        ? t("Flow local var {name} at {path}", { name: source.name || "?", path: source.path })
        : t("Flow local var {name}", { name: source.name || "?" })
    default:
      return ""
  }
}

export const describeVisualCompatibilityReason = (reason: VisualCompatibilityReason) => {
  switch (reason.code) {
    case "not_call_node":
      return t("Visual form only supports call nodes.")
    case "missing_method":
      return t("No method selected.")
    case "missing_schema":
      return t("The current method does not provide a supported visual form schema.")
    case "args_template_invalid_json":
      return t("Args template must be valid JSON.")
    case "args_template_not_object":
      return t("Args template must be a JSON object.")
    case "binding_target_unknown":
      return t("Binding target {pointer} is not defined by the visual form schema.", {
        pointer: reason.pointer || "/"
      })
    case "duplicate_field_binding":
      return t("Visual form only supports one binding per field ({pointer}).", {
        pointer: reason.pointer || "/"
      })
    case "extra_literal_field":
      return t("Args template contains a field that is not covered by the visual form schema ({pointer}).", {
        pointer: reason.pointer || "/"
      })
    default:
      return ""
  }
}

export const analyzeVisualCompatibility = (input: {
  kind: string
  method: string
  argsTemplate: string
  inputs: FlowInputBindingLike[]
  schema: MethodVisualSchema | null
}): VisualCompatibility => {
  const reasons: VisualCompatibilityReason[] = []
  if (input.kind !== "call") {
    reasons.push({ code: "not_call_node" })
    return { supported: false, reasons }
  }
  if (!String(input.method ?? "").trim()) {
    reasons.push({ code: "missing_method" })
    return { supported: false, reasons }
  }
  if (!input.schema) {
    reasons.push({ code: "missing_schema" })
    return { supported: false, reasons }
  }

  const parsed = parseArgsTemplateObject(input.argsTemplate)
  if (!parsed.ok) {
    reasons.push(parsed.reason)
    return { supported: false, reasons }
  }

  const schemaPointers = new Set(input.schema.fields.map((field) => field.pointer))
  const seenBindings = new Set<string>()
  for (const binding of normalizeBindings(input.inputs)) {
    if (!schemaPointers.has(binding.to)) {
      reasons.push({ code: "binding_target_unknown", pointer: binding.to })
      continue
    }
    if (seenBindings.has(binding.to)) {
      reasons.push({ code: "duplicate_field_binding", pointer: binding.to })
      continue
    }
    seenBindings.add(binding.to)
  }

  for (const pointer of collectLeafPointers(parsed.doc)) {
    if (!schemaPointers.has(pointer)) {
      reasons.push({ code: "extra_literal_field", pointer })
    }
  }

  return { supported: reasons.length === 0, reasons }
}

export const getFieldVisualState = (
  argsDoc: Record<string, unknown>,
  inputs: FlowInputBindingLike[],
  field: MethodFieldSchema
): FieldVisualState => {
  const bindings = normalizeBindings(inputs).filter((binding) => binding.to === field.pointer)
  const literal = readValueAtPointer(argsDoc, field.pointer)
  const literalValue = literal.found ? literal.value : field.defaultValue
  if (!bindings.length) {
    return {
      mode: "literal",
      literalValue,
      binding: null
    }
  }
  return {
    mode: "binding",
    literalValue,
    binding: toVisualBindingSource(bindings[0])
  }
}

export const buildNodeVisualFormModel = (input: {
  kind: string
  method: string
  argsTemplate: string
  inputs: FlowInputBindingLike[]
  schema: MethodVisualSchema | null
}): NodeVisualFormModel => {
  const compatibility = analyzeVisualCompatibility(input)
  if (!compatibility.supported || !input.schema) {
    return {
      schema: input.schema,
      compatibility,
      fields: []
    }
  }
  const parsed = parseArgsTemplateObject(input.argsTemplate)
  if (!parsed.ok) {
    return {
      schema: input.schema,
      compatibility: { supported: false, reasons: [parsed.reason] },
      fields: []
    }
  }
  return {
    schema: input.schema,
    compatibility,
    fields: input.schema.fields.map((field) => {
      const state = getFieldVisualState(parsed.doc, input.inputs, field)
      return {
        schema: field,
        state,
        bindingSummary: describeFieldBinding(state.binding)
      }
    })
  }
}

export const setLiteralFieldValue = (argsDoc: Record<string, unknown>, pointer: string, value: unknown) => {
  if (value === undefined) {
    const next = deleteValueAtPointer(argsDoc, pointer)
    return isPlainObject(next) ? next : {}
  }
  const next = setValueAtPointer(argsDoc, pointer, value)
  return isPlainObject(next) ? next : {}
}

export const clearBindingForPointer = (inputs: FlowInputBindingLike[], pointer: string): FlowInputBindingLike[] =>
  inputs.filter((binding) => String(binding.to ?? "").trim() !== pointer)

export const setBindingForPointer = (
  inputs: FlowInputBindingLike[],
  pointer: string,
  source: VisualBindingSource
): FlowInputBindingLike[] => {
  const next = clearBindingForPointer(inputs, pointer)
  switch (source.kind) {
    case "node_result":
      next.push({
        to: pointer,
        sourceKind: "node_result",
        nodeId: source.nodeId,
        path: source.path,
        field: "",
        name: "",
        required: source.required
      })
      break
    case "trigger":
      next.push({
        to: pointer,
        sourceKind: "trigger",
        nodeId: "",
        path: source.path,
        field: "",
        name: "",
        required: source.required
      })
      break
    case "flow_meta":
      next.push({
        to: pointer,
        sourceKind: "flow_meta",
        nodeId: "",
        path: "",
        field: source.field,
        name: "",
        required: source.required
      })
      break
    case "run_meta":
      next.push({
        to: pointer,
        sourceKind: "run_meta",
        nodeId: "",
        path: "",
        field: source.field,
        name: "",
        required: source.required
      })
      break
    case "flow_var":
      next.push({
        to: pointer,
        sourceKind: "flow_var",
        nodeId: "",
        path: source.path,
        field: "",
        name: source.name,
        required: source.required
      })
      break
  }
  return next
}
