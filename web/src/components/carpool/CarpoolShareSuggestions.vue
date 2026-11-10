<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { api } from '@/services/api'
import { formatDistance } from '@/units'
import { formatTripDates } from '@/utils/drives'

// The latest drives no carpool covers yet, each one a tap away from being shared. Nothing to propose is nothing shown;
// a failed read hides the section rather than the page.
const props = defineProps<{ vehicleId: string; canEdit: boolean; refreshKey?: number }>()
const emit = defineEmits<{ share: [driveId: string] }>()

const SHOWN = 3
const drives = ref<any[]>([])

async function load() {
  if (!props.vehicleId) return
  try {
    const res = await api.getDrives(props.vehicleId, { carpool: 'no', page: 1, limit: SHOWN })
    drives.value = res.drives ?? []
  } catch {
    drives.value = []
  }
}
const route = (d: any) => [d.start_address, d.end_address].filter(Boolean).join(' → ')
const when = (d: any) => formatTripDates({ start_time: d.start_time, end_time: d.end_time })

onMounted(() => void load())
watch(() => [props.vehicleId, props.refreshKey], () => void load())
</script>

<template>
  <section v-if="canEdit && drives.length" class="space-y-2" data-testid="carpool-share-suggestions">
    <div class="px-1">
      <h2 class="text-sm font-semibold text-white">{{ $t('carpool.shareSuggestions.title') }}</h2>
      <p class="text-xs text-slate-400">{{ $t('carpool.shareSuggestions.hint') }}</p>
    </div>
    <div v-for="d in drives" :key="d.id" class="card flex items-center gap-3 px-4 py-3">
      <div class="min-w-0 flex-1">
        <p class="truncate text-sm font-semibold text-white" :title="route(d)">{{ route(d) || when(d) }}</p>
        <p class="truncate text-xs text-slate-400">{{ when(d) }} · {{ formatDistance(d.distance_km) }}</p>
      </div>
      <button type="button" class="btn btn-secondary shrink-0" :aria-label="$t('carpool.shareSuggestions.shareAria', { date: when(d) })" @click="emit('share', d.id)">
        {{ $t('carpool.shareSuggestions.share') }}
      </button>
    </div>
  </section>
</template>
