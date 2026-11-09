import { describe, expect, it, vi } from 'vitest'
import { createBackGuard, type BackEnv } from './useBackToClose'

function fakeEnv() {
  let fire: () => void = () => {}
  const env: BackEnv & { stack: number; traverse(): void } = {
    stack: 0,
    push() {
      this.stack++
    },
    back: vi.fn(),
    listen(fn) {
      fire = fn
      return () => {
        fire = () => {}
      }
    },
    traverse() {
      this.stack--
      fire()
    },
  }
  return env
}

describe('createBackGuard', () => {
  it('holds one entry while active and closes on Back', () => {
    const env = fakeEnv()
    const close = vi.fn(() => true)
    const g = createBackGuard(env, close)
    g.set(true)
    g.set(true)
    expect(env.stack).toBe(1)
    env.traverse()
    expect(close).toHaveBeenCalledTimes(1)
    // the mode ends by itself afterwards: nothing more to pop
    g.set(false)
    expect(env.back).not.toHaveBeenCalled()
  })

  it('pops its own entry when the mode ends another way, without closing again', () => {
    const env = fakeEnv()
    const close = vi.fn(() => true)
    const g = createBackGuard(env, close)
    g.set(true)
    g.set(false)
    expect(env.back).toHaveBeenCalledTimes(1)
    env.traverse()
    expect(close).not.toHaveBeenCalled()
    expect(env.stack).toBe(0)
  })

  it('holds a new entry when the mode comes back before the pop landed', () => {
    const env = fakeEnv()
    const close = vi.fn(() => true)
    const g = createBackGuard(env, close)
    g.set(true)
    g.set(false)
    g.set(true)
    expect(env.stack).toBe(1) // the new entry waits for the pop
    env.traverse()
    expect(env.stack).toBe(1)
    expect(close).not.toHaveBeenCalled()
    env.traverse()
    expect(close).toHaveBeenCalledTimes(1)
  })

  it('keeps its entry when the mode cannot close yet', () => {
    const env = fakeEnv()
    let busy = true
    const close = vi.fn(() => !busy)
    const g = createBackGuard(env, close)
    g.set(true)
    env.traverse()
    expect(env.stack).toBe(1)
    busy = false
    env.traverse()
    expect(close).toHaveBeenCalledTimes(2)
    expect(env.stack).toBe(0)
  })

  it('ignores traversals once disposed', () => {
    const env = fakeEnv()
    const close = vi.fn(() => true)
    const g = createBackGuard(env, close)
    g.set(true)
    g.dispose()
    env.traverse()
    expect(close).not.toHaveBeenCalled()
    expect(env.back).not.toHaveBeenCalled()
  })
})


// The tests run in Node: a window with just the history and popstate parts the composable touches
function fakeWindow() {
  const listeners = new Set<() => void>()
  const history = { state: { position: 3 } as Record<string, unknown> | null, pushState: vi.fn(), back: vi.fn() }
  vi.stubGlobal('window', {
    history,
    addEventListener: (_: string, fn: () => void) => listeners.add(fn),
    removeEventListener: (_: string, fn: () => void) => listeners.delete(fn),
  })
  return { history, listeners, popstate: () => [...listeners].forEach((fn) => fn()) }
}

describe('createBackGuard held and release', () => {
  it('reports its entry, and lets it go without popping when the page navigated by itself', () => {
    const env = fakeEnv()
    const guard = createBackGuard(env, () => true)
    expect(guard.held()).toBe(false)
    guard.set(true)
    expect(guard.held()).toBe(true)
    guard.release()
    expect(guard.held()).toBe(false)
    guard.set(false)
    expect(env.back).not.toHaveBeenCalled()
  })
})
