<script setup lang="ts">
// 本文件实现 Win 前端的 `Presets` 页面。
import { computed, onMounted, reactive, ref, watch } from "vue"
import CardHeader from "@/components/CardHeader.vue"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"
import { parseIntegerInput } from "@/lib/numberInput"
import { usePresetsStore } from "@/stores/presets"
import { useSessionStore } from "@/stores/session"
import { useToastStore } from "@/stores/toast"

type WailsBinding = (...args: any[]) => Promise<any>

const callAuth = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.auth?.AuthService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(t("Auth binding '{method}' unavailable", { method }))
  }
  return fn(...args)
}

const callVarPool = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.varpool?.VarPoolService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(t("VarPool binding '{method}' unavailable", { method }))
  }
  return fn(...args)
}

const callTopicBus = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.topicbus?.TopicBusService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(t("TopicBus binding '{method}' unavailable", { method }))
  }
  return fn(...args)
}

const callFlow = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.flow?.FlowService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(t("Flow binding '{method}' unavailable", { method }))
  }
  return fn(...args)
}

const callManagement = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.management?.ManagementService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(t("Management binding '{method}' unavailable", { method }))
  }
  return fn(...args)
}

const callFile = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.file?.FileService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(t("File binding '{method}' unavailable", { method }))
  }
  return fn(...args)
}

const sessionStore = useSessionStore()
const presetStore = usePresetsStore()
const toast = useToastStore()
const { t } = useI18n()

const senderForm = reactive({
  targetId: "",
  topic: "stress",
  runId: "",
  total: 1000,
  payloadSize: 0,
  maxPerSec: 0
})

const receiverForm = reactive({
  targetId: "",
  topic: "stress",
  runId: "",
  total: 1000,
  payloadSize: 0
})

const echoForm = reactive({ message: "", targetId: "" })
const registerForm = reactive({ deviceId: "", targetId: "" })
const loginForm = reactive({ deviceId: "", nodeId: "", targetId: "" })

const varSetForm = reactive({
  name: "",
  value: "",
  visibility: "public",
  kind: "string",
  owner: "",
  targetId: ""
})
const varGetForm = reactive({ name: "", owner: "", targetId: "" })
const varRevokeForm = reactive({ name: "", owner: "", targetId: "" })
const varSubForm = reactive({ name: "", owner: "", targetId: "" })
const varUnsubForm = reactive({ name: "", owner: "", targetId: "" })

const topicPublishForm = reactive({
  topic: "",
  name: "stress",
  payload: "{}",
  targetId: ""
})
const topicSubForm = reactive({ topic: "", targetId: "" })
const topicUnsubForm = reactive({ topic: "", targetId: "" })

const flowListForm = reactive({ executorId: "" })
const flowGetForm = reactive({ flowId: "" })
const flowStatusForm = reactive({ runId: "" })

const mgmtConfigGetForm = reactive({ targetId: "", key: "" })
const mgmtConfigSetForm = reactive({ value: "" })

const fileListForm = reactive({ targetId: "", dir: "", recursive: false })
const fileReadForm = reactive({ name: "", maxBytes: 65536 })

const senderStatus = computed(() => presetStore.state.sender)
const receiverStatus = computed(() => presetStore.state.receiver)

const receiverMissing = computed(() => {
  const missing = receiverStatus.value.expected - receiverStatus.value.unique
  return missing > 0 ? missing : 0
})

const senderRate = computed(() => {
  const started = senderStatus.value.startedAt ? new Date(senderStatus.value.startedAt).getTime() : 0
  const updated = senderStatus.value.updatedAt ? new Date(senderStatus.value.updatedAt).getTime() : 0
  if (!started || !updated || updated <= started) return 0
  return Math.round((senderStatus.value.sent / (updated - started)) * 1000)
})

const receiverRate = computed(() => {
  const started = receiverStatus.value.startedAt ? new Date(receiverStatus.value.startedAt).getTime() : 0
  const updated = receiverStatus.value.updatedAt ? new Date(receiverStatus.value.updatedAt).getTime() : 0
  if (!started || !updated || updated <= started) return 0
  return Math.round((receiverStatus.value.unique / (updated - started)) * 1000)
})

const newReqId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID()
  }
  return `req_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`
}

const buildRunId = () => `run-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`

const inputClass =
  "mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

const textAreaClass =
  "mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

