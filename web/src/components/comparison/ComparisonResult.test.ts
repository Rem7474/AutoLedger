import { describe, expect, it, vi } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import ComparisonResult from './ComparisonResult.vue'

vi.mock('@/components/comparison/ComparisonCharts.vue', () => ({ default: { render: () => null } }))
vi.mock('@/components/comparison/ComparisonWhatIf.vue', () => ({ default: { render: () => null } }))

const cost = { energy: 100, maintenance: 50, insurance: 40, tax: 10, depreciation: 300, total: 500, per_month: 8, cost_per_km: 0.2 }
const result = {
  mode: 'PROJECTION',
  years_count: 5,
  annual_km: 12000,
  tracked: cost,
  ice: { ...cost, total: 700 },
  tracked_savings: 200,
  tracked_depreciation_known: true,
  cumulative: [],
  assumptions: [],
}
const render = (res: any) =>
  renderToString(createSSRApp({ render: () => h(ComparisonResult as any, { result: res, scenario: { id: 's1', name: 'Test' }, side: 'ev', currency: 'EUR' }) }).use(i18n))

describe('ComparisonResult', () => {
  it('offers only the way back while the result is computed', async () => {
    const html = await render(null)
    expect(html).toContain('Mes comparatifs')
    expect(html).not.toContain('Modifier')
    expect(html).not.toContain('Supprimer')
  })

  it('has Edit in the header and a quiet Delete at the bottom', async () => {
    const html = await render(result)
    expect(html.indexOf('Modifier')).toBeGreaterThan(-1)
    expect(html.indexOf('Modifier')).toBeLessThan(html.indexOf('CSV'))
    expect(html.indexOf('Supprimer')).toBeGreaterThan(html.indexOf('Hypothèses'))
    expect(html).toContain('text-critical-400')
    expect(html).not.toContain('Modifier les hypothèses')
  })
})
