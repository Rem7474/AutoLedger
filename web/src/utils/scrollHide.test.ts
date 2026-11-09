import { describe, expect, it } from 'vitest'
import { initialScrollHide, stepScrollHide, type ScrollHideState } from '@/utils/scrollHide'

const run = (offsets: number[], from: ScrollHideState = initialScrollHide) => offsets.reduce(stepScrollHide, from)

describe('stepScrollHide', () => {
  it('stays shown near the top', () => {
    expect(run([0, 10, 24]).hidden).toBe(false)
  })
  it('hides when read downwards past the delta', () => {
    expect(run([100, 130]).hidden).toBe(true)
  })
  it('ignores a small movement', () => {
    expect(run([105, 108], { hidden: false, anchor: 100 }).hidden).toBe(false)
  })
  it('accumulates slow movement from the anchor', () => {
    expect(run([108, 116, 124], { hidden: false, anchor: 100 }).hidden).toBe(true)
  })
  it('comes back when read upwards', () => {
    expect(run([100, 200, 180]).hidden).toBe(false)
  })
  it('stays hidden on a wobble of a few pixels', () => {
    expect(run([100, 200, 195, 200]).hidden).toBe(true)
  })
  it('is shown again at the top whatever came before', () => {
    expect(run([100, 300, 10]).hidden).toBe(false)
  })
  it('returns the same state object when nothing changes', () => {
    const s = run([100, 200])
    expect(stepScrollHide(s, 203)).toBe(s)
  })
})
