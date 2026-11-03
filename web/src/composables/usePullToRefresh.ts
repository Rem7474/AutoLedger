import { onBeforeUnmount, onMounted, reactive, shallowRef, toRefs, watch } from 'vue'
import type { Ref } from 'vue'
import { createPullController, type PullState } from '@/utils/pullController'

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

/**
 * Pull-to-refresh on a phone: dragging the page down from the top asks the page on screen to reload. Does nothing on a
 * wide window, while a sheet is open, or on a page that registered no handler (see `createPullController`).
 */
export function usePullToRefresh(scroller: Ref<HTMLElement | null>) {
  const state = reactive<PullState>({ offset: 0, armed: false, pulling: false, refreshing: false })
  let detach = () => {}

  function attach(el: HTMLElement) {
    const controller = createPullController({
      isPhone: () => window.matchMedia('(max-width: 639px)').matches,
      handler: () => handler.value,
      isBlocked: (target) => (target instanceof Element && !!target.closest(NO_PULL)) || !!document.querySelector('[role="dialog"]'),
      scrollTop: () => el.scrollTop,
      wait: (ms) => new Promise((resolve) => window.setTimeout(resolve, ms)),
      onChange: (next) => Object.assign(state, next),
    })

    const onStart = (e: TouchEvent) => controller.start(e.touches.length, e.touches[0]?.clientX ?? 0, e.touches[0]?.clientY ?? 0, e.target)
    const onMove = (e: TouchEvent) => {
      if (controller.move(e.touches[0].clientX, e.touches[0].clientY) && e.cancelable) e.preventDefault()
    }
    const onEnd = (e: TouchEvent) => void controller.end(e.type === 'touchcancel')

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

  return toRefs(state)
}
