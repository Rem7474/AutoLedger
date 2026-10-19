import { intlLocale } from '@/i18n'

// A vehicle's currency is fixed at creation (see CLAUDE.md) and travels with its own data —
// unlike the language or distance unit, it is not an account-wide preference, so every call site
// passes the currency of the vehicle the amount belongs to rather than reading a global setting.

/** Formats a cents amount as currency, e.g. formatMoney(154000, "EUR") -> "1 540,00 €". */
export function formatMoney(cents: number, currency: string): string {
  return formatAmount(cents / 100, currency)
}

// No currency is ever assumed: while the vehicle that owns an amount is not loaded yet (`currency` is empty),
// the figure reads as a plain number, without a symbol, rather than as an amount in a guessed currency.
function numberFormat(currency: string, options: Intl.NumberFormatOptions): Intl.NumberFormat {
  return new Intl.NumberFormat(intlLocale(), currency ? { style: 'currency', currency, ...options } : options)
}

/** Formats a currency amount already in main units (not cents), e.g. a €/km rate. */
export function formatAmount(amount: number, currency: string, maximumFractionDigits = 2): string {
  return numberFormat(currency, { maximumFractionDigits }).format(amount)
}

/** A figure shown when its value is not known: a missing amount reads as a dash, never as a zero. */
export const UNKNOWN_AMOUNT = '—'

/** Like `formatAmount`, but `null`, `undefined` and NaN are an unknown figure and read as a dash. */
export function formatAmountOrUnknown(amount: number | null | undefined, currency: string, maximumFractionDigits = 2): string {
  if (amount === null || amount === undefined || !Number.isFinite(amount)) return UNKNOWN_AMOUNT
  return formatAmount(amount, currency, maximumFractionDigits)
}

/** Just the symbol ("€", "$"...), for a chart axis or dataset label that cannot show a full amount. */
export function currencySymbol(currency: string): string {
  if (!currency) return ''
  const part = new Intl.NumberFormat(intlLocale(), { style: 'currency', currency }).formatToParts(0).find((p) => p.type === 'currency')
  return part?.value ?? currency
}

/** A money axis tick without decimals, shortened above a thousand ("400 k€"). */
export function formatAxisAmount(amount: number, currency: string): string {
  return numberFormat(currency, { notation: 'compact', maximumFractionDigits: Math.abs(amount) >= 1000 ? 1 : 0 }).format(amount)
}
