<script setup lang="ts">
// 本文件实现 File 页面使用的 `OfferNodeTreePicker` 组件。
import { computed, reactive, ref, watch } from "vue"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"

type WailsBinding = (...args: any[]) => Promise<any>

type NodeWire = {
  node_id?: number
  nodeId?: number
  has_children?: boolean
  hasChildren?: boolean
}

type ListNodesWire = {
  code?: number
  msg?: string
  nodes?: NodeWire[]
}

type OfferTreeNode = {
  key: string
  nodeId: number
  hasChildrenHint: boolean
  duplicate: boolean
  expanded: boolean
  loading: boolean
  error: string
  children: OfferTreeNode[] | null
}

const props = defineProps<{
  modelValue: number
  sourceId: number
  hubId: number
  excludeNodeId?: number
}>()

const emit = defineEmits<{
  (e: "update:modelValue", value: number): void
}>()
const { t } = useI18n()

const rootTargetId = ref("")
const root = ref<OfferTreeNode | null>(null)
const rootLoading = ref(false)
const rootError = ref("")

let epoch = 0
let seenNodeIDs = new Set<number>()
const nodeIndex = new Map<string, OfferTreeNode>()

const callMgmt = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.management?.ManagementService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(t("Management binding '{method}' unavailable", { method }))
  }
  return fn(...args)
}

const toErrorMessage = (err: unknown) => {
  if (!err) return t("Unknown error.")
  if (err instanceof Error) return err.message || t("Unknown error.")
  return String(err)
}

const ensureIdentity = () => {
  if (!props.sourceId) {
    throw new Error(t("Login required to query devices."))
  }
  if (!props.hubId) {
    throw new Error(t("Hub ID missing."))
  }
}

const resolveRootTarget = () => {
  const raw = rootTargetId.value.trim()
  if (!raw) {
    rootTargetId.value = String(props.hubId || "")
    return Number(props.hubId || 0)
  }
  const parsed = Number.parseInt(raw, 10)
  if (Number.isNaN(parsed) || parsed <= 0) {
    throw new Error(t("Root node must be a positive number."))
  }
  return parsed
}

const normalizeNodes = (resp: ListNodesWire) => {
  const list = Array.isArray(resp?.nodes) ? resp.nodes : []
  return list
    .map((node) => ({
      nodeId: Number(node?.node_id ?? node?.nodeId ?? 0),
      hasChildren: Boolean(node?.has_children ?? node?.hasChildren ?? false)
    }))
    .filter((node) => node.nodeId > 0)
    .sort((a, b) => a.nodeId - b.nodeId)
}

const clearTree = () => {
  root.value = null
  rootError.value = ""
  rootLoading.value = false
  nodeIndex.clear()
  seenNodeIDs = new Set<number>()
}

const makeChildKey = (parentKey: string, nodeId: number) => {
  const base = `${parentKey}/${nodeId}`
  let key = base
  let suffix = 1
  while (nodeIndex.has(key)) {
    suffix++
    key = `${base}@${suffix}`
  }
  return key
}

const registerNode = (node: OfferTreeNode) => {
  nodeIndex.set(node.key, node)
}

const buildChildNodes = (parentKey: string, parentID: number, children: Array<{ nodeId: number; hasChildren: boolean }>) => {
  const list: OfferTreeNode[] = []
  for (const child of children) {
    if (!child?.nodeId || child.nodeId === parentID) continue
    const duplicate = seenNodeIDs.has(child.nodeId)
    if (!duplicate) {
      seenNodeIDs.add(child.nodeId)
    }
    const node = reactive<OfferTreeNode>({
      key: makeChildKey(parentKey, child.nodeId),
      nodeId: child.nodeId,
      hasChildrenHint: Boolean(child.hasChildren),
      duplicate,
      expanded: false,
      loading: false,
      error: "",
      children: null
    }) as OfferTreeNode
    registerNode(node)
    list.push(node)
  }
  return list
}

