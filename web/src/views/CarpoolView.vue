<script setup lang="ts">
import { Users as PageIcon, Users, Plus } from '@/icons'
import LoadError from '@/components/LoadError.vue'
import PageHeader from '@/components/PageHeader.vue'
import { t } from '@/i18n'
import { APP_NAME } from '@/brand'
import { ref, onMounted, watch, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useVehicleStore } from '@/stores/vehicle'
import { useConfirm } from '@/composables/useConfirm'
import { api } from '@/services/api'
import CarpoolShareSuggestions from '@/components/carpool/CarpoolShareSuggestions.vue'
import CarpoolSummaryGrid from '@/components/carpool/CarpoolSummaryGrid.vue'
import CarpoolTripList from '@/components/carpool/CarpoolTripList.vue'
import CarpoolTripModal from '@/components/carpool/CarpoolTripModal.vue'
import CarpoolDetailModal from '@/components/carpool/CarpoolDetailModal.vue'
import DriveCostModal from '@/components/drives/DriveCostModal.vue'
import { downloadCsv } from '@/utils/csv'
import { toggleTag } from '@/utils/drives'
import { carpoolCsvHeaders, carpoolCsvRows, driveCostGapCents, recentPassengerNames } from '@/utils/carpool'
import { formatAmount } from '@/currency'
import { useRefreshHandler } from '@/composables/usePullToRefresh'
import { deleteWithUndo } from '@/composables/useUndoableDelete'
import EmptyState from '@/components/EmptyState.vue'
import InfiniteSentinel from '@/components/InfiniteSentinel.vue'
import { useInfiniteList } from '@/composables/useInfiniteList'

// The page reads the trips page by page (the latest first) and the totals apart, and owns the selection and which trip
// the modal edits; the summary, the list and the form are components. Other pages open it with a drive, drives or a trip group in the query string.
// `embedded`: the segment of the trips page, which brings the header
defineProps<{ embedded?: boolean }>()
const vehicleStore = useVehicleStore()
// Bumped when a carpool is saved: the drives still to share are read again
const shareRefresh = ref(0)
function onSaved() {
  shareRefresh.value++
  void loadData(true)
}
const { showConfirm, showAlert } = useConfirm()
const route = useRoute()
const router = useRouter()
const vehicleId = computed(() => vehicleStore.activeVehicle?.id ?? '')

const PAGE_SIZE = 30
const {
  items: trips,
  total,
  loading,
  error: loadError,
  loadingMore,
  moreError,
  hasMore,
  load: loadTrips,
  loadMore: loadMoreTrips,
} = useInfiniteList<any>(readTripsPage)
loading.value = true
const summary = ref<any>({
  total_trips: 0,
  total_passengers: 0,
  total_distance_km: 0,
  total_real_cost: 0,
  total_revenue: 0,
  total_net_cost: 0,
  coverage_rate_pct: null,
  net_cost_per_km: 0,
  total_passengers_share: 0,
  total_driver_share: 0,
})

// Create / edit modal
const showModal = ref(false)
const showDetail = ref(false)
const detailTripId = ref<string | null>(null)
// A carpool opened by a link can belong to a page that is not loaded: it is read on its own
const linkedTrip = ref<any | null>(null)
// Read from the list so the detail follows a recalculation or an edit
const detailTrip = computed(() => trips.value.find((t) => t.id === detailTripId.value) ?? (linkedTrip.value?.id === detailTripId.value ? linkedTrip.value : null))

const recentNames = computed(() => recentPassengerNames(trips.value))

function openDetail(trip: any) {
  detailTripId.value = trip.id
  showDetail.value = true
}
const editingTrip = ref<any | null>(null)
const createOptions = ref<{ driveIds?: string[]; tripGroupId?: string }>({})
const openToken = ref(0)

// Batch selection state
const selectedTripIds = ref<string[]>([])
const recalculating = ref(false)

const isAllSelected = computed(() => {
  return trips.value.length > 0 && selectedTripIds.value.length === trips.value.length
})

