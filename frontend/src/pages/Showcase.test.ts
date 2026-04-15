// 本文件覆盖 Win 前端 `Showcase` 页面的刷新行为。

// @vitest-environment jsdom

import { defineComponent, nextTick } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { setLocale } from "@/i18n"

const hoisted = vi.hoisted(() => {
  const screen = {
    id: "default",
    name: "Default Screen",
    updatedAt: "2026-04-14T00:00:00.000Z",
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
        kind: "var",
        title: "Temperature",
        targetId: 9,
        layout: { colSpan: 1 },
        var: {
          ownerId: 7,
          name: "temperature",
          mode: "slider",
          visibility: "public",
          type: "float64",
          slider: { min: 0, max: 100, step: 1, throttleMs: 50 },
          chart: { rangeMs: 60 * 60 * 1000, bucketMs: 60 * 1000 },
          switch: { onValue: "true", offValue: "false" }
        }
      }
    ]
  }

  const sessionStore = {
    connected: true,
    addr: "127.0.0.1:9000",
    auth: {
      nodeId: 7,
      hubId: 9
    }
  }

  const showcaseState = {
    loaded: true,
    screenMissing: false,
    lastLoadedAt: "",
    config: {
      currentScreenId: "default",
      screens: [screen]
    }
  }

  const showcaseStore = {
    state: showcaseState,
    setConfigReloadEnabled: vi.fn(),
    setIdentity: vi.fn(),
    load: vi.fn(async () => undefined),
    leave: vi.fn(async () => undefined),
    enterScreen: vi.fn(async () => undefined),
    currentScreen: vi.fn(() => screen),
    screenById: vi.fn((id: string) => (id === "default" ? screen : null)),
    switchToggle: vi.fn(async () => undefined),
    sliderInput: vi.fn(async () => undefined),
    sliderCommit: vi.fn(async () => undefined)
  }

  return {
    loadHomeState: vi.fn(),
    hydrateSessionConnectionSnapshot: vi.fn(async () => undefined),
    openAuxWindow: vi.fn(async () => "opened"),
    profileStore: {
      state: {
        current: "default"
      }
    },
    sessionStore,
    showcaseStore,
    toastStore: {
      success: vi.fn(),
      errorOf: vi.fn(),
      error: vi.fn(),
      warn: vi.fn()
    },
    varpoolStore: {
      state: {
        keys: []
      },
      setIdentity: vi.fn(),
      listOwnerNames: vi.fn(async () => [])
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

vi.mock("../../wailsjs/go/main/App", () => ({
  HomeState: hoisted.loadHomeState
}))

vi.mock("@/stores/session", () => ({
  useSessionStore: () => hoisted.sessionStore,
  hydrateSessionConnectionSnapshot: hoisted.hydrateSessionConnectionSnapshot
}))

vi.mock("@/stores/profile", () => ({
  useProfileStore: () => hoisted.profileStore
}))

vi.mock("@/stores/showcase", () => ({
  useShowcaseStore: () => hoisted.showcaseStore
}))

vi.mock("@/stores/toast", () => ({
  useToastStore: () => hoisted.toastStore
}))

vi.mock("@/stores/varpool", () => ({
  useVarPoolStore: () => hoisted.varpoolStore
}))

vi.mock("@/lib/auxWindow", () => ({
  openAuxWindow: hoisted.openAuxWindow
}))

import Showcase from "./Showcase.vue"

const ButtonStub = defineComponent({
  props: {
    disabled: { type: Boolean, default: false }
  },
  emits: ["click"],
  template: `<button :disabled="disabled" @click="$emit('click', $event)"><slot /></button>`
})

const BadgeStub = defineComponent({
  template: `<span><slot /></span>`
})

const TooltipStub = defineComponent({
  template: `<div><slot /></div>`
})

const OverlayStub = defineComponent({
  template: `<div><slot /><slot name="content" /></div>`
})

const CardHeaderStub = defineComponent({
  template: `<div><slot /><slot name="actions" /></div>`
})

const ShowcaseWidgetCardContentStub = defineComponent({
  props: {
    busy: { type: Boolean, default: false }
  },
  template: `<div data-test="widget-card" :data-busy="String(busy)" />`
})

const flushAsync = async () => {
  await Promise.resolve()
  await Promise.resolve()
  await nextTick()
}

describe("Showcase page", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setLocale("en")
    hoisted.loadHomeState.mockResolvedValue({ nodeId: 7, hubId: 9 })

    ;(globalThis as any).ResizeObserver = class {
      observe() {}
      disconnect() {}
    }
  })

  it("refreshes current vars without leaving the screen subscription set or greying widget cards", async () => {
    const wrapper = mount(Showcase, {
      global: {
        stubs: {
          Button: ButtonStub,
          Badge: BadgeStub,
          Tooltip: TooltipStub,
          Overlay: OverlayStub,
          CardHeader: CardHeaderStub,
          ShowcaseWidgetCardContent: ShowcaseWidgetCardContentStub
        }
      }
    })

    await flushAsync()
    await flushAsync()

    hoisted.showcaseStore.leave.mockClear()
    hoisted.showcaseStore.enterScreen.mockClear()
    hoisted.hydrateSessionConnectionSnapshot.mockClear()
    hoisted.toastStore.success.mockClear()

    let resolveRefresh: (() => void) | null = null
    hoisted.showcaseStore.enterScreen.mockImplementationOnce(
      () =>
        new Promise<void>((resolve) => {
          resolveRefresh = resolve
        })
    )

    const refreshButton = wrapper
      .findAll("button")
      .find((candidate) => candidate.text().includes("Refresh Vars"))
    expect(refreshButton).toBeTruthy()

    await refreshButton!.trigger("click")
    await nextTick()

    expect(hoisted.hydrateSessionConnectionSnapshot).toHaveBeenCalledTimes(1)
    expect(hoisted.showcaseStore.leave).not.toHaveBeenCalled()
    expect(hoisted.showcaseStore.enterScreen).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-test="widget-card"]').attributes("data-busy")).toBe("false")

    resolveRefresh?.()
    await flushAsync()

    expect(hoisted.toastStore.success).toHaveBeenCalled()
  })
})
