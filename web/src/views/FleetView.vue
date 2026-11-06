<script setup lang="ts">
import { scaleCut, type ScaleCut } from '@/utils/fleetChart'
import GlossaryTip from '@/components/GlossaryTip.vue'
import LoadError from '@/components/LoadError.vue'
import KpiCard from '@/components/KpiCard.vue'
import ModalShell from '@/components/ModalShell.vue'
import ListSkeleton from '@/components/ListSkeleton.vue'
import { formatNumber, formatPercent } from '@/utils/numbers'
import PageHeader from '@/components/PageHeader.vue'
import { Gauge as PageIcon } from 'lucide-vue-next'
import { ref, onMounted, computed, watch, nextTick, onBeforeUnmount } from 'vue'
import { Car, Zap, Gauge, Users, RefreshCw, Award, TrendingDown, ChevronRight } from 'lucide-vue-next'
import { useRefreshHandler } from '@/composables/usePullToRefresh'
import { Chart, registerables } from 'chart.js'
import { api, type FleetSummaryResponse } from '@/services/api'
import { formatAmount, formatAmountOrUnknown } from '@/currency'
import { budgetTone, budgetUsage, parseBudgetInput } from '@/utils/fleetBudget'
import { cheapestPerKmIds, rankFleetVehicles } from '@/utils/fleetRanking'
import { useConfirm } from '@/composables/useConfirm'
import { formatDistance, currentDistanceUnit, perDistance, formatPerDistanceValue, formatFuelConsumptionValue, fuelConsumptionUnit } from '@/units'
import { t } from '@/i18n'
import { formatMonthLabel } from '@/utils/dates'
import ChartTable from '@/components/charts/ChartTable.vue'
import { formatMonthName } from '@/utils/dashboard'

Chart.register(...registerables)

const loading = ref(false)
const loadError = ref<string | null>(null)
const summary = ref<FleetSummaryResponse | null>(null)
const chartCanvas = ref<HTMLCanvasElement | null>(null)
const cut = ref<ScaleCut | null>(null)
let chartInstance: Chart | null = null

const distanceUnitLabel = computed(() => currentDistanceUnit())
// On a phone a vehicle is one row; its figures open in a sheet
const detailId = ref<string | null>(null)
const detailOpen = computed({ get: () => detailId.value !== null, set: (open: boolean) => { if (!open) detailId.value = null } })
const { showAlert } = useConfirm()

const editingBudget = ref(false)
const budgetInput = ref('')
const savingBudget = ref(false)
// A budget is compared with the month's cost only in its own currency: it is not converted when the first vehicle changes
const budgetInFleetCurrency = computed(() => !!summary.value?.monthly_budget && summary.value.budget_currency === summary.value.currency)
const usage = computed(() => (summary.value?.monthly_budget && budgetInFleetCurrency.value ? budgetUsage(summary.value.current_month_cost, summary.value.monthly_budget) : null))
// The month's cost takes the verdict of the budget, when there is one
const costTone = computed(() => (usage.value ? budgetTone(usage.value.status) : 'default'))

function startEditBudget() {
  budgetInput.value = summary.value?.monthly_budget ? String(summary.value.monthly_budget) : ''
  editingBudget.value = true
}

async function saveBudget(amount: number | null) {
  if (!summary.value) return
  savingBudget.value = true
  try {
    const res = await api.setFleetBudget(amount)
    summary.value.monthly_budget = res.monthly_budget
    summary.value.budget_currency = res.budget_currency
    editingBudget.value = false
  } catch (err: any) {
    void showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger')
  } finally {
    savingBudget.value = false
  }
}

function submitBudget() {
  const amount = parseBudgetInput(budgetInput.value)
  if (amount === null) {
    void showAlert(t('fleet.budget.invalid'), t('shell.confirm.error'), 'danger')
    return
  }
  void saveBudget(amount)
}

useRefreshHandler(() => loadFleetSummary(true))

async function loadFleetSummary(silent = false) {
  if (!silent) loading.value = true
  loadError.value = null
  try {
    const data = await api.getFleetSummary()
    summary.value = data
    await nextTick()
    renderChart()
  } catch (err) {
    console.error('Failed to load fleet summary', err)
    loadError.value = (err as Error)?.message ?? ''
  } finally {
    loading.value = false
  }
}

