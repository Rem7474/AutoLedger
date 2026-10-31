<script setup lang="ts">
import CardOpenButton from '@/components/CardOpenButton.vue'
import RowMenu from '@/components/RowMenu.vue'
import { computed } from 'vue'
import { useIsMobile } from '@/composables/useIsMobile'
import { useLongPress } from '@/composables/useLongPress'
import { formatDistance } from '@/units'
import { useVehicleStore } from '@/stores/vehicle'
import BulkSelectionBar from '@/components/BulkSelectionBar.vue'
import SelectAllToggle from '@/components/SelectAllToggle.vue'
import { Users, Trash2, Pencil, Navigation, RotateCw, Download, Link2, Lock } from 'lucide-vue-next'
import { formatDate, hasLinkedLegs } from '@/utils/carpool'
import { formatAmount } from '@/currency'

// The carpool trips as compact cards (the legs, passengers and costs are in the detail, opened by a click), and the
// selection actions (recalculate, export). On a phone the checkboxes are hidden at rest: a long press selects a carpool and
// starts the selection, where the boxes show and a tap toggles the row instead of opening it.
const props = defineProps<{ trips: any[]; selectedTripIds: string[]; recalculating: boolean }>()
const emit = defineEmits<{
  clear: []
  open: [trip: any]
  'batch-recalculate': []
  export: []
  'toggle-all': []
  toggle: [tripId: string]
  recalculate: [trip: any]
  edit: [trip: any]
  delete: [trip: any]
}>()
const vehicleStore = useVehicleStore()
// Carpool amounts are in the vehicle's own currency
const fmt = (v: number) => formatAmount(Number(v || 0), vehicleStore.currency)

// A carpool made from drives follows their cost on its own: only the ones entered by distance are recalculated
const recalculable = computed(() => props.trips.filter((tr) => props.selectedTripIds.includes(tr.id) && !hasLinkedLegs(tr)))
const isMobile = useIsMobile()
const selecting = computed(() => props.selectedTripIds.length > 0)
const showBox = computed(() => vehicleStore.canEdit && (!isMobile.value || selecting.value))
let pressed: any = null
const press = useLongPress(() => {
  if (vehicleStore.canEdit && isMobile.value && pressed) emit('toggle', pressed.id)
})
function onPointerdown(e: PointerEvent, trip: any) {
  pressed = trip
  press.onPointerdown(e)
}
function onRowClick(trip: any) {
  if (isMobile.value && selecting.value) emit('toggle', trip.id)
  else emit('open', trip)
}
const isAllSelected = computed(() => props.trips.length > 0 && props.selectedTripIds.length === props.trips.length)
</script>

