import { describe, expect, it } from "vitest"
import {
  appendShowcaseLineChartSample,
  buildShowcaseLineChart,
  normalizeShowcaseLineChartConfig,
  SHOWCASE_LINE_CHART_DEFAULT_BUCKET_MS,
  SHOWCASE_LINE_CHART_DEFAULT_RANGE_MS,
  SHOWCASE_LINE_CHART_MAX_RANGE_MS
} from "./showcaseChart"

describe("showcaseChart", () => {
  it("normalizes invalid config back to safe defaults", () => {
    expect(normalizeShowcaseLineChartConfig()).toEqual({
      rangeMs: SHOWCASE_LINE_CHART_DEFAULT_RANGE_MS,
      bucketMs: SHOWCASE_LINE_CHART_DEFAULT_BUCKET_MS
    })

    expect(
      normalizeShowcaseLineChartConfig({
        rangeMs: 5 * 60 * 1000,
        bucketMs: 10 * 60 * 1000
      })
    ).toEqual({
      rangeMs: 5 * 60 * 1000,
      bucketMs: SHOWCASE_LINE_CHART_DEFAULT_BUCKET_MS
    })
  })

  it("trims old history while appending numeric samples", () => {
    const nowMs = Date.UTC(2026, 3, 9, 12, 0, 0)
    const history = [
      { timestamp: nowMs - SHOWCASE_LINE_CHART_MAX_RANGE_MS - 1, value: 1 },
      { timestamp: nowMs - 60_000, value: 2 }
    ]
    const next = appendShowcaseLineChartSample(history, { timestamp: nowMs, value: 3 }, nowMs)
    expect(next).toEqual([
      { timestamp: nowMs - 60_000, value: 2 },
      { timestamp: nowMs, value: 3 }
    ])
  })

  it("aggregates samples by bucket and builds a ready trend line", () => {
    const nowMs = Date.UTC(2026, 3, 9, 12, 0, 0)
    const result = buildShowcaseLineChart(
      [
        { timestamp: nowMs - 55_000, value: 10 },
        { timestamp: nowMs - 30_000, value: 12 },
        { timestamp: nowMs - 20_000, value: 15 },
        { timestamp: nowMs - 5_000, value: 11 }
      ],
      {
        rangeMs: 60_000,
        bucketMs: 10_000
      },
      nowMs
    )

    expect(result.status).toBe("ready")
    expect(result.points).toHaveLength(4)
    expect(result.points.map((point) => point.value)).toEqual([10, 12, 15, 11])
    expect(result.yMax).toBeGreaterThan(result.yMin)
  })

  it("reports insufficient when fewer than two visible buckets remain", () => {
    const nowMs = Date.UTC(2026, 3, 9, 12, 0, 0)
    const result = buildShowcaseLineChart(
      [{ timestamp: nowMs - 5_000, value: 42 }],
      {
        rangeMs: 60_000,
        bucketMs: 10_000
      },
      nowMs
    )

    expect(result.status).toBe("insufficient")
    expect(result.points).toHaveLength(1)
  })
})
