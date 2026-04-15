<script setup lang="ts">
// 本文件实现 Win 前端使用的独立 `StreamSourceWindow` 窗口。
import {
  computed,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  watch,
} from "vue";
import { useRoute } from "vue-router";
import { FolderOpen, RefreshCw, Send } from "lucide-vue-next";
import CardHeader from "@/components/CardHeader.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useI18n } from "@/i18n";
import { useSessionStore } from "@/stores/session";
import { useStreamStore } from "@/stores/stream";
import { useToastStore } from "@/stores/toast";
import { HomeState as LoadHomeState } from "../../wailsjs/go/main/App";

const route = useRoute();
const sessionStore = useSessionStore();
const stream = useStreamStore();
const toast = useToastStore();
const { t } = useI18n();

const textAreaClass =
  "min-h-[160px] w-full rounded-2xl border border-input bg-background px-3 py-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2";
const sourceInputId = "stream-source-window-input";
const fallbackIdentity = reactive({ nodeId: 0, hubId: 0 });
const text = ref("");
const loading = ref(true);
const busy = ref(false);
const sentItems = ref<Array<{ text: string; sent: number; at: string }>>([]);
const previewElement = ref<HTMLVideoElement | null>(null);
const capture = reactive({
  active: false,
  starting: false,
  stopping: false,
  error: "",
  status: "",
  deliveryIds: [] as string[],
  sentChunks: 0,
  sentBytes: 0,
});

const selfNodeId = computed(() =>
  Number(sessionStore.auth.nodeId || fallbackIdentity.nodeId || 0),
);
const hubId = computed(() =>
  Number(sessionStore.auth.hubId || fallbackIdentity.hubId || 0),
);
const sourceId = computed(() => String(route.query.sourceId ?? "").trim());
const source = computed(() => stream.sourceById(sourceId.value, "local"));
const bindings = computed(() =>
  source.value
    ? stream.deliveriesForSource(source.value.sourceId).filter(
        (item) =>
          String(item.state ?? "")
            .trim()
            .toLowerCase() !== "closed",
      )
    : [],
);
const isDesktopSource = computed(
  () =>
    source.value?.kind === "video" &&
    String(source.value?.inputKind ?? "")
      .trim()
      .toLowerCase() === "desktop",
);
const captureSupported = computed(
  () =>
    isDesktopSource.value &&
    typeof MediaRecorder !== "undefined" &&
    typeof navigator !== "undefined" &&
    !!navigator.mediaDevices?.getDisplayMedia,
);
const captureStatusText = computed(() => {
  if (capture.error) return capture.error;
  if (capture.starting) return t("Requesting desktop capture permission...");
  if (capture.stopping)
    return capture.status || t("Stopping desktop capture...");
  if (capture.active) {
    return t(
      "Desktop capture is live for {count} deliveries. Sent {chunks} chunks / {bytes} bytes.",
      {
        count: capture.deliveryIds.length,
        chunks: capture.sentChunks,
        bytes: capture.sentBytes,
      },
    );
  }
  return (
    capture.status ||
    t(
      "Desktop capture is idle. Start it from this window when deliveries are active.",
    )
  );
});

let recorder: MediaRecorder | null = null;
let previewStream: MediaStream | null = null;
let publishQueue: Promise<void> = Promise.resolve();
let captureRunID = 0;

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

const stopPreviewTracks = () => {
  if (previewStream) {
    for (const track of previewStream.getTracks()) track.stop();
  }
  previewStream = null;
  if (previewElement.value) {
    (
      previewElement.value as HTMLVideoElement & {
        srcObject?: MediaStream | null;
      }
    ).srcObject = null;
  }
};

const attachPreviewStream = (mediaStream: MediaStream | null) => {
  previewStream = mediaStream;
  if (!previewElement.value) return;
  (
    previewElement.value as HTMLVideoElement & {
      srcObject?: MediaStream | null;
    }
  ).srcObject = mediaStream;
  if (mediaStream) {
    const promise = previewElement.value.play();
    if (promise && typeof promise.catch === "function") {
      promise.catch(() => undefined);
    }
  }
};

const queueCapturePublish = async (task: () => Promise<void>) => {
  const next = publishQueue.then(task);
  publishQueue = next.catch(() => undefined);
  return next;
};

