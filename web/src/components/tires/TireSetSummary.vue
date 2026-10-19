<script setup lang="ts">
import { computed } from 'vue'
import KpiCard from '@/components/KpiCard.vue'
import { MOUNTED_POSITIONS, forecastMonthLabel, isForecastDue, positionLabel, summarizeFittedTires } from '@/utils/tires'
import { formatPercent } from '@/utils/numbers'
import { t } from '@/i18n'

// The three figures the tires page opens on, from the tires fitted on the vehicle
const props = defineProps<{ mounted: Record<string, any | null> }>()
const summary = computed(() => summarizeFittedTires(props.mounted))
const replacement = computed(() => {
  const date = summary.value.nextReplacement
  if (!date) return null
  return isForecastDue(date) ? t('tires.forecast.dueNow') : forecastMonthLabel(date)
})
</script>

<template>
  <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
    <KpiCard
      :label="$t('tires.tireSetSummary.fitted')"
      :value="String(summary.fitted)"
      :unit="` / ${MOUNTED_POSITIONS.length}`"
    />
    <KpiCard
      :label="$t('tires.tireSetSummary.mostWorn')"
      :value="summary.mostWorn ? formatPercent(summary.mostWorn.pct) : null"
      :unit="summary.mostWorn ? ` ${$t('tires.tireSetSummary.ofLifespan')}` : undefined"
    >
      <p v-if="summary.mostWorn" class="mt-1 text-xs text-slate-300">{{ positionLabel(summary.mostWorn.position) }} · {{ summary.mostWorn.name }}</p>
    </KpiCard>
    <KpiCard :label="$t('tires.tireSetSummary.nextReplacement')" :value="replacement" :tone="summary.nextReplacement && isForecastDue(summary.nextReplacement) ? 'warning' : 'default'" />
  </div>
</template>
