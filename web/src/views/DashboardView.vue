<script setup lang="ts">
import LoadError from '@/components/LoadError.vue'
import PageHeader from '@/components/PageHeader.vue'
import { Activity, BatteryCharging, ChartPie, LayoutDashboard as PageIcon, TrendingDown } from 'lucide-vue-next'
import { useRefreshHandler } from '@/composables/usePullToRefresh'
import SheetRow from '@/components/SheetRow.vue'
import { ref, onMounted, watch, computed } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import { usePreferencesStore } from '@/stores/preferences'
import { api, type MaintenanceReminder } from '@/services/api'

import BatteryHealthCard from '@/components/dashboard/BatteryHealthCard.vue'
import ResaleValueCard from '@/components/dashboard/ResaleValueCard.vue'
import EnergyEfficiencyPanel from '@/components/dashboard/EnergyEfficiencyPanel.vue'
import UrgentRemindersBanner from '@/components/dashboard/UrgentRemindersBanner.vue'
import NextDueCard from '@/components/dashboard/NextDueCard.vue'
import DashboardFollowUp from '@/components/dashboard/DashboardFollowUp.vue'
import DataQualityCard from '@/components/dashboard/DataQualityCard.vue'
import LeaseContractCard from '@/components/dashboard/LeaseContractCard.vue'
import TcoMetricsGrid from '@/components/dashboard/TcoMetricsGrid.vue'
import MonthlyCostChart from '@/components/dashboard/MonthlyCostChart.vue'
import CostBreakdownDonut from '@/components/dashboard/CostBreakdownDonut.vue'
import MileageCostChart from '@/components/dashboard/MileageCostChart.vue'
import TagBreakdown from '@/components/dashboard/TagBreakdown.vue'
import MonthDetailModal from '@/components/dashboard/MonthDetailModal.vue'
import MonthlyRangeSelector from '@/components/dashboard/MonthlyRangeSelector.vue'
import { distanceUnit, formatDistance } from '@/units'
import { formatAmount } from '@/currency'
import { formatMonthLabel } from '@/utils/dates'
import { formatPercent } from '@/utils/numbers'
import { lastDrivenMonth, topCostShares } from '@/utils/homeRows'
import { useIsMobile } from '@/composables/useIsMobile'
import { countFollowUpItems, followUpItems } from '@/utils/dashboard'
import { t } from '@/i18n'
import { hasReminderSchedule } from '@/utils/expenses'

// The page loads the TCO and the reminders; each card and chart of the dashboard is a component that
// receives the data it shows. A click on a month (chart or banner) opens its detail modal.
const vehicleStore = useVehicleStore()
const prefs = usePreferencesStore()
const vehicleId = computed(() => vehicleStore.activeVehicle?.id ?? '')
const tco = ref<any | null>(null)
const loading = ref(true)
const loadError = ref<string | null>(null)
const dashboardReminders = ref<MaintenanceReminder[]>([])
const followUpCount = computed(() => countFollowUpItems(tco.value?.completeness, dashboardReminders.value, dashboardReminders.value.some(hasReminderSchedule)))
const followUpLabels = computed(() => {
  const items = followUpItems(tco.value?.completeness, dashboardReminders.value, dashboardReminders.value.some(hasReminderSchedule))
  const labels: string[] = []
  if (items.data) labels.push(t('dashboard.followUp.toComplete'))
  if (items.maintenance === 'soon') labels.push(t('dashboard.followUp.maintenanceSoon'))
  if (items.maintenance === 'plan') labels.push(t('dashboard.followUp.maintenanceToPlan'))
  return labels
})
const selectedMonth = ref<any | null>(null)
// Bumped when a battery reading changes: the resale estimate depends on the health
const resaleRefresh = ref(0)

// Monthly costs for the charts; null until the TCO is loaded
const monthlyCosts = computed(() => (tco.value ? tco.value.monthly_costs || [] : null))

// Below sm the blocks under the trend are rows: each one says what it holds, so the figure is read without opening it.
// The battery and the resale value are loaded here for that only (the cards load their own when opened), and a phone only.
const isMobile = useIsMobile()
const batteryPercent = ref<number | null>(null)
const resaleValue = ref<number | null>(null)

useRefreshHandler(() => Promise.all([loadTCO(true), loadRowFigures()]))

async function loadRowFigures() {
  batteryPercent.value = null
  resaleValue.value = null
  const id = vehicleId.value
  if (!id || !isMobile.value) return
  const [health, residual] = await Promise.all([
    vehicleStore.canCharge ? api.getBatteryHealth(id).catch(() => null) : null,
    api.getResidualValue(id, { km_share: 0.5, health_weight: 1 }).catch(() => null),
  ])
  if (id !== vehicleId.value) return
  batteryPercent.value = typeof health?.health_percent === 'number' ? health.health_percent : null
  resaleValue.value = typeof residual?.current_value === 'number' ? residual.current_value : null
}
watch([vehicleId, resaleRefresh, isMobile], loadRowFigures, { immediate: true })

