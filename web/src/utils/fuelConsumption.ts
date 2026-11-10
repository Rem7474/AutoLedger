/** The figures of the fill-ups that matter for consumption (`GET /vehicles/{id}/fuel-logs`, measured between full tanks). */
export interface FuelStatsFigures {
  consumption_l_100km?: number | null
  avg_price_per_liter?: number | null
}

/** Litres per 100 km and what 100 km cost in fuel, both from the fill-ups; null when no consumption could be measured. */
export function fuelConsumptionFigures(stats: FuelStatsFigures | null | undefined): { litresPer100: number; costPer100: number | null } | null {
  const litres = Number(stats?.consumption_l_100km)
  if (!(litres > 0)) return null
  const price = Number(stats?.avg_price_per_liter)
  return { litresPer100: litres, costPer100: price > 0 ? litres * price : null }
}
