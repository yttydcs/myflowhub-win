// Context: covers detached Showcase window session snapshot hydration in the Win frontend.

// @vitest-environment jsdom

import { defineComponent, nextTick } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { setLocale } from "@/i18n"

const hoisted = vi.hoisted(() => {
  const screen = {
    id: "default",
    name: "Default Screen",
    layout: {
      mode: "columns",
      columns: {
        maxColumns: 3,
        minColumnWidth: 360,
        gap: 16
      },
      canvas: {
        baseWidth: 960,
        baseHeight: 720
      }
    },
    widgets: [
      {
        id: "widget-1",
        kind: "topic_button",
        title: "Ping",
        targetId: 1,
        layout: { colSpan: 1 },
        topicButton: {
          topic: "status",
          name: "ping",
          payloadText: ""
        }
      }
    ]
  }

  const showcaseStore = {
    state: {
      loaded: true,
      screenMissing: false
    },
    setFixedScreenId: vi.fn(),
    clearFixedScreenId: vi.fn(),
    setIdentity: vi.fn(),
    load: vi.fn(async () => undefined),
    enter: vi.fn(async () => undefined),
    leave: vi.fn(async () => undefined),
    screenById: vi.fn((id: string) => (id === "default" ? screen : null))
  }

  return {
    EventsOn: vi.fn(() => undefined),
    IsConnected: vi.fn(),
    LastAddr: vi.fn(),
    loadHomeState: vi.fn(),
    showcaseStore,
    toastStore: {
      errorOf: vi.fn()
    }
  }
})

vi.mock("vue-router", () => ({
  useRoute: () => ({
    query: {
      screenId: "default"
    }
  })
}))

vi.mock("../../wailsjs/runtime/runtime", () => ({
  EventsOn: hoisted.EventsOn
}))

vi.mock("../../wailsjs/go/session/SessionService", () => ({
  IsConnected: hoisted.IsConnected,
  LastAddr: hoisted.LastAddr
}))

vi.mock("../../wailsjs/go/main/App", () => ({
  HomeState: hoisted.loadHomeState
}))

vi.mock("@/stores/profile", () => ({
  useProfileStore: () => ({
    state: {
      current: "default"
    }
  })
}))

vi.mock("@/stores/showcase", () => ({
  useShowcaseStore: () => hoisted.showcaseStore
}))

vi.mock("@/stores/toast", () => ({
  useToastStore: () => hoisted.toastStore
}))

import ShowcaseWindow from "./ShowcaseWindow.vue"
import { useSessionStore } from "@/stores/session"

const BadgeStub = defineComponent({
  template: `<span><slot /></span>`
})

const ShowcaseWidgetCardContentStub = defineComponent({
  props: {
    connected: { type: Boolean, default: false },
    selfNodeId: { type: Number, default: 0 }
  },
  template: `
    <div
      data-test="widget-card"
      :data-connected="String(connected)"
      :data-self-node-id="String(selfNodeId)"
    />
  `
})

const flushAsync = async () => {
  await Promise.resolve()
  await Promise.resolve()
  await nextTick()
}

describe("ShowcaseWindow", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setLocale("en")

    const sessionStore = useSessionStore()
    sessionStore.connected = false
    sessionStore.addr = ""
    sessionStore.lastStateAt = ""
    sessionStore.auth.deviceId = ""
    sessionStore.auth.nodeId = 0
    sessionStore.auth.hubId = 0
    sessionStore.auth.role = ""
    sessionStore.auth.loggedIn = false
    sessionStore.auth.lastAuthMessage = ""
    sessionStore.auth.lastAuthAction = ""
    sessionStore.auth.lastAuthAt = ""

    hoisted.showcaseStore.state.loaded = true
    hoisted.showcaseStore.state.screenMissing = false
    hoisted.loadHomeState.mockResolvedValue({ nodeId: 7, hubId: 9 })
    hoisted.IsConnected.mockResolvedValue(true)
    hoisted.LastAddr.mockResolvedValue("127.0.0.1:9000")

    ;(globalThis as any).ResizeObserver = class {
      observe() {}
      disconnect() {}
    }
  })

  it("hydrates the session snapshot before rendering widget interaction state", async () => {
    const wrapper = mount(ShowcaseWindow, {
      global: {
        stubs: {
          Badge: BadgeStub,
          ShowcaseWidgetCardContent: ShowcaseWidgetCardContentStub
        }
      }
    })

    await flushAsync()
    await flushAsync()

    const sessionStore = useSessionStore()

    expect(sessionStore.connected).toBe(true)
    expect(sessionStore.addr).toBe("127.0.0.1:9000")
    expect(hoisted.showcaseStore.setIdentity).toHaveBeenCalledWith(7, 9)
    expect(wrapper.text()).toContain("Connected")
    expect(wrapper.get('[data-test="widget-card"]').attributes("data-connected")).toBe("true")
    expect(wrapper.get('[data-test="widget-card"]').attributes("data-self-node-id")).toBe("7")
  })
})