const connectionLabel = computed(() => (sessionStore.connected ? t("Connected") : t("Disconnected")))
const authLabel = computed(() => (sessionStore.auth.loggedIn ? t("Logged In") : t("Logged Out")))
const defaultTargetId = computed(
  () => Number(sessionStore.auth.hubId || sessionStore.auth.nodeId || 0)
)
const rateText = (value: number) => t("{count}/s", { count: value })

const ensureConnected = () => {
  if (!sessionStore.connected) {
    throw new Error(t("Connect to a session before sending requests."))
  }
}

const ensureSourceId = () => {
  const sourceID = Number(sessionStore.auth.nodeId || 0)
  if (!sourceID) {
    throw new Error(t("Login required."))
  }
  return sourceID
}

const ensureHubId = () => {
  const hubID = Number(sessionStore.auth.hubId || 0)
  if (!hubID) {
    throw new Error(t("Hub ID missing."))
  }
  return hubID
}

const parsePositiveInt = (raw: any, field: string) => {
  return parseIntegerInput(raw, {
    min: 1,
    invalidMessage: t("{field} must be a positive number.", {
      field: t(field)
    })
  })
}

const parseNonNegativeInt = (raw: any, field: string) => {
  return parseIntegerInput(raw, {
    min: 0,
    invalidMessage: t("{field} must be a non-negative number.", {
      field: t(field)
    })
  })
}

const resolveTargetId = (raw: unknown) => {
  const fallback = defaultTargetId.value
  return parseIntegerInput(raw, {
    allowBlank: fallback > 0,
    blankValue: fallback,
    requiredMessage: t("Target ID is required."),
    invalidMessage: t("Target ID must be a positive number."),
    min: 1
  })
}

const resolveOwnerId = (raw: unknown, fallback: number) =>
  parseIntegerInput(raw, {
    allowBlank: fallback > 0,
    blankValue: fallback,
    invalidMessage: t("Owner must be a positive number."),
    min: 1
  })

const runAction = async (label: string, action: () => Promise<void>) => {
  if (busy.value) return
  busy.value = true
  try {
    await action()
    toast.success(t(label))
  } catch (err) {
    console.warn(err)
    toast.errorOf(err, t("Action failed."))
  } finally {
    busy.value = false
  }
}

const startStressSender = async () => {
  await runAction("Stress sender started.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(senderForm.targetId)
    const topic = senderForm.topic.trim() || "stress"
    const total = parsePositiveInt(senderForm.total, "Total events")
    const payloadSize = parseNonNegativeInt(senderForm.payloadSize, "Payload size")
    const maxPerSec = parseNonNegativeInt(senderForm.maxPerSec, "Max per second")
    const runId = senderForm.runId.trim() || buildRunId()
    senderForm.runId = runId
    await presetStore.startSender({
      sourceId: sourceID,
      targetId: targetID,
      topic,
      runId,
      total,
      payloadSize,
      maxPerSec
    })
    await presetStore.loadSender()
  })
}

const stopStressSender = async () => {
  await runAction("Stress sender stopped.", async () => {
    await presetStore.stopSender()
    await presetStore.loadSender()
  })
}

const startStressReceiver = async () => {
  await runAction("Stress receiver started.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(receiverForm.targetId)
    const topic = receiverForm.topic.trim() || "stress"
    const total = parsePositiveInt(receiverForm.total, "Expected total")
    const payloadSize = parseNonNegativeInt(receiverForm.payloadSize, "Payload size")
    const runId = receiverForm.runId.trim() || buildRunId()
    receiverForm.runId = runId
    await presetStore.startReceiver({
      sourceId: sourceID,
      targetId: targetID,
      topic,
      runId,
      total,
      payloadSize,
      maxPerSec: 0
    })
    await presetStore.loadReceiver()
  })
}

const stopStressReceiver = async () => {
  await runAction("Stress receiver stopped.", async () => {
    await presetStore.stopReceiver()
    await presetStore.loadReceiver()
  })
}

const resetStressReceiver = async () => {
  await runAction("Stress receiver reset.", async () => {
    await presetStore.resetReceiver()
    await presetStore.loadReceiver()
  })
}

const sendEcho = async () => {
  await runAction("Node echo sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(echoForm.targetId)
    const text = echoForm.message.trim()
    if (!text) {
      throw new Error(t("Message is required."))
    }
    await callManagement("NodeEchoSimple", sourceID, targetID, text)
  })
}

