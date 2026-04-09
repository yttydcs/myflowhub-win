export type ShowcaseLineChartConfig = {
  rangeMs: number
  bucketMs: number
}

export type ShowcaseLineChartSample = {
  timestamp: number
  value: number
}

export type ShowcaseLineChartPoint = {
  timestamp: number
  value: number
}

export type ShowcaseLineChartBuildResult = {
  status: "empty" | "insufficient" | "ready"
  rangeMs: number
  bucketMs: number
  fromMs: number
  toMs: number
  points: ShowcaseLineChartPoint[]
  yMin: number
  yMax: number
}

export type ShowcaseChartOption = {
  value: number
  label: string
}

export const SHOWCASE_LINE_CHART_RANGE_OPTIONS: ShowcaseChartOption[] = [
  { value: 15 * 60 * 1000, label: "15m" },
  { value: 60 * 60 * 1000, label: "1h" },
  { value: 6 * 60 * 60 * 1000, label: "6h" },
  { value: 24 * 60 * 60 * 1000, label: "24h" }
]

export const SHOWCASE_LINE_CHART_BUCKET_OPTIONS: ShowcaseChartOption[] = [
  { value: 10 * 1000, label: "10s" },
  { value: 60 * 1000, label: "1m" },
  { value: 5 * 60 * 1000, label: "5m" },
  { value: 15 * 60 * 1000, label: "15m" },
  { value: 60 * 60 * 1000, label: "1h" }
]

export const SHOWCASE_LINE_CHART_DEFAULT_RANGE_MS = 60 * 60 * 1000
export const SHOWCASE_LINE_CHART_DEFAULT_BUCKET_MS = 60 * 1000
export const SHOWCASE_LINE_CHART_MAX_RANGE_MS = 24 * 60 * 60 * 1000
export const SHOWCASE_LINE_CHART_MAX_SAMPLES_PER_SERIES = 4096

const clampInt = (value: unknown, fallback: number, min: number, max: number) => {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) return fallback
  const rounded = Math.round(parsed)
  if (rounded < min) return min
  if (rounded > max) return max
  return rounded
}

export const normalizeShowcaseLineChartConfig = (raw?: Partial<ShowcaseLineChartConfig> | null): ShowcaseLineChartConfig => {
  const rangeMs = clampInt(
    raw?.rangeMs,
    SHOWCASE_LINE_CHART_DEFAULT_RANGE_MS,
    SHOWCASE_LINE_CHART_BUCKET_OPTIONS[0]?.value ?? 10 * 1000,
    SHOWCASE_LINE_CHART_MAX_RANGE_MS
  )
  const parsedBucket = Number(raw?.bucketMs)
  const bucketInvalid =
    !Number.isFinite(parsedBucket) ||
    Math.round(parsedBucket) < (SHOWCASE_LINE_CHART_BUCKET_OPTIONS[0]?.value ?? 10 * 1000) ||
    Math.round(parsedBucket) > rangeMs
  const bucketMs = bucketInvalid
    ? Math.min(SHOWCASE_LINE_CHART_DEFAULT_BUCKET_MS, rangeMs)
    : clampInt(parsedBucket, SHOWCASE_LINE_CHART_DEFAULT_BUCKET_MS, SHOWCASE_LINE_CHART_BUCKET_OPTIONS[0]?.value ?? 10 * 1000, rangeMs)
  return {
    rangeMs,
    bucketMs: Math.min(bucketMs, rangeMs)
  }
}

const trimHistoryByRange = (history: ShowcaseLineChartSample[], cutoffMs: number) =>
  history.filter((sample) => Number.isFinite(sample.timestamp) && sample.timestamp >= cutoffMs)

export const appendShowcaseLineChartSample = (
  history: ShowcaseLineChartSample[],
  sample: ShowcaseLineChartSample,
  nowMs = sample.timestamp
): ShowcaseLineChartSample[] => {
  if (!Number.isFinite(sample.timestamp) || !Number.isFinite(sample.value)) {
    return history
  }
  const cutoffMs = nowMs - SHOWCASE_LINE_CHART_MAX_RANGE_MS
  const next = trimHistoryByRange(history, cutoffMs)
  const last = next.at(-1)
  if (last && last.timestamp === sample.timestamp) {
    next[next.length - 1] = sample
  } else {
    next.push(sample)
  }
  if (next.length > SHOWCASE_LINE_CHART_MAX_SAMPLES_PER_SERIES) {
    next.splice(0, next.length - SHOWCASE_LINE_CHART_MAX_SAMPLES_PER_SERIES)
  }
  return next
}

export const ensureShowcaseChartOption = (options: ShowcaseChartOption[], value: number): ShowcaseChartOption[] => {
  if (options.some((option) => option.value === value)) return options
  return [{ value, label: formatDurationLabel(value) }, ...options]
}

export const formatDurationLabel = (durationMs: number) => {
  if (durationMs % (60 * 60 * 1000) === 0) {
    return `${durationMs / (60 * 60 * 1000)}h`
  }
  if (durationMs % (60 * 1000) === 0) {
    return `${durationMs / (60 * 1000)}m`
  }
  if (durationMs % 1000 === 0) {
    return `${durationMs / 1000}s`
  }
  return `${durationMs}ms`
}

export const buildShowcaseLineChart = (
  history: ShowcaseLineChartSample[],
  rawConfig?: Partial<ShowcaseLineChartConfig> | null,
  nowMs = Date.now()
): ShowcaseLineChartBuildResult => {
  const config = normalizeShowcaseLineChartConfig(rawConfig)
  const toMs = Number.isFinite(nowMs) ? Math.round(nowMs) : Date.now()
  const fromMs = toMs - config.rangeMs
  const bucketMap = new Map<number, ShowcaseLineChartSample>()

  for (const sample of history) {
    if (!Number.isFinite(sample.timestamp) || !Number.isFinite(sample.value)) continue
    if (sample.timestamp < fromMs || sample.timestamp > toMs) continue
    const bucketStart = Math.floor(sample.timestamp / config.bucketMs) * config.bucketMs
    bucketMap.set(bucketStart, sample)
  }

  const points = Array.from(bucketMap.entries())
    .sort((a, b) => a[0] - b[0])
    .map(([, sample]) => ({
      timestamp: sample.timestamp,
      value: sample.value
    }))

  if (points.length === 0) {
    return {
      status: "empty",
      rangeMs: config.rangeMs,
      bucketMs: config.bucketMs,
      fromMs,
      toMs,
      points: [],
      yMin: 0,
      yMax: 0
    }
  }

  const values = points.map((point) => point.value)
  const rawMin = Math.min(...values)
  const rawMax = Math.max(...values)
  const pad = rawMin === rawMax ? Math.max(1, Math.abs(rawMin) * 0.1) : Math.max((rawMax - rawMin) * 0.1, 0.1)
  const yMin = rawMin - pad
  const yMax = rawMax + pad

  return {
    status: points.length < 2 ? "insufficient" : "ready",
    rangeMs: config.rangeMs,
    bucketMs: config.bucketMs,
    fromMs,
    toMs,
    points,
    yMin,
    yMax
  }
}
