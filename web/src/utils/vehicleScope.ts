import type { FleetCurrencyTotal } from '@/services/api'

/** What the home page covers: the active vehicle, or every vehicle of the account side by side. */
export type VehicleScope = 'vehicle' | 'all'

/** The value of the vehicle selector for the household view; no vehicle has this id. */
export const ALL_VEHICLES = 'all'

/** The scope a URL asks for. The household view needs at least two vehicles: with fewer it is the one vehicle's page. */
export function scopeFromQuery(raw: unknown, vehicleCount: number): VehicleScope {
  const value = Array.isArray(raw) ? raw[0] : raw
  return value === ALL_VEHICLES && vehicleCount > 1 ? 'all' : 'vehicle'
}

/** The value the vehicle selector shows: the household entry in the household view, the active vehicle otherwise. */
export function selectorValue(scope: VehicleScope, activeVehicleId: string | undefined): string {
  return scope === 'all' ? ALL_VEHICLES : (activeVehicleId ?? '')
}

export interface CurrencyLine {
  currency: string
  amount: number
  vehicleCount: number
}

/**
 * The month's cost as one line per currency, the currency of the fleet first. Amounts of two currencies are never added:
 * a line is what the vehicles of that currency cost, in that currency.
 */
export function currencyLines(totals: FleetCurrencyTotal[] | null | undefined, fleetCurrency: string): CurrencyLine[] {
  const lines = (totals ?? []).map((x) => ({ currency: x.currency, amount: x.current_month_cost, vehicleCount: x.vehicle_count }))
  return lines.sort((a, b) => Number(b.currency === fleetCurrency) - Number(a.currency === fleetCurrency))
}
