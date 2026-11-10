<script setup lang="ts">
import { Navigation as PageIcon, Layers, List, RotateCcw, Plus, UploadCloud, Users } from '@/icons'
import LoadError from '@/components/LoadError.vue'
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'
import TabBar, { type TabItem } from '@/components/TabBar.vue'
import { t, intlLocale } from '@/i18n'
import { distanceUnit } from '@/units'
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { drivesTabQuery, parseDrivesTab, type DrivesTab } from '@/utils/drivesTab'
import { useVehicleStore } from '@/stores/vehicle'
import { usePreferencesStore } from '@/stores/preferences'
import { useConfirm } from '@/composables/useConfirm'
import { useDriveSelection } from '@/composables/useDriveSelection'
import { useCarpoolEditor } from '@/composables/useCarpoolEditor'
import { useDriveBulkActions } from '@/composables/useDriveBulkActions'
import { useTripGroups } from '@/composables/useTripGroups'
import { useDriveCostModal } from '@/composables/useDriveCostModal'
import { api } from '@/services/api'
import SelectAllToggle from '@/components/SelectAllToggle.vue'
import DrivesToolbar from '@/components/drives/DrivesToolbar.vue'
import DriveBulkActions from '@/components/drives/DriveBulkActions.vue'
import DriveCard from '@/components/drives/DriveCard.vue'
import InfiniteSentinel from '@/components/InfiniteSentinel.vue'
import AddressBackfillNotice from '@/components/drives/AddressBackfillNotice.vue'
import CarpoolView from '@/views/CarpoolView.vue'
import TripGroupsPanel from '@/components/drives/TripGroupsPanel.vue'
import TripSuggestions from '@/components/drives/TripSuggestions.vue'
import ToQualifyFilter from '@/components/drives/ToQualifyFilter.vue'
import DriveFilterGroups from '@/components/drives/DriveFilterGroups.vue'
import DriveListSkeleton from '@/components/drives/DriveListSkeleton.vue'
import CarpoolTripModal from '@/components/carpool/CarpoolTripModal.vue'
import DriveCostModal from '@/components/drives/DriveCostModal.vue'
import DriveGroupModal from '@/components/drives/DriveGroupModal.vue'
import TripEditModal from '@/components/drives/TripEditModal.vue'
import AddToTripModal from '@/components/drives/AddToTripModal.vue'
import ManualDriveModal from '@/components/drives/ManualDriveModal.vue'
import CSVImportModal from '@/components/CSVImportModal.vue'
import EmptySourceHints from '@/components/EmptySourceHints.vue'
import { useRefreshHandler } from '@/composables/usePullToRefresh'
import { useInfiniteList } from '@/composables/useInfiniteList'
import { groupByDay, dayLabel, groupDaysByMonth, monthHeader } from '@/utils/dayGroups'
import { formatAmount } from '@/currency'
import { toLocalDay } from '@/utils/dates'
import { deleteRowsWithUndo, deleteWithUndo } from '@/composables/useUndoableDelete'
import SwipeToDelete from '@/components/SwipeToDelete.vue'
import {
  filterTrips,
  currentYearMonth,
  countActiveDriveFilters,
  periodBounds,
  selectionSummary,
  todayIso,
  type PeriodMode,
  toggleTag,
} from '@/utils/drives'

// The page owns the drives list, the filters, the selection and which modal is open; the toolbar, the cards,
// the pagination and the modals are components that report back to it.
const vehicleStore = useVehicleStore()
const route = useRoute()
const router = useRouter()
const prefs = usePreferencesStore()
const { showConfirm, showAlert } = useConfirm()
const vehicleId = computed(() => vehicleStore.activeVehicle?.id ?? '')
// The list grows as it is scrolled: older drives are found with the filters, not by paging
const PAGE_SIZE = 30
const selectedTag = ref('')
const unqualifiedOnly = ref(false)
const hasTollOnly = ref(false)
const tollSource = ref('')
const carpoolFilter = ref<'' | 'yes' | 'no'>('')
const unqualifiedCount = ref(0)
const {
  items: drives,
  total,
  loading,
  error: loadError,
  loadingMore,
  moreError,
  hasMore,
  load: loadList,
  loadMore: loadMoreDrives,
} = useInfiniteList<any>(readDrivesPage)
loading.value = true

