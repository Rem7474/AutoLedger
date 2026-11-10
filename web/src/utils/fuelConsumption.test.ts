import { describe, expect, it } from 'vitest'
import { fuelConsumptionFigures } from './fuelConsumption'

describe('fuelConsumptionFigures', () => {
  it('gives the litres per 100 km and the cost of 100 km at the average price', () => {
    expect(fuelConsumptionFigures({ consumption_l_100km: 6, avg_price_per_liter: 1.5 })).toEqual({ litresPer100: 6, costPer100: 9 })
  })

  it('keeps the consumption when no price is known', () => {
    expect(fuelConsumptionFigures({ consumption_l_100km: 6, avg_price_per_liter: 0 })).toEqual({ litresPer100: 6, costPer100: null })
    expect(fuelConsumptionFigures({ consumption_l_100km: 6 })).toEqual({ litresPer100: 6, costPer100: null })
  })

  it('has nothing without a measured consumption', () => {
    expect(fuelConsumptionFigures({ consumption_l_100km: null, avg_price_per_liter: 1.5 })).toBeNull()
    expect(fuelConsumptionFigures({ consumption_l_100km: 0 })).toBeNull()
    expect(fuelConsumptionFigures(null)).toBeNull()
  })
})
