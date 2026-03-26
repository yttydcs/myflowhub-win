// @vitest-environment jsdom

import { defineComponent, nextTick } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { setLocale } from "@/i18n"

const projectsStore = {
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
      statusFlow
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
  emits: ["run-flow", "refresh-status"],
  template: `
    <div data-test="toolbar" :data-flow-status-label="flowStatusLabel" :data-current-run-id-label="currentRunIdLabel">
      <button data-test="run-flow" type="button" @click="$emit('run-flow')">Run</button>
      <button data-test="refresh-status" type="button" @click="$emit('refresh-status')">Refresh</button>
    </div>
  `
})

const FlowCanvasStub = defineComponent({
  props: {
    statusNodes: { type: Array, default: () => [] }
  },
  template: `<div data-test="canvas" :data-status-count="String(statusNodes.length)" />`
})

const FlowNodeInspectorStub = defineComponent({
  emits: ["open-method"],
  template: `<button data-test="open-method" type="button" @click="$emit('open-method')">Open Method</button>`
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
    queryExecCapabilities.mockResolvedValue(undefined)
    ensureNodeCapabilityLoaded.mockResolvedValue(false)
    runFlow.mockResolvedValue(undefined)
    statusFlow.mockResolvedValue(undefined)
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
})
