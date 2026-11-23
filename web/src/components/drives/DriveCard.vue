<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useVehicleStore } from '@/stores/vehicle'
import { usePreferencesStore } from '@/stores/preferences'
import CardOpenButton from '@/components/CardOpenButton.vue'
import ListItem from '@/components/ListItem.vue'
import RowLeading from '@/components/RowLeading.vue'
import { useIsMobile } from '@/composables/useIsMobile'
import { useLongPress } from '@/composables/useLongPress'
import DriveCarpoolChip from '@/components/drives/DriveCarpoolChip.vue'
import QualifyActions from '@/components/drives/QualifyActions.vue'
import { driveGlyph } from '@/utils/driveGlyph'
import { needsTollQualification, summarizeDriveCarpools } from '@/utils/drives'
import { formatDayTime, formatTimeOfDay } from '@/utils/dates'
import { formatAmount } from '@/currency'

// One drive of the list in three pieces: where it went, when it started, what it cost. A click anywhere on the row opens
// its detail (distance, duration, consumption, cost per km, edit, delete, carpool, tolls, the TeslaMate link); the carpool
// chip and the toll buttons of a drive that waits for them sit above that target. No box: a long press (finger or held
// mouse button), Ctrl/Cmd/Shift + click or Space starts the selection (`selecting`), where the glyph of a selected row
// becomes the rose check and a click on a row toggles it instead of opening it.
const props = defineProps<{ d: any; selected: boolean; selecting?: boolean }>()
const emit = defineEmits<{
  open: [drive: any]
  toggle: [drive: any]
  'toll-entry': [drive: any]
  'no-toll': [drive: any]
}>()
const router = useRouter()
const { t } = useI18n()
const vehicleStore = useVehicleStore()
const prefs = usePreferencesStore()
const formatDate = formatDayTime
const isMobile = useIsMobile()
const press = useLongPress(
  () => {
    if (vehicleStore.canEdit) emit('toggle', props.d)
  },
  { mouse: true },
)
function onRowClick(e: MouseEvent) {
  const modifier = e.ctrlKey || e.metaKey || e.shiftKey
  if (vehicleStore.canEdit && (props.selecting || modifier)) emit('toggle', props.d)
  else emit('open', props.d)
}
// Space on the focused row selects it (Enter opens it), the keyboard's way into the selection
function onSpace() {
  if (vehicleStore.canEdit) emit('toggle', props.d)
}
// A vehicle fed by TeslaMate mixes synced and manual drives: the badge tells them apart. Without TeslaMate every drive is manual.
const showManualBadge = computed(() => vehicleStore.capabilities.sync)
const carpoolSummary = computed(() => summarizeDriveCarpools(props.d.carpools))
// The glyph says what is particular to the drive (carpooled, work, personal); an ordinary drive keeps the plain trip glyph.
const leadingIcon = computed(() => driveGlyph(props.d, !!carpoolSummary.value, prefs.proPersoEnabled))

// One carpool opens it directly; several open the drive's cost breakdown, which lists them
function openCarpool() {
  const refs = props.d.carpools || []
  if (refs.length === 1) router.push({ path: '/carpools', query: { open: refs[0].id } })
  else emit('open', props.d)
}
// The drive waits for its toll: two taps, enter it or confirm there is none
const qualifying = computed(() => vehicleStore.canEdit && needsTollQualification(props.d))

// The quiet line under the route: the start time, then what is particular to the drive (work, personal, entered by hand).
const subtitleParts = computed(() => {
  const d = props.d
  const parts: string[] = [formatTimeOfDay(d.start_time)]
  if (prefs.proPersoEnabled && d.tags?.includes('Pro')) parts.push(t('drives.driveCard.work'))
  if (prefs.proPersoEnabled && d.tags?.includes('Perso')) parts.push(t('drives.driveCard.personal'))
  if (d.is_manual && showManualBadge.value) parts.push(t('drives.drivesView.manual'))
  return parts
})
</script>

<template>
  <div data-testid="drive-card" class="select-none max-sm:[-webkit-touch-callout:none]">
    <ListItem :class="{ 'border-rose-500/40 bg-slate-800/40 shadow-lg shadow-rose-950/20': selected }">
      <template #open>
        <CardOpenButton
          :label="$t('drives.driveCard.openBreakdown', { date: formatDate(d.start_time) })"
          :aria-pressed="vehicleStore.canEdit && selecting ? selected : undefined"
          @click="onRowClick"
          @keydown.space.prevent
          @keyup.space.prevent="onSpace"
          @pointerdown="press.onPointerdown"
          @pointermove="press.onPointermove"
          @pointerup="press.onPointerup"
          @pointercancel="press.onPointercancel"
          @contextmenu="press.onContextmenu"
        />
      </template>
      <template #lead>
        <!-- The glyph of the row, the rose check once it is selected -->
        <RowLeading :icon="leadingIcon" :selected="selected" always />
      </template>
      <template #title>
        <span :title="d.start_address">{{ d.start_address || $t('drives.driveCard.unknownStart') }}</span>
        <span class="text-slate-400"> → </span>
        <span :title="d.end_address">{{ d.end_address || $t('drives.driveCard.unknownEnd') }}</span>
      </template>
      <template #subtitle>
        {{ subtitleParts.join(' · ') }}
        <span v-if="carpoolSummary && isMobile" class="mt-1 block"><DriveCarpoolChip :summary="carpoolSummary" @click="openCarpool" /></span>
      </template>
      <!-- A wide screen has room for the chip between the route and the cost; a phone gives it its own line under the quiet line -->
      <template v-if="!isMobile && (carpoolSummary || qualifying)" #meta>
        <div class="flex items-center gap-2">
          <DriveCarpoolChip v-if="carpoolSummary" :summary="carpoolSummary" @click="openCarpool" />
          <QualifyActions
            v-if="qualifying"
            class="relative z-10"
            :primary-label="$t('drives.driveCard.toll')"
            :primary-title="$t('drives.driveCard.enterTheTollOfThis')"
            :secondary-label="$t('drives.driveCard.noToll')"
            :secondary-title="$t('drives.driveCard.confirmThatThisDriveHas')"
            @primary="emit('toll-entry', d)"
            @secondary="emit('no-toll', d)"
          />
        </div>
      </template>
      <!-- The toll to qualify: beside the figure on a wide screen, a line of its own inside the card on a phone -->
      <template v-if="isMobile && qualifying" #footer>
        <QualifyActions
          stretch
          :primary-label="$t('drives.driveCard.toll')"
          :primary-title="$t('drives.driveCard.enterTheTollOfThis')"
          :secondary-label="$t('drives.driveCard.noToll')"
          :secondary-title="$t('drives.driveCard.confirmThatThisDriveHas')"
          @primary="emit('toll-entry', d)"
          @secondary="emit('no-toll', d)"
        />
      </template>
      <template #value>
        <span :title="$t('drives.driveCard.actualCostPriceCalculatedFor')">{{ d.costs?.has_estimates ? '~' : '' }}{{ formatAmount(d.costs?.total_cost || 0, vehicleStore.currency) }}</span>
      </template>
    </ListItem>
  </div>
</template>