const loadChildren = async (node: OfferTreeNode, myEpoch: number, force: boolean) => {
  if (node.duplicate) return
  if (node.loading) return
  if (!force && node.children !== null) return

  node.loading = true
  node.error = ""
  try {
    const resp = await callMgmt<ListNodesWire>("ListNodesSimple", props.sourceId, node.nodeId)
    if (epoch !== myEpoch) return
    const code = Number(resp?.code ?? 0)
    if (code !== 1) {
      throw new Error(String(resp?.msg ?? `code=${code}`))
    }
    const children = normalizeNodes(resp).filter((entry) => entry.nodeId !== node.nodeId)
    node.children = buildChildNodes(node.key, node.nodeId, children)
    node.hasChildrenHint = node.children.length > 0
    node.loading = false
  } catch (err) {
    if (epoch !== myEpoch) return
    node.loading = false
    node.error = toErrorMessage(err)
    node.children = null
  }
}

const loadRoot = async () => {
  const myEpoch = ++epoch
  clearTree()
  rootLoading.value = true
  rootError.value = ""

  try {
    ensureIdentity()
    const rootID = resolveRootTarget()
    if (!rootID) {
      throw new Error(t("Root node is required."))
    }

    const rootNode = reactive<OfferTreeNode>({
      key: `root:${rootID}`,
      nodeId: rootID,
      hasChildrenHint: true,
      duplicate: false,
      expanded: true,
      loading: true,
      error: "",
      children: null
    }) as OfferTreeNode
    root.value = rootNode
    registerNode(rootNode)
    seenNodeIDs.add(rootID)

    const resp = await callMgmt<ListNodesWire>("ListNodesSimple", props.sourceId, rootID)
    if (epoch !== myEpoch) return
    const code = Number(resp?.code ?? 0)
    if (code !== 1) {
      throw new Error(String(resp?.msg ?? `code=${code}`))
    }
    const children = normalizeNodes(resp).filter((entry) => entry.nodeId !== rootID)
    rootNode.children = buildChildNodes(rootNode.key, rootNode.nodeId, children)
    rootNode.hasChildrenHint = rootNode.children.length > 0
    rootNode.loading = false
    rootLoading.value = false
  } catch (err) {
    if (epoch !== myEpoch) return
    rootLoading.value = false
    rootError.value = toErrorMessage(err)
    if (root.value) {
      root.value.loading = false
      root.value.error = rootError.value
    }
  }
}

const flattenVisible = (rootNode: OfferTreeNode | null) => {
  const out: { node: OfferTreeNode; depth: number }[] = []
  if (!rootNode) return out

  const walk = (node: OfferTreeNode, depth: number) => {
    out.push({ node, depth })
    if (!node.expanded) return
    if (!node.children || !node.children.length) return
    for (const child of node.children) {
      walk(child, depth + 1)
    }
  }

  walk(rootNode, 0)
  return out
}

const visibleNodes = computed(() => flattenVisible(root.value))

const isSelectable = (node: OfferTreeNode) => {
  if (!node?.nodeId) return false
  if (props.excludeNodeId && node.nodeId === Number(props.excludeNodeId)) return false
  return true
}

const selectionTitle = (node: OfferTreeNode) => {
  if (isSelectable(node)) return t("Select node")
  if (props.excludeNodeId && node.nodeId === Number(props.excludeNodeId)) {
    return t("Local node cannot be selected as remote target.")
  }
  return t("Unavailable.")
}

const nodeHint = (node: OfferTreeNode) => {
  if (node.duplicate) return t("Duplicate in tree: expansion disabled.")
  if (node.error) return t("Error: {error}", { error: node.error })
  if (node.loading) return t("Loading...")
  if (node.children && node.children.length === 0) return t("No children.")
  if (node.children && node.children.length > 0) return t("Children: {count}", { count: node.children.length })
  if (node.hasChildrenHint) return t("Children available (not loaded).")
  return t("Leaf node.")
}

const toggleNode = async (node: OfferTreeNode) => {
  if (node.duplicate) return
  if (node.expanded) {
    node.expanded = false
    return
  }
  node.expanded = true
  if (node.children !== null) {
    return
  }
  await loadChildren(node, epoch, false)
}

