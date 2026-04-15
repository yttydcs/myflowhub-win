// 本文件覆盖 Win 前端使用的独立 `FlowEditorWindow` 窗口行为

// @vitest-environment jsdom

import { defineComponent, nextTick } from "vue"
import { config, mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { setLocale } from "@/i18n"

const projectsStore = {
  state: {
    projects: [] as any[]
  },
  loadProjects: vi.fn(async () => undefined),
  getProjectByID: vi.fn(),
  saveProjectGraph: vi.fn()
}

const toastStore = {
  error: vi.fn(),
  errorOf: vi.fn(),
  info: vi.fn(),
  success: vi.fn(),
  warn: vi.fn()
}

const queryExecCapabilities = vi.fn(async () => undefined)
const ensureNodeCapabilityLoaded = vi.fn(async () => false)
const runFlow = vi.fn(async () => undefined)
const statusFlow = vi.fn(async () => undefined)
const listRunsFlow = vi.fn(async () => undefined)
const cancelRunFlow = vi.fn(async () => undefined)

vi.mock("vue-router", () => ({
  useRoute: () => ({
    query: {
      projectId: "project-1"
    }
  })
}))

vi.mock("@/stores/flowProjects", () => ({
  useFlowProjectsStore: () => projectsStore
}))

vi.mock("@/stores/session", () => ({
  useSessionStore: () => ({
    auth: {
      nodeId: 1,
      hubId: 100
    }
  })
}))

vi.mock("@/stores/toast", () => ({
  useToastStore: () => toastStore
}))

vi.mock("@/stores/flow", async () => {
  const actual = await vi.importActual<typeof import("@/stores/flow")>("@/stores/flow")
  const store = actual.useFlowStore()
  return {
    ...actual,
    useFlowStore: () => ({
      ...store,
      queryExecCapabilities,
      ensureNodeCapabilityLoaded,
      runFlow,
      statusFlow,
      listRunsFlow,
      cancelRunFlow
    })
  }
})

import FlowEditorWindow from "./FlowEditorWindow.vue"
import { useFlowStore } from "@/stores/flow"

const FlowEditorToolbarStub = defineComponent({
  props: {
    flowStatusLabel: { type: String, default: "" },
    currentRunIdLabel: { type: String, default: "" }
  },
  emits: ["run-flow", "refresh-status", "load-run-history", "cancel-run", "save-project"],
  template: `
    <div data-test="toolbar" :data-flow-status-label="flowStatusLabel" :data-current-run-id-label="currentRunIdLabel">
      <button data-test="run-flow" type="button" @click="$emit('run-flow')">Run</button>
      <button data-test="refresh-status" type="button" @click="$emit('refresh-status')">Refresh</button>
      <button data-test="load-run-history" type="button" @click="$emit('load-run-history')">History</button>
      <button data-test="cancel-run" type="button" @click="$emit('cancel-run')">Cancel</button>
      <button data-test="save-project" type="button" @click="$emit('save-project')">Save</button>
    </div>
  `
})

const FlowCanvasStub = defineComponent({
  props: {
    nodes: { type: Array, default: () => [] },
    edges: { type: Array, default: () => [] },
    statusNodes: { type: Array, default: () => [] }
  },
  emits: ["node-moved", "select-node"],
  template: `
    <div
      data-test="canvas"
      :data-status-count="String(statusNodes.length)"
      :data-node-count="String(nodes.length)"
      :data-edge-count="String(edges.length)"
    >
      <button
        data-test="move-node"
        type="button"
        @click="nodes.length && $emit('node-moved', nodes[0].id, 321, 654)"
      >
        Move
      </button>
      <button
        data-test="select-first-node"
        type="button"
        @click="nodes.length && $emit('select-node', nodes[0].id)"
      >
        Select
      </button>
    </div>
  `
})

const FlowNodeInspectorStub = defineComponent({
  emits: ["open-method", "edit-foreach-body"],
  template: `
    <div>
      <button data-test="open-method" type="button" @click="$emit('open-method')">Open Method</button>
      <button data-test="open-body" type="button" @click="$emit('edit-foreach-body')">Open Body</button>
    </div>
  `
})

const FlowEdgeInspectorStub = defineComponent({
  props: {
    sourceNodeKind: { type: String, default: "" },
    selectedEdge: { type: Object, default: null }
  },
  template: `
    <div
      data-test="edge-inspector"
      :data-source-kind="sourceNodeKind"
      :data-edge-case="selectedEdge?.case ?? ''"
    />
  `
})

const FlowMethodPickerDialogStub = defineComponent({
  props: {
    open: { type: Boolean, default: false },
    methodSearch: { type: String, default: "" }
  },
  template: `
    <div
      data-test="method-dialog"
      :data-open="String(open)"
      :data-method-search="methodSearch"
    />
  `
})

const ButtonStub = defineComponent({
  emits: ["click"],
  template: `<button type="button" @click="$emit('click', $event)"><slot /></button>`
})

const SimpleStub = defineComponent({
  template: `<div />`
})

const flushAsync = async () => {
  await Promise.resolve()
  await Promise.resolve()
  await nextTick()
}

describe("FlowEditorWindow", () => {
  beforeEach(() => {
    const flowStore = useFlowStore()
    setLocale("en")
    vi.clearAllMocks()
    config.global.stubs = {
      ...(config.global.stubs ?? {}),
      Button: ButtonStub
    }
    queryExecCapabilities.mockResolvedValue(undefined)
    ensureNodeCapabilityLoaded.mockResolvedValue(false)
    runFlow.mockResolvedValue(undefined)
    statusFlow.mockResolvedValue(undefined)
    listRunsFlow.mockResolvedValue(undefined)
    cancelRunFlow.mockResolvedValue(undefined)
    projectsStore.saveProjectGraph.mockResolvedValue({
      id: "project-1",
      flowId: "project-1",
      name: "Project 1",
      updatedAt: "2026-03-25T12:05:00.000Z",
      graph: {
        nodes: [],
        edges: []
      }
    })
    projectsStore.state.projects = []
    flowStore.newDraft()
    flowStore.loadGraphEditorState({
      nodes: [],
      edges: [],
      selectedNodeIndex: -1,
      selectedEdgeIndex: -1
    })
    projectsStore.getProjectByID.mockReturnValue({
      id: "project-1",
      flowId: "project-1",
      name: "Project 1",
      updatedAt: "2026-03-25T12:00:00.000Z",
      graph: {
        nodes: [
          {
            id: "call1",
            kind: "call",
            allow_fail: false,
            retry: 1,
            timeout_ms: 3000,
            spec: {
              method: "demo::existing",
              args_template: {}
            }
          }
        ],
        edges: []
      }
    })
  })

  it("opens the method picker with an empty filter instead of the selected method", async () => {
    const flowStore = useFlowStore()

    const wrapper = mount(FlowEditorWindow, {
      global: {
        stubs: {
          FlowEditorToolbar: FlowEditorToolbarStub,
          FlowCanvas: FlowCanvasStub,
          FlowNodeInspector: FlowNodeInspectorStub,
          FlowEdgeInspector: FlowEdgeInspectorStub,
          FlowMethodPickerDialog: FlowMethodPickerDialogStub,
          FlowFieldBindingDialog: SimpleStub,
          FlowAddNodeDialog: SimpleStub
        }
      }
    })

    await flushAsync()

    flowStore.state.execCapabilities = [
      {
        key: "100|0|demo::existing|v1",
        providerNode: 100,
        viaNode: 0,
        method: "demo::existing",
        version: "v1",
        defaultTimeoutMs: 0,
        permissions: [],
        tags: {},
        inputSchema: null,
        outputSchema: null,
        label: "100 · demo::existing@v1"
      }
    ]

    flowStore.selectNodeById("call1")
    await nextTick()

    await wrapper.get('[data-test="open-method"]').trigger("click")
    await flushAsync()

    const dialog = wrapper.get('[data-test="method-dialog"]')
    expect(dialog.attributes("data-open")).toBe("true")
    expect(dialog.attributes("data-method-search")).toBe("")
  })

  it("hydrates the selected call node capability when opening an existing project node", async () => {
    const flowStore = useFlowStore()

    mount(FlowEditorWindow, {
      global: {
        stubs: {
          FlowEditorToolbar: FlowEditorToolbarStub,
          FlowCanvas: FlowCanvasStub,
          FlowNodeInspector: FlowNodeInspectorStub,
          FlowEdgeInspector: FlowEdgeInspectorStub,
          FlowMethodPickerDialog: FlowMethodPickerDialogStub,
          FlowFieldBindingDialog: SimpleStub,
          FlowAddNodeDialog: SimpleStub
        }
      }
    })

    await flushAsync()

    flowStore.selectNodeById("call1")
    await flushAsync()

    expect(ensureNodeCapabilityLoaded).toHaveBeenCalledWith("call1")
  })

  it("refreshes status with the current run id and passes status nodes into the canvas", async () => {
    const flowStore = useFlowStore()

    const wrapper = mount(FlowEditorWindow, {
      global: {
        stubs: {
          FlowEditorToolbar: FlowEditorToolbarStub,
          FlowCanvas: FlowCanvasStub,
          FlowNodeInspector: FlowNodeInspectorStub,
          FlowEdgeInspector: FlowEdgeInspectorStub,
          FlowMethodPickerDialog: FlowMethodPickerDialogStub,
          FlowFieldBindingDialog: SimpleStub,
          FlowAddNodeDialog: SimpleStub
        }
      }
    })

    await flushAsync()

    flowStore.state.statusRunId = "run-1"
    flowStore.state.lastStatus = {
      status: "running",
      runId: "run-1",
      executorNode: 100,
      nodes: [
        {
          id: "call1",
          status: "running",
          code: 0,
          msg: ""
        }
      ]
    }
    await nextTick()

    await wrapper.get('[data-test="refresh-status"]').trigger("click")
    await flushAsync()

    expect(statusFlow).toHaveBeenCalledWith("run-1")
    expect(wrapper.get('[data-test="canvas"]').attributes("data-status-count")).toBe("1")
    expect(wrapper.get('[data-test="toolbar"]').attributes("data-flow-status-label")).toBe("Running")
    expect(wrapper.get('[data-test="toolbar"]').attributes("data-current-run-id-label")).toContain("run-1")
  })

  it("runs the current flow from the toolbar", async () => {
    const wrapper = mount(FlowEditorWindow, {
      global: {
        stubs: {
          FlowEditorToolbar: FlowEditorToolbarStub,
          FlowCanvas: FlowCanvasStub,
          FlowNodeInspector: FlowNodeInspectorStub,
          FlowEdgeInspector: FlowEdgeInspectorStub,
          FlowMethodPickerDialog: FlowMethodPickerDialogStub,
          FlowFieldBindingDialog: SimpleStub,
          FlowAddNodeDialog: SimpleStub
        }
      }
    })

    await flushAsync()

    await wrapper.get('[data-test="run-flow"]').trigger("click")
    await flushAsync()

    expect(runFlow).toHaveBeenCalled()
  })

  it("loads run history on project open and from the toolbar", async () => {
    const flowStore = useFlowStore()
    listRunsFlow.mockImplementation(async () => {
      flowStore.state.runHistory = [
        {
          runId: "run-2",
          status: "running",
          code: 1,
          msg: "",
          startedAt: "2026-03-25T12:00:00.000Z",
          endedAt: ""
        }
      ]
      flowStore.state.statusRunId = "run-2"
    })

    const wrapper = mount(FlowEditorWindow, {
      global: {
        stubs: {
          FlowEditorToolbar: FlowEditorToolbarStub,
          FlowCanvas: FlowCanvasStub,
          FlowNodeInspector: FlowNodeInspectorStub,
          FlowEdgeInspector: FlowEdgeInspectorStub,
          FlowMethodPickerDialog: FlowMethodPickerDialogStub,
          FlowFieldBindingDialog: SimpleStub,
          FlowAddNodeDialog: SimpleStub
        }
      }
    })

    await flushAsync()

    expect(listRunsFlow).toHaveBeenCalledWith(50)
    expect(wrapper.text()).toContain("1 runs")
    expect(wrapper.findAll("option").some((option) => option.text().includes("run-2"))).toBe(true)

    listRunsFlow.mockClear()
    await wrapper.get('[data-test="load-run-history"]').trigger("click")
    await flushAsync()

    expect(listRunsFlow).toHaveBeenCalledWith(50)
  })

  it("cancels the selected run from the toolbar", async () => {
    const flowStore = useFlowStore()

    const wrapper = mount(FlowEditorWindow, {
      global: {
        stubs: {
          FlowEditorToolbar: FlowEditorToolbarStub,
          FlowCanvas: FlowCanvasStub,
          FlowNodeInspector: FlowNodeInspectorStub,
          FlowEdgeInspector: FlowEdgeInspectorStub,
          FlowMethodPickerDialog: FlowMethodPickerDialogStub,
          FlowFieldBindingDialog: SimpleStub,
          FlowAddNodeDialog: SimpleStub
        }
      }
    })

    await flushAsync()

    flowStore.state.statusRunId = "run-1"
    await nextTick()

    await wrapper.get('[data-test="cancel-run"]').trigger("click")
    await flushAsync()

    expect(cancelRunFlow).toHaveBeenCalledWith("run-1")
  })

  it("renders the edge inspector for selected branch edges", async () => {
    const flowStore = useFlowStore()

    projectsStore.getProjectByID.mockReturnValue({
      id: "project-1",
      flowId: "project-1",
      name: "Project 1",
      updatedAt: "2026-03-25T12:00:00.000Z",
      graph: {
        nodes: [
          {
            id: "branch1",
            kind: "branch",
            allow_fail: false,
            retry: 1,
            timeout_ms: 3000,
            spec: {
              cases: [{ case: "approved" }],
              _ui: { x: 0, y: 0 }
            }
          },
          {
            id: "call1",
            kind: "call",
            allow_fail: false,
            retry: 1,
            timeout_ms: 3000,
            spec: {
              method: "demo::existing",
              args_template: {}
            }
          }
        ],
        edges: [{ from: "branch1", to: "call1", case: "approved" }]
      }
    })

    const wrapper = mount(FlowEditorWindow, {
      global: {
        stubs: {
          FlowEditorToolbar: FlowEditorToolbarStub,
          FlowCanvas: FlowCanvasStub,
          FlowNodeInspector: FlowNodeInspectorStub,
          FlowEdgeInspector: FlowEdgeInspectorStub,
          FlowMethodPickerDialog: FlowMethodPickerDialogStub,
          FlowFieldBindingDialog: SimpleStub,
          FlowAddNodeDialog: SimpleStub
        }
      }
    })

    await flushAsync()

    flowStore.selectEdgeByEndpoints("branch1", "call1")
    await nextTick()

    expect(wrapper.get('[data-test="edge-inspector"]').attributes("data-source-kind")).toBe("branch")
    expect(wrapper.get('[data-test="edge-inspector"]').attributes("data-edge-case")).toBe("approved")
  })

  it("switches the canvas into foreach body editor mode from the inspector", async () => {
    const flowStore = useFlowStore()

    projectsStore.getProjectByID.mockReturnValue({
      id: "project-1",
      flowId: "project-1",
      name: "Project 1",
      updatedAt: "2026-03-25T12:00:00.000Z",
      graph: {
        nodes: [
          {
            id: "foreach1",
            kind: "foreach",
            allow_fail: false,
            retry: 1,
            timeout_ms: 3000,
            spec: {
              source: { kind: "trigger", path: "/items" },
              required: true,
              body: {
                nodes: [
                  {
                    id: "inner1",
                    kind: "call",
                    allow_fail: false,
                    retry: 1,
                    timeout_ms: 3000,
                    spec: {
                      method: "demo::inner",
                      args_template: {},
                      _ui: { x: 10, y: 20 }
                    }
                  }
                ],
                edges: []
              },
              result_node_id: "inner1",
              _ui: { x: 0, y: 0 }
            }
          },
          {
            id: "call1",
            kind: "call",
            allow_fail: false,
            retry: 1,
            timeout_ms: 3000,
            spec: {
              method: "demo::existing",
              args_template: {}
            }
          }
        ],
        edges: [{ from: "foreach1", to: "call1" }]
      }
    })

    const wrapper = mount(FlowEditorWindow, {
      global: {
        stubs: {
          FlowEditorToolbar: FlowEditorToolbarStub,
          FlowCanvas: FlowCanvasStub,
          FlowNodeInspector: FlowNodeInspectorStub,
          FlowEdgeInspector: FlowEdgeInspectorStub,
          FlowMethodPickerDialog: FlowMethodPickerDialogStub,
          FlowFieldBindingDialog: SimpleStub,
          FlowAddNodeDialog: SimpleStub
        }
      }
    })

    await flushAsync()

    expect(wrapper.get('[data-test="canvas"]').attributes("data-node-count")).toBe("2")

    flowStore.selectNodeById("foreach1")
    await nextTick()
    await wrapper.get('[data-test="open-body"]').trigger("click")
    await flushAsync()

    expect(wrapper.get('[data-test="canvas"]').attributes("data-node-count")).toBe("1")
    expect(wrapper.text()).toContain("Foreach Body Editor")
  })

  it("renders body call visual form and opens the method picker from the nested inspector", async () => {
    const flowStore = useFlowStore()

    projectsStore.getProjectByID.mockReturnValue({
      id: "project-1",
      flowId: "project-1",
      name: "Project 1",
      updatedAt: "2026-03-25T12:00:00.000Z",
      graph: {
        nodes: [
          {
            id: "foreach1",
            kind: "foreach",
            allow_fail: false,
            retry: 1,
            timeout_ms: 3000,
            spec: {
              source: { kind: "trigger", path: "/items" },
              required: true,
              body: {
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
                        name: "Alice"
                      },
                      _ui: { x: 10, y: 20 }
                    }
                  }
                ],
                edges: []
              },
              result_node_id: "inner1",
              _ui: { x: 0, y: 0 }
            }
          }
        ],
        edges: []
      }
    })

    const wrapper = mount(FlowEditorWindow, {
      global: {
        stubs: {
          FlowEditorToolbar: FlowEditorToolbarStub,
          FlowCanvas: FlowCanvasStub,
          FlowNodeInspector: FlowNodeInspectorStub,
          FlowEdgeInspector: FlowEdgeInspectorStub,
          FlowMethodPickerDialog: FlowMethodPickerDialogStub,
          FlowFieldBindingDialog: SimpleStub,
          FlowAddNodeDialog: SimpleStub
        }
      }
    })

    await flushAsync()

    flowStore.state.execCapabilities = [
      {
        key: "100|0|demo::inner|v1",
        providerNode: 100,
        viaNode: 0,
        method: "demo::inner",
        version: "v1",
        defaultTimeoutMs: 0,
        permissions: [],
        tags: {},
        inputSchema: {
          type: "object",
          properties: {
            name: {
              type: "string",
              title: "Name"
            }
          },
          required: ["name"]
        },
        outputSchema: null,
        label: "100 · demo::inner@v1"
      }
    ]

    flowStore.selectNodeById("foreach1")
    await nextTick()
    await wrapper.get('[data-test="open-body"]').trigger("click")
    await flushAsync()
    await wrapper.get('[data-test="select-first-node"]').trigger("click")
    await flushAsync()

    expect(wrapper.text()).toContain("Call Method")
    expect(wrapper.text()).toContain("Method Fields")

    const methodButton = wrapper.findAll("*").find((node) => node.text().trim() === "Select Method")
    expect(methodButton).toBeTruthy()
    await methodButton!.trigger("click")
    await flushAsync()

    const dialog = wrapper.get('[data-test="method-dialog"]')
    expect(dialog.attributes("data-open")).toBe("true")
    expect(dialog.attributes("data-method-search")).toBe("")
  })

  it("adds bindings to non-call body nodes and syncs them back to the parent foreach JSON", async () => {
    const flowStore = useFlowStore()

    projectsStore.getProjectByID.mockReturnValue({
      id: "project-1",
      flowId: "project-1",
      name: "Project 1",
      updatedAt: "2026-03-25T12:00:00.000Z",
      graph: {
        nodes: [
          {
            id: "foreach1",
            kind: "foreach",
            allow_fail: false,
            retry: 1,
            timeout_ms: 3000,
            spec: {
              source: { kind: "trigger", path: "/items" },
              required: true,
              body: {
                nodes: [
                  {
                    id: "inner_set",
                    kind: "set_var",
                    allow_fail: false,
                    retry: 1,
                    timeout_ms: 3000,
                    spec: {
                      name: "session_payload",
                      template: {},
                      inputs: [],
                      _ui: { x: 10, y: 20 }
                    }
                  }
                ],
                edges: []
              },
              result_node_id: "inner_set",
              _ui: { x: 0, y: 0 }
            }
          }
        ],
        edges: []
      }
    })

    const wrapper = mount(FlowEditorWindow, {
      global: {
        stubs: {
          FlowEditorToolbar: FlowEditorToolbarStub,
          FlowCanvas: FlowCanvasStub,
          FlowNodeInspector: FlowNodeInspectorStub,
          FlowEdgeInspector: FlowEdgeInspectorStub,
          FlowMethodPickerDialog: FlowMethodPickerDialogStub,
          FlowFieldBindingDialog: SimpleStub,
          FlowAddNodeDialog: SimpleStub
        }
      }
    })

    await flushAsync()

    flowStore.selectNodeById("foreach1")
    await nextTick()
    await wrapper.get('[data-test="open-body"]').trigger("click")
    await flushAsync()
    await wrapper.get('[data-test="select-first-node"]').trigger("click")
    await flushAsync()

    expect(wrapper.text()).toContain("Body node authoring")
    expect(wrapper.text()).toContain("Set Var Node")

    const addBindingButton = wrapper
      .findAll("button")
      .find((button) => button.text().trim() === "Add Binding")
    expect(addBindingButton).toBeTruthy()
    await addBindingButton!.trigger("click")
    await flushAsync()

    const foreachNode = flowStore.state.nodes.find((node) => node.id === "foreach1")
    const bodyGraph = JSON.parse(foreachNode?.foreachBodyJson ?? "{}")
    expect(bodyGraph.nodes[0].spec.inputs).toHaveLength(1)
  })

  it("blocks saving when local project graphs form a recursive subflow chain", async () => {
    const flowA = "550e8400-e29b-41d4-a716-446655440000"
    const flowB = "123e4567-e89b-12d3-a456-426614174000"

    projectsStore.state.projects = [
      {
        projectId: "project-1",
        flowId: flowA,
        graph: {
          nodes: [
            {
              id: "sub1",
              kind: "subflow",
              allow_fail: false,
              retry: 1,
              timeout_ms: 3000,
              spec: {
                flow_id: flowB,
                input_template: {},
                _ui: { x: 0, y: 0 }
              }
            }
          ],
          edges: []
        }
      },
      {
        projectId: "project-2",
        flowId: flowB,
        graph: {
          nodes: [
            {
              id: "sub2",
              kind: "subflow",
              allow_fail: false,
              retry: 1,
              timeout_ms: 3000,
              spec: {
                flow_id: flowA,
                input_template: {},
                _ui: { x: 0, y: 0 }
              }
            }
          ],
          edges: []
        }
      }
    ]
    projectsStore.getProjectByID.mockReturnValue({
      id: "project-1",
      flowId: flowA,
      name: "Project 1",
      updatedAt: "2026-03-25T12:00:00.000Z",
      graph: projectsStore.state.projects[0].graph
    })

    const wrapper = mount(FlowEditorWindow, {
      global: {
        stubs: {
          FlowEditorToolbar: FlowEditorToolbarStub,
          FlowCanvas: FlowCanvasStub,
          FlowNodeInspector: FlowNodeInspectorStub,
          FlowEdgeInspector: FlowEdgeInspectorStub,
          FlowMethodPickerDialog: FlowMethodPickerDialogStub,
          FlowFieldBindingDialog: SimpleStub,
          FlowAddNodeDialog: SimpleStub
        }
      }
    })

    await flushAsync()

    await wrapper.get('[data-test="save-project"]').trigger("click")
    await flushAsync()

    expect(projectsStore.saveProjectGraph).not.toHaveBeenCalled()
    expect(toastStore.errorOf).toHaveBeenCalled()
  })

  it("saves body editor graph changes through the root project graph export", async () => {
    const flowStore = useFlowStore()

    projectsStore.getProjectByID.mockReturnValue({
      id: "project-1",
      flowId: "project-1",
      name: "Project 1",
      updatedAt: "2026-03-25T12:00:00.000Z",
      graph: {
        nodes: [
          {
            id: "foreach1",
            kind: "foreach",
            allow_fail: false,
            retry: 1,
            timeout_ms: 3000,
            spec: {
              source: { kind: "trigger", path: "/items" },
              required: true,
              body: {
                nodes: [
                  {
                    id: "inner1",
                    kind: "call",
                    allow_fail: false,
                    retry: 1,
                    timeout_ms: 3000,
                    spec: {
                      method: "demo::inner",
                      args_template: {},
                      _ui: { x: 10, y: 20 }
                    }
                  }
                ],
                edges: []
              },
              result_node_id: "inner1",
              _ui: { x: 0, y: 0 }
            }
          }
        ],
        edges: []
      }
    })

    const wrapper = mount(FlowEditorWindow, {
      global: {
        stubs: {
          FlowEditorToolbar: FlowEditorToolbarStub,
          FlowCanvas: FlowCanvasStub,
          FlowNodeInspector: FlowNodeInspectorStub,
          FlowEdgeInspector: FlowEdgeInspectorStub,
          FlowMethodPickerDialog: FlowMethodPickerDialogStub,
          FlowFieldBindingDialog: SimpleStub,
          FlowAddNodeDialog: SimpleStub
        }
      }
    })

    await flushAsync()

    flowStore.selectNodeById("foreach1")
    await nextTick()
    await wrapper.get('[data-test="open-body"]').trigger("click")
    await flushAsync()

    await wrapper.get('[data-test="move-node"]').trigger("click")
    await flushAsync()

    await wrapper.get('[data-test="save-project"]').trigger("click")
    await flushAsync()

    expect(projectsStore.saveProjectGraph).toHaveBeenCalled()
    const savedGraph = projectsStore.saveProjectGraph.mock.calls.at(-1)?.[1]
    const foreachNode = savedGraph.nodes.find((node: any) => node.id === "foreach1")

    expect(foreachNode.spec.body.nodes[0].spec._ui).toEqual({ x: 321, y: 654 })
  })
})
