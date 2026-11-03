// The date fields of a phone use the browser's own pickers (the wheels of iOS, the dialog of Android) instead of the
// floating calendar of the desktop. These helpers turn the values of the app into what a native input accepts.
export type NativeDateKind = 'date' | 'datetime-local' | 'month'

export function nativeKind(opts: { monthPicker?: boolean; enableTimePicker?: boolean }): NativeDateKind {
  if (opts.monthPicker) return 'month'
  return opts.enableTimePicker ? 'datetime-local' : 'date'
}

const pad = (n: number) => String(n).padStart(2, '0')

/** The value shown by a native input of this kind: `yyyy-MM-dd`, `yyyy-MM-ddTHH:mm` or `yyyy-MM`; '' when there is none. */
export function toNativeValue(value: unknown, kind: NativeDateKind): string {
  if (value === null || value === undefined || value === '') return ''
  if (value instanceof Date) {
    if (Number.isNaN(value.getTime())) return ''
    const day = `${value.getFullYear()}-${pad(value.getMonth() + 1)}-${pad(value.getDate())}`
    if (kind === 'month') return day.slice(0, 7)
    return kind === 'datetime-local' ? `${day}T${pad(value.getHours())}:${pad(value.getMinutes())}` : day
  }
  if (typeof value === 'object' && 'year' in value && 'month' in value) {
    const { year, month } = value as { year: number; month: number }
    return `${year}-${pad(month + 1)}`
  }
  const text = String(value)
  if (kind === 'month') return /^\d{4}-\d{2}/.test(text) ? text.slice(0, 7) : ''
  if (kind === 'date') return /^\d{4}-\d{2}-\d{2}/.test(text) ? text.slice(0, 10) : ''
  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/.test(text)) return text.slice(0, 16)
  return /^\d{4}-\d{2}-\d{2}$/.test(text) ? `${text}T00:00` : ''
}

const support = new Map<NativeDateKind, boolean>()

/**
 * Whether the browser really offers a picker for this kind. A browser that does not know a type (month on some Android
 * and desktop browsers) turns the field into a plain text box, so the calendar of the desktop stays in that case.
 */
export function supportsNativeInput(kind: NativeDateKind, doc: Pick<Document, 'createElement'> | undefined = typeof document === 'undefined' ? undefined : document): boolean {
  if (!doc) return false
  const known = support.get(kind)
  if (known !== undefined) return known
  const probe = doc.createElement('input')
  probe.setAttribute('type', kind)
  const ok = probe.type === kind
  support.set(kind, ok)
  return ok
}

/** The range [from, to] in chronological order, or null while one of the two days is missing. */
export function toRange(from: string, to: string): [string, string] | null {
  if (!from || !to) return null
  return from <= to ? [from, to] : [to, from]
}
