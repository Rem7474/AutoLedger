import { describe, expect, it, vi } from 'vitest'
import { browserEnv, browserStart, HOLD_MS, installViewTransitions, shouldAnimate, type TransitionEnv } from './viewTransition'

const phone: TransitionEnv = { supported: true, phone: true, reducedMotion: false }

describe('shouldAnimate', () => {
  it('animates a change of page on a phone', () => {
    expect(shouldAnimate({ name: 'drives' }, { name: 'dashboard' }, phone)).toBe(true)
  })

  it('leaves a change inside the same page alone (a filter, a tab)', () => {
    expect(shouldAnimate({ name: 'drives' }, { name: 'drives' }, phone)).toBe(false)
  })

  it('does not animate the first navigation or a nameless route', () => {
    expect(shouldAnimate({ name: 'drives' }, { name: undefined }, phone)).toBe(false)
    expect(shouldAnimate({ name: undefined }, { name: 'drives' }, phone)).toBe(false)
  })

  it('does not animate without the API, on a wide window or when less motion is asked', () => {
    expect(shouldAnimate({ name: 'a' }, { name: 'b' }, { ...phone, supported: false })).toBe(false)
    expect(shouldAnimate({ name: 'a' }, { name: 'b' }, { ...phone, phone: false })).toBe(false)
    expect(shouldAnimate({ name: 'a' }, { name: 'b' }, { ...phone, reducedMotion: true })).toBe(false)
  })
})

describe('installViewTransitions', () => {
  function setup(env: TransitionEnv = phone) {
    let resolveHook: ((to: unknown, from: unknown) => Promise<void> | void) | undefined
    let after: (() => void) | undefined
    let onError: (() => void) | undefined
    const router = {
      beforeResolve: (fn: typeof resolveHook) => { resolveHook = fn; return () => {} },
      afterEach: (fn: () => void) => { after = fn; return () => {} },
      onError: (fn: () => void) => { onError = fn; return () => {} },
    }
    let updated: Promise<void> | undefined
    const start = vi.fn((update: () => Promise<void>) => { updated = update() })
    installViewTransitions(router as never, () => env, () => start)
    return { navigate: (to: string, from: string) => resolveHook!({ name: to }, { name: from }), after: () => after!(), fail: () => onError!(), start, updated: () => updated }
  }

  it('lets a navigation through without a transition when it should not animate', async () => {
    const t = setup({ ...phone, reducedMotion: true })
    expect(t.navigate('drives', 'dashboard')).toBeUndefined()
    expect(t.start).not.toHaveBeenCalled()
  })

  it('holds the old page until the navigation has finished, then releases the transition', async () => {
    const t = setup()
    await t.navigate('drives', 'dashboard')
    expect(t.start).toHaveBeenCalledTimes(1)
    let released = false
    void t.updated()!.then(() => { released = true })
    await Promise.resolve()
    expect(released).toBe(false)
    t.after()
    await new Promise((r) => setTimeout(r, 0))
    expect(released).toBe(true)
  })

  it('releases the transition when the navigation fails', async () => {
    const t = setup()
    await t.navigate('drives', 'dashboard')
    let released = false
    void t.updated()!.then(() => { released = true })
    t.fail()
    await new Promise((r) => setTimeout(r, 0))
    expect(released).toBe(true)
  })

  it('does not hold the old page for ever when the navigation never ends', async () => {
    vi.useFakeTimers()
    try {
      const t = setup()
      await t.navigate('drives', 'dashboard')
      let released = false
      void t.updated()!.then(() => { released = true })
      await vi.advanceTimersByTimeAsync(HOLD_MS + 10)
      expect(released).toBe(true)
    } finally {
      vi.useRealTimers()
    }
  })
})

describe('the browser', () => {
  const win = (matches: Record<string, boolean>) => ({ matchMedia: (query: string) => ({ matches: Object.entries(matches).some(([part, on]) => on && query.includes(part)) }) }) as unknown as Window

  it('reads the phone width, the motion preference and the API from the window and document', () => {
    const withApi = { startViewTransition: () => {} } as unknown as Document
    expect(browserEnv(win({ 'max-width': true }), withApi)).toEqual({ supported: true, phone: true, reducedMotion: false })
    expect(browserEnv(win({ 'prefers-reduced-motion': true }), {} as Document)).toEqual({ supported: false, phone: false, reducedMotion: true })
  })

  it('finds startViewTransition on the document, bound to it, or nothing', () => {
    const calls: unknown[] = []
    const doc = { startViewTransition(this: unknown) { calls.push(this) } } as unknown as Document
    browserStart(doc)!(async () => {})
    expect(calls).toEqual([doc])
    expect(browserStart({} as Document)).toBeNull()
    expect(browserStart(undefined)).toBeNull()
  })
})
