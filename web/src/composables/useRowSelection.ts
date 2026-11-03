import { computed, ref, watch, type Ref } from 'vue'
import { useSelectableRow } from '@/composables/useSelectableRow'
import { markSelectionHintSeen } from '@/composables/useSelectionHint'

// The selection of a list whose rows all have an `id`: a long press on a phone starts it (the boxes then show and a
// tap toggles the row instead of opening it), `selected` feeds the actions of the contextual bar. A row that
// disappears from the list (deleted, filtered out) leaves the selection.
export function useRowSelection<T extends { id: string }>(items: Ref<T[]>, enabled: Ref<boolean>, open: (item: T) => void) {
  const selectedIds = ref<string[]>([])
  const selected = computed(() => items.value.filter((i) => selectedIds.value.includes(i.id)))
  const selecting = computed(() => selectedIds.value.length > 0)

  function toggle(item: T) {
    selectedIds.value = selectedIds.value.includes(item.id) ? selectedIds.value.filter((id) => id !== item.id) : [...selectedIds.value, item.id]
  }
  const isSelected = (item: T) => selectedIds.value.includes(item.id)
  const clear = () => {
    selectedIds.value = []
  }
  watch(selecting, (on) => {
    if (on) markSelectionHintSeen()
  })
  watch(items, (list) => {
    const kept = selectedIds.value.filter((id) => list.some((i) => i.id === id))
    if (kept.length !== selectedIds.value.length) selectedIds.value = kept
  })

  const row = useSelectableRow<T>({ enabled, selecting, toggle, open })
  return { selectedIds, selected, selecting, toggle, isSelected, clear, ...row }
}
