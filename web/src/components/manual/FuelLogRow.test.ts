import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import FuelLogRow from './FuelLogRow.vue'

const log = {
  id: 'f1',
  date: '2026-03-10T12:00:00',
  odometer: 45210,
  amount: 40,
  liters: 25,
  price_per_liter: 1.6,
  is_full_tank: true,
  consumption_l_100km: 6.4,
  cost_per_km: 0.11,
}
const render = (props: Record<string, unknown> = {}) =>
  renderToString(createSSRApp({ render: () => h(FuelLogRow as any, { log, currency: 'EUR', selected: false, canEdit: true, ...props }) }).use(i18n))

describe('FuelLogRow', () => {
  it('ouvre le plein par toute la ligne, sans menu de ligne', async () => {
    const html = await render()
    expect(html).toContain('card-open')
    expect(html).toContain('Modifier le plein du')
    expect(html).not.toContain('aria-haspopup="menu"')
  })

  it("montre le montant en valeur, les litres et le prix au litre sur la ligne discrète, le coût au km en sous-valeur", async () => {
    const html = await render()
    expect(html).toContain('40')
    expect(html).toContain('25')
    expect(html).toContain('0,110')
  })

  it("écrit un plein partiel comme un état, sans pastille", async () => {
    const html = await render({ log: { ...log, is_full_tank: false } })
    expect(html).toContain('partiel')
    expect(html).toContain('data-tone="neutral"')
    expect(html).not.toContain('rounded-full border')
  })

  it("n'ouvre rien en lecture seule", async () => {
    const html = await render({ canEdit: false })
    expect(html).not.toContain('card-open')
    expect(html).not.toContain('select-f1')
  })
})
