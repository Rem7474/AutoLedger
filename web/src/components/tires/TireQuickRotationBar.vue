<script setup lang="ts">
import { ArrowUpDown, RefreshCw, Shuffle } from 'lucide-vue-next'
import { t } from '@/i18n'
import { api } from '@/services/api'
import { useConfirm } from '@/composables/useConfirm'
import { useSubmit } from '@/composables/useSubmit'
import { useVehicleStore } from '@/stores/vehicle'

// Swaps the mounted tires front/rear or crosswise at the current odometer.
defineProps<{ hasMountedTires: boolean }>()
const emit = defineEmits<{ rotated: [] }>()

const vehicleStore = useVehicleStore()
const { showAlert } = useConfirm()
const { pending: rotating, run: runOnce } = useSubmit()

async function quickRotateAction(mode: 'FRONT_BACK' | 'CROSS') {
  if (!vehicleStore.activeVehicle) return
  const odo = Math.round(vehicleStore.activeVehicle.current_odometer || 0)

  try {
    await api.quickRotateTires(vehicleStore.activeVehicle.id, { mode, odometer: odo })
    emit('rotated')
  } catch (err: any) {
    showAlert(t('tires.tiresView.rotationError', { message: err.message }), t('shell.confirm.error'), 'danger')
  }
}
const handleQuickRotate = (mode: 'FRONT_BACK' | 'CROSS') => runOnce(() => quickRotateAction(mode))
</script>

<template>
  <div class="bg-slate-900 border border-slate-800 p-3 rounded-2xl flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs">
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
        <ArrowUpDown class="w-3.5 h-3.5 text-blue-400" />
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
