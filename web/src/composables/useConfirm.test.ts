import { describe, expect, it, vi } from 'vitest'

vi.mock('@/i18n', () => ({ t: (k: string) => k }))
vi.mock('@/composables/useToast', () => ({ useToast: () => ({ showToast: vi.fn() }) }))

import { useConfirm } from './useConfirm'

describe('useConfirm', () => {
  it('resolves a confirmation to true on confirm and false on cancel', async () => {
    const c = useConfirm()
    const yes = c.showConfirm('Sure?')
    expect(c.isOpen.value).toBe(true)
    c.onConfirm()
    expect(await yes).toBe(true)
    expect(c.isOpen.value).toBe(false)

    const no = c.showConfirm({ message: 'Sure?' })
    c.onCancel()
    expect(await no).toBe(false)
  })

  it('does not treat the alternative as a confirmation', async () => {
    const c = useConfirm()
    const p = c.showConfirm({ message: 'm', alternativeText: 'Other' })
    c.onAlternative()
    expect(await p).toBe(false)
  })

  it('tells which of the three buttons was pressed', async () => {
    const c = useConfirm()
    const opts = { message: 'm', confirmText: 'Delete', alternativeText: 'Scrap' }
    for (const [press, expected] of [
      [c.onConfirm, 'confirm'],
      [c.onAlternative, 'alternative'],
      [c.onCancel, 'cancel'],
    ] as const) {
      const p = c.showChoice(opts)
      expect(c.options.value.alternativeText).toBe('Scrap')
      press()
      expect(await p).toBe(expected)
    }
  })

  it('shows no alternative for a plain confirmation or an alert', async () => {
    const c = useConfirm()
    void c.showConfirm({ message: 'm' })
    expect(c.options.value.alternativeText).toBe('')
    c.onCancel()
    void c.showAlert('boom', undefined, 'danger')
    expect(c.options.value.alternativeText).toBe('')
    c.onConfirm()
  })
})
