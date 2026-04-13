<script setup lang="ts">
// Context: renders the node vars dialog helper used by the VarPool page.
import { computed, ref, watch } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import { useI18n } from "@/i18n"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"
import { useVarPoolStore } from "@/stores/varpool"

interface Props {
  open: boolean
  ownerId?: number
}

const props = withDefaults(defineProps<Props>(), {
  ownerId: 0
})

const emit = defineEmits<{
  (e: "close"): void
}>()

const sessionStore = useSessionStore()
const toast = useToastStore()
const varpool = useVarPoolStore()
const { t } = useI18n()

const loading = ref(false)
const ownerText = ref("")
const query = ref("")
const names = ref<string[]>([])
const errorText = ref("")
const page = ref(1)
const pageSize = 50

const inputClass =
  "mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

const normalizeOwner = (value: string) => {
  const trimmed = value.trim()
  if (!trimmed) {
    throw new Error(t("Owner NodeID is required."))
  }
  const parsed = Number.parseInt(trimmed, 10)
  if (Number.isNaN(parsed) || parsed <= 0) {
    throw new Error(t("Owner NodeID must be a positive number."))
  }
  return parsed
}

const currentOwnerId = computed(() => {
  try {
    return normalizeOwner(ownerText.value)
  } catch {
    return 0
  }
})

const watchedIds = computed(() => {
  const out = new Set<string>()
  for (const key of varpool.state.keys) {
    const name = String(key?.name ?? "").trim()
    const owner = Number(key?.owner ?? 0) || 0
    if (!name || owner <= 0) continue
    out.add(`${name}#${owner}`)
  }
  return out
})

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return names.value
  return names.value.filter((name) => name.toLowerCase().includes(q))
})

const pageCount = computed(() => {
  const total = filtered.value.length
  return Math.max(1, Math.ceil(total / pageSize))
})

const pageSafe = computed(() => Math.min(Math.max(1, page.value), pageCount.value))

const paged = computed(() => {
  const p = pageSafe.value
  const start = (p - 1) * pageSize
  return filtered.value.slice(start, start + pageSize)
})

watch(
  () => props.open,
  (open) => {
    if (!open) return
    ownerText.value = props.ownerId ? String(props.ownerId) : ""
    query.value = ""
    names.value = []
    errorText.value = ""
    page.value = 1
  }
)

watch(
  () => props.ownerId,
  (ownerId) => {
    if (!props.open) return
    ownerText.value = ownerId ? String(ownerId) : ownerText.value
  }
)

watch(pageCount, (count) => {
  if (page.value > count) {
    page.value = count
  }
})

const load = async () => {
  if (loading.value) return
  loading.value = true
  errorText.value = ""
  try {
    if (!sessionStore.connected) {
      throw new Error(t("Connect before listing node variables."))
    }
    const sourceId = Number(sessionStore.auth.nodeId || 0)
    const hubId = Number(sessionStore.auth.hubId || 0)
    if (!sourceId) {
      throw new Error(t("Login required to list node variables."))
    }
    varpool.setIdentity(sourceId, hubId)

    const ownerId = normalizeOwner(ownerText.value)
    const list = await varpool.listOwnerNames(ownerId)
    names.value = Array.isArray(list) ? list : []
    errorText.value = ""
    page.value = 1
  } catch (err) {
    console.warn(err)
    const message = err instanceof Error ? err.message : String(err)
    errorText.value = message || t("Unknown error.")
    toast.errorOf(err, t("Failed to load node variables."))
  } finally {
    loading.value = false
  }
}

const addWatch = async (name: string) => {
  const ownerId = normalizeOwner(ownerText.value)
  const id = `${name}#${ownerId}`
  if (watchedIds.value.has(id)) {
    toast.info(t("Already watched."), id)
    return
  }
  loading.value = true
  try {
    await varpool.addWatchKey({ name, owner: ownerId })
    void varpool.getVar({ name, owner: ownerId }).catch(() => {})
    toast.success(t("Watch added."), id)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to add watch."))
  } finally {
    loading.value = false
  }
}

