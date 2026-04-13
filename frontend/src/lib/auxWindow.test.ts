// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from "vitest"

const { environmentMock } = vi.hoisted(() => ({
  environmentMock: vi.fn()
}))

vi.mock("../../wailsjs/runtime/runtime", () => ({
  Environment: environmentMock
}))

import { openAuxWindow, resetAuxWindowRuntimeCacheForTests } from "./auxWindow"

describe("auxWindow", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    resetAuxWindowRuntimeCacheForTests()
    window.history.replaceState({}, "", "/#/home")
    delete (window as any).runtime
  })

  it("navigates in the current window for packaged Wails runtime", async () => {
    ;(window as any).runtime = {}
    environmentMock.mockResolvedValue({
      buildType: "production",
      platform: "windows",
      arch: "amd64"
    })
    const openSpy = vi.spyOn(window, "open").mockReturnValue(null)

    await expect(
      openAuxWindow({
        routePath: "#/topicbus-window?topic=status",
        name: "topicbus_status",
        size: "width=1080,height=760"
      })
    ).resolves.toBe("navigated")

    expect(window.location.hash).toBe("#/topicbus-window?topic=status")
    expect(openSpy).not.toHaveBeenCalled()
  })

  it("opens a new window in dev runtime", async () => {
    ;(window as any).runtime = {}
    environmentMock.mockResolvedValue({
      buildType: "dev",
      platform: "windows",
      arch: "amd64"
    })
    const focus = vi.fn()
    const openSpy = vi.spyOn(window, "open").mockReturnValue({
      focus
    } as unknown as Window)

    await expect(
      openAuxWindow({
        routePath: "#/log-window",
        name: "log_window",
        size: "width=980,height=720"
      })
    ).resolves.toBe("opened")

    expect(openSpy).toHaveBeenCalledWith(
      "http://localhost:3000/#/log-window",
      "log_window",
      "width=980,height=720"
    )
    expect(focus).toHaveBeenCalledTimes(1)
  })

  it("returns blocked when browser popup policy rejects the new window", async () => {
    const openSpy = vi.spyOn(window, "open").mockReturnValue(null)

    await expect(
      openAuxWindow({
        routePath: "#/showcase-window?screenId=default",
        name: "showcase_default",
        size: "width=980,height=720"
      })
    ).resolves.toBe("blocked")

    expect(openSpy).toHaveBeenCalledTimes(1)
    expect(environmentMock).not.toHaveBeenCalled()
  })

  it("rejects invalid auxiliary routes", async () => {
    await expect(
      openAuxWindow({
        routePath: "/logs",
        name: "invalid",
        size: "width=1,height=1"
      })
    ).rejects.toThrow("Aux window route must start with '#/'.")
  })
})
