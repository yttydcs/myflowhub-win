<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue"
import {
  ArrowUp,
  ChevronRight,
  Download,
  File,
  Folder,
  FolderPlus,
  ListChecks,
  RefreshCw,
  Send,
  Settings,
  Upload
} from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import { useFileStore, type FileEntry } from "@/stores/file"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"
import FileTasks from "@/windows/FileTasks.vue"
import { CanResolveFilePaths, OnFileDrop, OnFileDropOff, ResolveFilePaths } from "../../wailsjs/runtime/runtime"

const fileStore = useFileStore()
const sessionStore = useSessionStore()
const toast = useToastStore()

const settingsOpen = ref(false)
const downloadOpen = ref(false)
const offerOpen = ref(false)
const addNodeOpen = ref(false)
const newFolderOpen = ref(false)
const tasksInlineOpen = ref(false)
const uploadInputRef = ref<HTMLInputElement | null>(null)
const fileContextMenuRef = ref<HTMLElement | null>(null)

type FileContextMenuState = {
  open: boolean
  x: number
  y: number
  entry: FileEntry | null
}

const fileContextMenu = reactive<FileContextMenuState>({
  open: false,
  x: 0,
  y: 0,
  entry: null
})

const prefsDraft = reactive({ ...fileStore.state.prefs })
const downloadForm = reactive({
  saveDir: "",
  saveName: "",
  wantHash: true
})
const offerForm = reactive({
  targetId: "",
  wantHash: true
})
const newNodeId = ref("")
const newFolderName = ref("")

const selfNodeId = computed(() => Number(sessionStore.auth.nodeId || fileStore.state.selfNodeId || 0))
const currentNodeId = computed(() => Number(fileStore.state.currentNodeId || 0))
const currentDir = computed(() => fileStore.state.currentDir)
const selected = computed(() => fileStore.state.selected)

const isLocalNode = computed(() => currentNodeId.value > 0 && currentNodeId.value === selfNodeId.value)
const hasSelection = computed(() => Boolean(selected.value))
const isDirSelected = computed(() => selected.value?.isDir ?? false)
const isFileSelected = computed(() => hasSelection.value && !isDirSelected.value)
const canDownload = computed(() => isFileSelected.value && currentNodeId.value !== selfNodeId.value)
const canOffer = computed(() => isFileSelected.value && isLocalNode.value)
const canDropImport = computed(() => isLocalNode.value && currentNodeId.value > 0)
const canCreateDir = computed(() => currentNodeId.value > 0)
const createDirButtonTitle = computed(() => {
  if (canCreateDir.value) return "Create a folder in current directory."
  return "Select a node first."
})
const upButtonTitle = computed(() => (currentDir.value ? "Go to parent directory." : "Already at root directory."))
const downloadButtonTitle = computed(() => {
  if (canDownload.value) return "Download selected remote file."
  if (!currentNodeId.value) return "Select a node first."
  if (!hasSelection.value) return "Select a file first."
  if (isDirSelected.value) return "Download works for files only."
  if (currentNodeId.value === selfNodeId.value) return "Download is for remote files only."
  return "Download unavailable."
})
const dropHintText = computed(() =>
  canDropImport.value
    ? "Drag files here to import into current directory."
    : "Switch to Local Node to enable drag import."
)
const breadcrumbItems = computed(() => {
  const parts = currentDir.value.split("/").filter(Boolean)
  const items: Array<{ label: string; dir: string }> = [{ label: "/", dir: "" }]
  let built = ""
  for (const part of parts) {
    built = built ? `${built}/${part}` : part
    items.push({ label: part, dir: built })
  }
  return items
})

const joinDir = (base: string, name: string) => {
  const clean = base ? `${base}/${name}` : name
  return clean.replace(/\\/g, "/")
}

const normalizeDirValue = (dir: string) =>
  String(dir ?? "")
    .trim()
    .replace(/\\/g, "/")
    .replace(/^\/+/, "")
    .replace(/\/+$/, "")

