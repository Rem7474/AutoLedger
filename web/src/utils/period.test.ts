import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { setLocale } from '@/i18n'
import {
  addMonths,
  canStep,
  isRoundedToMonths,
  parsePeriod,
  periodBounds,
  periodFromLegacyRange,
  periodFromQuery,
  periodLabel,
  periodToQuery,
  presetOf,
  presetPeriod,
  stepPeriod,
  touchedMonths,
  PERIOD_PRESETS,
  type Period,
} from './period'

// 15 November 2026, local time
const NOW = new Date(2026, 10, 15, 12)
const months = (count: 1 | 3 | 6 | 12, end: string | null = null): Period => ({ kind: 'months', count, end })

describe('period bounds', () => {
  it('runs the current month up to today', () => {
    expect(periodBounds(months(1), NOW)).toEqual({ from: '2026-11-01', to: '2026-11-15' })
  })
  it('counts the current month in the last N months, across a year end', () => {
    expect(periodBounds(months(3), NOW)).toEqual({ from: '2026-09-01', to: '2026-11-15' })
    expect(periodBounds(months(12), NOW)).toEqual({ from: '2025-12-01', to: '2026-11-15' })
    expect(periodBounds(months(3), new Date(2026, 0, 10))).toEqual({ from: '2025-11-01', to: '2026-01-10' })
  })
  it('ends a past period on the last day of its month, February of a leap year included', () => {
    expect(periodBounds(months(1, '2026-10'), NOW)).toEqual({ from: '2026-10-01', to: '2026-10-31' })
    expect(periodBounds(months(1, '2024-02'), NOW)).toEqual({ from: '2024-02-01', to: '2024-02-29' })
    expect(periodBounds(months(6, '2026-06'), NOW)).toEqual({ from: '2026-01-01', to: '2026-06-30' })
  })
  it('takes a calendar year to today for the current one, to 31 December for a past one', () => {
    expect(periodBounds({ kind: 'year', year: 2026 }, NOW)).toEqual({ from: '2026-01-01', to: '2026-11-15' })
    expect(periodBounds({ kind: 'year', year: 2025 }, NOW)).toEqual({ from: '2025-01-01', to: '2025-12-31' })
  })
  it('keeps a free range as it is, and the whole history open', () => {
    expect(periodBounds({ kind: 'custom', from: '2026-03-12', to: '2026-04-04' }, NOW)).toEqual({ from: '2026-03-12', to: '2026-04-04' })
    expect(periodBounds({ kind: 'all' }, NOW)).toEqual({})
  })
})

describe('months touched', () => {
  it('rounds a free range outwards to the months it touches', () => {
    expect(touchedMonths({ kind: 'custom', from: '2026-03-12', to: '2026-04-04' }, NOW)).toEqual({ from: '2026-03', to: '2026-04' })
    expect(touchedMonths({ kind: 'all' }, NOW)).toEqual({ from: null, to: null })
  })
  it('says when a range was rounded', () => {
    expect(isRoundedToMonths({ kind: 'custom', from: '2026-03-12', to: '2026-04-04' })).toBe(true)
    expect(isRoundedToMonths({ kind: 'custom', from: '2026-03-01', to: '2026-04-30' })).toBe(false)
    expect(isRoundedToMonths(months(3))).toBe(false)
  })
  it('adds months across year ends', () => {
    expect(addMonths('2026-01', -1)).toBe('2025-12')
    expect(addMonths('2026-11', 3)).toBe('2027-02')
  })
})

describe('stepping with the arrows', () => {
  it('steps a month by a month, and back to the live one when it reaches the current month', () => {
    const back = stepPeriod(months(1), -1, NOW)
    expect(back).toEqual(months(1, '2026-10'))
    expect(stepPeriod(back, 1, NOW)).toEqual(months(1))
  })
  it('steps N months by N months', () => {
    expect(stepPeriod(months(3), -1, NOW)).toEqual(months(3, '2026-08'))
    expect(stepPeriod(months(3, '2026-08'), 1, NOW)).toEqual(months(3))
    expect(stepPeriod(months(12), -1, NOW)).toEqual(months(12, '2025-11'))
  })
  it('does not go past the current month', () => {
    const live = months(3)
    expect(stepPeriod(live, 1, NOW)).toBe(live)
    expect(canStep(months(1), 1, NOW)).toBe(false)
    expect(canStep(months(1), -1, NOW)).toBe(true)
    // 3 months ending in October would end in January
    expect(canStep(months(3, '2026-10'), 1, NOW)).toBe(false)
  })
  it('steps a year, but not past the current one', () => {
    expect(stepPeriod({ kind: 'year', year: 2026 }, -1, NOW)).toEqual({ kind: 'year', year: 2025 })
    expect(canStep({ kind: 'year', year: 2026 }, 1, NOW)).toBe(false)
  })
  it('moves a free range by its own length', () => {
    const range: Period = { kind: 'custom', from: '2026-03-10', to: '2026-03-16' }
    expect(stepPeriod(range, -1, NOW)).toEqual({ kind: 'custom', from: '2026-03-03', to: '2026-03-09' })
    expect(stepPeriod(range, 1, NOW)).toEqual({ kind: 'custom', from: '2026-03-17', to: '2026-03-23' })
    expect(canStep({ kind: 'custom', from: '2026-11-10', to: '2026-11-15' }, 1, NOW)).toBe(false)
  })
  it('does not step the whole history', () => {
    expect(canStep({ kind: 'all' }, -1, NOW)).toBe(false)
    expect(canStep({ kind: 'all' }, 1, NOW)).toBe(false)
  })
})

