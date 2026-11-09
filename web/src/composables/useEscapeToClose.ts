import { getCurrentScope, nextTick, onScopeDispose, watch } from 'vue'
import type { Ref } from 'vue'
import { browserBackEnv, createBackGuard } from '@/composables/useBackToClose'

// Escape and the system Back close the layer opened last: a confirmation over a modal, or a document preview over a
// form, does not close what is under it. Every modal, sheet and preview registers here while it is open.
//
// Back: while at least one layer takes part in it, one history entry is held for the whole stack. Back pops it and
// closes the top layer that takes part; if layers remain, the entry is held again. A layer may refuse (`close`
// returns false, e.g. a save in progress): the entry is then held again and the layer stays.
interface Layer {
  close: () => void | boolean
  /** Takes part in Back (a phone-only mode says no on a desktop). Evaluated when the stack changes. */
  back: () => boolean
  /** Closed by Escape. A layer that handles its own Escape key sets false. */
  escape: boolean
}
export interface LayerOptions {
  back?: boolean | (() => boolean)
  escape?: boolean
}

const layers: Layer[] = []
let listening = false
let guard: ReturnType<typeof createBackGuard> | null = null

const asLayer = (close: () => void | boolean, o: LayerOptions = {}): Layer => {
  const back = o.back ?? true
  return { close, back: typeof back === 'function' ? back : () => back, escape: o.escape ?? true }
}

/** True while a modal, sheet or preview is open: Escape then belongs to it. */
export const hasOpenEscapeLayer = () => layers.some((l) => l.escape)

/** Whether the guard holds a history entry right now. */
export const backEntryHeld = () => guard?.held() ?? false
/** The page navigated on its own while the entry was held (the entry was replaced): forget it without popping. */
export const releaseBackEntry = () => guard?.release()

function closeTopForBack(): boolean {
  for (let i = layers.length - 1; i >= 0; i--) {
    if (layers[i].back()) {
      layers[i].close()
      break
    }
  }
  // The stack settles once the layer is gone from it; the entry is held again if layers remain or the top one refused
  void nextTick().then(syncBack)
  return true
}

function syncBack() {
  if (typeof window === 'undefined') return
  guard ??= createBackGuard(browserBackEnv, closeTopForBack)
  guard.set(layers.some((l) => l.back()))
}

export function closeTopLayerOnEscape(e: Pick<KeyboardEvent, 'key' | 'defaultPrevented' | 'isComposing' | 'preventDefault'>): boolean {
  if (e.key !== 'Escape' || e.defaultPrevented || e.isComposing) return false
  const top = [...layers].reverse().find((l) => l.escape)
  if (!top) return false
  e.preventDefault()
  top.close()
  return true
}

function listen() {
  if (listening || typeof window === 'undefined') return
  listening = true
  window.addEventListener('keydown', closeTopLayerOnEscape)
}

function add(layer: Layer) {
  listen()
  layers.push(layer)
  syncBack()
}

function remove(layer: Layer) {
  const i = layers.indexOf(layer)
  if (i < 0) return
  layers.splice(i, 1)
  syncBack()
}

/** Registers a layer closed by Escape and Back until the returned function is called. */
export function pushEscapeLayer(close: () => void, options?: LayerOptions): () => void {
  const layer = asLayer(close, options)
  add(layer)
  return () => remove(layer)
}

/** Closes the layer with `close` on Escape and Back while `isOpen` is true (a ref or a getter), if no layer opened after it is still open. */
export function useEscapeToClose(isOpen: Ref<boolean> | (() => boolean), close: () => void | boolean, options?: LayerOptions) {
  const layer = asLayer(close, options)
  watch(
    isOpen,
    (open) => {
      if (open) {
        if (!layers.includes(layer)) add(layer)
      } else {
        remove(layer)
      }
    },
    { immediate: true },
  )
  if (getCurrentScope()) onScopeDispose(() => remove(layer))
}
