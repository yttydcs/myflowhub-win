import { appendJsonPointer } from "./flow_json_pointer"
import {
  getLocalMethodVisualSchema,
  type MethodFieldControl,
  type MethodFieldOption,
  type MethodFieldSchema,
  type MethodVisualSchema
} from "./flow_method_schemas"

export type CapabilityRouteSchemaSource = {
  method: string
  version?: string
  inputSchema?: unknown
}

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value)

const cloneSchema = (schema: MethodVisualSchema): MethodVisualSchema => ({
  ...schema,
  fields: schema.fields.map((field) => ({
    ...field,
    options: field.options?.map((option) => ({ ...option }))
  }))
})

const humanizeKey = (key: string) =>
  String(key ?? "")
    .replaceAll(/[_-]+/g, " ")
    .replaceAll(/\s+/g, " ")
    .trim()
    .replace(/\b\w/g, (ch) => ch.toUpperCase())

const parseSchemaPayload = (raw: unknown): Record<string, unknown> | null => {
  if (typeof raw === "string") {
    const trimmed = raw.trim()
    if (!trimmed) {
      return null
    }
    try {
      const parsed = JSON.parse(trimmed)
      return isPlainObject(parsed) ? parsed : null
    } catch {
      return null
    }
  }
  return isPlainObject(raw) ? raw : null
}

const hasUnsupportedSchemaFeature = (schema: Record<string, unknown>) =>
  "oneOf" in schema ||
  "anyOf" in schema ||
  "allOf" in schema ||
  "$ref" in schema ||
  schema.type === "array" ||
  Array.isArray(schema.type)

const getEnumOptions = (schema: Record<string, unknown>): MethodFieldOption[] | null => {
  const values = schema.enum
  if (!Array.isArray(values) || values.length === 0) {
    return null
  }
  const out: MethodFieldOption[] = []
  for (const item of values) {
    if (typeof item !== "string" && typeof item !== "number" && typeof item !== "boolean") {
      return null
    }
    out.push({ label: String(item), value: item })
  }
  return out
}

const getUiControlOverride = (schema: Record<string, unknown>): MethodFieldControl | null => {
  const raw = typeof schema["x-ui-control"] === "string" ? schema["x-ui-control"].trim().toLowerCase() : ""
  if (!raw) {
    return null
  }
  if (raw === "textarea" && schema.type === "string") {
    return "textarea"
  }
  return null
}

const inferFieldControl = (schema: Record<string, unknown>): { control: MethodFieldControl; options?: MethodFieldOption[] } | null => {
  const enumOptions = getEnumOptions(schema)
  if (enumOptions) {
    return { control: "select", options: enumOptions }
  }

  const uiControl = getUiControlOverride(schema)
  if (uiControl) {
    return { control: uiControl }
  }

  const type = typeof schema.type === "string" ? schema.type : ""
  switch (type) {
    case "string":
      return { control: "text" }
    case "integer":
    case "number":
      return { control: "number" }
    case "boolean":
      return { control: "switch" }
    case "object":
      return { control: "json" }
    default:
      return null
  }
}

const collectFieldsFromSchema = (
  schema: Record<string, unknown>,
  method: string,
  basePointer: string,
  out: MethodFieldSchema[]
): boolean => {
  if (hasUnsupportedSchemaFeature(schema)) {
    return false
  }
  const schemaType = typeof schema.type === "string" ? schema.type : ""
  const properties = isPlainObject(schema.properties) ? schema.properties : null

  if (schemaType === "object" || properties) {
    if (!properties || Object.keys(properties).length === 0) {
      if (!basePointer) {
        return false
      }
      const control = inferFieldControl({ ...schema, type: "object" })
      if (!control) {
        return false
      }
      out.push({
        key: `${method}:${basePointer}`,
        label: typeof schema.title === "string" && schema.title.trim() ? schema.title.trim() : basePointer.split("/").at(-1) ?? method,
        pointer: basePointer,
        control: control.control,
        description: typeof schema.description === "string" ? schema.description.trim() : undefined,
        required: false,
        bindable: true,
        defaultValue: schema.default
      })
      return true
    }

    const requiredSet = new Set(
      Array.isArray(schema.required) ? schema.required.filter((item) => typeof item === "string").map((item) => String(item)) : []
    )

    for (const [rawKey, childValue] of Object.entries(properties)) {
      if (!isPlainObject(childValue) || hasUnsupportedSchemaFeature(childValue)) {
        return false
      }
      const pointer = appendJsonPointer(basePointer, rawKey)
      const childProperties = isPlainObject(childValue.properties) ? childValue.properties : null
      const childType = typeof childValue.type === "string" ? childValue.type : ""
      const description = typeof childValue.description === "string" ? childValue.description.trim() : undefined
      const label = typeof childValue.title === "string" && childValue.title.trim() ? childValue.title.trim() : humanizeKey(rawKey)

      if ((childType === "object" || childProperties) && childProperties && Object.keys(childProperties).length > 0) {
        if (!collectFieldsFromSchema(childValue, method, pointer, out)) {
          return false
        }
        continue
      }

      const control = inferFieldControl(childValue)
      if (!control) {
        return false
      }
      out.push({
        key: `${method}:${pointer}`,
        label,
        pointer,
        control: control.control,
        description,
        required: requiredSet.has(rawKey),
        bindable: true,
        defaultValue: childValue.default,
        options: control.options
      })
    }
    return true
  }

  return false
}

const buildCapabilitySchema = (method: string, inputSchema: unknown): MethodVisualSchema | null => {
  const parsed = parseSchemaPayload(inputSchema)
  if (!parsed || hasUnsupportedSchemaFeature(parsed)) {
    return null
  }
  if (parsed.type !== "object" && !isPlainObject(parsed.properties)) {
    return null
  }
  const fields: MethodFieldSchema[] = []
  if (!collectFieldsFromSchema(parsed, method, "", fields) || !fields.length) {
    return null
  }
  return {
    method,
    title: typeof parsed.title === "string" && parsed.title.trim() ? parsed.title.trim() : method,
    supportsVisualForm: true,
    source: "capability",
    fields
  }
}

export const resolveMethodVisualSchema = (
  method: string,
  capability?: CapabilityRouteSchemaSource | null
): MethodVisualSchema | null => {
  const normalizedMethod = String(method ?? "").trim()
  if (!normalizedMethod) {
    return null
  }
  const local = getLocalMethodVisualSchema(normalizedMethod)
  if (local) {
    return cloneSchema(local)
  }
  if (!capability || String(capability.method ?? "").trim() !== normalizedMethod) {
    return null
  }
  const resolved = buildCapabilitySchema(normalizedMethod, capability.inputSchema)
  return resolved ? cloneSchema(resolved) : null
}