const sendRegister = async () => {
  await runAction("Register request sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(registerForm.targetId)
    const deviceId = registerForm.deviceId.trim()
    if (!deviceId) {
      throw new Error(t("Device ID is required."))
    }
    await callAuth("RegisterSimple", sourceID, targetID, deviceId)
  })
}

const sendLogin = async () => {
  await runAction("Login request sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(loginForm.targetId)
    const deviceId = loginForm.deviceId.trim()
    if (!deviceId) {
      throw new Error(t("Device ID is required."))
    }
    const nodeId = parsePositiveInt(loginForm.nodeId, "Node ID")
    await callAuth("LoginSimple", sourceID, targetID, deviceId, nodeId)
  })
}

const sendVarSet = async () => {
  await runAction("VarPool set sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(varSetForm.targetId)
    const name = varSetForm.name.trim()
    if (!name) {
      throw new Error(t("Variable name is required."))
    }
    const value = varSetForm.value
    if (!value.trim()) {
      throw new Error(t("Variable value is required."))
    }
    const visibility = varSetForm.visibility || "public"
    const kind = varSetForm.kind.trim() || "string"
    const owner = resolveOwnerId(varSetForm.owner, sourceID)
    await callVarPool("SetSimple", sourceID, targetID, {
      name,
      value,
      visibility,
      type: kind,
      owner
    })
    await callVarPool("GetSimple", sourceID, targetID, { name, owner })
  })
}

const sendVarGet = async () => {
  await runAction("VarPool get sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(varGetForm.targetId)
    const name = varGetForm.name.trim()
    if (!name) {
      throw new Error(t("Variable name is required."))
    }
    const owner = resolveOwnerId(varGetForm.owner, sourceID)
    await callVarPool("GetSimple", sourceID, targetID, { name, owner })
  })
}

const sendVarRevoke = async () => {
  await runAction("VarPool revoke sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(varRevokeForm.targetId)
    const name = varRevokeForm.name.trim()
    if (!name) {
      throw new Error(t("Variable name is required."))
    }
    const owner = resolveOwnerId(varRevokeForm.owner, sourceID)
    await callVarPool("RevokeSimple", sourceID, targetID, { name, owner })
  })
}

const sendVarSubscribe = async () => {
  await runAction("VarPool subscribe sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(varSubForm.targetId)
    const name = varSubForm.name.trim()
    if (!name) {
      throw new Error(t("Variable name is required."))
    }
    const owner = resolveOwnerId(varSubForm.owner, sourceID)
    await callVarPool("SubscribeSimple", sourceID, targetID, {
      name,
      owner,
      subscriber: sourceID
    })
  })
}

const sendVarUnsubscribe = async () => {
  await runAction("VarPool unsubscribe sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(varUnsubForm.targetId)
    const name = varUnsubForm.name.trim()
    if (!name) {
      throw new Error(t("Variable name is required."))
    }
    const owner = resolveOwnerId(varUnsubForm.owner, sourceID)
    await callVarPool("UnsubscribeSimple", sourceID, targetID, {
      name,
      owner,
      subscriber: sourceID
    })
  })
}

const sendTopicPublish = async () => {
  await runAction("Topic published.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(topicPublishForm.targetId)
    const topic = topicPublishForm.topic.trim()
    if (!topic) {
      throw new Error(t("Topic is required."))
    }
    const name = topicPublishForm.name.trim()
    if (!name) {
      throw new Error(t("Name is required."))
    }
    await callTopicBus(
      "PublishSimple",
      sourceID,
      targetID,
      topic,
      name,
      topicPublishForm.payload
    )
  })
}

const sendTopicSubscribe = async () => {
  await runAction("Topic subscribe sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(topicSubForm.targetId)
    const topic = topicSubForm.topic.trim()
    if (!topic) {
      throw new Error(t("Topic is required."))
    }
    await callTopicBus("SubscribeSimple", sourceID, targetID, topic)
  })
}

const sendTopicUnsubscribe = async () => {
  await runAction("Topic unsubscribe sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(topicUnsubForm.targetId)
    const topic = topicUnsubForm.topic.trim()
    if (!topic) {
      throw new Error(t("Topic is required."))
    }
    await callTopicBus("UnsubscribeSimple", sourceID, targetID, topic)
  })
}

const sendFlowList = async () => {
  await runAction("Flow list sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const hubID = ensureHubId()
    const executor = resolveTargetId(flowListForm.executorId)
    const req = { req_id: newReqId(), origin_node: sourceID, executor_node: executor }
    await callFlow("ListSimple", sourceID, hubID, req)
  })
}

