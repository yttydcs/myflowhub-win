<script setup lang="ts">
// 本文件实现 Win 前端的 `ModuleStub` 页面。
import PageHero from "@/components/PageHero.vue"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"

const { t } = useI18n()
</script>

<template>
  <section class="grid gap-6">
    <PageHero surface-class="bg-card/80 backdrop-blur">
      <template #actions>
        <div class="flex flex-wrap gap-2">
          <Button size="sm">{{ t("Primary Action") }}</Button>
          <Button variant="outline" size="sm">{{ t("Secondary") }}</Button>
        </div>
      </template>
    </PageHero>

    <div class="grid gap-4 lg:grid-cols-2">
      <div class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
        <h3 class="text-sm font-semibold">{{ t("Module Checklist") }}</h3>
        <ul class="mt-3 space-y-2 text-sm text-muted-foreground">
          <li>{{ t("Wire session events to the live panel.") }}</li>
          <li>{{ t("Connect Wails services to the action controls.") }}</li>
          <li>{{ t("Ship the final data tables and inspectors.") }}</li>
        </ul>
      </div>
      <div class="rounded-2xl border bg-card/90 p-5 text-card-foreground shadow-sm">
        <h3 class="text-sm font-semibold">{{ t("Next Milestone") }}</h3>
        <p class="mt-3 text-sm text-muted-foreground">
          {{ t("This view is a placeholder until the full module UI is delivered in the next milestones. It already inherits the global layout, navigation, and profile context.") }}
        </p>
      </div>
    </div>
  </section>
</template>
