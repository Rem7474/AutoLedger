<script setup lang="ts">
import { computed } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import { t } from '@/i18n'
import CardOpenButton from '@/components/CardOpenButton.vue'
import ListItem from '@/components/ListItem.vue'
import QuietLine from '@/components/QuietLine.vue'
import StatusLabel from '@/components/StatusLabel.vue'
import { hasTeslaMate } from '@/utils/vehicles'
import { summarizeDriveCarpools } from '@/utils/drives'
import { formatDayTime } from '@/utils/dates'
import { formatAmount } from '@/currency'
import { formatDistance } from '@/units'
import { formatCostPerDistance, NO_VALUE } from '@/utils/costPerDistance'

// One leg of a trip, in the trip's list and in its detail: the route, when and how far, what it cost. The row is flat (no
// card: it sits in a list indented under its trip) and opens the cost breakdown of the leg; it carries no menu and no
// other control. The addresses are cut to their first segment, the whole text is the title attribute.
const props = defineProps<{ leg: any }>()
defineEmits<{ open: [leg: any] }>()
const vehicleStore = useVehicleStore()
const start = computed(() => (props.leg.start_address || t('drives.driveCostModal.start')).split(',')[0])
const end = computed(() => (props.leg.end_address || t('drives.driveCostModal.end')).split(',')[0])
const fullRoute = computed(() => `${props.leg.start_address || t('drives.driveCostModal.start')} → ${props.leg.end_address || t('drives.driveCostModal.end')}`)
// A leg whose cost is unknown shows the empty placeholder, never a zero
const cost = computed(() => {
  const total = props.leg.costs?.total_cost
  if (total === null || total === undefined || !Number.isFinite(Number(total))) return NO_VALUE
  return `${props.leg.costs.has_estimates ? '~' : ''}${formatAmount(Number(total), vehicleStore.currency)}`
})
const carpools = computed(() => summarizeDriveCarpools(props.leg.carpools))
// A vehicle fed by TeslaMate mixes synced and manual legs: the label tells them apart
const showManual = computed(() => !!props.leg.is_manual && hasTeslaMate(vehicleStore.activeVehicle))
</script>

<template>
  <ListItem flat wrap-title data-testid="drive-leg">
    <template #open>
      <CardOpenButton :label="$t('drives.driveCard.openBreakdown', { date: formatDayTime(leg.start_time) })" @click="$emit('open', leg)" />
    </template>
    <template #title>
      <span :title="fullRoute">{{ start }}&nbsp;<span class="text-slate-400">→</span> {{ end }}</span>
    </template>
    <template #subtitle>
      <QuietLine>
        <span><span>{{ formatDayTime(leg.start_time) }}</span><span>{{ formatDistance(leg.distance_km, 1) }}</span></span>
        <span v-if="carpools" data-plain>
          <StatusLabel tone="info">{{ $t('drives.driveCard.carpooledSummary', { passengers: carpools.passengers }, carpools.passengers) }}</StatusLabel>
        </span>
        <span v-if="showManual"><span>{{ $t('drives.drivesView.manual') }}</span></span>
      </QuietLine>
    </template>
    <template #value>{{ cost }}</template>
    <template #subvalue>{{ formatCostPerDistance(leg.costs?.cost_per_km, vehicleStore.currency, 3, true) }}</template>
  </ListItem>
</template>
