import { computed, ref, type Ref } from 'vue'

export interface InfinitePage<T> {
  items: T[]
  total: number
}

/**
 * A list read page by page that grows as the reader scrolls: the pages already read stay in `items`, `loadMore`
 * appends the next one, and `load(true)` reads again every page that is shown (after an action, a sync or a pull)
 * so the list neither collapses nor jumps back to the first page.
 *
 * `fetchPage` reads one page (1-based). A change of filter calls `load()`, which starts again from the first page.
 */
export function useInfiniteList<T extends { id: string }>(fetchPage: (page: number) => Promise<InfinitePage<T>>) {
  const items = ref<T[]>([]) as Ref<T[]>
  const total = ref(0)
  const pages = ref(0)
  const loading = ref(false)
  const loadingMore = ref(false)
  const error = ref<string | null>(null)
  const moreError = ref<string | null>(null)
  // A read that is overtaken by a newer one (a filter changed meanwhile) is dropped
  let generation = 0

  const hasMore = computed(() => items.value.length < total.value && pages.value > 0)

  async function load(silent = false) {
    const mine = ++generation
    loadingMore.value = false
    if (!silent) loading.value = true
    error.value = null
    moreError.value = null
    try {
      const count = silent ? Math.max(1, pages.value) : 1
      const read = await Promise.all(Array.from({ length: count }, (_, i) => fetchPage(i + 1)))
      if (mine !== generation) return
      items.value = dedupe(read.flatMap((r) => r.items))
      total.value = read[read.length - 1].total
      pages.value = count
    } catch (err) {
      if (mine !== generation) return
      error.value = (err as Error)?.message ?? ''
    } finally {
      if (mine === generation) loading.value = false
    }
  }

  async function loadMore() {
    if (loading.value || loadingMore.value || !hasMore.value) return
    const mine = generation
    loadingMore.value = true
    moreError.value = null
    try {
      const res = await fetchPage(pages.value + 1)
      if (mine !== generation) return
      pages.value += 1
      items.value = dedupe([...items.value, ...res.items])
      total.value = res.total
    } catch (err) {
      if (mine !== generation) return
      moreError.value = (err as Error)?.message ?? ''
    } finally {
      if (mine === generation) loadingMore.value = false
    }
  }

  return { items, total, loading, loadingMore, error, moreError, hasMore, load, loadMore }
}

// A row added at the top while the reader scrolls shifts the pages: the same row can then close one page and open the next
function dedupe<T extends { id: string }>(list: T[]): T[] {
  const seen = new Set<string>()
  return list.filter((row) => (seen.has(row.id) ? false : (seen.add(row.id), true)))
}
