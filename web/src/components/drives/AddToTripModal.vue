<script setup lang="ts">
import { t } from '@/i18n'
import { ref, watch } from 'vue'
import { api } from '@/services/api'
import { useConfirm } from '@/composables/useConfirm'
import { Plus } from 'lucide-vue-next'
import ModalShell from '@/components/ModalShell.vue'

// Adds the selected drives to an existing trip group.
const props = defineProps<{ vehicleId: string; tripGroups: any[]; selectedDriveIds: string[] }>()
const emit = defineEmits<{ saved: [] }>()
const open = defineModel<boolean>('open', { required: true })
const { showAlert } = useConfirm()

const addToTripId = ref('')

watch(open, (isOpen) => {
  if (isOpen) addToTripId.value = props.tripGroups[0]?.id || ''
})

async function handleAddToTrip() {
  const tg = props.tripGroups.find((g) => g.id === addToTripId.value)
  if (!props.vehicleId || !tg) return
  const driveIds = Array.from(new Set([...(tg.drive_ids || []), ...props.selectedDriveIds]))
  try {
    await api.updateTripGroup(props.vehicleId, tg.id, { name: tg.name, notes: tg.notes, drive_ids: driveIds })
    open.value = false
    emit('saved')
  } catch (err: any) {
    showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger')
  }
}
</script>

<template>
  <ModalShell v-model:open="open" :title="$t('drives.addToTripModal.addDriveSToA', { length: selectedDriveIds.length })" :icon="Plus" icon-class="text-sky-400" size="sm">
    <form id="add-to-trip-form" @submit.prevent="handleAddToTrip">
      <p v-if="!tripGroups.length" class="text-xs text-slate-400">{{ $t('drives.addToTripModal.noExistingTripUseMerge') }}</p>
      <div v-else>
        <label for="add-to-trip" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('drives.addToTripModal.trip') }}</label>
        <select id="add-to-trip" v-model="addToTripId" class="field">
          <option v-for="tg in tripGroups" :key="tg.id" :value="tg.id">{{ $t('drives.addToTripModal.drives', { name: tg.name, length: tg.drive_ids.length }) }}</option>
        </select>
      </div>
    </form>
    <template #footer>
      <button type="button" @click="open = false" class="btn btn-lg btn-secondary">
        {{ $t('common.cancel') }}
      </button>
      <button type="submit" form="add-to-trip-form" :disabled="!tripGroups.length" :title="!tripGroups.length ? $t('drives.addToTripModal.noTrip') : undefined" class="btn btn-lg btn-primary">
        {{ $t('drives.addToTripModal.add') }}
      </button>
    </template>
  </ModalShell>
</template>
