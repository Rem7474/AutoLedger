<script setup lang="ts">
import type { Component } from 'vue'
import { computed } from 'vue'

// A state that is not clickable: coloured text with an optional small icon, no background and no border, so it never
// reads as a button (a pill shape is `.chip`, a filter or a shortcut). The label is always written: colour never
// carries the state alone.
const props = withDefaults(defineProps<{
  tone?: 'attention' | 'info' | 'danger' | 'neutral'
  // Rare: an isolated banner or alert. A list row shows the text alone, or the dot.
  icon?: Component
  // A small filled dot of the tone's colour before the text.
  dot?: boolean
  size?: 'sm' | 'md'
}>(), { tone: 'neutral', size: 'sm', dot: false })

const TONES = {
  attention: 'text-warning-400',
  info: 'text-info-300',
  danger: 'text-critical-400',
  neutral: 'text-slate-400',
} as const
const DOTS = {
  attention: 'bg-warning-400',
  info: 'bg-info-300',
  danger: 'bg-critical-400',
  neutral: 'bg-slate-400',
} as const

const classes = computed(() => [
  'inline-flex max-w-full items-center gap-1 font-semibold leading-tight',
  props.size === 'md' ? 'text-sm' : 'text-xs',
  TONES[props.tone],
])
const iconClass = computed(() => (props.size === 'md' ? 'h-4 w-4' : 'h-3.5 w-3.5'))
</script>

<template>
  <span :class="classes" :data-tone="tone">
    <span v-if="dot && !icon" :class="['h-1.5 w-1.5 shrink-0 rounded-full', DOTS[tone]]" aria-hidden="true" />
    <component :is="icon" v-if="icon" :class="['shrink-0', iconClass]" aria-hidden="true" />
    <span class="truncate"><slot /></span>
  </span>
</template>
