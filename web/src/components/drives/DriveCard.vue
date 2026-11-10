<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useVehicleStore } from '@/stores/vehicle'
import { usePreferencesStore } from '@/stores/preferences'
import { Navigation, Users, Briefcase, User, Pencil, Trash2, ExternalLink } from '@/icons'
import CardOpenButton from '@/components/CardOpenButton.vue'
import ListItem from '@/components/ListItem.vue'
import RowLeading from '@/components/RowLeading.vue'
import RowMenu from '@/components/RowMenu.vue'
import { useIsMobile } from '@/composables/useIsMobile'
import { useLongPress } from '@/composables/useLongPress'
import QualifyActions from '@/components/drives/QualifyActions.vue'
import { hasTeslaMate } from '@/utils/vehicles'
import { needsTollQualification, summarizeDriveCarpools, teslamateDriveUrl } from '@/utils/drives'
import { formatDayTime, formatTimeOfDay } from '@/utils/dates'
import { formatAmount } from '@/currency'
import { distanceUnit, formatDistance, formatPerDistanceValue } from '@/units'
import { formatCostPerDistance } from '@/utils/costPerDistance'

// One drive of the list: the glyph of the drive, where it went and when, what it cost. A click anywhere on the row opens
// its cost breakdown; the carpool chip and the menu sit above that target.
// On a phone there are no checkboxes at rest: a long press selects the drive and starts the selection (`selecting`), where
// the boxes show and a tap on a row toggles it instead of opening it.
const props = defineProps<{ d: any; selected: boolean; selecting?: boolean }>()
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
const { t } = useI18n()
const vehicleStore = useVehicleStore()
const prefs = usePreferencesStore()
const formatDate = formatDayTime
const isMobile = useIsMobile()
const showBox = computed(() => vehicleStore.canEdit && (!isMobile.value || !!props.selecting))
const press = useLongPress(() => {
  if (vehicleStore.canEdit && isMobile.value) emit('toggle', props.d)
})
function onRowClick() {
  if (isMobile.value && props.selecting) emit('toggle', props.d)
  else emit('open', props.d)
}
// A vehicle fed by TeslaMate mixes synced and manual drives: the badge tells them apart. Without TeslaMate every drive is manual.
const showManualBadge = computed(() => hasTeslaMate(vehicleStore.activeVehicle))
const carpoolSummary = computed(() => summarizeDriveCarpools(props.d.carpools))
// The glyph says what is particular to the drive (carpooled, work, personal); an ordinary drive keeps the plain trip glyph.
const leadingIcon = computed(() => {
  if (carpoolSummary.value) return Users
  if (prefs.proPersoEnabled && props.d.tags?.includes('Pro')) return Briefcase
  if (prefs.proPersoEnabled && props.d.tags?.includes('Perso')) return User
  return Navigation
})

// One carpool opens it directly; several open the drive's cost breakdown, which lists them
function openCarpool() {
  const refs = props.d.carpools || []
  if (refs.length === 1) router.push({ path: '/carpools', query: { open: refs[0].id } })
  else emit('open', props.d)
}
// The quiet line under the route: time, distance, duration, consumption and the tags that apply, as plain text
const subtitle = computed(() => {
  const d = props.d
  const parts = [formatTimeOfDay(d.start_time), formatDistance(d.distance_km, 1)]
  if (d.duration_min) parts.push(t('drives.driveCard.min', { duration_min: d.duration_min }))
  if (d.consumption_kwh_100km) {
    parts.push(
      t(d.energy_estimated ? 'drives.driveCard.kwh100kmEstimated' : 'drives.driveCard.kwh100km', {
        unit: distanceUnit(),
        consumption_kwh_100km: formatPerDistanceValue(d.consumption_kwh_100km),
      }),
    )
  }
  if (prefs.proPersoEnabled && d.tags?.includes('Pro')) parts.push(t('drives.driveCard.work'))
  if (prefs.proPersoEnabled && d.tags?.includes('Perso')) parts.push(t('drives.driveCard.personal'))
  if (d.is_manual && showManualBadge.value) parts.push(t('drives.drivesView.manual'))
  return parts.join(' · ')
})
</script>

