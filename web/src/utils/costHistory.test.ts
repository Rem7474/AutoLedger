import { describe, expect, it } from 'vitest'
import { monthsOfPeriod, periodLabel, rollingAverage, summarizePeriod, type CostMonth } from './costHistory'

const history = (amounts: number[], first = '2025-01'): CostMonth[] =>
  amounts.map((amount, i) => {
    const [y, m] = first.split('-').map(Number)
    const d = new Date(Date.UTC(y, m - 1 + i, 1))
    return { month: d.toISOString().slice(0, 7), amount }
  })

describe('monthsOfPeriod', () => {
  const list = history(Array.from({ length: 18 }, (_, i) => i + 1))

  it('takes the last N months of a preset, the current one included, and everything for the whole history', () => {
    expect(monthsOfPeriod(list, '1M', null).map((m) => m.amount)).toEqual([18])
    expect(monthsOfPeriod(list, '3M', null).map((m) => m.amount)).toEqual([16, 17, 18])
    expect(monthsOfPeriod(list, '1Y', null)).toHaveLength(12)
    expect(monthsOfPeriod(list, 'ALL', null)).toHaveLength(18)
  })

  it('takes only the months a range of days covers entirely', () => {
    const got = monthsOfPeriod(list, '1Y', ['2025-03-15', '2025-06-30'])
    expect(got.map((m) => m.month)).toEqual(['2025-04', '2025-05', '2025-06'])
  })

  it('reads the last day of February in a leap year', () => {
    const feb = history([1, 2, 3], '2028-01')
    expect(monthsOfPeriod(feb, 'ALL', ['2028-02-01', '2028-02-29']).map((m) => m.month)).toEqual(['2028-02'])
    expect(monthsOfPeriod(feb, 'ALL', ['2028-02-01', '2028-02-28'])).toEqual([])
  })

  it('is empty for a range inside one month', () => {
    expect(monthsOfPeriod(list, 'ALL', ['2025-03-05', '2025-03-20'])).toEqual([])
  })
})

describe('rollingAverage', () => {
  it('averages a month with the eleven before it', () => {
    const avg = rollingAverage(history([...Array(12).fill(10), 130]))
    expect(avg[11]).toBe(10)
    // the window slides: the first 10 leaves, 130 enters
    expect(avg[12]).toBeCloseTo((11 * 10 + 130) / 12)
  })

  it('averages over the months it has while the history is shorter than the window', () => {
    expect(rollingAverage(history([30, 0, 60]))).toEqual([30, 15, 30])
  })

  it('is empty for an empty history', () => {
    expect(rollingAverage([])).toEqual([])
  })
})

describe('summarizePeriod', () => {
  it('sums the period and averages it over its months', () => {
    expect(summarizePeriod(history([10, 20, 60]))).toEqual({ total: 90, monthlyAverage: 30, months: 3 })
  })

  it('has no average for an empty period', () => {
    expect(summarizePeriod([])).toEqual({ total: 0, monthlyAverage: null, months: 0 })
  })
})

describe('periodLabel', () => {
  it('names one month, or the first and the last of a span', () => {
    expect(periodLabel(history([1], '2026-10'))).toMatch(/2026/)
    const span = periodLabel(history([1, 2, 3], '2026-08'))
    expect(span).toContain('–')
    expect(periodLabel([])).toBe('')
  })
})
