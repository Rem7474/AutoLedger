import { beforeEach, describe, expect, it } from 'vitest'
import {
  currentDistanceUnit,
  displayDistanceToKm,
  DEFAULT_DISTANCE_UNIT,
  distanceUnit,
  formatDistance,
  formatDistanceValue,
  formatSpeed,
  kmToDisplayDistance,
  perDistance,
  perDistanceToPerKm,
  setDistanceUnit,
  speedUnit,
  currentVolumeUnit,
  DEFAULT_VOLUME_UNIT,
  displayConsumptionToL100km,
  displayVolumeToLitres,
  formatFuelConsumptionValue,
  formatVolume,
  fuelConsumptionUnit,
  l100kmToDisplayConsumption,
  litresToDisplayVolume,
  perVolume,
  perVolumeToPerLitre,
  setVolumeUnit,
  volumeUnitLabel,
} from './units'

describe('setDistanceUnit', () => {
  beforeEach(() => setDistanceUnit(null))

  it('applies a supported value', () => {
    setDistanceUnit('mi')
    expect(currentDistanceUnit()).toBe('mi')
  })

  it('falls back to the default for a missing or unsupported value', () => {
    setDistanceUnit('mi')
    setDistanceUnit(undefined)
    expect(currentDistanceUnit()).toBe(DEFAULT_DISTANCE_UNIT)

    setDistanceUnit('mi')
    setDistanceUnit('furlong')
    expect(currentDistanceUnit()).toBe(DEFAULT_DISTANCE_UNIT)
  })
})

describe('distance conversion', () => {
  it('is a no-op in km', () => {
    setDistanceUnit('km')
    expect(kmToDisplayDistance(100)).toBe(100)
    expect(displayDistanceToKm(100)).toBe(100)
  })

  it('converts both ways in miles', () => {
    setDistanceUnit('mi')
    expect(kmToDisplayDistance(160.9344)).toBeCloseTo(100, 5)
    expect(displayDistanceToKm(100)).toBeCloseTo(160.9344, 5)
  })

  it('round-trips without drift', () => {
    setDistanceUnit('mi')
    expect(displayDistanceToKm(kmToDisplayDistance(42000))).toBeCloseTo(42000, 6)
  })
})

describe('formatDistance', () => {
  it('rounds and appends the unit in km', () => {
    setDistanceUnit('km')
    expect(formatDistance(42000.4)).toBe(`${Math.round(42000.4).toLocaleString('fr-FR')} km`)
  })

  it('converts, rounds and appends the unit in miles', () => {
    setDistanceUnit('mi')
    expect(formatDistance(160.9344)).toBe(`${(100).toLocaleString('fr-FR')} mi`)
  })
})

describe('per-distance figures', () => {
  beforeEach(() => setDistanceUnit(null))

  it('keep their value in km', () => {
    expect(perDistance(0.2)).toBe(0.2)
    expect(perDistanceToPerKm(0.2)).toBe(0.2)
    expect(distanceUnit()).toBe('km')
    expect(speedUnit()).toBe('km/h')
  })

  it('grow by the mile ratio in miles and convert back', () => {
    setDistanceUnit('mi')
    expect(perDistance(0.1)).toBeCloseTo(0.1609344)
    expect(perDistanceToPerKm(perDistance(0.37))).toBeCloseTo(0.37)
    expect(distanceUnit()).toBe('mi')
    expect(formatSpeed(100)).toBe('62 mph')
  })

  it('format a distance with decimals when asked', () => {
    setDistanceUnit('mi')
    expect(formatDistanceValue(16.09344, 1)).toBe((10).toLocaleString('fr-FR', { minimumFractionDigits: 1 }))
    expect(formatDistance(1.609344, 1)).toBe(`${(1).toLocaleString('fr-FR', { minimumFractionDigits: 1 })} mi`)
  })

  it('formats consumption range bounds in the current unit', () => {
    setDistanceUnit('km')
    expect(perDistance(1).toLocaleString('fr-FR', { maximumFractionDigits: 1 })).toBe('1')
    expect(perDistance(100).toLocaleString('fr-FR', { maximumFractionDigits: 1 })).toBe('100')

    setDistanceUnit('mi')
    expect(perDistance(1).toLocaleString('fr-FR', { maximumFractionDigits: 1 })).toBe('1,6')
    expect(perDistance(100).toLocaleString('fr-FR', { maximumFractionDigits: 1 })).toBe('160,9')
  })
})

