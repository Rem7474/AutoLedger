import { describe, expect, it } from 'vitest'
import type { TollInvoiceMonth } from '@/services/api'
import { gapKind, gapOfTotal, INVOICE_MONTHS_SHOWN, listedInvoiceMonths, monthDate } from './tollInvoices'

const month = (m: string, tripTolls: number, total: number | null = null): TollInvoiceMonth => ({
  month: m,
  trip_tolls: tripTolls,
  invoice: total === null ? null : { id: m, vehicle_id: 'v', month: m, total, note: null, version: 1, updated_at: '', trip_tolls: tripTolls, gap: total - tripTolls },
})

describe('listedInvoiceMonths', () => {
  const months = [month('2026-11', 0), month('2026-10', 0), month('2026-09', 0), month('2026-08', 12), month('2026-07', 0, 30), month('2026-06', 0)]

  it('always lists the two latest months and keeps the others only with tolls or an invoice', () => {
    const { rows } = listedInvoiceMonths(months, true)
    expect(rows.map((r) => r.month)).toEqual(['2026-11', '2026-10', '2026-08', '2026-07'])
  })

  it('cuts the list and says how many months stay hidden', () => {
    const many = Array.from({ length: 12 }, (_, i) => month(`2026-${String(12 - i).padStart(2, '0')}`, 5))
    const cut = listedInvoiceMonths(many, false)
    expect(cut.rows).toHaveLength(INVOICE_MONTHS_SHOWN)
    expect(cut.hidden).toBe(12 - INVOICE_MONTHS_SHOWN)
    expect(listedInvoiceMonths(many, true)).toEqual({ rows: many, hidden: 0 })
  })

  it('lists nothing hidden when everything fits', () => {
    expect(listedInvoiceMonths([month('2026-11', 0)], false)).toEqual({ rows: [month('2026-11', 0)], hidden: 0 })
  })
})

describe('gapKind', () => {
  it('reads the sign of the gap, a fraction of a cent counting as none', () => {
    expect(gapKind(5.4)).toBe('unattached')
    expect(gapKind(-3.6)).toBe('lower')
    expect(gapKind(0)).toBe('none')
    expect(gapKind(0.004)).toBe('none')
    expect(gapKind(-0.004)).toBe('none')
  })
})

describe('gapOfTotal', () => {
  it('is the typed total minus the tolls of the trips, to the cent', () => {
    expect(gapOfTotal('30', 23.6)).toBe(6.4)
    expect(gapOfTotal(20, 23.6)).toBe(-3.6)
    expect(gapOfTotal('0', 0)).toBe(0)
  })

  it('is unknown while the total is empty or not a number', () => {
    expect(gapOfTotal('', 10)).toBeNull()
    expect(gapOfTotal('abc', 10)).toBeNull()
  })
})

describe('monthDate', () => {
  it('is a day of the month, whatever the timezone', () => {
    const d = monthDate('2026-03')
    expect([d.getFullYear(), d.getMonth()]).toEqual([2026, 2])
  })
})
