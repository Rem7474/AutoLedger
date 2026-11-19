import { describe, expect, it } from 'vitest'
import { hasOdometerReading, isHybridCapabilities, odometerRequired, readBatteryLevels } from './hybridForms'

describe('isHybridCapabilities', () => {
  it('needs both a battery and a tank', () => {
    expect(isHybridCapabilities({ charges: true, refuels: true })).toBe(true)
    expect(isHybridCapabilities({ charges: true, refuels: false })).toBe(false)
    expect(isHybridCapabilities({ charges: false, refuels: true })).toBe(false)
  })
})

describe('readBatteryLevels', () => {
  it('leaves blank fields as not given', () => {
    expect(readBatteryLevels('', '')).toEqual({ start: null, end: null, problem: null })
    expect(readBatteryLevels(null, undefined)).toEqual({ start: null, end: null, problem: null })
  })
  it('accepts one percentage alone', () => {
    expect(readBatteryLevels('20', '')).toEqual({ start: 20, end: null, problem: null })
    expect(readBatteryLevels('', '90')).toEqual({ start: null, end: 90, problem: null })
  })
  it('accepts the bounds and rounds to a whole percentage', () => {
    expect(readBatteryLevels('0', '100')).toEqual({ start: 0, end: 100, problem: null })
    expect(readBatteryLevels('20,4', '80.6')).toEqual({ start: 20, end: 81, problem: null })
  })
  it('flags a percentage outside 0-100', () => {
    expect(readBatteryLevels('-1', '50').problem).toBe('range')
    expect(readBatteryLevels('10', '101').problem).toBe('range')
  })
  it('flags a charge that does not end higher than it started', () => {
    expect(readBatteryLevels('60', '60').problem).toBe('order')
    expect(readBatteryLevels('80', '30').problem).toBe('order')
    expect(readBatteryLevels('30', '80').problem).toBeNull()
  })
  it('does not compare when only one end is given', () => {
    expect(readBatteryLevels('95', '').problem).toBeNull()
  })
})

describe('odometerRequired', () => {
  it('applies to a hybrid whose fields can be edited', () => {
    expect(odometerRequired(true, false)).toBe(true)
    expect(odometerRequired(true, true)).toBe(false)
    expect(odometerRequired(false, false)).toBe(false)
  })
})

describe('hasOdometerReading', () => {
  it('counts blank and zero as missing', () => {
    expect(hasOdometerReading('')).toBe(false)
    expect(hasOdometerReading(null)).toBe(false)
    expect(hasOdometerReading('0')).toBe(false)
    expect(hasOdometerReading(12500)).toBe(true)
    expect(hasOdometerReading('12500')).toBe(true)
  })
})
