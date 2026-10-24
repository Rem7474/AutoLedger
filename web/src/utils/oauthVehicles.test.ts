import { describe, expect, it } from 'vitest'
import { canApproveGrant, coveredVehicles, grantedVehicleIds, toggleVehicle, type GrantVehicleChoice } from './oauthVehicles'

const choices: GrantVehicleChoice[] = [
  { id: 'a', name: 'Family car', role: 'OWNER' },
  { id: 'b', name: 'Van', role: 'OWNER' },
  { id: 'c', name: 'Shared', role: 'VIEWER' },
]

describe('grantedVehicleIds', () => {
  it('grants every vehicle with no list when the user did not limit the access', () => {
    expect(grantedVehicleIds('all', new Set(['a']), choices)).toBeNull()
  })

  it('lists the ticked vehicles in the order of the page, whatever the order they were ticked in', () => {
    expect(grantedVehicleIds('some', new Set(['c', 'a']), choices)).toEqual(['a', 'c'])
  })

  it('ignores a selection that names a vehicle the page does not offer', () => {
    expect(grantedVehicleIds('some', new Set(['gone', 'b']), choices)).toEqual(['b'])
  })
})

describe('canApproveGrant', () => {
  it('refuses a limited access with no vehicle', () => {
    expect(canApproveGrant('some', new Set(), choices)).toBe(false)
    expect(canApproveGrant('some', new Set(['gone']), choices)).toBe(false)
  })

  it('accepts a limited access with a vehicle, and an unlimited one with no tick', () => {
    expect(canApproveGrant('some', new Set(['b']), choices)).toBe(true)
    expect(canApproveGrant('all', new Set(), choices)).toBe(true)
  })
})

describe('toggleVehicle', () => {
  it('ticks and unticks without changing the set it was given', () => {
    const start = new Set(['a'])
    const ticked = toggleVehicle(start, 'b')
    expect([...ticked].sort()).toEqual(['a', 'b'])
    expect([...toggleVehicle(ticked, 'a')]).toEqual(['b'])
    expect([...start]).toEqual(['a'])
  })
})

describe('coveredVehicles', () => {
  it('names the vehicles of the account and counts the ones it no longer shows', () => {
    expect(coveredVehicles(['b', 'x', 'a', 'y'], choices)).toEqual({ names: ['Van', 'Family car'], missing: 2 })
  })

  it('has nothing to show for an empty list', () => {
    expect(coveredVehicles([], choices)).toEqual({ names: [], missing: 0 })
  })
})
