<script setup lang="ts">
import { formatNumber, formatPercent } from '@/utils/numbers'
import type { Component } from 'vue'
import { distanceUnit, perDistance } from '@/units'
import { formatAmount } from '@/currency'

// One cost of a breakdown: an icon, its label (with badges and a formula line if any), its amount, its share of the
// total and its cost per km. Shared by the drive, trip and carpool details.
defineProps<{ icon: Component; label: string; sub?: string; amount: number; sharePct: number; costPerKm: number; currency: string }>()
</script>

<template>
  <div class="bg-slate-800/40 border border-slate-800 p-3 rounded-xl flex items-center justify-between">
    <div class="flex items-center gap-3">
      <div class="p-2 rounded-lg bg-slate-800 text-slate-300">
        <component :is="icon" class="w-4 h-4" />
      </div>
      <div>
        <div class="text-xs font-semibold text-white flex items-center gap-1.5">
          {{ label }}
          <slot name="badge" />
        </div>
        <div v-if="sub" class="text-xs text-slate-400 tabular-nums">{{ sub }}</div>
      </div>
    </div>
    <div class="text-right">
      <div class="text-sm font-bold tabular-nums text-white">{{ formatAmount(amount, currency) }}</div>
      <div class="text-xs text-slate-400 font-normal font-sans">({{ formatPercent(sharePct) }}) · <span>{{ formatAmount(perDistance(costPerKm), currency, 3) }}/{{ distanceUnit() }}</span></div>
    </div>
  </div>
</template>
