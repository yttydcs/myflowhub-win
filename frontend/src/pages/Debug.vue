<script setup lang="ts">
import { computed, reactive, ref } from "vue"
import { Button } from "@/components/ui/button"
import { useSessionStore } from "@/stores/session"

type WailsBinding = (...args: any[]) => Promise<any>

const callSession = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.session?.SessionService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(`Session binding '${method}' unavailable`)
  }
  return fn(...args)
}

const callDebug = async <T>(method: string, ...args: any[]): Promise<T> => {
  const api = (window as any)?.go?.debug?.DebugService
  const fn: WailsBinding | undefined = api?.[method]
  if (!fn) {
    throw new Error(`Debug binding '${method}' unavailable`)
  }
  return fn(...args)
}

const sessionStore = useSessionStore()

const form = reactive({
  addr: "127.0.0.1:9000",
  nodeName: "debugclient",
  major: "3",
  subProto: "1",
  sourceId: "1",
  targetId: "0",
  flags: "0",
  msgId: "",
  timestamp: "",
  payload: "",
  payloadHex: false
})

const message = ref("")
const busy = ref(false)

const connectionLabel = computed(() => (sessionStore.connected ? "Connected" : "Disconnected"))

const parseUint = (raw: string, field: string) => {
  const trimmed = raw.trim()
  if (!trimmed) return 0
  const parsed = Number.parseInt(trimmed, 10)
  if (Number.isNaN(parsed) || parsed < 0) {
    throw new Error(`${field} must be a non-negative integer.`)
  }
  return parsed
}

const connect = async () => {
  message.value = ""
  if (busy.value) return
  busy.value = true
  try {
    await callSession("Connect", form.addr)
    await callSession("LoginLegacy", form.nodeName)
  } catch (err) {
    console.warn(err)
    message.value = (err as Error)?.message || "Failed to connect."
  } finally {
    busy.value = false
  }
}

const disconnect = async () => {
  message.value = ""
  try {
    await callSession("Close")
  } catch (err) {
    console.warn(err)
    message.value = (err as Error)?.message || "Failed to disconnect."
  }
}

const sendFrame = async () => {
  message.value = ""
  try {
    const frame = {
      major: parseUint(form.major, "Major"),
      sub_proto: parseUint(form.subProto, "SubProto"),
      source_id: parseUint(form.sourceId, "SourceID"),
      target_id: parseUint(form.targetId, "TargetID"),
      flags: parseUint(form.flags, "Flags"),
      msg_id: parseUint(form.msgId, "MsgID"),
      timestamp: parseUint(form.timestamp, "Timestamp")
    }
    await callDebug("Send", frame, form.payload, form.payloadHex)
    message.value = "Frame sent."
  } catch (err) {
    console.warn(err)
    message.value = (err as Error)?.message || "Failed to send frame."
  }
}
</script>

<template>
  <section class="space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
          Debug Console
        </p>
        <h1 class="text-2xl font-semibold">Raw Frame Sender</h1>
        <p class="text-sm text-muted-foreground">
          Connect, login, and send custom headers and payloads.
        </p>
      </div>
      <div class="flex items-center gap-2 rounded-full border bg-card/90 px-3 py-1 text-xs text-muted-foreground">
        <span class="font-semibold uppercase tracking-[0.2em]">Session</span>
        <span class="text-foreground">{{ connectionLabel }}</span>
      </div>
    </div>

    <div class="grid gap-6 lg:grid-cols-[280px_minmax(0,1fr)]">
      <section class="rounded-2xl border bg-card/90 p-4 text-card-foreground shadow-sm">
        <h2 class="text-sm font-semibold">Connection</h2>
        <div class="mt-4 space-y-3">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Server Address
            </label>
            <input
              v-model="form.addr"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Node Name
            </label>
            <input
              v-model="form.nodeName"
              class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>
          <div class="flex flex-wrap gap-2">
            <Button size="sm" :disabled="busy" @click="connect">Connect</Button>
            <Button size="sm" variant="outline" @click="disconnect">Disconnect</Button>
          </div>
        </div>
      </section>

      <section class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
        <h2 class="text-sm font-semibold">Header</h2>
        <div class="mt-4 grid gap-4 md:grid-cols-3">
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Major
            </label>
            <input
              v-model="form.major"
              class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              SubProto
            </label>
            <input
              v-model="form.subProto"
              class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Flags
            </label>
            <input
              v-model="form.flags"
              class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Source ID
            </label>
            <input
              v-model="form.sourceId"
              class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Target ID
            </label>
            <input
              v-model="form.targetId"
              class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Msg ID (optional)
            </label>
            <input
              v-model="form.msgId"
              class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>
          <div>
            <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Timestamp (optional)
            </label>
            <input
              v-model="form.timestamp"
              class="mt-2 h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            />
          </div>
        </div>

        <div class="mt-6">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-semibold">Payload</h3>
            <label class="flex items-center gap-2 text-xs text-muted-foreground">
              <input v-model="form.payloadHex" type="checkbox" class="h-4 w-4 rounded border" />
              Hex payload
            </label>
          </div>
          <textarea
            v-model="form.payload"
            rows="6"
            class="mt-3 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            placeholder="Payload text or hex (when Hex payload is enabled)"
          />
          <div class="mt-4 flex justify-end">
            <Button @click="sendFrame">Send Frame</Button>
          </div>
        </div>
      </section>
    </div>

    <p v-if="message" class="text-sm text-rose-600">{{ message }}</p>
  </section>
</template>
