import { describe, expect, it } from 'vitest'
import { todayHero } from '@/utils/todayHero'

const month = { month: '2026-11', total: 300, scheduled: 100, distance_km: 400, energy: 120, tolls: 30, tires: 0, maintenance: 0, insurance: 100, financing: 0, other: 50, carpool_revenue: 0 }

describe('todayHero', () => {
  it('has nothing without a month', () => {
    expect(todayHero(null, 'EUR')).toBeNull()
    expect(todayHero(undefined, 'EUR')).toBeNull()
  })

  it('reads the spent amount, the scheduled part and the cost per km on the same month', () => {
    const hero = todayHero(month, 'EUR')!
    expect(hero.spent).toBe(200)
    expect(hero.scheduled).toBe(100)
    expect(hero.perKm).toBeCloseTo(0.5)
  })

  it('takes what the passengers paid back off the spending', () => {
    const hero = todayHero({ ...month, carpool_revenue: 40 }, 'EUR')!
    expect(hero.spent).toBe(160)
    expect(hero.perKm).toBeCloseTo(0.4)
  })

  it('orders the themes from the largest and leaves the empty ones out', () => {
    const hero = todayHero(month, 'EUR')!
    expect(hero.segments.map((s) => s.key)).toEqual(['energy', 'insurance', 'other', 'tolls'])
    expect(hero.segments[0].sharePct).toBeCloseTo(40)
  })

  it('has no cost per km without a distance or without spending', () => {
    expect(todayHero({ ...month, distance_km: 0 }, 'EUR')!.perKm).toBeNull()
    expect(todayHero({ ...month, total: 100, scheduled: 100 }, 'EUR')!.perKm).toBeNull()
  })
})
