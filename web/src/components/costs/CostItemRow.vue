<script setup lang="ts">
import { formatPercent } from '@/utils/numbers'
import type { Component } from 'vue'
import { distanceUnit, perDistance } from '@/units'
import { formatAmount } from '@/currency'

// One cost of a breakdown: an icon tinted with the color of its category (the one of the donut), its label (with
// badges and a formula line if any), its amount, its share of the total and its cost per km. Shared by the month,
// drive, trip and carpool details.
// - `sharePct` and `costPerKm` are left out for a line that is not part of the total (a credit such as the carpool
//   revenue); `credit` shows its amount as money coming back.
// - Slots: `badge` (next to the label), `aside` (under the figures), `actions` (buttons next to the figures) and
//   `below` (inside the card, under the line, for what the line holds: expenses, a form).
const props = defineProps<{
  icon: Component
  label: string
  sub?: string
  amount: number
  sharePct?: number
  costPerKm?: number
  currency: string
  color?: string
  credit?: boolean
}>()

// The color at 15 % opacity behind the icon (the colors of the categories are 6-digit hex values)
const tileStyle = (): Record<string, string> | undefined => (props.color ? { backgroundColor: `${props.color}26`, color: props.color } : undefined)
</script>

<template>
  <div class="bg-slate-800/40 border border-slate-800 p-3 rounded-xl space-y-2">
    <div class="flex items-center justify-between gap-3">
      <div class="flex items-center gap-3 min-w-0">
        <div class="p-2 rounded-lg shrink-0" :class="color ? '' : 'bg-slate-800 text-slate-300'" :style="tileStyle()" data-testid="cost-row-icon">
          <component :is="icon" class="w-4 h-4" />
        </div>
        <div class="min-w-0">
          <div class="text-xs font-semibold text-white flex items-center gap-1.5 flex-wrap">
            {{ label }}
            <slot name="badge" />
          </div>
          <div v-if="sub" class="text-xs text-slate-400 tabular-nums">{{ sub }}</div>
        </div>
      </div>
      <div class="flex items-center gap-2 shrink-0">
        <div class="text-right">
          <div class="text-sm font-bold tabular-nums" :class="credit ? 'text-success-400' : 'text-white'">{{ formatAmount(amount, currency) }}</div>
          <div v-if="sharePct !== undefined && costPerKm !== undefined" class="text-xs text-slate-400 font-normal font-sans">({{ formatPercent(sharePct) }}) · <span>{{ formatAmount(perDistance(costPerKm), currency, 3) }}/{{ distanceUnit() }}</span></div>
          <slot name="aside" />
        </div>
        <slot name="actions" />
      </div>
    </div>
    <slot name="below" />
  </div>
</template>
