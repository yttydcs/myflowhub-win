<script setup lang="ts">
// 本文件实现 Win 前端复用的 `Overlay` UI 组件。
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue"
import { cn } from "@/lib/utils"
import { createOverlayID, isTopOverlay, registerOverlay, unregisterOverlay, updateOverlay } from "@/lib/overlayStack"

interface Props {
  open: boolean
  overlayClass?: string
  zIndexClass?: string
  closeOnBackdrop?: boolean
  closeOnEsc?: boolean
  teleport?: boolean
  trapFocus?: boolean
  initialFocusSelector?: string
  restoreFocus?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  overlayClass: "bg-black/40 p-6",
  zIndexClass: "z-50",
  closeOnBackdrop: false,
  closeOnEsc: true,
  teleport: true,
  trapFocus: false,
  initialFocusSelector: "",
  restoreFocus: true
})

const emit = defineEmits<{
  (e: "close"): void
}>()

const overlayID = createOverlayID()
const containerRef = ref<HTMLElement | null>(null)
let previousActiveElement: HTMLElement | null = null

const FOCUSABLE_SELECTOR = [
  "button:not([disabled])",
  "[href]",
  "input:not([disabled])",
  "select:not([disabled])",
  "textarea:not([disabled])",
  "[tabindex]:not([tabindex='-1'])"
].join(", ")

const requestClose = () => {
  emit("close")
}

const isFocusableElementVisible = (element: HTMLElement) =>
  !element.hasAttribute("disabled") &&
  element.getAttribute("aria-hidden") !== "true" &&
  window.getComputedStyle(element).display !== "none" &&
  window.getComputedStyle(element).visibility !== "hidden"

const getFocusableElements = () => {
  const root = containerRef.value
  if (!root) return []
  return Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter(isFocusableElementVisible)
}

const focusInitialElement = async () => {
  if (!props.trapFocus) return
  await nextTick()
  const root = containerRef.value
  if (!root) return

  const preferred =
    props.initialFocusSelector.trim() ? root.querySelector<HTMLElement>(props.initialFocusSelector.trim()) : null
  if (preferred && isFocusableElementVisible(preferred)) {
    preferred.focus()
    return
  }

  const first = getFocusableElements()[0]
  if (first) {
    first.focus()
    return
  }

  root.focus()
}

const restorePreviousFocus = () => {
  if (!props.trapFocus || !props.restoreFocus) {
    previousActiveElement = null
    return
  }
  if (previousActiveElement && previousActiveElement.isConnected) {
    previousActiveElement.focus()
  }
  previousActiveElement = null
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
  (open, wasOpen) => {
    if (open) {
      previousActiveElement = document.activeElement instanceof HTMLElement ? document.activeElement : null
      register()
      void focusInitialElement()
      return
    }
    unregisterOverlay(overlayID)
    if (wasOpen) {
      restorePreviousFocus()
    }
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
  if (props.open) {
    restorePreviousFocus()
  }
})

const containerClass = computed(() =>
  cn("fixed inset-0 flex items-center justify-center", props.zIndexClass, props.overlayClass)
)

const onBackdropClick = (e: MouseEvent) => {
  if (!props.closeOnBackdrop) return
  if (e.target !== e.currentTarget) return
  requestClose()
}

const onKeydown = (event: KeyboardEvent) => {
  if (!props.open || !props.trapFocus || event.key !== "Tab") return
  const root = containerRef.value
  if (!root) return

  const focusable = getFocusableElements()
  if (!focusable.length) {
    event.preventDefault()
    root.focus()
    return
  }

  const active = document.activeElement instanceof HTMLElement ? document.activeElement : null
  const first = focusable[0]
  const last = focusable[focusable.length - 1]

  if (event.shiftKey) {
    if (!active || active === first || !root.contains(active)) {
      event.preventDefault()
      last.focus()
    }
    return
  }

  if (!active || active === last || !root.contains(active)) {
    event.preventDefault()
    first.focus()
  }
}
</script>

<template>
  <Teleport v-if="props.teleport" to="body">
    <div
      v-if="props.open"
      ref="containerRef"
      :class="containerClass"
      :tabindex="props.trapFocus ? -1 : undefined"
      @click="onBackdropClick"
      @keydown="onKeydown"
    >
      <slot />
    </div>
  </Teleport>
  <div
    v-else-if="props.open"
    ref="containerRef"
    :class="containerClass"
    :tabindex="props.trapFocus ? -1 : undefined"
    @click="onBackdropClick"
    @keydown="onKeydown"
  >
    <slot />
  </div>
</template>
