import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, h, reactive } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import DriveLegRow from './DriveLegRow.vue'

const vehicleStore = reactive<{ canEdit: boolean; currency: string; capabilities: { sync: boolean } }>({ canEdit: true, currency: 'EUR', capabilities: { sync: true } })
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => vehicleStore }))
const prefs = reactive({ proPersoEnabled: false })
vi.mock('@/stores/preferences', () => ({ usePreferencesStore: () => prefs }))

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
  vehicleStore.capabilities = { sync: true }
})

describe('DriveLegRow', () => {
  it('is a compact card opened as a whole, with the route cut to its first segment and the cost beside it', async () => {
    const html = await render(leg)
    expect(html).toContain('data-testid="drive-leg"')
    expect(html).toContain('inset-0')
    expect(html).toContain('card relative flex items-center px-3 gap-2.5 py-1.5')
    expect(html).not.toContain('py-2.5')
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
    vehicleStore.capabilities = { sync: false }
    expect(await render({ ...leg, is_manual: true })).not.toContain('Manuel')
  })

  it('builds its quiet line from groups, with no separator written in the markup', async () => {
    const html = await render(leg)
    expect(html).toContain('flex-col')
    expect(html).not.toContain('> · <')
  })

  it('carries the glyph of the drives list, smaller and at every width', async () => {
    const html = await render(leg)
    expect(html).toContain('size-8')
    expect(html).not.toContain('sm:hidden')
    // a carpooled leg takes the carpool glyph and a work leg the work glyph, like the drive row
    const plain = html.match(/<svg[\s\S]*?<\/svg>/)?.[0]
    const carpooled = (await render({ ...leg, carpools: [{ id: 'c1', passenger_count: 2, total_revenue: 20 }] })).match(/<svg[\s\S]*?<\/svg>/)?.[0]
    expect(carpooled).not.toBe(plain)
    prefs.proPersoEnabled = true
    const pro = (await render({ ...leg, tags: ['Pro'] })).match(/<svg[\s\S]*?<\/svg>/)?.[0]
    prefs.proPersoEnabled = false
    expect(pro).not.toBe(plain)
  })
})
