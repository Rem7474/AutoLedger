<script setup lang="ts">
import type { Component } from 'vue'
import { Check } from '@/icons'

// The round glyph at the start of a list row on a phone. At rest it says what the row is; once the row is selected it
// becomes the rose check, in the same place, so starting a selection moves nothing. It is not interactive: the row's
// open button sits above it and toggles the row while a selection is going on. From sm the row has a checkbox instead.
// `warn` tints it for a row that waits for the user (a cost to complete); `small` and `always` are for a nested row (the legs of a trip): a smaller glyph that stays at every width. `tint` gives it the fill and icon colour of a
// cost theme (THEME_STYLES.*.tint).
defineProps<{ icon: Component; selected: boolean; warn?: boolean; tint?: string; small?: boolean; always?: boolean }>()
</script>

<template>
  <span
    class="grid shrink-0 place-items-center rounded-full transition-colors duration-150"
    :class="[small ? 'size-8' : 'size-10', always ? '' : 'sm:hidden', selected ? 'bg-rose-600 text-white' : warn ? 'bg-warning-500/15 text-warning-400' : (tint ?? 'bg-slate-800 text-slate-300')]"
    aria-hidden="true"
  >
    <Check v-if="selected" :class="small ? 'size-4' : 'size-5'" />
    <component :is="icon" v-else :class="small ? 'size-4' : 'size-5'" />
  </span>
</template>
