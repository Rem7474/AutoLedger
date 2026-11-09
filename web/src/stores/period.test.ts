import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

function fakeStorage(initial: Record<string, string> = {}) {
  const data = { ...initial }
  return {
    getItem: (k: string) => (k in data ? data[k] : null),
    setItem: (k: string, v: string) => {
      data[k] = v
    },
  }
}

describe('period store', () => {
  beforeEach(() => {
    vi.resetModules()
    setActivePinia(createPinia())
  })

  it('starts on the last twelve months', async () => {
    vi.stubGlobal('localStorage', fakeStorage())
    const { usePeriodStore } = await import('./period')
    expect(usePeriodStore().costs).toEqual({ kind: 'months', count: 12, end: null })
  })

  it('remembers a choice across reloads', async () => {
    const storage = fakeStorage()
    vi.stubGlobal('localStorage', storage)
    const { usePeriodStore } = await import('./period')
    usePeriodStore().setCosts({ kind: 'months', count: 3, end: '2026-08' })
    vi.resetModules()
    setActivePinia(createPinia())
    const again = await import('./period')
    expect(again.usePeriodStore().costs).toEqual({ kind: 'months', count: 3, end: '2026-08' })
  })

  it('takes over the choice the home selector kept before the period was shared', async () => {
    vi.stubGlobal('localStorage', fakeStorage({ teslacost_dashboard_range: '6M' }))
    const { usePeriodStore } = await import('./period')
    expect(usePeriodStore().costs).toEqual({ kind: 'months', count: 6, end: null })
  })

  it('ignores a stored value that is not a period', async () => {
    vi.stubGlobal('localStorage', fakeStorage({ teslacost_period_costs: '{"kind":"months","count":4,"end":null}' }))
    const { usePeriodStore } = await import('./period')
    expect(usePeriodStore().costs).toEqual({ kind: 'months', count: 12, end: null })
    vi.resetModules()
    setActivePinia(createPinia())
    vi.stubGlobal('localStorage', fakeStorage({ teslacost_period_costs: 'not json' }))
    const again = await import('./period')
    expect(again.usePeriodStore().costs).toEqual({ kind: 'months', count: 12, end: null })
  })

  it('keeps the choice for the visit when storage is blocked', async () => {
    vi.stubGlobal('localStorage', {
      getItem: () => null,
      setItem: () => {
        throw new Error('quota')
      },
    })
    const { usePeriodStore } = await import('./period')
    const store = usePeriodStore()
    store.setCosts({ kind: 'all' })
    expect(store.costs).toEqual({ kind: 'all' })
  })
})
