import { describe, expect, it } from 'vitest'
import { sharedTollSegments } from './tolls'

describe('sharedTollSegments', () => {
  it('keeps the tickets that span several drives and carry their whole price', () => {
    const shared = { entry: 'A', spans: 2, full_price: 12.4 }
    const alone = { entry: 'B' }
    const unpriced = { entry: 'C', spans: 3, full_price: null }
    expect(sharedTollSegments([shared, alone, unpriced, { entry: 'D', spans: 1, full_price: 5 }])).toEqual([shared])
  })

  it('is empty without a detection', () => {
    expect(sharedTollSegments(undefined)).toEqual([])
    expect(sharedTollSegments(null)).toEqual([])
  })
})
