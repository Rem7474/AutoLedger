import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { dayKey, monthKey, reportingToday, reportingYear, reportingTimezone, setReportingTimezone } from './reportingTime'
import { dayKey as datesDayKey, todayIso } from './dates'
import { currentYearMonth, presetRange, todayIso as drivesTodayIso, unitRange } from './drives'
import { groupByDay } from './dayGroups'
import { nextOccurrenceDate } from './expenses'
import { periodBounds, periodFromQuery, presetPeriod, stepPeriod, type Period } from './period'

const PARIS = 'Europe/Paris'
const UTC = 'UTC'
const LA = 'America/Los_Angeles'

afterEach(() => {
  setReportingTimezone(null)
  vi.useRealTimers()
})

describe('the day an instant belongs to, by reporting zone', () => {
  // instant, zone, expected day
  const cases: [string, string, string][] = [
    // midnight: 23:30 UTC is already the next day in Paris (UTC+1), still the same day in Los Angeles
    ['2026-11-15T23:30:00Z', PARIS, '2026-11-16'],
    ['2026-11-15T23:30:00Z', UTC, '2026-11-15'],
    ['2026-11-15T23:30:00Z', LA, '2026-11-15'],
    ['2026-11-16T07:59:00Z', LA, '2026-11-15'],
    ['2026-11-16T08:00:00Z', LA, '2026-11-16'],
    // month and year ends
    ['2026-11-30T23:30:00Z', PARIS, '2026-12-01'],
    ['2026-11-30T23:30:00Z', UTC, '2026-11-30'],
    ['2026-12-31T23:30:00Z', PARIS, '2027-01-01'],
    ['2027-01-01T03:00:00Z', LA, '2026-12-31'],
    ['2028-02-28T23:30:00Z', PARIS, '2028-02-29'],
    // Paris spring forward (29 March 2026, 01:00 UTC) and fall back (25 October 2026, 01:00 UTC)
    ['2026-03-28T22:59:00Z', PARIS, '2026-03-28'],
    ['2026-03-28T23:00:00Z', PARIS, '2026-03-29'],
    ['2026-03-29T21:59:00Z', PARIS, '2026-03-29'],
    ['2026-03-29T22:00:00Z', PARIS, '2026-03-30'],
    ['2026-10-24T21:59:00Z', PARIS, '2026-10-24'],
    ['2026-10-24T22:00:00Z', PARIS, '2026-10-25'],
    ['2026-10-25T22:59:00Z', PARIS, '2026-10-25'],
    ['2026-10-25T23:00:00Z', PARIS, '2026-10-26'],
    // Los Angeles spring forward (8 March 2026, 10:00 UTC)
    ['2026-03-08T07:59:00Z', LA, '2026-03-07'],
    ['2026-03-08T08:00:00Z', LA, '2026-03-08'],
    ['2026-03-09T06:59:00Z', LA, '2026-03-08'],
    ['2026-03-09T07:00:00Z', LA, '2026-03-09'],
  ]
  it.each(cases)('%s in %s is %s', (instant, zone, expected) => {
    setReportingTimezone(zone)
    expect(dayKey(new Date(instant))).toBe(expected)
    expect(dayKey(instant)).toBe(expected)
    expect(monthKey(instant)).toBe(expected.slice(0, 7))
  })

  it('keeps a bare day as it is whatever the zone', () => {
    setReportingTimezone(LA)
    expect(datesDayKey('2026-05-02')).toBe('2026-05-02')
  })

  it('falls back to the browser zone when none is set or the name is unknown', () => {
    const browser = Intl.DateTimeFormat().resolvedOptions().timeZone
    expect(reportingTimezone()).toBe(browser)
    setReportingTimezone('Not/AZone')
    expect(reportingTimezone()).toBe(browser)
    setReportingTimezone(PARIS)
    expect(reportingTimezone()).toBe(PARIS)
    setReportingTimezone(null)
    expect(reportingTimezone()).toBe(browser)
  })
})

