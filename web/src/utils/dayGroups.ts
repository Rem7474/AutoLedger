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
