import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useToast } from './useToast'

describe('useToast', () => {
  const { toasts, showToast, dismissToast } = useToast()
  beforeEach(() => {
    vi.useFakeTimers()
    toasts.value = []
  })
  afterEach(() => vi.useRealTimers())

  it('removes a toast after the default time', () => {
    showToast('Saved')
    expect(toasts.value.map((x) => x.message)).toEqual(['Saved'])
    vi.advanceTimersByTime(4000)
    expect(toasts.value).toHaveLength(0)
  })

  it('keeps a toast with its own duration and carries its action', () => {
    const run = vi.fn()
    const id = showToast('Deleted', undefined, { ms: 6000, action: { label: 'Undo', run } })
    vi.advanceTimersByTime(5999)
    expect(toasts.value[0].id).toBe(id)
    toasts.value[0].action!.run()
    expect(run).toHaveBeenCalledTimes(1)
    vi.advanceTimersByTime(1)
    expect(toasts.value).toHaveLength(0)
  })

  it('shows the last three toasts', () => {
    for (const m of ['a', 'b', 'c', 'd']) showToast(m)
    expect(toasts.value.map((x) => x.message)).toEqual(['b', 'c', 'd'])
  })

  it('dismisses one toast by its id', () => {
    const first = showToast('a')
    showToast('b')
    dismissToast(first)
    expect(toasts.value.map((x) => x.message)).toEqual(['b'])
  })
})
