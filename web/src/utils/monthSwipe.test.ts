import { describe, expect, it } from 'vitest'
import { MONTH_SWIPE_PX, monthSwipeDirection, startsAtScreenEdge } from './monthSwipe'

describe('monthSwipeDirection', () => {
  it('goes to the month before on a swipe to the right and to the month after on a swipe to the left', () => {
    expect(monthSwipeDirection(MONTH_SWIPE_PX, 0)).toBe('previous')
    expect(monthSwipeDirection(-120, 20)).toBe('next')
  })
  it('ignores a move that is too short', () => {
    expect(monthSwipeDirection(MONTH_SWIPE_PX - 1, 0)).toBeNull()
  })
  it('ignores a mostly vertical move: the sheet scrolls or closes', () => {
    expect(monthSwipeDirection(80, 80)).toBeNull()
    expect(monthSwipeDirection(-90, 70)).toBeNull()
    expect(monthSwipeDirection(90, 60)).toBe('previous')
  })
})

describe('startsAtScreenEdge', () => {
  it('leaves the edges to the system back gesture', () => {
    expect(startsAtScreenEdge(10, 390)).toBe(true)
    expect(startsAtScreenEdge(380, 390)).toBe(true)
    expect(startsAtScreenEdge(200, 390)).toBe(false)
  })
})
