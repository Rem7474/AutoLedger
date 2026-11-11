import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, h, reactive } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import DriveLegRow from './DriveLegRow.vue'

const vehicleStore = reactive<{ canEdit: boolean; currency: string; activeVehicle: any }>({ canEdit: true, currency: 'EUR', activeVehicle: { powertrain: 'EV', teslamate_api_url: 'http://tm' } })
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => vehicleStore }))

const leg = {
  id: 'a',
  start_time: '2026-08-02T07:30:00Z',
  start_address: 'Zone industrielle des Grandes Terres Nord, 69800 Saint-Priest, France',
  end_address: 'Aeroport de Geneve, 1215 Le Grand-Saconnex, Suisse',
  distance_km: 412.5,
  costs: { total_cost: 61.2, cost_per_km: 0.148 },
}
const render = (l: any) => renderToString(createSSRApp({ render: () => h(DriveLegRow as any, { leg: l }) }).use(i18n))

afterEach(() => {
  vehicleStore.activeVehicle = { powertrain: 'EV', teslamate_api_url: 'http://tm' }
})

describe('DriveLegRow', () => {
  it('is one flat row opened as a whole, with the route cut to its first segment and the cost beside it', async () => {
    const html = await render(leg)
    expect(html).toContain('data-testid="drive-leg"')
    expect(html).toContain('inset-0')
    expect(html).not.toContain('card py-2.5')
    expect(html).not.toContain('aria-haspopup')
    expect(html).toContain('Zone industrielle des Grandes Terres Nord')
    expect(html).not.toContain('>Zone industrielle des Grandes Terres Nord, 69800')
    expect(html).toContain('title="Zone industrielle des Grandes Terres Nord, 69800 Saint-Priest, France → Aeroport de Geneve, 1215 Le Grand-Saconnex, Suisse"')
    expect(html).toContain('line-clamp-2')
    expect(html).toContain('412,5')
    expect(html).toContain('61,20')
  })

  it('shows the empty placeholder, not a zero, for a leg whose cost is unknown', async () => {
    const html = await render({ ...leg, costs: null })
    expect(html).toContain('>—<')
    expect(html).not.toContain('0,00')
  })

  it('marks a carpooled leg with a state text and a manual leg of a TeslaMate vehicle', async () => {
    const html = await render({ ...leg, is_manual: true, carpools: [{ id: 'c1', passenger_count: 2, total_revenue: 20 }] })
    expect(html).toContain('data-tone="info"')
    expect(html).toContain('Covoituré')
    expect(html).toContain('Manuel')
    vehicleStore.activeVehicle = { powertrain: 'ICE' }
    expect(await render({ ...leg, is_manual: true })).not.toContain('Manuel')
  })

  it('builds its quiet line from groups, with no separator written in the markup', async () => {
    const html = await render(leg)
    expect(html).toContain('flex-col')
    expect(html).not.toContain('> · <')
  })
})
