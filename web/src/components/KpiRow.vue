<script setup lang="ts">
import { ChevronRight } from 'lucide-vue-next'

// A figure as one row of a list, for the phone: the label and what it covers on the left, the figure on the right, a
// chevron for the place it opens. The whole row is the target: a link when it goes to a page, a button when it opens
// something here. An unknown figure (`null`) reads as a dash, never as 0.
withDefaults(defineProps<{ label: string; value: string | null; hint?: string; to?: string; tone?: 'default' | 'sky' }>(), { tone: 'default', hint: undefined, to: undefined })
const emit = defineEmits<{ action: [] }>()
</script>

<template>
  <component
    :is="to ? 'router-link' : 'button'"
    v-bind="to ? { to } : { type: 'button' }"
    class="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors active:bg-slate-800/60"
    @click="!to && emit('action')"
  >
    <span class="min-w-0 flex-1">
      <span class="block text-sm font-semibold text-white">{{ label }}</span>
      <span v-if="hint" class="mt-0.5 block text-xs text-slate-400 line-clamp-2">{{ hint }}</span>
    </span>
    <span class="shrink-0 text-lg font-bold tabular-nums" :class="value == null ? 'text-slate-400' : tone === 'sky' ? 'text-sky-400' : 'text-white'">{{ value ?? '—' }}</span>
    <ChevronRight class="h-5 w-5 shrink-0 text-slate-500" aria-hidden="true" />
  </component>
</template>
