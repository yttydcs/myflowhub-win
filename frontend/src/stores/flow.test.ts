// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from "vitest"
import { setLocale } from "@/i18n"
import {
  buildDetailStructuredFields,
  flowStatusLabelKey,
  useFlowStore,
  type ExecCapabilityRoute,
  type FlowEdge,
  type FlowGraphEditorState,
  type FlowNodeDraft
} from "./flow"

const store = useFlowStore()
const execCapQuerySimple = vi.fn()
const detailSimple = vi.fn()
const statusSimple = vi.fn()

const emptyStatusState = {
  status: "",
  runId: "",
  executorNode: 0,
  nodes: []
}

const createCallNode = (id: string, overrides: Partial<FlowNodeDraft> = {}): FlowNodeDraft => ({
  id,
  kind: "call",
  allowFail: false,
  retry: 1,
  timeoutMs: 3000,
  method: "varstore::get",
  target: 0,
  argsTemplate: "{}",
  composeTemplate: "{}",
  setVarName: "",
  inputs: [],
  specEditorMode: "form",
  specJson: JSON.stringify({ method: "varstore::get", args_template: {} }, null, 2),
  x: 0,
  y: 0,
  ...overrides
})

const createSetVarNode = (id: string, overrides: Partial<FlowNodeDraft> = {}): FlowNodeDraft => ({
  id,
  kind: "set_var",
  allowFail: false,
  retry: 1,
  timeoutMs: 3000,
  method: "",
  target: 0,
  argsTemplate: "{}",
  composeTemplate: "null",
  setVarName: "session_token",
  inputs: [],
  specEditorMode: "form",
  specJson: JSON.stringify({ name: "session_token", template: null }, null, 2),
  x: 0,
  y: 0,
  ...overrides
})

const loadGraph = (nodes: FlowNodeDraft[], edges: FlowEdge[] = [], selection?: Partial<FlowGraphEditorState>) => {
  store.loadGraphEditorState({
    nodes,
    edges,
    selectedNodeIndex: selection?.selectedNodeIndex ?? -1,
    selectedEdgeIndex: selection?.selectedEdgeIndex ?? -1
  })
}

const createCapabilityRoute = (method: string, overrides: Partial<ExecCapabilityRoute> = {}): ExecCapabilityRoute => ({
  key: `${method}|route`,
  providerNode: 1,
  viaNode: 0,
  method,
  version: "",
  defaultTimeoutMs: 3000,
  permissions: [],
  tags: {},
  inputSchema: null,
  outputSchema: null,
  label: method,
  ...overrides
})

const createVarStoreGetInputSchema = () => ({
  title: "VarStore Get",
  type: "object",
  required: ["owner", "name"],
  properties: {
    owner: { type: "integer" },
    name: { type: "string" }
  }
})

const seedStatusState = () => {
  store.state.statusRunId = "run-stale"
  store.state.lastStatus = {
    status: "running",
    runId: "run-stale",
    executorNode: 77,
    nodes: [
      {
        id: "node-1",
        status: "running",
        code: 102,
        msg: "pending"
      }
    ]
  }
}

const expectStatusStateReset = () => {
  expect(store.state.statusRunId).toBe("")
  expect(store.state.lastStatus).toEqual(emptyStatusState)
}

beforeEach(() => {
  setLocale("en")
  store.newDraft()
  loadGraph([])
  store.state.flowId = "flow-1"
  store.state.flowName = "Flow 1"
  store.state.targetId = "100"
  store.state.selfNodeId = 1
  store.state.hubId = 100
  store.clearMessage()
  execCapQuerySimple.mockReset()
  detailSimple.mockReset()
  statusSimple.mockReset()
  ;(window as any).go = {
    flow: {
      FlowService: {
        ExecCapQuerySimple: execCapQuerySimple,
        DetailSimple: detailSimple,
        StatusSimple: statusSimple
      }
    }
  }
})

