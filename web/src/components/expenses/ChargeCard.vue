<script setup lang="ts">
import { computed } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import { MapPin, Coins, Paperclip, Trash2, ExternalLink, AlertTriangle, Zap, Check } from 'lucide-vue-next'
import CardOpenButton from '@/components/CardOpenButton.vue'
import RowLeading from '@/components/RowLeading.vue'
import RowMenu from '@/components/RowMenu.vue'
import { useIsMobile } from '@/composables/useIsMobile'
import { useLongPress } from '@/composables/useLongPress'
import QualifyActions from '@/components/drives/QualifyActions.vue'
import { formatChargeWindow } from '@/utils/dates'
import { formatAmount } from '@/currency'
import { formatNumber } from '@/utils/numbers'
import { chargeDeletable, chargeOrigin, chargeOriginKey } from '@/utils/chargeProvenance'
import { chargeCostToReview, teslamateChargeUrl, teslamateCostEditUrl } from '@/utils/teslamateCharges'

// One charge of the list, laid out like a drive (DriveCard): a click opens it, the cost to complete is a two tap
// queue (enter it, or confirm a 0 as free), and the rest sits in the "..." menu (desktop; on a phone the selection bar
// carries the actions). On a phone the card follows the drive card: a glyph that becomes the check once the charge is
// selected, the place and the amount on the first line, the details under them, and the two taps across the full width.
// A long press selects the charge and starts the selection (`selecting`), where a tap toggles it instead of opening it.
const props = defineProps<{ c: any; selected: boolean; selecting?: boolean }>()
const emit = defineEmits<{
  open: [charge: any]
  toggle: [charge: any]
  delete: [charge: any]
  'review-cost': [charge: any, reviewed: boolean]
  'view-document': [docId: string | null | undefined, filename?: string | null, download?: boolean]
}>()
const vehicleStore = useVehicleStore()
const isMobile = useIsMobile()
const showBox = computed(() => vehicleStore.canEdit && (!isMobile.value || !!props.selecting))
const press = useLongPress(() => {
  if (vehicleStore.canEdit && isMobile.value) emit('toggle', props.c)
})
function onRowClick() {
  if (isMobile.value && props.selecting) emit('toggle', props.c)
  else emit('open', props.c)
}
const currency = computed(() => props.c.currency || vehicleStore.currency)
const review = computed(() => chargeCostToReview(props.c))
const toComplete = computed(() => vehicleStore.canEdit && review.value)
// What stands in the place of the amount while the cost is to be completed: nothing, or a 0 that may be a free charge
const reviewLabel = computed(() => (props.c.cost === null || props.c.cost === undefined ? 'expenses.chargesPanel.missingCost' : 'expenses.costReview.toConfirm'))
const detailsUrl = computed(() => teslamateChargeUrl(vehicleStore.activeVehicle, props.c))
// A synchronised charge whose cost TeslaMate still reads again: the cost is set there, so the first tap is a link to it
// (the form of the charge keeps its cost read-only). Without such a link the cost is entered in AutoLedger.
const costInTeslaMate = computed(() => (toComplete.value ? teslamateCostEditUrl(vehicleStore.activeVehicle, props.c) : null))
</script>

<template>
  <div
    class="relative bg-slate-900 border hover:border-slate-700/90 p-3 sm:p-4 rounded-2xl transition-all flex flex-col lg:flex-row lg:items-center justify-between gap-2 sm:gap-4 group max-sm:select-none max-sm:[-webkit-touch-callout:none]"
    :class="selected ? 'border-rose-500/40 bg-slate-800/40 shadow-lg shadow-rose-950/20' : review ? 'border-warning-500/40' : 'border-slate-800'"
  >
    <CardOpenButton
      v-if="vehicleStore.canEdit"
      :label="$t('expenses.costReview.openCharge', { date: formatChargeWindow(c.date, c.end_date) })"
      @click="onRowClick"
      @pointerdown="press.onPointerdown"
      @pointermove="press.onPointermove"
      @pointerup="press.onPointerup"
      @pointercancel="press.onPointercancel"
      @contextmenu="press.onContextmenu"
    />

    <!-- Phone -->
    <div class="flex items-start gap-3 sm:hidden">
      <RowLeading :icon="Zap" :selected="selected" :warn="review" />
      <!-- The check of the row, for screen readers on a phone (the glyph above is the visual one) -->
      <label v-if="showBox" :for="'charge-select-m-' + c.id" class="sr-only pointer-events-none">
        {{ $t('shell.bulkSelectionBar.selectRow') }}
        <input :id="'charge-select-m-' + c.id" type="checkbox" :checked="selected" @change="emit('toggle', c)" />
      </label>
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
    </div>
    <QualifyActions
      v-if="toComplete"
      class="relative z-10 sm:hidden"
      stretch
      :secondary-icon="Check"
      :primary-label="costInTeslaMate ? $t('expenses.costReview.setInTeslaMate') : $t('expenses.costReview.enterCostLong')"
      :primary-title="costInTeslaMate ? $t('expenses.teslamateLinks.setCostHint') : $t('expenses.costReview.enterCostHint')"
      :primary-href="costInTeslaMate ?? undefined"
      :secondary-label="c.cost !== null ? $t('expenses.costReview.free') : undefined"
      :secondary-title="$t('expenses.costReview.freeHint')"
      @primary="emit('open', c)"
      @secondary="emit('review-cost', c, true)"
    />

    <!-- Tablet and desktop -->
    <div class="flex min-w-0 flex-1 items-start gap-3 max-sm:hidden">
      <label v-if="showBox" :for="'charge-select-' + c.id" class="relative z-10 tap -ml-1 flex shrink-0 cursor-pointer items-center p-1" :title="$t('shell.bulkSelectionBar.selectRow')">
        <span class="sr-only">{{ $t('shell.bulkSelectionBar.selectRow') }}</span>
        <input :id="'charge-select-' + c.id" type="checkbox" :checked="selected" @change="emit('toggle', c)" class="select-box" />
      </label>
      <div class="min-w-0 flex-1">
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
    </div>

    <div class="relative z-10 flex items-center gap-1 sm:gap-2.5 flex-wrap lg:justify-end shrink-0 max-sm:hidden">
      <QualifyActions
        v-if="toComplete"
        :primary-label="costInTeslaMate ? $t('expenses.costReview.setInTeslaMate') : $t('expenses.costReview.enterCost')"
        :primary-title="costInTeslaMate ? $t('expenses.teslamateLinks.setCostHint') : $t('expenses.costReview.enterCostHint')"
        :primary-href="costInTeslaMate ?? undefined"
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

      <RowMenu
        v-if="vehicleStore.canEdit && (detailsUrl || chargeDeletable(c))"
        :label="$t('common.moreActions')"
        :heading="formatChargeWindow(c.date, c.end_date)"
      >
        <a v-if="detailsUrl" :href="detailsUrl" target="_blank" rel="noopener noreferrer" role="menuitem" class="menu-item">
          <ExternalLink class="w-3.5 h-3.5 text-slate-400" />{{ $t('expenses.teslamateLinks.details') }}
        </a>
        <button v-if="chargeDeletable(c)" type="button" role="menuitem" class="menu-item menu-item-danger" @click="emit('delete', c)">
          <Trash2 class="w-3.5 h-3.5 text-slate-400" />{{ $t('common.delete') }}
        </button>
      </RowMenu>
    </div>
  </div>
</template>
