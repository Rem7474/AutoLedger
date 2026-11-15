import { describe, expect, it } from 'vitest'
import {
  buildVehiclePayload,
  type OnboardingVehicleForm,
} from './onboarding'

function form(overrides: Partial<OnboardingVehicleForm> = {}): OnboardingVehicleForm {
  return {
    name: 'Model 3',
    make: 'Tesla',
    model: 'Model 3',
    vin: '',
    powertrain: 'EV',
    odometer: '15000',
    ...overrides,
  }
}

describe('buildVehiclePayload', () => {
  it('never sends telemetry_mode', () => {
    expect(buildVehiclePayload(form())).not.toHaveProperty('telemetry_mode')
  })

  it('creates a source-less vehicle by default', () => {
    const p = buildVehiclePayload(form({ vin: '' }))
    expect(p).toMatchObject({ name: 'Model 3', powertrain: 'EV', current_odometer: 15000, teslamate_auth_type: 'NONE' })
    expect(p.vin).toBeUndefined()
    expect(p).not.toHaveProperty('teslamate_api_url')
  })

  it('falls back to a zero odometer when the field is empty or invalid', () => {
    expect(buildVehiclePayload(form({ odometer: '' })).current_odometer).toBe(0)
    expect(buildVehiclePayload(form({ odometer: 'abc' })).current_odometer).toBe(0)
  })

  it('never carries a TeslaMate connection: sources are connected later', () => {
    for (const powertrain of ['EV', 'ICE', 'PHEV', 'REEV'] as const) {
      const p = buildVehiclePayload(form({ powertrain }))
      expect(p).not.toHaveProperty('teslamate_api_url')
      expect(p.teslamate_auth_type).toBe('NONE')
    }
  })

  it('keeps the VIN when given', () => {
    expect(buildVehiclePayload(form({ vin: 'VIN123' })).vin).toBe('VIN123')
  })

  it('sends the chosen currency, and none when unset', () => {
    expect(buildVehiclePayload(form({ currency: 'GBP' })).currency).toBe('GBP')
    expect(buildVehiclePayload(form())).not.toHaveProperty('currency')
  })
})
