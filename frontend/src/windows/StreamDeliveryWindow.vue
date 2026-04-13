<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { useRoute } from "vue-router"
import { RefreshCw } from "lucide-vue-next"
import CardHeader from "@/components/CardHeader.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"
import { useSessionStore } from "@/stores/session"
import { useStreamStore } from "@/stores/stream"
import { useToastStore } from "@/stores/toast"
import { HomeState as LoadHomeState } from "../../wailsjs/go/main/App"

const route = useRoute()
const sessionStore = useSessionStore()
const stream = useStreamStore()
const toast = useToastStore()
const { t } = useI18n()

const fallbackIdentity = reactive({ nodeId: 0, hubId: 0 })
const loading = reactive({ value: true })

const selfNodeId = computed(() => Number(sessionStore.auth.nodeId || fallbackIdentity.nodeId || 0))
const hubId = computed(() => Number(sessionStore.auth.hubId || fallbackIdentity.hubId || 0))
const deliveryId = computed(() => String(route.query.deliveryId ?? "").trim())
const delivery = computed(() => stream.state.deliveries.find((item) => item.deliveryId === deliveryId.value) ?? null)
const textFrames = computed(() => (delivery.value ? stream.textFramesFor(delivery.value.deliveryId) : []))
const stats = computed(() => (delivery.value ? stream.statsFor(delivery.value.deliveryId) : null))
const media = computed(() => (delivery.value ? stream.mediaForDelivery(delivery.value.deliveryId) : null))
const playerElement = ref<HTMLVideoElement | HTMLAudioElement | null>(null)
const playerError = ref("")
const isPlayableMedia = computed(() => delivery.value?.kind === "music" || delivery.value?.kind === "video")

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

const loadHomeDefaults = async () => {
  try {
    const state = await LoadHomeState()
    fallbackIdentity.nodeId = Number(state?.nodeId ?? 0)
    fallbackIdentity.hubId = Number(state?.hubId ?? 0)
  } catch (err) {
    console.warn(err)
  }
  stream.setIdentity(selfNodeId.value, hubId.value)
}

const refreshWindow = async () => {
  loading.value = true
  try {
    await loadHomeDefaults()
    await Promise.all([stream.loadPrefs(), stream.loadDeliveries(), stream.loadMedia()])
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to initialize Delivery output window."))
  } finally {
    loading.value = false
  }
}

const tryStartPlayback = () => {
  const player = playerElement.value
  if (!player) return
  const promise = player.play()
  if (promise && typeof promise.catch === "function") {
    promise.catch((err: unknown) => {
      playerError.value = err instanceof Error ? err.message : String(err ?? "")
    })
  }
}

const handleMediaCanPlay = () => {
  playerError.value = ""
  tryStartPlayback()
}

const handleMediaError = () => {
  const nativeError = playerElement.value?.error
  playerError.value = nativeError?.message || t("Playback failed for this media stream.")
}

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  () => {
    stream.setIdentity(selfNodeId.value, hubId.value)
  }
)

watch(
  () => deliveryId.value,
  () => {
    playerError.value = ""
    void refreshWindow()
  }
)

watch(
  () => media.value?.mediaUrl,
  () => {
    playerError.value = ""
  }
)

onMounted(() => {
  void refreshWindow()
})
</script>