// Vehicles in a currency other than the fleet one are not part of its totals, budget or chart
const otherCurrencyCount = computed(() => (summary.value?.totals ?? []).filter((x) => x.currency !== summary.value?.currency).reduce((n, x) => n + x.vehicle_count, 0))
const otherTotals = computed(() => (summary.value?.totals ?? []).filter((x) => x.currency !== summary.value?.currency))

// Ranking by full cost per km inside each currency, only among vehicles whose figures are complete enough to be compared
const rankedVehicles = computed(() => rankFleetVehicles(summary.value?.vehicles ?? [], summary.value?.currency ?? ''))
const detailVehicle = computed(() => rankedVehicles.value.find((v) => v.vehicle_id === detailId.value) ?? null)
const cheapestPerKmIdSet = computed(() => cheapestPerKmIds(rankedVehicles.value))
const showDrivers = computed(() => (summary.value?.member_km_shares.length ?? 0) > 1)

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

function renderChart() {
  if (!chartCanvas.value || !summary.value?.monthly_costs.length) return

  if (chartInstance) {
    chartInstance.destroy()
    chartInstance = null
  }

  const currency = summary.value.currency
  const months = summary.value.monthly_costs.map((m) => m.month)
  const energyCosts = summary.value.monthly_costs.map((m) => m.energy_cost)
  const otherCosts = summary.value.monthly_costs.map((m) => m.other_cost)
  cut.value = scaleCut(energyCosts.map((e, i) => e + otherCosts[i]))

  chartInstance = new Chart(chartCanvas.value, {
    type: 'bar',
    data: {
      labels: months.map(formatMonthLabel),
      datasets: [
        {
          label: t('fleet.chart.energyCost'),
          data: energyCosts,
          backgroundColor: 'rgba(59, 130, 246, 0.75)',
          borderRadius: 4,
          stack: 'combined',
        },
        {
          label: t('fleet.chart.otherCost'),
          data: otherCosts,
          backgroundColor: 'rgba(245, 158, 11, 0.75)',
          borderRadius: 4,
          stack: 'combined',
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: {
        mode: 'index',
        intersect: false,
      },
      plugins: {
        legend: {
          labels: {
            color: '#94a3b8',
          },
        },
        tooltip: {
          callbacks: {
            label(context) {
              const val = context.raw as number
              return `${context.dataset.label}: ${formatAmount(val, currency)}`
            },
          },
        },
      },
      scales: {
        x: {
          stacked: true,
          grid: { color: 'rgba(51, 65, 85, 0.3)' },
          ticks: { color: '#94a3b8' },
        },
        y: {
          stacked: true,
          max: cut.value?.ceiling,
          grid: { color: 'rgba(51, 65, 85, 0.3)' },
          ticks: {
            color: '#94a3b8',
            callback(value) {
              return formatAmount(Number(value), currency, 0)
            },
          },
        },
      },
    },
  })
}

onMounted(() => {
  void loadFleetSummary()
})

onBeforeUnmount(() => {
  if (chartInstance) {
    chartInstance.destroy()
    chartInstance = null
  }
})
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <PageHeader :title="t('fleet.title')" :icon="PageIcon">
      {{ t('fleet.subtitle') }}
      <template #actions>
      <button
        @click="loadFleetSummary()"
        :disabled="loading"
        class="btn btn-secondary"
      >
        <RefreshCw class="w-4 h-4" :class="{ 'animate-spin': loading }" />
        {{ t('fleet.refresh') }}
      </button>
    
      </template>
    </PageHeader>

    <LoadError v-if="loadError !== null" :message="loadError" @retry="loadFleetSummary()" />
    <ListSkeleton v-else-if="loading && !summary" :rows="3" />

    <!-- KPI Grid -->
    <div v-if="summary" class="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
      <KpiCard class="max-sm:hidden" :label="t('fleet.kpi.vehicles')" :icon="Car" :value="String(summary.total_vehicles)">
        <p class="mt-1 text-xs text-slate-400">{{ t('fleet.kpi.activeVehicles') }}</p>
      </KpiCard>

      <KpiCard
        class="max-sm:col-span-2"
        :label="t('fleet.kpi.cost')"
        :period="formatMonthName(summary.current_month)"
        :icon="TrendingDown"
        :value="formatAmount(summary.current_month_cost, summary.currency)"
        :tone="costTone"
        :tinted="costTone !== 'default'"
      >
        <p v-for="tot in otherTotals" :key="tot.currency" class="text-lg font-bold text-white tabular-nums" data-testid="fleet-currency-total">
          {{ formatAmount(tot.current_month_cost, tot.currency) }}
        </p>
        <p class="mt-1 text-xs text-slate-400">{{ otherTotals.length ? t('fleet.kpi.perCurrency') : t('fleet.kpi.currentMonthTotal') }}</p>
      </KpiCard>

      <KpiCard
        :label="t('fleet.kpi.distance')"
        :period="formatMonthName(summary.current_month)"
        :icon="Gauge"
        :value="formatDistance(summary.current_month_distance_km)"
      >
        <p class="mt-1 text-xs text-slate-400">{{ t('fleet.kpi.fleetTravelled') }}</p>
      </KpiCard>

      <KpiCard
        :label="t('fleet.kpi.energy')"
        :period="formatMonthName(summary.current_month)"
        :icon="Zap"
        :value="formatNumber(summary.current_month_energy_kwh)"
        unit=" kWh"
      >
        <p class="mt-1 text-xs text-slate-400">{{ t('fleet.kpi.energyDelivered') }}</p>
      </KpiCard>
    </div>

    <!-- Monthly budget -->
    <div v-if="summary" class="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm space-y-3">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <span class="text-xs font-medium text-slate-400">{{ t('fleet.budget.title') }}</span>
        <div v-if="!editingBudget" class="flex items-center gap-3 text-xs font-semibold">
          <button type="button" @click="startEditBudget" class="tap-text px-2 text-rose-400 hover:text-rose-300">
            {{ summary.monthly_budget ? t('fleet.budget.edit') : t('fleet.budget.set') }}
          </button>
          <button v-if="summary.monthly_budget" type="button" :disabled="savingBudget" @click="saveBudget(null)" class="tap-text text-slate-400 hover:text-white">
            {{ t('fleet.budget.remove') }}
          </button>
        </div>
      </div>
      <form v-if="editingBudget" class="flex flex-wrap items-center gap-2" @submit.prevent="submitBudget">
        <input
          v-model="budgetInput"
          type="text"
          inputmode="decimal"
          :aria-label="t('fleet.budget.title')"
          :placeholder="t('fleet.budget.placeholder', { currency: summary.currency })"
          class="field flex-1"
        />
        <button type="submit" :disabled="savingBudget" class="btn btn-lg btn-primary">{{ t('common.save') }}</button>
        <button type="button" @click="editingBudget = false" class="btn btn-lg btn-secondary">{{ t('common.cancel') }}</button>
      </form>
      <p v-if="summary.monthly_budget && !budgetInFleetCurrency && !editingBudget" class="text-xs text-slate-300 break-words" data-testid="fleet-budget-stale-currency">
        {{ t('fleet.budget.staleCurrency', { budget: formatAmount(summary.monthly_budget, summary.budget_currency ?? ''), currency: summary.currency }) }}
      </p>
      <template v-else-if="usage && summary.monthly_budget && !editingBudget">
        <div class="h-2 rounded-full bg-slate-800 overflow-hidden" role="progressbar" :aria-valuenow="Math.round(usage.percent)" aria-valuemin="0" aria-valuemax="100" :aria-label="t('fleet.budget.title')">
          <div
            class="h-full rounded-full"
            :class="usage.status === 'over' ? 'bg-rose-500' : usage.status === 'warning' ? 'bg-warning-400' : 'bg-success-500'"
            :style="{ width: `${Math.min(100, usage.percent)}%` }"
          ></div>
        </div>
        <p class="text-xs text-slate-300 break-words">
          {{ t('fleet.budget.spent', { spent: formatAmount(summary.current_month_cost, summary.currency), budget: formatAmount(summary.monthly_budget, summary.currency), percent: Math.round(usage.percent) }) }}
          <span :class="usage.status === 'over' ? 'text-rose-400' : 'text-slate-400'">
            · {{ usage.status === 'over' ? t('fleet.budget.over', { amount: formatAmount(-usage.remaining, summary.currency) }) : t('fleet.budget.remaining', { amount: formatAmount(usage.remaining, summary.currency) }) }}
          </span>
        </p>
      </template>
      <p v-else-if="!editingBudget" class="text-xs text-slate-400">{{ t('fleet.budget.none') }}</p>
      <p v-if="otherCurrencyCount > 0" class="text-xs text-slate-400" data-testid="fleet-budget-other-currency">
        {{ t('fleet.budget.otherCurrency', { currency: summary.currency, count: otherCurrencyCount }, otherCurrencyCount) }}
      </p>
    </div>

    <!-- Cost per distance comparison -->
    <div v-if="summary && summary.vehicles.length > 0" class="space-y-4 sm:rounded-xl sm:border sm:border-slate-800 sm:bg-slate-900 sm:p-5 sm:shadow-sm">
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

    <!-- Charts & Driver Share Row -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6" v-if="summary">
      <!-- Monthly Cost Chart -->
      <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm space-y-4" :class="showDrivers ? 'lg:col-span-2' : 'lg:col-span-3'">
        <div class="flex items-center justify-between border-b border-slate-800/80 pb-3">
          <h2 class="text-base font-semibold text-white">
            {{ t('fleet.chart.monthlyCostHistory') }}
          </h2>
          <span class="text-xs text-slate-400">{{ otherCurrencyCount > 0 ? t('fleet.chart.fleetCurrencyOnly', { currency: summary.currency }) : t('fleet.chart.stackedBreakdown') }}</span>
        </div>
        <div class="h-64 relative">
          <canvas ref="chartCanvas" role="img" :aria-label="t('fleet.chart.monthlyCostHistory')"></canvas>
        </div>
        <p v-if="cut" class="text-xs text-slate-400" data-testid="fleet-chart-cut">
          {{ t('fleet.chart.scaleCut', { months: cut.cutIndexes.map((i) => formatMonthLabel(summary!.monthly_costs[i].month)).join(', ') }) }}
        </p>
        <ChartTable
          :caption="t('fleet.chart.monthlyCostHistory')"
          :headers="[t('fleet.chart.month'), t('fleet.chart.energyCost'), t('fleet.chart.otherCost')]"
          :rows="summary.monthly_costs.map((m) => [formatMonthLabel(m.month), formatAmount(m.energy_cost, summary!.currency), formatAmount(m.other_cost, summary!.currency)])"
        />
      </div>

      <!-- Driver Distance Distribution: a share between drivers needs at least two of them -->
      <div v-if="showDrivers" class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm space-y-4">
        <div class="border-b border-slate-800/80 pb-3">
          <h2 class="text-base font-semibold text-white flex items-center gap-2">
            <Users class="w-4 h-4 text-slate-300" />
            {{ t('fleet.drivers.title') }}
          </h2>
          <p class="text-xs text-slate-400 mt-1">{{ t('fleet.drivers.subtitle') }}</p>
        </div>

        <div class="space-y-3">
          <div
            v-for="member in summary.member_km_shares"
            :key="member.person_id"
            class="space-y-1.5"
          >
            <div class="flex items-center justify-between text-xs">
              <span class="font-medium text-slate-200 truncate">{{ member.display_name }}</span>
              <span class="text-slate-400 shrink-0 tabular-nums">
                {{ formatDistance(member.distance_km) }} ({{ formatPercent(member.percentage) }})
              </span>
            </div>
            <div class="h-2 w-full bg-slate-800 rounded-full overflow-hidden" role="progressbar" aria-valuemin="0" aria-valuemax="100" :aria-valuenow="Math.round(Math.min(100, Math.max(0, member.percentage)))" :aria-label="member.display_name">
              <div
                class="h-full bg-slate-300 rounded-full transition-all duration-500"
                :style="{ width: `${Math.min(100, Math.max(0, member.percentage))}%` }"
              ></div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