const refreshList = async () => {
  if (!currentNodeId.value) return
  try {
    await fileStore.requestList(currentNodeId.value, currentDir.value)
  } catch (err) {
    console.warn(err)
    fileStore.state.listing = false
    fileStore.state.listMessage = "Failed to load directory."
    toast.errorOf(err, "Failed to load directory.")
  }
}

const selectNode = async (nodeId: number) => {
  fileStore.state.currentNodeId = nodeId
  fileStore.state.currentDir = ""
  fileStore.state.entries = []
  fileStore.state.selected = null
  await refreshList()
}

const selectEntry = (entry: FileEntry) => {
  fileStore.state.selected = entry
}

const openEntry = async (entry: FileEntry) => {
  if (entry.isDir) {
    fileStore.state.currentDir = joinDir(currentDir.value, entry.name)
    await refreshList()
    return
  }
  await fileStore.openPreview(currentNodeId.value, currentDir.value, entry.name)
}

const goUp = async () => {
  const parts = currentDir.value.split("/").filter(Boolean)
  if (parts.length === 0) return
  parts.pop()
  fileStore.state.currentDir = parts.join("/")
  await refreshList()
}

const goToDir = async (dir: string) => {
  const normalized = normalizeDirValue(dir)
  if (normalized === normalizeDirValue(currentDir.value)) return
  fileStore.state.currentDir = normalized
  await refreshList()
}

const openSettings = () => {
  Object.assign(prefsDraft, fileStore.state.prefs)
  settingsOpen.value = true
}

const saveSettings = async () => {
  try {
    await fileStore.savePrefs({ ...prefsDraft })
    settingsOpen.value = false
    toast.success("File settings saved.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to save file settings.")
  }
}

const openDownloadDialog = () => {
  if (!selected.value || selected.value.isDir) return
  downloadForm.saveDir = currentDir.value
  downloadForm.saveName = selected.value.name
  downloadForm.wantHash = Boolean(fileStore.state.prefs.wantSha256)
  downloadOpen.value = true
}

const confirmDownload = async () => {
  if (!selected.value) return
  try {
    await fileStore.startPull(
      currentNodeId.value,
      currentDir.value,
      selected.value.name,
      downloadForm.saveDir,
      downloadForm.saveName,
      downloadForm.wantHash
    )
    downloadOpen.value = false
    toast.success("Download started.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to start download.")
  }
}

const openOfferDialog = () => {
  if (!selected.value || selected.value.isDir) return
  const suggestion =
    fileStore.state.nodes.find((node) => node !== selfNodeId.value) ?? 0
  offerForm.targetId = suggestion ? String(suggestion) : ""
  offerForm.wantHash = Boolean(fileStore.state.prefs.wantSha256)
  offerOpen.value = true
}

const confirmOffer = async () => {
  if (!selected.value) return
  const targetId = Number.parseInt(offerForm.targetId.trim(), 10)
  if (!targetId) {
    toast.warn("Target Node ID is required.")
    return
  }
  try {
    await fileStore.startOffer(targetId, currentDir.value, selected.value.name, offerForm.wantHash)
    offerOpen.value = false
    toast.success("Offer sent.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to send offer.")
  }
}

const openTasks = () => {
  const opened = fileStore.openTasksWindow()
  if (!opened) {
    tasksInlineOpen.value = true
  }
}

const openCreateDirDialog = () => {
  if (!canCreateDir.value) return
  newFolderName.value = ""
  newFolderOpen.value = true
}

const confirmCreateDir = async () => {
  const name = newFolderName.value.trim()
  if (!name) {
    toast.warn("Folder name is required.")
    return
  }
  if (name === "." || name === "..") {
    toast.warn("Invalid folder name.")
    return
  }
  if (/[\\/]/.test(name)) {
    toast.warn("Folder name cannot contain path separators.")
    return
  }
  try {
    await fileStore.createDir(currentNodeId.value, currentDir.value, name)
    newFolderOpen.value = false
    toast.success("Folder created.")
    await refreshList()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to create folder.")
  }
}

