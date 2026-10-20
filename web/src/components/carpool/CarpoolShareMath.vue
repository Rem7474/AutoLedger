<script setup lang="ts">
import { Calculator } from 'lucide-vue-next'
import { euros, passengerLabel, type LegForm, type PassengerForm } from '@/utils/carpool'
import { formatAmount } from '@/currency'
import { formatDistance } from '@/units'

// The fair share of one passenger section by section: what each leg costs, who is on board, and the share of the leg.
const props = defineProps<{ passenger: PassengerForm; index: number; legs: LegForm[]; stops: string[]; live: any; currency: string }>()
const fmt = (v: number) => formatAmount(Number(v || 0), props.currency)
</script>

<template>
  <div class="space-y-2.5 rounded-xl border border-slate-800 bg-slate-950/70 p-3">
    <div class="flex items-center justify-between text-xs">
      <span class="font-bold text-slate-200 flex items-center gap-1.5">
        <Calculator class="w-3.5 h-3.5 text-sky-400" />
        {{ $t('carpool.carpoolTripModal.formulaPerSection', { name: passengerLabel(passenger.passenger_name, index) }) }}
      </span>
      <span class="text-xs text-slate-400 font-medium">
        {{ $t('carpool.carpoolTripModal.seatsBooked', passenger.seats) }}
      </span>
    </div>

    <div class="space-y-2">
      <div
        v-for="(leg, legIdx) in legs"
        :key="legIdx"
        class="text-xs p-2.5 rounded-lg border transition-colors"
        :class="passenger.board_stop_index <= legIdx && legIdx < passenger.alight_stop_index
          ? 'bg-slate-900/90 border-sky-500/30 text-slate-200'
          : 'bg-slate-900/30 border-slate-800/50 text-slate-400 opacity-60'"
      >
        <div class="flex items-center justify-between font-semibold">
          <span class="flex items-center gap-1.5">
            <span class="w-4 h-4 rounded-full bg-slate-800 flex items-center justify-center text-xs font-mono text-slate-300">
              {{ legIdx + 1 }}
            </span>
            <span>{{ stops[legIdx] }} → {{ stops[legIdx + 1] }}</span>
            <span v-if="Number(leg.distance_km)" class="text-slate-400 font-normal">({{ formatDistance(Number(leg.distance_km), 1) }})</span>
          </span>
          <span v-if="passenger.board_stop_index <= legIdx && legIdx < passenger.alight_stop_index" class="text-sky-300 font-bold">
            {{ fmt(euros((live.legDetails[legIdx]?.perPerson || 0) * (Number(passenger.seats) || 1))) }}
          </span>
          <span v-else class="text-slate-400 italic text-xs">
            {{ $t('carpool.carpoolTripModal.notTravelled') }}
          </span>
        </div>

        <div v-if="passenger.board_stop_index <= legIdx && legIdx < passenger.alight_stop_index" class="mt-1.5 pl-5.5 text-xs text-slate-400 flex flex-wrap items-center gap-x-2.5 gap-y-1">
          <span>{{ $t('carpool.carpoolTripModal.actualCostOfTheSection') }} <strong class="text-slate-200">{{ fmt(euros(live.legDetails[legIdx]?.total || 0)) }}</strong></span>
          <span>•</span>
          <span>{{ $t('carpool.carpoolTripModal.occupants') }} <strong class="text-slate-200">{{ $t('carpool.carpoolTripModal.oneDriverPlusPassengers', { legDetails: live.legDetails[legIdx]?.seats || 0, legDetails2: 1 + (live.legDetails[legIdx]?.seats || 0) }) }}</strong></span>
          <span>•</span>
          <span class="text-sky-300/90">
            {{ $t('carpool.carpoolTripModal.calcFormula', { total: fmt(euros(live.legDetails[legIdx]?.total || 0)), people: 1 + (live.legDetails[legIdx]?.seats || 0) }) }}{{ passenger.seats > 1 ? $t('carpool.carpoolTripModal.calcSeats', { seats: passenger.seats }) : '' }}
          </span>
        </div>
      </div>
    </div>

    <div class="pt-2 border-t border-slate-800/80 flex items-center justify-between text-xs">
      <span class="text-slate-400">{{ $t('carpool.carpoolTripModal.totalFairShareDue') }}</span>
      <div class="text-right">
        <span class="font-bold text-success-400 text-sm">{{ fmt(euros(live.shares[index] || 0)) }}</span>
        <span class="text-xs text-slate-400 block">{{ $t('carpool.carpoolTripModal.exactSumOfTheSections') }}</span>
      </div>
    </div>
  </div>
</template>
