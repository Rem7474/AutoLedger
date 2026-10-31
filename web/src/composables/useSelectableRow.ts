import { computed, type Ref } from 'vue'
import { useIsMobile } from '@/composables/useIsMobile'
import { useLongPress } from '@/composables/useLongPress'

// A list row that opens on a tap. On a phone its checkbox is hidden at rest: a long press selects the row and starts
// the selection (`selecting`), where the boxes show and a tap toggles the row instead of opening it. On a desktop the
// box is always there. `toggle` and `open` receive the item of the row, so one instance serves a whole list.
export function useSelectableRow<T = void>(
  { enabled, selecting, toggle, open }: { enabled: Ref<boolean>; selecting: Ref<boolean>; toggle: (item: T) => void; open: (item: T) => void },
) {
  const isMobile = useIsMobile()
  const showBox = computed(() => enabled.value && (!isMobile.value || selecting.value))
  let pressed: T | undefined
  const press = useLongPress(() => {
    if (enabled.value && isMobile.value) toggle(pressed as T)
  })
  function onPointerdown(e: PointerEvent, item?: T) {
    pressed = item
    press.onPointerdown(e)
  }
  function onRowClick(item?: T) {
    if (isMobile.value && selecting.value) toggle(item as T)
    else open(item as T)
  }
  return { showBox, press, onPointerdown, onRowClick }
}