describe('"now" follows the reporting zone, not the clock of the browser', () => {
  // Sunday 15 November 2026 23:30 UTC: Monday 16 00:30 in Paris, Sunday 15:30 in Los Angeles
  const midnight = '2026-11-15T23:30:00Z'
  const at = (iso: string, zone: string) => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(iso))
    setReportingTimezone(zone)
  }

  it.each([
    [PARIS, '2026-11-16', '2026-11'],
    [UTC, '2026-11-15', '2026-11'],
    [LA, '2026-11-15', '2026-11'],
  ])('today in %s at the midnight instant', (zone, today, month) => {
    at(midnight, zone)
    expect(reportingToday()).toBe(today)
    expect(todayIso()).toBe(today)
    expect(drivesTodayIso()).toBe(today)
    expect(dayKey()).toBe(today)
    expect(currentYearMonth()).toBe(month)
  })

  it('moves the month and the year with the zone', () => {
    at('2026-12-31T23:30:00Z', PARIS)
    expect(reportingToday()).toBe('2027-01-01')
    expect(reportingYear()).toBe(2027)
    expect(currentYearMonth()).toBe('2027-01')
    at('2026-12-31T23:30:00Z', LA)
    expect(reportingYear()).toBe(2026)
    expect(currentYearMonth()).toBe('2026-12')
  })

  it('ends the current month and the current year on the reporting today', () => {
    const month: Period = { kind: 'months', count: 1, end: null }
    at('2026-11-30T23:30:00Z', PARIS)
    expect(periodBounds(month, new Date())).toEqual({ from: '2026-12-01', to: '2026-12-01' })
    at('2026-11-30T23:30:00Z', LA)
    expect(periodBounds(month, new Date())).toEqual({ from: '2026-11-01', to: '2026-11-30' })
    at('2026-12-31T23:30:00Z', PARIS)
    expect(periodBounds({ kind: 'year', year: 2027 }, new Date())).toEqual({ from: '2027-01-01', to: '2027-01-01' })
    expect(periodBounds(presetPeriod('thisYear', new Date()), new Date())).toEqual({ from: '2027-01-01', to: '2027-01-01' })
  })

  it('a record dated by the server just after midnight falls inside the period bar', () => {
    // the server dates this instant in Paris: 16 November
    const created = '2026-11-15T23:30:00Z'
    at(created, PARIS)
    const { from, to } = periodBounds({ kind: 'months', count: 1, end: null }, new Date())
    const day = dayKey(created)
    expect(day).toBe('2026-11-16')
    expect(day >= from! && day <= to!).toBe(true)
  })

  it('names the week by the reporting day (Monday to Sunday)', () => {
    at(midnight, PARIS)
    expect(unitRange('WEEK', todayIso())).toEqual({ from: '2026-11-16', to: '2026-11-22' })
    at(midnight, UTC)
    expect(unitRange('WEEK', todayIso())).toEqual({ from: '2026-11-09', to: '2026-11-15' })
  })

  it('cuts the ready-made ranges from the reporting day', () => {
    at('2026-03-31T22:30:00Z', PARIS) // 1 April 00:30 in Paris
    expect(presetRange('lastMonth', new Date())).toEqual({ from: '2026-03-01', to: '2026-03-31' })
    expect(presetRange('last30', new Date())).toEqual({ from: '2026-03-03', to: '2026-04-01' })
    at('2026-03-31T22:30:00Z', LA) // 31 March 15:30
    expect(presetRange('lastMonth', new Date())).toEqual({ from: '2026-02-01', to: '2026-02-28' })
    expect(presetRange('thisYear', new Date())).toEqual({ from: '2026-01-01', to: '2026-03-31' })
  })

  it('accepts a URL period up to the reporting today and refuses the next day', () => {
    at(midnight, LA)
    expect(periodFromQuery({ from: '2026-11-01', to: '2026-11-16' }, new Date())).toBeNull()
    at(midnight, PARIS)
    expect(periodFromQuery({ from: '2026-11-01', to: '2026-11-16' }, new Date())).toEqual({ kind: 'custom', from: '2026-11-01', to: '2026-11-16' })
  })

  it('steps a free range back from the reporting today', () => {
    at(midnight, PARIS)
    const range: Period = { kind: 'custom', from: '2026-11-10', to: '2026-11-16' }
    expect(stepPeriod(range, 1, new Date())).toBe(range)
    expect(stepPeriod(range, -1, new Date())).toEqual({ kind: 'custom', from: '2026-11-03', to: '2026-11-09' })
  })

  it('finds the next yearly occurrence from the reporting today', () => {
    at('2026-12-31T23:30:00Z', PARIS)
    expect(nextOccurrenceDate(1, 1, new Date())).toBe('2027-01-01')
    at('2026-12-31T23:30:00Z', LA)
    expect(nextOccurrenceDate(1, 1, new Date())).toBe('2027-01-01')
    expect(nextOccurrenceDate(12, 31, new Date())).toBe('2026-12-31')
    at('2026-12-31T23:30:00Z', PARIS)
    expect(nextOccurrenceDate(12, 31, new Date())).toBe('2027-12-31')
  })
})

describe('journal day groups use the reporting day', () => {
  const rows = [{ t: '2026-11-15T23:30:00Z' }, { t: '2026-11-15T12:00:00Z' }, { t: '2026-11-15T22:30:00Z' }]
  it.each([
    [PARIS, ['2026-11-16', '2026-11-15']],
    [UTC, ['2026-11-15']],
    [LA, ['2026-11-15']],
  ])('in %s', (zone, days) => {
    setReportingTimezone(zone)
    expect(groupByDay(rows, (r) => r.t).map((g) => g.day)).toEqual(days)
  })
})
