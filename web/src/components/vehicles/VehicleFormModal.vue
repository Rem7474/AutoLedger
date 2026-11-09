<script setup lang="ts">
import ModalShell from '@/components/ModalShell.vue'
import NumberInput from '@/components/NumberInput.vue'
import { intlLocale, t } from '@/i18n'
import DistanceInput from '@/components/DistanceInput.vue'
import { computed, ref, watch } from 'vue'
import { api } from '@/services/api'
import { useStaleWrite } from '@/composables/useStaleWrite'
import { useConfirm } from '@/composables/useConfirm'
import { ChevronDown, ChevronRight, Zap, Flame, Sliders } from 'lucide-vue-next'
import { canLinkTeslaMate, emptyVehicleForm, vehicleFormFrom, vehicleSavePayload } from '@/utils/vehicles'
import { CURRENCIES } from '@/utils/expenses'
import { distanceUnit, formatPerDistanceValue } from '@/units'
import { useSubmit } from '@/composables/useSubmit'

// Adds a vehicle, or edits \`editing\`: its identity, tariffs and tolls. The TeslaMate connection is set in Data sources.
const props = withDefaults(defineProps<{ editing?: any | null }>(), { editing: null })
const emit = defineEmits<{ saved: [] }>()
const open = defineModel<boolean>('open', { required: true })
const { showAlert } = useConfirm()
const { explainStaleWrite } = useStaleWrite()

const isEditing = computed(() => !!props.editing)
const editingId = computed(() => props.editing?.id ?? null)
const form = ref(emptyVehicleForm())
const showAdvanced = ref(false)

const tariffPlans = ref<any[]>([])

const powertrainChoices = ['EV', 'PHEV', 'REEV', 'ICE'] as const

function setPowertrain(p: (typeof powertrainChoices)[number]) {
  if (isEditing.value) return
  form.value.powertrain = p
}

async function loadTariffPlans() {
  try {
    const res = await api.getTariffPlans()
    tariffPlans.value = res.plans || []
  } catch (err) {
    console.error('Failed to load tariff plans', err)
  }
}

watch(open, (isOpen) => {
  if (!isOpen) return
  showAdvanced.value = false
  form.value = props.editing ? vehicleFormFrom(props.editing) : emptyVehicleForm()
  void loadTariffPlans()
})

const { pending: submitting, run: runOnce } = useSubmit()

async function handleSaveAction() {
  try {
    const payload = vehicleSavePayload(form.value)

    if (isEditing.value && editingId.value) {
      await api.updateVehicle(editingId.value, { ...payload, expected_version: props.editing?.version })
    } else {
      await api.createVehicle(payload)
    }
    open.value = false
    emit('saved')
  } catch (err: any) {
    if (await explainStaleWrite(err)) {
      open.value = false
      emit('saved')
      return
    }
    void showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger')
  }
}
const handleSave = () => runOnce(handleSaveAction)
</script>

