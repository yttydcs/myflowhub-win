// @vitest-environment jsdom

import { defineComponent } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it } from "vitest"
import { setLocale } from "@/i18n"
import FlowNodeInspector from "./FlowNodeInspector.vue"
import type { FlowNodeDetailState, FlowNodeDraft, NodeVisualFormModel } from "@/stores/flow"

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
  setVarName: "",
  inputs: [],
  specEditorMode: "form",
  specJson: "{\n  \"method\": \"demo::missing-schema\"\n}",
  x: 0,
  y: 0
})

const createSetVarNode = (): FlowNodeDraft => ({
  id: "set-1",
  kind: "set_var",
  allowFail: false,
  retry: 1,
  timeoutMs: 3000,
  method: "",
  target: 0,
  argsTemplate: "{}",
  composeTemplate: "{\n  \"value\": null\n}",
  setVarName: "session_token",
  inputs: [
    {
      to: "/value",
      sourceKind: "flow_var",
      nodeId: "",
      path: "/payload/id",
      field: "",
      name: "auth_token",
      required: true
    }
  ],
  specEditorMode: "form",
  specJson: "{\n  \"name\": \"session_token\",\n  \"template\": {\"value\": null}\n}",
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

const createNodeDetail = (overrides: Partial<FlowNodeDetailState> = {}): FlowNodeDetailState => ({
  loading: false,
  error: "",
  requestedNodeId: "",
  requestedRunId: "",
  requestedPath: "",
  runId: "",
  path: "",
  node: null,
  resultValue: undefined,
  resultText: "",
  ...overrides
})

type InspectorProps = {
  selectedNode: FlowNodeDraft | null
  nodeIdDraft: string
  selectedNodeValidation: string[]
  selectedTargetLabel: string
  selectedCallVisualForm: NodeVisualFormModel | null
  ancestorNodeOptions: string[]
  nodeDetail: FlowNodeDetailState
  selectedNodeOutputSchemaText: string
  fieldDrafts: Record<string, unknown>
}

const mountInspector = (props: Partial<InspectorProps> = {}) =>
  mount(FlowNodeInspector, {
    props: {
      selectedNode: createCallNode(),
      nodeIdDraft: "call-1",
      selectedNodeValidation: [],
      selectedTargetLabel: "Current executor node 100",
      selectedCallVisualForm: null,
      ancestorNodeOptions: [],
      nodeDetail: createNodeDetail(),
      selectedNodeOutputSchemaText: "",
      fieldDrafts: {},
      ...props
    },
    global: {
      stubs: {
        CardHeader: CardHeaderStub,
        Button: ButtonStub
      }
    }
  })

describe("FlowNodeInspector", () => {
  beforeEach(() => {
    setLocale("en")
  })

  it("hides missing schema copy while keeping other compatibility reasons", () => {
    const wrapper = mountInspector({
      selectedCallVisualForm: createUnsupportedVisualForm()
    })

    const text = wrapper.text()

    expect(text).toContain("Visual form unavailable")
    expect(text).toContain("Args template contains a field that is not covered by the visual form schema (/extra).")
    expect(text).not.toContain("The current method does not provide a supported visual form schema.")
    expect(text).not.toContain("Query capabilities and choose a method that exposes a supported input schema.")
  })

  it("renders set_var authoring controls with flow local var binding sources", () => {
    const wrapper = mountInspector({
      selectedNode: createSetVarNode(),
      nodeIdDraft: "set-1",
      selectedTargetLabel: "Set var nodes write a flow-local variable for the current run.",
      ancestorNodeOptions: ["call-1"]
    })

    expect(wrapper.text()).toContain("Set Var Node")
    expect(wrapper.text()).toContain("Flow Local Var Name")
    expect(wrapper.find('option[value="set_var"]').exists()).toBe(true)
    expect(wrapper.find('option[value="flow_var"]').exists()).toBe(true)
  })

  it("renders loaded result detail and output schema for call nodes", () => {
    const wrapper = mountInspector({
      nodeDetail: createNodeDetail({
        requestedNodeId: "call-1",
        requestedRunId: "run-requested",
        runId: "run-resolved",
        node: {
          id: "call-1",
          status: "succeeded",
          code: 201,
          msg: "detail ok"
        },
        resultValue: {
          payload: {
            value: "hello"
          }
        },
        resultText: JSON.stringify(
          {
            payload: {
              value: "hello"
            }
          },
          null,
          2
        )
      }),
      selectedNodeOutputSchemaText: JSON.stringify(
        {
          type: "object",
          properties: {
            payload: {
              type: "object",
              properties: {
                value: { type: "string" }
              }
            }
          }
        },
        null,
        2
      )
    })

    const text = wrapper.text()
    const preBlocks = wrapper.findAll("pre")

    expect(text).toContain("Result Detail")
    expect(text).toContain("Load Detail")
    expect(text).toContain("Resolved Run ID")
    expect(text).toContain("run-resolved")
    expect(text).toContain("Resolved Path")
    expect(text).toContain("Root result")
    expect(text).toContain("Node Status")
    expect(text).toContain("Succeeded")
    expect(text).toContain("Code 201")
    expect(text).toContain("detail ok")
    expect(text).toContain("Structured Result")
    expect(text).toContain("/payload/value")
    expect(text).toContain("Output schema")
    expect(preBlocks).toHaveLength(2)
    expect(preBlocks[0].text()).toContain('"payload"')
    expect(preBlocks[0].text()).toContain('"hello"')
    expect(preBlocks[1].text()).toContain('"type": "object"')
    expect(preBlocks[1].text()).toContain('"payload"')
  })

  it("falls back to raw detail when viewing a non-root path", () => {
    const wrapper = mountInspector({
      nodeDetail: createNodeDetail({
        requestedNodeId: "call-1",
        requestedRunId: "run-requested",
        requestedPath: "/payload/value",
        runId: "run-resolved",
        path: "/payload/value",
        node: {
          id: "call-1",
          status: "succeeded",
          code: 200,
          msg: ""
        },
        resultValue: "hello",
        resultText: JSON.stringify("hello")
      }),
      selectedNodeOutputSchemaText: JSON.stringify(
        {
          type: "object",
          properties: {
            payload: {
              type: "object",
              properties: {
                value: { type: "string" }
              }
            }
          }
        },
        null,
        2
      )
    })

    expect(wrapper.text()).not.toContain("Structured Result")
    expect(wrapper.text()).toContain("Result")
    expect(wrapper.findAll("pre")).toHaveLength(2)
  })
})
