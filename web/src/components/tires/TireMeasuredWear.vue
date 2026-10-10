<script setup lang="ts">
import { formatNumber, formatPercent } from '@/utils/numbers'
import TireWearBar from '@/components/tires/TireWearBar.vue'

// Wear measured from tread depth, shown next to the theoretical wear by distance
defineProps<{ stat: any }>()
</script>

<template>
  <div v-if="stat.logs_count > 0" class="space-y-1.5">
    <div class="flex items-center justify-between text-xs text-slate-400">
      <span>{{ $t('tires.measuredWear.label') }} · {{ $t('tires.measuredWear.value', { depth: formatNumber(Number(stat.current_depth_mm), 1), pct: formatPercent(Number(stat.wear_percentage)) }) }}</span>
    </div>
    <TireWearBar :pct="stat.wear_percentage" :condition="stat.condition" />
  </div>
  <p v-else class="text-xs text-slate-400">{{ $t('tires.measuredWear.none') }}</p>
</template>
