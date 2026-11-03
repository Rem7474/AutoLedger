<script setup lang="ts">
import { computed } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import { PULL_THRESHOLD } from '@/utils/pullRefresh'

// The round badge that comes down from under the top bar while the page is pulled, and spins while it reloads.
const props = defineProps<{ offset: number; armed: boolean; pulling: boolean; refreshing: boolean }>()

const visible = computed(() => props.offset > 0 || props.refreshing)
const progress = computed(() => Math.min(1, props.offset / PULL_THRESHOLD))
</script>

<template>
  <div
    v-if="visible"
    class="pointer-events-none fixed left-1/2 top-16 z-30 -ml-5 flex h-10 w-10 items-center justify-center rounded-full border border-slate-700 bg-slate-800 shadow-lg"
    :class="{ 'transition-transform duration-200': !pulling }"
    :style="{ transform: `translateY(${offset - 48}px)`, opacity: refreshing ? 1 : progress }"
    role="status"
  >
    <RefreshCw
      class="h-5 w-5"
      :class="[armed || refreshing ? 'text-rose-400' : 'text-slate-400', refreshing ? 'animate-spin' : '']"
      :style="refreshing ? undefined : { transform: `rotate(${progress * 270}deg)` }"
      aria-hidden="true"
    />
    <span class="sr-only">{{ refreshing ? $t('shell.pullRefresh.refreshing') : '' }}</span>
  </div>
</template>
