<script setup lang="ts">
import { computed } from 'vue'
import NumberInput from '@/components/NumberInput.vue'
import CarpoolRouteStrip from '@/components/carpool/CarpoolRouteStrip.vue'
import CarpoolShareMath from '@/components/carpool/CarpoolShareMath.vue'
import { Users, Plus, Trash2, Calculator, ChevronDown, ChevronUp, Minus } from 'lucide-vue-next'
import { cents, euros, newPassenger, passengerLabel, type LegForm, type PassengerForm } from '@/utils/carpool'
import type { StopRange } from '@/utils/carpoolEditor'
import { currencySymbol, formatAmount } from '@/currency'
import { formatDistance } from '@/units'

// The passengers of a carpool: one card for the selected passenger, with where they ride on the route, their seats, what
// was received against the fair share and, on demand, the section by section calculation of that share. `selected` is the
// index of the passenger on the card; `expanded` the index of the one whose calculation is open.
const props = defineProps<{ legs: LegForm[]; stops: string[]; live: any; currency: string; recentNames?: string[] }>()
const passengers = defineModel<PassengerForm[]>('passengers', { required: true })
const expandedPassengerIndex = defineModel<number | null>('expanded', { required: true })
const selected = defineModel<number>('selected', { default: 0 })
const fmt = (v: number) => formatAmount(Number(v || 0), props.currency)

const index = computed(() => Math.min(Math.max(selected.value, 0), passengers.value.length - 1))
const p = computed(() => passengers.value[index.value])
const share = (i: number) => props.live.shares[i] || 0
const ridden = computed(() => props.legs.map((leg, i) => ({ leg, i })).filter(({ i }) => p.value.board_stop_index <= i && i < p.value.alight_stop_index))

function togglePassengerMath() {
  expandedPassengerIndex.value = expandedPassengerIndex.value === index.value ? null : index.value
}

function addPassenger() {
  passengers.value.push(newPassenger(passengers.value.length, props.legs.length))
  selected.value = passengers.value.length - 1
}

function removePassenger() {
  passengers.value.splice(index.value, 1)
  selected.value = Math.min(index.value, passengers.value.length - 1)
  expandedPassengerIndex.value = null
}

function setRange(range: StopRange) {
  p.value.board_stop_index = range.board
  p.value.alight_stop_index = range.alight
}

function setSeats(seats: number) {
  p.value.seats = Math.min(Math.max(seats, 1), 7)
}

// The recent names not already on this carpool
const suggestedNames = computed(() => {
  const used = new Set(passengers.value.map((q, i) => (i === index.value ? '' : q.passenger_name.trim().toLowerCase())))
  return (props.recentNames || []).filter((n) => !used.has(n.toLowerCase())).slice(0, 6)
})

function applyExactAmount() {
  p.value.amount_paid = euros(share(index.value))
}
</script>

