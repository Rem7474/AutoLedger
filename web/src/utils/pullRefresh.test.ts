import { describe, expect, it } from 'vitest'
import { createPullGesture, PULL_MAX, PULL_SLOP, PULL_THRESHOLD } from './pullRefresh'

// Distance down that makes the offset reach the threshold (resistance 0.6)
const FAR = PULL_SLOP + Math.ceil(PULL_THRESHOLD / 0.6) + 1

describe('createPullGesture', () => {
  it('does nothing until the finger has gone past the slop', () => {
    const g = createPullGesture()
    g.start(100, 100, 0)
    expect(g.move(100, 100 + PULL_SLOP)).toBeNull()
    expect(g.end()).toBeNull()
  })

  it('follows the finger with resistance and marks the first move', () => {
    const g = createPullGesture()
    g.start(100, 100, 0)
    const first = g.move(100, 100 + PULL_SLOP + 20)!
    expect(first.began).toBe(true)
    expect(first.offset).toBeCloseTo(12)
    expect(first.armed).toBe(false)
    const next = g.move(100, 100 + PULL_SLOP + 40)!
    expect(next.began).toBe(false)
    expect(next.offset).toBeCloseTo(24)
  })

  it('arms past the threshold and refreshes when let go there', () => {
    const g = createPullGesture()
    g.start(100, 100, 0)
    const move = g.move(100, 100 + FAR)!
    expect(move.armed).toBe(true)
    expect(g.end()).toBe('refresh')
  })

  it('cancels when let go before the threshold, or when the system cancels the touch', () => {
    const short = createPullGesture()
    short.start(100, 100, 0)
    short.move(100, 130)
    expect(short.end()).toBe('cancel')

    const cancelled = createPullGesture()
    cancelled.start(100, 100, 0)
    cancelled.move(100, 100 + FAR)
    expect(cancelled.end(true)).toBe('cancel')
  })

  it('stops growing at the ceiling', () => {
    const g = createPullGesture()
    g.start(100, 0, 0)
    expect(g.move(100, 2000)!.offset).toBe(PULL_MAX)
  })

  it('lets the page scroll when it did not start at the top', () => {
    const g = createPullGesture()
    g.start(100, 100, 40)
    expect(g.move(100, 100 + FAR)).toBeNull()
    expect(g.end()).toBeNull()
  })

  it('leaves an upward or sideways move to the page, for the rest of the touch', () => {
    const up = createPullGesture()
    up.start(100, 100, 0)
    expect(up.move(100, 100 - PULL_SLOP - 5)).toBeNull()
    expect(up.move(100, 100 + FAR)).toBeNull()
    expect(up.end()).toBeNull()

    const side = createPullGesture()
    side.start(100, 100, 0)
    expect(side.move(100 + PULL_SLOP + 5, 100 + 2)).toBeNull()
    expect(side.move(100 + PULL_SLOP + 5, 100 + FAR)).toBeNull()
    expect(side.end()).toBeNull()
  })

  it('drops a touch that is ignored (a second finger, a field)', () => {
    const g = createPullGesture()
    g.start(100, 100, 0)
    g.ignore()
    expect(g.move(100, 100 + FAR)).toBeNull()
    expect(g.end()).toBeNull()
  })

  it('starts again from nothing at the next touch', () => {
    const g = createPullGesture()
    g.start(100, 100, 0)
    g.move(100, 100 + FAR)
    g.end()
    g.start(100, 500, 0)
    expect(g.move(100, 500 + PULL_SLOP + 20)!.began).toBe(true)
  })
})
