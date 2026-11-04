<script setup lang="ts">
import DistanceInput from '@/components/DistanceInput.vue'
import AppDatePicker from '@/components/AppDatePicker.vue'
import { distanceUnit } from '@/units'

// The form of an odometer reading, to add one or to edit one: a card on the page on a wide screen, the body of a sheet on a phone
defineProps<{ editing: boolean; submitting: boolean; inSheet?: boolean }>()
const emit = defineEmits<{ submit: []; cancel: [] }>()
const date = defineModel<string>('date', { required: true })
const odometer = defineModel<string>('odometer', { required: true })
const notes = defineModel<string>('notes', { required: true })
</script>

<template>
<form class="bg-slate-800 rounded-xl p-4 space-y-4 max-sm:bg-transparent max-sm:p-0" @submit.prevent="emit('submit')">
  <div v-if="!inSheet" class="flex items-center justify-between">
    <span class="text-xs font-bold text-sky-400">
      {{ editing ? $t('manual.odometerReadingsPanel.edit') : $t('manual.odometerReadingsPanel.new') }}
    </span>
    <button v-if="editing" type="button" class="text-xs text-slate-400 hover:text-white" @click="emit('cancel')">
      {{ $t('manual.odometerReadingsPanel.cancelTheEdit') }}
    </button>
  </div>

  <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
    <div>
      <label for="checkpoint-date" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('manual.odometerReadingsPanel.readingDate') }}</label>
      <AppDatePicker id="checkpoint-date" v-model="date" required size="sm" />
    </div>
    <div>
      <label for="checkpoint-odometer" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('manual.odometerReadingsPanel.mileageKm', { unit: distanceUnit() }) }}</label>
      <DistanceInput text
        id="checkpoint-odometer"
        v-model="odometer"
        step="1"
        min="0"
        max="2000000"
        required
        :placeholder="$t('common.example', { value: '45000' })"
        class="field"
      />
    </div>
    <div>
      <label for="checkpoint-notes" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('manual.odometerReadingsPanel.reasonEvent') }}</label>
      <input
        id="checkpoint-notes"
        v-model="notes"
        type="text"
        :placeholder="$t('manual.odometerReadingsPanel.eGRoadworthinessInspection')"
        class="field"
      />
    </div>
  </div>

  <div class="flex justify-end pt-1">
    <button type="submit" :disabled="submitting" class="btn btn-lg btn-primary max-sm:w-full">
      <span>{{ editing ? $t('manual.odometerReadingsPanel.update') : $t('manual.odometerReadingsPanel.add') }}</span>
    </button>
  </div>
</form>
</template>
