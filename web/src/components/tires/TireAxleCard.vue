<script setup lang="ts">
import TireWheelCard from '@/components/tires/TireWheelCard.vue'

// One axle of the chassis view on a wide screen: a card with its two wheels, left and right.
defineProps<{
  title: string
  wheels: { pos: string; label: string }[]
  mounted: Record<string, any | null>
  selectedIds: string[]
  showModel?: boolean
  canMount?: boolean
}>()
const emit = defineEmits<{ open: [stat: any]; toggle: [tireId: string]; mount: [] }>()
</script>

<template>
  <section class="card p-4 sm:p-5" :aria-label="title">
    <h3 class="mb-2 text-sm font-bold text-white">{{ title }}</h3>
    <div class="grid grid-cols-1 gap-2 sm:grid-cols-2 sm:gap-4 sm:divide-x sm:divide-slate-800">
      <TireWheelCard
        v-for="w in wheels"
        :key="w.pos"
        :pos="w.pos"
        :label="w.label"
        :stat="mounted[w.pos]"
        :show-model="showModel"
        :selected="!!mounted[w.pos] && selectedIds.includes(mounted[w.pos].tire.id)"
        :selecting="selectedIds.length > 0"
        :can-mount="canMount"
        @open="emit('open', $event)"
        @toggle="emit('toggle', $event)"
        @mount="emit('mount')"
      />
    </div>
  </section>
</template>
