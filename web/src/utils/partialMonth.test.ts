import { describe, expect, it } from 'vitest'
import { costPerKmSeries, monthKeyOf } from './partialMonth'

const m = (month: string, distance_km: number | null, cost_per_km: number | null) => ({ month, distance_km, cost_per_km })
const NOV = new Date(2026, 10, 9)

describe('monthKeyOf', () => {
  it('names the month of a date in local time', () => {
    expect(monthKeyOf(new Date(2026, 0, 31, 23, 59))).toBe('2026-01')
    expect(monthKeyOf(NOV)).toBe('2026-11')
  })
})

describe('costPerKmSeries', () => {
  it('sets the month in progress apart and scales on the complete months', () => {
    const s = costPerKmSeries([m('2026-09', 900, 0.5), m('2026-10', 1000, 0.6), m('2026-11', 160, 2.5)], NOV)
    expect(s.complete).toEqual([0.5, 0.6, null])
    expect(s.partial).toEqual({ index: 2, value: 2.5 })
    expect(s.scaleMax).toBeCloseTo(0.69)
  })

  it('keeps every month on the line when none is in progress', () => {
    const s = costPerKmSeries([m('2026-08', 800, 0.4), m('2026-09', 900, 0.5)], NOV)
    expect(s.complete).toEqual([0.4, 0.5])
    expect(s.partial).toBeNull()
  })

  it('has no hollow point for a month in progress under the minimum distance, and leaves gaps as gaps', () => {
    const s = costPerKmSeries([m('2026-10', 20, 3), m('2026-11', 10, 4)], NOV)
    expect(s.complete).toEqual([null, null])
    expect(s.partial).toBeNull()
    expect(s.scaleMax).toBeUndefined()
  })
})
