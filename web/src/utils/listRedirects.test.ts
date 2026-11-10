import { describe, expect, it } from 'vitest'
import { energyRedirect, listRedirect } from './listRedirects'

describe('listRedirect', () => {
  it('sends the tolls list, also the default of /expenses, to the tolls theme', () => {
    expect(listRedirect('expenses', 'TOLLS')).toEqual({ path: '/costs', query: { theme: 'tolls' } })
    expect(listRedirect('expenses', undefined)).toEqual({ path: '/costs', query: { theme: 'tolls' } })
    expect(listRedirect('expenses', 'tolls')).toEqual({ path: '/costs', query: { theme: 'tolls' } })
  })

  it('sends the fixed costs to the whole cost, which has no single theme', () => {
    expect(listRedirect('expenses', 'FIXED')).toEqual({ path: '/costs', query: {} })
  })

  it('sends the maintenance list, also the default of /maintenance, to the maintenance theme', () => {
    expect(listRedirect('maintenance', '')).toEqual({ path: '/costs', query: { theme: 'maintenance' } })
    expect(listRedirect('maintenance', ['MAINTENANCE'])).toEqual({ path: '/costs', query: { theme: 'maintenance' } })
  })

  it('keeps the tool tabs on their page', () => {
    expect(listRedirect('expenses', 'DOCUMENTS')).toBeNull()
    expect(listRedirect('maintenance', 'REMINDERS')).toBeNull()
    expect(listRedirect('consumption', 'CHARGES')).toBeNull()
    expect(listRedirect('consumption', undefined)).toBeNull()
    expect(listRedirect('consumption', 'FUEL')).toBeNull()
  })
})

describe('energyRedirect', () => {
  it('sends a tool tab to the consumption page with its query', () => {
    expect(energyRedirect({ tab: 'FUEL' })).toEqual({ path: '/consumption', query: { tab: 'FUEL' } })
    expect(energyRedirect({ tab: ['estimate'], add: '1' })).toEqual({ path: '/consumption', query: { tab: ['estimate'], add: '1' } })
  })

  it('sends the bare address and an unknown tab to the energy theme of the costs', () => {
    expect(energyRedirect({})).toEqual({ path: '/costs', query: { theme: 'energy' } })
    expect(energyRedirect({ tab: 'nope', add: '1' })).toEqual({ path: '/costs', query: { add: '1', theme: 'energy' } })
  })
})
