<script setup lang="ts">
import type { Component } from 'vue'
import { ChevronRight } from 'lucide-vue-next'

// One line of the vehicle sheet: a glyph, what it opens, one quiet detail, a chevron. A link opens a screen of its own,
// a button opens a sheet over the sheet.
defineProps<{ icon: Component; title: string; detail?: string | null; to?: string }>()
defineEmits<{ click: [] }>()
</script>

<template>
  <component
    :is="to ? 'router-link' : 'button'"
    :to="to"
    :type="to ? undefined : 'button'"
    class="card flex w-full items-center gap-3 px-4 py-3 text-left transition-colors active:bg-slate-800/60"
    @click="$emit('click')"
  >
    <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-slate-800 text-slate-300"><component :is="icon" class="h-5 w-5" aria-hidden="true" /></span>
    <span class="min-w-0 flex-1">
      <span class="block text-sm font-semibold text-white">{{ title }}</span>
      <span v-if="detail" class="block truncate text-xs text-slate-400">{{ detail }}</span>
    </span>
    <ChevronRight class="h-5 w-5 shrink-0 text-slate-500" aria-hidden="true" />
  </component>
</template>
