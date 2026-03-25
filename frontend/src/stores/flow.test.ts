// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from "vitest"
import { setLocale } from "@/i18n"
import {
  useFlowStore,
  type ExecCapabilityRoute,
  type FlowEdge,
  type FlowGraphEditorState,
  type FlowNodeDraft
} from "./flow"

const store = useFlowStore()
const execCapQuerySimple = vi.fn()

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
  inputs: [],
  specEditorMode: "form",
  specJson: JSON.stringify({ method: "varstore::get", args_template: {} }, null, 2),
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
  ;(window as any).go = {
    flow: {
      FlowService: {
        ExecCapQuerySimple: execCapQuerySimple
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
              required: false
            },
            {
              to: "/name",
              sourceKind: "trigger",
              nodeId: "",
              path: "/payload/name",
              field: "",
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
