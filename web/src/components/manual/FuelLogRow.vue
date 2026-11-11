<script setup lang="ts">
import { computed } from 'vue'
import { Fuel } from '@/icons'
import CardOpenButton from '@/components/CardOpenButton.vue'
import ListItem from '@/components/ListItem.vue'
import RowLeading from '@/components/RowLeading.vue'
import RowSelectBox from '@/components/RowSelectBox.vue'
import StatusLabel from '@/components/StatusLabel.vue'
import { useIsMobile } from '@/composables/useIsMobile'
import { useLongPress } from '@/composables/useLongPress'
import { intlLocale, t } from '@/i18n'
import { formatAmount } from '@/currency'
import { formatCostPerDistance, NO_VALUE } from '@/utils/costPerDistance'
import { formatDistance, formatFuelConsumptionValue, formatVolumeValue, fuelConsumptionUnit, perVolume, volumeUnitLabel } from '@/units'

// One fill-up of the list: the fuel glyph, its date (and mileage), what was bought, what it cost. A click anywhere on the
// row opens the fill-up sheet, where it is edited or deleted. On a phone there are no checkboxes at rest: a long press
// selects the row and starts the selection (`selecting`), where the boxes show and a tap toggles the row instead of opening it.
const props = defineProps<{ log: any; currency: string; selected: boolean; selecting?: boolean; canEdit: boolean }>()
const emit = defineEmits<{ open: [log: any]; toggle: [log: any] }>()
const isMobile = useIsMobile()
const showBox = computed(() => props.canEdit && (!isMobile.value || !!props.selecting))
const press = useLongPress(() => {
  if (props.canEdit && isMobile.value) emit('toggle', props.log)
})
function onRowClick() {
  if (isMobile.value && props.selecting) emit('toggle', props.log)
  else emit('open', props.log)
}

const date = computed(() => new Date(props.log.date).toLocaleDateString(intlLocale(), { day: '2-digit', month: 'short', year: 'numeric' }))
const title = computed(() => {
  const l = props.log
  if (l.odometer != null) return `${date.value} · ${formatDistance(l.odometer)}`
  if (l.odometer_estimated != null) return `${date.value} · ≈ ${formatDistance(l.odometer_estimated)}`
  return date.value
})

// The quiet line as whole parts (a break falls between two parts, never inside one); the consumption leaves a phone.
// A state (partial tank, estimated mileage) is plain coloured text, not a pill.
const parts = computed(() => {
  const l = props.log
  const out: { text: string; wide?: boolean; state?: boolean }[] = []
  if (l.liters) out.push({ text: `${formatVolumeValue(l.liters, 2)} ${volumeUnitLabel()}` })
  if (l.price_per_liter) out.push({ text: `${formatAmount(perVolume(l.price_per_liter), props.currency, 3)}/${volumeUnitLabel()}` })
  if (!l.is_full_tank) out.push({ text: t('manual.fuelLogsPanel.partial'), state: true })
  if (l.odometer == null && l.odometer_estimated != null) out.push({ text: t('manual.fuelLogsPanel.estimated'), state: true })
  if (l.consumption_l_100km) {
    out.push({ wide: true, text: `${l.segment_estimated ? '≈ ' : ''}${formatFuelConsumptionValue(l.consumption_l_100km, 2)} ${fuelConsumptionUnit()}` })
  }
  return out
})
const costPerDistance = computed(() => formatCostPerDistance(props.log.cost_per_km, props.currency, 3, true))
</script>

<template>
  <div data-testid="fuel-row" class="max-sm:select-none max-sm:[-webkit-touch-callout:none]">
    <ListItem :class="{ 'border-rose-500/40 bg-slate-800/40 shadow-lg shadow-rose-950/20': selected }">
      <template v-if="canEdit" #open>
        <CardOpenButton
          :label="$t('manual.fuelLogsPanel.editFillUp', { date })"
          @click="onRowClick"
          @pointerdown="press.onPointerdown"
          @pointermove="press.onPointermove"
          @pointerup="press.onPointerup"
          @pointercancel="press.onPointercancel"
          @contextmenu="press.onContextmenu"
        />
      </template>
      <template #lead>
        <RowLeading :icon="Fuel" :selected="selected" />
        <RowSelectBox v-if="showBox" :id="log.id" :checked="selected" :label="$t('shell.bulkSelectionBar.selectRow')" @toggle="emit('toggle', log)" />
      </template>
      <template #title>{{ title }}</template>
      <template v-if="parts.length" #subtitle>
        <span class="whitespace-normal">
          <template v-for="(part, i) in parts" :key="i">
            <span v-if="i > 0" :class="part.wide ? 'max-sm:hidden' : ''"> · </span>
            <span :class="['whitespace-nowrap', part.wide ? 'max-sm:hidden' : '']">
              <StatusLabel v-if="part.state" tone="neutral">{{ part.text }}</StatusLabel>
              <template v-else>{{ part.text }}</template>
            </span>
          </template>
        </span>
      </template>
      <template #value>{{ formatAmount(log.amount || 0, currency) }}</template>
      <template v-if="costPerDistance !== NO_VALUE" #subvalue>{{ costPerDistance }}</template>
    </ListItem>
  </div>
</template>
