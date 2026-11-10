<script setup lang="ts">
import { computed } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import { Paperclip, Trash2, ExternalLink, Zap, Check } from '@/icons'
import CardOpenButton from '@/components/CardOpenButton.vue'
import RowLeading from '@/components/RowLeading.vue'
import ListItem from '@/components/ListItem.vue'
import RowMenu from '@/components/RowMenu.vue'
import { useIsMobile } from '@/composables/useIsMobile'
import { useLongPress } from '@/composables/useLongPress'
import QualifyActions from '@/components/drives/QualifyActions.vue'
import { t } from '@/i18n'
import { formatChargeWindow } from '@/utils/dates'
import { formatAmount } from '@/currency'
import { formatNumber } from '@/utils/numbers'
import { chargeDeletable, chargeOrigin, chargeOriginKey } from '@/utils/chargeProvenance'
import { chargeCostToReview, teslamateChargeUrl, teslamateCostEditUrl } from '@/utils/teslamateCharges'

// One charge of the list, a ListItem: the glyph (a checkbox from sm up, once the list can be edited), the place with
// the dates, the energy and the origin under it, and the amount with its price per kWh. A click opens the charge; the
// cost to complete is a two tap queue (enter it, or confirm a 0 as free) and the rest sits in the "..." menu (from sm up;
// on a phone the selection bar carries the actions). On a phone the glyph becomes the check once the charge is
// selected and the two taps sit across the full width under the row. A long press selects the charge and starts the
// selection (`selecting`), where a tap toggles it instead of opening it.
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
const subtitle = computed(() =>
  [
    t('expenses.chargesPanel.kwh2', { kwh_added: formatNumber(props.c.kwh_added, 2) }),
    formatChargeWindow(props.c.date, props.c.end_date),
    t(chargeOriginKey(props.c)),
    chargeOrigin(props.c) === 'TESLAMATE' && props.c.cost_source === 'MANUAL' ? t('expenses.chargesPanel.correctedCost') : null,
  ]
    .filter(Boolean)
    .join(' · '),
)
const detailsUrl = computed(() => teslamateChargeUrl(vehicleStore.activeVehicle, props.c))
// A synchronised charge whose cost TeslaMate still reads again: the cost is set there, so the first tap is a link to it
// (the form of the charge keeps its cost read-only). Without such a link the cost is entered in AutoLedger.
const costInTeslaMate = computed(() => (toComplete.value ? teslamateCostEditUrl(vehicleStore.activeVehicle, props.c) : null))
</script>

<template>
  <div class="space-y-1.5">
    <ListItem
      wrap-subtitle
      data-testid="charge-card"
      class="max-sm:select-none max-sm:[-webkit-touch-callout:none]"
      :class="selected ? 'border-rose-500/40 bg-slate-800/40' : review ? 'border-warning-500/40' : ''"
    >
      <template #open>
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
      </template>
      <template #lead>
        <RowLeading :icon="Zap" :selected="selected" :warn="review" />
        <!-- The check of the row, for screen readers on a phone (the glyph above is the visual one) -->
        <label v-if="showBox" :for="'charge-select-m-' + c.id" class="sr-only pointer-events-none sm:hidden">
          {{ $t('shell.bulkSelectionBar.selectRow') }}
          <input :id="'charge-select-m-' + c.id" type="checkbox" :checked="selected" @change="emit('toggle', c)" />
        </label>
        <label v-if="showBox" :for="'charge-select-' + c.id" class="relative z-10 tap flex cursor-pointer items-center p-1 max-sm:hidden" :title="$t('shell.bulkSelectionBar.selectRow')">
          <span class="sr-only">{{ $t('shell.bulkSelectionBar.selectRow') }}</span>
          <input :id="'charge-select-' + c.id" type="checkbox" :checked="selected" @change="emit('toggle', c)" class="select-box" />
        </label>
        <span v-else class="grid size-10 place-items-center rounded-full bg-slate-800 text-slate-300 max-sm:hidden" aria-hidden="true"><Zap class="size-5" /></span>
      </template>
      <template #title>
        <span :title="c.address">{{ c.address || $t('expenses.chargesPanel.unknownPlace') }}</span>
      </template>
      <template #subtitle>
        <span :title="subtitle">{{ subtitle }}</span>
      </template>
      <template v-if="c.document_id || toComplete || (vehicleStore.canEdit && (detailsUrl || chargeDeletable(c)))" #meta>
        <div class="relative z-10 flex items-center gap-2">
          <button
            v-if="c.document_id"
            type="button"
            class="tap flex items-center justify-center rounded-full p-1.5 text-sky-400 hover:bg-sky-500/10"
            :title="c.document_filename || $t('expenses.chargesPanel.viewTheReceipt')"
            :aria-label="$t('expenses.chargesPanel.viewTheReceipt')"
            @click="emit('view-document', c.document_id, c.document_filename, false)"
          >
            <Paperclip class="size-4" />
          </button>
          <QualifyActions
            v-if="toComplete"
            class="max-sm:hidden"
            :primary-label="costInTeslaMate ? $t('expenses.costReview.setInTeslaMate') : $t('expenses.costReview.enterCost')"
            :primary-title="costInTeslaMate ? $t('expenses.teslamateLinks.setCostHint') : $t('expenses.costReview.enterCostHint')"
            :primary-href="costInTeslaMate ?? undefined"
            :secondary-label="c.cost !== null ? $t('expenses.costReview.free') : undefined"
            :secondary-title="$t('expenses.costReview.freeHint')"
            :secondary-icon="Check"
            @primary="emit('open', c)"
            @secondary="emit('review-cost', c, true)"
          />
          <RowMenu
            v-if="vehicleStore.canEdit && (detailsUrl || chargeDeletable(c))"
            class="max-sm:hidden"
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
      </template>
      <template #value>
        <span v-if="review" class="text-warning-400">{{ $t(reviewLabel) }}</span>
        <span v-else-if="c.cost !== null" class="text-white">{{ formatAmount(c.cost, currency) }}</span>
      </template>
      <template v-if="!review && c.cost !== null && c.kwh_added > 0" #subvalue>
        {{ $t('expenses.chargesPanel.kwh', { cost: formatAmount(c.cost / c.kwh_added, currency, 3) }) }}
      </template>
    </ListItem>
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
  </div>
</template>
