<script setup lang="ts">
import { t } from '@/i18n'
import { formatMonthLabel } from '@/utils/dates'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { Chart, registerables } from 'chart.js'
import { computeMonthFixedVariable, monthlyCostSeries, type MonthDetailMode } from '@/utils/dashboard'
import { usePeriodStore } from '@/stores/period'
import { filterMonthsByPeriod } from '@/utils/period'
import { currencySymbol, formatAmount, formatAxisAmount } from '@/currency'
import { distanceUnit, perDistance } from '@/units'
import { useVehicleStore } from '@/stores/vehicle'
import ChartTable from '@/components/charts/ChartTable.vue'

Chart.register(...registerables)

// Stacked monthly costs per item; a click on a month opens its detail
const props = defineProps<{ monthlyCosts: any[] | null }>()
const emit = defineEmits<{ 'open-month': [month: any] }>()
const vehicleStore = useVehicleStore()
const currency = computed(() => vehicleStore.currency)

const monthlyChartRef = ref<HTMLCanvasElement | null>(null)
const periodStore = usePeriodStore()
const filteredMonthlyCosts = computed(() => filterMonthsByPeriod(props.monthlyCosts || [], periodStore.costs))
let monthlyChartInstance: Chart | null = null

// Economic: each cost spread over the period it covers, so a one-off purchase does not crush the other months.
// Cash: what was paid in the month.
const mode = ref<MonthDetailMode>('economic')
// The legend takes the short names so that it stays on two lines on a phone; the table and the tooltips keep the full ones
const seriesLabel = (key: string, short = false): string => {
  const labels: Record<string, string> = {
    energy: t('dashboard.donut.energy'),
    tolls: short ? t('dashboard.monthlyCostChart.tollsShort') : t('dashboard.donut.tolls'),
    tires: t('dashboard.monthlyCostChart.tires'),
    maintenance: t('dashboard.monthlyCostChart.maintenance'),
    insurance: t('dashboard.donut.insurance'),
    financing: t('dashboard.monthlyCostChart.financing'),
    other: short ? t('dashboard.monthlyCostChart.otherShort') : t('dashboard.donut.other'),
  }
  return labels[key] ?? key
}
// The same figures as a table for screen readers: one column per cost item, the month total last
const tableData = computed(() => {
  const { series, totals, nets } = monthlyCostSeries(filteredMonthlyCosts.value, mode.value)
  // The net column appears only when a month of the range has carpool revenue, so the table is unchanged otherwise
  const hasNet = nets.some((n) => n !== null)
  return {
    headers: [
      t('dashboard.monthlyCostChart.month'),
      ...series.map((s) => seriesLabel(s.key)),
      t('dashboard.monthlyCostChart.total'),
      ...(hasNet ? [t('dashboard.monthlyCostChart.netColumn')] : []),
    ],
    rows: filteredMonthlyCosts.value.map((m: any, i: number) => [
      formatMonthLabel(m.month),
      ...series.map((s) => formatAmount(s.data[i], currency.value)),
      formatAmount(totals[i], currency.value),
      ...(hasNet ? [nets[i] === null ? '' : formatAmount(nets[i] as number, currency.value)] : []),
    ]),
  }
})

