<script setup lang="ts">
import { Bell, BatteryCharging, ClipboardList, Coins, Disc, FileText, Gauge, Pencil, Plug, Scale, TrendingDown, Users, Zap, Activity, Car } from '@/icons'
import { computed, onMounted, ref, watch } from 'vue'
import PageHeader from '@/components/PageHeader.vue'
import VehicleTile from '@/components/vehicle/VehicleTile.vue'
import SheetRow from '@/components/SheetRow.vue'
import BatteryHealthCard from '@/components/dashboard/BatteryHealthCard.vue'
import ResaleValueCard from '@/components/dashboard/ResaleValueCard.vue'
import ModalShell from '@/components/ModalShell.vue'
import TariffPlansPanel from '@/components/tariffs/TariffPlansPanel.vue'
import LeaseContractCard from '@/components/dashboard/LeaseContractCard.vue'
import TcoDetailsList from '@/components/dashboard/TcoDetailsList.vue'
import VehicleFormModal from '@/components/vehicles/VehicleFormModal.vue'
import OwnershipWizardModal from '@/components/vehicles/OwnershipWizardModal.vue'
import VehicleMembersModal from '@/components/vehicles/VehicleMembersModal.vue'
import VehicleDataSourcesModal from '@/components/vehicles/VehicleDataSourcesModal.vue'
import { useVehicleStore } from '@/stores/vehicle'
import { api } from '@/services/api'
import { t } from '@/i18n'
import { formatDistance } from '@/units'
import { formatAmount } from '@/currency'
import { formatPercent } from '@/utils/numbers'
import { fmt, perUnit } from '@/components/dashboard/energyStats'
import { distanceUnit, formatFuelConsumptionValue, fuelConsumptionUnit, perDistance } from '@/units'
import { fuelConsumptionFigures } from '@/utils/fuelConsumption'
import { summarizeFittedTires, wearAndRemaining } from '@/utils/tires'
import { useTireList } from '@/composables/useTireList'
import { nextDueReminder } from '@/utils/dashboard'
import { useRefreshHandler } from '@/composables/usePullToRefresh'

// The sheet of the active vehicle: its odometer on top, then what is recorded about it, one line each. Each line opens
// the screen that already exists at its own address, and the system Back comes back here.
const vehicleStore = useVehicleStore()
const vehicle = computed(() => vehicleStore.activeVehicle)
const isOwner = computed(() => !vehicle.value?.role || vehicle.value.role === 'OWNER')

const reminders = ref<any[]>([])
const ownership = ref<any | null>(null)
const tco = ref<any | null>(null)
const consumption = ref<number | null>(null)
const fuelFigures = ref<ReturnType<typeof fuelConsumptionFigures>>(null)
const batteryPercent = ref<number | null>(null)
const resaleValue = ref<number | null>(null)
// Bumped when a battery reading changes: the resale estimate depends on the health
const resaleRefresh = ref(0)
const { mountedTires, hasMountedTires, loadTires } = useTireList()
async function load() {
  const v = vehicle.value
  if (!v) return
  void loadTires(true)
  const [r, o, c, health, residual, energy, fuel] = await Promise.all([
    api.getReminders(v.id).catch(() => []),
    isOwner.value ? api.getOwnership(v.id).catch(() => null) : Promise.resolve(null),
    api.getTCO(v.id).catch(() => null),
    vehicleStore.capabilities.charges ? api.getBatteryHealth(v.id).catch(() => null) : Promise.resolve(null),
    api.getResidualValue(v.id, { km_share: 0.5, health_weight: 1 }).catch(() => null),
    vehicleStore.capabilities.charges ? api.getEnergyStats(v.id).catch(() => null) : Promise.resolve(null),
    !vehicleStore.capabilities.charges && vehicleStore.capabilities.refuels ? api.getFuelLogs(v.id).catch(() => null) : Promise.resolve(null),
  ])
  if (v.id !== vehicle.value?.id) return
  reminders.value = r
  ownership.value = o
  tco.value = c
  const kwh100 = perUnit(energy?.summary?.consumption_kwh_100km)
  consumption.value = typeof kwh100 === 'number' && kwh100 > 0 ? kwh100 : null
  fuelFigures.value = fuelConsumptionFigures(fuel)
  batteryPercent.value = typeof health?.health_percent === 'number' ? health.health_percent : null
  resaleValue.value = typeof residual?.current_value === 'number' ? residual.current_value : null
}
onMounted(load)
watch([() => vehicle.value?.id, resaleRefresh], load)
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

