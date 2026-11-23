import { dayKey } from '@/utils/dates'
import { monthHeader } from '@/utils/dayGroups'

export interface MonthSection<T> {
  /** The calendar month in the reporting timezone, YYYY-MM. */
  month: string
  items: T[]
}

/** Splits a list sorted by time into runs of the same reporting month, keeping the order. */
export function groupByMonth<T>(items: T[], dateOf: (item: T) => string): MonthSection<T>[] {
  const sections: MonthSection<T>[] = []
  for (const item of items) {
    const month = dayKey(dateOf(item)).slice(0, 7)
    const last = sections[sections.length - 1]
    if (last && last.month === month) last.items.push(item)
    else sections.push({ month, items: [item] })
  }
  return sections
}

/** The header of a month: "November", with the year once it is not this one. */
export function monthSectionLabel(month: string, today: string, locale: string): string {
  return monthHeader(month, today, (date, withYear) => date.toLocaleDateString(locale, { month: 'long', ...(withYear ? { year: 'numeric' } : {}) }))
}

/** The rows shown once `pages` pages of `pageSize` rows were asked for, and whether more remain. */
export function pageSlice<T>(items: T[], pageSize: number, pages: number): { visible: T[]; more: boolean } {
  const count = Math.max(1, pages) * pageSize
  return { visible: items.slice(0, count), more: items.length > count }
}

/**
 * The distance each month adds to a mileage: the highest mileage of the month minus the highest one before it. The
 * first month of the history has nothing before it and counts from its own lowest mileage; a month that adds nothing
 * (a single reading, or the same mileage twice) has no entry.
 */
export function monthDistances(items: { date: string; odometer: number }[]): Map<string, number> {
  const byMonth = new Map<string, { min: number; max: number }>()
  for (const { date, odometer } of items) {
    const key = dayKey(date).slice(0, 7)
    const cur = byMonth.get(key)
    byMonth.set(key, cur ? { min: Math.min(cur.min, odometer), max: Math.max(cur.max, odometer) } : { min: odometer, max: odometer })
  }
  const result = new Map<string, number>()
  let before: number | null = null
  for (const key of [...byMonth.keys()].sort()) {
    const { min, max } = byMonth.get(key)!
    const distance = max - (before ?? min)
    if (distance > 0) result.set(key, distance)
    before = before === null ? max : Math.max(before, max)
  }
  return result
}
