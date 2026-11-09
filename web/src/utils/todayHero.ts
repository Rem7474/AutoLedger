import { buildMonthBreakdown } from '@/utils/dashboard'

export interface HeroSegment {
  key: string
  label: string
  color: string
  amount: number
  sharePct: number
}

export interface TodayHero {
  month: string
  /** Cash paid so far this month, net of what passengers paid back. */
  spent: number
  /** The part of the month still due (rent, subscriptions): not paid yet. */
  scheduled: number
  /** What the cost per distance is read on: the month's distance. */
  distanceKm: number
  /** Net cash spent per km over the month; null without a distance or without any spending. */
  perKm: number | null
  /** The themes the month's spending is made of, largest first, with nothing for an empty theme. */
  segments: HeroSegment[]
}

/**
 * The figures the page opens on, all on one basis: the cash paid in the month, net of the carpools. The cost per km
 * and the bar of themes are read on the same month, so they cannot disagree with the headline.
 */
export function todayHero(rawMonth: any | null | undefined, currency: string): TodayHero | null {
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
    distanceKm: breakdown.distanceKm,
    perKm: breakdown.distanceKm > 0 && spent > 0 ? spent / breakdown.distanceKm : null,
    segments,
  }
}
