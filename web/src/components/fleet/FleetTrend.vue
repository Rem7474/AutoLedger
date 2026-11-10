<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Users } from '@/icons'
import { Chart, registerables } from 'chart.js'
import ChartTable from '@/components/charts/ChartTable.vue'
import type { FleetSummaryResponse } from '@/services/api'
import { formatAmount } from '@/currency'
import { THEME_HEX } from '@/utils/themes'
import { scaleCut, type ScaleCut } from '@/utils/fleetChart'
import { formatPercent } from '@/utils/numbers'
import { formatDistance } from '@/units'
import { formatMonthLabel } from '@/utils/dates'
import { t } from '@/i18n'

Chart.register(...registerables)

// The twelve months of household cost, stacked energy and other, and the share of the distance each driver drove.
const props = defineProps<{ summary: FleetSummaryResponse }>()

const chartCanvas = ref<HTMLCanvasElement | null>(null)
const cut = ref<ScaleCut | null>(null)
let chartInstance: Chart | null = null

// Vehicles in a currency other than the fleet one are not part of the chart
const otherCurrencyCount = computed(() => props.summary.totals.filter((x) => x.currency !== props.summary.currency).reduce((n, x) => n + x.vehicle_count, 0))
// A share between drivers needs at least two of them
const showDrivers = computed(() => props.summary.member_km_shares.length > 1)

function renderChart() {
  if (!chartCanvas.value || !props.summary.monthly_costs.length) return

  if (chartInstance) {
    chartInstance.destroy()
    chartInstance = null
  }

  const currency = props.summary.currency
  const months = props.summary.monthly_costs.map((m) => m.month)
  const energyCosts = props.summary.monthly_costs.map((m) => m.energy_cost)
  const otherCosts = props.summary.monthly_costs.map((m) => m.other_cost)
  cut.value = scaleCut(energyCosts.map((e, i) => e + otherCosts[i]))

  chartInstance = new Chart(chartCanvas.value, {
    type: 'bar',
    data: {
      labels: months.map(formatMonthLabel),
      datasets: [
        {
          label: t('fleet.chart.energyCost'),
          data: energyCosts,
          backgroundColor: THEME_HEX.energy,
          borderRadius: 4,
          stack: 'combined',
        },
        {
          label: t('fleet.chart.otherCost'),
          data: otherCosts,
          backgroundColor: THEME_HEX.other,
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

watch(() => props.summary, async () => { await nextTick(); renderChart() })
onMounted(renderChart)
onBeforeUnmount(() => {
  if (chartInstance) {
    chartInstance.destroy()
    chartInstance = null
  }
})
</script>

<template>
  <!-- Charts & Driver Share Row -->
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
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
        {{ t('fleet.chart.scaleCut', { months: cut.cutIndexes.map((i) => formatMonthLabel(summary.monthly_costs[i].month)).join(', ') }) }}
      </p>
      <ChartTable
        :caption="t('fleet.chart.monthlyCostHistory')"
        :headers="[t('fleet.chart.month'), t('fleet.chart.energyCost'), t('fleet.chart.otherCost')]"
        :rows="summary.monthly_costs.map((m) => [formatMonthLabel(m.month), formatAmount(m.energy_cost, summary.currency), formatAmount(m.other_cost, summary.currency)])"
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
</template>
