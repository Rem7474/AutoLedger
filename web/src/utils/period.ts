import { intlLocale, t } from '@/i18n'
import { formatMonthLabel, monthRange, todayIso } from '@/utils/drives'

// The period shared by the home page, the energy page and the expenses: a run of whole months ending now or earlier,
// a calendar year, two days picked freely, or the whole history. Days are YYYY-MM-DD in local time and `to` is
// inclusive, like `monthRange` and `unitRange`.
export type MonthCount = 1 | 3 | 6 | 12
export type Period =
  /** `count` calendar months ending with the month `end` (YYYY-MM); null = the current month, up to today. */
  | { kind: 'months'; count: MonthCount; end: string | null }
  | { kind: 'year'; year: number }
  | { kind: 'custom'; from: string; to: string }
  | { kind: 'all' }

export const MONTH_COUNTS: readonly MonthCount[] = [1, 3, 6, 12]
export const DEFAULT_PERIOD: Period = { kind: 'months', count: 12, end: null }

const ISO_DAY = /^\d{4}-\d{2}-\d{2}$/
const YEAR_MONTH = /^\d{4}-(0[1-9]|1[0-2])$/

const isRealDay = (s: string) => {
  if (!ISO_DAY.test(s)) return false
  const [y, m, d] = s.split('-').map(Number)
  const date = new Date(y, m - 1, d)
  return date.getFullYear() === y && date.getMonth() === m - 1 && date.getDate() === d
}
const isMonthCount = (n: number): n is MonthCount => (MONTH_COUNTS as readonly number[]).includes(n)

