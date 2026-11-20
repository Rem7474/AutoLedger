import { describe, expect, it } from 'vitest'
import { previousMonthOf, todayHero } from '@/utils/todayHero'

const month = { month: '2026-11', total: 300, scheduled: 100, distance_km: 400, energy: 120, tolls: 30, tires: 0, maintenance: 0, insurance: 100, financing: 0, other: 50, carpool_revenue: 0 }

describe('todayHero', () => {
  it('has nothing without a month', () => {
    expect(todayHero(null, 'EUR')).toBeNull()
    expect(todayHero(undefined, 'EUR')).toBeNull()
  })

  it('reads the spent amount and the scheduled part of the month in progress', () => {
    const hero = todayHero(month, 'EUR')!
    expect(hero.spent).toBe(200)
    expect(hero.scheduled).toBe(100)
    expect(hero.previous).toBeNull()
  })

  it('takes what the passengers paid back off the spending', () => {
    expect(todayHero({ ...month, carpool_revenue: 40 }, 'EUR')!.spent).toBe(160)
  })

  it('orders the themes from the largest and leaves the empty ones out', () => {
    const hero = todayHero(month, 'EUR')!
    expect(hero.segments.map((s) => s.key)).toEqual(['energy', 'insurance', 'other', 'tolls'])
    expect(hero.segments[0].sharePct).toBeCloseTo(40)
  })

  it('reads the month in progress against the last full month, not against itself', () => {
    const october = { ...month, month: '2026-10', total: 500, scheduled: 0, distance_km: 1000 }
    const hero = todayHero(month, 'EUR', [october, month])!
    expect(hero.previous).toEqual({ month: '2026-10', total: 500, share: 0.4 })
  })
})

describe('previousMonthOf', () => {
  const full = (m: string, over = {}) => ({ ...month, month: m, total: 400, scheduled: 0, distance_km: 800, ...over })

  it('is net of what the passengers paid', () => {
    expect(previousMonthOf([full('2026-10', { carpool_revenue: 80 })], '2026-11', 160, 'EUR')).toMatchObject({ total: 320, share: 0.5 })
  })

  it('takes the latest earlier month, whatever the order of the list', () => {
    expect(previousMonthOf([full('2026-09'), full('2026-10', { total: 200 }), full('2026-11')], '2026-11', 0, 'EUR')).toMatchObject({ month: '2026-10', total: 200 })
    expect(previousMonthOf([full('2026-10', { total: 200 }), full('2026-09')], '2026-11', 0, 'EUR')!.month).toBe('2026-10')
  })

  it('skips a month with no spending, and has nothing when none is left', () => {
    expect(previousMonthOf([full('2026-09'), full('2026-10', { total: 0 })], '2026-11', 0, 'EUR')!.month).toBe('2026-09')
    expect(previousMonthOf([full('2026-11')], '2026-11', 0, 'EUR')).toBeNull()
    expect(previousMonthOf(undefined, '2026-11', 0, 'EUR')).toBeNull()
  })

  it('goes past 1 when the month in progress has already spent more', () => {
    expect(previousMonthOf([full('2026-10')], '2026-11', 500, 'EUR')!.share).toBeCloseTo(1.25)
  })
})
