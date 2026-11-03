import { createPullGesture, PULL_THRESHOLD } from '@/utils/pullRefresh'

// What the pull of the shell does with a touch, without the DOM: the composable binds the events of the scrolling
// element to these three calls and shows the state.

export interface PullState {
  offset: number
  armed: boolean
  pulling: boolean
  refreshing: boolean
}

export interface PullHost {
  isPhone(): boolean
  /** The page on screen, when it says how it reloads. */
  handler(): (() => unknown) | null
  /** The touch began on something that keeps it (a field, a marked area) or a sheet is open. */
  isBlocked(target: EventTarget | null): boolean
  scrollTop(): number
  /** Resolves after `ms`: the spinner stays long enough to be seen. */
  wait(ms: number): Promise<void>
  onChange(state: PullState): void
}

export const SPINNER_MS = 450

export function createPullController(host: PullHost) {
  const gesture = createPullGesture()
  const state: PullState = { offset: 0, armed: false, pulling: false, refreshing: false }

  function set(patch: Partial<PullState>) {
    Object.assign(state, patch)
    host.onChange({ ...state })
  }

  return {
    start(touches: number, x: number, y: number, target: EventTarget | null) {
      if (touches !== 1 || !host.isPhone() || !host.handler() || state.refreshing || host.isBlocked(target)) return gesture.ignore()
      gesture.start(x, y, host.scrollTop())
    },
    /** True when the move is a pull: the caller keeps the browser from scrolling. */
    move(x: number, y: number): boolean {
      const move = gesture.move(x, y)
      if (!move) return false
      set({ pulling: true, offset: move.offset, armed: move.armed })
      return true
    },
    async end(cancelled: boolean) {
      const end = gesture.end(cancelled)
      const refresh = host.handler()
      if (end !== 'refresh' || !refresh) {
        if (state.pulling || state.offset) set({ pulling: false, armed: false, offset: 0 })
        return
      }
      set({ pulling: false, armed: false, refreshing: true, offset: PULL_THRESHOLD * 0.75 })
      try {
        await Promise.all([Promise.resolve(refresh()), host.wait(SPINNER_MS)])
      } catch {
        // The page shows its own error
      } finally {
        set({ refreshing: false, offset: 0 })
      }
    },
  }
}
