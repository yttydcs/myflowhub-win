// @vitest-environment jsdom

import { defineComponent, nextTick, reactive } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { setLocale } from "@/i18n"

const authorityState = reactive({
  authorityId: 11,
  resolving: false
})

const permitState = reactive({
  loading: false,
  issuing: false,
  busyPermit: "",
  total: 0,
  items: [] as Array<{
    permit: string
    deviceId: string
    role: string
    issuedBy: number
    issuedAt: number
    expiresAt: number
  }>
})

let permitRows: Array<{
  permit: string
  deviceId: string
  role: string
  issuedBy: number
  issuedAt: number
  expiresAt: number
}> = []

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

const permitStore = {
  state: permitState,
  loadPermits: vi.fn(async () => {
    permitState.loading = true
    permitState.items = permitRows.map((item) => ({ ...item }))
    permitState.total = permitRows.length
    permitState.loading = false
    return permitState.items
  }),
  issuePermit: vi.fn(async (input: { deviceId: string; role: string; expiresAt: number }) => {
    permitState.issuing = true
    const issued = {
      permit: "permit_456",
      deviceId: input.deviceId,
      role: input.role,
      expiresAt: input.expiresAt
    }
    permitRows = [
      {
        permit: issued.permit,
        deviceId: issued.deviceId,
        role: issued.role,
        issuedBy: 9,
        issuedAt: Math.floor(Date.parse("2026-03-27T10:30:00Z") / 1000),
        expiresAt: issued.expiresAt
      },
      ...permitRows
    ]
    permitState.issuing = false
    return issued
  }),
  revokePermit: vi.fn(async (permit: string) => {
    permitState.busyPermit = permit
    const removed = permitRows.find((item) => item.permit === permit)
    permitRows = permitRows.filter((item) => item.permit !== permit)
    permitState.busyPermit = ""
    return {
      permit,
      deviceId: removed?.deviceId || "",
      role: removed?.role || ""
    }
  }),
  reset: vi.fn(() => {
    permitState.loading = false
    permitState.issuing = false
    permitState.busyPermit = ""
    permitState.total = 0
    permitState.items = []
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

vi.mock("@/stores/permitIssuance", () => ({
  usePermitIssuanceStore: () => permitStore
}))

vi.mock("@/stores/toast", () => ({
  useToastStore: () => toastStore
}))

import PermitIssuance from "./PermitIssuance.vue"

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
  mount(PermitIssuance, {
    global: {
      stubs: {
        CardHeader: CardHeaderStub,
        Button: ButtonStub,
        Badge: BadgeStub,
        Overlay: OverlayStub
      }
    }
  })

describe("PermitIssuance", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setLocale("zh-CN")
    authorityState.authorityId = 11
    authorityState.resolving = false
    permitRows = [
      {
        permit: "permit_123",
        deviceId: "device-1",
        role: "admin",
        issuedBy: 9,
        issuedAt: Math.floor(Date.parse("2026-03-27T10:00:00Z") / 1000),
        expiresAt: Math.floor(Date.parse("2026-03-27T11:00:00Z") / 1000)
      }
    ]
    permitState.loading = false
    permitState.issuing = false
    permitState.busyPermit = ""
    permitState.total = 0
    permitState.items = []
  })

  it("loads active permits automatically and renders the compact list layout", async () => {
    const wrapper = mountPage()

    await Promise.resolve()
    await nextTick()

    expect(permitStore.loadPermits).toHaveBeenCalledTimes(1)
    expect(wrapper.find("[data-permit-card-actions] [data-refresh-permits]").exists()).toBe(true)
    expect(wrapper.find("[data-permit-card-actions] [data-open-issue-dialog]").exists()).toBe(true)
    expect(wrapper.find("[data-permit-list]").exists()).toBe(true)
    expect(wrapper.findAll("[data-permit-row]")).toHaveLength(1)
    expect(wrapper.text()).not.toContain("Latest Permit")
    expect(wrapper.text()).not.toContain("Use Latest Permit")
    expect(wrapper.text()).not.toContain("共 1 条")
    expect(wrapper.text()).not.toContain("实时")
  })

  it("shows inline load errors without raising a toast on auto load", async () => {
    permitStore.loadPermits.mockRejectedValueOnce(new Error("auth list_register_permits: request timed out"))

    const wrapper = mountPage()

    await Promise.resolve()
    await nextTick()

    expect(permitStore.loadPermits).toHaveBeenCalledTimes(1)
    expect(wrapper.find("[data-permit-load-error]").exists()).toBe(true)
    expect(wrapper.text()).toContain("加载准入许可失败。")
    expect(wrapper.text()).toContain("auth list_register_permits: request timed out")
    expect(toastStore.errorOf).not.toHaveBeenCalled()
  })

  it("issues and revokes permits through the list-based flow", async () => {
    const wrapper = mountPage()

    await Promise.resolve()
    await nextTick()

    const expiresAt = "2026-03-27T10:30"
    await wrapper.get("[data-open-issue-dialog]").trigger("click")
    await nextTick()

    await wrapper.get("[data-issue-device-input]").setValue("device-2")
    await wrapper.get("[data-issue-role-input]").setValue("observer")
    await wrapper.get("[data-issue-expires-input]").setValue(expiresAt)
    await wrapper.get("[data-issue-submit]").trigger("click")
    await Promise.resolve()
    await nextTick()

    expect(permitStore.issuePermit).toHaveBeenCalledWith({
      deviceId: "device-2",
      role: "observer",
      expiresAt: Math.floor(Date.parse(expiresAt) / 1000)
    })
    expect(permitStore.loadPermits).toHaveBeenCalledTimes(2)
    expect(wrapper.find("[data-permit-issue-dialog]").exists()).toBe(false)
    expect(wrapper.text()).toContain("permit_456")

    await wrapper.findAll("[data-row-revoke]")[0].trigger("click")
    await Promise.resolve()
    await nextTick()

    expect(permitStore.revokePermit).toHaveBeenCalledWith("permit_456")
    expect(permitStore.loadPermits).toHaveBeenCalledTimes(3)
    expect(wrapper.text()).not.toContain("permit_456")
    expect(wrapper.text()).toContain("permit_123")
  })
})
