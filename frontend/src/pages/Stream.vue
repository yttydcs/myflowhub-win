<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue"
import { Cable, Pause, Play, Plus, RefreshCw, ScanSearch, Target } from "lucide-vue-next"
import CardHeader from "@/components/CardHeader.vue"
import PageHero from "@/components/PageHero.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import { t } from "@/i18n"
import {
  streamKinds,
  type StreamConsumer,
  type StreamConsumerDraft,
  type StreamSource,
  type StreamSourceDraft,
  useStreamStore
} from "@/stores/stream"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"

const stream = useStreamStore()
const sessionStore = useSessionStore()
const toast = useToastStore()

const sourceNameInputId = "stream-source-name"
const consumerNameInputId = "stream-consumer-name"

const inputClass =
  "h-10 w-full rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
const textAreaClass =
  "min-h-[132px] w-full rounded-2xl border border-input bg-background px-3 py-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

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

const sourceQuery = reactive({ producer: "", kind: "", tag: "" })
const consumerQuery = reactive({ consumer: "", kind: "", tag: "" })
const sourceDraft = reactive<StreamSourceDraft>(defaultSourceDraft())
const consumerDraft = reactive<StreamConsumerDraft>(defaultConsumerDraft())

const sourceDialogOpen = ref(false)
const consumerDialogOpen = ref(false)

const selfNodeId = computed(() => Number(sessionStore.auth.nodeId || 0))
const hubId = computed(() => Number(sessionStore.auth.hubId || 0))
const sources = computed(() => stream.state.sources)
const consumers = computed(() => stream.state.consumers)
const deliveries = computed(() => stream.state.deliveries)
const selectedSource = computed(() => sources.value.find((item) => item.sourceId === stream.state.selectedSourceId) ?? null)
const selectedConsumer = computed(() => consumers.value.find((item) => item.consumerId === stream.state.selectedConsumerId) ?? null)
const selectedDelivery = computed(() => deliveries.value.find((item) => item.deliveryId === stream.state.selectedDeliveryId) ?? null)
const selectedTextFrames = computed(() => (selectedDelivery.value ? stream.textFramesFor(selectedDelivery.value.deliveryId) : []))
const selectedStats = computed(() => (selectedDelivery.value ? stream.statsFor(selectedDelivery.value.deliveryId) : null))
const targetIdText = computed({
  get: () => stream.state.targetId,
  set: (value: string) => stream.setTargetId(value)
})

const localSourceCount = computed(() => sources.value.filter((item) => item.producer === selfNodeId.value).length)
const localConsumerCount = computed(() => consumers.value.filter((item) => item.consumer === selfNodeId.value).length)
const latestActivityAt = computed(() => stream.state.lastEventAt || stream.state.lastSyncAt)
const resolvedTargetId = computed(() => targetIdText.value || (hubId.value ? String(hubId.value) : ""))
const sourceFilters = computed(() => {
  const filters: string[] = []
  if (sourceQuery.producer.trim()) filters.push(t("Producer {id}", { id: sourceQuery.producer.trim() }))
  if (sourceQuery.kind.trim()) filters.push(t("Kind {kind}", { kind: sourceQuery.kind.trim() }))
  if (sourceQuery.tag.trim()) filters.push(t("Tag {tag}", { tag: sourceQuery.tag.trim() }))
  return filters
})
const consumerFilters = computed(() => {
  const filters: string[] = []
  if (consumerQuery.consumer.trim()) filters.push(t("Consumer {id}", { id: consumerQuery.consumer.trim() }))
  if (consumerQuery.kind.trim()) filters.push(t("Kind {kind}", { kind: consumerQuery.kind.trim() }))
  if (consumerQuery.tag.trim()) filters.push(t("Tag {tag}", { tag: consumerQuery.tag.trim() }))
  return filters
})