<template>
  <ModalShell
    v-model:open="open"
    :title="isEditing ? $t('vehicles.vehicleFormModal.edit') : $t('shell.topBar.addAVehicle')"
  >
    <form id="vehicle-modal-form" @submit.prevent="handleSave" class="space-y-4">
      <!-- 1. Identité du véhicule : Marque & Modèle (100% Free text according to CLAUDE.md) -->
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label for="vehicle-make" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('onboarding.onboardingView.make') }}</label>
          <input
            id="vehicle-make"
            v-model="form.make"
            :placeholder="$t('onboarding.onboardingView.makePlaceholder')"
            class="field transition-colors"
          />
        </div>
        <div>
          <label for="vehicle-model" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('onboarding.onboardingView.model') }}</label>
          <input
            id="vehicle-model"
            v-model="form.model"
            :placeholder="$t('onboarding.onboardingView.modelPlaceholder')"
            class="field transition-colors"
          />
        </div>
      </div>

      <!-- Display name -->
      <div>
        <label for="vehicle-name" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('vehicles.vehicleFormModal.vehicleName') }}</label>
        <input
          id="vehicle-name"
          v-model="form.name"
          required
          :placeholder="$t('vehicles.vehicleFormModal.eGMyCar')"
          class="field transition-colors"
        />
      </div>

      <!-- Powertrain: tiles rather than a select -->
      <div>
        <span class="block text-xs font-semibold text-slate-300 mb-1.5">{{ $t('vehicles.vehicleFormModal.powertrain') }}</span>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
          <button
            v-for="p in powertrainChoices"
            :key="p"
            type="button"
            :disabled="isEditing"
            @click="setPowertrain(p)"
            class="p-2.5 rounded-xl border flex items-center gap-2.5 transition-all text-left disabled:opacity-50 disabled:cursor-not-allowed"
            :class="form.powertrain === p ? 'bg-rose-500/10 border-rose-500/50 text-white shadow-sm ring-1 ring-rose-500/20' : 'bg-slate-800/40 border-slate-700/60 text-slate-400 hover:bg-slate-800 hover:text-slate-200'"
          >
            <Flame v-if="p === 'ICE'" class="w-4 h-4 text-slate-300 shrink-0" />
            <Zap v-else class="w-4 h-4 text-slate-300 shrink-0" />
            <span class="text-xs font-semibold leading-tight">{{ $t(`vehicles.powertrainOptions.${p}`) }}</span>
          </button>
        </div>
        <p v-if="isEditing" class="mt-1.5 text-xs text-slate-400">{{ $t('vehicles.vehicleFormModal.powertrainFixed') }}</p>
      </div>

      <!-- Currency and mileage -->
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label for="vehicle-current-odometer" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('vehicles.vehicleFormModal.currentMileage', { unit: distanceUnit() }) }}</label>
          <DistanceInput
            id="vehicle-current-odometer"
            v-model="form.current_odometer"
            step="1"
            :disabled="!!form.teslamate_api_url"
            class="field"
          />
          <p class="mt-1 text-xs text-slate-400">{{ $t('vehicles.vehicleFormModal.currentMileageHelp') }}</p>
        </div>
        <div>
          <label for="vehicle-currency" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('vehicles.vehicleFormModal.currency') }}</label>
          <select
            id="vehicle-currency"
            v-model="form.currency"
            :disabled="isEditing"
            class="field"
          >
            <option v-for="c in CURRENCIES" :key="c" :value="c">{{ c }}</option>
          </select>
          <p v-if="isEditing" class="mt-1 text-xs text-slate-400">{{ $t('vehicles.vehicleFormModal.currencyFixed') }}</p>
        </div>
      </div>

      <!-- VIN (optional) -->
      <div>
        <label for="vehicle-vin" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('vehicles.vehicleFormModal.vinOptional') }}</label>
        <input
          id="vehicle-vin"
          v-model="form.vin"
          placeholder="VIN"
          class="field transition-colors"
        />
      </div>

      <!-- 2. The connection to TeslaMate is set once the vehicle exists, in Data sources -->
      <p v-if="!isEditing && canLinkTeslaMate(form.powertrain)" class="pt-3 border-t border-slate-800 text-xs text-slate-400">
        {{ $t('vehicles.vehicleFormModal.teslamateLater') }}
      </p>

      <!-- 3. Additional options, folded so they do not crowd the creation form -->
      <div v-if="canLinkTeslaMate(form.powertrain)" class="pt-3 border-t border-slate-800">
        <button
          type="button"
          class="flex w-full items-center justify-between gap-3 py-2 text-left"
          :aria-expanded="showAdvanced"
          aria-controls="vehicle-advanced-options"
          @click="showAdvanced = !showAdvanced"
        >
          <span class="min-w-0">
            <span class="flex items-center gap-2 text-xs font-semibold text-slate-200">
              <Sliders class="h-4 w-4 text-slate-400" aria-hidden="true" />
              {{ $t('vehicles.vehicleFormModal.advancedOptions') }}
            </span>
            <span class="mt-1 block text-xs text-slate-400">{{ $t('vehicles.vehicleFormModal.advancedOptionsDesc') }}</span>
          </span>
          <ChevronDown v-if="showAdvanced" class="h-4 w-4 shrink-0 text-slate-400" aria-hidden="true" />
          <ChevronRight v-else class="h-4 w-4 shrink-0 text-slate-400" aria-hidden="true" />
        </button>

        <div v-if="showAdvanced" id="vehicle-advanced-options" class="space-y-3.5 pt-2">
          <!-- Tariffs and charger -->
          <div>
            <label for="vehicle-tariff-plan" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('tariffs.planSelectLabel') }}</label>
            <select id="vehicle-tariff-plan" v-model="form.tariff_plan_id" class="field">
              <option :value="null">{{ $t('tariffs.noPlanManual') }}</option>
              <option v-for="p in tariffPlans" :key="p.id" :value="p.id">
                {{ p.name }} ({{ p.plan_type }})
              </option>
            </select>
            <p class="text-xs text-slate-400 mt-1">
              {{ $t('tariffs.planSelectHint') }}
              <router-link to="/account" class="text-slate-100 underline" @click="open = false">{{ $t('tariffs.editor.manageLink') }}</router-link>
            </p>
          </div>

          <label for="vehicle-home-charger-default" class="flex items-start gap-2.5 p-3 rounded-xl bg-slate-800/40 border border-slate-700/60 cursor-pointer">
            <input id="vehicle-home-charger-default" type="checkbox" v-model="form.is_home_charger_default" class="select-box mt-0.5" />
            <div class="text-xs">
              <span class="font-semibold text-white block">{{ $t('vehicles.homeChargerDefaultLabel') }}</span>
              <span class="text-slate-400 block mt-0.5">{{ $t('vehicles.homeChargerDefaultHint') }}</span>
            </div>
          </label>

          <!-- Estimated consumption and rate -->
          <div class="grid grid-cols-2 gap-3 pt-1">
            <div>
              <label for="vehicle-pre-kwh" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('vehicles.vehicleFormModal.consumptionKwh100km', { unit: distanceUnit() }) }}</label>
              <DistanceInput kind="per-distance" id="vehicle-pre-kwh" v-model="form.estimated_kwh_100km" step="0.1" min="1" max="100" :placeholder="$t('common.example', { value: formatPerDistanceValue(16.5, 1) })" class="field" />
            </div>
            <div>
              <label for="vehicle-pre-rate" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('vehicles.vehicleFormModal.rateKwh', { currency: form.currency }) }}</label>
              <NumberInput id="vehicle-pre-rate" v-model="form.estimated_price_per_kwh" min="0" :placeholder="$t('common.example', { value: $n(0.22) })" class="field" />
            </div>
          </div>
        </div>
      </div>

      <!-- 4. Tolls: automatic detection relies on a French dataset -->
      <div class="pt-3 border-t border-slate-800">
        <label for="vehicle-toll-detection" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('vehicles.vehicleFormModal.tollDetection') }}</label>
        <select id="vehicle-toll-detection" v-model="form.toll_detection" class="field">
          <option value="auto">{{ $t('vehicles.vehicleFormModal.tollDetectionAuto') }}</option>
          <option value="on">{{ $t('vehicles.vehicleFormModal.tollDetectionOn') }}</option>
          <option value="off">{{ $t('vehicles.vehicleFormModal.tollDetectionOff') }}</option>
        </select>
        <p class="text-xs text-slate-400 mt-1">{{ $t('vehicles.vehicleFormModal.tollDetectionHint') }}</p>
      </div>
    </form>
    <template #footer>
      <button type="button" @click="open = false" class="btn btn-lg btn-secondary">
        {{ $t('common.cancel') }}
      </button>
      <button type="submit" :disabled="submitting" form="vehicle-modal-form" class="btn btn-lg btn-primary">
        {{ $t('common.save') }}
      </button>
    </template>
  </ModalShell>
</template>