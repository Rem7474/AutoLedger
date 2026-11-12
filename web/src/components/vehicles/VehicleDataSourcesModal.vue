<script setup lang="ts">
import { intlLocale } from '@/i18n'
import { t } from '@/i18n'
import { capabilitiesOf } from '@/utils/capabilities'
import { connectedVehiclesExcept, connectionFormFrom, connectionPayload, type ConnectionForm } from '@/utils/vehicles'
import { computed, ref, watch } from 'vue'
import { api } from '@/services/api'
import { Database, RefreshCw, Link2, CheckCircle2, AlertCircle, Pencil } from '@/icons'
import ModalShell from '@/components/ModalShell.vue'
import TeslaMateConnectionFields from '@/components/vehicles/TeslaMateConnectionFields.vue'
import { useConfirm } from '@/composables/useConfirm'
import { useStaleWrite } from '@/composables/useStaleWrite'
import { useSubmit } from '@/composables/useSubmit'
import { useVehicleStore } from '@/stores/vehicle'

// Where a vehicle's data comes from: the TeslaMate connection (shown and edited here) and what each origin has written so far.
const props = defineProps<{ vehicle: any | null }>()
const emit = defineEmits<{ saved: [] }>()
const open = defineModel<boolean>('open', { required: true })
const { showConfirm, showAlert } = useConfirm()
const { explainStaleWrite } = useStaleWrite()
const vehicleStore = useVehicleStore()

const ORIGINS = ['TESLAMATE', 'WEBHOOK', 'CSV', 'MANUAL'] as const

const loading = ref(false)
const error = ref('')
const summary = ref<{ teslamate_configured: boolean; activity: any[] } | null>(null)
const test = ref<{ loading: boolean; success?: boolean; error?: string } | null>(null)
// The connection form replaces the summary while it is open
const editing = ref(false)
const connection = ref<ConnectionForm | null>(null)

const isOwner = computed(() => props.vehicle?.role === 'OWNER')
const rows = computed(() =>
  ORIGINS.map((origin) => ({ origin, ...(summary.value?.activity.find((a) => a.origin === origin) ?? {}) }))
    .filter((r: any) => r.origin !== 'TESLAMATE' || summary.value?.teslamate_configured || r.drives || r.charges)
)

