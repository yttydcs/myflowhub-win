// @vitest-environment jsdom

import { defineComponent } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it } from "vitest"
import { setLocale } from "@/i18n"
import FlowBodyNodeInspector from "./FlowBodyNodeInspector.vue"
import type { FlowNodeDraft, NodeVisualFormModel } from "@/stores/flow"

const CardHeaderStub = defineComponent({
  props: {
    title: { type: String, default: "" },
    description: { type: String, default: "" },
    titleId: { type: String, default: "" }
  },
  template: `
    <div>
      <h2 :id="titleId">{{ title }}</h2>
      <p v-if="description">{{ description }}</p>
      <slot name="actions" />
    </div>
  `
})

const ButtonStub = defineComponent({
  emits: ["click"],
  template: `<button type="button" @click="$emit('click', $event)"><slot /></button>`
})

const createBaseNode = (): FlowNodeDraft => ({
  id: "inner-call",
  kind: "call",
  allowFail: false,
  retry: 1,
  timeoutMs: 3000,
  method: "demo::inner",
  target: 0,
  argsTemplate: "{\n  \"name\": \"Alice\"\n}",
  composeTemplate: "{}",
  setVarName: "",
  inputs: [],
  transformExprMode: "literal",
  transformLiteralJson: "null",
  transformSource: {
    sourceKind: "trigger",
    nodeId: "",
    path: "",
    field: "",
    name: ""
  },
  transformSourceRequired: true,
  transformOp: "add",
  transformArgsJson: "[]",
  transformObjectJson: "{}",
  transformArrayJson: "[]",
  branchCases: [],
  branchDefaultCase: "",
  foreachSource: {
    sourceKind: "trigger",
    nodeId: "",
    path: "",
    field: "",
    name: ""
  },
  foreachRequired: true,
  foreachBodyJson: "{\n  \"nodes\": [],\n  \"edges\": []\n}",
  foreachResultNodeId: "",
  subflowId: "",
  subflowInputTemplate: "{}",
  subflowResultNodeId: "",
  specEditorMode: "form",
  specJson: "{\n  \"method\": \"demo::inner\",\n  \"args_template\": {\"name\": \"Alice\"}\n}",
  x: 0,
  y: 0
})

const createSupportedVisualForm = (): NodeVisualFormModel => ({
  schema: {
    method: "demo::inner",
    title: "Inner Demo",
    supportsVisualForm: true,
    source: "capability",
    fields: [
      {
        key: "name",
        label: "Name",
        pointer: "/name",
        control: "text",
        required: true
      }
    ]
  },
  compatibility: {
    supported: true,
    reasons: []
  },
  fields: [
    {
      schema: {
        key: "name",
        label: "Name",
        pointer: "/name",
        control: "text",
        required: true
      },
      state: {
        mode: "literal",
        literalValue: "Alice",
        binding: null
      },
      bindingSummary: ""
    }
  ]
})

type InspectorProps = {
  selectedNode: FlowNodeDraft | null
  nodeIdDraft: string
  selectedTargetLabel: string
  selectedCallVisualForm: NodeVisualFormModel | null
  ancestorNodeOptions: string[]
  fieldDrafts: Record<string, unknown>
}

const mountInspector = (overrides: Partial<InspectorProps> = {}) =>
  mount(FlowBodyNodeInspector, {
    props: {
      selectedNode: createBaseNode(),
      nodeIdDraft: "inner-call",
      selectedTargetLabel: "Current executor node 100",
      selectedCallVisualForm: createSupportedVisualForm(),
      ancestorNodeOptions: ["prep1"],
      fieldDrafts: { "/name": "Alice" },
      ...overrides
    },
    global: {
      stubs: {
        CardHeader: CardHeaderStub,
        Button: ButtonStub
      }
    }
  })

describe("FlowBodyNodeInspector", () => {
  beforeEach(() => {
    setLocale("en")
  })

  it("renders call authoring controls and emits method/binding/literal actions", async () => {
    const wrapper = mountInspector()

    expect(wrapper.text()).toContain("Body node authoring")
    expect(wrapper.text()).toContain("Call Method")
    expect(wrapper.text()).toContain("Method Fields")

    const selectMethodButton = wrapper
      .findAll("button")
      .find((button) => button.text().includes("Select Method"))
    expect(selectMethodButton).toBeTruthy()
    await selectMethodButton!.trigger("click")
    expect(wrapper.emitted("open-method")).toHaveLength(1)

    const fxButton = wrapper
      .findAll("button")
      .find((button) => button.text().includes("Add fx"))
    expect(fxButton).toBeTruthy()
    await fxButton!.trigger("click")
    expect(wrapper.emitted("open-field-binding")).toHaveLength(1)

    const fieldInput = wrapper.find('input[id*="visual-"]')
    expect(fieldInput.exists()).toBe(true)
    await fieldInput.setValue("Bob")
    await fieldInput.trigger("blur")
    expect(wrapper.emitted("commit-field-literal")).toHaveLength(1)
  })

  it("renders set_var body nodes in ordinary mode and emits binding actions", async () => {
    const wrapper = mountInspector({
      selectedNode: {
        ...createBaseNode(),
        id: "inner-set-var",
        kind: "set_var",
        method: "",
        setVarName: "session_payload",
        specEditorMode: "form"
      },
      nodeIdDraft: "inner-set-var",
      selectedCallVisualForm: null
    })

    expect(wrapper.text()).toContain("Set Var Node")
    expect(wrapper.text()).toContain("Flow Local Var Name")
    expect(wrapper.text()).toContain("Input Bindings")

    const addBindingButton = wrapper
      .findAll("button")
      .find((button) => button.text().includes("Add Binding"))
    expect(addBindingButton).toBeTruthy()
    await addBindingButton!.trigger("click")
    expect(wrapper.emitted("add-binding")).toHaveLength(1)
  })
})
