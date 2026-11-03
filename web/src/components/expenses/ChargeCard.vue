<script setup lang="ts">
import { computed } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import { MapPin, Coins, Paperclip, Trash2, ExternalLink, AlertTriangle, Zap, Check } from 'lucide-vue-next'
import CardOpenButton from '@/components/CardOpenButton.vue'
import RowMenu from '@/components/RowMenu.vue'
import QualifyActions from '@/components/drives/QualifyActions.vue'
import { formatChargeWindow } from '@/utils/dates'
import { formatAmount } from '@/currency'
import { formatNumber } from '@/utils/numbers'
import { chargeDeletable, chargeOrigin, chargeOriginKey } from '@/utils/chargeProvenance'
import { chargeCostToReview, teslamateChargeCostUrl, teslamateChargeUrl } from '@/utils/teslamateCharges'

// One charge of the list, laid out like a drive (DriveCard): a click opens it, the cost to complete is a two tap
// queue (enter it, or confirm a 0 as free), the TeslaMate cost link stays at hand, and the rest sits in the "..." menu.
// On a phone the card follows the drive card: a glyph, the place and the amount on the first line, the details under
// them, the two taps across the full width, and the secondary links in the menu (a bottom sheet).
const props = defineProps<{ c: any }>()
const emit = defineEmits<{
  open: [charge: any]
  delete: [charge: any]
  'review-cost': [charge: any, reviewed: boolean]
  'view-document': [docId: string | null | undefined, filename?: string | null, download?: boolean]
}>()
const vehicleStore = useVehicleStore()
const currency = computed(() => props.c.currency || vehicleStore.currency)
const review = computed(() => chargeCostToReview(props.c))
const toComplete = computed(() => vehicleStore.canEdit && review.value)
// What stands in the place of the amount while the cost is to be completed: nothing, or a 0 that may be a free charge
const reviewLabel = computed(() => (props.c.cost === null || props.c.cost === undefined ? 'expenses.chargesPanel.missingCost' : 'expenses.costReview.toConfirm'))
const detailsUrl = computed(() => teslamateChargeUrl(vehicleStore.activeVehicle, props.c))
const setCostUrl = computed(() => teslamateChargeCostUrl(vehicleStore.activeVehicle, props.c))
</script>

