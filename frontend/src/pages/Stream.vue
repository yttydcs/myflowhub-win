<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue"
import { Cable, Pause, Play, Plus, RefreshCw, ScanSearch, Send, Target } from "lucide-vue-next"
import CardHeader from "@/components/CardHeader.vue"
import PageHero from "@/components/PageHero.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import { t } from "@/i18n"
import { streamKinds, type StreamConsumer, type StreamConsumerDraft, type StreamSource, type StreamSourceDraft, type StreamTab, useStreamStore } from "@/stores/stream"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"

const stream = useStreamStore()
const sessionStore = useSessionStore()
const toast = useToastStore()

const sourceNameInputId = "stream-source-name"
const consumerNameInputId = "stream-consumer-name"
const sourceStudioInputId = "stream-source-studio-input"

const inputClass =
  "h-10 w-full rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
const textAreaClass =
  "min-h-[132px] w-full rounded-2xl border border-input bg-background px-3 py-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

const tabs: { id: StreamTab; label: string }[] = [
  { id: "source", label: "Source" },
  { id: "consumer", label: "Consumer" },
  { id: "control", label: "Control" }
]

const defaultSourceDraft = (): StreamSourceDraft => ({
  sourceId: "",
  name: "",
  kind: "text",
  contentType: "text/plain",
  mode: "live",
  unitMode: "frame",
  tagsText: "",
  metadataText: ""
})

const defaultConsumerDraft = (): StreamConsumerDraft => ({
  consumerId: "",
  name: "",
  kind: "text",
  contentType: "text/plain",
  tagsText: "",
  metadataText: ""
})

const controlSourceQuery = reactive({ producer: "", kind: "", tag: "" })
const controlConsumerQuery = reactive({ consumer: "", kind: "", tag: "" })
const subscribeQuery = reactive({ producer: "", kind: "", tag: "", selectedSourceId: "" })
const sourceDraft = reactive<StreamSourceDraft>(defaultSourceDraft())
const consumerDraft = reactive<StreamConsumerDraft>(defaultConsumerDraft())
const sourceStudio = reactive({
  open: false,
  sourceId: "",
  text: "",
  sentItems: [] as Array<{ text: string; sent: number; at: string }>
})
const subscribeDialog = reactive({
  open: false,
  consumerId: ""
})

const sourceDialogOpen = ref(false)
const consumerDialogOpen = ref(false)
const selectedLocalSourceId = ref("")
const selectedLocalConsumerId = ref("")

const selfNodeId = computed(() => Number(sessionStore.auth.nodeId || 0))
const hubId = computed(() => Number(sessionStore.auth.hubId || 0))
const activeTab = computed({
  get: () => stream.state.activeTab,
  set: (value: StreamTab) => stream.setActiveTab(value)
})
const targetIdText = computed({
  get: () => stream.state.targetId,
  set: (value: string) => stream.setTargetId(value)
})

const localSources = computed(() => stream.state.localSources)
const localConsumers = computed(() => stream.state.localConsumers)
const catalogSources = computed(() => stream.state.sources)
const catalogConsumers = computed(() => stream.state.consumers)
const deliveries = computed(() => stream.state.deliveries)
const selectedLocalSource = computed(() => stream.sourceById(selectedLocalSourceId.value, "local"))
const selectedLocalConsumer = computed(() => stream.consumerById(selectedLocalConsumerId.value, "local"))
const selectedControlSource = computed(() => stream.sourceById(stream.state.selectedSourceId, "catalog"))
const selectedControlConsumer = computed(() => stream.consumerById(stream.state.selectedConsumerId, "catalog"))
const selectedDelivery = computed(() => deliveries.value.find((item) => item.deliveryId === stream.state.selectedDeliveryId) ?? null)
const selectedTextFrames = computed(() => (selectedDelivery.value ? stream.textFramesFor(selectedDelivery.value.deliveryId) : []))
const selectedStats = computed(() => (selectedDelivery.value ? stream.statsFor(selectedDelivery.value.deliveryId) : null))
const resolvedTargetId = computed(() => targetIdText.value || (hubId.value ? String(hubId.value) : ""))
const latestActivityAt = computed(() => stream.state.lastEventAt || stream.state.lastSyncAt)
const sourceStudioSource = computed(() => stream.sourceById(sourceStudio.sourceId, "local"))
const subscribeConsumer = computed(() => stream.consumerById(subscribeDialog.consumerId, "local"))
const subscribeSelectedSource = computed(() => stream.sourceById(subscribeQuery.selectedSourceId, "catalog"))
const canConnect = computed(() => {
  if (!selectedControlSource.value || !selectedControlConsumer.value) return false
  return selectedControlSource.value.kind === selectedControlConsumer.value.kind
})
const canSubscribe = computed(() => {
  if (!selectedControlSource.value || !selectedControlConsumer.value || !canConnect.value) return false
  return selectedControlConsumer.value.consumer === selfNodeId.value
})

const summaryCards = computed(() => [
  { label: t("Saved Sources"), value: String(localSources.value.length), hint: t("Persistent local list") },
  { label: t("Saved Consumers"), value: String(localConsumers.value.length), hint: t("Restored after login") },
  { label: t("Known Deliveries"), value: String(deliveries.value.length), hint: selectedDelivery.value ? t("Selected {id}", { id: selectedDelivery.value.deliveryId }) : t("No selection") },
  { label: t("Last Runtime Event"), value: latestActivityAt.value ? formatTimestamp(latestActivityAt.value) : t("No activity yet"), hint: stream.state.lastSyncAt ? t("Last sync {time}", { time: formatTimestamp(stream.state.lastSyncAt) }) : t("Waiting for first sync") }
])

const heroDescription = computed(() => {
  if (activeTab.value === "source") return t("Manage your saved local sources, keep the main list compact, and open a separate input studio only when you need to send content.")
  if (activeTab.value === "consumer") return t("Keep local consumers in a simple list, review current bindings, and subscribe through a dedicated dialog instead of inline forms.")
  return t("Browse remote catalogs, connect compatible endpoints, and inspect runtime deliveries from the control tab.")
})

const tabButtonClass = (tab: StreamTab) => [
  "rounded-full px-4 py-2 text-sm font-semibold transition",
  activeTab.value === tab ? "bg-primary text-primary-foreground shadow-sm" : "text-muted-foreground hover:bg-muted/70"
]

