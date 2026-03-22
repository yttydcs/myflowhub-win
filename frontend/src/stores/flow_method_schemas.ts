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

const LOCAL_METHOD_SCHEMAS: Record<string, MethodVisualSchema> = {
  "varstore::get": {
    method: "varstore::get",
    title: "VarStore Get",
    supportsVisualForm: true,
    source: "local_override",
    fields: [
      {
        key: "owner",
        label: "Owner",
        pointer: "/owner",
        control: "number",
        description: "Owner node ID of the variable.",
        required: true,
        bindable: true
      },
      {
        key: "name",
        label: "Name",
        pointer: "/name",
        control: "text",
        description: "Variable name.",
        required: true,
        bindable: true
      }
    ]
  },
  "varstore::set": {
    method: "varstore::set",
    title: "VarStore Set",
    supportsVisualForm: true,
    source: "local_override",
    fields: [
      {
        key: "owner",
        label: "Owner",
        pointer: "/owner",
        control: "number",
        description: "Owner node ID of the variable.",
        required: true,
        bindable: true
      },
      {
        key: "name",
        label: "Name",
        pointer: "/name",
        control: "text",
        description: "Variable name.",
        required: true,
        bindable: true
      },
      {
        key: "value",
        label: "Value",
        pointer: "/value",
        control: "textarea",
        description: "Variable value payload.",
        required: true,
        bindable: true
      },
      {
        key: "type",
        label: "Type",
        pointer: "/type",
        control: "text",
        description: "Optional variable type label.",
        bindable: true
      },
      {
        key: "visibility",
        label: "Visibility",
        pointer: "/visibility",
        control: "select",
        description: "Variable visibility.",
        defaultValue: "private",
        options: [
          { label: "private", value: "private" },
          { label: "public", value: "public" }
        ]
      }
    ]
  },
  "varstore::revoke": {
    method: "varstore::revoke",
    title: "VarStore Revoke",
    supportsVisualForm: true,
    source: "local_override",
    fields: [
      {
        key: "owner",
        label: "Owner",
        pointer: "/owner",
        control: "number",
        description: "Owner node ID of the variable.",
        required: true,
        bindable: true
      },
      {
        key: "name",
        label: "Name",
        pointer: "/name",
        control: "text",
        description: "Variable name.",
        required: true,
        bindable: true
      }
    ]
  }
}

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