describe('presets', () => {
  it('knows each preset and recognises it again', () => {
    for (const key of PERIOD_PRESETS) expect(presetOf(presetPeriod(key, NOW), NOW)).toBe(key)
  })
  it('does not call a stepped period a preset, except the one that is', () => {
    expect(presetOf(months(3, '2026-08'), NOW)).toBeNull()
    expect(presetOf(months(1, '2026-10'), NOW)).toBe('lastMonth')
    expect(presetOf({ kind: 'custom', from: '2026-03-01', to: '2026-03-31' }, NOW)).toBeNull()
  })
})

describe('labels', () => {
  beforeAll(() => setLocale('fr'))
  afterAll(() => setLocale('en'))
  it('says the month, the count, the run, the year and the range', () => {
    expect(periodLabel(months(1), NOW)).toBe('Novembre 2026')
    expect(periodLabel(months(1, '2026-10'), NOW)).toBe('Octobre 2026')
    expect(periodLabel(months(3), NOW)).toBe('3 derniers mois')
    expect(periodLabel(months(3, '2026-08'), NOW)).toBe('Juin 2026 – Août 2026')
    expect(periodLabel({ kind: 'year', year: 2025 }, NOW)).toBe('2025')
    expect(periodLabel({ kind: 'all' }, NOW)).toBe('Tout')
    expect(periodLabel({ kind: 'custom', from: '2026-03-12', to: '2026-04-04' }, NOW)).toMatch(/12 mars.*4 avr/)
  })
})

describe('the URL', () => {
  const cases: Period[] = [months(1), months(3), months(12), months(3, '2026-08'), { kind: 'year', year: 2025 }, { kind: 'all' }, { kind: 'custom', from: '2026-03-12', to: '2026-04-04' }]
  it('reads back what it wrote', () => {
    for (const p of cases) expect(periodFromQuery(periodToQuery(p), NOW)).toEqual(p)
  })
  it('writes the short forms', () => {
    expect(periodToQuery(months(3))).toEqual({ period: '3M' })
    expect(periodToQuery(months(3, '2026-08'))).toEqual({ period: '3M', end: '2026-08' })
    expect(periodToQuery({ kind: 'year', year: 2025 })).toEqual({ period: 'Y2025' })
  })
  it('reads the first value of a repeated parameter', () => {
    expect(periodFromQuery({ period: ['6M', '1M'] }, NOW)).toEqual(months(6))
  })
  it('names no period when the URL names none or names it wrongly', () => {
    expect(periodFromQuery({}, NOW)).toBeNull()
    expect(periodFromQuery({ period: '5M' }, NOW)).toBeNull()
    expect(periodFromQuery({ period: 'nonsense' }, NOW)).toBeNull()
    expect(periodFromQuery({ period: '3M', end: '2027-03' }, NOW)).toBeNull()
    expect(periodFromQuery({ period: '3M', end: '2026-13' }, NOW)).toBeNull()
    expect(periodFromQuery({ period: 'Y2031' }, NOW)).toBeNull()
    expect(periodFromQuery({ from: '2026-02-30', to: '2026-03-01' }, NOW)).toBeNull()
    expect(periodFromQuery({ from: '2026-04-04', to: '2026-03-12' }, NOW)).toBeNull()
    expect(periodFromQuery({ from: '2026-03-12', to: '2026-12-31' }, NOW)).toBeNull()
  })
  it('reads an end in the current month as the live period', () => {
    expect(periodFromQuery({ period: '3M', end: '2026-11' }, NOW)).toEqual(months(3))
  })
})

describe('stored values', () => {
  it('accepts what it wrote and refuses anything else', () => {
    expect(parsePeriod(months(6, '2026-05'), NOW)).toEqual(months(6, '2026-05'))
    expect(parsePeriod({ kind: 'months', count: 4, end: null }, NOW)).toBeNull()
    expect(parsePeriod({ kind: 'months', count: 3, end: '2027-01' }, NOW)).toBeNull()
    expect(parsePeriod({ kind: 'year', year: 'x' }, NOW)).toBeNull()
    expect(parsePeriod('3M', NOW)).toBeNull()
    expect(parsePeriod(null, NOW)).toBeNull()
  })
  it('maps the keys of the old home selector', () => {
    expect(periodFromLegacyRange('1M')).toEqual(months(1))
    expect(periodFromLegacyRange('1Y')).toEqual(months(12))
    expect(periodFromLegacyRange('ALL')).toEqual({ kind: 'all' })
    expect(periodFromLegacyRange('2Y')).toBeNull()
    expect(periodFromLegacyRange(null)).toBeNull()
  })
})