const importLocalPaths = async (paths: string[], errorPrefix: string) => {
  const normalized = Array.isArray(paths)
    ? paths.map((item) => String(item ?? "").trim()).filter(Boolean)
    : []
  if (!normalized.length) return
  if (!canDropImport.value) {
    toast.warn("Upload is available only when Local Node is selected.")
    return
  }
  try {
    const result = await fileStore.importLocalFiles(currentDir.value, normalized, false)
    if (result.imported.length > 0) {
      toast.success(`Imported ${result.imported.length} file(s).`)
      await refreshList()
    }
    if (result.skipped.length > 0) {
      const firstReason = result.skipped[0]?.reason ? ` (${result.skipped[0].reason})` : ""
      toast.warn(`Skipped ${result.skipped.length} item(s)${firstReason}.`)
    } else if (result.imported.length === 0) {
      toast.warn("No files were imported.")
    }
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, `${errorPrefix}.`)
  }
}

const importDroppedFiles = async (paths: string[]) => {
  await importLocalPaths(paths, "Failed to import dropped files")
}

const openUploadPicker = () => {
  if (!canDropImport.value) {
    toast.warn("Upload is available only when Local Node is selected.")
    return
  }
  uploadInputRef.value?.click()
}

const resolvePickedPaths = async (files: FileList) => {
  const list = Array.from(files || [])
  if (!list.length) return []
  const canResolve = await CanResolveFilePaths()
  if (!canResolve) {
    throw new Error("Runtime cannot resolve selected file paths.")
  }
  const resolved = await ResolveFilePaths(list)
  if (!Array.isArray(resolved)) return []
  return resolved.map((item) => String(item ?? "").trim()).filter(Boolean)
}

const onUploadInputChange = async (event: Event) => {
  const input = event.target as HTMLInputElement | null
  const files = input?.files
  if (!files || files.length === 0) return
  try {
    const paths = await resolvePickedPaths(files)
    await importLocalPaths(paths, "Failed to upload files")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to upload files.")
  } finally {
    if (input) input.value = ""
  }
}

const closeFileContextMenu = () => {
  fileContextMenu.open = false
  fileContextMenu.entry = null
}

const clampFileContextMenuToViewport = () => {
  const el = fileContextMenuRef.value
  if (!el) return
  const padding = 8
  const maxLeft = Math.max(padding, window.innerWidth - el.offsetWidth - padding)
  const maxTop = Math.max(padding, window.innerHeight - el.offsetHeight - padding)
  fileContextMenu.x = Math.min(Math.max(fileContextMenu.x, padding), maxLeft)
  fileContextMenu.y = Math.min(Math.max(fileContextMenu.y, padding), maxTop)
}

const openFileContextMenu = async (event: MouseEvent, entry?: FileEntry) => {
  event.preventDefault()
  if (entry) {
    selectEntry(entry)
    fileContextMenu.entry = entry
  } else {
    fileContextMenu.entry = selected.value
  }
  fileContextMenu.x = event.clientX
  fileContextMenu.y = event.clientY
  fileContextMenu.open = true
  await nextTick()
  clampFileContextMenuToViewport()
}

const openListContextMenu = async (event: MouseEvent) => {
  await openFileContextMenu(event)
}

const openEntryContextMenu = async (entry: FileEntry, event: MouseEvent) => {
  await openFileContextMenu(event, entry)
}

const onContextUpload = () => {
  closeFileContextMenu()
  openUploadPicker()
}

const onContextDownload = () => {
  closeFileContextMenu()
  if (!canDownload.value) {
    toast.warn(downloadButtonTitle.value)
    return
  }
  openDownloadDialog()
}

const onContextCreateFolder = () => {
  closeFileContextMenu()
  if (!canCreateDir.value) {
    toast.warn("Select a node first.")
    return
  }
  openCreateDirDialog()
}

const onContextSendRemote = () => {
  closeFileContextMenu()
  if (!canOffer.value) {
    toast.warn("Select a local file first.")
    return
  }
  openOfferDialog()
}

