import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, reactive } from 'vue'
import { useExpensesTabs } from './useExpensesTabs'

const route = reactive<{ query: Record<string, unknown>; meta: Record<string, unknown> }>({ query: {}, meta: {} })
const replace = vi.fn()
vi.mock('vue-router', () => ({ useRoute: () => route, useRouter: () => ({ replace }) }))
const store = reactive({ canCharge: true, canRefuel: false })
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => store }))

const badges = reactive({ urgentReminders: 0, overdueReminders: 0, chargesWithoutCost: 0, documents: 0 })
const keys = (tabs: { key: string }[]) => tabs.map((x) => x.key)

describe('useExpensesTabs', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    route.query = {}
    route.meta = {}
    store.canCharge = true
    store.canRefuel = false
    Object.assign(badges, { urgentReminders: 0, overdueReminders: 0, chargesWithoutCost: 0, documents: 0 })
  })

  it('lists the tabs of each section and of each powertrain', () => {
    expect(keys(useExpensesTabs(() => badges).tabs.value)).toEqual(['TOLLS', 'FIXED', 'DOCUMENTS'])
    route.meta = { section: 'maintenance' }
    expect(keys(useExpensesTabs(() => badges).tabs.value)).toEqual(['MAINTENANCE', 'REMINDERS'])
    route.meta = { section: 'energy' }
    expect(keys(useExpensesTabs(() => badges).tabs.value)).toEqual(['EFFICIENCY', 'CHARGES', 'ESTIMATE'])
    store.canRefuel = true
    expect(keys(useExpensesTabs(() => badges).tabs.value)).toEqual(['EFFICIENCY', 'CHARGES', 'FUEL'])
    store.canCharge = false
    expect(keys(useExpensesTabs(() => badges).tabs.value)).toEqual(['FUEL'])
  })

  it('opens the tab of the query, falling back to the first tab of the section', () => {
    route.query = { tab: 'fixed' }
    expect(useExpensesTabs(() => badges).activeTab.value).toBe('FIXED')
    route.query = { tab: 'REMINDERS' }
    expect(useExpensesTabs(() => badges).activeTab.value).toBe('TOLLS')
    route.query = { tab: ['documents', 'x'] }
    expect(useExpensesTabs(() => badges).activeTab.value).toBe('DOCUMENTS')
  })

  it('writes the tab to the query and follows the query', async () => {
    const { activeTab } = useExpensesTabs(() => badges)
    activeTab.value = 'DOCUMENTS'
    await nextTick()
    expect(replace).toHaveBeenCalledWith({ query: { tab: 'DOCUMENTS' } })
    route.query = { tab: 'FIXED' }
    await nextTick()
    expect(activeTab.value).toBe('FIXED')
  })

  it('shows badges only when there is something to count', () => {
    route.meta = { section: 'maintenance' }
    const { tabs } = useExpensesTabs(() => badges)
    expect(tabs.value[1].badge).toBeUndefined()
    badges.urgentReminders = 2
    expect(tabs.value[1]).toMatchObject({ badge: 2, badgeTone: 'warning' })
    badges.overdueReminders = 1
    expect(tabs.value[1].badgeTone).toBe('danger')
  })
})
