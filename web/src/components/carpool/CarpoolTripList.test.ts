import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, h, reactive } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import CarpoolTripList from './CarpoolTripList.vue'

const vehicleStore = reactive<{ canEdit: boolean; currency: string }>({ canEdit: true, currency: 'EUR' })
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => vehicleStore }))

const trip = { id: 't1', title: 'Lyon to Nice', date: '2026-10-10T08:00:00Z', distance_km: 470, legs: [{}, {}], passengers: [{}], total_cost: 100, total_revenue: 60, net_cost: 40, notes: '' }
const render = (trips: any[] = [trip]) =>
  renderToString(createSSRApp({ render: () => h(CarpoolTripList as any, { trips, selectedTripIds: [], recalculating: false }) }).use(i18n))

afterEach(() => {
  vehicleStore.canEdit = true
})

describe('CarpoolTripList', () => {
  it('opens a carpool from the whole row, with the other actions in a menu and no icon row', async () => {
    const html = await render()
    expect(html).toContain('inset-0')
    expect(html).toContain('aria-haspopup="menu"')
    expect(html).not.toContain('lucide-edit')
  })

  it('shows what is left to the driver, or the surplus once the cost is recovered', async () => {
    expect(await render()).toContain('40,00')
    const recovered = await render([{ ...trip, net_cost: -5 }])
    expect(recovered).toContain('+5,00')
    expect(recovered).toContain('text-success-400')
  })

  it('leaves the selection box and the menu out of a read-only list', async () => {
    vehicleStore.canEdit = false
    const html = await render()
    expect(html).not.toContain('carpool-select-')
    expect(html).not.toContain('aria-haspopup="menu"')
  })
})
