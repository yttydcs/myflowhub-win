// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from "vitest"
import { setLocale } from "@/i18n"

const runtimeState = vi.hoisted(() => {
  const listeners: Record<string, (evt: any) => void> = {}
  return {
    listeners,
    EventsOn: vi.fn((name: string, cb: (evt: any) => void) => {
      listeners[name] = cb
      return () => {
        delete listeners[name]
      }
    })
  }
})

vi.mock("../../wailsjs/runtime/runtime", () => ({
  EventsOn: runtimeState.EventsOn
}))

import { useStreamStore } from "./stream"

const store = useStreamStore()
const streamPrefs = vi.fn()
const saveStreamPrefs = vi.fn()
const listSourcesSimple = vi.fn()
const listConsumersSimple = vi.fn()
const announceSimple = vi.fn()
const announceConsumerSimple = vi.fn()
const publishTextSimple = vi.fn()

const resetState = () => {
  store.state.activeTab = "source"
  store.state.targetId = ""
  store.state.selfNodeId = 0
  store.state.defaultTargetId = 0
  store.state.localSources = []
  store.state.localConsumers = []
  store.state.sources = []
  store.state.consumers = []
  store.state.deliveries = []
  store.state.selectedSourceId = ""
  store.state.selectedConsumerId = ""
  store.state.selectedDeliveryId = ""
  store.state.lastSyncAt = ""
  store.state.lastEventAt = ""
  store.state.textFramesByDelivery = {}
  store.state.statsByDelivery = {}
}

beforeEach(() => {
  setLocale("en")
  resetState()
  streamPrefs.mockReset()
  saveStreamPrefs.mockReset()
  listSourcesSimple.mockReset()
  listConsumersSimple.mockReset()
  announceSimple.mockReset()
  announceConsumerSimple.mockReset()
  publishTextSimple.mockReset()
  ;(window as any).go = {
    main: {
      App: {
        StreamPrefs: streamPrefs,
        SaveStreamPrefs: saveStreamPrefs
      }
    },
    stream: {
      StreamService: {
        ListSourcesSimple: listSourcesSimple,
        ListConsumersSimple: listConsumersSimple,
        AnnounceSimple: announceSimple,
        AnnounceConsumerSimple: announceConsumerSimple,
        PublishTextSimple: publishTextSimple,
        DeliverySnapshot: vi.fn(async () => [])
      }
    }
  }
  store.setIdentity(7, 9)
})

describe("stream store", () => {
  it("loads persisted local catalogs and restores them through announce bindings", async () => {
    streamPrefs.mockResolvedValueOnce({
      activeTab: "consumer",
      targetId: 9,
      sources: [
        {
          sourceId: "source-1",
          name: "Local Text Source",
          kind: "text",
          contentType: "text/plain",
          mode: "live",
          unitMode: "frame",
          tags: ["alpha"],
          metadataRaw: "{\"room\":1}"
        }
      ],
      consumers: [
        {
          consumerId: "consumer-1",
          name: "Local Consumer",
          kind: "text",
          contentType: "text/plain",
          tags: ["alpha"],
          metadataRaw: "{\"buffer\":32}"
        }
      ]
    })
    announceSimple.mockResolvedValueOnce({
      source: {
        source_id: "source-1",
        producer: 7,
        name: "Local Text Source",
        kind: "text",
        content_type: "text/plain",
        mode: "live",
        unit_mode: "frame",
        tags: ["alpha"],
        metadata: { room: 1 }
      }
    })
    announceConsumerSimple.mockResolvedValueOnce({
      consumer_endpoint: {
        consumer_id: "consumer-1",
        consumer: 7,
        name: "Local Consumer",
        kind: "text",
        content_type: "text/plain",
        tags: ["alpha"],
        metadata: { buffer: 32 }
      }
    })

    await store.loadPrefs()
    const result = await store.restoreLocalCatalogs()

    expect(store.state.activeTab).toBe("consumer")
    expect(store.state.targetId).toBe("9")
    expect(store.state.localSources[0]).toMatchObject({ sourceId: "source-1", producer: 7 })
    expect(store.state.localConsumers[0]).toMatchObject({ consumerId: "consumer-1", consumer: 7 })
    expect(result).toEqual({ attempted: 2, failed: 0 })
    expect(announceSimple).toHaveBeenCalledWith(7, 9, {
      req_id: "",
      source: {
        source_id: "source-1",
        name: "Local Text Source",
        kind: "text",
        content_type: "text/plain",
        mode: "live",
        unit_mode: "frame",
        tags: ["alpha"],
        metadata: { room: 1 }
      }
    })
    expect(announceConsumerSimple).toHaveBeenCalledWith(7, 9, {
      req_id: "",
      consumer_endpoint: {
        consumer_id: "consumer-1",
        name: "Local Consumer",
        kind: "text",
        content_type: "text/plain",
        tags: ["alpha"],
        metadata: { buffer: 32 }
      }
    })
  })

  it("keeps remote catalog queries separate from local saved sources", async () => {
    store.state.localSources = [
      {
        sourceId: "source-local",
        producer: 7,
        name: "Saved Local",
        kind: "text",
        contentType: "text/plain",
        mode: "live",
        unitMode: "frame",
        tags: [],
        metadataRaw: ""
      }
    ]
    listSourcesSimple.mockResolvedValueOnce({
      sources: [
        {
          source_id: "source-remote",
          producer: 12,
          name: "Remote Source",
          kind: "text",
          content_type: "text/plain",
          mode: "live",
          unit_mode: "frame",
          tags: ["alpha"]
        }
      ]
    })

    const sources = await store.listSources("12", "text", "alpha")

    expect(sources).toHaveLength(1)
    expect(store.state.sources[0]).toMatchObject({ sourceId: "source-remote", producer: 12 })
    expect(store.state.localSources[0]).toMatchObject({ sourceId: "source-local", producer: 7 })
  })

  it("publishes text through the new binding for local text sources", async () => {
    store.state.localSources = [
      {
        sourceId: "source-1",
        producer: 7,
        name: "Local Text Source",
        kind: "text",
        contentType: "text/plain",
        mode: "live",
        unitMode: "frame",
        tags: [],
        metadataRaw: ""
      }
    ]
    publishTextSimple.mockResolvedValueOnce({
      code: 1,
      source_id: "source-1",
      sent: 2,
      delivery_ids: ["delivery-1", "delivery-2"]
    })

    const result = await store.publishText("source-1", "hello world")

    expect(publishTextSimple).toHaveBeenCalledWith(7, {
      source_id: "source-1",
      text: "hello world"
    })
    expect(result).toEqual({
      sourceId: "source-1",
      sent: 2,
      deliveryIds: ["delivery-1", "delivery-2"]
    })
  })
})
