<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

type WailsBinding = (...args: any[]) => Promise<any>

const callLocalHub = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.localhub?.LocalHubService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(`LocalHub binding '${method}' unavailable`)
  }
  return fn(...args)
}

type Snapshot = {
  supported: boolean
  platform: string
  arch: string
  rootDir: string
  binDir: string
  logsDir: string
  config: { host: string; port: number }
  latestLoaded: boolean
  latestError: string
  latest: { tag: string; name: string; publishedAt: string; assets: any[] }
  install: { installed: boolean; tag: string; binaryPath: string; installedAt: string }
  run: {
    running: boolean
    pid: number
    addr: string
    startedAt: string
    logPath: string
    exitedAt: string
    exitError: string
  }
  download: {
    active: boolean
    stage: string
    assetName: string
    expectedSha256: string
    totalBytes: number
    doneBytes: number
    message: string
    error: string
    startedAt: string
    updatedAt: string
  }
}

const snap = reactive<Snapshot>({
  supported: false,
  platform: "",
  arch: "",
  rootDir: "",
  binDir: "",
  logsDir: "",
  config: { host: "127.0.0.1", port: 9000 },
  latestLoaded: false,
  latestError: "",
  latest: { tag: "", name: "", publishedAt: "", assets: [] },
  install: { installed: false, tag: "", binaryPath: "", installedAt: "" },
  run: {
    running: false,
    pid: 0,
    addr: "",
    startedAt: "",
    logPath: "",
    exitedAt: "",
    exitError: ""
  },
  download: {
    active: false,
    stage: "",
    assetName: "",
    expectedSha256: "",
    totalBytes: 0,
    doneBytes: 0,
    message: "",
    error: "",
    startedAt: "",
    updatedAt: ""
  }
})

const form = reactive({
  host: "127.0.0.1",
  port: "9000"
})

const message = ref("")
const busy = reactive({
  loading: false,
  refreshing: false,
  saving: false,
  installing: false,
  starting: false,
  stopping: false,
  restarting: false
})

const normalizedPort = computed(() => {
  const raw = form.port.trim()
  if (!raw) return 0
  const parsed = Number.parseInt(raw, 10)
  if (Number.isNaN(parsed)) return 0
  return parsed
})

const nonLoopbackWarning = computed(() => {
  const host = form.host.trim().toLowerCase()
  if (!host) return false
  if (host === "127.0.0.1" || host === "localhost" || host === "::1") return false
  return true
})

const downloadProgress = computed(() => {
  const total = Number(snap.download.totalBytes || 0)
  const done = Number(snap.download.doneBytes || 0)
  if (!snap.download.active) return ""
  if (total > 0) {
    const pct = Math.max(0, Math.min(100, Math.round((done / total) * 100)))
    return `${pct}% (${done}/${total})`
  }
  return `${done} bytes`
})

const loadSnapshot = async () => {
  busy.loading = true
  message.value = ""
  try {
    const data = await callLocalHub<Snapshot>("Snapshot")
    Object.assign(snap, data)
    form.host = snap.config.host || "127.0.0.1"
    form.port = String(snap.config.port ?? 9000)
  } catch (err) {
    console.warn(err)
    message.value = (err as Error)?.message || "Failed to load Local Hub snapshot."
  } finally {
    busy.loading = false
  }
}

const saveConfig = async () => {
  if (busy.saving) return
  busy.saving = true
  message.value = ""
  try {
    const payload = { host: form.host.trim(), port: normalizedPort.value }
    await callLocalHub("SaveConfig", payload)
    await loadSnapshot()
  } catch (err) {
    console.warn(err)
    message.value = (err as Error)?.message || "Failed to save config."
  } finally {
    busy.saving = false
  }
}

const refreshLatest = async () => {
  if (busy.refreshing) return
  busy.refreshing = true
  message.value = ""
  try {
    await callLocalHub("RefreshLatest")
    await loadSnapshot()
  } catch (err) {
    console.warn(err)
    await loadSnapshot()
    message.value = (err as Error)?.message || "Failed to refresh latest release."
  } finally {
    busy.refreshing = false
  }
}

