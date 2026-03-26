<script setup lang="ts">
import { computed, ref, watch, watchEffect, type Component } from "vue"
import { RouterLink, RouterView, useRoute } from "vue-router"
import {
  Bug,
  ClipboardList,
  Database,
  FileText,
  Folder,
  Home as HomeIcon,
  KeyRound,
  LayoutDashboard,
  ListChecks,
  Menu,
  Network,
  Rss,
  Settings2,
  Server,
  Share2,
  ShieldCheck
} from "lucide-vue-next"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { t, useI18n } from "@/i18n"
import { Overlay } from "@/components/ui/overlay"
import { Tooltip } from "@/components/ui/tooltip"
import ToastHost from "@/components/ToastHost.vue"
import { useAppSettingsStore } from "@/stores/appSettings"
import { useFileStore } from "@/stores/file"
import { useLanguageStore } from "@/stores/language"
import { useProfileStore } from "@/stores/profile"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"
import { useVarPoolStore } from "@/stores/varpool"

type NavItem = {
  label: string
  description: string
  to: string
  icon: Component
  tone: string
}

const route = useRoute()

const fileStore = useFileStore()
const appSettings = useAppSettingsStore()
const languageStore = useLanguageStore()
const profileStore = useProfileStore()
const sessionStore = useSessionStore()
const toast = useToastStore()
const varpool = useVarPoolStore()
const { locale } = useI18n()

const isWindowLayout = computed(() => route.meta.layout === "window")
const isFullBleedWindow = computed(() => route.meta.windowMode === "full-bleed")
const windowMainClass = computed(() =>
  isFullBleedWindow.value
    ? "relative h-screen overflow-hidden p-0"
    : "relative min-h-screen overflow-y-auto"
)
const windowViewClass = computed(() =>
  isFullBleedWindow.value
    ? "h-full animate-in fade-in slide-in-from-bottom-2 duration-500"
    : "animate-in fade-in slide-in-from-bottom-2 duration-500"
)
const windowMainStyle = computed(() =>
  isFullBleedWindow.value
    ? undefined
    : { padding: "var(--app-shell-window-py) var(--app-shell-window-px)" }
)
const sidebarStyle = {
  paddingLeft: "var(--app-shell-sidebar-px)",
  paddingRight: "var(--app-shell-sidebar-px)",
  paddingTop: "var(--app-shell-sidebar-pt)",
  paddingBottom: "var(--app-shell-sidebar-pb)",
  scrollbarGutter: "stable"
}
const shellMainStyle = {
  padding: "var(--app-shell-main-py) var(--app-shell-main-px)",
  gap: "var(--app-shell-section-gap)"
}
const headerStyle = {
  padding: "var(--app-shell-header-py) var(--app-shell-header-px)"
}
const navItemStyle = {
  padding: "var(--app-nav-item-py) var(--app-nav-item-px)"
}
const collapsedNavItemStyle = {
  padding: "var(--app-nav-item-py)"
}

let varpoolStorageEpoch = 0
const loadVarPoolStorage = async () => {
  const myEpoch = ++varpoolStorageEpoch
  try {
    await varpool.loadWatchList()
    await varpool.loadSubPrefs()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to load VarPool settings."))
  }
  if (varpoolStorageEpoch !== myEpoch) return

  const connected = Boolean(sessionStore.connected)
  const loggedIn = Boolean(sessionStore.auth.loggedIn)
  const nodeId = Number(sessionStore.auth.nodeId || 0)
  const hubId = Number(sessionStore.auth.hubId || 0)
  varpool.setIdentity(nodeId, hubId)
  if (connected && loggedIn && nodeId > 0 && hubId > 0) {
    void restoreVarPoolSubs()
  }
}

