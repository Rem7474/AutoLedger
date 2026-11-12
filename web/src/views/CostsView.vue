<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ChartColumn, List, Wallet } from '@/icons'
import PageHeader from '@/components/PageHeader.vue'
import TabBar, { type TabItem } from '@/components/TabBar.vue'
import PeriodBar from '@/components/period/PeriodBar.vue'
import CostEntryRow from '@/components/costs/CostEntryRow.vue'
import SwipeToDelete from '@/components/SwipeToDelete.vue'
import TollModal from '@/components/expenses/TollModal.vue'
import MaintenanceModal from '@/components/expenses/MaintenanceModal.vue'
import ChargeModal from '@/components/expenses/ChargeModal.vue'
import FuelFillUpModal from '@/components/manual/FuelFillUpModal.vue'
import MonthlyCostChart from '@/components/dashboard/MonthlyCostChart.vue'
import CostBreakdownDonut from '@/components/dashboard/CostBreakdownDonut.vue'
import TagBreakdown from '@/components/dashboard/TagBreakdown.vue'
import { usePreferencesStore } from '@/stores/preferences'
import MileageCostChart from '@/components/dashboard/MileageCostChart.vue'
import MonthDetailModal from '@/components/dashboard/MonthDetailModal.vue'
import ChargingInsights from '@/components/expenses/ChargingInsights.vue'
import ChargingPlacesModal from '@/components/expenses/ChargingPlacesModal.vue'
import DocumentPreviewModal from '@/components/expenses/DocumentPreviewModal.vue'
import EmptyState from '@/components/EmptyState.vue'
import InfiniteSentinel from '@/components/InfiniteSentinel.vue'
import ListSkeleton from '@/components/ListSkeleton.vue'
import LoadError from '@/components/LoadError.vue'
import { useVehicleStore } from '@/stores/vehicle'
import { api, type CostEntriesResponse } from '@/services/api'
import { usePeriod } from '@/composables/usePeriod'
import { useInfiniteList } from '@/composables/useInfiniteList'
import { useRefreshHandler } from '@/composables/usePullToRefresh'
import { useCostEntryActions } from '@/composables/useCostEntryActions'
import { useDocumentPreview } from '@/composables/useDocumentPreview'
import { COST_FILTERS, amountValue, categoriesOfFilter, costPerKm, entryKey, entryTarget, isCostFilterKey, type CostFilterKey } from '@/utils/costEntries'
import { costsTabQuery, parseCostsTab, type CostsTab } from '@/utils/costsTab'
import { dayLabel, groupByDay, groupDaysByMonth, monthHeader } from '@/utils/dayGroups'
import { toLocalDay } from '@/utils/dates'
import { THEME_STYLES } from '@/utils/themes'
import { formatAmount } from '@/currency'
import { distanceUnit, perDistance } from '@/units'
import { intlLocale, t } from '@/i18n'

// The journal of what the active vehicle cost over the period, under the figures of that period. The figures are
// those of the lines the filters keep, so a filter changes both.
const PAGE_SIZE = 50
const vehicleStore = useVehicleStore()
const route = useRoute()
const router = useRouter()
const { period, bounds, label: periodLabel, rounded, canPrevious, canNext, set: setPeriod, step: stepPeriod } = usePeriod()

const filter = ref<CostFilterKey>(isCostFilterKey(route.query.theme) ? route.query.theme : 'all')
const currency = computed(() => vehicleStore.currency)

// The totals come with the first page; a read that was overtaken by a newer filter or period must not set them
const summary = ref<CostEntriesResponse | null>(null)
const readKey = () => `${vehicleStore.activeVehicle?.id}|${bounds.value.from}|${bounds.value.to}|${filter.value}`

async function readPage(page: number) {
  const vehicle = vehicleStore.activeVehicle
  if (!vehicle) return { items: [], total: 0 }
  const key = readKey()
  const res = await api.getCostEntries(vehicle.id, {
    from: bounds.value.from,
    to: bounds.value.to,
    categories: categoriesOfFilter(filter.value),
    limit: PAGE_SIZE,
    offset: (page - 1) * PAGE_SIZE,
  })
  if (page === 1 && key === readKey()) summary.value = res
  return { items: res.entries.map((e) => ({ ...e, id: entryKey(e) })), total: res.count }
}
const { items, total, loading, loadingMore, error, moreError, hasMore, load, loadMore } = useInfiniteList(readPage)

