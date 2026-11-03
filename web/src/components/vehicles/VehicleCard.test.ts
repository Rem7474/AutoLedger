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

describe('VehicleCard on a phone', () => {
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
