// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from "vitest"
import { setLocale } from "@/i18n"

const sessionStore = {
  connected: true,
  auth: {
    nodeId: 7,
    hubId: 9
  }
}

const flowProjectsState = vi.fn()
const saveFlowProjectsState = vi.fn()
const listSimple = vi.fn()
const setSimple = vi.fn()

let persistedProjects: any[] = []

const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value))

vi.mock("@/stores/flow", () => ({
  flowEventModeLabelKey: (value: string) => value,
  flowStatusLabelKey: (value: string) => value,
  flowTriggerTypeLabelKey: (value: string) => value
}))

vi.mock("@/stores/session", () => ({
  useSessionStore: () => sessionStore
}))

import { useFlowProjectsStore, type FlowProjectRecord, type FlowTriggerDraft } from "./flowProjects"

const store = useFlowProjectsStore()
const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

const createTrigger = (): FlowTriggerDraft => ({
  type: "interval",
  everyMs: 60000,
  cronExpr: "",
  eventMode: "publish",
  eventName: "",
  eventTopic: "",
  varOwner: 0,
  varName: ""
})

const createProjectRecord = (overrides: Partial<FlowProjectRecord> = {}): FlowProjectRecord => ({
  projectId: "project-1",
  flowId: "550e8400-e29b-41d4-a716-446655440000",
  name: "Demo Project",
  trigger: createTrigger(),
  graph: {
    nodes: [],
    edges: []
  },
  updatedAt: "2026-03-27T12:00:00.000Z",
  ...overrides
})

beforeEach(() => {
  setLocale("en")
  persistedProjects = []
  flowProjectsState.mockReset()
  saveFlowProjectsState.mockReset()
  listSimple.mockReset()
  setSimple.mockReset()

  flowProjectsState.mockImplementation(async () => ({
    projects: clone(persistedProjects)
  }))
  saveFlowProjectsState.mockImplementation(async (payload: any) => {
    persistedProjects = Array.isArray(payload?.projects) ? clone(payload.projects) : []
    return {
      projects: clone(persistedProjects)
    }
  })

  store.state.projects = []
  store.state.loading = false
  store.state.saving = false
  store.state.deployments = []
  store.state.deploymentsLoading = false
  store.state.deploymentsNodeId = ""

  ;(window as any).go = {
    main: {
      App: {
        FlowProjectsState: flowProjectsState,
        SaveFlowProjectsState: saveFlowProjectsState
      }
    },
    flow: {
      FlowService: {
        ListSimple: listSimple,
        SetSimple: setSimple
      }
    }
  }
})

describe("flowProjects store", () => {
  it("generates a UUID flowId for new projects", async () => {
    const project = await store.createProject({ name: "Fresh Project" })

    expect(project.flowId).toMatch(uuidPattern)
    expect(project.flowId.startsWith("fl_")).toBe(false)
    expect(persistedProjects).toHaveLength(1)
    expect(String(persistedProjects[0]?.flow_id ?? "")).toBe(project.flowId)
  })

  it("rejects non-UUID flowIds when updating project metadata", async () => {
    persistedProjects = [
      {
        project_id: "project-1",
        flow_id: "550e8400-e29b-41d4-a716-446655440000",
        name: "Demo Project",
        trigger: { type: "interval", every_ms: 60000 },
        graph: { nodes: [], edges: [] },
        updated_at: "2026-03-27T12:00:00.000Z"
      }
    ]

    await expect(
      store.updateProjectMeta({
        projectId: "project-1",
        flowId: "fl_legacy_bad_id",
        name: "Updated Name"
      })
    ).rejects.toThrow("Flow ID must be a UUID.")

    expect(saveFlowProjectsState).not.toHaveBeenCalled()
  })

  it("rejects non-UUID flowIds when saving payload metadata", async () => {
    persistedProjects = [
      {
        project_id: "project-1",
        flow_id: "550e8400-e29b-41d4-a716-446655440000",
        name: "Demo Project",
        trigger: { type: "interval", every_ms: 60000 },
        graph: { nodes: [], edges: [] },
        updated_at: "2026-03-27T12:00:00.000Z"
      }
    ]

    await expect(
      store.saveProjectPayload("project-1", {
        flow_id: "fl_payload_bad_id",
        name: "Payload Name",
        trigger: { type: "interval", every_ms: 60000 },
        graph: { nodes: [], edges: [] }
      } as any)
    ).rejects.toThrow("Flow ID must be a UUID.")

    expect(saveFlowProjectsState).not.toHaveBeenCalled()
  })

  it("fails legacy non-UUID projects locally before deploy requests", async () => {
    store.state.projects = [
      createProjectRecord({
        flowId: "fl_legacy_bad_id",
        graph: {
          nodes: [{ id: "node-1", kind: "call", spec: { method: "demo::call", args_template: {} } }],
          edges: []
        }
      })
    ]

    await expect(
      store.deployProject({
        projectId: "project-1",
        nodeId: "12",
        trigger: createTrigger(),
        overwrite: false
      })
    ).rejects.toThrow("Flow ID must be a UUID.")

    expect(listSimple).not.toHaveBeenCalled()
    expect(setSimple).not.toHaveBeenCalled()
  })

  it("normalizes cron triggers and deploys them with the cron wire", async () => {
    persistedProjects = [
      {
        project_id: "project-1",
        flow_id: "550e8400-e29b-41d4-a716-446655440000",
        name: "Cron Project",
        trigger: { type: "cron", cron: "0 */5 * * *" },
        graph: {
          nodes: [{ id: "node-1", kind: "call", spec: { method: "demo::call", args_template: {} } }],
          edges: []
        },
        updated_at: "2026-03-27T12:00:00.000Z"
      }
    ]

    await store.loadProjects()

    expect(store.state.projects[0]?.trigger).toEqual({
      type: "cron",
      everyMs: 60000,
      cronExpr: "0 */5 * * *",
      eventMode: "publish",
      eventName: "",
      eventTopic: "",
      varOwner: 0,
      varName: ""
    })

    listSimple.mockResolvedValue({ code: 1, flows: [] })
    setSimple.mockResolvedValue({ code: 1 })

    await expect(
      store.deployProject({
        projectId: "project-1",
        nodeId: "12",
        trigger: {
          ...createTrigger(),
          type: "cron",
          cronExpr: "0 */5 * * *"
        },
        overwrite: false
      })
    ).resolves.toEqual({ overwriteRequired: false })

    expect(setSimple).toHaveBeenCalledWith(
      7,
      9,
      expect.objectContaining({
        executor_node: 12,
        flow_id: "550e8400-e29b-41d4-a716-446655440000",
        trigger: { type: "cron", cron: "0 */5 * * *" }
      })
    )
    expect(store.state.projects[0]?.trigger).toMatchObject({
      type: "cron",
      cronExpr: "0 */5 * * *"
    })
    expect(persistedProjects[0]?.trigger).toEqual({ type: "cron", cron: "0 */5 * * *" })
  })
})
