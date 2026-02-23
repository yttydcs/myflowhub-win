<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useProfileStore } from "@/stores/profile"
import { useSessionStore } from "@/stores/session"
import { useTopicBusStore, type TopicBusEvent } from "@/stores/topicbus"
import { useToastStore } from "@/stores/toast"
import { HomeState as LoadHomeState } from "../../wailsjs/go/main/App"

const profileStore = useProfileStore()
const sessionStore = useSessionStore()
const topicbus = useTopicBusStore()
const toast = useToastStore()

const busy = ref(false)

const subForm = reactive({
  text: ""
})

const publishForm = reactive({
  topic: "",
  name: "",
  payload: ""
})

const maxEventsInput = ref(String(topicbus.state.maxEvents || 500))

const fallbackIdentity = reactive({
  nodeId: 0,
  hubId: 0
})

const inputClass =
  "mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

const textAreaClass =
  "mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

const connectedLabel = computed(() => (sessionStore.connected ? "Connected" : "Disconnected"))
const connectedTone = computed(() =>
  sessionStore.connected ? "bg-emerald-500/15 text-emerald-700" : "bg-rose-500/15 text-rose-700"
)

const selfNodeId = computed(() => sessionStore.auth.nodeId || fallbackIdentity.nodeId || 0)
const hubId = computed(() => sessionStore.auth.hubId || fallbackIdentity.hubId || 0)

const filteredEvents = computed(() => {
  const selected = topicbus.state.selectedTopic
  if (!selected) return topicbus.state.events
  return topicbus.state.events.filter((ev) => ev.topic === selected)
})

const selectedEventIndex = ref(-1)
const selectedEvent = computed(
  () => filteredEvents.value[selectedEventIndex.value] ?? null
)

const formatTimestamp = (ts: number) => {
  if (!ts) return ""
  const dt = new Date(ts)
  const pad = (value: number, len = 2) => String(value).padStart(len, "0")
  return `${dt.getFullYear()}-${pad(dt.getMonth() + 1)}-${pad(dt.getDate())} ${pad(
    dt.getHours()
  )}:${pad(dt.getMinutes())}:${pad(dt.getSeconds())}.${pad(dt.getMilliseconds(), 3)}`
}

const formatEventLine = (ev: TopicBusEvent) => {
  const ts = formatTimestamp(ev.ts)
  if (!ts) return `${ev.topic} | ${ev.name}`
  return `${ev.topic} | ${ev.name} | ${ts}`
}

const ensureReady = () => {
  if (!sessionStore.connected) {
    throw new Error("Connect to a session before sending TopicBus requests.")
  }
  if (!selfNodeId.value) {
    throw new Error("Login to a node before using TopicBus operations.")
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
}

const loadPreferences = async () => {
  try {
    await topicbus.loadPrefs()
    syncMaxEventsInput()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to load TopicBus preferences.")
  }
}

const syncMaxEventsInput = () => {
  maxEventsInput.value = String(topicbus.state.maxEvents || 500)
}

const applyMaxEvents = async () => {
  if (busy.value) return
  busy.value = true
  try {
    const raw = maxEventsInput.value.trim()
    const parsed = Number.parseInt(raw || String(topicbus.state.maxEvents || 500), 10)
    if (Number.isNaN(parsed) || parsed <= 0) {
      throw new Error("Max events must be a positive number.")
    }
    await topicbus.setMaxEvents(parsed)
    syncMaxEventsInput()
    toast.success("Max events updated.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to update max events.")
  } finally {
    busy.value = false
  }
}

const subscribeFromInput = async () => {
  if (busy.value) return
  busy.value = true
  try {
    const topics = topicbus.parseTopics(subForm.text)
    if (!topics.length) {
      throw new Error("Topic is required.")
    }
    await topicbus.updateTopics(topics, "add")
    if (!sessionStore.connected || !selfNodeId.value) {
      toast.info("Saved subscription list only; login to send subscribe.")
      return
    }
    await topicbus.subscribe(topics)
    toast.success("Subscribed.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to subscribe.")
  } finally {
    busy.value = false
  }
}

const unsubscribeFromInput = async () => {
  if (busy.value) return
  busy.value = true
  try {
    const topics = topicbus.parseTopics(subForm.text)
    if (!topics.length) {
      throw new Error("Topic is required.")
    }
    await topicbus.updateTopics(topics, "remove")
    if (!sessionStore.connected || !selfNodeId.value) {
      toast.info("Updated list only; login to send unsubscribe.")
      return
    }
    await topicbus.unsubscribe(topics)
    toast.success("Unsubscribed.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to unsubscribe.")
  } finally {
    busy.value = false
  }
}

const unsubscribeSelected = async () => {
  if (busy.value) return
  busy.value = true
  try {
    if (!topicbus.state.selectedTopic) {
      throw new Error("Select a topic to unsubscribe.")
    }
    const topic = topicbus.state.selectedTopic
    await topicbus.updateTopics([topic], "remove")
    topicbus.setSelectedTopic("")
    selectedEventIndex.value = -1
    if (!sessionStore.connected || !selfNodeId.value) {
      toast.info("Updated list only; login to send unsubscribe.")
      return
    }
    await topicbus.unsubscribe([topic])
    toast.success("Unsubscribed.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to unsubscribe.")
  } finally {
    busy.value = false
  }
}

const resubscribeAll = async () => {
  if (busy.value) return
  busy.value = true
  try {
    ensureReady()
    if (!topicbus.state.topics.length) {
      toast.info("No topics to resubscribe.")
      return
    }
    await topicbus.resubscribe()
    toast.success("Resubscribed.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to resubscribe.")
  } finally {
    busy.value = false
  }
}

const clearEvents = () => {
  topicbus.clearEvents()
  selectedEventIndex.value = -1
}

const fillSelectedTopic = () => {
  if (!topicbus.state.selectedTopic) {
    toast.warn("Select a topic to populate the publish form.")
    return
  }
  publishForm.topic = topicbus.state.selectedTopic
}

const clearPublishInputs = () => {
  publishForm.topic = ""
  publishForm.name = ""
  publishForm.payload = ""
}

const publishEvent = async () => {
  if (busy.value) return
  busy.value = true
  try {
    ensureReady()
    await topicbus.publish(publishForm.topic, publishForm.name, publishForm.payload)
    toast.success("Event published.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to publish event.")
  } finally {
    busy.value = false
  }
}

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  () => {
    topicbus.setIdentity(selfNodeId.value, hubId.value)
  }
)

