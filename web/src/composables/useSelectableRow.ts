import { computed, type Ref } from 'vue'
import { useIsMobile } from '@/composables/useIsMobile'
import { useLongPress } from '@/composables/useLongPress'

// A list row that opens on a tap. On a phone its checkbox is hidden at rest: a long press selects the row and starts
// the selection (`selecting`), where the boxes show and a tap toggles the row instead of opening it. On a desktop the
// box is always there.
export function useSelectableRow(
  { enabled, selecting, toggle, open }: { enabled: Ref<boolean>; selecting: Ref<boolean>; toggle: () => void; open: () => void },
) {
  const isMobile = useIsMobile()
  const showBox = computed(() => enabled.value && (!isMobile.value || selecting.value))
  const press = useLongPress(() => {
    if (enabled.value && isMobile.value) toggle()
  })
  function onRowClick() {
    if (isMobile.value && selecting.value) toggle()
    else open()
  }
  return { showBox, press, onRowClick }
}
