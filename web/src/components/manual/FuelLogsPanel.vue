<script setup lang="ts">
import EmptyState from '@/components/EmptyState.vue'
import CardOpenButton from '@/components/CardOpenButton.vue'
import BulkSelectionBar from '@/components/BulkSelectionBar.vue'
import RowSelectBox from '@/components/RowSelectBox.vue'
import SelectionHint from '@/components/SelectionHint.vue'
import { useRowSelection } from '@/composables/useRowSelection'
import { editDeleteActions, type SelectionAction } from '@/utils/selectionActions'
import { intlLocale, t } from '@/i18n'
import { ref, computed, watch, onMounted } from 'vue'
import { Fuel, Plus } from '@/icons'
import { useConfirm } from '@/composables/useConfirm'
import { api } from '@/services/api'
import { useVehicleStore } from '@/stores/vehicle'
import { formatAmount } from '@/currency'
import { distanceUnit, formatDistance, formatFuelConsumptionValue, formatVolumeValue, fuelConsumptionUnit, perDistance, perVolume, volumeUnitLabel } from '@/units'
import LoadError from '@/components/LoadError.vue'
import { useRefreshHandler } from '@/composables/usePullToRefresh'
import { deleteRowsWithUndo, deleteWithUndo } from '@/composables/useUndoableDelete'
import SwipeToDelete from '@/components/SwipeToDelete.vue'
import FuelFillUpModal from '@/components/manual/FuelFillUpModal.vue'

const props = defineProps<{
  vehicleId: string
  canEdit: boolean
}>()

const vehicleStore = useVehicleStore()
const currency = computed(() => vehicleStore.currency)
const { showConfirm, showAlert } = useConfirm()

const stats = ref<any | null>(null)
const loading = ref(false)
const loadError = ref<string | null>(null)
const showForm = ref(false)
const editingLog = ref<any | null>(null)
// Newest first for display; the API sends them oldest first
const logs = computed<any[]>(() => [...(stats.value?.logs || [])].reverse())

function fmtMoney(v: number | null | undefined, digits = 2): string {
  return formatAmount(Number(v || 0), currency.value, digits)
}

function fmtNum(v: number | null | undefined, digits = 1): string {
  return Number(v).toLocaleString(intlLocale(), { minimumFractionDigits: digits, maximumFractionDigits: digits })
}

function fmtDate(d: string): string {
  return new Date(d).toLocaleDateString(intlLocale(), { day: '2-digit', month: 'short', year: 'numeric' })
}

// `silent` reloads behind the list that is already shown instead of swapping it for the loading state.
// Compared strictly: the function is also bound as an event handler, which passes the event.
async function load(silent?: boolean) {
  if (silent !== true) loading.value = true
  loadError.value = null
  try {
    stats.value = await api.getFuelLogs(props.vehicleId)
  } catch (err: any) {
    console.error('Failed to load fuel logs', err)
    loadError.value = err?.message ?? ''
  } finally {
    loading.value = false
  }
}

function openAdd() {
  editingLog.value = null
  showForm.value = true
}

function openEdit(log: any) {
  editingLog.value = log
  showForm.value = true
}

// A single fill-up goes at once and comes back with Undo, the server is asked when that time is over
function remove(log: any) {
  const all = () => stats.value?.logs ?? []
  const index = all().findIndex((l: any) => l.id === log.id)
  const setLogs = (logs: any[]) => {
    stats.value = { ...stats.value, logs }
  }
  deleteWithUndo({
    message: t('manual.fuelLogsPanel.fillUpDeleted'),
    hide: () => setLogs(all().filter((l: any) => l.id !== log.id)),
    restore: () => {
      if (all().some((l: any) => l.id === log.id)) return
      const next = [...all()]
      next.splice(Math.min(Math.max(index, 0), next.length), 0, log)
      setLogs(next)
    },
    commit: () => api.deleteFuelLog(props.vehicleId, log.id),
    committed: () => void load(true),
    onError: (err: any) => void showAlert(err?.message || t('manual.fuelLogsPanel.deleteFailed'), t('shell.confirm.error'), 'danger'),
  })
}