const chooseDesktopMimeType = () => {
  if (typeof MediaRecorder === "undefined") return "";
  const candidates = [
    "video/webm;codecs=vp9,opus",
    "video/webm;codecs=vp8,opus",
    "video/webm;codecs=vp9",
    "video/webm;codecs=vp8",
    "video/webm",
  ];
  for (const candidate of candidates) {
    if (
      typeof MediaRecorder.isTypeSupported !== "function" ||
      MediaRecorder.isTypeSupported(candidate)
    ) {
      return candidate;
    }
  }
  return "";
};

const finishDesktopCapture = async (
  runID: number,
  sourceKey: string,
  deliveryIDs: string[],
) => {
  try {
    if (deliveryIDs.length) {
      await queueCapturePublish(async () => {
        await stream.publishCaptureChunk({
          sourceId: sourceKey,
          deliveryIds: deliveryIDs,
          final: true,
          ptsMs: Date.now(),
        });
      });
    }
  } catch (err) {
    console.warn(err);
    capture.error = err instanceof Error ? err.message : String(err ?? "");
  } finally {
    if (captureRunID === runID) {
      capture.active = false;
      capture.starting = false;
      capture.stopping = false;
      recorder = null;
      stopPreviewTracks();
      if (!capture.error && !capture.status)
        capture.status = t("Desktop capture stopped.");
    }
  }
};

const stopDesktopCapture = async (
  reason = t("Desktop capture stopped."),
  silent = false,
) => {
  if (!recorder) {
    capture.active = false;
    capture.starting = false;
    capture.stopping = false;
    capture.status = reason;
    stopPreviewTracks();
    return;
  }
  if (capture.stopping) return;
  capture.active = false;
  capture.stopping = true;
  capture.status = reason;
  try {
    if (recorder.state === "recording") {
      recorder.requestData();
      recorder.stop();
      return;
    }
  } catch (err) {
    console.warn(err);
    capture.error = err instanceof Error ? err.message : String(err ?? "");
    if (!silent) toast.errorOf(err, t("Failed to stop desktop capture."));
  }
  recorder = null;
  capture.stopping = false;
  stopPreviewTracks();
};

const startDesktopCapture = async () => {
  if (
    !source.value ||
    !isDesktopSource.value ||
    capture.active ||
    capture.starting ||
    capture.stopping
  )
    return;
  if (!captureSupported.value) {
    toast.error(t("Desktop capture is not supported in this runtime."));
    return;
  }
  const deliveryIDs = bindings.value.map((item) => item.deliveryId);
  if (!deliveryIDs.length) {
    toast.error(t("Desktop capture requires at least one active delivery."));
    return;
  }

  const sourceKey = source.value.sourceId;
  capture.starting = true;
  capture.error = "";
  capture.status = "";
  capture.deliveryIds = deliveryIDs;
  capture.sentChunks = 0;
  capture.sentBytes = 0;
  const runID = captureRunID + 1;
  captureRunID = runID;

  let mediaStream: MediaStream | null = null;
  try {
    mediaStream = await navigator.mediaDevices.getDisplayMedia({
      video: true,
      audio: false,
    });
    const mimeType = chooseDesktopMimeType();
    recorder = mimeType
      ? new MediaRecorder(mediaStream, { mimeType })
      : new MediaRecorder(mediaStream);
    attachPreviewStream(mediaStream);

    let firstChunk = true;
    recorder.ondataavailable = (event: BlobEvent) => {
      if (captureRunID !== runID || !event.data || event.data.size <= 0) return;
      void queueCapturePublish(async () => {
        const bytes = Array.from(
          new Uint8Array(await event.data.arrayBuffer()),
        );
        await stream.publishCaptureChunk({
          sourceId: sourceKey,
          deliveryIds: deliveryIDs,
          payload: bytes,
          ptsMs: Date.now(),
          sessionStart: firstChunk,
        });
        firstChunk = false;
        capture.sentChunks += 1;
        capture.sentBytes += bytes.length;
        capture.error = "";
      }).catch((err) => {
        console.warn(err);
        capture.error = err instanceof Error ? err.message : String(err ?? "");
        capture.status = capture.error;
        void stopDesktopCapture(capture.error, true);
      });
    };
    recorder.onerror = (event: Event) => {
      const nextError = String(
        (event as Event & { error?: Error }).error?.message ??
          t("Desktop capture recorder failed."),
      );
      capture.error = nextError;
      capture.status = nextError;
      void stopDesktopCapture(nextError, true);
    };
    recorder.onstop = () => {
      void finishDesktopCapture(runID, sourceKey, deliveryIDs);
    };
    for (const track of mediaStream.getTracks()) {
      track.onended = () => {
        if (captureRunID !== runID) return;
        capture.status = t("Desktop sharing stopped by the system.");
        void stopDesktopCapture(capture.status, true);
      };
    }

    recorder.start(500);
    capture.starting = false;
    capture.active = true;
    capture.status = t("Desktop capture started.");
    toast.success(t("Desktop capture started."));
  } catch (err) {
    if (mediaStream) {
      for (const track of mediaStream.getTracks()) track.stop();
    }
    recorder = null;
    capture.starting = false;
    capture.active = false;
    capture.error = err instanceof Error ? err.message : String(err ?? "");
    capture.status = capture.error;
    toast.errorOf(err, t("Failed to start desktop capture."));
  }
};

