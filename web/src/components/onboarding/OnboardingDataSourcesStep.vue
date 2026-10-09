<script setup lang="ts">
import { ref } from 'vue'
import { currentLocale, t } from '@/i18n'
import { api } from '@/services/api'
import { distanceUnit, formatDistanceValue } from '@/units'
import { downloadCsv } from '@/utils/csv'
import { csvTemplate, csvTemplateFilename, csvTemplateTypes, type CsvTemplateType } from '@/utils/dataSources'
import { supportsTeslaMate, teslaMateCarId, teslaMateCredentials, type emptyTeslaMateForm, type Powertrain } from '@/utils/onboarding'
import { AlertCircle, ArrowRight, CheckCircle2, Download, FileSpreadsheet, KeyRound, Link2, RefreshCw, Webhook } from 'lucide-vue-next'

// Optional last step of the wizard: the TeslaMate link, a Home Assistant webhook and CSV templates.
// The form is edited in place; the wizard reads it when it creates the vehicle.
const props = defineProps<{
  powertrain: Powertrain
  teslamate: ReturnType<typeof emptyTeslaMateForm>
  saving: boolean
}>()
const emit = defineEmits<{ back: []; finalize: [] }>()
const useWebhook = defineModel<boolean>('useWebhook', { required: true })

const testing = ref(false)
const testResult = ref<{ ok: boolean; message: string } | null>(null)

function downloadTemplate(type: CsvTemplateType) {
  const { headers, rows } = csvTemplate(type, currentLocale())
  downloadCsv(csvTemplateFilename(type), headers, rows)
}

async function testConnection() {
  const teslamate = props.teslamate
  testResult.value = null
  testing.value = true
  try {
    if (!teslamate.url) throw new Error(t('onboarding.apiUrlRequired'))
    const payload = {
      teslamate_api_url: teslamate.url,
      teslamate_auth_type: teslamate.authType,
      teslamate_car_id: teslaMateCarId(teslamate),
      ...teslaMateCredentials(teslamate),
    }
    const res = await api.testTeslaMateRaw(payload)
    const st = res.status
    testResult.value = {
      ok: true,
      message: t('onboarding.testSuccess', { unit: distanceUnit(), state: st?.state || t('onboarding.online'), odometer: formatDistanceValue(st?.odometer || 0) }),
    }
  } catch (err: any) {
    testResult.value = { ok: false, message: err.message }
  } finally {
    testing.value = false
  }
}
</script>

