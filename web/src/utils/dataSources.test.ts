import { describe, expect, it } from 'vitest'
import { capabilitiesOf } from './capabilities'
import { csvTemplate, csvTemplateFilename, csvTemplateTypes, webhookSnippet } from './dataSources'

describe('csvTemplateTypes', () => {
  it('offers charges and drives to an electric vehicle, fill-ups to a combustion one', () => {
    expect(csvTemplateTypes(capabilitiesOf({ powertrain: 'EV' }))).toEqual(['CHARGES', 'DRIVES', 'ODOMETER'])
    expect(csvTemplateTypes(capabilitiesOf({ powertrain: 'ICE' }))).toEqual(['FUEL', 'ODOMETER'])
    expect(csvTemplateTypes(capabilitiesOf({ powertrain: 'PHEV' }))).toEqual(['CHARGES', 'DRIVES', 'FUEL', 'ODOMETER'])
  })
  it('offers trips to a combustion vehicle a source feeds', () => {
    const ice = capabilitiesOf({ powertrain: 'ICE' })
    expect(csvTemplateTypes({ ...ice, drives: true })).toEqual(['DRIVES', 'FUEL', 'ODOMETER'])
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
