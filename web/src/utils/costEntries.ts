import type { Component } from 'vue'
import { Disc, Landmark, Receipt, Route, Shield, Wrench, Zap } from 'lucide-vue-next'
import type { ApiAmount, CostEntryLine } from '@/services/api'
import { t } from '@/i18n'
import type { Theme } from '@/utils/themes'

// The journal of what a vehicle cost: the lines of the cost ledger, read through the filters of the Costs page.

/** A filter of the Costs page: the ledger categories (lower case, as the API names them) it keeps. */
export type CostFilterKey = 'all' | 'energy' | 'tolls' | 'maintenance' | 'tires' | 'insurance' | 'financing' | 'other'

export const COST_FILTERS: { key: CostFilterKey; theme: Theme | null; categories: string[]; icon: Component | null }[] = [
  { key: 'all', theme: null, categories: [], icon: null },
  { key: 'energy', theme: 'energy', categories: ['energy'], icon: Zap },
  { key: 'tolls', theme: 'tolls', categories: ['toll', 'parking', 'travel_other'], icon: Route },
  { key: 'maintenance', theme: 'maintenance', categories: ['maintenance', 'repair'], icon: Wrench },
  { key: 'tires', theme: 'tires', categories: ['tires'], icon: Disc },
  { key: 'insurance', theme: 'insurance', categories: ['insurance'], icon: Shield },
  { key: 'financing', theme: 'financing', categories: ['financing'], icon: Landmark },
  { key: 'other', theme: 'other', categories: ['subscription', 'tax', 'accessory', 'other'], icon: Receipt },
]

export const isCostFilterKey = (value: unknown): value is CostFilterKey => COST_FILTERS.some((f) => f.key === value)

export const categoriesOfFilter = (key: CostFilterKey): string[] => COST_FILTERS.find((f) => f.key === key)?.categories ?? []

const ENTRY_THEMES: Record<string, Theme> = {
  energy: 'energy',
  toll: 'tolls',
  parking: 'tolls',
  travel_other: 'tolls',
  maintenance: 'maintenance',
  repair: 'repairs',
  insurance: 'insurance',
  subscription: 'subscriptions',
  tax: 'taxes',
  financing: 'financing',
  tires: 'tires',
  accessory: 'accessories',
}

/** The theme (hue) of a ledger line: the same for its glyph, its chip and its figure. */
export const themeOfEntry = (category: string): Theme => ENTRY_THEMES[category] ?? 'other'

/** The glyph of a ledger line, the one of the filter that keeps its category. */
export const iconOfEntry = (category: string): Component => COST_FILTERS.find((f) => f.icon && f.categories.includes(category))?.icon ?? Receipt

/** An amount of the API as a number of the vehicle's currency. */
export const amountValue = (a: ApiAmount | null | undefined): number => {
  const n = Number(a?.amount)
  return Number.isFinite(n) ? n : 0
}

/** The cost of a kilometre over a period, null while the period has no distance. */
export const costPerKm = (amount: number, distanceKm: number): number | null => (distanceKm > 0 ? amount / distanceKm : null)

/** The key of a line: the record it comes from, its day and its figure (an instalment repeats its record). */
export const entryKey = (e: CostEntryLine): string => `${e.source}:${e.source_id}:${e.date}:${e.category}:${e.amount.amount}`

/** What a line is called: the note the owner typed, else the place of a charge, else what was recorded (a charge, a fill-up), else its category. */
export function entryLabel(e: CostEntryLine): string {
  if (e.note) return e.note
  if (e.place) return e.place
  return e.kind === 'charge' || e.kind === 'fuel_fill_up' ? t(`costs.kinds.${e.kind}`) : t(`costs.categories.${e.category}`)
}

/** The second line of a row: the place of a charge that has a note, the energy bought. */
export function entryDetail(e: CostEntryLine, formatNumber: (n: number) => string): string {
  const parts: string[] = []
  if (e.note && e.place) parts.push(e.place)
  if (e.kwh != null) parts.push(`${formatNumber(e.kwh)} kWh`)
  if (e.litres != null) parts.push(`${formatNumber(e.litres)} L`)
  if (e.recurring) parts.push(t('costs.recurring'))
  return parts.join(' · ')
}
