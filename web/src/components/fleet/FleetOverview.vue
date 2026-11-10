<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Gauge, Zap } from '@/icons'
import LoadError from '@/components/LoadError.vue'
import ListSkeleton from '@/components/ListSkeleton.vue'
import FleetBudget from '@/components/fleet/FleetBudget.vue'
import FleetComparison from '@/components/fleet/FleetComparison.vue'
import FleetTrend from '@/components/fleet/FleetTrend.vue'
import { useRefreshHandler } from '@/composables/usePullToRefresh'
import { useVehicleStore } from '@/stores/vehicle'
import { api, type FleetSummaryResponse } from '@/services/api'
import { formatAmount } from '@/currency'
import { formatDistance } from '@/units'
import { formatNumber } from '@/utils/numbers'
import { formatMonthName } from '@/utils/dashboard'
import { currencyLines } from '@/utils/vehicleScope'
import { t } from '@/i18n'

// The home page when it covers every vehicle: the month so far of the household, one line per currency (amounts of two
// currencies are never added), then the budget, the vehicles side by side and the twelve months.
const vehicleStore = useVehicleStore()
const loading = ref(true)
const loadError = ref<string | null>(null)
const summary = ref<FleetSummaryResponse | null>(null)
const lines = computed(() => currencyLines(summary.value?.totals, summary.value?.currency ?? ''))

useRefreshHandler(() => load(true))

async function load(silent = false) {
  if (!silent) loading.value = true
  loadError.value = null
  try {
    summary.value = await api.getFleetSummary()
  } catch (err) {
    console.error('Failed to load fleet summary', err)
    loadError.value = (err as Error)?.message ?? ''
  } finally {
    loading.value = false
  }
}

function onBudgetSaved(budget: { monthly_budget: number | null; budget_currency: string | null }) {
  if (summary.value) summary.value = { ...summary.value, ...budget }
}

watch(() => vehicleStore.lastSyncTimestamp, () => void load(true))
onMounted(() => void load())
</script>

<template>
  <div class="space-y-4 sm:space-y-6" data-testid="fleet-overview">
    <LoadError v-if="loadError !== null" :message="loadError" @retry="load()" />
    <div v-else-if="loading && !summary" data-testid="today-skeleton">
      <ListSkeleton :rows="3" />
    </div>

    <template v-else-if="summary">
      <section class="card p-5" data-testid="fleet-hero">
        <p class="text-xs font-semibold text-slate-400">{{ t('fleet.today.spentIn', { month: formatMonthName(summary.current_month) }) }}</p>
        <p v-for="(line, i) in lines" :key="line.currency" :class="i === 0 ? 'mt-1 text-4xl font-extrabold text-white' : 'mt-1 text-xl font-bold text-slate-200'" class="tabular-nums" data-testid="fleet-currency-total">
          {{ formatAmount(line.amount, line.currency, Math.abs(line.amount) >= 1000 ? 0 : 2) }}
          <span v-if="lines.length > 1" class="text-xs font-normal text-slate-400">{{ line.currency }} · {{ t('fleet.today.vehicleCount', { count: line.vehicleCount }, line.vehicleCount) }}</span>
        </p>
        <p class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-slate-800 pt-3 text-sm text-slate-300">
          <span class="flex items-center gap-1.5"><Gauge class="h-4 w-4 text-slate-400" aria-hidden="true" />{{ formatDistance(summary.current_month_distance_km) }}</span>
          <span class="flex items-center gap-1.5"><Zap class="h-4 w-4 text-theme-energy" aria-hidden="true" />{{ formatNumber(summary.current_month_energy_kwh) }} kWh</span>
        </p>
      </section>
      <FleetBudget :summary="summary" @saved="onBudgetSaved" />
      <FleetComparison :summary="summary" />
      <FleetTrend :summary="summary" />
    </template>
  </div>
</template>
