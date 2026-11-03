import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { createPinia, setActivePinia } from 'pinia'
import { i18n } from '@/i18n'
import { useVehicleStore } from '@/stores/vehicle'
import ChargeCard from './ChargeCard.vue'

const vehicle = { id: 'v1', powertrain: 'EV', currency: 'EUR', role: 'OWNER', teslamate_url: 'http://tm.local', teslamate_grafana_url: 'http://g.local' }
const recent = new Date(Date.now() - 3 * 3600e3).toISOString()
const charge = {
  id: 'c1', teslamate_charge_id: 158, origin: 'TESLAMATE', cost_source: 'TESLAMATE', date: recent, end_date: recent,
  address: 'Valence', kwh_added: 8.9, currency: 'EUR', cost: 0, cost_to_review: true,
}

async function render(c: Record<string, unknown>, role = 'OWNER', teslamateUrl: string | null = vehicle.teslamate_url) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const store = useVehicleStore()
  store.vehicles = [{ ...vehicle, role, teslamate_url: teslamateUrl ?? undefined }]
  const app = createSSRApp({ render: () => h(ChargeCard, { c }) })
  app.use(pinia).use(i18n)
  return renderToString(app)
}

describe('ChargeCard', () => {
  beforeEach(() => {
    vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => {}, removeItem: () => {} })
  })

  it('sends a 0 still to confirm to TeslaMate to set the cost, or lets the owner confirm it as free', async () => {
    const html = await render(charge)
    expect(html).toContain('border-warning-500/40')
    expect(html).toContain('Saisir dans TeslaMate')
    expect(html).toContain('href="http://tm.local/charge-cost/158"')
    expect(html).toContain('Gratuite')
    expect(html).not.toContain('Saisir le coût')
    expect(html).toContain('aria-haspopup="menu"')
  })

  it('keeps entering the cost in AutoLedger when TeslaMate cannot give it back', async () => {
    const html = await render(charge, 'OWNER', null) // no TeslaMate URL on the vehicle
    expect(html).toContain('Saisir le coût')
    expect(html).not.toContain('Saisir dans TeslaMate')
    expect(html).not.toContain('charge-cost')
    expect(await render({ ...charge, date: new Date(Date.now() - 40 * 86400e3).toISOString() })).toContain('Saisir le coût')
  })

  it('offers only to enter the cost when there is none at all, and says it is missing', async () => {
    const html = await render({ ...charge, cost: null })
    expect(html).toContain('Coût manquant')
    expect(html).not.toContain('Gratuite')
  })

  it('shows a confirmed free charge as an ordinary one: no queue, no cost link', async () => {
    const html = await render({ ...charge, cost_to_review: false, cost_reviewed_at: recent })
    expect(html).not.toContain('border-warning-500/40')
    expect(html).not.toContain('Gratuite')
    expect(html).not.toContain('charge-cost')
  })

  it('does not offer to complete the cost to a viewer', async () => {
    const html = await render(charge, 'VIEWER')
    expect(html).not.toContain('Gratuite')
  })

  it('keeps the details link and the delete in the menu, and opens the charge from the whole card', async () => {
    const html = await render(charge)
    expect(html).toContain('inset-0')
    expect(html).not.toContain('lucide-pencil')
    expect(html).toContain('aria-haspopup="menu"')
  })

  it('offers the menu on a manual charge, which can be deleted', async () => {
    expect(await render({ ...charge, origin: 'MANUAL', teslamate_charge_id: null, cost: 4, cost_to_review: false })).toContain('aria-haspopup="menu"')
  })

  it('offers neither the open area nor the menu to a viewer', async () => {
    const html = await render(charge, 'VIEWER')
    expect(html).not.toContain('inset-0')
    expect(html).not.toContain('aria-haspopup="menu"')
  })
})
