<script setup lang="ts">
import { Users, ChevronRight } from 'lucide-vue-next'
import { formatAmount } from '@/currency'

// The carpools shared on a trip or a drive, each with what it costs, what was received and what is left to the driver; a
// click opens that carpool. A trip brings its full carpools, a drive the short references of its list row (no legs).
const props = defineProps<{ tripCarpools: any[]; vehicleCurrency: string; title: string }>()
const emit = defineEmits<{ open: [carpoolId: string] }>()
const fmt = (v: unknown) => formatAmount(Number(v || 0), props.vehicleCurrency)
const passengers = (c: any): number => c.passenger_count ?? c.passengers?.length ?? 0
// Nothing left to the driver: what was received covers the cost
const covered = (c: any) => Number(c.total_cost) > 0 && Number(c.net_cost) <= 0
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
      class="w-full text-left bg-slate-800/40 hover:bg-slate-800/70 border border-slate-700/60 rounded-xl px-3 py-2.5 transition-colors space-y-2"
    >
      <div class="flex items-center justify-between gap-3">
        <div class="min-w-0">
          <div class="text-xs font-semibold text-white truncate">{{ c.title }}</div>
          <div class="text-xs text-slate-400">{{ $t('drives.driveCostModal.carpoolPassengers', { passengers: passengers(c) }, passengers(c)) }}</div>
        </div>
        <ChevronRight class="w-4 h-4 shrink-0 text-slate-400" aria-hidden="true" />
      </div>
      <dl v-if="c.total_cost !== undefined" class="grid grid-cols-3 gap-2 text-xs">
        <div class="rounded-lg border border-slate-700/70 bg-slate-800/70 px-2 py-1.5">
          <dt class="text-slate-400">{{ $t('carpool.carpoolTripModal.actualCost') }}</dt>
          <dd class="font-bold tabular-nums text-white">{{ fmt(c.total_cost) }}</dd>
        </div>
        <div class="rounded-lg border border-emerald-500/20 bg-emerald-500/10 px-2 py-1.5">
          <dt class="text-slate-400">{{ $t('carpool.carpoolTripModal.received') }}</dt>
          <dd class="font-bold tabular-nums text-emerald-300">{{ fmt(c.total_revenue) }}</dd>
        </div>
        <div class="rounded-lg border px-2 py-1.5" :class="covered(c) ? 'border-emerald-500/30 bg-emerald-500/10' : 'border-slate-700/70 bg-slate-800/70'">
          <dt class="text-slate-400">{{ covered(c) ? $t('carpool.carpoolTripModal.surplus') : $t('carpool.carpoolTripModal.leftToDriver') }}</dt>
          <dd class="font-bold tabular-nums" :class="covered(c) ? 'text-emerald-300' : 'text-white'">{{ fmt(Math.abs(Number(c.net_cost || 0))) }}</dd>
        </div>
      </dl>
    </button>
  </div>
</template>
