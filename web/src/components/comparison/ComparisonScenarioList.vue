<script setup lang="ts">
import { computed } from 'vue'
import CardOpenButton from '@/components/CardOpenButton.vue'
import ListItem from '@/components/ListItem.vue'
import QuietLine from '@/components/QuietLine.vue'
import RowLeading from '@/components/RowLeading.vue'
import SwipeToDelete from '@/components/SwipeToDelete.vue'
import BulkSelectionBar from '@/components/BulkSelectionBar.vue'
import { useSelectableRow } from '@/composables/useSelectableRow'
import { GitCompare, Scale } from '@/icons'
import { distanceUnit, formatDistanceValue } from '@/units'
import { t } from '@/i18n'
import { MAX_COMPARE } from '@/utils/comparisonForm'
import { editDeleteActions, type SelectionAction } from '@/utils/selectionActions'

// The saved comparisons as list rows. A row opens the result of its comparison; its edit and delete are in that detail.
// Ticking 2 or 3 rows (a long press on a phone) opens the selection bar: compare them side by side, edit one, delete.
// A swipe to the left deletes a row on a phone.
const props = defineProps<{ scenarios: any[]; selectedIds: string[] }>()
const emit = defineEmits<{
  open: [scenario: any]
  toggle: [id: string]
  clear: []
  compare: []
  edit: [scenario: any]
  'delete-many': [scenarios: any[]]
}>()

const selected = computed(() => props.scenarios.filter((s) => props.selectedIds.includes(s.id)))
const actions = computed<SelectionAction[]>(() => [
  {
    key: 'compare',
    label: t('comparison.comparisonView.compare', { length: props.selectedIds.length }),
    icon: GitCompare,
    primary: true,
    run: () => emit('compare'),
    disabled: props.selectedIds.length < 2,
    reason: t('comparison.comparisonView.compareNeedsTwo'),
  },
  ...editDeleteActions(selected.value, { edit: (s) => emit('edit', s), remove: (list) => emit('delete-many', list) }),
])

const { showBox, press, onPointerdown, onRowClick } = useSelectableRow<any>({
  enabled: computed(() => true),
  selecting: computed(() => props.selectedIds.length > 0),
  toggle: (s) => emit('toggle', s.id),
  open: (s) => emit('open', s),
})
const isSelected = (id: string) => props.selectedIds.includes(id)
</script>

<template>
  <div class="space-y-3">
    <BulkSelectionBar :count="selectedIds.length" noun="scenario" :actions="actions" @clear="emit('clear')" />
    <p v-if="scenarios.length >= 2" class="text-xs text-slate-400 max-sm:hidden">{{ $t('comparison.comparisonView.tick2Or3Comparisons') }}</p>
    <ul class="grid gap-3 md:grid-cols-2">
      <li v-for="sc in scenarios" :key="sc.id">
        <SwipeToDelete :disabled="selectedIds.length > 0" @delete="emit('delete-many', [sc])">
          <ListItem wrap-title class="max-sm:select-none max-sm:[-webkit-touch-callout:none]" :class="isSelected(sc.id) ? '!border-rose-500/50 bg-rose-500/[0.02]' : ''">
            <template #open>
              <CardOpenButton
                :label="$t('common.openItem', { item: sc.name })"
                @click="onRowClick(sc)"
                @pointerdown="onPointerdown($event, sc)"
                @pointermove="press.onPointermove"
                @pointerup="press.onPointerup"
                @pointercancel="press.onPointercancel"
                @contextmenu="press.onContextmenu"
              />
            </template>
            <template #lead>
              <RowLeading :icon="Scale" :selected="isSelected(sc.id)" />
              <label
                v-if="showBox"
                :for="`cmp-select-${sc.id}`"
                class="relative z-10 tap flex cursor-pointer items-center p-1 max-sm:sr-only max-sm:pointer-events-none"
                :title="$t('comparison.comparisonView.selectFor', { name: sc.name })"
              >
                <span class="sr-only">{{ $t('comparison.comparisonView.selectFor', { name: sc.name }) }}</span>
                <input
                  :id="`cmp-select-${sc.id}`"
                  type="checkbox"
                  class="select-box"
                  :checked="isSelected(sc.id)"
                  :disabled="!isSelected(sc.id) && selectedIds.length >= MAX_COMPARE"
                  @change="emit('toggle', sc.id)"
                />
              </label>
            </template>
            <template #title><span :title="sc.name">{{ sc.name }}</span></template>
            <template #subtitle>
              <QuietLine>
                <span><span>{{ sc.mode === 'RETROSPECTIVE' ? $t('comparison.comparisonView.trackedVehicle') : $t('comparison.comparisonView.projection') }}</span></span>
                <span>
                  <span>{{ $t('comparison.comparisonView.kmPerYear', { unit: distanceUnit(), km: formatDistanceValue(Number(sc.annual_km || 0)) }) }}</span>
                  <span>{{ $t('comparison.comparisonView.yearsCount', { years: sc.years }) }}</span>
                </span>
              </QuietLine>
            </template>
          </ListItem>
        </SwipeToDelete>
      </li>
    </ul>
  </div>
</template>