// The detail's own Delete: the sheet closes and the fill-up goes with the same Undo
function removeEditing(log: any) {
  showForm.value = false
  remove(log)
}

// A selection goes at once under one Undo, the server is asked when that time is over
const logsList = computed({
  get: () => stats.value?.logs ?? [],
  set: (logs: any[]) => {
    stats.value = { ...stats.value, logs }
  },
})
function removeMany(items: any[]) {
  if (items.length === 1) return remove(items[0])
  deleteRowsWithUndo({
    list: logsList,
    items,
    key: (l) => l.id,
    message: t('manual.fuelLogsPanel.fillUpsDeleted', { count: items.length }),
    remove: (l) => api.deleteFuelLog(props.vehicleId, l.id),
    committed: () => void load(true),
    onError: (err: any) => void showAlert(err?.message || t('manual.fuelLogsPanel.deleteFailed'), t('shell.confirm.error'), 'danger'),
  })
}

const { selected, selectedIds, clear, isSelected, toggle, showBox, press, onPointerdown, onRowClick } = useRowSelection(
  logs,
  computed(() => props.canEdit),
  openEdit,
)
const actions = computed<SelectionAction[]>(() => editDeleteActions(selected.value, { edit: openEdit, remove: removeMany }))

watch(() => props.vehicleId, () => load())
onMounted(() => load())
useRefreshHandler(() => load(true))
</script>

