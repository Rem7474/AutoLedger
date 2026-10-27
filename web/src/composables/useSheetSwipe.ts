import { onScopeDispose, watch } from 'vue'
import type { Ref } from 'vue'
import { readGesture, releaseVelocity, sheetOffset, shouldDismiss } from '@/utils/sheetDrag'

const NO_DRAG = 'input, textarea, select, [contenteditable="true"], [data-no-sheet-drag]'
const SAMPLE_WINDOW_MS = 100
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
    let startX = 0
    let startY = 0
    let scrollTop = 0
    let mode: 'idle' | 'wait' | 'drag' | 'scroll' = 'idle'
    let samples: Array<{ y: number; t: number }> = []

    const isSheet = () => window.matchMedia(`(max-width: ${maxWidth}px)`).matches
    const reducedMotion = () => window.matchMedia('(prefers-reduced-motion: reduce)').matches

    function onStart(e: TouchEvent) {
      mode = 'idle'
      if (e.touches.length !== 1 || !isSheet()) return
      const target = e.target instanceof Element ? e.target : null
      if (target?.closest(NO_DRAG)) return
      startX = e.touches[0].clientX
      startY = e.touches[0].clientY
      scrollTop = scrollerUnder(e.target, el)?.scrollTop ?? 0
      samples = [{ y: startY, t: e.timeStamp }]
      mode = 'wait'
    }

    function onMove(e: TouchEvent) {
      if (mode === 'idle' || mode === 'scroll') return
      const touch = e.touches[0]
      const dx = touch.clientX - startX
      const dy = touch.clientY - startY
      if (mode === 'wait') {
        const read = readGesture(dx, dy, scrollTop)
        if (read === 'wait') return
        mode = read
        if (mode === 'scroll') return
        el.style.transition = 'none'
      }
      if (e.cancelable) e.preventDefault()
      samples.push({ y: touch.clientY, t: e.timeStamp })
      samples = samples.filter((s) => e.timeStamp - s.t <= SAMPLE_WINDOW_MS)
      el.style.transform = `translateY(${sheetOffset(dy)}px)`
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
      if (mode !== 'drag') {
        mode = 'idle'
        return
      }
      mode = 'idle'
      const touch = e.changedTouches[0]
      const dy = touch ? touch.clientY - startY : 0
      const height = el.getBoundingClientRect().height
      if (e.type !== 'touchcancel' && shouldDismiss(dy, releaseVelocity(samples), height)) {
        settle('translateY(100%)', () => {
          dismiss()
          el.style.transform = ''
        })
      } else {
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
