import { describe, expect, it, vi } from 'vitest'

vi.mock('@/i18n', () => ({ t: (key: string) => key }))

import { barActions, editDeleteActions, phoneShortcuts, sheetOrder, type SelectionAction } from './selectionActions'

const action = (key: string, overflow = false): SelectionAction => ({ key, label: key, icon: {} as never, run: () => {}, overflow })

describe('selection actions', () => {
  it('puts the destructive actions last in the sheet, the others keeping their order', () => {
    const all = [{ ...action('delete'), danger: true }, action('a'), action('b', true)]
    expect(sheetOrder(all).map((a) => a.key)).toEqual(['a', 'b', 'delete'])
  })

  it('keeps the overflow actions out of the bar', () => {
    const all = [action('a'), action('b', true), action('c')]
    expect(barActions(all).map((a) => a.key)).toEqual(['a', 'c'])
  })

  it('keeps the phone-only actions out of the bar and its shortcuts', () => {
    const all = [{ ...action('select'), phoneOnly: true }, action('a'), action('b')]
    expect(barActions(all).map((x) => x.key)).toEqual(['a', 'b'])
    expect(phoneShortcuts(all).map((x) => x.key)).toEqual(['a', 'b'])
  })

  it('shows at most three shortcuts on a phone', () => {
    const all = ['a', 'b', 'c', 'd'].map((k) => action(k))
    expect(phoneShortcuts(all).map((a) => a.key)).toEqual(['a', 'b', 'c'])
    expect(phoneShortcuts(all, 2)).toHaveLength(2)
  })

  describe('editDeleteActions', () => {
    const edit = vi.fn()
    const remove = vi.fn()

    it('edits one row and deletes it', () => {
      const [e, d] = editDeleteActions(['x'], { edit, remove })
      expect(e.disabled).toBe(false)
      expect(d.disabled).toBe(false)
      e.run()
      d.run()
      expect(edit).toHaveBeenCalledWith('x')
      expect(remove).toHaveBeenCalledWith(['x'])
    })

    it('greys the edit of several rows and says why, but still deletes them', () => {
      const [e, d] = editDeleteActions(['x', 'y'], { edit, remove })
      expect(e).toMatchObject({ disabled: true, reason: 'shell.bulkSelectionBar.selectOne' })
      expect(d.disabled).toBe(false)
    })

    it('greys the delete when a row cannot be deleted, with the given reason', () => {
      const [d] = editDeleteActions([1, 2], { remove, deletable: (n) => n === 1, lockedReason: 'synced' })
      expect(d).toMatchObject({ disabled: true, reason: 'synced' })
    })

    it('falls back to a generic reason', () => {
      const [d] = editDeleteActions([1], { remove, deletable: () => false })
      expect(d.reason).toBe('shell.bulkSelectionBar.cannotDelete')
    })

    it('leaves out the actions it was not given', () => {
      expect(editDeleteActions(['x'], {})).toEqual([])
      expect(editDeleteActions(['x'], { remove }).map((a) => a.key)).toEqual(['delete'])
    })
  })
})