const summaryCards = computed(() => [
  {
    label: t("Sources Loaded"),
    value: String(sources.value.length),
    hint: t("Local {count}", { count: localSourceCount.value })
  },
  {
    label: t("Consumers Loaded"),
    value: String(consumers.value.length),
    hint: t("Local {count}", { count: localConsumerCount.value })
  },
  {
    label: t("Known Deliveries"),
    value: String(deliveries.value.length),
    hint: selectedDelivery.value ? t("Selected {id}", { id: selectedDelivery.value.deliveryId }) : t("No selection")
  },
  {
    label: t("Last Runtime Event"),
    value: latestActivityAt.value ? formatTimestamp(latestActivityAt.value) : t("No activity yet"),
    hint: stream.state.lastSyncAt ? t("Last sync {time}", { time: formatTimestamp(stream.state.lastSyncAt) }) : t("Waiting for first sync")
  }
])

const canConnect = computed(() => {
  if (!selectedSource.value || !selectedConsumer.value) return false
  return selectedSource.value.kind === selectedConsumer.value.kind
})

const canSubscribe = computed(() => {
  if (!selectedConsumer.value || !canConnect.value) return false
  return selectedConsumer.value.consumer === selfNodeId.value
})

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

const descriptorMetaLine = (item: Pick<StreamSource, "contentType" | "mode" | "unitMode"> | Pick<StreamConsumer, "contentType">) => {
  return [
    "contentType" in item ? item.contentType : "",
    "mode" in item ? item.mode : "",
    "unitMode" in item ? item.unitMode : ""
  ]
    .map((entry) => String(entry ?? "").trim())
    .filter(Boolean)
    .join(" · ")
}

const tagsOrFallback = (tags: string[]) => (tags.length ? tags : [t("No tags")])
const metadataPreview = (value: string) => String(value ?? "").trim() || t("No metadata")
const isLocalSource = (source: StreamSource) => source.producer === selfNodeId.value
const isLocalConsumer = (consumer: StreamConsumer) => consumer.consumer === selfNodeId.value

const resetSourceDraft = () => {
  Object.assign(sourceDraft, defaultSourceDraft())
}

const resetConsumerDraft = () => {
  Object.assign(consumerDraft, defaultConsumerDraft())
}

const openSourceDialog = () => {
  resetSourceDraft()
  sourceDialogOpen.value = true
}

const openConsumerDialog = () => {
  resetConsumerDraft()
  consumerDialogOpen.value = true
}

const closeSourceDialog = () => {
  sourceDialogOpen.value = false
  resetSourceDraft()
}

const closeConsumerDialog = () => {
  consumerDialogOpen.value = false
  resetConsumerDraft()
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

const querySources = () => stream.listSources(sourceQuery.producer, sourceQuery.kind, sourceQuery.tag)
const queryConsumers = () => stream.listConsumers(consumerQuery.consumer, consumerQuery.kind, consumerQuery.tag)
const primeCatalogs = async () => {
  await Promise.all([querySources(), queryConsumers(), stream.loadDeliveries()])
}
const refreshSources = () => withToast(querySources, "Sources refreshed.", "Failed to query sources.")
const refreshConsumers = () => withToast(queryConsumers, "Consumers refreshed.", "Failed to query consumers.")
const refreshAll = () =>
  withToast(
    async () => {
      await primeCatalogs()
    },
    "Stream control plane refreshed.",
    "Failed to refresh Stream control plane."
  )

const submitSource = () =>
  withToast(async () => {
    await stream.announceSource(sourceDraft)
    closeSourceDialog()
  }, "Local source created.", "Failed to create local source.")

const submitConsumer = () =>
  withToast(async () => {
    await stream.announceConsumer(consumerDraft)
    closeConsumerDialog()
  }, "Local consumer created.", "Failed to create local consumer.")

const connectSelected = () =>
  selectedSource.value && selectedConsumer.value
    ? withToast(
        () =>
          stream.connect({
            producer: selectedSource.value!.producer,
            sourceId: selectedSource.value!.sourceId,
            consumer: selectedConsumer.value!.consumer,
            consumerId: selectedConsumer.value!.consumerId
          }),
        "Delivery connected.",
        "Failed to connect delivery."
      )
    : Promise.resolve()

const subscribeSelected = () =>
  selectedSource.value && selectedConsumer.value
    ? withToast(
        () =>
          stream.subscribe({
            producer: selectedSource.value!.producer,
            sourceId: selectedSource.value!.sourceId,
            consumerId: selectedConsumer.value!.consumerId
          }),
        "Subscribed.",
        "Failed to subscribe."
      )
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

const removeSource = (sourceId: string) => withToast(() => stream.withdrawSource(sourceId), "Source removed.", "Failed to remove source.")
const removeConsumer = (consumerId: string) => withToast(() => stream.withdrawConsumer(consumerId), "Consumer removed.", "Failed to remove consumer.")

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  () => {
    stream.setIdentity(selfNodeId.value, hubId.value)
    if (!targetIdText.value && hubId.value > 0) targetIdText.value = String(hubId.value)
    if (!sourceQuery.producer && selfNodeId.value > 0) sourceQuery.producer = String(selfNodeId.value)
    if (!consumerQuery.consumer && selfNodeId.value > 0) consumerQuery.consumer = String(selfNodeId.value)
    if (selfNodeId.value > 0 && hubId.value > 0) {
      void primeCatalogs().catch((err) => console.warn(err))
    }
  },
  { immediate: true }
)
</script>

