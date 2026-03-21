<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useProfileStore } from "@/stores/profile"
import { useSessionStore } from "@/stores/session"
import { formatTopicBusTimestamp, useTopicBusStore, type TopicBusChannelItem } from "@/stores/topicbus"
import { useToastStore } from "@/stores/toast"
import { HomeState as LoadHomeState } from "../../wailsjs/go/main/App"

type TopicBusTab = "overview" | "channels"

const tabs: { id: TopicBusTab; label: string }[] = [
  { id: "overview", label: "Overview" },
  { id: "channels", label: "Channels" }
]

const profileStore = useProfileStore()
const sessionStore = useSessionStore()
const topicbus = useTopicBusStore()
const toast = useToastStore()

const busy = ref(false)
const remoteBusy = ref(false)
const activeTab = ref<TopicBusTab>("overview")

const subForm = reactive({
  text: ""
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

const channelItems = computed(() => topicbus.channelItems())

const eventStatsByTopic = computed(() => {
  const stats = new Map<string, { count: number; lastTs: number }>()
  for (const event of topicbus.state.events) {
    const current = stats.get(event.topic) ?? { count: 0, lastTs: 0 }
    current.count += 1
    current.lastTs = Math.max(current.lastTs, Number(event.ts || 0))
    stats.set(event.topic, current)
  }
  return stats
})

const summaryItems = computed(() => [
  { label: "Connected", value: connectedLabel.value },
  { label: "NodeID", value: selfNodeId.value ? String(selfNodeId.value) : "-" },
  { label: "HubID", value: hubId.value ? String(hubId.value) : "-" },
  { label: "Local Topics", value: String(topicbus.state.topics.length) },
  { label: "Remote Active", value: String(topicbus.state.remoteTopics.length) },
  { label: "Cached Events", value: String(topicbus.state.events.length) },
  { label: "Last Frame", value: topicbus.state.lastFrameAt || "-" },
  { label: "Remote Sync", value: topicbus.state.remoteSyncedAt || "-" }
])

const tabButtonClass = (tab: TopicBusTab) => [
  "rounded-full px-4 py-2 text-sm font-semibold transition",
  activeTab.value === tab
    ? "bg-primary text-primary-foreground shadow-sm"
    : "text-muted-foreground hover:bg-muted/70"
]

const channelStatusText = (item: TopicBusChannelItem) => {
  if (item.localSaved && item.remoteSubscribed) return "Saved + active"
  if (item.localSaved) return "Saved locally"
  if (item.remoteSubscribed) return "Remote only"
  return "Known topic"
}

const channelRowClass = (topic: string) =>
  topicbus.state.selectedTopic === topic
    ? "border-primary/50 bg-primary/10"
    : "border-border/60 bg-background/70 hover:border-primary/40"

const topicEventCount = (topic: string) => eventStatsByTopic.value.get(topic)?.count ?? 0

const topicLastSeen = (topic: string) => {
  const ts = eventStatsByTopic.value.get(topic)?.lastTs ?? 0
  return ts ? formatTopicBusTimestamp(ts) : "-"
}

const selectChannel = (topic: string) => {
  topicbus.setSelectedTopic(topic)
}

const clearRemoteView = () => {
  topicbus.state.remoteTopics = []
  topicbus.state.remoteSyncedAt = ""
  if (topicbus.state.selectedTopic && !topicbus.state.topics.includes(topicbus.state.selectedTopic)) {
    topicbus.setSelectedTopic("")
  }
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

const syncMaxEventsInput = () => {
  maxEventsInput.value = String(topicbus.state.maxEvents || 500)
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

const syncRemoteTopics = async (silent = false) => {
  if (!sessionStore.connected || !selfNodeId.value) {
    clearRemoteView()
    return
  }
  if (remoteBusy.value) return
  remoteBusy.value = true
  try {
    await topicbus.refreshRemoteTopics()
    if (!silent) {
      toast.success("Remote subscriptions synced.")
    }
  } catch (err) {
    console.warn(err)
    if (!silent) {
      toast.errorOf(err, "Failed to sync remote subscriptions.")
    }
  } finally {
    remoteBusy.value = false
  }
}

const restoreAndSync = async () => {
  if (sessionStore.connected && selfNodeId.value && topicbus.state.topics.length) {
    try {
      await topicbus.resubscribe()
    } catch (err) {
      console.warn(err)
    }
  }
  await syncRemoteTopics(true)
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
      toast.info("Saved topic list only; login to send subscribe.")
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
      toast.info("Updated local list only; login to send unsubscribe.")
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

const syncChannelSubscription = async (item: TopicBusChannelItem) => {
  if (busy.value) return
  busy.value = true
  try {
    ensureReady()
    if (!item.localSaved && !item.remoteSubscribed) {
      await topicbus.updateTopics([item.topic], "add")
    }
    if (item.remoteSubscribed) {
      await topicbus.unsubscribe([item.topic])
      toast.success("Channel unsubscribed.")
    } else {
      if (!item.localSaved) {
        await topicbus.updateTopics([item.topic], "add")
      }
      await topicbus.subscribe([item.topic])
      toast.success("Channel subscribed.")
    }
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to update channel subscription.")
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
      toast.info("No saved topics to resubscribe.")
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
  toast.success("Cached events cleared.")
}

const openTopicWindow = (item?: TopicBusChannelItem) => {
  const base = window.location.href.split("#")[0]
  const query = new URLSearchParams()
  if (item?.topic) {
    query.set("topic", item.topic)
  } else {
    query.set("scope", "all")
  }
  const targetId = topicbus.state.targetId.trim() || (hubId.value ? String(hubId.value) : "")
  if (targetId) {
    query.set("targetId", targetId)
  }
  const name = item?.topic
    ? `topicbus_${encodeURIComponent(item.topic)}_${Date.now()}`
    : `topicbus_all_${Date.now()}`
  const url = `${base}#/topicbus-window?${query.toString()}`
  const win = window.open(url, name, "width=1080,height=760")
  if (win) {
    win.focus()
    return
  }
  toast.warn("TopicBus window was blocked by browser popup policy.")
}

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  () => {
    topicbus.setIdentity(selfNodeId.value, hubId.value)
  }
)

watch(
  () => profileStore.state.current,
  async () => {
    await loadHomeDefaults()
    await loadPreferences()
    await restoreAndSync()
  }
)

watch(
  () => sessionStore.connected,
  (connected) => {
    if (!connected) {
      clearRemoteView()
      return
    }
    if (sessionStore.auth.loggedIn) {
      void syncRemoteTopics(true)
    }
  }
)

let lastLoggedIn = false
watch(
  () => sessionStore.auth.loggedIn,
  (loggedIn) => {
    if (loggedIn && !lastLoggedIn) {
      void restoreAndSync()
    } else if (!loggedIn) {
      clearRemoteView()
    }
    lastLoggedIn = loggedIn
  }
)

onMounted(async () => {
  await loadHomeDefaults()
  await loadPreferences()
  await restoreAndSync()
})
</script>

<template>
  <section class="space-y-6">
    <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">Workspace</p>
          <h2 class="mt-1 text-lg font-semibold">TopicBus Console</h2>
          <p class="mt-2 text-sm text-muted-foreground">
            Keep the main page focused on settings, then open a clean channel window for live receive and send.
          </p>
        </div>

        <div class="inline-flex rounded-full border border-border/70 bg-background/80 p-1">
          <button
            v-for="tab in tabs"
            :key="tab.id"
            type="button"
            :class="tabButtonClass(tab.id)"
            :aria-pressed="activeTab === tab.id"
            @click="activeTab = tab.id"
          >
            {{ tab.label }}
          </button>
        </div>
      </div>
    </section>

    <section v-if="activeTab === 'overview'" class="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
      <div class="space-y-6">
        <section class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
                TopicBus Control
              </p>
              <h3 class="text-lg font-semibold">Identity & Saved Topics</h3>
              <p class="text-sm text-muted-foreground">
                Manage the target node, local topic list, and remote subscription sync.
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
              <Button variant="outline" :disabled="busy || remoteBusy" @click="syncRemoteTopics()">
                {{ remoteBusy ? "Syncing..." : "Sync Remote" }}
              </Button>
              <Button :disabled="busy" @click="resubscribeAll">Resubscribe</Button>
            </div>
          </div>

          <div class="mt-4">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Saved Topic List
            </label>
            <textarea
              v-model="subForm.text"
              :class="textAreaClass"
              rows="4"
              placeholder="topic.a, topic.b (comma, newline, or semicolon separated)"
            />
            <div class="mt-3 flex flex-wrap gap-2">
              <Button size="sm" :disabled="busy" @click="subscribeFromInput">Save + Subscribe</Button>
              <Button size="sm" variant="outline" :disabled="busy" @click="unsubscribeFromInput">
                Remove + Unsubscribe
              </Button>
            </div>
          </div>
        </section>

        <section class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">Cache & Window</p>
              <h3 class="text-lg font-semibold">Event Cache Settings</h3>
              <p class="text-sm text-muted-foreground">
                Channel windows only show messages received after the window opens. Your own publish actions do not echo back.
              </p>
            </div>
            <Badge variant="secondary">Window-first</Badge>
          </div>

          <div class="mt-4 grid gap-4 lg:grid-cols-[1fr_auto]">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Max Events
              </label>
              <input v-model="maxEventsInput" :class="inputClass" placeholder="500" />
            </div>
            <div class="flex flex-col justify-end gap-2">
              <Button variant="outline" :disabled="busy" @click="applyMaxEvents">Apply Limit</Button>
              <Button variant="ghost" :disabled="busy" @click="clearEvents">Clear Cached</Button>
            </div>
          </div>

          <div class="mt-4 rounded-xl border border-border/60 bg-background/70 px-4 py-3 text-sm text-muted-foreground">
            <p class="font-medium text-foreground">Recommended workflow</p>
            <p class="mt-1">
              Keep this page for setup, then jump to <span class="font-semibold text-foreground">Channels</span> and open
              a dedicated window for `All` or a specific topic.
            </p>
          </div>
        </section>
      </div>

      <div class="space-y-6">
        <section class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">Snapshot</p>
          <h3 class="mt-2 text-lg font-semibold">Current Status</h3>
          <div class="mt-4 space-y-3 text-sm text-muted-foreground">
            <div
              v-for="item in summaryItems"
              :key="item.label"
              class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border/60 bg-background/70 px-3 py-2"
            >
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ item.label }}</p>
              <p class="font-medium text-foreground">{{ item.value }}</p>
            </div>
          </div>
        </section>

        <section class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">Quick Window</p>
              <h3 class="text-lg font-semibold">Jump Into Live Traffic</h3>
            </div>
            <Badge variant="outline">{{ channelItems.length }} channels</Badge>
          </div>

          <div class="mt-4 space-y-3">
            <div class="rounded-xl border border-border/60 bg-background/70 px-4 py-3 text-sm">
              <p class="font-semibold text-foreground">All Channels</p>
              <p class="mt-1 text-muted-foreground">
                Aggregate every known channel in one clean receive/send window.
              </p>
              <div class="mt-3">
                <Button size="sm" variant="outline" @click="openTopicWindow()">Open All Window</Button>
              </div>
            </div>

            <div v-if="topicbus.state.selectedTopic" class="rounded-xl border border-border/60 bg-background/70 px-4 py-3 text-sm">
              <p class="font-semibold text-foreground">{{ topicbus.state.selectedTopic }}</p>
              <p class="mt-1 text-muted-foreground">Open the currently selected channel directly.</p>
              <div class="mt-3">
                <Button size="sm" @click="openTopicWindow({ topic: topicbus.state.selectedTopic, localSaved: true, remoteSubscribed: topicbus.state.remoteTopics.includes(topicbus.state.selectedTopic) })">
                  Open Selected Window
                </Button>
              </div>
            </div>
          </div>
        </section>
      </div>
    </section>

    <section v-else class="space-y-6">
      <section class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">Channels</p>
            <h3 class="mt-2 text-lg font-semibold">Known Topics</h3>
            <p class="mt-2 text-sm text-muted-foreground">
              Each row opens a dedicated window that starts listening from the moment it opens.
            </p>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <Badge variant="outline">{{ topicbus.state.topics.length }} saved</Badge>
            <Badge variant="secondary">{{ topicbus.state.remoteTopics.length }} remote active</Badge>
            <Button size="sm" variant="outline" :disabled="remoteBusy" @click="syncRemoteTopics()">
              {{ remoteBusy ? "Syncing..." : "Refresh Remote" }}
            </Button>
          </div>
        </div>
      </section>

      <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
        <div class="space-y-3">
          <article
            class="rounded-2xl border px-4 py-4 shadow-sm transition"
            :class="topicbus.state.selectedTopic ? 'border-border/60 bg-background/70 hover:border-primary/40' : 'border-primary/50 bg-primary/10'"
          >
            <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <button type="button" class="min-w-0 text-left" @click="selectChannel('')">
                <div class="flex flex-wrap items-center gap-2">
                  <p class="truncate text-base font-semibold">All</p>
                  <Badge variant="secondary">Aggregate</Badge>
                </div>
                <p class="mt-1 text-sm text-muted-foreground">
                  Watch all known channels in a single clean window. Cached events: {{ topicbus.state.events.length }}.
                </p>
              </button>

              <div class="flex flex-wrap items-center gap-2">
                <Button size="sm" variant="outline" @click="openTopicWindow()">Open Window</Button>
              </div>
            </div>
          </article>

          <article
            v-for="item in channelItems"
            :key="item.topic"
            class="rounded-2xl border px-4 py-4 shadow-sm transition"
            :class="channelRowClass(item.topic)"
          >
            <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <button type="button" class="min-w-0 text-left" @click="selectChannel(item.topic)">
                <div class="flex flex-wrap items-center gap-2">
                  <p class="truncate text-base font-semibold">{{ item.topic }}</p>
                  <Badge v-if="item.localSaved" variant="outline">Saved</Badge>
                  <Badge v-if="item.remoteSubscribed" variant="secondary">Active</Badge>
                </div>
                <p class="mt-1 text-sm text-muted-foreground">
                  {{ channelStatusText(item) }} · Cached {{ topicEventCount(item.topic) }} · Last seen {{ topicLastSeen(item.topic) }}
                </p>
              </button>

              <div class="flex flex-wrap items-center gap-2">
                <Button size="sm" variant="outline" @click="openTopicWindow(item)">Open Window</Button>
                <Button size="sm" variant="ghost" :disabled="busy" @click="syncChannelSubscription(item)">
                  {{ item.remoteSubscribed ? "Unsubscribe" : "Subscribe" }}
                </Button>
              </div>
            </div>
          </article>

          <p v-if="channelItems.length === 0" class="rounded-xl border border-dashed border-border/60 bg-background/60 px-4 py-6 text-sm text-muted-foreground">
            No known channels yet. Save topics in <span class="font-semibold text-foreground">Overview</span> or sync remote subscriptions first.
          </p>
        </div>
      </section>
    </section>
  </section>
</template>
