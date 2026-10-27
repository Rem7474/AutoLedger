import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope, nextTick, ref } from 'vue'
import { useSheetSwipe } from './useSheetSwipe'

// The unit tests run without a DOM: the panel and its touches are stand-ins that record what the composable does.
class FakeElement {
  style: Record<string, string> = {}
  listeners = new Map<string, (e: unknown) => void>()
  parentElement: FakeElement | null = null
  scrollHeight = 0
  clientHeight = 0
  scrollTop = 0
  overflowY = 'visible'
  isField = false
  height = 600
  addEventListener(type: string, fn: (e: unknown) => void) {
    this.listeners.set(type, fn)
  }
  removeEventListener(type: string) {
    this.listeners.delete(type)
  }
  closest() {
    return this.isField ? this : null
  }
  getBoundingClientRect() {
    return { height: this.height }
  }
}

let sheet = true
let reduced = false

function touch(type: string, target: FakeElement, y: number, t: number, fingers = 1) {
  const point = { clientX: 100, clientY: y }
  return { type, target, timeStamp: t, cancelable: true, preventDefault: vi.fn(), touches: Array(fingers).fill(point), changedTouches: [point] }
}

function setup(maxWidth?: number) {
  const panel = new FakeElement()
  const root = new FakeElement()
  panel.parentElement = root
  const dismiss = vi.fn()
  const el = ref<FakeElement | null>(panel)
  const scope = effectScope()
  scope.run(() => useSheetSwipe(el as never, dismiss, maxWidth))
  const fire = (e: ReturnType<typeof touch>) => panel.listeners.get(e.type)?.(e)
  return { panel, dismiss, el, scope, fire }
}