const breakdownHint = computed(() => {
  const top = topCostShares(tco.value, 1)
  return top.length ? top.map((r) => `${t(`dashboard.donut.${r.key}`)} ${formatPercent(r.sharePct, 0)}`).join(' · ') : t('dashboard.sheets.breakdownHint')
})
const mileageHint = computed(() => {
  const m = lastDrivenMonth(monthlyCosts.value)
  return m ? t('dashboard.sheets.mileageLast', { month: formatMonthLabel(m.month), distance: formatDistance(m.distance_km) }) : t('dashboard.sheets.mileageHint')
})
const batteryHint = computed(() => (batteryPercent.value === null ? t('dashboard.sheets.batteryHint') : t('dashboard.sheets.batteryNow', { pct: formatPercent(batteryPercent.value, 0) })))
const resaleHint = computed(() => (resaleValue.value === null ? t('dashboard.sheets.resaleHint') : t('dashboard.sheets.resaleNow', { value: formatAmount(resaleValue.value, vehicleStore.currency, 0) })))

function openMonthDetail(m: any) {
  if (!m) return
  selectedMonth.value = m
}

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
    dashboardReminders.value = remindersData
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
    loadTCO()
  }
)

onMounted(() => {
  loadTCO()
})
</script>

<template>
  <div class="space-y-6">
    <!-- Header Summary -->
    <PageHeader :title="$t('shell.nav.dashboard')" :icon="PageIcon">
      {{ $t('dashboard.dashboardView.subtitle', { unit: distanceUnit(), name: vehicleStore.activeVehicle?.name || $t('dashboard.dashboardView.yourVehicle') }) }}
      <!-- The quick-add button and the bottom bar already cover this on a phone -->
      <template #actions>
        <router-link
          to="/expenses"
          class="btn btn-lg btn-secondary tap-text max-sm:hidden"
        >
          {{ $t('dashboard.dashboardView.expense') }}
        </router-link>
      </template>
    </PageHeader>

    <!-- Empty state if no vehicle -->
    <div v-if="!vehicleStore.activeVehicle" class="p-8 text-center card">
      <p class="text-slate-400 mb-4">{{ $t('dashboard.dashboardView.noVehicleConfigured') }}</p>
      <router-link to="/vehicles" class="btn btn-lg btn-primary">
        {{ $t('dashboard.dashboardView.createYourFirstVehicle') }}
      </router-link>
    </div>

    <!-- SKELETON LOADING STATE -->
    <div v-else-if="loading" class="space-y-6 animate-pulse motion-reduce:animate-none">
      <!-- 4 KPI Card Skeletons -->
      <div class="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
        <div v-for="i in 4" :key="i" class="p-5 card h-32 flex flex-col justify-between">
          <div class="flex justify-between items-center">
            <div class="h-3 w-20 bg-slate-800 rounded"></div>
            <div class="h-8 w-8 bg-slate-800 rounded-xl"></div>
          </div>
          <div class="space-y-2">
            <div class="h-7 w-28 bg-slate-800 rounded"></div>
            <div class="h-3 w-36 bg-slate-800/60 rounded"></div>
          </div>
        </div>
      </div>

      <!-- Charts Skeletons -->
      <!-- One period for the monthly cost and mileage charts below; pinned only where there is room for it -->
      <div class="sm:sticky sm:top-[4.25rem] z-30 flex flex-wrap items-center justify-between gap-2 bg-slate-950/90 py-2 backdrop-blur-md">
        <span class="text-xs font-semibold text-slate-300 max-sm:sr-only">{{ $t('dashboard.dashboardView.chartsPeriod') }}</span>
        <MonthlyRangeSelector
          class="w-full sm:w-auto"
          :model-value="prefs.dashboardRange"
          :label="$t('dashboard.dashboardView.chartsPeriod')"
          @update:model-value="prefs.setDashboardRange"
        />
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div class="lg:col-span-2 card p-5 h-80 flex flex-col justify-between">
          <div class="h-4 w-44 bg-slate-800 rounded"></div>
          <div class="h-56 w-full bg-slate-800/40 rounded-xl"></div>
        </div>
        <div class="card p-5 h-80 flex flex-col justify-between">
          <div class="h-4 w-32 bg-slate-800 rounded"></div>
          <div class="h-52 w-52 rounded-full border-8 border-slate-800 mx-auto"></div>
        </div>
      </div>

      <!-- Second Chart Skeleton -->
      <div class="card p-5 h-80 flex flex-col justify-between">
        <div class="h-4 w-52 bg-slate-800 rounded"></div>
        <div class="h-56 w-full bg-slate-800/40 rounded-xl"></div>
      </div>
    </div>

    <!-- REAL CONTENT WHEN LOADED -->
    <LoadError v-else-if="loadError !== null" :message="loadError" @retry="loadTCO()" />

    <div v-else class="space-y-4 sm:space-y-6">
      <UrgentRemindersBanner :reminders="dashboardReminders" />

      <TcoMetricsGrid :tco="tco" @open-month="openMonthDetail" />

      <!-- What can be trusted and what is missing, under the figures -->
      <DashboardFollowUp v-if="tco?.completeness || followUpCount > 0" :count="followUpCount" :items="followUpLabels">
        <DataQualityCard :tco="tco" :vehicle-id="vehicleId" />
        <NextDueCard :reminders="dashboardReminders" :can-edit="vehicleStore.canEdit" />
      </DashboardFollowUp>

      <!-- One period for the charts and the energy summary below; pinned only where there is room for it -->
      <div class="sm:sticky sm:top-[4.25rem] z-30 flex flex-wrap items-center justify-between gap-2 bg-slate-950/90 py-2 backdrop-blur-md">
        <span class="text-xs font-semibold text-slate-300 max-sm:sr-only">{{ $t('dashboard.dashboardView.chartsPeriod') }}</span>
        <MonthlyRangeSelector
          class="w-full sm:w-auto"
          :model-value="prefs.dashboardRange"
          :label="$t('dashboard.dashboardView.chartsPeriod')"
          @update:model-value="prefs.setDashboardRange"
        />
      </div>

      <!-- Energy summary: consumption and real cost per 100 km (electric vehicles); the detail is on the energy page -->
      <EnergyEfficiencyPanel
        v-if="vehicleStore.activeVehicle && vehicleStore.canCharge && (!vehicleStore.canRefuel || (tco?.total_kwh_added || 0) > 0)"
        :vehicle-id="vehicleStore.activeVehicle.id"
        :grafana-url="vehicleStore.activeVehicle.teslamate_grafana_url"
        :sync-key="vehicleStore.lastSyncTimestamp"
        show-link
        summary
      />

      <!-- Below sm each block below the figures is one row that opens it in a sheet, so the page is a short list instead of
           a long stack of charts; a month opened from a sheet takes over from it -->
      <!-- The monthly trend stays on the page: it is the one chart worth reading at a glance. The breakdown is a row below it on a phone -->
      <div class="grid grid-cols-1 gap-4 sm:gap-6 lg:grid-cols-3">
        <MonthlyCostChart :monthly-costs="monthlyCosts" @open-month="openMonthDetail($event)" />
        <SheetRow v-slot="{ bare }" :title="$t('dashboard.sheets.breakdown')" :hint="breakdownHint" :icon="ChartPie">
          <CostBreakdownDonut :tco="tco" :bare="bare" />
        </SheetRow>
      </div>

      <SheetRow v-slot="{ bare, close }" :title="$t('dashboard.sheets.mileage')" :hint="mileageHint" :icon="Activity">
        <MileageCostChart :monthly-costs="monthlyCosts" :bare="bare" @open-month="close(); openMonthDetail($event)" />
      </SheetRow>

      <!-- Shows itself only once a drive carries a tag -->
      <TagBreakdown v-if="prefs.proPersoEnabled" :tco="tco" />

      <!-- Condition and value of the vehicle: slower-moving than the cost figures above, so they come last -->
      <LeaseContractCard :tco="tco" />

      <!-- Two unrelated questions, so two cards: how healthy the battery is, and what the vehicle is worth -->
      <div v-if="vehicleStore.activeVehicle" class="grid grid-cols-1 items-start gap-4 sm:gap-6" :class="vehicleStore.canCharge ? 'lg:grid-cols-2' : ''">
        <SheetRow v-if="vehicleStore.canCharge" v-slot="{ bare }" :title="$t('dashboard.sheets.battery')" :hint="batteryHint" :icon="BatteryCharging">
          <BatteryHealthCard :vehicle-id="vehicleStore.activeVehicle.id" :bare="bare" @updated="resaleRefresh++" />
        </SheetRow>
        <SheetRow v-slot="{ bare }" :title="$t('dashboard.sheets.resale')" :hint="resaleHint" :icon="TrendingDown">
          <ResaleValueCard
            :vehicle-id="vehicleStore.activeVehicle.id"
            :has-battery="vehicleStore.canCharge"
            :refresh-key="resaleRefresh"
            :bare="bare"
          />
        </SheetRow>
      </div>
    </div>

    <MonthDetailModal v-model:month="selectedMonth" :monthly-costs="tco?.monthly_costs || []" />
  </div>
</template>
