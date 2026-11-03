// Pull-to-refresh as a state machine kept free of the DOM, so the rules can be tested: a pull counts only when it starts
// with the page scrolled to the top and goes down more than across; the finger is followed with some resistance, and
// letting go past the threshold asks for a refresh.

export const PULL_SLOP = 8
export const PULL_THRESHOLD = 64
export const PULL_MAX = 96
const RESISTANCE = 0.6

const offsetFor = (distance: number) => Math.min(PULL_MAX, Math.max(0, distance) * RESISTANCE)

export interface PullMove {
  /** Distance the indicator is dragged, in px, with resistance and a ceiling. */
  offset: number
  /** True when letting go now refreshes. */
  armed: boolean
  /** True on the first move of a pull: the caller takes the gesture over from the browser. */
  began: boolean
}

export type PullEnd = 'refresh' | 'cancel' | null

export function createPullGesture() {
  let startX = 0
  let startY = 0
  let tracking = false
  let pulling = false
  let armed = false

  return {
    /** A touch begins; `scrollTop` is where the scrolling element is. */
    start(x: number, y: number, scrollTop: number) {
      tracking = scrollTop <= 0
      pulling = false
      armed = false
      startX = x
      startY = y
    },
    /** Drop the touch (a second finger, a field, a sheet). */
    ignore() {
      tracking = false
      pulling = false
      armed = false
    },
    move(x: number, y: number): PullMove | null {
      if (!tracking) return null
      const dy = y - startY
      const dx = x - startX
      if (!pulling) {
        // Upwards, or more across than down: this is a scroll or a swipe, not a pull
        if (dy < 0 || Math.abs(dx) > Math.abs(dy)) {
          if (dy < -PULL_SLOP || Math.abs(dx) > PULL_SLOP) tracking = false
          return null
        }
        if (dy <= PULL_SLOP) return null
        pulling = true
        const offset = offsetFor(dy - PULL_SLOP)
        armed = offset >= PULL_THRESHOLD
        return { offset, armed, began: true }
      }
      const offset = offsetFor(dy - PULL_SLOP)
      armed = offset >= PULL_THRESHOLD
      return { offset, armed, began: false }
    },
    end(cancelled = false): PullEnd {
      const wasPulling = pulling
      const wasArmed = armed
      tracking = false
      pulling = false
      armed = false
      if (!wasPulling) return null
      return wasArmed && !cancelled ? 'refresh' : 'cancel'
    },
  }
}
