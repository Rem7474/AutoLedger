import { describe, expect, it } from 'vitest'
import { capabilitiesOf } from './capabilities'
import { capabilityRedirect } from './capabilityRoutes'

describe('capabilityRedirect', () => {
  const ev = capabilitiesOf({ powertrain: 'EV' })
  const ice = capabilitiesOf({ powertrain: 'ICE' })
  const phev = capabilitiesOf({ powertrain: 'PHEV' })
  const withoutTrips = capabilitiesOf({ capabilities: { ...ice, drives: false } })

  it('sends a vehicle whose answer carries no trips capability from the trips page to the home of the vehicle', () => {
    expect(capabilityRedirect('drives', withoutTrips)).toBe('/vehicle')
  })

  it('lets every kind of vehicle open the trips page: trips can be entered by hand', () => {
    expect(capabilityRedirect('drives', ev)).toBeNull()
    expect(capabilityRedirect('drives', ice)).toBeNull()
    expect(capabilityRedirect('drives', phev)).toBeNull()
  })

  it('leaves the pages that need no capability alone', () => {
    expect(capabilityRedirect('costs', ice)).toBeNull()
    expect(capabilityRedirect(undefined, ice)).toBeNull()
  })
})
