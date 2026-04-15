<script setup lang="ts">
// 本文件实现 Win 前端复用的 `PageHero` 组件。
import { computed, useSlots } from "vue"
import { useRoute } from "vue-router"
import { useI18n } from "@/i18n"

const props = defineProps<{
  title?: string
  description?: string
  surfaceClass?: string
  descriptionClass?: string
}>()

const route = useRoute()
const { t } = useI18n()
const slots = useSlots()

const resolvedTitle = computed(() => {
  const title = props.title?.trim()
  if (title) return title
  return t((route.meta.title as string) ?? "Module")
})

const resolvedDescription = computed(() => {
  const description = props.description?.trim()
  if (description) return description
  return t((route.meta.subtitle as string) ?? "This module is ready for the next step.")
})

const hasActions = computed(() => Boolean(slots.actions))
</script>

<template>
  <section
    :class="[
      'rounded-2xl border p-5 text-card-foreground shadow-sm sm:p-6',
      surfaceClass?.trim() || 'bg-card/90'
    ]"
  >
    <div class="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
      <div class="min-w-0 space-y-2">
        <h1 class="text-2xl font-semibold tracking-tight">{{ resolvedTitle }}</h1>
        <p :class="['max-w-2xl text-sm text-muted-foreground', descriptionClass]">
          {{ resolvedDescription }}
        </p>
      </div>
      <div v-if="hasActions" class="flex flex-wrap items-center gap-2 md:justify-end">
        <slot name="actions" />
      </div>
    </div>
  </section>
</template>