// The chip of the filter in use is brought into view: the row scrolls sideways and the filter may come from the address
const chips = ref<HTMLElement | null>(null)
async function showActiveChip() {
  await nextTick()
  chips.value?.querySelector<HTMLElement>('[aria-pressed="true"]')?.scrollIntoView?.({ block: 'nearest', inline: 'center' })
}
onMounted(() => {
  void load()
  void showActiveChip()
})
watch([() => vehicleStore.activeVehicle?.id, () => bounds.value.from, () => bounds.value.to, filter], () => void load())
watch(filter, (key) => {
  void showActiveChip()
  void router.replace({ query: { ...route.query, theme: key === 'all' ? undefined : key } })
})
useRefreshHandler(() => load(true))

// A line opens its record, and a swipe deletes it with a way back: the line leaves the page at once and returns if the
// deletion is taken back or refused
const removed = new Map<string, { row: (typeof items.value)[number]; index: number }>()
const actions = useCostEntryActions({
  hide: (id) => {
    const index = items.value.findIndex((e) => e.id === id)
    if (index < 0) return
    removed.set(id, { row: items.value[index], index })
    items.value = items.value.filter((e) => e.id !== id)
  },
  restore: (id) => {
    const gone = removed.get(id)
    removed.delete(id)
    if (!gone || items.value.some((e) => e.id === id)) return
    const next = [...items.value]
    next.splice(Math.min(gone.index, next.length), 0, gone.row)
    items.value = next
  },
  reload: () => load(true),
})
const { showToll, showMaintenance, showCharge, showFuel, removeOpenedFuel, removeOpenedCharge, editing, documents, maintenanceExpenses, onDocumentAdded } = actions
// The quick entry sheet sends a maintenance entry here: the form opens on arrival and the parameter is dropped
watch(
  () => [route.query.add, vehicleStore.canEdit] as const,
  async ([add, canEdit]) => {
    if (add !== 'maintenance') return
    const { add: _drop, ...rest } = route.query
    // The address is clean before the form takes its history entry, so that Back does not come back to `add`
    await router.replace({ query: rest })
    if (canEdit) await nextTick(actions.addMaintenance)
  },
  { immediate: true },
)
const { previewDoc, closeDocPreview, viewOrDownloadDocument } = useDocumentPreview(() => vehicleStore.activeVehicle?.id)
// The charts of the whole cost, month by month: they come from the cost engine, like the figures of the home page, and
// follow the period; a bar opens the figures of its month
const prefs = usePreferencesStore()
const tco = ref<any | null>(null)
const monthlyCosts = computed(() => (tco.value ? tco.value.monthly_costs || [] : null))
const selectedMonth = ref<any | null>(null)
async function loadTco() {
  const vehicle = vehicleStore.activeVehicle
  if (!vehicle) return
  try {
    const res = await api.getTCO(vehicle.id)
    if (vehicle.id === vehicleStore.activeVehicle?.id) tco.value = res
  } catch (err) {
    console.error('Failed to load the monthly costs', err)
  }
}
onMounted(() => void loadTco())
watch([() => vehicleStore.activeVehicle?.id, () => vehicleStore.lastSyncTimestamp], () => void loadTco())

// The energy theme adds the figures by month and by place; managing the places changes both them and the lines
const showPlaces = ref(false)
const placesVersion = ref(0)
const insightsKey = computed(() => `${vehicleStore.lastSyncTimestamp}|${summary.value?.count ?? 0}|${placesVersion.value}`)
const onPlacesChanged = () => {
  placesVersion.value++
  void load(true)
}
const currentOdometer = computed(() => vehicleStore.activeVehicle?.current_odometer || 0)
const swipeable = (e: { source: string; source_id: string }) => vehicleStore.canEdit && entryTarget(e).type === 'edit'

// The lines under their month, each month under its days. A month's subtotal is shown once the month is complete: the
// newest month still lists only what is loaded while there are more pages, and the oldest month is complete once nothing
// older follows.
const months = computed(() => {
  const today = toLocalDay()
  const labels = { today: t('costs.list.today'), yesterday: t('costs.list.yesterday') }
  const format = (date: Date, withYear: boolean) =>
    date.toLocaleDateString(intlLocale(), { weekday: 'long', day: 'numeric', month: 'long', ...(withYear ? { year: 'numeric' } : {}) })
  const monthFormat = (date: Date, withYear: boolean) => date.toLocaleDateString(intlLocale(), { month: 'long', ...(withYear ? { year: 'numeric' } : {}) })
  const days = groupByDay(items.value, (e) => e.date).map((g) => ({ ...g, label: dayLabel(g.day, today, labels, format) }))
  const grouped = groupDaysByMonth(days)
  return grouped.map((m, i) => {
    const complete = !(hasMore.value && i === grouped.length - 1)
    const subtotal = m.days.reduce((sum, d) => sum + d.items.reduce((s, e) => s + amountValue(e.amount), 0), 0)
    return { ...m, label: monthHeader(m.month, today, monthFormat), subtotal: complete ? subtotal : null }
  })
})

