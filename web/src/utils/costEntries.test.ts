import { describe, expect, it } from 'vitest'
import type { CostEntryLine } from '@/services/api'
import { amountValue, categoriesOfFilter, COST_FILTERS, costPerKm, entryDetail, entryKey, entryLabel, iconOfEntry, isCostFilterKey, themeOfEntry, entryTarget } from './costEntries'

const line = (over: Partial<CostEntryLine> = {}): CostEntryLine => ({
  date: '2026-03-10',
  category: 'energy',
  kind: 'charge',
  amount: { amount: '12.50', currency: 'EUR' },
  note: null,
  place: null,
  place_category: null,
  kwh: null,
  litres: null,
  recurring: false,
  has_receipt: null,
  source: 'charge_logs',
  source_id: 'c1',
  ...over,
})

describe('the filters of the Costs page', () => {
  it('keep every ledger category under exactly one filter', () => {
    const ledger = ['energy', 'toll', 'parking', 'travel_other', 'maintenance', 'repair', 'insurance', 'subscription', 'tax', 'financing', 'tires', 'accessory', 'other']
    const kept = COST_FILTERS.flatMap((f) => f.categories)
    expect([...kept].sort()).toEqual([...ledger].sort())
  })

  it('know their key, and "all" keeps every category', () => {
    expect(isCostFilterKey('tolls')).toBe(true)
    expect(isCostFilterKey('nope')).toBe(false)
    expect(categoriesOfFilter('all')).toEqual([])
    expect(categoriesOfFilter('maintenance')).toEqual(['maintenance', 'repair'])
  })
})

describe('a ledger line', () => {
  it('takes the theme and the glyph of its category', () => {
    expect(themeOfEntry('parking')).toBe('tolls')
    expect(themeOfEntry('repair')).toBe('repairs')
    expect(themeOfEntry('unknown')).toBe('other')
    expect(iconOfEntry('toll')).toBe(iconOfEntry('parking'))
    expect(iconOfEntry('unknown')).toBe(iconOfEntry('tax'))
  })

  it('is named by the note, then the place, then its category', () => {
    expect(entryLabel(line({ note: 'Brakes', place: 'Home' }))).toBe('Brakes')
    expect(entryLabel(line({ place: 'Home' }))).toBe('Home')
    expect(entryLabel(line())).toBe('Recharge')
    expect(entryLabel(line({ kind: 'fuel_fill_up' }))).toBe('Plein')
    expect(entryLabel(line({ kind: 'financing', category: 'financing' }))).toBe('Financement')
  })

  it('says the place under a note, the energy bought and the recurrence', () => {
    const fmt = (n: number) => String(n)
    expect(entryDetail(line({ note: 'Night', place: 'Home', kwh: 30 }), fmt)).toBe('Home · 30 kWh')
    expect(entryDetail(line({ place: 'Home' }), fmt)).toBe('')
    expect(entryDetail(line({ category: 'insurance', recurring: true }), fmt)).toBe('récurrent')
    expect(entryDetail(line({ litres: 40 }), fmt)).toBe('40 L')
  })

  it('is told from another by its record, day and figure', () => {
    expect(entryKey(line())).not.toBe(entryKey(line({ date: '2026-04-10' })))
    expect(entryKey(line())).toBe(entryKey(line()))
  })
})

describe('the figures of a period', () => {
  it('reads an amount of the API, zero when it is not a number', () => {
    expect(amountValue({ amount: '92.50', currency: 'EUR' })).toBe(92.5)
    expect(amountValue({ amount: 'x', currency: 'EUR' })).toBe(0)
    expect(amountValue(undefined)).toBe(0)
  })

  it('has no cost per kilometre without a distance', () => {
    expect(costPerKm(50, 200)).toBe(0.25)
    expect(costPerKm(50, 0)).toBeNull()
  })
})

describe('entryTarget', () => {
  it('opens the form of the record a line comes from', () => {
    expect(entryTarget({ source: 'charge_logs', source_id: 'c1' })).toEqual({ type: 'edit', modal: 'charge', id: 'c1' })
    expect(entryTarget({ source: 'drive_expenses', source_id: 'e1' })).toEqual({ type: 'edit', modal: 'toll', id: 'e1' })
    expect(entryTarget({ source: 'maintenance_expenses', source_id: 'm1' })).toEqual({ type: 'edit', modal: 'maintenance', id: 'm1' })
  })
  it('sends the lines held by another screen there, and leaves the others alone', () => {
    expect(entryTarget({ source: 'fuel_logs', source_id: 'f1' })).toEqual({ type: 'link', to: '/consumption?tab=FUEL' })
    expect(entryTarget({ source: 'tires', source_id: 't1' })).toEqual({ type: 'link', to: '/tires' })
    expect(entryTarget({ source: 'vehicle_ownership', source_id: 'o1' })).toEqual({ type: 'none' })
  })
})
