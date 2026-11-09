<script setup lang="ts">
import { ref } from 'vue'
import { CalendarDays } from 'lucide-vue-next'
import PeriodSheet from '@/components/period/PeriodSheet.vue'
import { usePeriodStore } from '@/stores/period'
import { isRoundedToMonths, periodLabel } from '@/utils/period'

// The shared period inside a sheet of figures: a labelled row that opens the period sheet over the sheet it is in.
const store = usePeriodStore()
const open = ref(false)
</script>

<template>
  <div class="flex flex-wrap items-center gap-2">
    <span class="text-xs font-semibold text-slate-300">{{ $t('period.bar.label') }}</span>
    <button
      type="button"
      class="tap-text inline-flex min-w-0 items-center gap-2 rounded-lg border border-slate-700 bg-slate-800 px-3 text-sm font-semibold text-white"
      :aria-label="$t('period.bar.open', { period: periodLabel(store.costs) })"
      @click="open = true"
    >
      <CalendarDays class="h-4 w-4 shrink-0 text-slate-400" aria-hidden="true" />
      <span class="truncate">{{ periodLabel(store.costs) }}</span>
    </button>
    <span v-if="isRoundedToMonths(store.costs)" class="text-xs text-slate-400">{{ $t('period.bar.rounded') }}</span>
    <PeriodSheet v-model:open="open" nested :period="store.costs" @pick="store.setCosts" />
  </div>
</template>