const monthOf = (day: string) => day.slice(0, 7)
const dayOf = (iso: string) => {
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(y, m - 1, d)
}
const isoOf = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`

/** The month (YYYY-MM) `n` months after `yearMonth`, `n` negative for before. */
export function addMonths(yearMonth: string, n: number): string {
  const [y, m] = yearMonth.split('-').map(Number)
  const d = new Date(y, m - 1 + n, 1)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
}

/** The last month a period of whole months covers. */
const endMonthOf = (p: Extract<Period, { kind: 'months' }>, today: string) => p.end ?? monthOf(today)

/** The first and last day a period covers; an empty object for the whole history. */
export function periodBounds(p: Period, now: Date = new Date()): { from?: string; to?: string } {
  const today = todayIso(now)
  if (p.kind === 'all') return {}
  if (p.kind === 'custom') return { from: p.from, to: p.to }
  if (p.kind === 'year') return { from: `${p.year}-01-01`, to: p.year === now.getFullYear() ? today : `${p.year}-12-31` }
  const end = endMonthOf(p, today)
  const last = end === monthOf(today) ? today : monthRange(end).to
  return { from: `${addMonths(end, -(p.count - 1))}-01`, to: last }
}

/**
 * The months a period touches, as YYYY-MM from the first to the last; null on an open side. A monthly figure cannot
 * be split over a part of a month, so a free range takes every month it touches (and the page says so).
 */
export function touchedMonths(p: Period, now: Date = new Date()): { from: string | null; to: string | null } {
  const { from, to } = periodBounds(p, now)
  return { from: from ? monthOf(from) : null, to: to ? monthOf(to) : null }
}

/** Whether a free range starts or ends in the middle of a month, so that its monthly figures are rounded outwards. */
export function isRoundedToMonths(p: Period): boolean {
  if (p.kind !== 'custom') return false
  return p.from !== monthRange(monthOf(p.from)).from || p.to !== monthRange(monthOf(p.to)).to
}

/** The period one step before (-1) or after (+1): a month for a month, N months for N months, a year, the length of a free range. */
export function stepPeriod(p: Period, dir: -1 | 1, now: Date = new Date()): Period {
  const today = todayIso(now)
  const thisMonth = monthOf(today)
  if (p.kind === 'all') return p
  if (p.kind === 'year') {
    const year = p.year + dir
    return year > now.getFullYear() ? p : { kind: 'year', year }
  }
  if (p.kind === 'months') {
    const end = addMonths(endMonthOf(p, today), dir * p.count)
    if (end > thisMonth) return p
    return { ...p, end: end === thisMonth ? null : end }
  }
  const from = dayOf(p.from)
  const to = dayOf(p.to)
  const span = Math.round((to.getTime() - from.getTime()) / 86400000) + 1
  const next = {
    from: isoOf(new Date(from.getFullYear(), from.getMonth(), from.getDate() + dir * span)),
    to: isoOf(new Date(to.getFullYear(), to.getMonth(), to.getDate() + dir * span)),
  }
  return next.to > today ? p : { kind: 'custom', ...next }
}

/** Whether the arrow in that direction goes anywhere: not for the whole history, not past today. */
export const canStep = (p: Period, dir: -1 | 1, now: Date = new Date()): boolean => stepPeriod(p, dir, now) !== p && p.kind !== 'all'

/** The ready-made periods of the sheet, in the order it lists them. */
export type PeriodPreset = 'thisMonth' | 'lastMonth' | 'last3' | 'last6' | 'last12' | 'thisYear' | 'all'
export const PERIOD_PRESETS: readonly PeriodPreset[] = ['thisMonth', 'lastMonth', 'last3', 'last6', 'last12', 'thisYear', 'all']

export function presetPeriod(preset: PeriodPreset, now: Date = new Date()): Period {
  const thisMonth = monthOf(todayIso(now))
  switch (preset) {
    case 'thisMonth':
      return { kind: 'months', count: 1, end: null }
    case 'lastMonth':
      return { kind: 'months', count: 1, end: addMonths(thisMonth, -1) }
    case 'last3':
      return { kind: 'months', count: 3, end: null }
    case 'last6':
      return { kind: 'months', count: 6, end: null }
    case 'last12':
      return { kind: 'months', count: 12, end: null }
    case 'thisYear':
      return { kind: 'year', year: now.getFullYear() }
    default:
      return { kind: 'all' }
  }
}

/** The preset a period is, if it is one (the sheet highlights it); null for anything else. */
export function presetOf(p: Period, now: Date = new Date()): PeriodPreset | null {
  const same = (a: Period, b: Period) => JSON.stringify(a) === JSON.stringify(b)
  return PERIOD_PRESETS.find((k) => same(presetPeriod(k, now), p)) ?? null
}

const shortDay = (iso: string, withYear: boolean) =>
  dayOf(iso).toLocaleDateString(intlLocale(), { day: 'numeric', month: 'short', ...(withYear ? { year: 'numeric' } : {}) })

/** What the bar says: "Novembre 2026", "3 derniers mois", "Mai – Juillet 2026", "2026", "12 mars – 4 avr. 2026", "Tout". */
export function periodLabel(p: Period, now: Date = new Date()): string {
  const today = todayIso(now)
  if (p.kind === 'all') return t('period.all')
  if (p.kind === 'year') return t('period.year', { year: p.year })
  if (p.kind === 'custom') return t('period.range', { from: shortDay(p.from, monthOf(p.from).slice(0, 4) !== monthOf(p.to).slice(0, 4)), to: shortDay(p.to, true) })
  if (p.count === 1) return formatMonthLabel(endMonthOf(p, today))
  if (p.end === null) return t('period.lastMonths', { count: p.count })
  return t('period.range', { from: formatMonthLabel(addMonths(p.end, -(p.count - 1))), to: formatMonthLabel(p.end) })
}

/** The period as URL parameters: `period=3M`, `period=3M&end=2026-08`, `period=Y2026`, `period=all`, `from=…&to=…`. */
export function periodToQuery(p: Period): Record<string, string> {
  if (p.kind === 'all') return { period: 'all' }
  if (p.kind === 'year') return { period: `Y${p.year}` }
  if (p.kind === 'custom') return { from: p.from, to: p.to }
  return p.end === null ? { period: `${p.count}M` } : { period: `${p.count}M`, end: p.end }
}

type Query = Record<string, unknown>
const one = (v: unknown): string | null => (typeof v === 'string' ? v : Array.isArray(v) && typeof v[0] === 'string' ? v[0] : null)

/** The period a URL names, or null when it names none or names it wrongly (a future month, a day that does not exist, from after to). */
export function periodFromQuery(q: Query, now: Date = new Date()): Period | null {
  const today = todayIso(now)
  const from = one(q.from)
  const to = one(q.to)
  if (from && to) {
    return isRealDay(from) && isRealDay(to) && from <= to && to <= today ? { kind: 'custom', from, to } : null
  }
  const period = one(q.period)
  if (!period) return null
  if (period === 'all') return { kind: 'all' }
  const year = /^Y(\d{4})$/.exec(period)
  if (year) return Number(year[1]) <= now.getFullYear() ? { kind: 'year', year: Number(year[1]) } : null
  const months = /^(\d{1,2})M$/.exec(period)
  if (!months || !isMonthCount(Number(months[1]))) return null
  const count = Number(months[1]) as MonthCount
  const end = one(q.end)
  if (end === null) return { kind: 'months', count, end: null }
  if (!YEAR_MONTH.test(end) || end > monthOf(today)) return null
  return { kind: 'months', count, end: end === monthOf(today) ? null : end }
}

/** The period a stored value holds (JSON), or null when it is not one. */
export function parsePeriod(raw: unknown, now: Date = new Date()): Period | null {
  if (!raw || typeof raw !== 'object') return null
  const p = raw as Record<string, unknown>
  const today = todayIso(now)
  if (p.kind === 'all') return { kind: 'all' }
  if (p.kind === 'year') return typeof p.year === 'number' && Number.isInteger(p.year) && p.year <= now.getFullYear() ? { kind: 'year', year: p.year } : null
  if (p.kind === 'custom') {
    return typeof p.from === 'string' && typeof p.to === 'string' && isRealDay(p.from) && isRealDay(p.to) && p.from <= p.to && p.to <= today
      ? { kind: 'custom', from: p.from, to: p.to }
      : null
  }
  if (p.kind === 'months' && typeof p.count === 'number' && isMonthCount(p.count)) {
    if (p.end === null) return { kind: 'months', count: p.count, end: null }
    return typeof p.end === 'string' && YEAR_MONTH.test(p.end) && p.end <= monthOf(today) ? { kind: 'months', count: p.count, end: p.end === monthOf(today) ? null : p.end } : null
  }
  return null
}

/** The period a key of the old dashboard selector ("1M", "3M", "6M", "1Y", "ALL") stood for. */
export function periodFromLegacyRange(key: string | null): Period | null {
  switch (key) {
    case '1M':
      return { kind: 'months', count: 1, end: null }
    case '3M':
      return { kind: 'months', count: 3, end: null }
    case '6M':
      return { kind: 'months', count: 6, end: null }
    case '1Y':
      return { kind: 'months', count: 12, end: null }
    case 'ALL':
      return { kind: 'all' }
    default:
      return null
  }
}

/** The months of a monthly series (`month` = YYYY-MM) that a period touches; the whole series for the whole history. */
export function filterMonthsByPeriod<T extends { month: string }>(list: T[], p: Period, now: Date = new Date()): T[] {
  const { from, to } = touchedMonths(p, now)
  return list.filter((m) => (from === null || m.month >= from) && (to === null || m.month <= to))
}

/** What the place statistics endpoint takes: N months back from now when the period ends now, else exact days. */
export function placeStatsParams(p: Period, now: Date = new Date()): { months: number } | { from: string; to: string } {
  if (p.kind === 'all') return { months: 0 }
  if (p.kind === 'months' && p.end === null) return { months: p.count }
  const { from, to } = periodBounds(p, now)
  return { from: from as string, to: to as string }
}

const PERIOD_QUERY_KEYS = ['period', 'end', 'from', 'to']

/** A URL query with the period parameters replaced by those of `p`, every other parameter kept. */
export function withPeriodQuery(query: Query, p: Period): Query {
  const rest = Object.fromEntries(Object.entries(query).filter(([key]) => !PERIOD_QUERY_KEYS.includes(key)))
  return { ...rest, ...periodToQuery(p) }
}

/** Whether a URL query already names exactly this period (and no stray period parameter). */
export function queryNamesPeriod(query: Query, p: Period): boolean {
  const wanted = periodToQuery(p)
  return PERIOD_QUERY_KEYS.every((key) => one(query[key]) === (wanted[key] ?? null))
}
