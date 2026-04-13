<script setup lang="ts">
// Context: renders the flow edge inspector panel or dialog used by the Flow editor.
import { computed } from "vue"
import { X } from "lucide-vue-next"
import CardHeader from "@/components/CardHeader.vue"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"
import type { FlowEdge, FlowNodeKind } from "@/stores/flow"

const props = defineProps<{
  selectedEdge: FlowEdge | null
  sourceNodeKind: FlowNodeKind | null
}>()

const emit = defineEmits<{
  (event: "close"): void
  (event: "update:edge-case", value: string): void
}>()

const { t } = useI18n()

const isBranchSource = computed(() => props.sourceNodeKind === "branch")

const updateEdgeCase = (event: Event) => {
  emit("update:edge-case", String((event.target as HTMLInputElement | null)?.value ?? ""))
}
</script>

<template>
  <aside
    class="pointer-events-auto h-full w-full max-w-[420px] border-l border-border/70 bg-card/95 shadow-xl backdrop-blur"
    role="complementary"
    :aria-label="t('Edge Inspector')"
  >
    <CardHeader class="border-b border-border/70 px-5 py-4" :title="t('Edge Inspector')" title-class="text-base">
      <template #actions>
        <Button variant="ghost" size="icon" @click="emit('close')">
          <X class="h-4 w-4" />
          <span class="sr-only">{{ t("Close") }}</span>
        </Button>
      </template>
    </CardHeader>

    <div class="space-y-4 p-5">
      <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
        <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
          {{ t("Connection") }}
        </p>
        <p class="mt-2 text-sm font-semibold break-all">
          {{ selectedEdge?.from || "-" }} → {{ selectedEdge?.to || "-" }}
        </p>
      </div>

      <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
        <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
          {{ t("Source Kind") }}
        </p>
        <p class="mt-2 text-sm text-muted-foreground">
          {{ sourceNodeKind || t("Unknown") }}
        </p>
      </div>

      <div class="rounded-xl border border-border/70 bg-muted/20 p-4">
        <label class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
          {{ t("Edge Case") }}
        </label>
        <input
          :value="selectedEdge?.case ?? ''"
          class="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
          :disabled="!isBranchSource"
          :placeholder="isBranchSource ? 'approved' : t('Only branch outgoing edges use edge.case')"
          @input="updateEdgeCase"
        />
        <p class="mt-2 text-[11px] text-muted-foreground">
          {{
            isBranchSource
              ? t("Branch routes match outgoing edges by edge.case. Leave blank only if this edge is not part of branch routing.")
              : t("This edge does not originate from a branch node, so edge.case is not used.")
          }}
        </p>
      </div>
    </div>
  </aside>
</template>
