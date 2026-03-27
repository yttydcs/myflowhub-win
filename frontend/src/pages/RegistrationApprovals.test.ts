// @vitest-environment jsdom

import { defineComponent, nextTick, reactive } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { setLocale } from "@/i18n"

const authorityState = reactive({
  authorityId: 11,
  resolving: false
})

const approvalsState = reactive({
  loading: false,
  busyRequestId: "",
  filterDeviceId: "",
  total: 0,
  items: [] as Array<{
    requestId: string
    deviceId: string
    requestedRole: string
    displayName: string
    createdAt: number
    expiresAt: number
  }>,
  lastDecision: undefined as
    | undefined
    | { action: "approve" | "reject"; requestId: string; deviceId: string; nodeId?: number }
})

const sessionStore = {
  connected: true,
  auth: {
    loggedIn: true,
    nodeId: 7,
    hubId: 9
  }
}

const authorityStore = {
  state: authorityState,
  resolveAuthority: vi.fn(async () => authorityState.authorityId),
  setIdentity: vi.fn()
}

const seedItems = () => [
  {
    requestId: "req-1",
    deviceId: "device-1",
    requestedRole: "node",
    displayName: "Node A",
    createdAt: 1710000000,
    expiresAt: 1710003600
  },
  {
    requestId: "req-2",
    deviceId: "device-2",
    requestedRole: "",
    displayName: "",
    createdAt: 1710007200,
    expiresAt: 1710010800
  }
]

const applySeededApprovals = () => {
  const nextItems = seedItems()
  approvalsState.items = nextItems
  approvalsState.total = nextItems.length
}

const approvalsStore = {
  state: approvalsState,
  loadPending: vi.fn(async () => {
    applySeededApprovals()
  }),
  approveRegister: vi.fn(async (requestId: string, role: string) => {
    approvalsState.busyRequestId = requestId
    const target = approvalsState.items.find((item) => item.requestId === requestId)
    approvalsState.items = approvalsState.items.filter((item) => item.requestId !== requestId)
    approvalsState.total = approvalsState.items.length
    approvalsState.lastDecision = {
      action: "approve",
      requestId,
      deviceId: target?.deviceId || "",
      nodeId: role ? 21 : 0
    }
    approvalsState.busyRequestId = ""
    return { nodeId: role ? 21 : 0 }
  }),
  rejectRegister: vi.fn(async (requestId: string) => {
    approvalsState.busyRequestId = requestId
    const target = approvalsState.items.find((item) => item.requestId === requestId)
    approvalsState.items = approvalsState.items.filter((item) => item.requestId !== requestId)
    approvalsState.total = approvalsState.items.length
    approvalsState.lastDecision = {
      action: "reject",
      requestId,
      deviceId: target?.deviceId || ""
    }
    approvalsState.busyRequestId = ""
    return undefined
  }),
  reset: vi.fn(() => {
    approvalsState.loading = false
    approvalsState.busyRequestId = ""
    approvalsState.filterDeviceId = ""
    approvalsState.total = 0
    approvalsState.items = []
    approvalsState.lastDecision = undefined
  })
}

const toastStore = {
  success: vi.fn(),
  warn: vi.fn(),
  error: vi.fn(),
  errorOf: vi.fn(),
  info: vi.fn()
}

vi.mock("@/stores/session", () => ({
  useSessionStore: () => sessionStore
}))

vi.mock("@/stores/authority", () => ({
  useAuthorityStore: () => authorityStore
}))

vi.mock("@/stores/registrationApprovals", () => ({
  useRegistrationApprovalsStore: () => approvalsStore
}))

vi.mock("@/stores/toast", () => ({
  useToastStore: () => toastStore
}))

import RegistrationApprovals from "./RegistrationApprovals.vue"

const CardHeaderStub = defineComponent({
  props: {
    title: { type: String, default: "" },
    description: { type: String, default: "" }
  },
  template: `
    <section>
      <h2>{{ title }}</h2>
      <p v-if="description">{{ description }}</p>
      <slot name="actions" />
    </section>
  `
})

const ButtonStub = defineComponent({
  inheritAttrs: false,
  emits: ["click"],
  template: `<button type="button" v-bind="$attrs" @click="$emit('click', $event)"><slot /></button>`
})

