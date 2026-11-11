import { groupByDay, groupDaysByMonth, monthHeader } from '@/utils/dayGroups'
import { toLocalDay } from '@/utils/dates'

export interface FuelTotals {
  liters: number
  amount: number
  /** Average price per litre over the fill-ups that give both a quantity and an amount, or null when none does. */
  pricePerLiter: number | null
}

/** The figures of a month header: litres, cash paid and the price per litre that goes with them. */
export function fuelTotals(logs: { amount?: number | null; liters?: number | null }[]): FuelTotals {
  let liters = 0
  let amount = 0
  let pricedLiters = 0
  let pricedAmount = 0
  for (const l of logs) {
    const a = Number(l.amount) || 0
    const q = Number(l.liters) || 0
    amount += a
    liters += q
    if (q > 0 && a > 0) {
      pricedLiters += q
      pricedAmount += a
    }
  }
  return { liters, amount, pricePerLiter: pricedLiters > 0 ? pricedAmount / pricedLiters : null }
}

export interface FuelMonth<T> {
  month: string
  label: string
  items: T[]
  totals: FuelTotals
}

/** Fill-ups, newest first, under one header per local month, with the totals of each month. */
export function fuelMonths<T extends { date: string; amount?: number | null; liters?: number | null }>(
  logs: T[],
  today: string,
  format: (date: Date, withYear: boolean) => string,
): FuelMonth<T>[] {
  return groupDaysByMonth(groupByDay(logs, (l) => l.date)).map((m) => {
    const items = m.days.flatMap((d) => d.items)
    return { month: m.month, label: monthHeader(m.month, today, format), items, totals: fuelTotals(items) }
  })
}

/** A fill-up as the API sends it, with the figures of the interval it closes. */
export interface FuelLogFigures {
  date: string
  amount?: number | null
  liters?: number | null
  odometer?: number | null
  is_full_tank?: boolean
  /** Set on a full tank that closes a measurable interval with the previous full tank. */
  consumption_l_100km?: number | null
  segment_km?: number | null
  segment_estimated?: boolean
}

/** The fill-ups whose local day falls in the period (both bounds inclusive, an absent one is open), order kept. */
export function fuelLogsInPeriod<T extends { date: string }>(logs: T[], bounds: { from?: string; to?: string }): T[] {
  return logs.filter((l) => {
    const day = toLocalDay(l.date)
    return (!bounds.from || day >= bounds.from) && (!bounds.to || day <= bounds.to)
  })
}

export interface FuelPeriodSummary {
  fillUps: number
  totalCost: number
  pricePerLiter: number | null
  /** L/100 km over the intervals closed in the period, weighted by distance like the server's figure; null when none. */
  consumption: number | null
  /** Intervals closed in the period between two full tanks that give no consumption (no quantity or no distance). */
  unmeasurable: number
  /** Measurable intervals closed in the period that rely on an estimated mileage. */
  estimated: number
  /** Fill-ups of the period entered without a mileage. */
  withoutMileage: number
}

/**
 * The figures of the fill-ups kept by a period. `all` is the whole history, oldest first as the API sends it: the
 * consumption of an interval is measured over the full history, so a fill-up at the start of the period keeps the
 * full tank before it, and an interval belongs to the period of the fill-up that closes it.
 */
export function fuelPeriodSummary(all: FuelLogFigures[], bounds: { from?: string; to?: string }): FuelPeriodSummary {
  const kept = new Set(fuelLogsInPeriod(all, bounds))
  const visible = all.filter((l) => kept.has(l))
  const totals = fuelTotals(visible)
  let km = 0
  let litres = 0
  let estimated = 0
  let unmeasurable = 0
  let previousFull = false
  for (const l of all) {
    const inPeriod = kept.has(l)
    const consumption = Number(l.consumption_l_100km) || 0
    const segment = Number(l.segment_km) || 0
    if (inPeriod && consumption > 0 && segment > 0) {
      km += segment
      litres += (consumption * segment) / 100
      if (l.segment_estimated) estimated++
    } else if (inPeriod && l.is_full_tank && previousFull) {
      unmeasurable++
    }
    if (l.is_full_tank) previousFull = true
  }
  return {
    fillUps: visible.length,
    totalCost: totals.amount,
    pricePerLiter: totals.pricePerLiter,
    consumption: km > 0 ? (litres / km) * 100 : null,
    unmeasurable,
    estimated,
    withoutMileage: visible.filter((l) => l.odometer == null).length,
  }
}
