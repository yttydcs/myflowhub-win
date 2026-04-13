<script setup lang="ts">
// Context: renders the shared card header component used by Win frontend pages.
import { computed, useSlots } from "vue"

const props = withDefaults(
  defineProps<{
    title: string | number
    description?: string
    titleTag?: string
    titleClass?: string
    descriptionClass?: string
    titleId?: string
    descriptionId?: string
  }>(),
  {
    titleTag: "h2"
  }
)

const slots = useSlots()

const hasActions = computed(() => Boolean(slots.actions))
const hasDescription = computed(() => Boolean(props.description?.trim()))
</script>

<template>
  <div class="flex flex-wrap justify-between gap-3">
    <div :class="['min-w-0', hasDescription ? 'space-y-1.5' : '']">
      <component :is="titleTag" :id="titleId" :class="['font-semibold', titleClass]">
        {{ title }}
      </component>
      <p v-if="hasDescription" :id="descriptionId" :class="['text-sm text-muted-foreground', descriptionClass]">
        {{ description }}
      </p>
    </div>
    <div v-if="hasActions" class="flex flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
