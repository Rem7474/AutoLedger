<script setup lang="ts">
import { formatNumber, formatPercent } from '@/utils/numbers'
import { computed, onBeforeUnmount, ref } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import { Users, MapPin, Navigation, Pencil, RotateCw, Link2, Lock, LockOpen, CheckCircle2, Sparkles, ChevronRight, Zap, Disc, Wrench, Shield, Receipt, Copy, Check } from 'lucide-vue-next'
import CostDonut from '@/components/costs/CostDonut.vue'
import CostItemRow from '@/components/costs/CostItemRow.vue'
import { buildCarpoolBreakdown } from '@/utils/costBreakdown'
import { carpoolCoverage, carpoolShareSummary, driveCostGapCents, formatDate, hasLinkedLegs, passengerLabel, stopNames } from '@/utils/carpool'
import { formatAmount } from '@/currency'
import ModalShell from '@/components/ModalShell.vue'
import CostTotalBar from '@/components/CostTotalBar.vue'
import { formatCostPerDistance } from '@/utils/costPerDistance'
import { distanceUnit, formatDistance } from '@/units'

// Detail of a carpool trip, laid out like the drive and trip cost breakdown: summary, legs, cost split, passengers,
// total. A leg made from a drive opens the cost detail of that drive, like the legs of a trip. The form to change the
// carpool is the edit modal, reached from the footer.
const props = defineProps<{ trip: any | null; recalculating: boolean }>()
const emit = defineEmits<{ edit: [trip: any]; recalculate: [trip: any]; freeze: [trip: any]; unfreeze: [trip: any]; 'open-drive': [driveId: string] }>()
const open = defineModel<boolean>('open', { required: true })
const vehicleStore = useVehicleStore()
// Carpool amounts are in the vehicle's own currency
const fmt = (v: number) => formatAmount(Number(v || 0), vehicleStore.currency)

const breakdown = computed(() => buildCarpoolBreakdown(props.trip))
const coverage = computed(() => carpoolCoverage(props.trip))
// Positive when the passengers paid more than their share
const balance = computed(() => coverage.value.paid - coverage.value.fair)
// Same icons, colors and labels as the cost rows of the drive detail
const ROWS = {
  energy: { icon: Zap, tone: 'sky', label: 'drives.driveCostModal.electricEnergy' },
  tires: { icon: Disc, tone: 'emerald', label: 'drives.driveCostModal.tireWear' },
  maintenance: { icon: Wrench, tone: 'pink', label: 'drives.driveCostModal.maintenanceProvision' },
  insurance: { icon: Shield, tone: 'purple', label: 'drives.driveCostModal.insuranceShareFixedCost' },
  tolls: { icon: Receipt, tone: 'amber', label: 'drives.driveCostModal.tollsAndRoadCosts' },
  other: { icon: Receipt, tone: 'slate', label: 'dashboard.breakdown.other' },
} as const
const stops = computed(() => stopNames(props.trip?.legs || []))
const passengers = computed<any[]>(() => props.trip?.passengers || [])

// The recap of a passenger's share goes to the clipboard, ready to paste into a message; the button confirms for a moment
const copiedIndex = ref<number | null>(null)
let copiedTimer: ReturnType<typeof setTimeout> | undefined
async function copySummary(index: number) {
  try {
    await navigator.clipboard.writeText(carpoolShareSummary(props.trip, index, fmt))
  } catch {
    return
  }
  copiedIndex.value = index
  clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => (copiedIndex.value = null), 2000)
}
onBeforeUnmount(() => clearTimeout(copiedTimer))
const linked = computed(() => hasLinkedLegs(props.trip))
const frozen = computed(() => !!props.trip?.costs_frozen_at)
// What the drives cost now minus what the frozen carpool shows, in the vehicle's currency
const gap = computed(() => driveCostGapCents(props.trip?.legs || []) / 100)
</script>

