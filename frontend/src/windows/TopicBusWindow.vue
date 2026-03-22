<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue"
import { useRoute } from "vue-router"
import { EventsOn } from "../../wailsjs/runtime/runtime"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"
import { useProfileStore } from "@/stores/profile"
import { useSessionStore } from "@/stores/session"
import { formatTopicBusTimestamp, normalizeTopicBusEvent, useTopicBusStore, type TopicBusEvent } from "@/stores/topicbus"
import { useToastStore } from "@/stores/toast"
import { HomeState as LoadHomeState } from "../../wailsjs/go/main/App"

const route = useRoute()
const profileStore = useProfileStore()
const sessionStore = useSessionStore()
const topicbus = useTopicBusStore()
const toast = useToastStore()
const { t } = useI18n()

const busy = ref(false)
const eventListRef = ref<HTMLElement | null>(null)
const splitHostRef = ref<HTMLElement | null>(null)

const fallbackIdentity = reactive({
  nodeId: 0,
  hubId: 0
})

const sendForm = reactive({
  topic: "",
  name: "",
  payload: ""
})

const fallbackTitle = computed(() => t("TopicBus"))

const inputClass =
  "mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

const textAreaClass =
  "mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

const connectedLabel = computed(() => (sessionStore.connected ? t("Connected") : t("Disconnected")))
const connectedTone = computed(() =>
  sessionStore.connected ? "bg-emerald-500/15 text-emerald-700" : "bg-rose-500/15 text-rose-700"
)

const selfNodeId = computed(() => sessionStore.auth.nodeId || fallbackIdentity.nodeId || 0)
const hubId = computed(() => sessionStore.auth.hubId || fallbackIdentity.hubId || 0)

const requestedTopic = computed(() => String(route.query.topic ?? "").trim())
const requestedScope = computed(() => String(route.query.scope ?? "").trim().toLowerCase())
const isAllWindow = computed(() => requestedScope.value === "all" || !requestedTopic.value)
const windowTitle = computed(() => (isAllWindow.value ? t("All Channels") : requestedTopic.value || fallbackTitle.value))
const windowModeLabel = computed(() => (isAllWindow.value ? t("Aggregate Window") : t("Channel Window")))
const resolvedTopicLabel = computed(() => (isAllWindow.value ? t("All known topics") : requestedTopic.value || "-"))
const resolvedTargetLabel = computed(() => topicbus.state.targetId.trim() || (hubId.value ? String(hubId.value) : "-"))
const receivePanelPercent = computed(() => `${Math.round(splitRatio.value * 100)}%`)

const localEvents = ref<TopicBusEvent[]>([])
const selectedEventIndex = ref(-1)
const selectedEvent = computed(() => localEvents.value[selectedEventIndex.value] ?? null)
const infoItems = computed(() => [
  { label: t("Topic"), value: resolvedTopicLabel.value },
  { label: t("Self"), value: selfNodeId.value ? String(selfNodeId.value) : "-" },
  { label: t("Target"), value: resolvedTargetLabel.value },
  { label: t("Events"), value: String(localEvents.value.length) },
  { label: t("Cache"), value: String(topicbus.state.maxEvents) },
  { label: t("Receive"), value: receivePanelPercent.value }
])

const splitRatio = ref(0.62)
const resizeState = reactive({
  active: false,
  top: 0,
  height: 1
})

const trimRatio = (value: number) => Math.min(0.82, Math.max(0.28, value))
const panelBasisStyle = computed(() => ({ flexBasis: `${Math.round(splitRatio.value * 1000) / 10}%` }))

const acceptsEvent = (event: TopicBusEvent) => {
  if (isAllWindow.value) return true
  return event.topic === requestedTopic.value
}

const previewPayload = (raw: string) => {
  const compact = String(raw ?? "").replace(/\s+/g, " ").trim()
  if (!compact) return t("No payload.")
  if (compact.length <= 180) return compact
  return `${compact.slice(0, 180)}...`
}

const applyInitialTarget = () => {
  const queryTarget = String(route.query.targetId ?? "").trim()
  if (queryTarget) {
    topicbus.state.targetId = queryTarget
    return
  }
  if (!topicbus.state.targetId && hubId.value) {
    topicbus.state.targetId = String(hubId.value)
  }
}

const syncTopicDraft = () => {
  if (isAllWindow.value) {
    sendForm.topic = ""
    return
  }
  sendForm.topic = requestedTopic.value
}

const ensureReady = () => {
  if (!sessionStore.connected) {
    throw new Error(t("Connect to a session before sending TopicBus requests."))
  }
  if (!selfNodeId.value) {
    throw new Error(t("Login to a node before using TopicBus operations."))
  }
}

