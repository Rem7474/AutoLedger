<script setup lang="ts">
import { computed } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import { formatAmount, formatAmountOrUnknown } from '@/currency'
import { distanceUnit, formatDistanceValue, perDistance } from '@/units'
import { formatMonthName } from '@/utils/dashboard'
import type { TodayHero } from '@/utils/todayHero'
import GlossaryTip from '@/components/GlossaryTip.vue'

// The month so far: what was paid, what is already planned, the themes it is made of and the cost per distance on the
// same basis. One figure, one bar, one line; the detail is the journal of costs.
const props = defineProps<{ hero: TodayHero }>()
const vehicleStore = useVehicleStore()
const money = (v: number, digits = 2) => formatAmount(v || 0, vehicleStore.currency, digits)
const headline = computed(() => money(props.hero.spent, Math.abs(props.hero.spent) >= 1000 ? 0 : 2))
const perUnit = computed(() => formatAmountOrUnknown(props.hero.perKm == null ? null : perDistance(props.hero.perKm), vehicleStore.currency, 3))
</script>

<template>
  <section class="card p-5" data-testid="today-hero">
    <router-link to="/costs" class="block rounded-xl">
      <p class="text-xs font-semibold text-slate-400">
        {{ $t('dashboard.today.spentIn', { month: formatMonthName(hero.month) }) }}
      </p>
      <p class="mt-1 text-4xl font-extrabold tabular-nums text-white">{{ headline }}</p>
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

    <p class="mt-4 flex items-center justify-between gap-3 border-t border-slate-800 pt-3 text-sm text-slate-300" data-testid="today-per-km">
      <span class="flex items-center">
        {{ $t('dashboard.today.perKm', { unit: distanceUnit() }) }}
        <GlossaryTip term="cashPaid" />
      </span>
      <span class="font-semibold tabular-nums text-white">
        {{ perUnit }}<span v-if="hero.perKm != null" class="text-xs font-normal text-slate-400">/{{ distanceUnit() }}</span>
      </span>
    </p>
    <p v-if="hero.distanceKm > 0" class="mt-1 text-xs text-slate-400">{{ $t('dashboard.tcoMetricsGrid.monthDistance', { unit: distanceUnit(), distance_km: formatDistanceValue(hero.distanceKm) }) }}</p>
  </section>
</template>
