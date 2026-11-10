import { buildMonthBreakdown } from '@/utils/dashboard'

export interface HeroSegment {
  key: string
  label: string
  color: string
  amount: number
  sharePct: number
}

export interface PerKmReference {
  /** The "YYYY-MM" month it is read on. */
  month: string
  /** Net cash spent per km: what was paid minus what the passengers paid back, over the month's distance. */
  value: number
  distanceKm: number
}

export interface TodayHero {
  month: string
  /** Cash paid so far this month, net of what passengers paid back. */
  spent: number
  /** The part of the month still due (rent, subscriptions): not paid yet. */
  scheduled: number
  /** The cost per km of the last full month; null when no earlier month has both a distance and some spending. */
  perKm: PerKmReference | null
  /** The themes the month's spending is made of, largest first, with nothing for an empty theme. */
  segments: HeroSegment[]
}

/**
 * The cost per km of the latest month before `currentMonth` that has a distance and some spending, on the cash basis net
 * of the carpools. The month in progress is never used: a few days of driving against a full month of rent says nothing.
 */
export function lastFullMonthPerKm(monthlyCosts: any[] | undefined, currentMonth: string, currency: string): PerKmReference | null {
  const past = (monthlyCosts ?? []).filter((m) => typeof m?.month === 'string' && m.month < currentMonth).sort((a, b) => b.month.localeCompare(a.month))
  for (const raw of past) {
    const { netTotal, distanceKm } = buildMonthBreakdown(raw, 'cash', currency)
    if (distanceKm > 0 && netTotal > 0) return { month: raw.month, value: netTotal / distanceKm, distanceKm }
  }
  return null
}

/**
 * The figures the page opens on, all on one basis: the cash paid in the month, net of the carpools. The bar of themes
 * is read on the same month as the headline; the cost per km is the last full month's, on the same cash-net basis.
 */
export function todayHero(rawMonth: any | null | undefined, currency: string, monthlyCosts?: any[]): TodayHero | null {
  if (!rawMonth) return null
  const breakdown = buildMonthBreakdown(rawMonth, 'cash', currency)
  const scheduled = Number(rawMonth.scheduled) || 0
  const spent = breakdown.netTotal - scheduled
  const segments = breakdown.items
    .filter((i) => i.displayAmount > 0)
    .sort((a, b) => b.displayAmount - a.displayAmount)
    .map((i) => ({ key: i.key, label: i.label, color: i.color, amount: i.displayAmount, sharePct: i.sharePct }))
  return {
    month: rawMonth.month,
    spent,
    scheduled,
    perKm: lastFullMonthPerKm(monthlyCosts, rawMonth.month, currency),
    segments,
  }
}
