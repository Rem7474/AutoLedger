<script setup lang="ts" generic="T extends string | number">
import type { Component } from 'vue'
import { ref } from 'vue'
import { nextSegment } from '@/utils/segmentedKeys'

// An exclusive toggle for a chart (period, metric): a radiogroup with one selected segment, arrow keys move the
// selection (roving tabindex), Home and End jump to the ends. The selected segment uses the shared selected look.
export interface SegmentOption<V> {
  value: V
  label: string
  icon?: Component
}
const props = defineProps<{
  options: SegmentOption<T>[]
  label: string
}>()
const model = defineModel<T>({ required: true })
const buttons = ref<HTMLButtonElement[]>([])

function select(i: number) {
  const opt = props.options[i]
  if (!opt) return
  model.value = opt.value
  buttons.value[i]?.focus()
}

function onKey(e: KeyboardEvent, i: number) {
  const next = nextSegment(e.key, i, props.options.length)
  if (next < 0) return
  e.preventDefault()
  select(next)
}

// The tab stop is the selected segment, or the first one when the value matches none.
const tabbable = (i: number) => {
  const sel = props.options.findIndex((o) => o.value === model.value)
  return i === (sel < 0 ? 0 : sel)
}
</script>

<template>
  <div role="radiogroup" :aria-label="label" class="inline-flex gap-1 rounded-xl border border-slate-800 bg-slate-950 p-1">
    <button
      v-for="(o, i) in options"
      :key="String(o.value)"
      :ref="(el) => { if (el) buttons[i] = el as HTMLButtonElement }"
      type="button"
      role="radio"
      :aria-checked="o.value === model"
      :tabindex="tabbable(i) ? 0 : -1"
      :class="['chip min-h-11 min-w-11 sm:min-h-9', o.value === model && 'chip-selected']"
      @click="select(i)"
      @keydown="onKey($event, i)"
    >
      <component :is="o.icon" v-if="o.icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
      <span>{{ o.label }}</span>
    </button>
  </div>
</template>