const installLatest = async () => {
  if (busy.installing) return
  busy.installing = true
  message.value = ""
  let timer: number | undefined
  try {
    const promise = callLocalHub("InstallLatest")
    timer = window.setInterval(() => {
      void loadSnapshot()
    }, 500)
    await promise
    await loadSnapshot()
  } catch (err) {
    console.warn(err)
    await loadSnapshot()
    message.value = (err as Error)?.message || "Install failed."
  } finally {
    if (timer) window.clearInterval(timer)
    busy.installing = false
  }
}

const startHub = async () => {
  if (busy.starting) return
  busy.starting = true
  message.value = ""
  try {
    await saveConfig()
    await callLocalHub("Start")
    await loadSnapshot()
  } catch (err) {
    console.warn(err)
    await loadSnapshot()
    message.value = (err as Error)?.message || "Failed to start hub."
  } finally {
    busy.starting = false
  }
}

const stopHub = async () => {
  if (busy.stopping) return
  busy.stopping = true
  message.value = ""
  try {
    await callLocalHub("Stop")
    await loadSnapshot()
  } catch (err) {
    console.warn(err)
    await loadSnapshot()
    message.value = (err as Error)?.message || "Failed to stop hub."
  } finally {
    busy.stopping = false
  }
}

const restartHub = async () => {
  if (busy.restarting) return
  busy.restarting = true
  message.value = ""
  try {
    await saveConfig()
    await callLocalHub("Restart")
    await loadSnapshot()
  } catch (err) {
    console.warn(err)
    await loadSnapshot()
    message.value = (err as Error)?.message || "Failed to restart hub."
  } finally {
    busy.restarting = false
  }
}

onMounted(async () => {
  await loadSnapshot()
  if (!snap.latestLoaded) {
    void refreshLatest()
  }
})
</script>

