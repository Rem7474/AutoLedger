import { describe, expect, it } from 'vitest'
import { teslamateChargeCostUrl, teslamateChargeUrl } from './teslamateCharges'

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

describe('teslamateChargeCostUrl', () => {
  it('points to the cost page of a recent charge whose cost is unknown', () => {
    expect(teslamateChargeCostUrl(vehicle, charge, now)).toBe('http://tm.local/charge-cost/152')
    expect(teslamateChargeCostUrl(vehicle, { ...charge, cost: undefined }, now)).toBe('http://tm.local/charge-cost/152')
  })

  it('has no link once the cost is known, even a free one', () => {
    expect(teslamateChargeCostUrl(vehicle, { ...charge, cost: 0 }, now)).toBeNull()
    expect(teslamateChargeCostUrl(vehicle, { ...charge, cost: 4.2 }, now)).toBeNull()
  })

  it('has no link for a cost typed in AutoLedger, which a synchronisation never overwrites', () => {
    expect(teslamateChargeCostUrl(vehicle, { ...charge, cost_source: 'MANUAL' }, now)).toBeNull()
  })

  it('has no link when the next synchronisation would not read the charge again', () => {
    const old = { ...charge, date: '2026-09-01T10:00:00Z', end_date: '2026-09-01T11:00:00Z' }
    expect(teslamateChargeCostUrl(vehicle, old, now)).toBeNull()
  })

  it('has no link without a TeslaMate URL, without a TeslaMate charge, or for another origin', () => {
    expect(teslamateChargeCostUrl({ teslamate_grafana_url: 'http://g' }, charge, now)).toBeNull()
    expect(teslamateChargeCostUrl(null, charge, now)).toBeNull()
    expect(teslamateChargeCostUrl(vehicle, { ...charge, teslamate_charge_id: undefined }, now)).toBeNull()
    expect(teslamateChargeCostUrl(vehicle, { ...charge, origin: 'CSV' }, now)).toBeNull()
  })

  it('defaults the reference time to now', () => {
    expect(teslamateChargeCostUrl(vehicle, { ...charge, date: new Date().toISOString() })).toBe('http://tm.local/charge-cost/152')
  })
})
