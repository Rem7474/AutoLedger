/** The electric share of a plug-in hybrid or range extender (`GET /vehicles/{id}/hybrid-share`). */
export interface HybridShareQuality {
  segments_used: number
  segments_discarded: number
  km_covered: number
  km_total: number
  level: 'insufficient' | 'low' | 'good'
  insufficient_because?: 'not_enough_data' | 'energy_and_fuel_not_separable'
  window_months?: number
  capacity_from_percentages: boolean
  less_precise_energy_charges: number
}

export interface HybridSettings {
  electric_share: number | null
  fuel_l_100km: number | null
  battery_usable_kwh: number | null
}

export interface HybridShare {
  suggested_electric_share?: number
  fuel_l_100km_fuel_part?: number
  battery_usable_kwh?: number
  quality: HybridShareQuality
  accepted: HybridSettings
}

/** A share (0 to 1) as the whole percentage shown to the user. */
export const sharePercent = (share: number): number => Math.round(share * 100)

/** A percentage typed by the user as the share (0 to 1) the server stores, with the three decimals it keeps. */
export const percentToShare = (percent: number): number => Math.round(percent * 10) / 1000

/**
 * Where the suggestion stands against what the owner accepted: nothing to suggest yet (`insufficient`), already the
 * accepted value (`accepted`), or a value that can still be accepted (`available`).
 */
export function suggestionStatus(res: Pick<HybridShare, 'suggested_electric_share' | 'accepted'>): 'insufficient' | 'accepted' | 'available' {
  const suggested = res.suggested_electric_share
  if (suggested === undefined || suggested === null) return 'insufficient'
  const accepted = res.accepted.electric_share
  return accepted !== null && sharePercent(accepted) === sharePercent(suggested) ? 'accepted' : 'available'
}

/** The line under the title of the vehicle sheet row: the accepted share, else what is missing. */
export function hybridShareHint(res: Pick<HybridShare, 'accepted' | 'suggested_electric_share'> | null): 'accepted' | 'suggested' | 'none' {
  if (res?.accepted.electric_share !== null && res?.accepted.electric_share !== undefined) return 'accepted'
  return res && res.suggested_electric_share !== undefined ? 'suggested' : 'none'
}

/** The number in a text field, with a comma or a point as separator; null when it is empty or not a number. */
export function fieldNumber(text: string | number | null | undefined): number | null {
  if (text === null || text === undefined || String(text).trim() === '') return null
  const n = Number(String(text).trim().replace(',', '.'))
  return Number.isFinite(n) ? n : null
}

/** A share typed as a percentage is usable when it is a number from 0 to 100. */
export function percentIsValid(text: string | number | null | undefined): boolean {
  const p = fieldNumber(text)
  return p !== null && p >= 0 && p <= 100
}

export interface HybridShareBody {
  accept_suggestion?: boolean
  electric_share?: number | null
  fuel_l_100km?: number | null
  battery_usable_kwh?: number | null
}

/**
 * The request that saves what the owner typed. The consumption is typed in the display unit and converted to litres per
 * 100 km; a capacity left empty keeps the deduced one (null). Null when the percentage is not usable.
 */
export function typedBody(fields: { percent: string; fuel: string; capacity: string }, toL100: (display: number) => number): HybridShareBody | null {
  if (!percentIsValid(fields.percent)) return null
  const fuel = fieldNumber(fields.fuel)
  return {
    electric_share: percentToShare(fieldNumber(fields.percent) as number),
    fuel_l_100km: fuel === null ? null : toL100(fuel),
    battery_usable_kwh: fieldNumber(fields.capacity),
  }
}

/** Accepting the suggestion keeps the capacity correction the owner typed: it counts in the suggestion. */
export const acceptBody = (accepted: HybridSettings | null | undefined): HybridShareBody => ({ accept_suggestion: true, battery_usable_kwh: accepted?.battery_usable_kwh ?? null })

/** Stops using a value: the share and its consumption go, the capacity correction stays. */
export const clearBody = (capacity: string): HybridShareBody => ({ electric_share: null, fuel_l_100km: null, battery_usable_kwh: fieldNumber(capacity) })

/** Changes the capacity correction and nothing else. */
export const capacityBody = (accepted: HybridSettings | null | undefined, capacity: string): HybridShareBody => ({
  electric_share: accepted?.electric_share ?? null,
  fuel_l_100km: accepted?.fuel_l_100km ?? null,
  battery_usable_kwh: fieldNumber(capacity),
})

export type HybridDriveEnergy =
  | { kind: 'split'; electricPercent: number; kwh: number; litres: number | null }
  | { kind: 'unknown' }

/**
 * How the energy of a drive of a plug-in hybrid is split, read from the drive as the API returns it (null for any other
 * drive: an electric vehicle, a fuel-only one, or a hybrid drive synchronised from a source, which is all electric).
 * A manual drive carries the share frozen when it was saved; without one the server leaves its energy unknown.
 */
export function hybridDriveEnergy(drive: { electric_share?: number | null; costs?: any } | null | undefined): HybridDriveEnergy | null {
  const costs = drive?.costs
  if (!costs || costs.fuel_only) return null
  const share = drive?.electric_share
  if (share !== null && share !== undefined) {
    const litres = costs.fuel_liters === null || costs.fuel_liters === undefined ? null : Number(costs.fuel_liters)
    return { kind: 'split', electricPercent: sharePercent(share), kwh: Number(costs.electricity_kwh) || 0, litres }
  }
  return costs.energy_source === 'UNKNOWN' ? { kind: 'unknown' } : null
}
