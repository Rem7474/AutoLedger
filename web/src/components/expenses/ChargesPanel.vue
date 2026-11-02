<script setup lang="ts">
import ListSkeleton from '@/components/ListSkeleton.vue'
import EmptyState from '@/components/EmptyState.vue'
import EmptySourceHints from '@/components/EmptySourceHints.vue'
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { intlLocale } from '@/i18n'
import { useVehicleStore } from '@/stores/vehicle'
import { Zap } from 'lucide-vue-next'
import ChargeCard from '@/components/expenses/ChargeCard.vue'
import ToQualifyFilter from '@/components/drives/ToQualifyFilter.vue'
import { groupChargesByMonth } from '@/utils/expenses'
import { formatAmount } from '@/currency'
import { formatNumber } from '@/utils/numbers'
import { distanceUnit, formatPerDistanceValue, perDistance } from '@/units'
import { hasEnergyEstimate } from '@/utils/vehicles'

const props = defineProps<{
  charges: any[]
  chargesTotal: number
  chargesWithoutCost: number
  loading: boolean
  loadingMoreCharges: boolean
  missingCostOnly: boolean
}>()
const emit = defineEmits<{
  'toggle-missing-cost': []
  'import-csv': []
  'load-more': []
  edit: [charge: any]
  'review-cost': [charge: any, reviewed: boolean]
  delete: [charge: any]
  'view-document': [docId: string | null | undefined, filename?: string | null, download?: boolean]
}>()
const router = useRouter()
const vehicleStore = useVehicleStore()

// The charges loaded so far, which is the whole list until "load more" is needed.
const months = computed(() => groupChargesByMonth(props.charges, vehicleStore.currency))
const monthLabel = (date: string) => new Date(date).toLocaleDateString(intlLocale(), { month: 'long', year: 'numeric' })
</script>

<template>
  <div class="space-y-3">
    <!-- Energy estimate banner if configured -->
    <div
      v-if="hasEnergyEstimate(vehicleStore.activeVehicle)"
      class="p-3 bg-info-500/10 border border-info-500/20 rounded-2xl flex items-center justify-between text-xs text-info-300"
    >
      <div class="flex items-center gap-2">
        <Zap class="w-4 h-4 shrink-0 text-info-400" />
        <span>
          {{ $t('expenses.chargesPanel.energyEstimateActive') }}
          <strong>{{ $t('expenses.chargesPanel.kwh100km', { unit: distanceUnit(), estimated_kwh_100km: formatPerDistanceValue(Number(vehicleStore.activeVehicle.estimated_kwh_100km)) }) }}</strong> {{ $t('expenses.chargesPanel.at') }}
          <strong>{{ $t('expenses.chargesPanel.kwh3', { estimated_price_per_kwh: formatAmount(Number(vehicleStore.activeVehicle.estimated_price_per_kwh), vehicleStore.currency, 4) }) }}</strong>
          {{ $t('expenses.chargesPanel.automaticallyIncludedInTheTco') }}
        </span>
      </div>
      <button
        @click="router.push('/vehicles')"
        class="shrink-0 font-medium underline hover:text-info-200 transition-colors ml-2"
      >
        {{ $t('common.edit') }}
      </button>
    </div>

    <div v-if="chargesWithoutCost > 0 || missingCostOnly" class="flex flex-wrap items-center gap-2">
      <ToQualifyFilter
        :count="chargesWithoutCost"
        :active="missingCostOnly"
        :label="$t('expenses.costReview.toComplete', { count: chargesWithoutCost })"
        :title="$t('expenses.costReview.toCompleteHint')"
        @toggle="emit('toggle-missing-cost')"
      />
    </div>
    <ListSkeleton v-if="loading" />
    <EmptyState v-else-if="!charges.length" :icon="Zap" :title="$t('expenses.chargesPanel.empty')">
      <template #actions>
        <EmptySourceHints quick-kind="CHARGE" @import-csv="emit('import-csv')" />
      </template>
    </EmptyState>
    <div v-else class="space-y-3">
      <section v-for="month in months" :key="month.key" class="space-y-2">
        <header class="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-0.5 px-1 pt-2">
          <h3 class="text-sm font-bold capitalize text-white">{{ monthLabel(month.date) }}</h3>
          <p class="text-xs text-slate-400">
            {{ $t('expenses.chargesPanel.monthSummary', { count: $t('expenses.chargesPanel.monthCount', { count: month.charges.length }, month.charges.length), kwh: formatNumber(month.kwh, 1), cost: formatAmount(month.cost, vehicleStore.currency) }) }}
            <template v-if="month.pricePerKwh !== null"> · {{ $t('expenses.chargesPanel.averagePrice', { price: formatAmount(month.pricePerKwh, vehicleStore.currency, 3) }) }}</template>
          </p>
        </header>
        <ChargeCard
          v-for="c in month.charges"
          :key="c.id"
          :c="c"
          @open="emit('edit', $event)"
          @delete="emit('delete', $event)"
          @review-cost="(charge, reviewed) => emit('review-cost', charge, reviewed)"
          @view-document="(id, name, download) => emit('view-document', id, name, download)"
        />
      </section>
      <div class="flex items-center justify-between text-xs text-slate-400 px-1">
        <span>{{ $t('expenses.chargesPanel.chargeSShownOutOf', { length: charges.length, chargesTotal }) }}</span>
        <button
          v-if="charges.length < chargesTotal"
          @click="emit('load-more')"
          :disabled="loadingMoreCharges"
          class="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 font-semibold rounded-xl border border-slate-700 disabled:opacity-50"
        >
          {{ loadingMoreCharges ? $t('common.loading') : $t('expenses.chargesPanel.loadMore') }}
        </button>
      </div>
    </div>
  </div>
</template>
