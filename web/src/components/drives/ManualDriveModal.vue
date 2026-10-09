<script setup lang="ts">
import NumberInput from '@/components/NumberInput.vue'
import { ref, watch } from 'vue'
import { t } from '@/i18n'
import { api } from '@/services/api'
import { useVehicleStore } from '@/stores/vehicle'
import { usePreferencesStore } from '@/stores/preferences'
import { distanceUnit } from '@/units'
import { toLocalDateTimeInput } from '@/utils/dates'
import AppDatePicker from '@/components/AppDatePicker.vue'
import DistanceInput from '@/components/DistanceInput.vue'
import ModalShell from '@/components/ModalShell.vue'
import { Plus, Calendar, MapPin, Gauge, Zap } from 'lucide-vue-next'

const props = defineProps<{
  open: boolean
  vehicleId: string
  drive?: any | null
}>()

const emit = defineEmits<{
  (e: 'update:open', val: boolean): void
  (e: 'saved', drive: any): void
}>()

const vehicleStore = useVehicleStore()
const prefs = usePreferencesStore()

const loading = ref(false)
const error = ref('')

const startTime = ref(toLocalDateTimeInput())
const endTime = ref('')
const distanceKm = ref<number | null>(null)
const energyKwh = ref<number | null>(null)
const startAddress = ref('')
const endAddress = ref('')
const startOdometer = ref<number | null>(null)
const endOdometer = ref<number | null>(null)
const selectedTag = ref<'Pro' | 'Perso' | ''>('')

watch(
  () => props.open,
  (isOpen) => {
    if (!isOpen) return
    error.value = ''
    if (props.drive) {
      const d = props.drive
      startTime.value = toLocalDateTimeInput(d.start_time || new Date())
      endTime.value = d.end_time ? toLocalDateTimeInput(d.end_time) : ''
      distanceKm.value = d.distance_km ?? null
      // An estimated energy stays empty so saving keeps it estimated (the server recomputes it from the distance)
      energyKwh.value = d.energy_estimated ? null : (d.energy_consumed_kwh ?? null)
      startAddress.value = d.start_address || ''
      endAddress.value = d.end_address || ''
      startOdometer.value = d.start_odometer ?? null
      endOdometer.value = d.end_odometer ?? null
      selectedTag.value = d.tags?.includes('Pro') ? 'Pro' : d.tags?.includes('Perso') ? 'Perso' : ''
    } else {
      startTime.value = toLocalDateTimeInput()
      endTime.value = ''
      distanceKm.value = null
      energyKwh.value = null
      startAddress.value = ''
      endAddress.value = ''
      const odo = vehicleStore.activeVehicle?.current_odometer
      startOdometer.value = odo ? Math.round(odo) : null
      endOdometer.value = null
      selectedTag.value = ''
    }
  },
  { immediate: true },
)

function close() {
  emit('update:open', false)
}

