<script setup lang="ts">
// 本文件实现 Win 前端的 `Settings` 页面。
import { computed, reactive, ref, watch } from "vue"
import CardHeader from "@/components/CardHeader.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { type AppLocale, t } from "@/i18n"
import {
  defaultAppSettings,
  densityOptions,
  normalizeAppSettings,
  startPageOptions,
  useAppSettingsStore,
  type AppSettingsState
} from "@/stores/appSettings"
import { languageOptions, useLanguageStore } from "@/stores/language"
import { useProfileStore } from "@/stores/profile"
import { useTopicBusStore } from "@/stores/topicbus"
import { useToastStore } from "@/stores/toast"

const appSettings = useAppSettingsStore()
const languageStore = useLanguageStore()
const profileStore = useProfileStore()
const topicbus = useTopicBusStore()
const toast = useToastStore()

const panelStyle = { padding: "var(--app-panel-pad)" }

type ConnectionSettingsDraft = Pick<AppSettingsState, "defaultAddr" | "defaultDeviceId" | "autoConnect" | "autoLogin">
type InterfaceSettingsDraft = Pick<AppSettingsState, "defaultStartPage" | "density" | "reduceMotion">

const defaultSettings = defaultAppSettings()
const pickConnectionSettings = (input: Partial<AppSettingsState>): ConnectionSettingsDraft => {
  const normalized = normalizeAppSettings(input)
  return {
    defaultAddr: normalized.defaultAddr,
    defaultDeviceId: normalized.defaultDeviceId,
    autoConnect: normalized.autoConnect,
    autoLogin: normalized.autoLogin
  }
}
const pickInterfaceSettings = (input: Partial<AppSettingsState>): InterfaceSettingsDraft => {
  const normalized = normalizeAppSettings(input)
  return {
    defaultStartPage: normalized.defaultStartPage,
    density: normalized.density,
    reduceMotion: normalized.reduceMotion
  }
}

const connectionDraft = reactive<ConnectionSettingsDraft>({
  defaultAddr: defaultSettings.defaultAddr,
  defaultDeviceId: defaultSettings.defaultDeviceId,
  autoConnect: defaultSettings.autoConnect,
  autoLogin: defaultSettings.autoLogin
})
const interfaceDraft = reactive<InterfaceSettingsDraft>({
  defaultStartPage: defaultSettings.defaultStartPage,
  density: defaultSettings.density,
  reduceMotion: defaultSettings.reduceMotion
})
const languageDraft = ref<AppLocale>("en")
const topicbusBusy = ref(false)
const topicbusTargetIdInput = ref("")
const topicbusMaxEventsInput = ref("500")

const syncConnectionDraft = () => {
  Object.assign(connectionDraft, pickConnectionSettings(appSettings.state.settings))
}
const syncInterfaceDraft = () => {
  Object.assign(interfaceDraft, pickInterfaceSettings(appSettings.state.settings))
}
const syncLanguageDraft = () => {
  languageDraft.value = languageStore.state.preferences.language
}

const syncTopicBusDraft = () => {
  topicbusTargetIdInput.value = topicbus.state.targetId
  topicbusMaxEventsInput.value = String(topicbus.state.maxEvents || 500)
}

const loadPage = async () => {
  await Promise.all([
    appSettings.load(),
    appSettings.loadAbout(),
    languageStore.state.loaded ? Promise.resolve(languageStore.state.preferences) : languageStore.load(),
    topicbus.loadPrefs()
  ])
  syncConnectionDraft()
  syncInterfaceDraft()
  syncLanguageDraft()
  syncTopicBusDraft()
}

const buildAppSettingsPayload = (patch: Partial<AppSettingsState>): AppSettingsState =>
  normalizeAppSettings({
    ...appSettings.state.settings,
    ...patch
  })

const connectionDirty = computed(
  () => JSON.stringify(pickConnectionSettings(connectionDraft)) !== JSON.stringify(pickConnectionSettings(appSettings.state.settings))
)
const interfaceDirty = computed(
  () => JSON.stringify(pickInterfaceSettings(interfaceDraft)) !== JSON.stringify(pickInterfaceSettings(appSettings.state.settings))
)
const languageDirty = computed(() => languageDraft.value !== languageStore.state.preferences.language)
const topicbusDirty = computed(
  () =>
    topicbusTargetIdInput.value.trim() !== topicbus.state.targetId ||
    topicbusMaxEventsInput.value.trim() !== String(topicbus.state.maxEvents || 500)
)
const dirty = computed(
  () => connectionDirty.value || interfaceDirty.value || languageDirty.value || topicbusDirty.value
)