<template>
  <section class="space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">Session</p>
        <h1 class="text-2xl font-semibold">Local Hub</h1>
        <p class="text-sm text-muted-foreground">
          Download and run <span class="font-mono text-[12px] text-foreground">hub_server</span> as a sidecar process.
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <Badge v-if="snap.supported" variant="secondary">Supported: {{ snap.platform }}/{{ snap.arch }}</Badge>
        <Badge v-else variant="secondary">Unsupported: {{ snap.platform }}/{{ snap.arch }}</Badge>
        <Button variant="outline" size="sm" :disabled="busy.loading" @click="loadSnapshot">Reload</Button>
      </div>
    </div>

    <section class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 class="text-sm font-semibold">Latest Release</h2>
          <p class="text-xs text-muted-foreground">Source: GitHub Releases (myflowhub-server)</p>
        </div>
        <Button
          variant="outline"
          size="sm"
          :disabled="busy.refreshing || !snap.supported"
          @click="refreshLatest"
        >
          Refresh
        </Button>
      </div>

      <div class="mt-4 grid gap-3 text-sm md:grid-cols-2">
        <div class="rounded-xl border bg-background/70 p-3">
          <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Tag</p>
          <p class="mt-1 font-mono text-[12px]">{{ snap.latest.tag || "-" }}</p>
        </div>
        <div class="rounded-xl border bg-background/70 p-3">
          <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Published</p>
          <p class="mt-1 font-mono text-[12px]">{{ snap.latest.publishedAt || "-" }}</p>
        </div>
      </div>

      <p v-if="snap.latestError" class="mt-3 text-sm text-rose-600">{{ snap.latestError }}</p>
    </section>

    <section class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 class="text-sm font-semibold">Install</h2>
          <p class="text-xs text-muted-foreground">Binary is stored under your user config directory.</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <Button
            size="sm"
            :disabled="busy.installing || snap.download.active || !snap.supported"
            @click="installLatest"
          >
            {{ snap.install.installed ? "Reinstall Latest" : "Install Latest" }}
          </Button>
        </div>
      </div>

      <div class="mt-4 grid gap-3 text-sm md:grid-cols-2">
        <div class="rounded-xl border bg-background/70 p-3">
          <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Installed</p>
          <p class="mt-1">
            <span class="font-semibold">{{ snap.install.installed ? "Yes" : "No" }}</span>
            <span v-if="snap.install.tag" class="ml-2 font-mono text-[12px] text-muted-foreground">
              {{ snap.install.tag }}
            </span>
          </p>
          <p class="mt-1 truncate font-mono text-[11px] text-muted-foreground">
            {{ snap.install.binaryPath || "-" }}
          </p>
        </div>

        <div class="rounded-xl border bg-background/70 p-3">
          <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Download</p>
          <p class="mt-1 font-mono text-[12px]">
            <span v-if="snap.download.active">Stage: {{ snap.download.stage }}</span>
            <span v-else>Idle</span>
          </p>
          <p v-if="snap.download.active" class="mt-1 text-xs text-muted-foreground">
            {{ snap.download.assetName }} · {{ downloadProgress }}
          </p>
          <p v-if="snap.download.error" class="mt-2 text-xs text-rose-600">{{ snap.download.error }}</p>
        </div>
      </div>

      <div class="mt-3 text-xs text-muted-foreground">
        Root: <span class="font-mono text-[11px]">{{ snap.rootDir || "-" }}</span>
      </div>
    </section>

    <section class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 class="text-sm font-semibold">Run</h2>
          <p class="text-xs text-muted-foreground">
            Local Hub keeps running after the Win app exits. Stop it explicitly if needed.
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <Badge v-if="snap.run.running" variant="secondary">Running</Badge>
          <Badge v-else variant="secondary">Stopped</Badge>
        </div>
      </div>

      <div class="mt-4 grid gap-3 md:grid-cols-2">
        <div class="rounded-xl border bg-background/70 p-3 text-sm">
          <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Listen</p>
          <div class="mt-2 flex flex-wrap items-center gap-2">
            <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
              <span class="font-semibold uppercase tracking-[0.2em]">Host</span>
              <input
                v-model="form.host"
                class="h-7 w-40 rounded-md border border-input bg-background px-2 text-xs text-foreground"
                placeholder="127.0.0.1"
              />
            </div>
            <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
              <span class="font-semibold uppercase tracking-[0.2em]">Port</span>
              <input
                v-model="form.port"
                class="h-7 w-24 rounded-md border border-input bg-background px-2 text-xs text-foreground"
                placeholder="9000"
              />
            </div>
            <Button variant="outline" size="sm" :disabled="busy.saving" @click="saveConfig">
              Save
            </Button>
          </div>

          <p v-if="nonLoopbackWarning" class="mt-2 text-xs text-amber-700">
            Warning: non-loopback host may expose your hub to the LAN.
          </p>

          <p class="mt-3 text-xs text-muted-foreground">
            Port <span class="font-mono">0</span> means auto-pick an available port if conflict.
          </p>
        </div>

        <div class="rounded-xl border bg-background/70 p-3 text-sm">
          <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Status</p>
          <p class="mt-2 font-mono text-[12px]">addr: {{ snap.run.addr || "-" }}</p>
          <p class="mt-1 font-mono text-[12px]">pid: {{ snap.run.pid || "-" }}</p>
          <p class="mt-1 truncate font-mono text-[11px] text-muted-foreground">
            log: {{ snap.run.logPath || "-" }}
          </p>
          <p v-if="snap.run.exitError" class="mt-2 text-xs text-rose-600">
            exit: {{ snap.run.exitError }}
          </p>

          <div class="mt-3 flex flex-wrap items-center gap-2">
            <Button size="sm" :disabled="busy.starting || snap.run.running" @click="startHub">
              Start
            </Button>
            <Button
              size="sm"
              variant="outline"
              :disabled="busy.stopping || !snap.run.running"
              @click="stopHub"
            >
              Stop
            </Button>
            <Button
              size="sm"
              variant="outline"
              :disabled="busy.restarting || !snap.install.installed"
              @click="restartHub"
            >
              Restart
            </Button>
          </div>
        </div>
      </div>
    </section>

    <p v-if="message" class="text-sm text-rose-600">{{ message }}</p>
  </section>
</template>
