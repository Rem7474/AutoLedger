<script setup lang="ts">
import ListSkeleton from '@/components/ListSkeleton.vue'
import CardOpenButton from '@/components/CardOpenButton.vue'
import RowMenu from '@/components/RowMenu.vue'
import EmptyState from '@/components/EmptyState.vue'
import EmptySourceHints from '@/components/EmptySourceHints.vue'
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { intlLocale } from '@/i18n'
import { useVehicleStore } from '@/stores/vehicle'
import { Zap, Trash2, AlertTriangle, Paperclip, ExternalLink } from 'lucide-vue-next'
import { groupChargesByMonth } from '@/utils/expenses'
import { formatChargeWindow } from '@/utils/dates'
import { formatAmount } from '@/currency'
import { formatNumber } from '@/utils/numbers'
import { chargeDeletable, chargeOrigin, chargeOriginKey } from '@/utils/chargeProvenance'
import { teslamateChargeCostUrl, teslamateChargeUrl } from '@/utils/teslamateCharges'
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

    <button
      v-if="chargesWithoutCost > 0 || missingCostOnly"
      @click="emit('toggle-missing-cost')"
      class="w-full p-3 rounded-2xl text-left text-xs font-semibold flex items-center gap-2 border transition-colors"
      :class="missingCostOnly ? 'bg-warning-500/20 border-warning-500/40 text-warning-300' : 'bg-warning-500/10 border-warning-500/20 text-warning-400 hover:bg-warning-500/15'"
    >
      <AlertTriangle class="w-4 h-4 shrink-0" />
      <span v-if="missingCostOnly">{{ $t('expenses.chargesPanel.showingOnlyTheChargesWithout') }}</span>
      <span v-else>{{ $t('expenses.chargesPanel.chargeSWithoutACost', { chargesWithoutCost }) }}</span>
    </button>
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
        <div
          v-for="c in month.charges"
          :key="c.id"
          class="relative bg-slate-900 border hover:border-slate-700/90 p-3 rounded-2xl flex items-center gap-2 sm:gap-3 transition-colors"
          :class="c.cost === null ? 'border-warning-500/40' : 'border-slate-800'"
        >
          <!-- The row opens the charge; the receipt and the TeslaMate links above it keep their own click -->
          <CardOpenButton
            v-if="vehicleStore.canEdit"
            :label="chargeDeletable(c) ? $t('expenses.chargesPanel.editThisCharge') : $t('expenses.chargesPanel.fixCost')"
            @click="emit('edit', c)"
          />
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-x-1.5 gap-y-1 sm:gap-x-2 flex-wrap">
              <span class="text-sm font-bold text-info-400 shrink-0">
                {{ $t('expenses.chargesPanel.kwh2', { kwh_added: formatNumber(c.kwh_added, 2) }) }}
              </span>
              <span class="text-[11px] sm:text-xs px-1.5 sm:px-2 py-0.5 rounded-full bg-slate-800 text-slate-300 border border-slate-700 shrink-0">{{ $t(chargeOriginKey(c)) }}</span>
              <span v-if="chargeOrigin(c) === 'TESLAMATE' && c.cost_source === 'MANUAL'" class="text-[11px] sm:text-xs px-1.5 sm:px-2 py-0.5 rounded-full bg-slate-800 text-slate-300 border border-slate-700 shrink-0">{{ $t('expenses.chargesPanel.correctedCost') }}</span>
              <button
                v-if="c.document_id"
                @click="emit('view-document', c.document_id, c.document_filename, false)"
                class="relative z-10 text-xs px-2 py-0.5 rounded-lg bg-sky-500/10 hover:bg-sky-500/20 text-sky-400 border border-sky-500/20 flex items-center gap-1 transition-colors max-w-[200px] truncate"
                :title="$t('expenses.chargesPanel.viewTheReceipt')"
              >
                <Paperclip class="w-3 h-3 shrink-0" />
                <span class="truncate">{{ c.document_filename || $t('expenses.invoice') }}</span>
              </button>
            </div>
            <p class="text-xs text-slate-400 mt-0.5 truncate" :title="c.address">
              {{ formatChargeWindow(c.date, c.end_date) }} · <span class="text-slate-300">{{ c.address || $t('expenses.chargesPanel.unknownPlace') }}</span>
            </p>
          </div>
          <div class="text-right shrink-0">
            <template v-if="c.cost !== null">
              <span class="text-base font-extrabold text-white">{{ formatAmount(c.cost, c.currency || vehicleStore.currency) }}</span>
              <p v-if="c.kwh_added > 0" class="text-xs text-slate-400">
                {{ $t('expenses.chargesPanel.kwh', { cost: formatAmount(c.cost / c.kwh_added, c.currency || vehicleStore.currency, 3) }) }}
              </p>
            </template>
            <span v-else class="text-xs font-bold text-warning-400 flex items-center gap-1">
              <AlertTriangle class="w-3.5 h-3.5" /> {{ $t('expenses.chargesPanel.missingCost') }}
            </span>
            <a
              v-if="teslamateChargeCostUrl(vehicleStore.activeVehicle, c)"
              :href="teslamateChargeCostUrl(vehicleStore.activeVehicle, c)!"
              target="_blank"
              rel="noopener noreferrer"
              class="relative z-10 mt-0.5 inline-flex items-center gap-1 text-xs text-info-400 hover:text-info-300 font-medium"
              :title="$t('expenses.teslamateLinks.setCostHint')"
            >
              <ExternalLink class="w-3 h-3" /> {{ $t('expenses.teslamateLinks.setCost') }}
            </a>
          </div>
          <RowMenu
            v-if="vehicleStore.canEdit && (teslamateChargeUrl(vehicleStore.activeVehicle, c) || chargeDeletable(c))"
            :label="$t('common.moreActions')"
            :heading="formatChargeWindow(c.date, c.end_date)"
          >
            <a
              v-if="teslamateChargeUrl(vehicleStore.activeVehicle, c)"
              :href="teslamateChargeUrl(vehicleStore.activeVehicle, c)!"
              target="_blank"
              rel="noopener noreferrer"
              role="menuitem"
              class="menu-item"
            >
              <ExternalLink class="w-3.5 h-3.5 text-slate-400" />{{ $t('expenses.teslamateLinks.details') }}
            </a>
            <button v-if="chargeDeletable(c)" type="button" role="menuitem" class="menu-item hover:!text-rose-300" @click="emit('delete', c)">
              <Trash2 class="w-3.5 h-3.5 text-slate-400" />{{ $t('common.delete') }}
            </button>
          </RowMenu>
        </div>
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