<template>
  <section class="space-y-4" data-stream-delivery-window>
    <CardHeader
      :title="t('Delivery Output Window')"
      :description="t('Inspect one runtime delivery in a dedicated window.')"
      title-tag="h1"
      title-class="text-lg"
    >
      <template #actions>
        <div class="flex flex-wrap gap-2">
          <Badge variant="secondary">{{ t("Self {id}", { id: selfNodeId || "-" }) }}</Badge>
          <Badge variant="secondary">{{ t("Hub {id}", { id: hubId || "-" }) }}</Badge>
          <Button variant="outline" size="sm" @click="refreshWindow">
            <RefreshCw class="mr-2 h-4 w-4" />
            {{ t("Refresh") }}
          </Button>
        </div>
      </template>
    </CardHeader>

    <div v-if="loading.value" class="rounded-2xl border border-border/60 bg-card/90 p-6 text-card-foreground shadow-sm">
      <p class="text-sm text-muted-foreground">{{ t("Loading Stream delivery window...") }}</p>
    </div>

    <div v-else-if="!delivery" class="rounded-2xl border border-border/60 bg-card/90 p-6 text-card-foreground shadow-sm">
      <h2 class="text-base font-semibold">{{ t("Delivery not found in runtime state.") }}</h2>
      <p class="mt-2 text-sm text-muted-foreground">
        {{ t("Open this window from Runtime Deliveries after a connection or subscription is active.") }}
      </p>
    </div>

    <div v-else class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_320px]">
      <section class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
        <div class="flex flex-wrap items-center gap-2">
          <p class="text-lg font-semibold">{{ delivery.sourceId || delivery.deliveryId }}</p>
          <Badge :class="kindToneClass(delivery.kind)">{{ delivery.kind }}</Badge>
          <Badge variant="secondary">{{ delivery.state || t("Observed") }}</Badge>
        </div>
        <p class="mt-2 text-xs text-muted-foreground">{{ delivery.deliveryId }}</p>

        <div v-if="delivery.kind === 'text'" class="mt-5 rounded-2xl border border-emerald-200/20 bg-slate-950 px-4 py-4 text-sm text-emerald-100 shadow-inner">
          <div class="max-h-[520px] space-y-3 overflow-y-auto pr-1">
            <article v-for="frame in textFrames" :key="`${frame.deliveryId}:${frame.position}:${frame.updatedAt}`" class="rounded-xl border border-emerald-200/10 bg-white/5 px-3 py-3">
              <div class="flex items-center justify-between gap-3 text-[11px] text-emerald-200/70">
                <span>{{ frame.position }}</span>
                <span>{{ formatTimestamp(frame.updatedAt) }}</span>
              </div>
              <pre class="mt-2 whitespace-pre-wrap break-words font-mono text-[13px] leading-6">{{ frame.text }}</pre>
            </article>
            <div v-if="!textFrames.length" class="rounded-xl border border-dashed border-emerald-200/15 px-3 py-6 text-center text-emerald-200/60">
              {{ t("Waiting for text frames...") }}
            </div>
          </div>
        </div>

        <div v-else class="mt-5 space-y-4 rounded-2xl border border-border/60 bg-background/70 p-4 text-sm text-muted-foreground">
          <div
            v-if="isPlayableMedia && media && media.mediaUrl && media.state !== 'error' && media.state !== 'closed'"
            class="space-y-3 rounded-2xl border border-border/60 bg-card/90 p-3 text-card-foreground"
          >
            <video
              v-if="delivery.kind === 'video'"
              ref="playerElement"
              class="max-h-[520px] w-full rounded-xl bg-black"
              controls
              autoplay
              preload="auto"
              :src="media.mediaUrl"
              @canplay="handleMediaCanPlay"
              @loadeddata="handleMediaCanPlay"
              @error="handleMediaError"
            />
            <audio
              v-else
              ref="playerElement"
              class="w-full"
              controls
              autoplay
              preload="auto"
              :src="media.mediaUrl"
              @canplay="handleMediaCanPlay"
              @loadeddata="handleMediaCanPlay"
              @error="handleMediaError"
            />
            <p class="text-xs text-muted-foreground">
              {{
                media.state === "buffering"
                  ? t("Buffering media stream...")
                  : t("Progressive playback active. Received {bytes} bytes.", { bytes: media.availableBytes })
              }}
            </p>
          </div>

          <div v-if="media?.error || playerError" class="rounded-xl border border-rose-200/40 bg-rose-50 px-3 py-3 text-sm text-rose-700">
            {{ media?.error || playerError }}
          </div>

          <div class="rounded-xl border border-border/60 bg-card/70 px-3 py-3">
            <p>{{ t("Frames {frames} · Bytes {bytes}", { frames: delivery.framesIn, bytes: delivery.bytesIn }) }}</p>
            <p class="mt-2" v-if="stats">
              {{ t("Frames {frames} · Bytes {bytes} · ACK {ack} · Flags {flags}", { frames: stats.framesIn, bytes: stats.bytesIn, ack: stats.lastAckPos, flags: stats.lastFlags }) }}
            </p>
            <p class="mt-2" v-else>{{ t("No runtime stats available yet.") }}</p>
          </div>
        </div>
      </section>

      <aside class="space-y-4">
        <section class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
          <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Runtime summary") }}</p>
          <div class="mt-3 space-y-3 text-sm">
            <div>
              <p class="font-semibold">{{ t("Source") }}</p>
              <p class="mt-1 break-all text-muted-foreground">{{ delivery.sourceId || t("Unknown source") }}</p>
            </div>
            <div>
              <p class="font-semibold">{{ t("Consumer") }}</p>
              <p class="mt-1 break-all text-muted-foreground">{{ delivery.consumerId || t("Consumer {id}", { id: delivery.consumer || "-" }) }}</p>
            </div>
            <div>
              <p class="font-semibold">{{ t("Frames") }}</p>
              <p class="mt-1 text-muted-foreground">{{ delivery.framesIn }}</p>
            </div>
            <div>
              <p class="font-semibold">{{ t("Bytes") }}</p>
              <p class="mt-1 text-muted-foreground">{{ delivery.bytesIn }}</p>
            </div>
            <div>
              <p class="font-semibold">{{ t("Updated") }}</p>
              <p class="mt-1 text-muted-foreground">{{ formatTimestamp(delivery.updatedAt) || "-" }}</p>
            </div>
          </div>
        </section>
      </aside>
    </div>
  </section>
</template>