<template>
  <div>
    <div class="mb-6">
      <h2 class="text-lg font-semibold text-white flex items-center gap-2">
        <KeyRound class="w-5 h-5 text-rose-400" />
        {{ $t('onboarding.onboardingView.3TeslamateSynchronizationOptional') }}
      </h2>
      <p class="text-xs text-slate-400 mt-1">{{ $t('onboarding.onboardingView.youCanConnectYourTeslamateapi') }}</p>
    </div>

    <div class="space-y-4">
      <label v-if="supportsTeslaMate(powertrain)" class="flex items-center gap-3 p-4 bg-slate-800/60 border border-slate-700 rounded-xl cursor-pointer hover:bg-slate-800 transition-colors">
        <input v-model="teslamate.enabled" type="checkbox" class="w-5 h-5 shrink-0 rounded text-rose-500 focus:ring-rose-500/20 bg-slate-900 border-slate-700" />
        <Link2 class="w-5 h-5 shrink-0 text-rose-400" />
        <div class="min-w-0">
          <span class="text-sm font-medium text-white block">{{ $t('onboarding.onboardingView.enableTheTeslamateapiLink') }}</span>
          <span class="text-xs text-slate-400 block">{{ $t('onboarding.onboardingView.automaticallySyncsDrivesChargesAnd') }}</span>
        </div>
      </label>

      <div v-if="supportsTeslaMate(powertrain) && teslamate.enabled" class="p-4 bg-slate-800/40 border border-slate-800 rounded-2xl space-y-4">
        <div>
          <label for="onboarding-teslamate-url" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">{{ $t('onboarding.onboardingView.teslamateApiUrl') }}</label>
          <input id="onboarding-teslamate-url"
            v-model="teslamate.url"
            type="url"
            :placeholder="$t('onboarding.onboardingView.http192168150')"
            class="field placeholder-slate-500"
          />
        </div>

        <div>
          <label for="onboarding-teslamate-auth-type" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">{{ $t('onboarding.onboardingView.authenticationMode') }}</label>
          <select id="onboarding-teslamate-auth-type"
            v-model="teslamate.authType"
            class="field"
          >
            <option value="NONE">{{ $t('onboarding.onboardingView.noAuthentication') }}</option>
            <option value="BEARER">{{ $t('onboarding.onboardingView.apiKeyBearerToken') }}</option>
            <option value="BASIC">{{ $t('onboarding.onboardingView.httpBasicAuthUserPassword') }}</option>
          </select>
        </div>

        <div>
          <label for="onboarding-teslamate-car-id" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">{{ $t('vehicles.vehicleFormModal.carIdInTeslamate') }}</label>
          <input id="onboarding-teslamate-car-id"
            v-model.number="teslamate.carId"
            type="number"
            min="1"
            inputmode="numeric"
            class="w-full bg-slate-800 border border-slate-700 rounded-xl px-4 py-2.5 text-sm text-white focus:outline-none focus:border-rose-500 transition-colors"
          />
        </div>

        <div v-if="teslamate.authType === 'BEARER'">
          <label for="onboarding-teslamate-api-key" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">{{ $t('onboarding.onboardingView.apiToken') }}</label>
          <input id="onboarding-teslamate-api-key"
            v-model="teslamate.apiKey"
            type="password"
            :placeholder="$t('onboarding.onboardingView.yourSecretToken')"
            class="field placeholder-slate-500"
          />
        </div>

        <div v-if="teslamate.authType === 'BASIC'" class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <label for="onboarding-teslamate-user" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">{{ $t('onboarding.onboardingView.user') }}</label>
            <input id="onboarding-teslamate-user"
              v-model="teslamate.user"
              type="text"
              placeholder="admin"
              class="field placeholder-slate-500"
            />
          </div>
          <div>
            <label for="onboarding-teslamate-pass" class="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">{{ $t('onboarding.onboardingView.password') }}</label>
            <input id="onboarding-teslamate-pass"
              v-model="teslamate.pass"
              type="password"
              placeholder="••••••••"
              class="field placeholder-slate-500"
            />
          </div>
        </div>

        <!-- Test Connection Button -->
        <div class="pt-2">
          <button
            type="button"
            @click="testConnection"
            :disabled="testing || !teslamate.url"
            class="btn btn-lg btn-secondary w-full"
          >
            <RefreshCw v-if="testing" class="w-3.5 h-3.5 animate-spin text-rose-400" />
            <Link2 v-else class="w-3.5 h-3.5 text-rose-400" />
            <span>{{ testing ? $t('onboarding.testing') : $t('onboarding.testConnection') }}</span>
          </button>

          <div
            v-if="testResult"
            class="mt-2.5 p-3 rounded-xl text-xs flex items-start gap-2.5"
            :class="testResult.ok ? 'bg-success-500/10 text-success-300 border border-success-500/20' : 'bg-rose-500/10 text-rose-300 border border-rose-500/20'"
          >
            <CheckCircle2 v-if="testResult.ok" class="w-4 h-4 shrink-0 text-success-400 mt-0.5" />
            <AlertCircle v-else class="w-4 h-4 shrink-0 text-danger-400 mt-0.5" />
            <span>{{ testResult.message }}</span>
          </div>
        </div>
      </div>

      <label class="flex items-center gap-3 p-4 bg-slate-800/60 border border-slate-700 rounded-xl cursor-pointer hover:bg-slate-800 transition-colors">
        <input v-model="useWebhook" type="checkbox" class="w-5 h-5 shrink-0 rounded text-rose-500 focus:ring-rose-500/20 bg-slate-900 border-slate-700" />
        <Webhook class="w-5 h-5 shrink-0 text-rose-400" />
        <div class="min-w-0">
          <span class="text-sm font-medium text-white block">{{ $t('onboarding.webhookTitle') }}</span>
          <span class="text-xs text-slate-400 block">{{ $t('onboarding.webhookDescription') }}</span>
        </div>
      </label>

      <div class="p-4 bg-slate-800/60 border border-slate-700 rounded-xl">
        <div class="flex items-center gap-3">
          <span class="w-5 shrink-0" aria-hidden="true"></span>
          <FileSpreadsheet class="w-5 h-5 shrink-0 text-rose-400" />
          <div class="min-w-0">
            <span class="text-sm font-medium text-white block">{{ $t('onboarding.csvTitle') }}</span>
            <span class="text-xs text-slate-400 block">{{ $t('onboarding.csvDescription') }}</span>
          </div>
        </div>
        <div class="flex flex-wrap gap-2 mt-3">
          <button
            v-for="type in csvTemplateTypes(powertrain)"
            :key="type"
            type="button"
            @click="downloadTemplate(type)"
            class="btn btn-secondary"
          >
            <Download class="w-3.5 h-3.5 text-rose-400" />
            {{ $t(`onboarding.csvTemplate.${type}`) }}
          </button>
        </div>
      </div>

      <p class="text-xs text-slate-400">{{ $t('onboarding.dataSourcesLater') }}</p>

      <div class="flex gap-3 pt-2">
        <button
          type="button"
          @click="emit('back')"
          class="btn btn-lg btn-secondary"
        >
          {{ $t('onboarding.onboardingView.back') }}
        </button>
        <button
          type="button"
          :disabled="saving"
          @click="emit('finalize')"
          class="flex-1 py-3 px-4 bg-gradient-to-r from-rose-600 to-rose-500 hover:from-rose-500 hover:to-rose-400 text-white font-semibold rounded-xl shadow-lg shadow-rose-600/25 flex items-center justify-center gap-2 transition-all disabled:opacity-50"
        >
          <span>{{ saving ? $t('onboarding.saving') : $t('onboarding.finalize') }}</span>
          <ArrowRight class="w-4 h-4" />
        </button>
      </div>
    </div>
  </div>
</template>
