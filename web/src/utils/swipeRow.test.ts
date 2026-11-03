import { describe, expect, it } from 'vitest'
import { createSwipeGesture, SWIPE_COMMIT_PX, SWIPE_LOCK_PX, SWIPE_MAX_PX } from './swipeRow'

function swipe(...points: [number, number][]) {
  const g = createSwipeGesture()
  g.start(300, 100)
  const moves = points.map(([dx, dy]) => g.move(300 + dx, 100 + dy))
  return { g, moves }
}

describe('swipe gesture', () => {
  it('does not decide before the finger has moved a little', () => {
    const { moves, g } = swipe([-(SWIPE_LOCK_PX - 2), 0])
    expect(moves).toEqual([null])
    expect(g.dragging).toBe(false)
  })

  it('drags the row with the finger once the move is mostly horizontal and to the left', () => {
    const { moves, g } = swipe([-20, 3], [-50, 5])
    expect(moves[0]).toEqual({ offset: -20, armed: false })
    expect(moves[1]).toEqual({ offset: -50, armed: false })
    expect(g.dragging).toBe(true)
  })

  it('leaves a mostly vertical move to the scroll, even if it turns horizontal later', () => {
    const { moves, g } = swipe([-4, 30], [-80, 30])
    expect(moves).toEqual([null, null])
    expect(g.dragging).toBe(false)
  })

  it('does not drag a move to the right', () => {
    const { moves } = swipe([40, 0], [80, 0])
    expect(moves).toEqual([null, null])
  })

  it('needs the horizontal move to clearly beat the vertical one', () => {
    expect(swipe([-30, 25]).moves).toEqual([null])
    expect(swipe([-30, 10]).moves[0]).toEqual({ offset: -30, armed: false })
  })

  it('arms the delete at the commit distance and never goes past the maximum', () => {
    const { moves } = swipe([-(SWIPE_COMMIT_PX - 1), 0], [-SWIPE_COMMIT_PX, 0], [-400, 0])
    expect(moves[0]?.armed).toBe(false)
    expect(moves[1]?.armed).toBe(true)
    expect(moves[2]).toEqual({ offset: -SWIPE_MAX_PX, armed: true })
  })

  it('can be taken back before letting go', () => {
    const { g } = swipe([-120, 0], [-30, 0])
    expect(g.end(false)).toBe('none')
  })

  it('deletes when let go past the commit distance', () => {
    const { g } = swipe([-60, 0], [-110, 0])
    expect(g.end(false)).toBe('delete')
  })

  it('does not delete a touch the system cancelled, nor one let go too early', () => {
    expect(swipe([-120, 0]).g.end(true)).toBe('none')
    expect(swipe([-40, 0]).g.end(false)).toBe('none')
  })

  it('ignores moves and ends without a start, and forgets a finished swipe', () => {
    const g = createSwipeGesture()
    expect(g.move(10, 10)).toBeNull()
    expect(g.end(false)).toBe('none')
    g.start(300, 100)
    g.move(150, 100)
    g.end(false)
    expect(g.dragging).toBe(false)
    expect(g.move(100, 100)).toBeNull()
  })
})
