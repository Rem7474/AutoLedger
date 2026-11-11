import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, h, reactive } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import DriveCostModal from './DriveCostModal.vue'

const vehicleStore = reactive<{ canEdit: boolean; currency: string; activeVehicle: any }>({ canEdit: true, currency: 'EUR', activeVehicle: { powertrain: 'EV' } })
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => vehicleStore }))
vi.mock('@/stores/preferences', () => ({ usePreferencesStore: () => ({ proPersoEnabled: false }) }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))
vi.mock('@/services/api', () => ({ api: {} }))

const drive = { id: 'd1', start_time: '2026-10-29T17:17:00Z', distance_km: 22.8, is_manual: true, costs: { total_cost: 5.26, cost_per_km: 0.231 } }
const render = (d: any, manageable = true, extra: Record<string, unknown> = {}) =>
  renderToString(
    createSSRApp({
      render: () =>
        h(DriveCostModal as any, { open: true, drive: d, manageable, vehicleId: 'v1', tripDriveIds: [], tripLegs: [], startWithTollEntry: false, refreshDrive: async () => null, ...extra }),
    })
      .directive('dialog', {})
      .use(i18n),
  )

afterEach(() => {
  vehicleStore.canEdit = true
})

describe('DriveCostModal : modifier et supprimer', () => {
  it('propose Modifier dans l\'en-tête et Supprimer en bas du détail pour un trajet manuel', async () => {
    const html = await render(drive)
    expect(html).toContain('Modifier')
    expect(html).toContain('text-critical-400')
    expect(html).toContain('Supprimer')
  })

  it('n\'en propose aucun pour un trajet synchronisé, une lecture seule ou une page qui ne gère pas', async () => {
    for (const html of [await render({ ...drive, is_manual: false }), await render(drive, false)]) {
      expect(html).not.toContain('text-critical-400')
      expect(html).not.toContain('Supprimer')
    }
    vehicleStore.canEdit = false
    expect(await render(drive)).not.toContain('Supprimer')
  })
})

describe('DriveCostModal : un voyage', () => {
  const trip = { id: 'g1', is_trip_group: true, trip_group_name: 'Vacances', trip_group_notes: 'Aller-retour', start_time: '2026-07-01T08:00:00Z', distance_km: 900, costs: { total_cost: 80, cost_per_km: 0.09 } }

  it('propose Modifier dans l\'en-tête, Supprimer en bas, et les notes du voyage', async () => {
    const html = await render(trip)
    expect(html).toContain('text-critical-400')
    expect(html).toContain('Supprimer')
    expect(html).toContain('Aller-retour')
    // le bouton Modifier du pied de page a disparu : un seul bouton Modifier
    expect(html.match(/Modifier/g)).toHaveLength(1)
  })

  it('n\'en propose aucun pour une lecture seule, une page qui ne gère pas ou une suggestion', async () => {
    for (const html of [await render(trip, false), await render({ ...trip, is_suggestion: true })]) {
      expect(html).not.toContain('Supprimer')
      expect(html).not.toContain('Modifier')
    }
    vehicleStore.canEdit = false
    const html = await render(trip)
    expect(html).not.toContain('Supprimer')
    expect(html).not.toContain('Modifier')
  })
})

describe('DriveCostModal : une étape d\'un voyage', () => {
  it('propose de la retirer du voyage, en bas du détail, sans la supprimer', async () => {
    const html = await render({ ...drive, is_manual: false }, true, { inTrip: true })
    expect(html).toContain('Retirer de ce voyage')
    expect(html).not.toContain('Supprimer')
  })

  it('propose aussi Supprimer pour une étape saisie à la main', async () => {
    const html = await render(drive, true, { inTrip: true })
    expect(html).toContain('Retirer de ce voyage')
    expect(html).toContain('Supprimer')
  })

  it('ne la propose pas hors voyage, en lecture seule, sur une page qui ne gère pas ou sur le voyage lui-même', async () => {
    expect(await render(drive)).not.toContain('Retirer de ce voyage')
    expect(await render(drive, false, { inTrip: true })).not.toContain('Retirer de ce voyage')
    expect(await render({ id: 'g1', is_trip_group: true, trip_group_name: 'V', start_time: '2026-07-01T08:00:00Z', costs: {} }, true, { inTrip: true })).not.toContain('Retirer de ce voyage')
    vehicleStore.canEdit = false
    expect(await render(drive, true, { inTrip: true })).not.toContain('Retirer de ce voyage')
  })
})
