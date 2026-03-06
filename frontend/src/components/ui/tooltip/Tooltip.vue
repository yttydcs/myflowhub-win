<script setup lang="ts">
import { TooltipContent, TooltipPortal, TooltipRoot, TooltipTrigger } from "radix-vue"
import { cn } from "@/lib/utils"

interface Props {
  content: string
  side?: "top" | "right" | "bottom" | "left"
  sideOffset?: number
  delayDuration?: number
  contentClass?: string
}

const props = withDefaults(defineProps<Props>(), {
  side: "bottom",
  sideOffset: 8,
  delayDuration: 120,
  contentClass: ""
})
</script>

<template>
  <TooltipRoot :delay-duration="props.delayDuration">
    <TooltipTrigger as-child>
      <slot />
    </TooltipTrigger>
    <TooltipPortal>
      <TooltipContent
        :side="props.side"
        :side-offset="props.sideOffset"
        :class="
          cn(
            'z-50 max-w-80 rounded-md border border-border/70 bg-popover px-2 py-1.5 text-xs text-popover-foreground shadow-md',
            'data-[state=delayed-open]:animate-in data-[state=delayed-open]:fade-in-0 data-[state=delayed-open]:zoom-in-95',
            'data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95',
            'data-[side=bottom]:slide-in-from-top-1 data-[side=top]:slide-in-from-bottom-1',
            'data-[side=left]:slide-in-from-right-1 data-[side=right]:slide-in-from-left-1',
            props.contentClass
          )
        "
      >
        {{ props.content }}
      </TooltipContent>
    </TooltipPortal>
  </TooltipRoot>
</template>
