import { describe, expect, it } from 'vitest'
import { chargeCostToReview, teslamateChargeUrl, teslamateCostEditUrl } from './teslamateCharges'

const vehicle = { teslamate_grafana_url: 'http://grafana.local', teslamate_url: 'http://tm.local', teslamate_car_id: 2 }
const charge = {
  teslamate_charge_id: 152,
  origin: 'TESLAMATE',
  date: '2026-10-30T17:28:40Z',
  end_date: '2026-10-30T17:52:53Z',
  cost: null,
  cost_source: 'TESLAMATE',
}
const now = new Date('2026-11-05T12:00:00Z').getTime()

describe('teslamateChargeUrl', () => {
  it('builds the Charge Details link with a one minute margin around the charge', () => {
    const url = new URL(teslamateChargeUrl(vehicle, charge)!)
    expect(url.origin + url.pathname).toBe('http://grafana.local/d/BHhxFeZRz/charge-details')
    expect(url.searchParams.get('var-charging_process_id')).toBe('152')
    expect(url.searchParams.get('var-car_id')).toBe('2')
    expect(url.searchParams.get('var-base_url')).toBe('http://tm.local')
    expect(Number(url.searchParams.get('from'))).toBe(new Date(charge.date).getTime() - 60_000)
    expect(Number(url.searchParams.get('to'))).toBe(new Date(charge.end_date).getTime() + 60_000)
  })

  it('leaves the base URL out without a TeslaMate URL and falls back on car 1 and on the start for an open charge', () => {
    const url = new URL(
      teslamateChargeUrl({ teslamate_grafana_url: 'http://grafana.local' }, { ...charge, end_date: null })!,
    )
    expect(url.searchParams.has('var-base_url')).toBe(false)
    expect(url.searchParams.get('var-car_id')).toBe('1')
    expect(Number(url.searchParams.get('to'))).toBe(new Date(charge.date).getTime() + 60_000)
  })

  it('has no link without Grafana, without a TeslaMate charge, or for a charge of another origin', () => {
    expect(teslamateChargeUrl({}, charge)).toBeNull()
    expect(teslamateChargeUrl(null, charge)).toBeNull()
    expect(teslamateChargeUrl(vehicle, null)).toBeNull()
    expect(teslamateChargeUrl(vehicle, { ...charge, teslamate_charge_id: null })).toBeNull()
    expect(teslamateChargeUrl(vehicle, { ...charge, origin: 'WEBHOOK' })).toBeNull()
  })
})

describe('chargeCostToReview', () => {
  it('follows the server when it says so', () => {
    expect(chargeCostToReview({ cost: 0, cost_to_review: true })).toBe(true)
    expect(chargeCostToReview({ cost: null, cost_to_review: false })).toBe(false)
  })

  it('falls back on an unknown cost for a charge read without the flag', () => {
    expect(chargeCostToReview({ cost: null })).toBe(true)
    expect(chargeCostToReview({ cost: undefined })).toBe(true)
    expect(chargeCostToReview({ cost: 0 })).toBe(false)
    expect(chargeCostToReview(null)).toBe(false)
  })
})

describe('teslamateCostEditUrl', () => {
  const priced = { ...charge, cost: 9.44, cost_to_review: false }

  it('sends the cost of a synchronised charge to TeslaMate while it reads it again, priced or not', () => {
    expect(teslamateCostEditUrl(vehicle, priced, now)).toBe('http://tm.local/charge-cost/152')
    expect(teslamateCostEditUrl(vehicle, { ...charge, cost: 0 }, now)).toBe('http://tm.local/charge-cost/152')
  })

  it('leaves the cost to AutoLedger when TeslaMate cannot give it back', () => {
    expect(teslamateCostEditUrl({ teslamate_grafana_url: 'x' }, priced, now)).toBeNull() // no TeslaMate URL
    expect(teslamateCostEditUrl(vehicle, priced, new Date('2026-12-30T00:00:00Z').getTime())).toBeNull() // out of the window
    expect(teslamateCostEditUrl(vehicle, { ...priced, cost_source: 'MANUAL' }, now)).toBeNull() // already corrected here
    expect(teslamateCostEditUrl(vehicle, { ...priced, origin: 'WEBHOOK' }, now)).toBeNull() // not synchronised from TeslaMate
  })

  it('has no link without a TeslaMate charge id, and reads the reference time from the clock by default', () => {
    expect(teslamateCostEditUrl(vehicle, { ...priced, teslamate_charge_id: undefined }, now)).toBeNull()
    expect(teslamateCostEditUrl(vehicle, { ...priced, date: new Date().toISOString() })).toBe('http://tm.local/charge-cost/152')
  })
})
