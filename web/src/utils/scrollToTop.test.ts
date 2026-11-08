import { describe, expect, it, vi } from 'vitest'
import { scrollToTop } from './scrollToTop'

describe('scrollToTop', () => {
  it('scrolls to the top smoothly', () => {
    const el = { scrollTo: vi.fn() }
    scrollToTop(el, false)
    expect(el.scrollTo).toHaveBeenCalledWith({ top: 0, behavior: 'smooth' })
  })

  it('jumps instead when the user asks for less motion', () => {
    const el = { scrollTo: vi.fn() }
    scrollToTop(el, true)
    expect(el.scrollTo).toHaveBeenCalledWith({ top: 0, behavior: 'auto' })
  })

  it('does nothing without an element', () => {
    expect(() => scrollToTop(null, false)).not.toThrow()
  })
})
