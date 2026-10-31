import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import TireSetSummary from './TireSetSummary.vue'

const stat = (pct: number, condition = 'GOOD') => ({ tire: { brand: 'Michelin', model: 'Pilot' }, life_progress_pct: pct, condition, replacement_forecast: null })
const render = (mounted: Record<string, any>) => renderToString(createSSRApp({ render: () => h(TireSetSummary as any, { mounted }) }).use(i18n))
const full = { FL: stat(20), FR: stat(20), RL: stat(20), RR: stat(20) }

describe('TireSetSummary', () => {
  it('tints each figure with its verdict: all wheels fitted and a tire in good shape read as good', async () => {
    const html = await render(full)
    expect(html).toContain('border-success-500/30')
    expect(html).not.toContain('border-warning-500/30')
    expect(html).not.toContain('border-danger-500/30')
  })

  it('flags a missing wheel and a tire worn past 80 % of its lifespan', async () => {
    const html = await render({ ...full, RR: null, FL: stat(85) })
    expect(html).toContain('border-warning-500/30')
    expect(html).toContain('border-danger-500/30')
  })

  it('leaves the figures plain with no tire fitted', async () => {
    const html = await render({ FL: null, FR: null, RL: null, RR: null })
    expect(html).not.toContain('-500/30')
  })
})
