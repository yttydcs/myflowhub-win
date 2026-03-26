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
          Badge: BadgeStub
        }
      }
    })

    await Promise.resolve()
    await nextTick()

    expect(wrapper.text()).toContain("访问策略")
    expect(wrapper.text()).toContain("当前策略")
    expect(wrapper.text()).not.toContain("权限编排")
    expect(wrapper.findAll("textarea")).toHaveLength(0)
    expect(wrapper.text()).toContain("custom.scope")
  })

  it("shows the dedicated role management tab with preserved extra permissions", async () => {
    const wrapper = mount(AccessPolicy, {
      global: {
        stubs: {
          PageHero: PageHeroStub,
          CardHeader: CardHeaderStub,
          Button: ButtonStub,
          Badge: BadgeStub
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

    expect(wrapper.text()).toContain("内置角色预设")
    expect(wrapper.text()).toContain("observer")
    expect(wrapper.text()).toContain("custom.observe")
  })
})