const appSettingsBusy = computed(() => appSettings.state.loading || appSettings.state.saving)
const languageBusy = computed(() => languageStore.state.loading || languageStore.state.saving)
const restoreBusy = computed(
  () => appSettingsBusy.value || appSettings.state.aboutLoading || languageBusy.value
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
const localizedLanguageOptions = computed(() =>
  languageOptions.map((option) => ({
    ...option,
    translatedLabel: t(option.label)
  }))
)

const saveConnectionSettings = async () => {
  try {
    await appSettings.save(buildAppSettingsPayload({ ...connectionDraft }))
    syncConnectionDraft()
    toast.success(t("Connection settings saved."))
  } catch (err) {
    console.warn(err)
    await appSettings.load()
    syncConnectionDraft()
    toast.errorOf(err, t("Failed to save connection settings."))
  }
}

const saveInterfaceSettings = async () => {
  try {
    await appSettings.save(buildAppSettingsPayload({ ...interfaceDraft }))
    syncInterfaceDraft()
    toast.success(t("Interface settings saved."))
  } catch (err) {
    console.warn(err)
    await appSettings.load()
    syncInterfaceDraft()
    toast.errorOf(err, t("Failed to save interface settings."))
  }
}

const saveLanguageSettings = async () => {
  try {
    await languageStore.save(languageDraft.value)
    syncLanguageDraft()
    toast.success(t("Language settings saved."))
  } catch (err) {
    console.warn(err)
    await languageStore.load()
    syncLanguageDraft()
    toast.errorOf(err, t("Failed to save language settings."))
  }
}

const resetSettings = async () => {
  try {
    await Promise.all([appSettings.reset(), languageStore.reset()])
    syncConnectionDraft()
    syncInterfaceDraft()
    syncLanguageDraft()
    toast.info(t("Settings restored to defaults."))
  } catch (err) {
    console.warn(err)
    await Promise.allSettled([appSettings.load(), languageStore.load()])
    syncConnectionDraft()
    syncInterfaceDraft()
    syncLanguageDraft()
    toast.errorOf(err, t("Failed to restore defaults."))
  }
}

const applyTopicBusSettings = async () => {
  if (topicbusBusy.value) return
  topicbusBusy.value = true
  try {
    const rawTarget = topicbusTargetIdInput.value.trim()
    if (rawTarget) {
      const parsedTarget = Number.parseInt(rawTarget, 10)
      if (Number.isNaN(parsedTarget) || parsedTarget <= 0) {
        throw new Error(t("Target Node ID must be a positive number."))
      }
      topicbus.state.targetId = String(parsedTarget)
    } else {
      topicbus.state.targetId = ""
    }

    const rawMaxEvents = topicbusMaxEventsInput.value.trim()
    const parsedMaxEvents = Number.parseInt(rawMaxEvents || String(topicbus.state.maxEvents || 500), 10)
    if (Number.isNaN(parsedMaxEvents) || parsedMaxEvents <= 0) {
      throw new Error(t("Max events must be a positive number."))
    }
    topicbus.state.maxEvents = parsedMaxEvents
    await topicbus.savePrefs()
    syncTopicBusDraft()
    toast.success(t("TopicBus settings updated."))
  } catch (err) {
    console.warn(err)
    await topicbus.loadPrefs()
    syncTopicBusDraft()
    toast.errorOf(err, t("Failed to update TopicBus settings."))
  } finally {
    topicbusBusy.value = false
  }
}

const clearTopicBusEvents = () => {
  topicbus.clearEvents()
  toast.success(t("TopicBus cached events cleared."))
}

watch(
  () => profileStore.state.current,
  () => {
    void (async () => {
      try {
        await loadPage()
      } catch (err) {
        console.warn(err)
        toast.errorOf(err, t("Failed to load settings."))
      }
    })()
  },
  { immediate: true }
)
</script>

<template>
  <section class="grid gap-6">
    <div class="rounded-2xl border bg-card/90 text-card-foreground shadow-sm" :style="panelStyle">
      <CardHeader
        class="lg:items-center"
        :title="t('Defaults, interface, language, and build information')"
        :description="
          t(
            'Connection defaults and UI preferences are stored per profile. Language is global for the whole application. Save applies changes immediately and updates startup behavior for the next launch.'
          )
        "
        title-tag="h3"
        title-class="text-xl"
        description-class="max-w-2xl"
      >
        <template #actions>
          <div class="flex flex-wrap items-center gap-2">
            <Badge variant="secondary">{{ t("Profile: {profile}", { profile: profileStore.state.current }) }}</Badge>
            <Badge variant="muted">{{ dirty ? t("Unsaved changes") : t("Saved") }}</Badge>
            <Button variant="outline" size="sm" :disabled="restoreBusy" @click="resetSettings">
              {{ t("Restore Defaults") }}
            </Button>
          </div>
        </template>
      </CardHeader>
    </div>

    <div class="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
      <div class="space-y-6">
        <div class="rounded-2xl border bg-card/90 text-card-foreground shadow-sm" :style="panelStyle">
          <CardHeader class="items-center" :title="t('Connection and identity')" title-tag="h3" title-class="text-lg">
            <template #actions>
              <Badge :variant="connectionDraft.autoConnect || connectionDraft.autoLogin ? 'secondary' : 'muted'">
                {{
                  connectionDraft.autoConnect || connectionDraft.autoLogin
                    ? t("Automation enabled")
                    : t("Manual startup")
                }}
              </Badge>
            </template>
          </CardHeader>

          <div class="mt-5 grid gap-4">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Default Address") }}
              </label>
              <input
                v-model="connectionDraft.defaultAddr"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                placeholder="127.0.0.1:9000"
              />
            </div>

            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Default Device ID") }}
              </label>
              <input
                v-model="connectionDraft.defaultDeviceId"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                :placeholder="t('device-001')"
              />
            </div>

            <div class="grid gap-3 md:grid-cols-2">
              <label class="flex items-center gap-3 rounded-xl border border-border/60 bg-background/70 px-4 py-3">
                <input
                  v-model="connectionDraft.autoConnect"
                  type="checkbox"
                  class="h-4 w-4 rounded border border-input accent-primary"
                />
                <div>
                  <p class="text-sm font-semibold text-foreground">{{ t("Auto-connect") }}</p>
                  <p class="text-xs text-muted-foreground">{{ t("Connect to the default address on launch.") }}</p>
                </div>
              </label>

              <label class="flex items-center gap-3 rounded-xl border border-border/60 bg-background/70 px-4 py-3">
                <input
                  v-model="connectionDraft.autoLogin"
                  type="checkbox"
                  class="h-4 w-4 rounded border border-input accent-primary"
                />
                <div>
                  <p class="text-sm font-semibold text-foreground">{{ t("Auto-login") }}</p>
                  <p class="text-xs text-muted-foreground">
                    {{ t("Login after connect with the saved identity snapshot.") }}
                  </p>
                </div>
              </label>
            </div>
          </div>

          <div class="mt-6 flex flex-wrap items-center justify-end gap-2">
            <Button :disabled="appSettingsBusy || !connectionDirty" @click="saveConnectionSettings">
              {{ t("Save Connection Settings") }}
            </Button>
          </div>
        </div>

        <div class="rounded-2xl border bg-card/90 text-card-foreground shadow-sm" :style="panelStyle">
          <CardHeader :title="t('Launch and visual rhythm')" title-tag="h3" title-class="text-lg" />

          <div class="mt-5 grid gap-4">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Default Start Page") }}
              </label>
              <select
                v-model="interfaceDraft.defaultStartPage"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              >
                <option v-for="option in startPageOptions" :key="option.value" :value="option.value">
                  {{ t(option.label) }}
                </option>
              </select>
              <p class="mt-2 text-xs text-muted-foreground">
                {{
                  t(startPageOptions.find((option) => option.value === interfaceDraft.defaultStartPage)?.detail ?? "")
                }}
              </p>
            </div>

            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Density") }}
              </label>
              <select
                v-model="interfaceDraft.density"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              >
                <option v-for="option in densityOptions" :key="option.value" :value="option.value">
                  {{ t(option.label) }}
                </option>
              </select>
              <p class="mt-2 text-xs text-muted-foreground">
                {{ t(densityOptions.find((option) => option.value === interfaceDraft.density)?.detail ?? "") }}
              </p>
            </div>

            <label class="flex items-center gap-3 rounded-xl border border-border/60 bg-background/70 px-4 py-3">
              <input
                v-model="interfaceDraft.reduceMotion"
                type="checkbox"
                class="h-4 w-4 rounded border border-input accent-primary"
              />
              <div>
                <p class="text-sm font-semibold text-foreground">{{ t("Reduce Motion") }}</p>
                <p class="text-xs text-muted-foreground">
                  {{ t("Disable floating background motion and shorten UI transitions.") }}
                </p>
              </div>
            </label>
          </div>

          <div class="mt-6 flex flex-wrap items-center justify-end gap-2">
            <Button :disabled="appSettingsBusy || !interfaceDirty" @click="saveInterfaceSettings">
              {{ t("Save Interface Settings") }}
            </Button>
          </div>
        </div>

        <div class="rounded-2xl border bg-card/90 text-card-foreground shadow-sm" :style="panelStyle">
          <CardHeader :title="t('Language and regional behavior')" title-tag="h3" title-class="text-lg" />

          <div class="mt-5 grid gap-4">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Interface Language") }}
              </label>
              <select
                v-model="languageDraft"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              >
                <option v-for="option in localizedLanguageOptions" :key="option.value" :value="option.value">
                  {{ option.translatedLabel }}
                </option>
              </select>
              <p class="mt-2 text-xs text-muted-foreground">
                {{ t("Choose the display language for the entire application.") }}
              </p>
            </div>
          </div>

          <div class="mt-6 flex flex-wrap items-center justify-end gap-2">
            <Button :disabled="languageBusy || !languageDirty" @click="saveLanguageSettings">
              {{ t("Save Language Settings") }}
            </Button>
          </div>
        </div>
      </div>

      <div class="space-y-6">
        <div class="rounded-2xl border bg-card/90 text-card-foreground shadow-sm" :style="panelStyle">
          <CardHeader
            class="items-center"
            :title="t('TopicBus Settings')"
            :description="t('TopicBus windows only show events received after they open. Your own publish actions do not echo back.')"
            title-tag="h3"
            title-class="text-lg"
          >
            <template #actions>
              <Badge variant="secondary">{{ t("Target {id}", { id: topicbus.state.targetId || t("Hub NodeID") }) }}</Badge>
            </template>
          </CardHeader>

          <div class="mt-5 grid gap-4 md:grid-cols-2">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Default Target Node ID") }}
              </label>
              <input
                v-model="topicbusTargetIdInput"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                :placeholder="t('Hub NodeID')"
              />
              <p class="mt-2 text-xs text-muted-foreground">
                {{ t("Leave empty to use the current Hub NodeID.") }}
              </p>
            </div>

            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Max Events") }}
              </label>
              <input
                v-model="topicbusMaxEventsInput"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                :placeholder="t('500')"
              />
            </div>
          </div>

          <div class="mt-6 flex flex-wrap items-center justify-end gap-2">
            <Button variant="outline" :disabled="topicbusBusy" @click="clearTopicBusEvents">
              {{ t("Clear Cached") }}
            </Button>
            <Button :disabled="topicbusBusy || !topicbusDirty" @click="applyTopicBusSettings">
              {{ t("Save TopicBus Settings") }}
            </Button>
          </div>
        </div>

        <div class="rounded-2xl border bg-card/90 text-card-foreground shadow-sm" :style="panelStyle">
          <CardHeader
            :title="appSettings.state.about.appName"
            :description="t('Build metadata and local paths for troubleshooting, release validation, and handoff.')"
            title-tag="h3"
            title-class="text-lg"
          />

          <div class="mt-5 grid gap-3 sm:grid-cols-2">
            <div
              v-for="item in aboutItems"
              :key="item.label"
              class="rounded-xl border border-border/60 bg-background/70 px-4 py-3"
            >
              <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t(item.label) }}
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
