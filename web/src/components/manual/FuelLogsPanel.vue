<script setup lang="ts">
import EmptyState from '@/components/EmptyState.vue'
import BulkSelectionBar from '@/components/BulkSelectionBar.vue'
import FuelLogRow from '@/components/manual/FuelLogRow.vue'
import SelectionHint from '@/components/SelectionHint.vue'
import { useRowSelection } from '@/composables/useRowSelection'
import { editDeleteActions, type SelectionAction } from '@/utils/selectionActions'
import { intlLocale, t } from '@/i18n'
import { ref, computed, watch, onMounted } from 'vue'
import { Fuel, Plus } from '@/icons'
import { api } from '@/services/api'
import { useVehicleStore } from '@/stores/vehicle'
import { formatAmount } from '@/currency'
import { distanceUnit, formatFuelConsumptionValue, formatVolume, fuelConsumptionUnit, perVolume, volumeUnitLabel } from '@/units'
import { fuelLogsInPeriod, fuelMonths, fuelPeriodSummary } from '@/utils/fuelLogs'
import PeriodBar from '@/components/period/PeriodBar.vue'
import { usePeriod } from '@/composables/usePeriod'
import { dayKey } from '@/utils/dates'
import LoadError from '@/components/LoadError.vue'
import { useRefreshHandler } from '@/composables/usePullToRefresh'
import { useRecordDeletion } from '@/composables/useRecordDeletion'
import SwipeToDelete from '@/components/SwipeToDelete.vue'
import FuelFillUpModal from '@/components/manual/FuelFillUpModal.vue'

const props = defineProps<{
  vehicleId: string
  canEdit: boolean
}>()

const vehicleStore = useVehicleStore()
const currency = computed(() => vehicleStore.currency)
const { removeRecords } = useRecordDeletion()

const stats = ref<any | null>(null)
const loading = ref(false)
const loadError = ref<string | null>(null)
const showForm = ref(false)
const editingLog = ref<any | null>(null)
const { period, bounds, label: periodLabel, rounded, canPrevious, canNext, set: setPeriod, step: stepPeriod } = usePeriod('fuel')
// Every fill-up is loaded, and the consumption of each interval is measured by the server over the whole history: the
// period only chooses the rows, the months and the figures shown. The API sends them oldest first.
const allLogs = computed<any[]>(() => stats.value?.logs ?? [])
const summary = computed(() => fuelPeriodSummary(allLogs.value, bounds.value))
// Newest first for display
const logs = computed<any[]>(() => fuelLogsInPeriod(allLogs.value, bounds.value).reverse())

function fmtMoney(v: number | null | undefined, digits = 2): string {
  return formatAmount(Number(v || 0), currency.value, digits)
}

function fmtNum(v: number | null | undefined, digits = 1): string {
  return Number(v).toLocaleString(intlLocale(), { minimumFractionDigits: digits, maximumFractionDigits: digits })
}

// Fill-ups under a header per month, the newest first
const months = computed(() =>
  fuelMonths(logs.value, dayKey(), (date, withYear) => date.toLocaleDateString(intlLocale(), { month: 'long', ...(withYear ? { year: 'numeric' } : {}) })),
)

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

// One fill-up or a selection: the shared deletion hides it at once and offers Undo
const logsList = computed({
  get: () => stats.value?.logs ?? [],
  set: (logs: any[]) => {
    stats.value = { ...stats.value, logs }
  },
})
function removeMany(items: any[]) {
  void removeRecords({ kind: 'fuel_log', items, place: { list: logsList }, loaded: true, reload: () => load(true) })
}
const remove = (log: any) => removeMany([log])

// The detail's own Delete: the sheet closes and the fill-up goes with the same Undo
function removeEditing(log: any) {
  showForm.value = false
  remove(log)
}

