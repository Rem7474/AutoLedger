import { describe, expect, it } from 'vitest'
import { acceptBody, capacityBody, clearBody, fieldNumber, hybridShareHint, percentIsValid, percentToShare, sharePercent, suggestionStatus, typedBody } from './hybridShare'

const none = { electric_share: null, fuel_l_100km: null, battery_usable_kwh: null }

describe('hybrid share helpers', () => {
  it('shows a share as a whole percentage', () => {
    expect(sharePercent(0.4)).toBe(40)
    expect(sharePercent(0.4049)).toBe(40)
    expect(sharePercent(0.405)).toBe(41)
  })

  it('keeps the three decimals the server stores', () => {
    expect(percentToShare(40)).toBe(0.4)
    expect(percentToShare(33.33)).toBe(0.333)
    expect(percentToShare(0)).toBe(0)
    expect(percentToShare(100)).toBe(1)
  })

  it('tells nothing to suggest, an accepted value and a value still to accept', () => {
    expect(suggestionStatus({ accepted: none })).toBe('insufficient')
    expect(suggestionStatus({ suggested_electric_share: 0.4, accepted: none })).toBe('available')
    expect(suggestionStatus({ suggested_electric_share: 0.4, accepted: { ...none, electric_share: 0.6 } })).toBe('available')
    expect(suggestionStatus({ suggested_electric_share: 0.4, accepted: { ...none, electric_share: 0.4 } })).toBe('accepted')
    // 0 % is a value, not an absence
    expect(suggestionStatus({ suggested_electric_share: 0, accepted: { ...none, electric_share: 0 } })).toBe('accepted')
  })

  it('says on the sheet row what is accepted, else what is suggested, else nothing', () => {
    expect(hybridShareHint(null)).toBe('none')
    expect(hybridShareHint({ accepted: none })).toBe('none')
    expect(hybridShareHint({ suggested_electric_share: 0.3, accepted: none })).toBe('suggested')
    expect(hybridShareHint({ suggested_electric_share: 0.3, accepted: { ...none, electric_share: 0.5 } })).toBe('accepted')
    expect(hybridShareHint({ accepted: { ...none, electric_share: 0 } })).toBe('accepted')
  })
})

describe('hybrid share requests', () => {
  const toL100 = (v: number) => v * 2
  it('reads a number typed with a comma or a point, and nothing else', () => {
    expect(fieldNumber('12,5')).toBe(12.5)
    expect(fieldNumber(' 7.25 ')).toBe(7.25)
    expect(fieldNumber('')).toBeNull()
    expect(fieldNumber(null)).toBeNull()
    expect(fieldNumber(undefined)).toBeNull()
    expect(fieldNumber('abc')).toBeNull()
    expect(fieldNumber(0)).toBe(0)
  })

  it('accepts a share from 0 to 100 %', () => {
    expect(percentIsValid('0')).toBe(true)
    expect(percentIsValid('100')).toBe(true)
    expect(percentIsValid('35,5')).toBe(true)
    expect(percentIsValid('100,1')).toBe(false)
    expect(percentIsValid('-1')).toBe(false)
    expect(percentIsValid('')).toBe(false)
    expect(percentIsValid('x')).toBe(false)
  })

  it('builds the typed value with the consumption in litres per 100 km and the capacity correction', () => {
    expect(typedBody({ percent: '35', fuel: '3,5', capacity: '12,5' }, toL100)).toEqual({ electric_share: 0.35, fuel_l_100km: 7, battery_usable_kwh: 12.5 })
    expect(typedBody({ percent: '35', fuel: '', capacity: '' }, toL100)).toEqual({ electric_share: 0.35, fuel_l_100km: null, battery_usable_kwh: null })
    expect(typedBody({ percent: '140', fuel: '3', capacity: '' }, toL100)).toBeNull()
  })

  it('keeps the capacity correction when accepting, clearing or correcting', () => {
    expect(acceptBody(null)).toEqual({ accept_suggestion: true, battery_usable_kwh: null })
    expect(acceptBody({ ...none, battery_usable_kwh: 14 })).toEqual({ accept_suggestion: true, battery_usable_kwh: 14 })
    expect(clearBody('13')).toEqual({ electric_share: null, fuel_l_100km: null, battery_usable_kwh: 13 })
    expect(clearBody('')).toEqual({ electric_share: null, fuel_l_100km: null, battery_usable_kwh: null })
    expect(capacityBody({ electric_share: 0.4, fuel_l_100km: 6, battery_usable_kwh: null }, '12,5')).toEqual({ electric_share: 0.4, fuel_l_100km: 6, battery_usable_kwh: 12.5 })
    expect(capacityBody(null, '')).toEqual({ electric_share: null, fuel_l_100km: null, battery_usable_kwh: null })
  })
})