async function loadSummary() {
  if (!props.vehicle) return
  summary.value = null
  test.value = null
  error.value = ''
  loading.value = true
  try {
    summary.value = await api.getDataSources(props.vehicle.id)
  } catch (err: any) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

watch(open, (isOpen) => {
  if (!isOpen) return
  editing.value = false
  void loadSummary()
})

const copySources = computed(() => connectedVehiclesExcept(vehicleStore.vehicles, props.vehicle?.id))

function editConnection() {
  if (!props.vehicle) return
  connection.value = connectionFormFrom(props.vehicle, vehicleStore.vehicles)
  editing.value = true
}

// The save replaces the whole vehicle: `connectionPayload` carries everything else as it is
async function saveConnection(next: ConnectionForm | null) {
  if (!props.vehicle) return
  try {
    await api.updateVehicle(props.vehicle.id, connectionPayload(props.vehicle, next))
  } catch (err: any) {
    // A vehicle changed elsewhere is explained and reloaded; any other refusal stays on the form
    if (!(await explainStaleWrite(err))) {
      void showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger')
      return
    }
  }
  editing.value = false
  emit('saved')
  await loadSummary()
}

const { pending: saving, run: runOnce } = useSubmit()
const handleSave = () => runOnce(() => saveConnection(connection.value))

async function disconnect() {
  const ok = await showConfirm({
    title: t('vehicles.teslamateConnection.disconnectTitle'),
    message: t('vehicles.teslamateConnection.disconnectMessage'),
    confirmText: t('vehicles.teslamateConnection.disconnect'),
    type: 'danger',
  })
  if (ok) await runOnce(() => saveConnection(null))
}

async function testConnection() {
  if (!props.vehicle) return
  test.value = { loading: true }
  try {
    await api.testTeslaMate(props.vehicle.id)
    test.value = { loading: false, success: true }
  } catch (err: any) {
    test.value = { loading: false, success: false, error: err.message }
  }
}

function formatLast(iso?: string) {
  return iso ? new Date(iso).toLocaleDateString(intlLocale()) : t('vehicles.vehicleDataSourcesModal.never')
}
</script>

<template>
  <ModalShell v-model:open="open" size="md" body-class="space-y-4 text-xs">
    <template #title>
        <div class="flex min-w-0 items-center gap-3">
          <div class="p-2 bg-sky-500/10 text-sky-400 rounded-xl"><Database class="w-5 h-5" /></div>
          <div class="min-w-0">
            <h3 class="text-base font-bold text-white">{{ $t('vehicles.vehicleDataSourcesModal.title') }}</h3>
            <p class="text-xs text-slate-400 truncate">{{ vehicle?.name }}</p>
          </div>
        </div>
    </template>

      <form v-if="editing && connection" id="teslamate-connection-form" class="space-y-4" @submit.prevent="handleSave">
        <h4 class="text-sm font-bold text-white">{{ $t('vehicles.vehicleDataSourcesModal.teslamate') }}</h4>
        <TeslaMateConnectionFields v-model="connection" :configured="!!vehicle?.teslamate_api_url" :copy-sources="copySources" />
        <button
          v-if="vehicle?.teslamate_api_url"
          type="button"
          :disabled="saving"
          class="btn text-danger-400 hover:bg-danger-500/10"
          @click="disconnect"
        >
          {{ $t('vehicles.teslamateConnection.disconnect') }}
        </button>
      </form>
      <div v-else-if="loading" class="flex justify-center py-6"><RefreshCw class="w-5 h-5 animate-spin text-slate-400" /></div>
      <p v-else-if="error" class="text-danger-400">{{ error }}</p>
      <template v-else-if="summary">
        <div v-if="capabilitiesOf(vehicle).charges" class="bg-slate-800 rounded-xl p-4 space-y-3">
          <div class="flex items-center justify-between gap-2">
            <h4 class="font-bold text-white">{{ $t('vehicles.vehicleDataSourcesModal.teslamate') }}</h4>
            <span class="font-semibold" :class="summary.teslamate_configured ? 'text-success-400' : 'text-slate-400'">
              {{ !isOwner ? $t('vehicles.vehicleCard.managedByAdmin') : summary.teslamate_configured ? $t('vehicles.vehicleCard.configured') : $t('vehicles.vehicleCard.notConfigured') }}
            </span>
          </div>
          <p class="text-slate-400">{{ $t('vehicles.vehicleDataSourcesModal.teslamateHint') }}</p>
          <div v-if="isOwner" class="flex flex-wrap gap-2">
            <button
              v-if="summary.teslamate_configured"
              @click="testConnection"
              :disabled="test?.loading"
              class="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 font-semibold rounded-lg flex items-center gap-1.5 border border-slate-700"
            >
              <RefreshCw v-if="test?.loading" class="w-3.5 h-3.5 animate-spin" />
              <Link2 v-else class="w-3.5 h-3.5" />
              {{ $t('vehicles.vehicleCard.testApi') }}
            </button>
            <button
              @click="editConnection"
              class="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 font-semibold rounded-lg flex items-center gap-1.5 border border-slate-700"
            >
              <Pencil class="w-3.5 h-3.5" />
              {{ summary.teslamate_configured ? $t('vehicles.vehicleDataSourcesModal.editConnection') : $t('vehicles.vehicleDataSourcesModal.connect') }}
            </button>
          </div>
          <p v-if="test && !test.loading" class="flex items-start gap-2" :class="test.success ? 'text-success-300' : 'text-rose-300'">
            <CheckCircle2 v-if="test.success" class="w-4 h-4 shrink-0" />
            <AlertCircle v-else class="w-4 h-4 shrink-0" />
            <span>{{ test.success ? $t('vehicles.vehicleCard.online') : test.error }}</span>
          </p>
        </div>

        <div>
          <h4 class="font-bold text-white mb-2">{{ $t('vehicles.vehicleDataSourcesModal.activity') }}</h4>
          <ul class="space-y-2">
            <li v-for="r in rows" :key="r.origin" class="bg-slate-800 rounded-xl p-3">
              <div class="flex items-center justify-between gap-2">
                <span class="font-semibold text-slate-200">{{ $t('vehicles.vehicleDataSourcesModal.origin.' + r.origin) }}</span>
                <span class="text-slate-400">{{ $t('vehicles.vehicleDataSourcesModal.lastData', { date: formatLast((r as any).last_at) }) }}</span>
              </div>
              <p class="mt-1 text-slate-400">
                {{ $t('vehicles.vehicleDataSourcesModal.counts', { drives: (r as any).drives || 0, charges: (r as any).charges || 0, readings: (r as any).odometer_readings || 0 }) }}
              </p>
            </li>
          </ul>
          <p v-if="!summary.activity.length" class="text-slate-400 mt-2">{{ $t('vehicles.vehicleDataSourcesModal.empty') }}</p>
        </div>
      </template>

    <template v-if="editing" #footer>
      <button type="button" class="btn btn-lg btn-secondary" @click="editing = false">{{ $t('common.cancel') }}</button>
      <button type="submit" form="teslamate-connection-form" :disabled="saving" class="btn btn-lg btn-primary">
        {{ $t('vehicles.teslamateConnection.save') }}
      </button>
    </template>
  </ModalShell>
</template>