// Manual drive and CSV import modals
const showManualDriveModal = ref(false)
const driveToEdit = ref<any | null>(null)
const showCSVImportModal = ref(false)

function openManualDriveModal(drive?: any) {
  driveToEdit.value = drive || null
  showManualDriveModal.value = true
}

function openCSVImportModal() {
  showCSVImportModal.value = true
}

function onDriveSaved() {
  void loadDrives(true)
}

function onCSVImported() {
  void loadDrives(true)
}

// A single drive goes at once and comes back with Undo, the server is asked when that time is over
function handleDeleteManualDrive(drive: any) {
  const vehicle = vehicleStore.activeVehicle
  if (!vehicle || !drive.is_manual) return
  const index = drives.value.findIndex((d) => d.id === drive.id)
  deleteWithUndo({
    message: t('drives.drivesView.driveDeleted'),
    hide: () => {
      drives.value = drives.value.filter((d) => d.id !== drive.id)
      total.value = Math.max(0, total.value - 1)
    },
    restore: () => {
      if (drives.value.some((d) => d.id === drive.id)) return
      const next = [...drives.value]
      next.splice(Math.min(Math.max(index, 0), next.length), 0, drive)
      drives.value = next
      total.value += 1
    },
    commit: () => api.deleteDrive(vehicle.id, drive.id),
    committed: () => void loadDrives(true),
    onError: (err: any) => void showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger'),
  })
}

// The delete of the contextual bar: the drives entered by hand of the selection go at once under one Undo
function handleDeleteManualDrives(list: any[]) {
  const vehicle = vehicleStore.activeVehicle
  const manual = list.filter((d) => d.is_manual)
  if (!vehicle || manual.length === 0) return
  if (manual.length === 1) return handleDeleteManualDrive(manual[0])
  clearSelection()
  deleteRowsWithUndo({
    list: drives,
    items: manual,
    key: (d) => d.id,
    message: t('drives.drivesView.drivesDeleted', { count: manual.length }),
    remove: (d) => api.deleteDrive(vehicle.id, d.id),
    onHide: (n) => (total.value = Math.max(0, total.value - n)),
    onRestore: (n) => (total.value += n),
    committed: () => void loadDrives(true),
    onError: (err: any) => void showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger'),
  })
}

// Period / month and address filters (edited in the toolbar)
const periodMode = ref<PeriodMode>('ALL')
const selectedMonth = ref(currentYearMonth())
// The day the DAY, WEEK and YEAR modes step from
const anchorDay = ref(todayIso())
const customFrom = ref('')
const customTo = ref('')
const searchQuery = ref('')

// The trips and the suggestions are narrowed client-side by the same period and search as the drives
const tripFilter = computed(() => {
  return { ...periodBounds({ mode: periodMode.value, anchor: anchorDay.value, month: selectedMonth.value, from: customFrom.value, to: customTo.value }), q: searchQuery.value }
})
const filteredTrips = computed(() => filterTrips(tripGroups.value, tripFilter.value))
const filteredSuggestions = computed(() => filterTrips(tripSuggestions.value, tripFilter.value))
const shownTrips = computed(() => (tripQualifyOnly.value ? filteredSuggestions.value : filteredTrips.value))
const hasFilters = computed(() => periodMode.value !== 'ALL' || !!searchQuery.value.trim())

function onFiltersChange() {
  if (viewMode.value === 'TRIPS') return
  void loadDrives()
}

function resetAllFilters() {
  searchQuery.value = ''
  selectedTag.value = ''
  unqualifiedOnly.value = false
  hasTollOnly.value = false
  tollSource.value = ''
  carpoolFilter.value = ''
  periodMode.value = 'ALL'
  customFrom.value = ''
  customTo.value = ''
  void loadDrives()
}

const {
  selectedDrives,
  selectedDriveIds,
  selectedList,
  selectedOffPage,
  allPageSelected,
  somePageSelected,
  clearSelection,
  toggleSelectDrive,
  selectAll,
} = useDriveSelection(drives)