<template>
  <div
    class="relative bg-slate-900 border hover:border-slate-700/90 p-3 sm:p-4 rounded-2xl transition-all flex flex-col lg:flex-row lg:items-center justify-between gap-2 sm:gap-4 group"
    :class="review ? 'border-warning-500/40' : 'border-slate-800'"
  >
    <CardOpenButton
      v-if="vehicleStore.canEdit"
      :label="$t('expenses.costReview.openCharge', { date: formatChargeWindow(c.date, c.end_date) })"
      @click="emit('open', c)"
    />

    <!-- Phone -->
    <div class="flex items-start gap-3 sm:hidden">
      <span
        class="grid size-10 shrink-0 place-items-center rounded-full"
        :class="review ? 'bg-warning-500/15 text-warning-400' : 'bg-slate-800 text-slate-300'"
        aria-hidden="true"
      >
        <Zap class="size-5" />
      </span>
      <div class="min-w-0 flex-1">
        <div class="flex items-start justify-between gap-3">
          <p class="min-w-0 truncate text-sm font-semibold text-white" :title="c.address">{{ c.address || $t('expenses.chargesPanel.unknownPlace') }}</p>
          <div class="shrink-0 text-right">
            <p v-if="review" class="text-sm font-bold text-warning-400">{{ $t(reviewLabel) }}</p>
            <template v-else-if="c.cost !== null">
              <p class="text-sm font-extrabold tabular-nums text-white">{{ formatAmount(c.cost, currency) }}</p>
              <p v-if="c.kwh_added > 0" class="text-xs tabular-nums text-slate-400">{{ $t('expenses.chargesPanel.kwh', { cost: formatAmount(c.cost / c.kwh_added, currency, 3) }) }}</p>
            </template>
          </div>
        </div>
        <p class="mt-0.5 text-xs text-slate-400">{{ formatChargeWindow(c.date, c.end_date, false) }}</p>
        <div class="mt-1.5 flex flex-wrap items-center gap-1.5">
          <span class="rounded-full border border-slate-700/60 bg-slate-800 px-2 py-0.5 text-xs font-bold tabular-nums text-slate-200">{{ $t('expenses.chargesPanel.kwh2', { kwh_added: formatNumber(c.kwh_added, 2) }) }}</span>
          <span class="rounded-full border border-slate-700 bg-slate-800 px-2 py-0.5 text-xs text-slate-300">{{ $t(chargeOriginKey(c)) }}</span>
          <span v-if="chargeOrigin(c) === 'TESLAMATE' && c.cost_source === 'MANUAL'" class="rounded-full border border-slate-700 bg-slate-800 px-2 py-0.5 text-xs text-slate-300">{{ $t('expenses.chargesPanel.correctedCost') }}</span>
          <button
            v-if="c.document_id"
            type="button"
            @click="emit('view-document', c.document_id, c.document_filename, false)"
            class="relative z-10 flex max-w-full items-center gap-1 truncate rounded-full border border-sky-500/20 bg-sky-500/10 px-2 py-0.5 text-xs text-sky-400"
            :title="$t('expenses.chargesPanel.viewTheReceipt')"
          >
            <Paperclip class="h-3 w-3 shrink-0" />
            <span class="truncate">{{ c.document_filename || $t('expenses.invoice') }}</span>
          </button>
        </div>
      </div>
      <RowMenu v-if="vehicleStore.canEdit && (detailsUrl || setCostUrl || chargeDeletable(c))" :label="$t('common.moreActions')" :heading="formatChargeWindow(c.date, c.end_date)">
        <a v-if="setCostUrl" :href="setCostUrl" target="_blank" rel="noopener noreferrer" role="menuitem" class="menu-item">
          <ExternalLink class="w-3.5 h-3.5 text-slate-400" />{{ $t('expenses.teslamateLinks.setCost') }}
        </a>
        <a v-if="detailsUrl" :href="detailsUrl" target="_blank" rel="noopener noreferrer" role="menuitem" class="menu-item">
          <ExternalLink class="w-3.5 h-3.5 text-slate-400" />{{ $t('expenses.teslamateLinks.details') }}
        </a>
        <button v-if="chargeDeletable(c)" type="button" role="menuitem" class="menu-item hover:!text-rose-300" @click="emit('delete', c)">
          <Trash2 class="w-3.5 h-3.5 text-slate-400" />{{ $t('common.delete') }}
        </button>
      </RowMenu>
      <!-- Keeps the amounts of the list on one vertical line when a row has no menu -->
      <span v-else-if="vehicleStore.canEdit" class="-mr-1.5 w-7 shrink-0" aria-hidden="true" />
    </div>
    <QualifyActions
      v-if="toComplete"
      class="relative z-10 sm:hidden"
      stretch
      :secondary-icon="Check"
      :primary-label="$t('expenses.costReview.enterCostLong')"
      :primary-title="$t('expenses.costReview.enterCostHint')"
      :secondary-label="c.cost !== null ? $t('expenses.costReview.free') : undefined"
      :secondary-title="$t('expenses.costReview.freeHint')"
      @primary="emit('open', c)"
      @secondary="emit('review-cost', c, true)"
    />

    <!-- Tablet and desktop -->
    <div class="min-w-0 flex-1 max-sm:hidden">
      <div class="flex items-center gap-x-2 gap-y-1 flex-wrap mb-1">
        <span class="text-xs font-semibold text-slate-400 shrink-0">{{ formatChargeWindow(c.date, c.end_date) }}</span>
        <span class="text-xs px-2.5 py-0.5 rounded-full font-bold bg-slate-800 text-slate-200 border border-slate-700/60 shrink-0">
          {{ $t('expenses.chargesPanel.kwh2', { kwh_added: formatNumber(c.kwh_added, 2) }) }}
        </span>
        <span class="text-xs px-2 py-0.5 rounded-full bg-slate-800 text-slate-300 border border-slate-700 shrink-0">{{ $t(chargeOriginKey(c)) }}</span>
        <span v-if="chargeOrigin(c) === 'TESLAMATE' && c.cost_source === 'MANUAL'" class="text-xs px-2 py-0.5 rounded-full bg-slate-800 text-slate-300 border border-slate-700 shrink-0">{{ $t('expenses.chargesPanel.correctedCost') }}</span>
        <button
          v-if="c.document_id"
          type="button"
          @click="emit('view-document', c.document_id, c.document_filename, false)"
          class="relative z-10 text-xs px-2 py-0.5 rounded-full bg-sky-500/10 hover:bg-sky-500/20 text-sky-400 border border-sky-500/20 flex items-center gap-1 transition-colors max-w-[200px] truncate"
          :title="$t('expenses.chargesPanel.viewTheReceipt')"
        >
          <Paperclip class="w-3 h-3 shrink-0" />
          <span class="truncate">{{ c.document_filename || $t('expenses.invoice') }}</span>
        </button>
      </div>
      <div class="text-sm text-slate-300 flex items-center gap-1.5 min-w-0">
        <MapPin class="w-3.5 h-3.5 text-rose-400 shrink-0" />
        <span class="truncate font-medium" :title="c.address">{{ c.address || $t('expenses.chargesPanel.unknownPlace') }}</span>
      </div>
    </div>

    <div class="relative z-10 flex items-center gap-1 sm:gap-2.5 flex-wrap lg:justify-end shrink-0 max-sm:hidden">
      <QualifyActions
        v-if="toComplete"
        :primary-label="$t('expenses.costReview.enterCost')"
        :primary-title="$t('expenses.costReview.enterCostHint')"
        :secondary-label="c.cost !== null ? $t('expenses.costReview.free') : undefined"
        :secondary-title="$t('expenses.costReview.freeHint')"
        :secondary-icon="Check"
        @primary="emit('open', c)"
        @secondary="emit('review-cost', c, true)"
      />

      <div
        v-if="c.cost !== null && !review"
        class="px-2.5 py-1 sm:px-3 sm:py-1.5 bg-slate-800/80 border border-slate-700/70 rounded-xl flex items-center gap-2 text-left shadow-sm"
      >
        <Coins class="w-3.5 h-3.5 text-slate-400" />
        <div class="text-xs font-extrabold text-white flex items-center gap-1.5">
          <span>{{ formatAmount(c.cost, currency) }}</span>
          <span v-if="c.kwh_added > 0" class="text-xs font-normal text-slate-400 tabular-nums">
            {{ $t('expenses.chargesPanel.kwh', { cost: formatAmount(c.cost / c.kwh_added, currency, 3) }) }}
          </span>
        </div>
      </div>
      <div
        v-else
        class="px-2.5 py-1 sm:px-3 sm:py-1.5 bg-warning-500/10 border border-warning-500/30 rounded-xl flex items-center gap-2 text-xs font-bold text-warning-400"
      >
        <AlertTriangle class="w-3.5 h-3.5" /> {{ $t(reviewLabel) }}
      </div>

      <a
        v-if="setCostUrl"
        :href="setCostUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="row-action"
        :title="$t('expenses.teslamateLinks.setCost') + ' — ' + $t('expenses.teslamateLinks.setCostHint')" :aria-label="$t('expenses.teslamateLinks.setCost')"
      >
        <ExternalLink class="w-4 h-4" />
      </a>
      <RowMenu
        v-if="vehicleStore.canEdit && (detailsUrl || chargeDeletable(c))"
        :label="$t('common.moreActions')"
        :heading="formatChargeWindow(c.date, c.end_date)"
      >
        <a v-if="detailsUrl" :href="detailsUrl" target="_blank" rel="noopener noreferrer" role="menuitem" class="menu-item">
          <ExternalLink class="w-3.5 h-3.5 text-slate-400" />{{ $t('expenses.teslamateLinks.details') }}
        </a>
        <button v-if="chargeDeletable(c)" type="button" role="menuitem" class="menu-item hover:!text-rose-300" @click="emit('delete', c)">
          <Trash2 class="w-3.5 h-3.5 text-slate-400" />{{ $t('common.delete') }}
        </button>
      </RowMenu>
    </div>
  </div>
</template>