const loadHomeDefaults = async () => {
  try {
    const state = await LoadHomeState()
    fallbackIdentity.nodeId = Number(state?.nodeId ?? 0)
    fallbackIdentity.hubId = Number(state?.hubId ?? 0)
  } catch (err) {
    console.warn(err)
  }
  topicbus.setIdentity(selfNodeId.value, hubId.value)
  applyInitialTarget()
  syncTopicDraft()
}

const clearComposer = () => {
  if (isAllWindow.value) {
    sendForm.topic = ""
  } else {
    sendForm.topic = requestedTopic.value
  }
  sendForm.name = ""
  sendForm.payload = ""
}

const clearLocalEvents = () => {
  pendingEvents.length = 0
  if (flushTimer !== null) {
    window.clearTimeout(flushTimer)
    flushTimer = null
  }
  localEvents.value = []
  selectedEventIndex.value = -1
}

const scrollToLatest = async () => {
  await nextTick()
  if (eventListRef.value) {
    eventListRef.value.scrollTop = eventListRef.value.scrollHeight
  }
}

const trimLocalEvents = () => {
  const limit = Number(topicbus.state.maxEvents || 0) > 0 ? Number(topicbus.state.maxEvents) : 500
  if (localEvents.value.length <= limit) return
  localEvents.value = localEvents.value.slice(-limit)
  if (selectedEventIndex.value >= localEvents.value.length) {
    selectedEventIndex.value = localEvents.value.length - 1
  }
}

const pendingEvents: TopicBusEvent[] = []
let flushTimer: number | null = null
let lastFlushAt = 0

const flushPending = () => {
  if (!pendingEvents.length) return
  localEvents.value.push(...pendingEvents)
  pendingEvents.length = 0
  trimLocalEvents()
  if (selectedEventIndex.value < 0 && localEvents.value.length > 0) {
    selectedEventIndex.value = localEvents.value.length - 1
  }
  lastFlushAt = Date.now()
}

const scheduleFlush = () => {
  const now = Date.now()
  const elapsed = now - lastFlushAt
  if (elapsed >= 200) {
    flushPending()
    return
  }
  if (flushTimer !== null) return
  flushTimer = window.setTimeout(() => {
    flushTimer = null
    flushPending()
  }, Math.max(0, 200 - elapsed))
}

const queueEvent = (event: TopicBusEvent) => {
  pendingEvents.push(event)
  scheduleFlush()
}

let stopTopicBusListener: (() => void) | null = null

const onTopicBusRuntimeEvent = (evt: any) => {
  const normalized = normalizeTopicBusEvent(evt)
  if (!normalized) return
  if (!acceptsEvent(normalized)) return
  queueEvent(normalized)
}

const attachTopicBusListener = () => {
  const maybeStop = EventsOn("topicbus.event", onTopicBusRuntimeEvent)
  stopTopicBusListener = typeof maybeStop === "function" ? maybeStop : null
}

const detachTopicBusListener = () => {
  stopTopicBusListener?.()
  stopTopicBusListener = null
}

const stopResizeListeners = () => {
  window.removeEventListener("pointermove", onResizePointerMove)
  window.removeEventListener("pointerup", onResizePointerUp)
  window.removeEventListener("pointercancel", onResizePointerUp)
}

const startResize = (event: PointerEvent) => {
  const host = splitHostRef.value
  if (!host) return
  const rect = host.getBoundingClientRect()
  resizeState.active = true
  resizeState.top = rect.top
  resizeState.height = rect.height || 1
  stopResizeListeners()
  window.addEventListener("pointermove", onResizePointerMove)
  window.addEventListener("pointerup", onResizePointerUp)
  window.addEventListener("pointercancel", onResizePointerUp)
  event.preventDefault()
}

const onResizePointerMove = (event: PointerEvent) => {
  if (!resizeState.active) return
  const raw = (event.clientY - resizeState.top) / resizeState.height
  splitRatio.value = trimRatio(raw)
}

const onResizePointerUp = () => {
  resizeState.active = false
  stopResizeListeners()
}

