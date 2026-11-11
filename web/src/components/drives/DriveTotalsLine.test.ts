import { describe, expect, it, vi } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import DriveTotalsLine from './DriveTotalsLine.vue'
import { driveTotals } from '@/utils/drives'

vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => ({ currency: 'EUR' }) }))

const render = (props: any) => renderToString(createSSRApp({ render: () => h(DriveTotalsLine as any, props) }).use(i18n))
const drive = (km: number, cost: number | null, carpools: any[] = []) => ({ distance_km: km, costs: cost === null ? null : { total_cost: cost }, carpools })
const carpool = { id: 'c1', title: '', passenger_count: 1, total_revenue: 4.2 }

describe('DriveTotalsLine', () => {
  it('montre distance et coût brut du jour, sans net hors covoiturage', async () => {
    const html = await render({ totals: driveTotals([drive(36.4, 9.49)]), showNet: true })
    expect(html).toContain('36,4')
    expect(html).toContain('9,49')
    expect(html).not.toContain('net')
  })

  it('ajoute le net seulement quand le jour a du covoiturage', async () => {
    const html = await render({ totals: driveTotals([drive(36.4, 9.8, [carpool])]), showNet: true })
    expect(html).toContain('net 5,60')
    expect(await render({ totals: driveTotals([drive(36.4, 9.8, [carpool])]) })).not.toContain('net')
  })

  it('note les totaux incomplets quand un coût manque', async () => {
    const html = await render({ totals: driveTotals([drive(10, 3), drive(5, null)]), showNet: true })
    expect(html).toContain('incomplet')
    expect(await render({ totals: driveTotals([drive(10, 3)]), showNet: true })).not.toContain('incomplet')
  })

  it('ne dessine rien tant que le groupe peut continuer sur la page suivante', async () => {
    expect(await render({ totals: null, showNet: true })).not.toContain('drive-totals')
  })
})
