<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import { t } from '@/i18n'
import type { VehiclePerson } from '@/services/api'
import { defaultDriver } from '@/utils/vehicles'
import { usePreferencesStore } from '@/stores/preferences'
import BulkSelectionBar from '@/components/BulkSelectionBar.vue'
import ModalShell from '@/components/ModalShell.vue'
import { Receipt, Layers, Users, Plus, Download, Pencil, Trash2, ExternalLink, ListChecks, Briefcase, House, UserRound } from 'lucide-vue-next'
import type { SelectionAction } from '@/utils/selectionActions'
import { teslamateDriveUrl } from '@/utils/drives'

// Actions on the drives selected in the list; the page runs them.
const props = defineProps<{ selectedDriveIds: string[]; selectedDrives: any[]; selectedOffPage: number; allPageSelected: boolean; selectedSummaryMetrics: string; bulkApplyingToll: boolean; people: VehiclePerson[]; tollDetectionActive: boolean }>()
const emit = defineEmits<{
  clear: []
  carpool: []
  group: []
  'bulk-toll': []
  tag: [tag: 'Pro' | 'Perso']
  driver: [driverId: string | null]
  export: []
  'add-to-trip': []
  'select-page': []
  edit: [drive: any]
  delete: [drives: any[]]
}>()
const vehicleStore = useVehicleStore()
const prefs = usePreferencesStore()

// "" is the placeholder; DEFAULT_DRIVER clears the driver so the vehicle's default applies
const DEFAULT_DRIVER = '__default__'
const defaultDriverLabel = computed(() => {
  const d = defaultDriver(props.people, vehicleStore.activeVehicle?.default_driver_id)
  return d ? t('drives.defaultDriver', { name: d.name }) : t('drives.unassignedDriver')
})
function onDriverChange(e: Event) {
  const el = e.target as HTMLSelectElement
  const value = el.value
  el.value = ''
  if (!value) return
  emit('driver', value === DEFAULT_DRIVER ? null : value)
}

// On a phone the driver is picked in its own sheet, opened from the actions sheet
const driverSheetOpen = ref(false)
// The selection can end under the sheet (the Back gesture): the sheet goes with it
watch(
  () => props.selectedDriveIds.length,
  (n) => {
    if (n === 0) driverSheetOpen.value = false
  }
)
function pickDriver(value: string) {
  driverSheetOpen.value = false
  emit('driver', value === DEFAULT_DRIVER ? null : value)
}

// Only the drives entered by hand can be edited or deleted: the ones synced from TeslaMate follow their source
const actions = computed<SelectionAction[]>(() => {
  const drives = props.selectedDrives
  const n = props.selectedDriveIds.length
  const manual = drives.length > 0 && drives.every((d) => d.is_manual)
  const one = n === 1
  // On a phone the page has no select-all row (it would push the list down when the selection starts)
  const list: SelectionAction[] = [
    {
      key: 'select-page',
      label: props.allPageSelected ? t('drives.drivesView.deselectPage') : t('drives.drivesView.selectPage'),
      icon: ListChecks,
      run: () => emit('select-page'),
      phoneOnly: true,
    },
  ]
  list.push({
    key: 'edit',
    label: t('common.edit'),
    icon: Pencil,
    run: () => emit('edit', drives[0]),
    disabled: !one || !manual,
    reason: !one ? t('shell.bulkSelectionBar.selectOne') : !manual ? t('drives.driveBulkActions.syncedCannotEdit') : undefined,
  })
  list.push({
    key: 'carpool',
    label: n > 1 ? t('drives.driveBulkActions.carpoolMany', { count: n }) : t('drives.driveBulkActions.carpool'),
    icon: Users,
    run: () => emit('carpool'),
  })
  list.push({
    key: 'delete',
    label: t('common.delete'),
    icon: Trash2,
    danger: true,
    run: () => emit('delete', drives),
    disabled: !manual,
    reason: manual ? undefined : t('drives.driveBulkActions.syncedCannotDelete'),
  })
  if (props.tollDetectionActive) {
    list.push({
      key: 'toll',
      label: props.bulkApplyingToll ? t('drives.driveBulkActions.applying') : t('drives.driveBulkActions.autoToll', { count: n }),
      icon: Receipt,
      run: () => emit('bulk-toll'),
      busy: props.bulkApplyingToll,
      overflow: true,
    })
  }
  list.push({ key: 'group', label: t('drives.driveBulkActions.mergeToll'), icon: Layers, run: () => emit('group'), overflow: true })
  list.push({ key: 'trip', label: t('drives.driveBulkActions.addToTrip'), icon: Plus, run: () => emit('add-to-trip'), overflow: true })
  if (prefs.proPersoEnabled) {
    list.push({ key: 'work', label: t('drives.driveBulkActions.work'), icon: Briefcase, run: () => emit('tag', 'Pro'), overflow: true })
    list.push({ key: 'personal', label: t('drives.driveBulkActions.personal'), icon: House, run: () => emit('tag', 'Perso'), overflow: true })
  }
  if (props.people.length) {
    list.push({ key: 'driver', label: t('drives.driveBulkActions.changeDriverAction'), icon: UserRound, run: () => (driverSheetOpen.value = true), overflow: true, phoneOnly: true })
  }
  const tmUrl = props.selectedDrives.length === 1 ? teslamateDriveUrl(vehicleStore.activeVehicle, props.selectedDrives[0]) : null
  if (tmUrl) {
    list.push({ key: 'teslamate', label: t('expenses.teslamateLinks.details'), icon: ExternalLink, run: () => window.open(tmUrl, '_blank', 'noopener,noreferrer'), overflow: true })
  }
  list.push({ key: 'export', label: t('drives.driveBulkActions.export'), icon: Download, run: () => emit('export'), overflow: true })
  return list
})
</script>

<template>
  <BulkSelectionBar
    v-if="vehicleStore.canEdit"
    :count="selectedDriveIds.length"
    noun="drive"
    :off-screen-count="selectedOffPage"
    :metrics-summary="selectedSummaryMetrics"
    :actions="actions"
    @clear="emit('clear')"
  >
    <template #more>
      <label v-if="people.length" class="block px-3 pt-1.5 pb-1 text-xs font-semibold text-slate-400">
        {{ $t('drives.driveBulkActions.driver') }}
        <select value="" @change="onDriverChange" class="field mt-1 w-full font-normal" :title="$t('drives.driveBulkActions.changeDriver')">
          <option value="" disabled>{{ $t('drives.driveBulkActions.chooseDriver') }}</option>
          <option :value="DEFAULT_DRIVER">{{ defaultDriverLabel }}</option>
          <option v-for="p in people" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
      </label>
    </template>
  </BulkSelectionBar>

  <!-- Teleported like the actions sheet: the page container does not span the whole viewport -->
  <Teleport to="body">
    <ModalShell v-model:open="driverSheetOpen" :title="$t('drives.driveBulkActions.driver')" :icon="UserRound" icon-class="text-slate-300" size="sm">
      <div role="menu" class="space-y-1 [&_.menu-item]:min-h-12 [&_.menu-item]:gap-4 [&_.menu-item]:px-4 [&_.menu-item]:text-base">
        <button type="button" role="menuitem" class="menu-item" @click="pickDriver(DEFAULT_DRIVER)">{{ defaultDriverLabel }}</button>
        <button v-for="p in people" :key="p.id" type="button" role="menuitem" class="menu-item" @click="pickDriver(p.id)">{{ p.name }}</button>
      </div>
    </ModalShell>
  </Teleport>
</template>
