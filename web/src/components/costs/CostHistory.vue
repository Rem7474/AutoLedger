<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { Chart, registerables } from 'chart.js'
import KpiCard from '@/components/KpiCard.vue'
import LoadError from '@/components/LoadError.vue'
import MonthlyRangeSelector from '@/components/dashboard/MonthlyRangeSelector.vue'
import AppDateRangePicker from '@/components/AppDateRangePicker.vue'
import { api } from '@/services/api'
import { currencySymbol, formatAmount } from '@/currency'
import { t } from '@/i18n'
import { COST_COLORS } from '@/utils/costBreakdown'
import { ROLLING_AVERAGE_MONTHS, monthsOfPeriod, periodLabel, rollingAverage, shortMonth, summarizePeriod, type CostMonth } from '@/utils/costHistory'
import type { MonthlyRangeKey } from '@/utils/dashboard'

Chart.register(...registerables)

// What a family of costs paid month by month: the figures of a period chosen like everywhere else (presets or a range
// of days), the monthly bars and the smoothed average of the last twelve months. The list of the page sits below it.
const props = defineProps<{
  vehicleId: string
  kind: 'tolls' | 'service' | 'fixed'
  currency: string
  /** Changes when the list below was edited: the history is loaded again. */
  syncKey?: unknown
}>()

const AXIS_TEXT = '#94a3b8'
const GRID_COLOR = '#1e293b'
const AVERAGE_COLOR = '#e2e8f0'
const barColor = { tolls: COST_COLORS.tolls, service: COST_COLORS.maintenance, fixed: COST_COLORS.insurance }

// A year by default: the smoothed average needs a year to mean something
const range = ref<MonthlyRangeKey>('1Y')
const customRange = ref<[string, string] | null>(null)
const today = new Date().toLocaleDateString('en-CA')

const history = ref<CostMonth[]>([])
const failed = ref(false)
const loaded = ref(false)
let sequence = 0

async function load() {
  const current = ++sequence
  try {
    const res = await api.getCostHistory(props.vehicleId, props.kind)
    if (current !== sequence) return
    history.value = res.months
    failed.value = false
  } catch {
    if (current !== sequence) return
    failed.value = true
  } finally {
    if (current === sequence) loaded.value = true
  }
}
watch(() => [props.vehicleId, props.kind, props.syncKey], load, { immediate: true })

const months = computed(() => monthsOfPeriod(history.value, range.value, customRange.value))
const summary = computed(() => summarizePeriod(months.value))
const averages = computed(() => rollingAverage(history.value))
const smoothed = computed(() => averages.value[averages.value.length - 1] ?? null)
const smoothedPeriod = computed(() => periodLabel(history.value.slice(-ROLLING_AVERAGE_MONTHS)))
// The average of the months drawn, which a custom range can leave empty
const averageOfDrawn = computed(() => {
  const first = months.value[0]
  if (!first) return []
  const from = history.value.findIndex((m) => m.month === first.month)
  return averages.value.slice(from, from + months.value.length)
})
const label = computed(() => t(`expenses.costHistory.${props.kind}`))
const money = (v: number | null) => (v == null ? null : formatAmount(v, props.currency))

const canvas = ref<HTMLCanvasElement | null>(null)
let chart: Chart | null = null

function draw() {
  chart?.destroy()
  chart = null
  if (!canvas.value || !months.value.length) return
  chart = new Chart(canvas.value, {
    data: {
      labels: months.value.map((m) => shortMonth(m.month)),
      datasets: [
        { type: 'bar', label: label.value, data: months.value.map((m) => m.amount), backgroundColor: barColor[props.kind], pointStyle: 'rect' as const, order: 2 },
        { type: 'line', label: t('expenses.costHistory.smoothed'), data: averageOfDrawn.value, borderColor: AVERAGE_COLOR, backgroundColor: AVERAGE_COLOR, borderDash: [5, 4], borderWidth: 2, pointRadius: 0, pointStyle: 'line' as const, tension: 0, order: 1 },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: { position: 'bottom', labels: { color: AXIS_TEXT, font: { size: 11 }, boxWidth: 14, usePointStyle: true } },
        tooltip: { callbacks: { label: (ctx) => ` ${ctx.dataset.label}: ${formatAmount(Number(ctx.raw), props.currency)}` } },
      },
      scales: {
        x: { grid: { color: GRID_COLOR }, ticks: { color: AXIS_TEXT, maxRotation: 0, autoSkip: true } },
        y: { beginAtZero: true, grid: { color: GRID_COLOR }, ticks: { color: AXIS_TEXT } },
      },
    },
  })
}

watch([months, averageOfDrawn], async () => {
  await nextTick()
  draw()
}, { flush: 'post' })
onBeforeUnmount(() => chart?.destroy())
</script>

<template>
  <LoadError v-if="failed" @retry="load" />
  <section v-else-if="loaded" class="space-y-4 rounded-2xl border border-slate-800 bg-slate-900 p-4" :aria-label="$t('expenses.costHistory.title', { label })">
    <header class="space-y-2">
      <h3 class="text-sm font-bold text-white">{{ $t('expenses.costHistory.title', { label }) }}</h3>
      <div class="flex flex-wrap items-center gap-2">
        <span class="text-xs font-semibold text-slate-300">{{ $t('expenses.costHistory.period') }}</span>
        <MonthlyRangeSelector v-model="range" :custom="customRange !== null" :label="$t('expenses.costHistory.period')" @pick="customRange = null" />
        <AppDateRangePicker v-model="customRange" :max="today" :aria-label="$t('expenses.costHistory.period')" />
      </div>
    </header>

    <div class="grid grid-cols-2 gap-2.5 sm:grid-cols-3">
      <KpiCard class="col-span-2 sm:col-span-1" compact :label="$t('expenses.costHistory.total')" :value="money(summary.total)" :period="periodLabel(months)" />
      <KpiCard compact :label="$t('expenses.costHistory.average')" :value="money(summary.monthlyAverage)" :unit="$t('expenses.costHistory.perMonth')" :period="$t('expenses.costHistory.months', { count: summary.months }, summary.months)" />
      <KpiCard compact term="smoothedMonthlyCost" :label="$t('expenses.costHistory.smoothedAverage')" :value="money(smoothed)" :unit="$t('expenses.costHistory.perMonth')" :period="smoothedPeriod" />
    </div>

    <p v-if="!months.length" class="py-6 text-center text-xs text-slate-400">{{ $t('expenses.costHistory.noFullMonth') }}</p>
    <figure v-else class="space-y-1.5">
      <figcaption class="text-xs font-bold text-white">{{ $t('expenses.costHistory.monthly', { cur: currencySymbol(currency) }) }}</figcaption>
      <div class="h-52"><canvas ref="canvas" :aria-label="$t('expenses.costHistory.monthly', { cur: currencySymbol(currency) })"></canvas></div>
    </figure>
  </section>
</template>
