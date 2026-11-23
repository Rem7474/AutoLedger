import type { TollInvoiceMonth } from '@/services/api'

// The monthly toll invoices of the Costs page: which months the card lists and how the gap between an invoice and the
// tolls recorded on the trips is read.

/** How many months the card lists before "Show more". */
export const INVOICE_MONTHS_SHOWN = 6

/** A month is listed when it has an invoice or tolls to replace; the two latest months always are, so that the first invoice can be entered. */
export function listedInvoiceMonths(months: TollInvoiceMonth[], showAll: boolean): { rows: TollInvoiceMonth[]; hidden: number } {
  const relevant = months.filter((m, i) => i < 2 || m.invoice !== null || m.trip_tolls > 0)
  const rows = showAll ? relevant : relevant.slice(0, INVOICE_MONTHS_SHOWN)
  return { rows, hidden: relevant.length - rows.length }
}

/** What the gap means: the invoice holds tolls no trip carries, or the trips record more than was billed, or they agree. */
export type GapKind = 'unattached' | 'lower' | 'none'

/** Amounts are cents-exact decimals: a difference below half a cent is no gap. */
export function gapKind(gap: number): GapKind {
  if (gap > 0.005) return 'unattached'
  if (gap < -0.005) return 'lower'
  return 'none'
}

/** The gap of an invoice being typed, null while the total is empty or not a number. */
export function gapOfTotal(total: string | number, tripTolls: number): number | null {
  if (total === '' || total === null) return null
  const n = Number(total)
  return Number.isFinite(n) ? Math.round((n - tripTolls) * 100) / 100 : null
}

/** A month key (YYYY-MM) as a date at noon, safe from timezone shifts when formatted. */
export const monthDate = (month: string): Date => new Date(Number(month.slice(0, 4)), Number(month.slice(5, 7)) - 1, 15, 12)
