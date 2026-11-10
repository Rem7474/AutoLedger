import { describe, expect, it } from 'vitest'
import { ALL_VEHICLES, currencyLines, scopeFromQuery, selectorValue } from '@/utils/vehicleScope'

describe('scopeFromQuery', () => {
  it('asks for the household view with two vehicles or more', () => {
    expect(scopeFromQuery('all', 2)).toBe('all')
    expect(scopeFromQuery(['all', 'x'], 3)).toBe('all')
  })

  it('stays on the vehicle with one vehicle, or for any other value', () => {
    expect(scopeFromQuery('all', 1)).toBe('vehicle')
    expect(scopeFromQuery('all', 0)).toBe('vehicle')
    expect(scopeFromQuery('fleet', 3)).toBe('vehicle')
    expect(scopeFromQuery(undefined, 3)).toBe('vehicle')
    expect(scopeFromQuery(null, 3)).toBe('vehicle')
  })
})

describe('selectorValue', () => {
  it('shows the household entry in the household view and the active vehicle otherwise', () => {
    expect(selectorValue('all', 'v1')).toBe(ALL_VEHICLES)
    expect(selectorValue('vehicle', 'v1')).toBe('v1')
    expect(selectorValue('vehicle', undefined)).toBe('')
  })
})

describe('currencyLines', () => {
  const totals = [
    { currency: 'CHF', vehicle_count: 1, current_month_cost: 50 },
    { currency: 'EUR', vehicle_count: 2, current_month_cost: 120 },
  ]

  it('keeps one line per currency, the fleet currency first, and adds nothing', () => {
    expect(currencyLines(totals, 'EUR')).toEqual([
      { currency: 'EUR', amount: 120, vehicleCount: 2 },
      { currency: 'CHF', amount: 50, vehicleCount: 1 },
    ])
  })

  it('keeps the order when the fleet currency is not among the lines', () => {
    expect(currencyLines(totals, 'USD').map((l) => l.currency)).toEqual(['CHF', 'EUR'])
  })

  it('has no line without totals', () => {
    expect(currencyLines(null, 'EUR')).toEqual([])
    expect(currencyLines([], 'EUR')).toEqual([])
  })
})
