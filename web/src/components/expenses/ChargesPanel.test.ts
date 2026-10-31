import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, h, reactive } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import ChargesPanel from './ChargesPanel.vue'

const vehicleStore = reactive<{ canEdit: boolean; currency: string; activeVehicle: any }>({
  canEdit: true,
  currency: 'EUR',
  activeVehicle: { powertrain: 'EV', teslamate_api_url: null },
})
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => vehicleStore }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))

const manual = { id: 'c1', date: '2026-10-29T17:17:00Z', kwh_added: 22.5, cost: 6.3, currency: 'EUR', origin: 'MANUAL', address: 'Home' }
const render = (charges: any[] = [manual]) =>
  renderToString(
    createSSRApp({
      render: () => h(ChargesPanel as any, { charges, chargesTotal: charges.length, chargesWithoutCost: 0, loading: false, loadingMoreCharges: false, missingCostOnly: false }),
    }).use(i18n),
  )

afterEach(() => {
  vehicleStore.canEdit = true
})

describe('ChargesPanel', () => {
  it('opens a charge from the whole row, with no pencil button', async () => {
    const html = await render()
    expect(html).toContain('inset-0')
    expect(html).not.toContain('lucide-pencil')
  })

  it('keeps the menu for the actions the row does not carry', async () => {
    expect(await render()).toContain('aria-haspopup="menu"')
  })

  it('shows a read-only list with no open button and no menu', async () => {
    vehicleStore.canEdit = false
    const html = await render()
    expect(html).not.toContain('inset-0')
    expect(html).not.toContain('aria-haspopup="menu"')
  })
})

describe('ChargesPanel menu', () => {
  it('offers no menu for a synchronised charge that has no TeslaMate link: it cannot be deleted', async () => {
    const html = await renderSync()
    expect(html).not.toContain('aria-haspopup="menu"')
    expect(html).toContain('inset-0')
  })
})

const renderSync = () =>
  renderToString(
    createSSRApp({
      render: () => h(ChargesPanel as any, { charges: [{ ...manual, origin: 'TESLAMATE' }], chargesTotal: 1, chargesWithoutCost: 0, loading: false, loadingMoreCharges: false, missingCostOnly: false }),
    }).use(i18n),
  )