// Unified selection summary metrics (same as a Voyage)
// The page figures above already are the selection's when the whole page is selected
const selectedSummaryMetrics = computed(() =>
  selectedOffPage.value === 0 && selectedDriveIds.value.length === drives.value.length ? '' : selectionSummary(selectedList.value, vehicleStore.currency)
)

useRefreshHandler(() => loadDrives(true))

// The drives under a header per month, each month under its days, the newest first as the server sends them. A month's
// subtotal is shown once the month is complete: the newest month lists only what is loaded while there are more pages.
const driveMonths = computed(() => {
  const today = toLocalDay()
  const labels = { today: t('drives.drivesView.today'), yesterday: t('drives.drivesView.yesterday') }
  const format = (date: Date, withYear: boolean) =>
    date.toLocaleDateString(intlLocale(), { weekday: 'long', day: 'numeric', month: 'long', ...(withYear ? { year: 'numeric' } : {}) })
  const monthFormat = (date: Date, withYear: boolean) => date.toLocaleDateString(intlLocale(), { month: 'long', ...(withYear ? { year: 'numeric' } : {}) })
  const days = groupByDay(drives.value, (d) => d.start_time).map((g) => ({ ...g, label: dayLabel(g.day, today, labels, format) }))
  const grouped = groupDaysByMonth(days)
  return grouped.map((m, i) => {
    const complete = !(hasMore.value && i === grouped.length - 1)
    const subtotal = m.days.reduce((sum, g) => sum + g.items.reduce((s, d) => s + (d.costs?.total_cost || 0), 0), 0)
    return { ...m, label: monthHeader(m.month, today, monthFormat), subtotal: complete ? subtotal : null }
  })
})

// One page of the list for the current filters
async function readDrivesPage(page: number) {
  if (!vehicleStore.activeVehicle) return { items: [], total: 0 }
  const { from: fromStr, to: toStr } = periodBounds({ mode: periodMode.value, anchor: anchorDay.value, month: selectedMonth.value, from: customFrom.value, to: customTo.value })
  const res = await api.getDrives(vehicleStore.activeVehicle.id, {
    tag: selectedTag.value,
    page,
    limit: PAGE_SIZE,
    unqualified: unqualifiedOnly.value,
    hasToll: hasTollOnly.value,
    tollSource: hasTollOnly.value ? tollSource.value : '',
    carpool: carpoolFilter.value,
    from: fromStr,
    to: toStr,
    q: searchQuery.value.trim() || undefined,
  })
  unqualifiedCount.value = res.unqualified_count || 0
  return { items: res.drives as any[], total: res.total as number }
}

// `silent` reads again every page that is shown, behind the list, instead of starting over from the first one
function loadDrives(silent = false) {
  return loadList(silent)
}

// View mode: drives list or trip groups ("voyages"), kept in the address (`?tab=TRIPS`) so a reload or Back finds it again
const viewMode = ref<DrivesTab>(parseDrivesTab(route.query.tab))
const tripQualifyOnly = ref(false)
// The toolbar and the filters belong to the lists of drives and trips; the carpools segment brings its own page
const listMode = computed<'DRIVES' | 'TRIPS'>(() => (viewMode.value === 'TRIPS' ? 'TRIPS' : 'DRIVES'))

// On a phone the filters sit behind one button, so the first drive is not pushed below the fold; a wide screen shows them
const activeFilterCount = computed(() => countActiveDriveFilters({ mode: listMode.value, unqualifiedOnly: unqualifiedOnly.value, hasTollOnly: hasTollOnly.value, selectedTag: selectedTag.value, tripQualifyOnly: tripQualifyOnly.value, carpoolFilter: carpoolFilter.value }))
const {
  tripGroups,
  tripSuggestions,
  suggestionBusyKey,
  loadingTrips,
  tripsLoadError,
  expandedTripId,
  tripDrives,
  loadTripGroups,
  dismissTripSuggestion,
  suggestionName,
  createTripFromSuggestion,
  toggleTripDetails,
  removeDriveFromTrip,
  handleDeleteTrip,
} = useTripGroups({ loadDrives })