const onGlobalKeydown = (event: KeyboardEvent) => {
  if (!fileContextMenu.open) return
  if (event.key !== "Escape") return
  event.preventDefault()
  closeFileContextMenu()
}

const openAddNodeDialog = () => {
  newNodeId.value = ""
  addNodeOpen.value = true
}

const saveNode = async () => {
  const id = Number.parseInt(newNodeId.value.trim(), 10)
  if (!id) {
    toast.warn("Node ID must be a valid number.")
    return
  }
  try {
    await fileStore.saveNodes([...fileStore.state.nodes, id])
    addNodeOpen.value = false
    toast.success("Node saved.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to save node.")
  }
}

const removeNode = async (nodeId: number) => {
  const filtered = fileStore.state.nodes.filter((node) => node !== nodeId)
  await fileStore.saveNodes(filtered)
  if (currentNodeId.value === nodeId) {
    await selectNode(selfNodeId.value || 0)
  }
}

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId],
  async ([nodeId, hubId]) => {
    try {
      await fileStore.setIdentity(Number(nodeId), Number(hubId))
    } catch (err) {
      console.warn(err)
    }
  },
  { immediate: true }
)

onMounted(async () => {
  OnFileDrop((_x: number, _y: number, paths: string[]) => {
    void importDroppedFiles(paths)
  }, true)
  window.addEventListener("keydown", onGlobalKeydown)
  window.addEventListener("resize", closeFileContextMenu)
  window.addEventListener("scroll", closeFileContextMenu, true)
  await fileStore.loadPrefs()
  await fileStore.loadNodes()
  if (!currentNodeId.value) {
    const fallback = selfNodeId.value || fileStore.state.nodes[0] || 0
    if (fallback) {
      await selectNode(fallback)
    }
  }
  if (currentNodeId.value) {
    await refreshList()
  }
})

onBeforeUnmount(() => {
  window.removeEventListener("keydown", onGlobalKeydown)
  window.removeEventListener("resize", closeFileContextMenu)
  window.removeEventListener("scroll", closeFileContextMenu, true)
  OnFileDropOff()
})
</script>

