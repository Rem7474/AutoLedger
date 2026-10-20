<script setup lang="ts">
import { computed, ref } from 'vue'
import NumberInput from '@/components/NumberInput.vue'
import DistanceInput from '@/components/DistanceInput.vue'
import { Navigation, Plus, Trash2, Calculator, RotateCw, Lock, Undo2, ChevronDown } from 'lucide-vue-next'
import { COST_FIELDS, differsFromDrive, euros, type CostKey, type LegForm } from '@/utils/carpool'
import { costSegments } from '@/utils/carpoolEditor'
import { currencySymbol, formatAmount } from '@/currency'
import { distanceUnit, formatDistanceValue } from '@/units'

// The legs of a carpool with their actual costs. Estimating, adding and removing a leg belong to the form, so they are
// asked of it; the fields of a leg are edited in place. A leg made from a drive is a summary line that opens on its
// fields; a leg typed by hand is open, as its fields are all there is to it.
const props = defineProps<{
  legs: LegForm[]
  sourceMode: 'DRIVES' | 'MANUAL'
  isEditing: boolean
  submitted: boolean
  estimating: boolean
  live: any
  liveDistance: number
  currency: string
  followDrives: boolean
}>()
const emit = defineEmits<{ recalculate: []; 'add-leg': []; estimate: [index: number]; remove: [index: number]; 'update:followDrives': [value: boolean] }>()
const fmt = (v: number) => formatAmount(Number(v || 0), props.currency)
const linked = computed(() => props.legs.some((l) => !!l.drive_id))
// Legs whose state was toggled by the user: a drive leg opens, a manual leg closes
const toggled = ref(new Set<number>())
const isOpen = (leg: LegForm, i: number) => (leg.drive_id ? toggled.value.has(i) : !toggled.value.has(i))
function toggle(i: number) {
  const next = new Set(toggled.value)
  if (!next.delete(i)) next.add(i)
  toggled.value = next
}
const adjusted = (leg: LegForm) => COST_FIELDS.some((f) => differsFromDrive(leg, f.key))
const SHADES = ['bg-slate-200', 'bg-slate-500', 'bg-slate-300', 'bg-slate-600', 'bg-slate-400', 'bg-slate-700']
// The amount of a category goes back to what its drive costs
function restore(leg: LegForm, key: CostKey) {
  if (leg.drive_cost) leg[key] = leg.drive_cost[key] ?? 0
}
</script>

