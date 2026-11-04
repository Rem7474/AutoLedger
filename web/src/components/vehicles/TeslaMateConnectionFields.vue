<script setup lang="ts">
import { computed, ref } from 'vue'
import { RefreshCw, CheckCircle2, AlertCircle, Link2, Copy } from 'lucide-vue-next'
import { api } from '@/services/api'
import { t } from '@/i18n'
import { distanceUnit, formatDistanceValue } from '@/units'
import { copiedConnection, type ConnectionForm } from '@/utils/vehicles'

// The fields of a vehicle's teslamateapi connection and a test of what is typed. Saving belongs to the parent.
const props = defineProps<{
  /** Whether the vehicle already has a connection: its saved secrets are not shown and an empty field keeps them. */
  configured: boolean
  /** Other vehicles whose settings can be reused, offered while there is no connection yet. */
  copySources: any[]
}>()
const form = defineModel<ConnectionForm>({ required: true })

const testing = ref(false)
const result = ref<{ success: boolean; status?: any; error?: string } | null>(null)
const showCopy = computed(() => !props.configured && props.copySources.length > 0)

function copyFrom(source: any) {
  form.value = { ...form.value, ...copiedConnection(source) }
}

async function testConnection() {
  if (!form.value.teslamate_api_url) {
    result.value = { success: false, error: t('vehicles.teslamateConnection.urlFirst') }
    return
  }
  testing.value = true
  result.value = null
  try {
    const res = await api.testTeslaMateRaw(form.value)
    result.value = { success: true, status: res.status }
  } catch (err: any) {
    result.value = { success: false, error: err.message }
  } finally {
    testing.value = false
  }
}
</script>

