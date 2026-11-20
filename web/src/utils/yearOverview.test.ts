import { describe, expect, it } from 'vitest'
import { yearOverview } from '@/utils/yearOverview'

const m = (month: string, over = {}) => ({ month, total: 100, scheduled: 0, distance_km: 200, energy: 60, tolls: 0, tires: 0, maintenance: 0, insurance: 40, financing: 0, other: 0, carpool_revenue: 0, ...over })
const months = (from: number, to: number) => Array.from({ length: to - from + 1 }, (_, i) => m(`2026-${String(from + i).padStart(2, '0')}`))

describe('yearOverview', () => {
  it('has nothing without two complete months', () => {
    expect(yearOverview(undefined, '2026-11', 'EUR')).toBeNull()
    expect(yearOverview([m('2026-10'), m('2026-11')], '2026-11', 'EUR')).toBeNull()
    expect(yearOverview([m('2026-09'), m('2026-10')], '2026-11', 'EUR')).not.toBeNull()
  })

  it('reads the complete months and draws the month in progress apart, out of the figures', () => {
    const o = yearOverview([...months(8, 10), m('2026-11', { total: 30, energy: 30, insurance: 0 })], '2026-11', 'EUR')!
    expect(o.months).toBe(3)
    expect(o.total).toBe(300)
    expect(o.average).toBe(100)
    expect(o.perKm).toBeCloseTo(0.5)
    expect(o.bars.map((b) => [b.month, b.partial])).toEqual([['2026-08', false], ['2026-09', false], ['2026-10', false], ['2026-11', true]])
  })

  it('keeps the 12 latest complete months, whatever the order of the list', () => {
    const list = [...months(1, 10), m('2025-11'), m('2025-12'), m('2025-10')].reverse()
    const o = yearOverview(list, '2026-11', 'EUR')!
    expect(o.months).toBe(12)
    expect(o.bars[0].month).toBe('2025-11')
    expect(o.bars.at(-1)!.month).toBe('2026-10')
  })

  it('is net of the carpools, each bar cut into its themes to the net total', () => {
    const o = yearOverview([m('2026-09', { carpool_revenue: 50 }), m('2026-10')], '2026-11', 'EUR')!
    expect(o.bars[0].total).toBe(50)
    expect(o.bars[0].segments.reduce((sum, s) => sum + s.amount, 0)).toBeCloseTo(50)
    expect(o.max).toBe(100)
  })

  it('has no cost per km without a distance', () => {
    expect(yearOverview([m('2026-09', { distance_km: 0 }), m('2026-10', { distance_km: 0 })], '2026-11', 'EUR')!.perKm).toBeNull()
  })

  it('starts at the first month with spending, not before', () => {
    const empty = { total: 0, energy: 0, insurance: 0 }
    const o = yearOverview([m('2026-06', empty), m('2026-07', empty), m('2026-08'), m('2026-09'), m('2026-10')], '2026-11', 'EUR')!
    expect(o.months).toBe(3)
    expect(o.bars[0].month).toBe('2026-08')
    expect(yearOverview([m('2026-08', empty), m('2026-09', empty), m('2026-10')], '2026-11', 'EUR')).toBeNull()
  })

  it('leaves a month with no spending as an empty bar', () => {
    const o = yearOverview([m('2026-08'), m('2026-09', { energy: 0, insurance: 0, total: 0 }), m('2026-10')], '2026-11', 'EUR')!
    expect(o.bars[1]).toMatchObject({ total: 0, segments: [] })
  })
})
