import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, h, reactive } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import TripGroupsPanel from './TripGroupsPanel.vue'

const vehicleStore = reactive<{ canEdit: boolean; currency: string }>({ canEdit: true, currency: 'EUR' })
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => vehicleStore }))

const trip = { id: 'g1', name: 'Summer holiday', distance_km: 1200, drive_ids: ['a', 'b'], tolls_total: '31.5', carpool_count: 1, notes: '', start_date: '2026-07-01', end_date: '2026-07-10' }
const legs = [
  { id: 'a', start_time: '2026-07-01T08:00:00Z', start_address: 'Rue de la Paix, 75002 Paris, France', end_address: 'Lyon, France', distance_km: 465, costs: { total_cost: 30.5, cost_per_km: 0.066 } },
  { id: 'b', start_time: '2026-07-02T08:00:00Z', start_address: 'Lyon', end_address: '', distance_km: 12, is_manual: true, costs: null },
]
const render = (expanded: string | null = null, tripDrives: any[] = []) =>
  renderToString(
    createSSRApp({ render: () => h(TripGroupsPanel as any, { loadingTrips: false, tripGroups: [trip], hasFilters: false, expandedTripId: expanded, tripDrives }) }).use(i18n),
  )

afterEach(() => {
  vehicleStore.canEdit = true
})

describe('TripGroupsPanel', () => {
  it('opens a trip from the whole row, with no row menu', async () => {
    const html = await render()
    expect(html).toContain('inset-0')
    expect(html).not.toContain('aria-haspopup="menu"')
    expect(html).not.toContain('data-icon="edit"')
  })

  it('keeps the legs toggle above the open button so it still works', async () => {
    const html = await render()
    const toggle = html.slice(html.indexOf('aria-expanded="false"') - 200, html.indexOf('aria-expanded="false"'))
    expect(toggle).toContain('relative z-10')
  })

  it('has no selection box on a read-only list', async () => {
    vehicleStore.canEdit = false
    expect(await render()).not.toContain('id="select-g1"')
  })

  it('lists the legs of an open trip as flat rows that open the leg, with no remove control', async () => {
    const html = await render('g1', legs)
    expect(html).toContain('data-testid="trip-legs"')
    expect(html.match(/data-testid="drive-leg"/g)).toHaveLength(2)
    expect(html).toContain('title="Rue de la Paix, 75002 Paris, France → Lyon, France"')
    expect(html).not.toContain('aria-haspopup="menu"')
    expect(html).not.toContain('Retirer ce trajet du voyage')
    // a leg is a flat row inside the list, not a second card
    const list = html.slice(html.indexOf('data-testid="trip-legs"'))
    expect(list).not.toContain('card py-2.5')
  })

  it('shows the loading line while the legs load', async () => {
    const html = await render('g1', [])
    expect(html).toContain('Chargement des trajets')
    expect(html).not.toContain('data-testid="drive-leg"')
  })
})
