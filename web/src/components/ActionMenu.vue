<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { MoreHorizontal } from '@/icons'
import ModalShell from '@/components/ModalShell.vue'
import { useIsMobile } from '@/composables/useIsMobile'
import { hasOpenEscapeLayer } from '@/composables/useEscapeToClose'

// The "..." of one card or one page: the actions that are not "open it" in a small popover, so the card does not repeat a
// row of buttons. It is not for list rows (a list row is one tappable card, its actions live in its detail and in the
// selection bar); `ActionMenu.test.ts` lists the files allowed to use it. Entries are `.menu-item` buttons in the slot;
// a click on one closes it. On a phone the menu is a bottom sheet (the heading as title, 48px entries) pulled down to
// close, instead of a popover. `phone` keeps the button visible below `sm`; without it the button is desktop only.
const props = defineProps<{ label: string; heading?: string; phone?: boolean }>()
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
  <div ref="root" class="relative z-10 shrink-0" :class="{ 'max-sm:hidden': !phone }">
    <button type="button" class="tap row-action" :class="{ 'max-sm:!min-w-7 max-sm:!px-0.5 max-sm:-mr-1.5': !phone }" :aria-label="label" :title="label" aria-haspopup="menu" :aria-expanded="open" @click="open = !open">
      <MoreHorizontal class="h-4 w-4" aria-hidden="true" />
    </button>
    <Teleport v-if="isMobile" to="body">
      <ModalShell v-model:open="open" :title="heading || label" size="sm" body-class="space-y-1 p-3 pb-6">
        <div role="menu" class="space-y-1 [&_.menu-item]:min-h-12 [&_.menu-item]:gap-3 [&_.menu-item]:px-4 [&_.menu-item]:text-base [&_.menu-item_svg]:h-5 [&_.menu-item_svg]:w-5" @click="open = false">
          <slot />
        </div>
      </ModalShell>
    </Teleport>
    <div v-else-if="open" role="menu" class="absolute right-0 top-full z-40 mt-1 w-56 space-y-0.5 rounded-xl border border-slate-700 bg-slate-800 p-1.5 shadow-2xl" @click="open = false">
      <slot />
    </div>
  </div>
</template>
