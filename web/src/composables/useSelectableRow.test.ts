import { describe, expect, it, vi, beforeEach } from 'vitest'
import { ref } from 'vue'

const mobile = ref(true)
vi.mock('@/composables/useIsMobile', () => ({ useIsMobile: () => mobile }))

import { useSelectableRow } from './useSelectableRow'

function setup(selecting: boolean, enabled = true) {
  const toggle = vi.fn()
  const open = vi.fn()
  const row = useSelectableRow({ enabled: ref(enabled), selecting: ref(selecting), toggle, open })
  return { row, toggle, open }
}

describe('useSelectableRow', () => {
  beforeEach(() => {
    mobile.value = true
  })

  it('hides the box at rest on a phone and shows it once a selection started', () => {
    expect(setup(false).row.showBox.value).toBe(false)
    expect(setup(true).row.showBox.value).toBe(true)
  })

  it('always shows the box on a desktop, but never to someone who cannot edit', () => {
    mobile.value = false
    expect(setup(false).row.showBox.value).toBe(true)
    expect(setup(false, false).row.showBox.value).toBe(false)
  })

  it('opens the row on a tap, and toggles it while selecting on a phone', () => {
    const rest = setup(false)
    rest.row.onRowClick()
    expect(rest.open).toHaveBeenCalledTimes(1)
    expect(rest.toggle).not.toHaveBeenCalled()

    const selecting = setup(true)
    selecting.row.onRowClick()
    expect(selecting.toggle).toHaveBeenCalledTimes(1)
    expect(selecting.open).not.toHaveBeenCalled()
  })

  it('opens the row on a desktop even while selecting', () => {
    mobile.value = false
    const s = setup(true)
    s.row.onRowClick()
    expect(s.open).toHaveBeenCalledTimes(1)
  })
})
