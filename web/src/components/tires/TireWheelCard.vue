<script setup lang="ts">
import CardOpenButton from '@/components/CardOpenButton.vue'
import GlossaryTip from '@/components/GlossaryTip.vue'
import StatusLabel from '@/components/StatusLabel.vue'
import { formatPercent } from '@/utils/numbers'
import { t } from '@/i18n'
import { Disc } from '@/icons'
import TireWearBar from '@/components/tires/TireWearBar.vue'
import TireForecastLine from '@/components/tires/TireForecastLine.vue'
import TireMeasuredWear from '@/components/tires/TireMeasuredWear.vue'
import { formatAmount } from '@/currency'
import { useVehicleStore } from '@/stores/vehicle'
import { useSelectableRow } from '@/composables/useSelectableRow'
import { computed } from 'vue'
import { distanceUnit, formatDistance, formatDistanceValue, perDistance } from '@/units'
import { formatCostPerDistance } from '@/utils/costPerDistance'

// One wheel of an axle card: the mounted tire with its wear, or a placeholder when the wheel is empty. The tire is named only
// when the wheels do not all carry the same one (`showModel`); a condition is written only when it is not good.
const props = defineProps<{ pos: string; label: string; stat: any | null; showModel?: boolean; selected: boolean; selecting?: boolean; canMount?: boolean }>()
const emit = defineEmits<{ open: [stat: any]; toggle: [tireId: string]; mount: [] }>()
const vehicleStore = useVehicleStore()
const { showBox, press, onPointerdown, onRowClick } = useSelectableRow({
  enabled: computed(() => vehicleStore.canEdit),
  selecting: computed(() => !!props.selecting),
  toggle: () => emit('toggle', props.stat?.tire.id),
  open: () => emit('open', props.stat),
})
// The condition worth a word: a good tire is read from its gauge
const condition = computed(() => {
  const c = props.stat?.condition
  if (c === 'WARNING') return { tone: 'attention' as const, label: t('tires.condition.warning') }
  if (c === 'CRITICAL') return { tone: 'danger' as const, label: t('tires.condition.critical') }
  return null
})
</script>

<template>
  <div
    v-if="stat"
    class="relative space-y-3 rounded-2xl p-3 transition-colors hover:bg-slate-800/60 max-sm:select-none max-sm:[-webkit-touch-callout:none]"
    :class="{ 'ring-2 ring-rose-500/50': selected }"
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
    <div class="flex items-start justify-between gap-2">
      <div class="min-w-0">
        <label :for="'chassis-select-' + pos.toLowerCase() + '-' + stat.tire.id" :class="showBox ? 'relative z-10 cursor-pointer' : ''" class="flex items-center gap-2 text-sm font-bold text-white" :title="selected ? $t('tires.tireWheelCard.removeFromSelection') : $t('tires.tireWheelCard.selectForBulk')">
          <input
            v-if="showBox"
            :id="'chassis-select-' + pos.toLowerCase() + '-' + stat.tire.id"
            type="checkbox"
            :checked="selected"
            @change="emit('toggle', stat.tire.id)"
            class="select-box"
          />
          <span>{{ label }}</span>
        </label>
        <template v-if="showModel">
          <p class="mt-0.5 truncate text-xs text-slate-300" :title="`${stat.tire.brand} ${stat.tire.model}`">{{ stat.tire.brand }} {{ stat.tire.model }}</p>
          <p class="text-xs text-slate-400 font-mono">{{ stat.tire.dimension }}</p>
        </template>
      </div>
      <StatusLabel v-if="condition" :tone="condition.tone">{{ condition.label }}</StatusLabel>
    </div>

    <dl class="grid grid-cols-2 gap-3">
      <div>
        <dt class="text-xs text-slate-400">{{ $t('tires.tireWheelCard.totalDriven') }}</dt>
        <dd class="text-base font-bold tabular-nums text-white">{{ formatDistance(stat.total_distance_km) }}</dd>
      </div>
      <div :title="Number(stat.tire.purchase_price) > 0 ? undefined : $t('tires.costPerKmUnset')">
        <dt class="text-xs text-slate-400">{{ $t('tires.tireWheelCard.costKm', { unit: distanceUnit() }) }}</dt>
        <dd class="text-base font-bold tabular-nums text-white">{{ formatCostPerDistance(stat.cost_per_km, vehicleStore.currency, 4) }}</dd>
      </div>
    </dl>

    <!-- Lifespan progress bar -->
    <div class="space-y-1.5">
      <div class="flex items-center justify-between text-xs text-slate-400">
        <span>{{ $t('tires.tireWheelCard.estimatedLifespanWearKm', { unit: distanceUnit(), estimated_lifespan_km: formatDistanceValue(stat.estimated_lifespan_km) }) }}<GlossaryTip v-if="!(stat.logs_count > 0)" class="relative z-10" term="treadNotMeasured" /></span>
        <span class="font-bold text-slate-200">{{ formatPercent(Number(stat.life_progress_pct)) }}</span>
      </div>
      <TireWearBar :pct="stat.life_progress_pct" :condition="stat.condition" />
    </div>
    <TireMeasuredWear :stat="stat" />
    <TireForecastLine :forecast="stat.replacement_forecast" />
  </div>
  <div v-else class="flex min-h-32 flex-col items-center justify-center gap-1.5 rounded-2xl border border-dashed border-slate-800 p-4 text-center text-xs text-slate-400">
    <Disc class="w-8 h-8 opacity-30" aria-hidden="true" />
    <span>{{ $t('tires.tireWheelCard.noTireFittedAtThe', { label, pos }) }}</span>
    <button v-if="canMount" type="button" @click="emit('mount')" class="tap-text font-semibold text-slate-200 underline">
      {{ $t('tires.tireWheelCard.fitATire') }}
    </button>
  </div>
</template>
