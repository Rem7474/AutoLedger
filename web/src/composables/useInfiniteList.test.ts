import { describe, expect, it, vi } from 'vitest'
import { useInfiniteList, type InfinitePage } from './useInfiniteList'

const row = (n: number) => ({ id: `r${n}` })
const rows = (from: number, to: number) => Array.from({ length: to - from + 1 }, (_, i) => row(from + i))
// 25 rows, 10 per page
const server = (all = 25, size = 10) =>
  vi.fn(async (page: number): Promise<InfinitePage<{ id: string }>> => ({ items: rows((page - 1) * size + 1, Math.min(all, page * size)), total: all }))

describe('useInfiniteList', () => {
  it('reads the first page and says there is more', async () => {
    const list = useInfiniteList(server())
    await list.load()
    expect(list.items.value).toHaveLength(10)
    expect(list.total.value).toBe(25)
    expect(list.hasMore.value).toBe(true)
    expect(list.loading.value).toBe(false)
  })

  it('has nothing more before the first read', () => {
    expect(useInfiniteList(server()).hasMore.value).toBe(false)
  })

  it('appends the next pages until the end', async () => {
    const list = useInfiniteList(server())
    await list.load()
    await list.loadMore()
    await list.loadMore()
    expect(list.items.value.map((r) => r.id)).toEqual(rows(1, 25).map((r) => r.id))
    expect(list.hasMore.value).toBe(false)
    await list.loadMore()
    expect(list.items.value).toHaveLength(25)
  })

  it('does not read a second page while one is on its way', async () => {
    const fetchPage = server()
    const list = useInfiniteList(fetchPage)
    await list.load()
    const first = list.loadMore()
    expect(list.loadingMore.value).toBe(true)
    await list.loadMore()
    await first
    expect(fetchPage).toHaveBeenCalledTimes(2)
  })

  it('reads again every page shown on a silent load, without the loading state', async () => {
    const fetchPage = server()
    const list = useInfiniteList(fetchPage)
    await list.load()
    await list.loadMore()
    fetchPage.mockClear()
    const reload = list.load(true)
    expect(list.loading.value).toBe(false)
    await reload
    expect(fetchPage.mock.calls.map((c) => c[0])).toEqual([1, 2])
    expect(list.items.value).toHaveLength(20)
  })

  it('starts again from the first page on a plain load', async () => {
    const fetchPage = server()
    const list = useInfiniteList(fetchPage)
    await list.load()
    await list.loadMore()
    fetchPage.mockClear()
    const reload = list.load()
    expect(list.loading.value).toBe(true)
    await reload
    expect(fetchPage).toHaveBeenCalledTimes(1)
    expect(list.items.value).toHaveLength(10)
  })

  it('drops the answer of a read overtaken by a newer one', async () => {
    let answerOld: (p: InfinitePage<{ id: string }>) => void = () => {}
    const fetchPage = vi
      .fn()
      .mockReturnValueOnce(new Promise((resolve) => { answerOld = resolve }))
      .mockResolvedValueOnce({ items: [row(99)], total: 1 })
    const list = useInfiniteList(fetchPage)
    const old = list.load()
    await list.load()
    answerOld({ items: rows(1, 10), total: 25 })
    await old
    expect(list.items.value.map((r) => r.id)).toEqual(['r99'])
    expect(list.total.value).toBe(1)
  })

  it('drops the next page of a list that was filtered meanwhile', async () => {
    let answerMore: (p: InfinitePage<{ id: string }>) => void = () => {}
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows(1, 10), total: 25 })
      .mockReturnValueOnce(new Promise((resolve) => { answerMore = resolve }))
      .mockResolvedValueOnce({ items: [row(99)], total: 1 })
    const list = useInfiniteList(fetchPage)
    await list.load()
    const more = list.loadMore()
    await list.load()
    answerMore({ items: rows(11, 20), total: 25 })
    await more
    expect(list.items.value.map((r) => r.id)).toEqual(['r99'])
    expect(list.loadingMore.value).toBe(false)
  })

  it('reports a failed first read, and a failed next page apart so the rows stay', async () => {
    const fetchPage = vi
      .fn()
      .mockRejectedValueOnce(new Error('down'))
      .mockResolvedValueOnce({ items: rows(1, 10), total: 25 })
      .mockRejectedValueOnce(new Error('slow'))
    const list = useInfiniteList(fetchPage)
    await list.load()
    expect(list.error.value).toBe('down')
    await list.load()
    expect(list.error.value).toBeNull()
    await list.loadMore()
    expect(list.moreError.value).toBe('slow')
    expect(list.items.value).toHaveLength(10)
    expect(list.loadingMore.value).toBe(false)
  })

  it('keeps a row once when a shifted page repeats it', async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows(1, 3), total: 6 })
      .mockResolvedValueOnce({ items: rows(3, 5), total: 6 })
    const list = useInfiniteList(fetchPage)
    await list.load()
    await list.loadMore()
    expect(list.items.value.map((r) => r.id)).toEqual(['r1', 'r2', 'r3', 'r4', 'r5'])
  })
})
