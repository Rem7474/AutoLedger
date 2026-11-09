import { describe, expect, it } from 'vitest'
import type { MaintenanceReminder } from '@/services/api'
import { needsFirstReminder, upcomingItems } from '@/utils/upcoming'

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

describe('needsFirstReminder', () => {
  it('invites to plan one when no reminder has a schedule', () => {
    expect(needsFirstReminder([])).toBe(true)
    expect(needsFirstReminder([reminder('x', 'OK', { interval_km: null, remaining_km: null })])).toBe(true)
  })

  it('does not when a reminder is scheduled, however far', () => {
    expect(needsFirstReminder([reminder('x', 'OK')])).toBe(false)
    expect(needsFirstReminder([reminder('x', 'OVERDUE')])).toBe(false)
  })
})