const loadHomeDefaults = async () => {
  try {
    const state = await LoadHomeState();
    fallbackIdentity.nodeId = Number(state?.nodeId ?? 0);
    fallbackIdentity.hubId = Number(state?.hubId ?? 0);
  } catch (err) {
    console.warn(err);
  }
  stream.setIdentity(selfNodeId.value, hubId.value);
};

const refreshWindow = async () => {
  loading.value = true;
  try {
    await loadHomeDefaults();
    await Promise.all([stream.loadPrefs(), stream.loadDeliveries()]);
  } catch (err) {
    console.warn(err);
    toast.errorOf(err, t("Failed to initialize Source input window."));
  } finally {
    loading.value = false;
  }
};

const sendText = async () => {
  if (!source.value || source.value.kind !== "text" || busy.value) return;
  busy.value = true;
  try {
    const result = await stream.publishText(source.value.sourceId, text.value);
    sentItems.value = [
      { text: text.value, sent: result.sent, at: new Date().toISOString() },
      ...sentItems.value,
    ].slice(0, 8);
    text.value = "";
    toast.success(t("Text sent to source."));
  } catch (err) {
    console.warn(err);
    toast.errorOf(err, t("Failed to send text to source."));
  } finally {
    busy.value = false;
  }
};

const chooseMediaFile = async () => {
  if (!source.value || source.value.kind === "text" || busy.value) return;
  busy.value = true;
  try {
    const file = await stream.pickMediaFile();
    if (!file) return;
    await stream.updateSourceInput(source.value.sourceId, file);
    toast.success(t("Media file configured for source."));
  } catch (err) {
    console.warn(err);
    toast.errorOf(err, t("Failed to configure media file for source."));
  } finally {
    busy.value = false;
  }
};

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  () => {
    stream.setIdentity(selfNodeId.value, hubId.value);
  },
);

watch(
  () => sourceId.value,
  () => {
    text.value = "";
    sentItems.value = [];
    capture.error = "";
    capture.status = "";
    capture.deliveryIds = [];
    capture.sentChunks = 0;
    capture.sentBytes = 0;
    void refreshWindow();
  },
);

watch(
  () => bindings.value.length,
  (count) => {
    if ((capture.active || capture.stopping) && count === 0) {
      capture.status = t("No active deliveries remain. Capture stopped.");
      void stopDesktopCapture(capture.status, true);
    }
  },
);

watch(
  () => isDesktopSource.value,
  (enabled) => {
    if (!enabled && (capture.active || capture.starting || capture.stopping)) {
      capture.status = t(
        "Desktop source configuration changed. Capture stopped.",
      );
      void stopDesktopCapture(capture.status, true);
    }
  },
);

onMounted(() => {
  void refreshWindow();
});

onBeforeUnmount(() => {
  void stopDesktopCapture(
    t("Source window closed. Desktop capture stopped."),
    true,
  );
});
</script>

