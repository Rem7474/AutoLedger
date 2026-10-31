import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useLongPress } from './useLongPress'

const press = (type: string, x = 0, y = 0) => ({ pointerType: type, clientX: x, clientY: y }) as PointerEvent

describe('useLongPress', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('fires once the finger has stayed down for the delay, and swallows the click that follows, wherever it lands', () => {
    const listeners: Record<string, (e: Event) => void> = {}
    vi.stubGlobal('document', { addEventListener: (t: string, f: any) => (listeners[t] = f), removeEventListener: (t: string) => delete listeners[t] })
    const cb = vi.fn()
    const p = useLongPress(cb)
    p.onPointerdown(press('touch'))
    vi.advanceTimersByTime(449)
    expect(cb).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1)
    expect(cb).toHaveBeenCalledTimes(1)
    const click = { stopPropagation: vi.fn(), preventDefault: vi.fn() }
    listeners.click(click as any)
    expect(click.preventDefault).toHaveBeenCalled()
    expect(listeners.click).toBeUndefined()
    vi.unstubAllGlobals()
  })

  it('stops swallowing clicks after a moment when none comes', () => {
    const listeners: Record<string, (e: Event) => void> = {}
    vi.stubGlobal('document', { addEventListener: (t: string, f: any) => (listeners[t] = f), removeEventListener: (t: string) => delete listeners[t] })
    const p = useLongPress(vi.fn())
    p.onPointerdown(press('touch'))
    vi.advanceTimersByTime(450)
    expect(listeners.click).toBeDefined()
    vi.advanceTimersByTime(600)
    expect(listeners.click).toBeUndefined()
    vi.unstubAllGlobals()
  })

  it('does not fire on a short tap', () => {
    const cb = vi.fn()
    const p = useLongPress(cb)
    p.onPointerdown(press('touch'))
    vi.advanceTimersByTime(200)
    p.onPointerup()
    vi.advanceTimersByTime(1000)
    expect(cb).not.toHaveBeenCalled()
  })

  it('gives up when the finger moves, a scroll', () => {
    const cb = vi.fn()
    const p = useLongPress(cb)
    p.onPointerdown(press('touch', 0, 0))
    p.onPointermove(press('touch', 2, 3))
    vi.advanceTimersByTime(450)
    expect(cb).toHaveBeenCalledTimes(1)
    cb.mockClear()
    p.onPointerdown(press('touch', 0, 0))
    p.onPointermove(press('touch', 0, 25))
    vi.advanceTimersByTime(1000)
    expect(cb).not.toHaveBeenCalled()
  })

  it('ignores a mouse press', () => {
    const cb = vi.fn()
    const p = useLongPress(cb)
    p.onPointerdown(press('mouse'))
    vi.advanceTimersByTime(1000)
    expect(cb).not.toHaveBeenCalled()
  })
})
