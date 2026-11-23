import { describe, expect, it } from 'vitest'
import { tagEnergyLine } from './tagBreakdown'

describe('tagEnergyLine', () => {
  it('gives the kWh of a vehicle that charges', () => {
    expect(tagEnergyLine({ distance_km: 120, energy_kwh: 21.5 })).toContain('kWh')
  })
  it('gives the litres of a vehicle that burns fuel, never a 0 kWh', () => {
    const line = tagEnergyLine({ distance_km: 120, energy_kwh: 0, fuel_only: true, fuel_liters: 9.6 })
    expect(line).not.toContain('kWh')
    expect(line).toContain('9')
    expect(line).toMatch(/\bL\b/)
  })
  it('leaves the energy out while the litres are unknown', () => {
    const line = tagEnergyLine({ distance_km: 120, fuel_only: true })
    expect(line).not.toContain('kWh')
    expect(line).not.toContain('(')
  })
})
