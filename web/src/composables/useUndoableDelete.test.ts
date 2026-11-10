import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { t } from '@/i18n'
import { useToast } from './useToast'
import { ref } from 'vue'
import { deleteRowsWithUndo, deleteWithUndo, flushPendingDeletes, UNDO_MS } from './useUndoableDelete'

const { toasts } = useToast()

function setup(commit: () => Promise<unknown> = async () => {}) {
  const calls = { hide: vi.fn(), restore: vi.fn(), commit: vi.fn(commit), committed: vi.fn(), onError: vi.fn() }
  deleteWithUndo({ message: 'Cost deleted', ...calls })
  return calls
}

describe('deleteWithUndo', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    toasts.value = []
  })
  afterEach(async () => {
    await flushPendingDeletes()
    vi.useRealTimers()
  })

  it('takes the row off at once and offers to take the deletion back, without calling the server yet', () => {
    const c = setup()
    expect(c.hide).toHaveBeenCalledTimes(1)
    expect(c.commit).not.toHaveBeenCalled()
    expect(toasts.value).toHaveLength(1)
    expect(toasts.value[0].message).toBe('Cost deleted')
    expect(toasts.value[0].action?.label).toBe(t('common.undo'))
  })

  it('deletes on the server once the time to undo is over, and removes the snackbar', async () => {
    const c = setup()
    await vi.advanceTimersByTimeAsync(UNDO_MS - 1)
    expect(c.commit).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(1)
    expect(c.commit).toHaveBeenCalledTimes(1)
    expect(c.committed).toHaveBeenCalledTimes(1)
    expect(c.restore).not.toHaveBeenCalled()
    expect(toasts.value).toHaveLength(0)
  })

  it('puts the row back and never calls the server when undone', async () => {
    const c = setup()
    toasts.value[0].action!.run()
    expect(c.restore).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(UNDO_MS * 2)
    expect(c.commit).not.toHaveBeenCalled()
    expect(c.committed).not.toHaveBeenCalled()
  })

  it('restores the row and reports the error when the server refuses', async () => {
    const error = new Error('boom')
    const c = setup(async () => {
      throw error
    })
    await vi.advanceTimersByTimeAsync(UNDO_MS)
    expect(c.restore).toHaveBeenCalledTimes(1)
    expect(c.onError).toHaveBeenCalledWith(error)
    expect(c.committed).not.toHaveBeenCalled()
  })

  it('sends the waiting deletions at once when the page is left, and only once', async () => {
    const a = setup()
    const b = setup()
    await flushPendingDeletes()
    expect(a.commit).toHaveBeenCalledTimes(1)
    expect(b.commit).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(UNDO_MS * 2)
    expect(a.commit).toHaveBeenCalledTimes(1)
    expect(toasts.value).toHaveLength(0)
  })

  it('has nothing to send when no deletion is waiting', async () => {
    await expect(flushPendingDeletes()).resolves.toEqual([])
  })
})

describe('deleteRowsWithUndo', () => {
  const rows = () => [{ id: 1 }, { id: 2 }, { id: 3 }, { id: 4 }]
  beforeEach(() => {
    vi.useFakeTimers()
    toasts.value = []
  })
  afterEach(async () => {
    await flushPendingDeletes()
    vi.useRealTimers()
  })

  function run(remove: (r: { id: number }) => Promise<unknown>, ids = [2, 4]) {
    const list = ref(rows())
    const onError = vi.fn()
    const committed = vi.fn()
    deleteRowsWithUndo({ list, items: list.value.filter((r) => ids.includes(r.id)), key: (r) => r.id, message: '2 deleted', remove, committed, onError })
    return { list, onError, committed }
  }

  it('takes the rows off at once under one snackbar, and asks the server once the time is over', async () => {
    const remove = vi.fn(async () => {})
    const { list, committed } = run(remove)
    expect(list.value.map((r) => r.id)).toEqual([1, 3])
    expect(toasts.value).toHaveLength(1)
    expect(remove).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(UNDO_MS)
    expect(remove).toHaveBeenCalledTimes(2)
    expect(committed).toHaveBeenCalledTimes(1)
    expect(list.value.map((r) => r.id)).toEqual([1, 3])
  })

  it('puts every row back at its place when undone, without calling the server', async () => {
    const remove = vi.fn(async () => {})
    const { list } = run(remove)
    toasts.value[0].action!.run()
    expect(list.value.map((r) => r.id)).toEqual([1, 2, 3, 4])
    await vi.advanceTimersByTimeAsync(UNDO_MS * 2)
    expect(remove).not.toHaveBeenCalled()
  })

  it('brings back only the rows the server did not delete when it refuses one', async () => {
    const boom = new Error('boom')
    const remove = vi.fn(async (r: { id: number }) => {
      if (r.id === 4) throw boom
    })
    const { list, onError } = run(remove)
    await vi.advanceTimersByTimeAsync(UNDO_MS)
    expect(list.value.map((r) => r.id)).toEqual([1, 3, 4])
    expect(onError).toHaveBeenCalledWith(boom)
  })
})
