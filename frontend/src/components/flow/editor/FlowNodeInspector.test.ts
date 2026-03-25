// @vitest-environment jsdom

import { defineComponent } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it } from "vitest"
import { setLocale } from "@/i18n"
import FlowNodeInspector from "./FlowNodeInspector.vue"
import type { FlowNodeDraft, NodeVisualFormModel } from "@/stores/flow"

const CardHeaderStub = defineComponent({
  props: {
    title: { type: String, default: "" },
    description: { type: String, default: "" },
    titleId: { type: String, default: "" },
    descriptionId: { type: String, default: "" }
  },
  template: `
    <div>
      <h2 :id="titleId">{{ title }}</h2>
      <p v-if="description" :id="descriptionId">{{ description }}</p>
      <slot name="actions" />
    </div>
  `
})

const ButtonStub = defineComponent({
  emits: ["click"],
  template: `<button type="button" @click="$emit('click', $event)"><slot /></button>`
})

const createCallNode = (): FlowNodeDraft => ({
  id: "call-1",
  kind: "call",
  allowFail: false,
  retry: 1,
  timeoutMs: 3000,
  method: "demo::missing-schema",
  target: 0,
  argsTemplate: "{\n  \"extra\": true\n}",
  composeTemplate: "{}",
  inputs: [],
  specEditorMode: "form",
  specJson: "{\n  \"method\": \"demo::missing-schema\"\n}",
  x: 0,
  y: 0
})

const createUnsupportedVisualForm = (): NodeVisualFormModel => ({
  schema: null,
  compatibility: {
    supported: false,
    reasons: [
      { code: "missing_schema" },
      { code: "extra_literal_field", pointer: "/extra" }
    ]
  },
  fields: []
})

describe("FlowNodeInspector", () => {
  beforeEach(() => {
    setLocale("en")
  })

  it("hides missing schema copy while keeping other compatibility reasons", () => {
    const wrapper = mount(FlowNodeInspector, {
      props: {
        selectedNode: createCallNode(),
        nodeIdDraft: "call-1",
        selectedNodeValidation: [],
        selectedTargetLabel: "Current executor node 100",
        selectedCallVisualForm: createUnsupportedVisualForm(),
        ancestorNodeOptions: [],
        fieldDrafts: {}
      },
      global: {
        stubs: {
          CardHeader: CardHeaderStub,
          Button: ButtonStub
        }
      }
    })

    const text = wrapper.text()

    expect(text).toContain("Visual form unavailable")
    expect(text).toContain("Args template contains a field that is not covered by the visual form schema (/extra).")
    expect(text).not.toContain("The current method does not provide a supported visual form schema.")
    expect(text).not.toContain("Query capabilities and choose a method that exposes a supported input schema.")
  })
})