<template>
  <div class="space-y-3">
    <div v-if="showCopy" class="flex flex-wrap items-center gap-2 rounded-xl bg-slate-800 p-2.5 text-xs">
      <span class="text-slate-400">{{ $t('vehicles.teslamateConnection.copyFrom', { name: '' }) }}:</span>
      <button
        v-for="v in copySources"
        :key="v.id"
        type="button"
        class="btn btn-secondary"
        @click="copyFrom(v)"
      >
        <Copy class="h-3 w-3 text-slate-400" aria-hidden="true" />
        {{ v.name || v.model || $t('vehicles.teslamateConnection.copyFallbackName') }}
      </button>
    </div>

    <div>
      <label for="vehicle-teslamate-api-url" class="mb-1 block text-xs font-semibold text-slate-300">{{ $t('vehicles.teslamateConnection.apiUrl') }}</label>
      <input
        id="vehicle-teslamate-api-url"
        v-model="form.teslamate_api_url"
        type="url"
        inputmode="url"
        required
        :placeholder="$t('vehicles.teslamateConnection.apiUrlPlaceholder')"
        class="field"
      />
    </div>

    <div class="grid grid-cols-2 gap-3 max-sm:grid-cols-1">
      <div>
        <label for="vehicle-teslamate-car-id" class="mb-1 block text-xs font-semibold text-slate-300">{{ $t('vehicles.teslamateConnection.carId') }}</label>
        <input
          id="vehicle-teslamate-car-id"
          v-model.number="form.teslamate_car_id"
          type="number"
          inputmode="numeric"
          min="1"
          class="field"
        />
      </div>
      <div>
        <label for="vehicle-teslamate-auth-type" class="mb-1 block text-xs font-semibold text-slate-300">{{ $t('vehicles.teslamateConnection.authMode') }}</label>
        <select id="vehicle-teslamate-auth-type" v-model="form.teslamate_auth_type" class="field">
          <option value="NONE">{{ $t('vehicles.teslamateConnection.authNone') }}</option>
          <option value="BEARER">{{ $t('vehicles.teslamateConnection.authBearer') }}</option>
          <option value="BASIC">{{ $t('vehicles.teslamateConnection.authBasic') }}</option>
        </select>
      </div>
    </div>

    <div v-if="form.teslamate_auth_type === 'BEARER'">
      <label for="vehicle-teslamate-api-key" class="mb-1 block text-xs font-semibold text-slate-300">{{ $t('vehicles.teslamateConnection.apiKey') }}</label>
      <input
        id="vehicle-teslamate-api-key"
        v-model="form.teslamate_api_key"
        type="password"
        autocomplete="off"
        placeholder="••••••••"
        class="field"
      />
      <p v-if="configured" class="mt-1 text-xs text-slate-400">{{ $t('vehicles.teslamateConnection.secretKept') }}</p>
    </div>

    <div v-if="form.teslamate_auth_type === 'BASIC'" class="grid grid-cols-2 gap-3 max-sm:grid-cols-1">
      <div>
        <label for="vehicle-teslamate-basic-user" class="mb-1 block text-xs font-semibold text-slate-300">{{ $t('vehicles.teslamateConnection.basicUser') }}</label>
        <input
          id="vehicle-teslamate-basic-user"
          v-model="form.teslamate_basic_user"
          autocomplete="off"
          autocapitalize="none"
          placeholder="admin"
          class="field"
        />
      </div>
      <div>
        <label for="vehicle-teslamate-basic-pass" class="mb-1 block text-xs font-semibold text-slate-300">{{ $t('vehicles.teslamateConnection.basicPassword') }}</label>
        <input
          id="vehicle-teslamate-basic-pass"
          v-model="form.teslamate_basic_pass"
          type="password"
          autocomplete="off"
          placeholder="••••••••"
          class="field"
        />
      </div>
      <p v-if="configured" class="-mt-1 text-xs text-slate-400 sm:col-span-2">{{ $t('vehicles.teslamateConnection.secretKept') }}</p>
    </div>

    <div>
      <label for="vehicle-teslamate-grafana-url" class="mb-1 block text-xs font-semibold text-slate-300">{{ $t('vehicles.teslamateConnection.grafanaUrl') }}</label>
      <input
        id="vehicle-teslamate-grafana-url"
        v-model="form.teslamate_grafana_url"
        type="url"
        inputmode="url"
        :placeholder="$t('vehicles.teslamateConnection.urlPlaceholder')"
        class="field"
      />
      <p class="mt-1 text-xs text-slate-400">{{ $t('vehicles.teslamateConnection.grafanaHint') }}</p>
    </div>

    <div>
      <label for="vehicle-teslamate-url" class="mb-1 block text-xs font-semibold text-slate-300">{{ $t('vehicles.teslamateConnection.url') }}</label>
      <input
        id="vehicle-teslamate-url"
        v-model="form.teslamate_url"
        type="url"
        inputmode="url"
        :placeholder="$t('vehicles.teslamateConnection.urlPlaceholder')"
        class="field"
      />
      <p class="mt-1 text-xs text-slate-400">{{ $t('vehicles.teslamateConnection.urlHint') }}</p>
    </div>

    <button type="button" :disabled="testing" class="btn btn-secondary w-full" @click="testConnection">
      <RefreshCw v-if="testing" class="h-3.5 w-3.5 animate-spin text-slate-400" />
      <Link2 v-else class="h-3.5 w-3.5 text-slate-400" />
      <span>{{ testing ? $t('vehicles.teslamateConnection.testing') : $t('vehicles.teslamateConnection.test') }}</span>
    </button>

    <div
      v-if="result"
      class="flex items-start gap-2 rounded-xl border p-3 text-xs"
      :class="result.success ? 'border-success-500/20 bg-success-500/10 text-success-300' : 'border-danger-500/20 bg-danger-500/10 text-danger-300'"
      role="status"
    >
      <CheckCircle2 v-if="result.success" class="mt-0.5 h-4 w-4 shrink-0 text-success-400" />
      <AlertCircle v-else class="mt-0.5 h-4 w-4 shrink-0 text-danger-400" />
      <div class="flex-1">
        <template v-if="result.success">
          <strong class="font-semibold">{{ $t('vehicles.teslamateConnection.testSuccess') }}</strong>
          <p class="mt-0.5 text-success-200/80">
            {{ $t('vehicles.teslamateConnection.testStatus', { unit: distanceUnit(), state: result.status?.state || $t('vehicles.vehicleCard.online'), odometer: formatDistanceValue(result.status?.odometer || 0) }) }}
          </p>
        </template>
        <template v-else>
          <strong class="font-semibold">{{ $t('vehicles.teslamateConnection.testFailed') }}</strong>
          <p class="mt-0.5 text-danger-200/90">{{ result.error }}</p>
        </template>
      </div>
    </div>
  </div>
</template>
