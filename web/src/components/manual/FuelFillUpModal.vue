<script setup lang="ts">
import NumberInput from '@/components/NumberInput.vue'
import DetailDeleteButton from '@/components/DetailDeleteButton.vue'
import DistanceInput from '@/components/DistanceInput.vue'
import AppDatePicker from '@/components/AppDatePicker.vue'
import ModalShell from '@/components/ModalShell.vue'
import { ref, computed, watch } from 'vue'
import { intlLocale, t } from '@/i18n'
import { api } from '@/services/api'
import { useStaleWrite } from '@/composables/useStaleWrite'
import { useVehicleStore } from '@/stores/vehicle'
import { currencySymbol, formatAmount } from '@/currency'
import { distanceUnit, litresToDisplayVolume, perVolume, volumeUnitLabel } from '@/units'
import { toLocalDay } from '@/utils/dates'

// The detail of a fill-up, one sheet for every list that shows one: it records a new fill-up, or edits `editing` (a log
// as the fuel-logs endpoint returns it, with its version). Delete is asked of the host (`delete`), which closes nothing
// itself: the host removes the fill-up with its own Undo.
const props = defineProps<{ vehicleId: string; editing: any | null; canEdit: boolean }>()
const emit = defineEmits<{ saved: []; delete: [log: any] }>()
const open = defineModel<boolean>('open', { required: true })

const vehicleStore = useVehicleStore()
const currency = computed(() => vehicleStore.currency)
const { explainStaleWrite } = useStaleWrite()
const saving = ref(false)
const formError = ref('')

const fuelTypes = [
  { value: '', label: '', labelKey: 'manual.fuelLogsPanel.unspecified' },
  { value: 'SP95_E10', label: '', labelKey: 'comparison.fuelTypes.SP95_E10' },
  { value: 'SP98', label: '', labelKey: 'comparison.fuelTypes.SP98' },
  { value: 'DIESEL', label: '', labelKey: 'manual.fuelLogsPanel.diesel' },
  { value: 'E85', label: '', labelKey: 'comparison.fuelTypes.E85' },
  { value: 'GPL', label: '', labelKey: 'comparison.fuelTypes.GPL' },
]

function emptyForm() {
  return {
    date: toLocalDay(),
    odometer: null as number | null,
    amount: null as number | null,
    liters: null as number | null,
    price_per_liter: null as number | null,
    fuel_type: '',
    is_full_tank: true,
    notes: '',
  }
}

const form = ref(emptyForm())

// Any two of amount / liters / price per liter determine the third (the server derives the missing one)
const derivedHint = computed(() => {
  const { amount, liters, price_per_liter: price } = form.value
  if (amount && liters && !price) return t('manual.fuelLogsPanel.derivedPrice', { unit: volumeUnitLabel(), value: formatAmount(perVolume(amount / liters), currency.value, 3) })
  if (amount && price && !liters) return t('manual.fuelLogsPanel.derivedQuantity', { unit: volumeUnitLabel(), value: litresToDisplayVolume(amount / price).toLocaleString(intlLocale(), { maximumFractionDigits: 2 }) })
  if (liters && price && !amount) return t('manual.fuelLogsPanel.derivedAmount', { value: formatAmount(liters * price, currency.value) })
  return ''
})

function hasOdometer(v: unknown): boolean {
  return v !== null && v !== undefined && String(v) !== ''
}

// The form is seeded each time the sheet opens, from the fill-up being edited or empty
watch(open, (isOpen) => {
  if (!isOpen) return
  formError.value = ''
  const log = props.editing
  form.value = log
    ? {
        date: toLocalDay(log.date),
        odometer: log.odometer ?? null,
        amount: log.amount,
        liters: log.liters ?? null,
        price_per_liter: log.price_per_liter ?? null,
        fuel_type: log.fuel_type || '',
        is_full_tank: log.is_full_tank,
        notes: log.notes || '',
      }
    : emptyForm()
})

function payload() {
  const f = form.value
  return {
    date: f.date,
    odometer: hasOdometer(f.odometer) ? Number(f.odometer) : undefined,
    amount: f.amount ? Number(f.amount) : undefined,
    liters: f.liters ? Number(f.liters) : undefined,
    price_per_liter: f.price_per_liter ? Number(f.price_per_liter) : undefined,
    fuel_type: f.fuel_type || undefined,
    is_full_tank: f.is_full_tank,
    notes: f.notes || undefined,
  }
}

