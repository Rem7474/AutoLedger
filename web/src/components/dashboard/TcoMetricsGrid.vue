<script setup lang="ts">
import { intlLocale, t } from '@/i18n'
import { computed, ref } from 'vue'
import { canCharge, canRefuel, isFuelOnly } from '@/utils/vehicles'
import { useVehicleStore } from '@/stores/vehicle'
import { formatAmount, formatAmountOrUnknown } from '@/currency'
import GlossaryTip from '@/components/GlossaryTip.vue'
import KpiCard from '@/components/KpiCard.vue'
import KpiRow from '@/components/KpiRow.vue'
import ModalShell from '@/components/ModalShell.vue'
import { useIsMobile } from '@/composables/useIsMobile'
import TcoDetailsList from '@/components/dashboard/TcoDetailsList.vue'
import { currentMonthStats as buildCurrentMonthStats, formatMonthName, rollingWindowStats } from '@/utils/dashboard'
import { formatCostPerDistance } from '@/utils/costPerDistance'
import { Coins, Zap, Calendar, TrendingUp, ChevronDown } from 'lucide-vue-next'
import { distanceUnit, formatDistanceValue, formatFuelConsumptionValue, fuelConsumptionUnit, litresToDisplayVolume, perDistance, perVolume, volumeUnitLabel } from '@/units'

