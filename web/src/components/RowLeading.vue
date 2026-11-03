<script setup lang="ts">
import type { Component } from 'vue'
import { Check } from 'lucide-vue-next'

// The round glyph at the start of a list row on a phone. At rest it says what the row is; once the row is selected it
// becomes the rose check, in the same place, so starting a selection moves nothing. It is not interactive: the row's
// open button sits above it and toggles the row while a selection is going on. From sm the row has a checkbox instead.
// `warn` tints it for a row that waits for the user (a cost to complete).
defineProps<{ icon: Component; selected: boolean; warn?: boolean }>()
</script>

<template>
  <span
    class="grid size-10 shrink-0 place-items-center rounded-full transition-colors duration-150 sm:hidden"
    :class="selected ? 'bg-rose-600 text-white' : warn ? 'bg-warning-500/15 text-warning-400' : 'bg-slate-800 text-slate-300'"
    aria-hidden="true"
  >
    <Check v-if="selected" class="size-5" />
    <component :is="icon" v-else class="size-5" />
  </span>
</template>