// Modals
const showTripEditModal = ref(false)
const tripBeingEdited = ref<any | null>(null)
const showAddToTripModal = ref(false)
const showGroupModal = ref(false)
// The carpool editor opens on this page, so the filters and the selection stay
const { showCarpoolEditor, carpoolCreateOptions, carpoolOpenToken, recentCarpoolNames, openCarpoolEditor } = useCarpoolEditor()
function onCarpoolSaved() {
  clearSelection()
  void loadDrives(true)
}
const { bulkApplyingToll, vehiclePeople, handleBatchDriver, handleBatchTag, exportSelectedDrives, handleCarpoolSelectedDrives, handleBulkApplyToll } = useDriveBulkActions({
  drives,
  selectedDrives,
  selectedDriveIds,
  selectedList,
  clearSelection,
  loadDrives,
  openCarpoolEditor,
})
const {
  showCostModal,
  selectedCostDrive,
  costTripDriveIds,
  tripLegs,
  costStartWithToll,
  openSuggestionCostModal,
  createTripFromCostModal,
  dismissTripFromCostModal,
  openTripCostModal,
  openCostModal,
  refreshCostDrive,
} = useDriveCostModal({
  loadDrives,
  tripGroups,
  tripSuggestions,
  loadTripGroups,
  suggestionName,
  createTripFromSuggestion,
  dismissTripSuggestion,
})

watch(
  () => [vehicleStore.activeVehicle?.id, selectedTag.value, unqualifiedOnly.value, hasTollOnly.value, tollSource.value, carpoolFilter.value],
  () => {
    void loadDrives()
    if (viewMode.value === 'TRIPS') void loadTripGroups()
  }
)

// New data arrived (synchronization): reload in place, keeping the page, the filters and the selection
watch(
  () => vehicleStore.lastSyncTimestamp,
  () => {
    void loadDrives(true)
    if (viewMode.value === 'TRIPS') void loadTripGroups(true)
  }
)

watch(
  () => vehicleStore.activeVehicle?.id,
  () => clearSelection()
)

// The qualification queue and the toll source only exist with the toll detection: leave them when it goes away
watch(
  () => vehicleStore.tollDetectionActive,
  (active) => {
    if (active) return
    unqualifiedOnly.value = false
    tollSource.value = ''
  },
  { immediate: true }
)

// A work/personal filter left on would hide drives with no way to clear it once the classification is off
watch(
  () => prefs.proPersoEnabled,
  (enabled) => {
    if (!enabled) selectedTag.value = ''
  },
  { immediate: true }
)

onMounted(() => {
  void loadDrives()
  if (viewMode.value === 'TRIPS') void loadTripGroups()
})

async function markNoToll(d: any) {
  if (!vehicleStore.activeVehicle) return
  try {
    await api.setDriveTollReview(vehicleStore.activeVehicle.id, d.id, true)
    d.toll_reviewed_at = new Date().toISOString()
    d.needs_toll_qualification = false
    unqualifiedCount.value = Math.max(0, unqualifiedCount.value - 1)
    if (unqualifiedOnly.value) {
      drives.value = drives.value.filter((x) => x.id !== d.id)
      total.value = Math.max(0, total.value - 1)
    }
  } catch (err: any) {
    void showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger')
  }
}

function openTollEntry(d: any) {
  openCostModal(d, true)
}

const viewTabs = computed<TabItem[]>(() => [
  { key: 'DRIVES', label: t('drives.drivesView.drives'), icon: List },
  { key: 'TRIPS', label: t('drives.drivesView.trips'), icon: Layers },
  // Carpooling applies to any vehicle that has drives
  ...(vehicleStore.canCharge ? [{ key: 'CARPOOLS', label: t('drives.drivesView.carpools'), icon: Users }] : []),
])

function switchView(mode: DrivesTab) {
  viewMode.value = mode
  if (mode === 'TRIPS') void loadTripGroups()
  void router.replace({ query: drivesTabQuery(route.query, mode) })
}

// The address changed under the page (Back, a link): the tab follows it
watch(
  () => route.query.tab,
  (tab) => {
    const mode = parseDrivesTab(tab)
    if (mode !== viewMode.value) switchView(mode)
  }
)

