<script setup lang="ts">
import { ref } from 'vue'
import { ArrowUpDown, RefreshCw, Shuffle } from '@/icons'
import { t } from '@/i18n'
import { api } from '@/services/api'
import { useConfirm } from '@/composables/useConfirm'
import { deleteWithUndo } from '@/composables/useUndoableDelete'
import { rotatedTires, type RotationMode } from '@/utils/tireRotation'
import { useVehicleStore } from '@/stores/vehicle'

// Swaps the mounted tires front/rear or crosswise at the current odometer. The new layout shows at once and a snackbar
// offers to take it back; the server is only asked when that time is over (a rotation closes and opens mount sessions,
// so playing the opposite rotation back would leave sessions of 0 km).
const props = defineProps<{ hasMountedTires: boolean; tires: any[] }>()
const emit = defineEmits<{ rotated: []; layout: [tires: any[]] }>()

const vehicleStore = useVehicleStore()
const { showAlert } = useConfirm()
const rotating = ref(false)

function handleQuickRotate(mode: RotationMode) {
  const vehicle = vehicleStore.activeVehicle
  if (!vehicle || rotating.value) return
  const odometer = Math.round(vehicle.current_odometer || 0)
  const before = props.tires
  deleteWithUndo({
    message: t('tires.tiresView.rotated'),
    hide: () => {
      rotating.value = true
      emit('layout', rotatedTires(before, mode))
    },
    restore: () => {
      rotating.value = false
      emit('layout', before)
    },
    commit: () => api.quickRotateTires(vehicle.id, { mode, odometer }),
    committed: () => {
      rotating.value = false
      emit('rotated')
    },
    onError: (err: any) => void showAlert(t('tires.tiresView.rotationError', { message: err.message }), t('shell.confirm.error'), 'danger'),
  })
}
</script>

<template>
  <div class="p-3 card flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs">
    <div class="flex items-center gap-2 text-slate-300 font-semibold">
      <RefreshCw class="w-4 h-4 text-rose-400" />
      <span>{{ $t('tires.tiresView.quickVehicleRotationsIn1') }}</span>
    </div>
    <div class="flex items-center gap-2 flex-wrap">
      <button
        @click="handleQuickRotate('FRONT_BACK')"
        :disabled="rotating || !hasMountedTires"
        :title="$t('tires.tiresView.rotateTooltip', { pairs: 'FL ⇄ RL, FR ⇄ RR' })"
        class="tap btn btn-secondary"
      >
        <ArrowUpDown class="w-3.5 h-3.5 text-slate-400" />
        {{ $t('tires.tiresView.frontRear') }}
      </button>
      <button
        @click="handleQuickRotate('CROSS')"
        :disabled="rotating || !hasMountedTires"
        :title="$t('tires.tiresView.rotateTooltip', { pairs: 'FL ⇄ RR, FR ⇄ RL' })"
        class="tap btn btn-secondary"
      >
        <Shuffle class="w-3.5 h-3.5 text-sky-400" />
        {{ $t('tires.tiresView.crossRotation') }}
      </button>
    </div>
  </div>
</template>
