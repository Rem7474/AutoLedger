// The calendar the server cuts days and months with (its reporting timezone, sent with the account). Every "today",
// month key and period boundary of the app goes through here, so that a record dated by the server lands in the period
// the bar names whatever the clock or timezone of the browser. Without a configured zone (signed out, tests) the
// browser's own zone is used. Timestamps keep being displayed in the browser's zone; only the day a timestamp belongs
// to, and the day "now" is, follow the reporting zone.

let zone: string | null = null
const formatters = new Map<string, Intl.DateTimeFormat>()

const isZone = (name: string): boolean => {
  try {
    new Intl.DateTimeFormat('en-US', { timeZone: name })
    return true
  } catch {
    return false
  }
}

/** Sets the reporting zone (an IANA name); null, empty or an unknown name goes back to the browser's zone. */
export function setReportingTimezone(name: string | null | undefined): void {
  zone = name && isZone(name) ? name : null
}

/** The zone days are cut in. */
export const reportingTimezone = (): string => zone ?? Intl.DateTimeFormat().resolvedOptions().timeZone

function formatterFor(timeZone: string): Intl.DateTimeFormat {
  let f = formatters.get(timeZone)
  if (!f) {
    f = new Intl.DateTimeFormat('en-US', { timeZone, year: 'numeric', month: '2-digit', day: '2-digit', hourCycle: 'h23' })
    formatters.set(timeZone, f)
  }
  return f
}

/** The year, month (1-12) and day of an instant in the reporting zone. */
export function dayParts(d: Date | string | number = new Date()): { year: number; month: number; day: number } {
  const date = d instanceof Date ? d : new Date(d)
  const out = { year: 0, month: 0, day: 0 }
  for (const p of formatterFor(reportingTimezone()).formatToParts(date)) {
    if (p.type === 'year') out.year = Number(p.value)
    else if (p.type === 'month') out.month = Number(p.value)
    else if (p.type === 'day') out.day = Number(p.value)
  }
  return out
}

const pad = (n: number) => String(n).padStart(2, '0')

/** The reporting-zone calendar day (YYYY-MM-DD) of an instant, or of now when omitted. */
export function dayKey(d: Date | string | number = new Date()): string {
  const { year, month, day } = dayParts(d)
  return `${year}-${pad(month)}-${pad(day)}`
}

/** The reporting-zone calendar month (YYYY-MM) of an instant, or of now when omitted. */
export const monthKey = (d: Date | string | number = new Date()): string => dayKey(d).slice(0, 7)

/** Today in the reporting zone, YYYY-MM-DD. */
export const reportingToday = (now: Date = new Date()): string => dayKey(now)

/** The current year in the reporting zone. */
export const reportingYear = (now: Date = new Date()): number => dayParts(now).year
