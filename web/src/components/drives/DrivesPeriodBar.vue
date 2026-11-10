<script setup lang="ts">
import { computed } from 'vue'
import { CalendarDays } from '@/icons'
import ModalShell from '@/components/ModalShell.vue'
import PeriodBar from '@/components/period/PeriodBar.vue'
import DrivesPeriodControls from '@/components/drives/DrivesPeriodControls.vue'
import { t } from '@/i18n'
import { canStepDrivesPeriod, drivesPeriodLabel, stepDrivesPeriod, type PeriodMode } from '@/utils/drives'

// The period of the drives list on a phone: the shared bar above the tab bar, its sheet holding the unit chips,
// the day/month/year pickers and the range. The page keeps its own period model (days and weeks, which the cost
// pages do not have).
defineProps<{ total: number }>()
const emit = defineEmits<{ change: [] }>()
const periodMode = defineModel<PeriodMode>('periodMode', { required: true })
const selectedMonth = defineModel<string>('selectedMonth', { required: true })
const anchorDay = defineModel<string>('anchorDay', { required: true })
const customFrom = defineModel<string>('customFrom', { required: true })
const customTo = defineModel<string>('customTo', { required: true })

const state = computed(() => ({ mode: periodMode.value, anchor: anchorDay.value, month: selectedMonth.value, from: customFrom.value, to: customTo.value }))
const label = computed(() => drivesPeriodLabel(state.value) || t(periodMode.value === 'CUSTOM' ? 'drives.drivesToolbar.units.CUSTOM' : 'drives.drivesToolbar.units.ALL'))

function step(dir: -1 | 1) {
  const next = stepDrivesPeriod(state.value, dir)
  if (!next) return
  anchorDay.value = next.anchor
  selectedMonth.value = next.month
  emit('change')
}
</script>

<template>
  <PeriodBar
    :label="label"
    :can-previous="canStepDrivesPeriod(state, -1)"
    :can-next="canStepDrivesPeriod(state, 1)"
    @step="step"
  >
    <template #sheet="{ open, setOpen }">
      <ModalShell :open="open" :title="$t('drives.drivesToolbar.period')" :icon="CalendarDays" icon-class="text-slate-300" size="sm" @update:open="setOpen">
        <DrivesPeriodControls
          v-model:period-mode="periodMode"
          v-model:selected-month="selectedMonth"
          v-model:anchor-day="anchorDay"
          v-model:custom-from="customFrom"
          v-model:custom-to="customTo"
          @change="emit('change')"
        />
        <template #footer>
          <button type="button" class="btn btn-lg btn-primary w-full" @click="setOpen(false)">
            {{ $t('drives.drivesToolbar.showResults', { count: total }) }}
          </button>
        </template>
      </ModalShell>
    </template>
  </PeriodBar>
</template>
