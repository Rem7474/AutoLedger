<script setup lang="ts">
import type { Component } from 'vue'

defineProps<{
  title: string
  icon: Component
}>()
</script>

<template>
  <!-- Phone: a compact app-bar line, the icon (no tile) and the title with its actions on its right (they drop under it
       when they do not fit), no description. From sm: the icon, the title, the description and the actions on the right. -->
  <header class="flex flex-wrap items-center justify-between gap-x-3 gap-y-2 sm:flex-nowrap sm:gap-4">
    <div class="min-w-0">
      <div class="flex items-center gap-3">
        <div class="shrink-0 rounded-xl border border-rose-500/20 bg-rose-500/10 p-2 text-rose-400 max-sm:border-0 max-sm:bg-transparent max-sm:p-0">
          <component :is="icon" class="h-6 w-6 max-sm:h-5 max-sm:w-5" aria-hidden="true" />
        </div>
        <h2 class="truncate text-2xl font-bold tracking-tight text-white">{{ title }}</h2>
      </div>
      <p v-if="$slots.default" class="mt-1 text-sm text-slate-400 max-sm:hidden"><slot /></p>
    </div>
    <div v-if="$slots.actions" class="flex flex-wrap items-center gap-2 max-sm:ml-auto">
      <slot name="actions" />
    </div>
  </header>
</template>
