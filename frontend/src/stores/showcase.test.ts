// 本文件覆盖 `showcase` store 的关键交互行为。

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

const toastState = vi.hoisted(() => ({
  errorOf: vi.fn(),
  error: vi.fn(),
  success: vi.fn()
}))

vi.mock("../../wailsjs/runtime/runtime", () => ({
  EventsOn: runtimeState.EventsOn
}))

vi.mock("@/stores/toast", () => ({
  useToastStore: () => toastState
}))

import { useShowcaseStore, type ShowcaseWidget } from "./showcase"

const store = useShowcaseStore()
const sendSimple = vi.fn()
const setSimple = vi.fn()

const createSliderWidget = (): ShowcaseWidget => ({
  id: "widget-slider",
  kind: "var",
  title: "Temperature",
  targetId: 9,
  layout: { colSpan: 1 },
  var: {
    ownerId: 12,
    name: "temperature",
    mode: "slider",
    visibility: "public",
    type: "float64",
    slider: { min: 0, max: 100, step: 1, throttleMs: 50 },
    switch: { onValue: "true", offValue: "false" },
    chart: { rangeMs: 60 * 60 * 1000, bucketMs: 60 * 1000 }
  }
})

beforeEach(() => {
  setLocale("en")
  sendSimple.mockReset()
  setSimple.mockReset()
  toastState.errorOf.mockReset()
  toastState.error.mockReset()
  toastState.success.mockReset()

  const widget = createSliderWidget()
  store.state.loaded = true
  store.state.busy = false
  store.state.lastLoadedAt = ""
  store.state.selfNodeId = 7
  store.state.hubId = 9
  store.state.fixedScreenId = ""
  store.state.screenMissing = false
  store.state.config = {
    version: 1,
    currentScreenId: "default",
    screens: [
      {
        id: "default",
        name: "Default",
        updatedAt: "2026-04-14T00:00:00.000Z",
        layout: {
          mode: "columns",
          columns: { maxColumns: 3, minColumnWidth: 360, gap: 16 },
          canvas: { baseWidth: 960, baseHeight: 720 }
        },
        widgets: [widget]
      }
    ]
  }
  store.state.values = {}
  store.state.lastFrameAt = ""
  store.state.sliderDraft = {}
  store.setIdentity(7, 9)

  ;(window as any).go = {
    varpool: {
      VarPoolService: {
        SendSimple: sendSimple,
        SetSimple: setSimple
      }
    }
  }
})

describe("showcase store", () => {
  it("commits slider updates without waiting for SetSimple ack and clears draft after varpool.changed", async () => {
    const widget = store.state.config.screens[0]!.widgets[0] as ShowcaseWidget
    store.state.sliderDraft[widget.id] = 25
    sendSimple.mockResolvedValueOnce(undefined)

    await store.sliderCommit(widget)

    expect(sendSimple).toHaveBeenCalledWith(7, 9, "set", {
      name: "temperature",
      value: "25",
      visibility: "public",
      type: "float64",
      owner: 12
    })
    expect(setSimple).not.toHaveBeenCalled()
    expect(store.state.sliderDraft[widget.id]).toBe(25)

    runtimeState.listeners["varpool.changed"]?.({
      owner: 12,
      name: "temperature",
      value: "25",
      visibility: "public",
      type: "float64"
    })

    expect(store.state.sliderDraft[widget.id]).toBeUndefined()
    expect(Object.values(store.state.values)).toContainEqual(
      expect.objectContaining({
        ownerId: 12,
        name: "temperature",
        value: "25"
      })
    )
  })
})
