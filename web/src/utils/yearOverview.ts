import { buildMonthBreakdown } from '@/utils/dashboard'

export interface YearBarSegment {
  key: string
  color: string
  amount: number
}

export interface YearBar {
  /** The "YYYY-MM" month. */
  month: string
  /** Cash paid in the month, net of what the passengers paid back. */
  total: number
  /** The month still running: drawn apart, and left out of the figures. */
  partial: boolean
  /** The themes of the month, in the order of the legend, each cut down to the net total. */
  segments: YearBarSegment[]
}

export interface YearOverview {
  /** The complete months of the window, oldest first, then the month in progress when the list holds it. */
  bars: YearBar[]
  /** The complete months the figures read, at most 12, from the first month with spending. */
  months: number
  /** What was paid over them, net of the carpools. */
  total: number
  average: number
  /** Net cash spent per km over them; null when they have no distance. */
  perKm: number | null
  /** The largest bar, the top of the scale. */
  max: number
}

const WINDOW = 12

/**
 * The last 12 complete months and the month in progress, on one basis: the cash paid, net of the carpools. The figures
 * (total, average, cost per km) read the complete months only, so that a few days of the month in progress do not pull
 * them down; the month in progress is drawn as a thirteenth bar. Null with fewer than two complete months: a single bar
 * is not an overview.
 */
export function yearOverview(monthlyCosts: any[] | undefined, currentMonth: string, currency: string): YearOverview | null {
  const list = (monthlyCosts ?? []).filter((m) => typeof m?.month === 'string')
  const bar = (raw: any, partial: boolean): YearBar & { distanceKm: number } => {
    const { items, netTotal, distanceKm } = buildMonthBreakdown(raw, 'cash', currency)
    const shown = items.filter((i) => i.displayAmount > 0)
    const gross = shown.reduce((sum, i) => sum + i.displayAmount, 0)
    const scale = gross > 0 && netTotal > 0 ? netTotal / gross : 0
    return {
      month: raw.month, total: netTotal, partial, distanceKm,
      segments: shown.map((i) => ({ key: i.key, color: i.color, amount: i.displayAmount * scale })).filter((s) => s.amount > 0),
    }
  }
  // The months before the first spending are not part of the vehicle's history: they would only drag the average down.
  const history = list.filter((m) => m.month < currentMonth).sort((a, b) => a.month.localeCompare(b.month)).map((m) => bar(m, false))
  const first = history.findIndex((b) => b.total > 0)
  const bars = first < 0 ? [] : history.slice(first).slice(-WINDOW)
  if (bars.length < 2) return null
  const total = bars.reduce((sum, b) => sum + b.total, 0)
  const distance = bars.reduce((sum, b) => sum + b.distanceKm, 0)
  const current = list.find((m) => m.month === currentMonth)
  const all: YearBar[] = current ? [...bars, bar(current, true)] : bars
  return {
    bars: all.map(({ month, total: t, partial, segments }) => ({ month, total: t, partial, segments })),
    months: bars.length,
    total,
    average: total / bars.length,
    perKm: distance > 0 && total > 0 ? total / distance : null,
    max: Math.max(...all.map((b) => b.total), 0),
  }
}
