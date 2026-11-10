<script setup lang="ts">
import type { Component } from 'vue'
import { Plus, Ban, ExternalLink } from '@/icons'

// The two taps that qualify an item of a "to qualify" queue: accept it (enter the toll, create the trip)
// or rule it out (no toll, not a trip).
// The second tap is optional: a charge with no cost at all cannot be ruled out, only completed.
// `primaryHref` makes the first tap a link that opens elsewhere (TeslaMate) instead of a button. `stretch` makes the two buttons share the width of their row with a 44px height on a phone; `secondaryIcon` replaces
// the "rule out" icon where the second tap confirms something instead (a free charge).
defineProps<{ primaryLabel: string; primaryTitle: string; secondaryLabel?: string; secondaryTitle?: string; secondaryIcon?: Component; primaryHref?: string; stretch?: boolean; busy?: boolean }>()
const emit = defineEmits<{ primary: []; secondary: [] }>()
</script>

<template>
  <div class="flex items-center gap-1" :class="stretch && 'max-sm:w-full max-sm:gap-2'">
    <a
      v-if="primaryHref"
      :href="primaryHref"
      target="_blank"
      rel="noopener noreferrer"
      class="px-2.5 py-1 text-xs font-semibold rounded-lg border border-warning-500/40 bg-warning-500/10 text-warning-400 hover:bg-warning-500/20 flex items-center gap-1"
      :class="stretch && 'max-sm:min-h-11 max-sm:flex-1 max-sm:justify-center max-sm:rounded-xl max-sm:text-sm'"
      :title="primaryTitle"
    >
      <ExternalLink class="w-3.5 h-3.5" /> {{ primaryLabel }}
    </a>
    <button
      v-else
      type="button"
      :disabled="busy"
      @click="emit('primary')"
      class="px-2.5 py-1 text-xs font-semibold rounded-lg border border-warning-500/40 bg-warning-500/10 text-warning-400 hover:bg-warning-500/20 flex items-center gap-1 disabled:opacity-50"
      :class="stretch && 'max-sm:min-h-11 max-sm:flex-1 max-sm:justify-center max-sm:rounded-xl max-sm:text-sm'"
      :title="primaryTitle"
    >
      <Plus class="w-3.5 h-3.5" /> {{ primaryLabel }}
    </button>
    <button
      v-if="secondaryLabel"
      type="button"
      :disabled="busy"
      @click="emit('secondary')"
      class="px-2.5 py-1 text-xs font-semibold rounded-lg border border-slate-700 bg-slate-800 text-slate-400 hover:text-white flex items-center gap-1 disabled:opacity-50"
      :class="stretch && 'max-sm:min-h-11 max-sm:flex-1 max-sm:justify-center max-sm:rounded-xl max-sm:text-sm max-sm:text-slate-200'"
      :title="secondaryTitle"
    >
      <component :is="secondaryIcon ?? Ban" class="w-3.5 h-3.5" /> {{ secondaryLabel }}
    </button>
  </div>
</template>
