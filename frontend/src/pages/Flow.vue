<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { PencilLine, Plus, RefreshCw, Rocket, Settings2, Trash2 } from "lucide-vue-next"
import PageHero from "@/components/PageHero.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import { useI18n } from "@/i18n"
import type { DeviceTreeNode } from "@/stores/devices"
import { useDevicesStore } from "@/stores/devices"
import { useFlowProjectsStore, type FlowTriggerDraft } from "@/stores/flowProjects"
import { useProfileStore } from "@/stores/profile"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"

const flowProjects = useFlowProjectsStore()
const devicesStore = useDevicesStore()
const profileStore = useProfileStore()
const sessionStore = useSessionStore()
const toast = useToastStore()
const { t } = useI18n()

const activeTab = ref<"projects" | "deployments">("projects")
const currentDeployNodeId = ref("")

const createDialogOpen = ref(false)
const createForm = reactive({
  name: ""
})

const metaDialogOpen = ref(false)
const metaForm = reactive({
  projectId: "",
  projectName: "",
  flowId: ""
})

const deployDialogOpen = ref(false)
const deployForm = reactive({
  projectId: "",
  projectName: "",
  flowId: "",
  nodeId: "",
  trigger: {
    type: "interval",
    everyMs: 60000,
    eventMode: "publish",
    eventName: "",
    eventTopic: "",
    varOwner: 0,
    varName: ""
  } as FlowTriggerDraft
})

const nodePickerOpen = ref(false)
const nodePickerTarget = ref<"deploy" | "deployments">("deploy")

const ready = computed(() => Boolean(sessionStore.connected && sessionStore.auth.nodeId && sessionStore.auth.hubId))

const flattenVisible = (root: DeviceTreeNode | null) => {
  const out: { node: DeviceTreeNode; depth: number }[] = []
  if (!root) return out

  const walk = (node: DeviceTreeNode, depth: number) => {
    out.push({ node, depth })
    if (!node.expanded || !node.children?.length) return
    for (const child of node.children) {
      walk(child, depth + 1)
    }
  }

  walk(root, 0)
  return out
}

const visibleNodes = computed(() => flattenVisible(devicesStore.state.root))

const loadProjects = async () => {
  try {
    await flowProjects.loadProjects()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to load local flow projects."))
  }
}

const normalizeTriggerDraft = (trigger: FlowTriggerDraft) => ({
  type: trigger.type,
  everyMs: trigger.everyMs,
  eventMode: trigger.eventMode,
  eventName: trigger.eventName,
  eventTopic: trigger.eventTopic,
  varOwner: trigger.varOwner,
  varName: trigger.varName
})

const triggerTypeOptions = computed(() => [
  { value: "interval" as const, label: t("Interval") },
  { value: "event" as const, label: t("Event") },
  { value: "var_changed" as const, label: t("Variable Changed") }
])

const eventModeOptions = computed(() => [
  { value: "publish" as const, label: t("Publish") },
  { value: "received" as const, label: t("Received") },
  { value: "any" as const, label: t("Any") }
])

const deploymentStatusLabel = (status: string) => flowProjects.describeDeploymentStatus(status || "idle")
const deploymentTriggerLabel = (item: { trigger: Record<string, any> | null; everyMs: number }) =>
  flowProjects.describeTrigger(item.trigger, item.everyMs)

const ensureDeploymentsLoaded = async (options?: { force?: boolean }) => {
  if (activeTab.value !== "deployments") return
  if (!ready.value) return
  const nodeId = currentDeployNodeId.value.trim()
  if (!nodeId) return
  const sameNode = flowProjects.state.deploymentsNodeId === nodeId
  if (!options?.force && sameNode) return
  await reloadDeployments()
}

const setActiveTab = async (tab: "projects" | "deployments") => {
  activeTab.value = tab
  if (tab === "deployments") {
    try {
      await ensureDeploymentsLoaded()
    } catch (err) {
      console.warn(err)
      toast.errorOf(err, t("Failed to load current deployments."))
    }
  }
}

const openCreateDialog = () => {
  createForm.name = ""
  createDialogOpen.value = true
}

