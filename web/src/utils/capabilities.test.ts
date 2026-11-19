import { describe, expect, it } from 'vitest'
import { capabilitiesOf, energyKind } from './capabilities'

const caps = (over: Record<string, boolean>) => ({
  charges: false, refuels: false, drives: false, sync: false, odometer_auto: false, battery_health: false, tolls_auto: false, write: false, ...over,
})

describe('capabilitiesOf', () => {
  it('reads what the server sent, whatever the powertrain says', () => {
    const sent = caps({ refuels: true, drives: true, odometer_auto: true })
    expect(capabilitiesOf({ powertrain: 'EV', role: 'VIEWER', capabilities: sent })).toEqual(sent)
  })

  it('reads a missing flag of a partial answer as false', () => {
    expect(capabilitiesOf({ capabilities: { charges: true } })).toEqual(caps({ charges: true }))
  })

  it('reads the powertrain, the role and the connection when the answer carries no capabilities', () => {
    const ev = capabilitiesOf({ powertrain: 'EV', role: 'OWNER', teslamate_api_url: 'http://tm:8080' })
    expect(ev).toEqual(caps({ charges: true, drives: true, sync: true, odometer_auto: true, battery_health: true, tolls_auto: true, write: true }))

    const ice = capabilitiesOf({ powertrain: 'ICE', role: 'EDITOR', toll_detection_active: false })
    expect(ice).toEqual(caps({ refuels: true, drives: true, write: true }))

    const phev = capabilitiesOf({ powertrain: 'PHEV', role: 'VIEWER', teslamate_api_url: '  ' })
    expect(phev).toEqual(caps({ charges: true, refuels: true, drives: true, battery_health: true, tolls_auto: true }))
  })

  it('ignores a connection on a vehicle that cannot plug in', () => {
    expect(capabilitiesOf({ powertrain: 'ICE', teslamate_api_url: 'http://tm' }).sync).toBe(false)
  })

  it('reads no vehicle as an electric vehicle owned by the user', () => {
    expect(capabilitiesOf(null)).toEqual(caps({ charges: true, drives: true, battery_health: true, tolls_auto: true, write: true }))
  })
})

describe('energyKind', () => {
  it('reads fuel for a vehicle that burns fuel only, kWh for every vehicle that charges, a hybrid included', () => {
    expect(energyKind(capabilitiesOf({ powertrain: 'ICE' }))).toBe('FUEL')
    expect(energyKind(capabilitiesOf({ powertrain: 'EV' }))).toBe('ELECTRIC')
    expect(energyKind(capabilitiesOf({ powertrain: 'PHEV' }))).toBe('ELECTRIC')
    expect(energyKind(capabilitiesOf({ powertrain: 'REEV' }))).toBe('ELECTRIC')
  })
})
