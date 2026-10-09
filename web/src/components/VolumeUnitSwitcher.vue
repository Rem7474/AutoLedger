<script setup lang="ts">
import { computed } from 'vue'
import { Fuel } from 'lucide-vue-next'
import { t } from '@/i18n'
import { currentVolumeUnit, setVolumeUnit, SUPPORTED_VOLUME_UNITS, type VolumeUnit } from '@/units'
import { useAuthStore } from '@/stores/auth'
import { api } from '@/services/api'

// Called from the template (not precomputed) so the label follows a later language change.
const unitLabel = (u: VolumeUnit) => t(`account.volumeUnit_${u}`)

const authStore = useAuthStore()

const selected = computed({
  get: () => currentVolumeUnit(),
  set: (value: VolumeUnit) => {
    setVolumeUnit(value)
    if (authStore.isAuthenticated) api.updateVolumeUnit(value).catch(() => {})
  },
})
</script>

<template>
  <label class="flex items-center gap-1.5 text-xs text-slate-400">
    <Fuel class="w-3.5 h-3.5 shrink-0" aria-hidden="true" />
    <span class="sr-only">{{ $t('account.volumeUnit') }}</span>
    <select
      v-model="selected"
      class="field rounded-md text-slate-200"
    >
      <option v-for="u in SUPPORTED_VOLUME_UNITS" :key="u" :value="u">{{ unitLabel(u) }}</option>
    </select>
  </label>
</template>
