<script setup lang="ts">
import { intlLocale, t } from '@/i18n'
import { formatDistance } from '@/units'
import { computed } from 'vue'
import { ChevronLeft, ChevronRight, X, Search, Calendar } from 'lucide-vue-next'
import AppDatePicker from '@/components/AppDatePicker.vue'
import ListSummary from '@/components/ListSummary.vue'
import { currentYearMonth, formatMonthLabel, presetRange, shiftMonth, sumDriveCosts, type PeriodPreset } from '@/utils/drives'
import { formatAmount } from '@/currency'
import { useVehicleStore } from '@/stores/vehicle'

type PeriodMode = 'ALL' | 'MONTH' | 'CUSTOM'

// Period, month and address filters shared by the drives list and the trips list, plus the metrics of what is listed
// (the current page of drives, or the trips in trip mode).
// Every change of a filter is reported with change so the page reloads from the first page.
const props = withDefaults(defineProps<{ total: number; loading: boolean; drives: any[]; mode?: 'DRIVES' | 'TRIPS'; collapsed?: boolean }>(), { mode: 'DRIVES', collapsed: false })
const emit = defineEmits<{ change: [] }>()
const vehicleStore = useVehicleStore()
const periodMode = defineModel<PeriodMode>('periodMode', { required: true })
const selectedMonth = defineModel<string>('selectedMonth', { required: true })
const customFrom = defineModel<string>('customFrom', { required: true })
const customTo = defineModel<string>('customTo', { required: true })
const searchQuery = defineModel<string>('searchQuery', { required: true })

let searchDebounceTimeout: any = null

function onSearchInput() {
  if (searchDebounceTimeout) clearTimeout(searchDebounceTimeout)
  searchDebounceTimeout = setTimeout(() => emit('change'), 300)
}

function clearSearch() {
  searchQuery.value = ''
  emit('change')
}

const formattedSelectedMonth = computed(() => formatMonthLabel(selectedMonth.value))
const isCurrentMonth = computed(() => selectedMonth.value === currentYearMonth())

function prevMonth() {
  selectedMonth.value = shiftMonth(selectedMonth.value, -1)
  emit('change')
}

function nextMonth() {
  selectedMonth.value = shiftMonth(selectedMonth.value, 1)
  emit('change')
}

function resetToCurrentMonth() {
  selectedMonth.value = currentYearMonth()
  emit('change')
}

function setPeriodMode(mode: PeriodMode) {
  periodMode.value = mode
  emit('change')
}

function onMonthChange() {
  emit('change')
}

const PRESETS: PeriodPreset[] = ['lastMonth', 'last30', 'last90', 'thisYear']
function applyPreset(preset: PeriodPreset) {
  const { from, to } = presetRange(preset)
  customFrom.value = from
  customTo.value = to
  emit('change')
}

function onCustomDateChange() {
  emit('change')
}

// Summary metrics of current page drives
const pageDistance = computed(() => props.drives.reduce((acc, d) => acc + (d.distance_km || 0), 0))
const pageEnergy = computed(() => props.drives.reduce((acc, d) => acc + (d.energy_consumed_kwh || 0), 0))
const pageCost = computed(() => sumDriveCosts(props.drives))
const summaryChips = computed(() => {
  const distance = { label: t('drives.drivesToolbar.pageDistance').replace(/:$/, ''), value: formatDistance(pageDistance.value), tone: 'sky' as const }
  if (props.mode === 'TRIPS') return [{ ...distance, label: t('drives.drivesToolbar.totalDistance').replace(/:$/, '') }]
  return [
    distance,
    { label: t('drives.drivesToolbar.pageEnergy').replace(/:$/, ''), value: t('drives.drivesToolbar.kwh', { pageEnergy: Math.round(pageEnergy.value).toLocaleString(intlLocale()) }), tone: 'amber' as const },
    { label: t('drives.drivesToolbar.pageCost').replace(/:$/, ''), value: formatAmount(pageCost.value, vehicleStore.currency), tone: 'emerald' as const },
  ]
})
</script>

