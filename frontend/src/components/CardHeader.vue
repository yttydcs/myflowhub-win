<script setup lang="ts">
import { computed, useSlots } from "vue"

const props = withDefaults(
  defineProps<{
    title: string | number
    description?: string
    titleTag?: string
    titleClass?: string
    descriptionClass?: string
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
      <component :is="titleTag" :class="['font-semibold', titleClass]">
        {{ title }}
      </component>
      <p v-if="hasDescription" :class="['text-sm text-muted-foreground', descriptionClass]">
        {{ description }}
      </p>
    </div>
    <div v-if="hasActions" class="flex flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
