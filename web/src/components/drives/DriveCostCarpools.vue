<script setup lang="ts">
import { Users } from 'lucide-vue-next'
import { formatAmount } from '@/currency'

// The carpools shared on a trip or a drive; a click opens that carpool. A trip brings its full carpools, a drive the short
// references of its list row (no legs, no net cost).
defineProps<{ tripCarpools: any[]; vehicleCurrency: string; title: string }>()
const emit = defineEmits<{ open: [carpoolId: string] }>()
</script>

<template>
  <div class="space-y-1.5">
    <h4 class="text-xs font-bold text-white flex items-center gap-1.5">
      <Users class="w-3.5 h-3.5 text-sky-400" />
      {{ title }}
    </h4>
    <button
      v-for="c in tripCarpools"
      :key="c.id"
      type="button"
      @click="emit('open', c.id)"
      class="w-full text-left flex items-center justify-between gap-3 bg-slate-800/40 hover:bg-slate-800/70 border border-slate-700/60 rounded-xl px-3 py-2 transition-colors"
    >
      <div class="min-w-0">
        <div class="text-xs font-semibold text-white truncate">{{ c.title }}</div>
        <div class="text-xs text-slate-400">
          <template v-if="c.legs">{{ $t('drives.driveCostModal.carpoolLegsPassengers', { legs: c.legs.length || 1, passengers: c.passengers?.length || 0 }) }}</template>
          <template v-else>{{ $t('drives.driveCostModal.carpoolPassengers', { passengers: c.passenger_count || 0 }, c.passenger_count || 0) }}</template>
        </div>
      </div>
      <div class="text-right shrink-0">
        <div class="text-xs font-mono font-bold text-success-400">+{{ formatAmount(Number(c.total_revenue || 0), vehicleCurrency) }}</div>
        <div v-if="c.net_cost !== undefined" class="text-xs text-slate-400">{{ $t('drives.driveCostModal.carpoolNetCost', { amount: formatAmount(Number(c.net_cost || 0), vehicleCurrency) }) }}</div>
      </div>
    </button>
  </div>
</template>
