// 本文件覆盖 `flow_visual_form` store 的行为。

import { beforeEach, describe, expect, it } from "vitest"
import { setLocale } from "@/i18n"
import type { MethodVisualSchema } from "./flow_method_schemas"
import { analyzeVisualCompatibility, describeFieldBinding } from "./flow_visual_form"

const schema: MethodVisualSchema = {
  method: "demo::call",
  title: "Demo Call",
  supportsVisualForm: true,
  source: "capability",
  fields: [
    {
      key: "title",
      label: "Title",
      pointer: "/title",
      control: "text",
      bindable: true
    },
    {
      key: "enabled",
      label: "Enabled",
      pointer: "/enabled",
      control: "switch",
      bindable: true
    }
  ]
}

describe("flow_visual_form", () => {
  beforeEach(() => {
    setLocale("en")
  })

  it("formats binding summaries with user-facing labels", () => {
    expect(
      describeFieldBinding({
        kind: "node_result",
        nodeId: "12",
        path: "/payload/id",
        required: false
      })
    ).toBe("Node 12 result at /payload/id")

    expect(
      describeFieldBinding({
        kind: "trigger",
        path: "",
        required: true
      })
    ).toBe("Trigger payload")

    expect(
      describeFieldBinding({
        kind: "flow_var",
        name: "session_token",
        path: "/payload/id",
        required: false
      })
    ).toBe("Flow local var session_token at /payload/id")
  })

  it("returns structured compatibility reasons for unsupported bindings and extra fields", () => {
    const compatibility = analyzeVisualCompatibility({
      kind: "call",
      method: "demo::call",
      argsTemplate: JSON.stringify({
        title: "hello",
        extra: "out-of-schema"
      }),
      inputs: [
        {
          to: "/missing",
          sourceKind: "trigger",
          nodeId: "",
          path: "/payload/value",
          field: "",
          name: "",
          required: false
        },
        {
          to: "/title",
          sourceKind: "trigger",
          nodeId: "",
          path: "",
          field: "",
          name: "",
          required: false
        },
        {
          to: "/title",
          sourceKind: "run_meta",
          nodeId: "",
          path: "",
          field: "run_id",
          name: "",
          required: false
        }
      ],
      schema
    })

    expect(compatibility.supported).toBe(false)
    expect(compatibility.reasons).toEqual([
      { code: "binding_target_unknown", pointer: "/missing" },
      { code: "duplicate_field_binding", pointer: "/title" },
      { code: "extra_literal_field", pointer: "/extra" }
    ])
  })

  it("rejects invalid args templates before building the visual form", () => {
    const compatibility = analyzeVisualCompatibility({
      kind: "call",
      method: "demo::call",
      argsTemplate: "[]",
      inputs: [],
      schema
    })

    expect(compatibility.supported).toBe(false)
    expect(compatibility.reasons).toEqual([{ code: "args_template_not_object" }])
  })
})
