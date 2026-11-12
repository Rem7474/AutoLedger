import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, h, reactive } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import DriveCard from './DriveCard.vue'

const vehicleStore = reactive<{ canEdit: boolean; currency: string; capabilities: { sync: boolean } }>({
  canEdit: true,
  currency: 'EUR',
  capabilities: { sync: false },
})
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => vehicleStore }))
vi.mock('@/stores/preferences', () => ({ usePreferencesStore: () => ({ proPersoEnabled: false }) }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))

const drive = { id: 'd1', start_time: '2026-10-29T17:17:00Z', distance_km: 22.8, is_manual: true, costs: { total_cost: 5.26, cost_per_km: 0.231 } }
const render = (d: any = drive) => renderToString(createSSRApp({ render: () => h(DriveCard as any, { d, selected: false }) }).use(i18n))

afterEach(() => {
  vehicleStore.canEdit = true
  vehicleStore.capabilities = { sync: false }
})

describe('DriveCard', () => {
  it('ne mentionne pas « manuel » quand tous les trajets du véhicule sont manuels', async () => {
    expect(await render()).not.toContain('Manuel')
  })

  it('mentionne « manuel » sur un véhicule alimenté par TeslaMate, où cela distingue les trajets', async () => {
    vehicleStore.capabilities = { sync: true }
    expect(await render()).toContain('Manuel')
    expect(await render({ ...drive, is_manual: false })).not.toContain('Manuel')
  })

  it('laisse de côté la case de sélection dans une liste en lecture seule', async () => {
    expect(await render()).toContain('drive-select-')
    vehicleStore.canEdit = false
    expect(await render()).not.toContain('drive-select-')
  })

  it('n\'a plus de menu de ligne : modifier, supprimer et le lien TeslaMate sont dans le détail', async () => {
    const html = await render()
    expect(html).not.toContain('aria-haspopup="menu"')
    expect(html).not.toContain('role="menuitem"')
  })

  it('garde le raccourci covoiturage comme seul bouton de la ligne, en plus du bouton d\'ouverture', async () => {
    const carpooled = { ...drive, carpools: [{ id: 'c1', title: '', passenger_count: 2, total_revenue: 12 }] }
    const html = await render(carpooled)
    expect(html).toContain('Ouvrir le covoiturage de ce trajet')
    expect((await render()).match(/<button/g)).toHaveLength(1)
    expect(html.match(/<button/g)).toHaveLength(2)
  })

  it('met la consommation dans la ligne discrète mais la retire du téléphone', async () => {
    const html = await render({ ...drive, consumption_kwh_100km: 16.4, duration_min: 30 })
    expect(html).toContain('kWh/100')
    expect(html).toMatch(/max-sm:hidden[^>]*>[^<]*kWh\/100/)
    expect(html).toContain('30 min')
  })

  it('ouvre le détail depuis toute la ligne : le coût est sous le bouton d\'ouverture, pas au-dessus', async () => {
    const html = await render()
    expect(html).toContain('inset-0')
    const cost = html.slice(html.indexOf('€5.26') - 300, html.indexOf('€5.26'))
    expect(cost).not.toContain('z-10')
  })
})