function toggleSelectAll() {
  if (isAllSelected.value) {
    selectedTripIds.value = []
  } else {
    selectedTripIds.value = trips.value.map((t) => t.id)
  }
}

function toggleTripSelection(tripId: string) {
  const idx = selectedTripIds.value.indexOf(tripId)
  if (idx > -1) {
    selectedTripIds.value.splice(idx, 1)
  } else {
    selectedTripIds.value.push(tripId)
  }
}

function clearTripSelection() {
  selectedTripIds.value = []
}

function exportSelectedCarpools() {
  const selected = trips.value.filter((t) => selectedTripIds.value.includes(t.id))
  if (!selected.length) return
  downloadCsv(`${t('carpool.carpoolView.csvFileName')}_${new Date().toISOString().slice(0, 10)}.csv`, carpoolCsvHeaders(vehicleStore.currency), carpoolCsvRows(selected))
}

useRefreshHandler(() => loadData(true))

async function readTripsPage(page: number) {
  if (!vehicleStore.activeVehicle) return { items: [], total: 0 }
  const res = await api.getCarpools(vehicleStore.activeVehicle.id, { page, limit: PAGE_SIZE })
  return { items: (res.trips || []) as any[], total: res.total as number }
}

// The totals cover every carpool, not only the pages shown; a failed read keeps the figures already there
async function loadSummary() {
  if (!vehicleStore.activeVehicle) return
  try {
    summary.value = (await api.getCarpoolSummary(vehicleStore.activeVehicle.id)) || summary.value
  } catch (err) {
    console.error('Failed to load carpool summary', err)
  }
}

async function loadData(silent = false) {
  if (!vehicleStore.activeVehicle) {
    loading.value = false
    return
  }
  await Promise.all([loadTrips(silent), loadSummary()])
}

// A leg of a carpool opens the cost detail of its drive in place of the carpool detail, with an arrow back to it (like the
// legs of a trip)
const showDriveModal = ref(false)
const legDrive = ref<any | null>(null)

async function fetchDrive(driveId: string) {
  const res = await api.getDrives(vehicleId.value, { driveId, limit: 1 })
  return res.drives?.[0] ?? null
}

async function openLegDrive(driveId: string) {
  try {
    const drive = await fetchDrive(driveId)
    if (!drive) return
    legDrive.value = drive
    showDetail.value = false
    showDriveModal.value = true
  } catch (err: any) {
    void showAlert(t('drives.drivesView.detailsLoadError', { message: err.message }), t('shell.confirm.error'), 'danger')
  }
}

function backToCarpool() {
  showDriveModal.value = false
  showDetail.value = true
}

async function toggleLegDriveTag(drive: any, tag: string) {
  const tags = toggleTag(drive.tags, tag)
  try {
    await api.updateDriveTags(vehicleId.value, drive.id, tags)
    drive.tags = tags
  } catch (err: any) {
    void showAlert(t('drives.drivesView.tagUpdateError', { message: err.message }), t('shell.confirm.error'), 'danger')
  }
}

function openCreateModal(options: { driveIds?: string[]; tripGroupId?: string } = {}) {
  editingTrip.value = null
  createOptions.value = options
  openToken.value++
  showModal.value = true
}

function openEditModal(trip: any) {
  editingTrip.value = trip
  openToken.value++
  showModal.value = true
}

// A single carpool goes at once and comes back with Undo, the server is asked when that time is over
function handleDelete(trip: any) {
  const vehicle = vehicleStore.activeVehicle
  if (!vehicle) return
  const index = trips.value.findIndex((x) => x.id === trip.id)
  deleteWithUndo({
    message: t('carpool.carpoolView.carpoolDeleted'),
    hide: () => {
      trips.value = trips.value.filter((x) => x.id !== trip.id)
    },
    restore: () => {
      if (trips.value.some((x) => x.id === trip.id)) return
      const next = [...trips.value]
      next.splice(Math.min(Math.max(index, 0), next.length), 0, trip)
      trips.value = next
    },
    commit: () => api.deleteCarpool(vehicle.id, trip.id),
    committed: () => void loadData(true),
    onError: (err: any) => void showAlert(t('common.deleteError', { message: err.message }), t('shell.confirm.error'), 'danger'),
  })
}

