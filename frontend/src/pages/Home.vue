<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import CardHeader from "@/components/CardHeader.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"
import { useAppSettingsStore } from "@/stores/appSettings"
import { useProfileStore } from "@/stores/profile"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"
import {
  Close as CloseSession,
  Connect as ConnectSession,
  IsConnected,
  LastAddr
} from "../../wailsjs/go/session/SessionService"
import {
  EnsureKeys,
  LoginSimple,
  RegisterSimple
} from "../../wailsjs/go/auth/AuthService"
import {
  ClearHomeAuth,
  HomeState as LoadHomeState,
  SaveHomeState
} from "../../wailsjs/go/main/App"

type HomeState = {
  deviceId: string
  autoConnect: boolean
  autoLogin: boolean
  nodeId: number
  hubId: number
  role: string
}

const profileStore = useProfileStore()
const appSettings = useAppSettingsStore()
const sessionStore = useSessionStore()
const toast = useToastStore()
const { t } = useI18n()

const hardcodedDefaultAddr = "127.0.0.1:9000"
const addr = ref(hardcodedDefaultAddr)
const home = reactive<HomeState>({
  deviceId: "",
  autoConnect: false,
  autoLogin: false,
  nodeId: 0,
  hubId: 0,
  role: ""
})

const loading = ref(false)
const connecting = ref(false)
const authBusy = ref(false)

const statusLabel = computed(() => (sessionStore.connected ? t("Connected") : t("Disconnected")))
const statusTone = computed(() =>
  sessionStore.connected ? "bg-emerald-500/15 text-emerald-700" : "bg-rose-500/15 text-rose-700"
)
const loginLabel = computed(() => (home.nodeId ? t("Login") : t("Register")))
const panelStyle = { padding: "var(--app-panel-pad)" }
const settingsAutoConnectLabel = computed(() =>
  appSettings.state.settings.autoConnect
    ? t("Auto-connect enabled by default")
    : t("Auto-connect disabled by default")
)
const settingsAutoLoginLabel = computed(() =>
  appSettings.state.settings.autoLogin
    ? t("Auto-login enabled by default")
    : t("Auto-login disabled by default")
)

const formatId = (value: number) => (value > 0 ? String(value) : "-")

const nowIso = () => new Date().toISOString()

const syncStoreAuth = () => {
  sessionStore.auth.deviceId = home.deviceId
  sessionStore.auth.nodeId = home.nodeId
  sessionStore.auth.hubId = home.hubId
  sessionStore.auth.role = home.role
}

const applyHomeState = (state: any) => {
  home.deviceId = state?.deviceId ?? ""
  home.autoConnect = Boolean(state?.autoConnect)
  home.autoLogin = Boolean(state?.autoLogin)
  home.nodeId = Number(state?.nodeId ?? 0)
  home.hubId = Number(state?.hubId ?? 0)
  home.role = state?.role ?? ""
  syncStoreAuth()
}

const loadHomeState = async () => {
  loading.value = true
  try {
    const state = await LoadHomeState()
    applyHomeState(state)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to load saved Home settings."))
  } finally {
    loading.value = false
  }
}

const persistHomeState = async (patch?: Partial<HomeState>) => {
  if (loading.value) return
  const payload: HomeState = {
    deviceId: patch?.deviceId ?? home.deviceId,
    autoConnect: patch?.autoConnect ?? home.autoConnect,
    autoLogin: patch?.autoLogin ?? home.autoLogin,
    nodeId: patch?.nodeId ?? home.nodeId,
    hubId: patch?.hubId ?? home.hubId,
    role: patch?.role ?? home.role
  }
  try {
    const saved = await SaveHomeState(payload)
    applyHomeState(saved)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to save Home settings."))
  }
}

const refreshConnectionSnapshot = async () => {
  try {
    const connected = await IsConnected()
    sessionStore.connected = connected
    if (connected) {
      const last = await LastAddr()
      if (last) {
        sessionStore.addr = last
        addr.value = last
      }
    }
  } catch (err) {
    console.warn(err)
  }
}

const applySettingsDefaults = () => {
  const settings = appSettings.state.settings
  addr.value = settings.defaultAddr || hardcodedDefaultAddr
  if (!home.deviceId) {
    home.deviceId = settings.defaultDeviceId
    sessionStore.auth.deviceId = home.deviceId
  }
}

const loadPageState = async () => {
  await loadHomeState()
  try {
    await appSettings.load()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to load app settings."))
  }
  applySettingsDefaults()
  await refreshConnectionSnapshot()
  if (appSettings.state.settings.autoConnect && !sessionStore.connected && !connecting.value) {
    void connect()
  }
}