<template>
  <section class="space-y-6">
    <input ref="uploadInputRef" type="file" class="hidden" multiple @change="onUploadInputChange" />

    <div class="grid gap-6 lg:grid-cols-[280px_minmax(0,1fr)]">
      <aside class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
        <div class="flex items-center justify-between">
          <h2 class="text-sm font-semibold">Nodes</h2>
          <Button size="sm" variant="outline" @click="openAddNodeDialog">Add</Button>
        </div>
        <div class="mt-3 space-y-2">
          <button
            v-if="selfNodeId"
            type="button"
            class="w-full rounded-xl border px-3 py-2 text-left text-sm transition"
            :class="currentNodeId === selfNodeId ? 'border-primary/60 bg-primary/10' : 'border-transparent hover:border-border/60 hover:bg-muted/60'"
            @click="selectNode(selfNodeId)"
          >
            <p class="font-semibold">Local Node</p>
            <p class="text-xs text-muted-foreground">ID {{ selfNodeId }}</p>
          </button>

          <div v-if="!fileStore.state.nodes.length" class="text-xs text-muted-foreground">
            No remote nodes saved.
          </div>

          <div v-for="node in fileStore.state.nodes" :key="node" class="flex items-center gap-2">
            <button
              type="button"
              class="flex-1 rounded-xl border px-3 py-2 text-left text-sm transition"
              :class="currentNodeId === node ? 'border-primary/60 bg-primary/10' : 'border-transparent hover:border-border/60 hover:bg-muted/60'"
              @click="selectNode(node)"
            >
              <p class="font-semibold">Remote Node</p>
              <p class="text-xs text-muted-foreground">ID {{ node }}</p>
            </button>
            <Button size="icon" variant="outline" @click="removeNode(node)">
              ✕
            </Button>
          </div>
        </div>
      </aside>

      <div class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 class="text-lg font-semibold">
              Node {{ currentNodeId || "-" }}
            </h2>
            <div class="mt-1 flex flex-wrap items-center gap-1 text-xs">
              <span class="text-[11px] font-semibold uppercase tracking-[0.24em] text-muted-foreground/80">Dir</span>
              <div class="flex flex-wrap items-center gap-1">
                <template v-for="(item, index) in breadcrumbItems" :key="item.dir || '/'">
                  <button
                    type="button"
                    class="rounded px-1 py-0.5 font-medium transition"
                    :class="
                      index === breadcrumbItems.length - 1
                        ? 'bg-primary/10 text-foreground'
                        : 'text-muted-foreground hover:bg-muted/70 hover:text-foreground'
                    "
                    :title="item.dir || '/'"
                    @click="goToDir(item.dir)"
                  >
                    {{ item.label }}
                  </button>
                  <ChevronRight v-if="index < breadcrumbItems.length - 1" class="h-3 w-3 text-muted-foreground/70" />
                </template>
              </div>
            </div>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <Button size="icon" variant="outline" title="Tasks" @click="openTasks">
              <ListChecks class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">Tasks</span>
            </Button>
            <Button size="icon" variant="outline" title="Settings" @click="openSettings">
              <Settings class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">Settings</span>
            </Button>
            <Button size="icon" variant="outline" title="Refresh directory" @click="refreshList">
              <RefreshCw class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">Refresh</span>
            </Button>
            <span class="mx-1 hidden h-6 w-px bg-border/60 sm:block" />
            <Button
              size="icon"
              variant="outline"
              :disabled="!currentDir"
              :title="upButtonTitle"
              @click="goUp"
            >
              <ArrowUp class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">Up</span>
            </Button>
            <Button
              size="icon"
              variant="outline"
              :disabled="!canCreateDir"
              :title="createDirButtonTitle"
              @click="openCreateDirDialog"
            >
              <FolderPlus class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">New Folder</span>
            </Button>
            <Button
              size="icon"
              variant="outline"
              :disabled="!canDownload"
              :title="downloadButtonTitle"
              @click="openDownloadDialog"
            >
              <Download class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">Download</span>
            </Button>
            <Button size="icon" variant="outline" :disabled="!canOffer" title="Send selected file to remote node" @click="openOfferDialog">
              <Send class="h-4 w-4" aria-hidden="true" />
              <span class="sr-only">Send to Remote</span>
            </Button>
          </div>
        </div>

        <div
          class="file-drop-zone mt-4 rounded-xl border border-border/60 bg-background/60"
          :style="{ '--wails-drop-target': 'drop' }"
        >
          <div
            class="flex items-center justify-between gap-3 border-b border-border/60 px-4 py-2 text-xs font-semibold uppercase text-muted-foreground"
          >
            <span>Directory Listing</span>
            <span class="text-[11px] font-medium normal-case tracking-normal text-muted-foreground/85">
              {{ dropHintText }}
            </span>
          </div>
          <div v-if="fileStore.state.listing" class="px-4 py-6 text-sm text-muted-foreground">
            Loading...
          </div>
          <div
            v-else-if="fileStore.state.listMessage && fileStore.state.listMessage !== 'ok'"
            class="px-4 py-6 text-sm text-rose-600"
          >
            {{ fileStore.state.listMessage }}
          </div>
          <div v-else class="max-h-[420px] overflow-y-auto" @contextmenu.prevent="openListContextMenu">
            <div
              v-for="entry in fileStore.state.entries"
              :key="entry.name"
              class="flex cursor-pointer items-center gap-3 border-b border-border/40 px-4 py-3 text-sm transition hover:bg-muted/60"
              :class="selected?.name === entry.name ? 'bg-muted/70' : ''"
              @click="selectEntry(entry)"
              @dblclick="openEntry(entry)"
              @contextmenu.prevent="openEntryContextMenu(entry, $event)"
            >
              <span
                class="flex h-8 w-8 items-center justify-center rounded-lg"
                :class="entry.isDir ? 'bg-amber-500/20 text-amber-700' : 'bg-slate-500/20 text-slate-700'"
              >
                <Folder v-if="entry.isDir" class="h-4 w-4" aria-hidden="true" />
                <File v-else class="h-4 w-4" aria-hidden="true" />
              </span>
              <div class="flex-1">
                <p class="font-medium">{{ entry.name }}</p>
                <p class="text-xs text-muted-foreground">
                  {{ entry.isDir ? "Folder" : "File" }}
                </p>
              </div>
            </div>
            <div v-if="!fileStore.state.entries.length" class="px-4 py-6 text-sm text-muted-foreground">
              No items in this directory.
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="tasksInlineOpen" class="rounded-2xl border bg-card/90 p-4 shadow-sm">
      <FileTasks />
    </div>

    <Teleport to="body">
      <div
        v-if="fileContextMenu.open"
        class="fixed inset-0 z-40"
        @pointerdown="closeFileContextMenu"
        @contextmenu.prevent="closeFileContextMenu"
      />
      <div
        v-if="fileContextMenu.open"
        ref="fileContextMenuRef"
        class="fixed z-50 w-56 rounded-xl border border-border/60 bg-card/95 p-1 text-sm shadow-xl backdrop-blur"
        role="menu"
        aria-label="File actions"
        :style="{ left: `${fileContextMenu.x}px`, top: `${fileContextMenu.y}px` }"
        @pointerdown.stop
        @click.stop
        @contextmenu.prevent
      >
        <button
          type="button"
          class="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left hover:bg-muted/60 disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="!canDropImport"
          role="menuitem"
          @click="onContextUpload"
        >
          <Upload class="h-4 w-4" aria-hidden="true" />
          Upload
        </button>
        <button
          type="button"
          class="mt-1 flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left hover:bg-muted/60 disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="!canDownload"
          role="menuitem"
          @click="onContextDownload"
        >
          <Download class="h-4 w-4" aria-hidden="true" />
          Download
        </button>
        <button
          type="button"
          class="mt-1 flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left hover:bg-muted/60 disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="!canCreateDir"
          role="menuitem"
          @click="onContextCreateFolder"
        >
          <FolderPlus class="h-4 w-4" aria-hidden="true" />
          New Folder
        </button>
        <button
          type="button"
          class="mt-1 flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left hover:bg-muted/60 disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="!canOffer"
          role="menuitem"
          @click="onContextSendRemote"
        >
          <Send class="h-4 w-4" aria-hidden="true" />
          Send to Remote
        </button>
      </div>
    </Teleport>

    <Overlay :open="settingsOpen" @close="settingsOpen = false">
      <div class="w-full max-w-xl rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">File Settings</h2>
        <div class="mt-4 grid gap-4">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Base Dir
            </label>
            <input
              v-model="prefsDraft.baseDir"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Max Size (bytes)
              </label>
              <input
                v-model.number="prefsDraft.maxSizeBytes"
                type="number"
                min="0"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Max Concurrent
              </label>
              <input
                v-model.number="prefsDraft.maxConcurrent"
                type="number"
                min="1"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Chunk Bytes
              </label>
              <input
                v-model.number="prefsDraft.chunkBytes"
                type="number"
                min="4096"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Incomplete TTL (sec)
              </label>
              <input
                v-model.number="prefsDraft.incompleteTtlSec"
                type="number"
                min="60"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              />
            </div>
          </div>
          <label class="flex items-center gap-2 text-sm text-muted-foreground">
            <input v-model="prefsDraft.wantSha256" type="checkbox" class="h-4 w-4 rounded border" />
            Request SHA256 for transfers
          </label>
          <label class="flex items-center gap-2 text-sm text-muted-foreground">
            <input v-model="prefsDraft.autoAccept" type="checkbox" class="h-4 w-4 rounded border" />
            Auto-accept incoming offers
          </label>
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="settingsOpen = false">Cancel</Button>
          <Button @click="saveSettings">Save</Button>
        </div>
      </div>
    </Overlay>

    <Overlay :open="downloadOpen" @close="downloadOpen = false">
      <div class="w-full max-w-lg rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">Download File</h2>
        <div class="mt-4 space-y-3 text-sm text-muted-foreground">
          <p>Remote file: {{ currentDir || "/" }}/{{ selected?.name }}</p>
        </div>
        <div class="mt-4 grid gap-3">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Save Dir (relative)
            </label>
            <input
              v-model="downloadForm.saveDir"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Save Name
            </label>
            <input
              v-model="downloadForm.saveName"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>
          <label class="flex items-center gap-2 text-sm text-muted-foreground">
            <input v-model="downloadForm.wantHash" type="checkbox" class="h-4 w-4 rounded border" />
            Request SHA256
          </label>
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="downloadOpen = false">Cancel</Button>
          <Button @click="confirmDownload">Start</Button>
        </div>
      </div>
    </Overlay>

    <Overlay :open="offerOpen" @close="offerOpen = false">
      <div class="w-full max-w-lg rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">Send Offer</h2>
        <div class="mt-4 space-y-3 text-sm text-muted-foreground">
          <p>Local file: {{ currentDir || "/" }}/{{ selected?.name }}</p>
        </div>
        <div class="mt-4 grid gap-3">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Target Node ID
            </label>
            <input
              v-model="offerForm.targetId"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>
          <label class="flex items-center gap-2 text-sm text-muted-foreground">
            <input v-model="offerForm.wantHash" type="checkbox" class="h-4 w-4 rounded border" />
            Include SHA256
          </label>
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="offerOpen = false">Cancel</Button>
          <Button @click="confirmOffer">Send</Button>
        </div>
      </div>
    </Overlay>

    <Overlay :open="addNodeOpen" @close="addNodeOpen = false">
      <div class="w-full max-w-md rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">Add Remote Node</h2>
        <div class="mt-4">
          <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            Node ID
          </label>
          <input
            v-model="newNodeId"
            class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
          />
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="addNodeOpen = false">Cancel</Button>
          <Button @click="saveNode">Save</Button>
        </div>
      </div>
    </Overlay>

    <Overlay :open="newFolderOpen" @close="newFolderOpen = false">
      <div class="w-full max-w-md rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">New Folder</h2>
        <p class="mt-2 text-sm text-muted-foreground">
          Current dir: {{ currentDir || "/" }}
        </p>
        <div class="mt-4">
          <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            Folder Name
          </label>
          <input
            v-model="newFolderName"
            class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            @keydown.enter.prevent="confirmCreateDir"
          />
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="newFolderOpen = false">Cancel</Button>
          <Button @click="confirmCreateDir">Create</Button>
        </div>
      </div>
    </Overlay>

    <Overlay :open="fileStore.state.previewOpen" @close="fileStore.closePreview">
      <div class="w-full max-w-3xl rounded-2xl border bg-card/95 p-6 shadow-xl">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold">
            Preview {{ fileStore.state.previewTarget?.name }}
          </h2>
          <Button variant="outline" @click="fileStore.closePreview">Close</Button>
        </div>
        <p class="mt-2 text-xs text-muted-foreground">{{ fileStore.state.previewInfo }}</p>
        <pre
          class="mt-4 max-h-[60vh] overflow-y-auto rounded-lg border border-border/60 bg-background/80 p-4 text-xs text-foreground"
        >{{ fileStore.state.previewLoading ? "Loading..." : fileStore.state.previewText }}</pre>
      </div>
    </Overlay>
  </section>
</template>

<style scoped>
.file-drop-zone {
  transition: border-color 0.2s ease, background-color 0.2s ease, box-shadow 0.2s ease;
}

.file-drop-zone.wails-drop-target-active {
  border-color: hsl(var(--primary) / 0.45);
  background-color: hsl(var(--primary) / 0.08);
  box-shadow: inset 0 0 0 1px hsl(var(--primary) / 0.25);
}
</style>
