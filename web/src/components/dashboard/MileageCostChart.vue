<script setup lang="ts">
import { intlLocale, t } from '@/i18n'
import { formatMonthLabel } from '@/utils/dates'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { Chart, registerables } from 'chart.js'
import { Activity, PieChart } from '@/icons'
import { MIN_PLOTTED_DISTANCE_KM, plottedCostPerKm } from '@/utils/dashboard'
import { costPerKmSeries, monthKeyOf } from '@/utils/partialMonth'
import { usePeriodStore } from '@/stores/period'
import { filterMonthsByPeriod } from '@/utils/period'
import { currencySymbol, formatAmount } from '@/currency'
import { useVehicleStore } from '@/stores/vehicle'
import GlossaryTip from '@/components/GlossaryTip.vue'
import ChartTable from '@/components/charts/ChartTable.vue'
import { distanceUnit, formatDistanceValue, kmToDisplayDistance, perDistance } from '@/units'

Chart.register(...registerables)

// Monthly distance (bars, left axis) and cost per distance (line, right axis) in one chart; a click on a month opens its detail
const props = defineProps<{ monthlyCosts: any[] | null }>()
const emit = defineEmits<{ 'open-month': [month: any] }>()
const vehicleStore = useVehicleStore()

const canvasRef = ref<HTMLCanvasElement | null>(null)
const periodStore = usePeriodStore()
const months = computed(() => filterMonthsByPeriod(props.monthlyCosts || [], periodStore.costs))
let chart: Chart | null = null

// The same figures as a table for screen readers
const tableRows = computed(() =>
  months.value.map((m: any) => {
    const cost = plottedCostPerKm(m)
    return [
      formatMonthLabel(m.month) + (m.month === monthKeyOf(new Date()) ? ` (${t('dashboard.mileageCostChart.inProgress')})` : ''),
      formatDistanceValue(m.distance_km || 0),
      cost == null ? '—' : formatAmount(perDistance(cost), vehicleStore.currency, 3),
    ]
  }),
)

// The month in progress, said in words under the chart: its point is hollow and held at the top of the scale
const partialNote = computed(() => {
  const series = costPerKmSeries(months.value, new Date())
  if (!series.partial) return null
  return t('dashboard.mileageCostChart.inProgressNote', {
    month: formatMonthLabel(months.value[series.partial.index].month),
    cost: formatAmount(perDistance(series.partial.value), vehicleStore.currency, 3),
    unit: distanceUnit(),
  })
})

const AXIS = '#64748b'
const GRID = '#1e293b'
const BARS = 'rgba(148, 163, 184, 0.55)'
const LINE = '#e2e8f0'