// The most worn tire, said once in words; no state colour on the line, the colour is on the tires screen
const tiresDetail = computed(() => {
  if (!hasMountedTires.value) return null
  const worn = summarizeFittedTires(mountedTires.value).mostWorn
  if (!worn) return null
  const { worn: used, left } = wearAndRemaining(worn.pct)
  return t('tires.tireSetSummary.wornAndLeft', { worn: formatPercent(used, 0), left: formatPercent(left, 0) })
})
// An electric or hybrid vehicle says its kWh per 100; a combustion one its litres per 100 and what they cost, from the fill-ups
const consumptionDetail = computed(() => {
  if (vehicleStore.capabilities.charges) return consumption.value === null ? null : `${fmt(consumption.value, 1)} kWh/100 ${distanceUnit()}`
  const figures = fuelFigures.value
  if (!figures) return t('vehicles.vehicleView.consumptionFuelEmpty')
  const consumptionText = `${formatFuelConsumptionValue(figures.litresPer100, 1)} ${fuelConsumptionUnit()}`
  if (figures.costPer100 === null) return consumptionText
  const cost = `${formatAmount(perDistance(figures.costPer100), vehicleStore.currency, 2)} / 100 ${distanceUnit()}`
  return t('vehicles.vehicleView.consumptionFuel', { consumption: consumptionText, cost })
})
const batteryHint = computed(() => (batteryPercent.value === null ? t('dashboard.sheets.batteryHint') : t('dashboard.sheets.batteryNow', { pct: formatPercent(batteryPercent.value, 0) })))
const resaleHint = computed(() => (resaleValue.value === null ? t('dashboard.sheets.resaleHint') : t('dashboard.sheets.resaleNow', { value: formatAmount(resaleValue.value, vehicleStore.currency, 0) })))

const showTariffs = ref(false)
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
      <VehicleTile v-if="vehicleStore.capabilities.charges || vehicleStore.capabilities.refuels" :icon="Activity" :title="$t('vehicles.vehicleView.consumption')" :detail="consumptionDetail" to="/consumption" />
      <VehicleTile :icon="Disc" :title="$t('shell.nav.tires')" :detail="tiresDetail" to="/tires" />
      <VehicleTile :icon="ClipboardList" :title="$t('vehicles.vehicleView.readings')" to="/odometer" />
      <VehicleTile :icon="Scale" :title="$t('shell.nav.comparison')" to="/comparison" />
    </div>

    <!-- Condition and value of the vehicle -->
    <div class="space-y-2">
      <SheetRow v-if="vehicleStore.capabilities.charges" v-slot="{ bare }" :title="$t('dashboard.sheets.battery')" :hint="batteryHint" :icon="BatteryCharging">
        <BatteryHealthCard :vehicle-id="vehicle.id" :bare="bare" @updated="resaleRefresh++" />
      </SheetRow>
      <SheetRow v-slot="{ bare }" :title="$t('dashboard.sheets.resale')" :hint="resaleHint" :icon="TrendingDown">
        <ResaleValueCard :vehicle-id="vehicle.id" :has-battery="vehicleStore.capabilities.charges" :refresh-key="resaleRefresh" :bare="bare" />
      </SheetRow>
      <LeaseContractCard :tco="tco" />
      <SheetRow v-if="tco" v-slot="{ bare }" :title="$t('dashboard.sheets.totals')" :hint="$t('dashboard.sheets.totalsHint')" :icon="Coins">
        <TcoDetailsList :class="bare ? '' : 'p-4'" :tco="tco" />
      </SheetRow>
    </div>

    <div class="space-y-2">
      <VehicleTile v-if="isOwner" :icon="FileText" :title="$t('vehicles.vehicleView.contract')" :detail="ownership ? null : $t('vehicles.vehicleView.contractMissing')" @click="showOwnership = true" />
      <VehicleTile v-if="vehicleStore.capabilities.charges" :icon="Zap" :title="$t('vehicles.vehicleView.homeCharging')" :detail="$t('vehicles.vehicleView.homeChargingHint')" @click="showTariffs = true" />
      <VehicleTile :icon="Plug" :title="$t('vehicles.vehicleView.dataSources')" @click="showSources = true" />
      <VehicleTile :icon="Users" :title="$t('vehicles.vehicleView.sharing')" @click="showMembers = true" />
      <VehicleTile v-if="isOwner" :icon="Pencil" :title="$t('vehicles.vehicleView.edit')" @click="showEdit = true" />
      <VehicleTile v-if="vehicleStore.vehicles.length > 1" :icon="Car" :title="$t('vehicles.vehicleView.manage')" to="/vehicles" />
    </div>

    <ModalShell v-model:open="showTariffs" :title="$t('tariffs.editor.title')" :icon="Zap" size="lg">
      <TariffPlansPanel />
    </ModalShell>
    <VehicleFormModal v-model:open="showEdit" :editing="vehicle" @saved="onSaved" />
    <OwnershipWizardModal v-model:open="showOwnership" :vehicle="vehicle" :ownership="ownership" @saved="(o: any) => { ownership = o; vehicleStore.lastSyncTimestamp = Date.now() }" @deleted="() => { ownership = null; vehicleStore.lastSyncTimestamp = Date.now() }" />
    <VehicleMembersModal v-model:open="showMembers" :vehicle="vehicle" />
    <VehicleDataSourcesModal v-model:open="showSources" :vehicle="vehicle" @saved="onSaved" />
  </div>
</template>