function renderChart() {
  if (!monthlyChartRef.value || !props.monthlyCosts) return
  if (monthlyChartInstance) monthlyChartInstance.destroy()

  const filteredList = filteredMonthlyCosts.value
  const filteredLabels = filteredList.map((m: any) => formatMonthLabel(m.month))
  const { series, totals, nets, netPerKm } = monthlyCostSeries(filteredList, mode.value)

  monthlyChartInstance = new Chart(monthlyChartRef.value, {
    type: 'bar',
    data: {
      labels: filteredLabels,
      datasets: series.map((s) => ({
        label: `${seriesLabel(s.key)} (${currencySymbol(currency.value)})`,
        data: s.data,
        backgroundColor: s.color,
        borderRadius: 4,
      })),
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: {
        mode: 'index',
        intersect: false,
      },
      onClick: (event: any) => {
        if (!monthlyChartInstance) return
        const points = monthlyChartInstance.getElementsAtEventForMode(
          event.native || event,
          'index',
          { intersect: false },
          true
        )
        if (points && points.length > 0) {
          const dataIndex = points[0].index
          const monthItem = filteredList[dataIndex]
          if (monthItem) {
            emit('open-month', monthItem)
          }
        }
      },
      onHover: (event: any, elements: any[]) => {
        const canvas = monthlyChartRef.value
        if (canvas) {
          canvas.style.cursor = elements && elements.length > 0 ? 'pointer' : 'default'
        }
      },
      plugins: {
        legend: {
          position: 'top',
          labels: {
            color: '#94a3b8',
            font: { size: 11 },
            boxWidth: 10,
            boxHeight: 10,
            padding: 10,
            // The currency is already in the chart title: the legend names only the series
            generateLabels: (chart) =>
              Chart.defaults.plugins.legend.labels.generateLabels(chart).map((item) => ({
                ...item,
                text: seriesLabel(series[item.datasetIndex ?? 0].key, true),
              })),
          },
        },
        tooltip: {
          callbacks: {
            label: (context) => {
              return `${context.dataset.label} : ${formatAmount(Number(context.raw || 0), currency.value)}`
            },
            footer: (items) => {
              const dataIndex = items[0]?.dataIndex
              const total = dataIndex !== undefined ? totals[dataIndex] : items.reduce((sum, item) => sum + (Number(item.raw) || 0), 0)
              const lines = [t('dashboard.monthlyCostChart.monthTotal', { total: formatAmount(total, currency.value) })]
              if (dataIndex !== undefined) {
                const monthItem = filteredList[dataIndex]
                const fv = computeMonthFixedVariable(monthItem, mode.value)
                if (fv.totalAmount > 0) {
                  lines.push(t('dashboard.monthlyCostChart.fixedVariableTooltip', { fixedPct: fv.fixedPct, variablePct: fv.variablePct }))
                }
                // A month with carpool revenue: what the passengers paid brings the total down, the bars stay gross
                const net = nets[dataIndex]
                if (net !== null) {
                  lines.push(t('dashboard.monthlyCostChart.netOfCarpooling', { total: formatAmount(net, currency.value) }))
                  const perKm = netPerKm[dataIndex]
                  if (perKm !== null) {
                    lines.push(
                      t('dashboard.monthlyCostChart.netCostPerKm', {
                        unit: distanceUnit(),
                        cost: formatAmount(perDistance(perKm), currency.value, 3),
                      }),
                    )
                  }
                }
              }
              return lines
            },
          },
        },
      },
      scales: {
        x: { stacked: true, grid: { color: '#1e293b' }, ticks: { color: '#64748b', maxRotation: 0, autoSkipPadding: 8 } },
        y: {
          stacked: true,
          beginAtZero: true,
          grid: { color: '#1e293b' },
          ticks: { color: '#64748b', callback: (v) => formatAxisAmount(Number(v), currency.value) },
        },
      },
    },
  })
}

onMounted(renderChart)
watch([() => props.monthlyCosts, () => periodStore.costs, mode], renderChart, { flush: 'post' })
onUnmounted(() => {
  if (monthlyChartInstance) monthlyChartInstance.destroy()
})
</script>

<template>
  <div class="lg:col-span-2 flex flex-col card p-4 sm:p-5 shadow-sm">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 mb-4">
      <h3 class="text-sm font-bold text-white flex items-center gap-2">
        <span>{{ $t('dashboard.monthlyCostChart.monthlyExpenseTrend', { cur: currencySymbol(currency) }) }}</span>
      </h3>
      <div class="flex items-center gap-1 bg-slate-800/60 rounded-lg p-0.5 self-start" role="group" :aria-label="$t('dashboard.monthlyCostChart.view')">
        <button
          v-for="opt in (['economic', 'cash'] as const)"
          :key="opt"
          type="button"
          :aria-pressed="mode === opt"
          :class="['tap-text justify-center px-2.5 text-xs font-semibold rounded-md transition-colors pointer-coarse:min-w-11', mode === opt ? 'bg-rose-600 text-white' : 'text-slate-400 hover:text-white']"
          @click="mode = opt"
        >
          {{ opt === 'economic' ? $t('dashboard.monthlyCostChart.economic') : $t('dashboard.monthlyCostChart.cash') }}
        </button>
      </div>
    </div>
    <p class="-mt-2 mb-3 text-xs text-slate-400">{{ mode === 'economic' ? $t('dashboard.monthlyCostChart.economicHint') : $t('dashboard.monthlyCostChart.cashHint') }}</p>
    <div class="relative min-h-52 flex-1 sm:min-h-72">
      <canvas ref="monthlyChartRef" role="img" :aria-label="`${$t('dashboard.monthlyCostChart.monthlyCostTrendByCategory')} · ${mode === 'economic' ? $t('dashboard.monthlyCostChart.economic') : $t('dashboard.monthlyCostChart.cash')}`"></canvas>
    </div>
    <ChartTable :caption="$t('dashboard.monthlyCostChart.monthlyCostTrendByCategory')" :headers="tableData.headers" :rows="tableData.rows" />
  </div>
</template>
