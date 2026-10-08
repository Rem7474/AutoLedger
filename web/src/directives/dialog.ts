import type { Directive } from 'vue'
import { pushEscapeLayer } from '@/composables/useEscapeToClose'
import { useToast } from '@/composables/useToast'
import { t } from '@/i18n'

// v-dialog goes on the panel of a modal: it makes the panel a labelled modal dialog, moves the focus into it,
// keeps Tab inside, and gives the focus back to the control that opened it. Pass a close function to also close
// on Escape (the modals that already call useEscapeToClose omit it). Once a field of the panel has been edited, a
// click on the backdrop no longer reaches the modal's own handler, so a stray click cannot discard the input; the
// close button, Cancel and Escape still close it.
type Close = (() => void) | undefined

const FOCUSABLE = 'a[href],button:not([disabled]),input:not([disabled]):not([type="hidden"]),select:not([disabled]),textarea:not([disabled]),[tabindex]:not([tabindex="-1"])'

interface State {
  opener: HTMLElement | null
  backdrop: HTMLElement | null
  dirty: boolean
  onEdit: () => void
  onBackdropClick: (e: MouseEvent) => void
  removeLayer: (() => void) | null
  onKeydown: (e: KeyboardEvent) => void
}

const states = new WeakMap<HTMLElement, State>()
let idCounter = 0

/** True when a click must be kept from closing the modal: it lands on the backdrop itself while the form holds edits. */
export function shouldKeepOpen(dirty: boolean, target: EventTarget | null, backdrop: EventTarget | null): boolean {
  return dirty && target !== null && target === backdrop
}

/** Index of the element to focus when Tab leaves the panel's ends; -1 when the browser can move the focus itself. */
export function wrapFocusIndex(count: number, current: number, backwards: boolean): number {
  if (count === 0) return -1
  if (current < 0) return backwards ? count - 1 : 0
  if (backwards && current === 0) return count - 1
  if (!backwards && current === count - 1) return 0
  return -1
}

function focusables(panel: HTMLElement): HTMLElement[] {
  return Array.from(panel.querySelectorAll<HTMLElement>(FOCUSABLE)).filter((el) => el.offsetParent !== null || el === document.activeElement)
}

function initialFocus(panel: HTMLElement) {
  const coarse = typeof window.matchMedia === 'function' && window.matchMedia('(pointer: coarse)').matches
  const explicit = panel.querySelector<HTMLElement>('[data-autofocus]')
  const field = coarse ? null : panel.querySelector<HTMLElement>('input:not([disabled]):not([type="hidden"]):not([type="checkbox"]):not([type="radio"]),select:not([disabled]),textarea:not([disabled])')
  const target = explicit ?? field
  if (target) target.focus({ preventScroll: true })
  else {
    panel.setAttribute('tabindex', '-1')
    panel.focus({ preventScroll: true })
  }
}

export const vDialog: Directive<HTMLElement, Close> = {
  mounted(el, binding) {
    el.setAttribute('role', 'dialog')
    el.setAttribute('aria-modal', 'true')
    if (!el.hasAttribute('aria-labelledby') && !el.hasAttribute('aria-label')) {
      const heading = el.querySelector<HTMLElement>('h1,h2,h3')
      if (heading) {
        if (!heading.id) heading.id = `dialog-title-${++idCounter}`
        el.setAttribute('aria-labelledby', heading.id)
      }
    }
    const state: State = {
      opener: document.activeElement instanceof HTMLElement ? document.activeElement : null,
      backdrop: el.parentElement,
      dirty: false,
      onEdit: () => {
        state.dirty = true
      },
      onBackdropClick: (e) => {
        if (!shouldKeepOpen(state.dirty, e.target, state.backdrop)) return
        e.stopImmediatePropagation()
        useToast().showToast(t('common.unsavedChanges'))
      },
      removeLayer: typeof binding.value === 'function' ? pushEscapeLayer(binding.value) : null,
      onKeydown: (e) => {
        if (e.key !== 'Tab' || e.defaultPrevented) return
        const items = focusables(el)
        const target = wrapFocusIndex(items.length, items.indexOf(document.activeElement as HTMLElement), e.shiftKey)
        if (target >= 0) {
          e.preventDefault()
          items[target].focus()
        } else if (items.length === 0) e.preventDefault()
      },
    }
    el.addEventListener('keydown', state.onKeydown)
    el.addEventListener('input', state.onEdit)
    el.addEventListener('change', state.onEdit)
    state.backdrop?.addEventListener('click', state.onBackdropClick, true)
    states.set(el, state)
    initialFocus(el)
  },
  unmounted(el) {
    const state = states.get(el)
    if (!state) return
    el.removeEventListener('keydown', state.onKeydown)
    el.removeEventListener('input', state.onEdit)
    el.removeEventListener('change', state.onEdit)
    state.backdrop?.removeEventListener('click', state.onBackdropClick, true)
    state.removeLayer?.()
    states.delete(el)
    if (state.opener?.isConnected) state.opener.focus({ preventScroll: true })
  },
}
