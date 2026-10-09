<script setup lang="ts">
import { currentLocale, t } from '@/i18n'
import DistanceInput from '@/components/DistanceInput.vue'
import LanguageSwitcher from '@/components/LanguageSwitcher.vue'
import { CURRENCIES } from '@/utils/expenses'
import { APP_NAME } from '@/brand'
import { distanceUnit } from '@/units'
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useVehicleStore } from '@/stores/vehicle'
import { useQuickAddStore } from '@/stores/quickAdd'
import { api } from '@/services/api'
import { Eye, EyeOff, Zap, ShieldCheck, Car, ArrowRight, CheckCircle2, AlertCircle } from 'lucide-vue-next'
import { buildVehiclePayload, emptyTeslaMateForm, type Powertrain } from '@/utils/onboarding'
import { webhookSnippet } from '@/utils/dataSources'
import OnboardingDataSourcesStep from '@/components/onboarding/OnboardingDataSourcesStep.vue'
import OnboardingDone from '@/components/onboarding/OnboardingDone.vue'

const router = useRouter()
const authStore = useAuthStore()
const vehicleStore = useVehicleStore()
const quickAdd = useQuickAddStore()

const currentStep = ref(1)
const loading = ref(false)
const error = ref('')

// Step 1: Admin Account
const adminEmail = ref('')
const adminPassword = ref('')
const adminConfirmPassword = ref('')
const showAdminPassword = ref(false)
const passwordTouched = ref(false)
const confirmTouched = ref(false)
const passwordHint = computed(() =>
  passwordTouched.value && adminPassword.value.length > 0 && adminPassword.value.length < 8 ? t('onboarding.passwordTooShort') : ''
)
const confirmHint = computed(() =>
  confirmTouched.value && adminConfirmPassword.value !== '' && adminConfirmPassword.value !== adminPassword.value ? t('onboarding.passwordsDiffer') : ''
)

// Step 2: Vehicle Setup
const vehicleMake = ref('')
const vehicleModel = ref('')
const vehicleName = ref('')
const vehicleVin = ref('')
const vehiclePowertrain = ref<Powertrain>('EV')
const vehicleOdometer = ref<number | string | null>(null)
const vehicleNameEdited = ref(false)
const vehicleCurrency = ref(currentLocale() === 'fr' ? 'EUR' : 'USD')

function updateVehicleNameDefault() {
  if (vehicleNameEdited.value && vehicleName.value) return
  vehicleName.value = [vehicleMake.value.trim(), vehicleModel.value.trim()].filter(Boolean).join(' ')
}

// Step 3: data sources (optional, nothing preselected)
const teslamate = reactive(emptyTeslaMateForm())
const useWebhook = ref(false)
const webhook = ref<{ token: string; snippet: string } | null>(null)
const webhookFailed = ref(false)

onMounted(async () => {
  if (authStore.isAuthenticated) {
    await vehicleStore.fetchVehicles()
    if (vehicleStore.vehicles.length > 0) {
      router.push('/')
      return
    }
    // Admin account already exists & logged in, proceed to vehicle setup
    currentStep.value = 2
  }
})

async function handleStep1Submit() {
  error.value = ''
  if (adminPassword.value !== adminConfirmPassword.value) {
    error.value = t('onboarding.passwordsDiffer')
    return
  }
  if (adminPassword.value.length < 8) {
    error.value = t('onboarding.passwordTooShort')
    return
  }

  loading.value = true
  try {
    await authStore.register({ email: adminEmail.value, password: adminPassword.value })
    currentStep.value = 2
  } catch (err: any) {
    error.value = err.message || t('onboarding.adminCreationFailed')
  } finally {
    loading.value = false
  }
}

async function handleStep2Submit() {
  error.value = ''
  if (!vehicleName.value) {
    error.value = t('onboarding.vehicleNameRequired')
    return
  }
  currentStep.value = 3
}

