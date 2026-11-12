import { describe, expect, it } from 'vitest'
import { capabilitiesOf } from './capabilities'
import { csvColumns, csvTemplate, csvTemplateFilename, csvTemplateTypes, webhookSnippet } from './dataSources'

describe('csvTemplateTypes', () => {
  it('offers charges and drives to an electric vehicle, drives and fill-ups to a combustion one', () => {
    expect(csvTemplateTypes(capabilitiesOf({ powertrain: 'EV' }))).toEqual(['CHARGES', 'DRIVES', 'ODOMETER'])
    expect(csvTemplateTypes(capabilitiesOf({ powertrain: 'ICE' }))).toEqual(['DRIVES', 'FUEL', 'ODOMETER'])
    expect(csvTemplateTypes(capabilitiesOf({ powertrain: 'PHEV' }))).toEqual(['CHARGES', 'DRIVES', 'FUEL', 'ODOMETER'])
  })
  it('leaves the kWh column out of the drives template of a vehicle that burns fuel', () => {
    expect(csvTemplate('DRIVES', 'en', 'ELECTRIC').headers).toContain('kwh')
    const fuel = csvTemplate('DRIVES', 'en', 'FUEL')
    expect(fuel.headers).toEqual(['start_time', 'end_time', 'distance_km', 'tag'])
    expect(fuel.rows[0]).toHaveLength(fuel.headers.length)
    expect(csvColumns('DRIVES', 'FUEL').optional).not.toContain('kwh')
    expect(csvColumns('DRIVES', 'ELECTRIC').optional).toContain('kwh')
  })
})

describe('csvTemplate', () => {
  it('has a header and one example row with as many cells', () => {
    for (const type of ['CHARGES', 'DRIVES', 'FUEL', 'ODOMETER'] as const) {
      const { headers, rows } = csvTemplate(type)
      expect(rows).toHaveLength(1)
      expect(rows[0]).toHaveLength(headers.length)
    }
  })
  it('has a French variant of the same shape', () => {
    for (const type of ['CHARGES', 'DRIVES', 'FUEL', 'ODOMETER'] as const) {
      const fr = csvTemplate(type, 'fr')
      expect(fr.rows[0]).toHaveLength(fr.headers.length)
      expect(fr.headers).toHaveLength(csvTemplate(type).headers.length)
    }
  })
  it('names the file after its type', () => {
    expect(csvTemplateFilename('CHARGES')).toBe('charges-template.csv')
  })
})

describe('webhookSnippet', () => {
  it('targets the instance, authenticates with the token and sends valid JSON', () => {
    const snippet = webhookSnippet('https://ledger.example', 'tok_123', 'veh-1')
    expect(snippet).toContain('https://ledger.example/api/integrations/homeassistant/event')
    expect(snippet).toContain('Authorization: Bearer tok_123')
    const json = snippet.match(/-d '(.*)'/)![1]
    expect(JSON.parse(json)).toMatchObject({ vehicle_id: 'veh-1', event_type: 'odometer_update' })
  })
})
