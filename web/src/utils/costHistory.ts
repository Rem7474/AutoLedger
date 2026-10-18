import { intlLocale } from '@/i18n'
import { rangeToMonths, type MonthlyRangeKey } from '@/utils/dashboard'

// The monthly cash history of one family of costs (tolls, servicing, fixed costs), as the API returns it: every
// calendar month from the first record to the current one, a month without a record at zero.
export interface CostMonth {
  month: string // YYYY-MM
  amount: number
}

/** Months a window of N months looks back over for the smoothed average. */
export const ROLLING_AVERAGE_MONTHS = 12

const monthEnd = (month: string): string => {
  const [y, m] = month.split('-').map(Number)
  return new Date(Date.UTC(y, m, 0)).toISOString().slice(0, 10)
}

/**
 * The months of a period: the last N months of a preset (the current one included, everything for the whole
 * history), or, for a range of days picked on the calendar, the months it covers entirely (a monthly figure cannot be
 * split over a part of a month).
 */
export function monthsOfPeriod(history: CostMonth[], range: MonthlyRangeKey, custom: [string, string] | null): CostMonth[] {
  if (custom) {
    const [from, to] = custom
    return history.filter((m) => `${m.month}-01` >= from && monthEnd(m.month) <= to)
  }
  const months = rangeToMonths(range)
  return months ? history.slice(-months) : history
}

/**
 * The smoothed monthly cost at each month of the history: the average of the month and the eleven before it. A
 * history shorter than twelve months averages over the months it has, so the first months are not understated by
 * months that never existed.
 */
export function rollingAverage(history: CostMonth[], window = ROLLING_AVERAGE_MONTHS): number[] {
  const out: number[] = []
  let sum = 0
  for (let i = 0; i < history.length; i++) {
    sum += history[i].amount
    if (i >= window) sum -= history[i - window].amount
    out.push(sum / Math.min(window, i + 1))
  }
  return out
}

export interface CostPeriodSummary {
  total: number
  /** Total over the months of the period; null for an empty period. */
  monthlyAverage: number | null
  months: number
}

export function summarizePeriod(months: CostMonth[]): CostPeriodSummary {
  const total = months.reduce((sum, m) => sum + m.amount, 0)
  return { total, monthlyAverage: months.length ? total / months.length : null, months: months.length }
}

const monthName = (month: string, style: 'short' | 'long') =>
  new Date(`${month}-01T00:00:00Z`).toLocaleDateString(intlLocale(), { month: style, year: style === 'short' ? '2-digit' : 'numeric', timeZone: 'UTC' })

/** Short month label of an axis: "Oct 26". */
export const shortMonth = (month: string): string => monthName(month, 'short')

/** What a period covers, as read next to a figure: "October 2026" or "November 2025 – October 2026". */
export function periodLabel(months: CostMonth[]): string {
  if (!months.length) return ''
  const first = monthName(months[0].month, 'long')
  const last = monthName(months[months.length - 1].month, 'long')
  return first === last ? first : `${first} – ${last}`
}
