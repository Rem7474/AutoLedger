// A press held on a touch screen: calls `onLongPress` once the finger has stayed put for `delay` ms. Mouse and pen
// presses never count (a desktop has its checkboxes). The click that follows a long press is swallowed wherever it
// lands (the long press often changes the layout under the finger), so nothing opens or toggles by accident. Bind the
// returned handlers on the pressable element.
export function useLongPress(onLongPress: () => void, { delay = 450, tolerance = 10 } = {}) {
  let timer: ReturnType<typeof setTimeout> | null = null
  let startX = 0
  let startY = 0
  let fired = false
  const SWALLOW_MS = 600

  function cancel() {
    if (timer) clearTimeout(timer)
    timer = null
  }
  function swallowNextClick() {
    if (typeof document === 'undefined') return
    const stop = (e: Event) => {
      e.stopPropagation()
      e.preventDefault()
      done()
    }
    const done = () => {
      clearTimeout(expiry)
      document.removeEventListener('click', stop, true)
    }
    const expiry = setTimeout(done, SWALLOW_MS)
    document.addEventListener('click', stop, true)
  }
  function onPointerdown(e: PointerEvent) {
    if (e.pointerType !== 'touch') return
    fired = false
    startX = e.clientX
    startY = e.clientY
    cancel()
    timer = setTimeout(() => {
      timer = null
      fired = true
      if (typeof navigator !== 'undefined' && typeof navigator.vibrate === 'function') navigator.vibrate(12)
      swallowNextClick()
      onLongPress()
    }, delay)
  }
  function onPointermove(e: PointerEvent) {
    if (timer && Math.hypot(e.clientX - startX, e.clientY - startY) > tolerance) cancel()
  }
  // The browser's own long-press menu (copy, share…) would cover the selection it just started
  function onContextmenu(e: Event) {
    if (fired || timer) e.preventDefault()
  }
  return { onPointerdown, onPointermove, onPointerup: cancel, onPointercancel: cancel, onContextmenu }
}
