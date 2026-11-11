import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, defineComponent, h, reactive } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import ChargeModal from './ChargeModal.vue'

const vehicleStore = reactive<{ canEdit: boolean; currency: string; activeVehicle: any }>({
  canEdit: true,
  currency: 'EUR',
  activeVehicle: { id: 'v1', powertrain: 'EV', currency: 'EUR', teslamate_url: 'http://tm.local', teslamate_grafana_url: 'http://g.local' },
})
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => vehicleStore }))
vi.mock('@/services/api', () => ({ api: { getChargingPlaces: vi.fn(async () => ({ places: [] })) } }))
// The shell teleports to the body: render its slots in place
vi.mock('@/components/ModalShell.vue', () => ({
  default: defineComponent({
    render() {
      return h('div', [this.$slots.title?.(), this.$slots.default?.(), this.$slots.footer?.()])
    },
  }),
}))

const base = { id: 'c1', date: '2026-10-29T17:17:00Z', kwh_added: 22.5, cost: 6.3, currency: 'EUR', address: 'Home', version: 1 }
const render = (editing: any) =>
  renderToString(createSSRApp({ render: () => h(ChargeModal as any, { open: true, vehicleId: 'v1', editing, documents: [], currentOdometer: 0 }) }).use(i18n))

afterEach(() => {
  vehicleStore.canEdit = true
})

describe('ChargeModal origin', () => {
  it('says in one sentence that a synchronised charge is read-only, with the TeslaMate link inside it', async () => {
    const html = await render({ ...base, origin: 'TESLAMATE', teslamate_charge_id: 158, cost_source: 'MANUAL' })
    expect(html).toContain('Synchronisée depuis')
    expect(html).toContain('lecture seule')
    expect(html).toContain('href="http://g.local/d/BHhxFeZRz/charge-details')
  })
  it('says a manual charge was entered by hand', async () => {
    expect(await render({ ...base, origin: 'MANUAL' })).toContain('Saisie manuellement.')
  })
  it('says a CSV charge was imported from a file', async () => {
    expect(await render({ ...base, origin: 'CSV' })).toContain('Importée depuis un fichier CSV.')
  })
  it('groups webhooks, scripts and Home Assistant as an automatic import', async () => {
    expect(await render({ ...base, origin: 'WEBHOOK' })).toContain('Importée automatiquement.')
  })
  it('does not say anything about the origin of a new charge', async () => {
    expect(await render(null)).not.toContain('data-testid="charge-origin"')
  })
})

describe('ChargeModal delete', () => {
  it('offers Supprimer for a charge that can be deleted', async () => {
    expect(await render({ ...base, origin: 'MANUAL' })).toContain('Supprimer')
    expect(await render({ ...base, origin: 'CSV' })).toContain('Supprimer')
  })
  it('hides it for a synchronised charge, which would come back', async () => {
    expect(await render({ ...base, origin: 'TESLAMATE', teslamate_charge_id: 158 })).not.toContain('Supprimer')
  })
  it('hides it for a new charge and for a viewer', async () => {
    expect(await render(null)).not.toContain('Supprimer')
    vehicleStore.canEdit = false
    expect(await render({ ...base, origin: 'MANUAL' })).not.toContain('Supprimer')
  })
})
