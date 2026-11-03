import { onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import type { Ref } from 'vue'
import { createPullGesture, PULL_THRESHOLD } from '@/utils/pullRefresh'

type Refresh = () => unknown | Promise<unknown>

// The page on screen says how it reloads its data; the pull of the shell calls it.
const handler = shallowRef<Refresh | null>(null)

/** Registers what a pull down refreshes while the calling page is mounted. */
export function useRefreshHandler(refresh: Refresh) {
  onMounted(() => {
    handler.value = refresh
  })
  onBeforeUnmount(() => {
    if (handler.value === refresh) handler.value = null
  })
}

const NO_PULL = 'input, textarea, select, [contenteditable="true"], [data-no-pull]'
const SPINNER_MS = 450

/**
 * Pull-to-refresh on a phone: dragging the page down from the top asks the page on screen to reload. Does nothing on a
 * wide window, while a sheet is open, or on a page that registered no handler.
 */
export function usePullToRefresh(scroller: Ref<HTMLElement | null>) {
  const offset = ref(0)
  const armed = ref(false)
  const pulling = ref(false)
  const refreshing = ref(false)
  let detach = () => {}

  function attach(el: HTMLElement) {
    const gesture = createPullGesture()
    const isPhone = () => window.matchMedia('(max-width: 639px)').matches

    function onStart(e: TouchEvent) {
      const target = e.target instanceof Element ? e.target : null
      const blocked = e.touches.length !== 1 || !isPhone() || !handler.value || refreshing.value || target?.closest(NO_PULL) || document.querySelector('[role="dialog"]')
      if (blocked) return gesture.ignore()
      gesture.start(e.touches[0].clientX, e.touches[0].clientY, el.scrollTop)
    }

    function onMove(e: TouchEvent) {
      const move = gesture.move(e.touches[0].clientX, e.touches[0].clientY)
      if (!move) return
      if (e.cancelable) e.preventDefault()
      pulling.value = true
      offset.value = move.offset
      armed.value = move.armed
    }

    async function onEnd(e: TouchEvent) {
      const end = gesture.end(e.type === 'touchcancel')
      pulling.value = false
      armed.value = false
      if (end !== 'refresh' || !handler.value) {
        offset.value = 0
        return
      }
      refreshing.value = true
      offset.value = PULL_THRESHOLD * 0.75
      try {
        await Promise.all([Promise.resolve(handler.value()), new Promise((resolve) => window.setTimeout(resolve, SPINNER_MS))])
      } catch {
        // The page shows its own error
      } finally {
        refreshing.value = false
        offset.value = 0
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

  onMounted(() => {
    watch(scroller, (el) => {
      detach()
      detach = () => {}
      if (el) attach(el)
    }, { immediate: true })
  })
  onBeforeUnmount(() => detach())

  return { offset, armed, pulling, refreshing }
}