const sendFlowGet = async () => {
  await runAction("Flow get sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const hubID = ensureHubId()
    const executor = resolveTargetId(flowListForm.executorId)
    const flowId = flowGetForm.flowId.trim()
    if (!flowId) {
      throw new Error(t("Flow ID is required."))
    }
    const req = {
      req_id: newReqId(),
      origin_node: sourceID,
      executor_node: executor,
      flow_id: flowId
    }
    await callFlow("GetSimple", sourceID, hubID, req)
  })
}

const sendFlowRun = async () => {
  await runAction("Flow run sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const hubID = ensureHubId()
    const executor = resolveTargetId(flowListForm.executorId)
    const flowId = flowGetForm.flowId.trim()
    if (!flowId) {
      throw new Error(t("Flow ID is required."))
    }
    const req = {
      req_id: newReqId(),
      origin_node: sourceID,
      executor_node: executor,
      flow_id: flowId
    }
    await callFlow("RunSimple", sourceID, hubID, req)
  })
}

const sendFlowStatus = async () => {
  await runAction("Flow status sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const hubID = ensureHubId()
    const executor = resolveTargetId(flowListForm.executorId)
    const flowId = flowGetForm.flowId.trim()
    if (!flowId) {
      throw new Error(t("Flow ID is required."))
    }
    const runId = flowStatusForm.runId.trim()
    const req: Record<string, any> = {
      req_id: newReqId(),
      origin_node: sourceID,
      executor_node: executor,
      flow_id: flowId
    }
    if (runId) {
      req.run_id = runId
    }
    await callFlow("StatusSimple", sourceID, hubID, req)
  })
}

const sendMgmtListNodes = async () => {
  await runAction("Management list nodes sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(mgmtConfigGetForm.targetId)
    await callManagement("ListNodesSimple", sourceID, targetID)
  })
}

const sendMgmtListSubtree = async () => {
  await runAction("Management list subtree sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(mgmtConfigGetForm.targetId)
    await callManagement("ListSubtreeSimple", sourceID, targetID)
  })
}

const sendMgmtConfigList = async () => {
  await runAction("Management config list sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(mgmtConfigGetForm.targetId)
    await callManagement("ConfigListSimple", sourceID, targetID)
  })
}

const sendMgmtConfigGet = async () => {
  await runAction("Management config get sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(mgmtConfigGetForm.targetId)
    const key = mgmtConfigGetForm.key.trim()
    if (!key) {
      throw new Error(t("Config key is required."))
    }
    await callManagement("ConfigGetSimple", sourceID, targetID, key)
  })
}

const sendMgmtConfigSet = async () => {
  await runAction("Management config set sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const targetID = resolveTargetId(mgmtConfigGetForm.targetId)
    const key = mgmtConfigGetForm.key.trim()
    if (!key) {
      throw new Error(t("Config key is required."))
    }
    await callManagement("ConfigSetSimple", sourceID, targetID, key, mgmtConfigSetForm.value)
  })
}

const sendFileList = async () => {
  await runAction("File list sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const hubID = ensureHubId()
    const targetID = resolveTargetId(fileListForm.targetId)
    await callFile("ListSimple", sourceID, hubID, targetID, fileListForm.dir, fileListForm.recursive)
  })
}

const sendFileRead = async () => {
  await runAction("File read sent.", async () => {
    ensureConnected()
    const sourceID = ensureSourceId()
    const hubID = ensureHubId()
    const targetID = resolveTargetId(fileListForm.targetId)
    const name = fileReadForm.name.trim()
    if (!name) {
      throw new Error(t("File name is required."))
    }
    const maxBytes = parsePositiveInt(fileReadForm.maxBytes, "Max bytes")
    await callFile(
      "ReadTextSimple",
      sourceID,
      hubID,
      targetID,
      fileListForm.dir,
      name,
      maxBytes
    )
  })
}