const formatTimestamp = (value: string) => {
  const dt = new Date(String(value ?? ""))
  return Number.isNaN(dt.getTime()) ? "" : dt.toLocaleString()
}

const kindToneClass = (kind: string) => {
  switch (kind) {
    case "music":
      return "bg-amber-500/15 text-amber-700"
    case "video":
      return "bg-rose-500/15 text-rose-700"
    case "text":
      return "bg-emerald-500/15 text-emerald-700"
    default:
      return "bg-slate-500/15 text-slate-700"
  }
}

const metadataPreview = (value: string) => String(value ?? "").trim() || t("No metadata")
const descriptorMetaLine = (item: Pick<StreamSource, "contentType" | "mode" | "unitMode"> | Pick<StreamConsumer, "contentType">) =>
  ["contentType" in item ? item.contentType : "", "mode" in item ? item.mode : "", "unitMode" in item ? item.unitMode : ""]
    .map((entry) => String(entry ?? "").trim())
    .filter(Boolean)
    .join(" · ")
const tagsOrFallback = (tags: string[]) => (tags.length ? tags : [t("No tags")])
const sourceBindings = (sourceId: string) => stream.deliveriesForSource(sourceId)
const consumerBindings = (consumerId: string) => stream.deliveriesForConsumer(consumerId)

const resetSourceDraft = () => Object.assign(sourceDraft, defaultSourceDraft())
const resetConsumerDraft = () => Object.assign(consumerDraft, defaultConsumerDraft())
const closeSourceDialog = () => {
  sourceDialogOpen.value = false
  resetSourceDraft()
}
const closeConsumerDialog = () => {
  consumerDialogOpen.value = false
  resetConsumerDraft()
}
const closeSourceStudio = () => {
  sourceStudio.open = false
  sourceStudio.sourceId = ""
  sourceStudio.text = ""
}
const closeSubscribeDialog = () => {
  subscribeDialog.open = false
  subscribeDialog.consumerId = ""
  subscribeQuery.producer = ""
  subscribeQuery.kind = ""
  subscribeQuery.tag = ""
  subscribeQuery.selectedSourceId = ""
}

const withToast = async (action: () => Promise<unknown>, ok: string, fail: string) => {
  try {
    await action()
    toast.success(t(ok))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t(fail))
  }
}

const refreshControlSourcesRaw = () => stream.listSources(controlSourceQuery.producer, controlSourceQuery.kind, controlSourceQuery.tag)
const refreshControlConsumersRaw = () => stream.listConsumers(controlConsumerQuery.consumer, controlConsumerQuery.kind, controlConsumerQuery.tag)
const refreshControlSources = () => withToast(refreshControlSourcesRaw, "Sources refreshed.", "Failed to query sources.")
const refreshControlConsumers = () => withToast(refreshControlConsumersRaw, "Consumers refreshed.", "Failed to query consumers.")
const refreshDeliveries = () => withToast(() => stream.loadDeliveries(), "Runtime deliveries refreshed.", "Failed to load deliveries.")
const refreshControlPlane = () => withToast(async () => {
  await Promise.all([refreshControlSourcesRaw(), refreshControlConsumersRaw(), stream.loadDeliveries()])
}, "Stream control plane refreshed.", "Failed to refresh Stream control plane.")
const refreshLocalLists = () => withToast(async () => {
  if (!selfNodeId.value) return
  await Promise.all([stream.listSources(String(selfNodeId.value), "", "", "local"), stream.listConsumers(String(selfNodeId.value), "", "", "local")])
}, "Local stream lists refreshed.", "Failed to refresh local stream lists.")
const refreshSubscribeSourcesRaw = () => stream.listSources(subscribeQuery.producer, subscribeQuery.kind, subscribeQuery.tag)
const refreshSubscribeSources = () => withToast(refreshSubscribeSourcesRaw, "Subscription sources refreshed.", "Failed to load subscription sources.")

const submitSource = () => withToast(async () => {
  const created = await stream.announceSource(sourceDraft)
  if (created) selectedLocalSourceId.value = created.sourceId
  closeSourceDialog()
}, "Local source created.", "Failed to create local source.")

const submitConsumer = () => withToast(async () => {
  const created = await stream.announceConsumer(consumerDraft)
  if (created) selectedLocalConsumerId.value = created.consumerId
  closeConsumerDialog()
}, "Local consumer created.", "Failed to create local consumer.")

const openSourceDialog = () => {
  resetSourceDraft()
  sourceDialogOpen.value = true
}

const openConsumerDialog = () => {
  resetConsumerDraft()
  consumerDialogOpen.value = true
}

const openSourceStudio = (sourceId: string) => {
  sourceStudio.sourceId = sourceId
  sourceStudio.text = ""
  sourceStudio.open = true
  selectedLocalSourceId.value = sourceId
}

const openSubscribeDialog = async (consumerId: string) => {
  const consumer = stream.consumerById(consumerId, "local")
  if (!consumer) return
  subscribeDialog.consumerId = consumerId
  subscribeQuery.producer = selfNodeId.value ? String(selfNodeId.value) : ""
  subscribeQuery.kind = consumer.kind
  subscribeQuery.tag = ""
  subscribeQuery.selectedSourceId = ""
  subscribeDialog.open = true
  if (subscribeQuery.producer) {
    try {
      await refreshSubscribeSourcesRaw()
    } catch (err) {
      console.warn(err)
    }
  }
}

const publishSourceText = () => withToast(async () => {
  const result = await stream.publishText(sourceStudio.sourceId, sourceStudio.text)
  sourceStudio.sentItems.unshift({ text: sourceStudio.text, sent: result.sent, at: new Date().toISOString() })
  sourceStudio.sentItems = sourceStudio.sentItems.slice(0, 8)
  sourceStudio.text = ""
}, "Text sent to source.", "Failed to send text to source.")

const subscribeFromDialog = () => withToast(async () => {
  if (!subscribeSelectedSource.value || !subscribeDialog.consumerId) throw new Error(t("Select a source before subscribing."))
  await stream.subscribe({
    producer: subscribeSelectedSource.value.producer,
    sourceId: subscribeSelectedSource.value.sourceId,
    consumerId: subscribeDialog.consumerId
  })
  closeSubscribeDialog()
}, "Subscribed.", "Failed to subscribe.")

