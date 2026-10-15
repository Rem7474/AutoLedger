import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp } from 'vue'
import { createPinia } from 'pinia'
import { renderToString } from '@vue/server-renderer'
import { i18n, setLocale } from '@/i18n'
import TcoMetricsGrid from './TcoMetricsGrid.vue'

const months = [
  { month: '2026-08', distance_km: 400, total: 100, energy: 60 },
  { month: '2026-09', distance_km: 600, total: 200, energy: 140 },
]
const render = (tco: Record<string, unknown>) =>
  renderToString(createSSRApp(TcoMetricsGrid, { tco }).use(createPinia()).use(i18n))

beforeEach(() => {
  vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => undefined })
  setLocale('en')
})

afterEach(() => {
  vi.unstubAllGlobals()
  setLocale('fr')
})

describe('tco metrics grid', () => {
  it('gives the energy card the window of the cards beside it', async () => {
    const html = await render({ powertrain: 'ELECTRIC', monthly_costs: months, energy_cost: 9999, energy_cost_per_km: 9 })
    // Window total (60 + 140), not the since-purchase 9 999
    expect(html).toContain('200')
    expect(html).not.toContain('9,999')
    expect(html).not.toContain('9999')
    expect(html.match(/last 2 months/g)?.length).toBeGreaterThanOrEqual(3)
  })

  it('falls back to the since-purchase energy, without a period, when the monthly rows are missing', async () => {
    const html = await render({ powertrain: 'ELECTRIC', monthly_costs: [], energy_cost: 321, energy_cost_per_km: 0.123 })
    expect(html).toContain('321')
    expect(html).not.toContain('last 2 months')
  })

  it('names the full cost since purchase under the economic cost', async () => {
    const html = await render({ powertrain: 'ELECTRIC', monthly_costs: months, full_cost_per_km: 0.5, depreciation_known: true })
    expect(html).toContain('Full cost since purchase')
  })
})
