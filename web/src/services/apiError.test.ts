import { afterEach, describe, expect, it } from 'vitest'
import { setLocale } from '@/i18n'
import { apiErrorMessage, apiMessageText, asNetworkError, NetworkError } from './apiError'
import { setDistanceUnit } from '@/units'

afterEach(() => {
  setLocale('fr')
  setDistanceUnit(null)
})

describe('apiErrorMessage', () => {
  it('translates a known code in the current language', () => {
    expect(apiErrorMessage({ error: 'Vehicle not found', code: 'vehicle.not_found' }, 'x')).toBe('Véhicule introuvable')
    setLocale('en')
    expect(apiErrorMessage({ error: 'Vehicle not found', code: 'vehicle.not_found' }, 'x')).toBe('Vehicle not found')
  })

  it('fills the parameters of the message', () => {
    expect(apiErrorMessage({ code: 'tire.not_in_storage', params: { p0: 'abc' } }, 'x')).toBe("Le pneu abc n'est pas en stockage")
    setLocale('en')
    expect(apiErrorMessage({ code: 'carpool.too_many_seats', params: { p0: 4, p1: 2 } }, 'x')).toBe('More than 4 seats taken on leg 2')
  })

  it('shows distances and figures per km in the account\'s unit', () => {
    setLocale('en')
    const odometer = { code: 'tire.odometer_below_mount', params: { p0: 'km:16093.44', p1: 'km:32186.88' } }
    expect(apiErrorMessage(odometer, 'x')).toBe('The odometer (16,093 km) is lower than the fitting odometer (32,187 km)')
    setDistanceUnit('mi')
    expect(apiErrorMessage(odometer, 'x')).toBe('The odometer (10,000 mi) is lower than the fitting odometer (20,000 mi)')
    expect(apiErrorMessage({ code: 'comparison.ev_consumption', params: { p0: 'perkm:0.1', p1: 'perkm:100' } }, 'x')).toBe(
      'Invalid electric consumption (0.16 to 160.93 kWh/100 mi)',
    )
  })

  it('falls back to the server message for an unknown or missing code', () => {
    expect(apiErrorMessage({ error: 'Something odd', code: 'nope.unknown' }, 'x')).toBe('Something odd')
    expect(apiErrorMessage({ error: 'Plain message' }, 'x')).toBe('Plain message')
    expect(apiErrorMessage({}, 'fallback')).toBe('fallback')
    expect(apiErrorMessage(null, 'fallback')).toBe('fallback')
  })
})

describe('apiMessageText', () => {
  it('translates a payload message and its keyword parameters', () => {
    const warning = { code: 'sync.page_limit', message: 'x', params: { p0: 'kw:drives', p1: 500 } }
    expect(apiMessageText(warning)).toBe('Trajets : limite de 500 pages atteinte, historique partiellement importé')
    setLocale('en')
    expect(apiMessageText(warning)).toBe('Drives: limit of 500 pages reached, history partially imported')
  })

  it('translates an error nested in the parameters of another one', () => {
    const error = {
      code: 'sync.unreachable',
      error: 'x',
      params: { p0: 'http://tm:8080', p1: { code: 'teslamate.timeout', message: 'y', params: { p0: 'dial failed' } } },
    }
    setLocale('en')
    expect(apiErrorMessage(error, 'z')).toBe(
      'Cannot reach TeslaMate (http://tm:8080): dial failed (Timed out: check that the address and port are reachable and that TeslaMate is responding)',
    )
  })

  it('shows the server text of a message without a catalog entry', () => {
    expect(apiMessageText({ code: 'unknown.thing', message: 'Server text' })).toBe('Server text')
  })
})

describe('asNetworkError', () => {
  it('swaps the browser\'s failed-fetch TypeError for a translated message', () => {
    const swapped = asNetworkError(new TypeError('Failed to fetch'))
    expect(swapped).toBeInstanceOf(NetworkError)
    expect((swapped as Error).message).not.toMatch(/Failed to fetch/)
  })

  it('leaves every other error as it is', () => {
    const other = new Error('boom')
    expect(asNetworkError(other)).toBe(other)
  })
})
