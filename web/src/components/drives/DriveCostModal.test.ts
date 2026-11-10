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
const render = (d: any, manageable = true) =>
  renderToString(
    createSSRApp({
      render: () =>
        h(DriveCostModal as any, { open: true, drive: d, manageable, vehicleId: 'v1', tripDriveIds: [], tripLegs: [], startWithTollEntry: false, refreshDrive: async () => null }),
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
