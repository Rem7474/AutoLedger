import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, h, reactive } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import TripGroupsPanel from './TripGroupsPanel.vue'

const vehicleStore = reactive<{ canEdit: boolean; currency: string }>({ canEdit: true, currency: 'EUR' })
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => vehicleStore }))

const trip = { id: 'g1', name: 'Summer holiday', distance_km: 1200, drive_ids: ['a', 'b'], tolls_total: '31.5', carpool_count: 1, notes: '', start_date: '2026-07-01', end_date: '2026-07-10' }
const render = () =>
  renderToString(
    createSSRApp({ render: () => h(TripGroupsPanel as any, { loadingTrips: false, tripGroups: [trip], hasFilters: false, expandedTripId: null, tripDrives: [] }) }).use(i18n),
  )

afterEach(() => {
  vehicleStore.canEdit = true
})

describe('TripGroupsPanel', () => {
  it('opens a trip from the whole row and keeps the other actions in a menu', async () => {
    const html = await render()
    expect(html).toContain('inset-0')
    expect(html).toContain('aria-haspopup="menu"')
    expect(html).not.toContain('lucide-pencil')
  })

  it('keeps the legs toggle above the open button so it still works', async () => {
    const html = await render()
    const toggle = html.slice(html.indexOf('aria-expanded="false"') - 200, html.indexOf('aria-expanded="false"'))
    expect(toggle).toContain('relative z-10')
  })

  it('has no menu on a read-only list', async () => {
    vehicleStore.canEdit = false
    expect(await render()).not.toContain('aria-haspopup="menu"')
  })
})
