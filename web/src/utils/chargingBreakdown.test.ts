import { describe, expect, it } from 'vitest'
import type { ChargeClass } from '@/components/dashboard/energyStats'
import type { PlaceCategoryRow } from '@/utils/chargingInsights'
import { activeAxis, availableAxes, placeBreakdownRows, speedBreakdownRows } from './chargingBreakdown'

const cls = (c: ChargeClass['class'], kwh: number, price?: number, withoutCost = 0): ChargeClass => ({
  class: c, sessions: 3, kwh_added: kwh, energy_cost: 0, price_per_kwh: price, priced_kwh: kwh, sessions_without_cost: withoutCost,
})
const place = (category: string, kwh: number, sharePct: number, pricePerKwh: number | null, unknownSessions = 0): PlaceCategoryRow => ({
  category, sessions: 2, kwh, sharePct, pricePerKwh, unknownSessions,
})

describe('placeBreakdownRows', () => {
  it('keeps the share, the price and the unknown sessions of each place, labelled by the caller', () => {
    const rows = placeBreakdownRows([place('HOME', 75, 75, 0.2), place('', 25, 25, null, 2)], (c) => `L:${c || 'none'}`)
    expect(rows.map((r) => [r.key, r.label, r.kwh, r.sharePct, r.pricePerKwh, r.unknownSessions])).toEqual([
      ['HOME', 'L:HOME', 75, 75, 0.2, 0],
      ['unclassified', 'L:none', 25, 25, null, 2],
    ])
    expect(rows[0].color).not.toBe(rows[1].color)
  })
})

describe('speedBreakdownRows', () => {
  it('computes each share over the energy of the classes shown and drops a class with no energy', () => {
    const rows = speedBreakdownRows([cls('AC', 30, 0.2), cls('DC', 10, 0.5, 1), cls('UNKNOWN', 0)], (c) => c)
    expect(rows.map((r) => [r.key, r.sharePct, r.pricePerKwh, r.unknownSessions])).toEqual([
      ['AC', 75, 0.2, 0],
      ['DC', 25, 0.5, 1],
    ])
  })

  it('reads a missing price as null and a zero total as no share', () => {
    expect(speedBreakdownRows([cls('AC', 12)], (c) => c)[0].pricePerKwh).toBeNull()
    expect(speedBreakdownRows([], (c) => c)).toEqual([])
  })
})

describe('available and active axis', () => {
  const row = placeBreakdownRows([place('HOME', 1, 100, 0.2)], () => 'x')
  it('offers only the splits that have energy', () => {
    expect(availableAxes(row, row)).toEqual(['place', 'speed'])
    expect(availableAxes([], row)).toEqual(['speed'])
    expect(availableAxes(row, [])).toEqual(['place'])
    expect(availableAxes([], [])).toEqual([])
  })

  it('falls back to the first available axis when the chosen one has no data', () => {
    expect(activeAxis('speed', ['place', 'speed'])).toBe('speed')
    expect(activeAxis('speed', ['place'])).toBe('place')
    expect(activeAxis('place', [])).toBeNull()
  })
})
