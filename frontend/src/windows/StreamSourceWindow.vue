<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { useRoute } from "vue-router"
import { RefreshCw, Send } from "lucide-vue-next"
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

const textAreaClass =
  "min-h-[160px] w-full rounded-2xl border border-input bg-background px-3 py-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
const sourceInputId = "stream-source-window-input"
const fallbackIdentity = reactive({ nodeId: 0, hubId: 0 })
const text = ref("")
const loading = ref(true)
const busy = ref(false)
const sentItems = ref<Array<{ text: string; sent: number; at: string }>>([])

const selfNodeId = computed(() => Number(sessionStore.auth.nodeId || fallbackIdentity.nodeId || 0))
const hubId = computed(() => Number(sessionStore.auth.hubId || fallbackIdentity.hubId || 0))
const sourceId = computed(() => String(route.query.sourceId ?? "").trim())
const source = computed(() => stream.sourceById(sourceId.value, "local"))
const bindings = computed(() =>
  source.value
    ? stream.deliveriesForSource(source.value.sourceId).filter((item) => String(item.state ?? "").trim().toLowerCase() !== "closed")
    : []
)

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
    await Promise.all([stream.loadPrefs(), stream.loadDeliveries()])
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to initialize Source input window."))
  } finally {
    loading.value = false
  }
}

const sendText = async () => {
  if (!source.value || source.value.kind !== "text" || busy.value) return
  busy.value = true
  try {
    const result = await stream.publishText(source.value.sourceId, text.value)
    sentItems.value = [
      { text: text.value, sent: result.sent, at: new Date().toISOString() },
      ...sentItems.value
    ].slice(0, 8)
    text.value = ""
    toast.success(t("Text sent to source."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to send text to source."))
  } finally {
    busy.value = false
  }
}

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  () => {
    stream.setIdentity(selfNodeId.value, hubId.value)
  }
)

watch(
  () => sourceId.value,
  () => {
    text.value = ""
    sentItems.value = []
    void refreshWindow()
  }
)

onMounted(() => {
  void refreshWindow()
})
</script>

<template>
  <section class="space-y-4" data-stream-source-window>
    <CardHeader
      :title="t('Source Input Window')"
      :description="t('Send text from a dedicated window so the main Stream page stays focused on the list.')"
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

    <div v-if="loading" class="rounded-2xl border border-border/60 bg-card/90 p-6 text-card-foreground shadow-sm">
      <p class="text-sm text-muted-foreground">{{ t("Loading Stream source window...") }}</p>
    </div>

    <div v-else-if="!source" class="rounded-2xl border border-border/60 bg-card/90 p-6 text-card-foreground shadow-sm">
      <h2 class="text-base font-semibold">{{ t("Source not found in local catalog.") }}</h2>
      <p class="mt-2 text-sm text-muted-foreground">
        {{ t("Open this window from the Source list after a local source is created.") }}
      </p>
    </div>

    <div v-else class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_300px]">
      <section class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
        <div class="flex flex-wrap items-center gap-2">
          <p class="text-lg font-semibold">{{ source.name || source.sourceId }}</p>
          <Badge :class="kindToneClass(source.kind)">{{ source.kind }}</Badge>
          <Badge variant="secondary">{{ t("{count} active bindings", { count: bindings.length }) }}</Badge>
        </div>
        <p class="mt-2 text-xs text-muted-foreground">{{ source.sourceId }}</p>

        <div v-if="source.kind === 'text'" class="mt-5 space-y-4">
          <div>
            <label :for="sourceInputId" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Text input") }}</label>
            <textarea :id="sourceInputId" v-model="text" :class="['mt-2', textAreaClass]" :placeholder="t('Type text for the active deliveries of this source...')" />
          </div>
          <div class="flex flex-wrap gap-2">
            <Button :disabled="busy" data-stream-source-window-send @click="sendText">
              <Send class="mr-2 h-4 w-4" />
              {{ t("Send Text") }}
            </Button>
          </div>
        </div>

        <div v-else class="mt-5 rounded-2xl border border-border/60 bg-background/70 px-4 py-4 text-sm text-muted-foreground">
          {{ t("Direct input is currently available only for text sources. Other kinds still use the control plane and runtime viewer.") }}
        </div>
      </section>

      <aside class="space-y-4">
        <section class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
          <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Current bindings") }}</p>
          <div class="mt-3 space-y-2">
            <article v-for="binding in bindings" :key="binding.deliveryId" class="rounded-xl border border-border/60 bg-background/70 px-3 py-3 text-sm">
              <p class="font-semibold">{{ binding.consumerId || t("Consumer {id}", { id: binding.consumer }) }}</p>
              <p class="mt-1 text-xs text-muted-foreground">{{ binding.deliveryId }}</p>
            </article>
            <p v-if="!bindings.length" class="text-sm text-muted-foreground">
              {{ t("No active consumers are currently bound to this source.") }}
            </p>
          </div>
        </section>

        <section class="rounded-2xl border border-border/60 bg-card/90 p-5 text-card-foreground shadow-sm">
          <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">{{ t("Recent sends") }}</p>
          <div class="mt-3 space-y-2">
            <article v-for="item in sentItems" :key="`${item.at}:${item.text}`" class="rounded-xl border border-border/60 bg-background/70 px-3 py-3 text-sm">
              <div class="flex items-center justify-between gap-3">
                <p class="font-semibold">{{ t("Sent to {count} deliveries", { count: item.sent }) }}</p>
                <span class="text-xs text-muted-foreground">{{ formatTimestamp(item.at) }}</span>
              </div>
              <pre class="mt-2 whitespace-pre-wrap break-words font-mono text-[12px] leading-6 text-muted-foreground">{{ item.text }}</pre>
            </article>
            <p v-if="!sentItems.length" class="text-sm text-muted-foreground">
              {{ t("No text has been sent from this window yet.") }}
            </p>
          </div>
        </section>
      </aside>
    </div>
  </section>
</template>