const connect = async () => {
  const target = addr.value.trim() || appSettings.state.settings.defaultAddr || hardcodedDefaultAddr
  if (connecting.value) return
  connecting.value = true
  try {
    await ConnectSession(target)
    sessionStore.addr = target
    toast.success(t("Connected."), target)
  } catch (err) {
    const text = String(err ?? "")
    if (!text.includes("已经连接") && !text.toLowerCase().includes("already connected")) {
      console.warn(err)
      toast.errorOf(err, t("Failed to connect to target."))
    }
  } finally {
    connecting.value = false
  }
}

const disconnect = async () => {
  if (connecting.value) return
  connecting.value = true
  try {
    await CloseSession()
    toast.info(t("Disconnected."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to disconnect."))
  } finally {
    connecting.value = false
  }
}

const loginOrRegister = async () => {
  if (authBusy.value) return
  const deviceId = home.deviceId.trim()
  if (!deviceId) {
    toast.warn(t("Device ID is required."))
    return
  }
  if (!sessionStore.connected) {
    toast.warn(t("Connect before logging in."))
    return
  }
  authBusy.value = true
  try {
    await EnsureKeys()
    await persistHomeState({ deviceId })
    const isLogin = Boolean(home.nodeId)
    const resp = isLogin
      ? await LoginSimple(0, 0, deviceId, home.nodeId)
      : await RegisterSimple(0, 0, deviceId)
    const nodeId = Number(resp?.node_id ?? resp?.nodeId ?? 0)
    const hubId = Number(resp?.hub_id ?? resp?.hubId ?? 0)
    const role = String(resp?.role ?? "")
    const authMsg = String(resp?.msg ?? "")

    sessionStore.auth.lastAuthAction = isLogin ? "login_resp" : "register_resp"
    sessionStore.auth.lastAuthMessage = authMsg || t("OK")
    sessionStore.auth.lastAuthAt = nowIso()
    sessionStore.auth.loggedIn = true
    sessionStore.auth.deviceId = deviceId
    if (nodeId) sessionStore.auth.nodeId = nodeId
    if (hubId) sessionStore.auth.hubId = hubId
    if (role) sessionStore.auth.role = role

    await persistHomeState({
      deviceId,
      nodeId: nodeId || home.nodeId,
      hubId: hubId || home.hubId,
      role: role || home.role
    })

    if (isLogin) {
      toast.success(
        t("Logged in."),
        t("node={nodeId} hub={hubId}", {
          nodeId: nodeId || home.nodeId,
          hubId: hubId || home.hubId
        })
      )
    } else {
      toast.success(
        t("Registered."),
        t("node={nodeId} hub={hubId}", {
          nodeId: nodeId || home.nodeId,
          hubId: hubId || home.hubId
        })
      )
    }
  } catch (err) {
    console.warn(err)
    const errMsg = (err as Error)?.message || String(err ?? "") || t("Login/register failed.")
    toast.error(errMsg)
    sessionStore.auth.lastAuthAction = home.nodeId ? "login_resp" : "register_resp"
    sessionStore.auth.lastAuthMessage = errMsg
    sessionStore.auth.lastAuthAt = nowIso()
    sessionStore.auth.loggedIn = false
  } finally {
    authBusy.value = false
  }
}

const clearAuth = async () => {
  try {
    const state = await ClearHomeAuth()
    applyHomeState(state)
    sessionStore.auth.loggedIn = false
    toast.success(t("Cleared saved auth."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to clear local auth state."))
  }
}

watch(
  () => [sessionStore.connected, appSettings.state.settings.autoLogin],
  ([connected, autoLogin]) => {
    if (connected && autoLogin && !authBusy.value) {
      void loginOrRegister()
    }
    if (!connected) {
      sessionStore.auth.loggedIn = false
    }
  }
)

watch(
  () => home.deviceId,
  (value) => {
    sessionStore.auth.deviceId = value
  }
)

watch(
  () => profileStore.state.current,
  async () => {
    await loadPageState()
  }
)

onMounted(async () => {
  await loadPageState()
})
</script>

<template>
  <section class="grid gap-6">
    <div class="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
      <div class="space-y-6">
        <div class="rounded-2xl border bg-card/90 text-card-foreground shadow-sm" :style="panelStyle">
          <CardHeader
            class="items-center"
            :title="t('Target Session')"
            :description="t('Connect to hub nodes and keep the console in sync.')"
            title-tag="h3"
            title-class="text-lg"
          >
            <template #actions>
              <Badge :class="statusTone">{{ statusLabel }}</Badge>
            </template>
          </CardHeader>

          <div class="mt-4 grid gap-4 lg:grid-cols-[2fr_1fr]">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Address") }}
              </label>
              <input
                v-model="addr"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                placeholder="127.0.0.1:9000"
              />
            </div>
            <div class="flex flex-col justify-end gap-2">
              <Button :disabled="connecting" @click="connect">{{ t("Connect") }}</Button>
              <Button
                variant="outline"
                :disabled="connecting || !sessionStore.connected"
                @click="disconnect"
              >
                {{ t("Disconnect") }}
              </Button>
            </div>
          </div>

          <div class="mt-4 flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
            <span class="rounded-full border border-border/60 bg-background/70 px-3 py-1 text-xs font-semibold">
              {{ settingsAutoConnectLabel }}
            </span>
            <span class="text-xs">
              {{ t("Default startup behavior is managed in") }}
              <a href="#/settings" class="font-semibold text-primary hover:underline">{{ t("Settings") }}</a>.
            </span>
            <span v-if="sessionStore.lastError" class="text-rose-600">
              {{ sessionStore.lastError }}
            </span>
          </div>
        </div>

        <div class="rounded-2xl border bg-card/90 text-card-foreground shadow-sm" :style="panelStyle">
          <CardHeader
            class="items-center"
            :title="t('Device Identity')"
            :description="t('Register once, then sign in with your saved node ID.')"
            title-tag="h3"
            title-class="text-lg"
          >
            <template #actions>
              <Badge variant="secondary">
                {{ sessionStore.auth.loggedIn ? t("Logged In") : t("Not Logged In") }}
              </Badge>
            </template>
          </CardHeader>

          <div class="mt-4 grid gap-4 lg:grid-cols-[2fr_1fr]">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Device ID") }}
              </label>
              <input
                v-model="home.deviceId"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                :placeholder="t('device-001')"
              />
            </div>
            <div class="flex flex-col justify-end gap-2">
              <Button :disabled="authBusy || loading" @click="loginOrRegister">{{ loginLabel }}</Button>
              <Button variant="outline" :disabled="authBusy" @click="clearAuth">
                {{ t("Clear Saved Auth") }}
              </Button>
            </div>
          </div>

          <div class="mt-4 flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
            <span class="rounded-full border border-border/60 bg-background/70 px-3 py-1 text-xs font-semibold">
              {{ settingsAutoLoginLabel }}
            </span>
            <span class="text-xs">
              {{ t("Default device ID:") }}
              <span class="font-semibold text-foreground">{{ appSettings.state.settings.defaultDeviceId || "-" }}</span>
            </span>
            <span v-if="sessionStore.auth.lastAuthMessage" class="text-xs">
              {{ sessionStore.auth.lastAuthMessage }}
            </span>
          </div>
        </div>
      </div>

      <div class="space-y-6">
        <div class="rounded-2xl border bg-card/90 text-card-foreground shadow-sm" :style="panelStyle">
          <CardHeader :title="t('Current Credentials')" title-tag="h3" title-class="text-lg" />
          <div class="mt-4 grid gap-3 text-sm">
            <div class="rounded-xl border border-border/60 bg-background/70 p-3">
              <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Node ID") }}</p>
              <p class="text-base font-semibold">{{ formatId(home.nodeId) }}</p>
            </div>
            <div class="rounded-xl border border-border/60 bg-background/70 p-3">
              <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Hub ID") }}</p>
              <p class="text-base font-semibold">{{ formatId(home.hubId) }}</p>
            </div>
            <div class="rounded-xl border border-border/60 bg-background/70 p-3">
              <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Role") }}</p>
              <p class="text-base font-semibold">{{ home.role || "-" }}</p>
            </div>
          </div>
        </div>

        <div class="rounded-2xl border bg-card/90 text-card-foreground shadow-sm" :style="panelStyle">
          <CardHeader :title="t('Live Status')" title-tag="h3" title-class="text-lg" />
          <div class="mt-4 space-y-3 text-sm text-muted-foreground">
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Profile") }}</p>
              <p class="break-all">{{ profileStore.state.current }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Keys Path") }}</p>
              <p class="break-all">{{ profileStore.state.keysPath || "-" }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Last Auth") }}</p>
              <p>{{ sessionStore.auth.lastAuthAt || "-" }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Last Frame") }}</p>
              <p>{{ sessionStore.lastFrameAt || "-" }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

  </section>
</template>
