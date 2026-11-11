import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { t } from '@/i18n'
import { useVehicleStore } from '@/stores/vehicle'
import type { TabItem } from '@/components/TabBar.vue'
import { Receipt, Fuel, Wrench, Zap, Activity, Gauge, Calculator, Paperclip, Bell } from '@/icons'

export type ExpensesTab = 'REMINDERS' | 'CHARGES' | 'FUEL' | 'DOCUMENTS' | 'EFFICIENCY' | 'ESTIMATE'

// One view, three entries, the tools next to the lists of the Costs page: /expenses (receipts), /maintenance
// (reminders) and /consumption (efficiency, estimate, fill-ups, charges to complete). Owns the section, the tab list and
// the `tab` query parameter. A tab carries its label and icon only: what there is to count is said in the tab's content.
export function useExpensesTabs() {
  const router = useRouter()
  const route = useRoute()
  const vehicleStore = useVehicleStore()

  const isMaintenanceSection = computed(() => route.meta.section === 'maintenance')
  const isConsumptionSection = computed(() => route.meta.section === 'consumption')

  // Electric: efficiency, estimate, charges to complete. Combustion: fill-ups. Hybrid: efficiency, fill-ups, charges.
  // The page opens on how the vehicle consumes; the charges to complete come last.
  function consumptionTabs(): ExpensesTab[] {
    if (!vehicleStore.canCharge) return ['FUEL']
    return vehicleStore.canRefuel ? ['EFFICIENCY', 'FUEL', 'CHARGES'] : ['EFFICIENCY', 'ESTIMATE', 'CHARGES']
  }
  const sectionTabs = computed<ExpensesTab[]>(() => {
    if (isMaintenanceSection.value) return ['REMINDERS']
    if (isConsumptionSection.value) return consumptionTabs()
    return ['DOCUMENTS']
  })

  function parseTab(raw: unknown): ExpensesTab {
    const upper = String(Array.isArray(raw) ? raw[0] : (raw ?? '')).toUpperCase() as ExpensesTab
    return sectionTabs.value.includes(upper) ? upper : sectionTabs.value[0]
  }

  const activeTab = ref<ExpensesTab>(parseTab(route.query.tab))

  watch(activeTab, (newTab) => {
    if (route.query.tab !== newTab) {
      router.replace({ query: { ...route.query, tab: newTab } })
    }
  })

  watch([() => route.query.tab, () => route.meta.section, () => vehicleStore.canRefuel], ([qTab]) => {
    const tab = parseTab(qTab)
    if (activeTab.value !== tab) activeTab.value = tab
  })

  const sectionKey = computed(() => (isMaintenanceSection.value ? 'maintenance' : isConsumptionSection.value ? 'consumption' : 'expenses'))
  const pageTitle = computed(() => t(`expenses.expensesView.${sectionKey.value}Title`))
  const pageSubtitle = computed(() => t(`expenses.expensesView.${sectionKey.value}Subtitle`))
  const pageIcon = computed(() => (isMaintenanceSection.value ? Wrench : isConsumptionSection.value ? Activity : Receipt))

  const tabs = computed<TabItem[]>(() => {
    if (isMaintenanceSection.value) {
      return [
        { key: 'REMINDERS', label: t('expenses.expensesView.reminders'), icon: Bell },
      ]
    }
    if (isConsumptionSection.value) {
      const fuel: TabItem = { key: 'FUEL', label: t('expenses.expensesView.fillUps'), icon: Fuel }
      if (!vehicleStore.canCharge) return [fuel]
      const charges: TabItem = { key: 'CHARGES', label: t('expenses.expensesView.charges'), icon: Zap }
      const efficiency: TabItem = { key: 'EFFICIENCY', label: t('expenses.expensesView.efficiency'), icon: Gauge }
      const second: TabItem = vehicleStore.canRefuel ? fuel : { key: 'ESTIMATE', label: t('expenses.expensesView.estimate'), icon: Calculator }
      return [efficiency, second, charges]
    }
    return [
      { key: 'DOCUMENTS', label: t('expenses.expensesView.receipts'), icon: Paperclip },
    ]
  })

  return { activeTab, isMaintenanceSection, isConsumptionSection, pageTitle, pageSubtitle, pageIcon, tabs }
}
