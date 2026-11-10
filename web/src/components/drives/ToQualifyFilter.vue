<script setup lang="ts">
import { AlertTriangle } from '@/icons'

// Filter toggle showing how many items (drives to qualify, trips detected) still wait for a decision.
// `label` replaces the drives wording for another list (charges to complete).
defineProps<{ count: number; active: boolean; title: string; label?: string }>()
const emit = defineEmits<{ toggle: [] }>()
</script>

<template>
  <button
    v-if="count > 0 || active"
    type="button"
    @click="emit('toggle')"
    class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors flex items-center gap-1.5"
    :class="active ? 'bg-warning-500/20 text-warning-400 border border-warning-500/30' : 'text-warning-400/80 hover:text-warning-300 border border-transparent'"
    :title="title"
  >
    <AlertTriangle class="w-3.5 h-3.5" />
    {{ label ?? $t('drives.drivesView.toQualify', { unqualifiedCount: count }) }}
  </button>
</template>
