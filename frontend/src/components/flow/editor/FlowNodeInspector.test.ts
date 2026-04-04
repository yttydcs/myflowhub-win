// @vitest-environment jsdom

import { defineComponent } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it } from "vitest"
import { setLocale } from "@/i18n"
import FlowNodeInspector from "./FlowNodeInspector.vue"
import type { FlowNodeDetailState, FlowNodeDraft, NodeVisualFormModel } from "@/stores/flow"

const createAdvancedFields = () => ({
  transformExprMode: "literal" as const,
  transformLiteralJson: "null",
  transformSource: {
    sourceKind: "trigger" as const,
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
    sourceKind: "trigger" as const,
    nodeId: "",
    path: "",
    field: "",
    name: ""
  },
  foreachRequired: true,
  foreachBodyJson: JSON.stringify({ nodes: [], edges: [] }, null, 2),
  foreachResultNodeId: "",
  subflowId: "",
  subflowInputTemplate: "{}",
  subflowResultNodeId: ""
})

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
  retryBackoffMs: 0,
  timeoutMs: 3000,
  method: "demo::missing-schema",
  target: 0,
  argsTemplate: "{\n  \"extra\": true\n}",
  composeTemplate: "{}",
  setVarName: "",
  inputs: [],
  ...createAdvancedFields(),
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
  retryBackoffMs: 0,
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
  ...createAdvancedFields(),
  specEditorMode: "form",
  specJson: "{\n  \"name\": \"session_token\",\n  \"template\": {\"value\": null}\n}",
  x: 0,
  y: 0
})

const createTransformNode = (): FlowNodeDraft => ({
  id: "transform-1",
  kind: "transform",
  allowFail: false,
  retry: 1,
  retryBackoffMs: 0,
  timeoutMs: 3000,
  method: "",
  target: 0,
  argsTemplate: "{}",
  composeTemplate: "{}",
  setVarName: "",
  inputs: [],
  ...createAdvancedFields(),
  transformExprMode: "source",
  transformSource: {
    sourceKind: "trigger",
    nodeId: "",
    path: "/payload/value",
    field: "",
    name: ""
  },
  transformSourceRequired: false,
  specEditorMode: "form",
  specJson: "{\n  \"expr\": {\n    \"source\": { \"kind\": \"trigger\", \"path\": \"/payload/value\" },\n    \"required\": false\n  }\n}",
  x: 0,
  y: 0
})

const createBranchNode = (): FlowNodeDraft => ({
  id: "branch-1",
  kind: "branch",
  allowFail: false,
  retry: 1,
  retryBackoffMs: 0,
  timeoutMs: 3000,
  method: "",
  target: 0,
  argsTemplate: "{}",
  composeTemplate: "{}",
  setVarName: "",
  inputs: [],
  ...createAdvancedFields(),
  branchCases: [
    {
      key: "case-1",
      name: "approved",
      source: {
        sourceKind: "trigger",
        nodeId: "",
        path: "/payload/approved",
        field: "",
        name: ""
      },
      op: "eq",
      valueJson: "true"
    }
  ],
  branchDefaultCase: "approved",
  specEditorMode: "form",
  specJson:
    "{\n  \"cases\": [{\"name\": \"approved\", \"match\": {\"source\": {\"kind\": \"trigger\", \"path\": \"/payload/approved\"}, \"op\": \"eq\", \"value\": true}}],\n  \"default_case\": \"approved\"\n}",
  x: 0,
  y: 0
})

const createSubflowNode = (): FlowNodeDraft => ({
  id: "subflow-1",
  kind: "subflow",
  allowFail: false,
  retry: 1,
  retryBackoffMs: 0,
  timeoutMs: 3000,
  method: "",
  target: 0,
  argsTemplate: "{}",
  composeTemplate: "{}",
  setVarName: "",
  inputs: [
    {
      to: "/ticket_id",
      sourceKind: "trigger",
      nodeId: "",
      path: "/payload/id",
      field: "",
      name: "",
      required: true
    }
  ],
  ...createAdvancedFields(),
  subflowId: "123e4567-e89b-12d3-a456-426614174000",
  subflowInputTemplate: "{\n  \"ticket_id\": null\n}",
  subflowResultNodeId: "done",
  specEditorMode: "form",
  specJson:
    "{\n  \"flow_id\": \"123e4567-e89b-12d3-a456-426614174000\",\n  \"input_template\": {\"ticket_id\": null},\n  \"result_node_id\": \"done\"\n}",
  x: 0,
  y: 0
})

