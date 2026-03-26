// @vitest-environment jsdom

import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it } from "vitest"
import { setLocale } from "@/i18n"
import FlowNode from "./FlowNode.vue"

const mountFlowNode = (data: Record<string, unknown>) =>
  mount(FlowNode, {
    props: {
      id: "node-1",
      data,
      selected: false,
      connectable: true,
      targetPosition: "left",
      sourcePosition: "right"
    },
    global: {
      stubs: {
        Handle: {
          template: "<div />"
        }
      }
    }
  })

describe("FlowNode", () => {
  beforeEach(() => {
    setLocale("en")
  })

  it("renders cancelled status badges", () => {
    const wrapper = mountFlowNode({
      label: "node-1",
      kind: "call",
      meta: "demo::call",
      status: {
        status: "cancelled",
        code: 408,
        msg: "timeout"
      }
    })

    expect(wrapper.text()).toContain("Cancelled")
    expect(wrapper.text()).toContain("Code 408")
    expect(wrapper.text()).toContain("timeout")
  })

  it("renders set_var nodes with the set var label", () => {
    const wrapper = mountFlowNode({
      label: "set1",
      kind: "set_var",
      meta: "session_token"
    })

    expect(wrapper.text()).toContain("Set Var")
    expect(wrapper.text()).toContain("session_token")
  })
})