<template>
  <div class="space-y-2">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <h4 class="text-xs font-bold text-white flex items-center gap-1.5">
        <Navigation class="w-4 h-4 text-sky-400" />
        {{ $t('carpool.carpoolTripModal.legsAndActualCostsKm', { unit: distanceUnit(), liveDistance: formatDistanceValue(liveDistance, 1), total: fmt(euros(live.total)) }) }}
      </h4>
      <div class="flex items-center gap-2">
        <button
          v-if="isEditing && legs.length > 0 && !linked"
          type="button"
          @click="emit('recalculate')"
          :disabled="estimating"
          class="text-xs text-rose-400 hover:text-rose-300 font-semibold flex items-center gap-1.5 px-2.5 py-1 bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/30 rounded-lg transition-colors disabled:opacity-50"
          :title="$t('carpool.carpoolTripModal.updateTheActualCostsWith')"
        >
          <RotateCw class="w-3.5 h-3.5" :class="{ 'animate-spin': estimating }" />
          <span>{{ $t('carpool.carpoolTripModal.recalculateTheCosts') }}</span>
        </button>
        <button
          v-if="sourceMode === 'MANUAL'"
          type="button"
          @click="emit('add-leg')"
          class="text-xs text-sky-400 hover:text-sky-300 font-semibold flex items-center gap-1"
        >
          <Plus class="w-3.5 h-3.5" /> {{ $t('carpool.carpoolTripModal.addALeg') }}
        </button>
      </div>
    </div>
    <label v-if="linked" class="flex items-start gap-2 text-xs text-slate-300 bg-slate-950/50 border border-slate-800 rounded-xl px-3 py-2 cursor-pointer">
      <input
        type="checkbox"
        class="mt-0.5"
        :checked="followDrives"
        @change="emit('update:followDrives', ($event.target as HTMLInputElement).checked)"
      />
      <span>
        <span class="font-semibold text-white flex items-center gap-1.5">
          <Lock v-if="!followDrives" class="w-3.5 h-3.5 text-slate-400" />
          {{ $t('carpool.linked.follow') }}
        </span>
        <span class="text-slate-400">{{ followDrives ? $t('carpool.linked.followHint') : $t('carpool.linked.frozenHint') }}</span>
      </span>
    </label>
    <p v-if="!legs.length" role="status" :class="['text-xs', submitted ? 'text-danger-400' : 'text-slate-400']">{{ $t('carpool.carpoolTripModal.selectAtLeastOneDrive') }}</p>

    <div v-for="(leg, i) in legs" :key="i" class="bg-slate-950/50 border border-slate-800 rounded-xl">
      <button
        type="button"
        class="flex w-full flex-wrap items-center gap-x-3 gap-y-1.5 rounded-xl px-3 py-2.5 text-left text-xs"
        :aria-expanded="isOpen(leg, i)"
        @click="toggle(i)"
      >
        <span class="font-bold text-sky-300">{{ $t('carpool.carpoolTripModal.leg', { i: i + 1 }) }}</span>
        <span class="min-w-0 flex-1 truncate text-slate-200">{{ leg.start_label || $t('carpool.carpoolTripModal.start') }} → {{ leg.end_label || $t('carpool.carpoolTripModal.destination') }}</span>
        <span class="shrink-0 text-slate-400 tabular-nums">{{ formatDistanceValue(Number(leg.distance_km) || 0, 1) }} {{ distanceUnit() }}</span>
        <span class="shrink-0 font-semibold tabular-nums text-white">{{ fmt(euros(live.legDetails[i]?.total || 0)) }}</span>
        <ChevronDown class="h-4 w-4 shrink-0 text-slate-400 transition-transform" :class="isOpen(leg, i) ? 'rotate-180' : ''" aria-hidden="true" />
        <span class="flex h-1.5 w-full overflow-hidden rounded-full bg-slate-800" aria-hidden="true">
          <span
            v-for="(seg, k) in costSegments(leg)"
            :key="seg.key"
            :class="SHADES[k % SHADES.length]"
            :style="{ width: `${seg.pct}%` }"
            :title="`${$t(`carpool.costFields.${COST_FIELDS.find((f) => f.key === seg.key)?.label}`)}: ${fmt(seg.cents / 100)}`"
          ></span>
        </span>
        <span v-if="adjusted(leg)" class="rounded-full border border-slate-700 bg-slate-800 px-2 py-0.5 font-semibold text-slate-300">{{ $t('carpool.linked.adjusted') }}</span>
      </button>

      <div v-if="isOpen(leg, i)" class="space-y-2 border-t border-slate-800 px-3 pb-3 pt-2.5">
        <div class="flex flex-wrap items-center gap-2 text-xs">
          <label :for="`leg-start-${i}`" class="sr-only">{{ $t('carpool.carpoolTripModal.startOfLeg', { i: i + 1 }) }}</label>
          <input
            :id="`leg-start-${i}`"
            v-model="leg.start_label"
            :placeholder="$t('carpool.carpoolTripModal.start')"
            class="field w-36!"
          />
          <span class="text-slate-400">→</span>
          <label :for="`leg-end-${i}`" class="sr-only">{{ $t('carpool.carpoolTripModal.endOfLeg', { i: i + 1 }) }}</label>
          <input
            :id="`leg-end-${i}`"
            v-model="leg.end_label"
            :placeholder="$t('carpool.carpoolTripModal.destination')"
            class="field w-36!"
          />
          <label :for="`leg-distance-${i}`" class="text-slate-400">{{ distanceUnit() }}</label>
          <DistanceInput
            :id="`leg-distance-${i}`"
            v-model="leg.distance_km"
            step="0.1"
            min="0"
            :readonly="!!leg.drive_id"
            class="field w-20!"
          />
          <button
            v-if="!leg.drive_id"
            type="button"
            @click="emit('estimate', i)"
            class="flex items-center gap-1 text-xs text-sky-400 hover:text-sky-300"
            :title="$t('carpool.carpoolTripModal.estimateElectricityTiresMaintenanceAnd')"
          >
            <Calculator class="w-3.5 h-3.5" /> {{ $t('carpool.carpoolTripModal.estimate') }}
          </button>
          <span class="ml-auto text-slate-300">
            {{ $t('carpool.carpoolTripModal.onBoard', { legDetails: fmt(euros(live.legDetails[i]?.total || 0)), legDetails2: 1 + (live.legDetails[i]?.seats || 0) }) }}
            <strong>{{ $t('carpool.carpoolTripModal.person', { legDetails: fmt(euros(live.legDetails[i]?.perPerson || 0)) }) }}</strong>
          </span>
          <button v-if="sourceMode === 'MANUAL' && legs.length > 1" type="button" @click="emit('remove', i)" class="tap text-slate-400 hover:text-danger-400" :title="$t('carpool.carpoolTripModal.deleteTheLeg')" :aria-label="$t('carpool.carpoolTripModal.deleteTheLeg')">
            <Trash2 class="w-3.5 h-3.5" />
          </button>
        </div>
        <div class="grid grid-cols-3 sm:grid-cols-6 gap-2">
          <div v-for="f in COST_FIELDS" :key="f.key">
            <label :for="`leg-${f.key}-${i}`" class="block text-xs text-slate-400 mb-0.5">{{ $t(`carpool.costFields.${f.label}`) }} ({{ currencySymbol(currency) }})</label>
            <NumberInput
              :id="`leg-${f.key}-${i}`"
              v-model="(leg as any)[f.key]"
              min="0"
              class="field"
            />
            <p v-if="differsFromDrive(leg, f.key)" class="mt-0.5 flex flex-wrap items-center gap-x-1 text-xs text-slate-400">
              <span>{{ $t('carpool.linked.driveCost', { amount: fmt(leg.drive_cost?.[f.key] ?? 0) }) }}</span>
              <button
                v-if="followDrives"
                type="button"
                class="tap text-sky-400 hover:text-sky-300"
                :title="$t('carpool.linked.restore')"
                :aria-label="$t('carpool.linked.restore')"
                @click="restore(leg, f.key)"
              >
                <Undo2 class="w-3.5 h-3.5" />
              </button>
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
