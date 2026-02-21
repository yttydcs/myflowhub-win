<script setup lang="ts">
import { computed, ref } from "vue"
import { RouterLink, RouterView, useRoute } from "vue-router"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useFileStore } from "@/stores/file"
import { useProfileStore } from "@/stores/profile"
import { useSessionStore } from "@/stores/session"

type NavItem = {
  label: string
  description: string
  to: string
  short: string
  tone: string
}

const route = useRoute()

const fileStore = useFileStore()
const profileStore = useProfileStore()
const sessionStore = useSessionStore()

const isWindowLayout = computed(() => route.meta.layout === "window")

const statusDotClass = computed(() => {
  if (sessionStore.connected) return "bg-emerald-500"
  return "bg-rose-500"
})

const statusLabel = computed(() => {
  if (sessionStore.connected) return "Connected"
  return "Disconnected"
})

const connectionDetail = computed(() => {
  if (sessionStore.connected) {
    return sessionStore.addr ? `Connected to ${sessionStore.addr}` : "Connected"
  }
  if (sessionStore.lastError) {
    return `Last error: ${sessionStore.lastError}`
  }
  return "Awaiting session handshake."
})

const headerStatusText = computed(() => {
  if (sessionStore.connected) {
    return sessionStore.addr ? `Connected to ${sessionStore.addr}` : "Connected"
  }
  if (sessionStore.lastError) {
    return `Disconnected / Last error: ${sessionStore.lastError}`
  }
  return "Disconnected"
})