<template>
  <section class="space-y-6" data-stream-page>
    <PageHero :description="t('Track stream catalogs, connect the pair you need, and open forms only when you are ready to add a new endpoint.')">
      <template #actions>
        <Badge variant="secondary">{{ t("Self {id}", { id: selfNodeId || "-" }) }}</Badge>
        <Badge variant="secondary">{{ t("Hub {id}", { id: hubId || "-" }) }}</Badge>
        <Badge variant="secondary">{{ t("Target {id}", { id: resolvedTargetId || "-" }) }}</Badge>
        <Button variant="outline" @click="refreshAll">
          <RefreshCw class="mr-2 h-4 w-4" />
          {{ t("Refresh All") }}
        </Button>
        <Button variant="outline" data-stream-open-source @click="openSourceDialog">
          <Plus class="mr-2 h-4 w-4" />
          {{ t("New Source") }}
        </Button>
        <Button data-stream-open-consumer @click="openConsumerDialog">
          <Plus class="mr-2 h-4 w-4" />
          {{ t("New Consumer") }}
        </Button>
      </template>
    </PageHero>

    <section class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <article v-for="card in summaryCards" :key="card.label" class="rounded-2xl border border-border/60 bg-card/90 px-4 py-4 text-card-foreground shadow-sm">
        <p class="text-[11px] font-semibold uppercase tracking-[0.24em] text-muted-foreground">{{ card.label }}</p>
        <p class="mt-3 text-xl font-semibold">{{ card.value }}</p>
        <p class="mt-2 text-xs text-muted-foreground">{{ card.hint }}</p>
      </article>
    </section>

    <section class="grid gap-6 xl:grid-cols-[minmax(0,1.35fr)_380px]">
      <div class="grid gap-6 xl:grid-cols-2">
        <section class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
          <CardHeader :title="t('Source Catalog')" :description="t('Query producer catalogs and keep the main canvas focused on the endpoints you are comparing.')" title-class="text-lg">
            <template #actions>
              <Button variant="outline" size="sm" @click="refreshSources">
                <ScanSearch class="mr-2 h-4 w-4" />
                {{ t("Refresh") }}
              </Button>
            </template>
          </CardHeader>

          <div class="mt-4 rounded-2xl border border-border/60 bg-background/70 p-3">
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="sm:col-span-2">
                <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Producer Node ID") }}</label>
                <input v-model="sourceQuery.producer" :class="['mt-2', inputClass]" :placeholder="t('Producer Node ID')" />
              </div>
              <div>
                <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Kind") }}</label>
                <select v-model="sourceQuery.kind" :class="['mt-2', inputClass]">
                  <option value="">{{ t("All kinds") }}</option>
                  <option v-for="kind in streamKinds" :key="kind" :value="kind">{{ kind }}</option>
                </select>
              </div>
              <div>
                <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Tag") }}</label>
                <input v-model="sourceQuery.tag" :class="['mt-2', inputClass]" :placeholder="t('Tag filter')" />
              </div>
            </div>
            <div class="mt-3 flex flex-wrap gap-2">
              <Badge v-for="filter in sourceFilters" :key="filter" variant="secondary">{{ filter }}</Badge>
              <span v-if="!sourceFilters.length" class="text-xs text-muted-foreground">{{ t("No filters") }}</span>
            </div>
          </div>

          <div class="mt-4 max-h-[420px] space-y-3 overflow-y-auto pr-1">
            <button
              v-for="source in sources"
              :key="source.sourceId"
              type="button"
              data-stream-source-row
              class="w-full rounded-2xl border p-4 text-left transition"
              :class="stream.state.selectedSourceId === source.sourceId ? 'border-primary/50 bg-primary/10 shadow-sm' : 'border-border/60 bg-background/70 hover:border-border'"
              @click="stream.selectSource(source.sourceId)"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <p class="truncate text-sm font-semibold">{{ source.name || source.sourceId }}</p>
                    <Badge :class="kindToneClass(source.kind)">{{ source.kind }}</Badge>
                    <Badge variant="secondary">{{ t("Producer {id}", { id: source.producer }) }}</Badge>
                  </div>
                  <p class="mt-2 truncate text-xs text-muted-foreground">{{ source.sourceId }}</p>
                  <p class="mt-1 text-xs text-muted-foreground">{{ descriptorMetaLine(source) }}</p>
                  <div class="mt-3 flex flex-wrap gap-2">
                    <Badge v-for="tag in tagsOrFallback(source.tags)" :key="`${source.sourceId}:${tag}`" variant="secondary">{{ tag }}</Badge>
                  </div>
                </div>
                <Button v-if="isLocalSource(source)" variant="ghost" size="sm" @click.stop="removeSource(source.sourceId)">{{ t("Remove") }}</Button>
              </div>
            </button>

            <div v-if="!sources.length" class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-8 text-sm text-muted-foreground">
              {{ t("No sources loaded yet.") }}
            </div>
          </div>
        </section>

        <section class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
          <CardHeader :title="t('Consumer Catalog')" :description="t('Query consumer endpoints and compare them side by side before connecting or subscribing.')" title-class="text-lg">
            <template #actions>
              <Button variant="outline" size="sm" @click="refreshConsumers">
                <ScanSearch class="mr-2 h-4 w-4" />
                {{ t("Refresh") }}
              </Button>
            </template>
          </CardHeader>

          <div class="mt-4 rounded-2xl border border-border/60 bg-background/70 p-3">
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="sm:col-span-2">
                <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Consumer Node ID") }}</label>
                <input v-model="consumerQuery.consumer" :class="['mt-2', inputClass]" :placeholder="t('Consumer Node ID')" />
              </div>
              <div>
                <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Kind") }}</label>
                <select v-model="consumerQuery.kind" :class="['mt-2', inputClass]">
                  <option value="">{{ t("All kinds") }}</option>
                  <option v-for="kind in streamKinds" :key="kind" :value="kind">{{ kind }}</option>
                </select>
              </div>
              <div>
                <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Tag") }}</label>
                <input v-model="consumerQuery.tag" :class="['mt-2', inputClass]" :placeholder="t('Tag filter')" />
              </div>
            </div>
            <div class="mt-3 flex flex-wrap gap-2">
              <Badge v-for="filter in consumerFilters" :key="filter" variant="secondary">{{ filter }}</Badge>
              <span v-if="!consumerFilters.length" class="text-xs text-muted-foreground">{{ t("No filters") }}</span>
            </div>
          </div>

          <div class="mt-4 max-h-[420px] space-y-3 overflow-y-auto pr-1">
            <button
              v-for="consumer in consumers"
              :key="consumer.consumerId"
              type="button"
              data-stream-consumer-row
              class="w-full rounded-2xl border p-4 text-left transition"
              :class="stream.state.selectedConsumerId === consumer.consumerId ? 'border-primary/50 bg-primary/10 shadow-sm' : 'border-border/60 bg-background/70 hover:border-border'"
              @click="stream.selectConsumer(consumer.consumerId)"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <p class="truncate text-sm font-semibold">{{ consumer.name || consumer.consumerId }}</p>
                    <Badge :class="kindToneClass(consumer.kind)">{{ consumer.kind }}</Badge>
                    <Badge variant="secondary">{{ t("Consumer {id}", { id: consumer.consumer }) }}</Badge>
                  </div>
                  <p class="mt-2 truncate text-xs text-muted-foreground">{{ consumer.consumerId }}</p>
                  <p class="mt-1 text-xs text-muted-foreground">{{ descriptorMetaLine(consumer) }}</p>
                  <div class="mt-3 flex flex-wrap gap-2">
                    <Badge v-for="tag in tagsOrFallback(consumer.tags)" :key="`${consumer.consumerId}:${tag}`" variant="secondary">{{ tag }}</Badge>
                  </div>
                </div>
                <Button v-if="isLocalConsumer(consumer)" variant="ghost" size="sm" @click.stop="removeConsumer(consumer.consumerId)">{{ t("Remove") }}</Button>
              </div>
            </button>

            <div v-if="!consumers.length" class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-8 text-sm text-muted-foreground">
              {{ t("No consumers loaded yet.") }}
            </div>
          </div>
        </section>
      </div>

      <aside class="space-y-6">
        <section class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
          <CardHeader :title="t('Connection Target')" :description="t('The control request is sent to this node. Leave it aligned with Hub unless you are targeting another control plane.')" title-class="text-lg" />

          <div class="mt-4">
            <label class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Control Target") }}</label>
            <div class="mt-2 flex items-center gap-2 rounded-2xl border border-border/60 bg-background/70 px-3">
              <Target class="h-4 w-4 text-muted-foreground" />
              <input v-model="targetIdText" class="h-10 flex-1 bg-transparent text-sm outline-none" :placeholder="t('Hub ID')" />
            </div>
          </div>

          <div class="mt-5 space-y-3">
            <article class="rounded-2xl border border-border/60 bg-background/70 p-4">
              <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Selected Source") }}</p>
              <div v-if="selectedSource" class="mt-3 space-y-3">
                <div class="flex flex-wrap items-center gap-2">
                  <p class="text-sm font-semibold">{{ selectedSource.name || selectedSource.sourceId }}</p>
                  <Badge :class="kindToneClass(selectedSource.kind)">{{ selectedSource.kind }}</Badge>
                </div>
                <p class="text-xs text-muted-foreground">{{ selectedSource.sourceId }}</p>
                <p class="text-xs text-muted-foreground">{{ descriptorMetaLine(selectedSource) }}</p>
                <div class="flex flex-wrap gap-2">
                  <Badge v-for="tag in tagsOrFallback(selectedSource.tags)" :key="`${selectedSource.sourceId}-tag-${tag}`" variant="secondary">{{ tag }}</Badge>
                </div>
                <div class="rounded-xl border border-border/60 bg-card/80 p-3 text-xs text-muted-foreground">
                  <p class="font-semibold text-foreground">{{ t("Metadata") }}</p>
                  <pre class="mt-2 max-h-28 overflow-y-auto whitespace-pre-wrap break-words font-mono text-[11px] leading-5">{{ metadataPreview(selectedSource.metadataRaw) }}</pre>
                </div>
              </div>
              <p v-else class="mt-3 text-sm text-muted-foreground">{{ t("Select a source from the left list.") }}</p>
            </article>

            <article class="rounded-2xl border border-border/60 bg-background/70 p-4">
              <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Selected Consumer") }}</p>
              <div v-if="selectedConsumer" class="mt-3 space-y-3">
                <div class="flex flex-wrap items-center gap-2">
                  <p class="text-sm font-semibold">{{ selectedConsumer.name || selectedConsumer.consumerId }}</p>
                  <Badge :class="kindToneClass(selectedConsumer.kind)">{{ selectedConsumer.kind }}</Badge>
                </div>
                <p class="text-xs text-muted-foreground">{{ selectedConsumer.consumerId }}</p>
                <p class="text-xs text-muted-foreground">{{ descriptorMetaLine(selectedConsumer) }}</p>
                <div class="flex flex-wrap gap-2">
                  <Badge v-for="tag in tagsOrFallback(selectedConsumer.tags)" :key="`${selectedConsumer.consumerId}-tag-${tag}`" variant="secondary">{{ tag }}</Badge>
                </div>
                <div class="rounded-xl border border-border/60 bg-card/80 p-3 text-xs text-muted-foreground">
                  <p class="font-semibold text-foreground">{{ t("Metadata") }}</p>
                  <pre class="mt-2 max-h-28 overflow-y-auto whitespace-pre-wrap break-words font-mono text-[11px] leading-5">{{ metadataPreview(selectedConsumer.metadataRaw) }}</pre>
                </div>
              </div>
              <p v-else class="mt-3 text-sm text-muted-foreground">{{ t("Select a consumer from the left list.") }}</p>
            </article>
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
            <p v-if="selectedSource && selectedConsumer && !canConnect" class="mt-3 text-xs text-rose-600">{{ t("Source kind and consumer kind must match.") }}</p>
            <p v-else-if="selectedConsumer && !canSubscribe && canConnect" class="mt-3 text-xs text-muted-foreground">{{ t("Subscribe is for local consumer endpoints; use Connect for remote consumer nodes.") }}</p>
          </div>
        </section>

        <section class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
          <CardHeader :title="t('Runtime Deliveries')" :description="t('Inspect active or closed deliveries and send lightweight control signals from the same queue.')" title-class="text-lg">
            <template #actions>
              <Badge variant="secondary">{{ t("Observed {count}", { count: deliveries.length }) }}</Badge>
            </template>
          </CardHeader>

          <div class="mt-4 flex flex-wrap gap-2">
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
            <Button variant="ghost" :disabled="!selectedDelivery" @click="signalSelected('keyframe_request')">{{ t("Keyframe") }}</Button>
          </div>

          <div class="mt-4 max-h-[320px] space-y-3 overflow-y-auto pr-1">
            <button
              v-for="delivery in deliveries"
              :key="delivery.deliveryId"
              type="button"
              data-stream-delivery-row
              class="w-full rounded-2xl border p-4 text-left transition"
              :class="stream.state.selectedDeliveryId === delivery.deliveryId ? 'border-primary/50 bg-primary/10 shadow-sm' : 'border-border/60 bg-background/70 hover:border-border'"
              @click="stream.selectDelivery(delivery.deliveryId)"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <p class="truncate text-sm font-semibold">{{ delivery.deliveryId }}</p>
                    <Badge :class="kindToneClass(delivery.kind)">{{ delivery.kind }}</Badge>
                    <Badge variant="secondary">{{ delivery.state || t("Observed") }}</Badge>
                  </div>
                  <p class="mt-2 text-xs text-muted-foreground">{{ t("Frames {frames} · Bytes {bytes}", { frames: delivery.framesIn, bytes: delivery.bytesIn }) }}</p>
                  <p class="mt-1 text-xs text-muted-foreground">{{ t("Producer {producer} · Consumer {consumer}", { producer: delivery.producer, consumer: delivery.consumer }) }}</p>
                  <p v-if="delivery.lastError" class="mt-2 text-xs text-rose-600">{{ delivery.lastError }}</p>
                </div>
                <p class="text-xs text-muted-foreground">{{ formatTimestamp(delivery.updatedAt) }}</p>
              </div>
            </button>

            <div v-if="!deliveries.length" class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-8 text-sm text-muted-foreground">
              {{ t("No known deliveries yet.") }}
            </div>
          </div>
        </section>

        <section class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
          <CardHeader :title="t('Runtime Viewer')" :description="t('Read the selected delivery without opening a separate window.')" title-class="text-lg" />

          <div v-if="selectedDelivery" class="mt-4 space-y-4">
            <div class="grid gap-3 sm:grid-cols-2">
              <article class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3">
                <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Last position") }}</p>
                <p class="mt-2 text-lg font-semibold">{{ selectedDelivery.lastPosition }}</p>
              </article>
              <article class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3">
                <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Frames / Bytes") }}</p>
                <p class="mt-2 text-lg font-semibold">{{ selectedDelivery.framesIn }} / {{ selectedDelivery.bytesIn }}</p>
              </article>
            </div>

            <div v-if="selectedDelivery.kind === 'text'" class="rounded-2xl border border-emerald-200/20 bg-slate-950 px-4 py-4 text-sm text-emerald-100 shadow-inner">
              <div class="max-h-[360px] space-y-3 overflow-y-auto pr-1">
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

            <div v-else class="rounded-2xl border border-border/60 bg-background/70 p-4">
              <p class="text-sm text-muted-foreground">{{ t("This viewer currently exposes delivery health and stats. Rendering can be added later without changing the control plane.") }}</p>
              <div v-if="selectedStats" class="mt-4 rounded-2xl border border-border/60 bg-card/80 px-4 py-3 text-sm text-muted-foreground">
                {{ t("Frames {frames} · Bytes {bytes} · ACK {ack} · Flags {flags}", { frames: selectedStats.framesIn, bytes: selectedStats.bytesIn, ack: selectedStats.lastAckPos, flags: selectedStats.lastFlags }) }}
              </div>
            </div>
          </div>

          <div v-else class="mt-4 rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-10 text-center text-sm text-muted-foreground">
            {{ t("Select a delivery to inspect its runtime state.") }}
          </div>
        </section>
      </aside>
    </section>

    <Overlay :open="sourceDialogOpen" closeOnBackdrop trapFocus :initial-focus-selector="`#${sourceNameInputId}`" @close="closeSourceDialog">
      <div data-stream-source-dialog class="w-full max-w-2xl rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl">
        <CardHeader :title="t('New Source')" :description="t('Local producer endpoint. Open the form only when you need to publish a new source.')" title-class="text-lg" />

        <div class="mt-5 grid gap-4 md:grid-cols-2">
          <div class="md:col-span-2">
            <label :for="sourceNameInputId" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Display name") }}</label>
            <input :id="sourceNameInputId" v-model="sourceDraft.name" :class="['mt-2', inputClass]" :placeholder="t('Display name')" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Kind") }}</label>
            <select v-model="sourceDraft.kind" :class="['mt-2', inputClass]">
              <option v-for="kind in streamKinds" :key="kind" :value="kind">{{ kind }}</option>
            </select>
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Content type") }}</label>
            <input v-model="sourceDraft.contentType" :class="['mt-2', inputClass]" :placeholder="t('Content type')" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Mode") }}</label>
            <select v-model="sourceDraft.mode" :class="['mt-2', inputClass]">
              <option value="live">live</option>
              <option value="bounded">bounded</option>
            </select>
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Unit mode") }}</label>
            <select v-model="sourceDraft.unitMode" :class="['mt-2', inputClass]">
              <option value="frame">frame</option>
              <option value="chunk">chunk</option>
            </select>
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
        <CardHeader :title="t('New Consumer')" :description="t('Local consumer endpoint. Open the form only when you need to add a new destination.')" title-class="text-lg" />

        <div class="mt-5 grid gap-4 md:grid-cols-2">
          <div class="md:col-span-2">
            <label :for="consumerNameInputId" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Display name") }}</label>
            <input :id="consumerNameInputId" v-model="consumerDraft.name" :class="['mt-2', inputClass]" :placeholder="t('Display name')" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Kind") }}</label>
            <select v-model="consumerDraft.kind" :class="['mt-2', inputClass]">
              <option v-for="kind in streamKinds" :key="kind" :value="kind">{{ kind }}</option>
            </select>
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
  </section>
</template>
