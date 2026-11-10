<script setup lang="ts">
import { intlLocale, t } from '@/i18n'
import { computed, ref, watch, onMounted } from 'vue'
import { useRefreshHandler } from '@/composables/usePullToRefresh'
import { Fuel, Gauge, Lock, Plus, Trash2 } from '@/icons'
import ModalShell from '@/components/ModalShell.vue'
import OdometerReadingForm from '@/components/manual/OdometerReadingForm.vue'
import { useIsMobile } from '@/composables/useIsMobile'
import EmptyState from '@/components/EmptyState.vue'
import CardOpenButton from '@/components/CardOpenButton.vue'
import RowMenu from '@/components/RowMenu.vue'
import BulkSelectionBar from '@/components/BulkSelectionBar.vue'
import RowSelectBox from '@/components/RowSelectBox.vue'
import RowLeading from '@/components/RowLeading.vue'
import SelectionHint from '@/components/SelectionHint.vue'
import { useRowSelection } from '@/composables/useRowSelection'
import { editDeleteActions, type SelectionAction } from '@/utils/selectionActions'
import KpiCard from '@/components/KpiCard.vue'
import { useConfirm } from '@/composables/useConfirm'
import { useVehicleStore } from '@/stores/vehicle'
import { api } from '@/services/api'
import { useStaleWrite } from '@/composables/useStaleWrite'
import { distanceUnit, formatDistance, formatDistanceValue } from '@/units'
import { useSubmit } from '@/composables/useSubmit'
import { deleteWithUndo } from '@/composables/useUndoableDelete'
import SwipeToDelete from '@/components/SwipeToDelete.vue'
import { canRefuel } from '@/utils/vehicles'
import { mergeOdometerHistory } from '@/utils/odometerHistory'

const props = defineProps<{
  vehicle: any
  canEdit: boolean
}>()

const vehicleStore = useVehicleStore()
const { showConfirm, showAlert } = useConfirm()

const readings = ref<any[]>([])
const fuelLogs = ref<any[]>([])
// The readings plus the fill-ups that carry a mileage: both feed the monthly smoothing and the estimates
const history = computed(() => mergeOdometerHistory(readings.value, fuelLogs.value))
// The latest mileage of the history, and what the vehicle has driven since: the figures the page opens on
const lastReading = computed(() => history.value.reduce<(typeof history.value)[number] | null>((best, r) => (!best || r.date >= best.date ? r : best), null))
const sinceLastReading = computed(() => {
  const current = Number(props.vehicle?.current_odometer)
  return lastReading.value && Number.isFinite(current) && current >= lastReading.value.odometer ? current - lastReading.value.odometer : null
})
const loading = ref(false)
const { explainStaleWrite } = useStaleWrite()
const editingId = ref<string | null>(null)
const editingVersion = ref<number | undefined>(undefined)
const isMobile = useIsMobile()
// On a phone the form is a sheet: it opens for a new reading or an edit, and closes once saved or dismissed
const formOpen = ref(false)
const form = ref({
  date: new Date().toISOString().substring(0, 10),
  odometer: '',
  notes: '',
})

// `silent` reloads behind the list that is already shown (pull to refresh) instead of swapping it for the loading state.
async function load(silent = false) {
  if (!silent) loading.value = true
  try {
    const refuels = canRefuel(props.vehicle.powertrain)
    const [list, fuel] = await Promise.all([
      api.getOdometerCheckpoints(props.vehicle.id),
      refuels ? api.getFuelLogs(props.vehicle.id).catch((err: any) => { console.error('Failed to load fill-ups', err); return null }) : Promise.resolve(null),
    ])
    readings.value = list
    fuelLogs.value = fuel?.logs ?? []
  } catch (err: any) {
    void showAlert(t('manual.odometerReadingsPanel.loadError', { message: err.message }), t('shell.confirm.error'), 'danger')
  } finally {
    loading.value = false
  }
}