// The delete of the contextual bar: one confirmation for the whole selection
async function handleDeleteMany(list: any[]) {
  const vehicle = vehicleStore.activeVehicle
  if (!vehicle || list.length === 0) return
  if (list.length === 1) return handleDelete(list[0])
  const ok = await showConfirm({
    title: t('carpool.carpoolView.deleteTitle'),
    message: t('carpool.carpoolView.deleteManyMessage', { count: list.length }),
    confirmText: t('common.delete'),
    type: 'danger',
  })
  if (!ok) return
  try {
    for (const trip of list) await api.deleteCarpool(vehicle.id, trip.id)
  } catch (err: any) {
    void showAlert(t('common.deleteError', { message: err.message }), t('shell.confirm.error'), 'danger')
  }
  clearTripSelection()
  await loadData(true)
}

async function handleRecalculateSingle(trip: any) {
  if (!vehicleStore.activeVehicle) return
  recalculating.value = true
  try {
    await api.recalculateCarpools(vehicleStore.activeVehicle.id, [trip.id])
    void showAlert(t('carpool.carpoolView.recalcDone', { title: trip.title }), t('carpool.carpoolView.recalcDoneTitle'), 'success')
    await loadData(true)
  } catch (err: any) {
    void showAlert(t('carpool.carpoolView.recalcError', { message: err.message }), t('shell.confirm.error'), 'danger')
  } finally {
    recalculating.value = false
  }
}

async function handleFreeze(trip: any) {
  if (!vehicleStore.activeVehicle) return
  recalculating.value = true
  try {
    await api.freezeCarpool(vehicleStore.activeVehicle.id, trip.id)
    await loadData(true)
  } catch (err: any) {
    void showAlert(t('carpool.linked.error', { message: err.message }), t('shell.confirm.error'), 'danger')
  } finally {
    recalculating.value = false
  }
}

// Letting a frozen carpool follow its drives again changes its amounts: the difference is shown first
async function handleUnfreeze(trip: any) {
  if (!vehicleStore.activeVehicle) return
  const gap = driveCostGapCents(trip.legs || [])
  if (gap !== 0) {
    const ok = await showConfirm({
      title: t('carpool.linked.resyncTitle'),
      message: t('carpool.linked.resyncMessage', { amount: formatAmount(Math.abs(gap) / 100, vehicleStore.currency), direction: gap > 0 ? t('carpool.linked.more') : t('carpool.linked.less') }),
      confirmText: t('carpool.linked.resync'),
      type: 'warning',
    })
    if (!ok) return
  }
  recalculating.value = true
  try {
    await api.unfreezeCarpool(vehicleStore.activeVehicle.id, trip.id)
    await loadData(true)
  } catch (err: any) {
    void showAlert(t('carpool.linked.error', { message: err.message }), t('shell.confirm.error'), 'danger')
  } finally {
    recalculating.value = false
  }
}

// Freezing or resyncing several carpools at once: one request per carpool, a failure does not stop the others
async function applyFreezeToMany(list: any[], call: (vehicleId: string, id: string) => Promise<unknown>) {
  const vehicle = vehicleStore.activeVehicle
  if (!vehicle || list.length === 0) return
  recalculating.value = true
  let firstError = ''
  try {
    for (const trip of list) {
      try {
        await call(vehicle.id, trip.id)
      } catch (err: any) {
        firstError ||= err.message
      }
    }
    if (firstError) void showAlert(t('carpool.linked.error', { message: firstError }), t('shell.confirm.error'), 'danger')
    clearTripSelection()
    await loadData(true)
  } finally {
    recalculating.value = false
  }
}

const handleFreezeMany = (list: any[]) => applyFreezeToMany(list, api.freezeCarpool)