<template>
  <div class="space-y-3">
    <div class="flex items-center justify-between gap-2">
      <h4 class="text-xs font-bold text-white flex items-center gap-1.5">
        <Users class="w-4 h-4 text-slate-400" />
        {{ $t('carpool.carpoolTripModal.passengersBoardingAndAlighting') }}
      </h4>
    </div>

    <div class="flex flex-wrap gap-2" role="tablist" :aria-label="$t('carpool.editor.passengers')">
      <button
        v-for="(q, i) in passengers"
        :key="i"
        type="button"
        role="tab"
        :aria-selected="i === index"
        class="flex min-h-11 items-center gap-2 rounded-full border px-3 text-xs font-semibold"
        :class="i === index ? 'border-rose-500/60 bg-rose-500/10 text-rose-100' : 'border-slate-700 bg-slate-800 text-slate-200 hover:bg-slate-700'"
        @click="selected = i"
      >
        <span class="max-w-28 truncate">{{ passengerLabel(q.passenger_name, i) }}</span>
        <span class="tabular-nums" :class="i === index ? 'text-sky-300' : 'text-sky-400'">{{ fmt(euros(share(i))) }}</span>
      </button>
      <button type="button" class="flex min-h-11 items-center gap-1 rounded-full border border-dashed border-slate-600 px-3 text-xs font-semibold text-slate-300 hover:bg-slate-800" @click="addPassenger">
        <Plus class="w-3.5 h-3.5" /> {{ $t('carpool.carpoolTripModal.addAPassenger') }}
      </button>
    </div>

    <div v-if="p" class="space-y-4 rounded-xl border border-slate-800 bg-slate-950/50 p-3">
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-[1fr_auto] sm:items-start">
        <div>
          <label :for="`passenger-name-${index}`" class="block text-xs text-slate-400 mb-0.5">{{ $t('carpool.carpoolTripModal.name') }}</label>
          <input
            :id="`passenger-name-${index}`"
            v-model="p.passenger_name"
            :placeholder="passengerLabel('', index)"
            autocomplete="off"
            class="field"
          />
          <div v-if="!p.passenger_name.trim() && suggestedNames.length" class="mt-1.5 flex flex-wrap items-center gap-1.5 text-xs">
            <span class="text-slate-400">{{ $t('carpool.carpoolTripModal.recentPassengers') }}</span>
            <button
              v-for="name in suggestedNames"
              :key="name"
              type="button"
              class="rounded-full border border-slate-700 bg-slate-800 px-2.5 py-1 font-semibold text-slate-200 hover:bg-slate-700"
              @click="p.passenger_name = name"
            >
              {{ name }}
            </button>
          </div>
        </div>
        <div>
          <span :id="`passenger-seats-label-${index}`" class="block text-xs text-slate-400 mb-0.5">{{ $t('carpool.carpoolTripModal.seats') }}</span>
          <div class="flex items-center gap-1" role="group" :aria-labelledby="`passenger-seats-label-${index}`">
            <button type="button" class="btn btn-secondary tap" :disabled="p.seats <= 1" :aria-label="$t('carpool.editor.fewerSeats')" @click="setSeats(Number(p.seats) - 1)">
              <Minus class="h-4 w-4" aria-hidden="true" />
            </button>
            <output :id="`passenger-seats-${index}`" class="w-8 text-center text-sm font-bold tabular-nums text-white">{{ p.seats }}</output>
            <button type="button" class="btn btn-secondary tap" :disabled="p.seats >= 7" :aria-label="$t('carpool.editor.moreSeats')" @click="setSeats(Number(p.seats) + 1)">
              <Plus class="h-4 w-4" aria-hidden="true" />
            </button>
          </div>
        </div>
      </div>

      <div>
        <div class="mb-1 text-xs text-slate-400">{{ $t('carpool.editor.rides', { from: stops[p.board_stop_index], to: stops[p.alight_stop_index] }) }}</div>
        <CarpoolRouteStrip :stops="stops" :board="p.board_stop_index" :alight="p.alight_stop_index" @change="setRange" />
      </div>

      <div class="flex flex-wrap items-end gap-2 text-xs">
        <div class="w-36">
          <label :for="`passenger-paid-${index}`" class="block text-xs text-slate-400 mb-0.5">{{ $t('carpool.carpoolTripModal.paid', { cur: currencySymbol(currency) }) }}</label>
          <NumberInput
            :id="`passenger-paid-${index}`"
            v-model="p.amount_paid"
            min="0"
            class="field font-bold text-emerald-300"
          />
        </div>
        <button
          type="button"
          class="btn btn-secondary tap"
          :title="$t('carpool.carpoolTripModal.exactAmountTitle', { amount: fmt(euros(share(index))) })"
          @click="applyExactAmount"
        >
          {{ $t('carpool.carpoolTripModal.exactAmount') }}
        </button>
        <p class="pb-2 text-slate-400">
          {{ $t('carpool.carpoolTripModal.fairShare') }} <strong class="text-sky-300 tabular-nums">{{ fmt(euros(share(index))) }}</strong>
          <span v-if="cents(p.amount_paid) !== share(index)" class="ml-2">
            {{ cents(p.amount_paid) > share(index)
              ? $t('carpool.above', { amount: fmt(euros(cents(p.amount_paid) - share(index))) })
              : $t('carpool.below', { amount: fmt(euros(share(index) - cents(p.amount_paid))) }) }}
          </span>
        </p>
      </div>

      <div class="space-y-2">
        <ul class="flex flex-wrap gap-1.5 text-xs" :aria-label="$t('carpool.editor.shareByLeg')">
          <li v-for="{ leg, i } in ridden" :key="i" class="rounded-full border border-sky-500/20 bg-sky-500/10 px-2.5 py-1 text-slate-300">
            {{ stops[i] }} → {{ stops[i + 1] }}
            <span v-if="Number(leg.distance_km)" class="text-slate-400">({{ formatDistance(Number(leg.distance_km), 1) }})</span>
            <strong class="ml-1 tabular-nums text-sky-300">{{ fmt(euros((live.legDetails[i]?.perPerson || 0) * (Number(p.seats) || 1))) }}</strong>
          </li>
        </ul>
        <div class="flex flex-wrap items-center justify-between gap-2 text-xs">
          <button
            type="button"
            @click="togglePassengerMath"
            class="tap-text text-sky-400 hover:text-sky-300 font-medium flex items-center gap-1 transition-colors"
            :title="expandedPassengerIndex === index ? $t('carpool.carpoolTripModal.hideCalcDetail') : $t('carpool.carpoolTripModal.explainShare')"
          >
            <Calculator class="w-3.5 h-3.5" />
            <span>{{ expandedPassengerIndex === index ? $t('carpool.carpoolTripModal.hideCalc') : $t('carpool.carpoolTripModal.calcDetail') }}</span>
            <ChevronUp v-if="expandedPassengerIndex === index" class="w-3.5 h-3.5" />
            <ChevronDown v-else class="w-3.5 h-3.5" />
          </button>
          <button v-if="passengers.length > 1" type="button" @click="removePassenger" class="tap-text flex items-center gap-1 text-slate-400 hover:text-danger-400" :title="$t('carpool.carpoolTripModal.removeThisPassenger')" :aria-label="$t('carpool.carpoolTripModal.removeThisPassenger')">
            <Trash2 class="w-3.5 h-3.5" />
          </button>
        </div>
        <CarpoolShareMath v-if="expandedPassengerIndex === index" :passenger="p" :index="index" :legs="legs" :stops="stops" :live="live" :currency="currency" />
      </div>
    </div>
  </div>
</template>