<template>
  <div data-testid="drive-card" class="max-sm:select-none max-sm:[-webkit-touch-callout:none]">
    <ListItem :class="{ 'border-rose-500/40 bg-slate-800/40 shadow-lg shadow-rose-950/20': selected }">
      <template #open>
        <CardOpenButton
          :label="$t('drives.driveCard.openBreakdown', { date: formatDate(d.start_time) })"
          @click="onRowClick"
          @pointerdown="press.onPointerdown"
          @pointermove="press.onPointermove"
          @pointerup="press.onPointerup"
          @pointercancel="press.onPointercancel"
          @contextmenu="press.onContextmenu"
        />
      </template>
      <template #lead>
        <!-- Phone: the glyph of the row, the check once it is selected -->
        <RowLeading :icon="leadingIcon" :selected="selected" />
        <!-- Desktop: the selection box (a phone keeps it for screen readers only) -->
        <label
          v-if="showBox"
          :for="'drive-select-' + d.id"
          class="relative z-10 tap flex cursor-pointer items-center p-1 max-sm:sr-only max-sm:pointer-events-none"
          :title="$t('drives.driveCard.selectThisDrive')"
        >
          <span class="sr-only">{{ $t('drives.driveCard.selectThisDrive') }}</span>
          <input :id="'drive-select-' + d.id" type="checkbox" :checked="selected" @change="emit('toggle', d)" class="select-box" />
        </label>
      </template>
      <template #title>
        <span :title="d.start_address">{{ d.start_address || $t('drives.driveCard.unknownStart') }}</span>
        <span class="text-slate-400"> → </span>
        <span :title="d.end_address">{{ d.end_address || $t('drives.driveCard.unknownEnd') }}</span>
      </template>
      <template #subtitle>{{ subtitle }}</template>
      <template v-if="carpoolSummary || vehicleStore.canEdit" #meta>
        <div class="flex items-center gap-1">
          <button
            v-if="carpoolSummary"
            type="button"
            @click="openCarpool"
            class="relative z-10 flex items-center gap-1 rounded-full border border-sky-500/30 bg-sky-500/15 px-2 py-0.5 text-xs font-semibold text-sky-300 transition-colors hover:bg-sky-500/25"
            :title="$t('drives.driveCard.carpooledOpen')"
          >
            <Users class="size-3" />
            {{ $t('drives.driveCard.carpooledSummary', { passengers: carpoolSummary.passengers }, carpoolSummary.passengers) }}
            <span class="text-success-400">+{{ formatAmount(carpoolSummary.revenue, vehicleStore.currency) }}</span>
          </button>
          <!-- The row opens the breakdown; the other actions sit in its menu -->
          <RowMenu v-if="vehicleStore.canEdit" :label="$t('common.moreActions')" :heading="formatDate(d.start_time)">
            <a
              v-if="teslamateDriveUrl(vehicleStore.activeVehicle, d)"
              :href="teslamateDriveUrl(vehicleStore.activeVehicle, d)!"
              target="_blank"
              rel="noopener noreferrer"
              role="menuitem"
              class="menu-item"
            >
              <ExternalLink class="w-3.5 h-3.5 text-slate-400" />{{ $t('expenses.teslamateLinks.details') }}
            </a>
            <button type="button" role="menuitem" class="menu-item" @click="emit('carpool', d)">
              <Users class="w-3.5 h-3.5 text-slate-400" />{{ $t('drives.driveCard.carpool') }}
            </button>
            <template v-if="d.is_manual">
              <button type="button" role="menuitem" class="menu-item" @click="emit('edit', d)">
                <Pencil class="w-3.5 h-3.5 text-slate-400" />{{ $t('common.edit') }}
              </button>
              <button type="button" role="menuitem" class="menu-item menu-item-danger" @click="emit('delete', d)">
                <Trash2 class="w-3.5 h-3.5 text-slate-400" />{{ $t('common.delete') }}
              </button>
            </template>
          </RowMenu>
        </div>
      </template>
      <template #value>
        <span :title="$t('drives.driveCard.actualCostPriceCalculatedFor')">{{ d.costs?.has_estimates ? '~' : '' }}{{ formatAmount(d.costs?.total_cost || 0, vehicleStore.currency) }}</span>
      </template>
      <template #subvalue>{{ formatCostPerDistance(d.costs?.cost_per_km, vehicleStore.currency, 3, true) }}</template>
    </ListItem>

    <!-- Toll qualification: 2 taps, only while the drive waits for it -->
    <QualifyActions
      v-if="vehicleStore.canEdit && needsTollQualification(d)"
      class="relative z-10 mt-2 ml-3 w-fit"
      :primary-label="$t('drives.driveCard.toll')"
      :primary-title="$t('drives.driveCard.enterTheTollOfThis')"
      :secondary-label="$t('drives.driveCard.noToll')"
      :secondary-title="$t('drives.driveCard.confirmThatThisDriveHas')"
      @primary="emit('toll-entry', d)"
      @secondary="emit('no-toll', d)"
    />
  </div>
</template>