const createProject = async () => {
  try {
    await flowProjects.createProject({
      name: createForm.name
    })
    createDialogOpen.value = false
    toast.success(t("Project created."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to create project."))
  }
}

const openMetaDialog = (projectId: string) => {
  const project = flowProjects.getProjectByID(projectId)
  if (!project) {
    toast.error(t("Project not found."))
    return
  }
  metaForm.projectId = project.projectId
  metaForm.projectName = project.name
  metaForm.flowId = project.flowId
  metaDialogOpen.value = true
}

const saveMeta = async () => {
  try {
    await flowProjects.updateProjectMeta({
      projectId: metaForm.projectId,
      name: metaForm.projectName,
      flowId: metaForm.flowId
    })
    metaDialogOpen.value = false
    toast.success(t("Project metadata saved."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to save project metadata."))
  }
}

const deleteProject = async (projectId: string) => {
  const ok = window.confirm(t("Delete local project '{projectId}'? This does not delete remote deployment.", { projectId }))
  if (!ok) return
  try {
    await flowProjects.deleteProject(projectId)
    toast.success(t("Project deleted."))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to delete local project."))
  }
}

const openEditor = (projectId: string) => {
  try {
    const opened = flowProjects.openEditorWindow(projectId)
    if (!opened) {
      toast.warn(t("Editor window was blocked by browser popup policy."))
    }
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to open editor window."))
  }
}

const openDeployDialog = (projectId: string) => {
  const project = flowProjects.getProjectByID(projectId)
  if (!project) {
    toast.error(t("Project not found."))
    return
  }
  deployForm.projectId = project.projectId
  deployForm.projectName = project.name || project.flowId
  deployForm.flowId = project.flowId
  deployForm.nodeId = currentDeployNodeId.value || String(sessionStore.auth.hubId || "")
  deployForm.trigger = normalizeTriggerDraft(project.trigger)
  deployDialogOpen.value = true
}

const pickNode = async (target: "deploy" | "deployments") => {
  nodePickerTarget.value = target
  if (!devicesStore.state.rootTargetId && sessionStore.auth.hubId) {
    devicesStore.state.rootTargetId = String(sessionStore.auth.hubId)
  }
  try {
    await devicesStore.loadRoot()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to load device tree."))
  }
  nodePickerOpen.value = true
}

const chooseNode = async (nodeId: number) => {
  nodePickerOpen.value = false
  if (nodePickerTarget.value === "deploy") {
    deployForm.nodeId = String(nodeId)
    return
  }
  currentDeployNodeId.value = String(nodeId)
  if (activeTab.value === "deployments") {
    try {
      await reloadDeployments()
    } catch (err) {
      console.warn(err)
      toast.errorOf(err, t("Failed to load current deployments."))
    }
  }
}

const toggleNode = async (node: DeviceTreeNode) => {
  try {
    await devicesStore.toggle(node.key)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to expand node."))
  }
}

const reloadDeployments = async () => {
  try {
    await flowProjects.loadDeployments(currentDeployNodeId.value)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to load current deployments."))
  }
}

const deployNow = async () => {
  try {
    const first = await flowProjects.deployProject({
      projectId: deployForm.projectId,
      nodeId: deployForm.nodeId,
      trigger: deployForm.trigger,
      overwrite: false
    })
    if (first.overwriteRequired) {
      const confirmed = window.confirm(
        t("Flow '{flowId}' already exists on node {nodeId}. Overwrite deployment?", {
          flowId: deployForm.flowId,
          nodeId: deployForm.nodeId
        })
      )
      if (!confirmed) return
      await flowProjects.deployProject({
        projectId: deployForm.projectId,
        nodeId: deployForm.nodeId,
        trigger: deployForm.trigger,
        overwrite: true
      })
    }
    deployDialogOpen.value = false
    toast.success(t("Deployment saved to target node."))
    currentDeployNodeId.value = deployForm.nodeId
    activeTab.value = "deployments"
    await reloadDeployments()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to deploy project."))
  }
}

const deleteDeployment = async (flowId: string) => {
  const nodeId = currentDeployNodeId.value
  const ok = window.confirm(t("Delete deployment '{flowId}' from node {nodeId}?", { flowId, nodeId }))
  if (!ok) return
  try {
    await flowProjects.deleteDeployment(nodeId, flowId)
    toast.success(t("Deployment deleted."))
    await reloadDeployments()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Failed to delete deployment."))
  }
}

watch(
  () => profileStore.state.current,
  async () => {
    await loadProjects()
  },
  { immediate: true }
)

watch(
  () => sessionStore.auth.hubId,
  (hubId) => {
    if (!currentDeployNodeId.value && Number(hubId || 0) > 0) {
      currentDeployNodeId.value = String(hubId)
    }
  },
  { immediate: true }
)

watch(
  () => ready.value,
  async (isReady) => {
    if (!isReady || activeTab.value !== "deployments") return
    await ensureDeploymentsLoaded({ force: true })
  }
)

onMounted(async () => {
  if (!currentDeployNodeId.value && sessionStore.auth.hubId) {
    currentDeployNodeId.value = String(sessionStore.auth.hubId)
  }
})
</script>

<template>
  <section class="space-y-6">
    <PageHero>
      <template #actions>
        <div class="inline-flex rounded-full border border-border/70 bg-background/80 p-1">
          <button
            type="button"
            :class="[
              'rounded-full px-4 py-2 text-sm font-semibold transition',
              activeTab === 'projects'
                ? 'bg-primary text-primary-foreground shadow-sm'
                : 'text-muted-foreground hover:bg-muted/70'
            ]"
            @click="setActiveTab('projects')"
          >
            {{ t("Local Projects") }}
          </button>
          <button
            type="button"
            :class="[
              'rounded-full px-4 py-2 text-sm font-semibold transition',
              activeTab === 'deployments'
                ? 'bg-primary text-primary-foreground shadow-sm'
                : 'text-muted-foreground hover:bg-muted/70'
            ]"
            @click="setActiveTab('deployments')"
          >
            {{ t("Current Deployments") }}
          </button>
        </div>
      </template>
    </PageHero>

    <section v-if="activeTab === 'projects'" class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">{{ t("Local") }}</p>
          <h2 class="mt-1 text-lg font-semibold">{{ t("Flow Projects") }}</h2>
        </div>
        <Button @click="openCreateDialog">
          <Plus class="mr-2 h-4 w-4" />
          {{ t("New Project") }}
        </Button>
      </div>

      <div class="mt-4 space-y-2">
        <article
          v-for="project in flowProjects.state.projects"
          :key="project.projectId"
          class="rounded-xl border border-border/60 bg-background/70 p-4"
        >
          <div class="flex flex-wrap items-start justify-between gap-4">
            <div class="space-y-1">
              <p class="font-semibold">{{ project.name || project.flowId }}</p>
              <p class="text-xs text-muted-foreground">{{ t("Flow ID") }}: {{ project.flowId }}</p>
              <p class="text-xs text-muted-foreground">{{ t("Project ID") }}: {{ project.projectId }}</p>
              <p class="text-xs text-muted-foreground">{{ t("updated: {time}", { time: project.updatedAt }) }}</p>
            </div>

            <div class="flex flex-wrap items-center gap-2">
              <Button size="sm" variant="outline" @click="openMetaDialog(project.projectId)">
                <Settings2 class="mr-1 h-4 w-4" />
                {{ t("Meta") }}
              </Button>
              <Button size="sm" variant="outline" @click="openEditor(project.projectId)">
                <PencilLine class="mr-1 h-4 w-4" />
                {{ t("Edit") }}
              </Button>
              <Button size="sm" @click="openDeployDialog(project.projectId)">
                <Rocket class="mr-1 h-4 w-4" />
                {{ t("Deploy") }}
              </Button>
              <Button size="sm" variant="outline" @click="deleteProject(project.projectId)">
                <Trash2 class="mr-1 h-4 w-4" />
                {{ t("Delete") }}
              </Button>
            </div>
          </div>
        </article>

        <div v-if="!flowProjects.state.projects.length" class="rounded-xl border border-dashed p-4 text-xs text-muted-foreground">
          {{ t("No local projects yet. Create one and open the editor window.") }}
        </div>
      </div>
    </section>

    <section v-else class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">{{ t("Runtime") }}</p>
          <h2 class="mt-1 text-lg font-semibold">{{ t("Current Deployments") }}</h2>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
            <span class="font-semibold uppercase tracking-[0.2em]">{{ t("Node") }}</span>
            <input
              v-model="currentDeployNodeId"
              class="h-7 w-28 rounded-md border border-input bg-background px-2 text-xs text-foreground"
              :placeholder="t('Node ID')"
            />
          </div>
          <Button size="sm" variant="outline" @click="pickNode('deployments')">{{ t("Select node") }}</Button>
          <Button size="sm" :disabled="flowProjects.state.deploymentsLoading" @click="reloadDeployments">
            <RefreshCw class="mr-2 h-4 w-4" />
            {{ t("Refresh") }}
          </Button>
        </div>
      </div>

      <div class="mt-4 space-y-2">
        <article
          v-for="item in flowProjects.state.deployments"
          :key="item.flowId"
          class="rounded-xl border border-border/60 bg-background/70 p-3"
        >
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p class="font-semibold">{{ item.name || item.flowId }}</p>
              <p class="text-xs text-muted-foreground">{{ t("Flow ID") }}: {{ item.flowId }}</p>
            </div>
            <div class="flex items-center gap-2">
              <Badge variant="secondary">{{ deploymentStatusLabel(item.lastStatus) }}</Badge>
              <Button size="sm" variant="outline" @click="deleteDeployment(item.flowId)">
                <Trash2 class="mr-1 h-4 w-4" />
                {{ t("Delete") }}
              </Button>
            </div>
          </div>
          <div class="mt-2 grid gap-2 text-xs text-muted-foreground md:grid-cols-2">
            <p><span class="font-semibold text-foreground">{{ t("Trigger:") }}</span> {{ deploymentTriggerLabel(item) }}</p>
            <p><span class="font-semibold text-foreground">{{ t("Last run ID") }}:</span> {{ item.lastRunId || "-" }}</p>
          </div>
        </article>

        <div v-if="!flowProjects.state.deployments.length" class="rounded-xl border border-dashed p-4 text-xs text-muted-foreground">
          {{ t("No deployments found for this node.") }}
        </div>
      </div>
    </section>

    <Overlay :open="createDialogOpen" @close="createDialogOpen = false">
      <div class="w-full max-w-lg rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">{{ t("Create Flow Project") }}</h2>
        <div class="mt-4 grid gap-3">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("name") }}</label>
            <input
              v-model="createForm.name"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              :placeholder="t('Optional')"
            />
          </div>
        </div>
        <p class="mt-4 text-xs text-muted-foreground">
          {{ t("A unique local project id and a default random flow_id will be generated automatically. You can change metadata later from the project list.") }}
        </p>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="createDialogOpen = false">{{ t("Cancel") }}</Button>
          <Button @click="createProject">{{ t("Create") }}</Button>
        </div>
      </div>
    </Overlay>

    <Overlay :open="metaDialogOpen" @close="metaDialogOpen = false">
      <div class="w-full max-w-lg rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">{{ t("Project Metadata") }}</h2>
        <div class="mt-4 grid gap-4">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Project ID") }}</label>
            <input
              :value="metaForm.projectId"
              disabled
              class="mt-2 h-10 w-full rounded-md border border-input bg-muted/40 px-3 text-sm text-muted-foreground"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("name") }}</label>
            <input
              v-model="metaForm.projectName"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              :placeholder="t('Optional')"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Flow ID") }}</label>
            <input
              v-model="metaForm.flowId"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              :placeholder="t('Required')"
            />
          </div>
        </div>
        <p class="mt-4 text-xs text-muted-foreground">
          {{ t("flow_id must stay unique among local projects because it is the deployment identity used on target nodes.") }}
        </p>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="metaDialogOpen = false">{{ t("Cancel") }}</Button>
          <Button @click="saveMeta">{{ t("Save") }}</Button>
        </div>
      </div>
    </Overlay>

    <Overlay :open="deployDialogOpen" @close="deployDialogOpen = false">
      <div class="w-full max-w-2xl rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">{{ t("Deploy Project") }}</h2>
        <p class="mt-1 text-sm text-muted-foreground">{{ deployForm.projectName }} · {{ t("Flow ID") }} {{ deployForm.flowId }}</p>

        <div class="mt-4 grid gap-4 md:grid-cols-2">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Node ID") }}</label>
            <input
              v-model="deployForm.nodeId"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              :placeholder="t('Target node id')"
            />
          </div>
          <div class="flex items-end">
            <Button variant="outline" @click="pickNode('deploy')">{{ t("Select node") }}</Button>
          </div>
        </div>

        <div class="mt-4 grid gap-4 md:grid-cols-3">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Trigger") }}</label>
            <select
              v-model="deployForm.trigger.type"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            >
              <option v-for="option in triggerTypeOptions" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          </div>

          <div v-if="deployForm.trigger.type === 'interval'">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Every (ms)") }}</label>
            <input
              v-model.number="deployForm.trigger.everyMs"
              type="number"
              min="1"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>

          <template v-else-if="deployForm.trigger.type === 'event'">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Event Mode") }}</label>
              <select
                v-model="deployForm.trigger.eventMode"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              >
                <option v-for="option in eventModeOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Event Name") }}</label>
              <input
                v-model="deployForm.trigger.eventName"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Event Topic") }}</label>
              <input
                v-model="deployForm.trigger.eventTopic"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              />
            </div>
          </template>

          <template v-else>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Var Owner") }}</label>
              <input
                v-model.number="deployForm.trigger.varOwner"
                type="number"
                min="0"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t("Var Name") }}</label>
              <input
                v-model="deployForm.trigger.varName"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              />
            </div>
          </template>
        </div>

        <p class="mt-4 text-xs text-muted-foreground">
          {{ t("Deployment only sends flow.set; it does not trigger run. Trigger edits here will be saved back as project default.") }}
        </p>

        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="deployDialogOpen = false">{{ t("Cancel") }}</Button>
          <Button @click="deployNow">{{ t("Deploy") }}</Button>
        </div>
      </div>
    </Overlay>

    <Overlay :open="nodePickerOpen" @close="nodePickerOpen = false">
      <div class="w-full max-w-2xl rounded-2xl border bg-card/95 p-6 shadow-xl">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <h2 class="text-lg font-semibold">{{ t("Select node") }}</h2>
          <Button size="sm" variant="outline" @click="devicesStore.loadRoot">{{ t("Reload Tree") }}</Button>
        </div>
        <div class="mt-4 max-h-[70vh] space-y-2 overflow-y-auto">
          <article
            v-for="{ node, depth } in visibleNodes"
            :key="node.key"
            class="rounded-xl border border-border/60 bg-background/70 p-3"
          >
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div class="flex items-center gap-2" :style="{ marginLeft: `${depth * 16}px` }">
                <button
                  type="button"
                  class="h-7 w-7 rounded-md border border-border/70 bg-background text-xs"
                  :disabled="node.duplicate || node.loading"
                  @click="toggleNode(node)"
                >
                  <span v-if="node.loading">…</span>
                  <span v-else>{{ node.expanded ? "-" : "+" }}</span>
                </button>
                <div>
                  <p class="font-semibold">{{ t("Node {nodeId}", { nodeId: node.nodeId }) }}</p>
                  <p class="text-xs text-muted-foreground">
                    <span v-if="node.duplicate">{{ t("Duplicate node in current tree path.") }}</span>
                    <span v-else-if="node.error">{{ node.error }}</span>
                    <span v-else-if="node.children">{{ t("children {count}", { count: node.children.length }) }}</span>
                    <span v-else>{{ t("not loaded") }}</span>
                  </p>
                </div>
              </div>
              <Button size="sm" @click="chooseNode(node.nodeId)">{{ t("Select") }}</Button>
            </div>
          </article>

          <div v-if="!visibleNodes.length" class="rounded-xl border border-dashed p-4 text-xs text-muted-foreground">
            {{ t("No tree data. Connect and login first, then reload the tree.") }}
          </div>
        </div>
      </div>
    </Overlay>
  </section>
</template>
