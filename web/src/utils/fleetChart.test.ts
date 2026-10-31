import { describe, expect, it } from 'vitest'
import { scaleCut } from './fleetChart'

describe('scaleCut', () => {
  it('keeps the natural scale when no month stands out', () => {
    expect(scaleCut([100, 120, 90, 150, 110])).toBeNull()
    expect(scaleCut([100, 249])).toBeNull()
  })

  it('cuts the axis a little above the second highest month and names the months above it', () => {
    expect(scaleCut([100, 5000, 120, 90])).toEqual({ ceiling: 150, cutIndexes: [1] })
  })

  it('does not cut a month that is only twice the next one, and ignores empty months', () => {
    expect(scaleCut([0, 0, 100, 200, 90])).toBeNull()
    expect(scaleCut([0, 0, 100, 300, 90])).toEqual({ ceiling: 125, cutIndexes: [3] })
  })

  it('needs at least three months with an amount', () => {
    expect(scaleCut([10, 5000])).toBeNull()
    expect(scaleCut([0, 10, 5000])).toBeNull()
  })
})
