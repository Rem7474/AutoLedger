import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { t } from '@/i18n'
import { useExpenseDeletions } from './useExpenseDeletions'
import { UNDO_MS } from './useUndoableDelete'
import { useToast } from './useToast'

const { toasts } = useToast()

const api = vi.hoisted(() => ({ deleteDriveExpense: vi.fn(), deleteMaintenance: vi.fn(), deleteCharge: vi.fn(), deleteDocument: vi.fn() }))
vi.mock('@/services/api', () => ({ api }))
const store = vi.hoisted(() => ({ activeVehicle: { id: 'v1' } as { id: string } | null, currency: 'EUR' }))
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => store }))
const confirm = vi.hoisted(() => ({ showConfirm: vi.fn(), showAlert: vi.fn() }))
vi.mock('@/composables/useConfirm', () => ({ useConfirm: () => confirm }))

const setup = () => {
  const documents = ref<any[]>([{ id: 'doc1', filename: 'a.pdf' }, { id: 'doc2', filename: 'b.pdf' }])
  const lists = {
    tolls: ref<any[]>([{ id: 'x1' }, { id: 't1' }, { id: 't2' }]),
    maintenance: ref<any[]>([{ id: 'x1' }, { id: 'm1' }]),
    charges: ref<any[]>([{ id: 'x1' }, { id: 'c1' }]),
  }
  const reload = vi.fn().mockResolvedValue(undefined)
  return { documents, lists, reload, h: useExpenseDeletions(documents, reload, lists) }
}

