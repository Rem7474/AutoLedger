<script setup lang="ts">
import CardOpenButton from '@/components/CardOpenButton.vue'
import TireWearBar from '@/components/tires/TireWearBar.vue'
import { Disc } from 'lucide-vue-next'
import { computed } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import { useSelectableRow } from '@/composables/useSelectableRow'
import { forecastMonthLabel, getConditionBadge, isForecastDue, WEAR_TONE_TEXT, wearTone } from '@/utils/tires'
import { formatPercent } from '@/utils/numbers'
import { t } from '@/i18n'

// One wheel of the phone overview: its position, how worn the tire is and when it is due, in a tile that opens the tire.
// The model is named only when the wheels do not all carry the same one.
const props = defineProps<{ label: string; stat: any | null; showModel?: boolean; selected: boolean; selecting?: boolean; canMount?: boolean }>()
const emit = defineEmits<{ open: [stat: any]; toggle: [tireId: string]; mount: [] }>()
const vehicleStore = useVehicleStore()
const { showBox, press, onPointerdown, onRowClick } = useSelectableRow({
  enabled: computed(() => vehicleStore.canEdit),
  selecting: computed(() => !!props.selecting),
  toggle: () => emit('toggle', props.stat?.tire.id),
  open: () => emit('open', props.stat),
})
const wear = computed(() => Number(props.stat?.life_progress_pct) || 0)
const tone = computed(() => WEAR_TONE_TEXT[wearTone(wear.value, props.stat?.condition)])
const due = computed(() => {
  const date = props.stat?.replacement_forecast?.replacement_date as string | undefined
  if (!date) return null
  return isForecastDue(date) ? t('tires.forecast.dueNow') : forecastMonthLabel(date)
})
</script>

<template>
  <div
    v-if="stat"
    class="relative space-y-2 rounded-2xl border border-slate-800 bg-slate-900 p-3.5 select-none [-webkit-touch-callout:none]"
    :class="{ 'ring-2 ring-rose-500/50 border-rose-500/60': selected }"
  >
    <CardOpenButton
      :label="$t('tires.openTire', { name: `${stat.tire.brand} ${stat.tire.model}` })"
      @click="onRowClick()"
      @pointerdown="onPointerdown($event)"
      @pointermove="press.onPointermove"
      @pointerup="press.onPointerup"
      @pointercancel="press.onPointercancel"
      @contextmenu="press.onContextmenu"
    />
    <label :for="'tile-select-' + stat.tire.id" class="flex items-center gap-2 text-xs font-semibold text-slate-300" :class="showBox ? 'relative z-10 cursor-pointer' : ''">
      <input v-if="showBox" :id="'tile-select-' + stat.tire.id" type="checkbox" :checked="selected" class="select-box" @change="emit('toggle', stat.tire.id)" />
      <span class="truncate">{{ label }}</span>
    </label>
    <p v-if="showModel" class="truncate text-xs text-slate-400">{{ stat.tire.brand }} {{ stat.tire.model }}</p>
    <p class="text-2xl font-extrabold tabular-nums" :class="tone">
      {{ formatPercent(wear) }}
      <span class="sr-only">{{ getConditionBadge(stat.condition).label }}</span>
    </p>
    <TireWearBar :pct="stat.life_progress_pct" :condition="stat.condition" />
    <p class="truncate text-xs text-slate-400">{{ due ? $t('tires.overview.due', { date: due }) : getConditionBadge(stat.condition).label }}</p>
  </div>
  <div v-else class="flex min-h-32 flex-col items-center justify-center gap-1.5 rounded-2xl border border-dashed border-slate-800 bg-slate-900/40 p-3 text-center text-xs text-slate-400">
    <Disc class="h-6 w-6 opacity-30" aria-hidden="true" />
    <span>{{ label }}</span>
    <button v-if="canMount" type="button" class="tap-text font-semibold text-sky-400 underline" @click="emit('mount')">{{ $t('tires.tireWheelCard.fitATire') }}</button>
  </div>
</template>
