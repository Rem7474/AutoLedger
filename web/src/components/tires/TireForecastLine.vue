<script setup lang="ts">
import { computed } from 'vue'
import { CalendarClock } from 'lucide-vue-next'
import { t as translate, intlLocale } from '@/i18n'
import { distanceUnit, formatDistanceValue } from '@/units'
import { forecastMonthLabel, isForecastDue } from '@/utils/tires'

// Predicted replacement date of a tire, from the vehicle's mileage and the months the tire is on the car
const props = defineProps<{ forecast?: any | null }>()

const due = computed(() => isForecastDue(props.forecast?.replacement_date))
const label = computed(() => (due.value ? translate('tires.forecast.dueNow') : forecastMonthLabel(props.forecast.replacement_date)))
const tooltip = computed(() => {
  const f = props.forecast
  if (!f) return ''
  const months = (f.mounted_months as number[])
    .map((m) => new Date(2024, m - 1, 1).toLocaleDateString(intlLocale(), { month: 'short' }))
    .join(', ')
  const key = { learned: 'tooltipLearned', all_year: 'tooltipAllYear' }[f.months_source as string] ?? 'tooltipDefault'
  const base = translate(`tires.forecast.${key}`, {
    km: formatDistanceValue(Math.round(f.monthly_km)),
    unit: distanceUnit(),
    months,
  })
  const basis = translate(f.wear_basis === 'measured' ? 'tires.forecast.basisMeasured' : 'tires.forecast.basisDefault')
  return `${base} ${basis} ${translate('tires.forecast.rotation')}`
})
</script>

<template>
  <div v-if="forecast?.replacement_date" class="flex items-center justify-between gap-2 text-xs text-slate-400" :title="tooltip">
    <span class="flex items-center gap-1.5"><CalendarClock class="w-3.5 h-3.5" aria-hidden="true" />{{ $t('tires.forecast.label') }}</span>
    <span class="font-bold" :class="due ? 'text-danger-400' : 'text-slate-200'">{{ label }}</span>
  </div>
</template>
