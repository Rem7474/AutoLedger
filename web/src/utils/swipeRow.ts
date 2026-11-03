// A row dragged sideways to delete it, as a state machine without the DOM. The direction is decided once the finger has
// moved a little: a mostly vertical move is a scroll and the row is left alone, a mostly horizontal one to the left
// drags the row. Letting go past `SWIPE_COMMIT_PX` deletes; before that the row slides back.

/** How far the finger must move before the direction is decided: past the 10 px a long press tolerates, so a swipe has cancelled the press by then. */
export const SWIPE_LOCK_PX = 12
/** The drag at which letting go deletes the row. */
export const SWIPE_COMMIT_PX = 96
/** The row never goes further than this: the red area behind it is that wide. */
export const SWIPE_MAX_PX = 140
/** Horizontal movement must beat the vertical one by this factor, or the move is a scroll. */
const HORIZONTAL_BIAS = 1.5

export type SwipeEnd = 'delete' | 'none'
export interface SwipeMove {
  /** Where the row is: 0 at rest, negative to the left. */
  offset: number
  /** Letting go now deletes. */
  armed: boolean
}

type Phase = 'idle' | 'undecided' | 'dragging' | 'scrolling'

export function createSwipeGesture() {
  let phase: Phase = 'idle'
  let startX = 0
  let startY = 0
  let last: SwipeMove = { offset: 0, armed: false }

  return {
    start(x: number, y: number) {
      phase = 'undecided'
      startX = x
      startY = y
      last = { offset: 0, armed: false }
    },
    /** The row's position while it is dragged; null when the move is not a swipe (yet, or at all). */
    move(x: number, y: number): SwipeMove | null {
      if (phase === 'idle' || phase === 'scrolling') return null
      const dx = x - startX
      const dy = y - startY
      if (phase === 'undecided') {
        if (Math.hypot(dx, dy) < SWIPE_LOCK_PX) return null
        if (dx < 0 && Math.abs(dx) > Math.abs(dy) * HORIZONTAL_BIAS) phase = 'dragging'
        else {
          phase = 'scrolling'
          return null
        }
      }
      const offset = Math.max(-SWIPE_MAX_PX, Math.min(0, dx))
      last = { offset, armed: -offset >= SWIPE_COMMIT_PX }
      return last
    },
    /** What letting go does; `cancelled` is a touch the system took over. */
    end(cancelled: boolean): SwipeEnd {
      const result: SwipeEnd = phase === 'dragging' && last.armed && !cancelled ? 'delete' : 'none'
      phase = 'idle'
      last = { offset: 0, armed: false }
      return result
    },
    /** True once the move is a swipe: the click that follows must not open the row. */
    get dragging() {
      return phase === 'dragging'
    },
  }
}
