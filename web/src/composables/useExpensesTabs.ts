import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { t } from '@/i18n'
import { useVehicleStore } from '@/stores/vehicle'
import type { TabItem } from '@/components/TabBar.vue'
import { Receipt, Fuel, Wrench, Zap, Gauge, Calculator, Paperclip, Bell, Landmark } from 'lucide-vue-next'

export type ExpensesTab = 'TOLLS' | 'FIXED' | 'MAINTENANCE' | 'REMINDERS' | 'CHARGES' | 'FUEL' | 'DOCUMENTS' | 'EFFICIENCY' | 'ESTIMATE'

export interface ExpensesTabBadges {
  urgentReminders: number
  overdueReminders: number
  chargesWithoutCost: number
  documents: number
}

// One view, three menu entries: /expenses (tolls, fixed costs, receipts), /maintenance (maintenance, reminders)
// and /energy (efficiency, charges, estimate). Owns the section, the tab list and the `tab` query parameter.
// The badges are read lazily, so they may come from state declared after this call.
export function useExpensesTabs(badges: () => ExpensesTabBadges) {
  const router = useRouter()
  const route = useRoute()
  const vehicleStore = useVehicleStore()

  const isMaintenanceSection = computed(() => route.meta.section === 'maintenance')
  const isEnergySection = computed(() => route.meta.section === 'energy')

  // Electric: efficiency, charges, estimate. Combustion: fill-ups. Hybrid: efficiency, charges and fill-ups.
  function energyTabs(): ExpensesTab[] {
    if (!vehicleStore.canCharge) return ['FUEL']
    return vehicleStore.canRefuel ? ['EFFICIENCY', 'CHARGES', 'FUEL'] : ['EFFICIENCY', 'CHARGES', 'ESTIMATE']
  }
  const sectionTabs = computed<ExpensesTab[]>(() => {
    if (isMaintenanceSection.value) return ['MAINTENANCE', 'REMINDERS']
    if (isEnergySection.value) return energyTabs()
    return ['TOLLS', 'FIXED', 'DOCUMENTS']
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

  const sectionKey = computed(() => (isMaintenanceSection.value ? 'maintenance' : isEnergySection.value ? 'energy' : 'expenses'))
  const pageTitle = computed(() => t(`expenses.expensesView.${sectionKey.value}Title`))
  const pageSubtitle = computed(() => t(`expenses.expensesView.${sectionKey.value}Subtitle`))
  const pageIcon = computed(() => (isMaintenanceSection.value ? Wrench : isEnergySection.value ? Zap : Receipt))

  const tabs = computed<TabItem[]>(() => {
    const b = badges()
    if (isMaintenanceSection.value) {
      return [
        { key: 'MAINTENANCE', label: t('expenses.expensesView.maintenance'), icon: Wrench },
        {
          key: 'REMINDERS',
          label: t('expenses.expensesView.reminders'),
          icon: Bell,
          badge: b.urgentReminders > 0 ? b.urgentReminders : undefined,
          badgeTone: b.overdueReminders > 0 ? 'danger' : 'warning',
        },
      ]
    }
    if (isEnergySection.value) {
      const list: TabItem[] = [
        { key: 'EFFICIENCY', label: t('expenses.expensesView.efficiency'), icon: Gauge },
        {
          key: 'CHARGES',
          label: t('expenses.expensesView.charges'),
          icon: Zap,
          badge: b.chargesWithoutCost > 0 ? b.chargesWithoutCost : undefined,
          badgeTone: 'warning',
        },
      ]
      if (!vehicleStore.canRefuel) list.push({ key: 'ESTIMATE', label: t('expenses.expensesView.estimate'), icon: Calculator })
      const fuel: TabItem = { key: 'FUEL', label: t('expenses.expensesView.fillUps'), icon: Fuel }
      if (!vehicleStore.canCharge) return [fuel]
      if (vehicleStore.canRefuel) list.push(fuel)
      return list
    }
    return [
      { key: 'TOLLS', label: t('expenses.expensesView.tolls'), icon: Receipt },
      { key: 'FIXED', label: t('expenses.expensesView.fixedCosts'), icon: Landmark },
      {
        key: 'DOCUMENTS',
        label: t('expenses.expensesView.receipts'),
        icon: Paperclip,
        badge: b.documents > 0 ? b.documents : undefined,
      },
    ]
  })

  return { activeTab, isMaintenanceSection, isEnergySection, pageTitle, pageSubtitle, pageIcon, tabs }
}
