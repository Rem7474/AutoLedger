import { describe, expect, it } from 'vitest'
import { rotatedTires } from './tireRotation'

const entry = (id: string, current_position: string) => ({ id, tire: { current_position } })
const list = [entry('a', 'FL'), entry('b', 'FR'), entry('c', 'RL'), entry('d', 'RR'), entry('e', 'STORAGE'), entry('f', 'DISPOSED')]
const positions = (l: ReturnType<typeof rotatedTires<ReturnType<typeof entry>>>) => Object.fromEntries(l.map((e) => [e.id, e.tire.current_position]))

describe('rotatedTires', () => {
  it('swaps front and rear on each side', () => {
    expect(positions(rotatedTires(list, 'FRONT_BACK'))).toEqual({ a: 'RL', b: 'RR', c: 'FL', d: 'FR', e: 'STORAGE', f: 'DISPOSED' })
  })

  it('crosses the diagonals', () => {
    expect(positions(rotatedTires(list, 'CROSS'))).toEqual({ a: 'RR', b: 'RL', c: 'FR', d: 'FL', e: 'STORAGE', f: 'DISPOSED' })
  })

  it('moves only the mounted tires present and leaves the input untouched', () => {
    const partial = [entry('a', 'FL'), entry('e', 'STORAGE')]
    expect(positions(rotatedTires(partial, 'FRONT_BACK'))).toEqual({ a: 'RL', e: 'STORAGE' })
    expect(partial[0].tire.current_position).toBe('FL')
  })

  it('returns to the starting layout when applied twice', () => {
    expect(positions(rotatedTires(rotatedTires(list, 'CROSS'), 'CROSS'))).toEqual(positions(list))
  })
})
