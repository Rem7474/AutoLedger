import { describe, expect, it } from 'vitest'
import { fuelLogsInPeriod, fuelMonths, fuelPeriodSummary, fuelTotals } from './fuelLogs'

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

// Oldest first, as the API sends them; the consumption is measured by the server over the whole history
const history = [
  { date: '2026-01-05T12:00:00', amount: 40, liters: 40, odometer: 1000, is_full_tank: true },
  { date: '2026-02-10T12:00:00', amount: 50, liters: 30, odometer: 1500, is_full_tank: true, consumption_l_100km: 6, segment_km: 500 },
  { date: '2026-03-10T12:00:00', amount: 60, liters: 40, odometer: 2000, is_full_tank: true, consumption_l_100km: 8, segment_km: 500, segment_estimated: true },
  { date: '2026-03-20T12:00:00', amount: 20, liters: 10, is_full_tank: false },
  { date: '2026-04-05T12:00:00', amount: 70, is_full_tank: true },
]

describe('fuelLogsInPeriod', () => {
  it('garde les pleins dont le jour est dans la période, bornes comprises, sans changer leur ordre', () => {
    const kept = fuelLogsInPeriod(history, { from: '2026-02-10', to: '2026-03-20' })
    expect(kept.map((l) => l.date.slice(0, 10))).toEqual(['2026-02-10', '2026-03-10', '2026-03-20'])
  })
  it("traite une borne absente comme ouverte, donc tout l'historique sans borne", () => {
    expect(fuelLogsInPeriod(history, {})).toHaveLength(5)
    expect(fuelLogsInPeriod(history, { from: '2026-03-11' })).toHaveLength(2)
    expect(fuelLogsInPeriod(history, { to: '2026-01-31' })).toHaveLength(1)
  })
  it('rend une liste vide pour une période sans plein', () => {
    expect(fuelLogsInPeriod(history, { from: '2027-01-01', to: '2027-12-31' })).toEqual([])
  })
})

describe('fuelPeriodSummary', () => {
  it("compte les pleins, la dépense et le prix au litre de la période, et la consommation des intervalles qu'elle clôt", () => {
    const s = fuelPeriodSummary(history, { from: '2026-03-01', to: '2026-03-31' })
    expect(s.fillUps).toBe(2)
    expect(s.totalCost).toBe(80)
    expect(s.pricePerLiter).toBeCloseTo(80 / 50)
    // Le plein du 10 mars garde le plein de février comme point de départ : l'intervalle est mesuré sur tout l'historique
    expect(s.consumption).toBeCloseTo(8)
    expect(s.estimated).toBe(1)
    expect(s.withoutMileage).toBe(1)
    expect(s.unmeasurable).toBe(0)
  })
  it('pondère la consommation par la distance des intervalles, comme le chiffre du serveur', () => {
    const s = fuelPeriodSummary(
      [
        { date: '2026-01-01T12:00:00', amount: 1, odometer: 0, is_full_tank: true },
        { date: '2026-02-01T12:00:00', amount: 1, liters: 10, is_full_tank: true, consumption_l_100km: 10, segment_km: 100 },
        { date: '2026-03-01T12:00:00', amount: 1, liters: 30, is_full_tank: true, consumption_l_100km: 5, segment_km: 600 },
      ],
      {},
    )
    // 10 L sur 100 km et 30 L sur 600 km : 40 L sur 700 km, pas la moyenne simple de 10 et 5
    expect(s.consumption).toBeCloseTo((40 / 700) * 100)
  })
  it('ignore les intervalles clos hors de la période et signale ceux sans consommation', () => {
    const april = fuelPeriodSummary(history, { from: '2026-04-01', to: '2026-04-30' })
    expect(april.fillUps).toBe(1)
    expect(april.consumption).toBeNull()
    expect(april.unmeasurable).toBe(1)
    expect(april.pricePerLiter).toBeNull()
  })
  it("n'a ni plein ni consommation pour une période vide", () => {
    const s = fuelPeriodSummary(history, { from: '2027-01-01', to: '2027-01-31' })
    expect(s).toMatchObject({ fillUps: 0, totalCost: 0, pricePerLiter: null, consumption: null, unmeasurable: 0, withoutMileage: 0 })
  })
  it("rend tout l'historique sans borne", () => {
    const s = fuelPeriodSummary(history, {})
    expect(s.fillUps).toBe(5)
    expect(s.totalCost).toBe(240)
    expect(s.consumption).toBeCloseTo(((30 + 40) / 1000) * 100)
  })
})
