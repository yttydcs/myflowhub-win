<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { Copy, ExternalLink, PencilLine, Plus, Trash2 } from "lucide-vue-next"
import CardHeader from "@/components/CardHeader.vue"
import PageHero from "@/components/PageHero.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import { useI18n } from "@/i18n"
import { openAuxWindow } from "@/lib/auxWindow"
import { useProfileStore } from "@/stores/profile"
import { useShowcaseStore, type ShowcaseScreenSummary } from "@/stores/showcase"
import { useToastStore } from "@/stores/toast"

const profileStore = useProfileStore()
const showcase = useShowcaseStore()
const toast = useToastStore()
const { t } = useI18n()

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

const layoutLabel = (mode: ShowcaseScreenSummary["layoutMode"]) => (mode === "canvas_percent" ? t("Canvas") : t("Columns"))

const loadCenter = async () => {
  if (busy.value) return
  busy.value = true
  try {
    await showcase.load()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to load showcase screens."))
  } finally {
    busy.value = false
  }
}

const openWindow = async (screenId: string, kind: "editor" | "viewer") => {
  const route =
    kind === "editor"
      ? `#/showcase-editor-window?screenId=${encodeURIComponent(screenId)}`
      : `#/showcase-window?screenId=${encodeURIComponent(screenId)}`
  const namePrefix = kind === "editor" ? "showcase_editor" : "showcase_viewer"
  const size = kind === "editor" ? "width=1580,height=980" : "width=980,height=720"
  const result = await openAuxWindow({
    routePath: route,
    name: `${namePrefix}_${screenId}_${Date.now()}`,
    size
  })
  if (result === "blocked") {
    toast.warn(
      kind === "editor"
        ? t("Editor window was blocked by browser popup policy.")
        : t("Viewer window was blocked by browser popup policy.")
    )
  }
}

const openCreateDialog = () => {
  createForm.name = ""
  createDialogOpen.value = true
}

const createBlank = async () => {
  const name = createForm.name.trim()
  if (!name) {
    toast.error(t("Screen name is required."))
    return
  }
  if (busy.value) return
  busy.value = true
  try {
    const created = await showcase.createScreen(name)
    createDialogOpen.value = false
    toast.success(t("Screen created."))
    if (created) {
      openWindow(created.id, "editor")
    }
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to create screen."))
  } finally {
    busy.value = false
  }
}

const duplicateScreen = async (summary: ShowcaseScreenSummary) => {
  if (busy.value) return
  busy.value = true
  try {
    const duplicated = await showcase.duplicateScreen(summary.id)
    toast.success(t("Screen duplicated."))
    if (duplicated) {
      openWindow(duplicated.id, "editor")
    }
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to duplicate screen."))
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
    toast.error(t("Screen name is required."))
    return
  }
  if (busy.value) return
  busy.value = true
  try {
    await showcase.renameScreen(renameDialog.screenId, name)
    renameDialog.open = false
    toast.success(t("Screen renamed."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to rename screen."))
  } finally {
    busy.value = false
  }
}

const deleteScreen = async (summary: ShowcaseScreenSummary) => {
  const ok = window.confirm(t("Delete screen '{name}'?", { name: summary.name }))
  if (!ok || busy.value) return
  busy.value = true
  try {
    await showcase.deleteScreen(summary.id)
    toast.success(t("Screen deleted."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to delete screen."))
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
    <PageHero
      :description="
        t('Browse reusable screens, open a dedicated editor window, or launch a clean viewer window for runtime display.')
      "
    >
      <template #actions>
        <div class="flex flex-wrap items-center gap-2">
          <Badge variant="outline">{{ t("{count} screens", { count: screens.length }) }}</Badge>
          <Badge variant="secondary">{{ t("Last Sync {time}", { time: formatTimestamp(showcase.state.lastLoadedAt) }) }}</Badge>
          <Button :disabled="busy" @click="openCreateDialog">
            <Plus class="mr-2 h-4 w-4" />
            {{ t("New Blank") }}
          </Button>
        </div>
      </template>
    </PageHero>

    <section class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
      <CardHeader class="items-center" :title="t('Screens')" title-class="text-lg" />

      <div class="mt-5 space-y-3">
        <article
          v-for="screen in screens"
          :key="screen.id"
          class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3 shadow-sm transition hover:border-primary/50"
        >
          <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div class="min-w-0 flex flex-wrap items-center gap-2 text-sm lg:flex-1 lg:flex-nowrap">
              <p class="truncate font-semibold">{{ screen.name }}</p>
              <Badge v-if="screen.isCurrent" variant="outline">{{ t("Current") }}</Badge>
              <Badge variant="secondary">{{ layoutLabel(screen.layoutMode) }}</Badge>
              <Badge variant="outline">{{ t("{count} widgets", { count: screen.widgetCount }) }}</Badge>
              <span class="text-xs text-muted-foreground">{{ formatTimestamp(screen.updatedAt) }}</span>
            </div>

            <div class="flex flex-wrap items-center gap-2">
              <Button size="sm" variant="outline" @click="openWindow(screen.id, 'editor')">
                <PencilLine class="mr-1 h-4 w-4" />
                {{ t("Edit") }}
              </Button>
              <Button size="sm" variant="outline" @click="openWindow(screen.id, 'viewer')">
                <ExternalLink class="mr-1 h-4 w-4" />
                {{ t("View") }}
              </Button>
              <Button size="sm" variant="outline" @click="duplicateScreen(screen)">
                <Copy class="mr-1 h-4 w-4" />
                {{ t("Duplicate") }}
              </Button>
              <Button size="sm" variant="outline" @click="openRenameDialog(screen)">
                {{ t("Rename") }}
              </Button>
              <Button size="sm" variant="outline" @click="deleteScreen(screen)">
                <Trash2 class="mr-1 h-4 w-4" />
                {{ t("Delete") }}
              </Button>
            </div>
          </div>
        </article>
      </div>
    </section>

    <Overlay :open="createDialogOpen" @close="createDialogOpen = false">
      <div class="w-full max-w-lg rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">{{ t("Create Blank Screen") }}</h2>
        <p class="mt-2 text-sm text-muted-foreground">
          {{ t("Start with an empty screen, then open the dedicated editor window to add widgets and arrange layout.") }}
        </p>
        <div class="mt-5">
          <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Screen Name") }}</label>
          <input v-model="createForm.name" :class="inputClass" :placeholder="t('Factory overview')" />
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="createDialogOpen = false">{{ t("Cancel") }}</Button>
          <Button :disabled="busy" @click="createBlank">{{ t("Create") }}</Button>
        </div>
      </div>
    </Overlay>

    <Overlay :open="renameDialog.open" @close="renameDialog.open = false">
      <div class="w-full max-w-lg rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">{{ t("Rename Screen") }}</h2>
        <div class="mt-5">
          <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Screen Name") }}</label>
          <input v-model="renameDialog.name" :class="inputClass" />
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="renameDialog.open = false">{{ t("Cancel") }}</Button>
          <Button :disabled="busy" @click="saveRename">{{ t("Save") }}</Button>
        </div>
      </div>
    </Overlay>
  </section>
</template>
