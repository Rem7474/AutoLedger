<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { VueDatePicker } from '@vuepic/vue-datepicker'
import { enGB, fr } from 'date-fns/locale'
import { currentLocale, t } from '@/i18n'
import { useIsMobile } from '@/composables/useIsMobile'
import { supportsNativeInput, toRange } from '@/utils/nativeDate'

// Two days picked on one calendar: v-model is [from, to] as YYYY-MM-DD, or null when no range is chosen.
const props = withDefaults(
  defineProps<{
    modelValue?: [string, string] | null
    placeholder?: string
    disabled?: boolean
    id?: string
    /** Latest selectable day (YYYY-MM-DD); defaults to no limit. */
    max?: string
  }>(),
  { modelValue: null, placeholder: '', disabled: false, id: undefined, max: undefined },
)
const emit = defineEmits<{ 'update:modelValue': [value: [string, string] | null] }>()

const dateLocale = computed(() => (currentLocale() === 'fr' ? fr : enGB))

const internal = computed({
  get: () => (props.modelValue ? [...props.modelValue] : null),
  set: (val: unknown) => {
    if (Array.isArray(val) && val.length === 2 && val[0] && val[1]) {
      emit('update:modelValue', [String(val[0]), String(val[1])])
      return
    }
    emit('update:modelValue', null)
  },
})

// On a phone each end of the range is a native date field. A day picked before the other one is waited for, then the
// pair is sent in chronological order; the range is left once both fields are emptied.
const isMobile = useIsMobile()
const native = computed(() => isMobile.value && supportsNativeInput('date'))
const from = ref(props.modelValue?.[0] ?? '')
const to = ref(props.modelValue?.[1] ?? '')
watch(
  () => props.modelValue,
  (range) => {
    from.value = range?.[0] ?? ''
    to.value = range?.[1] ?? ''
  },
)
function pick(end: 'from' | 'to', e: Event) {
  const value = (e.target as HTMLInputElement).value
  if (end === 'from') from.value = value
  else to.value = value
  const range = toRange(from.value, to.value)
  if (range) {
    from.value = range[0]
    to.value = range[1]
    emit('update:modelValue', range)
  } else if (!from.value && !to.value) emit('update:modelValue', null)
}

const maxDate = computed(() => (props.max ? new Date(`${props.max}T00:00:00`) : undefined))
const actionRow = computed(() => ({
  selectBtnLabel: t('shell.datePicker.confirm'),
  cancelBtnLabel: t('common.cancel'),
}))
</script>

<template>
  <div v-if="native" class="flex w-full items-center gap-2">
    <input :id="id" type="date" :value="from" :max="max" :disabled="disabled" :aria-label="$t('shell.datePicker.rangeFrom')" class="field app-native-date" @change="pick('from', $event)" />
    <input type="date" :value="to" :max="max" :disabled="disabled" :aria-label="$t('shell.datePicker.rangeTo')" class="field app-native-date" @change="pick('to', $event)" />
  </div>
  <div v-else :class="['app-daterange-wrapper', { 'is-disabled': disabled }]">
    <VueDatePicker
      :uid="id"
      v-model="internal"
      range
      :dark="true"
      :locale="dateLocale"
      model-type="yyyy-MM-dd"
      :time-config="{ enableTimePicker: false }"
      :max-date="maxDate"
      :formats="{ input: 'dd/MM/yyyy', preview: 'dd/MM/yyyy' }"
      :placeholder="placeholder || $t('shell.datePicker.rangePlaceholder')"
      :disabled="disabled"
      :input-attrs="{ clearable: true }"
      teleport="body"
      :action-row="actionRow"
    />
  </div>
</template>

<style scoped>
.app-native-date {
  appearance: none;
  min-height: 2.75rem;
  text-align: left;
}

.app-native-date::-webkit-date-and-time-value {
  text-align: left;
}

.app-native-date::-webkit-calendar-picker-indicator {
  opacity: 0.6;
}

.app-daterange-wrapper {
  min-width: 13rem;
}

.app-daterange-wrapper :deep(.dp--input) {
  font-size: 0.75rem !important;
  line-height: 1rem !important;
  padding-top: 0.375rem !important;
  padding-bottom: 0.375rem !important;
  padding-left: 1.85rem !important;
  padding-right: 0.375rem !important;
}

.app-daterange-wrapper :deep(.dp--input-icon) {
  padding-left: 0.5rem !important;
}

.is-disabled {
  pointer-events: none;
  opacity: 0.6;
}
</style>
