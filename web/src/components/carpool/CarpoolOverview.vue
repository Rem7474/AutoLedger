<script setup lang="ts">
import { computed } from 'vue'
import { Car } from 'lucide-vue-next'
import { euros, passengerLabel, type PassengerForm } from '@/utils/carpool'
import { formatAmount } from '@/currency'

// Who is on board on each leg: the driver over the whole route, one bar per passenger over the legs they ride, and under
// each leg the number of people on board with what each of them pays. A tap on a bar selects that passenger.
const props = defineProps<{ passengers: PassengerForm[]; stops: string[]; live: any; currency: string }>()
const selected = defineModel<number>('selected', { default: 0 })
const fmt = (v: number) => formatAmount(Number(v || 0), props.currency)

const legCount = computed(() => Math.max(props.stops.length - 1, 1))
const columns = computed(() => ({ gridTemplateColumns: `repeat(${legCount.value}, minmax(0, 1fr))` }))
const span = (p: PassengerForm) => ({ gridColumn: `${p.board_stop_index + 1} / ${p.alight_stop_index + 1}` })
const people = (leg: number) => 1 + (props.live.legDetails[leg]?.seats || 0)
const each = (leg: number) => euros(props.live.legDetails[leg]?.perPerson || 0)
</script>

<template>
  <div class="space-y-1.5 rounded-xl border border-slate-800 bg-slate-950/50 p-3" role="group" :aria-label="$t('carpool.editor.overview')">
    <div class="grid gap-y-1.5" :style="columns">
      <div class="col-span-full flex h-7 items-center gap-1.5 rounded-full bg-slate-700 px-2.5 text-xs font-semibold text-slate-200" style="grid-column: 1 / -1">
        <Car class="h-3.5 w-3.5 shrink-0 text-slate-400" aria-hidden="true" />
        <span class="truncate">{{ $t('carpool.editor.driverBar') }}</span>
      </div>
      <button
        v-for="(p, i) in passengers"
        :key="i"
        type="button"
        class="flex h-7 min-w-0 items-center rounded-full border px-2.5 text-left text-xs font-semibold transition-colors"
        :class="i === selected ? 'border-rose-500/70 bg-sky-500/30 text-white' : 'border-sky-500/30 bg-sky-500/15 text-sky-100 hover:bg-sky-500/25'"
        :style="span(p)"
        :aria-pressed="i === selected"
        :title="passengerLabel(p.passenger_name, i)"
        @click="selected = i"
      >
        <span class="truncate">{{ passengerLabel(p.passenger_name, i) }}</span>
      </button>
    </div>
    <div class="grid gap-x-1 border-t border-slate-800 pt-1.5 text-center" :style="columns">
      <div v-for="leg in legCount" :key="leg" class="min-w-0 text-xs" :title="`${stops[leg - 1]} → ${stops[leg]}`">
        <div class="truncate text-slate-400">{{ $t('carpool.editor.peopleOnBoard', people(leg - 1)) }}</div>
        <div class="truncate font-semibold tabular-nums text-sky-300">{{ fmt(each(leg - 1)) }}</div>
      </div>
    </div>
  </div>
</template>
