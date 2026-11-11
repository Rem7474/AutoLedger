<script setup lang="ts">
import { computed } from 'vue'
import GlossaryTip from '@/components/GlossaryTip.vue'
import { UNKNOWN_AMOUNT } from '@/currency'

// The second level of a figure group: a label and a figure with no frame, set under the lead figure (`KpiCard lead`).
// A figure with no data is not drawn (`keepEmpty` draws a dash instead, for one that explains itself in the default slot).
const props = defineProps<{
  label: string
  value: string | null
  unit?: string
  term?: string
  keepEmpty?: boolean
  // A judgement on the figure (good or late), never a way to tell figures apart
  tone?: 'default' | 'success' | 'warning'
}>()
const unknown = computed(() => props.value == null || props.value === UNKNOWN_AMOUNT)
const tones = { default: 'text-white', success: 'text-success-400', warning: 'text-warning-400' }
const shown = computed(() => !unknown.value || !!props.keepEmpty)
</script>

<template>
  <div v-if="shown" class="min-w-0">
    <div class="text-xs font-medium text-slate-400">
      {{ label }}
      <GlossaryTip v-if="term" :term="term" />
    </div>
    <div class="mt-0.5 text-xl font-bold tabular-nums" :class="unknown ? 'text-slate-400' : tones[tone ?? 'default']">
      <template v-if="unknown">—</template>
      <template v-else>{{ value }}<span v-if="unit" class="whitespace-nowrap text-xs font-medium text-slate-400">{{ unit }}</span></template>
    </div>
    <slot />
  </div>
</template>