const connectSelected = () =>
  selectedControlSource.value && selectedControlConsumer.value
    ? withToast(() =>
        stream.connect({
          producer: selectedControlSource.value!.producer,
          sourceId: selectedControlSource.value!.sourceId,
          consumer: selectedControlConsumer.value!.consumer,
          consumerId: selectedControlConsumer.value!.consumerId
        }),
      "Delivery connected.", "Failed to connect delivery.")
    : Promise.resolve()

const subscribeSelected = () =>
  selectedControlSource.value && selectedControlConsumer.value
    ? withToast(() =>
        stream.subscribe({
          producer: selectedControlSource.value!.producer,
          sourceId: selectedControlSource.value!.sourceId,
          consumerId: selectedControlConsumer.value!.consumerId
        }),
      "Subscribed.", "Failed to subscribe.")
    : Promise.resolve()

const disconnectSelected = () =>
  selectedDelivery.value
    ? withToast(() => stream.disconnect(selectedDelivery.value!.deliveryId), "Delivery disconnected.", "Failed to disconnect delivery.")
    : Promise.resolve()

const unsubscribeSelected = () =>
  selectedDelivery.value
    ? withToast(() => stream.unsubscribe(selectedDelivery.value!.deliveryId), "Delivery unsubscribed.", "Failed to unsubscribe delivery.")
    : Promise.resolve()

const signalSelected = (op: string) =>
  selectedDelivery.value
    ? withToast(() => stream.signal(selectedDelivery.value!.deliveryId, op), "Signal sent.", "Failed to send signal.")
    : Promise.resolve()

const removeSource = (sourceId: string) => withToast(async () => {
  await stream.withdrawSource(sourceId)
  if (selectedLocalSourceId.value === sourceId) selectedLocalSourceId.value = localSources.value[0]?.sourceId ?? ""
  if (sourceStudio.sourceId === sourceId) closeSourceStudio()
}, "Source removed.", "Failed to remove source.")

const removeConsumer = (consumerId: string) => withToast(async () => {
  await stream.withdrawConsumer(consumerId)
  if (selectedLocalConsumerId.value === consumerId) selectedLocalConsumerId.value = localConsumers.value[0]?.consumerId ?? ""
  if (subscribeDialog.consumerId === consumerId) closeSubscribeDialog()
}, "Consumer removed.", "Failed to remove consumer.")

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  () => {
    stream.setIdentity(selfNodeId.value, hubId.value)
    if (!targetIdText.value && hubId.value > 0) targetIdText.value = String(hubId.value)
    if (!controlSourceQuery.producer && selfNodeId.value > 0) controlSourceQuery.producer = String(selfNodeId.value)
    if (!controlConsumerQuery.consumer && selfNodeId.value > 0) controlConsumerQuery.consumer = String(selfNodeId.value)
    void (async () => {
      try {
        await Promise.all([stream.loadPrefs(), stream.loadDeliveries()])
        if (selfNodeId.value > 0) {
          if (!selectedLocalSourceId.value && stream.state.localSources.length) selectedLocalSourceId.value = stream.state.localSources[0].sourceId
          if (!selectedLocalConsumerId.value && stream.state.localConsumers.length) selectedLocalConsumerId.value = stream.state.localConsumers[0].consumerId
        }
        if (selfNodeId.value > 0 && hubId.value > 0) {
          await Promise.all([refreshControlSourcesRaw(), refreshControlConsumersRaw()])
        }
      } catch (err) {
        console.warn(err)
      }
    })()
  },
  { immediate: true }
)

watch(
  () => localSources.value.map((item) => item.sourceId).join("|"),
  () => {
    if (!selectedLocalSourceId.value || !stream.sourceById(selectedLocalSourceId.value, "local")) {
      selectedLocalSourceId.value = localSources.value[0]?.sourceId ?? ""
    }
  },
  { immediate: true }
)

watch(
  () => localConsumers.value.map((item) => item.consumerId).join("|"),
  () => {
    if (!selectedLocalConsumerId.value || !stream.consumerById(selectedLocalConsumerId.value, "local")) {
      selectedLocalConsumerId.value = localConsumers.value[0]?.consumerId ?? ""
    }
  },
  { immediate: true }
)
</script>

