import { describe, expect, it } from 'vitest'
import { buildCarpoolBreakdown, buildDriveBreakdown, COST_COLORS } from './costBreakdown'

describe('buildDriveBreakdown', () => {
  const costs = { energy_cost: 3, tires_cost: 1, maintenance_cost: 0.5, insurance_cost: 0.5, tolls_cost: 5 }

  it('gives each item its share of the total and its cost per km', () => {
    const b = buildDriveBreakdown(costs, 100)
    expect(b.total).toBe(10)
    expect(b.byKey.tolls.sharePct).toBeCloseTo(50)
    expect(b.byKey.energy.costPerKm).toBeCloseTo(0.03)
    expect(b.items.reduce((s, i) => s + i.sharePct, 0)).toBeCloseTo(100)
  })

  it('uses the same colors as the monthly breakdown', () => {
    const b = buildDriveBreakdown(costs, 100)
    expect(b.byKey.energy.color).toBe(COST_COLORS.energy)
    expect(b.byKey.insurance.color).toBe(COST_COLORS.insurance)
  })

  it('does not divide by zero without costs or distance', () => {
    const b = buildDriveBreakdown(undefined, 0)
    expect(b.total).toBe(0)
    expect(b.items.every((i) => i.sharePct === 0 && i.costPerKm === 0)).toBe(true)
  })
})

describe('buildCarpoolBreakdown', () => {
  it('adds the other costs to the drive categories', () => {
    const b = buildCarpoolBreakdown({ energy_cost: 4, tolls_cost: 3, tires_cost: 1, maintenance_cost: 1, insurance_cost: 0, other_cost: 1, distance_km: 200 })
    expect(b.total).toBe(10)
    expect(b.byKey.other.sharePct).toBeCloseTo(10)
    expect(b.byKey.energy.costPerKm).toBeCloseTo(0.02)
    expect(b.items.map((i) => i.key)).toContain('other')
  })

  it('copes with a trip without costs', () => {
    expect(buildCarpoolBreakdown(undefined).total).toBe(0)
  })
})