const createForeachNode = (): FlowNodeDraft => ({
  id: "foreach-1",
  kind: "foreach",
  allowFail: false,
  retry: 1,
  retryBackoffMs: 0,
  timeoutMs: 3000,
  method: "",
  target: 0,
  argsTemplate: "{}",
  composeTemplate: "{}",
  setVarName: "",
  inputs: [],
  ...createAdvancedFields(),
  foreachSource: {
    sourceKind: "trigger",
    nodeId: "",
    path: "/items",
    field: "",
    name: ""
  },
  foreachRequired: true,
  foreachBodyJson: "{\n  \"nodes\": [],\n  \"edges\": []\n}",
  foreachResultNodeId: "item_result",
  specEditorMode: "form",
  specJson:
    "{\n  \"source\": {\"kind\": \"trigger\", \"path\": \"/items\"},\n  \"required\": true,\n  \"body\": {\"nodes\": [], \"edges\": []},\n  \"result_node_id\": \"item_result\"\n}",
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

  it("renders transform authoring controls in form mode", () => {
    const wrapper = mountInspector({
      selectedNode: createTransformNode(),
      nodeIdDraft: "transform-1"
    })

    const text = wrapper.text()
    const formButton = wrapper
      .findAll("button")
      .find((button) => button.text().trim() === "Form")

    expect(text).toContain("Transform")
    expect(text).toContain("Transform Node")
    expect(text).toContain("Expression Mode")
    expect(text).toContain("Source mode reuses the same trigger/meta/ancestor/local-var binding contract")
    expect(formButton?.attributes("disabled")).toBeUndefined()
  })

  it("renders branch form authoring controls", () => {
    const wrapper = mountInspector({
      selectedNode: createBranchNode(),
      nodeIdDraft: "branch-1",
      ancestorNodeOptions: ["call-1"]
    })

    const text = wrapper.text()

    expect(text).toContain("Branch Node")
    expect(text).toContain("Default Case")
    expect(text).toContain("Add Case")
    expect(text).toContain("Case 1")
    expect(text).toContain("Match Op")
    expect(text).toContain("Match Value (JSON)")
  })

  it("renders subflow form authoring controls", () => {
    const wrapper = mountInspector({
      selectedNode: createSubflowNode(),
      nodeIdDraft: "subflow-1",
      ancestorNodeOptions: ["call-1"]
    })

    const text = wrapper.text()

    expect(text).toContain("Subflow Node")
    expect(text).toContain("Flow ID")
    expect(text).toContain("Result Node ID (Optional)")
    expect(text).toContain("Input Template (JSON)")
    expect(text).toContain("Destination writes into the child flow input template.")
  })

  it("renders foreach form authoring controls", () => {
    const wrapper = mountInspector({
      selectedNode: createForeachNode(),
      nodeIdDraft: "foreach-1"
    })

    const text = wrapper.text()
    const formButton = wrapper
      .findAll("button")
      .find((button) => button.text().trim() === "Form")

    expect(text).toContain("Foreach")
    expect(text).toContain("Foreach Node")
    expect(text).toContain("Result Node ID")
    expect(text).toContain("Body Graph (JSON)")
    expect(text).toContain("Open Visual Body Editor")
    expect(text).toContain("The visual body editor reuses the same body JSON as its source of truth.")
    expect(formButton?.attributes("disabled")).toBeUndefined()
  })

  it("renders retry backoff while keeping loop sources out of root inspectors", () => {
    const wrapper = mountInspector({
      selectedNode: createForeachNode(),
      nodeIdDraft: "foreach-1"
    })

    expect(wrapper.text()).toContain("Retry Backoff (ms)")
    expect(wrapper.find('option[value="loop_item"]').exists()).toBe(false)
    expect(wrapper.find('option[value="loop_index"]').exists()).toBe(false)
  })
})
