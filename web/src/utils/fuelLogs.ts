import { groupByDay, groupDaysByMonth, monthHeader } from '@/utils/dayGroups'

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
