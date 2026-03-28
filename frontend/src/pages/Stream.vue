<script setup lang="ts">
import { computed, onMounted, reactive, watch } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { t } from "@/i18n"
import {
  streamKinds,
  type StreamConsumerDraft,
  type StreamSourceDraft,
  useStreamStore
} from "@/stores/stream"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"

const stream = useStreamStore()
const sessionStore = useSessionStore()
const toast = useToastStore()

const sourceQuery = reactive({ producer: "", kind: "", tag: "" })
const consumerQuery = reactive({ consumer: "", kind: "", tag: "" })
const sourceDraft = reactive<StreamSourceDraft>({
  sourceId: "",
  name: "",
  kind: "text",
  contentType: "text/plain",
  mode: "live",
  unitMode: "frame",
  tagsText: "",
  metadataText: ""
})
const consumerDraft = reactive<StreamConsumerDraft>({
  consumerId: "",
  name: "",
  kind: "text",
  contentType: "text/plain",
  tagsText: "",
  metadataText: ""
})

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

const canConnect = computed(() => {
  if (!selectedSource.value || !selectedConsumer.value) return false
  return selectedSource.value.kind === selectedConsumer.value.kind
})

const canSubscribe = computed(() => {
  if (!selectedConsumer.value || !canConnect.value) return false
  return selectedConsumer.value.consumer === selfNodeId.value
})

const withToast = async (action: () => Promise<unknown>, ok: string, fail: string) => {
  try {
    await action()
    toast.success(t(ok))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t(fail))
  }
}

const refreshSources = () => withToast(() => stream.listSources(sourceQuery.producer, sourceQuery.kind, sourceQuery.tag), "Sources refreshed.", "Failed to query sources.")
const refreshConsumers = () => withToast(() => stream.listConsumers(consumerQuery.consumer, consumerQuery.kind, consumerQuery.tag), "Consumers refreshed.", "Failed to query consumers.")
const submitSource = () =>
  withToast(async () => {
    await stream.announceSource(sourceDraft)
    sourceDraft.sourceId = ""
    sourceDraft.name = ""
    sourceDraft.tagsText = ""
    sourceDraft.metadataText = ""
  }, "Source announced.", "Failed to announce source.")
const submitConsumer = () =>
  withToast(async () => {
    await stream.announceConsumer(consumerDraft)
    consumerDraft.consumerId = ""
    consumerDraft.name = ""
    consumerDraft.tagsText = ""
    consumerDraft.metadataText = ""
  }, "Consumer announced.", "Failed to announce consumer.")
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
const removeSource = (sourceId: string) => withToast(() => stream.withdrawSource(sourceId), "Source withdrawn.", "Failed to withdraw source.")
const removeConsumer = (consumerId: string) => withToast(() => stream.withdrawConsumer(consumerId), "Consumer withdrawn.", "Failed to withdraw consumer.")

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  () => {
    stream.setIdentity(selfNodeId.value, hubId.value)
    if (!targetIdText.value && hubId.value > 0) targetIdText.value = String(hubId.value)
    if (!sourceQuery.producer && selfNodeId.value > 0) sourceQuery.producer = String(selfNodeId.value)
    if (!consumerQuery.consumer && selfNodeId.value > 0) consumerQuery.consumer = String(selfNodeId.value)
  },
  { immediate: true }
)

onMounted(() => {
  void stream.loadDeliveries().catch((err) => console.warn(err))
})
</script>