beforeEach(() => {
  sheet = true
  reduced = false
  vi.useFakeTimers()
  vi.stubGlobal('HTMLElement', FakeElement)
  vi.stubGlobal('Element', FakeElement)
  vi.stubGlobal('getComputedStyle', (e: FakeElement) => ({ overflowY: e.overflowY }))
  vi.stubGlobal('window', {
    matchMedia: (q: string) => ({ matches: q.includes('reduce') ? reduced : sheet }),
    setTimeout: (fn: () => void, ms: number) => setTimeout(fn, ms),
  })
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('useSheetSwipe', () => {
  it('follows the finger, then slides out and closes after a long pull', () => {
    const { panel, dismiss, fire } = setup()
    fire(touch('touchstart', panel, 100, 0))
    const move = touch('touchmove', panel, 180, 50)
    fire(move)
    expect(move.preventDefault).toHaveBeenCalled()
    expect(panel.style.transition).toBe('none')
    expect(panel.style.transform).toBe('translateY(80px)')
    fire(touch('touchmove', panel, 300, 100))
    fire(touch('touchend', panel, 300, 100))
    expect(panel.style.transform).toBe('translateY(100%)')
    expect(panel.style.transition).toContain('transform')
    expect(dismiss).not.toHaveBeenCalled()
    vi.advanceTimersByTime(200)
    expect(dismiss).toHaveBeenCalledTimes(1)
    expect(panel.style.transform).toBe('')
    expect(panel.style.transition).toBe('')
  })

  it('settles back after a short pull', () => {
    const { panel, dismiss, fire } = setup()
    fire(touch('touchstart', panel, 100, 0))
    fire(touch('touchmove', panel, 120, 300))
    fire(touch('touchmove', panel, 140, 700))
    fire(touch('touchend', panel, 140, 700))
    expect(panel.style.transform).toBe('')
    vi.advanceTimersByTime(200)
    expect(dismiss).not.toHaveBeenCalled()
    expect(panel.style.transition).toBe('')
  })

  it('settles back when the touch is cancelled, whatever the distance', () => {
    const { panel, dismiss, fire } = setup()
    fire(touch('touchstart', panel, 100, 0))
    fire(touch('touchmove', panel, 400, 20))
    fire(touch('touchcancel', panel, 400, 20))
    vi.advanceTimersByTime(200)
    expect(dismiss).not.toHaveBeenCalled()
    expect(panel.style.transform).toBe('')
  })

  it('skips the animation when motion is reduced', () => {
    reduced = true
    const { panel, dismiss, fire } = setup()
    fire(touch('touchstart', panel, 100, 0))
    fire(touch('touchmove', panel, 400, 20))
    fire(touch('touchend', panel, 400, 20))
    expect(dismiss).toHaveBeenCalledTimes(1)
    expect(panel.style.transform).toBe('')

    fire(touch('touchstart', panel, 100, 0))
    fire(touch('touchmove', panel, 120, 400))
    fire(touch('touchend', panel, 120, 400))
    expect(dismiss).toHaveBeenCalledTimes(1)
    expect(panel.style.transform).toBe('')
  })

  it('does nothing above the sheet breakpoint, from a field, or with several fingers', () => {
    const { panel, dismiss, fire } = setup()
    const field = new FakeElement()
    field.isField = true
    const attempts = [
      () => {
        sheet = false
        fire(touch('touchstart', panel, 100, 0))
      },
      () => {
        sheet = true
        fire(touch('touchstart', field, 100, 0))
      },
      () => fire(touch('touchstart', panel, 100, 0, 2)),
    ]
    for (const begin of attempts) {
      begin()
      const move = touch('touchmove', panel, 400, 20)
      fire(move)
      fire(touch('touchend', panel, 400, 20))
      expect(move.preventDefault).not.toHaveBeenCalled()
      expect(panel.style.transform).toBeUndefined()
    }
    vi.advanceTimersByTime(200)
    expect(dismiss).not.toHaveBeenCalled()
  })

  it('does not preventDefault on a touchmove that cannot be cancelled', () => {
    const { panel, fire } = setup()
    fire(touch('touchstart', panel, 100, 0))
    const move = { ...touch('touchmove', panel, 200, 20), cancelable: false }
    fire(move)
    expect(move.preventDefault).not.toHaveBeenCalled()
    expect(panel.style.transform).toBe('translateY(100px)')
  })

  it('uses the breakpoint it is given', () => {
    const queries: string[] = []
    vi.stubGlobal('window', { matchMedia: (q: string) => (queries.push(q), { matches: true }), setTimeout })
    const { panel, fire } = setup(767)
    fire(touch('touchstart', panel, 100, 0))
    expect(queries).toContain('(max-width: 767px)')
  })

  describe('on content that scrolls', () => {
    function scroller(panel: FakeElement, scrollTop: number) {
      const body = new FakeElement()
      body.parentElement = panel
      body.scrollHeight = 900
      body.clientHeight = 400
      body.overflowY = 'auto'
      body.scrollTop = scrollTop
      const row = new FakeElement()
      row.parentElement = body
      return row
    }

    it('leaves a pull down to the scroll when the content is not at its top', () => {
      const { panel, fire } = setup()
      const row = scroller(panel, 120)
      fire(touch('touchstart', row, 100, 0))
      const move = touch('touchmove', row, 300, 20)
      fire(move)
      fire(touch('touchend', row, 300, 20))
      expect(move.preventDefault).not.toHaveBeenCalled()
      expect(panel.style.transform).toBeUndefined()
    })

    it('drags the sheet when the content is at its top', () => {
      const { panel, dismiss, fire } = setup()
      const row = scroller(panel, 0)
      fire(touch('touchstart', row, 100, 0))
      fire(touch('touchmove', row, 300, 20))
      fire(touch('touchend', row, 300, 20))
      vi.advanceTimersByTime(200)
      expect(dismiss).toHaveBeenCalledTimes(1)
    })

    it('ignores ancestors that do not scroll', () => {
      const { panel, fire } = setup()
      const row = scroller(panel, 120)
      row.parentElement!.overflowY = 'visible'
      fire(touch('touchstart', row, 100, 0))
      fire(touch('touchmove', row, 300, 20))
      expect(panel.style.transform).toBe('translateY(200px)')
    })
  })

  it('follows the panel it is bound to and lets go of it when disposed', async () => {
    const { panel, el, scope } = setup()
    expect(panel.listeners.size).toBe(4)
    const other = new FakeElement()
    el.value = other
    await nextTick()
    expect(panel.listeners.size).toBe(0)
    expect(other.listeners.size).toBe(4)
    el.value = null
    await nextTick()
    expect(other.listeners.size).toBe(0)
    el.value = panel
    await nextTick()
    expect(panel.listeners.size).toBe(4)
    scope.stop()
    expect(panel.listeners.size).toBe(0)
  })
})
