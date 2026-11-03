import { nextTick } from 'vue'
import type { RouteLocationNormalized, Router } from 'vue-router'

// Going from one page to another crossfades on a phone, through the browser's View Transitions, instead of swapping
// the screen in one frame. Where the browser has no such API, the user asks for less motion, or the window is wide,
// the navigation is the plain one.

/** The longest the old page is held on screen while the new one loads (a slow chunk must not freeze the screen). */
export const HOLD_MS = 1500

export interface TransitionEnv {
  supported: boolean
  phone: boolean
  reducedMotion: boolean
}

type Place = Pick<RouteLocationNormalized, 'name'>

/** Only a change of page animates: a filter in the query string or a tab of the same page does not. */
export function shouldAnimate(to: Place, from: Place, env: TransitionEnv): boolean {
  if (!env.supported || !env.phone || env.reducedMotion) return false
  if (!from.name || !to.name) return false
  return to.name !== from.name
}

export function browserEnv(win: Window = window, doc: Document = document): TransitionEnv {
  return {
    supported: typeof (doc as Document & { startViewTransition?: unknown }).startViewTransition === 'function',
    phone: win.matchMedia('(max-width: 639px)').matches,
    reducedMotion: win.matchMedia('(prefers-reduced-motion: reduce)').matches,
  }
}

type StartViewTransition = (update: () => Promise<void>) => unknown

export function installViewTransitions(
  router: Pick<Router, 'beforeResolve' | 'afterEach' | 'onError'>,
  env: () => TransitionEnv = browserEnv,
  start: () => StartViewTransition | null = () => (typeof document !== 'undefined' && 'startViewTransition' in document ? (document as unknown as { startViewTransition: StartViewTransition }).startViewTransition.bind(document) : null),
) {
  let finish: (() => void) | null = null
  const end = () => {
    const done = finish
    finish = null
    // The new page is in the DOM once the pending render has run
    if (done) void nextTick().then(done)
  }

  router.beforeResolve((to, from) => {
    const startTransition = start()
    if (!startTransition || !shouldAnimate(to, from, env())) return
    return new Promise<void>((resolve) => {
      startTransition(() => {
        resolve()
        return new Promise<void>((done) => {
          finish = done
          setTimeout(end, HOLD_MS)
        })
      })
    })
  })
  router.afterEach(end)
  router.onError(end)
}
