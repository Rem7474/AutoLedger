<script setup lang="ts">
import LoadError from '@/components/LoadError.vue'
import { Car, ChevronRight } from 'lucide-vue-next'
import { useRefreshHandler } from '@/composables/usePullToRefresh'
import { ref, onMounted, watch, computed } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import { api, type MaintenanceReminder } from '@/services/api'
import { useTireList } from '@/composables/useTireList'

import TodayHero from '@/components/dashboard/TodayHero.vue'
import UpcomingList from '@/components/dashboard/UpcomingList.vue'
import DataQualityCard from '@/components/dashboard/DataQualityCard.vue'
import { currentMonthStats } from '@/utils/dashboard'
import { todayHero } from '@/utils/todayHero'
import { summarizeFittedTires } from '@/utils/tires'
import { formatAmount } from '@/currency'
import { formatPercent } from '@/utils/numbers'
import { t } from '@/i18n'

// The page of the present: what the month costs so far, what is coming up, what needs a look. The charts, the
// breakdown and the history are in the journal of costs, the battery, the resale value and the contract on the vehicle sheet.
const vehicleStore = useVehicleStore()
const vehicleId = computed(() => vehicleStore.activeVehicle?.id ?? '')
const tco = ref<any | null>(null)
const loading = ref(true)
const loadError = ref<string | null>(null)
const reminders = ref<MaintenanceReminder[]>([])
const { mountedTires, loadTires } = useTireList()
const tireSummary = computed(() => summarizeFittedTires(mountedTires.value))
const hero = computed(() => {
  const month = currentMonthStats(tco.value?.monthly_costs, new Date(), tco.value?.current_month)
  return todayHero(month?.raw, vehicleStore.currency)
})

// The battery and the resale value are loaded for the one line that points to the vehicle sheet
const batteryPercent = ref<number | null>(null)
const resaleValue = ref<number | null>(null)

useRefreshHandler(() => Promise.all([loadTCO(true), loadRowFigures(), loadTires(true)]))

async function loadRowFigures() {
  batteryPercent.value = null
  resaleValue.value = null
  const id = vehicleId.value
  if (!id) return
  const [health, residual] = await Promise.all([
    vehicleStore.canCharge ? api.getBatteryHealth(id).catch(() => null) : null,
    api.getResidualValue(id, { km_share: 0.5, health_weight: 1 }).catch(() => null),
  ])
  if (id !== vehicleId.value) return
  batteryPercent.value = typeof health?.health_percent === 'number' ? health.health_percent : null
  resaleValue.value = typeof residual?.current_value === 'number' ? residual.current_value : null
}
watch(vehicleId, loadRowFigures, { immediate: true })

// The line says the figure when there is one (battery health first, then the resale value), else what the sheet holds
const vehicleSheetHint = computed(() => {
  if (batteryPercent.value !== null) return t('dashboard.sheets.batteryNow', { pct: formatPercent(batteryPercent.value, 0) })
  if (resaleValue.value !== null) return t('dashboard.sheets.resaleNow', { value: formatAmount(resaleValue.value, vehicleStore.currency, 0) })
  return t('dashboard.sheets.vehicleSheetHint')
})

async function loadTCO(silent = false) {
  if (!vehicleStore.activeVehicle) {
    loading.value = false
    return
  }
  if (!silent) loading.value = true
  loadError.value = null
  try {
    const [tcoData, remindersData] = await Promise.all([
      api.getTCO(vehicleStore.activeVehicle.id),
      api.getReminders(vehicleStore.activeVehicle.id).catch(() => []),
    ])
    tco.value = tcoData
    reminders.value = remindersData
  } catch (err) {
    console.error('Failed to load TCO or reminders', err)
    loadError.value = (err as Error)?.message ?? ''
  } finally {
    loading.value = false
  }
}

watch(
  () => [vehicleStore.activeVehicle?.id, vehicleStore.lastSyncTimestamp],
  () => {
    void loadTCO()
    void loadTires(true)
  },
)

onMounted(() => {
  void loadTCO()
  void loadTires()
})
</script>

<template>
  <div class="space-y-6">
    <!-- The skeleton has the shape of the page: the hero card, the coming-up rows, the line to the vehicle -->
    <div v-if="loading" class="space-y-6 animate-pulse motion-reduce:animate-none" data-testid="today-skeleton">
      <div class="card p-5 space-y-3">
        <div class="h-3 w-44 bg-slate-800 rounded"></div>
        <div class="h-10 w-40 bg-slate-800 rounded"></div>
        <div class="h-2 w-full bg-slate-800/60 rounded-full"></div>
        <div class="h-3 w-56 max-w-full bg-slate-800/60 rounded"></div>
      </div>
      <div class="card divide-y divide-slate-800/70">
        <div v-for="i in 2" :key="i" class="flex items-center gap-3 p-4">
          <div class="h-10 w-10 shrink-0 bg-slate-800 rounded-xl"></div>
          <div class="space-y-2 min-w-0">
            <div class="h-4 w-32 bg-slate-800 rounded"></div>
            <div class="h-3 w-44 max-w-full bg-slate-800/60 rounded"></div>
          </div>
        </div>
      </div>
      <div class="h-16 w-full bg-slate-900 rounded-2xl"></div>
    </div>

    <LoadError v-else-if="loadError !== null" :message="loadError" @retry="loadTCO()" />

    <div v-else class="space-y-4 sm:space-y-6">
      <TodayHero v-if="hero" :hero="hero" />
      <UpcomingList :reminders="reminders" :tires="tireSummary" :can-edit="vehicleStore.canEdit" />
      <DataQualityCard :tco="tco" :vehicle-id="vehicleId" />

      <!-- Battery health, resale value and the contract are on the vehicle sheet; the page keeps one line that says where -->
      <router-link v-if="vehicleStore.activeVehicle" to="/vehicle" class="flex w-full items-center gap-3 card px-4 py-3 transition-colors active:bg-slate-800/60">
        <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-slate-800 text-slate-300"><Car class="h-5 w-5" aria-hidden="true" /></span>
        <span class="min-w-0 flex-1">
          <span class="block text-sm font-semibold text-white">{{ $t('dashboard.sheets.vehicleSheet') }}</span>
          <span class="block truncate text-xs text-slate-400">{{ vehicleSheetHint }}</span>
        </span>
        <ChevronRight class="h-5 w-5 shrink-0 text-slate-500" aria-hidden="true" />
      </router-link>
    </div>
  </div>
</template>
