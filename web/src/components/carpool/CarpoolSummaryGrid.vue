<script setup lang="ts">
import { computed } from 'vue'
import KpiCard from '@/components/KpiCard.vue'
import { Users, TrendingUp, CreditCard, Receipt } from 'lucide-vue-next'
import { useVehicleStore } from '@/stores/vehicle'
import { formatAmount } from '@/currency'
import { formatPercentOrUnknown } from '@/utils/numbers'
import { t } from '@/i18n'
import { distanceUnit, formatDistanceValue, perDistance } from '@/units'

// Totals of all the carpool trips of the vehicle
const props = defineProps<{ summary: any }>()

const vehicleStore = useVehicleStore()
// Carpool amounts are in the vehicle's own currency
const fmt = (v: number) => formatAmount(Number(v || 0), vehicleStore.currency)
// A vehicle with no cost yet has no recovery rate: a dash, not 0 %
const coverageRate = computed(() => (props.summary.coverage_rate_pct == null ? null : formatPercentOrUnknown(props.summary.coverage_rate_pct)))
// Revenue at least equal to the passengers' fair share
const covered = computed(() => props.summary.total_revenue >= props.summary.total_passengers_share)
// A verdict only when there is something to judge: passengers' shares to cover, a recovery rate that exists
const receivedTone = computed(() => (props.summary.total_passengers_share > 0 ? (covered.value ? 'success' : 'warning') : 'default'))
const recoveryTone = computed(() => (props.summary.coverage_rate_pct != null && props.summary.coverage_rate_pct >= 100 ? 'success' : 'default'))
// The totals cover every trip of the vehicle, whatever the list below shows
const period = computed(() => t('carpool.carpoolSummaryGrid.allTrips'))
// No distance shared yet: the cost per km is unknown, not free
const netCostPerDistance = computed(() =>
  Number(props.summary.total_distance_km) > 0
    ? formatAmount(perDistance(Number(props.summary.net_cost_per_km || 0)), vehicleStore.currency, 3)
    : null,
)
</script>

<template>
  <div class="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-3 sm:gap-4">
    <KpiCard
      :label="$t('carpool.carpoolSummaryGrid.carpooledTrips')"
      :period="period"
      :icon="Users"
      :value="String(summary.total_trips)"
      :unit="` ${$t('carpool.carpoolSummaryGrid.trips', {}, summary.total_trips)}`"
    >
      <p class="mt-1 text-xs text-slate-400">{{ $t('carpool.carpoolSummaryGrid.kmShared', { unit: distanceUnit(), total_distance_km: formatDistanceValue(summary.total_distance_km || 0) }) }}</p>
    </KpiCard>

    <KpiCard
      :label="$t('carpool.carpoolSummaryGrid.passengersCarried')"
      :period="period"
      :icon="Users"
      :value="String(summary.total_passengers)"
      :unit="` ${$t('carpool.carpoolSummaryGrid.people', {}, summary.total_passengers)}`"
    >
      <p class="mt-1 text-xs text-slate-400">{{ $t('carpool.carpoolSummaryGrid.fairShareDue', { total_passengers_share: fmt(summary.total_passengers_share) }) }}</p>
    </KpiCard>

    <KpiCard
      :label="$t('carpool.carpoolSummaryGrid.totalReceivedFromPassengers')"
      :period="period"
      :icon="CreditCard"
      :value="fmt(summary.total_revenue)"
      :tone="receivedTone"
      :tinted="receivedTone !== 'default'"
    >
      <p class="mt-1 text-xs" :class="covered ? 'text-success-400' : 'text-warning-400'">
        {{ covered ? $t('carpool.carpoolSummaryGrid.covered') : $t('carpool.carpoolSummaryGrid.below', { amount: fmt(summary.total_passengers_share - summary.total_revenue) }) }}
      </p>
    </KpiCard>

    <KpiCard
      :label="$t('carpool.carpoolSummaryGrid.costRecoveryRate')"
      :period="period"
      :icon="TrendingUp"
      :value="coverageRate"
      :tone="recoveryTone"
      :tinted="recoveryTone !== 'default'"
    >
      <p class="mt-1 text-xs text-slate-400">{{ $t('carpool.carpoolSummaryGrid.ofTheTotalActualCost', { total_real_cost: fmt(summary.total_real_cost) }) }}</p>
    </KpiCard>

    <KpiCard
      class="col-span-2 lg:col-span-1"
      :label="$t('carpool.carpoolSummaryGrid.driverSNetCost')"
      :period="period"
      :icon="Receipt"
      :value="netCostPerDistance"
      :unit="` / ${distanceUnit()}`"
    >
      <p class="mt-1 text-xs text-slate-400">{{ $t('carpool.carpoolSummaryGrid.driverSFairShare', { total_driver_share: fmt(summary.total_driver_share) }) }}</p>
    </KpiCard>
  </div>
</template>
