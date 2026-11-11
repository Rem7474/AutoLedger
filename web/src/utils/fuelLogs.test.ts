import { describe, expect, it } from 'vitest'
import { fuelMonths, fuelTotals } from './fuelLogs'

const fmt = (d: Date, withYear: boolean) => `${d.getMonth() + 1}${withYear ? '/' + d.getFullYear() : ''}`

describe('fuelTotals', () => {
  it('additionne litres et montant, le prix au litre ne portant que sur les pleins qui donnent les deux', () => {
    const t = fuelTotals([{ amount: 40, liters: 25 }, { amount: 20, liters: null }, { amount: 30, liters: 20 }])
    expect(t.liters).toBe(45)
    expect(t.amount).toBe(90)
    expect(t.pricePerLiter).toBeCloseTo(70 / 45)
  })
  it('ne donne pas de prix au litre sans quantité', () => {
    expect(fuelTotals([{ amount: 20 }]).pricePerLiter).toBeNull()
    expect(fuelTotals([]).pricePerLiter).toBeNull()
  })
})

describe('fuelMonths', () => {
  it("range les pleins sous un en-tête par mois, dans l'ordre reçu, avec les totaux du mois", () => {
    const logs = [
      { date: '2026-03-20T12:00:00', amount: 40, liters: 25 },
      { date: '2026-03-02T12:00:00', amount: 30, liters: 20 },
      { date: '2026-02-11T12:00:00', amount: 50, liters: 30 },
    ]
    const months = fuelMonths(logs, '2026-03-25', fmt)
    expect(months.map((m) => [m.month, m.label, m.items.length])).toEqual([['2026-03', '3', 2], ['2026-02', '2', 1]])
    expect(months[0].totals.amount).toBe(70)
  })
  it("ajoute l'année pour un mois d'une autre année", () => {
    expect(fuelMonths([{ date: '2025-12-05T12:00:00' }], '2026-01-10', fmt)[0].label).toBe('12/2025')
  })
})
