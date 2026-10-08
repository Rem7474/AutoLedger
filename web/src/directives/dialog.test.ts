import { describe, expect, it } from 'vitest'
import { shouldKeepOpen, wrapFocusIndex } from './dialog'

describe('wrapFocusIndex', () => {
  it('lets the browser move the focus inside the panel', () => {
    expect(wrapFocusIndex(3, 1, false)).toBe(-1)
    expect(wrapFocusIndex(3, 1, true)).toBe(-1)
  })
  it('wraps from the last control to the first and back', () => {
    expect(wrapFocusIndex(3, 2, false)).toBe(0)
    expect(wrapFocusIndex(3, 0, true)).toBe(2)
  })
  it('enters the panel when the focus is outside it', () => {
    expect(wrapFocusIndex(3, -1, false)).toBe(0)
    expect(wrapFocusIndex(3, -1, true)).toBe(2)
  })
  it('does nothing without a focusable control', () => {
    expect(wrapFocusIndex(0, -1, false)).toBe(-1)
  })
})

describe('shouldKeepOpen', () => {
  const backdrop = new EventTarget()
  const panelChild = new EventTarget()
  it('keeps a modal with edits open when the backdrop itself is clicked', () => {
    expect(shouldKeepOpen(true, backdrop, backdrop)).toBe(true)
  })
  it('lets a pristine modal close on a backdrop click', () => {
    expect(shouldKeepOpen(false, backdrop, backdrop)).toBe(false)
  })
  it('ignores clicks that do not land on the backdrop', () => {
    expect(shouldKeepOpen(true, panelChild, backdrop)).toBe(false)
    expect(shouldKeepOpen(true, null, null)).toBe(false)
  })
})
