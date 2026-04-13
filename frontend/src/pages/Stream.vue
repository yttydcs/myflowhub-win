<script setup lang="ts">
// Context: implements the Stream page for configuring sources, consumers, and active deliveries.
import { computed, reactive, ref, watch } from "vue";
import {
  Cable,
  Pause,
  Play,
  Plus,
  RefreshCw,
  ScanSearch,
  Target,
} from "lucide-vue-next";
import CardHeader from "@/components/CardHeader.vue";
import PageHero from "@/components/PageHero.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Overlay } from "@/components/ui/overlay";
import { t } from "@/i18n";
import { openAuxWindow as openSharedAuxWindow } from "@/lib/auxWindow";
import {
  streamKinds,
  type StreamConsumer,
  type StreamConsumerDraft,
  type StreamDelivery,
  type StreamSource,
  type StreamSourceDraft,
  type StreamTab,
  useStreamStore,
} from "@/stores/stream";
import { useSessionStore } from "@/stores/session";
import { useToastStore } from "@/stores/toast";

const stream = useStreamStore();
const sessionStore = useSessionStore();
const toast = useToastStore();

const sourceNameInputId = "stream-source-name";
const consumerNameInputId = "stream-consumer-name";

const inputClass =
  "h-10 w-full rounded-xl border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2";
const textAreaClass =
  "min-h-[132px] w-full rounded-2xl border border-input bg-background px-3 py-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2";

const tabs: { id: StreamTab; label: string }[] = [
  { id: "source", label: "Source" },
  { id: "consumer", label: "Consumer" },
  { id: "control", label: "Control" },
];

const defaultSourceDraft = (): StreamSourceDraft => ({
  sourceId: "",
  name: "",
  kind: "text",
  contentType: "text/plain",
  mode: "live",
  unitMode: "frame",
  tagsText: "",
  metadataText: "",
  inputKind: "",
  filePath: "",
});

const defaultConsumerDraft = (): StreamConsumerDraft => ({
  consumerId: "",
  name: "",
  kind: "text",
  contentType: "text/plain",
  tagsText: "",
  metadataText: "",
});

const controlSourceQuery = reactive({ producer: "", kind: "", tag: "" });
const controlConsumerQuery = reactive({ consumer: "", kind: "", tag: "" });
const subscribeQuery = reactive({
  producer: "",
  kind: "",
  tag: "",
  selectedSourceId: "",
});
const sourceDraft = reactive<StreamSourceDraft>(defaultSourceDraft());
const consumerDraft = reactive<StreamConsumerDraft>(defaultConsumerDraft());
const subscribeDialog = reactive({
  open: false,
  consumerId: "",
});

const sourceDialogOpen = ref(false);
const consumerDialogOpen = ref(false);
const controlDialogOpen = ref(false);

const selfNodeId = computed(() => Number(sessionStore.auth.nodeId || 0));
const hubId = computed(() => Number(sessionStore.auth.hubId || 0));
const activeTab = computed({
  get: () => stream.state.activeTab,
  set: (value: StreamTab) => stream.setActiveTab(value),
});
const targetIdText = computed({
  get: () => stream.state.targetId,
  set: (value: string) => stream.setTargetId(value),
});

const localSources = computed(() => stream.state.localSources);
const localConsumers = computed(() => stream.state.localConsumers);
const catalogSources = computed(() => stream.state.sources);
const catalogConsumers = computed(() => stream.state.consumers);
const deliveries = computed(() => stream.state.deliveries);
const selectedControlSource = computed(() =>
  stream.sourceById(stream.state.selectedSourceId, "catalog"),
);
const selectedControlConsumer = computed(() =>
  stream.consumerById(stream.state.selectedConsumerId, "catalog"),
);
const selectedDelivery = computed(
  () =>
    deliveries.value.find(
      (item) => item.deliveryId === stream.state.selectedDeliveryId,
    ) ?? null,
);
const resolvedTargetId = computed(
  () => targetIdText.value || (hubId.value ? String(hubId.value) : ""),
);
const subscribeConsumer = computed(() =>
  stream.consumerById(subscribeDialog.consumerId, "local"),
);
const subscribeSelectedSource = computed(() =>
  stream.sourceById(subscribeQuery.selectedSourceId, "catalog"),
);
const canConnect = computed(() => {
  if (!selectedControlSource.value || !selectedControlConsumer.value)
    return false;
  return (
    selectedControlSource.value.kind === selectedControlConsumer.value.kind
  );
});
const canSubscribe = computed(() => {
  if (
    !selectedControlSource.value ||
    !selectedControlConsumer.value ||
    !canConnect.value
  )
    return false;
  return selectedControlConsumer.value.consumer === selfNodeId.value;
});

const heroDescription = computed(() => {
  if (activeTab.value === "source") {
    return t(
      "Manage your saved local sources in a compact list and open a dedicated input window only when you need to send content.",
    );
  }
  if (activeTab.value === "consumer") {
    return t(
      "Keep local consumers in a simple list, review current bindings, and subscribe through a dedicated dialog only when you need to change them.",
    );
  }
  return t(
    "Choose control pairs through a focused dialog, then inspect runtime deliveries and open output windows from the control tab.",
  );
});

const tabButtonClass = (tab: StreamTab) => [
  "rounded-full px-4 py-2 text-sm font-semibold transition",
  activeTab.value === tab
    ? "bg-primary text-primary-foreground shadow-sm"
    : "text-muted-foreground hover:bg-muted/70",
];

const formatTimestamp = (value: string) => {
  const dt = new Date(String(value ?? ""));
  return Number.isNaN(dt.getTime()) ? "" : dt.toLocaleString();
};

const kindToneClass = (kind: string) => {
  switch (kind) {
    case "music":
      return "bg-amber-500/15 text-amber-700";
    case "video":
      return "bg-rose-500/15 text-rose-700";
    case "text":
      return "bg-emerald-500/15 text-emerald-700";
    default:
      return "bg-slate-500/15 text-slate-700";
  }
};

const isActiveDelivery = (delivery: Pick<StreamDelivery, "state">) =>
  String(delivery.state ?? "")
    .trim()
    .toLowerCase() !== "closed";
const sourceBindings = (sourceId: string) =>
  stream.deliveriesForSource(sourceId).filter(isActiveDelivery);
const consumerBindings = (consumerId: string) =>
  stream.deliveriesForConsumer(consumerId).filter(isActiveDelivery);