<template>
  <section class="space-y-6">
    <div class="rounded-3xl border border-border/60 bg-card/90 p-6 shadow-sm">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">{{ t("Stream Control Plane") }}</p>
          <h2 class="mt-2 text-2xl font-semibold">{{ t("Sources, consumers, and delivery viewers") }}</h2>
          <p class="mt-2 max-w-3xl text-sm text-muted-foreground">
            {{ t("Use this page to query typed stream catalogs, create local endpoints, connect deliveries, and inspect runtime traffic.") }}
          </p>
        </div>
        <div class="grid gap-3 sm:grid-cols-3">
          <div class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3">
            <p class="text-[11px] font-semibold uppercase tracking-[0.25em] text-muted-foreground">{{ t("Self Node") }}</p>
            <p class="mt-1 text-lg font-semibold">{{ selfNodeId || "-" }}</p>
          </div>
          <div class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3">
            <p class="text-[11px] font-semibold uppercase tracking-[0.25em] text-muted-foreground">{{ t("Hub") }}</p>
            <p class="mt-1 text-lg font-semibold">{{ hubId || "-" }}</p>
          </div>
          <div class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3">
            <label class="text-[11px] font-semibold uppercase tracking-[0.25em] text-muted-foreground">{{ t("Control Target") }}</label>
            <input v-model="targetIdText" class="mt-2 h-10 w-full rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" :placeholder="t('Hub ID')" />
          </div>
        </div>
      </div>
    </div>

    <div class="grid gap-6 xl:grid-cols-[1fr_1fr_1.2fr]">
      <section class="space-y-6">
        <div class="rounded-3xl border border-border/60 bg-card/90 p-5 shadow-sm">
          <div class="flex items-center justify-between gap-3">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.25em] text-muted-foreground">{{ t("Sources") }}</p>
              <h3 class="mt-1 text-lg font-semibold">{{ t("Producer catalogs") }}</h3>
            </div>
            <Button variant="outline" @click="refreshSources">{{ t("Refresh") }}</Button>
          </div>
          <div class="mt-4 grid gap-3">
            <input v-model="sourceQuery.producer" class="h-10 rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" :placeholder="t('Producer Node ID')" />
            <div class="grid gap-3 sm:grid-cols-2">
              <select v-model="sourceQuery.kind" class="h-10 rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2">
                <option value="">{{ t("All kinds") }}</option>
                <option v-for="kind in streamKinds" :key="kind" :value="kind">{{ kind }}</option>
              </select>
              <input v-model="sourceQuery.tag" class="h-10 rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" :placeholder="t('Tag filter')" />
            </div>
          </div>
          <div class="mt-4 space-y-3">
            <button
              v-for="source in sources"
              :key="source.sourceId"
              type="button"
              class="w-full rounded-2xl border p-4 text-left transition"
              :class="stream.state.selectedSourceId === source.sourceId ? 'border-primary/50 bg-primary/10 shadow-sm' : 'border-border/60 bg-background/70 hover:border-border'"
              @click="stream.selectSource(source.sourceId)"
            >
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <div class="flex flex-wrap items-center gap-2">
                    <h4 class="text-sm font-semibold">{{ source.name || source.sourceId }}</h4>
                    <Badge :class="kindToneClass(source.kind)">{{ source.kind }}</Badge>
                    <Badge variant="secondary">{{ t("Producer {id}", { id: source.producer }) }}</Badge>
                  </div>
                  <p class="mt-2 text-xs text-muted-foreground">{{ source.sourceId }}</p>
                  <p class="mt-1 text-xs text-muted-foreground">{{ [source.contentType, source.mode, source.unitMode].filter(Boolean).join(" · ") }}</p>
                </div>
                <Button v-if="source.producer === selfNodeId" variant="ghost" size="sm" @click.stop="removeSource(source.sourceId)">{{ t("Withdraw") }}</Button>
              </div>
            </button>
            <div v-if="!sources.length" class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-6 text-sm text-muted-foreground">
              {{ t("No sources loaded yet.") }}
            </div>
          </div>
        </div>

        <div class="rounded-3xl border border-border/60 bg-card/90 p-5 shadow-sm">
          <p class="text-xs font-semibold uppercase tracking-[0.25em] text-muted-foreground">{{ t("Local Source") }}</p>
          <div class="mt-4 grid gap-3">
            <input v-model="sourceDraft.name" class="h-10 rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" :placeholder="t('Display name')" />
            <div class="grid gap-3 sm:grid-cols-2">
              <select v-model="sourceDraft.kind" class="h-10 rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2">
                <option v-for="kind in streamKinds" :key="kind" :value="kind">{{ kind }}</option>
              </select>
              <input v-model="sourceDraft.contentType" class="h-10 rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" :placeholder="t('Content type')" />
            </div>
            <div class="grid gap-3 sm:grid-cols-2">
              <select v-model="sourceDraft.mode" class="h-10 rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"><option value="live">live</option><option value="bounded">bounded</option></select>
              <select v-model="sourceDraft.unitMode" class="h-10 rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"><option value="frame">frame</option><option value="chunk">chunk</option></select>
            </div>
            <input v-model="sourceDraft.tagsText" class="h-10 rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" :placeholder="t('Tags, comma separated')" />
            <textarea v-model="sourceDraft.metadataText" rows="4" class="rounded-2xl border border-input bg-background px-3 py-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" :placeholder="t('Optional metadata JSON')" />
            <Button @click="submitSource">{{ t("Announce Source") }}</Button>
          </div>
        </div>
      </section>

      <section class="space-y-6">
        <div class="rounded-3xl border border-border/60 bg-card/90 p-5 shadow-sm">
          <div class="flex items-center justify-between gap-3">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.25em] text-muted-foreground">{{ t("Consumers") }}</p>
              <h3 class="mt-1 text-lg font-semibold">{{ t("Consumer endpoints") }}</h3>
            </div>
            <Button variant="outline" @click="refreshConsumers">{{ t("Refresh") }}</Button>
          </div>
          <div class="mt-4 grid gap-3">
            <input v-model="consumerQuery.consumer" class="h-10 rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" :placeholder="t('Consumer Node ID')" />
            <div class="grid gap-3 sm:grid-cols-2">
              <select v-model="consumerQuery.kind" class="h-10 rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2">
                <option value="">{{ t("All kinds") }}</option>
                <option v-for="kind in streamKinds" :key="kind" :value="kind">{{ kind }}</option>
              </select>
              <input v-model="consumerQuery.tag" class="h-10 rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" :placeholder="t('Tag filter')" />
            </div>
          </div>
          <div class="mt-4 space-y-3">
            <button
              v-for="consumer in consumers"
              :key="consumer.consumerId"
              type="button"
              class="w-full rounded-2xl border p-4 text-left transition"
              :class="stream.state.selectedConsumerId === consumer.consumerId ? 'border-primary/50 bg-primary/10 shadow-sm' : 'border-border/60 bg-background/70 hover:border-border'"
              @click="stream.selectConsumer(consumer.consumerId)"
            >
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <div class="flex flex-wrap items-center gap-2">
                    <h4 class="text-sm font-semibold">{{ consumer.name || consumer.consumerId }}</h4>
                    <Badge :class="kindToneClass(consumer.kind)">{{ consumer.kind }}</Badge>
                    <Badge variant="secondary">{{ t("Consumer {id}", { id: consumer.consumer }) }}</Badge>
                  </div>
                  <p class="mt-2 text-xs text-muted-foreground">{{ consumer.consumerId }}</p>
                  <p class="mt-1 text-xs text-muted-foreground">{{ consumer.contentType }}</p>
                </div>
                <Button v-if="consumer.consumer === selfNodeId" variant="ghost" size="sm" @click.stop="removeConsumer(consumer.consumerId)">{{ t("Withdraw") }}</Button>
              </div>
            </button>
            <div v-if="!consumers.length" class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-6 text-sm text-muted-foreground">
              {{ t("No consumers loaded yet.") }}
            </div>
          </div>
        </div>

        <div class="rounded-3xl border border-border/60 bg-card/90 p-5 shadow-sm">
          <p class="text-xs font-semibold uppercase tracking-[0.25em] text-muted-foreground">{{ t("Local Consumer") }}</p>
          <div class="mt-4 grid gap-3">
            <input v-model="consumerDraft.name" class="h-10 rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" :placeholder="t('Display name')" />
            <div class="grid gap-3 sm:grid-cols-2">
              <select v-model="consumerDraft.kind" class="h-10 rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2">
                <option v-for="kind in streamKinds" :key="kind" :value="kind">{{ kind }}</option>
              </select>
              <input v-model="consumerDraft.contentType" class="h-10 rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" :placeholder="t('Content type')" />
            </div>
            <input v-model="consumerDraft.tagsText" class="h-10 rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" :placeholder="t('Tags, comma separated')" />
            <textarea v-model="consumerDraft.metadataText" rows="4" class="rounded-2xl border border-input bg-background px-3 py-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" :placeholder="t('Optional metadata JSON')" />
            <Button @click="submitConsumer">{{ t("Announce Consumer") }}</Button>
          </div>
        </div>
      </section>

      <section class="space-y-6">
        <div class="rounded-3xl border border-border/60 bg-card/90 p-5 shadow-sm">
          <p class="text-xs font-semibold uppercase tracking-[0.25em] text-muted-foreground">{{ t("Actions") }}</p>
          <div class="mt-4 rounded-2xl border border-border/60 bg-background/60 p-4">
            <p class="text-sm font-medium">{{ selectedSource ? selectedSource.name || selectedSource.sourceId : t("No source selected") }}</p>
            <p class="mt-1 text-sm font-medium">{{ selectedConsumer ? selectedConsumer.name || selectedConsumer.consumerId : t("No consumer selected") }}</p>
            <div class="mt-4 flex flex-wrap gap-2">
              <Button :disabled="!canConnect" @click="connectSelected">{{ t("Connect") }}</Button>
              <Button variant="outline" :disabled="!canSubscribe" @click="subscribeSelected">{{ t("Subscribe") }}</Button>
            </div>
            <p v-if="selectedSource && selectedConsumer && !canConnect" class="mt-3 text-xs text-rose-600">{{ t("Source kind and consumer kind must match.") }}</p>
            <p v-else-if="selectedConsumer && !canSubscribe && canConnect" class="mt-3 text-xs text-muted-foreground">{{ t("Subscribe is for local consumer endpoints; use Connect for remote consumer nodes.") }}</p>
          </div>
        </div>

        <div class="rounded-3xl border border-border/60 bg-card/90 p-5 shadow-sm">
          <div class="flex items-center justify-between gap-3">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.25em] text-muted-foreground">{{ t("Deliveries") }}</p>
              <h3 class="mt-1 text-lg font-semibold">{{ t("Known runtime sessions") }}</h3>
            </div>
            <Badge variant="secondary">{{ deliveries.length }}</Badge>
          </div>
          <div class="mt-4 flex flex-wrap gap-2">
            <Button variant="outline" :disabled="!selectedDelivery" @click="disconnectSelected">{{ t("Disconnect") }}</Button>
            <Button variant="outline" :disabled="!selectedDelivery" @click="unsubscribeSelected">{{ t("Unsubscribe") }}</Button>
            <Button variant="ghost" :disabled="!selectedDelivery" @click="signalSelected('pause')">{{ t("Pause") }}</Button>
            <Button variant="ghost" :disabled="!selectedDelivery" @click="signalSelected('resume')">{{ t("Resume") }}</Button>
            <Button variant="ghost" :disabled="!selectedDelivery" @click="signalSelected('keyframe_request')">{{ t("Keyframe") }}</Button>
          </div>
          <div class="mt-4 space-y-3">
            <button
              v-for="delivery in deliveries"
              :key="delivery.deliveryId"
              type="button"
              class="w-full rounded-2xl border p-4 text-left transition"
              :class="stream.state.selectedDeliveryId === delivery.deliveryId ? 'border-primary/50 bg-primary/10 shadow-sm' : 'border-border/60 bg-background/70 hover:border-border'"
              @click="stream.selectDelivery(delivery.deliveryId)"
            >
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <div class="flex flex-wrap items-center gap-2">
                    <h4 class="text-sm font-semibold">{{ delivery.deliveryId }}</h4>
                    <Badge :class="kindToneClass(delivery.kind)">{{ delivery.kind }}</Badge>
                    <Badge variant="secondary">{{ delivery.state || t("observed") }}</Badge>
                  </div>
                  <p class="mt-2 text-xs text-muted-foreground">{{ t("Frames {frames} · Bytes {bytes}", { frames: delivery.framesIn, bytes: delivery.bytesIn }) }}</p>
                  <p v-if="delivery.lastError" class="mt-1 text-xs text-rose-600">{{ delivery.lastError }}</p>
                </div>
                <p class="text-xs text-muted-foreground">{{ formatTimestamp(delivery.updatedAt) }}</p>
              </div>
            </button>
            <div v-if="!deliveries.length" class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-6 text-sm text-muted-foreground">{{ t("No known deliveries yet.") }}</div>
          </div>
        </div>

        <div class="rounded-3xl border border-border/60 bg-card/90 p-5 shadow-sm">
          <div class="flex items-center justify-between gap-3">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.25em] text-muted-foreground">{{ t("Viewer") }}</p>
              <h3 class="mt-1 text-lg font-semibold">{{ selectedDelivery ? selectedDelivery.kind : t("No delivery selected") }}</h3>
            </div>
            <Badge v-if="selectedDelivery" :class="kindToneClass(selectedDelivery.kind)">{{ selectedDelivery.kind }}</Badge>
          </div>
          <div v-if="selectedDelivery" class="mt-4 space-y-4">
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3"><p class="text-[11px] font-semibold uppercase tracking-[0.25em] text-muted-foreground">{{ t("Last position") }}</p><p class="mt-1 text-lg font-semibold">{{ selectedDelivery.lastPosition }}</p></div>
              <div class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3"><p class="text-[11px] font-semibold uppercase tracking-[0.25em] text-muted-foreground">{{ t("Frames / Bytes") }}</p><p class="mt-1 text-lg font-semibold">{{ selectedDelivery.framesIn }} / {{ selectedDelivery.bytesIn }}</p></div>
            </div>
            <div v-if="selectedDelivery.kind === 'text'" class="rounded-2xl border border-border/60 bg-slate-950 px-4 py-4 text-sm text-emerald-100 shadow-inner">
              <div class="max-h-[360px] space-y-3 overflow-y-auto pr-1">
                <div v-for="frame in selectedTextFrames" :key="`${frame.deliveryId}:${frame.position}:${frame.updatedAt}`" class="rounded-xl border border-emerald-200/10 bg-white/5 px-3 py-3">
                  <div class="flex items-center justify-between gap-3 text-[11px] text-emerald-200/70"><span>{{ frame.position }}</span><span>{{ formatTimestamp(frame.updatedAt) }}</span></div>
                  <pre class="mt-2 whitespace-pre-wrap break-words font-mono text-[13px] leading-6">{{ frame.text }}</pre>
                </div>
                <div v-if="!selectedTextFrames.length" class="rounded-xl border border-dashed border-emerald-200/15 px-3 py-6 text-center text-emerald-200/60">{{ t("Waiting for text frames...") }}</div>
              </div>
            </div>
            <div v-else class="rounded-2xl border border-border/60 bg-background/70 p-4">
              <p class="text-sm text-muted-foreground">{{ t("This viewer currently exposes delivery health and stats. Rendering can be added later without changing the control plane.") }}</p>
              <div v-if="selectedStats" class="mt-4 rounded-2xl border border-border/60 bg-card/80 px-4 py-3 text-sm text-muted-foreground">
                {{ t("Frames {frames} · Bytes {bytes} · ACK {ack} · Flags {flags}", { frames: selectedStats.framesIn, bytes: selectedStats.bytesIn, ack: selectedStats.lastAckPos, flags: selectedStats.lastFlags }) }}
              </div>
            </div>
          </div>
          <div v-else class="mt-4 rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-10 text-center text-sm text-muted-foreground">{{ t("Select a delivery to inspect its runtime state.") }}</div>
        </div>
      </section>
    </div>
  </section>
</template>