// The headline is the net of what the passengers paid when the whole cost is listed; under a filter the lines are
// not the whole cost, so it is what was paid for that theme
const paid = computed(() => amountValue(summary.value?.paid_total))
const received = computed(() => amountValue(summary.value?.carpool_received))
const headline = computed(() => (filter.value === 'all' ? amountValue(summary.value?.net_total) : paid.value))
const perKm = computed(() => costPerKm(headline.value, summary.value?.distance_km ?? 0))
const headlineLabel = computed(() => (filter.value === 'all' ? t('costs.headline.net') : t('costs.headline.paidTheme', { theme: t(`costs.filters.${filter.value}`) })))
const showBreakdown = computed(() => filter.value === 'all' && received.value > 0)
// Journal first, the charts behind a second tab kept in the address (`?tab=ANALYSIS`) so that a reload and Back find it.
// Only the whole cost and the energy theme have an analysis: under another theme the page is the journal alone.
const requestedTab = ref<CostsTab>(parseCostsTab(route.query.tab))
const hasAnalysis = computed(() => filter.value === 'all' || (filter.value === 'energy' && vehicleStore.capabilities.charges))
const tab = computed<CostsTab>(() => (hasAnalysis.value ? requestedTab.value : 'JOURNAL'))
const tabs = computed<TabItem[]>(() => [
  { key: 'JOURNAL', label: t('costs.tabs.journal'), icon: List },
  { key: 'ANALYSIS', label: t('costs.tabs.analysis'), icon: ChartColumn },
])
function switchTab(next: CostsTab) {
  requestedTab.value = next
  void router.replace({ query: costsTabQuery(route.query, next) })
}
// The address changed under the page (Back, a link): the tab follows it
watch(
  () => route.query.tab,
  (value) => {
    requestedTab.value = parseCostsTab(value)
  },
)
const filterTheme = (key: CostFilterKey) => COST_FILTERS.find((f) => f.key === key)?.theme ?? null
const chipClass = (key: CostFilterKey) => {
  if (filter.value === key) return 'border-rose-500 bg-rose-600 text-white'
  return 'border-slate-700 bg-slate-800 text-slate-200'
}
</script>