const retryNode = async (node: OfferTreeNode) => {
  if (node.duplicate) return
  node.expanded = true
  await loadChildren(node, epoch, true)
}

const selectNode = (node: OfferTreeNode) => {
  if (!isSelectable(node)) return
  emit("update:modelValue", node.nodeId)
}

watch(
  () => props.hubId,
  (hubId) => {
    if (!rootTargetId.value.trim() && hubId) {
      rootTargetId.value = String(hubId)
    }
  },
  { immediate: true }
)

watch(
  () => [props.sourceId, props.hubId],
  ([sourceId, hubId]) => {
    if (!sourceId || !hubId) {
      epoch++
      clearTree()
      return
    }
    if (!rootTargetId.value.trim()) {
      rootTargetId.value = String(hubId)
    }
    void loadRoot()
  },
  { immediate: true }
)
</script>

<template>
  <div class="rounded-xl border border-border/60 bg-background/50 p-3">
    <div class="flex flex-wrap items-end justify-between gap-2">
      <div class="min-w-0">
        <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Target Node") }}</p>
        <p class="mt-1 text-xs text-muted-foreground">{{ t("Select destination from node tree.") }}</p>
      </div>
      <div class="flex items-center gap-2">
        <input
          v-model="rootTargetId"
          class="h-8 w-32 rounded-md border border-input bg-background px-2 text-xs text-foreground"
          :placeholder="t('Root ID')"
          @keydown.enter.prevent="loadRoot"
        />
        <Button size="sm" variant="outline" :disabled="rootLoading" @click="loadRoot">
          {{ rootLoading ? t("Loading...") : t("Reload") }}
        </Button>
      </div>
    </div>

    <p v-if="rootError" class="mt-2 text-xs text-rose-600">
      {{ rootError }}
    </p>

    <div class="mt-3 max-h-56 overflow-y-auto rounded-lg border border-border/60 bg-card/80 p-1">
      <div v-if="rootLoading && !root" class="px-2 py-6 text-xs text-muted-foreground">{{ t("Loading tree...") }}</div>
      <div v-else-if="!root" class="px-2 py-6 text-xs text-muted-foreground">{{ t("No nodes loaded.") }}</div>
      <div v-else class="space-y-1">
        <div
          v-for="{ node, depth } in visibleNodes"
          :key="node.key"
          class="rounded-md border border-transparent px-2 py-1.5 transition"
          :class="modelValue === node.nodeId ? 'border-primary/60 bg-primary/10' : 'hover:border-border/60 hover:bg-muted/50'"
        >
          <div class="flex items-center gap-2">
            <button
              type="button"
              class="h-6 w-6 rounded border border-border/60 bg-background text-xs text-foreground transition hover:bg-muted/60 disabled:opacity-50"
              :style="{ marginLeft: `${depth * 14}px` }"
              :disabled="node.duplicate || node.loading"
              @click.stop="toggleNode(node)"
            >
              <span v-if="node.loading">…</span>
              <span v-else>{{ node.expanded ? "-" : "+" }}</span>
            </button>

            <button
              type="button"
              class="min-w-0 flex-1 text-left disabled:cursor-not-allowed disabled:opacity-60"
              :disabled="!isSelectable(node)"
              :title="selectionTitle(node)"
              @click="selectNode(node)"
            >
              <p class="truncate text-sm font-medium">
                {{ t("Node {id}", { id: node.nodeId }) }}
                <span v-if="depth === 0" class="text-xs font-normal text-muted-foreground">
                  ({{ t("root") }})
                </span>
              </p>
              <p class="truncate text-[11px] text-muted-foreground">
                {{ nodeHint(node) }}
              </p>
            </button>

            <Button
              v-if="node.error && !node.duplicate"
              size="sm"
              variant="outline"
              :disabled="node.loading"
              @click.stop="retryNode(node)"
            >
              {{ t("Retry") }}
            </Button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
