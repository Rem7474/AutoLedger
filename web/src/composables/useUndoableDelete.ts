import type { Ref } from 'vue'
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

export interface UndoableListDelete<T> {
  /** The list the rows are shown from: the rows leave it at once and come back at their place. */
  list: Ref<T[]>
  items: T[]
  /** What tells one row from another. */
  key: (item: T) => string | number
  message: string
  /** The deletion of one row, asked of the server one after the other once the time to take it back is over. */
  remove: (item: T) => Promise<unknown>
  /** A count shown elsewhere follows the list: called with the number of rows that left it, and with the number that came back. */
  onHide?: (count: number) => void
  onRestore?: (count: number) => void
  /** The server has deleted every row. */
  committed?: () => void
  /** The server refused or could not be reached for one row: the rows it did not delete are back on the screen. */
  onError: (err: unknown) => void
}

/** `deleteWithUndo` for rows of a list held in a ref (one row or a selection, a single snackbar for all of them). */
export function deleteRowsWithUndo<T>(d: UndoableListDelete<T>): void {
  const keys = new Set(d.items.map(d.key))
  const places = d.items.map((item) => ({ item, index: d.list.value.findIndex((row) => d.key(row) === d.key(item)) })).sort((x, y) => x.index - y.index)
  const deleted = new Set<string | number>()
  deleteWithUndo({
    message: d.message,
    hide: () => {
      const left = d.list.value.filter((row) => !keys.has(d.key(row)))
      d.onHide?.(d.list.value.length - left.length)
      d.list.value = left
    },
    restore: () => {
      // Each row goes back to the place it had, among the rows that are on the screen now
      const next = [...d.list.value]
      for (const { item, index } of places) {
        if (deleted.has(d.key(item)) || next.some((row) => d.key(row) === d.key(item))) continue
        next.splice(Math.min(Math.max(index, 0), next.length), 0, item)
      }
      d.onRestore?.(next.length - d.list.value.length)
      d.list.value = next
    },
    commit: async () => {
      for (const item of d.items) {
        await d.remove(item)
        deleted.add(d.key(item))
      }
    },
    committed: d.committed,
    onError: d.onError,
  })
}
