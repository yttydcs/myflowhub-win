<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { Copy, ExternalLink, PencilLine, Plus, Trash2 } from "lucide-vue-next"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import { useProfileStore } from "@/stores/profile"
import { useShowcaseStore, type ShowcaseScreenSummary } from "@/stores/showcase"
import { useToastStore } from "@/stores/toast"

const profileStore = useProfileStore()
const showcase = useShowcaseStore()
const toast = useToastStore()

const busy = ref(false)
const createDialogOpen = ref(false)
const renameDialog = reactive({
  open: false,
  screenId: "",
  name: ""
})

const inputClass =
  "mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

const screens = computed(() =>
  showcase
    .listScreenSummaries()
    .slice()
    .sort((left, right) => Date.parse(right.updatedAt || "") - Date.parse(left.updatedAt || ""))
)

const createForm = reactive({
  name: ""
})

const formatTimestamp = (value: string) => {
  const raw = String(value ?? "").trim()
  if (!raw) return "-"
  const parsed = Date.parse(raw)
  if (Number.isNaN(parsed)) return raw
  return new Date(parsed).toLocaleString()
}

const layoutLabel = (mode: ShowcaseScreenSummary["layoutMode"]) => (mode === "canvas_percent" ? "Canvas" : "Columns")

const loadCenter = async () => {
  if (busy.value) return
  busy.value = true
  try {
    await showcase.load()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to load showcase screens.")
  } finally {
    busy.value = false
  }
}

const openWindow = (screenId: string, kind: "editor" | "viewer") => {
  const base = window.location.href.split("#")[0]
  const route =
    kind === "editor"
      ? `#/showcase-editor-window?screenId=${encodeURIComponent(screenId)}`
      : `#/showcase-window?screenId=${encodeURIComponent(screenId)}`
  const namePrefix = kind === "editor" ? "showcase_editor" : "showcase_viewer"
  const size = kind === "editor" ? "width=1580,height=980" : "width=980,height=720"
  const win = window.open(`${base}${route}`, `${namePrefix}_${screenId}_${Date.now()}`, size)
  if (win) {
    win.focus()
  } else {
    toast.warn(kind === "editor" ? "Editor window was blocked by browser popup policy." : "Viewer window was blocked by browser popup policy.")
  }
}

const openCreateDialog = () => {
  createForm.name = ""
  createDialogOpen.value = true
}

const createBlank = async () => {
  const name = createForm.name.trim()
  if (!name) {
    toast.error("Screen name is required.")
    return
  }
  if (busy.value) return
  busy.value = true
  try {
    const created = await showcase.createScreen(name)
    createDialogOpen.value = false
    toast.success("Screen created.")
    if (created) {
      openWindow(created.id, "editor")
    }
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to create screen.")
  } finally {
    busy.value = false
  }
}

const duplicateScreen = async (summary: ShowcaseScreenSummary) => {
  if (busy.value) return
  busy.value = true
  try {
    const duplicated = await showcase.duplicateScreen(summary.id)
    toast.success("Screen duplicated.")
    if (duplicated) {
      openWindow(duplicated.id, "editor")
    }
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to duplicate screen.")
  } finally {
    busy.value = false
  }
}

const openRenameDialog = (summary: ShowcaseScreenSummary) => {
  renameDialog.open = true
  renameDialog.screenId = summary.id
  renameDialog.name = summary.name
}

const saveRename = async () => {
  const name = renameDialog.name.trim()
  if (!name) {
    toast.error("Screen name is required.")
    return
  }
  if (busy.value) return
  busy.value = true
  try {
    await showcase.renameScreen(renameDialog.screenId, name)
    renameDialog.open = false
    toast.success("Screen renamed.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to rename screen.")
  } finally {
    busy.value = false
  }
}

const deleteScreen = async (summary: ShowcaseScreenSummary) => {
  const ok = window.confirm(`Delete screen '${summary.name}'?`)
  if (!ok || busy.value) return
  busy.value = true
  try {
    await showcase.deleteScreen(summary.id)
    toast.success("Screen deleted.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to delete screen.")
  } finally {
    busy.value = false
  }
}