<template>
  <section class="space-y-6" data-stream-page>
    <PageHero :description="heroDescription">
      <template #actions>
        <Badge variant="secondary">{{ t("Self {id}", { id: selfNodeId || "-" }) }}</Badge>
        <Badge variant="secondary">{{ t("Hub {id}", { id: hubId || "-" }) }}</Badge>
        <Badge variant="secondary">{{ t("Target {id}", { id: resolvedTargetId || "-" }) }}</Badge>
        <div class="inline-flex rounded-full border border-border/70 bg-background/80 p-1">
          <button v-for="tab in tabs" :key="tab.id" type="button" :class="tabButtonClass(tab.id)" :aria-pressed="activeTab === tab.id" @click="activeTab = tab.id">
            {{ t(tab.label) }}
          </button>
        </div>
      </template>
    </PageHero>

    <section class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <article v-for="card in summaryCards" :key="card.label" class="rounded-2xl border border-border/60 bg-card/90 px-4 py-4 text-card-foreground shadow-sm">
        <p class="text-[11px] font-semibold uppercase tracking-[0.24em] text-muted-foreground">{{ card.label }}</p>
        <p class="mt-3 text-xl font-semibold">{{ card.value }}</p>
        <p class="mt-2 text-xs text-muted-foreground">{{ card.hint }}</p>
      </article>
    </section>

    <section v-if="activeTab === 'source'" class="grid gap-6 xl:grid-cols-[minmax(0,1.3fr)_360px]">
      <section class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
        <CardHeader :title="t('Local Sources')" :description="t('Keep local sources persistent and focused. Add or remove sources here, then open a separate studio only when you need to input content.')" title-class="text-lg">
          <template #actions>
            <div class="flex flex-wrap gap-2">
              <Button variant="outline" size="sm" @click="refreshLocalLists">
                <RefreshCw class="mr-2 h-4 w-4" />
                {{ t("Refresh") }}
              </Button>
              <Button size="sm" data-stream-open-source @click="openSourceDialog">
                <Plus class="mr-2 h-4 w-4" />
                {{ t("New Source") }}
              </Button>
            </div>
          </template>
        </CardHeader>

        <div class="mt-4 space-y-3">
          <button
            v-for="source in localSources"
            :key="source.sourceId"
            type="button"
            data-stream-local-source-row
            class="w-full rounded-2xl border p-4 text-left transition"
            :class="selectedLocalSourceId === source.sourceId ? 'border-primary/50 bg-primary/10 shadow-sm' : 'border-border/60 bg-background/70 hover:border-border'"
            @click="selectedLocalSourceId = source.sourceId"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-2">
                  <p class="truncate text-sm font-semibold">{{ source.name || source.sourceId }}</p>
                  <Badge :class="kindToneClass(source.kind)">{{ source.kind }}</Badge>
                  <Badge variant="secondary">{{ t("{count} bindings", { count: sourceBindings(source.sourceId).length }) }}</Badge>
                </div>
                <p class="mt-2 truncate text-xs text-muted-foreground">{{ source.sourceId }}</p>
                <p class="mt-1 text-xs text-muted-foreground">{{ descriptorMetaLine(source) || t("No descriptor details") }}</p>
                <div class="mt-3 flex flex-wrap gap-2">
                  <Badge v-for="tag in tagsOrFallback(source.tags)" :key="`${source.sourceId}:${tag}`" variant="secondary">{{ tag }}</Badge>
                </div>
                <div class="mt-3 flex flex-wrap gap-2">
                  <Badge v-for="binding in sourceBindings(source.sourceId).slice(0, 3)" :key="binding.deliveryId" variant="secondary">
                    {{ t("Consumer {id}", { id: binding.consumer }) }}
                  </Badge>
                  <span v-if="!sourceBindings(source.sourceId).length" class="text-xs text-muted-foreground">{{ t("No active bindings") }}</span>
                </div>
              </div>
              <div class="flex flex-col items-end gap-2">
                <Button variant="outline" size="sm" data-stream-open-studio @click.stop="openSourceStudio(source.sourceId)">{{ t("Input") }}</Button>
                <Button variant="ghost" size="sm" @click.stop="removeSource(source.sourceId)">{{ t("Remove") }}</Button>
              </div>
            </div>
          </button>

          <div v-if="!localSources.length" class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-10 text-center text-sm text-muted-foreground">
            {{ t("No local sources yet. Use New Source to create one.") }}
          </div>
        </div>
      </section>

      <aside class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
        <CardHeader :title="t('Source Details')" :description="t('Keep the list simple. Use this side panel to review metadata and current bindings before opening the input studio.')" title-class="text-lg" />

        <div v-if="selectedLocalSource" class="mt-4 space-y-4">
          <div class="flex flex-wrap items-center gap-2">
            <p class="text-lg font-semibold">{{ selectedLocalSource.name || selectedLocalSource.sourceId }}</p>
            <Badge :class="kindToneClass(selectedLocalSource.kind)">{{ selectedLocalSource.kind }}</Badge>
          </div>
          <p class="text-xs text-muted-foreground">{{ selectedLocalSource.sourceId }}</p>
          <p class="text-xs text-muted-foreground">{{ descriptorMetaLine(selectedLocalSource) }}</p>
          <div class="flex flex-wrap gap-2">
            <Badge v-for="tag in tagsOrFallback(selectedLocalSource.tags)" :key="`detail-${selectedLocalSource.sourceId}-${tag}`" variant="secondary">{{ tag }}</Badge>
          </div>
          <div class="rounded-2xl border border-border/60 bg-background/70 p-4">
            <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Metadata") }}</p>
            <pre class="mt-3 whitespace-pre-wrap break-words font-mono text-[12px] leading-6 text-muted-foreground">{{ metadataPreview(selectedLocalSource.metadataRaw) }}</pre>
          </div>
          <div class="rounded-2xl border border-border/60 bg-background/70 p-4">
            <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Current bindings") }}</p>
            <div class="mt-3 space-y-2">
              <article v-for="binding in sourceBindings(selectedLocalSource.sourceId)" :key="binding.deliveryId" class="rounded-xl border border-border/60 bg-card/80 px-3 py-3 text-sm">
                <p class="font-semibold">{{ t("Consumer {id}", { id: binding.consumer }) }}</p>
                <p class="mt-1 text-xs text-muted-foreground">{{ binding.consumerId }} · {{ binding.state }}</p>
              </article>
              <p v-if="!sourceBindings(selectedLocalSource.sourceId).length" class="text-sm text-muted-foreground">{{ t("Nothing is bound to this source yet.") }}</p>
            </div>
          </div>
          <div class="flex flex-wrap gap-2">
            <Button @click="openSourceStudio(selectedLocalSource.sourceId)">{{ t("Open Input Studio") }}</Button>
            <Button variant="outline" @click="removeSource(selectedLocalSource.sourceId)">{{ t("Remove Source") }}</Button>
          </div>
        </div>

        <div v-else class="mt-4 rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-10 text-center text-sm text-muted-foreground">
          {{ t("Select a local source to inspect its metadata and bindings.") }}
        </div>
      </aside>
    </section>

    <section v-else-if="activeTab === 'consumer'" class="grid gap-6 xl:grid-cols-[minmax(0,1.3fr)_360px]">
      <section class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
        <CardHeader :title="t('Local Consumers')" :description="t('Store local consumers as a compact list. Review current bindings here and open a separate dialog only when you want to subscribe to a source.')" title-class="text-lg">
          <template #actions>
            <div class="flex flex-wrap gap-2">
              <Button variant="outline" size="sm" @click="refreshLocalLists">
                <RefreshCw class="mr-2 h-4 w-4" />
                {{ t("Refresh") }}
              </Button>
              <Button size="sm" data-stream-open-consumer @click="openConsumerDialog">
                <Plus class="mr-2 h-4 w-4" />
                {{ t("New Consumer") }}
              </Button>
            </div>
          </template>
        </CardHeader>

        <div class="mt-4 space-y-3">
          <button
            v-for="consumer in localConsumers"
            :key="consumer.consumerId"
            type="button"
            data-stream-local-consumer-row
            class="w-full rounded-2xl border p-4 text-left transition"
            :class="selectedLocalConsumerId === consumer.consumerId ? 'border-primary/50 bg-primary/10 shadow-sm' : 'border-border/60 bg-background/70 hover:border-border'"
            @click="selectedLocalConsumerId = consumer.consumerId"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-2">
                  <p class="truncate text-sm font-semibold">{{ consumer.name || consumer.consumerId }}</p>
                  <Badge :class="kindToneClass(consumer.kind)">{{ consumer.kind }}</Badge>
                  <Badge variant="secondary">{{ t("{count} bindings", { count: consumerBindings(consumer.consumerId).length }) }}</Badge>
                </div>
                <p class="mt-2 truncate text-xs text-muted-foreground">{{ consumer.consumerId }}</p>
                <p class="mt-1 text-xs text-muted-foreground">{{ descriptorMetaLine(consumer) || t("No descriptor details") }}</p>
                <div class="mt-3 flex flex-wrap gap-2">
                  <Badge v-for="binding in consumerBindings(consumer.consumerId).slice(0, 3)" :key="binding.deliveryId" variant="secondary">
                    {{ binding.sourceId || t("Unknown source") }}
                  </Badge>
                  <span v-if="!consumerBindings(consumer.consumerId).length" class="text-xs text-muted-foreground">{{ t("No current source bindings") }}</span>
                </div>
              </div>
              <div class="flex flex-col items-end gap-2">
                <Button variant="outline" size="sm" data-stream-open-subscribe @click.stop="openSubscribeDialog(consumer.consumerId)">{{ t("Subscribe") }}</Button>
                <Button variant="ghost" size="sm" @click.stop="removeConsumer(consumer.consumerId)">{{ t("Remove") }}</Button>
              </div>
            </div>
          </button>

          <div v-if="!localConsumers.length" class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-10 text-center text-sm text-muted-foreground">
            {{ t("No local consumers yet. Use New Consumer to create one.") }}
          </div>
        </div>
      </section>

      <aside class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
        <CardHeader :title="t('Consumer Details')" :description="t('This side panel only shows what the consumer is currently bound to. Use the subscription dialog when you need to change it.')" title-class="text-lg" />

        <div v-if="selectedLocalConsumer" class="mt-4 space-y-4">
          <div class="flex flex-wrap items-center gap-2">
            <p class="text-lg font-semibold">{{ selectedLocalConsumer.name || selectedLocalConsumer.consumerId }}</p>
            <Badge :class="kindToneClass(selectedLocalConsumer.kind)">{{ selectedLocalConsumer.kind }}</Badge>
          </div>
          <p class="text-xs text-muted-foreground">{{ selectedLocalConsumer.consumerId }}</p>
          <div class="rounded-2xl border border-border/60 bg-background/70 p-4">
            <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Current source bindings") }}</p>
            <div class="mt-3 space-y-2">
              <article v-for="binding in consumerBindings(selectedLocalConsumer.consumerId)" :key="binding.deliveryId" class="rounded-xl border border-border/60 bg-card/80 px-3 py-3 text-sm">
                <div class="flex items-center justify-between gap-2">
                  <div>
                    <p class="font-semibold">{{ binding.sourceId || t("Unknown source") }}</p>
                    <p class="mt-1 text-xs text-muted-foreground">{{ t("Producer {id}", { id: binding.producer }) }} · {{ binding.state }}</p>
                  </div>
                  <Button variant="ghost" size="sm" @click="stream.unsubscribe(binding.deliveryId)">{{ t("Unsubscribe") }}</Button>
                </div>
              </article>
              <p v-if="!consumerBindings(selectedLocalConsumer.consumerId).length" class="text-sm text-muted-foreground">{{ t("This consumer is not bound to any source.") }}</p>
            </div>
          </div>
          <div class="rounded-2xl border border-border/60 bg-background/70 p-4">
            <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Metadata") }}</p>
            <pre class="mt-3 whitespace-pre-wrap break-words font-mono text-[12px] leading-6 text-muted-foreground">{{ metadataPreview(selectedLocalConsumer.metadataRaw) }}</pre>
          </div>
          <div class="flex flex-wrap gap-2">
            <Button @click="openSubscribeDialog(selectedLocalConsumer.consumerId)">{{ t("Open Subscribe Dialog") }}</Button>
            <Button variant="outline" @click="removeConsumer(selectedLocalConsumer.consumerId)">{{ t("Remove Consumer") }}</Button>
          </div>
        </div>

        <div v-else class="mt-4 rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-10 text-center text-sm text-muted-foreground">
          {{ t("Select a local consumer to review its bindings.") }}
        </div>
      </aside>
    </section>

    <section v-else class="grid gap-6 xl:grid-cols-[minmax(0,1.35fr)_380px]">
      <div class="grid gap-6 xl:grid-cols-2">
        <section class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
          <CardHeader :title="t('Source Catalog')" :description="t('Query producer catalogs and keep control selections focused here.')" title-class="text-lg">
            <template #actions>
              <Button variant="outline" size="sm" @click="refreshControlSources">
                <ScanSearch class="mr-2 h-4 w-4" />
                {{ t("Refresh") }}
              </Button>
            </template>
          </CardHeader>

          <div class="mt-4 rounded-2xl border border-border/60 bg-background/70 p-3">
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="sm:col-span-2">
                <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Producer Node ID") }}</label>
                <input v-model="controlSourceQuery.producer" :class="['mt-2', inputClass]" :placeholder="t('Producer Node ID')" />
              </div>
              <div>
                <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Kind") }}</label>
                <select v-model="controlSourceQuery.kind" :class="['mt-2', inputClass]">
                  <option value="">{{ t("All kinds") }}</option>
                  <option v-for="kind in streamKinds" :key="kind" :value="kind">{{ kind }}</option>
                </select>
              </div>
              <div>
                <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Tag") }}</label>
                <input v-model="controlSourceQuery.tag" :class="['mt-2', inputClass]" :placeholder="t('Tag filter')" />
              </div>
            </div>
          </div>

          <div class="mt-4 max-h-[420px] space-y-3 overflow-y-auto pr-1">
            <button v-for="source in catalogSources" :key="source.sourceId" type="button" class="w-full rounded-2xl border p-4 text-left transition" :class="stream.state.selectedSourceId === source.sourceId ? 'border-primary/50 bg-primary/10 shadow-sm' : 'border-border/60 bg-background/70 hover:border-border'" @click="stream.selectSource(source.sourceId)">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <p class="truncate text-sm font-semibold">{{ source.name || source.sourceId }}</p>
                    <Badge :class="kindToneClass(source.kind)">{{ source.kind }}</Badge>
                    <Badge variant="secondary">{{ t("Producer {id}", { id: source.producer }) }}</Badge>
                  </div>
                  <p class="mt-2 text-xs text-muted-foreground">{{ source.sourceId }}</p>
                </div>
              </div>
            </button>
            <div v-if="!catalogSources.length" class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-8 text-sm text-muted-foreground">
              {{ t("No sources loaded yet.") }}
            </div>
          </div>
        </section>

        <section class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
          <CardHeader :title="t('Consumer Catalog')" :description="t('Query consumer endpoints and compare them before connecting.')" title-class="text-lg">
            <template #actions>
              <Button variant="outline" size="sm" @click="refreshControlConsumers">
                <ScanSearch class="mr-2 h-4 w-4" />
                {{ t("Refresh") }}
              </Button>
            </template>
          </CardHeader>

          <div class="mt-4 rounded-2xl border border-border/60 bg-background/70 p-3">
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="sm:col-span-2">
                <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Consumer Node ID") }}</label>
                <input v-model="controlConsumerQuery.consumer" :class="['mt-2', inputClass]" :placeholder="t('Consumer Node ID')" />
              </div>
              <div>
                <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Kind") }}</label>
                <select v-model="controlConsumerQuery.kind" :class="['mt-2', inputClass]">
                  <option value="">{{ t("All kinds") }}</option>
                  <option v-for="kind in streamKinds" :key="kind" :value="kind">{{ kind }}</option>
                </select>
              </div>
              <div>
                <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Tag") }}</label>
                <input v-model="controlConsumerQuery.tag" :class="['mt-2', inputClass]" :placeholder="t('Tag filter')" />
              </div>
            </div>
          </div>

          <div class="mt-4 max-h-[420px] space-y-3 overflow-y-auto pr-1">
            <button v-for="consumer in catalogConsumers" :key="consumer.consumerId" type="button" class="w-full rounded-2xl border p-4 text-left transition" :class="stream.state.selectedConsumerId === consumer.consumerId ? 'border-primary/50 bg-primary/10 shadow-sm' : 'border-border/60 bg-background/70 hover:border-border'" @click="stream.selectConsumer(consumer.consumerId)">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <p class="truncate text-sm font-semibold">{{ consumer.name || consumer.consumerId }}</p>
                    <Badge :class="kindToneClass(consumer.kind)">{{ consumer.kind }}</Badge>
                    <Badge variant="secondary">{{ t("Consumer {id}", { id: consumer.consumer }) }}</Badge>
                  </div>
                  <p class="mt-2 text-xs text-muted-foreground">{{ consumer.consumerId }}</p>
                </div>
              </div>
            </button>
            <div v-if="!catalogConsumers.length" class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-8 text-sm text-muted-foreground">
              {{ t("No consumers loaded yet.") }}
            </div>
          </div>
        </section>
      </div>

      <aside class="space-y-6">
        <section class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
          <CardHeader :title="t('Control Target')" :description="t('Leave target aligned with Hub unless you are routing control requests elsewhere.')" title-class="text-lg">
            <template #actions>
              <Button variant="outline" size="sm" @click="refreshControlPlane">
                <RefreshCw class="mr-2 h-4 w-4" />
                {{ t("Refresh All") }}
              </Button>
            </template>
          </CardHeader>

          <div class="mt-4">
            <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Control Target") }}</label>
            <div class="mt-2 flex items-center gap-2 rounded-2xl border border-border/60 bg-background/70 px-3">
              <Target class="h-4 w-4 text-muted-foreground" />
              <input v-model="targetIdText" class="h-10 flex-1 bg-transparent text-sm outline-none" :placeholder="t('Hub ID')" />
            </div>
          </div>

          <div class="mt-5 rounded-2xl border border-border/60 bg-background/70 p-4">
            <div class="flex items-center gap-2">
              <Cable class="h-4 w-4 text-muted-foreground" />
              <p class="text-sm font-semibold">{{ t("Connect Pair") }}</p>
            </div>
            <p class="mt-2 text-xs text-muted-foreground">{{ t("Choose one source and one consumer, then connect or subscribe from this focused panel.") }}</p>
            <div class="mt-4 flex flex-wrap gap-2">
              <Button :disabled="!canConnect" @click="connectSelected">{{ t("Connect") }}</Button>
              <Button variant="outline" :disabled="!canSubscribe" @click="subscribeSelected">{{ t("Subscribe") }}</Button>
            </div>
            <p v-if="selectedControlSource && selectedControlConsumer && !canConnect" class="mt-3 text-xs text-rose-600">{{ t("Source kind and consumer kind must match.") }}</p>
          </div>
        </section>

        <section class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
          <CardHeader :title="t('Runtime Deliveries')" :description="t('Inspect deliveries and send lightweight runtime signals from one place.')" title-class="text-lg">
            <template #actions>
              <Badge variant="secondary">{{ t("Observed {count}", { count: deliveries.length }) }}</Badge>
            </template>
          </CardHeader>

          <div class="mt-4 flex flex-wrap gap-2">
            <Button variant="outline" size="sm" @click="refreshDeliveries">{{ t("Refresh") }}</Button>
            <Button variant="outline" :disabled="!selectedDelivery" @click="disconnectSelected">{{ t("Disconnect") }}</Button>
            <Button variant="outline" :disabled="!selectedDelivery" @click="unsubscribeSelected">{{ t("Unsubscribe") }}</Button>
            <Button variant="ghost" :disabled="!selectedDelivery" @click="signalSelected('pause')">
              <Pause class="mr-2 h-4 w-4" />
              {{ t("Pause") }}
            </Button>
            <Button variant="ghost" :disabled="!selectedDelivery" @click="signalSelected('resume')">
              <Play class="mr-2 h-4 w-4" />
              {{ t("Resume") }}
            </Button>
          </div>

          <div class="mt-4 max-h-[320px] space-y-3 overflow-y-auto pr-1">
            <button v-for="delivery in deliveries" :key="delivery.deliveryId" type="button" data-stream-delivery-row class="w-full rounded-2xl border p-4 text-left transition" :class="stream.state.selectedDeliveryId === delivery.deliveryId ? 'border-primary/50 bg-primary/10 shadow-sm' : 'border-border/60 bg-background/70 hover:border-border'" @click="stream.selectDelivery(delivery.deliveryId)">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <p class="truncate text-sm font-semibold">{{ delivery.deliveryId }}</p>
                    <Badge :class="kindToneClass(delivery.kind)">{{ delivery.kind }}</Badge>
                    <Badge variant="secondary">{{ delivery.state || t("Observed") }}</Badge>
                  </div>
                  <p class="mt-2 text-xs text-muted-foreground">{{ t("Frames {frames} · Bytes {bytes}", { frames: delivery.framesIn, bytes: delivery.bytesIn }) }}</p>
                  <p class="mt-1 text-xs text-muted-foreground">{{ t("Producer {producer} · Consumer {consumer}", { producer: delivery.producer, consumer: delivery.consumer }) }}</p>
                </div>
                <p class="text-xs text-muted-foreground">{{ formatTimestamp(delivery.updatedAt) }}</p>
              </div>
            </button>
            <div v-if="!deliveries.length" class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-8 text-sm text-muted-foreground">
              {{ t("No known deliveries yet.") }}
            </div>
          </div>

          <div v-if="selectedDelivery" class="mt-4 rounded-2xl border border-border/60 bg-background/70 p-4">
            <p class="text-sm font-semibold">{{ t("Selected Delivery") }}</p>
            <p class="mt-2 text-xs text-muted-foreground">{{ selectedDelivery.deliveryId }}</p>
            <div v-if="selectedDelivery.kind === 'text'" class="mt-4 rounded-2xl border border-emerald-200/20 bg-slate-950 px-4 py-4 text-sm text-emerald-100 shadow-inner">
              <div class="max-h-[220px] space-y-3 overflow-y-auto pr-1">
                <article v-for="frame in selectedTextFrames" :key="`${frame.deliveryId}:${frame.position}:${frame.updatedAt}`" class="rounded-xl border border-emerald-200/10 bg-white/5 px-3 py-3">
                  <div class="flex items-center justify-between gap-3 text-[11px] text-emerald-200/70">
                    <span>{{ frame.position }}</span>
                    <span>{{ formatTimestamp(frame.updatedAt) }}</span>
                  </div>
                  <pre class="mt-2 whitespace-pre-wrap break-words font-mono text-[13px] leading-6">{{ frame.text }}</pre>
                </article>
                <div v-if="!selectedTextFrames.length" class="rounded-xl border border-dashed border-emerald-200/15 px-3 py-6 text-center text-emerald-200/60">
                  {{ t("Waiting for text frames...") }}
                </div>
              </div>
            </div>
            <div v-else-if="selectedStats" class="mt-4 rounded-2xl border border-border/60 bg-card/80 px-4 py-3 text-sm text-muted-foreground">
              {{ t("Frames {frames} · Bytes {bytes} · ACK {ack} · Flags {flags}", { frames: selectedStats.framesIn, bytes: selectedStats.bytesIn, ack: selectedStats.lastAckPos, flags: selectedStats.lastFlags }) }}
            </div>
          </div>
        </section>
      </aside>
    </section>

    <Overlay :open="sourceDialogOpen" closeOnBackdrop trapFocus :initial-focus-selector="`#${sourceNameInputId}`" @close="closeSourceDialog">
      <div data-stream-source-dialog class="w-full max-w-2xl rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl">
        <CardHeader :title="t('New Source')" :description="t('Create a persistent local source without filling the main page with form fields.')" title-class="text-lg" />
        <div class="mt-5 grid gap-4 md:grid-cols-2">
          <div class="md:col-span-2">
            <label :for="sourceNameInputId" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Display name") }}</label>
            <input :id="sourceNameInputId" v-model="sourceDraft.name" :class="['mt-2', inputClass]" :placeholder="t('Display name')" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Kind") }}</label>
            <select v-model="sourceDraft.kind" :class="['mt-2', inputClass]"><option v-for="kind in streamKinds" :key="kind" :value="kind">{{ kind }}</option></select>
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Content type") }}</label>
            <input v-model="sourceDraft.contentType" :class="['mt-2', inputClass]" :placeholder="t('Content type')" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Mode") }}</label>
            <select v-model="sourceDraft.mode" :class="['mt-2', inputClass]"><option value="live">live</option><option value="bounded">bounded</option></select>
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Unit mode") }}</label>
            <select v-model="sourceDraft.unitMode" :class="['mt-2', inputClass]"><option value="frame">frame</option><option value="chunk">chunk</option></select>
          </div>
          <div class="md:col-span-2">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Tags") }}</label>
            <input v-model="sourceDraft.tagsText" :class="['mt-2', inputClass]" :placeholder="t('Tags, comma separated')" />
          </div>
          <div class="md:col-span-2">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Metadata") }}</label>
            <textarea v-model="sourceDraft.metadataText" :class="['mt-2', textAreaClass]" :placeholder="t('Optional metadata JSON')" />
          </div>
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="closeSourceDialog">{{ t("Cancel") }}</Button>
          <Button data-stream-submit-source @click="submitSource">{{ t("Create Source") }}</Button>
        </div>
      </div>
    </Overlay>

    <Overlay :open="consumerDialogOpen" closeOnBackdrop trapFocus :initial-focus-selector="`#${consumerNameInputId}`" @close="closeConsumerDialog">
      <div data-stream-consumer-dialog class="w-full max-w-2xl rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl">
        <CardHeader :title="t('New Consumer')" :description="t('Create a persistent local consumer endpoint without expanding the main list into a form page.')" title-class="text-lg" />
        <div class="mt-5 grid gap-4 md:grid-cols-2">
          <div class="md:col-span-2">
            <label :for="consumerNameInputId" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Display name") }}</label>
            <input :id="consumerNameInputId" v-model="consumerDraft.name" :class="['mt-2', inputClass]" :placeholder="t('Display name')" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Kind") }}</label>
            <select v-model="consumerDraft.kind" :class="['mt-2', inputClass]"><option v-for="kind in streamKinds" :key="kind" :value="kind">{{ kind }}</option></select>
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Content type") }}</label>
            <input v-model="consumerDraft.contentType" :class="['mt-2', inputClass]" :placeholder="t('Content type')" />
          </div>
          <div class="md:col-span-2">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Tags") }}</label>
            <input v-model="consumerDraft.tagsText" :class="['mt-2', inputClass]" :placeholder="t('Tags, comma separated')" />
          </div>
          <div class="md:col-span-2">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Metadata") }}</label>
            <textarea v-model="consumerDraft.metadataText" :class="['mt-2', textAreaClass]" :placeholder="t('Optional metadata JSON')" />
          </div>
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="closeConsumerDialog">{{ t("Cancel") }}</Button>
          <Button data-stream-submit-consumer @click="submitConsumer">{{ t("Create Consumer") }}</Button>
        </div>
      </div>
    </Overlay>

    <Overlay :open="sourceStudio.open" closeOnBackdrop trapFocus :initial-focus-selector="`#${sourceStudioInputId}`" @close="closeSourceStudio">
      <div data-stream-source-studio class="w-full max-w-3xl rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl">
        <CardHeader :title="t('Source Input Studio')" :description="t('Use a dedicated workspace for source input so the list page stays compact.')" title-class="text-lg" />

        <div v-if="sourceStudioSource" class="mt-5 space-y-4">
          <div class="flex flex-wrap items-center gap-2">
            <p class="text-lg font-semibold">{{ sourceStudioSource.name || sourceStudioSource.sourceId }}</p>
            <Badge :class="kindToneClass(sourceStudioSource.kind)">{{ sourceStudioSource.kind }}</Badge>
            <Badge variant="secondary">{{ t("{count} active bindings", { count: sourceBindings(sourceStudioSource.sourceId).length }) }}</Badge>
          </div>
          <p class="text-xs text-muted-foreground">{{ sourceStudioSource.sourceId }}</p>

          <div v-if="sourceStudioSource.kind === 'text'" class="space-y-4">
            <div>
              <label :for="sourceStudioInputId" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Text input") }}</label>
              <textarea :id="sourceStudioInputId" v-model="sourceStudio.text" :class="['mt-2', textAreaClass]" :placeholder="t('Type text for the active deliveries of this source...')" />
            </div>
            <div class="flex flex-wrap gap-2">
              <Button data-stream-submit-studio @click="publishSourceText">
                <Send class="mr-2 h-4 w-4" />
                {{ t("Send Text") }}
              </Button>
              <Button variant="outline" @click="closeSourceStudio">{{ t("Close") }}</Button>
            </div>
          </div>

          <div v-else class="rounded-2xl border border-border/60 bg-background/70 px-4 py-4 text-sm text-muted-foreground">
            {{ t("Direct input is currently available only for text sources. Other kinds still use the control plane and runtime viewer.") }}
          </div>

          <div class="rounded-2xl border border-border/60 bg-background/70 p-4">
            <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Recent sends") }}</p>
            <div class="mt-3 space-y-2">
              <article v-for="item in sourceStudio.sentItems" :key="`${item.at}:${item.text}`" class="rounded-xl border border-border/60 bg-card/80 px-3 py-3 text-sm">
                <div class="flex items-center justify-between gap-3">
                  <p class="font-semibold">{{ t("Sent to {count} deliveries", { count: item.sent }) }}</p>
                  <span class="text-xs text-muted-foreground">{{ formatTimestamp(item.at) }}</span>
                </div>
                <pre class="mt-2 whitespace-pre-wrap break-words font-mono text-[12px] leading-6 text-muted-foreground">{{ item.text }}</pre>
              </article>
              <p v-if="!sourceStudio.sentItems.length" class="text-sm text-muted-foreground">{{ t("No text has been sent from this studio yet.") }}</p>
            </div>
          </div>
        </div>
      </div>
    </Overlay>

    <Overlay :open="subscribeDialog.open" closeOnBackdrop trapFocus @close="closeSubscribeDialog">
      <div data-stream-subscribe-dialog class="w-full max-w-3xl rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl">
        <CardHeader :title="t('Subscribe Consumer')" :description="t('Choose a source from the current node or another node, then subscribe without expanding the main consumer list into a form.') " title-class="text-lg" />

        <div v-if="subscribeConsumer" class="mt-5 space-y-4">
          <div class="flex flex-wrap items-center gap-2">
            <p class="text-lg font-semibold">{{ subscribeConsumer.name || subscribeConsumer.consumerId }}</p>
            <Badge :class="kindToneClass(subscribeConsumer.kind)">{{ subscribeConsumer.kind }}</Badge>
          </div>

          <div class="rounded-2xl border border-border/60 bg-background/70 p-3">
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="sm:col-span-2">
                <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Producer Node ID") }}</label>
                <input v-model="subscribeQuery.producer" :class="['mt-2', inputClass]" :placeholder="t('Producer Node ID')" />
              </div>
              <div>
                <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Kind") }}</label>
                <select v-model="subscribeQuery.kind" :class="['mt-2', inputClass]">
                  <option value="">{{ t("All kinds") }}</option>
                  <option v-for="kind in streamKinds" :key="kind" :value="kind">{{ kind }}</option>
                </select>
              </div>
              <div>
                <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Tag") }}</label>
                <input v-model="subscribeQuery.tag" :class="['mt-2', inputClass]" :placeholder="t('Tag filter')" />
              </div>
            </div>
            <div class="mt-3 flex flex-wrap gap-2">
              <Button variant="outline" size="sm" @click="refreshSubscribeSources">
                <ScanSearch class="mr-2 h-4 w-4" />
                {{ t("Refresh") }}
              </Button>
            </div>
          </div>

          <div class="max-h-[320px] space-y-3 overflow-y-auto pr-1">
            <button v-for="source in catalogSources" :key="source.sourceId" type="button" data-stream-subscribe-source-row class="w-full rounded-2xl border p-4 text-left transition" :class="subscribeQuery.selectedSourceId === source.sourceId ? 'border-primary/50 bg-primary/10 shadow-sm' : 'border-border/60 bg-background/70 hover:border-border'" @click="subscribeQuery.selectedSourceId = source.sourceId">
              <div class="flex flex-wrap items-center gap-2">
                <p class="truncate text-sm font-semibold">{{ source.name || source.sourceId }}</p>
                <Badge :class="kindToneClass(source.kind)">{{ source.kind }}</Badge>
                <Badge variant="secondary">{{ t("Producer {id}", { id: source.producer }) }}</Badge>
              </div>
              <p class="mt-2 text-xs text-muted-foreground">{{ source.sourceId }}</p>
            </button>
            <div v-if="!catalogSources.length" class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-8 text-sm text-muted-foreground">
              {{ t("No subscription sources loaded yet.") }}
            </div>
          </div>

          <div class="flex justify-end gap-2">
            <Button variant="outline" @click="closeSubscribeDialog">{{ t("Cancel") }}</Button>
            <Button data-stream-submit-subscribe @click="subscribeFromDialog">{{ t("Subscribe") }}</Button>
          </div>
        </div>
      </div>
    </Overlay>
  </section>
</template>
