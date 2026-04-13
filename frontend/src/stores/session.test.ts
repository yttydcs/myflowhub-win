// Context: covers shared session snapshot hydration in the Win frontend.

// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from "vitest"

const runtimeState = vi.hoisted(() => ({
  EventsOn: vi.fn(() => undefined)
}))

const sessionBindings = vi.hoisted(() => ({
  IsConnected: vi.fn(),
  LastAddr: vi.fn()
}))

vi.mock("../../wailsjs/runtime/runtime", () => ({
  EventsOn: runtimeState.EventsOn
}))

vi.mock("../../wailsjs/go/session/SessionService", () => ({
  IsConnected: sessionBindings.IsConnected,
  LastAddr: sessionBindings.LastAddr
}))

import { hydrateSessionConnectionSnapshot, useSessionStore } from "./session"

const store = useSessionStore()

const resetStore = () => {
  store.connected = false
  store.addr = ""
  store.lastStateAt = ""
  store.lastError = ""
  store.lastErrorAt = ""
  store.lastFrameAt = ""
  store.auth.deviceId = ""
  store.auth.nodeId = 0
  store.auth.hubId = 0
  store.auth.role = ""
  store.auth.loggedIn = false
  store.auth.lastAuthMessage = ""
  store.auth.lastAuthAction = ""
  store.auth.lastAuthAt = ""
}

describe("session store", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    resetStore()
  })

  it("hydrates the connected snapshot from runtime bindings", async () => {
    sessionBindings.IsConnected.mockResolvedValue(true)
    sessionBindings.LastAddr.mockResolvedValue("127.0.0.1:9000")

    await hydrateSessionConnectionSnapshot()

    expect(store.connected).toBe(true)
    expect(store.addr).toBe("127.0.0.1:9000")
    expect(store.lastStateAt).not.toBe("")
    expect(sessionBindings.LastAddr).toHaveBeenCalledTimes(1)
  })

  it("marks the session disconnected without overwriting the remembered addr", async () => {
    store.addr = "127.0.0.1:9000"
    store.auth.loggedIn = true
    sessionBindings.IsConnected.mockResolvedValue(false)

    await hydrateSessionConnectionSnapshot()

    expect(store.connected).toBe(false)
    expect(store.addr).toBe("127.0.0.1:9000")
    expect(store.auth.loggedIn).toBe(false)
    expect(store.lastStateAt).not.toBe("")
    expect(sessionBindings.LastAddr).not.toHaveBeenCalled()
  })
})