let varpoolRestoreEpoch = 0
const restoreVarPoolSubs = async () => {
  const myEpoch = ++varpoolRestoreEpoch
  try {
    const result = await varpool.restoreDesiredSubscriptions({ concurrency: 4 })
    if (varpoolRestoreEpoch !== myEpoch) return
    if (result.attempted && result.failed) {
      toast.warn(
        t("VarPool auto-subscribe incomplete."),
        t("{failed}/{attempted} failed.", { failed: result.failed, attempted: result.attempted })
      )
    }
  } catch (err) {
    if (varpoolRestoreEpoch !== myEpoch) return
    console.warn(err)
  }
}

watch(
  () => profileStore.state.current,
  () => {
    void (async () => {
      try {
        await appSettings.load()
      } catch (err) {
        console.warn(err)
        toast.errorOf(err, t("Failed to load app settings."))
      }
    })()
  },
  { immediate: true }
)

if (!languageStore.state.loaded && !languageStore.state.loading) {
  void languageStore.load().catch((err) => {
    console.warn(err)
    toast.errorOf(err, t("Failed to load language preference."))
  })
}

watch(
  () => profileStore.state.current,
  () => {
    void loadVarPoolStorage()
  },
  { immediate: true }
)

watch(
  () => [sessionStore.connected, sessionStore.auth.loggedIn, sessionStore.auth.nodeId, sessionStore.auth.hubId],
  ([connected, loggedIn, nodeId, hubId]) => {
    varpool.setIdentity(Number(nodeId || 0), Number(hubId || 0))
    if (connected && loggedIn && Number(nodeId || 0) > 0 && Number(hubId || 0) > 0) {
      void restoreVarPoolSubs()
    }
  }
)

const statusDotClass = computed(() => {
  if (sessionStore.connected) return "bg-emerald-500"
  return "bg-rose-500"
})

const headerStatusText = computed(() => {
  if (sessionStore.connected) {
    return sessionStore.addr
      ? t("Connected to {addr}", { addr: sessionStore.addr })
      : t("Connected")
  }
  if (sessionStore.lastError) {
    return t("Disconnected / Last error: {error}", { error: sessionStore.lastError })
  }
  return t("Disconnected")
})

const navGroups = ref<{ title: string; items: NavItem[] }[]>([
  {
    title: "Session",
    items: [
      {
        label: "Home",
        description: "Session, auth, and status",
        to: "/home",
        icon: HomeIcon,
        tone: "bg-sky-500/15 text-sky-700"
      },
      {
        label: "Local Hub",
        description: "Download and run hub_server",
        to: "/local-hub",
        icon: Server,
        tone: "bg-emerald-500/15 text-emerald-700"
      }
    ]
  },
  {
    title: "Signals",
    items: [
      {
        label: "VarPool",
        description: "Values and subscriptions",
        to: "/varpool",
        icon: Database,
        tone: "bg-emerald-500/15 text-emerald-700"
      },
      {
        label: "TopicBus",
        description: "Publish and replay",
        to: "/topicbus",
        icon: Rss,
        tone: "bg-cyan-500/15 text-cyan-700"
      },
      {
        label: "Showcase",
        description: "Compose screens",
        to: "/showcase",
        icon: LayoutDashboard,
        tone: "bg-fuchsia-500/15 text-fuchsia-700"
      }
    ]
  },
  {
    title: "Operations",
    items: [
      {
        label: "Devices",
        description: "Query nodes/devices",
        to: "/devices",
        icon: Network,
        tone: "bg-violet-500/15 text-violet-700"
      },
      {
        label: "File Console",
        description: "Browse and transfer",
        to: "/file",
        icon: Folder,
        tone: "bg-amber-500/15 text-amber-700"
      },
      {
        label: "Flow",
        description: "Design and deploy",
        to: "/flow",
        icon: Share2,
        tone: "bg-indigo-500/15 text-indigo-700"
      }
    ]
  },
  {
    title: "Authority",
    items: [
      {
        label: "Access Policy",
        description: "Roles and permissions",
        to: "/access-policy",
        icon: ShieldCheck,
        tone: "bg-emerald-500/15 text-emerald-700"
      },
      {
        label: "Registration Approvals",
        description: "Pending registrations",
        to: "/registration-approvals",
        icon: ClipboardList,
        tone: "bg-sky-500/15 text-sky-700"
      },
      {
        label: "Permit Issuance",
        description: "Issue and revoke permits",
        to: "/permit-issuance",
        icon: KeyRound,
        tone: "bg-amber-500/15 text-amber-700"
      }
    ]
  },
  {
    title: "Tools",
    items: [
      {
        label: "Debug",
        description: "Custom frames",
        to: "/debug",
        icon: Bug,
        tone: "bg-rose-500/15 text-rose-700"
      },
      {
        label: "Presets",
        description: "Stress patterns",
        to: "/presets",
        icon: ListChecks,
        tone: "bg-orange-500/15 text-orange-700"
      },
      {
        label: "Logs",
        description: "Live stream",
        to: "/logs",
        icon: FileText,
        tone: "bg-stone-500/15 text-stone-700"
      }
    ]
  },
  {
    title: "Other",
    items: [
      {
        label: "Settings",
        description: "Defaults, UI, language, and about",
        to: "/settings",
        icon: Settings2,
        tone: "bg-slate-500/15 text-slate-700"
      }
    ]
  }
])

