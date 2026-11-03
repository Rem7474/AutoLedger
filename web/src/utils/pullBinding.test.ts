import { describe, expect, it, vi } from 'vitest'
import { bindPullEvents, type PullEventTarget, type TouchEventLike } from './pullBinding'
import type { createPullController } from './pullController'

type Listener = (event: TouchEventLike) => void

/** An element that only keeps its listeners, so the tests can fire the events by hand. */
function fakeElement() {
  const listeners = new Map<string, { listener: Listener; options?: { passive: boolean } }>()
  const el: PullEventTarget = {
    addEventListener: (type, listener, options) => void listeners.set(type, { listener, options }),
    removeEventListener: (type, listener) => {
      if (listeners.get(type)?.listener === listener) listeners.delete(type)
    },
  }
  const fire = (type: string, event: Partial<TouchEventLike> = {}) => {
    const entry = listeners.get(type)
    if (!entry) throw new Error(`no ${type} listener`)
    entry.listener({ type, touches: [], target: null, cancelable: true, preventDefault: () => {}, ...event })
  }
  return { el, listeners, fire }
}

function fakeController(moves = true) {
  return {
    start: vi.fn(),
    move: vi.fn(() => moves),
    end: vi.fn(async () => {}),
  } as unknown as ReturnType<typeof createPullController> & { start: ReturnType<typeof vi.fn>; move: ReturnType<typeof vi.fn>; end: ReturnType<typeof vi.fn> }
}

describe('bindPullEvents', () => {
  it('hands the first finger of a touch start to the controller, with where it landed', () => {
    const { el, fire } = fakeElement()
    const controller = fakeController()
    const target = {} as EventTarget
    bindPullEvents(el, controller)
    fire('touchstart', { touches: [{ clientX: 12, clientY: 34 }, { clientX: 99, clientY: 99 }], target })
    expect(controller.start).toHaveBeenCalledWith(2, 12, 34, target)
  })

  it('reports a start without any finger as no touch at the origin', () => {
    const { el, fire } = fakeElement()
    const controller = fakeController()
    bindPullEvents(el, controller)
    fire('touchstart', { touches: [] })
    expect(controller.start).toHaveBeenCalledWith(0, 0, 0, null)
  })

  it('keeps the browser from scrolling while the controller pulls', () => {
    const { el, fire } = fakeElement()
    const controller = fakeController(true)
    const preventDefault = vi.fn()
    bindPullEvents(el, controller)
    fire('touchmove', { touches: [{ clientX: 5, clientY: 80 }], preventDefault })
    expect(controller.move).toHaveBeenCalledWith(5, 80)
    expect(preventDefault).toHaveBeenCalledTimes(1)
  })

  it('lets an ordinary scroll through', () => {
    const { el, fire } = fakeElement()
    const preventDefault = vi.fn()
    bindPullEvents(el, fakeController(false))
    fire('touchmove', { touches: [{ clientX: 5, clientY: 80 }], preventDefault })
    expect(preventDefault).not.toHaveBeenCalled()
  })

  it('does not cancel a move the browser no longer lets it cancel', () => {
    const { el, fire } = fakeElement()
    const preventDefault = vi.fn()
    bindPullEvents(el, fakeController(true))
    fire('touchmove', { touches: [{ clientX: 5, clientY: 80 }], cancelable: false, preventDefault })
    expect(preventDefault).not.toHaveBeenCalled()
  })

  it('ignores a move that carries no finger', () => {
    const { el, fire } = fakeElement()
    const controller = fakeController(true)
    bindPullEvents(el, controller)
    fire('touchmove', { touches: [] })
    expect(controller.move).not.toHaveBeenCalled()
  })

  it('tells the controller whether the touch ended or was cancelled', () => {
    const { el, fire } = fakeElement()
    const controller = fakeController()
    bindPullEvents(el, controller)
    fire('touchend')
    fire('touchcancel')
    expect(controller.end.mock.calls).toEqual([[false], [true]])
  })

  it('listens passively to the start only: a pull has to be able to cancel the scroll', () => {
    const { el, listeners } = fakeElement()
    bindPullEvents(el, fakeController())
    expect(listeners.get('touchstart')?.options).toEqual({ passive: true })
    expect(listeners.get('touchmove')?.options).toEqual({ passive: false })
  })

  it('stops listening to every event when detached', () => {
    const { el, listeners } = fakeElement()
    const detach = bindPullEvents(el, fakeController())
    expect([...listeners.keys()].sort()).toEqual(['touchcancel', 'touchend', 'touchmove', 'touchstart'])
    detach()
    expect(listeners.size).toBe(0)
  })
})
