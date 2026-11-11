<script setup lang="ts">
import CardOpenButton from '@/components/CardOpenButton.vue'
import ListItem from '@/components/ListItem.vue'
import StatusLabel from '@/components/StatusLabel.vue'
import { computed } from 'vue'
import { useSelectableRow } from '@/composables/useSelectableRow'
import { formatDistance } from '@/units'
import { useVehicleStore } from '@/stores/vehicle'
import BulkSelectionBar from '@/components/BulkSelectionBar.vue'
import SelectAllToggle from '@/components/SelectAllToggle.vue'
import { Users, RotateCw, Download, Link2, Lock, ListChecks } from '@/icons'
import RowLeading from '@/components/RowLeading.vue'
import SwipeToDelete from '@/components/SwipeToDelete.vue'
import { formatDate, hasLinkedLegs, splitByFreeze } from '@/utils/carpool'
import { editDeleteActions, type SelectionAction } from '@/utils/selectionActions'
import { t } from '@/i18n'
import { formatAmount } from '@/currency'

// The carpool trips as list rows (the notes, legs, passengers and costs are in the detail, opened by a click; its edit, recalculate and delete are there too), and the
// selection actions (edit, delete, freeze or resync the amounts of the carpools made from drives, recalculate, export). On a phone the checkboxes are hidden at rest: a long press selects a carpool and
// starts the selection, where the boxes show and a tap toggles the row instead of opening it.
const props = defineProps<{ trips: any[]; selectedTripIds: string[]; recalculating: boolean; /** How many carpools the vehicle has, when only some pages are loaded. */ total?: number }>()
const emit = defineEmits<{
  clear: []
  open: [trip: any]
  'batch-recalculate': []
  'batch-freeze': [trips: any[]]
  'batch-resync': [trips: any[]]
  export: []
  'toggle-all': []
  toggle: [tripId: string]
  edit: [trip: any]
  delete: [trip: any]
  'delete-many': [trips: any[]]
}>()
const vehicleStore = useVehicleStore()
// Carpool amounts are in the vehicle's own currency
const fmt = (v: number) => formatAmount(Number(v || 0), vehicleStore.currency)

// A carpool made from drives follows their cost on its own: only the ones entered by distance are recalculated
const recalculable = computed(() => props.trips.filter((tr) => props.selectedTripIds.includes(tr.id) && !hasLinkedLegs(tr)))
const selectedTrips = computed(() => props.trips.filter((tr) => props.selectedTripIds.includes(tr.id)))
// The carpools made from drives either follow their cost or hold the amounts as entered (frozen)
const freezeSplit = computed(() => splitByFreeze(selectedTrips.value))
const actions = computed<SelectionAction[]>(() => [
  // On a phone the list has no select-all row (it would push the cards down when the selection starts)
  {
    key: 'select-all',
    label: isAllSelected.value ? t('common.deselectAll') : t('common.selectAll'),
    icon: ListChecks,
    run: () => emit('toggle-all'),
    phoneOnly: true,
  },
  ...editDeleteActions(selectedTrips.value, { edit: (tr) => emit('edit', tr), remove: (list) => emit('delete-many', list) }),
  {
    key: 'freeze',
    label: t('carpool.linked.freezeMany', { count: freezeSplit.value.freezable.length }),
    icon: Lock,
    run: () => emit('batch-freeze', freezeSplit.value.freezable),
    busy: props.recalculating,
    disabled: freezeSplit.value.freezable.length === 0,
    reason: t('carpool.linked.freezeNone'),
    overflow: true,
  },
  {
    key: 'resync',
    label: t('carpool.linked.resyncMany', { count: freezeSplit.value.resyncable.length }),
    icon: Link2,
    run: () => emit('batch-resync', freezeSplit.value.resyncable),
    busy: props.recalculating,
    disabled: freezeSplit.value.resyncable.length === 0,
    reason: t('carpool.linked.resyncNone'),
    overflow: true,
  },
  {
    key: 'recalculate',
    label: t('carpool.carpoolTripList.recalculateActualCosts', { length: recalculable.value.length }),
    icon: RotateCw,
    run: () => emit('batch-recalculate'),
    busy: props.recalculating,
    disabled: recalculable.value.length === 0,
    reason: t('carpool.carpoolTripList.recalculateNone'),
    overflow: true,
  },
  { key: 'export', label: t('carpool.carpoolTripList.exportCsv'), icon: Download, run: () => emit('export'), overflow: true },
])
const { showBox, press, onPointerdown, onRowClick } = useSelectableRow<any>({
  enabled: computed(() => vehicleStore.canEdit),
  selecting: computed(() => props.selectedTripIds.length > 0),
  toggle: (trip) => emit('toggle', trip.id),
  open: (trip) => emit('open', trip),
})
const isAllSelected = computed(() => props.trips.length > 0 && props.selectedTripIds.length === props.trips.length)
</script>

