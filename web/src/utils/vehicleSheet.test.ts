import { describe, expect, it } from 'vitest'
import { vehicleSheetGroups, type VehicleSheetInput } from './vehicleSheet'

const base: VehicleSheetInput = { charges: true, refuels: false, isOwner: true, hasTco: true, vehicleCount: 2 }

describe('vehicle sheet groups', () => {
  it('puts the comparison simulator in possession, not in follow-up or settings', () => {
    const g = vehicleSheetGroups(base)
    expect(g.ownership).toContain('comparison')
    expect(g.follow).not.toContain('comparison')
    expect(g.settings).not.toContain('comparison')
  })

  it('lays out an electric vehicle owner', () => {
    expect(vehicleSheetGroups(base)).toEqual({
      follow: ['maintenance', 'consumption', 'tires', 'readings'],
      ownership: ['battery', 'resale', 'lease', 'totals', 'contract', 'comparison'],
      settings: ['homeCharging', 'dataSources', 'sharing', 'edit', 'manage'],
    })
  })

  it('a combustion vehicle has no battery, no hybrid share and no home charging', () => {
    const g = vehicleSheetGroups({ ...base, charges: false, refuels: true })
    expect(g.follow).toContain('consumption')
    expect(g.ownership).not.toContain('battery')
    expect(g.ownership).not.toContain('hybridShare')
    expect(g.settings).not.toContain('homeCharging')
  })

  it('a hybrid shows the electric share next to the battery', () => {
    const g = vehicleSheetGroups({ ...base, refuels: true })
    expect(g.ownership.slice(0, 2)).toEqual(['battery', 'hybridShare'])
    expect(g.settings[0]).toBe('homeCharging')
  })

  it('a vehicle that neither charges nor refuels has no consumption line', () => {
    expect(vehicleSheetGroups({ ...base, charges: false }).follow).not.toContain('consumption')
  })

  it('a member sees neither the contract nor the edit line, and totals only with data', () => {
    const g = vehicleSheetGroups({ ...base, isOwner: false, hasTco: false })
    expect(g.ownership).not.toContain('contract')
    expect(g.ownership).not.toContain('totals')
    expect(g.settings).not.toContain('edit')
    expect(g.settings).toContain('sharing')
  })

  it('hides the vehicle manager for a single vehicle', () => {
    expect(vehicleSheetGroups({ ...base, vehicleCount: 1 }).settings).not.toContain('manage')
  })
})
