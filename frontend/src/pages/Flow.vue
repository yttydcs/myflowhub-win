<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { PencilLine, Plus, RefreshCw, Rocket, Trash2 } from "lucide-vue-next"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
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

const currentDeployNodeId = ref("")

const createDialogOpen = ref(false)
const createForm = reactive({
  projectId: "",
  flowId: "",
  name: ""
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
    toast.errorOf(err, "Failed to load local flow projects.")
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

const openCreateDialog = () => {
  createForm.projectId = ""
  createForm.flowId = ""
  createForm.name = ""
  createDialogOpen.value = true
}

const createProject = async () => {
  try {
    await flowProjects.createProject({
      projectId: createForm.projectId,
      flowId: createForm.flowId,
      name: createForm.name
    })
    createDialogOpen.value = false
    toast.success("Project created.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to create project.")
  }
}

const deleteProject = async (projectId: string) => {
  const ok = window.confirm(`Delete local project '${projectId}'? This does not delete remote deployment.`)
  if (!ok) return
  try {
    await flowProjects.deleteProject(projectId)
    toast.success("Project deleted.")
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to delete local project.")
  }
}

const openEditor = (projectId: string) => {
  try {
    const opened = flowProjects.openEditorWindow(projectId)
    if (!opened) {
      toast.warn("Editor window was blocked by browser popup policy.")
    }
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to open editor window.")
  }
}

const openDeployDialog = (projectId: string) => {
  const project = flowProjects.getProjectByID(projectId)
  if (!project) {
    toast.error("Project not found.")
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
    toast.errorOf(err, "Failed to load device tree.")
  }
  nodePickerOpen.value = true
}

const chooseNode = (nodeId: number) => {
  if (nodePickerTarget.value === "deploy") {
    deployForm.nodeId = String(nodeId)
  } else {
    currentDeployNodeId.value = String(nodeId)
  }
  nodePickerOpen.value = false
}

const toggleNode = async (node: DeviceTreeNode) => {
  try {
    await devicesStore.toggle(node.key)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to expand node.")
  }
}

const reloadDeployments = async () => {
  try {
    await flowProjects.loadDeployments(currentDeployNodeId.value)
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to load current deployments.")
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
        `Flow '${deployForm.flowId}' already exists on node ${deployForm.nodeId}. Overwrite deployment?`
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
    toast.success("Deployment saved to target node.")
    currentDeployNodeId.value = deployForm.nodeId
    await reloadDeployments()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to deploy project.")
  }
}

const deleteDeployment = async (flowId: string) => {
  const nodeId = currentDeployNodeId.value
  const ok = window.confirm(`Delete deployment '${flowId}' from node ${nodeId}?`)
  if (!ok) return
  try {
    await flowProjects.deleteDeployment(nodeId, flowId)
    toast.success("Deployment deleted.")
    await reloadDeployments()
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, "Failed to delete deployment.")
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

onMounted(async () => {
  if (!currentDeployNodeId.value && sessionStore.auth.hubId) {
    currentDeployNodeId.value = String(sessionStore.auth.hubId)
  }
  if (ready.value && currentDeployNodeId.value) {
    await reloadDeployments()
  }
})
</script>

<template>
  <section class="space-y-6">
    <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">Runtime</p>
          <h2 class="mt-1 text-lg font-semibold">Current Deployments</h2>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
            <span class="font-semibold uppercase tracking-[0.2em]">Node</span>
            <input
              v-model="currentDeployNodeId"
              class="h-7 w-28 rounded-md border border-input bg-background px-2 text-xs text-foreground"
              placeholder="Node ID"
            />
          </div>
          <Button size="sm" variant="outline" @click="pickNode('deployments')">Choose from tree</Button>
          <Button size="sm" :disabled="flowProjects.state.deploymentsLoading" @click="reloadDeployments">
            <RefreshCw class="mr-2 h-4 w-4" />
            Refresh
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
              <p class="text-xs text-muted-foreground">flow_id: {{ item.flowId }}</p>
            </div>
            <div class="flex items-center gap-2">
              <Badge variant="secondary">{{ item.lastStatus || "idle" }}</Badge>
              <Button size="sm" variant="outline" @click="deleteDeployment(item.flowId)">
                <Trash2 class="mr-1 h-4 w-4" />
                Delete
              </Button>
            </div>
          </div>
          <div class="mt-2 grid gap-2 text-xs text-muted-foreground md:grid-cols-2">
            <p><span class="font-semibold text-foreground">Trigger:</span> {{ item.triggerLabel }}</p>
            <p><span class="font-semibold text-foreground">last_run_id:</span> {{ item.lastRunId || "-" }}</p>
          </div>
        </article>

        <div v-if="!flowProjects.state.deployments.length" class="rounded-xl border border-dashed p-4 text-xs text-muted-foreground">
          No deployments found for this node.
        </div>
      </div>
    </section>

    <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">Local</p>
          <h2 class="mt-1 text-lg font-semibold">Flow Projects</h2>
        </div>
        <Button @click="openCreateDialog">
          <Plus class="mr-2 h-4 w-4" />
          New Project
        </Button>
      </div>

      <div class="mt-4 space-y-2">
        <article
          v-for="project in flowProjects.state.projects"
          :key="project.projectId"
          class="rounded-xl border border-border/60 bg-background/70 p-3"
        >
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p class="font-semibold">{{ project.name || project.flowId }}</p>
              <p class="text-xs text-muted-foreground">project_id: {{ project.projectId }}</p>
              <p class="text-xs text-muted-foreground">flow_id: {{ project.flowId }}</p>
              <p class="text-xs text-muted-foreground">updated: {{ project.updatedAt }}</p>
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <Button size="sm" variant="outline" @click="openEditor(project.projectId)">
                <PencilLine class="mr-1 h-4 w-4" />
                Edit
              </Button>
              <Button size="sm" @click="openDeployDialog(project.projectId)">
                <Rocket class="mr-1 h-4 w-4" />
                Deploy
              </Button>
              <Button size="sm" variant="outline" @click="deleteProject(project.projectId)">
                <Trash2 class="mr-1 h-4 w-4" />
                Delete
              </Button>
            </div>
          </div>
        </article>

        <div v-if="!flowProjects.state.projects.length" class="rounded-xl border border-dashed p-4 text-xs text-muted-foreground">
          No local projects yet. Create one and open the editor window.
        </div>
      </div>
    </section>

    <Overlay :open="createDialogOpen" @close="createDialogOpen = false">
      <div class="w-full max-w-lg rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">Create Flow Project</h2>
        <div class="mt-4 grid gap-3">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">project_id</label>
            <input
              v-model="createForm.projectId"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="Optional, auto generated if empty"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">flow_id</label>
            <input
              v-model="createForm.flowId"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="Required"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">name</label>
            <input
              v-model="createForm.name"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="Optional"
            />
          </div>
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="createDialogOpen = false">Cancel</Button>
          <Button @click="createProject">Create</Button>
        </div>
      </div>
    </Overlay>

    <Overlay :open="deployDialogOpen" @close="deployDialogOpen = false">
      <div class="w-full max-w-2xl rounded-2xl border bg-card/95 p-6 shadow-xl">
        <h2 class="text-lg font-semibold">Deploy Project</h2>
        <p class="mt-1 text-sm text-muted-foreground">{{ deployForm.projectName }} · flow_id {{ deployForm.flowId }}</p>

        <div class="mt-4 grid gap-4 md:grid-cols-2">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Node ID</label>
            <input
              v-model="deployForm.nodeId"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              placeholder="Target node id"
            />
          </div>
          <div class="flex items-end">
            <Button variant="outline" @click="pickNode('deploy')">Choose from device tree</Button>
          </div>
        </div>

        <div class="mt-4 grid gap-4 md:grid-cols-3">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Trigger</label>
            <select
              v-model="deployForm.trigger.type"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            >
              <option value="interval">interval</option>
              <option value="event">event</option>
              <option value="var_changed">var_changed</option>
            </select>
          </div>

          <div v-if="deployForm.trigger.type === 'interval'">
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Every (ms)</label>
            <input
              v-model.number="deployForm.trigger.everyMs"
              type="number"
              min="1"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>

          <template v-else-if="deployForm.trigger.type === 'event'">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Event Mode</label>
              <select
                v-model="deployForm.trigger.eventMode"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              >
                <option value="publish">publish</option>
                <option value="received">received</option>
                <option value="any">any</option>
              </select>
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Event Name</label>
              <input
                v-model="deployForm.trigger.eventName"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Event Topic</label>
              <input
                v-model="deployForm.trigger.eventTopic"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              />
            </div>
          </template>

          <template v-else>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Var Owner</label>
              <input
                v-model.number="deployForm.trigger.varOwner"
                type="number"
                min="0"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Var Name</label>
              <input
                v-model="deployForm.trigger.varName"
                class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              />
            </div>
          </template>
        </div>

        <p class="mt-4 text-xs text-muted-foreground">
          Deployment only sends <code>flow.set</code>; it does not trigger run. Trigger edits here will be saved back as project default.
        </p>

        <div class="mt-6 flex justify-end gap-2">
          <Button variant="outline" @click="deployDialogOpen = false">Cancel</Button>
          <Button @click="deployNow">Deploy</Button>
        </div>
      </div>
    </Overlay>

    <Overlay :open="nodePickerOpen" @close="nodePickerOpen = false">
      <div class="w-full max-w-2xl rounded-2xl border bg-card/95 p-6 shadow-xl">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <h2 class="text-lg font-semibold">Select Node</h2>
          <Button size="sm" variant="outline" @click="devicesStore.loadRoot">Reload Tree</Button>
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
                  <p class="font-semibold">Node {{ node.nodeId }}</p>
                  <p class="text-xs text-muted-foreground">
                    <span v-if="node.duplicate">Duplicate node in current tree path.</span>
                    <span v-else-if="node.error">{{ node.error }}</span>
                    <span v-else-if="node.children">children {{ node.children.length }}</span>
                    <span v-else>not loaded</span>
                  </p>
                </div>
              </div>
              <Button size="sm" @click="chooseNode(node.nodeId)">Select</Button>
            </div>
          </article>

          <div v-if="!visibleNodes.length" class="rounded-xl border border-dashed p-4 text-xs text-muted-foreground">
            No tree data. Connect and login first, then reload the tree.
          </div>
        </div>
      </div>
    </Overlay>
  </section>
</template>
