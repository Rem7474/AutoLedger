import { buildMonthBreakdown } from '@/utils/dashboard'

export interface HeroSegment {
  key: string
  label: string
  color: string
  amount: number
  sharePct: number
}

export interface PreviousMonth {
  /** The "YYYY-MM" month it is read on. */
  month: string
  /** Net cash spent over the whole month: what was paid minus what the passengers paid back. */
  total: number
  /** The month in progress as a share of it, 0 to many: 0.8 is "80 % of it so far". */
  share: number
}

export interface TodayHero {
  month: string
  /** Cash paid so far this month, net of what passengers paid back. */
  spent: number
  /** The part of the month still due (rent, subscriptions): not paid yet. */
  scheduled: number
  /** The last full month, to read the month in progress against; null when no earlier month has some spending. */
  previous: PreviousMonth | null
  /** The themes the month's spending is made of, largest first, with nothing for an empty theme. */
  segments: HeroSegment[]
}

/**
 * The latest month before `currentMonth` that has some spending, on the cash basis net of the carpools, and the month in
 * progress as a share of it. The month in progress is never the reference: it is the thing being read.
 */
export function previousMonthOf(monthlyCosts: any[] | undefined, currentMonth: string, spent: number, currency: string): PreviousMonth | null {
  const past = (monthlyCosts ?? []).filter((m) => typeof m?.month === 'string' && m.month < currentMonth).sort((a, b) => b.month.localeCompare(a.month))
  for (const raw of past) {
    const { netTotal } = buildMonthBreakdown(raw, 'cash', currency)
    if (netTotal > 0) return { month: raw.month, total: netTotal, share: spent / netTotal }
  }
  return null
}

/**
 * The figures the page opens on, all on one basis: the cash paid in the month, net of the carpools. The bar of themes
 * is read on the same month as the headline; the last full month is its reference, on the same cash-net basis.
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
    previous: previousMonthOf(monthlyCosts, rawMonth.month, spent, currency),
    segments,
  }
}
