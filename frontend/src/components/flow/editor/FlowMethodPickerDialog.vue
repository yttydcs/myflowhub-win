<script setup lang="ts">
import { computed } from "vue"
import { Button } from "@/components/ui/button"
import { Overlay } from "@/components/ui/overlay"
import CardHeader from "@/components/CardHeader.vue"
import { useI18n } from "@/i18n"
import type { ExecCapabilityRoute } from "@/stores/flow"

interface Props {
  open: boolean
  effectiveExecutorNode: number
  capabilityQueryNodeLabel: string
  selectedTargetLabel: string
  queryNodeIdDraft: string
  methodSearch: string
  loading: boolean
  capabilityCount: number
  filteredCapabilities: ExecCapabilityRoute[]
  pendingCapabilityKey: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: "close"): void
  (e: "update:queryNodeIdDraft", value: string): void
  (e: "update:methodSearch", value: string): void
  (e: "refresh"): void
  (e: "select-capability", key: string): void
  (e: "apply"): void
}>()

const { t } = useI18n()
const dialogTitleId = "flow-method-dialog-title"
const dialogDescriptionId = "flow-method-dialog-description"
const queryNodeInputId = "flow-method-query-node"
const queryNodeHelpId = "flow-method-query-node-help"
const filterInputId = "flow-method-filter"

const capabilitySummary = computed(() => {
  if (!props.capabilityCount) {
    return t("No capability loaded yet. Refresh to query the selected node.")
  }
  if (props.filteredCapabilities.length === props.capabilityCount) {
    return t("Showing all {count} capabilities.", { count: props.capabilityCount })
  }
  return t("Showing {matched} of {total} capabilities.", {
    matched: props.filteredCapabilities.length,
    total: props.capabilityCount
  })
})

const permissionPreview = (route: ExecCapabilityRoute) => route.permissions.slice(0, 2)

const tagPreview = (route: ExecCapabilityRoute) => Object.entries(route.tags).slice(0, 2)
</script>

