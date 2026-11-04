<script setup lang="ts">
import { intlLocale } from '@/i18n'
import { ref } from 'vue'
import { distanceUnit, formatDistance, formatDistanceValue, formatPerDistanceValue, perDistance } from '@/units'
import { formatAmount } from '@/currency'
import { useVehicleStore } from '@/stores/vehicle'
import RowMenu from '@/components/RowMenu.vue'
import CardOpenButton from '@/components/CardOpenButton.vue'
import { api } from '@/services/api'
import { Car, Plus, Trash2, Edit2, CheckCircle2, AlertCircle, X, Gauge, Link2, FileText, Zap, Users, Pencil, Database, Check } from 'lucide-vue-next'
import { canCharge, hasEnergyEstimate, isElectricOnly, ownershipSummary, roleLabel } from '@/utils/vehicles'

// One vehicle of the list, with its contract summary and a TeslaMate connection test of its own
const props = defineProps<{ v: any; ownership: any | null }>()
const emit = defineEmits<{
  edit: [vehicle: any]
  delete: [vehicleId: string]
  members: [vehicle: any]
  ownership: [vehicle: any]
  dataSources: [vehicle: any]
}>()
const vehicleStore = useVehicleStore()

const cardTest = ref<{ loading: boolean; success?: boolean; status?: any; error?: string } | undefined>(undefined)

async function testCardConnection() {
  cardTest.value = { loading: true }
  try {
    const res = await api.testTeslaMate(props.v.id)
    cardTest.value = { loading: false, success: true, status: res.status }
  } catch (err: any) {
    cardTest.value = { loading: false, success: false, error: err.message }
  }
}

function clearCardTestResult() {
  cardTest.value = undefined
}
</script>

