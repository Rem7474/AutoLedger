import { describe, expect, it } from 'vitest'
import { activeDestination, activeSidebarEntry, GARAGE_DESTINATION, NAV_DESTINATIONS, navDestinations } from './navigation'

describe('navDestinations', () => {
  it('has four destinations, the second one the trips of a vehicle that has some and the consumption of one that has none', () => {
    expect(navDestinations(true).map((d) => d.name)).toEqual(['dashboard', 'drives', 'costs', 'vehicle'])
    expect(navDestinations(false).map((d) => d.name)).toEqual(['dashboard', 'consumption', 'costs', 'vehicle'])
  })

  it('keeps every destination and the garage on a distinct path', () => {
    const paths = [...NAV_DESTINATIONS, GARAGE_DESTINATION].map((d) => d.path)
    expect(new Set(paths).size).toBe(paths.length)
  })
})

describe('activeDestination', () => {
  const withDrives = navDestinations(true)

  it('lights the destination a page belongs to', () => {
    expect(activeDestination(withDrives, 'dashboard')).toBe('dashboard')
    expect(activeDestination(withDrives, 'drives')).toBe('drives')
    expect(activeDestination(withDrives, 'expenses')).toBe('costs')
  })

  it('lights the vehicle for every page reached from its sheet, the maintenance included', () => {
    for (const page of ['odometer', 'tires', 'comparison', 'maintenance', 'vehicles', 'consumption']) {
      expect(activeDestination(withDrives, page), page).toBe('vehicle')
    }
  })

  it('gives the consumption page to its own destination when it is one, and to the vehicle otherwise', () => {
    expect(activeDestination(navDestinations(false), 'consumption')).toBe('consumption')
    expect(activeDestination(withDrives, 'consumption')).toBe('vehicle')
  })

  it('lights nothing for a page outside the four (the account page)', () => {
    expect(activeDestination(withDrives, 'account')).toBeUndefined()
    expect(activeDestination(withDrives, undefined)).toBeUndefined()
  })
})

describe('activeSidebarEntry', () => {
  it('lights the garage alone on its page, and the destination elsewhere', () => {
    const d = navDestinations(true)
    expect(activeSidebarEntry(d, 'vehicles')).toBe('vehicles')
    expect(activeSidebarEntry(d, 'tires')).toBe('vehicle')
    expect(activeSidebarEntry(d, 'account')).toBeUndefined()
  })
})