const flatNav = computed(() => navGroups.value.flatMap((group) => group.items))

const profileState = computed(() => profileStore.state)
const profiles = computed(() => profileStore.state.profiles)
const selectedProfile = computed({
  get: () => profileStore.state.current,
  set: (value) => {
    void profileStore.setProfile(value)
  }
})

const pageTitle = computed(() => t((route.meta.title as string) ?? "Module"))
const pageSubtitle = computed(
  () =>
    t(
      (route.meta.subtitle as string) ??
        "This module will be wired to backend services in upcoming tasks."
    )
)
const showModuleCard = computed(() => route.name === "home")

const createProfile = async () => {
  closeProfileMenu()
  const name = window.prompt(t("New profile name"))
  if (!name) return
  await profileStore.setProfile(name)
}

const profileMenuOpen = ref(false)
const sidebarCollapsed = ref(false)

const sidebarGridClass = computed(() =>
  sidebarCollapsed.value
    ? "relative grid h-screen grid-cols-1 lg:grid-cols-[96px_minmax(0,1fr)]"
    : "relative grid h-screen grid-cols-1 lg:grid-cols-[260px_minmax(0,1fr)]"
)
const sidebarAsideClass = computed(() =>
  sidebarCollapsed.value
    ? "hidden h-screen overflow-x-hidden overflow-y-auto border-r border-border/60 bg-background/80 px-3 pb-6 pt-6 shadow-sm backdrop-blur lg:flex lg:flex-col"
    : "hidden h-screen overflow-x-hidden overflow-y-auto border-r border-border/60 bg-background/80 px-5 pb-6 pt-8 shadow-sm backdrop-blur lg:flex lg:flex-col"
)
const sidebarBrandClass = computed(() =>
  sidebarCollapsed.value ? "flex flex-col items-center gap-2" : "flex items-center gap-3"
)
const sidebarBrandMarkClass = computed(() =>
  sidebarCollapsed.value
    ? "flex h-10 w-10 items-center justify-center rounded-xl bg-primary text-xs font-semibold text-primary-foreground shadow-sm"
    : "flex h-12 w-12 items-center justify-center rounded-2xl bg-primary text-sm font-semibold text-primary-foreground shadow-sm"
)
const sidebarToggleLabel = computed(() =>
  sidebarCollapsed.value ? t("Expand sidebar") : t("Collapse sidebar")
)
const sidebarNavClass = computed(() =>
  sidebarCollapsed.value ? "mt-6 space-y-4" : "mt-6 space-y-6"
)
const sidebarGroupClass = computed(() => (sidebarCollapsed.value ? "space-y-2" : "space-y-3"))
const sidebarNavListClass = computed(() => (sidebarCollapsed.value ? "space-y-3" : "space-y-2"))
const sidebarNavItemBaseClass = computed(() =>
  sidebarCollapsed.value
    ? "group flex items-center justify-center rounded-xl border transition"
    : "group flex items-center gap-3 rounded-xl border px-3 py-2.5 transition"
)
const sidebarNavIconClass = computed(() =>
  sidebarCollapsed.value
    ? "flex h-10 w-10 shrink-0 items-center justify-center rounded-xl"
    : "flex h-9 w-9 shrink-0 items-center justify-center rounded-lg"
)
const navItemStateClass = (isActive: boolean) => {
  if (isActive) {
    return sidebarCollapsed.value
      ? "border-transparent bg-transparent text-primary shadow-none"
      : "border-primary/40 bg-primary/10 text-foreground shadow-sm"
  }
  return sidebarCollapsed.value
    ? "border-transparent text-muted-foreground hover:bg-muted/60"
    : "border-transparent hover:border-border/60 hover:bg-muted/70"
}
const navItemIconToneClass = (isActive: boolean, item: NavItem) => {
  if (isActive) {
    return sidebarCollapsed.value ? "bg-primary/15 text-primary" : "bg-primary text-primary-foreground"
  }
  return item.tone
}

