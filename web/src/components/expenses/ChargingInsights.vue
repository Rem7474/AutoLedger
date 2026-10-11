<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { Chart, registerables } from 'chart.js'
import { MapPin, PiggyBank } from 'lucide-vue-next'
import CostDonut from '@/components/costs/CostDonut.vue'
import { api } from '@/services/api'
import { currencySymbol, formatAmount } from '@/currency'
import { intlLocale, t } from '@/i18n'
import { formatNumber, formatPercent } from '@/utils/numbers'
import {
  PLACE_PERIODS,
  donutItems,
  groupColors,
  groupLabel,
  monthlyEnergyDatasets,
  monthlyPrices,
  savingsView,
  unclassifiedShare,
  type BreakdownMode,
  type PlaceStats,
} from '@/utils/chargingInsights'
import { placeCategoryKey } from '@/utils/chargingPlaces'

Chart.register(...registerables)

// Energy and price of the charges by month and by place, and what charging everything at home would have cost.
const props = defineProps<{ vehicleId: string; currency: string; syncKey?: unknown }>()
const emit = defineEmits<{ 'manage-places': [] }>()

const AXIS_TEXT = '#94a3b8'
const GRID_COLOR = '#1e293b'

const months = ref<number>(12)
const mode = ref<BreakdownMode>('energy')
const stats = ref<PlaceStats | null>(null)
const failed = ref(false)
let sequence = 0

async function load() {
  const current = ++sequence
  try {
    const res = await api.getChargingPlaceStats(props.vehicleId, months.value)
    if (current !== sequence) return
    stats.value = res
    failed.value = false
  } catch {
    if (current !== sequence) return
    failed.value = true
  }
}
watch(() => [props.vehicleId, months.value, props.syncKey], load, { immediate: true })

const unclassifiedLabel = computed(() => t('expenses.chargingInsights.unclassified'))
const colors = computed(() => groupColors(stats.value?.groups ?? []))
const savings = computed(() => (stats.value ? savingsView(stats.value.home_savings) : null))
const visible = computed(() => !!stats.value && stats.value.summary.sessions > 0)
const share = computed(() => (stats.value ? unclassifiedShare(stats.value) : 0))
const donut = computed(() => (stats.value ? donutItems(stats.value.groups, mode.value, colors.value, unclassifiedLabel.value) : []))
const monthLabel = (m: string) => new Date(`${m}-01T00:00:00Z`).toLocaleDateString(intlLocale(), { month: 'short', year: '2-digit', timeZone: 'UTC' })
const periodLabel = (n: number) => t(n === 0 ? 'expenses.chargingInsights.all' : `expenses.chargingInsights.months${n}`)

const energyRef = ref<HTMLCanvasElement | null>(null)
const priceRef = ref<HTMLCanvasElement | null>(null)
let energyChart: Chart | null = null
let priceChart: Chart | null = null

function draw() {
  energyChart?.destroy()
  priceChart?.destroy()
  energyChart = priceChart = null
  const s = stats.value
  if (!s || !s.series.length) return
  const labels = s.series.map((m) => monthLabel(m.month))
  if (energyRef.value) {
    energyChart = new Chart(energyRef.value, {
      type: 'bar',
      data: {
        labels,
        datasets: monthlyEnergyDatasets(s, colors.value, unclassifiedLabel.value).map((d) => ({ label: d.label, data: d.data, backgroundColor: d.color, stack: 'energy' })),
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { position: 'bottom', labels: { color: AXIS_TEXT, font: { size: 11 }, boxWidth: 10 } },
          tooltip: { callbacks: { label: (ctx) => ` ${ctx.dataset.label}: ${formatNumber(Number(ctx.raw), 1)} kWh` } },
        },
        scales: {
          x: { stacked: true, grid: { color: GRID_COLOR }, ticks: { color: AXIS_TEXT } },
          y: { stacked: true, grid: { color: GRID_COLOR }, ticks: { color: AXIS_TEXT } },
        },
      },
    })
  }
  if (priceRef.value) {
    priceChart = new Chart(priceRef.value, {
      type: 'line',
      data: {
        labels,
        datasets: [{ label: t('expenses.chargingInsights.price'), data: monthlyPrices(s), borderColor: '#38bdf8', backgroundColor: '#38bdf8', tension: 0.25, pointRadius: 3, spanGaps: false }],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { display: false },
          tooltip: { callbacks: { label: (ctx) => (ctx.raw == null ? ` ${t('expenses.chargingInsights.noPrice')}` : ` ${formatAmount(Number(ctx.raw), props.currency, 3)}/kWh`) } },
        },
        scales: {
          x: { grid: { color: GRID_COLOR }, ticks: { color: AXIS_TEXT } },
          y: { beginAtZero: true, grid: { color: GRID_COLOR }, ticks: { color: AXIS_TEXT } },
        },
      },
    })
  }
}

