// @vitest-environment jsdom

import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"

const getVarValueTextMock = vi.fn()
const lineChartStateMock = vi.fn()
const resolveEffectiveModeMock = vi.fn()
const sliderValueMock = vi.fn()

vi.mock("@/stores/showcase", () => ({
  useShowcaseStore: () => ({
    getVarValueText: getVarValueTextMock,
    lineChartState: lineChartStateMock,
    resolveEffectiveMode: resolveEffectiveModeMock,
    sliderValue: sliderValueMock
  })
}))

import ShowcaseWidgetCardContent from "./ShowcaseWidgetCardContent.vue"

const lineChartWidget = {
  id: "widget-line",
  kind: "var",
  title: "Temperature",
  targetId: 9,
  layout: { colSpan: 1 },
  var: {
    ownerId: 7,
    name: "temperature",
    mode: "line_chart",
    visibility: "public",
    type: "float64",
    slider: { min: 0, max: 100, step: 1, throttleMs: 50 },
    switch: { onValue: "true", offValue: "false" },
    chart: { rangeMs: 60 * 60 * 1000, bucketMs: 60 * 1000 }
  }
} as const

describe("ShowcaseWidgetCardContent", () => {
  it("renders line chart controls and forwards local range overrides", async () => {
    getVarValueTextMock.mockReturnValue("42.5")
    resolveEffectiveModeMock.mockReturnValue("line_chart")
    sliderValueMock.mockReturnValue(0)
    lineChartStateMock.mockImplementation((_widget, overrides) => ({
      status: "ready",
      ready: true,
      message: "",
      chart: {
        rangeMs: overrides?.rangeMs ?? 60 * 60 * 1000,
        bucketMs: overrides?.bucketMs ?? 60 * 1000
      },
      fromMs: Date.UTC(2026, 3, 9, 11, 0, 0),
      toMs: Date.UTC(2026, 3, 9, 12, 0, 0),
      points: [
        { timestamp: Date.UTC(2026, 3, 9, 11, 0, 0), value: 39 },
        { timestamp: Date.UTC(2026, 3, 9, 11, 30, 0), value: 41 },
        { timestamp: Date.UTC(2026, 3, 9, 12, 0, 0), value: 42.5 }
      ],
      yMin: 38,
      yMax: 43,
      latestValue: "42.5"
    }))

    const wrapper = mount(ShowcaseWidgetCardContent, {
      props: {
        widget: lineChartWidget,
        connected: true,
        selfNodeId: 7
      }
    })

    expect(wrapper.text()).toContain("Temperature")
    expect(wrapper.text()).toContain("Range")
    expect(wrapper.text()).toContain("Granularity")
    expect(wrapper.find("svg").exists()).toBe(true)

    const selects = wrapper.findAll("select")
    expect(selects).toHaveLength(2)
    await selects[0]!.setValue(String(15 * 60 * 1000))

    const lastCall = lineChartStateMock.mock.calls.at(-1)
    expect(lastCall?.[1]).toMatchObject({ rangeMs: 15 * 60 * 1000 })
  })

  it("shows the fallback message when chart samples are insufficient", () => {
    getVarValueTextMock.mockReturnValue("42.5")
    resolveEffectiveModeMock.mockReturnValue("line_chart")
    sliderValueMock.mockReturnValue(0)
    lineChartStateMock.mockReturnValue({
      status: "insufficient",
      ready: false,
      message: "Need more samples to draw trend.",
      chart: { rangeMs: 60 * 60 * 1000, bucketMs: 60 * 1000 },
      fromMs: Date.UTC(2026, 3, 9, 11, 0, 0),
      toMs: Date.UTC(2026, 3, 9, 12, 0, 0),
      points: [{ timestamp: Date.UTC(2026, 3, 9, 12, 0, 0), value: 42.5 }],
      yMin: 41,
      yMax: 43,
      latestValue: "42.5"
    })

    const wrapper = mount(ShowcaseWidgetCardContent, {
      props: {
        widget: lineChartWidget
      }
    })

    expect(wrapper.text()).toContain("Need more samples to draw trend.")
    expect(wrapper.find("svg").exists()).toBe(false)
  })
})
