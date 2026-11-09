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
    expect(keys(useExpensesTabs(() => badges).tabs.value)).toEqual(['DOCUMENTS'])
    route.meta = { section: 'maintenance' }
    expect(keys(useExpensesTabs(() => badges).tabs.value)).toEqual(['REMINDERS'])
    route.meta = { section: 'energy' }
    expect(keys(useExpensesTabs(() => badges).tabs.value)).toEqual(['CHARGES', 'EFFICIENCY', 'ESTIMATE'])
    store.canRefuel = true
    expect(keys(useExpensesTabs(() => badges).tabs.value)).toEqual(['CHARGES', 'EFFICIENCY', 'FUEL'])
    store.canCharge = false
    expect(keys(useExpensesTabs(() => badges).tabs.value)).toEqual(['FUEL'])
  })

  it('opens the energy page on the charges of an electric or hybrid vehicle, and on the fill-ups of a combustion one', () => {
    route.meta = { section: 'energy' }
    expect(useExpensesTabs(() => badges).activeTab.value).toBe('CHARGES')
    route.query = { tab: 'efficiency' }
    expect(useExpensesTabs(() => badges).activeTab.value).toBe('EFFICIENCY')
    route.query = {}
    store.canRefuel = true
    expect(useExpensesTabs(() => badges).activeTab.value).toBe('CHARGES')
    store.canCharge = false
    expect(useExpensesTabs(() => badges).activeTab.value).toBe('FUEL')
  })

  it('opens the tab of the query, falling back to the first tab of the section', () => {
    route.meta = { section: 'energy' }
    route.query = { tab: 'estimate' }
    expect(useExpensesTabs(() => badges).activeTab.value).toBe('ESTIMATE')
    route.meta = {}
    route.query = { tab: 'REMINDERS' }
    expect(useExpensesTabs(() => badges).activeTab.value).toBe('DOCUMENTS')
    route.query = { tab: ['documents', 'x'] }
    expect(useExpensesTabs(() => badges).activeTab.value).toBe('DOCUMENTS')
  })

  it('writes the tab to the query and follows the query', async () => {
    route.meta = { section: 'energy' }
    const { activeTab } = useExpensesTabs(() => badges)
    activeTab.value = 'EFFICIENCY'
    await nextTick()
    expect(replace).toHaveBeenCalledWith({ query: { tab: 'EFFICIENCY' } })
    route.query = { tab: 'estimate' }
    await nextTick()
    expect(activeTab.value).toBe('ESTIMATE')
  })

  it('shows badges only when there is something to count', () => {
    route.meta = { section: 'maintenance' }
    const { tabs } = useExpensesTabs(() => badges)
    expect(tabs.value[0].badge).toBeUndefined()
    badges.urgentReminders = 2
    expect(tabs.value[0]).toMatchObject({ badge: 2, badgeTone: 'warning' })
    badges.overdueReminders = 1
    expect(tabs.value[0].badgeTone).toBe('danger')
  })
})
