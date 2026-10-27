import { onScopeDispose, watch } from 'vue'
import type { Ref } from 'vue'
import { createSheetGesture } from '@/utils/sheetDrag'

const NO_DRAG = 'input, textarea, select, [contenteditable="true"], [data-no-sheet-drag]'
const SETTLE_MS = 200

/** The scrolling element the finger is on, inside the panel: where a pull down scrolls rather than closes. */
function scrollerUnder(target: EventTarget | null, panel: HTMLElement): HTMLElement | null {
  let el = target instanceof HTMLElement ? target : null
  while (el && el !== panel.parentElement) {
    if (el.scrollHeight > el.clientHeight + 1 && /(auto|scroll)/.test(getComputedStyle(el).overflowY)) return el
    el = el.parentElement
  }
  return null
}

/**
 * Closes a bottom sheet when it is dragged down: from its handle and header, or from its content when that is scrolled
 * to the top. The sheet follows the finger and settles back unless it was pulled far or flicked. It does nothing above
 * `maxWidth` px, where the panel is a dialog and not a sheet.
 */
export function useSheetSwipe(panel: Ref<HTMLElement | null>, dismiss: () => void, maxWidth = 639) {
  let detach = () => {}

  function attach(el: HTMLElement) {
    const gesture = createSheetGesture()

    const isSheet = () => window.matchMedia(`(max-width: ${maxWidth}px)`).matches
    const reducedMotion = () => window.matchMedia('(prefers-reduced-motion: reduce)').matches

    function onStart(e: TouchEvent) {
      const target = e.target instanceof Element ? e.target : null
      if (e.touches.length !== 1 || !isSheet() || target?.closest(NO_DRAG)) return gesture.ignore()
      const touch = e.touches[0]
      gesture.start(touch.clientX, touch.clientY, e.timeStamp, scrollerUnder(e.target, el)?.scrollTop ?? 0)
    }

    function onMove(e: TouchEvent) {
      const touch = e.touches[0]
      const move = gesture.move(touch.clientX, touch.clientY, e.timeStamp)
      if (!move) return
      if (move.began) el.style.transition = 'none'
      if (e.cancelable) e.preventDefault()
      el.style.transform = `translateY(${move.offset}px)`
    }

    function settle(transform: string, then?: () => void) {
      if (reducedMotion()) {
        el.style.transition = ''
        el.style.transform = then ? transform : ''
        then?.()
        return
      }
      el.style.transition = `transform ${SETTLE_MS}ms ease-out`
      el.style.transform = transform
      window.setTimeout(() => {
        el.style.transition = ''
        if (!then) el.style.transform = ''
        then?.()
      }, SETTLE_MS)
    }

    function onEnd(e: TouchEvent) {
      const touch = e.changedTouches[0]
      const end = gesture.end(touch ? touch.clientY : 0, el.getBoundingClientRect().height, e.type === 'touchcancel')
      if (end === 'dismiss') {
        settle('translateY(100%)', () => {
          dismiss()
          el.style.transform = ''
        })
      } else if (end === 'settle') {
        settle('')
      }
    }

    el.addEventListener('touchstart', onStart, { passive: true })
    el.addEventListener('touchmove', onMove, { passive: false })
    el.addEventListener('touchend', onEnd)
    el.addEventListener('touchcancel', onEnd)
    detach = () => {
      el.removeEventListener('touchstart', onStart)
      el.removeEventListener('touchmove', onMove)
      el.removeEventListener('touchend', onEnd)
      el.removeEventListener('touchcancel', onEnd)
    }
  }

  watch(
    panel,
    (el) => {
      detach()
      detach = () => {}
      if (el) attach(el)
    },
    { flush: 'post', immediate: true },
  )
  onScopeDispose(() => detach())
}
