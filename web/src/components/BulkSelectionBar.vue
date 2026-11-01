<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { X, MoreVertical } from 'lucide-vue-next'
import { barActions, phoneShortcuts, type SelectionAction } from '@/utils/selectionActions'
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
    // The actions that apply to the selection; one that does not stays visible but disabled, with its reason
    actions?: SelectionAction[]
  }>(),
  {
    noun: 'item',
    offScreenCount: 0,
    disabled: false,
    actions: () => [],
  }
)

const emit = defineEmits<{
  (e: 'clear'): void
}>()

// On a phone the bar is a contextual app bar that takes the place of the header while something is selected: the
// close button and the counter on the left, the first actions as icons and "More" on the right; "More" opens a sheet
// with every action, the ones that do not apply greyed with their reason.
const isMobile = useIsMobile()

// The "More" popover holds the occasional actions, so the bar keeps one primary action and a few secondary ones
const moreOpen = ref(false)
const hasMore = computed(() => props.actions.length > 0 && (isMobile.value || props.actions.some((a) => a.overflow)))
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
    <Teleport to="body" :disabled="!isMobile">
    <div
      class="fixed inset-x-0 top-0 z-[55] flex items-center justify-between gap-1 rounded-b-2xl border border-t-0 border-slate-800 bg-slate-900 px-2 pb-2 pt-[max(0.25rem,env(safe-area-inset-top))] shadow-2xl shadow-black/40 transition-all sm:sticky sm:inset-x-auto sm:bottom-auto sm:top-16 sm:z-30 sm:flex-wrap sm:gap-3 sm:rounded-2xl sm:border-t sm:border-slate-700/80 sm:bg-slate-800/95 sm:p-3 sm:shadow-2xl sm:backdrop-blur-md"
    >
      <!-- Phone: close, counter, shortcuts and "more" on one line. Desktop: counter on the left, buttons on the right -->
      <div class="flex min-w-0 flex-1 items-center gap-1 sm:flex-none sm:gap-2.5" aria-live="polite">
        <button
          type="button"
          class="tap rounded-full p-1.5 text-slate-300 hover:bg-slate-800 sm:hidden"
          :disabled="disabled"
          :aria-label="$t('shell.bulkSelectionBar.clearTheSelectionEsc')"
          @click="emit('clear')"
        >
          <X class="h-5 w-5" aria-hidden="true" />
        </button>
        <span class="px-2.5 py-0.5 bg-rose-500/20 text-rose-400 font-bold text-xs rounded-lg border border-rose-500/30 max-sm:hidden">
          {{ count }}
        </span>
        <span class="truncate text-lg font-medium text-white sm:text-sm sm:font-semibold sm:text-slate-200">
          <span class="sm:hidden">{{ count }}</span>
          {{ $t(`shell.bulkSelectionBar.selected.${noun}`, count) }}
        </span>

        <span v-if="metricsSummary" class="hidden sm:inline-flex items-center text-xs text-slate-400 border-l border-slate-700/80 pl-2.5 ml-0.5 font-medium">
          {{ metricsSummary }}
        </span>
        <span v-if="offScreenCount > 0" class="text-xs text-slate-400 max-sm:hidden">
          {{ $t('shell.bulkSelectionBar.outOfView', { offScreenCount }) }}
        </span>
      </div>

      <div class="flex shrink-0 items-center sm:gap-2 sm:flex-wrap">
        <!-- Phone: the first actions as icons -->
        <template v-if="isMobile">
          <button
            v-for="a in phoneShortcuts(actions)"
            :key="a.key"
            type="button"
            class="tap rounded-full p-2 transition-colors hover:bg-slate-800 disabled:opacity-40"
            :class="a.danger ? 'text-rose-300' : 'text-slate-100'"
            :disabled="a.disabled || a.busy"
            :aria-label="a.label"
            :title="a.disabled && a.reason ? `${a.label}: ${a.reason}` : a.label"
            @click="a.run()"
          >
            <component :is="a.icon" class="h-5 w-5" aria-hidden="true" />
          </button>
        </template>
        <!-- Desktop: every non-overflow action as a button -->
        <template v-else>
          <button
            v-for="(a, i) in barActions(actions)"
            :key="a.key"
            type="button"
            class="btn"
            :class="a.primary ?? i === 0 ? 'btn-primary' : 'btn-secondary'"
            :disabled="a.disabled || a.busy"
            :title="a.disabled && a.reason ? a.reason : undefined"
            @click="a.run()"
          >
            <component :is="a.icon" class="w-3.5 h-3.5" :class="{ 'animate-spin': a.busy }" />
            <span>{{ a.label }}</span>
          </button>
          <slot />
        </template>

        <div v-if="hasMore || $slots.more" ref="moreRoot" class="relative">
          <button
            v-if="isMobile"
            type="button"
            class="tap rounded-full p-2 text-slate-100 hover:bg-slate-800"
            :aria-expanded="moreOpen"
            aria-haspopup="menu"
            :aria-label="$t('shell.bulkSelectionBar.moreActions')"
            @click="moreOpen = !moreOpen"
          >
            <MoreVertical class="h-5 w-5" aria-hidden="true" />
          </button>
          <button v-else type="button" class="btn btn-secondary" :aria-expanded="moreOpen" aria-haspopup="menu" @click="moreOpen = !moreOpen">
            <span>{{ $t('shell.bulkSelectionBar.more') }}</span>
          </button>
          <div
            v-if="moreOpen && !isMobile"
            role="menu"
            class="absolute right-0 top-full z-40 mt-2 w-64 space-y-0.5 rounded-xl border border-slate-700 bg-slate-800 p-1.5 shadow-2xl"
            @click="closeOnAction"
            @change="moreOpen = false"
          >
            <button
              v-for="a in actions.filter((x) => x.overflow)"
              :key="a.key"
              type="button"
              role="menuitem"
              class="menu-item disabled:opacity-40"
              :disabled="a.disabled || a.busy"
              :title="a.disabled && a.reason ? a.reason : undefined"
              @click="a.run()"
            >
              <component :is="a.icon" class="w-3.5 h-3.5 text-slate-400" />{{ a.label }}
            </button>
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
    </Teleport>

    <!-- Phone: every action, the ones that do not apply greyed with the reason -->
    <Teleport v-if="isMobile" to="body">
      <ModalShell v-model:open="moreOpen" :title="`${count} ${$t(`shell.bulkSelectionBar.selected.${noun}`, count)}`" size="sm">
        <div role="menu" class="space-y-1 [&_.menu-item]:min-h-12 [&_.menu-item]:gap-4 [&_.menu-item]:px-4 [&_.menu-item]:text-base [&_.menu-item_svg]:h-5 [&_.menu-item_svg]:w-5" @click="closeOnAction" @change="moreOpen = false">
          <button
            v-for="a in actions"
            :key="a.key"
            type="button"
            role="menuitem"
            class="menu-item disabled:opacity-40"
            :class="{ 'text-rose-300': a.danger && !a.disabled }"
            :disabled="a.disabled || a.busy"
            @click="a.run()"
          >
            <component :is="a.icon" class="text-slate-400" />
            <span class="min-w-0">
              <span class="block">{{ a.label }}</span>
              <span v-if="a.disabled && a.reason" class="block text-xs font-normal text-slate-400">{{ a.reason }}</span>
            </span>
          </button>
          <slot name="more" />
        </div>
      </ModalShell>
    </Teleport>
  </template>
</template>
