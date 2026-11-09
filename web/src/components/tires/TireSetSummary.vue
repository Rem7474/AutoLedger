<script setup lang="ts">
import { computed } from 'vue'
import KpiCard from '@/components/KpiCard.vue'
import { MOUNTED_POSITIONS, forecastMonthLabel, isForecastDue, positionLabel, summarizeFittedTires, wearAndRemaining } from '@/utils/tires'
import { formatPercent } from '@/utils/numbers'
import { t } from '@/i18n'

// The three figures the tires page opens on, from the tires fitted on the vehicle
const props = defineProps<{ mounted: Record<string, any | null> }>()
const summary = computed(() => summarizeFittedTires(props.mounted))
// The verdict each figure already carries: every wheel fitted is good, a missing one is worth a look; the wear of the
// most worn tire is its own tone; a replacement that falls due is a warning
const fittedTone = computed(() => (summary.value.fitted === 0 ? 'default' : summary.value.fitted === MOUNTED_POSITIONS.length ? 'success' : 'warning'))
const wearToneOf = { ok: 'default', warning: 'warning', danger: 'danger' } as const
const wornTone = computed(() => (summary.value.mostWorn ? wearToneOf[summary.value.mostWorn.tone] : 'default'))
// The figure says it once in words; the gauge is what is left and carries the verdict by its colour.
const wornText = computed(() => {
  const worn = summary.value.mostWorn
  if (!worn) return null
  const { worn: used, left } = wearAndRemaining(worn.pct)
  return t('tires.tireSetSummary.wornAndLeft', { worn: formatPercent(used, 0), left: formatPercent(left, 0) })
})
const wornLeft = computed(() => (summary.value.mostWorn ? wearAndRemaining(summary.value.mostWorn.pct).left : 0))
const gaugeColour = { default: 'bg-success-500', warning: 'bg-warning-500', danger: 'bg-danger-500' } as const
const replacementTone = computed(() => (summary.value.nextReplacement && isForecastDue(summary.value.nextReplacement) ? 'warning' : 'default'))
const replacement = computed(() => {
  const date = summary.value.nextReplacement
  if (!date) return null
  return isForecastDue(date) ? t('tires.forecast.dueNow') : forecastMonthLabel(date)
})
</script>

<template>
  <div class="grid grid-cols-2 gap-3 sm:grid-cols-3">
    <KpiCard
      :label="$t('tires.tireSetSummary.fitted')"
      :value="String(summary.fitted)"
      :tone="fittedTone"
      :tinted="fittedTone !== 'default'"
      :unit="` / ${MOUNTED_POSITIONS.length}`"
    />
    <KpiCard
      class="order-last col-span-2 sm:order-none sm:col-span-1"
      :label="$t('tires.tireSetSummary.mostWorn')"
      :value="wornText"
      tone="default"
    >
      <div v-if="summary.mostWorn" class="mt-2 h-1.5 overflow-hidden rounded-full bg-slate-700" role="img" :aria-label="wornText ?? undefined">
        <div class="h-full rounded-full transition-all" :class="gaugeColour[wornTone as keyof typeof gaugeColour]" :style="{ width: `${wornLeft}%` }"></div>
      </div>
      <p v-if="summary.mostWorn" class="mt-1 text-xs text-slate-300">{{ positionLabel(summary.mostWorn.position) }} · {{ summary.mostWorn.name }}</p>
    </KpiCard>
    <KpiCard :label="$t('tires.tireSetSummary.nextReplacement')" :value="replacement" :tone="replacementTone" :tinted="replacementTone !== 'default'" />
  </div>
</template>
