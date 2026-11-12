<script setup lang="ts">
import ModalShell from '@/components/ModalShell.vue'
import StatusLabel from '@/components/StatusLabel.vue'
import DetailDeleteButton from '@/components/DetailDeleteButton.vue'
import { Copy, Edit2, Shuffle } from '@/icons'
import { distanceUnit, formatDistanceValue } from '@/units'
import { formatDate, positionLabel } from '@/utils/tires'

// One fitting of a tire: its dates and distance, the actions that start from it (copy, duplicate to other tires,
// edit) as labelled buttons, and the delete at the bottom. It only displays: every action is emitted for the page to run.
defineProps<{ session: any | null; kmDriven: number | null }>()
const emit = defineEmits<{
  copy: [session: any]
  duplicate: [session: any]
  edit: [session: any]
  delete: [session: any]
}>()
const open = defineModel<boolean>('open', { required: true })

function run(action: 'copy' | 'duplicate' | 'edit' | 'delete', session: any) {
  open.value = false
  if (action === 'copy') emit('copy', session)
  else if (action === 'duplicate') emit('duplicate', session)
  else if (action === 'edit') emit('edit', session)
  else emit('delete', session)
}
</script>

<template>
  <ModalShell v-if="session" v-model:open="open" nested size="sm" body-class="space-y-4" footer-class="justify-end">
    <template #title>
      <div class="min-w-0">
        <h3 class="truncate text-base font-bold text-white">{{ positionLabel(session.position) }}</h3>
        <StatusLabel :tone="session.dismounted_date ? 'neutral' : 'info'">
          {{ session.dismounted_date ? $t('tires.tireHistoryModal.sessionOver') : $t('tires.tireHistoryModal.currentlyFitted') }}
        </StatusLabel>
      </div>
    </template>

    <div class="space-y-3 rounded-2xl bg-slate-800 p-3.5 text-xs">
      <div>
        <div class="text-slate-400">{{ $t('tires.tireHistoryModal.fitting') }}</div>
        <div class="font-medium text-slate-200">{{ $t('tires.tireHistoryModal.dateAtKm', { unit: distanceUnit(), date: formatDate(session.mounted_date), km: formatDistanceValue(session.mounted_odometer) }) }}</div>
      </div>
      <div>
        <div class="text-slate-400">{{ $t('tires.tireHistoryModal.removal') }}</div>
        <div class="font-medium text-slate-200">
          {{ session.dismounted_date ? $t('tires.tireHistoryModal.removedAt', { unit: distanceUnit(), date: formatDate(session.dismounted_date), odometer: formatDistanceValue(session.dismounted_odometer) }) : $t('tires.tireHistoryModal.currentlyOnVehicle') }}
        </div>
      </div>
      <p v-if="session.notes" class="italic text-slate-400">"{{ session.notes }}"</p>
      <div class="font-bold text-white">{{ $t('tires.tireHistoryModal.kmDriven', { unit: distanceUnit(), distance_km: formatDistanceValue(kmDriven ?? 0) }) }}</div>
    </div>

    <div class="flex flex-col gap-2">
      <button type="button" class="btn btn-secondary justify-start" @click="run('edit', session)">
        <Edit2 class="h-4 w-4" aria-hidden="true" />{{ $t('tires.tireHistoryModal.editTheSession') }}
      </button>
      <button type="button" class="btn btn-secondary justify-start" @click="run('copy', session)">
        <Copy class="h-4 w-4" aria-hidden="true" />{{ $t('tires.tireHistoryModal.copyThisSessionSData') }}
      </button>
      <button type="button" class="btn btn-secondary justify-start" @click="run('duplicate', session)">
        <Shuffle class="h-4 w-4" aria-hidden="true" />{{ $t('tires.tireHistoryModal.duplicateToOtherTires') }}
      </button>
    </div>

    <template #footer>
      <DetailDeleteButton class="mr-auto" :aria-label="$t('tires.tireHistoryModal.deleteTheSession')" @delete="run('delete', session)" />
      <button type="button" class="btn btn-lg btn-secondary" @click="open = false">{{ $t('common.close') }}</button>
    </template>
  </ModalShell>
</template>
