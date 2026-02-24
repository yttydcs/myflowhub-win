<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from "vue"
import { cn } from "@/lib/utils"
import { createOverlayID, isTopOverlay, registerOverlay, unregisterOverlay, updateOverlay } from "@/lib/overlayStack"

interface Props {
  open: boolean
  overlayClass?: string
  zIndexClass?: string
  closeOnBackdrop?: boolean
  closeOnEsc?: boolean
  teleport?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  overlayClass: "bg-black/40 p-6",
  zIndexClass: "z-50",
  closeOnBackdrop: false,
  closeOnEsc: true,
  teleport: true
})

const emit = defineEmits<{
  (e: "close"): void
}>()

const overlayID = createOverlayID()

const requestClose = () => {
  emit("close")
}

const register = () => {
  registerOverlay({
    id: overlayID,
    closeOnEsc: Boolean(props.closeOnEsc),
    onEsc: () => {
      if (!props.open) return
      if (!isTopOverlay(overlayID)) return
      requestClose()
    }
  })
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      register()
      return
    }
    unregisterOverlay(overlayID)
  },
  { immediate: true }
)

watch(
  () => props.closeOnEsc,
  (closeOnEsc) => {
    if (!props.open) return
    updateOverlay(overlayID, { closeOnEsc: Boolean(closeOnEsc) })
  }
)

onBeforeUnmount(() => {
  unregisterOverlay(overlayID)
})

const containerClass = computed(() =>
  cn("fixed inset-0 flex items-center justify-center", props.zIndexClass, props.overlayClass)
)

const onBackdropClick = (e: MouseEvent) => {
  if (!props.closeOnBackdrop) return
  if (e.target !== e.currentTarget) return
  requestClose()
}
</script>

<template>
  <Teleport v-if="props.teleport" to="body">
    <div v-if="props.open" :class="containerClass" @click="onBackdropClick">
      <slot />
    </div>
  </Teleport>
  <div v-else-if="props.open" :class="containerClass" @click="onBackdropClick">
    <slot />
  </div>
</template>
