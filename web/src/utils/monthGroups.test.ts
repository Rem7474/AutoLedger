import { describe, expect, it } from 'vitest'
import { groupByMonth, monthDistances, monthSectionLabel, pageSlice } from './monthGroups'

// Dates given as plain days are read as they are, whatever the zone the tests run in
const row = (date: string, odometer = 0) => ({ date, odometer })

describe('groupByMonth', () => {
  it('has no section for an empty list', () => {
    expect(groupByMonth([], (r: { date: string }) => r.date)).toEqual([])
  })

  it('starts a section at each new month and keeps the order', () => {
    const a = row('2026-11-20')
    const b = row('2026-11-02')
    const c = row('2026-10-30')
    const sections = groupByMonth([a, b, c], (r) => r.date)
    expect(sections.map((s) => [s.month, s.items.length])).toEqual([['2026-11', 2], ['2026-10', 1]])
    expect(sections[0].items).toEqual([a, b])
  })

  it('opens a new section when a month comes back after another', () => {
    const sections = groupByMonth([row('2026-11-03'), row('2026-10-03'), row('2026-11-01')], (r) => r.date)
    expect(sections.map((s) => s.month)).toEqual(['2026-11', '2026-10', '2026-11'])
  })

  it('tells the years apart', () => {
    expect(groupByMonth([row('2026-01-05'), row('2025-01-05')], (r) => r.date).map((s) => s.month)).toEqual(['2026-01', '2025-01'])
  })
})

describe('monthSectionLabel', () => {
  it('gives the month alone in the current year and with the year in another', () => {
    expect(monthSectionLabel('2026-10', '2026-11-23', 'en-GB')).toBe('October')
    expect(monthSectionLabel('2025-12', '2026-11-23', 'en-GB')).toBe('December 2025')
  })
})

describe('pageSlice', () => {
  const items = [1, 2, 3, 4, 5]
  it('cuts the first page and says more remain', () => {
    expect(pageSlice(items, 2, 1)).toEqual({ visible: [1, 2], more: true })
  })
  it('grows with each page until the whole list shows', () => {
    expect(pageSlice(items, 2, 2)).toEqual({ visible: [1, 2, 3, 4], more: true })
    expect(pageSlice(items, 2, 3)).toEqual({ visible: items, more: false })
  })
  it('says nothing remains when the list fits exactly', () => {
    expect(pageSlice(items, 5, 1).more).toBe(false)
  })
  it('never shows less than one page', () => {
    expect(pageSlice(items, 2, 0).visible).toEqual([1, 2])
  })
  it('handles an empty list', () => {
    expect(pageSlice([], 3, 1)).toEqual({ visible: [], more: false })
  })
})

describe('monthDistances', () => {
  it('has no figure for an empty history', () => {
    expect(monthDistances([]).size).toBe(0)
  })

  it('counts the first month from its own lowest mileage and the next from the previous highest', () => {
    const m = monthDistances([row('2026-09-02', 1000), row('2026-09-28', 1400), row('2026-10-15', 1900), row('2026-11-01', 2000), row('2026-11-20', 2600)])
    expect([...m]).toEqual([['2026-09', 400], ['2026-10', 500], ['2026-11', 700]])
  })

  it('gives no figure to a month with a single reading and nothing before it', () => {
    expect(monthDistances([row('2026-10-15', 1900)]).has('2026-10')).toBe(false)
  })

  it('gives no figure to a month that adds nothing', () => {
    const m = monthDistances([row('2026-09-02', 1000), row('2026-10-02', 1000)])
    expect(m.has('2026-10')).toBe(false)
  })

  it('reads the history whatever its order', () => {
    const m = monthDistances([row('2026-10-15', 1900), row('2026-09-02', 1000)])
    expect(m.get('2026-10')).toBe(900)
  })

  it('does not go back when a later month holds a lower mileage', () => {
    const m = monthDistances([row('2026-09-02', 5000), row('2026-10-02', 4000), row('2026-11-02', 5200)])
    expect(m.has('2026-10')).toBe(false)
    expect(m.get('2026-11')).toBe(200)
  })
})
