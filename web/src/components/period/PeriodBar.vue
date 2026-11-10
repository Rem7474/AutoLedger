<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { CalendarDays, ChevronLeft, ChevronRight } from '@/icons'
import PeriodSheet from '@/components/period/PeriodSheet.vue'
import { initialScrollHide, stepScrollHide } from '@/utils/scrollHide'
import type { Period } from '@/utils/period'

// The period of a page: its label opens the sheet, the arrows go to the neighbouring period. On a phone the bar sits
// just above the tab bar and gives way while the page is read downwards; from `md` it is part of the page.
defineProps<{ period?: Period; label: string; rounded?: boolean; canPrevious: boolean; canNext: boolean }>()
const emit = defineEmits<{ step: [dir: -1 | 1]; pick: [period: Period] }>()
// A page whose period is not a `Period` (Trajets) brings its own sheet in the `sheet` slot

const sheet = ref(false)
const hidden = ref(false)
let scroller: HTMLElement | null = null
let state = initialScrollHide
const onScroll = () => {
  if (!scroller) return
  state = stepScrollHide(state, scroller.scrollTop)
  hidden.value = state.hidden
}
onMounted(() => {
  scroller = document.querySelector<HTMLElement>('.app-scroll')
  scroller?.addEventListener('scroll', onScroll, { passive: true })
})
onBeforeUnmount(() => scroller?.removeEventListener('scroll', onScroll))
</script>

<template>
  <div
    class="period-bar fixed inset-x-0 z-40 bottom-[calc(4.25rem+env(safe-area-inset-bottom))] px-3 transition-transform duration-200 md:static md:z-30 md:px-0 md:translate-y-0"
    :class="hidden && !sheet ? 'translate-y-[calc(100%+6rem)]' : 'translate-y-0'"
    data-testid="period-bar"
  >
    <div class="mx-auto flex max-w-lg items-center gap-1 rounded-2xl border border-slate-800 bg-slate-900/95 p-1 shadow-lg backdrop-blur-md md:max-w-none md:shadow-none">
      <button type="button" class="tap flex shrink-0 items-center justify-center rounded-xl text-slate-300 hover:text-white disabled:opacity-30" :disabled="!canPrevious" :aria-label="$t('period.bar.previous')" @click="emit('step', -1)">
        <ChevronLeft class="h-5 w-5" aria-hidden="true" />
      </button>
      <button type="button" class="tap-text flex min-w-0 flex-1 items-center justify-center gap-2 rounded-xl px-2 text-sm font-semibold text-white" :aria-label="$t('period.bar.open', { period: label })" @click="sheet = true">
        <CalendarDays class="h-4 w-4 shrink-0 text-slate-400" aria-hidden="true" />
        <span class="truncate">{{ label }}</span>
        <span v-if="rounded" class="truncate text-xs font-normal text-slate-400 max-sm:sr-only">{{ $t('period.bar.rounded') }}</span>
      </button>
      <button type="button" class="tap flex shrink-0 items-center justify-center rounded-xl text-slate-300 hover:text-white disabled:opacity-30" :disabled="!canNext" :aria-label="$t('period.bar.next')" @click="emit('step', 1)">
        <ChevronRight class="h-5 w-5" aria-hidden="true" />
      </button>
    </div>
  </div>
  <!-- Outside the bar: a transformed ancestor would become the containing block of the fixed sheet -->
  <slot name="sheet" :open="sheet" :set-open="(v: boolean) => (sheet = v)">
    <PeriodSheet v-if="period" v-model:open="sheet" :period="period" @pick="emit('pick', $event)" />
  </slot>
</template>
