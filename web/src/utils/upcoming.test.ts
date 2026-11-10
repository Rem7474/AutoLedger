import { describe, expect, it } from 'vitest'
import type { MaintenanceReminder } from '@/services/api'
import { carpoolOwed, undatedReminders, upcomingItems } from '@/utils/upcoming'

const reminder = (id: string, status: MaintenanceReminder['status'], extra: Partial<MaintenanceReminder> = {}) =>
  ({ id, title: id, status, interval_km: 10000, remaining_km: 5000, ...extra }) as MaintenanceReminder
const worn = (tone: 'ok' | 'warning' | 'danger') => ({ fitted: 4, mostWorn: { pct: 85, position: 'FL', name: 'Brand Model', tone }, nextReplacement: null })

describe('upcomingItems', () => {
  it('is empty with nothing to show', () => {
    expect(upcomingItems([], null)).toEqual([])
    expect(upcomingItems([], worn('ok'))).toEqual([])
  })

  it('lists overdue, then due soon, then the next planned reminder', () => {
    const items = upcomingItems([reminder('plan', 'OK'), reminder('soon', 'DUE_SOON', { remaining_km: 500 }), reminder('late', 'OVERDUE', { remaining_km: -100 })], null)
    expect(items.map((i) => [i.reminder?.id, i.tone])).toEqual([
      ['late', 'late'],
      ['soon', 'soon'],
      ['plan', 'plain'],
    ])
  })

  it('does not repeat a due-soon reminder as the next planned one', () => {
    const items = upcomingItems([reminder('soon', 'DUE_SOON', { remaining_km: 9000 })], null)
    expect(items).toHaveLength(1)
  })

  it('keeps one planned reminder only', () => {
    const items = upcomingItems([reminder('a', 'OK', { remaining_km: 8000 }), reminder('b', 'OK', { remaining_km: 2000 })], null)
    expect(items.map((i) => i.reminder?.id)).toEqual(['b'])
  })

  it('adds the most worn tire once it is a warning, with its own tone', () => {
    expect(upcomingItems([], worn('warning')).map((i) => [i.kind, i.tone, i.to])).toEqual([['tire', 'soon', '/tires']])
    expect(upcomingItems([reminder('soon', 'DUE_SOON')], worn('danger')).map((i) => i.key)).toEqual(['tire', 'reminder-soon'])
  })
})

describe('undated reminders', () => {
  const undated = (id: string) => reminder(id, 'OK', { interval_km: null, remaining_km: null })

  it('counts the reminders in order that have neither a date nor a mileage', () => {
    const list = [undated('a'), undated('b'), reminder('c', 'OK'), reminder('d', 'OVERDUE')]
    expect(undatedReminders(list).map((r) => r.id)).toEqual(['a', 'b'])
  })

  it('lists them as one row, after the planned ones, instead of saying nothing is planned', () => {
    const items = upcomingItems([undated('a'), undated('b')], null)
    expect(items.map((i) => [i.kind, i.count, i.to])).toEqual([['undated', 2, '/maintenance?tab=REMINDERS']])
    expect(upcomingItems([undated('a'), reminder('plan', 'OK')], null).map((i) => i.kind)).toEqual(['reminder', 'undated'])
  })
})

describe('upcoming extras', () => {
  const lease = (remainingMonths: number, extra = {}) => ({ endDate: new Date('2027-03-01'), remainingMonths, isEnded: false, isNotStarted: false, ...extra })

  it('lists what the passengers still owe, and nothing once it is settled', () => {
    const [item] = upcomingItems([], null, { carpoolOwed: 24.5 })
    expect([item.kind, item.amount, item.to, item.tone]).toEqual(['carpool', 24.5, '/carpools', 'plain'])
    expect(upcomingItems([], null, { carpoolOwed: 0 })).toEqual([])
    expect(upcomingItems([], null, { carpoolOwed: null })).toEqual([])
  })

  it('lists the end of a lease inside the horizon, as a warning when it is close', () => {
    expect(upcomingItems([], null, { lease: lease(5) }).map((i) => [i.kind, i.tone, i.months])).toEqual([['lease', 'plain', 5]])
    expect(upcomingItems([], null, { lease: lease(3) }).map((i) => i.tone)).toEqual(['soon'])
  })

  it('leaves a lease out when it is far, over, not started, or without an end date', () => {
    expect(upcomingItems([], null, { lease: lease(7) })).toEqual([])
    expect(upcomingItems([], null, { lease: lease(2, { isEnded: true }) })).toEqual([])
    expect(upcomingItems([], null, { lease: lease(2, { isNotStarted: true }) })).toEqual([])
    expect(upcomingItems([], null, { lease: lease(2, { endDate: null }) })).toEqual([])
    expect(upcomingItems([], null, { lease: null })).toEqual([])
  })
})

describe('carpoolOwed', () => {
  it('is what the fair shares exceed the payments by', () => {
    expect(carpoolOwed({ total_passengers_share: 100, total_revenue: 76 })).toBe(24)
  })

  it('is null when settled, overpaid or unknown', () => {
    expect(carpoolOwed({ total_passengers_share: 100, total_revenue: 100 })).toBeNull()
    expect(carpoolOwed({ total_passengers_share: 100, total_revenue: 120 })).toBeNull()
    expect(carpoolOwed(null)).toBeNull()
  })
})
