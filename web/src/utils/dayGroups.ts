import { toLocalDay } from '@/utils/dates'

export interface DayGroup<T> {
  /** The local calendar day, YYYY-MM-DD. */
  day: string
  items: T[]
}

/** Splits a list sorted by time into runs of the same local day, keeping the order. */
export function groupByDay<T>(items: T[], timeOf: (item: T) => string): DayGroup<T>[] {
  const groups: DayGroup<T>[] = []
  for (const item of items) {
    const day = toLocalDay(timeOf(item))
    const last = groups[groups.length - 1]
    if (last && last.day === day) last.items.push(item)
    else groups.push({ day, items: [item] })
  }
  return groups
}

interface DayLabels {
  today: string
  yesterday: string
}

/**
 * The header of a day: "Today", "Yesterday", then the weekday and date, with the year once it is not this one.
 * `today` is a local YYYY-MM-DD; `format` gives the date in the language of the page.
 */
export function dayLabel(day: string, today: string, labels: DayLabels, format: (date: Date, withYear: boolean) => string): string {
  if (day === today) return labels.today
  const [y, m, d] = day.split('-').map(Number)
  const date = new Date(y, m - 1, d)
  const [ty, tm, td] = today.split('-').map(Number)
  const yesterday = new Date(ty, tm - 1, td - 1)
  if (toLocalDay(yesterday) === day) return labels.yesterday
  return format(date, y !== ty)
}

export interface MonthGroup<G extends { day: string }> {
  /** The local calendar month, YYYY-MM. */
  month: string
  days: G[]
}

/** Gathers day groups, sorted by time, into runs of the same local month, keeping the order. */
export function groupDaysByMonth<G extends { day: string }>(days: G[]): MonthGroup<G>[] {
  const months: MonthGroup<G>[] = []
  for (const g of days) {
    const month = g.day.slice(0, 7)
    const last = months[months.length - 1]
    if (last && last.month === month) last.days.push(g)
    else months.push({ month, days: [g] })
  }
  return months
}

/** The header of a month: its name, with the year once it is not this one. `today` is a local YYYY-MM-DD. */
export function monthHeader(month: string, today: string, format: (date: Date, withYear: boolean) => string): string {
  const [y, m] = month.split('-').map(Number)
  return format(new Date(y, m - 1, 1), y !== Number(today.slice(0, 4)))
}