const compactList = (values: string[]) => {
  const normalized = Array.from(
    new Set(values.map((value) => String(value ?? "").trim()).filter(Boolean)),
  );
  if (!normalized.length) return "";
  if (normalized.length <= 2) return normalized.join(" · ");
  return `${normalized.slice(0, 2).join(" · ")} +${normalized.length - 2}`;
};

const sourceBindingSummary = (source: StreamSource) => {
  const bindings = sourceBindings(source.sourceId);
  if (!bindings.length) return t("No active bindings");
  return t("{count} active bindings", { count: bindings.length });
};

const consumerBindingSummary = (consumer: StreamConsumer) => {
  const bindings = consumerBindings(consumer.consumerId);
  if (!bindings.length) return t("No current source bindings");
  const summary = compactList(bindings.map((binding) => binding.sourceId));
  return summary || t("{count} active bindings", { count: bindings.length });
};

const deliverySummary = (delivery: StreamDelivery) => {
  const sourceLabel = delivery.sourceId || t("Unknown source");
  const consumerLabel =
    delivery.consumerId || t("Consumer {id}", { id: delivery.consumer || "-" });
  return `${sourceLabel} -> ${consumerLabel}`;
};

const resetSourceDraft = () => Object.assign(sourceDraft, defaultSourceDraft());
const resetConsumerDraft = () =>
  Object.assign(consumerDraft, defaultConsumerDraft());
const closeSourceDialog = () => {
  sourceDialogOpen.value = false;
  resetSourceDraft();
};
const closeConsumerDialog = () => {
  consumerDialogOpen.value = false;
  resetConsumerDraft();
};
const closeSubscribeDialog = () => {
  subscribeDialog.open = false;
  subscribeDialog.consumerId = "";
  subscribeQuery.producer = "";
  subscribeQuery.kind = "";
  subscribeQuery.tag = "";
  subscribeQuery.selectedSourceId = "";
};
const closeControlDialog = () => {
  controlDialogOpen.value = false;
};

const withToast = async (
  action: () => Promise<unknown>,
  ok: string,
  fail: string,
) => {
  try {
    await action();
    toast.success(t(ok));
  } catch (err) {
    console.warn(err);
    toast.errorOf(err, t(fail));
  }
};

const openStreamAuxWindow = async (
  routePath: string,
  name: string,
  size: string,
  blockedMessage: string,
) => {
  const result = await openSharedAuxWindow({
    routePath,
    name,
    size,
  });
  if (result === "blocked") {
    toast.warn(t(blockedMessage));
  }
};

const refreshControlSourcesRaw = () =>
  stream.listSources(
    controlSourceQuery.producer,
    controlSourceQuery.kind,
    controlSourceQuery.tag,
  );
const refreshControlConsumersRaw = () =>
  stream.listConsumers(
    controlConsumerQuery.consumer,
    controlConsumerQuery.kind,
    controlConsumerQuery.tag,
  );
const refreshControlSources = () =>
  withToast(
    refreshControlSourcesRaw,
    "Sources refreshed.",
    "Failed to query sources.",
  );
const refreshControlConsumers = () =>
  withToast(
    refreshControlConsumersRaw,
    "Consumers refreshed.",
    "Failed to query consumers.",
  );
const refreshDeliveries = () =>
  withToast(
    () => stream.loadDeliveries(),
    "Runtime deliveries refreshed.",
    "Failed to load deliveries.",
  );
const refreshControlPlane = () =>
  withToast(
    async () => {
      await Promise.all([
        refreshControlSourcesRaw(),
        refreshControlConsumersRaw(),
        stream.loadDeliveries(),
      ]);
    },
    "Stream control plane refreshed.",
    "Failed to refresh Stream control plane.",
  );
const refreshLocalLists = () =>
  withToast(
    async () => {
      if (!selfNodeId.value) return;
      await Promise.all([
        stream.listSources(String(selfNodeId.value), "", "", "local"),
        stream.listConsumers(String(selfNodeId.value), "", "", "local"),
      ]);
    },
    "Local stream lists refreshed.",
    "Failed to refresh local stream lists.",
  );
const refreshSubscribeSourcesRaw = () =>
  stream.listSources(
    subscribeQuery.producer,
    subscribeQuery.kind,
    subscribeQuery.tag,
  );
const refreshSubscribeSources = () =>
  withToast(
    refreshSubscribeSourcesRaw,
    "Subscription sources refreshed.",
    "Failed to load subscription sources.",
  );

const submitSource = () =>
  withToast(
    async () => {
      await stream.announceSource(sourceDraft);
      closeSourceDialog();
    },
    "Local source created.",
    "Failed to create local source.",
  );

const submitConsumer = () =>
  withToast(
    async () => {
      await stream.announceConsumer(consumerDraft);
      closeConsumerDialog();
    },
    "Local consumer created.",
    "Failed to create local consumer.",
  );

const openSourceDialog = () => {
  resetSourceDraft();
  sourceDialogOpen.value = true;
};

const openConsumerDialog = () => {
  resetConsumerDraft();
  consumerDialogOpen.value = true;
};

const chooseSourceMediaFile = async () => {
  try {
    const file = await stream.pickMediaFile();
    if (!file) return;
    if (file.kind !== "music" && file.kind !== "video") {
      throw new Error(
        t("Selected file is not a supported audio or video file."),
      );
    }
    sourceDraft.kind = file.kind;
    sourceDraft.contentType = file.contentType || sourceDraft.contentType;
    sourceDraft.mode = "bounded";
    sourceDraft.unitMode = "chunk";
    sourceDraft.inputKind = "file";
    sourceDraft.filePath = file.path;
    if (!sourceDraft.name && file.name) sourceDraft.name = file.name;
  } catch (err) {
    console.warn(err);
    toast.errorOf(err, t("Failed to select a media file."));
  }
};

const clearSourceMediaFile = () => {
  if (sourceDraft.kind === "text") {
    sourceDraft.inputKind = "";
  } else {
    sourceDraft.inputKind = sourceDraft.kind === "video" ? "file" : "file";
  }
  sourceDraft.filePath = "";
};

const openSourceWindow = async (sourceId: string) => {
  const normalized = String(sourceId ?? "").trim();
  if (!normalized) return;
  await openStreamAuxWindow(
    `#/stream-source-window?sourceId=${encodeURIComponent(normalized)}`,
    `stream_source_${normalized}_${Date.now()}`,
    "width=1100,height=780",
    "Stream input window was blocked by browser popup policy.",
  );
};

