import { describe, expect, it } from 'vitest'
import { lastDrivenMonth, topCostShares } from './homeRows'

describe('topCostShares', () => {
  it('lists the largest items first, with their share of the total', () => {
    const tco = { energy_cost: 60, insurance_cost: 30, tolls_cost: 10 }
    const top = topCostShares(tco)
    expect(top.map((r) => r.key)).toEqual(['energy', 'insurance'])
    expect(top[0].sharePct).toBeCloseTo(60)
    expect(top[1].sharePct).toBeCloseTo(30)
  })

  it('has nothing to list while no cost is known', () => {
    expect(topCostShares(null)).toEqual([])
    expect(topCostShares({})).toEqual([])
  })
})

describe('lastDrivenMonth', () => {
  it('skips the trailing months without a distance', () => {
    const months = [{ month: '2026-09', distance_km: 800 }, { month: '2026-10', distance_km: 1200 }, { month: '2026-11', distance_km: 0 }]
    expect(lastDrivenMonth(months)).toEqual({ month: '2026-10', distance_km: 1200 })
  })

  it('is unknown without any distance', () => {
    expect(lastDrivenMonth([{ month: '2026-11' }])).toBeNull()
    expect(lastDrivenMonth(null)).toBeNull()
  })
})
