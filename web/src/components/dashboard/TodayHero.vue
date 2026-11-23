<script setup lang="ts">
import { computed } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import { formatAmount } from '@/currency'
import { formatMonthName } from '@/utils/dashboard'
import { t, intlLocale } from '@/i18n'
import type { TodayHero } from '@/utils/todayHero'
import GlossaryTip from '@/components/GlossaryTip.vue'

// The month so far: what was paid, what is already planned, the themes it is made of and the last full month, in the same
// unit, to read it against. One figure, one bar, one line; the detail is the journal of costs.
const props = defineProps<{ hero: TodayHero }>()
const vehicleStore = useVehicleStore()
const money = (v: number, digits = 2) => formatAmount(v || 0, vehicleStore.currency, digits)
const headline = computed(() => money(props.hero.spent, Math.abs(props.hero.spent) >= 1000 ? 0 : 2))
// The month in progress is not finished: it is told as a share of the last full month, which is true on any day of the
// month, and not as a gap that would read "-80 %" on the 5th
const monthOnly = (month: string) => capitalize(new Date(Number(month.slice(0, 4)), Number(month.slice(5, 7)) - 1, 1).toLocaleDateString(intlLocale(), { month: 'long' }))
const capitalize = (text: string) => text.charAt(0).toUpperCase() + text.slice(1)
const comparison = computed(() => {
  const previous = props.hero.previous
  if (!previous) return ''
  const above = Math.round((previous.share - 1) * 100)
  if (previous.share >= 1 && above > 0) return t('dashboard.today.shareAbove', { percent: above })
  if (Math.round(previous.share * 100) >= 100) return t('dashboard.today.shareReached')
  return t('dashboard.today.shareBelow', { percent: Math.round(previous.share * 100), month: monthOnly(props.hero.month) })
})
</script>

<template>
  <section class="card p-5" data-testid="today-hero">
    <router-link to="/costs" class="block rounded-xl">
      <p class="text-xs font-semibold text-slate-400">
        {{ $t('dashboard.today.spentIn', { month: formatMonthName(hero.month) }) }}
      </p>
      <p class="mt-1 text-4xl font-extrabold tabular-nums text-hero">{{ headline }}</p>
      <p v-if="hero.scheduled > 0" class="mt-1 text-xs text-slate-300">{{ $t('dashboard.tcoMetricsGrid.alreadyScheduled', { amount: money(hero.scheduled) }) }}</p>
    </router-link>

    <!-- The themes of the month: a bar, each segment with the colour of its theme and the legend that says what it is -->
    <template v-if="hero.segments.length">
      <!-- The themes of the scheduled part are not known apart: the bar is the whole month, said so when something is scheduled -->
      <p v-if="hero.scheduled > 0" class="mt-4 text-xs text-slate-400">{{ $t('dashboard.today.wholeMonth') }}</p>
      <div class="flex h-2 overflow-hidden rounded-full bg-slate-800" :class="hero.scheduled > 0 ? 'mt-1.5' : 'mt-4'" role="img" :aria-label="$t('dashboard.today.themesLabel')">
        <span v-for="s in hero.segments" :key="s.key" class="h-full" :style="{ width: `${s.sharePct}%`, backgroundColor: s.color }"></span>
      </div>
      <ul class="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs text-slate-300" data-testid="today-themes">
        <li v-for="s in hero.segments" :key="s.key" class="flex items-center gap-1.5">
          <span class="h-2 w-2 shrink-0 rounded-full" :style="{ backgroundColor: s.color }" aria-hidden="true"></span>
          <span>{{ s.label }}</span>
          <span class="tabular-nums text-slate-400">{{ money(s.amount, 0) }}</span>
        </li>
      </ul>
    </template>

    <div v-if="hero.previous" class="mt-4 border-t border-slate-800 pt-3 text-sm text-slate-300" data-testid="today-previous">
      <p class="flex items-center justify-between gap-3">
        <span class="flex items-center">
          {{ $t('dashboard.today.previousWhole', { month: formatMonthName(hero.previous.month) }) }}
          <GlossaryTip term="cashPaid" />
        </span>
        <span class="font-semibold tabular-nums text-white">{{ money(hero.previous.total, hero.previous.total >= 1000 ? 0 : 2) }}</span>
      </p>
      <p class="mt-1 text-xs text-slate-400" data-testid="today-share">{{ comparison }}</p>
    </div>
  </section>
</template>