const syncDefaults = () => {
  const fallbackTarget = defaultTargetId.value
  if (fallbackTarget) {
    const targetText = String(fallbackTarget)
    const formsWithTarget = [
      senderForm,
      receiverForm,
      echoForm,
      registerForm,
      loginForm,
      varSetForm,
      varGetForm,
      varRevokeForm,
      varSubForm,
      varUnsubForm,
      topicPublishForm,
      topicSubForm,
      topicUnsubForm,
      mgmtConfigGetForm,
      fileListForm
    ]
    for (const form of formsWithTarget) {
      if (!form.targetId) {
        form.targetId = targetText
      }
    }
    if (!flowListForm.executorId) {
      flowListForm.executorId = targetText
    }
  }
  if (sessionStore.auth.deviceId) {
    if (!registerForm.deviceId) {
      registerForm.deviceId = sessionStore.auth.deviceId
    }
    if (!loginForm.deviceId) {
      loginForm.deviceId = sessionStore.auth.deviceId
    }
  }
  if (sessionStore.auth.nodeId) {
    const nodeText = String(sessionStore.auth.nodeId)
    if (!loginForm.nodeId) {
      loginForm.nodeId = nodeText
    }
    if (!varSetForm.owner) {
      varSetForm.owner = nodeText
    }
    if (!varGetForm.owner) {
      varGetForm.owner = nodeText
    }
    if (!varRevokeForm.owner) {
      varRevokeForm.owner = nodeText
    }
    if (!varSubForm.owner) {
      varSubForm.owner = nodeText
    }
    if (!varUnsubForm.owner) {
      varUnsubForm.owner = nodeText
    }
  }
}

watch(
  () => [sessionStore.auth.nodeId, sessionStore.auth.hubId, sessionStore.auth.deviceId],
  () => {
    syncDefaults()
  }
)

onMounted(async () => {
  syncDefaults()
  try {
    await presetStore.loadSender()
    await presetStore.loadReceiver()
  } catch (err) {
    console.warn(err)
  }
})
</script>