<template>
  <div v-if="vehicleStore.activeVehicle" class="space-y-4">
    <PageHeader :title="$t('costs.title')" :icon="Wallet">{{ $t('costs.subtitle') }}</PageHeader>

    <div class="md:sticky md:top-[4.25rem] md:z-30 md:bg-slate-950/90 md:py-2 md:backdrop-blur-md">
      <PeriodBar :period="period" :label="periodLabel" :rounded="rounded" :can-previous="canPrevious" :can-next="canNext" @step="stepPeriod" @pick="setPeriod" />
    </div>

    <!-- The figures of the lines kept by the filter -->
    <div class="card space-y-1 px-4 py-4" data-testid="costs-headline">
      <p class="text-xs font-semibold text-slate-400">{{ headlineLabel }}</p>
      <div class="flex items-baseline justify-between gap-3">
        <p class="text-3xl font-extrabold tabular-nums text-hero">{{ formatAmount(headline, currency) }}</p>
        <p v-if="perKm !== null" class="shrink-0 text-sm tabular-nums text-slate-300">
          {{ $t('costs.headline.perKm', { value: formatAmount(perDistance(perKm), currency, 3), unit: distanceUnit() }) }}
        </p>
      </div>
      <p v-if="showBreakdown" class="text-xs tabular-nums text-slate-400">
        {{ $t('costs.headline.paidLine', { amount: formatAmount(paid, currency) }) }} · {{ $t('costs.headline.receivedLine', { amount: formatAmount(received, currency) }) }}
      </p>
    </div>

    <!-- The theme chips and, under them, the tabs of the page: one block, the journal starts right under it -->
    <div class="space-y-1">
    <div ref="chips" class="-mx-4 overflow-x-auto px-4 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden" role="group" :aria-label="$t('costs.filtersLabel')">
      <div class="flex w-max gap-2">
        <button
          v-for="f in COST_FILTERS"
          :key="f.key"
          type="button"
          class="tap-text inline-flex items-center gap-1.5 rounded-full border px-3 text-sm font-semibold"
          :class="chipClass(f.key)"
          :aria-pressed="filter === f.key"
          :data-testid="`costs-filter-${f.key}`"
          @click="filter = f.key"
        >
          <component :is="f.icon" v-if="f.icon" class="size-4" :class="filter === f.key ? 'text-white' : THEME_STYLES[filterTheme(f.key) ?? 'other'].text" aria-hidden="true" />
          {{ $t(`costs.filters.${f.key}`) }}
        </button>
      </div>
    </div>

    <TabBar v-if="hasAnalysis" :model-value="tab" :tabs="tabs" :label="$t('costs.tabsLabel')" id-prefix="costs-tab" @update:model-value="switchTab($event as CostsTab)" />
    </div>

    <!-- Analysis: the month-by-month charts for the whole cost, the places for the energy theme -->
    <template v-if="tab === 'ANALYSIS'">
      <template v-if="filter === 'all'">
        <ListSkeleton v-if="monthlyCosts === null" :rows="3" />
        <p v-else-if="monthlyCosts.length === 0" class="px-1 py-6 text-center text-sm text-slate-400">{{ $t('costs.analysisEmpty') }}</p>
        <template v-else>
          <MonthlyCostChart :monthly-costs="monthlyCosts" @open-month="selectedMonth = $event" />
          <CostBreakdownDonut :tco="tco" />
          <MileageCostChart :monthly-costs="monthlyCosts" @open-month="selectedMonth = $event" />
          <!-- Shows itself only once a drive carries a tag -->
          <TagBreakdown v-if="prefs.proPersoEnabled" :tco="tco" />
        </template>
      </template>
      <ChargingInsights
        v-else
        :vehicle-id="vehicleStore.activeVehicle.id"
        :currency="currency"
        :has-charges="(summary?.count ?? 0) > 0"
        :sync-key="insightsKey"
        :with-period="false"
        @manage-places="showPlaces = true"
      />
    </template>

    <template v-else>
      <ListSkeleton v-if="loading" :rows="6" />
      <LoadError v-else-if="error !== null" :message="error" @retry="load()" />
      <EmptyState v-else-if="!items.length" :icon="Wallet" :title="$t('costs.empty.title')">{{ $t('costs.empty.body') }}</EmptyState>
      <div v-else class="space-y-3">
        <p v-if="hasMore" class="px-1 text-xs text-slate-400">{{ $t('costs.list.shownOf', { count: items.length, total }) }}</p>
        <section v-for="month in months" :key="month.month" class="space-y-2">
          <div class="flex items-baseline justify-between gap-3 px-1 pt-2">
            <h2 class="text-sm font-bold capitalize text-white">{{ month.label }}</h2>
            <p v-if="month.subtotal !== null" class="text-xs font-semibold tabular-nums text-slate-300">{{ formatAmount(month.subtotal, currency) }}</p>
          </div>
          <div v-for="group in month.days" :key="group.day" class="space-y-2">
            <h3 class="px-1 pt-1 text-xs font-semibold text-slate-400">{{ group.label }}</h3>
            <SwipeToDelete v-for="e in group.items" :key="e.id" :disabled="!swipeable(e)" @delete="actions.remove(e, e.id)">
              <CostEntryRow :entry="e" :currency="currency" :readonly="!vehicleStore.canEdit" @open="actions.open" />
            </SwipeToDelete>
          </div>
        </section>
        <InfiniteSentinel v-if="hasMore" :loading="loadingMore" :error="moreError" :label="$t('costs.list.loadMore')" @more="loadMore" />
        <p v-if="filter === 'all'" class="px-1 text-xs text-slate-500">{{ $t('costs.estimateNote') }}</p>
      </div>
    </template>

    <TollModal v-model:open="showToll" :vehicle-id="vehicleStore.activeVehicle.id" :editing="editing" :documents="documents" @saved="load(true)" @document-added="onDocumentAdded" @view-document="viewOrDownloadDocument" />
    <MaintenanceModal
      v-model:open="showMaintenance"
      :vehicle-id="vehicleStore.activeVehicle.id"
      :editing="editing"
      :documents="documents"
      :maintenance-expenses="maintenanceExpenses"
      :current-odometer="currentOdometer"
      default-category="MAINTENANCE"
      @saved="load(true)"
      @document-added="onDocumentAdded"
      @view-document="viewOrDownloadDocument"
    />
    <ChargeModal v-model:open="showCharge" :vehicle-id="vehicleStore.activeVehicle.id" :editing="editing" :documents="documents" :current-odometer="currentOdometer" @saved="load(true)" @delete="removeOpenedCharge" @document-added="onDocumentAdded" @view-document="viewOrDownloadDocument" />
    <FuelFillUpModal v-model:open="showFuel" :vehicle-id="vehicleStore.activeVehicle.id" :editing="editing" :can-edit="vehicleStore.canEdit" @saved="load(true)" @delete="removeOpenedFuel" />
    <MonthDetailModal v-model:month="selectedMonth" :monthly-costs="monthlyCosts || []" />
    <ChargingPlacesModal v-model:open="showPlaces" :vehicle-id="vehicleStore.activeVehicle.id" :can-edit="vehicleStore.canEdit" @changed="onPlacesChanged" />
    <DocumentPreviewModal :preview-doc="previewDoc" @close="closeDocPreview" />
  </div>
</template>