const navGroups = ref<{ title: string; items: NavItem[] }[]>([
  {
    title: "Session",
    items: [
      {
        label: "Home",
        description: "Session, auth, and status",
        to: "/home",
        short: "HM",
        tone: "bg-sky-500/15 text-sky-700"
      },
      {
        label: "Devices",
        description: "Query nodes/devices",
        to: "/devices",
        short: "DV",
        tone: "bg-violet-500/15 text-violet-700"
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
        short: "VP",
        tone: "bg-emerald-500/15 text-emerald-700"
      },
      {
        label: "TopicBus",
        description: "Publish and replay",
        to: "/topicbus",
        short: "TB",
        tone: "bg-cyan-500/15 text-cyan-700"
      }
    ]
  },
  {
    title: "Operations",
    items: [
      {
        label: "File Console",
        description: "Browse and transfer",
        to: "/file",
        short: "FL",
        tone: "bg-amber-500/15 text-amber-700"
      },
      {
        label: "Flow",
        description: "Design and deploy",
        to: "/flow",
        short: "FW",
        tone: "bg-indigo-500/15 text-indigo-700"
      },
      {
        label: "Management",
        description: "Nodes and config",
        to: "/management",
        short: "MG",
        tone: "bg-slate-500/15 text-slate-700"
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
        short: "DB",
        tone: "bg-rose-500/15 text-rose-700"
      },
      {
        label: "Presets",
        description: "Stress patterns",
        to: "/presets",
        short: "PR",
        tone: "bg-orange-500/15 text-orange-700"
      },
      {
        label: "Logs",
        description: "Live stream",
        to: "/logs",
        short: "LG",
        tone: "bg-stone-500/15 text-stone-700"
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

const pageTitle = computed(() => (route.meta.title as string) ?? "Module")
const pageSubtitle = computed(
  () =>
    (route.meta.subtitle as string) ??
    "This module will be wired to backend services in upcoming tasks."
)
const showModuleCard = computed(() => route.name === "home")

const createProfile = async () => {
  closeProfileMenu()
  const name = window.prompt("New profile name")
  if (!name) return
  await profileStore.setProfile(name)
}

const profileMenuOpen = ref(false)
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
</script>

<template>
  <div class="app-surface min-h-screen text-foreground">
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

        <main class="relative min-h-screen overflow-y-auto px-6 py-6">
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
    <div v-else class="relative h-screen overflow-hidden">
      <div class="pointer-events-none absolute inset-0 overflow-hidden">
        <div
          class="absolute -top-24 left-10 h-64 w-64 animate-float rounded-full bg-sky-200/60 blur-3xl"
        />
        <div
          class="absolute right-0 top-32 h-72 w-72 animate-float-slow rounded-full bg-amber-200/70 blur-3xl"
        />
      </div>

      <div
        v-if="fileStore.state.offer"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-6"
      >
        <div class="w-full max-w-lg rounded-2xl border border-border/60 bg-card/95 p-6 shadow-xl">
          <div class="flex items-start justify-between gap-4">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
                Incoming Transfer
              </p>
              <h2 class="mt-1 text-lg font-semibold">Accept file offer?</h2>
            </div>
            <Badge variant="secondary">Provider {{ fileStore.state.offer.provider }}</Badge>
          </div>

          <div class="mt-4 space-y-2 text-sm text-muted-foreground">
            <p>
              <span class="font-semibold text-foreground">File:</span>
              {{ fileStore.state.offer.name }}
            </p>
            <p>
              <span class="font-semibold text-foreground">Remote Dir:</span>
              {{ fileStore.state.offer.dir || "/" }}
            </p>
            <p>
              <span class="font-semibold text-foreground">Size:</span>
              {{ fileStore.state.offer.size }} bytes
            </p>
            <p v-if="fileStore.state.offer.sha256">
              <span class="font-semibold text-foreground">SHA256:</span>
              {{ fileStore.state.offer.sha256 }}
            </p>
          </div>

          <div class="mt-4">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Save Directory (relative to base dir)
            </label>
            <input
              v-model="fileStore.state.offerSaveDir"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              placeholder="Optional, defaults to remote dir"
            />
          </div>

          <div class="mt-6 flex justify-end gap-2">
            <Button variant="outline" @click="fileStore.rejectOffer">Reject</Button>
            <Button @click="fileStore.acceptOffer">Accept</Button>
          </div>
        </div>
      </div>

      <div class="relative grid h-screen grid-cols-1 lg:grid-cols-[260px_minmax(0,1fr)]">
        <aside
          class="hidden h-screen flex-col overflow-hidden border-r border-border/60 bg-background/80 px-5 pb-6 pt-8 shadow-sm backdrop-blur lg:flex"
        >
          <div class="flex items-center gap-3">
            <div
              class="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary text-sm font-semibold text-primary-foreground shadow-sm"
            >
              MH
            </div>
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
                MyFlowHub
              </p>
              <h1 class="text-lg font-semibold">Tool Console</h1>
            </div>
          </div>

          <div class="mt-6 rounded-2xl border bg-card/80 p-4 text-card-foreground shadow-sm">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
                Session
              </span>
              <div class="flex items-center gap-2 text-xs text-muted-foreground">
                <span :class="['h-2 w-2 rounded-full', statusDotClass]" />
                <span>{{ statusLabel }}</span>
              </div>
            </div>
            <p class="mt-2 text-sm font-medium">Console ready</p>
            <p class="text-xs text-muted-foreground">{{ connectionDetail }}</p>
          </div>

          <nav class="mt-6 flex-1 space-y-6 overflow-y-auto pr-1">
            <div v-for="group in navGroups" :key="group.title" class="space-y-3">
              <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
                {{ group.title }}
              </p>
              <div class="space-y-2">
                <RouterLink v-for="item in group.items" :key="item.to" :to="item.to" v-slot="{ isActive }">
                  <div
                    :class="[
                      'group flex items-center gap-3 rounded-xl border px-3 py-2.5 transition',
                      isActive
                        ? 'border-primary/40 bg-primary/10 text-foreground shadow-sm'
                        : 'border-transparent hover:border-border/60 hover:bg-muted/70'
                    ]"
                  >
                    <div
                      :class="[
                        'flex h-9 w-9 items-center justify-center rounded-lg text-[11px] font-semibold uppercase',
                        isActive ? 'bg-primary text-primary-foreground' : item.tone
                      ]"
                    >
                      {{ item.short }}
                    </div>
                    <div>
                      <p class="text-sm font-medium">{{ item.label }}</p>
                      <p class="text-xs text-muted-foreground">{{ item.description }}</p>
                    </div>
                  </div>
                </RouterLink>
              </div>
            </div>
          </nav>
        </aside>

        <div class="flex h-screen flex-col overflow-hidden">
          <header class="flex-none border-b border-border/60 bg-background/85 px-6 py-4 backdrop-blur">
            <div class="flex flex-wrap items-center justify-between gap-4">
              <div class="flex items-center gap-3">
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
                    Active profile
                  </p>
                  <p class="text-sm font-semibold">{{ selectedProfile }}</p>
                </div>
                <div class="relative">
                  <Button variant="outline" size="sm" @click="toggleProfileMenu">
                    {{ selectedProfile }}
                  </Button>
                  <div
                    v-if="profileMenuOpen"
                    class="fixed inset-0 z-20"
                    @click="closeProfileMenu"
                  />
                  <div
                    v-if="profileMenuOpen"
                    class="absolute right-0 top-11 z-30 w-52 rounded-xl border bg-card/95 p-2 text-sm shadow-xl"
                  >
                    <p class="px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.3em] text-muted-foreground">
                      Profiles
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
                      New profile
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
                  {{ item.label }}
                </span>
              </RouterLink>
            </div>

            <p v-if="profileStore.state.message" class="mt-2 text-xs text-muted-foreground">
              {{ profileStore.state.message }}
            </p>
          </header>

          <main class="flex-1 space-y-6 overflow-y-auto px-6 py-8">
            <section v-if="showModuleCard" class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
              <div class="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                <div>
                  <h2 class="text-xl font-semibold">{{ pageTitle }}</h2>
                </div>
                <div class="flex flex-wrap items-center gap-2">
                  <Badge variant="secondary">Profile: {{ selectedProfile }}</Badge>
                  <Badge variant="muted">{{ profileState.keysPath ? "Keys ready" : "Keys pending" }}</Badge>
                </div>
              </div>
              <div v-if="profileState.keysPath" class="mt-4 grid gap-3 text-xs text-muted-foreground md:grid-cols-3">
                <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
                  <p class="font-semibold text-foreground">Base Dir</p>
                  <p class="break-all">{{ profileState.baseDir }}</p>
                </div>
                <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
                  <p class="font-semibold text-foreground">Settings</p>
                  <p class="break-all">{{ profileState.settingsPath }}</p>
                </div>
                <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
                  <p class="font-semibold text-foreground">Keys</p>
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
