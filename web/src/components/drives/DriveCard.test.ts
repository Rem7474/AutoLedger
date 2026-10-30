import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, h, reactive } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import DriveCard from './DriveCard.vue'

const vehicleStore = reactive<{ canEdit: boolean; currency: string; activeVehicle: any }>({
  canEdit: true,
  currency: 'EUR',
  activeVehicle: { powertrain: 'EV', teslamate_api_url: null },
})
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => vehicleStore }))
vi.mock('@/stores/preferences', () => ({ usePreferencesStore: () => ({ proPersoEnabled: false }) }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))

const drive = { id: 'd1', start_time: '2026-10-29T17:17:00Z', distance_km: 22.8, is_manual: true, costs: { total_cost: 5.26, cost_per_km: 0.231 } }
const render = (d: any = drive) => renderToString(createSSRApp({ render: () => h(DriveCard as any, { d, selected: false }) }).use(i18n))

afterEach(() => {
  vehicleStore.canEdit = true
  vehicleStore.activeVehicle = { powertrain: 'EV', teslamate_api_url: null }
})

describe('DriveCard', () => {
  it('leaves out the manual badge when every drive of the vehicle is manual', async () => {
    expect(await render()).not.toContain('bg-slate-800 text-slate-300 border border-slate-700/60')
  })

  it('shows the manual badge on a vehicle fed by TeslaMate, where it tells the drives apart', async () => {
    vehicleStore.activeVehicle = { powertrain: 'EV', teslamate_api_url: 'http://teslamate.local' }
    expect(await render()).toContain('bg-slate-800 text-slate-300 border border-slate-700/60')
    expect(await render({ ...drive, is_manual: false })).not.toContain('bg-slate-800 text-slate-300 border border-slate-700/60')
  })

  it('puts the selection box in the corner of the card on a phone and keeps the text clear of it', async () => {
    const editable = await render()
    expect(editable).toContain('absolute right-1 top-1')
    expect(editable).toContain('pr-10 sm:pr-0')
    vehicleStore.canEdit = false
    const readOnly = await render()
    expect(readOnly).not.toContain('drive-select-')
    expect(readOnly).not.toContain('pr-10')
  })
})
