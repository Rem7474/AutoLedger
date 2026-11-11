<script setup lang="ts">
import { Layers } from '@/icons'
import DriveLegRow from '@/components/drives/DriveLegRow.vue'

// The legs of a trip, as the same rows as in the trip's list: each opens the cost breakdown of the leg
defineProps<{ tripLegs: any[] }>()
const emit = defineEmits<{ open: [leg: any] }>()
</script>

<template>
  <div class="space-y-1.5">
    <h4 class="text-xs font-bold text-white flex items-center gap-1.5">
      <Layers class="w-3.5 h-3.5 text-slate-400" />
      {{ $t('drives.driveCostModal.tripLegs', { count: tripLegs.length }) }}
    </h4>
    <div class="space-y-1.5">
      <DriveLegRow class="bg-slate-800" v-for="leg in tripLegs" :key="leg.id" :leg="leg" @open="emit('open', $event)" />
    </div>
  </div>
</template>
