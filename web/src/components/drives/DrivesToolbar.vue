<script setup lang="ts">
import { intlLocale, t } from '@/i18n'
import { formatDistance } from '@/units'
import { computed, ref } from 'vue'
import { X, Search, SlidersHorizontal, Calendar } from 'lucide-vue-next'
import ModalShell from '@/components/ModalShell.vue'
import ListSummary from '@/components/ListSummary.vue'
import DrivesPeriodControls from '@/components/drives/DrivesPeriodControls.vue'
import { useIsMobile } from '@/composables/useIsMobile'
import { formatMonthLabel, periodBounds, type PeriodMode } from '@/utils/drives'

// One bar for the list filters: the period, the address search, and the quick filters of the page in the `filters`
// slot. On a phone the bar is a summary of the period and a Filters button; the period and the quick filters are
// edited in a sheet. Also shows what is listed: the count and, in trip mode, the distance.
// Every change of a filter is reported with change so the page reloads from the first page.
const props = withDefaults(defineProps<{ total: number; loading: boolean; drives: any[]; mode?: 'DRIVES' | 'TRIPS'; activeCount?: number }>(), { mode: 'DRIVES', activeCount: 0 })
const emit = defineEmits<{ change: [] }>()
const periodMode = defineModel<PeriodMode>('periodMode', { required: true })
const selectedMonth = defineModel<string>('selectedMonth', { required: true })
const anchorDay = defineModel<string>('anchorDay', { required: true })
const customFrom = defineModel<string>('customFrom', { required: true })
const customTo = defineModel<string>('customTo', { required: true })
const searchQuery = defineModel<string>('searchQuery', { required: true })

const isMobile = useIsMobile()
const sheetOpen = ref(false)

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

// What the period is, in a few words, for the phone's summary button
const periodSummary = computed(() => {
  const b = periodBounds({ mode: periodMode.value, anchor: anchorDay.value, month: selectedMonth.value, from: customFrom.value, to: customTo.value })
  if (!b.from && !b.to) return t('drives.drivesToolbar.units.ALL')
  const fmt = (iso?: string) => (iso ? new Date(`${iso}T00:00:00`).toLocaleDateString(intlLocale(), { day: 'numeric', month: 'short', year: 'numeric' }) : '…')
  if (b.from === b.to) return fmt(b.from)
  if (periodMode.value === 'MONTH') return formattedSelectedMonth.value
  if (periodMode.value === 'YEAR') return (b.from ?? '').slice(0, 4)
  return `${fmt(b.from)} – ${fmt(b.to)}`
})

// The drives list is paged and the API returns the count of the filter, not its totals: the distance, energy and
// cost of a page would read as the filter's, so only trips (not paged) carry a total. A selection has its own summary.
const tripsDistance = computed(() => props.drives.reduce((acc, d) => acc + (d.distance_km || 0), 0))
const summaryChips = computed(() =>
  props.mode === 'TRIPS' ? [{ label: t('drives.drivesToolbar.totalDistance').replace(/:$/, ''), value: formatDistance(tripsDistance.value), tone: 'sky' as const }] : [],
)
</script>

