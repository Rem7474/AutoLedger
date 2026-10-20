import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n, setLocale } from '@/i18n'
import { allocate, emptyLeg, newPassenger } from '@/utils/carpool'
import CarpoolOverview from './CarpoolOverview.vue'

const legs = [emptyLeg(), emptyLeg(), emptyLeg()]
legs.forEach((leg, i) => { leg.electricity_cost = 6 * (i + 1) })
const stops = ['A', 'B', 'C', 'D']
const passengers = [
  { ...newPassenger(0, 3), passenger_name: 'Lea', board_stop_index: 0, alight_stop_index: 2 },
  { ...newPassenger(1, 3), passenger_name: 'Tom', seats: 2, board_stop_index: 1, alight_stop_index: 3 },
]
const render = (selected = 0) =>
  renderToString(createSSRApp(CarpoolOverview, { passengers, stops, currency: 'EUR', live: allocate(legs, passengers), selected }).use(i18n))

beforeEach(() => {
  vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => undefined })
})

afterEach(() => {
  vi.unstubAllGlobals()
  setLocale('fr')
})

describe('carpool overview', () => {
  it('places each passenger bar on the legs they ride', async () => {
    const html = await render()
    expect(html).toContain('grid-column:1 / 3')
    expect(html).toContain('grid-column:2 / 4')
  })

  it('counts the people on board on each leg, driver included', async () => {
    const html = await render()
    expect(html).toContain('2 personnes')
    expect(html).toContain('4 personnes')
    expect(html).toContain('3 personnes')
  })

  it('marks the selected passenger', async () => {
    const html = await render(1)
    expect(html.match(/aria-pressed="true"/g)).toHaveLength(1)
    expect(html).toMatch(/aria-pressed="true"[^>]*title="Tom"/)
  })
})
