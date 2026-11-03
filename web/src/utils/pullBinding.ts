import type { createPullController } from '@/utils/pullController'

// The touch events of the scrolling element, handed to the pull controller. Typed on what the binding reads, so it
// runs against any element-like object.

interface TouchPointLike {
  clientX: number
  clientY: number
}

export interface TouchEventLike {
  type: string
  touches: ArrayLike<TouchPointLike>
  target: EventTarget | null
  cancelable: boolean
  preventDefault(): void
}

export interface PullEventTarget {
  addEventListener(type: string, listener: (event: TouchEventLike) => void, options?: { passive: boolean }): void
  removeEventListener(type: string, listener: (event: TouchEventLike) => void): void
}

/** Listens to the touches of `el` for the controller; the returned function stops listening. */
export function bindPullEvents(el: PullEventTarget, controller: ReturnType<typeof createPullController>): () => void {
  const onStart = (e: TouchEventLike) => controller.start(e.touches.length, e.touches[0]?.clientX ?? 0, e.touches[0]?.clientY ?? 0, e.target)
  const onMove = (e: TouchEventLike) => {
    const point = e.touches[0]
    if (point && controller.move(point.clientX, point.clientY) && e.cancelable) e.preventDefault()
  }
  const onEnd = (e: TouchEventLike) => void controller.end(e.type === 'touchcancel')

  el.addEventListener('touchstart', onStart, { passive: true })
  // Not passive: a pull that started at the top must keep the browser from scrolling or reloading the page.
  el.addEventListener('touchmove', onMove, { passive: false })
  el.addEventListener('touchend', onEnd)
  el.addEventListener('touchcancel', onEnd)
  return () => {
    el.removeEventListener('touchstart', onStart)
    el.removeEventListener('touchmove', onMove)
    el.removeEventListener('touchend', onEnd)
    el.removeEventListener('touchcancel', onEnd)
  }
}
