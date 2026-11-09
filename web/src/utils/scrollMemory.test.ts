import { describe, expect, it, vi } from 'vitest'
import { createScrollMemory, installScrollMemory, restoreScroll, scrollAction, scrollKey, RESTORE_MS, type RestoreEnv, type Store } from './scrollMemory'

const memStore = (): Store & { data: Map<string, string> } => {
  const data = new Map<string, string>()
  return { data, getItem: (k) => data.get(k) ?? null, setItem: (k, v) => void data.set(k, v) }
}

describe('scrollKey', () => {
  it('tells two entries of the same page apart, and a page that replaced another in the same entry', () => {
    expect(scrollKey(4, '/drives')).not.toBe(scrollKey(7, '/drives'))
    expect(scrollKey(4, '/drives')).not.toBe(scrollKey(4, '/carpools'))
  })

  it('reads a missing position as the first entry', () => {
    expect(scrollKey(undefined, '/')).toBe('0|/')
  })
})

describe('createScrollMemory', () => {
  it('keeps a rounded position per key and reads it back after a reload', () => {
    const store = memStore()
    createScrollMemory(store).set('1|/a', 812.6)
    expect(createScrollMemory(store).get('1|/a')).toBe(813)
    expect(createScrollMemory(store).get('2|/a')).toBeUndefined()
  })

  it('forgets the oldest entries first', () => {
    const memory = createScrollMemory(null)
    for (let i = 0; i < 70; i++) memory.set(`${i}|/p`, i)
    expect(memory.get('0|/p')).toBeUndefined()
    expect(memory.get('69|/p')).toBe(69)
  })

  it('starts empty when the storage is unreadable or refuses to write', () => {
    const broken: Store = {
      getItem: () => '{not json',
      setItem: () => {
        throw new Error('full')
      },
    }
    const memory = createScrollMemory(broken)
    expect(() => memory.set('1|/a', 5)).not.toThrow()
    expect(memory.get('1|/a')).toBe(5)
  })
})

describe('scrollAction', () => {
  it('restores the saved position on a return, and leaves the page alone when none was saved', () => {
    expect(scrollAction('pop', false, 640)).toEqual({ to: 640 })
    expect(scrollAction('pop', false, undefined)).toBeNull()
  })

  it('starts a new page at the top, and keeps a change that stays on the page', () => {
    expect(scrollAction('push', false, 640)).toEqual({ to: 0 })
    expect(scrollAction('push', true, undefined)).toBeNull()
  })
})

function fakeEnv() {
  let time = 0
  const frames: Array<() => void> = []
  let interact: () => void = () => {}
  const env: RestoreEnv = {
    now: () => time,
    raf: (fn) => frames.push(fn),
    cancelRaf: vi.fn(),
    onInteract(fn) {
      interact = fn
      return () => {}
    },
  }
  return {
    env,
    tick(ms = 16) {
      time += ms
      frames.splice(0).forEach((f) => f())
    },
    interact: () => interact(),
  }
}

describe('restoreScroll', () => {
  it('applies the position at once when the page is tall enough', () => {
    const f = fakeEnv()
    const el = { scrollTop: 0, scrollHeight: 3000, clientHeight: 800 }
    restoreScroll(el, 900, f.env)
    expect(el.scrollTop).toBe(900)
  })

  it('waits for the data to make the page tall enough', () => {
    const f = fakeEnv()
    const el = { scrollTop: 0, scrollHeight: 800, clientHeight: 800 }
    restoreScroll(el, 900, f.env)
    expect(el.scrollTop).toBe(0)
    f.tick()
    el.scrollHeight = 2500
    f.tick()
    expect(el.scrollTop).toBe(900)
  })

  it('gives up after the delay without moving the page', () => {
    const f = fakeEnv()
    const el = { scrollTop: 0, scrollHeight: 800, clientHeight: 800 }
    restoreScroll(el, 900, f.env)
    f.tick(RESTORE_MS + 1)
    el.scrollHeight = 5000
    f.tick()
    expect(el.scrollTop).toBe(0)
  })

  it('gives up when the reader scrolls by hand', () => {
    const f = fakeEnv()
    const el = { scrollTop: 0, scrollHeight: 800, clientHeight: 800 }
    restoreScroll(el, 900, f.env)
    f.interact()
    el.scrollHeight = 5000
    f.tick()
    expect(el.scrollTop).toBe(0)
  })

  it('can be cancelled', () => {
    const f = fakeEnv()
    const el = { scrollTop: 0, scrollHeight: 800, clientHeight: 800 }
    const cancel = restoreScroll(el, 900, f.env)
    cancel()
    el.scrollHeight = 5000
    f.tick()
    expect(el.scrollTop).toBe(0)
  })
})

describe('installScrollMemory', () => {
  function fakeRouter() {
    let listener: (to: unknown, from: unknown, info: { type: string }) => void = () => {}
    const before: Array<() => void> = []
    const after: Array<(to: { path: string }, from: { path: string; matched: unknown[] }, failure?: unknown) => void> = []
    const router = {
      options: { history: { listen: (fn: typeof listener) => void (listener = fn) } },
      beforeEach: (fn: () => void) => void before.push(fn),
      afterEach: (fn: (typeof after)[number]) => void after.push(fn),
    }
    const go = (to: string, from: string, type: 'push' | 'pop', failure?: unknown, first = false) => {
      listener(null, null, { type })
      before.forEach((f) => f())
      after.forEach((f) => f({ path: to }, { path: from, matched: first ? [] : [1] }, failure))
    }
    return { router, go }
  }

  it('saves the page left, starts a new page at the top and restores the page returned to', () => {
    const { router, go } = fakeRouter()
    const el = { scrollTop: 0, scrollHeight: 4000, clientHeight: 800 } as unknown as HTMLElement
    let position = 1
    const f = fakeEnv()
    installScrollMemory(router as never, { scroller: () => el, memory: createScrollMemory(null), env: f.env, position: () => position })

    go('/drives', '/', 'push', undefined, true)
    el.scrollTop = 800
    position = 2
    go('/carpools', '/drives', 'push')
    expect(el.scrollTop).toBe(0)

    position = 1
    go('/drives', '/carpools', 'pop')
    expect(el.scrollTop).toBe(800)
  })

  it('leaves the scroll alone on a navigation that stays on the page, and on a failed one', () => {
    const { router, go } = fakeRouter()
    const el = { scrollTop: 300, scrollHeight: 4000, clientHeight: 800 } as unknown as HTMLElement
    installScrollMemory(router as never, { scroller: () => el, memory: createScrollMemory(null), env: fakeEnv().env, position: () => 1 })
    go('/drives', '/drives', 'push')
    expect(el.scrollTop).toBe(300)
    go('/carpools', '/drives', 'push', new Error('cancelled'))
    expect(el.scrollTop).toBe(300)
  })

  it('restores what was saved on the first navigation of a page load, and does nothing without the container', () => {
    const { router, go } = fakeRouter()
    const memory = createScrollMemory(null)
    memory.set('1|/drives', 450)
    const el = { scrollTop: 0, scrollHeight: 4000, clientHeight: 800 } as unknown as HTMLElement
    let container: HTMLElement | null = el
    installScrollMemory(router as never, { scroller: () => container, memory, env: fakeEnv().env, position: () => 1 })
    go('/drives', '/', 'push', undefined, true)
    expect(el.scrollTop).toBe(450)
    container = null
    expect(() => go('/carpools', '/drives', 'push')).not.toThrow()
  })
})