<template>
  <div class="space-y-3">
    <!-- Phone: the period and the filters are in a sheet -->
    <div v-if="isMobile" class="space-y-2 rounded-2xl border border-slate-800 bg-slate-900 p-3">
      <div class="flex items-center gap-2">
        <button type="button" class="btn btn-secondary min-w-0 flex-1 justify-start" @click="sheetOpen = true">
          <Calendar class="h-3.5 w-3.5 shrink-0 text-slate-400" aria-hidden="true" />
          <span class="truncate">{{ periodSummary }}</span>
        </button>
        <button type="button" class="btn btn-secondary shrink-0" @click="sheetOpen = true">
          <SlidersHorizontal class="h-3.5 w-3.5" aria-hidden="true" />
          <span>{{ $t('drives.drivesView.filters') }}</span>
          <span v-if="activeCount > 0" class="rounded-full bg-rose-600 px-1.5 text-xs font-bold text-white">{{ activeCount }}</span>
        </button>
      </div>
      <div class="relative">
        <label for="drives-search-input-phone" class="sr-only">{{ $t('drives.drivesToolbar.searchForACityOr') }}</label>
        <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
        <input
          id="drives-search-input-phone"
          type="text"
          v-model="searchQuery"
          @input="onSearchInput"
          :placeholder="$t('drives.drivesToolbar.searchForACityAddress')"
          class="w-full rounded-xl border border-slate-800 bg-slate-950 py-1.5 pl-9 pr-8 text-xs text-white placeholder-slate-500 transition-colors focus:border-rose-500 focus:outline-none"
        />
        <button
          v-if="searchQuery"
          @click="clearSearch"
          class="absolute right-2.5 top-1/2 -translate-y-1/2 p-0.5 text-slate-400 hover:text-white"
          :title="$t('drives.drivesToolbar.clearTheSearch')" :aria-label="$t('drives.drivesToolbar.clearTheSearch')"
        >
          <X class="h-3.5 w-3.5" />
        </button>
      </div>
    </div>

    <ModalShell v-if="isMobile" v-model:open="sheetOpen" :title="$t('drives.drivesView.filters')" :icon="SlidersHorizontal" icon-class="text-slate-300" size="sm">
      <div class="space-y-5">
        <section class="space-y-2">
          <h4 class="text-xs font-semibold text-slate-400">{{ $t('drives.drivesToolbar.period') }}</h4>
          <DrivesPeriodControls
            v-model:period-mode="periodMode"
            v-model:selected-month="selectedMonth"
            v-model:anchor-day="anchorDay"
            v-model:custom-from="customFrom"
            v-model:custom-to="customTo"
            @change="emit('change')"
          />
        </section>
        <section v-if="$slots.filters" class="space-y-2">
          <h4 class="text-xs font-semibold text-slate-400">{{ $t('drives.drivesView.quickFilters') }}</h4>
          <div class="flex flex-wrap items-center gap-2"><slot name="filters" /></div>
        </section>
      </div>
      <template #footer>
        <button type="button" class="btn btn-lg btn-primary w-full" @click="sheetOpen = false">
          {{ $t('drives.drivesToolbar.showResults', { count: total }) }}
        </button>
      </template>
    </ModalShell>

    <!-- Larger screens: one bar -->
    <div v-else class="space-y-3 rounded-2xl border border-slate-800 bg-slate-900 p-3">
      <div class="flex flex-col justify-between gap-3 lg:flex-row lg:items-center">
        <DrivesPeriodControls
          v-model:period-mode="periodMode"
          v-model:selected-month="selectedMonth"
          v-model:anchor-day="anchorDay"
          v-model:custom-from="customFrom"
          v-model:custom-to="customTo"
          @change="emit('change')"
        />

        <!-- Search Bar -->
        <div class="relative min-w-[240px] max-w-sm flex-1">
          <label for="drives-search-input" class="sr-only">{{ $t('drives.drivesToolbar.searchForACityOr') }}</label>
          <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
          <input
            id="drives-search-input"
            type="text"
            v-model="searchQuery"
            @input="onSearchInput"
            :placeholder="$t('drives.drivesToolbar.searchForACityAddress')"
            class="w-full rounded-xl border border-slate-800 bg-slate-950 py-1.5 pl-9 pr-8 text-xs text-white placeholder-slate-500 transition-colors focus:border-rose-500 focus:outline-none"
          />
          <button
            v-if="searchQuery"
            @click="clearSearch"
            class="absolute right-2.5 top-1/2 -translate-y-1/2 p-0.5 text-slate-400 hover:text-white"
            :title="$t('drives.drivesToolbar.clearTheSearch')" :aria-label="$t('drives.drivesToolbar.clearTheSearch')"
          >
            <X class="h-3.5 w-3.5" />
          </button>
        </div>
      </div>

      <!-- Quick filters of the page, in the same bar -->
      <div v-if="$slots.filters" class="flex flex-wrap items-center gap-2 border-t border-slate-800 pt-3">
        <slot name="filters" />
      </div>
    </div>

    <!-- The figures of what is listed: shown once, as chips -->
    <ListSummary v-if="total > 0 && !loading" :chips="summaryChips">
      <strong class="text-white">{{ total }}</strong>
      {{ mode === 'TRIPS' ? $t('drives.drivesToolbar.tripSFound', {}, total) : $t('drives.drivesToolbar.driveSFound', {}, total) }}
      <i18n-t v-if="periodMode === 'MONTH'" keypath="drives.drivesToolbar.inMonth" tag="span">
        <template #month><span class="font-semibold text-slate-200">{{ formattedSelectedMonth }}</span></template>
      </i18n-t>
    </ListSummary>
  </div>
</template>
