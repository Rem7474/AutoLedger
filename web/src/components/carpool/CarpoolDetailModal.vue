<script setup lang="ts">
import { computed } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import { Users, MapPin, Navigation, Pencil, RotateCw, Link2, Lock, LockOpen, ChevronRight, Zap, Disc, Wrench, Shield, Receipt } from 'lucide-vue-next'
import CostDonut from '@/components/costs/CostDonut.vue'
import CostItemRow from '@/components/costs/CostItemRow.vue'
import { buildCarpoolBreakdown } from '@/utils/costBreakdown'
import CarpoolRecap from '@/components/carpool/CarpoolRecap.vue'
import CarpoolRouteStrip from '@/components/carpool/CarpoolRouteStrip.vue'
import { allocate, cents, driveCostGapCents, formatDate, hasLinkedLegs, passengerLabel, stopNames, titleShowsRoute } from '@/utils/carpool'
import { formatAmount } from '@/currency'
import ModalShell from '@/components/ModalShell.vue'
import CostTotalBar from '@/components/CostTotalBar.vue'
import { formatCostPerDistance } from '@/utils/costPerDistance'
import { distanceUnit, formatDistance } from '@/units'

// Detail of a carpool trip: summary, where the amounts come from, the recap (what it costs, what was received, what is left
// to the driver), the passengers on the route, the legs, the cost split and the total of the drive and trip cost breakdown. A leg made from a drive opens the cost detail of that drive, like the legs of a trip. The form to change the
// carpool is the edit modal, reached from the footer.
const props = defineProps<{ trip: any | null; recalculating: boolean }>()
const emit = defineEmits<{ edit: [trip: any]; recalculate: [trip: any]; freeze: [trip: any]; unfreeze: [trip: any]; 'open-drive': [driveId: string] }>()
const open = defineModel<boolean>('open', { required: true })
const vehicleStore = useVehicleStore()
// Carpool amounts are in the vehicle's own currency
const fmt = (v: number) => formatAmount(Number(v || 0), vehicleStore.currency)

const breakdown = computed(() => buildCarpoolBreakdown(props.trip))
// The recap reads the carpool the way the editor does: the shares come from the legs and the seats on board
const live = computed(() => allocate(props.trip?.legs || [], props.trip?.passengers || []))
const revenue = computed(() => passengers.value.reduce((sum, p) => sum + cents(p.amount_paid), 0))
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
// The default title is the route: the route is written again only for a title the user changed
const routeRepeated = computed(() => titleShowsRoute(props.trip?.title, stops.value))
const passengers = computed<any[]>(() => props.trip?.passengers || [])

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
            <p class="text-xs text-slate-400">{{ formatDate(trip.date) }} · {{ formatDistance(trip.distance_km, 1) }}</p>
          </div>
        </div>
    </template>

        <!-- The route, when the title does not say it, and the notes -->
        <div v-if="!routeRepeated || trip.notes" class="bg-slate-800/60 border border-slate-700/60 p-3.5 rounded-2xl space-y-2">
          <div v-if="!routeRepeated" class="text-sm font-semibold text-white flex items-center gap-2">
            <MapPin class="w-4 h-4 text-rose-400 shrink-0" />
            <span class="truncate">{{ stops[0] }}</span>
            <span class="text-slate-400">→</span>
            <span class="truncate">{{ stops[stops.length - 1] }}</span>
          </div>
          <p v-if="trip.notes" class="text-xs text-slate-400" :class="routeRepeated ? '' : 'pt-2 border-t border-slate-700/60'">{{ trip.notes }}</p>
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

        <CarpoolRecap :live="live" :revenue="revenue" :currency="vehicleStore.currency" />

        <!-- Passengers, each on the route -->
        <div class="space-y-1.5">
          <h4 class="text-xs font-bold text-white flex items-center gap-1.5">
            <Users class="w-3.5 h-3.5 text-sky-400" />
            {{ $t('carpool.carpoolTripList.passengers', { length: trip.passengers?.length || 0 }) }}
          </h4>
          <div v-for="(p, index) in passengers" :key="p.id" class="bg-slate-800/40 border border-slate-800 rounded-xl px-3 py-2.5 text-xs space-y-1.5">
            <div class="flex items-baseline justify-between gap-2">
              <span class="font-semibold text-slate-200 truncate">{{ passengerLabel(p.passenger_name, index) }} <span class="font-normal text-slate-400">· {{ $t('carpool.carpoolTripList.seats', p.seats) }}</span></span>
              <span class="shrink-0 tabular-nums" :title="$t('carpool.carpoolDetailModal.receivedOfShare')"><strong class="text-emerald-300">{{ fmt(p.amount_paid) }}</strong> <span class="text-slate-400">/ {{ fmt(p.cost_share) }}</span></span>
            </div>
            <CarpoolRouteStrip readonly :stops="stops" :board="p.board_stop_index" :alight="p.alight_stop_index" />
            <p v-if="p.balance !== 0" class="text-slate-400">{{ p.balance > 0 ? $t('carpool.above', { amount: fmt(p.balance) }) : $t('carpool.below', { amount: fmt(-p.balance) }) }}</p>
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

        <CostTotalBar
          :total="fmt(trip.total_cost)"
          :per-distance="formatCostPerDistance(trip.distance_km > 0 ? trip.total_cost / trip.distance_km : 0, vehicleStore.currency, 3, true)"
          :total-label="$t('drives.driveCostModal.totalCostPrice')"
          :per-distance-label="$t('drives.driveCostModal.costPerKilometre', { unit: distanceUnit() })"
          :strong="trip.net_cost <= 0"
        />

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
