<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useToastStore } from "@/stores/toast"

type WailsBinding = (...args: any[]) => Promise<any>

const callLocalHub = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.localhub?.LocalHubService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(`LocalHub binding '${method}' unavailable`)
  }
  return fn(...args)
}

type HubConfig = {
  host: string
  port: number
  nodeId: number
  parent: string
  parentEnable: boolean
  parentReconnectSec: number
  authDefaultRole: string
  authDefaultPerms: string
  authNodeRoles: string
  authRolePerms: string
  extraArgs: string
}

type Snapshot = {
  supported: boolean
  platform: string
  arch: string
  rootDir: string
  binDir: string
  logsDir: string
  config: HubConfig
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

const toast = useToastStore()

const snap = reactive<Snapshot>({
  supported: false,
  platform: "",
  arch: "",
  rootDir: "",
  binDir: "",
  logsDir: "",
  config: {
    host: "127.0.0.1",
    port: 9000,
    nodeId: 1,
    parent: "",
    parentEnable: false,
    parentReconnectSec: 3,
    authDefaultRole: "",
    authDefaultPerms: "",
    authNodeRoles: "",
    authRolePerms: "",
    extraArgs: ""
  },
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
  port: "9000",
  nodeId: "1",
  parent: "",
  parentEnable: false,
  parentReconnectSec: "3",
  authDefaultRole: "",
  authDefaultPerms: "",
  authNodeRoles: "",
  authRolePerms: "",
  extraArgs: ""
})

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

const normalizedNodeId = computed(() => {
  const raw = form.nodeId.trim()
  if (!raw) return 0
  const parsed = Number.parseInt(raw, 10)
  if (Number.isNaN(parsed) || parsed <= 0) return 0
  return parsed
})

const normalizedParentReconnectSec = computed(() => {
  const raw = form.parentReconnectSec.trim()
  if (!raw) return 0
  const parsed = Number.parseInt(raw, 10)
  if (Number.isNaN(parsed) || parsed < 0) return -1
  return parsed
})

const missingParentWarning = computed(() => {
  return Boolean(form.parentEnable && !form.parent.trim())
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
  try {
    const data = await callLocalHub<Snapshot>("Snapshot")
    Object.assign(snap, data)
    form.host = snap.config.host || "127.0.0.1"
    form.port = String(snap.config.port ?? 9000)
    form.nodeId = String(snap.config.nodeId ?? 1)
    form.parent = String(snap.config.parent ?? "")
    form.parentEnable = Boolean(snap.config.parentEnable)
    form.parentReconnectSec = String(snap.config.parentReconnectSec ?? 3)
    form.authDefaultRole = String(snap.config.authDefaultRole ?? "")
    form.authDefaultPerms = String(snap.config.authDefaultPerms ?? "")
    form.authNodeRoles = String(snap.config.authNodeRoles ?? "")
    form.authRolePerms = String(snap.config.authRolePerms ?? "")
    form.extraArgs = String(snap.config.extraArgs ?? "")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to load Local Hub snapshot.")
  } finally {
    busy.loading = false
  }
}

const saveConfig = async (silent = false) => {
  if (busy.saving) return

  const rawPort = form.port.trim()
  if (rawPort && Number.isNaN(Number.parseInt(rawPort, 10))) {
    toast.warn("Port must be a number.")
    return
  }
  const nodeId = normalizedNodeId.value
  if (!nodeId) {
    toast.warn("Node ID must be a positive number.")
    return
  }
  const parentReconnectSec = normalizedParentReconnectSec.value
  if (parentReconnectSec < 0) {
    toast.warn("Parent reconnect seconds must be 0 or a positive number.")
    return
  }
  if (form.parentEnable && !form.parent.trim()) {
    toast.warn("Parent address is required when parent link is enabled.")
    return
  }

  busy.saving = true
  try {
    const payload = {
      host: form.host.trim(),
      port: normalizedPort.value,
      nodeId,
      parent: form.parent.trim(),
      parentEnable: Boolean(form.parentEnable),
      parentReconnectSec,
      authDefaultRole: form.authDefaultRole.trim(),
      authDefaultPerms: form.authDefaultPerms.trim(),
      authNodeRoles: form.authNodeRoles.trim(),
      authRolePerms: form.authRolePerms.trim(),
      extraArgs: form.extraArgs
    }
    await callLocalHub("SaveConfig", payload)
    await loadSnapshot()
    if (!silent) {
      toast.success("Config saved.")
    }
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to save config.")
  } finally {
    busy.saving = false
  }
}

const refreshLatest = async () => {
  if (busy.refreshing) return
  busy.refreshing = true
  try {
    await callLocalHub("RefreshLatest")
    await loadSnapshot()
    toast.success("Latest release refreshed.")
  } catch (err) {
    console.warn(err)
    await loadSnapshot()
    toast.errorOf(err, "Failed to refresh latest release.")
  } finally {
    busy.refreshing = false
  }
}

const installLatest = async () => {
  if (busy.installing) return
  busy.installing = true
  let timer: number | undefined
  try {
    const promise = callLocalHub("InstallLatest")
    timer = window.setInterval(() => {
      void loadSnapshot()
    }, 500)
    await promise
    await loadSnapshot()
    toast.success("Installed.")
  } catch (err) {
    console.warn(err)
    await loadSnapshot()
    toast.errorOf(err, "Install failed.")
  } finally {
    if (timer) window.clearInterval(timer)
    busy.installing = false
  }
}

const startHub = async () => {
  if (busy.starting) return
  busy.starting = true
  try {
    await saveConfig(true)
    await callLocalHub("Start")
    await loadSnapshot()
    toast.success("Local Hub started.", snap.run.addr || undefined)
  } catch (err) {
    console.warn(err)
    await loadSnapshot()
    toast.errorOf(err, "Failed to start hub.")
  } finally {
    busy.starting = false
  }
}

const stopHub = async () => {
  if (busy.stopping) return
  busy.stopping = true
  try {
    await callLocalHub("Stop")
    await loadSnapshot()
    toast.info("Local Hub stopped.")
  } catch (err) {
    console.warn(err)
    await loadSnapshot()
    toast.errorOf(err, "Failed to stop hub.")
  } finally {
    busy.stopping = false
  }
}

const restartHub = async () => {
  if (busy.restarting) return
  busy.restarting = true
  try {
    await saveConfig(true)
    await callLocalHub("Restart")
    await loadSnapshot()
    toast.success("Local Hub restarted.", snap.run.addr || undefined)
  } catch (err) {
    console.warn(err)
    await loadSnapshot()
    toast.errorOf(err, "Failed to restart hub.")
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
            :disabled="busy.installing || snap.download.active || snap.run.running || !snap.supported"
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

      <p v-if="snap.run.running" class="mt-3 text-xs text-muted-foreground">
        Stop Local Hub before reinstalling/upgrading.
      </p>

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
              Save Config
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

      <div class="mt-4 rounded-xl border bg-background/70 p-3 text-sm">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Hub Params</p>
          <Button variant="outline" size="sm" :disabled="busy.saving" @click="saveConfig">
            Save Config
          </Button>
        </div>

        <div class="mt-3 grid gap-3 lg:grid-cols-2">
          <div class="flex flex-wrap items-center gap-2">
            <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
              <span class="font-semibold uppercase tracking-[0.2em]">Node ID</span>
              <input
                v-model="form.nodeId"
                class="h-7 w-20 rounded-md border border-input bg-background px-2 text-xs text-foreground"
                placeholder="1"
              />
            </div>

            <label
              class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground"
            >
              <input
                v-model="form.parentEnable"
                type="checkbox"
                class="h-4 w-4 rounded border border-input accent-primary"
              />
              Parent link
            </label>

            <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
              <span class="font-semibold uppercase tracking-[0.2em]">Reconnect</span>
              <input
                v-model="form.parentReconnectSec"
                class="h-7 w-20 rounded-md border border-input bg-background px-2 text-xs text-foreground disabled:opacity-60"
                placeholder="3"
                :disabled="!form.parentEnable"
              />
              <span class="text-[11px] text-muted-foreground">sec</span>
            </div>
          </div>

          <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
            <span class="font-semibold uppercase tracking-[0.2em]">Parent</span>
            <input
              v-model="form.parent"
              class="h-7 w-64 max-w-full rounded-md border border-input bg-background px-2 text-xs text-foreground disabled:opacity-60"
              placeholder="127.0.0.1:9000"
              :disabled="!form.parentEnable"
            />
          </div>
        </div>

        <p v-if="missingParentWarning" class="mt-2 text-xs text-amber-700">
          Parent address is required when parent link is enabled.
        </p>

        <div class="mt-4 grid gap-3 lg:grid-cols-2">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Auth Default Role
            </label>
            <input
              v-model="form.authDefaultRole"
              class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="node"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Auth Default Perms
            </label>
            <input
              v-model="form.authDefaultPerms"
              class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="file.read,file.write"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Auth Node Roles
            </label>
            <input
              v-model="form.authNodeRoles"
              class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="1:admin;2:node"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Auth Role Perms
            </label>
            <input
              v-model="form.authRolePerms"
              class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="admin:p1,p2;node:p3"
            />
          </div>
        </div>

        <details class="mt-4 rounded-lg border border-border/60 bg-background/60 px-3 py-2">
          <summary
            class="cursor-pointer text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
          >
            Advanced Args
          </summary>
          <div class="mt-3">
            <textarea
              v-model="form.extraArgs"
              rows="5"
              class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              placeholder="-send-workers=4\n-proc-workers=8"
            />
            <p class="mt-2 text-xs text-muted-foreground">
              One full argument per line. Lines starting with <span class="font-mono">#</span> are ignored.
            </p>
          </div>
        </details>
      </div>
    </section>

  </section>
</template>
