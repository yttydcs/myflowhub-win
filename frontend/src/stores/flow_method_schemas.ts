export type MethodFieldControl =
  | "text"
  | "textarea"
  | "number"
  | "select"
  | "switch"
  | "json"

export type MethodFieldOption = {
  label: string
  value: string | number | boolean
}

export type MethodFieldSchema = {
  key: string
  label: string
  pointer: string
  control: MethodFieldControl
  description?: string
  required?: boolean
  bindable?: boolean
  defaultValue?: unknown
  options?: MethodFieldOption[]
}

export type MethodVisualSchema = {
  method: string
  title: string
  supportsVisualForm: boolean
  source: "local_override" | "capability"
  fields: MethodFieldSchema[]
}

const LOCAL_METHOD_SCHEMAS: Record<string, MethodVisualSchema> = {}

const cloneSchema = (schema: MethodVisualSchema): MethodVisualSchema => ({
  ...schema,
  fields: schema.fields.map((field) => ({
    ...field,
    options: field.options?.map((option) => ({ ...option }))
  }))
})

export const getLocalMethodVisualSchema = (method: string): MethodVisualSchema | null => {
  const normalized = String(method ?? "").trim()
  if (!normalized) {
    return null
  }
  const schema = LOCAL_METHOD_SCHEMAS[normalized]
  return schema ? cloneSchema(schema) : null
}
