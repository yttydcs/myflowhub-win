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
  issuing: false,
  revoking: false,
  lastIssued: undefined as
    | undefined
    | {
        permit: string
        deviceId: string
        role: string
        expiresAt: number
        issuedAt: string
        revoked: boolean
      },
  lastRevoke: undefined as
    | undefined
    | {
        permit: string
        deviceId: string
        role: string
        revokedAt: string
      }
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

const permitStore = {
  state: permitState,
  issuePermit: vi.fn(
    async (input: { deviceId: string; role: string; expiresAt: number }) => {
      permitState.issuing = true
      permitState.lastIssued = {
        permit: "permit_123",
        deviceId: input.deviceId,
        role: input.role,
        expiresAt: input.expiresAt,
        issuedAt: "2026-03-27T10:00:00.000Z",
        revoked: false
      }
      permitState.lastRevoke = undefined
      permitState.issuing = false
      return permitState.lastIssued
    }
  ),
  revokePermit: vi.fn(async (permit: string) => {
    permitState.revoking = true
    permitState.lastRevoke = {
      permit,
      deviceId: permitState.lastIssued?.deviceId || "",
      role: permitState.lastIssued?.role || "",
      revokedAt: "2026-03-27T10:05:00.000Z"
    }
    if (permitState.lastIssued?.permit === permit) {
      permitState.lastIssued.revoked = true
    }
    permitState.revoking = false
    return permitState.lastRevoke
  }),
  reset: vi.fn(() => {
    permitState.issuing = false
    permitState.revoking = false
    permitState.lastIssued = undefined
    permitState.lastRevoke = undefined
  })
}

const toastStore = {
  success: vi.fn(),
  warn: vi.fn(),
  error: vi.fn(),
  errorOf: vi.fn(),
  info: vi.fn()
}

const clipboardWriteText = vi.fn()

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
    permitState.issuing = false
    permitState.revoking = false
    permitState.lastIssued = undefined
    permitState.lastRevoke = undefined

    Object.defineProperty(window.navigator, "clipboard", {
      configurable: true,
      value: {
        writeText: clipboardWriteText
      }
    })
  })

  it("renders compact action rows and opens focused dialogs only when needed", async () => {
    const wrapper = mountPage()

    await Promise.resolve()
    await nextTick()

    expect(wrapper.text()).toContain("准入许可")
    expect(wrapper.text()).toContain("许可动作")
    expect(wrapper.text()).not.toContain("解析")
    expect(wrapper.find("[data-permit-issue-dialog]").exists()).toBe(false)
    expect(wrapper.find("[data-permit-revoke-dialog]").exists()).toBe(false)

    await wrapper.get("[data-open-issue-dialog]").trigger("click")
    await nextTick()

    expect(wrapper.find("[data-permit-issue-dialog]").exists()).toBe(true)

    await wrapper.get("[data-open-revoke-dialog]").trigger("click")
    await nextTick()

    expect(wrapper.find("[data-permit-revoke-dialog]").exists()).toBe(true)
  })

  it("issues, copies, and revokes the latest permit through the focused flow", async () => {
    const wrapper = mountPage()

    await Promise.resolve()
    await nextTick()

    const expiresAt = "2026-03-27T10:30"
    await wrapper.get("[data-open-issue-dialog]").trigger("click")
    await nextTick()

    await wrapper.get("[data-issue-device-input]").setValue("device-1")
    await wrapper.get("[data-issue-role-input]").setValue("admin")
    await wrapper.get("[data-issue-expires-input]").setValue(expiresAt)
    await wrapper.get("[data-issue-submit]").trigger("click")
    await Promise.resolve()
    await nextTick()

    expect(permitStore.issuePermit).toHaveBeenCalledWith({
      deviceId: "device-1",
      role: "admin",
      expiresAt: Math.floor(Date.parse(expiresAt) / 1000)
    })
    expect(wrapper.find("[data-permit-issue-dialog]").exists()).toBe(false)
    expect(wrapper.get("[data-latest-permit-card]").text()).toContain("permit_123")

    await wrapper.get("[data-copy-permit]").trigger("click")
    await Promise.resolve()
    expect(clipboardWriteText).toHaveBeenCalledWith("permit_123")

    await wrapper.get("[data-open-latest-revoke-dialog]").trigger("click")
    await nextTick()

    expect(wrapper.find("[data-permit-revoke-dialog]").exists()).toBe(true)
    const revokeInput = wrapper.get("[data-revoke-permit-input]")
    expect(revokeInput.element.tagName).toBe("INPUT")
    expect((revokeInput.element as HTMLInputElement).value).toBe("permit_123")
    expect(wrapper.find("[data-latest-permit-details]").exists()).toBe(true)

    await wrapper.get("[data-revoke-submit]").trigger("click")
    await Promise.resolve()
    await nextTick()

    expect(permitStore.revokePermit).toHaveBeenCalledWith("permit_123")
    expect(wrapper.find("[data-permit-revoke-dialog]").exists()).toBe(false)
    expect(wrapper.get("[data-latest-permit-card]").text()).toContain("已撤销")
  })
})
