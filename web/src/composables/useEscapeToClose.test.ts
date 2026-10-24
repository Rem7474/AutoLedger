import { describe, expect, it, vi } from 'vitest'
import { effectScope, ref } from 'vue'
import { closeTopLayerOnEscape, hasOpenEscapeLayer, useEscapeToClose } from './useEscapeToClose'

const press = (key = 'Escape', extra: Record<string, unknown> = {}) => ({ key, defaultPrevented: false, isComposing: false, preventDefault: vi.fn(), ...extra })

describe('useEscapeToClose', () => {
  it('closes an open layer on Escape and ignores other keys', () => {
    const scope = effectScope()
    const close = vi.fn()
    scope.run(() => useEscapeToClose(ref(true), close))
    expect(closeTopLayerOnEscape(press('Enter'))).toBe(false)
    expect(close).not.toHaveBeenCalled()
    const e = press()
    expect(closeTopLayerOnEscape(e)).toBe(true)
    expect(e.preventDefault).toHaveBeenCalled()
    expect(close).toHaveBeenCalledTimes(1)
    scope.stop()
  })

  it('only closes the layer opened last', () => {
    const scope = effectScope()
    const under = vi.fn()
    const over = vi.fn()
    const overOpen = ref(false)
    scope.run(() => {
      useEscapeToClose(ref(true), under)
      useEscapeToClose(overOpen, over)
    })
    overOpen.value = true
    return Promise.resolve().then(() => {
      closeTopLayerOnEscape(press())
      expect(over).toHaveBeenCalledTimes(1)
      expect(under).not.toHaveBeenCalled()
      overOpen.value = false
      return Promise.resolve().then(() => {
        closeTopLayerOnEscape(press())
        expect(under).toHaveBeenCalledTimes(1)
        scope.stop()
      })
    })
  })

  it('does nothing once closed, disposed, composing or already handled', async () => {
    const scope = effectScope()
    const close = vi.fn()
    const open = ref(true)
    scope.run(() => useEscapeToClose(open, close))
    expect(closeTopLayerOnEscape(press('Escape', { isComposing: true }))).toBe(false)
    expect(closeTopLayerOnEscape(press('Escape', { defaultPrevented: true }))).toBe(false)
    open.value = false
    await Promise.resolve()
    expect(closeTopLayerOnEscape(press())).toBe(false)
    open.value = true
    await Promise.resolve()
    scope.stop()
    expect(closeTopLayerOnEscape(press())).toBe(false)
    expect(close).not.toHaveBeenCalled()
  })

  it('tells whether a layer is open', async () => {
    const scope = effectScope()
    const open = ref(false)
    scope.run(() => useEscapeToClose(open, vi.fn()))
    expect(hasOpenEscapeLayer()).toBe(false)
    open.value = true
    await Promise.resolve()
    expect(hasOpenEscapeLayer()).toBe(true)
    scope.stop()
    expect(hasOpenEscapeLayer()).toBe(false)
  })
})
