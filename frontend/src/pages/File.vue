<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import { useFileStore } from "@/stores/file"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"
import FileTasks from "@/windows/FileTasks.vue"

const fileStore = useFileStore()
const sessionStore = useSessionStore()
const toast = useToastStore()

const settingsOpen = ref(false)
const downloadOpen = ref(false)
const offerOpen = ref(false)
const addNodeOpen = ref(false)
const tasksInlineOpen = ref(false)

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

const joinDir = (base: string, name: string) => {
  const clean = base ? `${base}/${name}` : name
  return clean.replace(/\\/g, "/")
}

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

const selectEntry = (entry: any) => {
  fileStore.state.selected = entry
}

const openEntry = async (entry: any) => {
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
</script>

<template>
  <section class="space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
          File Console
        </p>
        <h1 class="text-2xl font-semibold">File Browser</h1>
        <p class="text-sm text-muted-foreground">
          Browse nodes and manage transfers with resume + optional hashing.
        </p>
      </div>
      <div class="flex flex-wrap gap-2">
        <Button variant="outline" @click="openTasks">Tasks</Button>
        <Button variant="outline" @click="openSettings">Settings</Button>
        <Button variant="outline" @click="refreshList">Refresh</Button>
      </div>
    </div>

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
            <p class="text-xs text-muted-foreground">
              Dir: {{ currentDir || "/" }}
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <Button size="sm" variant="outline" :disabled="!currentDir" @click="goUp">Up</Button>
            <Button size="sm" variant="outline" :disabled="!canDownload" @click="openDownloadDialog">
              Download
            </Button>
            <Button size="sm" variant="outline" :disabled="!canOffer" @click="openOfferDialog">
              Offer
            </Button>
          </div>
        </div>

        <div class="mt-4 rounded-xl border border-border/60 bg-background/60">
          <div class="border-b border-border/60 px-4 py-2 text-xs font-semibold uppercase text-muted-foreground">
            Directory Listing
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
          <div v-else class="max-h-[420px] overflow-y-auto">
            <div
              v-for="entry in fileStore.state.entries"
              :key="entry.name"
              class="flex cursor-pointer items-center gap-3 border-b border-border/40 px-4 py-3 text-sm transition hover:bg-muted/60"
              :class="selected?.name === entry.name ? 'bg-muted/70' : ''"
              @click="selectEntry(entry)"
              @dblclick="openEntry(entry)"
            >
              <span
                class="flex h-8 w-8 items-center justify-center rounded-lg text-[11px] font-semibold uppercase"
                :class="entry.isDir ? 'bg-amber-500/20 text-amber-700' : 'bg-slate-500/20 text-slate-700'"
              >
                {{ entry.isDir ? "DIR" : "FILE" }}
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
