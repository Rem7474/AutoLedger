<script setup lang="ts">
import { computed } from 'vue'

// A small non-interactive counter (tab badge, quick-add, selection). Hidden at 0 unless `showZero`. A count that is
// only a number needs a spoken name: `label` ("3 costs to complete") is read instead of the bare figure.
const props = withDefaults(defineProps<{
  count: number
  tone?: 'attention' | 'neutral' | 'danger'
  showZero?: boolean
  label?: string
  max?: number
}>(), { tone: 'neutral', showZero: false, max: 99 })

const TONES = {
  attention: 'bg-warning-500/20 text-warning-300',
  neutral: 'bg-slate-700 text-slate-200',
  danger: 'bg-critical-500/20 text-critical-300',
} as const

const visible = computed(() => props.count > 0 || props.showZero)
const text = computed(() => (props.count > props.max ? `${props.max}+` : String(props.count)))
</script>

<template>
  <span
    v-if="visible"
    :class="['inline-flex h-5 min-w-5 shrink-0 items-center justify-center rounded-full px-1.5 text-[0.6875rem] font-bold leading-none tabular-nums', TONES[tone]]"
    :data-tone="tone"
    :role="label ? 'img' : undefined"
    :aria-label="label"
  >{{ text }}</span>
</template>
