// What the guard needs from the browser; a test passes a stand-in.
export interface BackEnv {
  /** Adds a history entry on the current page (same address). */
  push(): void
  /** Removes the entry added by `push`. */
  back(): void
  /** Calls `fn` on every history traversal; returns the function that stops listening. */
  listen(fn: () => void): () => void
}

/**
 * Makes the system back gesture (or button) close a transient mode instead of leaving the page. One guard serves the
 * whole stack of layers (see useEscapeToClose). While `set(true)` is
 * in effect the guard holds one history entry; Back pops it and calls `close`. When the mode ends some other way, the
 * guard pops its own entry so the next Back goes where the user expects.
 */
export function createBackGuard(env: BackEnv, close: () => boolean) {
  let onTop = false // our entry is the current one
  let popping = false // we asked for `back()` and its traversal has not arrived yet
  let wanted = false
  const stop = env.listen(() => {
    if (popping) {
      // Our own pop: if the mode came back before it landed, hold a new entry now
      popping = false
      if (wanted) {
        env.push()
        onTop = true
      }
      return
    }
    if (!onTop) return
    onTop = false
    // A mode that cannot close right now (busy) keeps its entry, so the next Back still closes it
    if (!close()) {
      env.push()
      onTop = true
    }
  })
  return {
    set(active: boolean) {
      wanted = active
      if (active && !onTop && !popping) {
        env.push()
        onTop = true
      } else if (!active && onTop) {
        onTop = false
        popping = true
        env.back()
      }
    },
    /** Whether the guard's entry is the current one. */
    held: () => onTop,
    /** Gives the entry up without popping it: the page's own navigation has replaced it. */
    release() {
      onTop = false
    },
    dispose() {
      stop()
      // The entry stays when the page itself is left: popping it would navigate back to the page being left
      onTop = false
      popping = false
      wanted = false
    },
  }
}

export const browserBackEnv: BackEnv = {
  // Keeps the router's own state in the entry, so it still finds its place on traversal
  push: () => window.history.pushState({ ...window.history.state, backGuard: true }, ''),
  back: () => window.history.back(),
  listen: (fn) => {
    window.addEventListener('popstate', fn)
    return () => window.removeEventListener('popstate', fn)
  },
}
