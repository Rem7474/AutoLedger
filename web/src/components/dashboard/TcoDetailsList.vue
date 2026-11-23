<script setup lang="ts">
import { intlLocale } from '@/i18n'
import { computed } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import { formatAmount, formatAmountOrUnknown } from '@/currency'
import GlossaryTip from '@/components/GlossaryTip.vue'
import { rollingWindowStats } from '@/utils/dashboard'
import { distanceUnit, formatDistanceValue, formatFuelConsumptionValue, fuelConsumptionUnit, litresToDisplayVolume, perDistance, perVolume, volumeUnitLabel } from '@/units'

// The totals behind the headline figures, as a list: under the figures on a wide screen, in a sheet on a phone
const props = defineProps<{ tco: any | null }>()
const vehicleStore = useVehicleStore()
const money = (v: number, digits = 2) => formatAmount(v || 0, vehicleStore.currency, digits)
const perUnit = (v: number | null | undefined) => formatAmountOrUnknown(v == null ? null : perDistance(v), vehicleStore.currency, 3)
const rolling = computed(() => rollingWindowStats(props.tco?.monthly_costs))
</script>

<template>
<div class="text-xs text-slate-300">
<dl class="grid grid-cols-1 gap-x-6 gap-y-1.5 sm:grid-cols-2">
  <div class="flex justify-between gap-3">
    <dt class="text-slate-400">{{ $t('dashboard.tcoMetricsGrid.fullCostLabel') }}</dt>
    <dd class="font-semibold text-white">{{ money(tco?.full_cost) }}</dd>
  </div>
  <div v-if="tco?.depreciation_cost" class="flex justify-between gap-3">
    <dt class="text-slate-400">{{ $t('dashboard.tcoMetricsGrid.depreciationLabel') }}</dt>
    <dd class="font-semibold text-white">{{ money(tco.depreciation_cost, 0) }}</dd>
  </div>
  <div class="flex justify-between gap-3">
    <dt class="text-slate-400">{{ $t('dashboard.tcoMetricsGrid.totalCost') }}</dt>
    <dd class="font-semibold text-white">{{ money(tco?.total_cost, 0) }}</dd>
  </div>
  <div v-if="rolling" class="flex justify-between gap-3">
    <dt class="text-slate-400">{{ $t('dashboard.tcoMetricsGrid.economicWindowLabel', { months: rolling.months }, rolling.months) }}</dt>
    <dd class="font-semibold text-white">{{ money(rolling.economic, 0) }}</dd>
  </div>
  <div class="flex justify-between gap-3">
    <dt class="text-slate-400">{{ $t('dashboard.tcoMetricsGrid.runningCostLabel', { unit: distanceUnit() }) }} <GlossaryTip term="runningCost" /></dt>
    <dd class="font-semibold text-white">{{ perUnit(tco?.usage_cost_per_km) }}/{{ distanceUnit() }}</dd>
  </div>
  <div class="flex justify-between gap-3">
    <dt class="text-slate-400">{{ $t('dashboard.tcoMetricsGrid.distanceBasis') }}</dt>
    <dd class="font-semibold text-white">{{ formatDistanceValue(tco?.distance_basis_km || 0) }} {{ distanceUnit() }}</dd>
  </div>
  <div class="flex justify-between gap-3">
    <dt class="text-slate-400">{{ $t('dashboard.tcoMetricsGrid.tollsAndParking') }}</dt>
    <dd class="font-semibold text-white">{{ money(tco?.tolls_cost, 0) }} · {{ perUnit(tco?.tolls_cost_per_km) }}/{{ distanceUnit() }}</dd>
  </div>
  <div v-if="tco?.carpool_revenue" class="flex justify-between gap-3">
    <dt class="text-success-400">{{ $t('dashboard.tcoMetricsGrid.netOfCarpoolingLabel') }}</dt>
    <dd class="font-semibold text-success-400">{{ money(tco.full_cost_net) }}</dd>
  </div>
  <template v-if="vehicleStore.capabilities.refuels">
    <div v-if="tco?.total_liters" class="flex justify-between gap-3">
      <dt class="text-slate-400">{{ volumeUnitLabel() }}</dt>
      <dd class="font-semibold text-white">{{ Math.round(litresToDisplayVolume(tco.total_liters)).toLocaleString(intlLocale()) }}</dd>
    </div>
    <div v-if="tco?.consumption_l_100km" class="flex justify-between gap-3">
      <dt class="text-slate-400">{{ fuelConsumptionUnit() }}</dt>
      <dd class="font-semibold text-white">{{ formatFuelConsumptionValue(tco.consumption_l_100km, 2) }}</dd>
    </div>
    <div v-if="tco?.avg_cost_per_liter" class="flex justify-between gap-3">
      <dt class="text-slate-400">{{ $t('dashboard.tcoMetricsGrid.avgPrice') }}</dt>
      <dd class="font-semibold text-white">{{ money(perVolume(tco.avg_cost_per_liter), 3) }}/{{ volumeUnitLabel() }}</dd>
    </div>
  </template>
  <div v-if="vehicleStore.capabilities.charges && tco?.total_kwh_added" class="flex justify-between gap-3">
    <dt class="text-slate-400">{{ $t('dashboard.tcoMetricsGrid.kwhLabel') }}</dt>
    <dd class="font-semibold text-white">{{ Math.round(tco.total_kwh_added).toLocaleString(intlLocale()) }} kWh</dd>
  </div>
  <div v-if="vehicleStore.capabilities.charges && vehicleStore.capabilities.refuels" class="flex justify-between gap-3 sm:col-span-2">
    <dt class="text-slate-400">{{ $t('dashboard.tcoMetricsGrid.hybridSplitLabel') }}</dt>
    <dd class="font-semibold text-white">{{ $t('dashboard.tcoMetricsGrid.hybridSplit', { electricity: money((tco?.energy_cost || 0) - (tco?.fuel_energy_cost || 0), 0), fuel: money(tco?.fuel_energy_cost || 0, 0) }) }}</dd>
  </div>
</dl>
<p v-if="tco?.completeness?.charges_without_cost" class="mt-2 text-xs text-warning-400">
  {{ $t('dashboard.tcoMetricsGrid.chargeSWithoutACost', { charges_without_cost: tco.completeness.charges_without_cost }) }}
</p>
</div>
</template>