<template>
  <ModalShell v-if="trip" v-model:open="open" size="lg" footer-class="items-center justify-between">
    <template #title>
        <div class="flex items-center gap-2.5 min-w-0 pr-2">
          <div class="p-2 rounded-xl shrink-0 bg-rose-500/10 text-rose-400">
            <Users class="w-5 h-5" />
          </div>
          <div class="min-w-0 truncate">
            <h3 class="text-base font-bold text-white truncate">{{ trip.title }}</h3>
            <p class="text-xs text-slate-400">{{ formatDate(trip.date) }}</p>
          </div>
        </div>
    </template>

        <!-- Summary route -->
        <div class="bg-slate-800/60 border border-slate-700/60 p-3.5 rounded-2xl space-y-2">
          <div class="text-sm font-semibold text-white flex items-center gap-2">
            <MapPin class="w-4 h-4 text-rose-400 shrink-0" />
            <span class="truncate">{{ stops[0] }}</span>
            <span class="text-slate-400">→</span>
            <span class="truncate">{{ stops[stops.length - 1] }}</span>
          </div>
          <div class="flex items-center gap-3 text-xs text-slate-300 flex-wrap">
            <span class="font-bold text-rose-400">{{ formatDistance(trip.distance_km, 1) }}</span>
            <span class="text-sky-400 font-semibold">{{ $t('carpool.carpoolTripList.legs', { length: trip.legs.length }) }}</span>
            <span class="text-blue-400 font-semibold">{{ $t('carpool.carpoolTripList.passengers', { length: trip.passengers?.length || 0 }) }}</span>
          </div>
          <p v-if="trip.notes" class="text-xs text-slate-400 pt-2 border-t border-slate-700/60">{{ trip.notes }}</p>
        </div>

        <!-- Where the amounts come from -->
        <div v-if="linked" class="flex items-start gap-2.5 text-xs bg-slate-800/40 border border-slate-800 rounded-xl px-3 py-2.5">
          <Lock v-if="frozen" class="w-4 h-4 text-slate-400 shrink-0 mt-0.5" />
          <Link2 v-else class="w-4 h-4 text-sky-400 shrink-0 mt-0.5" />
          <div class="min-w-0 space-y-0.5">
            <p class="font-semibold text-slate-200">{{ frozen ? $t('carpool.linked.frozen') : $t('carpool.linked.linked') }}</p>
            <p class="text-slate-400">{{ frozen ? $t('carpool.linked.frozenHint') : $t('carpool.linked.followHint') }}</p>
            <p v-if="frozen && gap !== 0" class="text-slate-300">
              {{ $t('carpool.linked.gap', { amount: fmt(Math.abs(gap)), direction: gap > 0 ? $t('carpool.linked.more') : $t('carpool.linked.less') }) }}
            </p>
          </div>
        </div>

        <!-- Legs, as rows -->
        <div class="space-y-1.5">
          <h4 class="text-xs font-bold text-white flex items-center gap-1.5">
            <Navigation class="w-3.5 h-3.5 text-sky-400" />
            {{ $t('carpool.carpoolTripList.legs', { length: trip.legs.length }) }}
          </h4>
          <component
            :is="leg.drive_id ? 'button' : 'div'"
            v-for="(leg, i) in trip.legs"
            :key="leg.id"
            :type="leg.drive_id ? 'button' : undefined"
            class="w-full text-left flex items-center justify-between gap-3 bg-slate-800/40 border border-slate-800 rounded-xl px-3 py-2 transition-colors"
            :class="leg.drive_id ? 'hover:bg-slate-800/70 hover:border-slate-700' : ''"
            @click="leg.drive_id && emit('open-drive', leg.drive_id)"
          >
            <div class="min-w-0">
              <div class="text-xs text-slate-200 truncate">{{ stops[Number(i)] }} → {{ stops[Number(i) + 1] }}</div>
              <div class="text-xs text-slate-400">
                {{ $t('carpool.carpoolDetailModal.onBoard', { count: 1 + leg.passenger_seats }) }} · {{ $t('carpool.carpoolDetailModal.perPerson', { amount: fmt(leg.cost_per_person) }) }}
              </div>
            </div>
            <div class="flex items-center gap-3 shrink-0">
              <span class="text-xs font-bold text-rose-400">{{ formatDistance(leg.distance_km, 1) }}</span>
              <span class="text-xs font-mono font-bold text-white">{{ fmt(leg.total_cost) }}</span>
              <ChevronRight v-if="leg.drive_id" class="w-4 h-4 text-slate-400" />
            </div>
          </component>
        </div>

        <!-- Same layout as the drive detail: donut on the left, itemized costs on the right -->
        <div class="grid grid-cols-1 md:grid-cols-5 gap-6 items-start">
          <div class="md:col-span-2 bg-slate-800/30 border border-slate-800 rounded-xl p-4 flex flex-col items-center justify-center">
            <h4 class="text-xs font-bold text-white mb-2 self-start">{{ $t('drives.driveCostModal.breakdownTitle') }}</h4>
            <div class="w-full h-56 sm:h-64 relative">
              <CostDonut
                :items="breakdown.items"
                :empty-label="$t('drives.driveCostModal.noCost')"
                :chart-label="$t('drives.driveCostModal.breakdownAria')"
                :currency="vehicleStore.currency"
              />
            </div>
          </div>

          <div class="md:col-span-3 space-y-2.5">
            <template v-for="item in breakdown.items" :key="item.key">
              <CostItemRow
                v-if="item.amount > 0 || item.key !== 'other'"
                :icon="ROWS[item.key as keyof typeof ROWS].icon"
                :tone="ROWS[item.key as keyof typeof ROWS].tone"
                :label="$t(ROWS[item.key as keyof typeof ROWS].label)"
                :amount="item.amount"
                :share-pct="item.sharePct"
                :cost-per-km="item.costPerKm"
                :currency="vehicleStore.currency"
              />
            </template>
          </div>
        </div>

        <!-- Passengers -->
        <div class="space-y-1.5">
          <h4 class="text-xs font-bold text-white">
            <span class="flex items-center gap-1.5">
              <Users class="w-3.5 h-3.5 text-blue-400" />
              {{ $t('carpool.carpoolTripList.passengers', { length: trip.passengers?.length || 0 }) }}
            </span>
          </h4>
          <div v-for="(p, index) in passengers" :key="p.id" class="bg-slate-800/40 border border-slate-800 rounded-xl px-3 py-2 text-xs space-y-1">
            <div class="flex items-center justify-between gap-2">
              <span class="font-semibold text-slate-200 truncate">{{ passengerLabel(p.passenger_name, index) }}</span>
              <span class="font-bold text-success-400 shrink-0">+{{ fmt(p.amount_paid) }}</span>
            </div>
            <div class="text-xs text-slate-400 truncate">
              {{ stops[p.board_stop_index] }} → {{ stops[p.alight_stop_index] }} • {{ $t('carpool.carpoolTripList.seats', p.seats) }}
            </div>
            <div v-if="trip.passengers.length > 1" class="flex items-center justify-between text-xs">
              <span class="text-slate-400">{{ $t('carpool.carpoolTripList.share', { cost_share: fmt(p.cost_share) }) }}</span>
              <span :class="p.balance >= 0 ? 'text-success-400' : 'text-warning-400'">
                {{ p.balance >= 0 ? $t('carpool.above', { amount: fmt(p.balance) }) : $t('carpool.below', { amount: fmt(-p.balance) }) }}
              </span>
            </div>
            <button
              type="button"
              class="inline-flex items-center gap-1.5 pt-0.5 font-semibold text-sky-400 hover:text-sky-300"
              :title="$t('carpool.carpoolDetailModal.copySummaryTitle')"
              @click="copySummary(index)"
            >
              <Check v-if="copiedIndex === index" class="w-3.5 h-3.5" />
              <Copy v-else class="w-3.5 h-3.5" />
              {{ copiedIndex === index ? $t('carpool.carpoolDetailModal.copied') : $t('carpool.carpoolDetailModal.copySummary') }}
            </button>
          </div>
        </div>

        <!-- Balance: the total of the drive detail, then who paid what and what is left, each figure once -->
        <CostTotalBar
          :total="fmt(trip.total_cost)"
          :per-distance="formatCostPerDistance(trip.distance_km > 0 ? trip.total_cost / trip.distance_km : 0, vehicleStore.currency, 3, true)"
          :total-label="$t('drives.driveCostModal.totalCostPrice')"
          :per-distance-label="$t('drives.driveCostModal.costPerKilometre', { unit: distanceUnit() })"
          :strong="trip.net_cost <= 0"
        >
          <!-- What the passengers paid, against what they owed (the tick) -->
          <div class="pt-3 border-t border-slate-700/60 space-y-2">
            <div class="flex items-baseline justify-between gap-2">
              <span class="text-xs font-semibold text-slate-300">{{ $t('carpool.carpoolDetailModal.paidByPassengers') }}</span>
              <span class="text-sm font-bold font-mono" :class="coverage.status === 'below' ? 'text-warning-400' : 'text-success-400'">{{ formatPercent(coverage.paidPct, 0) }}</span>
            </div>
            <div
              class="relative h-2.5 rounded-full bg-slate-950 border border-slate-700/60"
              role="progressbar"
              :aria-valuenow="Math.round(coverage.paidPct)"
              aria-valuemin="0"
              aria-valuemax="100"
              :aria-label="$t('carpool.carpoolDetailModal.paidByPassengers')"
            >
              <div
                class="absolute inset-y-0 left-0 rounded-full transition-all"
                :class="coverage.status === 'below' ? 'bg-warning-500' : 'bg-success-500'"
                :style="{ width: Math.min(100, coverage.paidPct) + '%' }"
              ></div>
              <div
                class="absolute -top-1 -bottom-1 w-0.5 rounded bg-white"
                :style="{ left: `calc(${Math.min(100, coverage.fairPct)}% - 1px)` }"
                :title="$t('carpool.carpoolDetailModal.fairShare', { percent: coverage.fairPct.toFixed(0) })"
              ></div>
            </div>
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-x-3 gap-y-0.5 text-xs text-slate-400">
              <span>
                {{ $t('carpool.carpoolDetailModal.received', { amount: fmt(coverage.paid) }) }} ·
                {{ $t('carpool.carpoolDetailModal.fairShareAmount', { percent: coverage.fairPct.toFixed(0), amount: fmt(coverage.fair) }) }}
              </span>
              <span v-if="coverage.status !== 'fair'" :class="balance > 0 ? 'text-success-400' : 'text-warning-400'">
                {{ balance > 0 ? $t('carpool.above', { amount: fmt(balance) }) : $t('carpool.carpoolSummaryGrid.below', { amount: fmt(-balance) }) }}
              </span>
            </div>
          </div>

          <div class="pt-3 border-t border-slate-700/60 flex flex-col sm:flex-row sm:items-center justify-between gap-2 text-xs text-slate-300">
            <div class="flex items-center gap-2">
              <CheckCircle2 v-if="trip.net_cost <= 0" class="w-4 h-4 text-success-400 shrink-0" />
              <Sparkles v-else class="w-4 h-4 text-warning-400 shrink-0" />
              <span>{{ $t('carpool.carpoolTripList.driverSShare') }} <strong>{{ fmt(trip.driver_cost_share) }}</strong></span>
            </div>
            <div>
              <template v-if="trip.net_cost > 0">
                {{ $t('carpool.carpoolTripList.leftToTheDriver') }} <strong class="text-white text-sm">{{ fmt(trip.net_cost) }}</strong>
              </template>
              <span v-else class="font-bold text-success-400 text-sm">{{ $t('carpool.carpoolTripList.netSurplus', { net_cost: fmt(Math.abs(trip.net_cost)) }) }}</span>
            </div>
          </div>
        </CostTotalBar>

    <template #footer>
        <button @click="open = false" class="btn btn-lg btn-secondary">
          {{ $t('common.close') }}
        </button>
        <div v-if="vehicleStore.canEdit" class="flex items-center gap-2">
          <button
            v-if="linked"
            type="button"
            @click="frozen ? emit('unfreeze', trip) : emit('freeze', trip)"
            :disabled="recalculating"
            class="btn btn-secondary"
          >
            <LockOpen v-if="frozen" class="w-3.5 h-3.5" />
            <Lock v-else class="w-3.5 h-3.5" />
            <span>{{ frozen ? $t('carpool.linked.reopen') : $t('carpool.linked.freeze') }}</span>
          </button>
          <button
            v-else
            type="button"
            @click="emit('recalculate', trip)"
            :disabled="recalculating"
            class="btn btn-secondary"
            :title="$t('carpool.carpoolTripList.recalculateTheActualCostsOf')" :aria-label="$t('carpool.carpoolTripList.recalculateTheActualCostsOf')"
          >
            <RotateCw class="w-3.5 h-3.5" :class="{ 'animate-spin': recalculating }" />
          </button>
          <button
            type="button"
            @click="emit('edit', trip)"
            class="btn btn-lg btn-primary"
          >
            <Pencil class="w-4 h-4" />
            <span>{{ $t('common.edit') }}</span>
          </button>
        </div>
    </template>
  </ModalShell>
</template>
