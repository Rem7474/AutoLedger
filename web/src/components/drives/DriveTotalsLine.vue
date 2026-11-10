<script setup lang="ts">
import { computed } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import StatusLabel from '@/components/StatusLabel.vue'
import { formatAmount } from '@/currency'
import { formatDistance } from '@/units'
import type { DriveTotals } from '@/utils/drives'

// The figures at the right of a day or month header: distance and gross cost of the drives listed under it, then the net
// of the carpools when `showNet` and there are some. Nothing is drawn while the group is still loading (`totals` null).
const props = defineProps<{ totals: DriveTotals | null; showNet?: boolean }>()
const vehicleStore = useVehicleStore()
const net = computed(() => (props.showNet ? props.totals?.net ?? null : null))
</script>

<template>
  <p v-if="totals" data-testid="drive-totals" class="ml-auto flex flex-wrap items-baseline justify-end gap-x-1.5 text-xs font-semibold tabular-nums text-slate-300">
    <span>{{ formatDistance(totals.distanceKm, 1) }}</span>
    <span class="text-slate-500" aria-hidden="true">·</span>
    <span>{{ formatAmount(totals.gross, vehicleStore.currency) }}</span>
    <template v-if="net !== null">
      <span class="text-slate-500" aria-hidden="true">·</span>
      <span>{{ $t('drives.drivesView.totalNet', { amount: formatAmount(net, vehicleStore.currency) }) }}</span>
    </template>
    <StatusLabel v-if="totals.incomplete" tone="neutral" :title="$t('drives.drivesView.totalIncompleteHint')">{{ $t('drives.drivesView.totalIncomplete') }}</StatusLabel>
  </p>
</template>
