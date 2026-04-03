// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from "vitest"
import { setLocale } from "@/i18n"
import {
  buildDetailStructuredFields,
  createGraphEditorStateFromDraft,
  exportLooseGraphDraftFromEditorState,
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
  ...createAdvancedFields(),
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
  ...createAdvancedFields(),
  specEditorMode: "form",
  specJson: JSON.stringify({ name: "session_token", template: null }, null, 2),
  x: 0,
  y: 0,
  ...overrides
})

const createJsonOnlyNode = (
  id: string,
  kind: "transform" | "branch" | "foreach" | "subflow",
  spec: Record<string, unknown>,
  overrides: Partial<FlowNodeDraft> = {}
): FlowNodeDraft => ({
  id,
  kind,
  allowFail: false,
  retry: 1,
  timeoutMs: 3000,
  method: "",
  target: 0,
  argsTemplate: "{}",
  composeTemplate: "{}",
  setVarName: "",
  inputs: [],
  ...createAdvancedFields(),
  specEditorMode: "json",
  specJson: JSON.stringify(
    {
      ...spec,
      _ui: { x: 0, y: 0 }
    },
    null,
    2
  ),
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

  it("round-trips cron payloads and advanced node form compatibility without dropping edge cases", () => {
    store.loadFromPayload({
      flow_id: "flow-cron",
      name: "Cron Flow",
      trigger: {
        type: "cron",
        cron: "0 */5 * * *"
      },
      graph: {
        nodes: [
          {
            id: "transform1",
            kind: "transform",
            allow_fail: false,
             retry: 1,
             timeout_ms: 3000,
             spec: {
               expr: { literal: 3 },
               _ui: { x: 0, y: 0 }
             }
           },
          {
            id: "branch1",
            kind: "branch",
            allow_fail: false,
             retry: 1,
             timeout_ms: 3000,
             spec: {
               cases: [
                  {
                    name: "batch",
                    match: {
                      source: { kind: "trigger", path: "/items/count" },
                      op: "gt",
                      value: 1
                    }
                  },
                  {
                    name: "single",
                    match: {
                      source: { kind: "trigger", path: "/items/count" },
                      op: "lte",
                      value: 1
                    }
                  }
                ],
                default_case: "single",
                _ui: { x: 200, y: 0 }
             }
           },
          {
            id: "foreach1",
            kind: "foreach",
            allow_fail: false,
            retry: 1,
            timeout_ms: 3000,
            spec: {
              source: { kind: "trigger", path: "/items" },
              required: true,
              body: { nodes: [], edges: [] },
              result_node_id: "item_result",
              _ui: { x: 400, y: 0 }
            }
          },
          {
            id: "sub1",
            kind: "subflow",
            allow_fail: false,
             retry: 1,
             timeout_ms: 3000,
             spec: {
               flow_id: "123e4567-e89b-12d3-a456-426614174000",
               input_template: { ticket_id: 1 },
               result_node_id: "done",
               _ui: { x: 600, y: 0 }
            }
          }
        ],
        edges: [
          { from: "transform1", to: "branch1" },
          { from: "branch1", to: "foreach1", case: "batch" },
          { from: "branch1", to: "sub1", case: "single" }
        ]
      }
    })

    expect(store.state.triggerType).toBe("cron")
    expect(store.state.cronExpr).toBe("0 */5 * * *")
    expect(store.state.nodes.map((node) => node.kind)).toEqual(["transform", "branch", "foreach", "subflow"])
    expect(store.state.nodes.map((node) => node.specEditorMode)).toEqual(["form", "form", "form", "form"])
    expect(store.state.edges).toEqual([
      { from: "transform1", to: "branch1", case: undefined },
      { from: "branch1", to: "foreach1", case: "batch" },
      { from: "branch1", to: "sub1", case: "single" }
    ])
    expect(store.state.nodes[0]).toMatchObject({
      transformExprMode: "literal",
      transformLiteralJson: "3"
    })
    expect(store.state.nodes[1]).toMatchObject({
      branchDefaultCase: "single",
      branchCases: [
        {
          name: "batch",
          op: "gt",
          source: {
            sourceKind: "trigger",
            path: "/items/count"
          },
          valueJson: "1"
        },
        {
          name: "single",
          op: "lte",
          source: {
            sourceKind: "trigger",
            path: "/items/count"
          },
          valueJson: "1"
        }
      ]
    })
    expect(store.state.nodes[2]).toMatchObject({
      foreachSource: {
        sourceKind: "trigger",
        nodeId: "",
        path: "/items",
        field: "",
        name: ""
      },
      foreachRequired: true,
      foreachBodyJson: JSON.stringify({ nodes: [], edges: [] }, null, 2),
      foreachResultNodeId: "item_result"
    })
    expect(store.state.nodes[3]).toMatchObject({
      subflowId: "123e4567-e89b-12d3-a456-426614174000",
      subflowInputTemplate: JSON.stringify({ ticket_id: 1 }, null, 2),
      subflowResultNodeId: "done"
    })

    expect(store.exportPayload()).toMatchObject({
      flow_id: "flow-cron",
      name: "Cron Flow",
      trigger: {
        type: "cron",
        cron: "0 */5 * * *"
      },
      graph: {
        edges: [
          { from: "transform1", to: "branch1" },
          { from: "branch1", to: "foreach1", case: "batch" },
          { from: "branch1", to: "sub1", case: "single" }
        ]
      }
    })
  })

  it("round-trips transform nodes between form and json spec modes", () => {
    loadGraph([
      createJsonOnlyNode(
        "transform1",
        "transform",
        {
          expr: {
            source: { kind: "trigger", path: "/payload/count" },
            required: false
          }
        },
        {
          specEditorMode: "form",
          transformExprMode: "source",
          transformSource: {
            sourceKind: "trigger",
            nodeId: "",
            path: "/payload/count",
            field: "",
            name: ""
          },
          transformSourceRequired: false
        }
      )
    ])

    expect(store.setNodeSpecEditorMode("transform1", "json")).toBe(true)
    expect(store.state.nodes[0].specJson).toContain('"source"')
    expect(store.state.nodes[0].specJson).toContain('"/payload/count"')

    store.state.nodes[0].specJson = JSON.stringify(
      {
        expr: {
          op: "add",
          args: [{ literal: 1 }, { literal: 2 }]
        }
      },
      null,
      2
    )

    expect(store.setNodeSpecEditorMode("transform1", "form")).toBe(true)
    expect(store.state.nodes[0]).toMatchObject({
      specEditorMode: "form",
      transformExprMode: "op",
      transformOp: "add",
      transformArgsJson: JSON.stringify([{ literal: 1 }, { literal: 2 }], null, 2)
    })
  })

  it("round-trips branch nodes between form and json spec modes", () => {
    loadGraph([
      createJsonOnlyNode(
        "branch1",
        "branch",
        {
          cases: [
            {
              name: "approved",
              match: {
                source: { kind: "trigger", path: "/payload/approved" },
                op: "eq",
                value: true
              }
            }
          ],
          default_case: "rejected"
        },
        {
          specEditorMode: "form",
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
          branchDefaultCase: "rejected"
        }
      )
    ])

    expect(store.setNodeSpecEditorMode("branch1", "json")).toBe(true)
    expect(store.state.nodes[0].specJson).toContain('"default_case": "rejected"')

    store.state.nodes[0].specJson = JSON.stringify(
      {
        cases: [
          {
            name: "approved",
            match: {
              source: { kind: "trigger", path: "/payload/approved" },
              op: "exists"
            }
          }
        ],
        default_case: "approved"
      },
      null,
      2
    )

    expect(store.setNodeSpecEditorMode("branch1", "form")).toBe(true)
    expect(store.state.nodes[0]).toMatchObject({
      specEditorMode: "form",
      branchDefaultCase: "approved",
      branchCases: [
        {
          name: "approved",
          op: "exists",
          valueJson: "null"
        }
      ]
    })
  })

  it("round-trips subflow nodes between form and json spec modes", () => {
    loadGraph([
      createJsonOnlyNode(
        "sub1",
        "subflow",
        {
          flow_id: "123e4567-e89b-12d3-a456-426614174000",
          input_template: { ticket_id: 1 },
          inputs: [
            {
              to: "/ticket_id",
              source: { kind: "trigger", path: "/payload/id" },
              required: true
            }
          ],
          result_node_id: "done"
        },
        {
          specEditorMode: "form",
          subflowId: "123e4567-e89b-12d3-a456-426614174000",
          subflowInputTemplate: JSON.stringify({ ticket_id: 1 }, null, 2),
          subflowResultNodeId: "done",
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
          ]
        }
      )
    ])

    expect(store.setNodeSpecEditorMode("sub1", "json")).toBe(true)
    expect(store.state.nodes[0].specJson).toContain('"flow_id": "123e4567-e89b-12d3-a456-426614174000"')

    store.state.nodes[0].specJson = JSON.stringify(
      {
        flow_id: "123e4567-e89b-12d3-a456-426614174000",
        input_template: { ticket_id: 2 },
        result_node_id: "final"
      },
      null,
      2
    )

    expect(store.setNodeSpecEditorMode("sub1", "form")).toBe(true)
    expect(store.state.nodes[0]).toMatchObject({
      specEditorMode: "form",
      subflowId: "123e4567-e89b-12d3-a456-426614174000",
      subflowInputTemplate: JSON.stringify({ ticket_id: 2 }, null, 2),
      subflowResultNodeId: "final"
    })
  })

  it("rejects switching unsupported advanced transform JSON back to form", () => {
    loadGraph([
      createJsonOnlyNode("transform1", "transform", {
        expr: {
          script: "return 1"
        }
      })
    ])

    expect(() => store.setNodeSpecEditorMode("transform1", "form")).toThrowError(
      "Node kind transform advanced spec contains fields that ordinary mode cannot represent yet."
    )
  })

  it("round-trips foreach nodes between form and json spec modes", () => {
    loadGraph([
      createJsonOnlyNode(
        "foreach1",
        "foreach",
        {
          source: { kind: "trigger", path: "/items" },
          required: false,
          body: {
            nodes: [{ id: "item_result", kind: "compose", spec: { template: null } }],
            edges: []
          },
          result_node_id: "item_result"
        },
        {
          specEditorMode: "form",
          foreachSource: {
            sourceKind: "trigger",
            nodeId: "",
            path: "/items",
            field: "",
            name: ""
          },
          foreachRequired: false,
          foreachBodyJson: JSON.stringify(
            {
              nodes: [{ id: "item_result", kind: "compose", spec: { template: null } }],
              edges: []
            },
            null,
            2
          ),
          foreachResultNodeId: "item_result"
        }
      )
    ])

    expect(store.setNodeSpecEditorMode("foreach1", "json")).toBe(true)
    expect(store.state.nodes[0].specJson).toContain('"result_node_id": "item_result"')

    store.state.nodes[0].specJson = JSON.stringify(
      {
        source: { kind: "flow_var", name: "items_batch", path: "/items" },
        required: true,
        body: { nodes: [], edges: [] },
        result_node_id: "done"
      },
      null,
      2
    )

    expect(store.setNodeSpecEditorMode("foreach1", "form")).toBe(true)
    expect(store.state.nodes[0]).toMatchObject({
      specEditorMode: "form",
      foreachSource: {
        sourceKind: "flow_var",
        nodeId: "",
        path: "/items",
        field: "",
        name: "items_batch"
      },
      foreachRequired: true,
      foreachBodyJson: JSON.stringify({ nodes: [], edges: [] }, null, 2),
      foreachResultNodeId: "done"
    })
  })

  it("rejects switching unsupported advanced foreach JSON back to form", () => {
    loadGraph([
      createJsonOnlyNode("foreach1", "foreach", {
        source: { kind: "trigger", path: "/items" },
        required: true,
        body: { nodes: [], edges: [] },
        result_node_id: "done",
        max_parallel: 4
      })
    ])

    expect(() => store.setNodeSpecEditorMode("foreach1", "form")).toThrowError(
      "Node kind foreach advanced spec contains fields that ordinary mode cannot represent yet."
    )
  })

  it("updates selected branch edge cases through the minimal edge editor state", () => {
    loadGraph(
      [
        createJsonOnlyNode("branch1", "branch", { cases: [] }),
        createCallNode("call1", { method: "demo::call" })
      ],
      [{ from: "branch1", to: "call1" }],
      { selectedEdgeIndex: 0 }
    )

    store.setSelectedEdgeCase("approved")

    expect(store.state.edges[0]).toEqual({ from: "branch1", to: "call1", case: "approved" })
    expect(store.exportGraphDraft().edges).toEqual([{ from: "branch1", to: "call1", case: "approved" }])
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

  it("round-trips graph drafts through exported editor-state helpers for foreach body sessions", () => {
    const snapshot = createGraphEditorStateFromDraft({
      nodes: [
        {
          id: "inner1",
          kind: "call",
          allow_fail: false,
          retry: 1,
          timeout_ms: 3000,
          spec: {
            method: "demo::inner",
            args_template: {
              value: 1
            },
            _ui: {
              x: 12,
              y: 34
            }
          }
        }
      ],
      edges: []
    })

    snapshot.nodes[0].specEditorMode = "json"
    snapshot.nodes[0].specJson = JSON.stringify(
      {
        method: "demo::inner",
        args_template: {
          value: 2
        },
        _ui: {
          x: 12,
          y: 34
        }
      },
      null,
      2
    )

    expect(exportLooseGraphDraftFromEditorState(snapshot)).toEqual({
      nodes: [
        {
          id: "inner1",
          kind: "call",
          allow_fail: false,
          retry: 1,
          timeout_ms: 3000,
          spec: {
            method: "demo::inner",
            args_template: {
              value: 2
            },
            _ui: {
              x: 12,
              y: 34
            }
          }
        }
      ],
      edges: []
    })
  })
})
