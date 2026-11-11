import { describe, expect, it } from 'vitest'
import { driveTotals } from './drives'

const drive = (km: number, cost: number | null, carpools: any[] = []) => ({ distance_km: km, costs: cost === null ? null : { total_cost: cost }, carpools })

describe('driveTotals', () => {
  it('additionne distance et coût brut, sans net sans covoiturage', () => {
    const t = driveTotals([drive(10, 2.5), drive(26.4, 6.99)])
    expect(t.distanceKm).toBeCloseTo(36.4)
    expect(t.gross).toBeCloseTo(9.49)
    expect(t.net).toBeNull()
    expect(t.incomplete).toBe(false)
  })

  it('retranche le revenu du covoiturage, compté une fois quand il couvre deux trajets du jour', () => {
    const ref = { id: 'c1', title: '', passenger_count: 1, total_revenue: 4.2 }
    const t = driveTotals([drive(15, 4, [ref]), drive(21, 5.8, [ref])])
    expect(t.gross).toBeCloseTo(9.8)
    expect(t.net).toBeCloseTo(5.6)
  })

  it('compte une fois au mois le covoiturage qui couvre deux jours, alors que chaque jour le porte en entier', () => {
    const ref = { id: 'c1', title: '', passenger_count: 1, total_revenue: 4.2 }
    const monday = [drive(15, 4, [ref])]
    const tuesday = [drive(21, 5.8, [ref])]
    expect(driveTotals(monday).net).toBeCloseTo(-0.2)
    expect(driveTotals(tuesday).net).toBeCloseTo(1.6)
    expect(driveTotals([...monday, ...tuesday]).net).toBeCloseTo(5.6)
  })

  it('signale un coût inconnu et additionne seulement les coûts connus', () => {
    expect(driveTotals([drive(10, 3), drive(5, null)])).toMatchObject({ gross: 3, distanceKm: 15, incomplete: true })
  })
})
