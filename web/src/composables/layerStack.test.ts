import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'

// The registry holds module state and one history guard: each test loads a fresh copy against a stand-in window
function setup() {
  const listeners = new Map<string, () => void>()
  let entries = 1
  const history = {
    state: { position: 3 },
    pushState: vi.fn(() => void entries++),
    back: vi.fn(),
  }
  vi.stubGlobal('window', {
    history,
    addEventListener: (type: string, fn: () => void) => listeners.set(type, fn),
    removeEventListener: (type: string) => listeners.delete(type),
  })
  // The browser pops the entry, then reports it
  const goBack = () => {
    entries--
    listeners.get('popstate')?.()
  }
  history.back.mockImplementation(() => queueMicrotask(goBack))
  return { history, goBack, entries: () => entries }
}

describe('layer stack and the system Back', () => {
  beforeEach(() => vi.resetModules())
  afterEach(() => vi.unstubAllGlobals())

  it('holds one entry for the whole stack and closes one layer per Back', async () => {
    const w = setup()
    const { pushEscapeLayer } = await import('./useEscapeToClose')
    let removeUnder = () => {}
    let removeOver = () => {}
    const under = vi.fn(() => removeUnder())
    const over = vi.fn(() => removeOver())
    removeUnder = pushEscapeLayer(under)
    removeOver = pushEscapeLayer(over)
    expect(w.history.pushState).toHaveBeenCalledTimes(1)

    w.goBack()
    await nextTick()
    await nextTick()
    expect(over).toHaveBeenCalledTimes(1)
    expect(under).not.toHaveBeenCalled()
    // One layer remains: the entry is held again
    expect(w.history.pushState).toHaveBeenCalledTimes(2)

    w.goBack()
    await nextTick()
    await nextTick()
    expect(under).toHaveBeenCalledTimes(1)
    expect(w.history.pushState).toHaveBeenCalledTimes(2)
  })

  it('pops its entry when the last layer closes another way', async () => {
    const w = setup()
    const { pushEscapeLayer } = await import('./useEscapeToClose')
    const remove = pushEscapeLayer(() => {})
    remove()
    expect(w.history.back).toHaveBeenCalledTimes(1)
  })

  it('holds the entry again when the layer refuses to close', async () => {
    const w = setup()
    const { pushEscapeLayer } = await import('./useEscapeToClose')
    pushEscapeLayer(() => false)
    w.goBack()
    await nextTick()
    await nextTick()
    expect(w.history.pushState).toHaveBeenCalledTimes(2)
  })

  it('does not hold an entry for a layer that does not take part in Back', async () => {
    const w = setup()
    const { pushEscapeLayer, backEntryHeld } = await import('./useEscapeToClose')
    pushEscapeLayer(() => {}, { back: false })
    expect(w.history.pushState).not.toHaveBeenCalled()
    expect(backEntryHeld()).toBe(false)
  })

  it('closes the top layer that takes part, and leaves Escape to the layers that use it', async () => {
    const w = setup()
    const { pushEscapeLayer, hasOpenEscapeLayer, closeTopLayerOnEscape } = await import('./useEscapeToClose')
    const modal = vi.fn()
    const selection = vi.fn()
    pushEscapeLayer(modal)
    pushEscapeLayer(selection, { back: false, escape: false })
    expect(hasOpenEscapeLayer()).toBe(true)
    w.goBack()
    expect(modal).toHaveBeenCalledTimes(1)
    expect(selection).not.toHaveBeenCalled()
    const e = { key: 'Escape', defaultPrevented: false, isComposing: false, preventDefault: vi.fn() }
    expect(closeTopLayerOnEscape(e)).toBe(true)
    expect(modal).toHaveBeenCalledTimes(2)
    expect(selection).not.toHaveBeenCalled()
  })

  it('tells when the held entry was replaced by the page itself', async () => {
    setup()
    const { pushEscapeLayer, backEntryHeld, releaseBackEntry } = await import('./useEscapeToClose')
    pushEscapeLayer(() => {})
    expect(backEntryHeld()).toBe(true)
    releaseBackEntry()
    expect(backEntryHeld()).toBe(false)
  })

  it('does nothing without a window', async () => {
    vi.stubGlobal('window', undefined)
    const { pushEscapeLayer, backEntryHeld } = await import('./useEscapeToClose')
    expect(() => pushEscapeLayer(() => {})()).not.toThrow()
    expect(backEntryHeld()).toBe(false)
  })
})
