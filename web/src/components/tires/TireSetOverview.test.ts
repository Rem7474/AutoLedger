import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createPinia } from 'pinia'
import { i18n } from '@/i18n'
import TireSetOverview from './TireSetOverview.vue'

const stat = (pct: number) => ({ tire: { id: `t${pct}`, brand: 'Brand', model: 'Model', dimension: '205/55R16' }, life_progress_pct: pct, condition: 'GOOD', replacement_forecast: null })
const wheels = [
  { pos: 'FL', labelKey: 'tires.positions.FL' },
  { pos: 'FR', labelKey: 'tires.positions.FR' },
]
const render = (mounted: Record<string, any>) =>
  renderToString(createSSRApp({ render: () => h(TireSetOverview as any, { mounted, wheels, selectedIds: [] }) }).use(createPinia()).use(i18n))
const plain = (html: string) => html.replace(/ | /g, ' ')

beforeEach(() => {
  vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => undefined })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('TireSetOverview', () => {
  it('says the wear and what is left in words, on the set card and on each wheel', async () => {
    const html = plain(await render({ FL: stat(21), FR: stat(18) }))
    expect(html).toContain('usé à 21 % · Avant gauche')
    expect(html).toContain('reste 79 %')
    expect(html).toContain('usé à 18 %')
    expect(html).toContain('reste 82 %')
    expect(html).not.toContain('Pneu le plus usé · 20')
  })

  it('draws the gauge of what is left', async () => {
    const html = await render({ FL: stat(21), FR: stat(18) })
    expect(html).toContain('width:79%')
    expect(html).toContain('width:82%')
  })
})