async function save() {
  formError.value = ''
  if (!form.value.date) {
    formError.value = t('manual.fuelLogsPanel.dateRequired')
    return
  }
  saving.value = true
  try {
    if (props.editing) {
      await api.updateFuelLog(props.vehicleId, props.editing.id, { ...payload(), expected_version: props.editing.version })
    } else {
      await api.createFuelLog(props.vehicleId, payload(), currency.value)
    }
    open.value = false
    emit('saved')
  } catch (err: any) {
    // A stale write is explained, the sheet closes and the host reads the list again
    if (await explainStaleWrite(err)) {
      open.value = false
      emit('saved')
      return
    }
    formError.value = err?.message || t('manual.fuelLogsPanel.saveFailed')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <ModalShell pane v-model:open="open" size="sm" :title="editing ? $t('manual.fuelLogsPanel.edit') : $t('manual.fuelLogsPanel.new')">
    <form id="fuel-log-form" class="space-y-3.5" @submit.prevent="save">
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label for="fuel-date" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('common.date') }}</label>
          <AppDatePicker id="fuel-date" v-model="form.date" required size="sm" />
        </div>
        <div>
          <label for="fuel-odometer" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('manual.fuelLogsPanel.mileageKmOptional', { unit: distanceUnit() }) }}</label>
          <DistanceInput id="fuel-odometer" v-model="form.odometer" inputmode="numeric" min="0" step="1" class="field" />
        </div>
      </div>

      <div class="grid grid-cols-3 gap-3">
        <div>
          <label for="fuel-amount" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('manual.fuelLogsPanel.amount', { cur: currencySymbol(currency) }) }}</label>
          <NumberInput id="fuel-amount" v-model="form.amount" min="0.01" class="field" />
        </div>
        <div>
          <label for="fuel-liters" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('manual.fuelLogsPanel.litres', { unit: volumeUnitLabel() }) }}</label>
          <DistanceInput kind="volume" :digits="2" id="fuel-liters" v-model="form.liters" min="0.01" class="field" />
        </div>
        <div>
          <label for="fuel-price" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('manual.fuelLogsPanel.priceL', { cur: currencySymbol(currency), unit: volumeUnitLabel() }) }}</label>
          <DistanceInput kind="per-volume" :digits="3" id="fuel-price" v-model="form.price_per_liter" min="0.001" class="field" />
        </div>
      </div>
      <p class="text-xs text-slate-400 -mt-1.5">
        {{ $t('manual.fuelLogsPanel.enterTheAmountOrThe', { derivedHint, unit: volumeUnitLabel() }) }}
      </p>
      <p v-if="!hasOdometer(form.odometer)" class="text-xs text-slate-400 -mt-2">
        {{ $t('manual.fuelLogsPanel.withoutAMileageItIs') }}
      </p>

      <div class="grid grid-cols-2 gap-3 items-end">
        <div>
          <label for="fuel-type" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('manual.fuelLogsPanel.fuel') }}</label>
          <select id="fuel-type" v-model="form.fuel_type" class="field">
            <option v-for="t in fuelTypes" :key="t.value" :value="t.value">{{ t.labelKey ? $t(t.labelKey) : t.label }}</option>
          </select>
        </div>
        <label for="fuel-full" class="flex items-center gap-2 text-xs text-slate-300 pb-2">
          <input id="fuel-full" v-model="form.is_full_tank" type="checkbox" class="rounded border-slate-600 bg-slate-800" />
          {{ $t('manual.fuelLogsPanel.fullTank') }}
        </label>
      </div>

      <div>
        <label for="fuel-notes" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('manual.fuelLogsPanel.notesOptional') }}</label>
        <input id="fuel-notes" v-model="form.notes" maxlength="200" class="field" />
      </div>

      <p v-if="formError" class="text-xs text-critical-400" role="alert">{{ formError }}</p>
    </form>

    <template #footer>
      <DetailDeleteButton v-if="editing" class="mr-auto" :can-edit="canEdit" @delete="emit('delete', editing)" />
      <button type="button" class="btn btn-lg btn-secondary" @click="open = false">{{ $t('common.cancel') }}</button>
      <button type="submit" form="fuel-log-form" :disabled="saving" class="btn btn-lg btn-primary">
        {{ saving ? $t('manual.fuelLogsPanel.saving') : $t('common.save') }}
      </button>
    </template>
  </ModalShell>
</template>
