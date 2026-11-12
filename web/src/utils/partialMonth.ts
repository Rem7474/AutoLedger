import { plottedCostPerKm } from '@/utils/dashboard'
import { monthKey } from '@/utils/reportingTime'

// A month still running has little distance and a few costs: its cost per distance swings far from the months before
// and, drawn on the same scale, flattens all of them. It is drawn apart, as a hollow point named "in progress", and the
// scale is that of the complete months.

type Month = { month: string; distance_km?: number | null; cost_per_km?: number | null }

/** The key (`YYYY-MM`) of the month a date falls in, in the reporting timezone. */
export const monthKeyOf = (d: Date): string => monthKey(d)

export interface CostPerKmSeries {
  /** One value per month, null for a gap and for the month in progress. */
  complete: (number | null)[]
  /** The month in progress, when the list holds it and it has a plottable figure. */
  partial: { index: number; value: number } | null
  /** The top of the scale: that of the complete months, with some room. Undefined when none is plotted. */
  scaleMax: number | undefined
}

export function costPerKmSeries(months: Month[], today: Date): CostPerKmSeries {
  const current = monthKeyOf(today)
  const complete: (number | null)[] = []
  let partial: CostPerKmSeries['partial'] = null
  months.forEach((m, index) => {
    const value = plottedCostPerKm(m)
    if (m.month === current) {
      complete.push(null)
      if (value !== null) partial = { index, value }
    } else {
      complete.push(value)
    }
  })
  const plotted = complete.filter((v): v is number => v !== null)
  return { complete, partial, scaleMax: plotted.length > 0 ? Math.max(...plotted) * 1.15 : undefined }
}