const openDeliveryWindow = async (deliveryId: string) => {
  const normalized = String(deliveryId ?? "").trim();
  if (!normalized) return;
  stream.selectDelivery(normalized);
  await openStreamAuxWindow(
    `#/stream-delivery-window?deliveryId=${encodeURIComponent(normalized)}`,
    `stream_delivery_${normalized}_${Date.now()}`,
    "width=1180,height=820",
    "Stream output window was blocked by browser popup policy.",
  );
};

const openSubscribeDialog = async (consumerId: string) => {
  const consumer = stream.consumerById(consumerId, "local");
  if (!consumer) return;
  subscribeDialog.consumerId = consumerId;
  subscribeQuery.producer = selfNodeId.value ? String(selfNodeId.value) : "";
  subscribeQuery.kind = consumer.kind;
  subscribeQuery.tag = "";
  subscribeQuery.selectedSourceId = "";
  subscribeDialog.open = true;
  if (subscribeQuery.producer) {
    try {
      await refreshSubscribeSourcesRaw();
    } catch (err) {
      console.warn(err);
    }
  }
};

const openControlDialog = async () => {
  if (!controlSourceQuery.producer && selfNodeId.value > 0)
    controlSourceQuery.producer = String(selfNodeId.value);
  if (!controlConsumerQuery.consumer && selfNodeId.value > 0)
    controlConsumerQuery.consumer = String(selfNodeId.value);
  controlDialogOpen.value = true;
  if (catalogSources.value.length && catalogConsumers.value.length) return;
  try {
    await Promise.all([
      refreshControlSourcesRaw(),
      refreshControlConsumersRaw(),
    ]);
  } catch (err) {
    console.warn(err);
  }
};

const subscribeFromDialog = () =>
  withToast(
    async () => {
      if (!subscribeSelectedSource.value || !subscribeDialog.consumerId) {
        throw new Error(t("Select a source before subscribing."));
      }
      await stream.subscribe({
        producer: subscribeSelectedSource.value.producer,
        sourceId: subscribeSelectedSource.value.sourceId,
        consumerId: subscribeDialog.consumerId,
      });
      closeSubscribeDialog();
    },
    "Subscribed.",
    "Failed to subscribe.",
  );

const connectSelected = () =>
  selectedControlSource.value && selectedControlConsumer.value
    ? withToast(
        async () => {
          await stream.connect({
            producer: selectedControlSource.value!.producer,
            sourceId: selectedControlSource.value!.sourceId,
            consumer: selectedControlConsumer.value!.consumer,
            consumerId: selectedControlConsumer.value!.consumerId,
          });
          closeControlDialog();
        },
        "Delivery connected.",
        "Failed to connect delivery.",
      )
    : Promise.resolve();

const subscribeSelected = () =>
  selectedControlSource.value && selectedControlConsumer.value
    ? withToast(
        async () => {
          await stream.subscribe({
            producer: selectedControlSource.value!.producer,
            sourceId: selectedControlSource.value!.sourceId,
            consumerId: selectedControlConsumer.value!.consumerId,
          });
          closeControlDialog();
        },
        "Subscribed.",
        "Failed to subscribe.",
      )
    : Promise.resolve();

const disconnectSelected = () =>
  selectedDelivery.value
    ? withToast(
        () => stream.disconnect(selectedDelivery.value!.deliveryId),
        "Delivery disconnected.",
        "Failed to disconnect delivery.",
      )
    : Promise.resolve();

const unsubscribeSelected = () =>
  selectedDelivery.value
    ? withToast(
        () => stream.unsubscribe(selectedDelivery.value!.deliveryId),
        "Delivery unsubscribed.",
        "Failed to unsubscribe delivery.",
      )
    : Promise.resolve();

const signalSelected = (op: string) =>
  selectedDelivery.value
    ? withToast(
        () => stream.signal(selectedDelivery.value!.deliveryId, op),
        "Signal sent.",
        "Failed to send signal.",
      )
    : Promise.resolve();

const removeSource = (sourceId: string) =>
  withToast(
    () => stream.withdrawSource(sourceId),
    "Source removed.",
    "Failed to remove source.",
  );

const removeConsumer = (consumerId: string) =>
  withToast(
    async () => {
      await stream.withdrawConsumer(consumerId);
      if (subscribeDialog.consumerId === consumerId) closeSubscribeDialog();
    },
    "Consumer removed.",
    "Failed to remove consumer.",
  );

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  () => {
    stream.setIdentity(selfNodeId.value, hubId.value);
    if (!targetIdText.value && hubId.value > 0)
      targetIdText.value = String(hubId.value);
    if (!controlSourceQuery.producer && selfNodeId.value > 0)
      controlSourceQuery.producer = String(selfNodeId.value);
    if (!controlConsumerQuery.consumer && selfNodeId.value > 0)
      controlConsumerQuery.consumer = String(selfNodeId.value);
    void (async () => {
      try {
        await Promise.all([stream.loadPrefs(), stream.loadDeliveries()]);
        if (selfNodeId.value > 0 && hubId.value > 0) {
          await Promise.all([
            refreshControlSourcesRaw(),
            refreshControlConsumersRaw(),
          ]);
        }
      } catch (err) {
        console.warn(err);
      }
    })();
  },
  { immediate: true },
);

watch(
  () => sourceDraft.kind,
  (kind) => {
    if (kind === "text") {
      sourceDraft.contentType = "text/plain";
      sourceDraft.mode = "live";
      sourceDraft.unitMode = "frame";
      sourceDraft.inputKind = "";
      sourceDraft.filePath = "";
      return;
    }
    if (sourceDraft.mode !== "bounded") sourceDraft.mode = "bounded";
    if (sourceDraft.unitMode !== "chunk") sourceDraft.unitMode = "chunk";
    if (kind === "video") {
      if (
        sourceDraft.inputKind !== "desktop" &&
        sourceDraft.inputKind !== "file"
      )
        sourceDraft.inputKind = "file";
      return;
    }
    if (sourceDraft.inputKind === "desktop" || !sourceDraft.inputKind)
      sourceDraft.inputKind = "file";
  },
);

watch(
  () => sourceDraft.inputKind,
  (inputKind) => {
    if (inputKind !== "desktop") return;
    sourceDraft.filePath = "";
    sourceDraft.kind = "video";
    sourceDraft.mode = "bounded";
    sourceDraft.unitMode = "chunk";
    if (
      !String(sourceDraft.contentType ?? "").trim() ||
      !String(sourceDraft.contentType).startsWith("video/")
    ) {
      sourceDraft.contentType = "video/webm";
    }
  },
);
</script>

