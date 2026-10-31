<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useVehicleStore } from '@/stores/vehicle'
import { usePreferencesStore } from '@/stores/preferences'
import { MapPin, Clock, Users, Pencil, Trash2 } from 'lucide-vue-next'
import CardOpenButton from '@/components/CardOpenButton.vue'
import RowMenu from '@/components/RowMenu.vue'
import QualifyActions from '@/components/drives/QualifyActions.vue'
import { hasTeslaMate } from '@/utils/vehicles'
import { needsTollQualification, summarizeDriveCarpools } from '@/utils/drives'
import { formatDayTime } from '@/utils/dates'
import { formatAmount } from '@/currency'
import { distanceUnit, formatDistance, formatPerDistanceValue } from '@/units'
import { formatCostPerDistance } from '@/utils/costPerDistance'

// One drive of the list: click opens its cost breakdown.
const props = defineProps<{ d: any; selected: boolean }>()
const emit = defineEmits<{
  open: [drive: any]
  toggle: [drive: any]
  'toll-entry': [drive: any]
  'no-toll': [drive: any]
  edit: [drive: any]
  delete: [drive: any]
  carpool: [drive: any]
}>()
const router = useRouter()
const vehicleStore = useVehicleStore()
const prefs = usePreferencesStore()
const formatDate = formatDayTime
// A vehicle fed by TeslaMate mixes synced and manual drives: the badge tells them apart. Without TeslaMate every drive is manual.
const showManualBadge = computed(() => hasTeslaMate(vehicleStore.activeVehicle))
const carpoolSummary = computed(() => summarizeDriveCarpools(props.d.carpools))

// One carpool opens it directly; several open the drive's cost breakdown, which lists them
function openCarpool() {
  const refs = props.d.carpools || []
  if (refs.length === 1) router.push({ path: '/carpools', query: { open: refs[0].id } })
  else emit('open', props.d)
}
</script>