// Like one carpool: the total difference with what the drives cost now is shown before the amounts are replaced
async function handleUnfreezeMany(list: any[]) {
  if (list.length === 0) return
  const gap = list.reduce((sum, trip) => sum + driveCostGapCents(trip.legs || []), 0)
  if (gap !== 0) {
    const ok = await showConfirm({
      title: t('carpool.linked.resyncManyTitle', { count: list.length }),
      message: t('carpool.linked.resyncMessage', { amount: formatAmount(Math.abs(gap) / 100, vehicleStore.currency), direction: gap > 0 ? t('carpool.linked.more') : t('carpool.linked.less') }),
      confirmText: t('carpool.linked.resync'),
      type: 'warning',
    })
    if (!ok) return
  }
  await applyFreezeToMany(list, api.unfreezeCarpool)
}

async function handleBatchRecalculate() {
  if (!vehicleStore.activeVehicle || !selectedTripIds.value.length) return
  const count = selectedTripIds.value.length
  recalculating.value = true
  try {
    const res = await api.recalculateCarpools(vehicleStore.activeVehicle.id, selectedTripIds.value)
    void showAlert(t('carpool.carpoolView.recalcSelectedDone', { count: res.updated_count || count }), t('carpool.carpoolView.recalcDoneTitle'), 'success')
    clearTripSelection()
    await loadData(true)
  } catch (err: any) {
    void showAlert(t('carpool.carpoolView.recalcError', { message: err.message }), t('shell.confirm.error'), 'danger')
  } finally {
    recalculating.value = false
  }
}

watch(
  () => vehicleStore.activeVehicle?.id,
  () => clearTripSelection()
)

watch(
  () => [vehicleStore.activeVehicle?.id, vehicleStore.lastSyncTimestamp],
  () => loadData()
)

// A link from a drive or a trip (?open=<carpool id>) opens that carpool once the list is there, then leaves the address clean
async function openCarpoolFromRoute() {
  const id = typeof route.query.open === 'string' ? route.query.open : ''
  if (!id) return
  let trip = trips.value.find((t) => t.id === id)
  if (!trip && vehicleId.value) {
    trip = await api.getCarpool(vehicleId.value, id).catch(() => null)
    linkedTrip.value = trip
  }
  if (trip) openDetail(trip)
  const { open: _open, ...rest } = route.query
  router.replace({ path: route.path, query: rest })
}

function checkRouteQueryForCarpool() {
  const driveIds = [route.query.new_drive_id, ...String(route.query.new_drive_ids || '').split(',')]
    .map((v) => String(v || '').trim())
    .filter(Boolean)
  const tripGroupId = route.query.new_trip_group_id as string
  if (tripGroupId) openCreateModal({ tripGroupId })
  else if (driveIds.length) openCreateModal({ driveIds })
}

watch(
  () => [route.query.new_drive_id, route.query.new_drive_ids, route.query.new_trip_group_id],
  () => checkRouteQueryForCarpool()
)

watch(
  () => route.query.open,
  () => openCarpoolFromRoute()
)

onMounted(async () => {
  const loaded = loadData()
  checkRouteQueryForCarpool()
  await loaded
  void openCarpoolFromRoute()
})
</script>

