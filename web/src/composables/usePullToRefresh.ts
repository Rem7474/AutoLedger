import { onBeforeUnmount, onMounted, reactive, toRefs, watch } from 'vue'
import type { Ref } from 'vue'
import { bindPullEvents } from '@/utils/pullBinding'
import { createPullController, type PullState } from '@/utils/pullController'
import { currentRefresh, registerRefresh, type Refresh } from '@/utils/refreshRegistry'

/** Registers what a pull down refreshes while the calling page is mounted. */
export function useRefreshHandler(refresh: Refresh) {
  let withdraw = () => {}
  onMounted(() => {
    withdraw = registerRefresh(refresh)
  })
  onBeforeUnmount(() => withdraw())
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
      handler: currentRefresh,
      isBlocked: (target) => (target instanceof Element && !!target.closest(NO_PULL)) || !!document.querySelector('[role="dialog"]'),
      scrollTop: () => el.scrollTop,
      wait: (ms) => new Promise((resolve) => window.setTimeout(resolve, ms)),
      onChange: (next) => Object.assign(state, next),
    })

    detach = bindPullEvents(el, controller)
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
