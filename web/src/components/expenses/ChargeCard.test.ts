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

async function render(c: Record<string, unknown>, role = 'OWNER', teslamateUrl: string | null = vehicle.teslamate_url, selected = false) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const store = useVehicleStore()
  store.vehicles = [{ ...vehicle, role, teslamate_url: teslamateUrl ?? undefined }]
  const app = createSSRApp({ render: () => h(ChargeCard, { c, selected }) })
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
    expect(html).not.toContain('aria-haspopup="menu"')
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

  it('opens the charge from the whole card and has no row menu, whatever the charge', async () => {
    for (const c of [charge, { ...charge, origin: 'MANUAL', teslamate_charge_id: null, cost: 4, cost_to_review: false }]) {
      const html = await render(c)
      expect(html).toContain('inset-0')
      expect(html).not.toContain('aria-haspopup="menu"')
      expect(html).not.toContain('Détails dans TeslaMate')
    }
  })

  it('writes the states as text (StatusLabel), not as pills', async () => {
    const html = await render({ ...charge, cost: null })
    expect(html).toMatch(/data-tone="attention"[^>]*>[\s\S]*?Coût manquant/)
    expect(html).not.toContain('bg-warning-500/10 border border-warning-500/30')
    const corrected = await render({ ...charge, cost: 5, cost_to_review: false, cost_source: 'MANUAL' })
    expect(corrected).toMatch(/data-tone="info"[^>]*>[\s\S]*?Coût corrigé/)
  })

  it('does not say where the charge comes from on the row', async () => {
    for (const origin of ['TESLAMATE', 'MANUAL', 'CSV', 'WEBHOOK']) {
      const html = await render({ ...charge, origin, cost: 4, cost_to_review: false })
      expect(html).not.toContain('Home Assistant / script')
      expect(html).not.toContain('Import CSV')
      expect(html).not.toContain('Manuelle')
    }
  })

  it('shows the energy as plain text, not as a pill', async () => {
    expect(await render({ ...charge, cost: 4, cost_to_review: false })).not.toContain('rounded-full border border-slate-700/60 bg-slate-800')
  })

  it('has a selection box on the row, checked and highlighted once the charge is selected', async () => {
    const html = await render(charge)
    expect(html).toContain('charge-select-c1')
    expect(html).not.toContain('border-rose-500/40')
    const selected = await render(charge, 'OWNER', vehicle.teslamate_url, true)
    expect(selected).toContain('border-rose-500/40')
    expect(selected).toContain('checked')
    // the glyph of the phone row becomes the check
    expect(selected).toContain('bg-rose-600')
    expect(html).toContain('bg-warning-500/15')
  })

  it('leaves the selection box out of a read-only list', async () => {
    expect(await render(charge, 'VIEWER')).not.toContain('charge-select-')
  })

  it('offers no open area to a viewer', async () => {
    const html = await render(charge, 'VIEWER')
    expect(html).not.toContain('inset-0')
    expect(html).not.toContain('aria-haspopup="menu"')
  })
})