describe('useExpenseDeletions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
    store.activeVehicle = { id: 'v1' }
    confirm.showConfirm.mockResolvedValue(true)
    for (const fn of Object.values(api)) fn.mockResolvedValue(undefined)
  })

  // One row goes through its own wording, a selection through one confirmation for all of them
  const kinds = [
    ['tolls', 'handleDeleteTolls', 'deleteDriveExpense'],
    ['maintenance entries', 'handleDeleteMaints', 'deleteMaintenance'],
    ['charges', 'handleDeleteCharges', 'deleteCharge'],
  ] as const

  const listOf = { tolls: 'tolls', 'maintenance entries': 'maintenance', charges: 'charges' } as const

  for (const [label, handler, call] of kinds) {
    it(`takes a single one of the ${label} off the list at once and deletes it once the time to undo is over`, async () => {
      const { h, reload, lists } = setup()
      const list = lists[listOf[label]]
      await h[handler]([{ id: 'x1', amount: '12', kwh_added: 30, description: 'Oil' }])
      expect(confirm.showConfirm).not.toHaveBeenCalled()
      expect(list.value.map((r) => r.id)).not.toContain('x1')
      expect(api[call]).not.toHaveBeenCalled()
      await vi.advanceTimersByTimeAsync(UNDO_MS)
      expect(api[call]).toHaveBeenCalledWith('v1', 'x1')
      expect(reload).toHaveBeenCalledTimes(1)
    })

    it(`puts a single one of the ${label} back where it was when the deletion is taken back`, async () => {
      const { h, reload, lists } = setup()
      const list = lists[listOf[label]]
      const before = list.value.map((r) => r.id)
      await h[handler]([{ id: 'x1' }])
      toasts.value.at(-1)!.action!.run()
      expect(list.value.map((r) => r.id)).toEqual(before)
      await vi.advanceTimersByTimeAsync(UNDO_MS * 2)
      expect(api[call]).not.toHaveBeenCalled()
      expect(reload).not.toHaveBeenCalled()
    })

    it(`asks once for a selection of ${label}, deletes them one by one and reloads`, async () => {
      const { h, reload } = setup()
      await h[handler]([{ id: 'x1' }, { id: 'x2' }, { id: 'x3' }])
      expect(confirm.showConfirm).toHaveBeenCalledTimes(1)
      expect(confirm.showConfirm.mock.calls[0][0].message).toBe(t('expenses.expensesView.deleteManyMessage', { count: 3 }))
      expect(api[call].mock.calls).toEqual([['v1', 'x1'], ['v1', 'x2'], ['v1', 'x3']])
      expect(reload).toHaveBeenCalledTimes(1)
    })

    it(`deletes nothing from a selection of ${label} that is not confirmed`, async () => {
      confirm.showConfirm.mockResolvedValue(false)
      const { h, reload } = setup()
      await h[handler]([{ id: 'x1' }, { id: 'x2' }])
      expect(api[call]).not.toHaveBeenCalled()
      expect(reload).not.toHaveBeenCalled()
    })

    it(`brings a single one of the ${label} back and reports it when the server refuses the delete`, async () => {
      api[call].mockRejectedValueOnce(new Error('boom'))
      const { h, reload, lists } = setup()
      const list = lists[listOf[label]]
      await h[handler]([{ id: 'x1' }])
      await vi.advanceTimersByTimeAsync(UNDO_MS)
      expect(list.value.map((r) => r.id)).toContain('x1')
      expect(confirm.showAlert).toHaveBeenCalledWith(t('common.deleteError', { message: 'boom' }), t('shell.confirm.error'), 'danger')
      expect(reload).not.toHaveBeenCalled()
    })

    it(`reloads a selection of ${label} even when one delete fails, and stops at the failure`, async () => {
      api[call].mockResolvedValueOnce(undefined).mockRejectedValueOnce(new Error('boom'))
      const { h, reload } = setup()
      await h[handler]([{ id: 'x1' }, { id: 'x2' }, { id: 'x3' }])
      expect(api[call]).toHaveBeenCalledTimes(2)
      expect(confirm.showAlert).toHaveBeenCalledWith(t('common.deleteError', { message: 'boom' }), t('shell.confirm.error'), 'danger')
      expect(reload).toHaveBeenCalledTimes(1)
    })
  }

  it('does nothing without an active vehicle or an empty selection', async () => {
    const { h, reload } = setup()
    await h.handleDeleteCharges([])
    expect(confirm.showConfirm).not.toHaveBeenCalled()
    store.activeVehicle = null
    await h.handleDeleteTolls([{ id: 'x1' }, { id: 'x2' }])
    await h.handleDeleteMaints([{ id: 'x1' }])
    await h.handleDeleteCharge({ id: 'c1', kwh_added: 1 })
    await h.handleDeleteToll({ id: 't1', amount: '1' })
    await h.handleDeleteMaint({ id: 'm1', amount: '1' })
    await h.handleDeleteDocument({ id: 'doc1', filename: 'a.pdf' } as any)
    expect(confirm.showConfirm).not.toHaveBeenCalled()
    expect(reload).not.toHaveBeenCalled()
  })

  it('removes a deleted receipt from the list, confirms it and reloads', async () => {
    const { h, documents, reload } = setup()
    await h.handleDeleteDocument(documents.value[0])
    expect(api.deleteDocument).toHaveBeenCalledWith('v1', 'doc1')
    expect(documents.value.map((d) => d.id)).toEqual(['doc2'])
    expect(confirm.showAlert).toHaveBeenCalledWith(t('expenses.expensesView.receiptDeleted'), t('common.success'), 'success')
    expect(reload).toHaveBeenCalledTimes(1)
  })

  it('keeps the receipt and reports the error when its delete fails, and does not delete when declined', async () => {
    const { h, documents } = setup()
    confirm.showConfirm.mockResolvedValueOnce(false)
    await h.handleDeleteDocument(documents.value[0])
    expect(api.deleteDocument).not.toHaveBeenCalled()
    api.deleteDocument.mockRejectedValueOnce(new Error('nope'))
    await h.handleDeleteDocument(documents.value[0])
    expect(documents.value).toHaveLength(2)
    expect(confirm.showAlert).toHaveBeenCalledWith(t('common.errorWithMessage', { message: 'nope' }), t('shell.confirm.error'), 'danger')
  })
})