<template>
  <div class="space-y-4">
    <!-- Sticky Bulk Selection Bar -->
    <BulkSelectionBar
      v-if="vehicleStore.canEdit"
      :count="selectedTripIds.length"
      noun="carpool"
      @clear="emit('clear')"
    >
      <button
        v-if="recalculable.length"
        type="button"
        @click="emit('batch-recalculate')"
        :disabled="recalculating"
        class="btn btn-primary"
        :title="$t('carpool.carpoolTripList.recalculateTheActualCostsOf2')"
      >
        <RotateCw class="w-3.5 h-3.5" :class="{ 'animate-spin': recalculating }" />
        <span>{{ $t('carpool.carpoolTripList.recalculateActualCosts', { length: recalculable.length }) }}</span>
      </button>

      <button
        type="button"
        @click="emit('export')"
        class="btn btn-secondary"
        :title="$t('carpool.carpoolTripList.exportTheSelectionAsCsv')"
      >
        <Download class="w-3.5 h-3.5 text-slate-300" />
        <span>{{ $t('carpool.carpoolTripList.exportCsv') }}</span>
      </button>
    </BulkSelectionBar>

    <!-- Header row: Select all checkbox & Total info -->
    <div v-if="vehicleStore.canEdit" class="flex items-center justify-between text-xs text-slate-400 px-2">
      <SelectAllToggle
        :checked="isAllSelected"
        :indeterminate="selectedTripIds.length > 0 && !isAllSelected"
        :label="isAllSelected ? $t('common.deselectAll') : $t('common.selectAll')"
        @toggle="emit('toggle-all')"
      />
      <span>{{ $t('carpool.carpoolTripList.carpoolSInTotal', { length: trips.length }) }}</span>
    </div>

    <div
      v-for="trip in trips"
      :key="trip.id"
      class="relative bg-slate-900 border rounded-2xl p-3 sm:p-4 transition-all flex items-start gap-3 max-sm:select-none max-sm:[-webkit-touch-callout:none]"
      :class="selectedTripIds.includes(trip.id) ? 'border-rose-500/50 bg-rose-500/[0.02]' : 'border-slate-800 hover:border-slate-700'"
    >
      <CardOpenButton
        :label="$t('common.openItem', { item: trip.title })"
        @click="onRowClick(trip)"
        @pointerdown="onPointerdown($event, trip)"
        @pointermove="press.onPointermove"
        @pointerup="press.onPointerup"
        @pointercancel="press.onPointercancel"
        @contextmenu="press.onContextmenu"
      />
      <!-- Select Checkbox -->
      <label
        v-if="showBox"
        :for="'carpool-select-' + trip.id"
        class="relative z-10 tap mt-0.5 -ml-1 p-1 shrink-0 flex items-center cursor-pointer"
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

      <div class="min-w-0 flex-1">
        <!-- What matters: which carpool, and what is left to the driver -->
        <div class="flex items-start justify-between gap-3">
          <h3 class="text-sm font-semibold text-white truncate min-w-0" :title="trip.title">{{ trip.title }}</h3>
          <div class="shrink-0 text-right">
            <div v-if="trip.net_cost > 0" class="text-sm font-extrabold text-white">{{ fmt(trip.net_cost) }}</div>
            <div v-else class="text-sm font-extrabold text-success-400">+{{ fmt(Math.abs(trip.net_cost)) }}</div>
            <div class="text-xs text-slate-400">{{ trip.net_cost > 0 ? $t('carpool.carpoolTripList.leftToTheDriverShort') : $t('carpool.carpoolTripList.netSurplusShort') }}</div>
          </div>
        </div>

        <!-- The rest, in one quiet line -->
        <div class="mt-1 flex items-center gap-x-3 gap-y-1 flex-wrap text-xs text-slate-400">
          <span class="font-semibold">{{ formatDate(trip.date) }}</span>
          <span class="px-2 py-0.5 rounded-full font-bold bg-slate-800 text-slate-200 border border-slate-700/60">{{ formatDistance(trip.distance_km, 1) }}</span>
          <span class="flex items-center gap-1"><Navigation class="w-3 h-3 text-sky-400" />{{ $t('carpool.carpoolTripList.legs', { length: trip.legs.length }) }}</span>
          <span class="flex items-center gap-1"><Users class="w-3 h-3 text-sky-400" />{{ $t('carpool.carpoolTripList.passengers', { length: trip.passengers?.length || 0 }) }}</span>
          <span
            class="px-2 py-0.5 rounded-full font-semibold border"
            :class="trip.net_cost <= 0 ? 'bg-success-500/15 text-success-400 border-success-500/30' : 'bg-warning-500/10 text-warning-400 border-warning-500/20'"
          >
            {{ trip.net_cost <= 0 ? $t('carpool.carpoolTripList.fullyRecovered') : $t('carpool.carpoolTripList.recovered', { percent: trip.total_cost > 0 ? Math.min(100, Math.round((trip.total_revenue / trip.total_cost) * 100)) : 0 }) }}
          </span>
          <span
            v-if="hasLinkedLegs(trip)"
            class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full font-medium bg-slate-800 text-slate-300 border border-slate-700/60"
            :title="trip.costs_frozen_at ? $t('carpool.linked.frozenHint') : $t('carpool.linked.followHint')"
          >
            <Lock v-if="trip.costs_frozen_at" class="w-3 h-3" />
            <Link2 v-else class="w-3 h-3" />
            {{ trip.costs_frozen_at ? $t('carpool.linked.frozen') : $t('carpool.linked.linked') }}
          </span>
          <span v-if="trip.notes" class="truncate max-w-full">{{ trip.notes }}</span>
        </div>
      </div>

      <!-- The row opens the detail; the other actions sit in its menu -->
      <RowMenu v-if="vehicleStore.canEdit" :label="$t('common.moreActions')" :heading="trip.title">
        <button v-if="!hasLinkedLegs(trip)" type="button" role="menuitem" class="menu-item" :disabled="recalculating" :title="$t('carpool.carpoolTripList.recalculateTheActualCostsOf')" @click="emit('recalculate', trip)">
          <RotateCw class="w-3.5 h-3.5 text-slate-400" />{{ $t('carpool.carpoolTripList.recalculate') }}
        </button>
        <button type="button" role="menuitem" class="menu-item" @click="emit('edit', trip)">
          <Pencil class="w-3.5 h-3.5 text-slate-400" />{{ $t('common.edit') }}
        </button>
        <button type="button" role="menuitem" class="menu-item hover:!text-rose-300" @click="emit('delete', trip)">
          <Trash2 class="w-3.5 h-3.5 text-slate-400" />{{ $t('common.delete') }}
        </button>
      </RowMenu>
    </div>
  </div>
</template>
