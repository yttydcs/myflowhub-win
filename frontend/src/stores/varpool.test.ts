// 本文件覆盖 `varpool` store 在缺省字段响应下的缓存合并行为。

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

import { useVarPoolStore } from "./varpool"

const store = useVarPoolStore()
const getSimple = vi.fn()

beforeEach(() => {
  setLocale("en")
  getSimple.mockReset()

  store.state.targetId = ""
  store.state.selfNodeId = 0
  store.state.defaultTargetId = 0
  store.state.keys = []
  store.state.data = {}
  store.state.lastFrameAt = ""

  ;(window as any).go = {
    main: {
      App: {}
    },
    varpool: {
      VarPoolService: {
        GetSimple: getSimple
      }
    }
  }

  store.setIdentity(7, 9)
  store.updateValue(
    { name: "dht11_gpio8_temperature_c", owner: 4 },
    {
      value: "25",
      owner: 4,
      visibility: "public",
      kind: "int"
    }
  )
})

describe("varpool store", () => {
  it("preserves cached value when get_resp omits the value field", async () => {
    getSimple.mockResolvedValueOnce({
      code: 1,
      msg: "ok",
      name: "dht11_gpio8_temperature_c",
      owner: 4,
      visibility: "public",
      type: "int"
    })

    await store.getVar({ name: "dht11_gpio8_temperature_c", owner: 4 })

    expect(getSimple).toHaveBeenCalledWith(7, 9, {
      name: "dht11_gpio8_temperature_c",
      owner: 4
    })
    expect(store.valueForKey({ name: "dht11_gpio8_temperature_c", owner: 4 })).toMatchObject({
      value: "25",
      visibility: "public",
      kind: "int"
    })
  })
})
