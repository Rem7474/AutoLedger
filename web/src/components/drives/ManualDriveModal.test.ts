import { describe, expect, it, vi } from 'vitest'
import { createSSRApp, defineComponent, h, reactive } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import ManualDriveModal from './ManualDriveModal.vue'

const vehicleStore = reactive<{ capabilities: { charges: boolean; refuels: boolean } }>({ capabilities: { charges: true, refuels: false } })
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => vehicleStore }))
vi.mock('@/stores/preferences', () => ({ usePreferencesStore: () => ({ proPersoEnabled: false }) }))
vi.mock('@/components/ModalShell.vue', () => ({
  default: defineComponent({ setup: (_p, { slots }) => () => h('div', [slots.default?.(), slots.footer?.()]) }),
}))

const render = () =>
  renderToString(createSSRApp({ render: () => h(ManualDriveModal as any, { open: true, vehicleId: 'v1' }) }).use(i18n))

describe('ManualDriveModal', () => {
  it('propose l\'énergie en kWh pour un véhicule électrique', async () => {
    vehicleStore.capabilities = { charges: true, refuels: false }
    const html = await render()
    expect(html).toContain('manual-drive-energy')
    expect(html).toContain('kWh')
  })

  it('garde le champ kWh d\'un hybride rechargeable, qui se recharge', async () => {
    vehicleStore.capabilities = { charges: true, refuels: true }
    expect(await render()).toContain('manual-drive-energy')
  })

  it('n\'a aucun champ kWh pour un véhicule thermique : son carburant vient des pleins', async () => {
    vehicleStore.capabilities = { charges: false, refuels: true }
    const html = await render()
    expect(html).not.toContain('manual-drive-energy')
    expect(html).not.toContain('kWh')
  })
})
