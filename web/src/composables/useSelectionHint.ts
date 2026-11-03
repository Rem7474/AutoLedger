import { ref } from 'vue'

// Whether the long-press hint has been seen: shared by every list so a selection made anywhere hides it everywhere.
const KEY = 'autoledger.selectionHint'
const seen = ref(true)
let loaded = false

export function useSelectionHint() {
  if (!loaded) {
    loaded = true
    try {
      seen.value = localStorage.getItem(KEY) === '1'
    } catch {
      seen.value = false
    }
  }
  return { seen, dismiss: markSelectionHintSeen }
}

export function markSelectionHintSeen() {
  seen.value = true
  try {
    localStorage.setItem(KEY, '1')
  } catch {
    // private mode: the hint just comes back next time
  }
}
