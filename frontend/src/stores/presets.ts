import { reactive } from "vue"
import { EventsOn } from "../../wailsjs/runtime/runtime"

type WailsBinding = (...args: any[]) => Promise<any>

const callPresets = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.presets?.PresetService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(`Preset binding '${method}' unavailable`)
  }
  return fn(...args)
}

export type TopicStressConfig = {
  sourceId: number
  targetId: number
  topic: string
  runId: string
  total: number
  payloadSize: number
  maxPerSec: number
}

export type TopicStressSenderStatus = {
  active: boolean
  topic: string
  runId: string
  total: number
  payloadSize: number
  maxPerSec: number
  sent: number
  errors: number
  startedAt: string
  updatedAt: string
}

export type TopicStressReceiverStatus = {
  active: boolean
  topic: string
  runId: string
  expected: number
  payloadSize: number
  rx: number
  unique: number
  dup: number
  corrupt: number
  invalid: number
  outOfOrder: number
  lastSeq: number
  startedAt: string
  updatedAt: string
}

type PresetState = {
  sender: TopicStressSenderStatus
  receiver: TopicStressReceiverStatus
}

const defaultSender: TopicStressSenderStatus = {
  active: false,
  topic: "",
  runId: "",
  total: 0,
  payloadSize: 0,
  maxPerSec: 0,
  sent: 0,
  errors: 0,
  startedAt: "",
  updatedAt: ""
}

const defaultReceiver: TopicStressReceiverStatus = {
  active: false,
  topic: "",
  runId: "",
  expected: 0,
  payloadSize: 0,
  rx: 0,
  unique: 0,
  dup: 0,
  corrupt: 0,
  invalid: 0,
  outOfOrder: 0,
  lastSeq: 0,
  startedAt: "",
  updatedAt: ""
}

const state = reactive<PresetState>({
  sender: { ...defaultSender },
  receiver: { ...defaultReceiver }
})

let initialized = false

const toIso = (value: any) => {
  if (!value) return ""
  if (typeof value === "string") return value
  try {
    return new Date(value).toISOString()
  } catch {
    return ""
  }
}

const mapSender = (data: any): TopicStressSenderStatus => ({
  active: Boolean(data?.active ?? false),
  topic: String(data?.topic ?? ""),
  runId: String(data?.runId ?? data?.run_id ?? ""),
  total: Number(data?.total ?? 0),
  payloadSize: Number(data?.payloadSize ?? data?.payload_size ?? 0),
  maxPerSec: Number(data?.maxPerSec ?? data?.max_per_sec ?? 0),
  sent: Number(data?.sent ?? 0),
  errors: Number(data?.errors ?? 0),
  startedAt: toIso(data?.startedAt ?? data?.started_at),
  updatedAt: toIso(data?.updatedAt ?? data?.updated_at)
})

const mapReceiver = (data: any): TopicStressReceiverStatus => ({
  active: Boolean(data?.active ?? false),
  topic: String(data?.topic ?? ""),
  runId: String(data?.runId ?? data?.run_id ?? ""),
  expected: Number(data?.expected ?? 0),
  payloadSize: Number(data?.payloadSize ?? data?.payload_size ?? 0),
  rx: Number(data?.rx ?? 0),
  unique: Number(data?.unique ?? 0),
  dup: Number(data?.dup ?? 0),
  corrupt: Number(data?.corrupt ?? 0),
  invalid: Number(data?.invalid ?? 0),
  outOfOrder: Number(data?.outOfOrder ?? data?.out_of_order ?? 0),
  lastSeq: Number(data?.lastSeq ?? data?.last_seq ?? 0),
  startedAt: toIso(data?.startedAt ?? data?.started_at),
  updatedAt: toIso(data?.updatedAt ?? data?.updated_at)
})

const loadSender = async () => {
  const status = await callPresets<TopicStressSenderStatus>("TopicStressSenderState")
  state.sender = mapSender(status)
}

const loadReceiver = async () => {
  const status = await callPresets<TopicStressReceiverStatus>("TopicStressReceiverState")
  state.receiver = mapReceiver(status)
}

const startSender = async (cfg: TopicStressConfig) => {
  await callPresets("StartTopicStressSender", cfg)
}

const stopSender = async () => {
  await callPresets("StopTopicStressSender")
}

const startReceiver = async (cfg: TopicStressConfig) => {
  await callPresets("StartTopicStressReceiver", cfg)
}

const stopReceiver = async () => {
  await callPresets("StopTopicStressReceiver")
}

const resetReceiver = async () => {
  await callPresets("ResetTopicStressReceiver")
}

const ensureListeners = () => {
  if (initialized) return
  initialized = true

  EventsOn("presets.topic_stress.sender", (evt: any) => {
    state.sender = mapSender(evt)
  })
  EventsOn("presets.topic_stress.receiver", (evt: any) => {
    state.receiver = mapReceiver(evt)
  })
}

export const usePresetsStore = () => {
  ensureListeners()
  return {
    state,
    loadReceiver,
    loadSender,
    resetReceiver,
    startReceiver,
    startSender,
    stopReceiver,
    stopSender
  }
}
