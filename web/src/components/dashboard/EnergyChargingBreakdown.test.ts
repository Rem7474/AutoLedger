import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import type { BreakdownRow } from '@/utils/chargingBreakdown'
import EnergyChargingBreakdown from './EnergyChargingBreakdown.vue'

const row = (key: string, label: string, kwh: number, sharePct: number, pricePerKwh: number | null, unknownSessions = 0): BreakdownRow => ({
  key, label, color: '#38bdf8', kwh, sharePct, pricePerKwh, unknownSessions,
})
const render = (props: Record<string, unknown>) =>
  renderToString(createSSRApp({ render: () => h(EnergyChargingBreakdown as any, { currency: 'EUR', ...props }) }).use(i18n))

describe('EnergyChargingBreakdown', () => {
  it('offers the place and speed toggle when both splits have energy, place first', async () => {
    const html = await render({ place: [row('HOME', 'Home', 100, 100, 0.2)], speed: [row('AC', 'Wallbox', 100, 100, 0.2)] })
    expect(html).toContain('role="radiogroup"')
    expect(html).toContain('Lieu')
    expect(html).toContain('Vitesse')
    expect(html).toContain('Home')
    expect(html).not.toContain('Wallbox')
  })

  it('draws the figures in the legend, not in separate cards', async () => {
    const html = await render({ place: [row('HOME', 'Domicile', 75, 75, 0.2), row('', 'Autre', 25, 25, null)], speed: [] })
    expect(html).toContain('75 %')
    expect(html).toContain('75 kWh')
    expect(html).toContain('25 %')
    expect(html).not.toContain('bg-slate-800')
  })

  it('has no toggle when only one split has energy and says nothing when none has', async () => {
    const onlySpeed = await render({ place: [], speed: [row('DC', 'Rapide', 10, 100, 0.5)] })
    expect(onlySpeed).not.toContain('role="radiogroup"')
    expect(onlySpeed).toContain('Rapide')
    expect(await render({ place: [], speed: [] })).not.toContain('charging-breakdown')
  })

  it('notes the sessions left out of the price, and those of unknown duration on the speed split', async () => {
    expect(await render({ place: [row('HOME', 'Domicile', 10, 100, 0.2, 2)], speed: [] })).toContain('2 recharge(s) sans coût')
    expect(await render({ place: [], speed: [row('AC', 'Lente', 10, 100, 0.2)], unknownSpeedSessions: 3 })).toContain('3')
  })
})
