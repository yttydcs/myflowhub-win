import { collectLeafPointers, deleteValueAtPointer, readValueAtPointer, setValueAtPointer } from "./flow_json_pointer"
import type { MethodFieldSchema, MethodVisualSchema } from "./flow_method_schemas"

export type FlowInputBindingLike = {
  to: string
  sourceKind: string
  nodeId: string
  path: string
  field: string
  required: boolean
}

export type VisualBindingSource =
  | { kind: "node_result"; nodeId: string; path: string; required: boolean }
  | { kind: "trigger"; path: string; required: boolean }
  | { kind: "flow_meta"; field: "flow_id"; required: boolean }
  | { kind: "run_meta"; field: "run_id"; required: boolean }

export type FieldVisualState = {
  mode: "literal" | "binding"
  literalValue: unknown
  binding: VisualBindingSource | null
}

export type VisualCompatibility = {
  supported: boolean
  reasons: string[]
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
  !binding.required

const normalizeBindings = (inputs: FlowInputBindingLike[]) =>
  inputs.filter((binding) => !isBindingBlank(binding)).map((binding) => ({
    to: String(binding.to ?? "").trim(),
    sourceKind: String(binding.sourceKind ?? "").trim(),
    nodeId: String(binding.nodeId ?? "").trim(),
    path: String(binding.path ?? "").trim(),
    field: String(binding.field ?? "").trim(),
    required: Boolean(binding.required)
  }))

const parseArgsTemplateObject = (raw: string): { ok: true; doc: Record<string, unknown> } | { ok: false; reason: string } => {
  const trimmed = String(raw ?? "").trim() || "{}"
  try {
    const parsed = JSON.parse(trimmed)
    if (!isPlainObject(parsed)) {
      return { ok: false, reason: "Args template must be a JSON object." }
    }
    return { ok: true, doc: parsed }
  } catch {
    return { ok: false, reason: "Args template must be valid JSON." }
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
      return source.path ? `${source.nodeId} -> ${source.path}` : `${source.nodeId} -> /`
    case "trigger":
      return source.path ? `Trigger -> ${source.path}` : "Trigger"
    case "flow_meta":
      return `Flow Meta -> ${source.field}`
    case "run_meta":
      return `Run Meta -> ${source.field}`
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
  const reasons: string[] = []
  if (input.kind !== "call") {
    reasons.push("Visual form only supports call nodes.")
    return { supported: false, reasons }
  }
  if (!String(input.method ?? "").trim()) {
    reasons.push("No method selected.")
    return { supported: false, reasons }
  }
  if (!input.schema) {
    reasons.push("The current method does not provide a supported visual form schema.")
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
      reasons.push(`Binding target ${binding.to} is not defined by the visual form schema.`)
      continue
    }
    if (seenBindings.has(binding.to)) {
      reasons.push(`Visual form only supports one binding per field (${binding.to}).`)
      continue
    }
    seenBindings.add(binding.to)
  }

  for (const pointer of collectLeafPointers(parsed.doc)) {
    if (!schemaPointers.has(pointer)) {
      reasons.push(`Args template contains a field that is not covered by the visual form schema (${pointer}).`)
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
        required: source.required
      })
      break
  }
  return next
}
