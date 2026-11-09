import type { Router } from 'vue-router'

// The shell scrolls inside `.app-scroll`, which is never replaced, so a page opened after another starts where the
// previous one was left and the browser cannot restore a position on Back. This keeps the position of every page of
// the history and puts it back when the reader comes back to it (Back or Forward); a page opened afresh starts at the top.

/** How long a position is waited for while the page loads its data. */
export const RESTORE_MS = 1500
const STORAGE_KEY = 'al-scroll'
const MAX_ENTRIES = 60

/**
 * One history entry: `position` is vue-router's counter in `history.state`, the path tells apart the page that replaced
 * another one in the same entry (a link of a sheet), so two visits of the same page keep their own position.
 */
export const scrollKey = (position: unknown, path: string): string => `${typeof position === 'number' ? position : 0}|${path}`

export interface Store {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
}

/** Positions by history entry, kept for the life of the tab (a reload lands on the same entries). */
export function createScrollMemory(store: Store | null) {
  let map = new Map<string, number>()
  try {
    const raw = store?.getItem(STORAGE_KEY)
    if (raw) map = new Map(Object.entries(JSON.parse(raw) as Record<string, number>))
  } catch {
    // unreadable or blocked storage: start empty
  }
  const persist = () => {
    try {
      store?.setItem(STORAGE_KEY, JSON.stringify(Object.fromEntries(map)))
    } catch {
      // storage full or blocked: the positions stay in memory
    }
  }
  return {
    get: (key: string) => map.get(key),
    set(key: string, top: number) {
      map.delete(key)
      map.set(key, Math.round(top))
      // The oldest entries go first
      while (map.size > MAX_ENTRIES) map.delete(map.keys().next().value as string)
      persist()
    },
  }
}

export interface RestoreEnv {
  now(): number
  raf(fn: () => void): number
  cancelRaf(id: number): void
  /** Calls `fn` on the first touch, wheel or key of the reader; returns the function that stops listening. */
  onInteract(fn: () => void): () => void
}

interface Scroller {
  scrollTop: number
  scrollHeight: number
  clientHeight: number
}

/**
 * Brings `el` to `top` once the page is tall enough to reach it (its data arrives after the first render), waiting at most
 * `RESTORE_MS`, and gives up as soon as the reader scrolls by hand. Returns the function that cancels it.
 */
export function restoreScroll(el: Scroller, top: number, env: RestoreEnv): () => void {
  const deadline = env.now() + RESTORE_MS
  let frame = 0
  let stopped = false
  const stop = () => {
    stopped = true
    env.cancelRaf(frame)
    release()
  }
  const release = env.onInteract(stop)
  const step = () => {
    if (stopped) return
    if (el.scrollHeight - el.clientHeight >= top) {
      el.scrollTop = top
      stop()
    } else if (env.now() >= deadline) {
      stop()
    } else {
      frame = env.raf(step)
    }
  }
  step()
  return stop
}

export const browserRestoreEnv: RestoreEnv = {
  now: () => performance.now(),
  raf: (fn) => requestAnimationFrame(fn),
  cancelRaf: (id) => cancelAnimationFrame(id),
  onInteract(fn) {
    const events = ['touchstart', 'wheel', 'keydown'] as const
    for (const e of events) window.addEventListener(e, fn, { passive: true, once: true })
    return () => events.forEach((e) => window.removeEventListener(e, fn))
  },
}

type Direction = 'push' | 'pop'

/**
 * What a finished navigation does to the scroll: a return to an entry restores its position, a new page starts at the
 * top, a change that stays on the page (filter, tab) leaves it where it is.
 */
export function scrollAction(type: Direction, samePage: boolean, saved: number | undefined): { to: number } | null {
  if (type === 'pop') return saved === undefined ? null : { to: saved }
  return samePage ? null : { to: 0 }
}

/** Remembers the scroll of the page left and restores it on Back/Forward. */
export function installScrollMemory(
  router: Router,
  o: {
    scroller: () => HTMLElement | null
    memory?: ReturnType<typeof createScrollMemory>
    env?: RestoreEnv
    position?: () => unknown
  },
) {
  const memory = o.memory ?? createScrollMemory(typeof sessionStorage === 'undefined' ? null : sessionStorage)
  const env = o.env ?? browserRestoreEnv
  const position = o.position ?? (() => window.history.state?.position)
  let type: Direction = 'push'
  let current: string | null = null
  let cancel: (() => void) | null = null

  // Before the guards: on Back, `history.state` already names the entry being reached, so the page left is tracked here
  router.options.history.listen((_to, _from, info) => {
    type = info.type === 'pop' ? 'pop' : 'push'
  })
  router.beforeEach(() => {
    cancel?.()
    cancel = null
    const el = o.scroller()
    if (el && current !== null) memory.set(current, el.scrollTop)
  })
  router.afterEach((to, from, failure) => {
    // The first navigation of a page load is a reload or a return to the tab's entry: it restores what it has
    const navigation: Direction = from.matched.length === 0 ? 'pop' : type
    type = 'push'
    if (failure) return
    current = scrollKey(position(), to.path)
    const el = o.scroller()
    if (!el) return
    const action = scrollAction(navigation, to.path === from.path, memory.get(current))
    if (!action) return
    if (action.to === 0) el.scrollTop = 0
    else cancel = restoreScroll(el, action.to, env)
  })
}
