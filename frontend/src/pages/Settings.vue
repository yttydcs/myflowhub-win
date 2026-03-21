<script setup lang="ts">
import { computed, reactive, watch } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  densityOptions,
  normalizeAppSettings,
  startPageOptions,
  useAppSettingsStore,
  type AppSettingsState
} from "@/stores/appSettings"
import { useProfileStore } from "@/stores/profile"
import { useToastStore } from "@/stores/toast"

const appSettings = useAppSettingsStore()
const profileStore = useProfileStore()
const toast = useToastStore()

const panelStyle = { padding: "var(--app-panel-pad)" }

const draft = reactive<AppSettingsState>({
  defaultAddr: "127.0.0.1:9000",
  defaultDeviceId: "",
  autoConnect: false,
  autoLogin: false,
  defaultStartPage: "home",
  density: "comfortable",
  reduceMotion: false
})

const syncDraft = () => {
  Object.assign(draft, normalizeAppSettings(appSettings.state.settings))
}

const loadPage = async () => {
  await Promise.all([appSettings.load(), appSettings.loadAbout()])
  syncDraft()
}

const dirty = computed(
  () => JSON.stringify(normalizeAppSettings(draft)) !== JSON.stringify(normalizeAppSettings(appSettings.state.settings))
)

const busy = computed(
  () => appSettings.state.loading || appSettings.state.saving || appSettings.state.aboutLoading
)

const aboutItems = computed(() => [
  { label: "App Version", value: appSettings.state.about.appVersion },
  { label: "Build Time", value: appSettings.state.about.buildTime },
  { label: "Build Mode", value: appSettings.state.about.buildMode },
  { label: "Commit", value: appSettings.state.about.commit },
  { label: "Platform", value: appSettings.state.about.platform },
  { label: "Go Version", value: appSettings.state.about.goVersion },
  { label: "Wails Version", value: appSettings.state.about.wailsVersion },
  { label: "Profile", value: appSettings.state.about.profile },
  { label: "Base Dir", value: appSettings.state.about.baseDir, breakAll: true },
  { label: "Settings Path", value: appSettings.state.about.settingsPath, breakAll: true },
  { label: "Keys Path", value: appSettings.state.about.keysPath, breakAll: true }
])

const saveSettings = async () => {
  try {
    await appSettings.save({ ...draft })
    syncDraft()
    toast.success("Settings saved.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to save settings.")
  }
}

const resetSettings = async () => {
  try {
    await appSettings.reset()
    syncDraft()
    toast.info("Settings restored to defaults.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to restore defaults.")
  }
}

watch(
  () => profileStore.state.current,
  () => {
    void (async () => {
      try {
        await loadPage()
      } catch (err) {
        console.warn(err)
        toast.errorOf(err, "Failed to load settings.")
      }
    })()
  },
  { immediate: true }
)
</script>