<template>
  <div
    class="relative bg-slate-900 border border-slate-800 hover:border-slate-700/90 p-3 sm:p-4 rounded-2xl transition-all flex items-start gap-3 group"
    :class="{ 'border-rose-500/40 bg-slate-800/40 shadow-lg shadow-rose-950/20': selected }"
  >
    <CardOpenButton :label="$t('drives.driveCard.openBreakdown', { date: formatDate(d.start_time) })" @click="emit('open', d)" />
    <!-- Selection checkbox -->
    <label
      v-if="vehicleStore.canEdit"
      :for="'drive-select-' + d.id"
      class="relative z-10 tap mt-0.5 -ml-1 p-1 shrink-0 flex items-center cursor-pointer"
      :title="$t('drives.driveCard.selectThisDrive')"
    >
      <span class="sr-only">{{ $t('drives.driveCard.selectThisDrive') }}</span>
      <input :id="'drive-select-' + d.id" type="checkbox" :checked="selected" @change="emit('toggle', d)" class="select-box" />
    </label>

    <div class="min-w-0 flex-1">
      <!-- What matters: where, and what it cost -->
      <div class="flex items-start justify-between gap-3">
        <div class="text-sm text-white flex items-center gap-1.5 flex-wrap min-w-0">
          <MapPin class="w-3.5 h-3.5 text-rose-400 shrink-0" />
          <span class="truncate max-w-[140px] sm:max-w-[220px] md:max-w-xs font-semibold" :title="d.start_address">{{ d.start_address || $t('drives.driveCard.unknownStart') }}</span>
          <span class="text-slate-400 shrink-0">→</span>
          <span class="truncate max-w-[140px] sm:max-w-[220px] md:max-w-xs font-semibold" :title="d.end_address">{{ d.end_address || $t('drives.driveCard.unknownEnd') }}</span>
        </div>
        <div class="shrink-0 text-right" :title="$t('drives.driveCard.actualCostPriceCalculatedFor')">
          <div class="text-sm font-extrabold text-white">{{ d.costs?.has_estimates ? '~' : '' }}{{ formatAmount(d.costs?.total_cost || 0, vehicleStore.currency) }}</div>
          <div class="text-xs font-mono text-slate-400">{{ formatCostPerDistance(d.costs?.cost_per_km, vehicleStore.currency, 3, true) }}</div>
        </div>
      </div>

      <!-- The rest, in one quiet line: when, how far, how long, how much energy, and the tags that apply -->
      <div class="mt-1 flex items-center gap-x-3 gap-y-1 flex-wrap text-xs text-slate-400">
        <span class="font-semibold">{{ formatDate(d.start_time) }}</span>
        <span class="px-2 py-0.5 rounded-full font-bold bg-slate-800 text-slate-200 border border-slate-700/60">{{ formatDistance(d.distance_km, 1) }}</span>
        <span v-if="d.duration_min" class="flex items-center gap-1"><Clock class="w-3 h-3" />{{ $t('drives.driveCard.min', { duration_min: d.duration_min }) }}</span>
        <span
          v-if="d.consumption_kwh_100km"
          class="font-mono"
          :class="d.energy_estimated ? 'text-slate-400' : 'text-info-400'"
          :title="d.energy_estimated ? $t('drives.driveCard.estimatedConsumption') : undefined"
        >
          {{ $t(d.energy_estimated ? 'drives.driveCard.kwh100kmEstimated' : 'drives.driveCard.kwh100km', { unit: distanceUnit(), consumption_kwh_100km: formatPerDistanceValue(d.consumption_kwh_100km) }) }}
        </span>
        <span
          v-if="prefs.proPersoEnabled && d.tags?.includes('Pro')"
          class="px-2 py-0.5 rounded-full font-semibold bg-blue-500/15 text-blue-300 border border-blue-500/30"
        >{{ $t('drives.driveCard.work') }}</span>
        <span
          v-if="prefs.proPersoEnabled && d.tags?.includes('Perso')"
          class="px-2 py-0.5 rounded-full font-semibold bg-success-500/15 text-success-300 border border-success-500/30"
        >{{ $t('drives.driveCard.personal') }}</span>
        <span v-if="d.is_manual && showManualBadge" class="px-2 py-0.5 rounded-full font-semibold bg-slate-800 text-slate-300 border border-slate-700/60">{{ $t('drives.drivesView.manual') }}</span>
        <button
          v-if="carpoolSummary"
          type="button"
          @click="openCarpool"
          class="relative z-10 px-2 py-0.5 rounded-full font-semibold bg-sky-500/15 text-sky-300 border border-sky-500/30 hover:bg-sky-500/25 transition-colors flex items-center gap-1"
          :title="$t('drives.driveCard.carpooledOpen')"
        >
          <Users class="w-3 h-3" />
          {{ $t('drives.driveCard.carpooledSummary', { passengers: carpoolSummary.passengers }, carpoolSummary.passengers) }}
          <span class="font-mono text-success-400">+{{ formatAmount(carpoolSummary.revenue, vehicleStore.currency) }}</span>
        </button>
      </div>

      <!-- Toll qualification: 2 taps, only while the drive waits for it -->
      <QualifyActions
        v-if="vehicleStore.canEdit && needsTollQualification(d)"
        class="relative z-10 mt-2 w-fit"
        :primary-label="$t('drives.driveCard.toll')"
        :primary-title="$t('drives.driveCard.enterTheTollOfThis')"
        :secondary-label="$t('drives.driveCard.noToll')"
        :secondary-title="$t('drives.driveCard.confirmThatThisDriveHas')"
        @primary="emit('toll-entry', d)"
        @secondary="emit('no-toll', d)"
      />
    </div>

    <!-- The row opens the breakdown; the other actions sit in its menu -->
    <RowMenu v-if="vehicleStore.canEdit" :label="$t('common.moreActions')">
      <button type="button" role="menuitem" class="menu-item" @click="emit('carpool', d)">
        <Users class="w-3.5 h-3.5 text-slate-400" />{{ $t('drives.driveCard.carpool') }}
      </button>
      <template v-if="d.is_manual">
        <button type="button" role="menuitem" class="menu-item" @click="emit('edit', d)">
          <Pencil class="w-3.5 h-3.5 text-slate-400" />{{ $t('common.edit') }}
        </button>
        <button type="button" role="menuitem" class="menu-item hover:!text-rose-300" @click="emit('delete', d)">
          <Trash2 class="w-3.5 h-3.5 text-slate-400" />{{ $t('common.delete') }}
        </button>
      </template>
    </RowMenu>
  </div>
</template>
