// @vitest-environment jsdom

import { defineComponent, nextTick, reactive } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { setLocale } from "@/i18n"

const authorityState = reactive({
  authorityId: 11,
  resolving: false
})

const accessPolicyState = reactive({
  loading: false,
  saving: false,
  policy: {
    defaultRole: "admin",
    defaultPerms: ["file.read", "custom.scope"],
    nodeRoles: [{ nodeId: 3, role: "node" }],
    rolePerms: [
      { role: "admin", perms: ["*"] },
      { role: "observer", perms: ["exec.cap.query", "custom.observe"] }
    ]
  },
  runtime: [],
  runtimeTotal: 0,
  runtimeError: "",
  warnings: [],
  lastSave: undefined as unknown
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

const accessPolicyStore = {
  state: accessPolicyState,
  loadPolicy: vi.fn(async () => undefined),
  savePolicy: vi.fn(async () => ({
    authorityId: 11,
    runtimeError: "",
    runtime: [],
    runtimeTotal: 0
  })),
  getNodePerms: vi.fn(async () => ({
    nodeId: 3,
    role: "node",
    perms: ["file.read"]
  })),
  reset: vi.fn(() => {
    accessPolicyState.loading = false
    accessPolicyState.saving = false
    accessPolicyState.policy.defaultRole = "node"
    accessPolicyState.policy.defaultPerms = []
    accessPolicyState.policy.nodeRoles = []
    accessPolicyState.policy.rolePerms = []
    accessPolicyState.runtime = []
    accessPolicyState.runtimeTotal = 0
    accessPolicyState.runtimeError = ""
    accessPolicyState.warnings = []
    accessPolicyState.lastSave = undefined
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

vi.mock("@/stores/accessPolicy", () => ({
  useAccessPolicyStore: () => accessPolicyStore
}))

vi.mock("@/stores/toast", () => ({
  useToastStore: () => toastStore
}))

import AccessPolicy from "./AccessPolicy.vue"

const PageHeroStub = defineComponent({
  props: {
    title: { type: String, default: "" },
    description: { type: String, default: "" }
  },
  template: `
    <section>
      <h1>{{ title }}</h1>
      <p>{{ description }}</p>
      <slot name="actions" />
    </section>
  `
})

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
  emits: ["click"],
  template: `<button type="button" @click="$emit('click', $event)"><slot /></button>`
})

const BadgeStub = defineComponent({
  template: `<span><slot /></span>`
})

const OverlayStub = defineComponent({
  props: {
    open: { type: Boolean, default: false }
  },
  template: `<div v-if="open"><slot /></div>`
})

const seedPolicy = () => {
  accessPolicyState.policy.defaultRole = "admin"
  accessPolicyState.policy.defaultPerms = ["file.read", "custom.scope"]
  accessPolicyState.policy.nodeRoles = [{ nodeId: 3, role: "node" }]
  accessPolicyState.policy.rolePerms = [
    { role: "admin", perms: ["*"] },
    { role: "observer", perms: ["exec.cap.query", "custom.observe"] }
  ]
}

describe("AccessPolicy", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setLocale("zh-CN")
    authorityState.authorityId = 11
    authorityState.resolving = false
    accessPolicyState.loading = false
    accessPolicyState.saving = false
    accessPolicyState.runtime = []
    accessPolicyState.runtimeTotal = 0
    accessPolicyState.runtimeError = ""
    accessPolicyState.warnings = []
    accessPolicyState.lastSave = undefined
    seedPolicy()
    accessPolicyStore.loadPolicy.mockImplementation(async () => {
      seedPolicy()
    })
  })

  it("uses the updated access policy naming and tab layout", async () => {
    const wrapper = mount(AccessPolicy, {
      global: {
        stubs: {
          PageHero: PageHeroStub,
          CardHeader: CardHeaderStub,
          Button: ButtonStub,
          Badge: BadgeStub,
          Overlay: OverlayStub
        }
      }
    })

    await Promise.resolve()
    await nextTick()

    expect(wrapper.text()).toContain("访问策略")
    expect(wrapper.text()).toContain("当前策略")
    expect(wrapper.text()).not.toContain("权限编排")
    expect(wrapper.text()).toContain("编辑默认准入")
    expect(wrapper.text()).toContain("操作面板")
    expect(wrapper.text()).toContain("还没有查询任何节点")
    expect(wrapper.text()).toContain("custom.scope")

    const runtimeToggleButton = wrapper
      .findAll("button")
      .find((button) => button.text() === "显示运行时详情")

    expect(runtimeToggleButton).toBeTruthy()
    await runtimeToggleButton!.trigger("click")
    await nextTick()

    expect(wrapper.text()).toContain("暂无运行时条目")

    const editDefaultButton = wrapper
      .findAll("button")
      .find((button) => button.text() === "编辑默认准入")

    expect(editDefaultButton).toBeTruthy()
    await editDefaultButton!.trigger("click")
    await nextTick()

    expect(wrapper.text()).toContain("权限列表")
    expect(wrapper.text()).toContain("保留的额外权限")
  })

  it("renders node overrides as a compact list and opens the override dialog", async () => {
    const wrapper = mount(AccessPolicy, {
      global: {
        stubs: {
          PageHero: PageHeroStub,
          CardHeader: CardHeaderStub,
          Button: ButtonStub,
          Badge: BadgeStub,
          Overlay: OverlayStub
        }
      }
    })

    await Promise.resolve()
    await nextTick()

    expect(wrapper.text()).toContain("节点 3")
    expect(wrapper.text()).toContain("角色：node")

    const editButtons = wrapper
      .findAll("button")
      .filter((button) => button.text() === "编辑")

    expect(editButtons.length).toBeGreaterThan(0)
    await editButtons[0].trigger("click")
    await nextTick()

    expect(wrapper.text()).toContain("编辑节点覆盖")
    expect(wrapper.text()).toContain("节点 ID")
  })

  it("shows the dedicated role list and opens the role editor dialog", async () => {
    const wrapper = mount(AccessPolicy, {
      global: {
        stubs: {
          PageHero: PageHeroStub,
          CardHeader: CardHeaderStub,
          Button: ButtonStub,
          Badge: BadgeStub,
          Overlay: OverlayStub
        }
      }
    })

    await Promise.resolve()
    await nextTick()

    const roleTab = wrapper
      .findAll("button")
      .find((button) => button.text() === "角色管理")

    expect(roleTab).toBeTruthy()
    await roleTab!.trigger("click")
    await nextTick()

    expect(wrapper.text()).toContain("observer")
    expect(wrapper.text()).toContain("custom.observe")

    const editButtons = wrapper
      .findAll("button")
      .filter((button) => button.text() === "编辑")

    expect(editButtons).toHaveLength(2)
    await editButtons[1].trigger("click")
    await nextTick()

    expect(wrapper.text()).toContain("编辑角色")
    expect(wrapper.text()).toContain("custom.observe")
  })
})
