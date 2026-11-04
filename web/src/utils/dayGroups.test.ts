import { describe, expect, it } from 'vitest'
import { dayLabel, groupByDay } from './dayGroups'

const at = (iso: string) => ({ start: new Date(iso).toISOString() })
// Local times, so the groups do not depend on the zone the tests run in
const local = (y: number, m: number, d: number, h: number) => ({ start: new Date(y, m - 1, d, h).toISOString() })

describe('groupByDay', () => {
  it('has no group for an empty list', () => {
    expect(groupByDay([], (x: { start: string }) => x.start)).toEqual([])
  })

  it('keeps the order and starts a group at each new day', () => {
    const a = local(2026, 11, 3, 18)
    const b = local(2026, 11, 3, 8)
    const c = local(2026, 11, 2, 22)
    const groups = groupByDay([a, b, c], (x) => x.start)
    expect(groups.map((g) => [g.day, g.items.length])).toEqual([['2026-11-03', 2], ['2026-11-02', 1]])
    expect(groups[0].items).toEqual([a, b])
  })

  it('opens a new group when a day comes back after another', () => {
    const groups = groupByDay([local(2026, 11, 3, 9), local(2026, 11, 2, 9), local(2026, 11, 3, 7)], (x) => x.start)
    expect(groups.map((g) => g.day)).toEqual(['2026-11-03', '2026-11-02', '2026-11-03'])
  })

  it('reads the time of whatever the list holds', () => {
    expect(groupByDay([at('2026-05-01T12:00:00Z')], (x) => x.start)).toHaveLength(1)
  })
})

describe('dayLabel', () => {
  const labels = { today: 'Today', yesterday: 'Yesterday' }
  const format = (date: Date, withYear: boolean) => `${date.getDate()}/${date.getMonth() + 1}${withYear ? `/${date.getFullYear()}` : ''}`

  it('names today and yesterday', () => {
    expect(dayLabel('2026-11-04', '2026-11-04', labels, format)).toBe('Today')
    expect(dayLabel('2026-11-03', '2026-11-04', labels, format)).toBe('Yesterday')
  })

  it('knows yesterday across a month and a year', () => {
    expect(dayLabel('2026-10-31', '2026-11-01', labels, format)).toBe('Yesterday')
    expect(dayLabel('2025-12-31', '2026-01-01', labels, format)).toBe('Yesterday')
  })

  it('gives the date of an earlier day, with the year only when it is not this one', () => {
    expect(dayLabel('2026-10-12', '2026-11-04', labels, format)).toBe('12/10')
    expect(dayLabel('2025-10-12', '2026-11-04', labels, format)).toBe('12/10/2025')
  })

  it('does not call tomorrow yesterday', () => {
    expect(dayLabel('2026-11-05', '2026-11-04', labels, format)).toBe('5/11')
  })
})