function resetForm() {
  editingId.value = null
  editingVersion.value = undefined
  form.value = {
    date: new Date().toISOString().substring(0, 10),
    odometer: props.vehicle?.current_odometer ? String(Math.round(props.vehicle.current_odometer)) : '',
    notes: '',
  }
}

function openNew() {
  resetForm()
  formOpen.value = true
}

function startEdit(r: any) {
  formOpen.value = true
  editingId.value = r.id
  editingVersion.value = r.version
  form.value = {
    date: new Date(r.date).toISOString().substring(0, 10),
    odometer: String(Math.round(r.odometer)),
    notes: r.notes || '',
  }
}

const { pending: submitting, run: runOnce } = useSubmit()

async function saveAction() {
  const odo = Number(form.value.odometer)
  if (form.value.odometer === '' || Number.isNaN(odo) || odo < 0) {
    void showAlert(t('manual.odometerReadingsPanel.validOdometer'), t('common.requiredField'), 'warning')
    return
  }
  try {
    const payload = {
      date: form.value.date,
      odometer: odo,
      notes: form.value.notes.trim() || undefined,
    }
    if (editingId.value) {
      await api.updateOdometerCheckpoint(props.vehicle.id, editingId.value, { ...payload, expected_version: editingVersion.value })
    } else {
      await api.createOdometerCheckpoint(props.vehicle.id, payload)
    }
    formOpen.value = false
    resetForm()
    await load(true)
    // A reading raises the odometer of a combustion vehicle and reshapes the monthly smoothing
    await vehicleStore.fetchVehicles()
    vehicleStore.lastSyncTimestamp = Date.now()
  } catch (err: any) {
    if (await explainStaleWrite(err)) {
      formOpen.value = false
      resetForm()
      await load(true)
      return
    }
    void showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger')
  }
}
const save = () => runOnce(saveAction)

// A single reading goes at once and comes back with Undo, the server is asked when that time is over
function remove(r: any) {
  const index = readings.value.findIndex((x) => x.id === r.id)
  deleteWithUndo({
    message: t('manual.odometerReadingsPanel.readingDeleted'),
    hide: () => {
      readings.value = readings.value.filter((x) => x.id !== r.id)
    },
    restore: () => {
      if (readings.value.some((x) => x.id === r.id)) return
      const next = [...readings.value]
      next.splice(Math.min(Math.max(index, 0), next.length), 0, r)
      readings.value = next
    },
    commit: () => api.deleteOdometerCheckpoint(props.vehicle.id, r.id),
    committed: () => {
      void load(true)
      vehicleStore.lastSyncTimestamp = Date.now()
    },
    onError: (err: any) => void showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger'),
  })
}

async function removeMany(items: any[]) {
  if (items.length === 1) return remove(items[0])
  const ok = await showConfirm({
    title: t('manual.odometerReadingsPanel.deleteTitle'),
    message: t('manual.odometerReadingsPanel.deleteManyMessage', { count: items.length }),
    confirmText: t('common.delete'),
    type: 'danger',
  })
  if (!ok) return
  try {
    for (const item of items) await api.deleteOdometerCheckpoint(props.vehicle.id, item.id)
  } catch (err: any) {
    void showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger')
  }
  await load(true)
  vehicleStore.lastSyncTimestamp = Date.now()
}

// Only the manual readings can be edited or deleted: fill-ups and Home Assistant readings are listed for context
const isManualReading = (r: any) => r.kind !== 'FUEL' && r.source !== 'HA'
const manualReadings = computed(() => history.value.filter(isManualReading))
const { selected, selectedIds, clear, isSelected, toggle, showBox, press, onPointerdown, onRowClick } = useRowSelection(
  manualReadings,
  computed(() => props.canEdit),
  (r) => startEdit(r),
)
const actions = computed<SelectionAction[]>(() => editDeleteActions(selected.value, { edit: (r) => startEdit(r), remove: removeMany }))

// Dismissing the sheet drops an edit in progress, so the next "Add" starts from a clean form
watch(formOpen, (open) => {
  if (!open) resetForm()
})
watch(() => props.vehicle?.id, () => {
  resetForm()
  void load()
})
onMounted(() => {
  resetForm()
  void load()
})
useRefreshHandler(() => load(true))
</script>