// The edit form replaces the detail, like the edit of a carpool
function editTripFromCostModal(virtual: any) {
  const tg = tripGroups.value.find((g) => g.id === virtual.id)
  if (!tg) return
  showCostModal.value = false
  openTripEdit(tg)
}

// Its legs may have changed: the trips, the drives (their trip badge) and the open leg list are reloaded
async function onTripSaved() {
  expandedTripId.value = null
  await loadTripGroups()
  void loadDrives(true)
}

function openTripEdit(tg: any) {
  tripBeingEdited.value = tg
  showTripEditModal.value = true
}

async function openAddToTrip() {
  await loadTripGroups()
  showAddToTripModal.value = true
}

async function onAddedToTrip() {
  clearSelection()
  await loadTripGroups()
  void loadDrives(true)
}

function onGroupCreated() {
  clearSelection()
  void loadDrives(true)
}

async function toggleDriveTag(drive: any, tagToToggle: string) {
  if (!vehicleStore.activeVehicle) return
  const currentTags = toggleTag(drive.tags, tagToToggle)
  try {
    await api.updateDriveTags(vehicleStore.activeVehicle.id, drive.id, currentTags)
    drive.tags = currentTags
  } catch (err: any) {
    void showAlert(t('drives.drivesView.tagUpdateError', { message: err.message }), t('shell.confirm.error'), 'danger')
  }
}
</script>

