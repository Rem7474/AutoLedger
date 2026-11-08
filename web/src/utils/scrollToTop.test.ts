import { afterEach, describe, expect, it, vi } from 'vitest'
import { scrollShellToTop, scrollToTop } from './scrollToTop'

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

describe('scrollShellToTop', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('does nothing without a document', () => {
    vi.stubGlobal('document', undefined)
    expect(() => scrollShellToTop()).not.toThrow()
  })

  it('scrolls the shell container, jumping when reduced motion is requested', () => {
    const shell = { scrollTo: vi.fn() }
    const query = vi.fn().mockReturnValue(shell)
    vi.stubGlobal('document', { querySelector: query })
    vi.stubGlobal('window', { matchMedia: () => ({ matches: true }) })
    scrollShellToTop()
    expect(query).toHaveBeenCalledWith('.app-scroll')
    expect(shell.scrollTo).toHaveBeenCalledWith({ top: 0, behavior: 'auto' })
  })

  it('scrolls smoothly otherwise', () => {
    const shell = { scrollTo: vi.fn() }
    vi.stubGlobal('document', { querySelector: () => shell })
    vi.stubGlobal('window', { matchMedia: () => ({ matches: false }) })
    scrollShellToTop()
    expect(shell.scrollTo).toHaveBeenCalledWith({ top: 0, behavior: 'smooth' })
  })
})
