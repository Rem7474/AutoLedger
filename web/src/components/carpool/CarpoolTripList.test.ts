import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, h, reactive } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import CarpoolTripList from './CarpoolTripList.vue'

const vehicleStore = reactive<{ canEdit: boolean; currency: string }>({ canEdit: true, currency: 'EUR' })
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => vehicleStore }))
// The overflow actions only render in the open "More" menu: the stub lists every action of the bar with its state
vi.mock('@/components/BulkSelectionBar.vue', async () => {
  const { defineComponent, h } = await import('vue')
  return {
    default: defineComponent({
      props: ['count', 'noun', 'actions'],
      render() {
        return h('ul', (this.actions as any[]).map((a) => h('li', { 'data-key': a.key, 'data-disabled': String(!!a.disabled), title: a.disabled ? a.reason : '' }, a.label)))
      },
    }),
  }
})

const trip = { id: 't1', title: 'Lyon to Nice', date: '2026-10-10T08:00:00Z', distance_km: 470, legs: [{}, {}], passengers: [{}], total_cost: 100, total_revenue: 60, net_cost: 40, notes: '' }
const render = (trips: any[] = [trip], selectedTripIds: string[] = [], total?: number) =>
  renderToString(createSSRApp({ render: () => h(CarpoolTripList as any, { trips, selectedTripIds, recalculating: false, total }) }).use(i18n))

afterEach(() => {
  vehicleStore.canEdit = true
})

describe('CarpoolTripList', () => {
  it('counts the carpools of the vehicle, not only the pages loaded', async () => {
    expect(await render([trip])).toContain('1 covoiturage(s) au total')
    expect(await render([trip], [], 500)).toContain('500 covoiturage(s) au total')
  })

  it('opens a carpool from the whole row, with no row menu and no icon row', async () => {
    const html = await render()
    expect(html).toContain('inset-0')
    expect(html).not.toContain('aria-haspopup="menu"')
    expect(html).not.toContain('data-icon="edit"')
    expect(html).not.toContain('data-icon="delete"')
  })

  it('says the recovery and the origin of the amounts as plain text in the quiet line', async () => {
    const html = await render([trip, { ...trip, id: 'b', net_cost: 0, legs: [{ drive_id: 'd1' }], costs_frozen_at: '2026-10-10T08:00:00Z' }])
    expect(html).toContain('data-tone="attention"')
    expect(html).toContain('60 % payé par les passagers')
    expect(html).toContain('data-tone="neutral"')
    expect(html).toContain('Montants figés')
    expect(html).not.toContain('rounded-full font-semibold border')
  })

  it('shows what is left to the driver, or the surplus once the cost is recovered', async () => {
    expect(await render()).toContain('40,00')
    const recovered = await render([{ ...trip, net_cost: -5 }])
    expect(recovered).toContain('+5,00')
    expect(recovered).toContain('text-success-400')
  })

  it('leaves the selection box out of a read-only list', async () => {
    vehicleStore.canEdit = false
    const html = await render()
    expect(html).not.toContain('carpool-select-')
  })

  it('offers to freeze or resync the selected carpools made from drives, with how many each applies to', async () => {
    const linked = { ...trip, id: 'a', legs: [{ drive_id: 'd1' }] }
    const frozen = { ...trip, id: 'b', legs: [{ drive_id: 'd2' }], costs_frozen_at: '2026-10-10T08:00:00Z' }
    const html = await render([linked, frozen, { ...trip, id: 'c' }], ['a', 'b', 'c'])
    expect(html).toContain('data-key="freeze" data-disabled="false" title>Figer les montants (1)')
    expect(html).toContain('data-key="resync" data-disabled="false" title>Resynchroniser (1)')
  })

  it('keeps the freeze action visible but disabled when no selected carpool is made from drives', async () => {
    const html = await render([trip], ['t1'])
    expect(html).toContain('data-key="freeze" data-disabled="true" title="Aucun covoiturage lié aux trajets à figer">Figer les montants (0)')
    expect(html).toContain('data-key="resync" data-disabled="true" title="Aucun covoiturage figé à resynchroniser">Resynchroniser (0)')
  })
})
