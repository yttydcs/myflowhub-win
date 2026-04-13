// @vitest-environment jsdom

import { defineComponent, nextTick, reactive } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { setLocale } from "@/i18n"

const sessionStore = reactive({
  connected: true,
  auth: {
    nodeId: 7,
    hubId: 9
  }
})

const varpoolState = reactive({
  keys: [{ name: "cpu.load", owner: 7 }]
})

const varpoolStore = {
  state: varpoolState,
  setIdentity: vi.fn(),
  listOwnerNames: vi.fn(async () => ["cpu.load", "mem.free"]),
  addWatchKey: vi.fn(async () => undefined),
  getVar: vi.fn(async () => undefined)
}

const toastStore = {
  success: vi.fn(),
  info: vi.fn(),
  errorOf: vi.fn()
}

vi.mock("@/stores/session", () => ({
  useSessionStore: () => sessionStore
}))

vi.mock("@/stores/varpool", () => ({
  useVarPoolStore: () => varpoolStore
}))

vi.mock("@/stores/toast", () => ({
  useToastStore: () => toastStore
}))

import NodeVarsDialog from "./NodeVarsDialog.vue"

const OverlayStub = defineComponent({
  props: {
    open: { type: Boolean, default: false }
  },
  template: `<div v-if="open"><slot /></div>`
})

const ButtonStub = defineComponent({
  props: {
    disabled: { type: Boolean, default: false }
  },
  emits: ["click"],
  template: `<button type="button" :disabled="disabled" @click="$emit('click', $event)"><slot /></button>`
})

const BadgeStub = defineComponent({
  template: `<span><slot /></span>`
})

describe("NodeVarsDialog", () => {
  beforeEach(() => {
    setLocale("en")
    vi.clearAllMocks()
  })

  it("keeps the dialog card bounded and moves the long body into an internal scroll region", async () => {
    const wrapper = mount(NodeVarsDialog, {
      props: {
        open: false,
        ownerId: 7
      },
      global: {
        stubs: {
          Overlay: OverlayStub,
          Button: ButtonStub,
          Badge: BadgeStub
        }
      }
    })

    await wrapper.setProps({ open: true })
    await nextTick()

    const dialog = wrapper.get("[data-node-vars-dialog]")
    const scroll = wrapper.get("[data-node-vars-scroll]")

    expect(dialog.classes()).toContain("max-h-[85vh]")
    expect(dialog.classes()).toContain("overflow-hidden")
    expect(scroll.classes()).toContain("flex-1")
    expect(scroll.classes()).toContain("overflow-y-auto")
    expect(wrapper.text()).toContain("Node Variables")
    expect(wrapper.text()).toContain("Add Watch")

    const inputs = wrapper.findAll("input")
    expect(inputs[0]).toBeTruthy()
    expect((inputs[0].element as HTMLInputElement).value).toBe("7")
  })
})
