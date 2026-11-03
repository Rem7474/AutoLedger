import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'

vi.mock('@/composables/useIsMobile', () => ({ useIsMobile: () => ref(true) }))

import { useRowSelection } from './useRowSelection'

const rows = () => ref([{ id: 'a' }, { id: 'b' }, { id: 'c' }])

describe('useRowSelection', () => {
  it('toggles rows and exposes the selected items in list order', () => {
    const items = rows()
    const sel = useRowSelection(items, ref(true), vi.fn())
    sel.toggle(items.value[2])
    sel.toggle(items.value[0])
    expect(sel.selected.value.map((i) => i.id)).toEqual(['a', 'c'])
    expect(sel.selecting.value).toBe(true)
    sel.toggle(items.value[0])
    expect(sel.isSelected(items.value[0])).toBe(false)
    sel.clear()
    expect(sel.selecting.value).toBe(false)
  })

  it('opens a row on a tap at rest and toggles it once a selection is running', () => {
    const items = rows()
    const open = vi.fn()
    const sel = useRowSelection(items, ref(true), open)
    sel.onRowClick(items.value[0])
    expect(open).toHaveBeenCalledWith(items.value[0])
    sel.toggle(items.value[1])
    sel.onRowClick(items.value[0])
    expect(open).toHaveBeenCalledTimes(1)
    expect(sel.selected.value.map((i) => i.id)).toEqual(['b', 'a'].sort())
  })

  it('drops from the selection the rows that left the list', async () => {
    const items = rows()
    const sel = useRowSelection(items, ref(true), vi.fn())
    sel.toggle(items.value[0])
    sel.toggle(items.value[1])
    items.value = [items.value[1], items.value[2]]
    await nextTick()
    expect(sel.selectedIds.value).toEqual(['b'])
  })
})
