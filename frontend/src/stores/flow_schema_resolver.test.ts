import { describe, expect, it } from "vitest"
import { resolveMethodVisualSchema } from "./flow_schema_resolver"

describe("flow_schema_resolver", () => {
  it("prefers local override schemas and returns cloned data", () => {
    const first = resolveMethodVisualSchema("varstore::set", {
      method: "varstore::set",
      inputSchema: {
        type: "object",
        properties: {
          ignored: { type: "string" }
        }
      }
    })

    expect(first).not.toBeNull()
    expect(first?.source).toBe("local_override")
    expect(first?.fields.map((field) => field.pointer)).toEqual([
      "/owner",
      "/name",
      "/value",
      "/type",
      "/visibility"
    ])

    first!.fields[0].label = "Mutated"
    first!.fields[4].options![0].label = "changed"

    const second = resolveMethodVisualSchema("varstore::set", {
      method: "varstore::set",
      inputSchema: {
        type: "object",
        properties: {
          ignored: { type: "string" }
        }
      }
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
            items: { type: "array" }
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
  })
})
