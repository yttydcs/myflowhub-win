<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"
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
const { t } = useI18n()

const busy = ref(false)
const remoteBusy = ref(false)
const activeTab = ref<TopicBusTab>("overview")

const subForm = reactive({
  text: ""
})

const fallbackIdentity = reactive({
  nodeId: 0,
  hubId: 0
})

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
  { label: t("Connected"), value: connectedLabel.value },
  { label: t("NodeID"), value: selfNodeId.value ? String(selfNodeId.value) : "-" },
  { label: t("HubID"), value: hubId.value ? String(hubId.value) : "-" },
  { label: t("Local Topics"), value: String(topicbus.state.topics.length) },
  { label: t("Remote Active"), value: String(topicbus.state.remoteTopics.length) },
  { label: t("Cached Events"), value: String(topicbus.state.events.length) },
  { label: t("Last Frame"), value: topicbus.state.lastFrameAt || "-" },
  { label: t("Remote Sync"), value: topicbus.state.remoteSyncedAt || "-" }
])

const tabButtonClass = (tab: TopicBusTab) => [
  "rounded-full px-4 py-2 text-sm font-semibold transition",
  activeTab.value === tab
    ? "bg-primary text-primary-foreground shadow-sm"
    : "text-muted-foreground hover:bg-muted/70"
]

const channelStatusText = (item: TopicBusChannelItem) => {
  if (item.localSaved && item.remoteSubscribed) return t("Saved + active")
  if (item.localSaved) return t("Saved locally")
  if (item.remoteSubscribed) return t("Remote only")
  return t("Known topic")
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
}

