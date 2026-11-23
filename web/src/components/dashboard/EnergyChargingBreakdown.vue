<script setup lang="ts">
import CostDonut from '@/components/costs/CostDonut.vue'
import SegmentedControl from '@/components/SegmentedControl.vue'
import { t } from '@/i18n'
import { computed, ref } from 'vue'
import { fmt, fmtMoney } from './energyStats'
import { activeAxis, availableAxes, type BreakdownAxis, type BreakdownRow } from '@/utils/chargingBreakdown'

// Where the energy was charged and how fast, as one donut with a legend that carries the figures. Both splits are
// the same energy: a toggle picks one instead of showing two blocks.
const props = defineProps<{
  place: BreakdownRow[]
  speed: BreakdownRow[]
  currency: string
  // Sessions with no duration, left out of the speed split
  unknownSpeedSessions?: number
}>()

const chosen = ref<BreakdownAxis>('place')
const axes = computed(() => availableAxes(props.place, props.speed))
const axis = computed(() => activeAxis(chosen.value, axes.value))
const rows = computed(() => (axis.value === 'speed' ? props.speed : props.place))
const items = computed(() => rows.value.map((r) => ({ label: r.label, color: r.color, amount: r.kwh })))
const totalKwh = computed(() => rows.value.reduce((sum, r) => sum + r.kwh, 0))
const note = computed(() => {
  if (axis.value === 'speed') {
    const count = props.unknownSpeedSessions ?? 0
    return count > 0 ? t('dashboard.energyEfficiencyPanel.unknownDurationSessions', { count }) : ''
  }
  const count = rows.value.reduce((sum, r) => sum + r.unknownSessions, 0)
  return count > 0 ? t('dashboard.energyEfficiencyPanel.placeUnknown', { count }) : ''
})
const options = computed(() => [
  { value: 'place' as const, label: t('dashboard.energyEfficiencyPanel.breakdownPlace') },
  { value: 'speed' as const, label: t('dashboard.energyEfficiencyPanel.breakdownSpeed') },
])
const chartLabel = computed(() => (axis.value === 'speed' ? t('dashboard.energyEfficiencyPanel.howTheCarIsCharged') : t('dashboard.energyEfficiencyPanel.byPlace')))
</script>

<template>
  <div v-if="axis" class="space-y-3" data-testid="charging-breakdown">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h4 class="text-xs font-bold text-slate-200">{{ $t('dashboard.energyEfficiencyPanel.charging') }}</h4>
      <SegmentedControl v-if="axes.length > 1" v-model="chosen" :options="options" :label="$t('dashboard.energyEfficiencyPanel.charging')" />
    </div>
    <div class="grid grid-cols-1 items-center gap-4 md:grid-cols-5">
      <div class="relative mx-auto h-40 w-full max-w-xs md:col-span-2">
        <CostDonut :items="items" unit="kWh" :show-legend="false" :empty-label="$t('dashboard.energyEfficiencyPanel.noCharge')" :chart-label="chartLabel" />
        <div class="pointer-events-none absolute inset-0 flex items-center justify-center text-center" aria-hidden="true">
          <span class="text-base font-bold leading-tight text-white">{{ fmt(totalKwh, 0) }}<span class="block text-xs font-medium text-slate-400">kWh</span></span>
        </div>
      </div>
      <ul class="divide-y divide-slate-800 text-xs md:col-span-3">
        <li v-for="r in rows" :key="r.key" class="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-0.5 py-2 first:pt-0 last:pb-0">
          <span class="flex items-center gap-2 text-sm font-semibold text-white">
            <span class="h-2.5 w-2.5 shrink-0 rounded-full" :style="{ backgroundColor: r.color }" aria-hidden="true"></span>
            {{ r.label }}
          </span>
          <span class="tabular-nums text-slate-300">
            {{ Math.round(r.sharePct) }} % · {{ fmt(r.kwh, 0) }} kWh ·
            {{ r.pricePerKwh === null ? $t('expenses.chargingInsights.noPrice') : $t('expenses.chargingInsights.pricePerKwh', { price: fmtMoney(r.pricePerKwh, currency, 3) }) }}
          </span>
        </li>
      </ul>
    </div>
    <p v-if="note" class="text-xs text-slate-400">{{ note }}</p>
  </div>
</template>