describe("flow store", () => {
  it("lists ancestors in graph order and rejects non-ancestor bindings", () => {
    loadGraph(
      [createCallNode("n1"), createCallNode("n2"), createCallNode("n3")],
      [
        { from: "n1", to: "n2" },
        { from: "n2", to: "n3" },
        { from: "n1", to: "n3" }
      ]
    )

    expect(store.listAncestorNodeIds("n3")).toEqual(["n1", "n2"])
    expect(() =>
      store.setFieldBinding("n2", "/name", {
        kind: "node_result",
        nodeId: "n3",
        path: "/payload/id",
        required: true
      })
    ).toThrowError("Source node must be an ancestor.")
  })

  it("writes bindings into the graph draft and preserves literals when clearing bindings", () => {
    loadGraph(
      [
        createCallNode("source"),
        createCallNode("target", {
          method: "varstore::set"
        })
      ],
      [{ from: "source", to: "target" }]
    )

    store.setFieldLiteralValue("target", "/value", "seed")
    store.setFieldBinding("target", "/value", {
      kind: "node_result",
      nodeId: "source",
      path: "/payload/id",
      required: true
    })

    expect(store.state.nodes[1].inputs).toEqual([
      {
        to: "/value",
        sourceKind: "node_result",
        nodeId: "source",
        path: "/payload/id",
        field: "",
        name: "",
        required: true
      }
    ])

    const graph = store.exportGraphDraft()
    expect(graph.edges).toEqual([{ from: "source", to: "target" }])
    expect(graph.nodes[1]).toMatchObject({
      id: "target",
      kind: "call",
      spec: {
        method: "varstore::set",
        args_template: { value: "seed" },
        inputs: [
          {
            to: "/value",
            source: {
              kind: "node_result",
              node_id: "source",
              path: "/payload/id"
            },
            required: true
          }
        ]
      }
    })

    store.clearFieldBinding("target", "/value")

    expect(store.state.nodes[1].inputs).toEqual([])
    expect(JSON.parse(store.state.nodes[1].argsTemplate)).toEqual({ value: "seed" })
  })

  it("round-trips a call node between form and json spec modes", () => {
    loadGraph([
      createCallNode("n1", {
        method: "varstore::set",
        argsTemplate: JSON.stringify({ value: "seed" }, null, 2)
      })
    ])

    expect(store.setNodeSpecEditorMode("n1", "json")).toBe(true)
    expect(store.state.nodes[0].specEditorMode).toBe("json")
    expect(JSON.parse(store.state.nodes[0].specJson)).toMatchObject({
      method: "varstore::set",
      args_template: { value: "seed" },
      _ui: { x: 0, y: 0 }
    })

    store.state.nodes[0].specJson = JSON.stringify(
      {
        method: "varstore::set",
        target: 12,
        args_template: {
          value: "updated"
        },
        inputs: [
          {
            to: "/value",
            source: {
              kind: "trigger",
              path: "/payload/value"
            },
            required: true
          }
        ],
        _ui: {
          x: 80,
          y: 120
        }
      },
      null,
      2
    )

    expect(store.setNodeSpecEditorMode("n1", "form")).toBe(true)
    expect(store.state.nodes[0]).toMatchObject({
      specEditorMode: "form",
      method: "varstore::set",
      target: 12,
      inputs: [
        {
          to: "/value",
          sourceKind: "trigger",
          nodeId: "",
          path: "/payload/value",
          field: "",
          name: "",
          required: true
        }
      ]
    })
    expect(JSON.parse(store.state.nodes[0].argsTemplate)).toEqual({ value: "updated" })
  })

  it("prunes stale literals and bindings when switching methods in form mode", () => {
    loadGraph(
      [
        createCallNode("n1", {
          method: "varstore::set",
          argsTemplate: JSON.stringify(
            {
              owner: 7,
              name: "token",
              value: "secret",
              visibility: "public"
            },
            null,
            2
          ),
          inputs: [
            {
              to: "/visibility",
              sourceKind: "trigger",
              nodeId: "",
              path: "/payload/visibility",
              field: "",
              name: "",
              required: false
            },
            {
              to: "/name",
              sourceKind: "trigger",
              nodeId: "",
              path: "/payload/name",
              field: "",
              name: "",
              required: true
            }
          ]
        })
      ],
      [],
      { selectedNodeIndex: 0 }
    )

    store.state.execCapabilities = [
      createCapabilityRoute("varstore::get", {
        inputSchema: createVarStoreGetInputSchema()
      })
    ]
    store.applyCallCapability("varstore::get|route")

    expect(store.state.nodes[0].method).toBe("varstore::get")
    expect(JSON.parse(store.state.nodes[0].argsTemplate)).toEqual({
      owner: 7,
      name: "token"
    })
    expect(store.state.nodes[0].inputs).toEqual([
      {
        to: "/name",
        sourceKind: "trigger",
        nodeId: "",
        path: "/payload/name",
        field: "",
        name: "",
        required: true
      }
    ])

    const visualForm = store.getNodeVisualForm("n1")
    expect(visualForm.compatibility.supported).toBe(true)
    expect(visualForm.compatibility.reasons).toEqual([])
  })

  it("hydrates an existing call node capability without replacing the current capability list", async () => {
    loadGraph(
      [
        createCallNode("n1", {
          method: "varstore::get",
          target: 0
        })
      ],
      [],
      { selectedNodeIndex: 0 }
    )

    store.state.execCapabilities = [
      createCapabilityRoute("demo::existing", {
        key: "demo::existing|route",
        providerNode: 100
      })
    ]

    execCapQuerySimple.mockResolvedValue({
      code: 1,
      routes: [
        {
          provider_node: 100,
          via_node: 0,
          method: "varstore::get",
          version: "v1",
          default_timeout_ms: 3000,
          permissions: [],
          tags: {},
          input_schema: createVarStoreGetInputSchema(),
          output_schema: {
            type: "object",
            properties: {
              owner: { type: "integer" },
              name: { type: "string" }
            }
          }
        }
      ]
    })

    await expect(store.ensureNodeCapabilityLoaded("n1")).resolves.toBe(true)

    expect(execCapQuerySimple).toHaveBeenCalledWith(
      1,
      100,
      expect.objectContaining({
        method: "varstore::get",
        prefix: true,
        include_schema: true
      })
    )
    expect(store.state.execCapabilities.map((route) => route.method)).toEqual([
      "demo::existing",
      "varstore::get"
    ])

    const visualForm = store.getNodeVisualForm("n1")
    expect(visualForm.compatibility.supported).toBe(true)
    expect(visualForm.schema?.source).toBe("capability")
  })

  it("loads node detail through the flow service and stores the formatted response", async () => {
    loadGraph(
      [createCallNode("call1", { method: "demo::call" })],
      [],
      { selectedNodeIndex: 0 }
    )

    detailSimple.mockResolvedValue({
      code: 1,
      run_id: "run-resolved",
      path: "/payload/value",
      node: {
        id: "call1",
        status: "succeeded",
        code: 201,
        msg: "detail ok"
      },
      result: {
        payload: {
          value: "hello"
        }
      }
    })

    await expect(store.loadNodeDetail("call1", "run-requested", "/payload/value")).resolves.toBe(true)

    expect(detailSimple).toHaveBeenCalledWith(
      1,
      100,
      expect.objectContaining({
        flow_id: "flow-1",
        run_id: "run-requested",
        node_id: "call1",
        path: "/payload/value"
      })
    )
    expect(store.state.nodeDetail).toEqual({
      loading: false,
      error: "",
      requestedNodeId: "call1",
      requestedRunId: "run-requested",
      requestedPath: "/payload/value",
      runId: "run-resolved",
      path: "/payload/value",
      node: {
        id: "call1",
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
    })
    expect(store.state.statusRunId).toBe("run-resolved")
  })

  it("builds structured detail fields from supported root output schemas", () => {
    const fields = buildDetailStructuredFields(
      JSON.stringify(
        {
          type: "object",
          properties: {
            payload: {
              type: "object",
              properties: {
                value: { type: "string" },
                metadata: {
                  type: "object",
                  properties: {}
                }
              }
            }
          }
        },
        null,
        2
      ),
      {
        payload: {
          value: "hello",
          metadata: {
            version: 2
          }
        }
      }
    )

    expect(fields).toEqual([
      expect.objectContaining({
        pointer: "/payload/value",
        valueText: "hello",
        missing: false,
        multiline: false
      }),
      expect.objectContaining({
        pointer: "/payload/metadata",
        valueText: JSON.stringify(
          {
            version: 2
          },
          null,
          2
        ),
        missing: false,
        multiline: true
      })
    ])
  })

  it("disables structured detail fields when the query targets a non-root path", () => {
    expect(
      buildDetailStructuredFields(
        JSON.stringify(
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
        ),
        {
          payload: {
            value: "hello"
          }
        },
        "/payload/value"
      )
    ).toEqual([])
  })

  it("resets status state when starting a new draft or replacing graph content", () => {
    seedStatusState()
    store.newDraft()
    expectStatusStateReset()

    seedStatusState()
    store.loadGraphDraft({
      nodes: [{ id: "call1", kind: "call", spec: { method: "demo::call", args_template: {} } }],
      edges: []
    })
    expectStatusStateReset()

    seedStatusState()
    store.loadGraphEditorState({
      nodes: [createCallNode("call1")],
      edges: [],
      selectedNodeIndex: 0,
      selectedEdgeIndex: -1
    })
    expectStatusStateReset()
  })

  it("maps cancelled status labels to the dedicated label key", () => {
    expect(flowStatusLabelKey("cancelled")).toBe("Cancelled")
    expect(flowStatusLabelKey(" CANCELLED ")).toBe("Cancelled")
  })

  it("maps flow status payloads into lastStatus without storing detail payloads", async () => {
    statusSimple.mockResolvedValue({
      code: 1,
      status: "running",
      run_id: "run-fresh",
      executor_node: 100,
      nodes: [
        {
          id: "call1",
          status: "queued",
          code: 102,
          msg: "waiting"
        },
        {
          id: "call2",
          status: "cancelled",
          code: 499,
          msg: "stopped"
        }
      ],
      detail: {
        ignored: true
      }
    })

    await store.statusFlow("run-requested")

    expect(statusSimple).toHaveBeenCalledWith(
      1,
      100,
      expect.objectContaining({
        flow_id: "flow-1",
        run_id: "run-requested"
      })
    )
    expect(store.state.lastStatus).toEqual({
      status: "running",
      runId: "run-fresh",
      executorNode: 100,
      nodes: [
        {
          id: "call1",
          status: "queued",
          code: 102,
          msg: "waiting"
        },
        {
          id: "call2",
          status: "cancelled",
          code: 499,
          msg: "stopped"
        }
      ]
    })
    expect(store.state.statusRunId).toBe("run-fresh")
  })

  it("formats the selected call node output schema as JSON text", () => {
    loadGraph([createCallNode("call1", { method: "demo::call" })])

    store.state.execCapabilities = [
      createCapabilityRoute("demo::call", {
        providerNode: 100,
        outputSchema: {
          type: "object",
          properties: {
            value: { type: "string" }
          }
        }
      })
    ]

    expect(store.getNodeOutputSchemaText("call1")).toBe(
      JSON.stringify(
        {
          type: "object",
          properties: {
            value: { type: "string" }
          }
        },
        null,
        2
      )
    )
  })

  it("writes flow local var bindings into call nodes", () => {
    loadGraph([createCallNode("call1", { method: "demo::call" })])

    store.setFieldBinding("call1", "/name", {
      kind: "flow_var",
      name: "session_token",
      path: "/payload/id",
      required: true
    })

    expect(store.state.nodes[0].inputs).toEqual([
      {
        to: "/name",
        sourceKind: "flow_var",
        nodeId: "",
        path: "/payload/id",
        field: "",
        name: "session_token",
        required: true
      }
    ])

    expect(store.exportGraphDraft().nodes[0]).toMatchObject({
      kind: "call",
      spec: {
        method: "demo::call",
        args_template: {},
        inputs: [
          {
            to: "/name",
            source: {
              kind: "flow_var",
              name: "session_token",
              path: "/payload/id"
            },
            required: true
          }
        ]
      }
    })
  })

  it("round-trips a set_var node with flow_var bindings between form and json spec modes", () => {
    loadGraph([
      createSetVarNode("set1", {
        composeTemplate: JSON.stringify({ value: null }, null, 2),
        inputs: [
          {
            to: "/value",
            sourceKind: "flow_var",
            nodeId: "",
            path: "/payload/id",
            field: "",
            name: "source_token",
            required: true
          }
        ]
      })
    ])

    expect(store.exportGraphDraft().nodes[0]).toMatchObject({
      id: "set1",
      kind: "set_var",
      spec: {
        name: "session_token",
        template: { value: null },
        inputs: [
          {
            to: "/value",
            source: {
              kind: "flow_var",
              name: "source_token",
              path: "/payload/id"
            },
            required: true
          }
        ]
      }
    })

    expect(store.setNodeSpecEditorMode("set1", "json")).toBe(true)
    expect(JSON.parse(store.state.nodes[0].specJson)).toMatchObject({
      name: "session_token",
      template: { value: null }
    })

    store.state.nodes[0].specJson = JSON.stringify(
      {
        name: "session_value",
        template: {
          value: "updated"
        },
        inputs: [
          {
            to: "/value",
            source: {
              kind: "flow_var",
              name: "upstream_token",
              path: "/payload/value"
            },
            required: false
          }
        ],
        _ui: {
          x: 24,
          y: 48
        }
      },
      null,
      2
    )

    expect(store.setNodeSpecEditorMode("set1", "form")).toBe(true)
    expect(store.state.nodes[0]).toMatchObject({
      kind: "set_var",
      setVarName: "session_value",
      composeTemplate: JSON.stringify({ value: "updated" }, null, 2),
      inputs: [
        {
          to: "/value",
          sourceKind: "flow_var",
          nodeId: "",
          path: "/payload/value",
          field: "",
          name: "upstream_token",
          required: false
        }
      ]
    })
  })

  it("keeps graph editor signatures stable across selection changes while round-tripping editor state", () => {
    loadGraph(
      [createCallNode("n1"), createCallNode("n2")],
      [{ from: "n1", to: "n2" }]
    )

    const baseSignature = store.graphEditorSignature()

    store.selectNodeById("n1")
    const nodeSelectionState = store.exportGraphEditorState()
    expect(nodeSelectionState.selectedNodeIndex).toBe(0)
    expect(store.graphEditorSignature()).toBe(baseSignature)

    store.selectEdgeByEndpoints("n1", "n2")
    const edgeSelectionState = store.exportGraphEditorState()
    expect(edgeSelectionState.selectedEdgeIndex).toBe(0)
    expect(store.graphEditorSignature()).toBe(baseSignature)

    store.loadGraphEditorState({
      ...edgeSelectionState,
      selectedNodeIndex: 1,
      selectedEdgeIndex: -1
    })

    expect(store.state.selectedNodeIndex).toBe(1)
    expect(store.state.selectedEdgeIndex).toBe(-1)
    expect(store.graphEditorSignature()).toBe(baseSignature)
  })
})
