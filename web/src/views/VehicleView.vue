<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Bell, Car, ClipboardList, Disc, FileText, Gauge, Pencil, Plug, Scale, Users } from 'lucide-vue-next'
import PageHeader from '@/components/PageHeader.vue'
import VehicleTile from '@/components/vehicle/VehicleTile.vue'
import VehicleFormModal from '@/components/vehicles/VehicleFormModal.vue'
import OwnershipWizardModal from '@/components/vehicles/OwnershipWizardModal.vue'
import VehicleMembersModal from '@/components/vehicles/VehicleMembersModal.vue'
import VehicleDataSourcesModal from '@/components/vehicles/VehicleDataSourcesModal.vue'
import { useVehicleStore } from '@/stores/vehicle'
import { api } from '@/services/api'
import { t } from '@/i18n'
import { formatDistance } from '@/units'
import { nextDueReminder } from '@/utils/dashboard'
import { useRefreshHandler } from '@/composables/usePullToRefresh'

// The sheet of the active vehicle: its odometer on top, then what is recorded about it, one line each. Each line opens
// the screen that already exists at its own address, and the system Back comes back here.
const vehicleStore = useVehicleStore()
const vehicle = computed(() => vehicleStore.activeVehicle)
const isOwner = computed(() => !vehicle.value?.role || vehicle.value.role === 'OWNER')

const reminders = ref<any[]>([])
const ownership = ref<any | null>(null)
async function load() {
  const v = vehicle.value
  if (!v) return
  const [r, o] = await Promise.all([
    api.getReminders(v.id).catch(() => []),
    isOwner.value ? api.getOwnership(v.id).catch(() => null) : Promise.resolve(null),
  ])
  reminders.value = r
  ownership.value = o
}
onMounted(load)
watch(() => vehicle.value?.id, load)
useRefreshHandler(async () => {
  await vehicleStore.fetchVehicles()
  await load()
})

const nextService = computed(() => {
  const next = nextDueReminder(reminders.value)
  if (!next) return null
  const left = next.reminder.remaining_km
  return left != null ? t('vehicles.vehicleView.nextServiceIn', { title: next.reminder.title, distance: formatDistance(left) }) : next.reminder.title
})

const showEdit = ref(false)
const showOwnership = ref(false)
const showMembers = ref(false)
const showSources = ref(false)
async function onSaved() {
  await vehicleStore.fetchVehicles()
}
</script>

<template>
  <div v-if="vehicle" class="space-y-5">
    <PageHeader :title="vehicle.name" :icon="Car">{{ $t(`vehicles.powertrains.${vehicle.powertrain}`) }}</PageHeader>

    <!-- The odometer, and the way to record a reading -->
    <div class="card flex items-center gap-3 px-4 py-4">
      <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-slate-800 text-rose-400"><Gauge class="h-5 w-5" aria-hidden="true" /></span>
      <div class="min-w-0 flex-1">
        <p class="text-xs font-semibold text-slate-400">{{ $t('shell.nav.odometer') }}</p>
        <p class="text-2xl font-extrabold text-white">{{ formatDistance(vehicle.current_odometer || 0) }}</p>
      </div>
      <router-link v-if="vehicleStore.canEdit" to="/odometer" class="btn btn-secondary tap-text">{{ $t('vehicles.vehicleView.record') }}</router-link>
    </div>

    <div class="space-y-2">
      <VehicleTile :icon="Bell" :title="$t('vehicles.vehicleView.maintenance')" :detail="nextService" to="/maintenance?tab=REMINDERS" />
      <VehicleTile :icon="Disc" :title="$t('shell.nav.tires')" to="/tires" />
      <VehicleTile :icon="ClipboardList" :title="$t('vehicles.vehicleView.readings')" to="/odometer" />
      <VehicleTile :icon="Scale" :title="$t('shell.nav.comparison')" to="/comparison" />
    </div>

    <div class="space-y-2">
      <VehicleTile v-if="isOwner" :icon="FileText" :title="$t('vehicles.vehicleView.contract')" :detail="ownership ? null : $t('vehicles.vehicleView.contractMissing')" @click="showOwnership = true" />
      <VehicleTile :icon="Plug" :title="$t('vehicles.vehicleView.dataSources')" @click="showSources = true" />
      <VehicleTile :icon="Users" :title="$t('vehicles.vehicleView.sharing')" @click="showMembers = true" />
      <VehicleTile v-if="isOwner" :icon="Pencil" :title="$t('vehicles.vehicleView.edit')" @click="showEdit = true" />
      <VehicleTile v-if="vehicleStore.vehicles.length > 1" :icon="Car" :title="$t('vehicles.vehicleView.manage')" to="/vehicles" />
    </div>

    <VehicleFormModal v-model:open="showEdit" :editing="vehicle" @saved="onSaved" />
    <OwnershipWizardModal v-model:open="showOwnership" :vehicle="vehicle" :ownership="ownership" @saved="(o: any) => { ownership = o; vehicleStore.lastSyncTimestamp = Date.now() }" @deleted="() => { ownership = null; vehicleStore.lastSyncTimestamp = Date.now() }" />
    <VehicleMembersModal v-model:open="showMembers" :vehicle="vehicle" />
    <VehicleDataSourcesModal v-model:open="showSources" :vehicle="vehicle" @saved="onSaved" />
  </div>
</template>
