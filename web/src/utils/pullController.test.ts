import { describe, expect, it, vi } from 'vitest'
import { createPullController, SPINNER_MS, type PullHost, type PullState } from './pullController'
import { PULL_SLOP, PULL_THRESHOLD } from './pullRefresh'

const FAR = PULL_SLOP + Math.ceil(PULL_THRESHOLD / 0.6) + 1

function setup(over: Partial<PullHost> = {}) {
  const states: PullState[] = []
  const refresh = vi.fn(async () => {})
  const waits: number[] = []
  const host: PullHost = {
    isPhone: () => true,
    handler: () => refresh,
    isBlocked: () => false,
    scrollTop: () => 0,
    wait: async (ms) => { waits.push(ms) },
    onChange: (s) => states.push(s),
    ...over,
  }
  return { c: createPullController(host), states, refresh, waits, last: () => states[states.length - 1] }
}

describe('createPullController', () => {
  it('follows a pull and reports the offset and whether it is armed', () => {
    const t = setup()
    t.c.start(1, 100, 100, null)
    expect(t.c.move(100, 100 + PULL_SLOP + 30)).toBe(true)
    expect(t.last()).toMatchObject({ pulling: true, armed: false })
    expect(t.c.move(100, 100 + FAR)).toBe(true)
    expect(t.last()).toMatchObject({ pulling: true, armed: true })
  })

  it('refreshes the page on screen when let go past the threshold, and shows the spinner for a moment', async () => {
    const t = setup()
    t.c.start(1, 100, 100, null)
    t.c.move(100, 100 + FAR)
    await t.c.end(false)
    expect(t.refresh).toHaveBeenCalledTimes(1)
    expect(t.waits).toEqual([SPINNER_MS])
    expect(t.states.some((s) => s.refreshing)).toBe(true)
    expect(t.last()).toEqual({ offset: 0, armed: false, pulling: false, refreshing: false })
  })

  it('goes back without refreshing when let go early or when the touch is cancelled', async () => {
    const early = setup()
    early.c.start(1, 100, 100, null)
    early.c.move(100, 130)
    await early.c.end(false)
    expect(early.refresh).not.toHaveBeenCalled()
    expect(early.last()).toEqual({ offset: 0, armed: false, pulling: false, refreshing: false })

    const cancelled = setup()
    cancelled.c.start(1, 100, 100, null)
    cancelled.c.move(100, 100 + FAR)
    await cancelled.c.end(true)
    expect(cancelled.refresh).not.toHaveBeenCalled()
  })

  it('says nothing for a touch that never pulled', async () => {
    const t = setup()
    t.c.start(1, 100, 100, null)
    await t.c.end(false)
    expect(t.states).toEqual([])
  })

  it('leaves the touch alone with two fingers, on a wide window, without a handler, or on a blocked target', () => {
    const blockedBy: Partial<PullHost>[] = [
      { isPhone: () => false },
      { handler: () => null },
      { isBlocked: () => true },
    ]
    for (const over of blockedBy) {
      const t = setup(over)
      t.c.start(1, 100, 100, null)
      expect(t.c.move(100, 100 + FAR)).toBe(false)
    }
    const two = setup()
    two.c.start(2, 100, 100, null)
    expect(two.c.move(100, 100 + FAR)).toBe(false)
  })

  it('does not pull a page that is not at the top', () => {
    const t = setup({ scrollTop: () => 120 })
    t.c.start(1, 100, 100, null)
    expect(t.c.move(100, 100 + FAR)).toBe(false)
  })

  it('does not start a second refresh while one is running', async () => {
    let release: () => void = () => {}
    const slow = vi.fn(() => new Promise<void>((resolve) => { release = resolve }))
    const t = setup({ handler: () => slow })
    t.c.start(1, 100, 100, null)
    t.c.move(100, 100 + FAR)
    const done = t.c.end(false)
    t.c.start(1, 100, 100, null)
    expect(t.c.move(100, 100 + FAR)).toBe(false)
    release()
    await done
    expect(slow).toHaveBeenCalledTimes(1)
  })

  it('recovers when the refresh fails: the page shows its own error', async () => {
    const t = setup({ handler: () => async () => { throw new Error('offline') } })
    t.c.start(1, 100, 100, null)
    t.c.move(100, 100 + FAR)
    await t.c.end(false)
    expect(t.last()).toEqual({ offset: 0, armed: false, pulling: false, refreshing: false })
  })
})
