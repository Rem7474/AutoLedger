<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import ModalShell from '@/components/ModalShell.vue'
import AppDateRangePicker from '@/components/AppDateRangePicker.vue'
import { useIsMobile } from '@/composables/useIsMobile'
import { todayIso } from '@/utils/drives'
import { PERIOD_PRESETS, presetOf, presetPeriod, type Period } from '@/utils/period'

// The ready-made periods as chips, then a free range with its two labelled ends. The free range only shows once
// "Custom" is chosen; a preset applies at once.
const props = defineProps<{ period: Period }>()
const emit = defineEmits<{ pick: [period: Period] }>()
const open = defineModel<boolean>('open', { required: true })

const isMobile = useIsMobile()
const custom = ref(false)
const range = ref<[string, string] | null>(null)
const today = todayIso()

watch(open, (isOpen) => {
  if (!isOpen) return
  custom.value = props.period.kind === 'custom'
  range.value = props.period.kind === 'custom' ? [props.period.from, props.period.to] : null
})

const current = computed(() => (custom.value ? null : presetOf(props.period)))

// The sheet closes first, then the period changes: the URL is rewritten once the sheet has let go of its history entry.
async function choose(period: Period) {
  open.value = false
  await nextTick()
  emit('pick', period)
}
const pickPreset = (key: (typeof PERIOD_PRESETS)[number]) => choose(presetPeriod(key))
function apply() {
  if (range.value) void choose({ kind: 'custom', from: range.value[0], to: range.value[1] })
}
</script>

<template>
  <ModalShell v-model:open="open" size="sm" :title="$t('period.sheet.title')">
    <div class="space-y-4">
      <div class="flex flex-wrap gap-2" role="group" :aria-label="$t('period.sheet.presets')">
        <button
          v-for="key in PERIOD_PRESETS"
          :key="key"
          type="button"
          :aria-pressed="current === key"
          :class="['tap-text rounded-full border px-3 text-sm font-semibold transition-colors', current === key ? 'border-rose-600 bg-rose-600 text-white' : 'border-slate-700 bg-slate-800 text-slate-200 hover:text-white']"
          @click="pickPreset(key)"
        >
          {{ $t(`period.sheet.preset.${key}`) }}
        </button>
        <button
          type="button"
          :aria-pressed="custom"
          :class="['tap-text rounded-full border px-3 text-sm font-semibold transition-colors', custom ? 'border-rose-600 bg-rose-600 text-white' : 'border-slate-700 bg-slate-800 text-slate-200 hover:text-white']"
          @click="custom = true"
        >
          {{ $t('period.sheet.custom') }}
        </button>
      </div>

      <div v-if="custom" class="space-y-3">
        <div v-if="isMobile" class="grid grid-cols-2 gap-2 px-1 text-xs font-semibold text-slate-300">
          <span>{{ $t('period.sheet.from') }}</span>
          <span>{{ $t('period.sheet.to') }}</span>
        </div>
        <AppDateRangePicker v-model="range" :max="today" />
        <button type="button" class="btn btn-lg btn-primary w-full" :disabled="!range" @click="apply">{{ $t('period.sheet.apply') }}</button>
      </div>
    </div>
  </ModalShell>
</template>
