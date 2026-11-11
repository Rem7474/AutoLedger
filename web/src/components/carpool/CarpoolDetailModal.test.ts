import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, h, reactive } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import CarpoolDetailModal from './CarpoolDetailModal.vue'

const vehicleStore = reactive<{ canEdit: boolean; currency: string }>({ canEdit: true, currency: 'EUR' })
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => vehicleStore }))

const trip = {
  id: 't1',
  title: 'Lyon to Nice',
  date: '2026-10-10T08:00:00Z',
  distance_km: 470,
  legs: [{ id: 'l1', distance_km: 470, total_cost: 100, cost_per_person: 50, passenger_seats: 1 }],
  passengers: [{ id: 'p1', passenger_name: 'Ana', seats: 1, amount_paid: 10, cost_share: 50, balance: -40, board_stop_index: 0, alight_stop_index: 1 }],
  total_cost: 100,
  total_revenue: 10,
  net_cost: 90,
  notes: '',
}
const render = () =>
  renderToString(createSSRApp({ render: () => h(CarpoolDetailModal as any, { open: true, trip, recalculating: false }) }).directive('dialog', {}).use(i18n))

afterEach(() => {
  vehicleStore.canEdit = true
})

describe('CarpoolDetailModal : modifier, recalculer et supprimer', () => {
  it('propose Modifier dans l\'en-tête, Recalculer au pied et Supprimer en bas du détail', async () => {
    const html = await render()
    expect(html.match(/Modifier/g)).toHaveLength(1)
    expect(html).toContain('text-critical-400')
    expect(html).toContain('Supprimer')
    expect(html).toContain('Recalculer')
  })

  it('n\'en propose aucun en lecture seule', async () => {
    vehicleStore.canEdit = false
    const html = await render()
    expect(html).not.toContain('Modifier')
    expect(html).not.toContain('Supprimer')
    expect(html).not.toContain('text-critical-400')
  })
})
