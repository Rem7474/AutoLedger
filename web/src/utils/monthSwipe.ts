// Moving between the months of the month detail with a horizontal swipe. The decision is taken when the finger lifts:
// a long enough move, clearly more horizontal than vertical (a vertical one is the sheet scrolling or closing). A swipe
// to the right brings the month before, one to the left the month after, like turning the pages of a calendar.

/** How far the finger must travel sideways. */
export const MONTH_SWIPE_PX = 56
/** The horizontal move must beat the vertical one by this factor. */
const HORIZONTAL_BIAS = 1.5
/** The edges of the screen belong to the system's back gesture. */
export const MONTH_SWIPE_EDGE_PX = 24

export type MonthSwipe = 'previous' | 'next'

export function monthSwipeDirection(dx: number, dy: number): MonthSwipe | null {
  if (Math.abs(dx) < MONTH_SWIPE_PX || Math.abs(dx) < HORIZONTAL_BIAS * Math.abs(dy)) return null
  return dx > 0 ? 'previous' : 'next'
}

/** Whether a touch that starts at `x` (of a screen `width` wide) is left to the system. */
export const startsAtScreenEdge = (x: number, width: number): boolean => x < MONTH_SWIPE_EDGE_PX || x > width - MONTH_SWIPE_EDGE_PX
