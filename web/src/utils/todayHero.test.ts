import { describe, expect, it } from 'vitest'
import { lastFullMonthPerKm, todayHero } from '@/utils/todayHero'

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
    expect(hero.perKm).toBeNull()
  })

  it('takes what the passengers paid back off the spending', () => {
    expect(todayHero({ ...month, carpool_revenue: 40 }, 'EUR')!.spent).toBe(160)
  })

  it('orders the themes from the largest and leaves the empty ones out', () => {
    const hero = todayHero(month, 'EUR')!
    expect(hero.segments.map((s) => s.key)).toEqual(['energy', 'insurance', 'other', 'tolls'])
    expect(hero.segments[0].sharePct).toBeCloseTo(40)
  })

  it('reads the cost per km on the last full month, not on the month in progress', () => {
    const october = { ...month, month: '2026-10', total: 500, scheduled: 0, distance_km: 1000 }
    const hero = todayHero(month, 'EUR', [october, month])!
    expect(hero.perKm).toEqual({ month: '2026-10', value: 0.5, distanceKm: 1000 })
  })
})

describe('lastFullMonthPerKm', () => {
  const full = (m: string, over = {}) => ({ ...month, month: m, total: 400, scheduled: 0, distance_km: 800, ...over })

  it('is net of what the passengers paid', () => {
    expect(lastFullMonthPerKm([full('2026-10', { carpool_revenue: 80 })], '2026-11', 'EUR')!.value).toBeCloseTo(0.4)
  })

  it('takes the latest earlier month, whatever the order of the list', () => {
    expect(lastFullMonthPerKm([full('2026-09'), full('2026-10', { distance_km: 400 }), full('2026-11')], '2026-11', 'EUR')).toMatchObject({ month: '2026-10', value: 1 })
    expect(lastFullMonthPerKm([full('2026-10', { distance_km: 400 }), full('2026-09')], '2026-11', 'EUR')!.month).toBe('2026-10')
  })

  it('skips a month with no distance or no spending, and has nothing when none is left', () => {
    expect(lastFullMonthPerKm([full('2026-09'), full('2026-10', { distance_km: 0 })], '2026-11', 'EUR')!.month).toBe('2026-09')
    expect(lastFullMonthPerKm([full('2026-09'), full('2026-10', { total: 0 })], '2026-11', 'EUR')!.month).toBe('2026-09')
    expect(lastFullMonthPerKm([full('2026-11')], '2026-11', 'EUR')).toBeNull()
    expect(lastFullMonthPerKm(undefined, '2026-11', 'EUR')).toBeNull()
  })
})
