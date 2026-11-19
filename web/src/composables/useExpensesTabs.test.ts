import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, reactive } from 'vue'
import { useExpensesTabs } from './useExpensesTabs'

const route = reactive<{ query: Record<string, unknown>; meta: Record<string, unknown> }>({ query: {}, meta: {} })
const replace = vi.fn()
vi.mock('vue-router', () => ({ useRoute: () => route, useRouter: () => ({ replace }) }))
const store = reactive({ capabilities: { charges: true, refuels: false } })
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => store }))

const keys = (tabs: { key: string }[]) => tabs.map((x) => x.key)

describe('useExpensesTabs', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    route.query = {}
    route.meta = {}
    store.capabilities.charges = true
    store.capabilities.refuels = false
  })

  it('lists the tabs of each section and of each powertrain', () => {
    expect(keys(useExpensesTabs().tabs.value)).toEqual(['DOCUMENTS'])
    route.meta = { section: 'maintenance' }
    expect(keys(useExpensesTabs().tabs.value)).toEqual(['REMINDERS'])
    route.meta = { section: 'consumption' }
    expect(keys(useExpensesTabs().tabs.value)).toEqual(['EFFICIENCY', 'ESTIMATE', 'CHARGES'])
    store.capabilities.refuels = true
    expect(keys(useExpensesTabs().tabs.value)).toEqual(['EFFICIENCY', 'FUEL', 'CHARGES'])
    store.capabilities.charges = false
    expect(keys(useExpensesTabs().tabs.value)).toEqual(['FUEL'])
  })

  it('opens the consumption page on the efficiency of an electric or hybrid vehicle, and on the fill-ups of a combustion one', () => {
    route.meta = { section: 'consumption' }
    expect(useExpensesTabs().activeTab.value).toBe('EFFICIENCY')
    route.query = { tab: 'charges' }
    expect(useExpensesTabs().activeTab.value).toBe('CHARGES')
    route.query = {}
    store.capabilities.refuels = true
    expect(useExpensesTabs().activeTab.value).toBe('EFFICIENCY')
    store.capabilities.charges = false
    expect(useExpensesTabs().activeTab.value).toBe('FUEL')
  })

  it('opens the tab of the query, falling back to the first tab of the section', () => {
    route.meta = { section: 'consumption' }
    route.query = { tab: 'estimate' }
    expect(useExpensesTabs().activeTab.value).toBe('ESTIMATE')
    route.meta = {}
    route.query = { tab: 'REMINDERS' }
    expect(useExpensesTabs().activeTab.value).toBe('DOCUMENTS')
    route.query = { tab: ['documents', 'x'] }
    expect(useExpensesTabs().activeTab.value).toBe('DOCUMENTS')
  })

  it('writes the tab to the query and follows the query', async () => {
    route.meta = { section: 'consumption' }
    const { activeTab } = useExpensesTabs()
    activeTab.value = 'EFFICIENCY'
    await nextTick()
    expect(replace).toHaveBeenCalledWith({ query: { tab: 'EFFICIENCY' } })
    route.query = { tab: 'estimate' }
    await nextTick()
    expect(activeTab.value).toBe('ESTIMATE')
  })

  it('gives a tab a label and an icon only, never a counter', () => {
    route.meta = { section: 'consumption' }
    for (const tab of useExpensesTabs().tabs.value) expect(Object.keys(tab).sort()).toEqual(['icon', 'key', 'label'])
  })
})
