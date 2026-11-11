<script setup lang="ts">
import { Users } from '@/icons'
import { useVehicleStore } from '@/stores/vehicle'
import { formatAmount } from '@/currency'

// The carpool shortcut of a drive row: the one clickable chip of the row, above the stretched open button.
defineProps<{ summary: { passengers: number; revenue: number } }>()
defineEmits<{ click: [] }>()
const vehicleStore = useVehicleStore()
</script>

<template>
  <button
    type="button"
    class="relative z-10 inline-flex max-w-full items-center gap-1 rounded-full border border-sky-500/30 bg-sky-500/15 px-2 py-0.5 text-xs font-semibold text-sky-300 transition-colors hover:bg-sky-500/25"
    :title="$t('drives.driveCard.carpooledOpen')"
    @click="$emit('click')"
  >
    <Users class="size-3 shrink-0" />
    <span class="truncate max-sm:hidden">{{ $t('drives.driveCard.carpooledSummary', { passengers: summary.passengers }, summary.passengers) }}</span>
    <span class="truncate sm:hidden">{{ $t('drives.driveCard.carpooledShort', { passengers: summary.passengers }, summary.passengers) }}</span>
    <span class="text-success-400">+{{ formatAmount(summary.revenue, vehicleStore.currency) }}</span>
  </button>
</template>