async function handleSubmit() {
  const targetVehicleId = props.vehicleId || vehicleStore.activeVehicle?.id
  if (!targetVehicleId) return
  error.value = ''
  if (!distanceKm.value || distanceKm.value <= 0) {
    error.value = t('drives.manualModal.distanceRequired')
    return
  }

  loading.value = true
  try {
    const payload: any = {
      start_time: new Date(startTime.value).toISOString(),
      distance_km: Number(distanceKm.value),
      start_address: startAddress.value.trim() || undefined,
      end_address: endAddress.value.trim() || undefined,
      start_odometer: startOdometer.value ? Number(startOdometer.value) : undefined,
      end_odometer: endOdometer.value ? Number(endOdometer.value) : undefined,
      energy_consumed_kwh: energyKwh.value ? Number(energyKwh.value) : undefined,
      tags: selectedTag.value ? [selectedTag.value] : [],
    }

    if (endTime.value) {
      payload.end_time = new Date(endTime.value).toISOString()
    }

    let saved: any
    if (props.drive) {
      saved = await api.updateDrive(targetVehicleId, props.drive.id, payload)
    } else {
      saved = await api.createDrive(targetVehicleId, payload)
    }

    emit('saved', saved)
    close()
  } catch (err: any) {
    error.value = err.message || t('common.error')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <ModalShell :open="open" @update:open="close" size="md" :icon="Plus" icon-class="text-rose-400" :title="drive ? $t('drives.manualModal.titleEdit') : $t('drives.manualModal.titleNew')">
    <form id="manual-drive-form" @submit.prevent="handleSubmit" class="space-y-4">
        <div v-if="error" class="p-3 bg-danger-500/10 border border-danger-500/20 rounded-xl text-xs text-danger-400">
          {{ error }}
        </div>

        <!-- Times -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label for="manual-drive-start-time" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">
              {{ $t('drives.manualModal.dateStart') }}
            </label>
            <AppDatePicker id="manual-drive-start-time" v-model="startTime" enable-time-picker required />
          </div>
          <div>
            <label for="manual-drive-end-time" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">
              {{ $t('drives.manualModal.dateEnd') }}
            </label>
            <AppDatePicker id="manual-drive-end-time" v-model="endTime" enable-time-picker />
          </div>
        </div>

        <!-- Distance & Energy -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label for="manual-drive-distance" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">
              {{ $t('drives.manualModal.distance', { unit: distanceUnit() }) }}
            </label>
            <DistanceInput
              id="manual-drive-distance"
              v-model="distanceKm"
              required
              min="0.1"
              step="0.1"
              class="field"
            />
          </div>
          <div>
            <label for="manual-drive-energy" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">
              {{ $t('drives.manualModal.energy') }}
            </label>
            <NumberInput
              id="manual-drive-energy"
              v-model="energyKwh"
              min="0"
              :placeholder="$t('drives.manualModal.energyPlaceholder')"
              class="field placeholder-slate-500"
            />
          </div>
        </div>

        <!-- Locations -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label for="manual-drive-start-address" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">
              {{ $t('drives.manualModal.origin') }}
            </label>
            <input
              id="manual-drive-start-address"
              v-model="startAddress"
              type="text"
              :placeholder="$t('drives.manualModal.originPlaceholder')"
              class="field placeholder-slate-500"
            />
          </div>
          <div>
            <label for="manual-drive-end-address" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">
              {{ $t('drives.manualModal.destination') }}
            </label>
            <input
              id="manual-drive-end-address"
              v-model="endAddress"
              type="text"
              :placeholder="$t('drives.manualModal.destinationPlaceholder')"
              class="field placeholder-slate-500"
            />
          </div>
        </div>

        <!-- Odometers -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label for="manual-drive-start-odo" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">
              {{ $t('drives.manualModal.startOdo', { unit: distanceUnit() }) }}
            </label>
            <DistanceInput
              id="manual-drive-start-odo"
              v-model="startOdometer"
              step="any"
              min="0"
              class="field"
            />
          </div>
          <div>
            <label for="manual-drive-end-odo" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">
              {{ $t('drives.manualModal.endOdo', { unit: distanceUnit() }) }}
            </label>
            <DistanceInput
              id="manual-drive-end-odo"
              v-model="endOdometer"
              step="any"
              min="0"
              class="field"
            />
          </div>
        </div>

        <!-- Tag (Pro / Perso) -->
        <div v-if="prefs.proPersoEnabled">
          <span class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">
            {{ $t('drives.manualModal.tag') }}
          </span>
          <div class="flex items-center gap-2">
            <button
              type="button"
              @click="selectedTag = selectedTag === 'Pro' ? '' : 'Pro'"
              class="px-3 py-1.5 rounded-lg text-xs font-semibold border transition-all"
              :class="selectedTag === 'Pro' ? 'bg-sky-500/20 text-sky-300 border-sky-500' : 'bg-slate-800 text-slate-400 border-slate-700 hover:text-white'"
            >
              Pro
            </button>
            <button
              type="button"
              @click="selectedTag = selectedTag === 'Perso' ? '' : 'Perso'"
              class="px-3 py-1.5 rounded-lg text-xs font-semibold border transition-all"
              :class="selectedTag === 'Perso' ? 'bg-success-500/20 text-success-300 border-success-500' : 'bg-slate-800 text-slate-400 border-slate-700 hover:text-white'"
            >
              Perso
            </button>
          </div>
        </div>
    </form>

    <template #footer>
      <button type="button" @click="close" class="btn btn-lg btn-secondary">
        {{ $t('common.cancel') }}
      </button>
      <button type="submit" form="manual-drive-form" :disabled="loading" class="btn btn-lg btn-primary">
        {{ loading ? $t('drives.manualModal.saving') : $t('drives.manualModal.save') }}
      </button>
    </template>
  </ModalShell>
</template>
