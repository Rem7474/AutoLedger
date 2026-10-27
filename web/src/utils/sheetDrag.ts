/** Distance, in px, after which a released sheet closes whatever the speed. */
export const DISMISS_DISTANCE = 110
/** Speed, in px/ms, of a flick that closes a sheet dragged only a little. */
export const DISMISS_VELOCITY = 0.5
/** A flick shorter than this does not close the sheet, so a tap that slips is not a dismissal. */
export const MIN_FLICK_DISTANCE = 30
/** Pixels of movement before a touch is read as a drag or as a scroll. */
export const DRAG_SLOP = 4

/** How far the sheet follows the finger: down only, never above its resting place. */
export function sheetOffset(dy: number): number {
  return dy > 0 ? dy : 0
}

/** Whether releasing the sheet after a drag of `dy` px at `velocity` px/ms closes it. */
export function shouldDismiss(dy: number, velocity: number, sheetHeight: number): boolean {
  if (dy >= Math.min(DISMISS_DISTANCE, Math.max(sheetHeight / 3, MIN_FLICK_DISTANCE))) return true
  return dy >= MIN_FLICK_DISTANCE && velocity >= DISMISS_VELOCITY
}

export type GestureStart = 'drag' | 'scroll' | 'wait'

/**
 * Reads the first movement of a touch: a pull down from the top of what the finger is on drags the sheet; a move up,
 * sideways, or down inside content that is scrolled leaves the touch to the page.
 */
export function readGesture(dx: number, dy: number, scrollTop: number): GestureStart {
  if (Math.abs(dx) < DRAG_SLOP && Math.abs(dy) < DRAG_SLOP) return 'wait'
  if (dy <= 0 || Math.abs(dx) > Math.abs(dy)) return 'scroll'
  return scrollTop > 0 ? 'scroll' : 'drag'
}

/** Speed in px/ms of the last samples of a drag, 0 when there are fewer than two. */
export function releaseVelocity(samples: ReadonlyArray<{ y: number; t: number }>): number {
  if (samples.length < 2) return 0
  const first = samples[0]
  const last = samples[samples.length - 1]
  const dt = last.t - first.t
  return dt > 0 ? (last.y - first.y) / dt : 0
}