const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

const toggleProfileMenu = () => {
  profileMenuOpen.value = !profileMenuOpen.value
}
const closeProfileMenu = () => {
  profileMenuOpen.value = false
}
const selectProfile = async (name: string) => {
  await profileStore.setProfile(name)
  closeProfileMenu()
}

watchEffect(() => {
  if (typeof document === "undefined") return
  void locale.value
  document.title = `${pageTitle.value} - MyFlowHub`
})
</script>

<template>
  <div class="app-surface min-h-screen text-foreground">
    <ToastHost />
    <div v-if="isWindowLayout" class="relative min-h-screen">
      <div class="relative min-h-screen overflow-hidden">
        <div class="pointer-events-none absolute inset-0 overflow-hidden">
          <div
            class="absolute -top-24 left-10 h-64 w-64 animate-float rounded-full bg-sky-200/50 blur-3xl"
          />
          <div
            class="absolute right-0 top-32 h-72 w-72 animate-float-slow rounded-full bg-amber-200/60 blur-3xl"
          />
        </div>

        <main :class="windowMainClass" :style="windowMainStyle">
          <RouterView v-slot="{ Component }">
            <component
              :is="Component"
              :key="route.fullPath"
              :class="windowViewClass"
            />
          </RouterView>
        </main>
      </div>
    </div>
    <div v-else class="relative h-screen overflow-hidden">
      <div class="pointer-events-none absolute inset-0 overflow-hidden">
        <div
          class="absolute -top-24 left-10 h-64 w-64 animate-float rounded-full bg-sky-200/60 blur-3xl"
        />
        <div
          class="absolute right-0 top-32 h-72 w-72 animate-float-slow rounded-full bg-amber-200/70 blur-3xl"
        />
      </div>

      <Overlay :open="Boolean(fileStore.state.offer)" @close="fileStore.rejectOffer">
        <div
          v-if="fileStore.state.offer"
          class="w-full max-w-lg rounded-2xl border border-border/60 bg-card/95 p-6 shadow-xl"
        >
          <div class="flex items-start justify-between gap-4">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
                {{ t("Incoming Transfer") }}
              </p>
              <h2 class="mt-1 text-lg font-semibold">{{ t("Accept file offer?") }}</h2>
            </div>
            <Badge variant="secondary">{{ t("Provider {provider}", { provider: fileStore.state.offer.provider }) }}</Badge>
          </div>

          <div class="mt-4 space-y-2 text-sm text-muted-foreground">
            <p>
              <span class="font-semibold text-foreground">{{ t("File:") }}</span>
              {{ fileStore.state.offer.name }}
            </p>
            <p>
              <span class="font-semibold text-foreground">{{ t("Remote Dir:") }}</span>
              {{ fileStore.state.offer.dir || "/" }}
            </p>
            <p>
              <span class="font-semibold text-foreground">{{ t("Size:") }}</span>
              {{ t("{count} bytes", { count: fileStore.state.offer.size }) }}
            </p>
            <p v-if="fileStore.state.offer.sha256">
              <span class="font-semibold text-foreground">{{ t("SHA256:") }}</span>
              {{ fileStore.state.offer.sha256 }}
            </p>
          </div>

          <div class="mt-4">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Save Directory (relative to base dir)") }}
            </label>
            <input
              v-model="fileStore.state.offerSaveDir"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              :placeholder="t('Optional, defaults to remote dir')"
            />
          </div>

          <div class="mt-6 flex justify-end gap-2">
            <Button variant="outline" @click="fileStore.rejectOffer">{{ t("Reject") }}</Button>
            <Button @click="fileStore.acceptOffer">{{ t("Accept") }}</Button>
          </div>
        </div>
      </Overlay>

      <div :class="sidebarGridClass">
        <aside :class="sidebarAsideClass" :style="sidebarStyle">
          <div :class="sidebarBrandClass">
            <div :class="sidebarBrandMarkClass">MH</div>
            <div v-if="!sidebarCollapsed">
              <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
                MyFlowHub
              </p>
              <h1 class="text-lg font-semibold">{{ t("Tool Console") }}</h1>
            </div>
          </div>

          <nav :class="sidebarNavClass">
            <div v-for="group in navGroups" :key="group.title" :class="sidebarGroupClass">
              <p
                v-if="!sidebarCollapsed"
                class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground"
              >
                {{ t(group.title) }}
              </p>
              <div :class="sidebarNavListClass">
                <RouterLink v-for="item in group.items" :key="item.to" :to="item.to" v-slot="{ isActive }">
                  <Tooltip v-if="sidebarCollapsed" :content="t(item.label)" side="right">
                    <div
                      :class="[sidebarNavItemBaseClass, navItemStateClass(isActive)]"
                      :style="collapsedNavItemStyle"
                    >
                      <div
                        :class="[
                          sidebarNavIconClass,
                          navItemIconToneClass(isActive, item)
                        ]"
                      >
                        <component :is="item.icon" class="h-5 w-5" aria-hidden="true" />
                      </div>
                      <span class="sr-only">{{ t(item.label) }}</span>
                    </div>
                  </Tooltip>
                  <div
                    v-else
                    :class="[sidebarNavItemBaseClass, navItemStateClass(isActive)]"
                    :style="navItemStyle"
                  >
                    <div
                      :class="[
                        sidebarNavIconClass,
                        navItemIconToneClass(isActive, item)
                      ]"
                    >
                      <component :is="item.icon" class="h-5 w-5" aria-hidden="true" />
                    </div>
                    <div>
                      <p class="text-sm font-medium">{{ t(item.label) }}</p>
                      <p class="text-xs text-muted-foreground">{{ t(item.description) }}</p>
                    </div>
                  </div>
                </RouterLink>
              </div>
            </div>
          </nav>
        </aside>

        <div class="flex h-screen flex-col overflow-hidden">
          <header class="flex-none border-b border-border/60 bg-background/85 backdrop-blur" :style="headerStyle">
            <div class="flex flex-wrap items-center justify-between gap-4">
              <div class="flex items-center gap-3">
                <Button
                  as="button"
                  type="button"
                  size="icon"
                  variant="ghost"
                  class="hidden lg:inline-flex"
                  :aria-label="sidebarToggleLabel"
                  :title="sidebarToggleLabel"
                  @click="toggleSidebar"
                >
                  <Menu class="h-5 w-5" aria-hidden="true" />
                  <span class="sr-only">{{ sidebarToggleLabel }}</span>
                </Button>
                <div class="flex items-center gap-2 rounded-full bg-muted/70 px-3 py-1 text-xs font-semibold text-foreground shadow-sm">
                  <span :class="['h-2 w-2 rounded-full', statusDotClass]" />
                  <span class="text-xs font-semibold text-muted-foreground">
                    {{ headerStatusText }}
                  </span>
                </div>
              </div>

              <div class="flex flex-wrap items-center gap-2">
                <div class="hidden text-right sm:block">
                  <p class="text-[11px] font-semibold uppercase tracking-[0.3em] text-muted-foreground">
                    {{ t("Active profile") }}
                  </p>
                  <p class="text-sm font-semibold">{{ selectedProfile }}</p>
                </div>
                <div class="relative">
                  <Button variant="outline" size="sm" @click="toggleProfileMenu">
                    {{ selectedProfile }}
                  </Button>
                  <Overlay
                    :open="profileMenuOpen"
                    :teleport="false"
                    overlayClass="bg-transparent p-0"
                    zIndexClass="z-20"
                    closeOnBackdrop
                    @close="closeProfileMenu"
                  />
                  <div
                    v-if="profileMenuOpen"
                    class="absolute right-0 top-11 z-30 w-52 rounded-xl border bg-card/95 p-2 text-sm shadow-xl"
                  >
                    <p class="px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.3em] text-muted-foreground">
                      {{ t("Profiles") }}
                    </p>
                    <button
                      v-for="profile in profiles"
                      :key="profile"
                      type="button"
                      class="mt-1 w-full rounded-lg px-3 py-2 text-left text-sm transition hover:bg-muted/70"
                      @click="selectProfile(profile)"
                    >
                      <span class="font-semibold text-foreground">{{ profile }}</span>
                    </button>
                    <div class="my-2 h-px bg-border/60" />
                    <button
                      type="button"
                      class="w-full rounded-lg px-3 py-2 text-left text-sm font-semibold text-primary transition hover:bg-primary/10"
                      @click="createProfile"
                    >
                      {{ t("New profile") }}
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <div class="mt-4 flex gap-2 overflow-x-auto pb-1 lg:hidden">
              <RouterLink v-for="item in flatNav" :key="item.to" :to="item.to" v-slot="{ isActive }">
                <span
                  :class="[
                    'flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-semibold transition',
                    isActive ? 'border-primary/40 bg-primary/10 text-primary' : 'border-border/60 text-muted-foreground'
                  ]"
                >
                  <span
                    :class="['h-2 w-2 rounded-full', isActive ? 'bg-primary' : 'bg-muted-foreground/60']"
                  />
                  {{ t(item.label) }}
                </span>
              </RouterLink>
            </div>

          </header>

          <main class="flex flex-1 flex-col overflow-y-auto" :style="shellMainStyle">
            <section v-if="showModuleCard" class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
              <div class="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                <div>
                  <h2 class="text-xl font-semibold">{{ pageTitle }}</h2>
                </div>
                <div class="flex flex-wrap items-center gap-2">
                  <Badge variant="secondary">{{ t("Profile: {profile}", { profile: selectedProfile }) }}</Badge>
                  <Badge variant="muted">{{ profileState.keysPath ? t("Keys ready") : t("Keys pending") }}</Badge>
                </div>
              </div>
              <div v-if="profileState.keysPath" class="mt-4 grid gap-3 text-xs text-muted-foreground md:grid-cols-3">
                <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
                  <p class="font-semibold text-foreground">{{ t("Base Dir") }}</p>
                  <p class="break-all">{{ profileState.baseDir }}</p>
                </div>
                <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
                  <p class="font-semibold text-foreground">{{ t("Settings") }}</p>
                  <p class="break-all">{{ profileState.settingsPath }}</p>
                </div>
                <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
                  <p class="font-semibold text-foreground">{{ t("Keys") }}</p>
                  <p class="break-all">{{ profileState.keysPath }}</p>
                </div>
              </div>
            </section>

            <RouterView v-slot="{ Component }">
              <component
                :is="Component"
                :key="route.fullPath"
                class="animate-in fade-in slide-in-from-bottom-2 duration-500"
              />
            </RouterView>
          </main>
        </div>
      </div>
    </div>
  </div>
</template>
