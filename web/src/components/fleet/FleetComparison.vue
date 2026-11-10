<script setup lang="ts">
import { computed, ref } from 'vue'
import GlossaryTip from '@/components/GlossaryTip.vue'
import ModalShell from '@/components/ModalShell.vue'
import { Award, ChevronRight, TrendingDown } from 'lucide-vue-next'
import type { FleetSummaryResponse } from '@/services/api'
import { formatAmount, formatAmountOrUnknown } from '@/currency'
import { formatPercent } from '@/utils/numbers'
import { cheapestPerKmIds, rankFleetVehicles } from '@/utils/fleetRanking'
import { formatDistance, currentDistanceUnit, perDistance, formatPerDistanceValue, formatFuelConsumptionValue, fuelConsumptionUnit } from '@/units'
import { t } from '@/i18n'

// The vehicles side by side: a row each on a phone (its figures open in a sheet), a table from md.
const props = defineProps<{ summary: FleetSummaryResponse }>()

const distanceUnitLabel = computed(() => currentDistanceUnit())
const detailId = ref<string | null>(null)
const detailOpen = computed({ get: () => detailId.value !== null, set: (open: boolean) => { if (!open) detailId.value = null } })

// Ranking by full cost per km inside each currency, only among vehicles whose figures are complete enough to be compared
const rankedVehicles = computed(() => rankFleetVehicles(props.summary.vehicles, props.summary.currency))
const detailVehicle = computed(() => rankedVehicles.value.find((v) => v.vehicle_id === detailId.value) ?? null)
const cheapestPerKmIdSet = computed(() => cheapestPerKmIds(rankedVehicles.value))

function completenessClass(pct: number): string {
  if (pct >= 80) return 'text-success-400'
  if (pct >= 60) return 'text-warning-400'
  return 'text-rose-400'
}

// The energy cost of 100 distance units: a known zero (free charging) is shown, an unknown figure is a dash
function energyCost100(v: { energy_cost_per_100km: number | null; currency: string }): string {
  return v.energy_cost_per_100km == null ? '—' : formatAmount(perDistance(v.energy_cost_per_100km), v.currency)
}

function energyPer100(v: { kwh_per_100km: number | null; liters_per_100km: number | null }): string {
  const parts: string[] = []
  if (v.kwh_per_100km != null) parts.push(t('fleet.compare.kwh100', { value: formatPerDistanceValue(v.kwh_per_100km), unit: distanceUnitLabel.value }))
  if (v.liters_per_100km != null) parts.push(t('fleet.compare.l100', { value: formatFuelConsumptionValue(v.liters_per_100km), unit: fuelConsumptionUnit() }))
  return parts.length ? parts.join(' + ') : '—'
}
</script>

