import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, h, reactive } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createMemoryHistory, createRouter } from 'vue-router'
import { i18n, setLocale } from '@/i18n'
import VehicleCard from './VehicleCard.vue'

const store = reactive({ activeVehicleId: 'a', setActiveVehicle: vi.fn() })
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => store }))

const vehicle = (over: Record<string, unknown> = {}) => ({ id: 'a', name: 'Daily', role: 'OWNER', powertrain: 'EV', currency: 'EUR', current_odometer: 1000, ...over })
const render = (v: Record<string, unknown>) =>
  renderToString(
    createSSRApp({ render: () => h(VehicleCard as any, { v, ownership: null }) })
      .use(i18n)
      .use(createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: {} }, { path: '/odometer', component: {} }] })),
  )

beforeEach(() => setLocale('en'))

describe('VehicleCard selection', () => {
  it('lets the whole card select a vehicle that is not the active one', async () => {
    const html = await render(vehicle({ id: 'b' }))
    expect(html).toContain('aria-label="Select Daily"')
    expect(html).not.toContain('>Active<')
  })

  it('marks the active vehicle and leaves nothing to select', async () => {
    const html = await render(vehicle())
    expect(html).not.toContain('aria-label="Select Daily"')
    expect(html).toContain('Active')
  })
})

describe('VehicleCard actions', () => {
  it('keeps the setup actions in the menu instead of repeating a row of buttons', async () => {
    const html = await render(vehicle())
    expect(html).toContain('aria-haspopup="menu"')
    expect(html).not.toContain('btn-secondary')
  })

  it('shows the same facts on an electric and a combustion vehicle', async () => {
    const ev = await render(vehicle())
    const ice = await render(vehicle({ powertrain: 'ICE' }))
    expect(ev).toContain('Electric')
    expect(ice).toContain('Combustion')
    expect(ev).toContain('TeslaMate connection')
    expect(ice).not.toContain('TeslaMate connection')
  })

  it('lets the owner open the data sources from the connection state', async () => {
    expect(await render(vehicle())).toContain('Not configured')
    expect(await render(vehicle({ role: 'EDITOR' }))).toContain('Managed by the administrator')
  })

  it('gives a co-driver no contract to edit', async () => {
    const html = await render(vehicle({ role: 'VIEWER' }))
    expect(html).toContain('Read-only')
    expect(html).not.toContain('Enter the contract')
  })
})
