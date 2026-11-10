import { describe, expect, it } from 'vitest'
import { costsTabQuery, parseCostsTab } from './costsTab'

describe('costs tab', () => {
  it('reads the analysis tab from the address and falls back to the journal', () => {
    expect(parseCostsTab('ANALYSIS')).toBe('ANALYSIS')
    expect(parseCostsTab(['ANALYSIS', 'x'])).toBe('ANALYSIS')
    expect(parseCostsTab(undefined)).toBe('JOURNAL')
    expect(parseCostsTab('nope')).toBe('JOURNAL')
  })

  it('writes the analysis tab and drops it for the journal, keeping the theme and the period', () => {
    const query = { theme: 'energy', period: '3M', tab: 'ANALYSIS' }
    expect(costsTabQuery({ theme: 'energy', period: '3M' }, 'ANALYSIS')).toEqual(query)
    expect(costsTabQuery(query, 'JOURNAL')).toEqual({ theme: 'energy', period: '3M' })
  })
})
