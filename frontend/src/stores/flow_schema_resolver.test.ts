import { describe, expect, it } from "vitest"
import { resolveMethodVisualSchema } from "./flow_schema_resolver"

const createVarStoreSetInputSchema = () => ({
  title: "VarStore Set",
  type: "object",
  required: ["owner", "name", "value"],
  properties: {
    owner: {
      type: "integer",
      description: "Owner node ID of the variable."
    },
    name: {
      type: "string",
      description: "Variable name."
    },
    value: {
      type: "string",
      description: "Variable value payload.",
      "x-ui-control": "textarea"
    },
    type: {
      type: "string",
      description: "Optional variable type label."
    },
    visibility: {
      type: "string",
      enum: ["private", "public"],
      default: "private"
    }
  }
})

const createVarStoreGetInputSchema = () => ({
  title: "VarStore Get",
  type: "object",
  required: ["owner", "name"],
  properties: {
    owner: {
      type: "integer"
    },
    name: {
      type: "string"
    }
  }
})

const createVarStoreRevokeInputSchema = () => ({
  title: "VarStore Revoke",
  type: "object",
  required: ["owner", "name"],
  properties: {
    owner: {
      type: "integer"
    },
    name: {
      type: "string"
    }
  }
})

describe("flow_schema_resolver", () => {
  it("returns cloned capability schemas and honors x-ui-control hints", () => {
    const first = resolveMethodVisualSchema("varstore::set", {
      method: "varstore::set",
      inputSchema: createVarStoreSetInputSchema()
    })

    expect(first).not.toBeNull()
    expect(first?.source).toBe("capability")
    expect(first?.fields.map((field) => field.pointer)).toEqual([
      "/owner",
      "/name",
      "/value",
      "/type",
      "/visibility"
    ])
    expect(first?.fields.find((field) => field.pointer === "/value")).toMatchObject({
      control: "textarea",
      required: true
    })
    expect(first?.fields.find((field) => field.pointer === "/visibility")).toMatchObject({
      control: "select",
      defaultValue: "private",
      options: [
        { label: "private", value: "private" },
        { label: "public", value: "public" }
      ]
    })

    first!.fields[0].label = "Mutated"
    first!.fields[4].options![0].label = "changed"

    const second = resolveMethodVisualSchema("varstore::set", {
      method: "varstore::set",
      inputSchema: createVarStoreSetInputSchema()
    })

    expect(second?.fields[0].label).toBe("Owner")
    expect(second?.fields[4].options?.[0].label).toBe("private")
  })

  it("builds a capability schema from the supported json schema subset", () => {
    const schema = resolveMethodVisualSchema("demo::call", {
      method: "demo::call",
      inputSchema: JSON.stringify({
        title: "Demo Capability",
        type: "object",
        required: ["name"],
        properties: {
          name: {
            type: "string",
            description: "Display name"
          },
          enabled: {
            type: "boolean",
            default: true
          },
          mode: {
            enum: ["fast", "safe"]
          },
          config: {
            type: "object",
            properties: {
              retries: {
                type: "integer",
                default: 3
              },
              nested_flag: {
                type: "boolean"
              }
            },
            required: ["retries"]
          },
          metadata: {
            type: "object",
            properties: {}
          }
        }
      })
    })

    expect(schema).toMatchObject({
      method: "demo::call",
      title: "Demo Capability",
      supportsVisualForm: true,
      source: "capability"
    })
    expect(schema?.fields).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          pointer: "/name",
          label: "Name",
          control: "text",
          description: "Display name",
          required: true
        }),
        expect.objectContaining({
          pointer: "/enabled",
          label: "Enabled",
          control: "switch",
          defaultValue: true
        }),
        expect.objectContaining({
          pointer: "/mode",
          label: "Mode",
          control: "select",
          options: [
            { label: "fast", value: "fast" },
            { label: "safe", value: "safe" }
          ]
        }),
        expect.objectContaining({
          pointer: "/config/retries",
          label: "Retries",
          control: "number",
          required: true,
          defaultValue: 3
        }),
        expect.objectContaining({
          pointer: "/config/nested_flag",
          label: "Nested Flag",
          control: "switch"
        }),
        expect.objectContaining({
          pointer: "/metadata",
          label: "Metadata",
          control: "json"
        })
      ])
    )
  })

  it("parses the first batch of backend capability schemas for flow ordinary mode", () => {
    const varstoreGet = resolveMethodVisualSchema("varstore::get", {
      method: "varstore::get",
      inputSchema: createVarStoreGetInputSchema()
    })

    expect(varstoreGet?.fields).toEqual([
      expect.objectContaining({
        pointer: "/owner",
        control: "number",
        required: true
      }),
      expect.objectContaining({
        pointer: "/name",
        control: "text",
        required: true
      })
    ])

    const varstoreSet = resolveMethodVisualSchema("varstore::set", {
      method: "varstore::set",
      inputSchema: createVarStoreSetInputSchema()
    })

    expect(varstoreSet?.fields).toEqual([
      expect.objectContaining({
        pointer: "/owner",
        control: "number",
        required: true
      }),
      expect.objectContaining({
        pointer: "/name",
        control: "text",
        required: true
      }),
      expect.objectContaining({
        pointer: "/value",
        control: "textarea",
        required: true
      }),
      expect.objectContaining({
        pointer: "/type",
        control: "text",
        required: false
      }),
      expect.objectContaining({
        pointer: "/visibility",
        control: "select",
        required: false
      })
    ])

    const varstoreRevoke = resolveMethodVisualSchema("varstore::revoke", {
      method: "varstore::revoke",
      inputSchema: createVarStoreRevokeInputSchema()
    })

    expect(varstoreRevoke?.fields).toEqual([
      expect.objectContaining({
        pointer: "/owner",
        control: "number",
        required: true
      }),
      expect.objectContaining({
        pointer: "/name",
        control: "text",
        required: true
      })
    ])

    const topicbus = resolveMethodVisualSchema("topicbus::publish", {
      method: "topicbus::publish",
      inputSchema: {
        title: "Publish Event",
        type: "object",
        required: ["name"],
        properties: {
          topic: {
            type: "string"
          },
          name: {
            type: "string"
          },
          ts: {
            type: "integer"
          },
          payload: {
            type: "object",
            properties: {}
          }
        }
      }
    })

    expect(topicbus).toMatchObject({
      title: "Publish Event",
      source: "capability"
    })
    expect(topicbus?.fields).toEqual([
      expect.objectContaining({
        pointer: "/topic",
        control: "text",
        required: false
      }),
      expect.objectContaining({
        pointer: "/name",
        control: "text",
        required: true
      }),
      expect.objectContaining({
        pointer: "/ts",
        control: "number",
        required: false
      }),
      expect.objectContaining({
        pointer: "/payload",
        control: "json",
        required: false
      })
    ])

    const fileList = resolveMethodVisualSchema("file::list", {
      method: "file::list",
      inputSchema: {
        title: "List Directory",
        type: "object",
        properties: {
          dir: {
            type: "string"
          }
        }
      }
    })

    expect(fileList?.fields).toEqual([
      expect.objectContaining({
        pointer: "/dir",
        control: "text",
        required: false
      })
    ])

    const fileReadText = resolveMethodVisualSchema("file::read_text", {
      method: "file::read_text",
      inputSchema: {
        title: "Read Text File",
        type: "object",
        required: ["name"],
        properties: {
          dir: {
            type: "string"
          },
          name: {
            type: "string"
          },
          max_bytes: {
            type: "integer"
          }
        }
      }
    })

    expect(fileReadText?.fields).toEqual([
      expect.objectContaining({
        pointer: "/dir",
        control: "text",
        required: false
      }),
      expect.objectContaining({
        pointer: "/name",
        control: "text",
        required: true
      }),
      expect.objectContaining({
        pointer: "/max_bytes",
        control: "number",
        required: false
      })
    ])

    const fileMkdir = resolveMethodVisualSchema("file::mkdir", {
      method: "file::mkdir",
      inputSchema: {
        title: "Create Directory",
        type: "object",
        required: ["name"],
        properties: {
          dir: {
            type: "string"
          },
          name: {
            type: "string"
          }
        }
      }
    })

    expect(fileMkdir?.fields).toEqual([
      expect.objectContaining({
        pointer: "/dir",
        control: "text",
        required: false
      }),
      expect.objectContaining({
        pointer: "/name",
        control: "text",
        required: true
      })
    ])
  })

  it("accepts nullable wrappers around the supported schema subset", () => {
    const schema = resolveMethodVisualSchema("demo::nullable", {
      method: "demo::nullable",
      inputSchema: {
        title: "Nullable Demo",
        type: ["null", "object"],
        required: ["name"],
        properties: {
          name: {
            type: ["string", "null"],
            description: "Display name"
          },
          notes: {
            type: ["null", "string"],
            "x-ui-control": "textarea"
          },
          enabled: {
            type: ["boolean", "null"],
            default: true
          },
          retry_count: {
            type: ["null", "integer"],
            default: 2
          },
          metadata: {
            type: ["object", "null"],
            properties: {}
          },
          config: {
            type: ["null", "object"],
            properties: {
              threshold: {
                type: ["number", "null"]
              }
            }
          }
        }
      }
    })

    expect(schema).toMatchObject({
      method: "demo::nullable",
      title: "Nullable Demo",
      supportsVisualForm: true,
      source: "capability"
    })
    expect(schema?.fields).toEqual([
      expect.objectContaining({
        pointer: "/name",
        control: "text",
        required: true,
        description: "Display name"
      }),
      expect.objectContaining({
        pointer: "/notes",
        control: "textarea",
        required: false
      }),
      expect.objectContaining({
        pointer: "/enabled",
        control: "switch",
        defaultValue: true
      }),
      expect.objectContaining({
        pointer: "/retry_count",
        control: "number",
        defaultValue: 2
      }),
      expect.objectContaining({
        pointer: "/metadata",
        control: "json"
      }),
      expect.objectContaining({
        pointer: "/config/threshold",
        control: "number"
      })
    ])
  })

  it("returns null for unsupported or mismatched capability schemas", () => {
    expect(
      resolveMethodVisualSchema("demo::call", {
        method: "other::call",
        inputSchema: {
          type: "object",
          properties: {
            name: { type: "string" }
          }
        }
      })
    ).toBeNull()

    expect(
      resolveMethodVisualSchema("demo::call", {
        method: "demo::call",
        inputSchema: {
          type: "object",
          properties: {
            items: { type: "array" },
            notes: { type: "string", "x-ui-control": "textarea" }
          }
        }
      })
    ).toBeNull()

    expect(
      resolveMethodVisualSchema("demo::call", {
        method: "demo::call",
        inputSchema: JSON.stringify({
          type: "object",
          oneOf: []
        })
      })
    ).toBeNull()

    expect(
      resolveMethodVisualSchema("demo::call", {
        method: "demo::call",
        inputSchema: {
          type: ["object", "null", "string"],
          properties: {
            name: { type: "string" }
          }
        }
      })
    ).toBeNull()

    expect(
      resolveMethodVisualSchema("demo::call", {
        method: "demo::call",
        inputSchema: {
          type: "object",
          properties: {
            name: { type: ["string", "number"] }
          }
        }
      })
    ).toBeNull()
  })
})