// The four figures the page opens on, each with the period it covers; the other totals sit in the details below
const props = defineProps<{ tco: any | null }>()
const emit = defineEmits<{ 'open-month': [month: any] }>()
const vehicleStore = useVehicleStore()
// Every TCO figure is in the vehicle's own currency
const money = (v: number, digits = 2) => formatAmount(v || 0, vehicleStore.currency, digits)
// A headline amount drops its cents from 1 000 on: they are noise next to the digits that matter
const headline = (v: number) => money(v, Math.abs(v || 0) >= 1000 ? 0 : 2)
const detailsRef = ref<HTMLDetailsElement | null>(null)
// A phone opens the totals in a sheet: a block unfolded below the cards would change nothing on the screen
const isMobile = useIsMobile()
const detailsSheet = ref(false)
const showDetails = () => {
  if (isMobile.value) {
    detailsSheet.value = true
    return
  }
  if (detailsRef.value) detailsRef.value.open = true
  detailsRef.value?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
}
// The cost per 100 distance units lives in the energy summary: one definition, not a second figure on this card
const showEnergyPanel = () => {
  const panel = document.getElementById('energy-efficiency')
  if (panel instanceof HTMLDetailsElement) panel.open = true
  panel?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
// The last calendar months first: a recent change shows there, the since-purchase figures stay as secondary text
const rolling = computed(() => rollingWindowStats(props.tco?.monthly_costs))
const month = computed(() => buildCurrentMonthStats(props.tco?.monthly_costs, new Date(), props.tco?.current_month))
// A hybrid's energy is its charges and its fill-ups: the label names both
const energyLabel = computed(() => {
  const powertrain = props.tco?.powertrain
  if (isFuelOnly(powertrain)) return t('dashboard.tcoMetricsGrid.fuel')
  return canRefuel(powertrain) ? t('dashboard.tcoMetricsGrid.energyHybrid') : t('dashboard.tcoMetricsGrid.energy')
})
// The window the since-purchase figures are compared with, shown under the label of the cards it applies to
const windowLabel = computed(() => (rolling.value ? t('dashboard.tcoMetricsGrid.windowMonths', { months: rolling.value.months }, rolling.value.months) : undefined))
// Estimated energy inside the figure the card shows: the window's own, or the since-purchase total without one
const estimatedEnergy = computed(() => (rolling.value ? rolling.value.estimatedEnergy : props.tco?.estimated_energy_cost) || 0)
// The energy card covers the same window as the cards beside it; the since-purchase total without monthly rows
const energyCost = computed(() => (rolling.value ? rolling.value.energy : props.tco?.energy_cost) || 0)
const energyPerKm = computed(() => (rolling.value ? rolling.value.energyPerKm : (props.tco?.energy_cost_per_km ?? null)))
// Unknown (no distance yet) reads as a dash, not as 0,00
const perUnit = (v: number | null | undefined) => formatAmountOrUnknown(v == null ? null : perDistance(v), vehicleStore.currency, 3)
// Economic cost per km over the window; the since-purchase full cost when the monthly rows are missing
const cashPaid = computed(() => headline(rolling.value ? rolling.value.cash : props.tco?.total_cost))
const monthHint = computed(() => {
  const m = month.value
  if (!m) return undefined
  const parts = [t('dashboard.tcoMetricsGrid.spentSoFar'), `${t('dashboard.tcoMetricsGrid.monthDistance', { unit: distanceUnit(), distance_km: formatDistanceValue(m.distance_km) })} · ${formatCostPerDistance(m.cost_per_km, vehicleStore.currency, 3, true)}`]
  if (m.scheduled > 0) parts.push(t('dashboard.tcoMetricsGrid.alreadyScheduled', { amount: money(m.scheduled) }))
  return parts.join(' · ')
})
const cashHint = computed(() => [windowLabel.value, estimatedEnergy.value ? t('dashboard.tcoMetricsGrid.ofWhichEstimatedEnergy', { amount: money(estimatedEnergy.value, 0) }) : null].filter(Boolean).join(' · ') || undefined)
const energyHint = computed(() => [windowLabel.value, `${perUnit(energyPerKm.value)}/${distanceUnit()}`].filter(Boolean).join(' · '))
const costPerKm = computed(() => (rolling.value ? rolling.value.economicPerKm : (props.tco?.full_cost_per_km ?? null)))
</script>

<template>
  <section class="space-y-3">
    <!-- Phone: the figure the page is read for, then the other three as a short list, each row leading somewhere visible -->
    <div v-if="isMobile" class="space-y-3">
      <KpiCard
        lead
        :label="rolling ? $t('dashboard.tcoMetricsGrid.economicPerKm', { unit: distanceUnit() }) : $t('dashboard.tcoMetricsGrid.fullCostPerKm', { unit: distanceUnit() })"
        :period="windowLabel"
        term="economicCost"
        :icon="TrendingUp"
        tone="success"
        :value="costPerKm == null ? null : perUnit(costPerKm)"
        :unit="`/${distanceUnit()}`"
        :action-label="$t('dashboard.tcoMetricsGrid.details')"
        @action="showDetails"
      >
        <p v-if="tco && tco.depreciation_known === false" class="mt-2 text-xs font-medium text-warning-400">
          {{ $t('dashboard.tcoMetricsGrid.depreciationMissing') }}
          <router-link to="/vehicles" class="relative z-10 underline">{{ $t('dashboard.tcoMetricsGrid.depreciationMissingAction') }}</router-link>
        </p>
        <p v-else-if="rolling && tco?.full_cost_per_km" class="mt-2 text-xs text-slate-300">
          {{ $t('dashboard.tcoMetricsGrid.sincePurchasePerKm', { amount: perUnit(tco.full_cost_per_km), unit: distanceUnit() }) }}
        </p>
      </KpiCard>
      <div class="divide-y divide-slate-800 overflow-hidden card">
        <KpiRow
          v-if="month"
          :label="$t('dashboard.tcoMetricsGrid.currentMonth', { month: formatMonthName(month.month) })"
          :value="headline(month.total - month.scheduled)"
          :hint="monthHint"
          @action="emit('open-month', month.raw)"
        />
        <KpiRow :label="rolling ? $t('dashboard.tcoMetricsGrid.cashPaid') : $t('dashboard.tcoMetricsGrid.totalCost')" :value="cashPaid" :hint="cashHint" to="/expenses" />
        <KpiRow :label="energyLabel" :value="headline(energyCost)" :hint="energyHint" tone="sky" to="/energy" />
      </div>
      <Teleport to="body">
        <ModalShell v-model:open="detailsSheet" :title="$t('dashboard.tcoMetricsGrid.details')" size="md">
          <TcoDetailsList :tco="tco" />
        </ModalShell>
      </Teleport>
    </div>

    <div v-else class="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
      <!-- Economic cost per km -->
      <KpiCard
        aligned
        lead
        class="col-span-2 lg:col-span-1"
        :label="rolling ? $t('dashboard.tcoMetricsGrid.economicPerKm', { unit: distanceUnit() }) : $t('dashboard.tcoMetricsGrid.fullCostPerKm', { unit: distanceUnit() })"
        :period="windowLabel"
        term="economicCost"
        :icon="TrendingUp"
        tone="success"
        :value="costPerKm == null ? null : perUnit(costPerKm)"
        :unit="`/${distanceUnit()}`"
        :action-label="$t('dashboard.tcoMetricsGrid.details')"
        @action="showDetails"
      >
        <p v-if="tco && tco.depreciation_known === false" class="mt-2 text-xs font-medium text-warning-400">
          {{ $t('dashboard.tcoMetricsGrid.depreciationMissing') }}
          <router-link to="/vehicles" class="relative z-10 underline">{{ $t('dashboard.tcoMetricsGrid.depreciationMissingAction') }}</router-link>
        </p>
        <p v-else-if="rolling && tco?.full_cost_per_km" class="mt-2 text-xs text-slate-300">
          {{ $t('dashboard.tcoMetricsGrid.sincePurchasePerKm', { amount: perUnit(tco.full_cost_per_km), unit: distanceUnit() }) }}
        </p>
      </KpiCard>

      <!-- Cash paid -->
      <KpiCard
        aligned
        :label="rolling ? $t('dashboard.tcoMetricsGrid.cashPaid') : $t('dashboard.tcoMetricsGrid.totalCost')"
        :period="windowLabel"
        term="cashPaid"
        :icon="Coins"
        :value="headline(rolling ? rolling.cash : tco?.total_cost)"
        :action-label="$t('dashboard.tcoMetricsGrid.viewExpenses')"
        to="/expenses"
      >
        <p v-if="estimatedEnergy" class="mt-2 text-xs text-slate-300">
          {{ $t('dashboard.tcoMetricsGrid.ofWhichEstimatedEnergy', { amount: money(estimatedEnergy, 0) }) }}
          <span class="relative z-10"><GlossaryTip term="estimatedEnergy" /></span>
        </p>
      </KpiCard>

      <!-- Current month -->
      <KpiCard
        aligned
        :label="month ? $t('dashboard.tcoMetricsGrid.currentMonth', { month: formatMonthName(month.month) }) : $t('dashboard.tcoMetricsGrid.currentMonthShort')"
        :icon="Calendar"
        :value="month ? headline(month.total - month.scheduled) : null"
        :action-label="month ? $t('dashboard.tcoMetricsGrid.viewTheBreakdown') : undefined"
        @action="month && emit('open-month', month.raw)"
      >
        <template v-if="month">
          <p class="mt-1 text-xs text-slate-300">{{ $t('dashboard.tcoMetricsGrid.spentSoFar') }}</p>
          <p v-if="month.scheduled > 0" class="mt-0.5 text-xs text-slate-300">
            {{ $t('dashboard.tcoMetricsGrid.alreadyScheduled', { amount: money(month.scheduled) }) }}
          </p>
          <p class="mt-2 text-xs text-slate-300">
            {{ $t('dashboard.tcoMetricsGrid.monthDistance', { unit: distanceUnit(), distance_km: formatDistanceValue(month.distance_km) }) }}
            · {{ formatCostPerDistance(month.cost_per_km, vehicleStore.currency, 3, true) }}
          </p>
        </template>
      </KpiCard>

      <!-- Energy -->
      <KpiCard
        aligned
        class="col-span-2 lg:col-span-1"
        :label="energyLabel"
        :period="windowLabel"
        :icon="Zap"
        tone="sky"
        :value="headline(energyCost)"
        :action-label="canCharge(tco?.powertrain) ? $t('dashboard.tcoMetricsGrid.energyPanelLink') : undefined"
        @action="showEnergyPanel"
      >
        <p class="mt-2 text-xs text-slate-300">{{ perUnit(energyPerKm) }}/{{ distanceUnit() }}</p>
      </KpiCard>
    </div>

    <details v-if="!isMobile" ref="detailsRef" class="rounded-2xl border border-slate-800 bg-slate-900/60 px-4 py-2 text-xs text-slate-300 group">
      <summary class="flex cursor-pointer list-none items-center justify-between gap-2 font-semibold text-slate-300">
        {{ $t('dashboard.tcoMetricsGrid.details') }}
        <ChevronDown class="h-4 w-4 transition-transform group-open:rotate-180" aria-hidden="true" />
      </summary>
      <TcoDetailsList class="mt-3" :tco="tco" />
    </details>
  </section>
</template>
