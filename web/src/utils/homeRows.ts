import { donutRows } from '@/utils/donutBreakdown'

/** The two largest cost items of the full cost, as keys with their share (already rounded for display). */
export function topCostShares(tco: any, count = 2): { key: string; sharePct: number }[] {
  return donutRows(tco, 'full')
    .slice(0, count)
    .map((r) => ({ key: r.key, sharePct: r.sharePct }))
}

/** The latest month that has a distance: an empty current month would otherwise read as "0 km". */
export function lastDrivenMonth(months: any[] | null | undefined): { month: string; distance_km: number } | null {
  for (let i = (months?.length ?? 0) - 1; i >= 0; i--) {
    const m = months![i]
    if (Number(m?.distance_km) > 0) return { month: m.month, distance_km: Number(m.distance_km) }
  }
  return null
}
