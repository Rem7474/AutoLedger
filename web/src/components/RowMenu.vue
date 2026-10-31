<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { MoreHorizontal } from 'lucide-vue-next'
import { hasOpenEscapeLayer } from '@/composables/useEscapeToClose'

// The "..." of a list row: the actions that are not "open it" (the row itself opens it) in a small popover, so a
// list does not repeat a row of icons. Entries are `.menu-item` buttons in the slot; a click on one closes it.
defineProps<{ label: string }>()
const open = ref(false)
const root = ref<HTMLElement | null>(null)
function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape' && open.value && !hasOpenEscapeLayer()) open.value = false
}
function onDocumentClick(e: MouseEvent) {
  if (open.value && root.value && !root.value.contains(e.target as Node)) open.value = false
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
    <div v-if="open" role="menu" class="absolute right-0 top-full z-40 mt-1 w-44 space-y-0.5 rounded-xl border border-slate-700 bg-slate-800 p-1.5 shadow-2xl" @click="open = false">
      <slot />
    </div>
  </div>
</template>