watch(stats, async () => {
  await nextTick()
  draw()
}, { flush: 'post' })
onBeforeUnmount(() => {
  energyChart?.destroy()
  priceChart?.destroy()
})
</script>

<template>
  <section v-if="visible && stats" class="bg-slate-900 border border-slate-800 rounded-2xl p-4 space-y-4" aria-labelledby="charging-insights-title">
    <header class="flex flex-wrap items-center justify-between gap-2">
      <h3 id="charging-insights-title" class="text-sm font-bold text-white">{{ $t('expenses.chargingInsights.title') }}</h3>
      <div class="flex items-center gap-2">
        <label for="charging-insights-period" class="sr-only">{{ $t('expenses.chargingInsights.period') }}</label>
        <select id="charging-insights-period" v-model.number="months" class="field !py-1 !text-xs w-auto">
          <option v-for="n in PLACE_PERIODS" :key="n" :value="n">{{ periodLabel(n) }}</option>
        </select>
        <button type="button" class="btn btn-secondary" @click="emit('manage-places')">
          <MapPin class="w-3.5 h-3.5 text-sky-400" />
          <span>{{ $t('expenses.chargingInsights.places') }}</span>
        </button>
      </div>
    </header>

    <dl class="grid grid-cols-2 lg:grid-cols-4 gap-2.5">
      <div class="bg-slate-800/50 border border-slate-700/50 p-3 rounded-xl">
        <dt class="text-xs text-slate-400">{{ $t('expenses.chargingInsights.energy') }}</dt>
        <dd class="text-lg font-extrabold text-white">{{ $t('expenses.chargingInsights.kwh', { value: formatNumber(stats.summary.kwh_added, 0) }) }}</dd>
        <dd class="text-xs text-slate-400">{{ $t('expenses.chargingInsights.sessionsCount', { count: stats.summary.sessions }, stats.summary.sessions) }}</dd>
      </div>
      <div class="bg-slate-800/50 border border-slate-700/50 p-3 rounded-xl">
        <dt class="text-xs text-slate-400">{{ $t('expenses.chargingInsights.spent') }}</dt>
        <dd class="text-lg font-extrabold text-white">{{ formatAmount(stats.summary.cost, currency) }}</dd>
        <dd v-if="stats.summary.unknown_sessions" class="text-xs text-warning-400">{{ $t('expenses.chargingInsights.unknownCost', { count: stats.summary.unknown_sessions, kwh: formatNumber(stats.summary.unknown_kwh, 0) }) }}</dd>
      </div>
      <div class="bg-slate-800/50 border border-slate-700/50 p-3 rounded-xl">
        <dt class="text-xs text-slate-400">{{ $t('expenses.chargingInsights.price') }}</dt>
        <dd class="text-lg font-extrabold text-white">
          <template v-if="stats.summary.price_per_kwh !== null">{{ $t('expenses.chargingInsights.pricePerKwh', { price: formatAmount(stats.summary.price_per_kwh, currency, 3) }) }}</template>
          <template v-else>—</template>
        </dd>
        <dd v-if="stats.summary.free_sessions" class="text-xs text-slate-400">{{ $t('expenses.chargingInsights.free', { count: stats.summary.free_sessions }) }}</dd>
      </div>
      <div class="bg-success-500/5 border border-success-500/30 p-3 rounded-xl">
        <dt class="text-xs font-medium text-success-400 flex items-center gap-1.5">
          <PiggyBank class="w-3.5 h-3.5" aria-hidden="true" />
          {{ $t('expenses.chargingInsights.savings.title') }}
        </dt>
        <template v-if="savings && savings.state === 'ok'">
          <dd class="text-lg font-extrabold" :class="savings.better ? 'text-success-400' : 'text-warning-400'">
            {{ $t(savings.better ? 'expenses.chargingInsights.savings.better' : 'expenses.chargingInsights.savings.worse', { amount: formatAmount(savings.amount, currency) }) }}
          </dd>
          <dd class="text-xs text-slate-400">
            {{ $t(savings.better ? 'expenses.chargingInsights.savings.basis' : 'expenses.chargingInsights.savings.worseBasis', {
              kwh: formatNumber(savings.awayKwh, 0),
              cost: formatAmount(savings.awayCost, currency),
              home: formatAmount(savings.hypotheticalCost, currency),
              price: formatAmount(savings.homePrice, currency, 3),
            }) }}
          </dd>
          <dd v-if="savings.unclassifiedKwh > 0" class="text-xs text-slate-500">{{ $t('expenses.chargingInsights.savings.unclassified', { kwh: formatNumber(savings.unclassifiedKwh, 0) }) }}</dd>
        </template>
        <dd v-else-if="savings" class="text-xs text-slate-400 mt-1">{{ $t(savings.reasonKey) }}</dd>
      </div>
    </dl>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <figure class="space-y-1.5">
        <figcaption class="text-xs font-bold text-white">{{ $t('expenses.chargingInsights.monthlyEnergy') }}</figcaption>
        <div class="h-52"><canvas ref="energyRef" :aria-label="$t('expenses.chargingInsights.monthlyEnergy')"></canvas></div>
      </figure>
      <figure class="space-y-1.5">
        <figcaption class="text-xs font-bold text-white">{{ $t('expenses.chargingInsights.monthlyPrice', { cur: currencySymbol(currency) }) }}</figcaption>
        <div class="h-52"><canvas ref="priceRef" :aria-label="$t('expenses.chargingInsights.monthlyPrice', { cur: currencySymbol(currency) })"></canvas></div>
      </figure>
    </div>

    <div class="space-y-2">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h4 class="text-xs font-bold text-white">{{ $t('expenses.chargingInsights.breakdown') }}</h4>
        <div class="flex items-center gap-1 bg-slate-900/80 p-0.5 rounded-lg border border-slate-700/70" role="group">
          <button
            v-for="m in (['energy', 'cost'] as const)"
            :key="m"
            type="button"
            :aria-pressed="mode === m"
            :class="['px-2.5 py-1 text-xs font-semibold rounded-md transition-colors', mode === m ? 'bg-sky-600 text-white' : 'text-slate-400 hover:text-white']"
            @click="mode = m"
          >
            {{ $t(m === 'energy' ? 'expenses.chargingInsights.byEnergy' : 'expenses.chargingInsights.byCost') }}
          </button>
        </div>
      </div>
      <div class="grid grid-cols-1 md:grid-cols-5 gap-4 items-start">
        <div class="md:col-span-2 h-56 relative">
          <CostDonut
            :items="donut"
            :chart-label="$t('expenses.chargingInsights.chartLabel')"
            :unit="mode === 'energy' ? 'kWh' : ''"
            :currency="currency"
          />
        </div>
        <div class="md:col-span-3 overflow-x-auto">
          <table class="w-full text-xs">
            <thead>
              <tr class="text-slate-400 text-left">
                <th class="py-1 pr-2 font-medium">{{ $t('expenses.chargingInsights.colPlace') }}</th>
                <th class="py-1 px-2 font-medium text-right">{{ $t('expenses.chargingInsights.colEnergy') }}</th>
                <th class="py-1 px-2 font-medium text-right">{{ $t('expenses.chargingInsights.colCost') }}</th>
                <th class="py-1 pl-2 font-medium text-right">{{ $t('expenses.chargingInsights.colPrice', { cur: currencySymbol(currency) }) }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="g in stats.groups" :key="g.key || 'none'" class="border-t border-slate-800 align-top">
                <td class="py-1.5 pr-2">
                  <span class="inline-block w-2.5 h-2.5 rounded-full mr-1.5" :style="{ backgroundColor: colors[g.key] }"></span>
                  <span class="font-semibold text-white">{{ groupLabel(g, unclassifiedLabel) }}</span>
                  <span v-if="g.category" class="text-slate-500"> · {{ $t(placeCategoryKey(g.category)) }}</span>
                  <p class="text-slate-400">
                    {{ $t('expenses.chargingInsights.sessionsCount', { count: g.sessions }, g.sessions) }}
                    <template v-if="g.free_sessions"> · {{ $t('expenses.chargingInsights.free', { count: g.free_sessions }) }}</template>
                  </p>
                  <p v-if="g.unknown_sessions" class="text-warning-400">{{ $t('expenses.chargingInsights.unknownCost', { count: g.unknown_sessions, kwh: formatNumber(g.unknown_kwh, 0) }) }}</p>
                </td>
                <td class="py-1.5 px-2 text-right text-slate-200 tabular-nums">{{ formatNumber(g.kwh_added, 1) }}</td>
                <td class="py-1.5 px-2 text-right text-slate-200 tabular-nums">{{ formatAmount(g.cost, currency) }}</td>
                <td class="py-1.5 pl-2 text-right text-slate-200 tabular-nums">{{ g.price_per_kwh !== null ? formatAmount(g.price_per_kwh, currency, 3) : '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
      <p class="text-xs text-slate-500">{{ $t('expenses.chargingInsights.basis') }}</p>
      <p v-if="share > 0.05" class="text-xs text-warning-400">
        {{ $t('expenses.chargingInsights.noteUnclassified', { share: formatPercent(share * 100, 0) }) }}
        <button type="button" class="underline hover:text-warning-300" @click="emit('manage-places')">{{ $t('expenses.chargingInsights.classify') }}</button>
      </p>
      <p v-if="stats.summary.zero_kwh_sessions" class="text-xs text-slate-500">{{ $t('expenses.chargingInsights.noteZero', { count: stats.summary.zero_kwh_sessions }, stats.summary.zero_kwh_sessions) }}</p>
    </div>
  </section>
</template>