async function handleFinalSubmit() {
  error.value = ''
  loading.value = true
  try {
    const payload = buildVehiclePayload({
      name: vehicleName.value,
      make: vehicleMake.value,
      model: vehicleModel.value,
      vin: vehicleVin.value,
      powertrain: vehiclePowertrain.value,
      odometer: Number(vehicleOdometer.value),
      currency: vehicleCurrency.value,
      teslamate,
    })

    const vehicle = await api.createVehicle(payload)
    if (useWebhook.value) {
      try {
        const { token } = await api.createAPIToken({ name: 'Home Assistant' })
        webhook.value = { token, snippet: webhookSnippet(window.location.origin, token, vehicle.id) }
      } catch {
        webhookFailed.value = true
      }
    }
    await vehicleStore.fetchVehicles()
    currentStep.value = 4
  } catch (err: any) {
    error.value = err.message || t('onboarding.vehicleSaveFailed')
  } finally {
    loading.value = false
  }
}

function finishOnboarding() {
  router.push('/')
}

async function addFirstEntry() {
  await router.push('/')
  quickAdd.open()
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-4 bg-slate-950">
    <div class="w-full max-w-xl bg-slate-900 border border-slate-800 rounded-3xl p-6 sm:p-10 shadow-2xl">
      <div class="flex justify-end mb-2">
        <LanguageSwitcher />
      </div>

      <!-- Header -->
      <div class="text-center mb-8">
        <div class="inline-flex p-3.5 bg-gradient-to-tr from-rose-500 to-warning-500 rounded-2xl shadow-lg shadow-rose-500/25 mb-4">
          <Zap class="w-8 h-8 text-white" />
        </div>
        <h1 class="text-2xl sm:text-3xl font-bold tracking-tight text-white">{{ $t('onboarding.onboardingView.welcomeTo', { APP_NAME }) }}</h1>
        <p class="text-sm text-slate-400 mt-1.5">{{ $t('onboarding.onboardingView.initialSetupWizardForYour') }}</p>
      </div>

      <!-- Step Indicators -->
      <div class="flex items-center justify-between max-w-md mx-auto mb-8 relative">
        <div class="absolute left-0 top-1/2 -translate-y-1/2 h-0.5 w-full bg-slate-800 -z-0"></div>
        <div
          v-for="step in 3"
          :key="step"
          :aria-current="currentStep === step ? 'step' : undefined"
          class="relative z-10 w-9 h-9 rounded-full flex items-center justify-center text-sm font-semibold transition-all"
          :class="[
            currentStep === step
              ? 'bg-rose-500 text-white shadow-lg shadow-rose-500/30 ring-4 ring-rose-500/20'
              : currentStep > step
              ? 'bg-success-500 text-white'
              : 'bg-slate-800 text-slate-400 border border-slate-700'
          ]"
        >
          <CheckCircle2 v-if="currentStep > step" class="w-5 h-5" />
          <span v-else aria-hidden="true">{{ step }}</span>
          <span class="sr-only">{{ $t('onboarding.onboardingView.stepOf', { step, total: 3 }) }}</span>
        </div>
      </div>

      <!-- Error alert -->
      <div v-if="error" class="mb-6 p-3.5 bg-danger-500/10 border border-danger-500/20 rounded-xl flex items-center gap-2.5 text-sm text-danger-400">
        <AlertCircle class="w-4 h-4 shrink-0" />
        <span>{{ error }}</span>
      </div>

      <!-- STEP 1: Admin Account Creation -->
      <div v-if="currentStep === 1">
        <div class="mb-6">
          <h2 class="text-lg font-semibold text-white flex items-center gap-2">
            <ShieldCheck class="w-5 h-5 text-rose-400" />
            {{ $t('onboarding.onboardingView.1CreateTheAdministratorAccount') }}
          </h2>
          <p class="text-xs text-slate-400 mt-1">{{ $t('onboarding.onboardingView.thisIsTheMainAccount', { APP_NAME }) }}</p>
        </div>

        <form @submit.prevent="handleStep1Submit" class="space-y-4">
          <div>
            <label for="onboarding-admin-email" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">{{ $t('onboarding.onboardingView.administratorEmail') }}</label>
            <input id="onboarding-admin-email"
              v-model="adminEmail"
              type="email"
              autocomplete="email"
              required
              :placeholder="$t('onboarding.onboardingView.adminYourDomainCom')"
              class="field placeholder-slate-500"
            />
          </div>

          <div>
            <label for="onboarding-admin-password" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">{{ $t('onboarding.onboardingView.password8CharactersMin') }}</label>
            <div class="relative">
              <input id="onboarding-admin-password"
                v-model="adminPassword"
                :type="showAdminPassword ? 'text' : 'password'"
                autocomplete="new-password"
                required
                placeholder=""
                :aria-invalid="passwordHint ? 'true' : undefined"
                aria-describedby="onboarding-admin-password-hint"
                class="field pr-12 placeholder-slate-500"
                @blur="passwordTouched = true"
              />
              <button
                type="button"
                class="tap absolute inset-y-0 right-0 flex items-center justify-center px-3 text-slate-400 hover:text-slate-200"
                :aria-label="showAdminPassword ? $t('onboarding.hidePassword') : $t('onboarding.showPassword')"
                :aria-pressed="showAdminPassword"
                @click="showAdminPassword = !showAdminPassword"
              >
                <EyeOff v-if="showAdminPassword" class="w-4 h-4" />
                <Eye v-else class="w-4 h-4" />
              </button>
            </div>
            <p id="onboarding-admin-password-hint" class="mt-1.5 min-h-4 text-xs text-danger-400" aria-live="polite">{{ passwordHint }}</p>
          </div>

          <div>
            <label for="onboarding-admin-confirm-password" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">{{ $t('onboarding.onboardingView.confirmThePassword') }}</label>
            <input id="onboarding-admin-confirm-password"
              v-model="adminConfirmPassword"
              :type="showAdminPassword ? 'text' : 'password'"
              autocomplete="new-password"
              required
              placeholder=""
              :aria-invalid="confirmHint ? 'true' : undefined"
              aria-describedby="onboarding-admin-confirm-hint"
              class="field placeholder-slate-500"
              @blur="confirmTouched = true"
            />
            <p id="onboarding-admin-confirm-hint" class="mt-1.5 min-h-4 text-xs text-danger-400" aria-live="polite">{{ confirmHint }}</p>
          </div>

          <button
            type="submit"
            :disabled="loading"
            class="w-full py-3.5 px-4 bg-gradient-to-r from-rose-600 to-rose-500 hover:from-rose-500 hover:to-rose-400 text-white font-semibold rounded-xl shadow-lg shadow-rose-600/25 flex items-center justify-center gap-2 transition-all disabled:opacity-50"
          >
            <span>{{ loading ? $t('onboarding.creating') : $t('onboarding.continueToVehicle') }}</span>
            <ArrowRight class="w-4 h-4" />
          </button>
        </form>
      </div>

      <!-- STEP 2: Vehicle Setup -->
      <div v-else-if="currentStep === 2">
        <div class="mb-6">
          <h2 class="text-lg font-semibold text-white flex items-center gap-2">
            <Car class="w-5 h-5 text-rose-400" />
            {{ $t('onboarding.onboardingView.2YourFirstVehicle') }}
          </h2>
          <p class="text-xs text-slate-400 mt-1">{{ $t('onboarding.onboardingView.setUpYourMainVehicle') }}</p>
        </div>

        <form @submit.prevent="handleStep2Submit" class="space-y-4">
          <!-- Make & Model Inputs -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label for="onboarding-vehicle-make" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">{{ $t('onboarding.onboardingView.make') }}</label>
              <input id="onboarding-vehicle-make"
                v-model="vehicleMake"
                @input="updateVehicleNameDefault"
                type="text"
                :placeholder="$t('onboarding.onboardingView.makePlaceholder')"
                class="field placeholder-slate-500"
              />
            </div>
            <div>
              <label for="onboarding-vehicle-model" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">{{ $t('onboarding.onboardingView.model') }}</label>
              <input id="onboarding-vehicle-model"
                v-model="vehicleModel"
                @input="updateVehicleNameDefault"
                type="text"
                :placeholder="$t('onboarding.onboardingView.modelPlaceholder')"
                class="field placeholder-slate-500"
              />
            </div>
          </div>

          <div>
            <label for="onboarding-vehicle-name" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">{{ $t('onboarding.onboardingView.vehicleName') }}</label>
            <input id="onboarding-vehicle-name"
              v-model="vehicleName"
              @input="vehicleNameEdited = true"
              type="text"
              required
              :placeholder="$t('onboarding.onboardingView.vehicleName')"
              class="field placeholder-slate-500"
            />
          </div>

          <div>
            <label for="onboarding-vehicle-powertrain" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">{{ $t('onboarding.onboardingView.powertrain') }}</label>
            <select id="onboarding-vehicle-powertrain"
              v-model="vehiclePowertrain"
              class="field"
            >
              <option v-for="p in ['EV', 'PHEV', 'REEV', 'ICE']" :key="p" :value="p">{{ $t(`vehicles.powertrainOptions.${p}`) }}</option>
            </select>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
            <div class="sm:col-span-2">
              <label for="onboarding-vehicle-currency" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">{{ $t('onboarding.onboardingView.currency') }}</label>
              <select id="onboarding-vehicle-currency" v-model="vehicleCurrency" class="field">
                <option v-for="c in CURRENCIES" :key="c" :value="c">{{ c }}</option>
              </select>
              <p class="mt-1 text-xs text-slate-400">{{ $t('onboarding.onboardingView.currencyHelp') }}</p>
            </div>
            <div>
              <label for="onboarding-vehicle-odometer" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">{{ $t('onboarding.onboardingView.currentOdometerKm', { unit: distanceUnit() }) }}</label>
              <DistanceInput id="onboarding-vehicle-odometer"
                v-model="vehicleOdometer"
                min="0"
                step="1"
                required
                class="field placeholder-slate-500"
              />
            </div>
            <div>
              <label for="onboarding-vehicle-vin" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">{{ $t('onboarding.onboardingView.vinOptional') }}</label>
              <input id="onboarding-vehicle-vin"
                v-model="vehicleVin"
                type="text"
                placeholder="VIN"
                class="field placeholder-slate-500"
              />
            </div>
          </div>

          <button
            type="submit"
            :disabled="loading"
            class="w-full py-3.5 px-4 bg-gradient-to-r from-rose-600 to-rose-500 hover:from-rose-500 hover:to-rose-400 text-white font-semibold rounded-xl shadow-lg shadow-rose-600/25 flex items-center justify-center gap-2 transition-all disabled:opacity-50"
          >
            <span>{{ $t('onboarding.continueToSources') }}</span>
            <ArrowRight class="w-4 h-4" />
          </button>
        </form>
      </div>

      <!-- STEP 3: Data sources (optional) -->
      <OnboardingDataSourcesStep
        v-else-if="currentStep === 3"
        v-model:use-webhook="useWebhook"
        :powertrain="vehiclePowertrain"
        :teslamate="teslamate"
        :saving="loading"
        @back="currentStep = 2"
        @finalize="handleFinalSubmit"
      />

      <!-- STEP 4: Completed -->
      <OnboardingDone
        v-else-if="currentStep === 4"
        :webhook="webhook"
        :webhook-failed="webhookFailed"
        @finish="finishOnboarding"
        @add-entry="addFirstEntry"
      />
    </div>
  </div>
</template>