<template>
  <Overlay
    :open="props.open"
    :trap-focus="true"
    :initial-focus-selector="`#${filterInputId}`"
    overlayClass="bg-slate-950/60 p-4"
    zIndexClass="z-40"
    closeOnBackdrop
    @close="emit('close')"
  >
    <div
      class="flex max-h-[80vh] w-full max-w-4xl flex-col rounded-2xl border bg-card p-6 text-card-foreground shadow-2xl"
      role="dialog"
      aria-modal="true"
      :aria-labelledby="dialogTitleId"
      :aria-describedby="dialogDescriptionId"
    >
      <CardHeader
        class="items-start"
        :title="t('Select Capability')"
        :description="t('Pick a registered capability and the editor will keep method and target aligned.')"
        title-class="text-lg"
        :title-id="dialogTitleId"
        :description-id="dialogDescriptionId"
      >
        <template #actions>
          <div class="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            <span class="rounded-full border border-border/60 px-3 py-1">
              {{ t("Executor {nodeId}", { nodeId: props.effectiveExecutorNode || "-" }) }}
            </span>
            <span class="rounded-full border border-border/60 px-3 py-1">
              {{ t("Query Node {nodeId}", { nodeId: props.capabilityQueryNodeLabel }) }}
            </span>
            <span class="rounded-full border border-border/60 px-3 py-1">
              {{ props.selectedTargetLabel }}
            </span>
          </div>
        </template>
      </CardHeader>

      <div class="mt-5 flex flex-wrap items-end gap-3">
        <div class="min-w-[200px] max-w-[240px] flex-1">
          <label :for="queryNodeInputId" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            {{ t("Query Node ID") }}
          </label>
          <input
            :id="queryNodeInputId"
            :value="props.queryNodeIdDraft"
            class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            inputmode="numeric"
            :placeholder="t('Current executor')"
            :aria-describedby="queryNodeHelpId"
            @input="emit('update:queryNodeIdDraft', ($event.target as HTMLInputElement).value)"
          />
          <p :id="queryNodeHelpId" class="mt-1 text-[11px] text-muted-foreground">
            {{ t("Used only for capability lookup. It will not be written back into the call node.") }}
          </p>
        </div>
        <div class="min-w-[240px] flex-[2]">
          <label :for="filterInputId" class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            {{ t("Filter") }}
          </label>
          <input
            :id="filterInputId"
            :value="props.methodSearch"
            class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            :placeholder="t('Search method / node / permission / tag')"
            @input="emit('update:methodSearch', ($event.target as HTMLInputElement).value)"
          />
        </div>
        <Button variant="outline" :disabled="props.loading" @click="emit('refresh')">
          {{ props.loading ? t("Refreshing...") : t("Refresh Capabilities") }}
        </Button>
      </div>

      <p class="mt-3 text-xs text-muted-foreground">
        {{ capabilitySummary }}
      </p>

      <div class="mt-5 flex-1 overflow-y-auto rounded-xl border border-border/70 bg-background/80">
        <div
          v-if="props.loading && !props.capabilityCount"
          class="px-4 py-10 text-center text-sm text-muted-foreground"
        >
          {{ t("Loading capability list...") }}
        </div>
        <div v-else-if="!props.filteredCapabilities.length" class="px-4 py-10 text-center text-sm text-muted-foreground">
          {{
            props.capabilityCount
              ? t("No capability matched the current filter.")
              : t("No capability loaded yet. Refresh to query the selected node.")
          }}
        </div>
        <div v-else class="divide-y divide-border/60">
          <button
            v-for="route in props.filteredCapabilities"
            :key="route.key"
            type="button"
            class="flex w-full items-start justify-between gap-4 px-4 py-4 text-left transition hover:bg-muted/50"
            :class="props.pendingCapabilityKey === route.key ? 'bg-muted/70' : ''"
            :aria-pressed="props.pendingCapabilityKey === route.key"
            @click="emit('select-capability', route.key)"
          >
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <p class="break-all text-sm font-semibold">{{ route.method }}</p>
                <span class="rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-[0.2em] text-muted-foreground">
                  {{ route.providerNode === props.effectiveExecutorNode ? t("Self") : t("Remote") }}
                </span>
                <span
                  class="rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-[0.2em]"
                  :class="
                    route.inputSchema
                      ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-700'
                      : 'border-border/70 text-muted-foreground'
                  "
                >
                  {{ route.inputSchema ? t("Input schema") : t("No schema") }}
                </span>
              </div>
              <p class="mt-1 text-xs text-muted-foreground">
                {{ t("Provider {nodeId}", { nodeId: route.providerNode }) }}
                <span v-if="route.viaNode > 0"> {{ t("via {nodeId}", { nodeId: route.viaNode }) }}</span>
                <span v-if="route.version"> · {{ route.version }}</span>
              </p>
              <div class="mt-3 flex flex-wrap gap-2 text-[11px] text-muted-foreground">
                <span
                  v-if="route.defaultTimeoutMs > 0"
                  class="rounded-full border border-border/70 bg-background px-2 py-1"
                >
                  {{ t("Default timeout {ms} ms", { ms: route.defaultTimeoutMs }) }}
                </span>
                <span
                  v-for="permission in permissionPreview(route)"
                  :key="`${route.key}-perm-${permission}`"
                  class="rounded-full border border-border/70 bg-background px-2 py-1"
                >
                  {{ permission }}
                </span>
                <span
                  v-for="[key, value] in tagPreview(route)"
                  :key="`${route.key}-tag-${key}`"
                  class="rounded-full border border-border/70 bg-background px-2 py-1"
                >
                  {{ key }}: {{ value }}
                </span>
              </div>
            </div>
            <span
              class="shrink-0 rounded-full border px-3 py-1 text-xs"
              :class="
                props.pendingCapabilityKey === route.key
                  ? 'border-primary text-primary'
                  : 'border-border/70 text-muted-foreground'
              "
            >
              {{ props.pendingCapabilityKey === route.key ? t("Selected") : t("Choose") }}
            </span>
          </button>
        </div>
      </div>

      <div class="mt-5 flex flex-wrap items-center justify-between gap-3">
        <p class="text-xs text-muted-foreground">
          {{ t("Applying a capability updates the method and hidden call target. The query node stays temporary.") }}
        </p>
        <div class="flex gap-2">
          <Button variant="outline" @click="emit('close')">{{ t("Cancel") }}</Button>
          <Button :disabled="!props.pendingCapabilityKey" @click="emit('apply')">{{ t("Apply Method") }}</Button>
        </div>
      </div>
    </div>
  </Overlay>
</template>