<template>
  <div class="space-y-5">
    <p class="text-xs text-slate-400 max-sm:hidden">
      {{ $t('manual.odometerReadingsPanel.odometerReadingsSpreadTheUntracked') }}
    </p>

    <div v-if="!loading" class="grid grid-cols-2 gap-2.5 sm:grid-cols-3">
      <KpiCard
        compact
        class="col-span-2 sm:col-span-1"
        :label="$t('manual.odometerReadingsPanel.currentReading')"
        :value="vehicle?.current_odometer ? formatDistanceValue(vehicle.current_odometer) : null"
        :unit="` ${distanceUnit()}`"
      />
      <KpiCard
        compact
        :label="$t('manual.odometerReadingsPanel.lastReading')"
        :value="lastReading ? formatDistanceValue(lastReading.odometer) : null"
        :unit="` ${distanceUnit()}`"
      >
        <p v-if="lastReading" class="mt-1 text-xs text-slate-400">{{ new Date(lastReading.date).toLocaleDateString(intlLocale(), { day: 'numeric', month: 'short', year: 'numeric' }) }}</p>
      </KpiCard>
      <KpiCard
        compact
        :label="$t('manual.odometerReadingsPanel.sinceLastReading')"
        :value="sinceLastReading === null ? null : formatDistanceValue(sinceLastReading)"
        :unit="` ${distanceUnit()}`"
      />
    </div>

    <!-- A phone adds and edits in a sheet, opened by the Add button of the history -->
    <OdometerReadingForm
      v-if="canEdit && !isMobile"
      v-model:date="form.date"
      v-model:odometer="form.odometer"
      v-model:notes="form.notes"
      :editing="!!editingId"
      :submitting="submitting"
      @submit="save"
      @cancel="resetForm"
    />
    <Teleport v-if="canEdit && isMobile" to="body">
      <ModalShell v-model:open="formOpen" :title="editingId ? $t('manual.odometerReadingsPanel.edit') : $t('manual.odometerReadingsPanel.new')" size="sm">
        <p class="mb-4 text-xs text-slate-400">{{ $t('manual.odometerReadingsPanel.odometerReadingsSpreadTheUntracked') }}</p>
        <OdometerReadingForm
          v-model:date="form.date"
          v-model:odometer="form.odometer"
          v-model:notes="form.notes"
          in-sheet
          :editing="!!editingId"
          :submitting="submitting"
          @submit="save"
          @cancel="formOpen = false"
        />
      </ModalShell>
    </Teleport>

    <div class="space-y-3">
      <div class="flex items-center justify-between">
        <h4 class="text-xs font-bold text-white">{{ $t('manual.odometerReadingsPanel.historyOfRecordedReadings') }}</h4>
        <div v-if="isMobile" class="flex items-center gap-1">
          <button v-if="canEdit" type="button" class="btn btn-primary" @click="openNew">
            <Plus class="h-4 w-4" aria-hidden="true" />{{ $t('manual.odometerReadingsPanel.add') }}
          </button>
          <slot name="tools" />
        </div>
        <span v-else class="text-xs text-slate-400">{{ $t('manual.odometerReadingsPanel.pointS', { length: history.length }) }}</span>
      </div>

      <div v-if="loading" class="py-8 text-center text-xs text-slate-400">{{ $t('manual.odometerReadingsPanel.loadingTheReadings') }}</div>

      <EmptyState v-else-if="history.length === 0" compact :title="$t('manual.odometerReadingsPanel.noManualReadingYet')" />

      <div v-else class="space-y-2">
        <BulkSelectionBar v-if="canEdit" :count="selectedIds.length" :actions="actions" @clear="clear" />
        <SelectionHint v-if="canEdit && manualReadings.length > 0" />
        <SwipeToDelete v-for="r in history" :key="r.key" :disabled="!canEdit || !isManualReading(r) || selectedIds.length > 0" @delete="remove(r)">
        <div
          class="relative hover:border-slate-700/90 card p-3 sm:p-4 flex items-center justify-between gap-3 transition-colors max-sm:select-none max-sm:[-webkit-touch-callout:none]"
          :class="{ '!border-rose-500/40 bg-slate-800/40': isSelected(r) }"
        >
          <CardOpenButton
            v-if="canEdit && isManualReading(r)"
            :label="$t('manual.odometerReadingsPanel.editReading', { unit: distanceUnit(), km: formatDistanceValue(r.odometer) })"
            @click="onRowClick(r)"
            @pointerdown="onPointerdown($event, r)"
            @pointermove="press.onPointermove"
            @pointerup="press.onPointerup"
            @pointercancel="press.onPointercancel"
            @contextmenu="press.onContextmenu"
          />
          <RowLeading :icon="r.kind === 'FUEL' ? Fuel : Gauge" :selected="isSelected(r)" />
          <RowSelectBox v-if="showBox && isManualReading(r)" :id="r.key" :checked="isSelected(r)" :label="$t('shell.bulkSelectionBar.selectRow')" @toggle="toggle(r)" />
          <!-- A reading that cannot be selected (a fill-up, one sent by Home Assistant) shows a lock in the column of the checkbox -->
          <span
            v-else-if="showBox"
            class="grid w-8 shrink-0 place-items-center text-slate-500 max-sm:hidden"
            :title="r.kind === 'FUEL' ? $t('manual.odometerReadingsPanel.fromFillUpHint') : r.source === 'HA' ? $t('manual.odometerReadingsPanel.fromHomeAssistantHint') : undefined"
            aria-hidden="true"
          >
            <Lock class="size-3.5" />
          </span>
          <div class="min-w-0 flex-1">
            <div class="text-sm font-semibold text-white flex flex-wrap items-center gap-2">
              <span class="tabular-nums">{{ formatDistance(r.odometer) }}</span>
              <span v-if="r.kind === 'FUEL'" class="text-xs px-2 py-0.5 rounded-full border border-slate-600 text-slate-400 font-normal" :title="$t('manual.odometerReadingsPanel.fromFillUpHint')">{{ $t('manual.odometerReadingsPanel.fromFillUp') }}</span>
              <span v-else-if="r.source === 'HA'" class="text-xs px-2 py-0.5 rounded-full border border-sky-500/30 bg-sky-500/10 text-sky-300 font-normal" :title="$t('manual.odometerReadingsPanel.fromHomeAssistantHint')">{{ $t('manual.odometerReadingsPanel.fromHomeAssistant') }}</span>
            </div>
            <div class="text-xs text-slate-400 mt-0.5">
              {{ new Date(r.date).toLocaleDateString(intlLocale(), { day: 'numeric', month: 'short', year: 'numeric' }) }}
            </div>
            <p v-if="r.notes" class="text-xs text-slate-400 truncate mt-0.5">{{ r.notes }}</p>
          </div>

          <router-link v-if="r.kind === 'FUEL'" to="/consumption?tab=FUEL" class="relative z-10 tap p-1.5 text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-colors shrink-0" :aria-label="$t('manual.odometerReadingsPanel.seeFillUps')" :title="$t('manual.odometerReadingsPanel.seeFillUps')">
            <Fuel class="w-4 h-4" />
          </router-link>
          <RowMenu v-else-if="canEdit" :label="$t('common.moreActions')" :heading="new Date(r.date).toLocaleDateString(intlLocale(), { day: 'numeric', month: 'short', year: 'numeric' })">
            <button type="button" role="menuitem" class="menu-item menu-item-danger" :aria-label="$t('manual.odometerReadingsPanel.deleteReading', { unit: distanceUnit(), km: formatDistanceValue(r.odometer) })" @click="remove(r)">
              <Trash2 class="w-3.5 h-3.5 text-slate-400" />{{ $t('common.delete') }}
            </button>
          </RowMenu>
        </div>
        </SwipeToDelete>
      </div>
    </div>
  </div>
</template>