const { selected, selectedIds, clear, isSelected, toggle } = useRowSelection(
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

    <div class="md:sticky md:top-[4.25rem] md:z-30 md:bg-slate-950/90 md:py-2 md:backdrop-blur-md">
      <PeriodBar :period="period" :label="periodLabel" :rounded="rounded" :can-previous="canPrevious" :can-next="canNext" @step="stepPeriod" @pick="setPeriod" />
    </div>

    <p v-if="logs.length > 0" class="px-1 text-xs font-semibold text-slate-400" data-testid="fuel-period-label">{{ $t('manual.fuelLogsPanel.periodFigures', { period: periodLabel }) }}</p>
    <div v-if="logs.length > 0" class="grid grid-cols-2 md:grid-cols-4 gap-3">
      <div class="card p-3">
        <div class="text-xs text-slate-400">{{ $t('manual.fuelLogsPanel.fillUps') }}</div>
        <div class="text-lg font-bold text-white">{{ summary.fillUps }}</div>
      </div>
      <div class="card p-3">
        <div class="text-xs text-slate-400">{{ $t('manual.fuelLogsPanel.totalSpent') }}</div>
        <div class="text-lg font-bold text-white">{{ fmtMoney(summary.totalCost, 0) }}</div>
      </div>
      <div class="card p-3">
        <div class="text-xs text-slate-400">{{ $t('manual.fuelLogsPanel.averagePricePerLitre', { unit: volumeUnitLabel() }) }}</div>
        <div class="text-lg font-bold text-white">{{ summary.pricePerLiter ? `${fmtMoney(perVolume(summary.pricePerLiter), 3)}/${volumeUnitLabel()}` : '—' }}</div>
      </div>
      <div class="card p-3">
        <div class="text-xs text-slate-400">{{ $t('manual.fuelLogsPanel.averageConsumption') }}</div>
        <div class="font-bold text-white" :class="summary.consumption ? 'text-lg' : 'text-base'">{{ summary.consumption ? `${formatFuelConsumptionValue(summary.consumption, 2)} ${fuelConsumptionUnit()}` : $t('manual.fuelLogsPanel.notMeasurable') }}</div>
      </div>
    </div>
    <p v-if="logs.length > 0 && summary.unmeasurable > 0" class="text-xs text-warning-400">
      {{ $t('manual.fuelLogsPanel.intervalSBetweenFullTanks', { unmeasurable_segments: summary.unmeasurable }) }}
    </p>
    <p v-if="logs.length > 0 && summary.withoutMileage > 0" class="text-xs text-slate-400">
      {{ $t('manual.fuelLogsPanel.fillUpSWithoutMileage', { fill_ups_without_mileage: summary.withoutMileage }) }}
      <template v-if="summary.estimated > 0">{{ $t('manual.fuelLogsPanel.theConsumptionOfIntervalS', { estimated_segments: summary.estimated }) }}</template>
    </p>
    <p v-if="logs.length > 0 && !summary.consumption && summary.unmeasurable === 0" class="text-xs text-slate-400">
      {{ $t('manual.fuelLogsPanel.consumptionIsCalculatedBetweenTwo') }}
    </p>

    <div v-if="loading && !stats" class="text-sm text-slate-400">{{ $t('manual.fuelLogsPanel.loading') }}</div>
    <LoadError v-else-if="loadError !== null" :message="loadError" @retry="load" />
    <EmptyState v-else-if="allLogs.length > 0 && logs.length === 0" :icon="Fuel" :title="$t('manual.fuelLogsPanel.noFillUpInPeriod')" data-testid="fuel-period-empty">
      {{ $t('manual.fuelLogsPanel.noFillUpInPeriodHint', { period: periodLabel }) }}
    </EmptyState>
    <EmptyState v-else-if="logs.length === 0" :icon="Fuel" :title="$t('manual.fuelLogsPanel.noFillUpRecordedEnter')">
      {{ $t('manual.fuelLogsPanel.noFillUpHint', { unit: distanceUnit() }) }}
    </EmptyState>
    <template v-else>
    <BulkSelectionBar v-if="canEdit" :count="selectedIds.length" :actions="actions" @clear="clear" />
    <SelectionHint v-if="canEdit" />
    <section v-for="m in months" :key="m.month" class="space-y-2">
      <div class="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-0.5 px-1 pt-2">
        <h2 class="text-sm font-bold capitalize text-white">{{ m.label }}</h2>
        <p class="ml-auto flex flex-wrap items-baseline justify-end gap-x-1.5 text-xs font-semibold tabular-nums text-slate-300" data-testid="fuel-month-totals">
          <span v-if="m.totals.liters > 0">{{ formatVolume(m.totals.liters, 1) }}</span>
          <span v-if="m.totals.liters > 0" class="text-slate-500" aria-hidden="true">·</span>
          <span>{{ fmtMoney(m.totals.amount) }}</span>
          <template v-if="m.totals.pricePerLiter">
            <span class="text-slate-500" aria-hidden="true">·</span>
            <span>{{ fmtMoney(perVolume(m.totals.pricePerLiter), 3) }}/{{ volumeUnitLabel() }}</span>
          </template>
        </p>
      </div>
      <SwipeToDelete v-for="log in m.items" :key="log.id" :disabled="!canEdit || selectedIds.length > 0" @delete="remove(log)">
        <FuelLogRow
          :log="log"
          :currency="currency"
          :can-edit="canEdit"
          :selected="isSelected(log)"
          :selecting="selectedIds.length > 0"
          @open="openEdit"
          @toggle="toggle"
        />
      </SwipeToDelete>
    </section>
    </template>

    <FuelFillUpModal v-model:open="showForm" :vehicle-id="vehicleId" :editing="editingLog" :can-edit="canEdit" @saved="load()" @delete="removeEditing" />
  </div>
</template>