<template>
  <div class="space-y-3">
    <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-3 bg-slate-900 border border-slate-800 p-3 rounded-2xl">
      <!-- Period Mode & Navigation (Mix A + C); behind the filters button on a phone while it is closed -->
      <div class="flex-wrap items-center gap-2" :class="collapsed ? 'hidden sm:flex' : 'flex'">
        <!-- Period mode selector -->
        <div class="flex items-center bg-slate-950 p-1 rounded-xl border border-slate-800/80">
          <button
            @click="setPeriodMode('ALL')"
            class="px-2.5 py-1 rounded-lg text-xs font-semibold transition-colors"
            :class="periodMode === 'ALL' ? 'bg-slate-700 text-white border border-slate-600' : 'text-slate-400 hover:text-white border border-transparent'"
          >
            {{ $t('drives.drivesToolbar.all') }}
          </button>
          <button
            @click="setPeriodMode('MONTH')"
            class="px-2.5 py-1 rounded-lg text-xs font-semibold transition-colors flex items-center gap-1"
            :class="periodMode === 'MONTH' ? 'bg-slate-700 text-white border border-slate-600' : 'text-slate-400 hover:text-white border border-transparent'"
          >
            <Calendar class="w-3.5 h-3.5" />
            {{ $t('drives.drivesToolbar.byMonth') }}
          </button>
          <button
            @click="setPeriodMode('CUSTOM')"
            class="px-2.5 py-1 rounded-lg text-xs font-semibold transition-colors"
            :class="periodMode === 'CUSTOM' ? 'bg-slate-700 text-white border border-slate-600' : 'text-slate-400 hover:text-white border border-transparent'"
          >
            {{ $t('drives.drivesToolbar.period') }}
          </button>
        </div>

        <!-- Month Selector with Prev/Next buttons (when periodMode === 'MONTH') -->
        <div v-if="periodMode === 'MONTH'" class="flex items-center gap-1.5 bg-slate-950 px-2 py-1 rounded-xl border border-slate-800/80">
          <button
            @click="prevMonth"
            class="tap p-1 text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-colors"
            :title="$t('drives.drivesToolbar.previousMonth')" :aria-label="$t('drives.drivesToolbar.previousMonth')"
          >
            <ChevronLeft class="w-4 h-4" />
          </button>
          <label for="drives-month-select" class="sr-only">{{ $t('drives.drivesToolbar.selectTheMonth') }}</label>
          <div class="w-40 sm:w-44">
            <AppDatePicker
              id="drives-month-select"
              v-model="selectedMonth"
              month-picker
              size="xs"
              :clearable="false"
              @change="onMonthChange"
            />
          </div>
          <button
            @click="nextMonth"
            class="tap p-1 text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-colors"
            :title="$t('drives.drivesToolbar.nextMonth')" :aria-label="$t('drives.drivesToolbar.nextMonth')"
          >
            <ChevronRight class="w-4 h-4" />
          </button>
          <button
            v-if="!isCurrentMonth"
            @click="resetToCurrentMonth"
            class="text-xs text-rose-400 hover:text-rose-300 font-medium ml-1 px-1.5 py-0.5 bg-rose-500/10 rounded-md border border-rose-500/20"
            :title="$t('drives.drivesToolbar.backToTheCurrentMonth')"
          >
            {{ $t('drives.drivesToolbar.thisMonth') }}
          </button>
        </div>

        <!-- Custom Date Range (when periodMode === 'CUSTOM') -->
        <div v-if="periodMode === 'CUSTOM'" class="flex items-center gap-2 bg-slate-950 px-2.5 py-1 rounded-xl border border-slate-800/80 text-xs">
          <label for="drives-filter-from" class="text-slate-400 shrink-0">{{ $t('drives.drivesToolbar.from') }}</label>
          <div class="w-36">
            <AppDatePicker
              id="drives-filter-from"
              v-model="customFrom"
              size="xs"
              @change="onCustomDateChange"
            />
          </div>
          <label for="drives-filter-to" class="text-slate-400 shrink-0">{{ $t('drives.drivesToolbar.to') }}</label>
          <div class="w-36">
            <AppDatePicker
              id="drives-filter-to"
              v-model="customTo"
              size="xs"
              @change="onCustomDateChange"
            />
          </div>
          <div class="flex items-center gap-1 border-l border-slate-800 pl-2">
            <button
              v-for="p in PRESETS"
              :key="p"
              type="button"
              class="rounded-lg border border-transparent px-2 py-1 font-semibold text-slate-400 transition-colors hover:border-slate-700 hover:text-white"
              @click="applyPreset(p)"
            >
              {{ $t(`drives.drivesToolbar.presets.${p}`) }}
            </button>
          </div>
        </div>
      </div>

      <!-- Search Bar -->
      <div class="relative min-w-[240px] max-w-sm flex-1">
        <label for="drives-search-input" class="sr-only">{{ $t('drives.drivesToolbar.searchForACityOr') }}</label>
        <Search class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
        <input
          id="drives-search-input"
          type="text"
          v-model="searchQuery"
          @input="onSearchInput"
          :placeholder="$t('drives.drivesToolbar.searchForACityAddress')"
          class="w-full bg-slate-950 border border-slate-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-rose-500 transition-colors"
        />
        <button
          v-if="searchQuery"
          @click="clearSearch"
          class="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white p-0.5"
          :title="$t('drives.drivesToolbar.clearTheSearch')" :aria-label="$t('drives.drivesToolbar.clearTheSearch')"
        >
          <X class="w-3.5 h-3.5" />
        </button>
      </div>
    </div>

    <!-- The figures of what is listed: shown once, as chips -->
    <ListSummary v-if="total > 0 && !loading" :chips="summaryChips">
      <strong class="text-white">{{ total }}</strong>
      {{ mode === 'TRIPS' ? $t('drives.drivesToolbar.tripSFound') : $t('drives.drivesToolbar.driveSFound') }}
      <i18n-t v-if="periodMode === 'MONTH'" keypath="drives.drivesToolbar.inMonth" tag="span">
        <template #month><span class="font-semibold text-slate-200">{{ formattedSelectedMonth }}</span></template>
      </i18n-t>
    </ListSummary>
  </div>
</template>
