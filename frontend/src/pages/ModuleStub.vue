<script setup lang="ts">
import { computed } from "vue"
import { useRoute } from "vue-router"
import { Button } from "@/components/ui/button"

const route = useRoute()
const title = computed(() => (route.meta.title as string) ?? "Module")
const subtitle = computed(
  () =>
    (route.meta.subtitle as string) ??
    "This module is ready for UI build-out and service wiring."
)
</script>

<template>
  <section class="grid gap-6">
    <div class="rounded-2xl border bg-card/80 p-6 text-card-foreground shadow-sm backdrop-blur">
      <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div class="space-y-1">
          <p class="text-xs font-semibold uppercase tracking-[0.3em] text-muted-foreground">
            Tool Console
          </p>
          <h2 class="text-2xl font-semibold">{{ title }}</h2>
          <p class="text-sm text-muted-foreground">
            {{ subtitle }}
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <Button size="sm">Primary Action</Button>
          <Button variant="outline" size="sm">Secondary</Button>
        </div>
      </div>
    </div>

    <div class="grid gap-4 lg:grid-cols-2">
      <div class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
        <h3 class="text-sm font-semibold">Module Checklist</h3>
        <ul class="mt-3 space-y-2 text-sm text-muted-foreground">
          <li>Wire session events to the live panel.</li>
          <li>Connect Wails services to the action controls.</li>
          <li>Ship the final data tables and inspectors.</li>
        </ul>
      </div>
      <div class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
        <h3 class="text-sm font-semibold">Next Milestone</h3>
        <p class="mt-3 text-sm text-muted-foreground">
          This view is a placeholder until the full module UI is delivered in the
          next milestones. It already inherits the global layout, navigation, and
          profile context.
        </p>
      </div>
    </div>
  </section>
</template>