<template>
  <section class="space-y-4" data-stream-source-window>
    <CardHeader
      :title="t('Source Input Window')"
      :description="
        t(
          'Send text from a dedicated window so the main Stream page stays focused on the list.',
        )
      "
      title-tag="h1"
      title-class="text-lg"
    >
      <template #actions>
        <div class="flex flex-wrap gap-2">
          <Badge variant="secondary">{{
            t("Self {id}", { id: selfNodeId || "-" })
          }}</Badge>
          <Badge variant="secondary">{{
            t("Hub {id}", { id: hubId || "-" })
          }}</Badge>
          <Button variant="outline" size="sm" @click="refreshWindow">
            <RefreshCw class="mr-2 h-4 w-4" />
            {{ t("Refresh") }}
          </Button>
        </div>
      </template>
    </CardHeader>

    <div
      v-if="loading"
      class="rounded-2xl border border-border/60 bg-card/90 p-6 text-card-foreground shadow-sm"
    >
      <p class="text-sm text-muted-foreground">
        {{ t("Loading Stream source window...") }}
      </p>
    </div>

    <div
      v-else-if="!source"
      class="rounded-2xl border border-border/60 bg-card/90 p-6 text-card-foreground shadow-sm"
    >
      <h2 class="text-base font-semibold">
        {{ t("Source not found in local catalog.") }}
      </h2>
      <p class="mt-2 text-sm text-muted-foreground">
        {{
          t(
            "Open this window from the Source list after a local source is created.",
          )
        }}
      </p>
    </div>

    <div v-else class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_300px]">
      <section
        class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm"
      >
        <div class="flex flex-wrap items-center gap-2">
          <p class="text-lg font-semibold">
            {{ source.name || source.sourceId }}
          </p>
          <Badge :class="kindToneClass(source.kind)">{{ source.kind }}</Badge>
          <Badge variant="secondary">{{
            t("{count} active bindings", { count: bindings.length })
          }}</Badge>
          <Badge v-if="source.inputKind" variant="secondary">{{
            source.inputKind
          }}</Badge>
        </div>
        <p class="mt-2 text-xs text-muted-foreground">{{ source.sourceId }}</p>

        <div v-if="source.kind === 'text'" class="mt-5 space-y-4">
          <div>
            <label
              :for="sourceInputId"
              class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
              >{{ t("Text input") }}</label
            >
            <textarea
              :id="sourceInputId"
              v-model="text"
              :class="['mt-2', textAreaClass]"
              :placeholder="
                t('Type text for the active deliveries of this source...')
              "
            />
          </div>
          <div class="flex flex-wrap gap-2">
            <Button
              :disabled="busy"
              data-stream-source-window-send
              @click="sendText"
            >
              <Send class="mr-2 h-4 w-4" />
              {{ t("Send Text") }}
            </Button>
          </div>
        </div>

        <div
          v-else-if="isDesktopSource"
          class="mt-5 space-y-4 rounded-2xl border border-border/60 bg-background/70 px-4 py-4 text-sm text-muted-foreground"
        >
          <p>
            {{
              t(
                "Desktop capture sends live media chunks directly into the current active deliveries.",
              )
            }}
          </p>
          <div
            v-if="!captureSupported"
            class="rounded-xl border border-rose-200/40 bg-rose-50 px-3 py-3 text-sm text-rose-700"
          >
            {{ t("Desktop capture is not supported in this runtime.") }}
          </div>
          <div class="grid gap-3 md:grid-cols-2">
            <div
              class="rounded-xl border border-border/60 bg-card/70 px-3 py-3"
            >
              <p
                class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
              >
                {{ t("Capture status") }}
              </p>
              <p class="mt-2 text-card-foreground">{{ captureStatusText }}</p>
            </div>
            <div
              class="rounded-xl border border-border/60 bg-card/70 px-3 py-3"
            >
              <p
                class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
              >
                {{ t("Capture summary") }}
              </p>
              <p class="mt-2 text-card-foreground">
                {{
                  t("Chunks {chunks} · Bytes {bytes} · Deliveries {count}", {
                    chunks: capture.sentChunks,
                    bytes: capture.sentBytes,
                    count: capture.deliveryIds.length || bindings.length,
                  })
                }}
              </p>
            </div>
          </div>
          <div class="rounded-xl border border-border/60 bg-card/70 px-3 py-3">
            <p
              class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
            >
              {{ t("Local preview") }}
            </p>
            <video
              ref="previewElement"
              data-stream-source-preview
              class="mt-3 aspect-video w-full rounded-xl bg-black"
              autoplay
              muted
              playsinline
            />
            <p class="mt-2 text-xs text-muted-foreground">
              {{ t("Preview appears here after the desktop capture starts.") }}
            </p>
          </div>
          <div
            v-if="capture.error"
            class="rounded-xl border border-rose-200/40 bg-rose-50 px-3 py-3 text-sm text-rose-700"
          >
            {{ capture.error }}
          </div>
          <div class="flex flex-wrap gap-2">
            <Button
              data-stream-source-start-capture
              :disabled="
                busy ||
                capture.starting ||
                capture.active ||
                capture.stopping ||
                !captureSupported
              "
              @click="startDesktopCapture"
            >
              {{ t("Start Capture") }}
            </Button>
            <Button
              data-stream-source-stop-capture
              variant="outline"
              :disabled="!capture.active && !capture.stopping"
              @click="stopDesktopCapture()"
            >
              {{ t("Stop Capture") }}
            </Button>
          </div>
        </div>

        <div
          v-else
          class="mt-5 space-y-4 rounded-2xl border border-border/60 bg-background/70 px-4 py-4 text-sm text-muted-foreground"
        >
          <p>
            {{
              t(
                "This source uses a local media file as input. Once a delivery becomes active, the producer starts sending it immediately in chunk mode.",
              )
            }}
          </p>
          <div class="rounded-xl border border-border/60 bg-card/70 px-3 py-3">
            <p
              class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
            >
              {{ t("Configured file") }}
            </p>
            <p class="mt-2 break-all text-sm text-card-foreground">
              {{ source.filePath || t("No media file configured yet.") }}
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <Button :disabled="busy" variant="outline" @click="chooseMediaFile">
              <FolderOpen class="mr-2 h-4 w-4" />
              {{ source.filePath ? t("Replace File") : t("Choose File") }}
            </Button>
          </div>
        </div>
      </section>

      <aside class="space-y-4">
        <section
          class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm"
        >
          <p
            class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
          >
            {{ t("Current bindings") }}
          </p>
          <div class="mt-3 space-y-2">
            <article
              v-for="binding in bindings"
              :key="binding.deliveryId"
              class="rounded-xl border border-border/60 bg-background/70 px-3 py-3 text-sm"
            >
              <p class="font-semibold">
                {{
                  binding.consumerId ||
                  t("Consumer {id}", { id: binding.consumer })
                }}
              </p>
              <p class="mt-1 text-xs text-muted-foreground">
                {{ binding.deliveryId }}
              </p>
            </article>
            <p v-if="!bindings.length" class="text-sm text-muted-foreground">
              {{ t("No active consumers are currently bound to this source.") }}
            </p>
          </div>
        </section>

        <section
          v-if="source.kind === 'text'"
          class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm"
        >
          <p
            class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
          >
            {{ t("Recent sends") }}
          </p>
          <div class="mt-3 space-y-2">
            <article
              v-for="item in sentItems"
              :key="`${item.at}:${item.text}`"
              class="rounded-xl border border-border/60 bg-background/70 px-3 py-3 text-sm"
            >
              <div class="flex items-center justify-between gap-3">
                <p class="font-semibold">
                  {{ t("Sent to {count} deliveries", { count: item.sent }) }}
                </p>
                <span class="text-xs text-muted-foreground">{{
                  formatTimestamp(item.at)
                }}</span>
              </div>
              <pre
                class="mt-2 whitespace-pre-wrap break-words font-mono text-[12px] leading-6 text-muted-foreground"
                >{{ item.text }}</pre
              >
            </article>
            <p v-if="!sentItems.length" class="text-sm text-muted-foreground">
              {{ t("No text has been sent from this window yet.") }}
            </p>
          </div>
        </section>

        <section
          v-else-if="isDesktopSource"
          class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm"
        >
          <p
            class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground"
          >
            {{ t("Capture session") }}
          </p>
          <div class="mt-3 space-y-3 text-sm">
            <div>
              <p class="font-semibold">{{ t("Deliveries") }}</p>
              <p class="mt-1 text-muted-foreground">
                {{ capture.deliveryIds.length || bindings.length }}
              </p>
            </div>
            <div>
              <p class="font-semibold">{{ t("Chunks") }}</p>
              <p class="mt-1 text-muted-foreground">{{ capture.sentChunks }}</p>
            </div>
            <div>
              <p class="font-semibold">{{ t("Bytes") }}</p>
              <p class="mt-1 text-muted-foreground">{{ capture.sentBytes }}</p>
            </div>
          </div>
        </section>
      </aside>
    </div>
  </section>
</template>