const onClose = () => emit("close")

const canLoad = computed(() => Boolean(props.open && currentOwnerId.value > 0 && !loading.value))
</script>

<template>
  <Overlay :open="props.open" overlayClass="bg-black/40 p-4" closeOnBackdrop @close="onClose">
    <div
      data-node-vars-dialog
      class="flex max-h-[85vh] w-full max-w-3xl flex-col overflow-hidden rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl"
    >
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">{{ t("VarPool") }}</p>
          <h2 class="mt-1 text-lg font-semibold">{{ t("Node Variables") }}</h2>
          <p class="text-sm text-muted-foreground">
            {{ t("Lists public variables on a node. Click Add Watch to include it in your watched list.") }}
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <Badge variant="secondary">{{ t("Owner {owner}", { owner: currentOwnerId || "-" }) }}</Badge>
          <Button size="sm" variant="outline" @click="onClose">{{ t("Close") }}</Button>
        </div>
      </div>

      <div data-node-vars-scroll class="mt-5 min-h-0 flex-1 overflow-y-auto">
        <div class="space-y-4 px-1 py-1 pr-2">
          <div class="grid gap-4 md:grid-cols-[1fr_auto]">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Owner NodeID") }}
              </label>
              <input v-model="ownerText" :class="inputClass" :placeholder="t('Node ID')" @keydown.enter.prevent="load" />
            </div>
            <div class="flex items-end gap-2">
              <Button :disabled="!canLoad" @click="load">{{ loading ? t("Loading...") : t("Load") }}</Button>
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-[1fr_auto]">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Search") }}
              </label>
              <input v-model="query" :class="inputClass" :placeholder="t('Filter by name...')" />
            </div>
            <div class="flex items-end justify-end gap-2 text-sm text-muted-foreground">
              <span>{{ t("{count} results", { count: filtered.length }) }}</span>
              <span>·</span>
              <span>{{ t("Page {page} / {total}", { page: pageSafe, total: pageCount }) }}</span>
            </div>
          </div>

          <div>
            <div v-if="errorText" class="rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-700">
              {{ errorText }}
            </div>

            <div v-else class="overflow-hidden rounded-xl border border-border/60">
              <div
                v-for="name in paged"
                :key="name"
                class="flex items-center justify-between gap-3 border-b border-border/50 bg-background/70 px-4 py-3 text-sm last:border-b-0"
              >
                <div class="min-w-0">
                  <p class="truncate font-mono text-[12px]">{{ name }}</p>
                  <p class="text-xs text-muted-foreground">
                    {{ watchedIds.has(`${name}#${currentOwnerId}`) ? t("Already watched") : t("Not watched") }}
                  </p>
                </div>
                <div class="flex items-center gap-2">
                  <Badge
                    v-if="watchedIds.has(`${name}#${currentOwnerId}`)"
                    variant="secondary"
                  >
                    {{ t("Watched") }}
                  </Badge>
                  <Button
                    size="sm"
                    variant="outline"
                    :disabled="loading || watchedIds.has(`${name}#${currentOwnerId}`) || !currentOwnerId"
                    @click="addWatch(name)"
                  >
                    {{ t("Add Watch") }}
                  </Button>
                </div>
              </div>

              <div v-if="paged.length === 0" class="px-4 py-6 text-sm text-muted-foreground">
                {{ names.length ? t("No matches.") : t("Load a node to view variables.") }}
              </div>
            </div>

            <div class="mt-4 flex items-center justify-between">
              <Button
                size="sm"
                variant="outline"
                :disabled="loading || pageSafe <= 1"
                @click="page = pageSafe - 1"
              >
                {{ t("Prev") }}
              </Button>
              <div class="text-xs text-muted-foreground">
                {{ t("Showing {shown} of {total}", { shown: paged.length, total: filtered.length }) }}
              </div>
              <Button
                size="sm"
                variant="outline"
                :disabled="loading || pageSafe >= pageCount"
                @click="page = pageSafe + 1"
              >
                {{ t("Next") }}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Overlay>
</template>

