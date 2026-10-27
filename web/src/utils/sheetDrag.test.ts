import { describe, expect, it } from 'vitest'
import { createSheetGesture, DISMISS_DISTANCE, readGesture, releaseVelocity, shouldDismiss, sheetOffset } from './sheetDrag'

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

describe('createSheetGesture', () => {
  it('is undecided until the finger moved, then follows it down and closes after a long pull', () => {
    const g = createSheetGesture()
    g.start(100, 100, 0, 0)
    expect(g.move(100, 102, 10)).toBeNull()
    expect(g.move(100, 130, 20)).toEqual({ offset: 30, began: true })
    expect(g.move(100, 200, 30)).toEqual({ offset: 100, began: false })
    expect(g.end(260, 700, false)).toBe('dismiss')
  })

  it('settles back after a short, slow pull', () => {
    const g = createSheetGesture()
    g.start(100, 100, 0, 0)
    g.move(100, 120, 200)
    g.move(100, 140, 600)
    expect(g.end(140, 700, false)).toBe('settle')
  })

  it('closes after a fast flick, reading the speed of the last moments only', () => {
    const g = createSheetGesture()
    g.start(100, 100, 0, 0)
    g.move(100, 105, 500)
    g.move(100, 120, 900)
    g.move(100, 150, 940)
    g.move(100, 160, 960)
    expect(g.end(160, 700, false)).toBe('dismiss')
  })

  it('settles back instead of closing when the touch is cancelled', () => {
    const g = createSheetGesture()
    g.start(100, 100, 0, 0)
    g.move(100, 300, 20)
    expect(g.end(300, 700, true)).toBe('settle')
  })

  it('does not hold the sheet back from a pull that goes back up', () => {
    const g = createSheetGesture()
    g.start(100, 100, 0, 0)
    g.move(100, 150, 10)
    expect(g.move(100, 60, 20)).toEqual({ offset: 0, began: false })
  })

  it('leaves the touch to the scroll inside scrolled content, from the first move to the release', () => {
    const g = createSheetGesture()
    g.start(100, 100, 0, 80)
    expect(g.move(100, 130, 10)).toBeNull()
    expect(g.move(100, 300, 20)).toBeNull()
    expect(g.end(300, 700, false)).toBe('none')
  })

  it('leaves a sideways or upward move to the page', () => {
    const g = createSheetGesture()
    g.start(100, 100, 0, 0)
    expect(g.move(140, 110, 10)).toBeNull()
    expect(g.end(110, 700, false)).toBe('none')
    g.start(100, 100, 0, 0)
    expect(g.move(100, 80, 10)).toBeNull()
  })

  it('ignores a touch the sheet does not handle, and a release without a drag', () => {
    const g = createSheetGesture()
    g.start(100, 100, 0, 0)
    g.ignore()
    expect(g.move(100, 300, 10)).toBeNull()
    expect(g.end(300, 700, false)).toBe('none')
    expect(g.end(300, 700, false)).toBe('none')
  })
})
