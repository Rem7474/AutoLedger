<script setup lang="ts">
import { computed } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import { formatAmount, formatAmountOrUnknown } from '@/currency'
import { distanceUnit, perDistance } from '@/units'
import { intlLocale, t } from '@/i18n'
import { formatMonthName } from '@/utils/dashboard'
import type { YearOverview } from '@/utils/yearOverview'

// The last 12 complete months and the one in progress, in the cash paid net of the carpools, on the basis of the month
// above. The figures read the complete months; the month in progress is a hatched bar at the end.
const props = defineProps<{ overview: YearOverview }>()
const vehicleStore = useVehicleStore()
const money = (v: number) => formatAmount(v || 0, vehicleStore.currency, Math.abs(v) >= 1000 ? 0 : 2)
const perUnit = computed(() => formatAmountOrUnknown(props.overview.perKm == null ? null : perDistance(props.overview.perKm), vehicleStore.currency, 3))
const initial = (month: string) => new Date(Number(month.slice(0, 4)), Number(month.slice(5, 7)) - 1, 1).toLocaleDateString(intlLocale(), { month: 'narrow' })
const height = (total: number) => `${props.overview.max > 0 ? Math.max((total / props.overview.max) * 100, total > 0 ? 3 : 0) : 0}%`
const summary = computed(() => t('dashboard.year.summary', { months: props.overview.months, total: money(props.overview.total), average: money(props.overview.average) }))
const barLabel = (b: { month: string; total: number; partial: boolean }) => `${formatMonthName(b.month)}${b.partial ? ` (${t('dashboard.year.inProgress')})` : ''}: ${money(b.total)}`
</script>

<template>
  <section class="card p-5" data-testid="year-overview">
    <router-link to="/costs" class="block rounded-xl">
      <h2 class="text-xs font-semibold text-slate-400">{{ $t('dashboard.year.title', { months: overview.months }) }}</h2>
      <dl class="mt-2 grid grid-cols-3 gap-3">
        <div>
          <dt class="text-xs text-slate-400">{{ $t('dashboard.year.total') }}</dt>
          <dd class="text-lg font-bold tabular-nums text-white" data-testid="year-total">{{ money(overview.total) }}</dd>
        </div>
        <div>
          <dt class="text-xs text-slate-400">{{ $t('dashboard.year.average') }}</dt>
          <dd class="text-lg font-bold tabular-nums text-white" data-testid="year-average">{{ money(overview.average) }}</dd>
        </div>
        <div>
          <dt class="text-xs text-slate-400">{{ $t('dashboard.year.perKm', { unit: distanceUnit() }) }}</dt>
          <dd class="text-lg font-bold tabular-nums text-white" data-testid="year-per-km">{{ perUnit }}<span v-if="overview.perKm != null" class="text-xs font-normal text-slate-400">/{{ distanceUnit() }}</span></dd>
        </div>
      </dl>
    </router-link>

    <div class="mt-4 flex h-28 items-end gap-1" role="img" :aria-label="summary" data-testid="year-bars">
      <div v-for="b in overview.bars" :key="b.month" class="flex h-full min-w-0 flex-1 flex-col justify-end" :title="barLabel(b)">
        <div
          class="flex flex-col-reverse overflow-hidden rounded-t"
          :class="b.partial ? 'opacity-60 outline outline-1 -outline-offset-1 outline-dashed outline-slate-400' : ''"
          :style="{ height: height(b.total) }"
          :data-partial="b.partial || undefined"
        >
          <span v-for="s in b.segments" :key="s.key" class="block w-full" :style="{ height: `${(s.amount / b.total) * 100}%`, backgroundColor: s.color }"></span>
        </div>
      </div>
    </div>
    <div class="mt-1 flex gap-1 text-[10px] uppercase text-slate-500" aria-hidden="true">
      <span v-for="b in overview.bars" :key="b.month" class="min-w-0 flex-1 text-center">{{ initial(b.month) }}</span>
    </div>
    <p v-if="overview.bars.some((b) => b.partial)" class="mt-2 text-xs text-slate-400">{{ $t('dashboard.year.partialNote') }}</p>
  </section>
</template>
