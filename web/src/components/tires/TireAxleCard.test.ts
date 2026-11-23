import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia } from 'pinia'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import TireAxleCard from './TireAxleCard.vue'

const stat = (condition = 'GOOD', logs = 0) => ({
  tire: { id: 't', brand: 'Brand', model: 'Model', dimension: '205/55 R16', purchase_price: '100' },
  total_distance_km: 8000,
  cost_per_km: 0.02,
  estimated_lifespan_km: 40000,
  life_progress_pct: 20,
  condition,
  logs_count: logs,
  replacement_forecast: null,
})
const wheels = [{ pos: 'FL', label: 'Front left' }, { pos: 'FR', label: 'Front right' }]
const render = (props: Record<string, unknown>) =>
  renderToString(createSSRApp({ render: () => h(TireAxleCard as any, { title: 'Front axle', wheels, selectedIds: [], ...props }) }).use(createPinia()).use(i18n))

beforeEach(() => {
  vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => undefined })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('TireAxleCard', () => {
  it('shows the two wheels of the axle under its title, without a condition badge for a good tire', async () => {
    const html = await render({ mounted: { FL: stat(), FR: stat() } })
    expect(html).toContain('Front axle')
    expect(html).toContain('Front left')
    expect(html).toContain('Front right')
    expect(html).not.toContain('Bon état')
    expect(html).not.toContain('Aucune mesure')
  })

  it('names the tire only when the set is not uniform, and writes a condition that is not good', async () => {
    const quiet = await render({ mounted: { FL: stat(), FR: stat() } })
    expect(quiet).not.toContain('205/55 R16')
    const mixed = await render({ mounted: { FL: stat('CRITICAL'), FR: stat() }, showModel: true })
    expect(mixed).toContain('205/55 R16')
    expect(mixed).toContain('Usure critique')
  })

  it('keeps an empty wheel as a placeholder', async () => {
    const html = await render({ mounted: { FL: stat(), FR: null } })
    expect(html).toContain('border-dashed')
  })
})
