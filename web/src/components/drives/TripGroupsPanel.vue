<script setup lang="ts">
import CardOpenButton from '@/components/CardOpenButton.vue'
import QuietLine from '@/components/QuietLine.vue'
import ListItem from '@/components/ListItem.vue'
import RowLeading from '@/components/RowLeading.vue'
import DriveLegRow from '@/components/drives/DriveLegRow.vue'
import EmptyState from '@/components/EmptyState.vue'
import BulkSelectionBar from '@/components/BulkSelectionBar.vue'
import RowSelectBox from '@/components/RowSelectBox.vue'
import SelectionHint from '@/components/SelectionHint.vue'
import { useRowSelection } from '@/composables/useRowSelection'
import type { SelectionAction } from '@/utils/selectionActions'
import { computed } from 'vue'
import { t } from '@/i18n'
import { distanceUnit, formatDistance, perDistance } from '@/units'
import { useVehicleStore } from '@/stores/vehicle'
import { Layers, Users, ChevronDown, Pencil, Trash2 } from '@/icons'
import { formatTripDates } from '@/utils/drives'
import { formatDayTime } from '@/utils/dates'
import { formatAmount } from '@/currency'

// The trip groups ("voyages") list; a trip can be expanded to show its legs, which open their own cost detail. A click on the row opens its cost detail,
// where the edit, the carpool and the delete are; the selection bar offers the same three for one selected trip.
const props = defineProps<{ loadingTrips: boolean; tripGroups: any[]; hasFilters: boolean; expandedTripId: string | null; tripDrives: any[] }>()
const emit = defineEmits<{
  'open-cost': [trip: any]
  'toggle-details': [trip: any]
  edit: [trip: any]
  delete: [trip: any]
  'open-leg': [leg: any]
  carpool: [trip: any]
}>()
const vehicleStore = useVehicleStore()
const formatDate = formatDayTime

// Edit, carpool and delete work on one trip at a time: with several selected they stay visible but disabled
const { selected, selectedIds, clear, isSelected, toggle, showBox, press, onPointerdown, onRowClick } = useRowSelection(
  computed(() => props.tripGroups),
  computed(() => vehicleStore.canEdit),
  (tg) => emit('open-cost', tg),
)
const actions = computed<SelectionAction[]>(() => {
  const one = selected.value.length === 1 ? selected.value[0] : null
  const single = { disabled: !one, reason: one ? undefined : t('shell.bulkSelectionBar.selectOne') }
  return [
    { key: 'carpool', label: t('drives.tripGroupsPanel.carpool'), icon: Users, run: () => one && emit('carpool', one), ...single },
    { key: 'edit', label: t('drives.tripGroupsPanel.renameTheTrip'), icon: Pencil, run: () => one && emit('edit', one), ...single },
    { key: 'delete', label: t('drives.tripGroupsPanel.deleteTheTrip'), icon: Trash2, danger: true, run: () => one && emit('delete', one), ...single },
  ]
})
</script>

<template>
  <div class="space-y-3">
    <div v-if="loadingTrips" class="text-center py-12 text-slate-400">{{ $t('common.loading') }}</div>
    <EmptyState
      v-else-if="!tripGroups.length"
      :icon="Layers"
      :title="hasFilters ? $t('drives.tripGroupsPanel.noTripMatchesTheFilters') : $t('drives.tripGroupsPanel.noTripsSelectSeveralDrives')"
    >
      <template v-if="!hasFilters">{{ $t('drives.tripGroupsPanel.noTripsHint') }}</template>
    </EmptyState>
    <template v-else>
    <BulkSelectionBar v-if="vehicleStore.canEdit" :count="selectedIds.length" :actions="actions" @clear="clear" />
    <SelectionHint v-if="vehicleStore.canEdit" />
    <template
      v-for="tg in tripGroups"
      :key="tg.id"
    >
      <ListItem wrap-title class="max-sm:select-none max-sm:[-webkit-touch-callout:none]" :class="{ '!border-rose-500/40 bg-slate-800/40': isSelected(tg) }">
        <template #open>
          <CardOpenButton
            :label="$t('common.openItem', { item: tg.name })"
            @click="onRowClick(tg)"
            @pointerdown="onPointerdown($event, tg)"
            @pointermove="press.onPointermove"
            @pointerup="press.onPointerup"
            @pointercancel="press.onPointercancel"
            @contextmenu="press.onContextmenu"
          />
        </template>
        <template #lead>
          <RowLeading :icon="Layers" :selected="isSelected(tg)" />
          <RowSelectBox v-if="showBox" class="max-sm:sr-only max-sm:pointer-events-none" :id="tg.id" :checked="isSelected(tg)" :label="$t('shell.bulkSelectionBar.selectRow')" @toggle="toggle(tg)" />
        </template>
        <template #title><span :title="tg.name">{{ tg.name }}</span></template>
        <!-- The quiet line: when, how far, how many legs and carpools, and the legs toggle -->
        <template #subtitle>
          <QuietLine>
            <span><span>{{ formatTripDates(tg) }}</span></span>
            <span><span>{{ formatDistance(tg.distance_km) }}</span><span>{{ $t('drives.tripGroupsPanel.legS', { length: tg.drive_ids?.length || 0 }) }}</span></span>
            <span data-plain>
              <span v-if="tg.carpool_count">{{ $t('drives.tripGroupsPanel.carpoolS', { carpool_count: tg.carpool_count }) }}</span>
              <button
                type="button"
                @click="emit('toggle-details', tg)"
                class="relative z-10 inline-flex items-center gap-1 font-semibold text-sky-400 hover:text-sky-300"
                :aria-expanded="expandedTripId === tg.id"
              >
                {{ expandedTripId === tg.id ? $t('drives.tripGroupsPanel.hide') : $t('drives.tripGroupsPanel.legs') }}
                <ChevronDown class="w-3.5 h-3.5 transition-transform" :class="{ 'rotate-180': expandedTripId === tg.id }" />
              </button>
            </span>
          </QuietLine>
        </template>
        <template #value>
          <span :title="$t('drives.tripGroupsPanel.consolidatedCostOfTheTrip')">
            {{ Number(tg.tolls_total || 0) > 0 ? $t('drives.tripGroupsPanel.costsAmount', { amount: formatAmount(Number(tg.tolls_total), vehicleStore.currency) }) : $t('drives.tripGroupsPanel.costDetail') }}
          </span>
        </template>
        <template v-if="tg.distance_km > 0 && tg.tolls_total" #subvalue>
          {{ formatAmount(perDistance(Number(tg.tolls_total) / tg.distance_km), vehicleStore.currency, 3) }}/{{ distanceUnit() }}
        </template>
      </ListItem>
      <!-- The legs of the trip: a list indented under its row, one flat row per leg, each opening the cost breakdown of the leg -->
      <div v-if="expandedTripId === tg.id" class="ml-5 mt-1.5 space-y-1.5 border-l-2 border-slate-800 pl-2" data-testid="trip-legs">
        <DriveLegRow v-for="d in tripDrives" :key="d.id" :leg="d" @open="emit('open-leg', $event)" />
        <p v-if="!tripDrives.length" class="px-3 py-2 text-xs text-slate-400">{{ $t('drives.tripGroupsPanel.loadingTheDrives') }}</p>
      </div>
    </template>
    </template>
  </div>
</template>
