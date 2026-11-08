import { describe, expect, it } from 'vitest'
import { drivesTabQuery, parseDrivesTab } from './drivesTab'

describe('parseDrivesTab', () => {
  it('reads the trips tab, and the list of drives for anything else', () => {
    expect(parseDrivesTab('TRIPS')).toBe('TRIPS')
    expect(parseDrivesTab(['TRIPS', 'DRIVES'])).toBe('TRIPS')
    expect(parseDrivesTab('DRIVES')).toBe('DRIVES')
    expect(parseDrivesTab('other')).toBe('DRIVES')
    expect(parseDrivesTab(undefined)).toBe('DRIVES')
  })
})

describe('drivesTabQuery', () => {
  it('sets the trips tab and keeps the other parameters', () => {
    expect(drivesTabQuery({ open: 'x' }, 'TRIPS')).toEqual({ open: 'x', tab: 'TRIPS' })
  })

  it('drops the tab for the list of drives', () => {
    expect(drivesTabQuery({ tab: 'TRIPS', open: 'x' }, 'DRIVES')).toEqual({ open: 'x' })
  })
})
