// Context: provides the overlay test support code used by the Win desktop host.

// @vitest-environment jsdom

import { nextTick } from "vue"
import { mount } from "@vue/test-utils"
import { afterEach, describe, expect, it } from "vitest"
import { Overlay } from "./"

const createTriggerButton = () => {
  const button = document.createElement("button")
  button.type = "button"
  button.textContent = "trigger"
  document.body.appendChild(button)
  button.focus()
  return button
}

describe("Overlay focus management", () => {
  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("moves focus to the preferred initial element when opened", async () => {
    createTriggerButton()

    mount(Overlay, {
      props: {
        open: true,
        trapFocus: true,
        teleport: false,
        initialFocusSelector: "#preferred"
      },
      slots: {
        default: `
          <div>
            <button id="fallback" type="button">fallback</button>
            <input id="preferred" />
          </div>
        `
      },
      attachTo: document.body
    })

    await nextTick()

    expect((document.activeElement as HTMLElement | null)?.id).toBe("preferred")
  })

  it("cycles focus within the overlay on Tab and Shift+Tab", async () => {
    createTriggerButton()

    const wrapper = mount(Overlay, {
      props: {
        open: true,
        trapFocus: true,
        teleport: false
      },
      slots: {
        default: `
          <div>
            <button id="first" type="button">first</button>
            <button id="last" type="button">last</button>
          </div>
        `
      },
      attachTo: document.body
    })

    await nextTick()

    const first = wrapper.get("#first").element as HTMLButtonElement
    const last = wrapper.get("#last").element as HTMLButtonElement

    last.focus()
    await wrapper.get("#last").trigger("keydown", { key: "Tab" })
    expect(document.activeElement).toBe(first)

    first.focus()
    await wrapper.get("#first").trigger("keydown", { key: "Tab", shiftKey: true })
    expect(document.activeElement).toBe(last)
  })

  it("restores focus to the previous element when closed", async () => {
    const trigger = createTriggerButton()

    const wrapper = mount(Overlay, {
      props: {
        open: true,
        trapFocus: true,
        teleport: false
      },
      slots: {
        default: `
          <div>
            <input id="focus-target" />
          </div>
        `
      },
      attachTo: document.body
    })

    await nextTick()
    expect((document.activeElement as HTMLElement | null)?.id).toBe("focus-target")

    await wrapper.setProps({ open: false })
    await nextTick()

    expect(document.activeElement).toBe(trigger)
  })
})
