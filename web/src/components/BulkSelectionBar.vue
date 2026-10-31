<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { X, MoreHorizontal } from 'lucide-vue-next'
import { hasOpenEscapeLayer } from '@/composables/useEscapeToClose'
import { useIsMobile } from '@/composables/useIsMobile'
import ModalShell from '@/components/ModalShell.vue'

const props = withDefaults(
  defineProps<{
    count: number
    // What is selected: picks the wording of the counter
    noun?: 'item' | 'tire' | 'carpool' | 'drive'
    offScreenCount?: number
    metricsSummary?: string
    disabled?: boolean
  }>(),
  {
    noun: 'item',
    offScreenCount: 0,
    disabled: false,
  }
)

const emit = defineEmits<{
  (e: 'clear'): void
}>()

// On a phone the bar sits at the bottom edge over the navigation, in reach of the thumb: the counter and a close button
// on top, the actions in one scrolling row, and "More" opens a bottom sheet.
const isMobile = useIsMobile()

// The "More" popover holds the occasional actions, so the bar keeps one primary action and a few secondary ones
const moreOpen = ref(false)
const moreRoot = ref<HTMLElement | null>(null)

function onKeyDown(e: KeyboardEvent) {
  if (e.key !== 'Escape' || hasOpenEscapeLayer()) return
  if (moreOpen.value) {
    moreOpen.value = false
    return
  }
  if (props.count > 0 && !props.disabled) emit('clear')
}

function closeOnAction(e: MouseEvent) {
  if ((e.target as HTMLElement).closest('button')) moreOpen.value = false
}

function onDocumentClick(e: MouseEvent) {
  if (moreOpen.value && !isMobile.value && moreRoot.value && !moreRoot.value.contains(e.target as Node)) moreOpen.value = false
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
  <template v-if="count > 0">
    <div
      class="fixed inset-x-0 bottom-0 z-[55] flex flex-col gap-2 rounded-t-2xl border border-b-0 border-slate-700/80 bg-slate-800/95 px-3 pt-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] shadow-2xl backdrop-blur-md transition-all sm:sticky sm:inset-x-auto sm:bottom-auto sm:top-16 sm:z-30 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between sm:gap-3 sm:rounded-2xl sm:border-b sm:p-3"
    >
      <!-- Left info & summary -->
      <div class="flex items-center gap-2.5 min-w-0" aria-live="polite">
        <span class="px-2.5 py-0.5 bg-rose-500/20 text-rose-400 font-bold text-xs rounded-lg border border-rose-500/30">
          {{ count }}
        </span>
        <span class="text-xs sm:text-sm font-semibold text-slate-200">
          {{ $t(`shell.bulkSelectionBar.selected.${noun}`, count) }}
        </span>

        <!-- Optional metrics preview (e.g. unified trip metrics) -->
        <span
          v-if="metricsSummary"
          class="hidden sm:inline-flex items-center text-xs text-slate-400 border-l border-slate-700/80 pl-2.5 ml-0.5 font-medium"
        >
          {{ metricsSummary }}
        </span>

        <!-- Off-page indicator -->
        <span v-if="offScreenCount > 0" class="text-xs text-slate-400">
          {{ $t('shell.bulkSelectionBar.outOfView', { offScreenCount }) }}
        </span>

        <button
          type="button"
          class="tap ml-auto rounded-lg p-1.5 text-slate-300 hover:bg-slate-700 sm:hidden"
          :disabled="disabled"
          :aria-label="$t('shell.bulkSelectionBar.clearTheSelectionEsc')"
          @click="emit('clear')"
        >
          <X class="h-5 w-5" aria-hidden="true" />
        </button>
      </div>

      <!-- Right contextual actions + Clear button -->
      <div class="flex items-center gap-2 max-sm:-mx-3 max-sm:overflow-x-auto max-sm:px-3 max-sm:[&>*]:shrink-0 max-sm:[&_.btn]:whitespace-nowrap sm:flex-wrap sm:shrink-0">
        <slot />

        <div v-if="$slots.more" ref="moreRoot" class="relative">
          <button
            type="button"
            class="btn btn-secondary"
            :aria-expanded="moreOpen"
            aria-haspopup="menu"
            @click="moreOpen = !moreOpen"
          >
            <MoreHorizontal class="w-3.5 h-3.5" />
            <span>{{ $t('shell.bulkSelectionBar.more') }}</span>
          </button>
          <div
            v-if="moreOpen && !isMobile"
            role="menu"
            class="absolute right-0 top-full z-40 mt-2 w-64 space-y-0.5 rounded-xl border border-slate-700 bg-slate-800 p-1.5 shadow-2xl"
            @click="closeOnAction"
            @change="moreOpen = false"
          >
            <slot name="more" />
          </div>
        </div>

        <button
          type="button"
          @click="emit('clear')"
          :disabled="disabled"
          class="btn btn-secondary max-sm:hidden"
          :title="$t('shell.bulkSelectionBar.clearTheSelectionEsc')"
        >
          <X class="w-3.5 h-3.5" />
          <span>{{ $t('common.cancel') }}</span>
        </button>
      </div>
    </div>
    <!-- Keeps the end of the list clear of the bar -->
    <div class="h-24 sm:hidden" aria-hidden="true"></div>

    <Teleport v-if="isMobile" to="body">
      <ModalShell v-model:open="moreOpen" :title="$t('shell.bulkSelectionBar.more')" size="sm">
        <div role="menu" class="space-y-1 [&_.menu-item]:min-h-12 [&_.menu-item]:gap-3 [&_.menu-item]:px-4 [&_.menu-item]:text-base [&_.menu-item_svg]:h-5 [&_.menu-item_svg]:w-5" @click="closeOnAction" @change="moreOpen = false">
          <slot name="more" />
        </div>
      </ModalShell>
    </Teleport>
  </template>
</template>