<template>
  <div class="space-y-3">
    <div v-if="canEdit" class="flex items-center justify-end gap-3">
      <button
        type="button"
        class="btn btn-lg btn-primary"
        @click="openAdd"
      >
        <Plus class="w-3.5 h-3.5" /> {{ $t('manual.fuelLogsPanel.newFillUp') }}
      </button>
    </div>

    <div v-if="stats && stats.fill_ups > 0" class="grid grid-cols-2 md:grid-cols-4 gap-3">
      <div class="card p-3">
        <div class="text-xs text-slate-400">{{ $t('manual.fuelLogsPanel.fillUps') }}</div>
        <div class="text-lg font-bold text-white">{{ stats.fill_ups }}</div>
      </div>
      <div class="card p-3">
        <div class="text-xs text-slate-400">{{ $t('manual.fuelLogsPanel.totalSpent') }}</div>
        <div class="text-lg font-bold text-white">{{ fmtMoney(stats.total_cost, 0) }}</div>
      </div>
      <div class="card p-3">
        <div class="text-xs text-slate-400">{{ $t('manual.fuelLogsPanel.averagePricePerLitre', { unit: volumeUnitLabel() }) }}</div>
        <div class="text-lg font-bold text-white">{{ stats.avg_price_per_liter ? `${fmtMoney(perVolume(stats.avg_price_per_liter), 3)}/${volumeUnitLabel()}` : '—' }}</div>
      </div>
      <div class="card p-3">
        <div class="text-xs text-slate-400">{{ $t('manual.fuelLogsPanel.averageConsumption') }}</div>
        <div class="text-lg font-bold text-white">{{ stats.consumption_l_100km ? `${formatFuelConsumptionValue(stats.consumption_l_100km, 2)} ${fuelConsumptionUnit()}` : $t('manual.fuelLogsPanel.notMeasurable') }}</div>
      </div>
    </div>
    <p v-if="stats && stats.unmeasurable_segments > 0" class="text-xs text-warning-400">
      {{ $t('manual.fuelLogsPanel.intervalSBetweenFullTanks', { unmeasurable_segments: stats.unmeasurable_segments }) }}
    </p>
    <p v-if="stats && stats.fill_ups_without_mileage > 0" class="text-xs text-slate-400">
      {{ $t('manual.fuelLogsPanel.fillUpSWithoutMileage', { fill_ups_without_mileage: stats.fill_ups_without_mileage }) }}
      <template v-if="stats.estimated_segments > 0">{{ $t('manual.fuelLogsPanel.theConsumptionOfIntervalS', { estimated_segments: stats.estimated_segments }) }}</template>
    </p>
    <p v-if="stats && stats.fill_ups > 0 && !stats.consumption_l_100km && stats.unmeasurable_segments === 0" class="text-xs text-slate-400">
      {{ $t('manual.fuelLogsPanel.consumptionIsCalculatedBetweenTwo') }}
    </p>

    <div v-if="loading && !stats" class="text-sm text-slate-400">{{ $t('manual.fuelLogsPanel.loading') }}</div>
    <LoadError v-else-if="loadError !== null" :message="loadError" @retry="load" />
    <EmptyState v-else-if="logs.length === 0" :icon="Fuel" :title="$t('manual.fuelLogsPanel.noFillUpRecordedEnter')">
      {{ $t('manual.fuelLogsPanel.noFillUpHint', { unit: distanceUnit() }) }}
    </EmptyState>
    <template v-else>
    <BulkSelectionBar v-if="canEdit" :count="selectedIds.length" :actions="actions" @clear="clear" />
    <SelectionHint v-if="canEdit" />
    <ul class="space-y-2">
      <li v-for="log in logs" :key="log.id">
      <SwipeToDelete :disabled="!canEdit || selectedIds.length > 0" @delete="remove(log)">
      <div
        class="relative hover:border-slate-700/90 card p-3 flex items-center justify-between gap-3 transition-colors max-sm:select-none max-sm:[-webkit-touch-callout:none]"
        :class="{ '!border-rose-500/40 bg-slate-800/40': isSelected(log) }"
      >
        <CardOpenButton
          v-if="canEdit"
          :label="$t('manual.fuelLogsPanel.editFillUp', { date: fmtDate(log.date) })"
          @click="onRowClick(log)"
          @pointerdown="onPointerdown($event, log)"
          @pointermove="press.onPointermove"
          @pointerup="press.onPointerup"
          @pointercancel="press.onPointercancel"
          @contextmenu="press.onContextmenu"
        />
        <RowSelectBox v-if="showBox" :id="log.id" :checked="isSelected(log)" :label="$t('shell.bulkSelectionBar.selectRow')" @toggle="toggle(log)" />
        <div class="min-w-0 flex-1">
          <div class="text-sm font-semibold text-white flex flex-wrap items-center gap-2">
            {{ fmtDate(log.date) }}
            <template v-if="log.odometer != null"> · {{ formatDistance(log.odometer) }}</template>
            <template v-else-if="log.odometer_estimated != null"> · ≈ {{ formatDistance(log.odometer_estimated) }} <span class="text-xs px-2 py-0.5 rounded-full border border-slate-600 text-slate-400">{{ $t('manual.fuelLogsPanel.estimated') }}</span></template>
            <span v-if="!log.is_full_tank" class="text-xs px-2 py-0.5 rounded-full border border-slate-600 text-slate-400">{{ $t('manual.fuelLogsPanel.partial') }}</span>
          </div>
          <div class="text-xs text-slate-400 mt-0.5">
            {{ fmtMoney(log.amount) }}
            <template v-if="log.liters"> · {{ formatVolumeValue(log.liters, 2) }} {{ volumeUnitLabel() }}</template>
            <template v-if="log.price_per_liter"> · {{ fmtMoney(perVolume(log.price_per_liter), 3) }}/{{ volumeUnitLabel() }}</template>
            <template v-if="log.consumption_l_100km"> · <span class="text-slate-300">{{ log.segment_estimated ? '≈ ' : '' }}{{ formatFuelConsumptionValue(log.consumption_l_100km, 2) }} {{ fuelConsumptionUnit() }}</span></template>
            <template v-if="log.cost_per_km"> · {{ fmtMoney(perDistance(log.cost_per_km), 3) }}/{{ distanceUnit() }}</template>
          </div>
        </div>
      </div>
      </SwipeToDelete>
      </li>
    </ul>
    </template>

    <FuelFillUpModal v-model:open="showForm" :vehicle-id="vehicleId" :editing="editingLog" :can-edit="canEdit" @saved="load()" @delete="removeEditing" />
  </div>
</template>
