<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { MoreHorizontal } from 'lucide-vue-next'
import ModalShell from '@/components/ModalShell.vue'
import { useIsMobile } from '@/composables/useIsMobile'
import { hasOpenEscapeLayer } from '@/composables/useEscapeToClose'

// The "..." of a list row: the actions that are not "open it" (the row itself opens it) in a small popover, so a
// list does not repeat a row of icons. Entries are `.menu-item` buttons in the slot; a click on one closes it.
// On a phone the menu is a bottom sheet with large touch targets, pulled down to close, instead of a popover.
const props = defineProps<{ label: string; heading?: string }>()
const isMobile = useIsMobile()
const open = ref(false)
const root = ref<HTMLElement | null>(null)
function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape' && open.value && !hasOpenEscapeLayer()) open.value = false
}
function onDocumentClick(e: MouseEvent) {
  if (open.value && !isMobile.value && root.value && !root.value.contains(e.target as Node)) open.value = false
}
onMounted(() => {
  window.addEventListener('keydown', onKeyDown)
  document.addEventListener('click', onDocumentClick)
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKeyDown)
  document.removeEventListener('click', onDocumentClick)
})
</script>

<template>
  <div ref="root" class="relative z-10 shrink-0">
    <button type="button" class="tap row-action" :aria-label="label" :title="label" aria-haspopup="menu" :aria-expanded="open" @click="open = !open">
      <MoreHorizontal class="h-4 w-4" aria-hidden="true" />
    </button>
    <Teleport v-if="isMobile" to="body">
      <ModalShell v-model:open="open" :title="heading || label" size="sm" body-class="space-y-1 p-3 pb-6">
        <div role="menu" class="space-y-1 [&_.menu-item]:min-h-12 [&_.menu-item]:gap-3 [&_.menu-item]:px-4 [&_.menu-item]:text-base [&_.menu-item_svg]:h-5 [&_.menu-item_svg]:w-5" @click="open = false">
          <slot />
        </div>
      </ModalShell>
    </Teleport>
    <div v-else-if="open" role="menu" class="absolute right-0 top-full z-40 mt-1 w-44 space-y-0.5 rounded-xl border border-slate-700 bg-slate-800 p-1.5 shadow-2xl" @click="open = false">
      <slot />
    </div>
  </div>
</template>