<template>
  <section class="grid gap-6">
    <div class="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
      <div class="space-y-6">
        <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <CardHeader
            class="items-center"
            :title="t('Topic Stress Sender')"
            :description="t('Publish high-rate topic events to validate throughput.')"
            title-tag="h3"
            title-class="text-lg"
          >
            <template #actions>
              <span class="rounded-full border bg-background px-3 py-1 text-xs text-muted-foreground">
                {{ senderStatus.active ? t("Active") : t("Idle") }}
              </span>
            </template>
          </CardHeader>

          <div class="mt-4 grid gap-4 md:grid-cols-2">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Target Node ID") }}
              </label>
              <input v-model="senderForm.targetId" :class="inputClass" :placeholder="t('Target Node ID')" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Topic") }}
              </label>
              <input v-model="senderForm.topic" :class="inputClass" :placeholder="t('stress')" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Run ID") }}
              </label>
              <input v-model="senderForm.runId" :class="inputClass" :placeholder="t('auto')" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Total Events") }}
              </label>
              <input v-model="senderForm.total" type="number" :class="inputClass" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Payload Size (bytes)") }}
              </label>
              <input v-model="senderForm.payloadSize" type="number" :class="inputClass" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Max / sec (0 = unlimited)") }}
              </label>
              <input v-model="senderForm.maxPerSec" type="number" :class="inputClass" />
            </div>
          </div>

          <div class="mt-4 flex flex-wrap gap-2">
            <Button size="sm" :disabled="busy" @click="startStressSender">{{ t("Start Sender") }}</Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="stopStressSender">
              {{ t("Stop Sender") }}
            </Button>
          </div>

          <div class="mt-4 grid gap-3 text-sm text-muted-foreground md:grid-cols-3">
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Sent") }}</p>
              <p class="text-foreground">{{ senderStatus.sent }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Errors") }}</p>
              <p class="text-foreground">{{ senderStatus.errors }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Rate") }}</p>
              <p class="text-foreground">{{ rateText(senderRate) }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Started") }}</p>
              <p class="text-foreground">{{ senderStatus.startedAt || "-" }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Updated") }}</p>
              <p class="text-foreground">{{ senderStatus.updatedAt || "-" }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Run ID") }}</p>
              <p class="truncate text-foreground">{{ senderStatus.runId || "-" }}</p>
            </div>
          </div>
        </div>

        <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <CardHeader
            class="items-center"
            :title="t('Topic Stress Receiver')"
            :description="t('Subscribe and validate incoming events for loss or corruption.')"
            title-tag="h3"
            title-class="text-lg"
          >
            <template #actions>
              <span class="rounded-full border bg-background px-3 py-1 text-xs text-muted-foreground">
                {{ receiverStatus.active ? t("Active") : t("Idle") }}
              </span>
            </template>
          </CardHeader>

          <div class="mt-4 grid gap-4 md:grid-cols-2">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Target Node ID") }}
              </label>
              <input
                v-model="receiverForm.targetId"
                :class="inputClass"
                :placeholder="t('Target Node ID')"
              />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Topic") }}
              </label>
              <input v-model="receiverForm.topic" :class="inputClass" :placeholder="t('stress')" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Run ID") }}
              </label>
              <input v-model="receiverForm.runId" :class="inputClass" :placeholder="t('auto')" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Expected Total") }}
              </label>
              <input v-model="receiverForm.total" type="number" :class="inputClass" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Payload Size (bytes)") }}
              </label>
              <input v-model="receiverForm.payloadSize" type="number" :class="inputClass" />
            </div>
          </div>

          <div class="mt-4 flex flex-wrap gap-2">
            <Button size="sm" :disabled="busy" @click="startStressReceiver">{{ t("Start Receiver") }}</Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="stopStressReceiver">
              {{ t("Stop Receiver") }}
            </Button>
            <Button size="sm" variant="ghost" :disabled="busy" @click="resetStressReceiver">
              {{ t("Reset Counters") }}
            </Button>
          </div>

          <div class="mt-4 grid gap-3 text-sm text-muted-foreground md:grid-cols-3">
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Received") }}</p>
              <p class="text-foreground">{{ receiverStatus.rx }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Unique") }}</p>
              <p class="text-foreground">{{ receiverStatus.unique }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Missing") }}</p>
              <p class="text-foreground">{{ receiverMissing }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Duplicates") }}</p>
              <p class="text-foreground">{{ receiverStatus.dup }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Corrupt") }}</p>
              <p class="text-foreground">{{ receiverStatus.corrupt }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Invalid") }}</p>
              <p class="text-foreground">{{ receiverStatus.invalid }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Out of Order") }}</p>
              <p class="text-foreground">{{ receiverStatus.outOfOrder }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Last Seq") }}</p>
              <p class="text-foreground">{{ receiverStatus.lastSeq }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Rate") }}</p>
              <p class="text-foreground">{{ rateText(receiverRate) }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="space-y-6">
        <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <CardHeader :title="t('Identity Snapshot')" title-tag="h3" title-class="text-lg" />
          <div class="mt-4 space-y-3 text-sm text-muted-foreground">
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Connection") }}</p>
              <p class="text-foreground">{{ connectionLabel }}</p>
              <p class="text-xs text-muted-foreground">{{ sessionStore.addr || "-" }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Auth") }}</p>
              <p class="text-foreground">{{ authLabel }}</p>
              <p class="text-xs text-muted-foreground">
                {{ sessionStore.auth.lastAuthMessage || "-" }}
              </p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Node") }}</p>
              <p class="text-foreground">{{ sessionStore.auth.nodeId || "-" }}</p>
              <p class="text-xs text-muted-foreground">
                {{ t("Hub: {hubId}", { hubId: sessionStore.auth.hubId || "-" }) }}
              </p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Last Error") }}</p>
              <p class="text-foreground">{{ sessionStore.lastError || "-" }}</p>
            </div>
          </div>
        </div>

        <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
          <CardHeader :title="t('Defaults & Tips')" title-tag="h3" title-class="text-lg" />
          <div class="mt-4 space-y-3 text-sm text-muted-foreground">
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Default Target") }}</p>
              <p class="text-foreground">{{ defaultTargetId || "-" }}</p>
            </div>
            <div class="rounded-lg border border-border/60 bg-background/70 px-3 py-2">
              <p class="text-xs font-semibold uppercase tracking-[0.2em]">{{ t("Device ID") }}</p>
              <p class="text-foreground">{{ sessionStore.auth.deviceId || "-" }}</p>
            </div>
            <p class="text-xs">
              {{ t("Each preset uses its own target fields; update only what you need for debugging.") }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <div class="grid gap-6 lg:grid-cols-2">
      <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <CardHeader :title="t('Node Echo')" title-tag="h3" title-class="text-lg" />
        <div class="mt-4 grid gap-4">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Target Node ID") }}
            </label>
            <input v-model="echoForm.targetId" :class="inputClass" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Message") }}
            </label>
            <input v-model="echoForm.message" :class="inputClass" :placeholder="t('hello')" />
          </div>
          <div class="flex flex-wrap gap-2">
            <Button size="sm" :disabled="busy" @click="sendEcho">{{ t("Send Echo") }}</Button>
          </div>
        </div>
      </div>

      <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <CardHeader :title="t('Register Device')" title-tag="h3" title-class="text-lg" />
        <div class="mt-4 grid gap-4">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Target Node ID") }}
            </label>
            <input v-model="registerForm.targetId" :class="inputClass" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Device ID") }}
            </label>
            <input v-model="registerForm.deviceId" :class="inputClass" />
          </div>
          <div class="flex flex-wrap gap-2">
            <Button size="sm" :disabled="busy" @click="sendRegister">{{ t("Register") }}</Button>
          </div>
        </div>
      </div>

      <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <CardHeader :title="t('Login Device')" title-tag="h3" title-class="text-lg" />
        <div class="mt-4 grid gap-4">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Target Node ID") }}
            </label>
            <input v-model="loginForm.targetId" :class="inputClass" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Device ID") }}
            </label>
            <input v-model="loginForm.deviceId" :class="inputClass" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Node ID") }}
            </label>
            <input v-model="loginForm.nodeId" type="number" :class="inputClass" />
          </div>
          <div class="flex flex-wrap gap-2">
            <Button size="sm" :disabled="busy" @click="sendLogin">{{ t("Login") }}</Button>
          </div>
        </div>
      </div>

      <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <CardHeader :title="t('Set Variable')" title-tag="h3" title-class="text-lg" />
        <div class="mt-4 grid gap-4">
          <div class="grid gap-3 sm:grid-cols-2">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Target Node ID") }}
              </label>
              <input v-model="varSetForm.targetId" :class="inputClass" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Owner Node ID") }}
              </label>
              <input v-model="varSetForm.owner" :class="inputClass" />
            </div>
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Name") }}
            </label>
            <input v-model="varSetForm.name" :class="inputClass" :placeholder="t('status.flag')" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Value") }}
            </label>
            <input v-model="varSetForm.value" :class="inputClass" :placeholder="t('ready')" />
          </div>
          <div class="grid gap-3 sm:grid-cols-2">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Visibility") }}
              </label>
              <select v-model="varSetForm.visibility" :class="inputClass">
                <option value="public">{{ t("public") }}</option>
                <option value="private">{{ t("private") }}</option>
              </select>
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Type") }}
              </label>
              <input v-model="varSetForm.kind" :class="inputClass" :placeholder="t('string')" />
            </div>
          </div>
          <div class="flex flex-wrap gap-2">
            <Button size="sm" :disabled="busy" @click="sendVarSet">{{ t("Set Variable") }}</Button>
          </div>
        </div>
      </div>
      <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <CardHeader :title="t('Get / Revoke')" title-tag="h3" title-class="text-lg" />
        <div class="mt-4 grid gap-4">
          <div class="grid gap-3 sm:grid-cols-2">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Target Node ID") }}
              </label>
              <input v-model="varGetForm.targetId" :class="inputClass" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Owner Node ID") }}
              </label>
              <input v-model="varGetForm.owner" :class="inputClass" />
            </div>
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Name") }}
            </label>
            <input v-model="varGetForm.name" :class="inputClass" />
          </div>
          <div class="flex flex-wrap gap-2">
            <Button size="sm" :disabled="busy" @click="sendVarGet">{{ t("Get Variable") }}</Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="sendVarRevoke">
              {{ t("Revoke Variable") }}
            </Button>
          </div>
        </div>
      </div>

      <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <CardHeader :title="t('Subscribe')" title-tag="h3" title-class="text-lg" />
        <div class="mt-4 grid gap-4">
          <div class="grid gap-3 sm:grid-cols-2">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Target Node ID") }}
              </label>
              <input v-model="varSubForm.targetId" :class="inputClass" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Owner Node ID") }}
              </label>
              <input v-model="varSubForm.owner" :class="inputClass" />
            </div>
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Name") }}
            </label>
            <input v-model="varSubForm.name" :class="inputClass" />
          </div>
          <div class="flex flex-wrap gap-2">
            <Button size="sm" :disabled="busy" @click="sendVarSubscribe">{{ t("Subscribe") }}</Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="sendVarUnsubscribe">
              {{ t("Unsubscribe") }}
            </Button>
          </div>
        </div>
      </div>

      <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <CardHeader :title="t('Publish Event')" title-tag="h3" title-class="text-lg" />
        <div class="mt-4 grid gap-4">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Target Node ID") }}
            </label>
            <input v-model="topicPublishForm.targetId" :class="inputClass" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Topic") }}
            </label>
            <input v-model="topicPublishForm.topic" :class="inputClass" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Name") }}
            </label>
            <input v-model="topicPublishForm.name" :class="inputClass" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Payload") }}
            </label>
            <textarea v-model="topicPublishForm.payload" :class="textAreaClass" rows="4" />
          </div>
          <div class="flex flex-wrap gap-2">
            <Button size="sm" :disabled="busy" @click="sendTopicPublish">{{ t("Publish") }}</Button>
          </div>
        </div>
      </div>

      <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <CardHeader :title="t('Subscribe')" title-tag="h3" title-class="text-lg" />
        <div class="mt-4 grid gap-4">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Target Node ID") }}
            </label>
            <input v-model="topicSubForm.targetId" :class="inputClass" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Topic") }}
            </label>
            <input v-model="topicSubForm.topic" :class="inputClass" />
          </div>
          <div class="flex flex-wrap gap-2">
            <Button size="sm" :disabled="busy" @click="sendTopicSubscribe">{{ t("Subscribe") }}</Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="sendTopicUnsubscribe">
              {{ t("Unsubscribe") }}
            </Button>
          </div>
        </div>
      </div>
      <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <CardHeader :title="t('Flow Commands')" title-tag="h3" title-class="text-lg" />
        <div class="mt-4 grid gap-4">
          <div class="grid gap-3 sm:grid-cols-2">
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Executor Node ID") }}
              </label>
              <input v-model="flowListForm.executorId" :class="inputClass" />
            </div>
            <div>
              <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {{ t("Flow ID") }}
              </label>
              <input v-model="flowGetForm.flowId" :class="inputClass" />
            </div>
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Run ID (optional)") }}
            </label>
            <input v-model="flowStatusForm.runId" :class="inputClass" />
          </div>
          <div class="flex flex-wrap gap-2">
            <Button size="sm" :disabled="busy" @click="sendFlowList">{{ t("List") }}</Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="sendFlowGet">
              {{ t("Get") }}
            </Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="sendFlowRun">
              {{ t("Run") }}
            </Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="sendFlowStatus">
              {{ t("Status") }}
            </Button>
          </div>
        </div>
      </div>

      <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <CardHeader :title="t('Config & Nodes')" title-tag="h3" title-class="text-lg" />
        <div class="mt-4 grid gap-4">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Target Node ID") }}
            </label>
            <input v-model="mgmtConfigGetForm.targetId" :class="inputClass" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Config Key") }}
            </label>
            <input v-model="mgmtConfigGetForm.key" :class="inputClass" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Config Value") }}
            </label>
            <input v-model="mgmtConfigSetForm.value" :class="inputClass" />
          </div>
          <div class="flex flex-wrap gap-2">
            <Button size="sm" :disabled="busy" @click="sendMgmtListNodes">{{ t("List Nodes") }}</Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="sendMgmtListSubtree">
              {{ t("List Subtree") }}
            </Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="sendMgmtConfigList">
              {{ t("Config List") }}
            </Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="sendMgmtConfigGet">
              {{ t("Config Get") }}
            </Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="sendMgmtConfigSet">
              {{ t("Config Set") }}
            </Button>
          </div>
        </div>
      </div>

      <div class="rounded-2xl border bg-card/90 p-6 text-card-foreground shadow-sm">
        <CardHeader :title="t('List & Read')" title-tag="h3" title-class="text-lg" />
        <div class="mt-4 grid gap-4">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Target Node ID") }}
            </label>
            <input v-model="fileListForm.targetId" :class="inputClass" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Directory") }}
            </label>
            <input v-model="fileListForm.dir" :class="inputClass" placeholder="/" />
          </div>
          <label class="flex items-center gap-2 text-xs text-muted-foreground">
            <input v-model="fileListForm.recursive" type="checkbox" class="h-4 w-4 rounded" />
            {{ t("Recursive") }}
          </label>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("File Name (for read)") }}
            </label>
            <input v-model="fileReadForm.name" :class="inputClass" />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {{ t("Max Bytes") }}
            </label>
            <input v-model="fileReadForm.maxBytes" type="number" :class="inputClass" />
          </div>
          <div class="flex flex-wrap gap-2">
            <Button size="sm" :disabled="busy" @click="sendFileList">{{ t("List Dir") }}</Button>
            <Button size="sm" variant="outline" :disabled="busy" @click="sendFileRead">
              {{ t("Read File") }}
            </Button>
          </div>
        </div>
      </div>
    </div>

  </section>
</template>
