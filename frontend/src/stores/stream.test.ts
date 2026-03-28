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
const listSourcesSimple = vi.fn()
const listConsumersSimple = vi.fn()
const connectSimple = vi.fn()
const disconnectSimple = vi.fn()

const resetState = () => {
  store.state.targetId = ""
  store.state.selfNodeId = 0
  store.state.defaultTargetId = 0
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

const emitRuntimeEvent = (name: string, payload: any) => {
  const handler = runtimeState.listeners[name]
  if (!handler) {
    throw new Error(`listener ${name} not registered`)
  }
  handler(payload)
}

beforeEach(() => {
  setLocale("en")
  resetState()
  listSourcesSimple.mockReset()
  listConsumersSimple.mockReset()
  connectSimple.mockReset()
  disconnectSimple.mockReset()
  ;(window as any).go = {
    stream: {
      StreamService: {
        ListSourcesSimple: listSourcesSimple,
        ListConsumersSimple: listConsumersSimple,
        ConnectSimple: connectSimple,
        DisconnectSimple: disconnectSimple
      }
    }
  }
  store.setIdentity(7, 9)
})

describe("stream store", () => {
  it("normalizes stream catalogs and updates delivery state through connect/disconnect", async () => {
    listSourcesSimple.mockResolvedValueOnce({
      sources: [
        {
          source_id: "source-1",
          producer: 12,
          name: "Demo Source",
          kind: "text",
          content_type: "text/plain",
          mode: "live",
          unit_mode: "frame",
          tags: ["alpha", "beta"],
          metadata: { channel: "room-1" }
        }
      ]
    })
    listConsumersSimple.mockResolvedValueOnce({
      consumer_endpoints: [
        {
          consumer_id: "consumer-1",
          consumer: 7,
          name: "Local Consumer",
          kind: "text",
          content_type: "text/plain",
          tags: ["alpha"],
          metadata: { buffer: 32 }
        }
      ]
    })
    connectSimple.mockResolvedValueOnce({
      code: 1,
      accept: true,
      delivery_id: "delivery-1",
      producer: 12,
      consumer: 7,
      consumer_id: "consumer-1",
      source: {
        source_id: "source-1",
        kind: "text",
        content_type: "text/plain",
        mode: "live",
        unit_mode: "frame"
      }
    })
    disconnectSimple.mockResolvedValueOnce({ code: 1 })

    const sources = await store.listSources("12", "text", "alpha")
    const consumers = await store.listConsumers("7", "text", "alpha")
    const delivery = await store.connect({
      producer: 12,
      sourceId: "source-1",
      consumer: 7,
      consumerId: "consumer-1"
    })
    await store.disconnect("delivery-1")

    expect(listSourcesSimple).toHaveBeenCalledWith(7, 9, {
      req_id: "",
      producer: 12,
      kind: "text",
      tag: "alpha"
    })
    expect(sources).toHaveLength(1)
    expect(sources[0]).toMatchObject({
      sourceId: "source-1",
      kind: "text",
      contentType: "text/plain",
      mode: "live",
      unitMode: "frame",
      tags: ["alpha", "beta"]
    })
    expect(sources[0].metadataRaw).toContain("\"channel\": \"room-1\"")

    expect(listConsumersSimple).toHaveBeenCalledWith(7, 9, {
      req_id: "",
      consumer: 7,
      kind: "text",
      tag: "alpha"
    })
    expect(consumers).toHaveLength(1)
    expect(consumers[0]).toMatchObject({
      consumerId: "consumer-1",
      consumer: 7,
      kind: "text",
      contentType: "text/plain"
    })
    expect(consumers[0].metadataRaw).toContain("\"buffer\": 32")

    expect(connectSimple).toHaveBeenCalledWith(7, 9, {
      req_id: "",
      producer: 12,
      source_id: "source-1",
      consumer: 7,
      consumer_id: "consumer-1"
    })
    expect(delivery).toMatchObject({
      deliveryId: "delivery-1",
      sourceId: "source-1",
      consumerId: "consumer-1",
      kind: "text",
      state: "active"
    })
    expect(store.state.selectedDeliveryId).toBe("delivery-1")
    expect(store.state.deliveries[0]).toMatchObject({
      deliveryId: "delivery-1",
      state: "closed"
    })
  })

  it("keeps bounded text history and mirrors stats events into deliveries", () => {
    emitRuntimeEvent("stream.delivery", {
      deliveryId: "delivery-2",
      sourceId: "source-2",
      producer: 18,
      consumer: 7,
      consumerId: "consumer-2",
      kind: "text",
      contentType: "text/plain",
      state: "active",
      updatedAt: "2026-03-28T08:00:00Z"
    })

    for (let index = 0; index < 205; index += 1) {
      emitRuntimeEvent("stream.text", {
        deliveryId: "delivery-2",
        kind: "text",
        text: `frame-${index}`,
        position: index,
        ptsMs: index * 10,
        flags: 0,
        updatedAt: `2026-03-28T08:00:${String(index % 60).padStart(2, "0")}Z`
      })
    }

    emitRuntimeEvent("stream.stats", {
      deliveryId: "delivery-2",
      kind: "text",
      bytesIn: 4096,
      framesIn: 205,
      lastPosition: 204,
      lastPtsMs: 2040,
      lastAckPos: 128,
      lastFlags: 3,
      updatedAt: "2026-03-28T08:05:00Z"
    })

    const frames = store.textFramesFor("delivery-2")
    const stats = store.statsFor("delivery-2")

    expect(store.state.selectedDeliveryId).toBe("delivery-2")
    expect(frames).toHaveLength(200)
    expect(frames[0].text).toBe("frame-5")
    expect(frames[199].text).toBe("frame-204")
    expect(stats).toMatchObject({
      deliveryId: "delivery-2",
      bytesIn: 4096,
      framesIn: 205,
      lastPosition: 204,
      lastAckPos: 128,
      lastFlags: 3
    })
    expect(store.state.deliveries[0]).toMatchObject({
      deliveryId: "delivery-2",
      bytesIn: 4096,
      framesIn: 205,
      lastPosition: 204,
      lastAckPos: 128,
      lastFlags: 3,
      updatedAt: "2026-03-28T08:05:00Z"
    })
    expect(store.state.lastEventAt).not.toBe("")
  })
})
