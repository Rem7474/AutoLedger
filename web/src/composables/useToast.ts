import { ref } from 'vue'

export interface ToastAction {
  label: string
  run: () => void
}

export interface Toast {
  id: number
  message: string
  title?: string
  /** A button of the toast (Undo): pressing it also dismisses the toast. */
  action?: ToastAction
}

const TOAST_MS = 4000
const toasts = ref<Toast[]>([])
let nextId = 1

export function useToast() {
  function dismissToast(id: number) {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }

  /** `ms` is how long the toast stays: a toast with an action stays as long as the action is possible. */
  function showToast(message: string, title?: string, options: { action?: ToastAction; ms?: number } = {}): number {
    const id = nextId++
    toasts.value = [...toasts.value.slice(-2), { id, message, title, action: options.action }]
    setTimeout(() => dismissToast(id), options.ms ?? TOAST_MS)
    return id
  }

  return { toasts, showToast, dismissToast }
}
