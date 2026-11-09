import { describe, expect, it } from 'vitest'
import { listRedirect } from './listRedirects'

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
    expect(listRedirect('energy', 'CHARGES')).toBeNull()
    expect(listRedirect('energy', undefined)).toBeNull()
    expect(listRedirect('energy', 'FUEL')).toBeNull()
  })
})