watch(
  () => topicbus.state.selectedTopic,
  () => {
    selectedEventIndex.value = -1
  }
)

watch(filteredEvents, (next) => {
  if (selectedEventIndex.value >= next.length) {
    selectedEventIndex.value = -1
  }
})

watch(
  () => profileStore.state.current,
  async () => {
    await loadHomeDefaults()
    await loadPreferences()
    if (sessionStore.connected && selfNodeId.value && topicbus.state.topics.length) {
      void resubscribeAll()
    }
  }
)

let lastLoggedIn = false
watch(
  () => sessionStore.auth.loggedIn,
  (loggedIn) => {
    if (loggedIn && !lastLoggedIn && topicbus.state.topics.length) {
      void resubscribeAll()
    }
    lastLoggedIn = loggedIn
  }
)

onMounted(async () => {
  await loadHomeDefaults()
  await loadPreferences()
  if (sessionStore.connected && selfNodeId.value && topicbus.state.topics.length) {
    void resubscribeAll()
  }
})
</script>

<template>
  <section class="grid gap-6">
    <div class="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
      <div class="space-y-6">
        <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
                TopicBus Control
              </p>
              <h3 class="text-lg font-semibold">Target & Subscriptions</h3>
              <p class="text-sm text-muted-foreground">
                Subscribe to topics and stream published events.
              </p>
            </div>
            <Badge :class="connectedTone">{{ connectedLabel }}</Badge>
          </div>

          <div class="mt-4 grid gap-4 lg:grid-cols-[1fr_auto]">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Target Node ID
              </label>
              <input
                v-model="topicbus.state.targetId"
                :placeholder="hubId ? String(hubId) : 'Hub NodeID'"
                :class="inputClass"
              />
            </div>
            <div class="flex flex-col justify-end gap-2">
              <Button :disabled="busy" @click="resubscribeAll">Resubscribe</Button>
              <Button variant="outline" :disabled="busy" @click="unsubscribeSelected">
                Unsubscribe Selected
              </Button>
            </div>
          </div>

          <div class="mt-4">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Subscribe / Unsubscribe
            </label>
            <textarea
              v-model="subForm.text"
              :class="textAreaClass"
              rows="3"
              placeholder="topic.a, topic.b (comma, newline, or semicolon separated)"
            />
            <div class="mt-3 flex flex-wrap gap-2">
              <Button size="sm" :disabled="busy" @click="subscribeFromInput">Subscribe</Button>
              <Button size="sm" variant="outline" :disabled="busy" @click="unsubscribeFromInput">
                Unsubscribe
              </Button>
            </div>
          </div>

          <div class="mt-4 grid gap-4 lg:grid-cols-[1fr_auto]">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Max Events
              </label>
              <input v-model="maxEventsInput" :class="inputClass" placeholder="500" />
            </div>
            <div class="flex flex-col justify-end gap-2">
              <Button variant="outline" :disabled="busy" @click="applyMaxEvents">
                Apply Limit
              </Button>
              <Button variant="ghost" :disabled="busy" @click="clearEvents">Clear Events</Button>
            </div>
          </div>

          <div class="mt-4 flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
            <span>NodeID: {{ selfNodeId || "-" }}</span>
            <span>HubID: {{ hubId || "-" }}</span>
            <span>Topics: {{ topicbus.state.topics.length }}</span>
          </div>
        </div>

        <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
                Publish
              </p>
              <h3 class="text-lg font-semibold">Send Topic Events</h3>
              <p class="text-sm text-muted-foreground">
                Publish JSON or plain text payloads to any topic.
              </p>
            </div>
            <Badge variant="secondary">Publish</Badge>
          </div>

          <div class="mt-4 grid gap-4">
            <div class="grid gap-3 lg:grid-cols-[1fr_auto]">
              <div>
                <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  Topic
                </label>
                <input
                  v-model="publishForm.topic"
                  :class="inputClass"
                  placeholder="topic.status"
                />
              </div>
              <div class="flex flex-col justify-end gap-2">
                <Button size="sm" variant="outline" :disabled="busy" @click="fillSelectedTopic">
                  Use Selected
                </Button>
              </div>
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Name
              </label>
              <input v-model="publishForm.name" :class="inputClass" placeholder="event name" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Payload
              </label>
              <textarea
                v-model="publishForm.payload"
                :class="textAreaClass"
                rows="4"
                placeholder="JSON or plain text"
              />
            </div>
            <div class="flex flex-wrap gap-2">
              <Button :disabled="busy" @click="publishEvent">Publish</Button>
              <Button variant="outline" :disabled="busy" @click="clearPublishInputs">
                Clear
              </Button>
            </div>
          </div>
        </div>
      </div>

      <div class="space-y-6">
        <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
            Subscription List
          </p>
          <h3 class="mt-2 text-lg font-semibold">Active Topics</h3>
          <div class="mt-4 space-y-2 text-sm text-muted-foreground">
            <button
              class="flex w-full items-center justify-between rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-left text-sm transition hover:border-primary"
              :class="topicbus.state.selectedTopic ? '' : 'border-primary text-foreground'"
              @click="topicbus.setSelectedTopic('')"
            >
              <span>All</span>
              <span>{{ topicbus.state.topics.length }}</span>
            </button>
            <button
              v-for="topic in topicbus.state.topics"
              :key="topic"
              class="flex w-full items-center justify-between rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-left text-sm transition hover:border-primary"
              :class="topicbus.state.selectedTopic === topic ? 'border-primary text-foreground' : ''"
              @click="topicbus.setSelectedTopic(topic)"
            >
              <span>{{ topic }}</span>
            </button>
            <p v-if="topicbus.state.topics.length === 0">No topics yet.</p>
          </div>
        </div>

        <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
            Snapshot
          </p>
          <h3 class="mt-2 text-lg font-semibold">TopicBus Status</h3>
          <div class="mt-4 space-y-3 text-sm text-muted-foreground">
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">Selected Topic</p>
              <p>{{ topicbus.state.selectedTopic || "All" }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">Last Frame</p>
              <p>{{ topicbus.state.lastFrameAt || "-" }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">Events Cached</p>
              <p>{{ topicbus.state.events.length }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
      <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
              Event Stream
            </p>
            <h3 class="text-lg font-semibold">Publish Events</h3>
          </div>
          <Badge variant="outline">
            {{ topicbus.state.selectedTopic || "All" }} · {{ filteredEvents.length }}
          </Badge>
        </div>

        <div class="mt-4 max-h-[420px] space-y-2 overflow-y-auto pr-2 text-sm">
          <button
            v-for="(event, index) in filteredEvents"
            :key="`${event.topic}-${event.name}-${event.ts}-${index}`"
            class="w-full rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-left text-xs transition hover:border-primary"
            :class="selectedEventIndex === index ? 'border-primary text-foreground' : ''"
            @click="selectedEventIndex = index"
          >
            <span class="block truncate">{{ formatEventLine(event) }}</span>
          </button>
          <p v-if="filteredEvents.length === 0" class="text-sm text-muted-foreground">
            No events yet.
          </p>
        </div>
      </div>

      <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
          Event Detail
        </p>
        <h3 class="mt-2 text-lg font-semibold">Selected Payload</h3>
        <div class="mt-4 min-h-[320px] rounded-xl border border-border/60 bg-background/70 p-4">
          <pre class="whitespace-pre-wrap text-xs text-muted-foreground">
{{ selectedEvent?.dataRaw || "Select an event to inspect the payload." }}
          </pre>
        </div>
      </div>
    </div>

  </section>
</template>