<template>
  <!-- The whole card selects the vehicle; setting it up is behind the menu, so the card does not repeat a row of buttons -->
  <div
    class="card relative flex h-full flex-col p-4 transition-colors sm:p-5"
    :class="{ 'border-rose-500/40 shadow-lg shadow-rose-500/5': vehicleStore.activeVehicleId === v.id }"
  >
    <CardOpenButton
      v-if="vehicleStore.activeVehicleId !== v.id"
      :label="$t('vehicles.vehicleCard.selectNamed', { name: v.name })"
      @click="vehicleStore.setActiveVehicle(v.id)"
    />
    <div class="flex items-start justify-between gap-2">
      <div class="flex min-w-0 items-center gap-3">
        <div
          class="shrink-0 rounded-xl bg-slate-800 p-2.5 sm:p-3"
          :class="vehicleStore.activeVehicleId === v.id ? 'text-rose-400' : 'text-slate-300'"
        >
          <Car class="h-6 w-6" aria-hidden="true" />
        </div>
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
            <h3 class="text-base font-bold text-white">{{ v.name }}</h3>
            <span
              v-if="v.role"
              class="rounded-full border px-2 py-0.5 text-xs font-semibold"
              :class="{
                'border-slate-600/40 bg-slate-700/60 text-slate-200': v.role === 'OWNER',
                'border-slate-600/30 bg-slate-800 text-slate-300': v.role === 'EDITOR',
                'border-slate-700/40 bg-slate-800/60 text-slate-400': v.role === 'VIEWER',
              }"
            >
              {{ roleLabel(v.role) }}
            </span>
            <span
              v-if="vehicleStore.activeVehicleId === v.id"
              class="inline-flex items-center gap-1 rounded-full border border-rose-500/30 bg-rose-500/10 px-2 py-0.5 text-xs font-semibold text-rose-300"
            >
              <Check class="h-3 w-3" aria-hidden="true" />{{ $t('vehicles.vehicleCard.active') }}
            </span>
          </div>
          <p v-if="v.vin" class="break-all font-mono text-xs text-slate-400">{{ v.vin }}</p>
        </div>
      </div>

      <div class="flex shrink-0 items-center gap-1">
        <button
          v-if="v.role === 'OWNER'"
          type="button"
          class="row-action relative z-10 max-sm:hidden"
          :title="$t('common.edit')"
          :aria-label="$t('common.edit')"
          @click="emit('edit', v)"
        >
          <Edit2 class="h-4 w-4" aria-hidden="true" />
        </button>
        <RowMenu phone :label="$t('common.moreActions')" :heading="v.name">
          <button v-if="v.role === 'OWNER'" type="button" role="menuitem" class="menu-item sm:hidden" @click="emit('edit', v)">
            <Edit2 class="h-3.5 w-3.5 text-slate-400" aria-hidden="true" />{{ $t('common.edit') }}
          </button>
          <button v-if="v.role === 'OWNER'" type="button" role="menuitem" class="menu-item" @click="emit('ownership', v)">
            <FileText class="h-3.5 w-3.5 text-slate-400" aria-hidden="true" />{{ $t('vehicles.vehicleCard.acquisitionAndFinancing') }}
          </button>
          <button type="button" role="menuitem" class="menu-item" @click="emit('dataSources', v)">
            <Database class="h-3.5 w-3.5 text-slate-400" aria-hidden="true" />{{ $t('vehicles.vehicleCard.dataSources') }}
          </button>
          <button type="button" role="menuitem" class="menu-item" @click="emit('members', v)">
            <Users class="h-3.5 w-3.5 text-slate-400" aria-hidden="true" />{{ $t('vehicles.vehicleCard.sharingAndAccess') }}
          </button>
          <router-link to="/odometer" role="menuitem" class="menu-item" @click="vehicleStore.setActiveVehicle(v.id)">
            <Gauge class="h-3.5 w-3.5 text-slate-400" aria-hidden="true" />{{ $t('shell.nav.odometer') }}
          </router-link>
          <button v-if="v.role === 'OWNER' && v.teslamate_api_url" type="button" role="menuitem" class="menu-item" :disabled="cardTest?.loading" @click="testCardConnection()">
            <Link2 class="h-3.5 w-3.5 text-slate-400" aria-hidden="true" />{{ cardTest?.loading ? $t('vehicles.vehicleCard.testing') : $t('vehicles.vehicleCard.testApi') }}
          </button>
          <button v-if="v.role === 'OWNER'" type="button" role="menuitem" class="menu-item hover:!text-danger-400" @click="emit('delete', v.id)">
            <Trash2 class="h-3.5 w-3.5 text-slate-400" aria-hidden="true" />{{ $t('common.delete') }}
          </button>
        </RowMenu>
      </div>
    </div>

    <!-- The same facts in the same places on every card: what the vehicle is, then how it is fed -->
    <dl class="mt-3 grid grid-cols-2 gap-x-4 gap-y-3 border-t border-slate-800 pt-3 text-xs sm:mt-4 sm:pt-4">
      <div>
        <dt class="text-slate-400">{{ $t('vehicles.vehicleCard.currentOdometer') }}</dt>
        <dd class="mt-0.5 flex items-center gap-1.5 text-sm font-bold text-slate-200">
          <Gauge class="h-3.5 w-3.5 text-slate-400" aria-hidden="true" />
          {{ formatDistance(v.current_odometer) }}
        </dd>
      </div>
      <div>
        <dt class="text-slate-400">{{ $t('vehicles.vehicleCard.powertrain') }}</dt>
        <dd class="mt-0.5 text-sm font-semibold text-slate-200">{{ $t(`vehicles.powertrains.${v.powertrain}`) }}</dd>
      </div>

      <div v-if="canCharge(v.powertrain)">
        <dt class="text-slate-400">{{ $t('vehicles.vehicleCard.teslamateConnection') }}</dt>
        <dd class="mt-0.5 text-sm font-semibold">
          <span v-if="v.role !== 'OWNER'" class="text-slate-400">{{ $t('vehicles.vehicleCard.managedByAdmin') }}</span>
          <!-- Where the connection is set up: the state is the way in -->
          <button
            v-else
            type="button"
            class="relative z-10 -mx-1 rounded-lg px-1 text-left hover:bg-slate-800"
            :class="cardTest?.error ? 'text-danger-400' : cardTest?.success || v.teslamate_api_url ? 'text-success-400' : 'text-sky-400'"
            @click="emit('dataSources', v)"
          >
            {{
              cardTest?.success
                ? $t('vehicles.vehicleCard.online')
                : cardTest?.error
                ? $t('vehicles.vehicleCard.connectionError')
                : v.teslamate_api_url
                ? $t('vehicles.vehicleCard.configured')
                : $t('vehicles.vehicleCard.notConfigured')
            }}
          </button>
        </dd>
      </div>

      <div v-if="isElectricOnly(v.powertrain)">
        <dt class="text-slate-400">{{ $t('vehicles.vehicleCard.estimatedEnergy') }}</dt>
        <dd v-if="hasEnergyEstimate(v)" class="mt-0.5 flex items-center gap-1 text-xs font-semibold text-slate-200">
          <Zap class="h-3.5 w-3.5 shrink-0 text-slate-400" aria-hidden="true" />
          {{ $t('vehicles.vehicleCard.kwh100kmKwh', { unit: distanceUnit(), estimated_kwh_100km: formatPerDistanceValue(Number(v.estimated_kwh_100km)), price: `${formatAmount(v.estimated_price_per_kwh, v.currency, 3)}/kWh` }) }}
        </dd>
        <dd v-else class="mt-0.5 text-xs text-slate-400">{{ $t('vehicles.vehicleCard.notConfigured') }}</dd>
      </div>

      <div class="col-span-2">
        <dt class="mb-1 text-slate-400">{{ v.role === 'OWNER' ? $t('vehicles.vehicleCard.acquisitionAndContract') : $t('vehicles.vehicleCard.vehicleAccess') }}</dt>
        <dd v-if="v.role === 'OWNER'">
          <button
            type="button"
            class="group relative z-10 inline-flex max-w-full items-center gap-1.5 rounded-xl border px-2.5 py-1.5 text-left text-xs font-semibold transition-colors"
            :class="
              ownership
                ? 'border-sky-500/30 bg-sky-500/10 text-sky-300 hover:bg-sky-500/20'
                : 'border-warning-500/30 bg-warning-500/10 text-warning-300 hover:bg-warning-500/20'
            "
            :title="ownership ? $t('vehicles.vehicleCard.editContract') : $t('vehicles.vehicleCard.setUpContract')"
            @click="emit('ownership', v)"
          >
            <FileText v-if="ownership" class="h-3.5 w-3.5 shrink-0 text-sky-400" aria-hidden="true" />
            <Plus v-else class="h-3.5 w-3.5 shrink-0 text-warning-400" aria-hidden="true" />
            <span class="min-w-0">{{ ownership ? ownershipSummary(ownership, v.currency) : $t('vehicles.vehicleCard.enterContract') }}</span>
            <Pencil v-if="ownership" class="ml-0.5 h-3 w-3 shrink-0 text-sky-400 opacity-60 group-hover:opacity-100" aria-hidden="true" />
          </button>
        </dd>
        <dd v-else class="text-xs font-semibold text-slate-300">
          {{ v.role === 'EDITOR' ? $t('vehicles.vehicleCard.editorRole') : $t('vehicles.vehicleCard.readOnly') }}
        </dd>
      </div>
    </dl>

    <!-- Test feedback per card -->
    <div
      v-if="cardTest && !cardTest.loading"
      class="relative z-10 mt-3 flex items-start justify-between gap-2 rounded-xl border p-3 text-xs"
      :class="cardTest.success ? 'border-success-500/20 bg-success-500/10 text-success-300' : 'border-danger-500/20 bg-danger-500/10 text-danger-300'"
    >
      <div class="flex items-start gap-2">
        <CheckCircle2 v-if="cardTest.success" class="mt-0.5 h-4 w-4 shrink-0 text-success-400" aria-hidden="true" />
        <AlertCircle v-else class="mt-0.5 h-4 w-4 shrink-0 text-danger-400" aria-hidden="true" />
        <div>
          <span v-if="cardTest.success">
            {{ $t('vehicles.testSuccess', { unit: distanceUnit(), state: cardTest.status?.state || $t('vehicles.vehicleCard.online'), odometer: formatDistanceValue(cardTest.status?.odometer || 0) }) }}
          </span>
          <span v-else>{{ cardTest.error }}</span>
        </div>
      </div>
      <button type="button" class="tap p-0.5 text-slate-400 hover:text-white" :aria-label="$t('common.close')" @click="clearCardTestResult()">
        <X class="h-3.5 w-3.5" aria-hidden="true" />
      </button>
    </div>
  </div>
</template>