const publishEvent = async () => {
  if (busy.value) return
  busy.value = true
  try {
    ensureReady()
    const topic = isAllWindow.value ? sendForm.topic : requestedTopic.value
    await topicbus.publish(topic, sendForm.name, sendForm.payload)
    sendForm.name = ""
    sendForm.payload = ""
    toast.success(t("Event published. This window will not echo your own message."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to publish event."))
  } finally {
    busy.value = false
  }
}

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  () => {
    topicbus.setIdentity(selfNodeId.value, hubId.value)
    applyInitialTarget()
  }
)

watch(
  () => [requestedTopic.value, requestedScope.value],
  () => {
    clearLocalEvents()
    syncTopicDraft()
  }
)

watch(
  () => profileStore.state.current,
  async () => {
    clearLocalEvents()
    await topicbus.loadPrefs()
    await loadHomeDefaults()
  }
)

watch(
  () => localEvents.value.length,
  () => {
    void scrollToLatest()
  }
)

onMounted(async () => {
  try {
    await topicbus.loadPrefs()
    await loadHomeDefaults()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to initialize TopicBus window."))
  }
  attachTopicBusListener()
})

onBeforeUnmount(() => {
  if (flushTimer !== null) {
    window.clearTimeout(flushTimer)
    flushTimer = null
  }
  pendingEvents.length = 0
  detachTopicBusListener()
  stopResizeListeners()
})
</script>

<template>
  <section class="flex h-full min-h-0 flex-col bg-card/70 text-card-foreground">
    <header class="flex-none border-b border-border/60 bg-card/92 px-5 py-4 shadow-sm">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
            {{ t("TopicBus Window") }}
          </p>
          <h1 class="mt-1 text-xl font-semibold">{{ windowTitle }}</h1>
          <p class="mt-2 text-sm text-muted-foreground">
            {{
              isAllWindow
                ? t("Watch every known topic from the moment this window opens.")
                : t("Focus on one topic with a dedicated receive and send workspace.")
            }}
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <Badge variant="outline">{{ windowModeLabel }}</Badge>
          <Badge :class="connectedTone">{{ connectedLabel }}</Badge>
        </div>
      </div>
    </header>

    <div class="flex-1 min-h-0 overflow-hidden">
      <div class="grid h-full min-h-0 gap-4 p-4 xl:grid-cols-[minmax(0,1fr)_320px]">
        <section
          ref="splitHostRef"
          class="flex min-h-0 min-w-0 flex-col overflow-hidden rounded-2xl border bg-card/90 p-3 text-card-foreground shadow-sm"
          :class="resizeState.active ? 'select-none' : ''"
        >
          <section class="min-h-0 shrink-0" :style="panelBasisStyle">
            <div class="flex h-full min-h-0 flex-col overflow-hidden rounded-xl border border-border/60 bg-background/70">
              <div class="flex flex-wrap items-center justify-between gap-3 border-b border-border/60 px-4 py-3">
                <div>
                  <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">{{ t("Receive") }}</p>
                  <h2 class="text-sm font-semibold">{{ t("Live Event Stream") }}</h2>
                  <p class="mt-1 text-xs text-muted-foreground">
                    {{ t("New events only. Select one item to inspect it from the side panel.") }}
                  </p>
                </div>
                <Badge variant="outline">{{ t("{count} events", { count: localEvents.length }) }}</Badge>
              </div>

              <div ref="eventListRef" class="min-h-0 flex-1 space-y-3 overflow-y-auto p-4">
                <button
                  v-for="(event, index) in localEvents"
                  :key="`${event.topic}-${event.name}-${event.ts}-${index}`"
                  type="button"
                  class="w-full rounded-xl border border-border/60 bg-card/90 px-4 py-3 text-left transition hover:border-primary/40"
                  :class="selectedEventIndex === index ? 'border-primary/50 bg-primary/10' : ''"
                  @click="selectedEventIndex = index"
                >
                  <div class="flex flex-wrap items-center gap-2">
                    <Badge variant="outline">{{ event.topic }}</Badge>
                    <span class="text-sm font-semibold text-foreground">{{ event.name }}</span>
                    <span class="text-xs text-muted-foreground">{{ formatTopicBusTimestamp(event.ts) || "-" }}</span>
                  </div>
                  <p class="mt-2 text-xs text-muted-foreground">
                    {{ previewPayload(event.dataRaw) }}
                  </p>
                </button>

                <div
                  v-if="localEvents.length === 0"
                  class="flex h-full min-h-[220px] items-center justify-center rounded-xl border border-dashed border-border/60 bg-background/60 px-6 text-center text-sm text-muted-foreground"
                >
                  {{ t("Waiting for new TopicBus events in this window.") }}
                </div>
              </div>
            </div>
          </section>

          <div
            class="group my-2 flex h-3 shrink-0 cursor-row-resize items-center justify-center rounded-full bg-muted/60 transition hover:bg-muted"
            @pointerdown="startResize"
          >
            <div class="h-1 w-16 rounded-full bg-border/80 transition group-hover:bg-primary/40" />
          </div>

          <section class="min-h-0 flex-1">
            <div class="flex h-full min-h-0 flex-col overflow-hidden rounded-xl border border-border/60 bg-background/70">
              <div class="border-b border-border/60 px-4 py-3">
                <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">{{ t("Send") }}</p>
                <h2 class="text-sm font-semibold">{{ t("Publish Event") }}</h2>
              </div>

              <div class="min-h-0 flex-1 overflow-y-auto p-4">
                <div class="grid gap-4">
                  <div class="grid gap-4 lg:grid-cols-[1fr_1fr]">
                    <div>
                      <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                        {{ t("Target Node ID") }}
                      </label>
                      <input
                        v-model="topicbus.state.targetId"
                        :class="inputClass"
                        :placeholder="hubId ? String(hubId) : t('Hub NodeID')"
                      />
                    </div>
                    <div>
                      <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                        {{ t("Topic") }}
                      </label>
                      <input
                        v-model="sendForm.topic"
                        :class="inputClass"
                        :placeholder="isAllWindow ? t('topic.status') : ''"
                        :readonly="!isAllWindow"
                      />
                    </div>
                  </div>

                  <div>
                    <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                      {{ t("Name") }}
                    </label>
                    <input v-model="sendForm.name" :class="inputClass" :placeholder="t('event name')" />
                  </div>

                  <div class="min-h-0">
                    <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                      {{ t("Payload") }}
                    </label>
                    <textarea
                      v-model="sendForm.payload"
                      :class="textAreaClass"
                      rows="6"
                      :placeholder="t('JSON or plain text')"
                    />
                  </div>

                  <div class="flex flex-wrap items-center justify-between gap-3">
                    <p class="text-xs text-muted-foreground">
                      {{
                        isAllWindow
                          ? t("Choose a topic before sending from the aggregate window.")
                          : t("Topic is locked to this channel for safer publishing.")
                      }}
                    </p>
                    <Button :disabled="busy" @click="publishEvent">{{ t("Publish") }}</Button>
                  </div>
                </div>
              </div>
            </div>
          </section>
        </section>

        <aside class="flex min-h-0 min-w-0 flex-col gap-4 xl:overflow-hidden">
          <section class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
            <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">{{ t("Other Info") }}</p>
            <h2 class="mt-2 text-lg font-semibold">{{ t("Window Snapshot") }}</h2>

            <div class="mt-4 grid gap-3">
              <div
                v-for="item in infoItems"
                :key="item.label"
                class="rounded-xl border border-border/60 bg-background/70 px-3 py-2"
              >
                <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ item.label }}</p>
                <p class="mt-1 text-sm font-medium text-foreground">{{ item.value }}</p>
              </div>
            </div>

            <div class="mt-4 rounded-xl border border-border/60 bg-background/70 px-4 py-3">
              <div class="flex flex-wrap items-center gap-2">
                <Badge v-if="selectedEvent" variant="outline">{{ selectedEvent.topic }}</Badge>
                <span class="text-xs text-muted-foreground">
                  {{ selectedEvent ? formatTopicBusTimestamp(selectedEvent.ts) || "-" : t("No event selected") }}
                </span>
              </div>
              <p class="mt-2 text-sm font-semibold text-foreground">
                {{ selectedEvent ? selectedEvent.name : t("Selected Event") }}
              </p>
              <pre
                v-if="selectedEvent"
                class="mt-3 max-h-64 overflow-auto whitespace-pre-wrap break-all rounded-lg border border-border/60 bg-card/90 p-3 text-xs text-muted-foreground"
              >
{{ selectedEvent.dataRaw }}
              </pre>
              <p v-else class="mt-2 text-sm text-muted-foreground">
                {{ t("Select one event from the receive list to inspect its full payload here.") }}
              </p>
            </div>
          </section>

          <section class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm xl:flex-1">
            <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">{{ t("Control Buttons") }}</p>
            <h2 class="mt-2 text-lg font-semibold">{{ t("Window Actions") }}</h2>
            <p class="mt-2 text-sm text-muted-foreground">
              {{ t("Keep the main workspace focused on traffic and publishing. Use this side panel for quick actions.") }}
            </p>

            <div class="mt-4 grid gap-3">
              <Button variant="outline" class="justify-start" @click="scrollToLatest">{{ t("Scroll to Latest") }}</Button>
              <Button variant="outline" class="justify-start" @click="clearLocalEvents">{{ t("Clear Receive List") }}</Button>
              <Button variant="outline" class="justify-start" @click="clearComposer">{{ t("Reset Draft") }}</Button>
            </div>

            <div class="mt-4 rounded-xl border border-border/60 bg-background/70 px-4 py-3 text-sm text-muted-foreground">
              {{ t("Your own publish action will not echo back into this window. Open another subscriber if you need to observe your outbound event.") }}
            </div>
          </section>
        </aside>
      </div>
    </div>
  </section>
</template>
