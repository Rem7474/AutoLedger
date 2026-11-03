import { t } from '@/i18n'
import { useToast } from '@/composables/useToast'

/** How long a deletion can be taken back before it is sent to the server. */
export const UNDO_MS = 6000

interface Pending {
  commit: () => Promise<unknown>
  timer: ReturnType<typeof setTimeout>
  toastId: number
}

const pending = new Set<Pending>()

export interface UndoableDelete {
  /** The text of the snackbar ("Cost deleted"). */
  message: string
  /** Takes the row off the screen at once. */
  hide: () => void
  /** Puts the row back: the deletion was taken back, or the server refused it. */
  restore: () => void
  /** The deletion itself, once the time to take it back is over. */
  commit: () => Promise<unknown>
  /** The server has deleted the row. */
  committed?: () => void
  /** The server refused or could not be reached: the row is already restored. */
  onError: (err: unknown) => void
}

/**
 * Deletes without asking first: the row goes at once, a snackbar offers to take it back, and the server is only asked
 * when that time is over (or when the page is left, see `flushPendingDeletes`).
 */
export function deleteWithUndo(d: UndoableDelete): void {
  const { showToast, dismissToast } = useToast()
  d.hide()

  const entry = {} as Pending
  async function settle() {
    pending.delete(entry)
    clearTimeout(entry.timer)
    dismissToast(entry.toastId)
    try {
      await d.commit()
    } catch (err) {
      d.restore()
      d.onError(err)
      return
    }
    d.committed?.()
  }
  function undo() {
    pending.delete(entry)
    clearTimeout(entry.timer)
    d.restore()
  }

  entry.commit = settle
  entry.timer = setTimeout(() => void settle(), UNDO_MS)
  entry.toastId = showToast(d.message, undefined, { ms: UNDO_MS, action: { label: t('common.undo'), run: undo } })
  pending.add(entry)
}

/** Sends every deletion still waiting: the page is being left, and a deletion shown as done must not be lost. */
export function flushPendingDeletes(): Promise<unknown[]> {
  return Promise.all([...pending].map((entry) => entry.commit()))
}

if (typeof document !== 'undefined') {
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden') void flushPendingDeletes()
  })
}