<template>
  <div class="space-y-4">
    <!-- Sticky Bulk Selection Bar -->
    <BulkSelectionBar
      v-if="vehicleStore.canEdit"
      :count="selectedTripIds.length"
      noun="carpool"
      :actions="actions"
      @clear="emit('clear')"
    />

    <!-- Header row: Select all checkbox & Total info -->
    <div v-if="vehicleStore.canEdit" class="flex items-center justify-between text-xs text-slate-400 px-2 max-sm:hidden">
      <SelectAllToggle
        :checked="isAllSelected"
        :indeterminate="selectedTripIds.length > 0 && !isAllSelected"
        :label="isAllSelected ? $t('common.deselectAll') : $t('common.selectAll')"
        @toggle="emit('toggle-all')"
      />
      <span>{{ $t('carpool.carpoolTripList.carpoolSInTotal', { length: total ?? trips.length }) }}</span>
    </div>

    <SwipeToDelete v-for="trip in trips" :key="trip.id" :disabled="!vehicleStore.canEdit || selectedTripIds.length > 0" @delete="emit('delete', trip)">
      <ListItem class="max-sm:select-none max-sm:[-webkit-touch-callout:none]" :class="selectedTripIds.includes(trip.id) ? '!border-rose-500/50 bg-rose-500/[0.02]' : ''">
        <template #open>
          <CardOpenButton
            :label="$t('common.openItem', { item: trip.title })"
            @click="onRowClick(trip)"
            @pointerdown="onPointerdown($event, trip)"
            @pointermove="press.onPointermove"
            @pointerup="press.onPointerup"
            @pointercancel="press.onPointercancel"
            @contextmenu="press.onContextmenu"
          />
        </template>
        <template #lead>
          <RowLeading :icon="Users" :selected="selectedTripIds.includes(trip.id)" />
          <label
            v-if="showBox"
            :for="'carpool-select-' + trip.id"
            class="relative z-10 tap flex cursor-pointer items-center p-1 max-sm:sr-only max-sm:pointer-events-none"
            :title="$t('carpool.carpoolTripList.selectThisCarpool')"
          >
            <span class="sr-only">{{ $t('carpool.carpoolTripList.selectThisCarpool') }}</span>
            <input
              :id="'carpool-select-' + trip.id"
              type="checkbox"
              :checked="selectedTripIds.includes(trip.id)"
              @change="emit('toggle', trip.id)"
              class="select-box"
            />
          </label>
        </template>
        <template #title><span :title="trip.title">{{ trip.title }}</span></template>
        <!-- The quiet line: when, how far, who, and how far the cost is recovered; the amounts as entered or following the drives is a second state in the same line -->
        <template #subtitle>
          <span class="whitespace-normal">
            <span class="whitespace-nowrap">{{ formatDate(trip.date) }}</span>
            <span> · </span><span class="whitespace-nowrap">{{ formatDistance(trip.distance_km, 1) }}</span>
            <span> · </span><span class="whitespace-nowrap">{{ $t('carpool.carpoolTripList.legs', { length: trip.legs.length }) }}</span>
            <span> · </span><span class="whitespace-nowrap">{{ $t('carpool.carpoolTripList.passengers', { length: trip.passengers?.length || 0 }) }}</span>
            <span> · </span>
            <StatusLabel class="align-top [&>.truncate]:whitespace-normal [&>.truncate]:overflow-visible" :tone="trip.net_cost <= 0 ? 'neutral' : 'attention'">
              {{ trip.net_cost <= 0 ? $t('carpool.carpoolTripList.fullyRecovered') : $t('carpool.carpoolTripList.recovered', { percent: trip.total_cost > 0 ? Math.min(100, Math.round((trip.total_revenue / trip.total_cost) * 100)) : 0 }) }}
            </StatusLabel>
            <template v-if="hasLinkedLegs(trip)">
              <span> · </span>
              <span class="whitespace-nowrap" :title="trip.costs_frozen_at ? $t('carpool.linked.frozenHint') : $t('carpool.linked.followHint')">{{ trip.costs_frozen_at ? $t('carpool.linked.frozen') : $t('carpool.linked.linked') }}</span>
            </template>
          </span>
        </template>
        <template #value>
          <span v-if="trip.net_cost > 0">{{ fmt(trip.net_cost) }}</span>
          <span v-else class="text-success-400">+{{ fmt(Math.abs(trip.net_cost)) }}</span>
        </template>
        <template #subvalue>{{ trip.net_cost > 0 ? $t('carpool.carpoolTripList.leftToTheDriverShort') : $t('carpool.carpoolTripList.netSurplusShort') }}</template>
      </ListItem>
    </SwipeToDelete>
  </div>
</template>
