// Context: covers detached TopicBus windows activating their own subscription.

// @vitest-environment jsdom

import { defineComponent, nextTick } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { setLocale } from "@/i18n"

const hoisted = vi.hoisted(() => ({
  EventsOn: vi.fn(() => undefined),
  IsConnected: vi.fn(),
  LastAddr: vi.fn(),
  loadHomeState: vi.fn(),
  topicBusPrefs: vi.fn(),
  saveTopicBusPrefs: vi.fn(),
  subscribeSimple: vi.fn(),
  subscribeBatchSimple: vi.fn(),
  toastStore: {
    errorOf: vi.fn(),
    success: vi.fn()
  }
}))

vi.mock("vue-router", () => ({
  useRoute: () => ({
    query: {
      topic: "dev.codex.msg",
      targetId: "1"
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

vi.mock("@/stores/toast", () => ({
  useToastStore: () => hoisted.toastStore
}))

import TopicBusWindow from "./TopicBusWindow.vue"
import { useSessionStore } from "@/stores/session"
import { useTopicBusStore } from "@/stores/topicbus"

const BadgeStub = defineComponent({
  template: `<span><slot /></span>`
})

const ButtonStub = defineComponent({
  template: `<button type="button"><slot /></button>`
})

const CardHeaderStub = defineComponent({
  props: {
    title: { type: String, default: "" },
    description: { type: String, default: "" }
  },
  template: `<div><h2>{{ title }}</h2><p>{{ description }}</p><slot name="actions" /></div>`
})

const flushAsync = async () => {
  await Promise.resolve()
  await Promise.resolve()
  await nextTick()
}

describe("TopicBusWindow", () => {
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

    const topicbus = useTopicBusStore()
    topicbus.state.targetId = ""
    topicbus.state.selfNodeId = 0
    topicbus.state.defaultTargetId = 0
    topicbus.state.topics = []
    topicbus.state.remoteTopics = []
    topicbus.state.selectedTopic = ""
    topicbus.state.maxEvents = 500
    topicbus.state.events = []
    topicbus.state.lastFrameAt = ""
    topicbus.state.remoteSyncedAt = ""

    hoisted.loadHomeState.mockResolvedValue({ nodeId: 2, hubId: 1 })
    hoisted.IsConnected.mockResolvedValue(true)
    hoisted.LastAddr.mockResolvedValue("47.111.165.7:9000")
    hoisted.topicBusPrefs.mockResolvedValue({ topics: [], maxEvents: 500, targetId: 0 })
    hoisted.saveTopicBusPrefs.mockImplementation(async (prefs: any) => prefs)
    hoisted.subscribeSimple.mockResolvedValue({ code: 1, msg: "ok", topic: "dev.codex.msg" })
    hoisted.subscribeBatchSimple.mockResolvedValue({ code: 1, msg: "ok", topics: ["dev.codex.msg"] })

    ;(window as any).go = {
      main: {
        App: {
          TopicBusPrefs: hoisted.topicBusPrefs,
          SaveTopicBusPrefs: hoisted.saveTopicBusPrefs
        }
      },
      topicbus: {
        TopicBusService: {
          SubscribeSimple: hoisted.subscribeSimple,
          SubscribeBatchSimple: hoisted.subscribeBatchSimple
        }
      }
    }
  })

  it("subscribes to the requested topic when a detached topic window opens", async () => {
    mount(TopicBusWindow, {
      global: {
        stubs: {
          Badge: BadgeStub,
          Button: ButtonStub,
          CardHeader: CardHeaderStub
        }
      }
    })

    await flushAsync()
    await flushAsync()
    await flushAsync()

    await vi.waitFor(() => {
      expect(hoisted.subscribeSimple).toHaveBeenCalledWith(2, 1, "dev.codex.msg")
    })
    expect(hoisted.saveTopicBusPrefs).toHaveBeenCalledWith({
      topics: ["dev.codex.msg"],
      maxEvents: 500,
      targetId: 1
    })
  })
})