<template>
  <div class="space-y-6">
    <!-- The page's own header, or inside the trips page the one action of the segment -->
    <div v-if="embedded" class="flex justify-end">
      <button v-if="vehicleStore.canEdit" type="button" class="btn btn-lg btn-primary" @click="openCreateModal()">
        <Plus class="w-4 h-4" />
        {{ $t('carpool.carpoolView.newCarpool') }}
      </button>
    </div>
    <PageHeader v-else :title="$t('carpool.carpoolView.carpooling')" :icon="PageIcon">
      {{ $t('carpool.carpoolView.multiLegTripsPassengersGetting') }}
      <template #actions>
      <button
        v-if="vehicleStore.canEdit"
        @click="openCreateModal()"
        class="btn btn-lg btn-primary self-start sm:self-auto"
      >
        <Plus class="w-4 h-4" />
        {{ $t('carpool.carpoolView.newCarpool') }}
      </button>

      </template>
    </PageHeader>

    <!-- Viewer mode banner -->
    <div
      v-if="!vehicleStore.canEdit"
      class="p-3.5 card flex items-center gap-3 text-xs text-slate-400"
    >
      <Users class="w-4 h-4 text-slate-400 shrink-0" />
      <span>{{ $t('carpool.carpoolView.youAreViewingThisVehicle') }} <strong>{{ $t('carpool.carpoolView.readOnly') }}</strong>{{ $t('carpool.carpoolView.modeCreatingAndEditingCarpools') }}</span>
    </div>

    <!-- KPI Summary Grid: totals of the trips below, nothing to total before the first one -->
    <CarpoolSummaryGrid v-if="trips.length > 0" :summary="summary" />

    <CarpoolShareSuggestions v-if="vehicleId" :vehicle-id="vehicleId" :can-edit="vehicleStore.canEdit" :refresh-key="shareRefresh" @share="(id) => openCreateModal({ driveIds: [id] })" />

    <!-- Loading -->
    <div v-if="loading" class="space-y-4 animate-pulse">
      <div v-for="i in 2" :key="i" class="card p-6 h-48"></div>
    </div>

    <!-- Empty State -->
    <LoadError v-else-if="loadError !== null" :message="loadError" @retry="loadData()" />

    <EmptyState v-else-if="trips.length === 0" :icon="Users" :title="$t('carpool.carpoolView.noCarpooledTripYet')">
      {{ $t('carpool.carpoolView.selectTheLegsOfYour', { APP_NAME }) }}
      <template v-if="vehicleStore.canEdit" #actions>
        <button @click="openCreateModal()" class="btn btn-primary">
          <Plus class="w-4 h-4" />
          {{ $t('carpool.carpoolView.createMyFirstCarpool') }}
        </button>
      </template>
    </EmptyState>

    <!-- Trips List -->
    <CarpoolTripList
      v-else
      :trips="trips"
      :total="total"
      :selected-trip-ids="selectedTripIds"
      :recalculating="recalculating"
      @clear="clearTripSelection"
      @batch-recalculate="handleBatchRecalculate"
      @export="exportSelectedCarpools"
      @toggle-all="toggleSelectAll"
      @toggle="toggleTripSelection"
      @recalculate="handleRecalculateSingle"
      @open="openDetail"
      @edit="openEditModal"
      @delete="handleDelete"
      @delete-many="handleDeleteMany"
      @batch-freeze="handleFreezeMany"
      @batch-resync="handleUnfreezeMany"
    />

    <InfiniteSentinel
      v-if="hasMore"
      :loading="loadingMore"
      :error="moreError"
      :label="$t('carpool.carpoolView.loadMore')"
      @more="loadMoreTrips"
    />

    <CarpoolDetailModal
      v-model:open="showDetail"
      :trip="detailTrip"
      :recalculating="recalculating"
      @edit="(trip) => { showDetail = false; openEditModal(trip) }"
      @recalculate="handleRecalculateSingle"
      @freeze="handleFreeze"
      @unfreeze="handleUnfreeze"
      @open-drive="openLegDrive"
    />

    <DriveCostModal
      v-model:open="showDriveModal"
      v-model:drive="legDrive"
      :vehicle-id="vehicleId"
      :trip-drive-ids="[]"
      :trip-legs="[]"
      :start-with-toll-entry="false"
      :refresh-drive="fetchDrive"
      :back-label="$t('carpool.carpoolView.backToTheCarpool')"
      @back="backToCarpool"
      @toggle-tag="toggleLegDriveTag"
      @carpool="openCreateModal"
    />

    <CarpoolTripModal
      v-model:open="showModal"
      :vehicle-id="vehicleId"
      :editing="editingTrip"
      :create-options="createOptions"
      :open-token="openToken"
      :recent-names="recentNames"
      @saved="onSaved"
    />
  </div>
</template>