const BadgeStub = defineComponent({
  inheritAttrs: false,
  template: `<span v-bind="$attrs"><slot /></span>`
})

const OverlayStub = defineComponent({
  props: {
    open: { type: Boolean, default: false }
  },
  template: `<div v-if="open"><slot /></div>`
})

const mountPage = () =>
  mount(RegistrationApprovals, {
    global: {
      stubs: {
        CardHeader: CardHeaderStub,
        Button: ButtonStub,
        Badge: BadgeStub,
        Overlay: OverlayStub
      }
    }
  })

describe("RegistrationApprovals", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setLocale("zh-CN")
    authorityState.authorityId = 11
    authorityState.resolving = false
    approvalsState.loading = false
    approvalsState.busyRequestId = ""
    approvalsState.filterDeviceId = ""
    applySeededApprovals()
    approvalsState.lastDecision = undefined
  })

  it("renders a compact queue and opens the review dialog on demand", async () => {
    const wrapper = mountPage()

    await Promise.resolve()
    await nextTick()

    expect(wrapper.text()).toContain("注册审批")
    expect(wrapper.findAll("[data-approval-row]")).toHaveLength(2)
    expect(wrapper.text()).not.toContain("批准请求")
    expect(wrapper.text()).not.toContain("拒绝请求")
    expect(wrapper.find("[data-approval-refresh]").exists()).toBe(true)
    expect(wrapper.findAll("button").some((button) => button.text() === "解析")).toBe(false)

    const reviewButtons = wrapper.findAll("[data-approval-review-open]")
    expect(reviewButtons).toHaveLength(2)

    await reviewButtons[0].trigger("click")
    await nextTick()

    expect(wrapper.find("[data-approval-review-dialog]").exists()).toBe(true)
    expect(wrapper.text()).toContain("审阅请求")
    expect(wrapper.text()).toContain("Node A")
  })

  it("keeps a single refresh entry point in the queue header", async () => {
    const wrapper = mountPage()

    await Promise.resolve()
    await nextTick()

    approvalsStore.loadPending.mockClear()

    await wrapper.get("[data-approval-refresh]").trigger("click")
    await Promise.resolve()
    await nextTick()

    expect(approvalsStore.loadPending).toHaveBeenCalledTimes(1)
    expect(wrapper.findAll("button").filter((button) => button.text() === "刷新")).toHaveLength(1)
  })

  it("aligns the filter apply button height with the device filter input", async () => {
    const wrapper = mountPage()

    await Promise.resolve()
    await nextTick()

    const filterButton = wrapper.get("[data-approval-filter-apply]")

    expect(filterButton.classes()).toContain("h-10")
    await filterButton.trigger("click")
    await Promise.resolve()
    await nextTick()

    expect(approvalsStore.loadPending).toHaveBeenCalled()
  })

  it("approves the selected request from the review dialog", async () => {
    const wrapper = mountPage()

    await Promise.resolve()
    await nextTick()

    await wrapper.findAll("[data-approval-review-open]")[0].trigger("click")
    await nextTick()

    await wrapper.get("[data-review-role-input]").setValue("admin")
    await wrapper.get("[data-review-approve]").trigger("click")
    await Promise.resolve()
    await nextTick()

    expect(approvalsStore.approveRegister).toHaveBeenCalledWith("req-1", "admin")
    expect(wrapper.find("[data-approval-review-dialog]").exists()).toBe(false)
    expect(approvalsState.lastDecision).toMatchObject({
      action: "approve",
      requestId: "req-1"
    })
  })

  it("rejects the selected request from the same focused dialog", async () => {
    const wrapper = mountPage()

    await Promise.resolve()
    await nextTick()

    await wrapper.findAll("[data-approval-review-open]")[1].trigger("click")
    await nextTick()

    await wrapper.get("[data-review-reason-input]").setValue("not expected")
    await wrapper.get("[data-review-reject]").trigger("click")
    await Promise.resolve()
    await nextTick()

    expect(approvalsStore.rejectRegister).toHaveBeenCalledWith("req-2", "not expected")
    expect(wrapper.find("[data-approval-review-dialog]").exists()).toBe(false)
    expect(approvalsState.lastDecision).toMatchObject({
      action: "reject",
      requestId: "req-2"
    })
  })
})