function renderCharts() {
  chart?.destroy()
  chart = null
  if (!props.monthlyCosts || !canvasRef.value) return
  const list = months.value
  const labels = list.map((m: any) => formatMonthLabel(m.month))
  // The month in progress is a hollow point apart, outside the scale; a month under the minimum distance, or without a
  // known cost, is a gap rather than a point
  const series = costPerKmSeries(list, new Date())
  const top = series.scaleMax === undefined ? undefined : perDistance(series.scaleMax)
  const partial = series.partial
  const partialAt = partial ? perDistance(partial.value) : null
  const costLabel = t('dashboard.mileageCostChart.averageCostPerKm', { unit: distanceUnit(), cur: currencySymbol(vehicleStore.currency) })
  const at = (i: number) => (partial && partialAt !== null && i === partial.index ? partialAt : null)
  chart = new Chart(canvasRef.value, {
    type: 'bar',
    data: {
      labels,
      datasets: [
        {
          type: 'bar',
          label: t('dashboard.mileageCostChart.kmDriven', { unit: distanceUnit() }),
          data: list.map((m: any) => Math.round(kmToDisplayDistance(m.distance_km || 0))),
          backgroundColor: BARS,
          hoverBackgroundColor: 'rgba(148, 163, 184, 0.8)',
          borderRadius: 6,
          yAxisID: 'y',
          order: 2,
        },
        {
          type: 'line',
          label: costLabel,
          data: series.complete.map((v) => (v == null ? null : perDistance(v))),
          borderColor: LINE,
          backgroundColor: LINE,
          borderWidth: 2.5,
          pointRadius: 4,
          pointHoverRadius: 6,
          tension: 0.3,
          spanGaps: false,
          yAxisID: 'y1',
          order: 1,
        },
        {
          type: 'line',
          label: `${costLabel} · ${t('dashboard.mileageCostChart.inProgress')}`,
          // Drawn at the top of the scale when it is above it: the tooltip and the table give the figure
          data: list.map((_m: any, i: number) => {
            const v = at(i)
            return v === null ? null : top === undefined ? v : Math.min(v, top)
          }),
          actual: list.map((_m: any, i: number) => at(i)),
          showLine: false,
          clip: false,
          borderColor: LINE,
          backgroundColor: 'transparent',
          borderWidth: 2,
          pointRadius: 5,
          pointHoverRadius: 7,
          pointStyle: 'circle',
          yAxisID: 'y1',
          order: 1,
        } as any,
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      onClick: (event: any) => {
        const points = chart?.getElementsAtEventForMode(event.native || event, 'index', { intersect: false }, true)
        const month = points?.length ? list[points[0].index] : null
        if (month) emit('open-month', month)
      },
      onHover: (_event: any, elements: any[]) => {
        if (canvasRef.value) canvasRef.value.style.cursor = elements?.length ? 'pointer' : 'default'
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
            // The month in progress is explained under the chart, not in the legend
            filter: (item) => item.datasetIndex !== 2,
          },
        },
        tooltip: {
          callbacks: {
            label: (context: any) => {
              if (context.datasetIndex === 0) {
                return t('dashboard.mileageCostChart.tooltipDistance', { unit: distanceUnit(), distance: Number(context.raw).toLocaleString(intlLocale()) })
              }
              const actual = context.dataset.actual?.[context.dataIndex]
              const text = t('dashboard.mileageCostChart.tooltipCostPerKm', { unit: distanceUnit(), cost: formatAmount(actual ?? Number(context.raw), vehicleStore.currency, 3) })
              return actual == null ? text : `${text} (${t('dashboard.mileageCostChart.inProgress')})`
            },
          },
        },
      },
      scales: {
        x: { grid: { color: GRID }, ticks: { color: AXIS, maxRotation: 0, autoSkipPadding: 8 } },
        y: {
          position: 'left',
          beginAtZero: true,
          grid: { color: GRID },
          ticks: { color: AXIS, callback: (v: any) => Number(v).toLocaleString(intlLocale()) },
          title: { display: true, text: t('dashboard.mileageCostChart.distanceKm', { unit: distanceUnit() }), color: AXIS, font: { size: 11 } },
        },
        y1: {
          position: 'right',
          beginAtZero: true,
          grid: { display: false },
          ...(top !== undefined ? { max: top } : {}),
          ticks: { color: AXIS, callback: (v: any) => formatAmount(Number(v), vehicleStore.currency, 2) },
          title: { display: true, text: t('dashboard.mileageCostChart.costKm', { unit: distanceUnit(), cur: currencySymbol(vehicleStore.currency) }), color: AXIS, font: { size: 11 } },
        },
      },
    },
  })
}

onMounted(renderCharts)
watch([() => props.monthlyCosts, () => periodStore.costs], renderCharts, { flush: 'post' })
onUnmounted(() => {
  chart?.destroy()
})
</script>

<template>
  <div class="card space-y-3 p-5 shadow-sm">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
      <div>
        <h3 class="text-sm font-bold text-white flex items-center gap-2">
          <Activity class="w-4 h-4 text-slate-400" aria-hidden="true" />
          <span>{{ $t('dashboard.mileageCostChart.monthlyMileageAndCostPer', { unit: distanceUnit() }) }}</span>
          <GlossaryTip term="monthBars" class="-ml-1" />
        </h3>
      </div>
      <button
        v-if="months.length"
        type="button"
        class="self-start sm:self-auto px-2.5 py-1 bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors"
        @click="emit('open-month', months[months.length - 1])"
      >
        <PieChart class="w-3.5 h-3.5" aria-hidden="true" />
        <span>{{ $t('dashboard.mileageCostChart.lastMonthSDetail') }}</span>
      </button>
    </div>

    <div class="h-56 sm:h-72">
      <canvas ref="canvasRef" role="img" :aria-label="$t('dashboard.mileageCostChart.monthlyMileageAndCostPer', { unit: distanceUnit() })"></canvas>
    </div>
    <p v-if="partialNote" class="text-xs text-slate-300" data-testid="cost-km-in-progress">{{ partialNote }}</p>
    <p class="text-xs text-slate-400">{{ $t('dashboard.mileageCostChart.minDistanceNote', { distance: formatDistanceValue(MIN_PLOTTED_DISTANCE_KM), unit: distanceUnit() }) }}</p>
    <ChartTable
      :caption="$t('dashboard.mileageCostChart.monthlyMileageAndCostPer', { unit: distanceUnit() })"
      :headers="[$t('dashboard.monthlyCostChart.month'), $t('dashboard.mileageCostChart.distanceKm', { unit: distanceUnit() }), $t('dashboard.mileageCostChart.costKm', { unit: distanceUnit(), cur: currencySymbol(vehicleStore.currency) })]"
      :rows="tableRows"
    />
  </div>
</template>
