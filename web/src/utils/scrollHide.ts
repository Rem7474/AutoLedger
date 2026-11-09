// A bar that gets out of the way while the page is read downwards and comes back as soon as it is read upwards.
export interface ScrollHideState {
  hidden: boolean
  /** The offset at which the state last changed: the movement is measured from it. */
  anchor: number
}

export const SCROLL_HIDE_DELTA = 12
export const SCROLL_HIDE_TOP = 24

export const initialScrollHide: ScrollHideState = { hidden: false, anchor: 0 }

/** The state after the scroller reached offset `y`. Near the top the bar is always shown; small movements change nothing. */
export function stepScrollHide(state: ScrollHideState, y: number): ScrollHideState {
  if (y <= SCROLL_HIDE_TOP) return state.hidden || state.anchor !== y ? { hidden: false, anchor: y } : state
  const moved = y - state.anchor
  if (moved > SCROLL_HIDE_DELTA) return { hidden: true, anchor: y }
  if (moved < -SCROLL_HIDE_DELTA) return { hidden: false, anchor: y }
  return state
}
