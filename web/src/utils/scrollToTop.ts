// Touching the tab of the page already shown brings that page back to its top, like the tab bars of the system do.
// The shell scrolls inside `.app-scroll`, not in the window.

export interface Scrollable {
  scrollTo(options: ScrollToOptions): void
}

/** Scrolls `el` to its top, smoothly unless the user asks for less motion; does nothing without an element. */
export function scrollToTop(el: Scrollable | null, reducedMotion: boolean): void {
  el?.scrollTo({ top: 0, behavior: reducedMotion ? 'auto' : 'smooth' })
}

/** Brings the page shown in the app shell back to its top. */
export function scrollShellToTop(): void {
  if (typeof document === 'undefined') return
  scrollToTop(document.querySelector<HTMLElement>('.app-scroll'), window.matchMedia('(prefers-reduced-motion: reduce)').matches)
}
