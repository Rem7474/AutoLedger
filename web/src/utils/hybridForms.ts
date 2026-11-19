// A number typed on a phone keyboard, with a decimal comma; blank or invalid is null. (utils/quickAdd imports this file,
// so it cannot import its own parser back.)
function toNumber(value: string): number | null {
  const text = value.trim().replace(',', '.')
  if (text === '') return null
  const n = Number(text)
  return Number.isFinite(n) ? n : null
}

/** A plug-in hybrid or range extender: a battery to plug in and a tank. */
export function isHybridCapabilities(caps: { charges: boolean; refuels: boolean }): boolean {
  return caps.charges && caps.refuels
}

export type BatteryLevelProblem = 'range' | 'order'

export interface BatteryLevels {
  start: number | null
  end: number | null
  problem: BatteryLevelProblem | null
}

// A percentage typed in a field: blank is "not given", anything outside 0-100 is kept so the form can say so.
function percent(text: string | number | null | undefined): number | null {
  const n = toNumber(text === null || text === undefined ? '' : String(text))
  return n === null ? null : Math.round(n)
}

/**
 * The optional battery percentages of a charge, with the same rules as the server: each from 0 to 100, and a charge
 * ends higher than it started. The form shows the problem before the request, the server stays the authority.
 */
export function readBatteryLevels(start: string | number | null | undefined, end: string | number | null | undefined): BatteryLevels {
  const s = percent(start)
  const e = percent(end)
  if ([s, e].some((v) => v !== null && (v < 0 || v > 100))) return { start: s, end: e, problem: 'range' }
  if (s !== null && e !== null && e <= s) return { start: s, end: e, problem: 'order' }
  return { start: s, end: e, problem: null }
}

/** Whether the odometer must be typed: a hybrid needs it to tell its electric kilometres from its fuel ones. */
export function odometerRequired(hybrid: boolean, locked: boolean): boolean {
  return hybrid && !locked
}

/** A blank or zero reading counts as missing. */
export function hasOdometerReading(value: string | number | null | undefined): boolean {
  const n = toNumber(value === null || value === undefined ? '' : String(value))
  return n !== null && n > 0
}
