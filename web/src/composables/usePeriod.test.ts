import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { effectScope, nextTick, reactive, type EffectScope } from 'vue'

// The page's route and router are stand-ins: `replace` applies the query the way the real router does
const route = reactive<{ path: string; query: Record<string, string> }>({ path: '/', query: {} })
const replace = vi.fn(async ({ query }: { query: Record<string, string> }) => {
  route.query = query
})
vi.mock('vue-router', () => ({ useRoute: () => route, useRouter: () => ({ replace }) }))

// Each test runs its composable in a scope that is stopped afterwards, like a page that is left
let scope: EffectScope
async function load() {
  vi.resetModules()
  setActivePinia(createPinia())
  const { usePeriod: real } = await import('./usePeriod')
  const usePeriod = () => scope.run(real)!
  const { usePeriodStore } = await import('@/stores/period')
  return { usePeriod, store: usePeriodStore() }
}

describe('usePeriod', () => {
  beforeEach(() => {
    vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => undefined })
    scope = effectScope()
    route.path = '/'
    route.query = {}
    replace.mockClear()
  })
  afterEach(() => scope.stop())

  it('writes the stored period in the URL, keeping the other parameters', async () => {
    const { usePeriod } = await load()
    route.query = { tab: 'charges' }
    usePeriod()
    await nextTick()
    expect(replace).toHaveBeenCalledWith({ query: { tab: 'charges', period: '12M' } })
  })

  it('adopts the period a URL names on arrival and leaves the URL alone', async () => {
    const { usePeriod, store } = await load()
    route.query = { period: '3M', end: '2026-08' }
    const p = usePeriod()
    await nextTick()
    expect(store.costs).toEqual({ kind: 'months', count: 3, end: '2026-08' })
    expect(p.period.value).toEqual(store.costs)
    expect(replace).not.toHaveBeenCalled()
  })

  it('ignores a URL that names no valid period', async () => {
    const { usePeriod, store } = await load()
    route.query = { period: 'nonsense' }
    usePeriod()
    await nextTick()
    expect(store.costs).toEqual({ kind: 'months', count: 12, end: null })
    expect(replace).toHaveBeenCalledWith({ query: { period: '12M' } })
  })

  it('follows a step and a pick in the URL', async () => {
    const { usePeriod } = await load()
    const p = usePeriod()
    await nextTick()
    expect(p.canNext.value).toBe(false)
    expect(p.canPrevious.value).toBe(true)
    p.step(-1)
    await nextTick()
    expect(route.query.period).toBe('12M')
    expect(route.query.end).toMatch(/^\d{4}-\d{2}$/)
    expect(p.canNext.value).toBe(true)
    p.set({ kind: 'custom', from: '2026-01-10', to: '2026-02-20' })
    await nextTick()
    expect(route.query).toEqual({ from: '2026-01-10', to: '2026-02-20' })
    expect(p.rounded.value).toBe(true)
    expect(p.bounds.value).toEqual({ from: '2026-01-10', to: '2026-02-20' })
    expect(p.label.value).not.toBe('')
  })

  it('puts the URL right when something else rewrote it', async () => {
    const { usePeriod } = await load()
    usePeriod()
    await nextTick()
    replace.mockClear()
    route.query = {}
    await nextTick()
    expect(replace).toHaveBeenCalledWith({ query: { period: '12M' } })
  })

  it('leaves the URL of another page alone once the page has been left', async () => {
    const { usePeriod } = await load()
    const p = usePeriod()
    await nextTick()
    replace.mockClear()
    route.path = '/drives'
    route.query = {}
    p.step(-1)
    await nextTick()
    expect(replace).not.toHaveBeenCalled()
  })
})
