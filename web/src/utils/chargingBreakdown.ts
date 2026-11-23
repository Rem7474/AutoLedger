import type { ChargeClass } from '@/components/dashboard/energyStats'
import type { PlaceCategoryRow } from '@/utils/chargingInsights'
import { categoryColor } from '@/utils/chargingInsights'

// The two ways the energy of a vehicle is split on the consumption page (where it was charged, how fast), reduced to the
// same row so that one donut and one legend draw either.
export type BreakdownAxis = 'place' | 'speed'

export interface BreakdownRow {
  key: string
  label: string
  color: string
  kwh: number
  /** Share of the energy shown, 0..100. */
  sharePct: number
  /** Null when no session of the row has a cost. */
  pricePerKwh: number | null
  /** Sessions left out of the price because they have no cost. */
  unknownSessions: number
}

export const SPEED_COLOR: Record<'AC' | 'DC', string> = { AC: '#38bdf8', DC: '#f59e0b' }

export function placeBreakdownRows(rows: PlaceCategoryRow[], label: (category: string) => string): BreakdownRow[] {
  return rows.map((r) => ({
    key: r.category || 'unclassified',
    label: label(r.category),
    color: categoryColor(r.category),
    kwh: r.kwh,
    sharePct: r.sharePct,
    pricePerKwh: r.pricePerKwh,
    unknownSessions: r.unknownSessions,
  }))
}

export function speedBreakdownRows(classes: ChargeClass[], label: (c: ChargeClass['class']) => string): BreakdownRow[] {
  const total = classes.reduce((sum, c) => sum + c.kwh_added, 0)
  return classes
    .filter((c) => c.kwh_added > 0)
    .map((c) => ({
      key: c.class,
      label: label(c.class),
      color: SPEED_COLOR[c.class as 'AC' | 'DC'] ?? '#94a3b8',
      kwh: c.kwh_added,
      sharePct: total > 0 ? (c.kwh_added / total) * 100 : 0,
      pricePerKwh: c.price_per_kwh ?? null,
      unknownSessions: c.sessions_without_cost,
    }))
}

/** The axes worth offering: a split with no energy is not an option. */
export function availableAxes(place: BreakdownRow[], speed: BreakdownRow[]): BreakdownAxis[] {
  const axes: BreakdownAxis[] = []
  if (place.length > 0) axes.push('place')
  if (speed.length > 0) axes.push('speed')
  return axes
}

/** The axis to show: the chosen one when it is still available, else the first available one. */
export function activeAxis(chosen: BreakdownAxis, axes: BreakdownAxis[]): BreakdownAxis | null {
  if (axes.includes(chosen)) return chosen
  return axes[0] ?? null
}
