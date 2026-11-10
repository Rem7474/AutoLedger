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
  it('ne mentionne pas « manuel » quand tous les trajets du véhicule sont manuels', async () => {
    expect(await render()).not.toContain('Manuel')
  })

  it('mentionne « manuel » sur un véhicule alimenté par TeslaMate, où cela distingue les trajets', async () => {
    vehicleStore.activeVehicle = { powertrain: 'EV', teslamate_api_url: 'http://teslamate.local' }
    expect(await render()).toContain('Manuel')
    expect(await render({ ...drive, is_manual: false })).not.toContain('Manuel')
  })

  it('laisse de côté la case de sélection et le menu dans une liste en lecture seule', async () => {
    expect(await render()).toContain('drive-select-')
    vehicleStore.canEdit = false
    const readOnly = await render()
    expect(readOnly).not.toContain('drive-select-')
    expect(readOnly).not.toContain('aria-haspopup="menu"')
  })

  it('ouvre le détail depuis toute la ligne : le coût est sous le bouton d\'ouverture, pas au-dessus', async () => {
    const html = await render()
    expect(html).toContain('inset-0')
    const cost = html.slice(html.indexOf('€5.26') - 300, html.indexOf('€5.26'))
    expect(cost).not.toContain('z-10')
  })
})
