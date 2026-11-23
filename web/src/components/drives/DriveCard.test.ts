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

  it('n\'a pas de case à cocher : le glyphe devient la coche de la sélection, annoncée sur le bouton de la ligne', async () => {
    expect(await render()).not.toContain('type="checkbox"')
    expect(await render()).not.toContain('aria-pressed')
    const selecting = renderToString(createSSRApp({ render: () => h(DriveCard as any, { d: drive, selected: true, selecting: true }) }).use(i18n))
    const html = await selecting
    expect(html).not.toContain('type="checkbox"')
    expect(html).toContain('aria-pressed="true"')
    vehicleStore.canEdit = false
    expect(await render()).not.toContain('aria-pressed')
  })

  it('tient en trois informations : l\'itinéraire, l\'heure de départ et le coût', async () => {
    const html = await render({ ...drive, start_address: 'Domicile', end_address: 'Bureau', duration_min: 30, consumption_kwh_100km: 16.4, costs: { ...drive.costs, fuel_liters: 1.6 } })
    expect(html).toContain('Domicile')
    expect(html).not.toContain('22,8')
    expect(html).not.toContain('30 min')
    expect(html).not.toContain('kWh/100')
    expect(html).not.toContain('0,231')
    expect(html).not.toMatch(/1,6\s(<!--.*?-->)?\s?L/)
    expect(html).toContain('5,26')
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

  it('garde les boutons de qualification péage dans la carte du trajet, pas dessous', async () => {
    const html = (await render({ ...drive, is_manual: false, needs_toll_qualification: true, distance_km: 105, costs: { total_cost: 24, cost_per_km: 0.23 } })).trim()
    expect(html).toContain('Sans péage')
    let depth = 0
    let closedAt = -1
    for (const m of html.matchAll(/<div\b|<\/div>/g)) {
      depth += m[0] === '</div>' ? -1 : 1
      if (depth === 0) { closedAt = m.index! + m[0].length; break }
    }
    expect(closedAt).toBe(html.length)
    expect(html.lastIndexOf('Sans péage')).toBeLessThan(closedAt)
  })

  it('ouvre le détail depuis toute la ligne : le coût est sous le bouton d\'ouverture, pas au-dessus', async () => {
    const html = await render()
    expect(html).toContain('inset-0')
    const cost = html.slice(html.indexOf('5,26') - 300, html.indexOf('5,26'))
    expect(cost).not.toContain('z-10')
  })
})
