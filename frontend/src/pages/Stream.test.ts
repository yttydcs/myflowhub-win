// @vitest-environment jsdom

import { defineComponent, nextTick, reactive } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { setLocale } from "@/i18n"

const streamState = reactive({
  targetId: "",
  selfNodeId: 0,
  defaultTargetId: 0,
  sources: [] as Array<{
    sourceId: string
    producer: number
    name: string
    kind: string
    contentType: string
    mode: string
    unitMode: string
    tags: string[]
    metadataRaw: string
  }>,
  consumers: [] as Array<{
    consumerId: string
    consumer: number
    name: string
    kind: string
    contentType: string
    tags: string[]
    metadataRaw: string
  }>,
  deliveries: [] as Array<{
    deliveryId: string
    producer: number
    consumer: number
    consumerId: string
    sourceId: string
    kind: string
    state: string
    bytesIn: number
    framesIn: number
    updatedAt: string
  }>,
  selectedSourceId: "",
  selectedConsumerId: "",
  selectedDeliveryId: "",
  lastSyncAt: "",
  lastEventAt: "",
  textFramesByDelivery: {} as Record<string, unknown[]>,
  statsByDelivery: {} as Record<string, unknown>
})

const resetStreamState = () => {
  streamState.targetId = ""
  streamState.selfNodeId = 0
  streamState.defaultTargetId = 0
  streamState.sources = []
  streamState.consumers = []
  streamState.deliveries = []
  streamState.selectedSourceId = ""
  streamState.selectedConsumerId = ""
  streamState.selectedDeliveryId = ""
  streamState.lastSyncAt = ""
  streamState.lastEventAt = ""
  streamState.textFramesByDelivery = {}
  streamState.statsByDelivery = {}
}

const streamStore = {
  state: streamState,
  setIdentity: vi.fn((nodeId: number, hubId: number) => {
    streamState.selfNodeId = nodeId
    streamState.defaultTargetId = hubId
  }),
  setTargetId: vi.fn((value: string) => {
    streamState.targetId = String(value ?? "").trim()
  }),
  listSources: vi.fn(async () => streamState.sources),
  listConsumers: vi.fn(async () => streamState.consumers),
  loadDeliveries: vi.fn(async () => streamState.deliveries),
  announceSource: vi.fn(async (draft: { name: string; kind: string; contentType: string; mode: string; unitMode: string; tagsText: string; metadataText: string }) => {
    const source = {
      sourceId: `source-${streamState.sources.length + 1}`,
      producer: streamState.selfNodeId || 7,
      name: draft.name,
      kind: draft.kind,
      contentType: draft.contentType,
      mode: draft.mode,
      unitMode: draft.unitMode,
      tags: [],
      metadataRaw: draft.metadataText
    }
    streamState.sources = [source, ...streamState.sources]
    streamState.selectedSourceId = source.sourceId
    return source
  }),
  announceConsumer: vi.fn(async (draft: { name: string; kind: string; contentType: string; metadataText: string }) => {
    const consumer = {
      consumerId: `consumer-${streamState.consumers.length + 1}`,
      consumer: streamState.selfNodeId || 7,
      name: draft.name,
      kind: draft.kind,
      contentType: draft.contentType,
      tags: [],
      metadataRaw: draft.metadataText
    }
    streamState.consumers = [consumer, ...streamState.consumers]
    streamState.selectedConsumerId = consumer.consumerId
    return consumer
  }),
  connect: vi.fn(async () => undefined),
  subscribe: vi.fn(async () => undefined),
  disconnect: vi.fn(async () => undefined),
  unsubscribe: vi.fn(async () => undefined),
  signal: vi.fn(async () => undefined),
  withdrawSource: vi.fn(async () => undefined),
  withdrawConsumer: vi.fn(async () => undefined),
  selectSource: vi.fn((sourceId: string) => {
    streamState.selectedSourceId = sourceId
  }),
  selectConsumer: vi.fn((consumerId: string) => {
    streamState.selectedConsumerId = consumerId
  }),
  selectDelivery: vi.fn((deliveryId: string) => {
    streamState.selectedDeliveryId = deliveryId
  }),
  textFramesFor: vi.fn(() => []),
  statsFor: vi.fn(() => null)
}

const sessionStore = reactive({
  auth: {
    nodeId: 7,
    hubId: 9
  }
})

const toastStore = {
  success: vi.fn(),
  warn: vi.fn(),
  error: vi.fn(),
  errorOf: vi.fn(),
  info: vi.fn()
}

vi.mock("@/stores/stream", () => ({
  streamKinds: ["music", "video", "text", "custom"],
  useStreamStore: () => streamStore
}))

vi.mock("@/stores/session", () => ({
  useSessionStore: () => sessionStore
}))

vi.mock("@/stores/toast", () => ({
  useToastStore: () => toastStore
}))

import Stream from "./Stream.vue"

const PageHeroStub = defineComponent({
  props: {
    description: { type: String, default: "" }
  },
  template: `
    <section>
      <p v-if="description">{{ description }}</p>
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
  mount(Stream, {
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

describe("Stream page", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setLocale("zh-CN")
    resetStreamState()
    sessionStore.auth.nodeId = 7
    sessionStore.auth.hubId = 9
  })

  it("auto-loads catalogs on first render while keeping add forms inside dialogs", async () => {
    const wrapper = mountPage()

    await Promise.resolve()
    await nextTick()

    expect(streamStore.setIdentity).toHaveBeenCalledWith(7, 9)
    expect(streamStore.setTargetId).toHaveBeenCalledWith("9")
    expect(streamStore.listSources).toHaveBeenCalledWith("7", "", "")
    expect(streamStore.listConsumers).toHaveBeenCalledWith("7", "", "")
    expect(streamStore.loadDeliveries).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain("查看流目录、连接所需端点，并只在需要时打开新增表单。")
    expect(wrapper.find("[data-stream-source-dialog]").exists()).toBe(false)
    expect(wrapper.find("[data-stream-consumer-dialog]").exists()).toBe(false)
  })

  it("opens the source dialog on demand and closes it after creating a local source", async () => {
    const wrapper = mountPage()

    await Promise.resolve()
    await nextTick()

    await wrapper.get("[data-stream-open-source]").trigger("click")
    await nextTick()

    expect(wrapper.find("[data-stream-source-dialog]").exists()).toBe(true)

    await wrapper.get("#stream-source-name").setValue("Local Text Source")
    await wrapper.get("[data-stream-submit-source]").trigger("click")
    await Promise.resolve()
    await nextTick()

    expect(streamStore.announceSource).toHaveBeenCalledTimes(1)
    expect(wrapper.find("[data-stream-source-dialog]").exists()).toBe(false)
    expect(streamState.sources).toHaveLength(1)
    expect(streamState.sources[0].name).toBe("Local Text Source")
  })
})
