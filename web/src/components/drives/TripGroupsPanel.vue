<script setup lang="ts">
import CardOpenButton from '@/components/CardOpenButton.vue'
import RowMenu from '@/components/RowMenu.vue'
import EmptyState from '@/components/EmptyState.vue'
import { intlLocale } from '@/i18n'
import { distanceUnit, formatDistance, perDistance } from '@/units'
import { useVehicleStore } from '@/stores/vehicle'
import { Layers, X, Users, ChevronDown, Pencil, Trash2 } from 'lucide-vue-next'
import { formatTripDates } from '@/utils/drives'
import { formatDayTime } from '@/utils/dates'
import { formatAmount } from '@/currency'

// The trip groups ("voyages") list; a trip can be expanded to show its drives.
defineProps<{ loadingTrips: boolean; tripGroups: any[]; hasFilters: boolean; expandedTripId: string | null; tripDrives: any[] }>()
const emit = defineEmits<{
  'open-cost': [trip: any]
  'toggle-details': [trip: any]
  edit: [trip: any]
  delete: [trip: any]
  'remove-drive': [trip: any, driveId: string]
  carpool: [trip: any]
}>()
const vehicleStore = useVehicleStore()
const formatDate = formatDayTime
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
    <template
      v-for="tg in tripGroups"
      :key="tg.id"
    >
      <div
        class="relative bg-slate-900 border border-slate-800 hover:border-slate-700/90 p-3 sm:p-4 rounded-2xl transition-all flex items-start gap-3 group"
      >
        <CardOpenButton :label="$t('common.openItem', { item: tg.name })" @click="emit('open-cost', tg)" />
        <div class="min-w-0 flex-1">
          <!-- What matters: which trip, and what it cost -->
          <div class="flex items-start justify-between gap-3">
            <div class="text-sm text-white flex items-center gap-1.5 min-w-0">
              <Layers class="w-3.5 h-3.5 text-sky-400 shrink-0" />
              <span class="font-semibold truncate" :title="tg.name">{{ tg.name }}</span>
            </div>
            <div class="shrink-0 text-right" :title="$t('drives.tripGroupsPanel.consolidatedCostOfTheTrip')">
              <div class="text-sm font-extrabold text-white">
                {{ Number(tg.tolls_total || 0) > 0 ? $t('drives.tripGroupsPanel.costsAmount', { amount: formatAmount(Number(tg.tolls_total), vehicleStore.currency) }) : $t('drives.tripGroupsPanel.costDetail') }}
              </div>
              <div v-if="tg.distance_km > 0 && tg.tolls_total" class="text-xs font-mono text-slate-400">
                {{ formatAmount(perDistance(Number(tg.tolls_total) / tg.distance_km), vehicleStore.currency, 3) }}/{{ distanceUnit() }}
              </div>
            </div>
          </div>

          <!-- The rest, in one quiet line: when, how far, how many legs, carpools, and the legs toggle -->
          <div class="mt-1 flex items-center gap-x-3 gap-y-1 flex-wrap text-xs text-slate-400">
            <span class="font-semibold">{{ formatTripDates(tg) }}</span>
            <span class="px-2 py-0.5 rounded-full font-bold bg-slate-800 text-slate-200 border border-slate-700/60">{{ formatDistance(tg.distance_km) }}</span>
            <span class="px-2 py-0.5 rounded-full font-semibold bg-sky-500/10 text-sky-300 border border-sky-500/30">{{ $t('drives.tripGroupsPanel.legS', { length: tg.drive_ids?.length || 0 }) }}</span>
            <span v-if="tg.carpool_count" class="px-2 py-0.5 rounded-full font-semibold bg-sky-500/15 text-sky-300 border border-sky-500/30 flex items-center gap-1">
              <Users class="w-3 h-3" />{{ $t('drives.tripGroupsPanel.carpoolS', { carpool_count: tg.carpool_count }) }}
            </span>
            <span v-if="tg.notes" class="truncate max-w-full">{{ tg.notes }}</span>
            <button
              type="button"
              @click="emit('toggle-details', tg)"
              class="relative z-10 flex items-center gap-1 font-semibold text-sky-400 hover:text-sky-300"
              :aria-expanded="expandedTripId === tg.id"
            >
              {{ expandedTripId === tg.id ? $t('drives.tripGroupsPanel.hide') : $t('drives.tripGroupsPanel.legs') }}
              <ChevronDown class="w-3.5 h-3.5 transition-transform" :class="{ 'rotate-180': expandedTripId === tg.id }" />
            </button>
          </div>
        </div>

        <!-- The row opens the cost breakdown; the other actions sit in its menu -->
        <RowMenu v-if="vehicleStore.canEdit" :label="$t('common.moreActions')" :heading="tg.name">
          <button type="button" role="menuitem" class="menu-item" :title="$t('drives.tripGroupsPanel.carpoolThisTrip')" @click="emit('carpool', tg)">
            <Users class="w-3.5 h-3.5 text-slate-400" />{{ $t('drives.tripGroupsPanel.carpool') }}
          </button>
          <button type="button" role="menuitem" class="menu-item" @click="emit('edit', tg)">
            <Pencil class="w-3.5 h-3.5 text-slate-400" />{{ $t('drives.tripGroupsPanel.renameTheTrip') }}
          </button>
          <button type="button" role="menuitem" class="menu-item hover:!text-rose-300" @click="emit('delete', tg)">
            <Trash2 class="w-3.5 h-3.5 text-slate-400" />{{ $t('drives.tripGroupsPanel.deleteTheTrip') }}
          </button>
        </RowMenu>
      </div>
      <!-- Expanded drives drawer -->
      <div v-if="expandedTripId === tg.id" class="space-y-1.5 bg-slate-950/40 border border-slate-800/80 rounded-2xl p-3 -mt-1 ml-4 mr-4">
        <div v-for="d in tripDrives" :key="d.id" class="flex items-center justify-between gap-3 text-xs text-slate-300 bg-slate-800/40 rounded-lg px-2.5 py-1.5 min-w-0">
          <span class="truncate min-w-0 flex-1">
            {{ formatDate(d.start_time) }}{{ $t('drives.tripGroupsPanel.dateSeparator') }}{{ (d.start_address || $t('drives.driveCostModal.start')).split(',')[0] }} → {{ (d.end_address || $t('drives.driveCostModal.end')).split(',')[0] }}
            <span class="text-slate-400">({{ formatDistance(d.distance_km, 1) }})</span>
          </span>
          <button v-if="vehicleStore.canEdit" @click="emit('remove-drive', tg, d.id)" class="tap text-slate-400 hover:text-danger-400 shrink-0 p-1" :title="$t('drives.tripGroupsPanel.removeThisDriveFromThe')" :aria-label="$t('drives.tripGroupsPanel.removeThisDriveFromThe')">
            <X class="w-3.5 h-3.5" />
          </button>
        </div>
        <p v-if="!tripDrives.length" class="text-xs text-slate-400">{{ $t('drives.tripGroupsPanel.loadingTheDrives') }}</p>
      </div>
    </template>
    </template>
  </div>
</template>