<template>
  <section class="grid gap-6">
    <div class="rounded-2xl border bg-card/90 text-card-foreground shadow-sm" :style="panelStyle">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
            App Settings
          </p>
          <h3 class="mt-1 text-xl font-semibold">Defaults, interface, and build information</h3>
          <p class="mt-2 max-w-2xl text-sm text-muted-foreground">
            These preferences are stored per profile. Save applies them immediately and updates startup behavior for the next launch.
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <Badge variant="secondary">Profile: {{ profileStore.state.current }}</Badge>
          <Badge variant="muted">{{ dirty ? "Unsaved changes" : "Saved" }}</Badge>
        </div>
      </div>
    </div>

    <div class="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
      <div class="space-y-6">
        <div class="rounded-2xl border bg-card/90 text-card-foreground shadow-sm" :style="panelStyle">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
                Startup Defaults
              </p>
              <h3 class="mt-1 text-lg font-semibold">Connection and identity</h3>
            </div>
            <Badge :variant="draft.autoConnect || draft.autoLogin ? 'secondary' : 'muted'">
              {{ draft.autoConnect || draft.autoLogin ? "Automation enabled" : "Manual startup" }}
            </Badge>
          </div>

          <div class="mt-5 grid gap-4">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Default Address
              </label>
              <input
                v-model="draft.defaultAddr"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                placeholder="127.0.0.1:9000"
              />
            </div>

            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Default Device ID
              </label>
              <input
                v-model="draft.defaultDeviceId"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                placeholder="device-001"
              />
            </div>

            <div class="grid gap-3 md:grid-cols-2">
              <label class="flex items-center gap-3 rounded-xl border border-border/60 bg-background/70 px-4 py-3">
                <input
                  v-model="draft.autoConnect"
                  type="checkbox"
                  class="h-4 w-4 rounded border border-input accent-primary"
                />
                <div>
                  <p class="text-sm font-semibold text-foreground">Auto-connect</p>
                  <p class="text-xs text-muted-foreground">Connect to the default address on launch.</p>
                </div>
              </label>

              <label class="flex items-center gap-3 rounded-xl border border-border/60 bg-background/70 px-4 py-3">
                <input
                  v-model="draft.autoLogin"
                  type="checkbox"
                  class="h-4 w-4 rounded border border-input accent-primary"
                />
                <div>
                  <p class="text-sm font-semibold text-foreground">Auto-login</p>
                  <p class="text-xs text-muted-foreground">Login after connect with the saved identity snapshot.</p>
                </div>
              </label>
            </div>
          </div>
        </div>

        <div class="rounded-2xl border bg-card/90 text-card-foreground shadow-sm" :style="panelStyle">
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
            Interface
          </p>
          <h3 class="mt-1 text-lg font-semibold">Launch and visual rhythm</h3>

          <div class="mt-5 grid gap-4">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Default Start Page
              </label>
              <select
                v-model="draft.defaultStartPage"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              >
                <option v-for="option in startPageOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
              <p class="mt-2 text-xs text-muted-foreground">
                {{ startPageOptions.find((option) => option.value === draft.defaultStartPage)?.detail }}
              </p>
            </div>

            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Density
              </label>
              <select
                v-model="draft.density"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              >
                <option v-for="option in densityOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
              <p class="mt-2 text-xs text-muted-foreground">
                {{ densityOptions.find((option) => option.value === draft.density)?.detail }}
              </p>
            </div>

            <label class="flex items-center gap-3 rounded-xl border border-border/60 bg-background/70 px-4 py-3">
              <input
                v-model="draft.reduceMotion"
                type="checkbox"
                class="h-4 w-4 rounded border border-input accent-primary"
              />
              <div>
                <p class="text-sm font-semibold text-foreground">Reduce Motion</p>
                <p class="text-xs text-muted-foreground">
                  Disable floating background motion and shorten UI transitions.
                </p>
              </div>
            </label>
          </div>

          <div class="mt-6 flex flex-wrap items-center justify-end gap-2">
            <Button variant="outline" :disabled="busy" @click="resetSettings">Restore Defaults</Button>
            <Button :disabled="busy || !dirty" @click="saveSettings">Save Settings</Button>
          </div>
        </div>
      </div>

      <div class="space-y-6">
        <div class="rounded-2xl border bg-card/90 text-card-foreground shadow-sm" :style="panelStyle">
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
            About
          </p>
          <h3 class="mt-1 text-lg font-semibold">{{ appSettings.state.about.appName }}</h3>
          <p class="mt-2 text-sm text-muted-foreground">
            Build metadata and local paths for troubleshooting, release validation, and handoff.
          </p>

          <div class="mt-5 grid gap-3 sm:grid-cols-2">
            <div
              v-for="item in aboutItems"
              :key="item.label"
              class="rounded-xl border border-border/60 bg-background/70 px-4 py-3"
            >
              <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ item.label }}
              </p>
              <p class="mt-1 text-sm font-medium text-foreground" :class="{ 'break-all': item.breakAll }">
                {{ item.value }}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
