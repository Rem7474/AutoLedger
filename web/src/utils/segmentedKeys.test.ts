import { describe, expect, it } from 'vitest'
import { nextSegment } from './segmentedKeys'

describe('nextSegment', () => {
  it('avance et recule avec les flèches, en bouclant', () => {
    expect(nextSegment('ArrowRight', 0, 3)).toBe(1)
    expect(nextSegment('ArrowDown', 2, 3)).toBe(0)
    expect(nextSegment('ArrowLeft', 0, 3)).toBe(2)
    expect(nextSegment('ArrowUp', 2, 3)).toBe(1)
  })

  it('saute aux extrémités avec Home et End', () => {
    expect(nextSegment('Home', 2, 3)).toBe(0)
    expect(nextSegment('End', 0, 3)).toBe(2)
  })

  it('ignore les autres touches et une liste vide', () => {
    expect(nextSegment('a', 0, 3)).toBe(-1)
    expect(nextSegment('Enter', 1, 3)).toBe(-1)
    expect(nextSegment('ArrowRight', 0, 0)).toBe(-1)
  })
})
