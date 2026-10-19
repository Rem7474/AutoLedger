import { describe, expect, it } from 'vitest'
import { dataAlertLine, type DataAlert } from './dataAlerts'

describe('dataAlertLine', () => {
  it('names the amount, the usual one and where to correct an expense', () => {
    const alert: DataAlert = { kind: 'amount_outlier', category: 'PARKING', source_id: 'e1', date: '2026-10-05T10:00:00Z', amount: 60, median: 3 }
    const line = dataAlertLine(alert, 'EUR')
    expect(line.to).toBe('/expenses')
    expect(line.text).toContain('60')
    expect(line.text).toContain('20')
    expect(line.text).toContain('3')
  })

  it('shows the price in the vehicle currency with its own unit and sends to the right energy tab', () => {
    const charge: DataAlert = { kind: 'price_outlier', category: 'CHARGE', source_id: 'c1', date: '2026-10-05T10:00:00Z', unit_price: 2, low: 0.2, high: 0.55 }
    const line = dataAlertLine(charge, 'GBP')
    expect(line.to).toBe('/energy?tab=CHARGES')
    expect(line.text).toMatch(/2,00\s*£/)
    expect(line.text).toContain('kWh')
    const fuel = dataAlertLine({ ...charge, category: 'FUEL' }, 'CHF')
    expect(fuel.to).toBe('/energy?tab=FUEL')
    expect(fuel.text).toContain('CHF')
    expect(fuel.text).not.toContain('kWh')
  })
})