watch(
  () => profileStore.state.current,
  async () => {
    await loadCenter()
  }
)

onMounted(async () => {
  await loadCenter()
})
</script>

<template>
  <section class="space-y-6">
    <section class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">Workspace</p>
          <h1 class="mt-2 text-2xl font-semibold">Showcase Center</h1>
          <p class="mt-2 max-w-2xl text-sm text-muted-foreground">
            Browse reusable screens, open a dedicated editor window, or launch a clean viewer window for runtime display.
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <Badge variant="outline">{{ screens.length }} screens</Badge>
          <Badge variant="secondary">Last Sync {{ formatTimestamp(showcase.state.lastLoadedAt) }}</Badge>
          <Button :disabled="busy" @click="openCreateDialog">
            <Plus class="mr-2 h-4 w-4" />
            New Blank
          </Button>
        </div>
      </div>
    </section>

    <section class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Library</p>
          <h2 class="mt-1 text-lg font-semibold">Screens</h2>
        </div>
      </div>

      <div class="mt-5 space-y-3">
        <article
          v-for="screen in screens"
          :key="screen.id"
          class="rounded-2xl border border-border/60 bg-background/70 p-4 shadow-sm transition hover:border-primary/50"
        >
          <div class="flex flex-wrap items-start justify-between gap-4">
            <div class="min-w-0 space-y-2">
              <div class="flex flex-wrap items-center gap-2">
                <p class="text-base font-semibold">{{ screen.name }}</p>
                <Badge v-if="screen.isCurrent" variant="outline">Current</Badge>
              </div>
              <div class="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                <Badge variant="secondary">{{ layoutLabel(screen.layoutMode) }}</Badge>
                <Badge variant="outline">{{ screen.widgetCount }} widgets</Badge>
                <span>Updated {{ formatTimestamp(screen.updatedAt) }}</span>
              </div>
              <p class="text-xs text-muted-foreground">screen_id {{ screen.id }}</p>
            </div>

            <div class="flex flex-wrap items-center gap-2">
              <Button size="sm" variant="outline" @click="openWindow(screen.id, 'editor')">
                <PencilLine class="mr-1 h-4 w-4" />
                Edit
              </Button>
              <Button size="sm" variant="outline" @click="openWindow(screen.id, 'viewer')">
                <ExternalLink class="mr-1 h-4 w-4" />
                View
              </Button>
              <Button size="sm" variant="outline" @click="duplicateScreen(screen)">
                <Copy class="mr-1 h-4 w-4" />
                Duplicate
              </Button>
              <Button size="sm" variant="outline" @click="openRenameDialog(screen)">
                Rename
              </Button>
              <Button size="sm" variant="outline" @click="deleteScreen(screen)">
                <Trash2 class="mr-1 h-4 w-4" />
                Delete
              </Button>
            </div>
          </div>
        </article>
      </div>
    </section>

    <Overlay :open="createDialogOpen" @close="createDialogOpen = false">
      <div class="w-full max-w-lg rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">Create Blank Screen</h2>
        <p class="mt-2 text-sm text-muted-foreground">
          Start with an empty screen, then open the dedicated editor window to add widgets and arrange layout.
        </p>
        <div class="mt-5">
          <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Screen Name</label>
          <input v-model="createForm.name" :class="inputClass" placeholder="Factory overview" />
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="createDialogOpen = false">Cancel</Button>
          <Button :disabled="busy" @click="createBlank">Create</Button>
        </div>
      </div>
    </Overlay>

    <Overlay :open="renameDialog.open" @close="renameDialog.open = false">
      <div class="w-full max-w-lg rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">Rename Screen</h2>
        <div class="mt-5">
          <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Screen Name</label>
          <input v-model="renameDialog.name" :class="inputClass" />
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="renameDialog.open = false">Cancel</Button>
          <Button :disabled="busy" @click="saveRename">Save</Button>
        </div>
      </div>
    </Overlay>
  </section>
</template>