const loadPreferences = async () => {
  try {
    await topicbus.loadPrefs()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to load TopicBus preferences."))
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
      toast.success(t("Remote subscriptions synced."))
    }
  } catch (err) {
    console.warn(err)
    if (!silent) {
      toast.errorOf(err, t("Failed to sync remote subscriptions."))
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

const subscribeFromInput = async () => {
  if (busy.value) return
  busy.value = true
  try {
    const topics = topicbus.parseTopics(subForm.text)
    if (!topics.length) {
      throw new Error(t("Topic is required."))
    }
    await topicbus.updateTopics(topics, "add")
    if (!sessionStore.connected || !selfNodeId.value) {
      toast.info(t("Saved topic list only; login to send subscribe."))
      return
    }
    await topicbus.subscribe(topics)
    toast.success(t("Subscribed."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to subscribe."))
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
      throw new Error(t("Topic is required."))
    }
    await topicbus.updateTopics(topics, "remove")
    if (!sessionStore.connected || !selfNodeId.value) {
      toast.info(t("Updated local list only; login to send unsubscribe."))
      return
    }
    await topicbus.unsubscribe(topics)
    toast.success(t("Unsubscribed."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to unsubscribe."))
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
      toast.success(t("Channel unsubscribed."))
    } else {
      if (!item.localSaved) {
        await topicbus.updateTopics([item.topic], "add")
      }
      await topicbus.subscribe([item.topic])
      toast.success(t("Channel subscribed."))
    }
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to update channel subscription."))
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
      toast.info(t("No saved topics to resubscribe."))
      return
    }
    await topicbus.resubscribe()
    toast.success(t("Resubscribed."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to resubscribe."))
  } finally {
    busy.value = false
  }
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
  toast.warn(t("TopicBus window was blocked by browser popup policy."))
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
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">{{ t("Workspace") }}</p>
          <h2 class="mt-1 text-lg font-semibold">{{ t("TopicBus Console") }}</h2>
          <p class="mt-2 text-sm text-muted-foreground">
            {{ t("Keep the main page focused on settings, then open a clean channel window for live receive and send.") }}
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
            {{ t(tab.label) }}
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
                {{ t("TopicBus Control") }}
              </p>
              <h3 class="text-lg font-semibold">{{ t("Identity & Saved Topics") }}</h3>
              <p class="text-sm text-muted-foreground">
                {{ t("Manage the target node, local topic list, and remote subscription sync.") }}
              </p>
            </div>
            <Badge :class="connectedTone">{{ connectedLabel }}</Badge>
          </div>

          <div class="mt-4 grid gap-4 lg:grid-cols-[1fr_auto]">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Target Node ID") }}
              </label>
              <input
                v-model="topicbus.state.targetId"
                :placeholder="hubId ? String(hubId) : t('Hub NodeID')"
                :class="inputClass"
              />
            </div>
            <div class="flex flex-col justify-end gap-2">
              <Button variant="outline" :disabled="busy || remoteBusy" @click="syncRemoteTopics()">
                {{ remoteBusy ? t("Syncing...") : t("Sync Remote") }}
              </Button>
              <Button :disabled="busy" @click="resubscribeAll">{{ t("Resubscribe") }}</Button>
            </div>
          </div>

          <div class="mt-4">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Saved Topic List") }}
            </label>
            <textarea
              v-model="subForm.text"
              :class="textAreaClass"
              rows="4"
              :placeholder="t('topic.a, topic.b (comma, newline, or semicolon separated)')"
            />
            <div class="mt-3 flex flex-wrap gap-2">
              <Button size="sm" :disabled="busy" @click="subscribeFromInput">{{ t("Save + Subscribe") }}</Button>
              <Button size="sm" variant="outline" :disabled="busy" @click="unsubscribeFromInput">
                {{ t("Remove + Unsubscribe") }}
              </Button>
            </div>
          </div>
        </section>
      </div>

      <div class="space-y-6">
        <section class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">{{ t("Snapshot") }}</p>
          <h3 class="mt-2 text-lg font-semibold">{{ t("Current Status") }}</h3>
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
              <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">{{ t("Quick Window") }}</p>
              <h3 class="text-lg font-semibold">{{ t("Jump Into Live Traffic") }}</h3>
            </div>
            <Badge variant="outline">{{ t("{count} channels", { count: channelItems.length }) }}</Badge>
          </div>

          <div class="mt-4 space-y-3">
            <div class="rounded-xl border border-border/60 bg-background/70 px-4 py-3 text-sm">
              <p class="font-semibold text-foreground">{{ t("All Channels") }}</p>
              <p class="mt-1 text-muted-foreground">
                {{ t("Aggregate every known channel in one clean receive/send window.") }}
              </p>
              <div class="mt-3">
                <Button size="sm" variant="outline" @click="openTopicWindow()">{{ t("Open All Window") }}</Button>
              </div>
            </div>

            <div v-if="topicbus.state.selectedTopic" class="rounded-xl border border-border/60 bg-background/70 px-4 py-3 text-sm">
              <p class="font-semibold text-foreground">{{ topicbus.state.selectedTopic }}</p>
              <p class="mt-1 text-muted-foreground">{{ t("Open the currently selected channel directly.") }}</p>
              <div class="mt-3">
                <Button size="sm" @click="openTopicWindow({ topic: topicbus.state.selectedTopic, localSaved: true, remoteSubscribed: topicbus.state.remoteTopics.includes(topicbus.state.selectedTopic) })">
                  {{ t("Open Selected Window") }}
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
            <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">{{ t("Channels") }}</p>
            <h3 class="mt-2 text-lg font-semibold">{{ t("Known Topics") }}</h3>
            <p class="mt-2 text-sm text-muted-foreground">
              {{ t("Each row opens a dedicated window that starts listening from the moment it opens.") }}
            </p>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <Badge variant="outline">{{ t("{count} saved", { count: topicbus.state.topics.length }) }}</Badge>
            <Badge variant="secondary">{{ t("{count} remote active", { count: topicbus.state.remoteTopics.length }) }}</Badge>
            <Button size="sm" variant="outline" :disabled="remoteBusy" @click="syncRemoteTopics()">
              {{ remoteBusy ? t("Syncing...") : t("Refresh Remote") }}
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
                  <p class="truncate text-base font-semibold">{{ t("All") }}</p>
                  <Badge variant="secondary">{{ t("Aggregate") }}</Badge>
                </div>
                <p class="mt-1 text-sm text-muted-foreground">
                  {{ t("Watch all known channels in a single clean window. Cached events: {count}.", { count: topicbus.state.events.length }) }}
                </p>
              </button>

              <div class="flex flex-wrap items-center gap-2">
                <Button size="sm" variant="outline" @click="openTopicWindow()">{{ t("Open Window") }}</Button>
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
                  <Badge v-if="item.localSaved" variant="outline">{{ t("Saved") }}</Badge>
                  <Badge v-if="item.remoteSubscribed" variant="secondary">{{ t("Active") }}</Badge>
                </div>
                <p class="mt-1 text-sm text-muted-foreground">
                  {{ t("{status} · Cached {count} · Last seen {lastSeen}", { status: channelStatusText(item), count: topicEventCount(item.topic), lastSeen: topicLastSeen(item.topic) }) }}
                </p>
              </button>

              <div class="flex flex-wrap items-center gap-2">
                <Button size="sm" variant="outline" @click="openTopicWindow(item)">{{ t("Open Window") }}</Button>
                <Button size="sm" variant="ghost" :disabled="busy" @click="syncChannelSubscription(item)">
                  {{ item.remoteSubscribed ? t("Unsubscribe") : t("Subscribe") }}
                </Button>
              </div>
            </div>
          </article>

          <p v-if="channelItems.length === 0" class="rounded-xl border border-dashed border-border/60 bg-background/60 px-4 py-6 text-sm text-muted-foreground">
            {{ t("No known channels yet. Save topics in Overview or sync remote subscriptions first.") }}
          </p>
        </div>
      </section>
    </section>
  </section>
</template>
