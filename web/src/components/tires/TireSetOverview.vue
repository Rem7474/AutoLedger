<script setup lang="ts">
import { computed } from 'vue'
import TireWearBar from '@/components/tires/TireWearBar.vue'
import TireWheelTile from '@/components/tires/TireWheelTile.vue'
import { MOUNTED_POSITIONS, describeFittedSet, forecastMonthLabel, isForecastDue, positionLabel, summarizeFittedTires, WEAR_TONE_TEXT, wearAndRemaining } from '@/utils/tires'
import { formatPercent } from '@/utils/numbers'
import { t } from '@/i18n'

// The tires fitted on a phone: the set in one card (what it is, how worn the worst tire is, when the first one is due),
// then one tile per wheel to open its tire. Replaces four near-identical cards when the wheels carry the same tires.
const props = defineProps<{
  mounted: Record<string, any | null>
  wheels: { pos: string; labelKey: string }[]
  selectedIds: string[]
  canMount?: boolean
}>()
const emit = defineEmits<{ open: [stat: any]; toggle: [tireId: string]; mount: [] }>()

const summary = computed(() => summarizeFittedTires(props.mounted))
const set = computed(() => describeFittedSet(props.mounted))
const figures = computed(() => {
  const worn = summary.value.mostWorn
  if (!worn) return { left: '', worn: '' }
  const { worn: used, left } = wearAndRemaining(worn.pct)
  return { left: t('tires.overview.left', { left: formatPercent(left, 0) }), worn: t('tires.overview.wornCaption', { worn: formatPercent(used, 0) }) }
})
const nextDue = computed(() => {
  const date = summary.value.nextReplacement
  if (!date) return null
  return isForecastDue(date) ? t('tires.forecast.dueNow') : forecastMonthLabel(date)
})
</script>

<template>
  <section class="space-y-3">
    <div v-if="summary.fitted > 0" class="card p-4">
      <h2 class="truncate text-base font-bold text-white">{{ set.uniform ? set.name : $t('tires.overview.mixed') }}</h2>
      <p class="mt-0.5 text-xs text-slate-400">
        <span v-if="set.dimension" class="tabular-nums">{{ set.dimension }}</span>
        <span v-if="set.dimension && summary.fitted < MOUNTED_POSITIONS.length"> · </span>
        <span v-if="summary.fitted < MOUNTED_POSITIONS.length" class="font-medium text-warning-400">{{ $t('tires.overview.partlyFitted', { count: summary.fitted, total: MOUNTED_POSITIONS.length }) }}</span>
      </p>
      <dl class="mt-3 grid grid-cols-2 gap-3">
        <div v-if="summary.mostWorn">
          <dt class="text-xs font-semibold text-slate-400">{{ $t('tires.tireSetSummary.mostWorn') }}</dt>
          <dd>
            <p class="text-lg font-bold tabular-nums" :class="WEAR_TONE_TEXT[summary.mostWorn.tone]">{{ figures.left }}</p>
            <TireWearBar remaining class="mt-1.5" :pct="summary.mostWorn.pct" />
            <p class="mt-1 text-xs tabular-nums text-slate-400">{{ figures.worn }} · {{ positionLabel(summary.mostWorn.position) }}</p>
          </dd>
        </div>
        <div v-if="nextDue">
          <dt class="text-xs font-semibold text-slate-400">{{ $t('tires.tireSetSummary.nextReplacement') }}</dt>
          <dd class="text-lg font-bold text-white">{{ nextDue }}</dd>
        </div>
      </dl>
    </div>
    <div class="grid grid-cols-2 gap-3">
      <TireWheelTile
        v-for="w in wheels"
        :key="w.pos"
        :label="$t(w.labelKey)"
        :stat="mounted[w.pos]"
        :show-model="!set.uniform"
        :selected="!!mounted[w.pos] && selectedIds.includes(mounted[w.pos].tire.id)"
        :selecting="selectedIds.length > 0"
        :can-mount="canMount"
        @open="emit('open', $event)"
        @toggle="emit('toggle', $event)"
        @mount="emit('mount')"
      />
    </div>
  </section>
</template>