<template>
  <section class="space-y-6" data-stream-page>
    <PageHero :description="heroDescription">
      <template #actions>
        <Badge variant="secondary">{{
          t("Self {id}", { id: selfNodeId || "-" })
        }}</Badge>
        <Badge variant="secondary">{{
          t("Hub {id}", { id: hubId || "-" })
        }}</Badge>
        <Badge variant="secondary">{{
          t("Target {id}", { id: resolvedTargetId || "-" })
        }}</Badge>
        <div
          class="inline-flex rounded-full border border-border/70 bg-background/80 p-1"
        >
          <button
            v-for="tab in tabs"
            :key="tab.id"
            type="button"
            :data-stream-tab="tab.id"
            :class="tabButtonClass(tab.id)"
            :aria-pressed="activeTab === tab.id"
            @click="activeTab = tab.id"
          >
            {{ t(tab.label) }}
          </button>
        </div>
      </template>
    </PageHero>

    <section v-if="activeTab === 'source'" class="space-y-6">
      <section
        class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm"
      >
        <CardHeader
          :title="t('Local Sources')"
          :description="
            t(
              'Keep local sources persistent and focused. Add or remove sources here, then open a dedicated input window only when you need to send content.',
            )
          "
          title-class="text-lg"
        >
          <template #actions>
            <div class="flex flex-wrap gap-2">
              <Button variant="outline" size="sm" @click="refreshLocalLists">
                <RefreshCw class="mr-2 h-4 w-4" />
                {{ t("Refresh") }}
              </Button>
              <Button
                size="sm"
                data-stream-open-source
                @click="openSourceDialog"
              >
                <Plus class="mr-2 h-4 w-4" />
                {{ t("New Source") }}
              </Button>
            </div>
          </template>
        </CardHeader>

        <div class="mt-4 space-y-2">
          <article
            v-for="source in localSources"
            :key="source.sourceId"
            data-stream-local-source-row
            class="flex flex-col gap-3 rounded-2xl border border-border/60 bg-background/70 p-4 text-card-foreground md:flex-row md:items-center md:justify-between"
          >
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <p class="truncate text-sm font-semibold">
                  {{ source.name || source.sourceId }}
                </p>
                <Badge :class="kindToneClass(source.kind)">{{
                  source.kind
                }}</Badge>
              </div>
              <p class="mt-1 truncate text-xs text-muted-foreground">
                {{ source.sourceId }} · {{ sourceBindingSummary(source) }}
              </p>
            </div>
            <div class="flex flex-wrap gap-2">
              <Button
                variant="outline"
                size="sm"
                data-stream-open-source-window
                @click="openSourceWindow(source.sourceId)"
              >
                {{ t("Input Window") }}
              </Button>
              <Button
                variant="ghost"
                size="sm"
                @click="removeSource(source.sourceId)"
              >
                {{ t("Remove") }}
              </Button>
            </div>
          </article>

          <div
            v-if="!localSources.length"
            class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-10 text-center text-sm text-muted-foreground"
          >
            {{ t("No local sources yet. Use New Source to create one.") }}
          </div>
        </div>
      </section>
    </section>

    <section v-else-if="activeTab === 'consumer'" class="space-y-6">
      <section
        class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm"
      >
        <CardHeader
          :title="t('Local Consumers')"
          :description="
            t(
              'Store local consumers as a compact list. Review current bindings here and open a separate dialog only when you want to subscribe to a source.',
            )
          "
          title-class="text-lg"
        >
          <template #actions>
            <div class="flex flex-wrap gap-2">
              <Button variant="outline" size="sm" @click="refreshLocalLists">
                <RefreshCw class="mr-2 h-4 w-4" />
                {{ t("Refresh") }}
              </Button>
              <Button
                size="sm"
                data-stream-open-consumer
                @click="openConsumerDialog"
              >
                <Plus class="mr-2 h-4 w-4" />
                {{ t("New Consumer") }}
              </Button>
            </div>
          </template>
        </CardHeader>

        <div class="mt-4 space-y-2">
          <article
            v-for="consumer in localConsumers"
            :key="consumer.consumerId"
            data-stream-local-consumer-row
            class="flex flex-col gap-3 rounded-2xl border border-border/60 bg-background/70 p-4 text-card-foreground md:flex-row md:items-center md:justify-between"
          >
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <p class="truncate text-sm font-semibold">
                  {{ consumer.name || consumer.consumerId }}
                </p>
                <Badge :class="kindToneClass(consumer.kind)">{{
                  consumer.kind
                }}</Badge>
              </div>
              <p class="mt-1 truncate text-xs text-muted-foreground">
                {{ consumer.consumerId }} ·
                {{ consumerBindingSummary(consumer) }}
              </p>
            </div>
            <div class="flex flex-wrap gap-2">
              <Button
                variant="outline"
                size="sm"
                data-stream-open-subscribe
                @click="openSubscribeDialog(consumer.consumerId)"
              >
                {{ t("Subscribe") }}
              </Button>
              <Button
                variant="ghost"
                size="sm"
                @click="removeConsumer(consumer.consumerId)"
              >
                {{ t("Remove") }}
              </Button>
            </div>
          </article>

          <div
            v-if="!localConsumers.length"
            class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-10 text-center text-sm text-muted-foreground"
          >
            {{ t("No local consumers yet. Use New Consumer to create one.") }}
          </div>
        </div>
      </section>
    </section>

    <section
      v-else
      class="grid gap-6 xl:grid-cols-[minmax(0,0.92fr)_minmax(0,1.08fr)]"
    >
      <section
        class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm"
      >
        <CardHeader
          :title="t('Control Target')"
          :description="
            t(
              'Leave target aligned with Hub unless you are routing control requests elsewhere.',
            )
          "
          title-class="text-lg"
        >
          <template #actions>
            <Button variant="outline" size="sm" @click="refreshControlPlane">
              <RefreshCw class="mr-2 h-4 w-4" />
              {{ t("Refresh All") }}
            </Button>
          </template>
        </CardHeader>

        <div class="mt-4">
          <label
            class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
            >{{ t("Control Target") }}</label
          >
          <div
            class="mt-2 flex items-center gap-2 rounded-2xl border border-border/60 bg-background/70 px-3"
          >
            <Target class="h-4 w-4 text-muted-foreground" />
            <input
              v-model="targetIdText"
              class="h-10 flex-1 bg-transparent text-sm outline-none"
              :placeholder="t('Hub ID')"
            />
          </div>
        </div>

        <div
          class="mt-5 rounded-2xl border border-border/60 bg-background/70 p-4"
        >
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="flex items-center gap-2">
              <Cable class="h-4 w-4 text-muted-foreground" />
              <p class="text-sm font-semibold">{{ t("Selected Pair") }}</p>
            </div>
            <Button data-stream-open-control-picker @click="openControlDialog">
              {{
                t(
                  selectedControlSource || selectedControlConsumer
                    ? "Change Pair"
                    : "Select Pair",
                )
              }}
            </Button>
          </div>
          <p class="mt-2 text-xs text-muted-foreground">
            {{
              t(
                "Use a focused picker for remote source and consumer selection, instead of leaving both catalogs on the page.",
              )
            }}
          </p>

          <div class="mt-4 grid gap-3">
            <div class="rounded-2xl border border-border/60 bg-card/90 p-4">
              <p
                class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
              >
                {{ t("Selected Source") }}
              </p>
              <div v-if="selectedControlSource" class="mt-3 min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <p class="truncate text-sm font-semibold">
                    {{
                      selectedControlSource.name ||
                      selectedControlSource.sourceId
                    }}
                  </p>
                  <Badge :class="kindToneClass(selectedControlSource.kind)">{{
                    selectedControlSource.kind
                  }}</Badge>
                  <Badge variant="secondary">{{
                    t("Producer {id}", { id: selectedControlSource.producer })
                  }}</Badge>
                </div>
                <p class="mt-2 truncate text-xs text-muted-foreground">
                  {{ selectedControlSource.sourceId }}
                </p>
              </div>
              <p v-else class="mt-3 text-sm text-muted-foreground">
                {{ t("No source selected yet.") }}
              </p>
            </div>

            <div class="rounded-2xl border border-border/60 bg-card/90 p-4">
              <p
                class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
              >
                {{ t("Selected Consumer") }}
              </p>
              <div v-if="selectedControlConsumer" class="mt-3 min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <p class="truncate text-sm font-semibold">
                    {{
                      selectedControlConsumer.name ||
                      selectedControlConsumer.consumerId
                    }}
                  </p>
                  <Badge :class="kindToneClass(selectedControlConsumer.kind)">{{
                    selectedControlConsumer.kind
                  }}</Badge>
                  <Badge variant="secondary">{{
                    t("Consumer {id}", { id: selectedControlConsumer.consumer })
                  }}</Badge>
                </div>
                <p class="mt-2 truncate text-xs text-muted-foreground">
                  {{ selectedControlConsumer.consumerId }}
                </p>
              </div>
              <p v-else class="mt-3 text-sm text-muted-foreground">
                {{ t("No consumer selected yet.") }}
              </p>
            </div>
          </div>

          <p
            v-if="
              selectedControlSource && selectedControlConsumer && !canConnect
            "
            class="mt-4 text-xs text-rose-600"
          >
            {{ t("Source kind and consumer kind must match.") }}
          </p>
          <p
            v-else-if="
              selectedControlSource && selectedControlConsumer && !canSubscribe
            "
            class="mt-4 text-xs text-muted-foreground"
          >
            {{
              t(
                "Subscribe only works when the selected consumer belongs to this node.",
              )
            }}
          </p>
        </div>
      </section>

      <section
        class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm"
      >
        <CardHeader
          :title="t('Runtime Deliveries')"
          :description="
            t(
              'Inspect deliveries and send lightweight runtime signals from one place.',
            )
          "
          title-class="text-lg"
        >
          <template #actions>
            <Badge variant="secondary">{{
              t("Observed {count}", { count: deliveries.length })
            }}</Badge>
          </template>
        </CardHeader>

        <div class="mt-4 flex flex-wrap gap-2">
          <Button variant="outline" size="sm" @click="refreshDeliveries">{{
            t("Refresh")
          }}</Button>
          <Button
            variant="outline"
            :disabled="!selectedDelivery"
            @click="disconnectSelected"
            >{{ t("Disconnect") }}</Button
          >
          <Button
            variant="outline"
            :disabled="!selectedDelivery"
            @click="unsubscribeSelected"
            >{{ t("Unsubscribe") }}</Button
          >
          <Button
            variant="ghost"
            :disabled="!selectedDelivery"
            @click="signalSelected('pause')"
          >
            <Pause class="mr-2 h-4 w-4" />
            {{ t("Pause") }}
          </Button>
          <Button
            variant="ghost"
            :disabled="!selectedDelivery"
            @click="signalSelected('resume')"
          >
            <Play class="mr-2 h-4 w-4" />
            {{ t("Resume") }}
          </Button>
        </div>

        <div class="mt-4 max-h-[360px] space-y-2 overflow-y-auto pr-1">
          <article
            v-for="delivery in deliveries"
            :key="delivery.deliveryId"
            data-stream-delivery-row
            class="flex cursor-pointer flex-col gap-3 rounded-2xl border p-4 text-card-foreground transition md:flex-row md:items-center md:justify-between"
            :class="
              stream.state.selectedDeliveryId === delivery.deliveryId
                ? 'border-primary/50 bg-primary/10 shadow-sm'
                : 'border-border/60 bg-background/70 hover:border-border'
            "
            @click="stream.selectDelivery(delivery.deliveryId)"
          >
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <p class="truncate text-sm font-semibold">
                  {{ deliverySummary(delivery) }}
                </p>
                <Badge :class="kindToneClass(delivery.kind)">{{
                  delivery.kind
                }}</Badge>
                <Badge variant="secondary">{{
                  delivery.state || t("Observed")
                }}</Badge>
              </div>
              <p class="mt-1 truncate text-xs text-muted-foreground">
                {{ delivery.deliveryId }}
              </p>
              <p class="mt-1 text-xs text-muted-foreground">
                {{
                  t("Frames {frames} · Bytes {bytes}", {
                    frames: delivery.framesIn,
                    bytes: delivery.bytesIn,
                  })
                }}
              </p>
            </div>
            <div class="flex flex-wrap items-center gap-2 md:justify-end">
              <span class="text-xs text-muted-foreground">{{
                formatTimestamp(delivery.updatedAt)
              }}</span>
              <Button
                variant="outline"
                size="sm"
                data-stream-open-delivery-window
                @click.stop="openDeliveryWindow(delivery.deliveryId)"
              >
                {{ t("Output Window") }}
              </Button>
            </div>
          </article>
          <div
            v-if="!deliveries.length"
            class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-8 text-sm text-muted-foreground"
          >
            {{ t("No known deliveries yet.") }}
          </div>
        </div>
      </section>
    </section>

    <Overlay
      :open="sourceDialogOpen"
      closeOnBackdrop
      trapFocus
      :initial-focus-selector="`#${sourceNameInputId}`"
      @close="closeSourceDialog"
    >
      <div
        data-stream-source-dialog
        class="w-full max-w-2xl rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl"
      >
        <CardHeader
          :title="t('New Source')"
          :description="
            t(
              'Create a persistent local source without filling the main page with form fields.',
            )
          "
          title-class="text-lg"
        />
        <div class="mt-5 grid gap-4 md:grid-cols-2">
          <div class="md:col-span-2">
            <label
              :for="sourceNameInputId"
              class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
              >{{ t("Display name") }}</label
            >
            <input
              :id="sourceNameInputId"
              v-model="sourceDraft.name"
              :class="['mt-2', inputClass]"
              :placeholder="t('Display name')"
            />
          </div>
          <div>
            <label
              class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
              >{{ t("Kind") }}</label
            >
            <select
              v-model="sourceDraft.kind"
              data-stream-source-kind
              :class="['mt-2', inputClass]"
            >
              <option v-for="kind in streamKinds" :key="kind" :value="kind">
                {{ kind }}
              </option>
            </select>
          </div>
          <div>
            <label
              class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
              >{{ t("Content type") }}</label
            >
            <input
              v-model="sourceDraft.contentType"
              :class="['mt-2', inputClass]"
              :placeholder="t('Content type')"
            />
          </div>
          <div>
            <label
              class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
              >{{ t("Mode") }}</label
            >
            <select v-model="sourceDraft.mode" :class="['mt-2', inputClass]">
              <option value="live">live</option>
              <option value="bounded">bounded</option>
            </select>
          </div>
          <div>
            <label
              class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
              >{{ t("Unit mode") }}</label
            >
            <select
              v-model="sourceDraft.unitMode"
              :class="['mt-2', inputClass]"
            >
              <option value="frame">frame</option>
              <option value="chunk">chunk</option>
            </select>
          </div>
          <div v-if="sourceDraft.kind === 'video'" class="md:col-span-2">
            <label
              class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
              >{{ t("Input mode") }}</label
            >
            <select
              v-model="sourceDraft.inputKind"
              data-stream-source-input-mode
              :class="['mt-2', inputClass]"
            >
              <option value="file">{{ t("Local File") }}</option>
              <option value="desktop">{{ t("Desktop Capture") }}</option>
            </select>
          </div>
          <div
            v-if="
              sourceDraft.kind !== 'text' && sourceDraft.inputKind !== 'desktop'
            "
            data-stream-source-file-config
            class="md:col-span-2"
          >
            <label
              class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
              >{{ t("Media file") }}</label
            >
            <div class="mt-2 flex flex-wrap gap-2">
              <input
                :value="sourceDraft.filePath"
                :class="['min-w-0 flex-1', inputClass]"
                :placeholder="t('Select a local media file')"
                readonly
              />
              <Button
                type="button"
                variant="outline"
                @click="chooseSourceMediaFile"
                >{{ t("Choose File") }}</Button
              >
              <Button
                v-if="sourceDraft.filePath"
                type="button"
                variant="outline"
                @click="clearSourceMediaFile"
                >{{ t("Clear File") }}</Button
              >
            </div>
            <p class="mt-2 text-xs text-muted-foreground">
              {{ t("File-backed media input uses bounded/chunk delivery.") }}
            </p>
          </div>
          <div
            v-else-if="
              sourceDraft.kind === 'video' &&
              sourceDraft.inputKind === 'desktop'
            "
            data-stream-source-desktop-config
            class="md:col-span-2 rounded-2xl border border-border/60 bg-background/60 px-4 py-4 text-sm text-muted-foreground"
          >
            <p>
              {{
                t(
                  "Desktop capture starts from the Source Window with a user click and does not persist a previous screen selection.",
                )
              }}
            </p>
            <p class="mt-2 text-xs text-muted-foreground">
              {{
                t(
                  "This round captures desktop video only. System audio is not included.",
                )
              }}
            </p>
          </div>
          <div class="md:col-span-2">
            <label
              class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
              >{{ t("Tags") }}</label
            >
            <input
              v-model="sourceDraft.tagsText"
              :class="['mt-2', inputClass]"
              :placeholder="t('Tags, comma separated')"
            />
          </div>
          <div class="md:col-span-2">
            <label
              class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
              >{{ t("Metadata") }}</label
            >
            <textarea
              v-model="sourceDraft.metadataText"
              :class="['mt-2', textAreaClass]"
              :placeholder="t('Optional metadata JSON')"
            />
          </div>
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="closeSourceDialog">{{
            t("Cancel")
          }}</Button>
          <Button data-stream-submit-source @click="submitSource">{{
            t("Create Source")
          }}</Button>
        </div>
      </div>
    </Overlay>

    <Overlay
      :open="consumerDialogOpen"
      closeOnBackdrop
      trapFocus
      :initial-focus-selector="`#${consumerNameInputId}`"
      @close="closeConsumerDialog"
    >
      <div
        data-stream-consumer-dialog
        class="w-full max-w-2xl rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl"
      >
        <CardHeader
          :title="t('New Consumer')"
          :description="
            t(
              'Create a persistent local consumer endpoint without expanding the main list into a form page.',
            )
          "
          title-class="text-lg"
        />
        <div class="mt-5 grid gap-4 md:grid-cols-2">
          <div class="md:col-span-2">
            <label
              :for="consumerNameInputId"
              class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
              >{{ t("Display name") }}</label
            >
            <input
              :id="consumerNameInputId"
              v-model="consumerDraft.name"
              :class="['mt-2', inputClass]"
              :placeholder="t('Display name')"
            />
          </div>
          <div>
            <label
              class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
              >{{ t("Kind") }}</label
            >
            <select v-model="consumerDraft.kind" :class="['mt-2', inputClass]">
              <option v-for="kind in streamKinds" :key="kind" :value="kind">
                {{ kind }}
              </option>
            </select>
          </div>
          <div>
            <label
              class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
              >{{ t("Content type") }}</label
            >
            <input
              v-model="consumerDraft.contentType"
              :class="['mt-2', inputClass]"
              :placeholder="t('Content type')"
            />
          </div>
          <div class="md:col-span-2">
            <label
              class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
              >{{ t("Tags") }}</label
            >
            <input
              v-model="consumerDraft.tagsText"
              :class="['mt-2', inputClass]"
              :placeholder="t('Tags, comma separated')"
            />
          </div>
          <div class="md:col-span-2">
            <label
              class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
              >{{ t("Metadata") }}</label
            >
            <textarea
              v-model="consumerDraft.metadataText"
              :class="['mt-2', textAreaClass]"
              :placeholder="t('Optional metadata JSON')"
            />
          </div>
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="closeConsumerDialog">{{
            t("Cancel")
          }}</Button>
          <Button data-stream-submit-consumer @click="submitConsumer">{{
            t("Create Consumer")
          }}</Button>
        </div>
      </div>
    </Overlay>

    <Overlay
      :open="controlDialogOpen"
      closeOnBackdrop
      trapFocus
      @close="closeControlDialog"
    >
      <div
        data-stream-control-dialog
        class="w-full max-w-5xl rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl"
      >
        <CardHeader
          :title="t('Control Pair Picker')"
          :description="
            t(
              'Choose a source and a consumer from the current control target, then connect or subscribe without expanding the main page.',
            )
          "
          title-class="text-lg"
        >
          <template #actions>
            <Badge variant="secondary">{{
              t("Target {id}", { id: resolvedTargetId || "-" })
            }}</Badge>
          </template>
        </CardHeader>

        <div class="mt-5 grid gap-6 xl:grid-cols-2">
          <section
            class="rounded-2xl border border-border/60 bg-background/70 p-4"
          >
            <div class="flex items-center justify-between gap-3">
              <p class="text-sm font-semibold">{{ t("Select Source") }}</p>
              <Button
                variant="outline"
                size="sm"
                @click="refreshControlSources"
              >
                <ScanSearch class="mr-2 h-4 w-4" />
                {{ t("Refresh") }}
              </Button>
            </div>

            <div class="mt-4 grid gap-3 sm:grid-cols-2">
              <div class="sm:col-span-2">
                <label
                  class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
                  >{{ t("Producer Node ID") }}</label
                >
                <input
                  v-model="controlSourceQuery.producer"
                  :class="['mt-2', inputClass]"
                  :placeholder="t('Producer Node ID')"
                />
              </div>
              <div>
                <label
                  class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
                  >{{ t("Kind") }}</label
                >
                <select
                  v-model="controlSourceQuery.kind"
                  :class="['mt-2', inputClass]"
                >
                  <option value="">{{ t("All kinds") }}</option>
                  <option v-for="kind in streamKinds" :key="kind" :value="kind">
                    {{ kind }}
                  </option>
                </select>
              </div>
              <div>
                <label
                  class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
                  >{{ t("Tag") }}</label
                >
                <input
                  v-model="controlSourceQuery.tag"
                  :class="['mt-2', inputClass]"
                  :placeholder="t('Tag filter')"
                />
              </div>
            </div>

            <div class="mt-4 max-h-[320px] space-y-3 overflow-y-auto pr-1">
              <button
                v-for="source in catalogSources"
                :key="source.sourceId"
                type="button"
                data-stream-control-source-row
                class="w-full rounded-2xl border p-4 text-left transition"
                :class="
                  stream.state.selectedSourceId === source.sourceId
                    ? 'border-primary/50 bg-primary/10 shadow-sm'
                    : 'border-border/60 bg-card/90 hover:border-border'
                "
                @click="stream.selectSource(source.sourceId)"
              >
                <div class="flex items-start justify-between gap-3">
                  <div class="min-w-0 flex-1">
                    <div class="flex flex-wrap items-center gap-2">
                      <p class="truncate text-sm font-semibold">
                        {{ source.name || source.sourceId }}
                      </p>
                      <Badge :class="kindToneClass(source.kind)">{{
                        source.kind
                      }}</Badge>
                      <Badge variant="secondary">{{
                        t("Producer {id}", { id: source.producer })
                      }}</Badge>
                    </div>
                    <p class="mt-2 text-xs text-muted-foreground">
                      {{ source.sourceId }}
                    </p>
                  </div>
                </div>
              </button>
              <div
                v-if="!catalogSources.length"
                class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-8 text-sm text-muted-foreground"
              >
                {{ t("No sources loaded yet.") }}
              </div>
            </div>
          </section>

          <section
            class="rounded-2xl border border-border/60 bg-background/70 p-4"
          >
            <div class="flex items-center justify-between gap-3">
              <p class="text-sm font-semibold">{{ t("Select Consumer") }}</p>
              <Button
                variant="outline"
                size="sm"
                @click="refreshControlConsumers"
              >
                <ScanSearch class="mr-2 h-4 w-4" />
                {{ t("Refresh") }}
              </Button>
            </div>

            <div class="mt-4 grid gap-3 sm:grid-cols-2">
              <div class="sm:col-span-2">
                <label
                  class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
                  >{{ t("Consumer Node ID") }}</label
                >
                <input
                  v-model="controlConsumerQuery.consumer"
                  :class="['mt-2', inputClass]"
                  :placeholder="t('Consumer Node ID')"
                />
              </div>
              <div>
                <label
                  class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
                  >{{ t("Kind") }}</label
                >
                <select
                  v-model="controlConsumerQuery.kind"
                  :class="['mt-2', inputClass]"
                >
                  <option value="">{{ t("All kinds") }}</option>
                  <option v-for="kind in streamKinds" :key="kind" :value="kind">
                    {{ kind }}
                  </option>
                </select>
              </div>
              <div>
                <label
                  class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
                  >{{ t("Tag") }}</label
                >
                <input
                  v-model="controlConsumerQuery.tag"
                  :class="['mt-2', inputClass]"
                  :placeholder="t('Tag filter')"
                />
              </div>
            </div>

            <div class="mt-4 max-h-[320px] space-y-3 overflow-y-auto pr-1">
              <button
                v-for="consumer in catalogConsumers"
                :key="consumer.consumerId"
                type="button"
                data-stream-control-consumer-row
                class="w-full rounded-2xl border p-4 text-left transition"
                :class="
                  stream.state.selectedConsumerId === consumer.consumerId
                    ? 'border-primary/50 bg-primary/10 shadow-sm'
                    : 'border-border/60 bg-card/90 hover:border-border'
                "
                @click="stream.selectConsumer(consumer.consumerId)"
              >
                <div class="flex items-start justify-between gap-3">
                  <div class="min-w-0 flex-1">
                    <div class="flex flex-wrap items-center gap-2">
                      <p class="truncate text-sm font-semibold">
                        {{ consumer.name || consumer.consumerId }}
                      </p>
                      <Badge :class="kindToneClass(consumer.kind)">{{
                        consumer.kind
                      }}</Badge>
                      <Badge variant="secondary">{{
                        t("Consumer {id}", { id: consumer.consumer })
                      }}</Badge>
                    </div>
                    <p class="mt-2 text-xs text-muted-foreground">
                      {{ consumer.consumerId }}
                    </p>
                  </div>
                </div>
              </button>
              <div
                v-if="!catalogConsumers.length"
                class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-8 text-sm text-muted-foreground"
              >
                {{ t("No consumers loaded yet.") }}
              </div>
            </div>
          </section>
        </div>

        <div
          class="mt-6 rounded-2xl border border-border/60 bg-background/70 p-4"
        >
          <div class="flex flex-wrap items-center gap-2">
            <p class="text-sm font-semibold">{{ t("Connect Pair") }}</p>
            <Badge
              v-if="selectedControlSource"
              :class="kindToneClass(selectedControlSource.kind)"
              >{{ selectedControlSource.kind }}</Badge
            >
            <Badge v-if="selectedControlConsumer" variant="secondary">{{
              t("Consumer {id}", { id: selectedControlConsumer.consumer })
            }}</Badge>
          </div>
          <p class="mt-2 text-xs text-muted-foreground">
            {{
              t(
                "Choose one source and one consumer, then connect or subscribe from this focused panel.",
              )
            }}
          </p>
          <p
            v-if="selectedControlSource"
            class="mt-3 text-xs text-muted-foreground"
          >
            {{ selectedControlSource.sourceId }}
          </p>
          <p
            v-if="selectedControlConsumer"
            class="mt-1 text-xs text-muted-foreground"
          >
            {{ selectedControlConsumer.consumerId }}
          </p>
          <p
            v-if="
              selectedControlSource && selectedControlConsumer && !canConnect
            "
            class="mt-3 text-xs text-rose-600"
          >
            {{ t("Source kind and consumer kind must match.") }}
          </p>
          <p
            v-else-if="
              selectedControlSource && selectedControlConsumer && !canSubscribe
            "
            class="mt-3 text-xs text-muted-foreground"
          >
            {{
              t(
                "Subscribe only works when the selected consumer belongs to this node.",
              )
            }}
          </p>
        </div>

        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="closeControlDialog">{{
            t("Cancel")
          }}</Button>
          <Button
            variant="outline"
            :disabled="!canSubscribe"
            data-stream-submit-control-subscribe
            @click="subscribeSelected"
            >{{ t("Subscribe") }}</Button
          >
          <Button
            :disabled="!canConnect"
            data-stream-submit-control-connect
            @click="connectSelected"
            >{{ t("Connect") }}</Button
          >
        </div>
      </div>
    </Overlay>

    <Overlay
      :open="subscribeDialog.open"
      closeOnBackdrop
      trapFocus
      @close="closeSubscribeDialog"
    >
      <div
        data-stream-subscribe-dialog
        class="w-full max-w-3xl rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl"
      >
        <CardHeader
          :title="t('Subscribe Consumer')"
          :description="
            t(
              'Choose a source from the current node or another node, then subscribe without expanding the main consumer list into a form.',
            )
          "
          title-class="text-lg"
        />

        <div v-if="subscribeConsumer" class="mt-5 space-y-4">
          <div class="flex flex-wrap items-center gap-2">
            <p class="text-lg font-semibold">
              {{ subscribeConsumer.name || subscribeConsumer.consumerId }}
            </p>
            <Badge :class="kindToneClass(subscribeConsumer.kind)">{{
              subscribeConsumer.kind
            }}</Badge>
          </div>

          <div class="rounded-2xl border border-border/60 bg-background/70 p-3">
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="sm:col-span-2">
                <label
                  class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
                  >{{ t("Producer Node ID") }}</label
                >
                <input
                  v-model="subscribeQuery.producer"
                  :class="['mt-2', inputClass]"
                  :placeholder="t('Producer Node ID')"
                />
              </div>
              <div>
                <label
                  class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
                  >{{ t("Kind") }}</label
                >
                <select
                  v-model="subscribeQuery.kind"
                  :class="['mt-2', inputClass]"
                >
                  <option value="">{{ t("All kinds") }}</option>
                  <option v-for="kind in streamKinds" :key="kind" :value="kind">
                    {{ kind }}
                  </option>
                </select>
              </div>
              <div>
                <label
                  class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
                  >{{ t("Tag") }}</label
                >
                <input
                  v-model="subscribeQuery.tag"
                  :class="['mt-2', inputClass]"
                  :placeholder="t('Tag filter')"
                />
              </div>
            </div>
            <div class="mt-3 flex flex-wrap gap-2">
              <Button
                variant="outline"
                size="sm"
                @click="refreshSubscribeSources"
              >
                <ScanSearch class="mr-2 h-4 w-4" />
                {{ t("Refresh") }}
              </Button>
            </div>
          </div>

          <div class="max-h-[320px] space-y-3 overflow-y-auto pr-1">
            <button
              v-for="source in catalogSources"
              :key="source.sourceId"
              type="button"
              data-stream-subscribe-source-row
              class="w-full rounded-2xl border p-4 text-left transition"
              :class="
                subscribeQuery.selectedSourceId === source.sourceId
                  ? 'border-primary/50 bg-primary/10 shadow-sm'
                  : 'border-border/60 bg-background/70 hover:border-border'
              "
              @click="subscribeQuery.selectedSourceId = source.sourceId"
            >
              <div class="flex flex-wrap items-center gap-2">
                <p class="truncate text-sm font-semibold">
                  {{ source.name || source.sourceId }}
                </p>
                <Badge :class="kindToneClass(source.kind)">{{
                  source.kind
                }}</Badge>
                <Badge variant="secondary">{{
                  t("Producer {id}", { id: source.producer })
                }}</Badge>
              </div>
              <p class="mt-2 text-xs text-muted-foreground">
                {{ source.sourceId }}
              </p>
            </button>
            <div
              v-if="!catalogSources.length"
              class="rounded-2xl border border-dashed border-border/70 bg-background/40 px-4 py-8 text-sm text-muted-foreground"
            >
              {{ t("No subscription sources loaded yet.") }}
            </div>
          </div>

          <div class="flex justify-end gap-2">
            <Button variant="outline" @click="closeSubscribeDialog">{{
              t("Cancel")
            }}</Button>
            <Button data-stream-submit-subscribe @click="subscribeFromDialog">{{
              t("Subscribe")
            }}</Button>
          </div>
        </div>
      </div>
    </Overlay>
  </section>
</template>
