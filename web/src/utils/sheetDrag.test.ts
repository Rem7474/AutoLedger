import { describe, expect, it } from 'vitest'
import { DISMISS_DISTANCE, readGesture, releaseVelocity, shouldDismiss, sheetOffset } from './sheetDrag'

describe('sheetOffset', () => {
  it('follows the finger down and stays put when it goes up', () => {
    expect(sheetOffset(80)).toBe(80)
    expect(sheetOffset(-40)).toBe(0)
    expect(sheetOffset(0)).toBe(0)
  })
})

describe('shouldDismiss', () => {
  it('closes after a long drag, slow or not', () => {
    expect(shouldDismiss(DISMISS_DISTANCE, 0, 700)).toBe(true)
    expect(shouldDismiss(300, 0, 700)).toBe(true)
  })

  it('closes a short sheet after a third of its height', () => {
    expect(shouldDismiss(60, 0, 180)).toBe(true)
    expect(shouldDismiss(59, 0, 180)).toBe(false)
  })

  it('closes after a fast flick of a few dozen pixels', () => {
    expect(shouldDismiss(40, 0.8, 700)).toBe(true)
    expect(shouldDismiss(40, 0.2, 700)).toBe(false)
  })

  it('does not close on a tiny move, even a fast one', () => {
    expect(shouldDismiss(10, 2, 700)).toBe(false)
    expect(shouldDismiss(0, 0, 700)).toBe(false)
  })

  it('does not close when the finger went back up', () => {
    expect(shouldDismiss(-50, -1, 700)).toBe(false)
  })
})

describe('readGesture', () => {
  it('waits until the finger moved a few pixels', () => {
    expect(readGesture(1, 2, 0)).toBe('wait')
  })

  it('drags on a pull down from the top of the content', () => {
    expect(readGesture(0, 12, 0)).toBe('drag')
  })

  it('leaves a pull down inside scrolled content to the scroll', () => {
    expect(readGesture(0, 12, 40)).toBe('scroll')
  })

  it('leaves a move up or a sideways move to the page', () => {
    expect(readGesture(0, -12, 0)).toBe('scroll')
    expect(readGesture(20, 8, 0)).toBe('scroll')
  })
})

describe('releaseVelocity', () => {
  it('is the distance over the time of the samples', () => {
    expect(releaseVelocity([{ y: 100, t: 0 }, { y: 130, t: 50 }, { y: 160, t: 100 }])).toBeCloseTo(0.6)
  })

  it('is zero without enough samples or time', () => {
    expect(releaseVelocity([])).toBe(0)
    expect(releaseVelocity([{ y: 1, t: 1 }])).toBe(0)
    expect(releaseVelocity([{ y: 1, t: 5 }, { y: 9, t: 5 }])).toBe(0)
  })
})