describe('volume unit', () => {
  beforeEach(() => {
    setDistanceUnit('km')
    setVolumeUnit('l')
  })

  it('falls back to litres for a missing or unsupported value', () => {
    setVolumeUnit('gal_us')
    expect(currentVolumeUnit()).toBe('gal_us')
    setVolumeUnit('pint')
    expect(currentVolumeUnit()).toBe(DEFAULT_VOLUME_UNIT)
    setVolumeUnit(undefined)
    expect(currentVolumeUnit()).toBe(DEFAULT_VOLUME_UNIT)
  })

  it('is a no-op in litres', () => {
    expect(litresToDisplayVolume(40)).toBe(40)
    expect(displayVolumeToLitres(40)).toBe(40)
    expect(perVolume(1.8)).toBe(1.8)
    expect(volumeUnitLabel()).toBe('L')
  })

  it('converts quantities and prices both ways in gallons', () => {
    setVolumeUnit('gal_us')
    expect(litresToDisplayVolume(3.785411784)).toBeCloseTo(1, 6)
    expect(displayVolumeToLitres(10)).toBeCloseTo(37.85411784, 6)
    expect(perVolume(1)).toBeCloseTo(3.785411784, 6)
    expect(perVolumeToPerLitre(perVolume(1.789))).toBeCloseTo(1.789, 9)
    setVolumeUnit('gal_uk')
    expect(displayVolumeToLitres(1)).toBeCloseTo(4.54609, 6)
    expect(formatVolume(4.54609, 0)).toBe('1 gal UK')
  })
})

describe('fuel consumption', () => {
  beforeEach(() => {
    setDistanceUnit('km')
    setVolumeUnit('l')
  })

  it('stays L/100 km with the default units', () => {
    expect(fuelConsumptionUnit()).toBe('L/100 km')
    expect(l100kmToDisplayConsumption(6.5)).toBe(6.5)
    expect(formatFuelConsumptionValue(6.5, 1)).toBe((6.5).toLocaleString('fr-FR', { minimumFractionDigits: 1 }))
  })

  it('follows the distance unit alone: litres per 100 miles', () => {
    setDistanceUnit('mi')
    expect(fuelConsumptionUnit()).toBe('L/100 mi')
    expect(l100kmToDisplayConsumption(10)).toBeCloseTo(16.09344, 5)
  })

  it('follows the volume unit alone: gallons per 100 km', () => {
    setVolumeUnit('gal_us')
    expect(fuelConsumptionUnit()).toBe('gal US/100 km')
    expect(l100kmToDisplayConsumption(3.785411784)).toBeCloseTo(1, 6)
  })

  it('is miles per gallon with miles and gallons, US or UK', () => {
    setDistanceUnit('mi')
    setVolumeUnit('gal_us')
    expect(fuelConsumptionUnit()).toBe('mpg US')
    expect(l100kmToDisplayConsumption(8)).toBeCloseTo(29.4, 1)
    setVolumeUnit('gal_uk')
    expect(fuelConsumptionUnit()).toBe('mpg UK')
    expect(l100kmToDisplayConsumption(8)).toBeCloseTo(35.3, 1)
  })

  it('round-trips a typed consumption back to L/100 km in every unit pair', () => {
    for (const d of ['km', 'mi']) {
      for (const v of ['l', 'gal_us', 'gal_uk']) {
        setDistanceUnit(d)
        setVolumeUnit(v)
        expect(displayConsumptionToL100km(l100kmToDisplayConsumption(7.3))).toBeCloseTo(7.3, 6)
      }
    }
  })

  it('does not divide by zero', () => {
    setDistanceUnit('mi')
    setVolumeUnit('gal_us')
    expect(l100kmToDisplayConsumption(0)).toBe(0)
    expect(displayConsumptionToL100km(0)).toBe(0)
  })
})
