<script setup lang="ts">
import { computed } from 'vue'
import KpiCard from '@/components/KpiCard.vue'
import { MOUNTED_POSITIONS, forecastMonthLabel, isForecastDue, positionLabel, summarizeFittedTires } from '@/utils/tires'
import { formatPercent } from '@/utils/numbers'
import { t } from '@/i18n'

// The three figures the tires page opens on, from the tires fitted on the vehicle
const props = defineProps<{ mounted: Record<string, any | null> }>()
const summary = computed(() => summarizeFittedTires(props.mounted))
// The verdict each figure already carries: every wheel fitted is good, a missing one is worth a look; the wear of the
// most worn tire is its own tone; a replacement that falls due is a warning
const fittedTone = computed(() => (summary.value.fitted === 0 ? 'default' : summary.value.fitted === MOUNTED_POSITIONS.length ? 'success' : 'warning'))
const wearToneOf = { ok: 'success', warning: 'warning', danger: 'danger' } as const
const wornTone = computed(() => (summary.value.mostWorn ? wearToneOf[summary.value.mostWorn.tone] : 'default'))
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
      :value="summary.mostWorn ? formatPercent(summary.mostWorn.pct) : null"
      :tone="wornTone"
      :tinted="wornTone !== 'default'"
      :unit="summary.mostWorn ? ` ${$t('tires.tireSetSummary.ofLifespan')}` : undefined"
    >
      <p v-if="summary.mostWorn" class="mt-1 text-xs text-slate-300">{{ positionLabel(summary.mostWorn.position) }} · {{ summary.mostWorn.name }}</p>
    </KpiCard>
    <KpiCard :label="$t('tires.tireSetSummary.nextReplacement')" :value="replacement" :tone="replacementTone" :tinted="replacementTone !== 'default'" />
  </div>
</template>
