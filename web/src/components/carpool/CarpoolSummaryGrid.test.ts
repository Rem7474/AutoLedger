import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp } from 'vue'
import { createPinia } from 'pinia'
import { renderToString } from '@vue/server-renderer'
import { i18n, setLocale } from '@/i18n'
import { useVehicleStore } from '@/stores/vehicle'
import CarpoolSummaryGrid from './CarpoolSummaryGrid.vue'

const summary = {
  total_trips: 12, total_passengers: 20, total_distance_km: 480, total_revenue: 50, total_passengers_share: 80,
  total_real_cost: 100, total_driver_share: 20, net_cost_per_km: 0.05, coverage_rate_pct: 64.1,
}
const render = (value: Record<string, unknown>) => {
  const pinia = createPinia()
  const app = createSSRApp(CarpoolSummaryGrid, { summary: { ...summary, ...value } }).use(pinia).use(i18n)
  useVehicleStore(pinia).vehicles = [{ id: 'v1', currency: 'EUR' }]
  return renderToString(app)
}

beforeEach(() => {
  vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => undefined })
})

afterEach(() => {
  vi.unstubAllGlobals()
  setLocale('fr')
})

describe('carpool summary grid', () => {
  it('shows the recovery rate in the account language', async () => {
    setLocale('en')
    expect(await render({})).toContain('64.1')
  })

  it('draws no recovery rate when there is none, rather than 0 %', async () => {
    setLocale('en')
    const html = await render({ coverage_rate_pct: null })
    expect(html).not.toContain('Cost recovery rate')
    expect(html).not.toContain('0 %')
    expect(await render({})).toContain('Cost recovery rate')
  })

  it('says how much is missing while the revenue is under the fair share', async () => {
    setLocale('en')
    const html = await render({})
    expect(html).toContain('text-warning-400')
    expect(await render({ total_revenue: 80 })).toContain('text-success-400')
  })

  it('names the period of the totals once and draws no cost per distance when no distance is shared', async () => {
    setLocale('en')
    const html = await render({})
    expect((html.match(/All trips/g) ?? []).length).toBe(1)
    expect(html).toContain('€0.05')
    const none = await render({ total_distance_km: 0, net_cost_per_km: 0 })
    expect(none).not.toContain('€0.0')
    expect(none).not.toContain('Driver&#39;s net cost')
  })

  it('tints the received total with its verdict, only when there are passengers\' shares to cover', async () => {
    expect(await render({})).toContain('border-warning-500/30')
    const covered = await render({ total_revenue: 80 })
    expect(covered).toContain('border-success-500/30')
    expect(covered).not.toContain('border-warning-500/30')
    expect(await render({ total_passengers_share: 0, total_revenue: 0 })).not.toMatch(/border-(success|warning)-500\/30/)
  })

  it('colours the recovery rate only once the whole cost is recovered', async () => {
    const figure = /text-success-400 mt-0.5 text-xl/
    expect(await render({})).not.toMatch(figure)
    expect(await render({ coverage_rate_pct: 100 })).toMatch(figure)
    expect(await render({ coverage_rate_pct: null })).not.toMatch(figure)
  })

  it('names the period the totals cover, and all trips when none is given', async () => {
    const pinia = createPinia()
    const app = createSSRApp(CarpoolSummaryGrid, { summary, periodLabel: 'Novembre 2026' }).use(pinia).use(i18n)
    useVehicleStore(pinia).vehicles = [{ id: 'v1', currency: 'EUR' }]
    const html = await renderToString(app)
    expect(html).toContain('Novembre 2026')
    expect(html).not.toContain('Tous les trajets')
    expect(await render({})).toContain('Tous les trajets')
  })
})