<template>
  <div class="space-y-6 max-md:pb-16">
    <!-- Drives informational banner when no telemetry is linked -->
    <div v-if="vehicleStore.activeVehicle && !vehicleStore.hasTeslaMate && total === 0 && !loading && loadError === null" class="rounded-2xl border border-info-500/30 bg-info-500/10 p-4 text-sm text-info-200">
      <span>{{ $t('drives.drivesView.noTelemetryBanner') }}</span>
    </div>

    <AddressBackfillNotice v-if="vehicleId" :vehicle-id="vehicleId" :can-edit="vehicleStore.canEdit" @resolved="loadDrives(true)" />

    <!-- Header & Filter Tabs -->
    <PageHeader :title="$t('shell.nav.drives')" :icon="PageIcon">
      {{ $t('drives.drivesView.subtitle', { unit: distanceUnit() }) }}
      <template #actions>
      <!-- Actions & Tag Filters container -->
      <div v-if="viewMode !== 'CARPOOLS'" class="flex flex-wrap items-center gap-3 self-start sm:self-center">
        <div v-if="vehicleStore.canEdit" class="flex items-center gap-2">
          <button
            @click="openCSVImportModal"
            class="btn btn-lg btn-secondary hover:border-slate-600"
          >
            <UploadCloud class="w-4 h-4 text-info-400" />
            <span>{{ $t('drives.drivesView.importCsv') }}</span>
          </button>
          <button
            @click="openManualDriveModal()"
            class="btn btn-lg btn-primary"
          >
            <Plus class="w-4 h-4" />
            <span>{{ $t('drives.drivesView.newDrive') }}</span>
          </button>
        </div>


      </div>
    
      </template>
    </PageHeader>

    <TabBar :model-value="viewMode" :tabs="viewTabs" :label="$t('shell.nav.drives')" id-prefix="drives-tab" @update:model-value="switchView($event as DrivesTab)" />

    <!-- Filters & Navigation Toolbar (Drives Mode) -->
    <DrivesToolbar
      v-if="viewMode !== 'CARPOOLS'"
      v-model:period-mode="periodMode"
      v-model:selected-month="selectedMonth"
      v-model:anchor-day="anchorDay"
      v-model:custom-from="customFrom"
      v-model:custom-to="customTo"
      v-model:search-query="searchQuery"
      :mode="listMode"
      :active-count="activeFilterCount"
      :total="viewMode === 'DRIVES' ? total : shownTrips.length"
      :loading="viewMode === 'DRIVES' ? loading : loadingTrips"
      :drives="viewMode === 'DRIVES' ? drives : shownTrips"
      @change="onFiltersChange"
    >
      <template #filters>
        <DriveFilterGroups
          v-if="viewMode === 'DRIVES'"
          v-model:unqualified-only="unqualifiedOnly"
          v-model:has-toll-only="hasTollOnly"
          v-model:toll-source="tollSource"
          v-model:carpool-filter="carpoolFilter"
          v-model:selected-tag="selectedTag"
          :unqualified-count="unqualifiedCount"
          :toll-detection-active="vehicleStore.tollDetectionActive"
          :pro-perso-enabled="prefs.proPersoEnabled"
        />
        <!-- The trips: the queue of detected trips to qualify -->
        <div v-else-if="tripSuggestions.length > 0 || tripQualifyOnly" id="drives-filter-groups" class="flex flex-wrap items-center gap-2">
          <ToQualifyFilter
            :count="tripSuggestions.length"
            :active="tripQualifyOnly"
            :title="$t('drives.tripSuggestions.toQualifyHint')"
            @toggle="tripQualifyOnly = !tripQualifyOnly"
          />
        </div>
      </template>
    </DrivesToolbar>

    <!-- Sticky Bulk Selection Bar -->
    <DriveBulkActions
      v-if="viewMode !== 'CARPOOLS'"
      :selected-drive-ids="selectedDriveIds"
      :selected-drives="selectedList"
      :selected-off-page="selectedOffPage"
      :all-page-selected="allPageSelected"
      :selected-summary-metrics="selectedSummaryMetrics"
      :bulk-applying-toll="bulkApplyingToll"
      :toll-detection-active="vehicleStore.tollDetectionActive"
      :people="vehiclePeople"
      @clear="clearSelection"
      @select-page="selectAll"
      @carpool="handleCarpoolSelectedDrives"
      @group="showGroupModal = true"
      @bulk-toll="handleBulkApplyToll"
      @tag="handleBatchTag"
      @driver="handleBatchDriver"
      @export="exportSelectedDrives"
      @add-to-trip="openAddToTrip"
      @edit="openManualDriveModal"
      @delete="handleDeleteManualDrives"
    />

    <template v-if="viewMode === 'DRIVES'">
    <DriveListSkeleton v-if="loading" />

    <LoadError v-else-if="loadError !== null" :message="loadError" @retry="loadDrives()" />

    <!-- EMPTY STATE -->
    <EmptyState v-else-if="!drives.length" :icon="PageIcon" :title="$t('drives.drivesView.noDriveFoundForThis')">
      <template #actions>
      <EmptySourceHints v-if="!(searchQuery || periodMode !== 'ALL' || selectedTag || unqualifiedOnly || hasTollOnly || carpoolFilter)" @import-csv="openCSVImportModal" />
      <button
        v-if="searchQuery || periodMode !== 'ALL' || selectedTag || unqualifiedOnly || hasTollOnly"
        @click="resetAllFilters"
        class="btn btn-secondary"
      >
        <RotateCcw class="w-3.5 h-3.5" />
        {{ $t('drives.drivesView.resetTheFilters') }}
      </button>
      </template>
    </EmptyState>

    <!-- REAL DRIVES LIST -->
    <div v-else class="space-y-3">
      <!-- Select all toggle & Total info -->
      <div class="flex items-center justify-between text-xs text-slate-400 px-2 max-sm:hidden">
        <SelectAllToggle
          v-if="vehicleStore.canEdit"
          :checked="allPageSelected"
          :indeterminate="somePageSelected"
          :label="allPageSelected ? $t('drives.drivesView.deselectPage') : $t('drives.drivesView.selectPage')"
          @toggle="selectAll"
        />
        <span v-else></span>
        <span>{{ $t('drives.drivesView.shownOf', { count: drives.length, total }) }}</span>
      </div>

      <section v-for="month in driveMonths" :key="month.month" class="space-y-2">
        <div class="flex items-baseline justify-between gap-3 px-1 pt-2">
          <h2 class="text-sm font-bold capitalize text-white">{{ month.label }}</h2>
          <p v-if="month.subtotal !== null" class="text-xs font-semibold tabular-nums text-slate-300">{{ formatAmount(month.subtotal, vehicleStore.currency) }}</p>
        </div>
        <div v-for="group in month.days" :key="group.day" class="space-y-2">
          <h3 class="px-1 pt-1 text-xs font-semibold text-slate-400">{{ group.label }}</h3>
          <SwipeToDelete v-for="d in group.items" :key="d.id" :disabled="!d.is_manual || selectedDriveIds.length > 0" @delete="handleDeleteManualDrive(d)">
            <DriveCard
              :d="d"
              :selected="selectedDriveIds.includes(d.id)"
              :selecting="selectedDriveIds.length > 0"
              @open="openCostModal"
              @toggle="toggleSelectDrive"
              @toll-entry="openTollEntry"
              @no-toll="markNoToll"
              @edit="openManualDriveModal"
              @delete="handleDeleteManualDrive"
              @carpool="(drive) => openCarpoolEditor({ driveIds: [drive.id] })"
            />
          </SwipeToDelete>
        </div>
      </section>

      <InfiniteSentinel
        v-if="hasMore"
        :loading="loadingMore"
        :error="moreError"
        :label="$t('drives.drivesView.loadMore')"
        @more="loadMoreDrives"
      />
    </div>
    </template>

    <!-- CARPOOLS: the page of the carpools, without its own header -->
    <CarpoolView v-else-if="viewMode === 'CARPOOLS'" embedded />

    <!-- TRIP GROUPS ("VOYAGES") -->
    <template v-else>
    <LoadError v-if="tripsLoadError !== null" :message="tripsLoadError" @retry="loadTripGroups()" />
    <TripSuggestions
      v-if="tripQualifyOnly"
      :suggestions="filteredSuggestions"
      :busy-key="suggestionBusyKey"
      @open="openSuggestionCostModal"
      @create="createTripFromSuggestion"
      @dismiss="dismissTripSuggestion"
    />
    <TripGroupsPanel
      v-else
      :loading-trips="loadingTrips"
      :trip-groups="filteredTrips"
      :has-filters="hasFilters"
      :expanded-trip-id="expandedTripId"
      :trip-drives="tripDrives"
      @open-cost="openTripCostModal"
      @toggle-details="toggleTripDetails"
      @edit="openTripEdit"
      @delete="handleDeleteTrip"
      @remove-drive="removeDriveFromTrip"
      @carpool="(trip) => openCarpoolEditor({ tripGroupId: trip.id })"
    />
    </template>

    <DriveCostModal
      v-model:open="showCostModal"
      v-model:drive="selectedCostDrive"
      :vehicle-id="vehicleId"
      :trip-drive-ids="costTripDriveIds"
      :trip-legs="tripLegs"
      :start-with-toll-entry="costStartWithToll"
      :refresh-drive="refreshCostDrive"
      @toggle-tag="toggleDriveTag"
      @edit-trip="editTripFromCostModal"
      @create-trip="createTripFromCostModal"
      @dismiss-trip="dismissTripFromCostModal"
      @carpool="openCarpoolEditor"
    />

    <CarpoolTripModal
      v-if="vehicleId"
      v-model:open="showCarpoolEditor"
      :vehicle-id="vehicleId"
      :editing="null"
      :create-options="carpoolCreateOptions"
      :open-token="carpoolOpenToken"
      :recent-names="recentCarpoolNames"
      @saved="onCarpoolSaved"
    />

    <DriveGroupModal
      v-model:open="showGroupModal"
      :vehicle-id="vehicleId"
      :selected-drive-ids="selectedDriveIds"
      :selected-list="selectedList"
      @saved="onGroupCreated"
    />

    <TripEditModal
      v-model:open="showTripEditModal"
      :vehicle-id="vehicleId"
      :trip="tripBeingEdited"
      @saved="onTripSaved"
    />

    <AddToTripModal
      v-model:open="showAddToTripModal"
      :vehicle-id="vehicleId"
      :trip-groups="tripGroups"
      :selected-drive-ids="selectedDriveIds"
      @saved="onAddedToTrip"
    />

    <ManualDriveModal
      v-model:open="showManualDriveModal"
      :vehicle-id="vehicleId"
      :drive="driveToEdit"
      @saved="onDriveSaved"
    />

    <CSVImportModal
      v-model:open="showCSVImportModal"
      :vehicle-id="vehicleId"
      default-type="DRIVES"
      @imported="onCSVImported"
    />
  </div>
</template>
