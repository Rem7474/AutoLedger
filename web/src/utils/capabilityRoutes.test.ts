import { describe, expect, it } from 'vitest'
import { capabilitiesOf } from './capabilities'
import { capabilityRedirect } from './capabilityRoutes'

describe('capabilityRedirect', () => {
  const ev = capabilitiesOf({ powertrain: 'EV' })
  const ice = capabilitiesOf({ powertrain: 'ICE' })
  const iceWithTrips = capabilitiesOf({ capabilities: { ...ice, drives: true } })

  it('sends a vehicle without trips from the trips page to the home of the vehicle', () => {
    expect(capabilityRedirect('drives', ice)).toBe('/vehicle')
  })

  it('lets a vehicle with the capability open the page, however it got it', () => {
    expect(capabilityRedirect('drives', ev)).toBeNull()
    expect(capabilityRedirect('drives', iceWithTrips)).toBeNull()
  })

  it('leaves the pages that need no capability alone', () => {
    expect(capabilityRedirect('costs', ice)).toBeNull()
    expect(capabilityRedirect(undefined, ice)).toBeNull()
  })
})
