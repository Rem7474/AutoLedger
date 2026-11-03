<script setup lang="ts">
import { ref, type Component } from 'vue'
import { ChartColumn, ChevronRight } from 'lucide-vue-next'
import ModalShell from '@/components/ModalShell.vue'
import { useIsMobile } from '@/composables/useIsMobile'

// A block that is not the list itself (the analysis above it, an export below it) would push the rows down the screen on a
// phone: there it is one quiet row that opens the block in a sheet. On a wide screen the block stays where it is. The slot
// is rendered once, in one place or the other; `bare` tells it not to draw its own card, since the sheet already frames
// it, and `close` lets it hand over to another sheet.
withDefaults(defineProps<{ title: string; hint?: string; icon?: Component; enabled?: boolean }>(), { enabled: true, icon: () => ChartColumn, hint: undefined })
// The default is a factory: Vue would otherwise call a function-valued default (an icon component is one) as if it built the value
const isMobile = useIsMobile()
const open = ref(false)
const noop = () => {}
const close = () => (open.value = false)
</script>

<template>
  <slot v-if="!isMobile" :bare="false" :close="noop" />
  <template v-else-if="enabled">
    <button
      type="button"
      class="flex w-full items-center gap-3 rounded-2xl border border-slate-800 bg-slate-900 px-4 py-3 text-left transition-colors active:bg-slate-800/60"
      aria-haspopup="dialog"
      @click="open = true"
    >
      <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-slate-800 text-slate-300"><component :is="icon" class="h-5 w-5" aria-hidden="true" /></span>
      <span class="min-w-0 flex-1">
        <span class="block text-sm font-semibold text-white">{{ title }}</span>
        <span v-if="hint" class="block truncate text-xs text-slate-400">{{ hint }}</span>
      </span>
      <ChevronRight class="h-5 w-5 shrink-0 text-slate-500" aria-hidden="true" />
    </button>
    <Teleport to="body">
      <ModalShell v-model:open="open" :title="title" size="lg">
        <slot :bare="true" :close="close" />
      </ModalShell>
    </Teleport>
  </template>
</template>