<template>
  <!-- Cost per distance comparison -->
  <div v-if="summary.vehicles.length > 0" class="space-y-4 sm:rounded-xl sm:border sm:border-slate-800 sm:bg-slate-900 sm:p-5 sm:shadow-sm">
    <div class="flex flex-col gap-1 border-b border-slate-800/80 pb-3 sm:flex-row sm:items-center sm:justify-between sm:gap-3">
      <h2 class="text-base font-semibold text-white flex items-center gap-2">
        <TrendingDown class="w-4 h-4 text-rose-400" />
        {{ t('fleet.compare.title') }}
      </h2>
      <span class="text-xs text-slate-400">{{ t('fleet.compare.hint') }}</span>
    </div>
    <ul class="space-y-2 md:hidden">
      <li v-for="v in rankedVehicles" :key="v.vehicle_id">
        <button
          type="button"
          class="flex w-full items-center gap-3 card p-4 text-left transition-colors active:bg-slate-800/60"
          aria-haspopup="dialog"
          @click="detailId = v.vehicle_id"
        >
          <span class="min-w-0 flex-1">
            <span class="block truncate font-semibold text-white">{{ v.name }}</span>
            <span v-if="cheapestPerKmIdSet.has(v.vehicle_id)" class="mt-0.5 inline-flex items-center gap-1 text-xs font-semibold text-success-400">
              <Award class="h-3 w-3" aria-hidden="true" />{{ t('fleet.compare.cheapest') }}
            </span>
            <span v-else-if="!v.comparable" class="mt-0.5 block truncate text-xs text-slate-300">{{ v.depreciation_known === false ? t('fleet.compare.depreciationMissing') : t('fleet.compare.indicative') }}</span>
            <span class="mt-0.5 block truncate text-xs text-slate-400">
              {{ formatAmount(v.month_cost, v.currency) }} · {{ t('fleet.compare.completeness') }}
              <span class="font-semibold tabular-nums" :class="completenessClass(v.completeness_pct)">{{ formatPercent(v.completeness_pct, 0) }}</span>
            </span>
          </span>
          <span class="shrink-0 text-right">
            <span class="block text-lg font-bold tabular-nums text-white">{{ formatAmountOrUnknown(v.full_cost_per_km == null ? null : perDistance(v.full_cost_per_km), v.currency) }}</span>
            <span class="block text-xs text-slate-400">{{ t('fleet.compare.fullPerDistance', { unit: distanceUnitLabel }) }}</span>
          </span>
          <ChevronRight class="h-5 w-5 shrink-0 text-slate-500" aria-hidden="true" />
        </button>
      </li>
    </ul>
    <Teleport to="body">
      <ModalShell v-model:open="detailOpen" :title="detailVehicle?.name ?? ''" size="md">
        <dl v-if="detailVehicle" class="grid grid-cols-2 gap-x-4 gap-y-4 text-sm">
          <div>
            <dt class="text-xs text-slate-400">{{ t('fleet.compare.fullPerDistance', { unit: distanceUnitLabel }) }}</dt>
            <dd class="font-semibold tabular-nums text-white">{{ formatAmountOrUnknown(detailVehicle.full_cost_per_km == null ? null : perDistance(detailVehicle.full_cost_per_km), detailVehicle.currency) }}</dd>
          </div>
          <div>
            <dt class="text-xs text-slate-400">{{ t('fleet.compare.runningPerDistance', { unit: distanceUnitLabel }) }} <GlossaryTip term="cashPaid" /></dt>
            <dd class="tabular-nums text-slate-200">{{ formatAmountOrUnknown(detailVehicle.running_cost_per_km == null ? null : perDistance(detailVehicle.running_cost_per_km), detailVehicle.currency) }}</dd>
          </div>
          <div>
            <dt class="text-xs text-slate-400">{{ t('fleet.compare.monthCost') }}</dt>
            <dd class="tabular-nums text-slate-200">{{ formatAmount(detailVehicle.month_cost, detailVehicle.currency) }}</dd>
          </div>
          <div>
            <dt class="text-xs text-slate-400">{{ t('fleet.compare.monthDistance') }}</dt>
            <dd class="tabular-nums text-slate-200">{{ formatDistance(detailVehicle.month_distance_km) }}</dd>
          </div>
          <div>
            <dt class="text-xs text-slate-400">{{ t('fleet.compare.energyCost', { unit: distanceUnitLabel }) }}</dt>
            <dd class="tabular-nums text-slate-200">{{ energyCost100(detailVehicle) }}</dd>
          </div>
          <div>
            <dt class="text-xs text-slate-400">{{ t('fleet.compare.energy') }} <GlossaryTip term="atThePlug" /></dt>
            <dd class="tabular-nums text-slate-200">{{ energyPer100(detailVehicle) }}</dd>
          </div>
          <div>
            <dt class="text-xs text-slate-400">{{ t('fleet.compare.annual') }}</dt>
            <dd class="tabular-nums text-slate-200">{{ detailVehicle.annual_cost != null ? formatAmount(detailVehicle.annual_cost, detailVehicle.currency) : '—' }}</dd>
          </div>
          <div>
            <dt class="text-xs text-slate-400">{{ t('fleet.compare.completeness') }}</dt>
            <dd class="font-semibold tabular-nums" :class="completenessClass(detailVehicle.completeness_pct)">{{ formatPercent(detailVehicle.completeness_pct, 0) }}</dd>
          </div>
        </dl>
      </ModalShell>
    </Teleport>
    <div class="hidden overflow-x-auto md:block">
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-xs text-slate-400">
            <th class="py-2 pr-4 font-medium">{{ t('fleet.compare.vehicle') }}</th>
            <th class="py-2 pr-4 font-medium text-right">{{ t('fleet.compare.runningPerDistance', { unit: distanceUnitLabel }) }} <GlossaryTip term="cashPaid" /></th>
            <th class="py-2 pr-4 font-medium text-right">{{ t('fleet.compare.fullPerDistance', { unit: distanceUnitLabel }) }}</th>
            <th class="py-2 pr-4 font-medium text-right">{{ t('fleet.compare.energyCost', { unit: distanceUnitLabel }) }}</th>
            <th class="py-2 pr-4 font-medium text-right">{{ t('fleet.compare.energy') }} <GlossaryTip term="atThePlug" /></th>
            <th class="py-2 pr-4 font-medium text-right">{{ t('fleet.compare.monthDistance') }}</th>
            <th class="py-2 pr-4 font-medium text-right">{{ t('fleet.compare.monthCost') }}</th>
            <th class="py-2 pr-4 font-medium text-right">{{ t('fleet.compare.annual') }}</th>
            <th class="py-2 font-medium text-right">{{ t('fleet.compare.completeness') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="v in rankedVehicles" :key="v.vehicle_id" class="border-t border-slate-800/60">
            <td class="py-2 pr-4">
              <span class="block font-semibold text-white">{{ v.name }}</span>
              <span v-if="cheapestPerKmIdSet.has(v.vehicle_id)" class="mt-0.5 inline-flex items-center gap-1 text-xs font-semibold text-success-400">
                <Award class="w-3 h-3" />{{ t('fleet.compare.cheapest') }}
              </span>
              <span v-else-if="!v.comparable" class="mt-0.5 block text-xs text-slate-300">{{ v.depreciation_known === false ? t('fleet.compare.depreciationMissing') : t('fleet.compare.indicative') }}</span>
            </td>
            <td class="py-2 pr-4 text-right tabular-nums">{{ formatAmountOrUnknown(v.running_cost_per_km == null ? null : perDistance(v.running_cost_per_km), v.currency) }}</td>
            <td class="py-2 pr-4 text-right tabular-nums font-semibold text-white">{{ formatAmountOrUnknown(v.full_cost_per_km == null ? null : perDistance(v.full_cost_per_km), v.currency) }}</td>
            <td class="py-2 pr-4 text-right tabular-nums">{{ energyCost100(v) }}</td>
            <td class="py-2 pr-4 text-right tabular-nums">{{ energyPer100(v) }}</td>
            <td class="py-2 pr-4 text-right tabular-nums">{{ formatDistance(v.month_distance_km) }}</td>
            <td class="py-2 pr-4 text-right tabular-nums">{{ formatAmount(v.month_cost, v.currency) }}</td>
            <td class="py-2 pr-4 text-right tabular-nums">{{ v.annual_cost != null ? formatAmount(v.annual_cost, v.currency) : '—' }}</td>
            <td class="py-2 text-right tabular-nums font-semibold" :class="completenessClass(v.completeness_pct)">{{ formatPercent(v.completeness_pct, 0) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <p class="text-xs text-slate-400">{{ t('fleet.compare.footnote') }}</p>
  </div>

</template>
