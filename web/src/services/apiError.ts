import { i18n, intlLocale, t } from '@/i18n'
import { distanceUnit, formatDistanceValue, perDistance } from '@/units'

export interface ApiErrorBody {
  error?: string
  message?: string
  code?: string
  params?: Record<string, unknown>
}

// Catalogs holding a code: failures in "errors", warnings, labels and explanations in "messages".
const CATALOGS = ['errors', 'messages']

/** The catalog key of a code, or null when no catalog defines it. */
function keyOf(code: string | undefined): string | null {
  if (!code) return null
  return CATALOGS.map((catalog) => `${catalog}.${code}`).find((key) => i18n.global.te(key)) ?? null
}

/**
 * Parameters ready for interpolation: a nested API message is translated, a keyword ("kw:drives") too when it has a label,
 * and a distance ("km:42000") or a figure per km ("perkm:16.5") is shown in the account's unit, named by {unit}.
 */
function resolveParams(params: Record<string, unknown> | undefined): Record<string, unknown> {
  const out: Record<string, unknown> = { unit: distanceUnit() }
  for (const [name, value] of Object.entries(params ?? {})) {
    if (value && typeof value === 'object' && 'code' in value) {
      out[name] = apiErrorMessage(value as ApiErrorBody, String((value as ApiErrorBody).message ?? ''))
    } else if (typeof value === 'string' && value.startsWith('km:')) {
      out[name] = formatDistanceValue(Number(value.slice(3)))
    } else if (typeof value === 'string' && value.startsWith('perkm:')) {
      out[name] = perDistance(Number(value.slice(6))).toLocaleString(intlLocale(), { maximumFractionDigits: 2 })
    } else if (typeof value === 'string' && value.startsWith('kw:') && i18n.global.te(`messages.keywords.${value.slice(3)}`)) {
      out[name] = t(`messages.keywords.${value.slice(3)}`)
    } else if (typeof value === 'number') {
      out[name] = value.toLocaleString(intlLocale(), { useGrouping: false, maximumFractionDigits: 2 })
    } else {
      out[name] = value
    }
  }
  return out
}

/** The text of an API error or message in the current language: the catalog entry of its code, else the server's text. */
export function apiErrorMessage(body: ApiErrorBody | null | undefined, fallback: string): string {
  const key = keyOf(body?.code)
  if (key) return t(key, resolveParams(body?.params))
  return body?.error || body?.message || fallback
}

/** The error of a save refused because the item was changed since it was opened (409, code request.stale). */
export class StaleWriteError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'StaleWriteError'
  }
}

export const isStaleWrite = (err: unknown): err is StaleWriteError => err instanceof StaleWriteError

/** Same for a message object of a payload (a warning, an assumption, a label). */
export const apiMessageText = (message: ApiErrorBody | null | undefined): string => apiErrorMessage(message, '')

/** A request that never reached the server (offline, server unreachable): its message says so in the user's language. */
export class NetworkError extends Error {
  constructor() {
    super(t('shell.api.offline'))
    this.name = 'NetworkError'
  }
}

/** What a failed `fetch` throws on a network failure is a `TypeError` with the browser's English text: swap it for a `NetworkError`. */
export function asNetworkError(err: unknown): unknown {
  return err instanceof TypeError ? new NetworkError() : err
}
